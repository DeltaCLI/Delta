import os
import math
import time
import json
import logging
import argparse
from pathlib import Path
from dataclasses import dataclass, field
from typing import Optional, List, Dict, Any

import torch
import torch.nn as nn
import torch.nn.functional as F
import torch.distributed as dist
from torch.nn.parallel import DistributedDataParallel as DDP
from torch.utils.data import Dataset, DataLoader, DistributedSampler
import numpy as np
from tqdm import tqdm
from torch.utils.tensorboard import SummaryWriter

# Import model and dataset from train.py
from train import (
    ModelConfig, 
    TrainingConfig, 
    DataConfig, 
    CommandDataset, 
    DeltaGPT, 
    get_cosine_lr_schedule,
    generate_onnx_converter
)

# Configure logging
logging.basicConfig(
    format="%(asctime)s - %(levelname)s - %(name)s - %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
    level=logging.INFO,
)
logger = logging.getLogger(__name__)

class Trainer:
    """Trainer for multi-GPU training using PyTorch DDP"""
    
    def __init__(self, model_config, train_config, data_config):
        self.model_config = model_config
        self.train_config = train_config
        self.data_config = data_config
        
        # Set up distributed training
        self.setup_distributed()
        
        # Set up logging (only on main process)
        self.setup_logging()
        
        # Initialize datasets and data loaders
        self.setup_datasets()
        
        # Initialize model and optimizer
        self.setup_model_and_optimizer()
    
    def setup_distributed(self):
        """Set up distributed training"""
        if "RANK" not in os.environ:
            # Not running in distributed mode
            self.rank = 0
            self.local_rank = 0
            self.world_size = 1
            self.is_main_process = True
            self.distributed = False
            self.device = torch.device("cuda" if torch.cuda.is_available() else "cpu")
            return
            
        self.distributed = True
        
        # Initialize process group
        dist.init_process_group(backend="nccl")
        
        # Get distributed info
        self.rank = dist.get_rank()
        self.local_rank = int(os.environ.get("LOCAL_RANK", 0))
        self.world_size = dist.get_world_size()
        self.is_main_process = self.rank == 0
        
        # Set device
        self.device = torch.device(f"cuda:{self.local_rank}")
        torch.cuda.set_device(self.local_rank)
        
        # Adjust gradient accumulation steps based on world size
        self.train_config.gradient_accumulation_steps //= self.world_size
        if self.train_config.gradient_accumulation_steps < 1:
            self.train_config.gradient_accumulation_steps = 1
            
        # Set random seed based on rank for reproducibility
        torch.manual_seed(42 + self.rank)
        
        if self.is_main_process:
            logger.info(f"Distributed training with {self.world_size} GPUs")
            logger.info(f"Local rank: {self.local_rank}, Global rank: {self.rank}")
            logger.info(f"Gradient accumulation steps: {self.train_config.gradient_accumulation_steps}")
    
    def setup_logging(self):
        """Set up TensorBoard logging"""
        if self.is_main_process:
            os.makedirs(self.train_config.output_dir, exist_ok=True)
            log_dir = os.path.join(self.train_config.output_dir, "logs")
            os.makedirs(log_dir, exist_ok=True)
            self.writer = SummaryWriter(log_dir=log_dir)
        else:
            self.writer = None
    
    def setup_datasets(self):
        """Set up datasets and data loaders"""
        if self.is_main_process:
            logger.info("Loading datasets...")
        
        try:
            # Create datasets
            self.train_dataset = CommandDataset(
                self.data_config.data_dir, 
                split="train", 
                block_size=self.model_config.block_size,
                cache_dir=self.data_config.cache_dir,
            )
            
            self.val_dataset = CommandDataset(
                self.data_config.data_dir, 
                split="val", 
                block_size=self.model_config.block_size,
                cache_dir=self.data_config.cache_dir,
            )
            
            # Use DistributedSampler for training
            if self.distributed:
                train_sampler = DistributedSampler(
                    self.train_dataset,
                    num_replicas=self.world_size,
                    rank=self.rank,
                    shuffle=self.data_config.shuffle,
                )
                
                val_sampler = DistributedSampler(
                    self.val_dataset,
                    num_replicas=self.world_size,
                    rank=self.rank,
                    shuffle=False,
                )
            else:
                train_sampler = None
                val_sampler = None
            
            # Create data loaders
            self.train_loader = DataLoader(
                self.train_dataset,
                batch_size=self.train_config.batch_size,
                shuffle=(train_sampler is None and self.data_config.shuffle),
                sampler=train_sampler,
                num_workers=self.data_config.num_workers,
                pin_memory=True,
                drop_last=True,
            )
            
            self.val_loader = DataLoader(
                self.val_dataset,
                batch_size=self.train_config.batch_size,
                shuffle=False,
                sampler=val_sampler,
                num_workers=self.data_config.num_workers,
                pin_memory=True,
            )
            
            if self.is_main_process:
                logger.info(f"Train dataset size: {len(self.train_dataset)}")
                logger.info(f"Validation dataset size: {len(self.val_dataset)}")
                logger.info(f"Batch size per GPU: {self.train_config.batch_size}")
                logger.info(f"Total batch size: {self.train_config.batch_size * self.world_size}")
        
        except Exception as e:
            logger.error(f"Error loading datasets: {e}")
            logger.error("No training data found. Please process command data first.")
            if self.distributed:
                dist.destroy_process_group()
            exit(1)
    
    def setup_model_and_optimizer(self):
        """Initialize model and optimizer"""
        if self.is_main_process:
            logger.info(f"Initializing model with {self.model_config.n_layer} layers...")
        
        # Create model
        self.model = DeltaGPT(self.model_config).to(self.device)
        
        # Set up mixed precision
        self.use_mixed_precision = (
            self.train_config.dtype in ["float16", "bfloat16"] and 
            self.device.type == "cuda"
        )
        
        if self.use_mixed_precision:
            if self.train_config.dtype == "bfloat16" and torch.cuda.is_bf16_supported():
                self.amp_dtype = torch.bfloat16
            else:
                self.amp_dtype = torch.float16
            
            if self.is_main_process:
                logger.info(f"Using mixed precision training with {self.amp_dtype}")
            
            self.scaler = torch.cuda.amp.GradScaler()
        else:
            self.amp_dtype = torch.float32
            self.scaler = None
            
            if self.is_main_process:
                logger.info("Using full precision training")
        
        # Load from checkpoint if specified
        if self.train_config.init_from == "resume" and self.train_config.resume_checkpoint:
            self.load_checkpoint()
        else:
            # Initialize parameters
            self.iter_num = 0
            self.best_val_loss = float("inf")
        
        # Configure optimizer
        self.optimizer = self.model.configure_optimizers(
            self.train_config.weight_decay,
            self.train_config.learning_rate,
            (self.train_config.beta1, self.train_config.beta2),
            self.device.type
        )
        
        # Set up learning rate scheduler
        self.lr_scheduler = get_cosine_lr_schedule(
            self.optimizer,
            self.train_config.warmup_iters,
            self.train_config.lr_decay_iters,
            self.train_config.min_lr
        )
        
        # Wrap with DDP for distributed training
        if self.distributed:
            self.model = DDP(
                self.model, 
                device_ids=[self.local_rank],
                output_device=self.local_rank,
                find_unused_parameters=False,
                broadcast_buffers=False,
            )
            
            # Synchronize parameters
            if self.is_main_process:
                logger.info("Synchronizing model parameters across GPUs...")
            dist.barrier()
        
        # Compile model if requested
        if self.train_config.compile and hasattr(torch, "compile") and self.device.type == "cuda":
            if self.is_main_process:
                logger.info("Compiling model with torch.compile()...")
            
            # Get the actual model (bypass DDP wrapper if needed)
            model_to_compile = self.model.module if self.distributed else self.model
            
            # Compile the model
            self.model = torch.compile(self.model)
            
            if self.is_main_process:
                logger.info("Model compilation complete")
        
        # Log model parameters
        if self.is_main_process:
            param_count = sum(p.numel() for p in self.model.parameters())
            logger.info(f"Model parameters: {param_count:,}")
            
            effective_batch_size = (
                self.train_config.batch_size * 
                self.world_size * 
                self.train_config.gradient_accumulation_steps
            )
            logger.info(f"Effective batch size: {effective_batch_size}")
    
    def load_checkpoint(self):
        """Load model from checkpoint"""
        checkpoint_path = self.train_config.resume_checkpoint
        
        if self.is_main_process:
            logger.info(f"Loading checkpoint from {checkpoint_path}")
        
        try:
            checkpoint = torch.load(checkpoint_path, map_location=self.device)
            
            # Get the actual model (bypass DDP wrapper if needed)
            model_to_load = self.model.module if self.distributed else self.model
            
            # Clean up checkpoint state dict if needed
            state_dict = checkpoint["model"]
            unwanted_prefix = "_orig_mod."
            for k, v in list(state_dict.items()):
                if k.startswith(unwanted_prefix):
                    state_dict[k[len(unwanted_prefix):]] = state_dict.pop(k)
            
            # Load model weights
            model_to_load.load_state_dict(state_dict)
            
            # Load optimizer state if available
            if "optimizer" in checkpoint and not self.distributed:
                self.optimizer.load_state_dict(checkpoint["optimizer"])
            
            # Load iteration count and best validation loss
            self.iter_num = checkpoint.get("iter_num", 0)
            self.best_val_loss = checkpoint.get("best_val_loss", float("inf"))
            
            if self.is_main_process:
                logger.info(f"Resuming from iteration {self.iter_num} with validation loss {self.best_val_loss:.4f}")
        
        except Exception as e:
            logger.error(f"Error loading checkpoint: {e}")
            self.iter_num = 0
            self.best_val_loss = float("inf")
    
    def save_checkpoint(self, is_best=False):
        """Save model checkpoint"""
        if not self.is_main_process:
            return
        
        # Get the actual model (bypass DDP wrapper if needed)
        model_to_save = self.model.module if self.distributed else self.model
        
        # Create checkpoint
        checkpoint = {
            "model": model_to_save.state_dict(),
            "optimizer": self.optimizer.state_dict(),
            "model_config": vars(self.model_config),
            "iter_num": self.iter_num,
            "best_val_loss": self.best_val_loss,
        }
        
        # Save checkpoint
        checkpoint_path = os.path.join(self.train_config.output_dir, f"checkpoint_{self.iter_num}.pt")
        torch.save(checkpoint, checkpoint_path)
        logger.info(f"Saved checkpoint to {checkpoint_path}")
        
        # Save as latest
        latest_path = os.path.join(self.train_config.output_dir, "checkpoint_latest.pt")
        torch.save(checkpoint, latest_path)
        
        # Save as best if applicable
        if is_best:
            best_path = os.path.join(self.train_config.output_dir, "checkpoint_best.pt")
            torch.save(checkpoint, best_path)
            logger.info(f"Saved best checkpoint to {best_path}")
    
    def evaluate(self):
        """Evaluate model on validation set"""
        self.model.eval()
        val_losses = []
        
        with torch.no_grad():
            for eval_iter, (x_val, y_val) in enumerate(self.val_loader):
                if eval_iter >= self.train_config.eval_iters:
                    break
                
                x_val = x_val.to(self.device)
                y_val = y_val.to(self.device)
                
                # Forward pass with mixed precision
                if self.use_mixed_precision:
                    with torch.autocast(device_type="cuda", dtype=self.amp_dtype):
                        _, val_loss = self.model(x_val, y_val)
                else:
                    _, val_loss = self.model(x_val, y_val)
                
                val_losses.append(val_loss.item())
        
        # Aggregate losses across GPUs
        if self.distributed:
            val_loss_tensor = torch.tensor(val_losses, device=self.device)
            dist.reduce(val_loss_tensor, dst=0, op=dist.ReduceOp.SUM)
            
            if self.is_main_process:
                val_loss_tensor /= (self.world_size * len(val_losses))
                val_losses = val_loss_tensor.tolist()
        
        # Calculate average validation loss
        avg_val_loss = sum(val_losses) / len(val_losses) if val_losses else float("inf")
        
        # Return to training mode
        self.model.train()
        
        return avg_val_loss
    
    def train(self):
        """Main training loop"""
        if self.is_main_process:
            logger.info("Starting training...")
        
        self.model.train()
        
        # Get data iterator
        train_iter = iter(self.train_loader)
        
        for iter_num in range(self.iter_num, self.train_config.max_iters):
            self.iter_num = iter_num
            
            # Get batch
            try:
                x, y = next(train_iter)
            except StopIteration:
                # Reshuffle dataset for distributed training
                if self.distributed:
                    self.train_loader.sampler.set_epoch(iter_num)
                
                # Reset iterator
                train_iter = iter(self.train_loader)
                x, y = next(train_iter)
            
            x = x.to(self.device)
            y = y.to(self.device)
            
            # Update learning rate
            lr = self.train_config.learning_rate
            if self.train_config.decay_lr:
                lr = self.lr_scheduler.get_last_lr()[0]
                for param_group in self.optimizer.param_groups:
                    param_group["lr"] = lr
            
            # Forward and backward pass with mixed precision
            if self.use_mixed_precision:
                with torch.autocast(device_type="cuda", dtype=self.amp_dtype):
                    _, loss = self.model(x, y)
                    loss = loss / self.train_config.gradient_accumulation_steps
                
                # Scale loss and backward pass
                self.scaler.scale(loss).backward()
                
                # Step optimizer after accumulation
                if (iter_num + 1) % self.train_config.gradient_accumulation_steps == 0:
                    # Unscale before clipping
                    if self.train_config.grad_clip != 0.0:
                        self.scaler.unscale_(self.optimizer)
                        torch.nn.utils.clip_grad_norm_(self.model.parameters(), self.train_config.grad_clip)
                    
                    # Optimizer step
                    self.scaler.step(self.optimizer)
                    self.scaler.update()
                    self.optimizer.zero_grad(set_to_none=True)
            else:
                # Standard precision training
                _, loss = self.model(x, y)
                loss = loss / self.train_config.gradient_accumulation_steps
                loss.backward()
                
                # Step optimizer after accumulation
                if (iter_num + 1) % self.train_config.gradient_accumulation_steps == 0:
                    # Clip gradients
                    if self.train_config.grad_clip != 0.0:
                        torch.nn.utils.clip_grad_norm_(self.model.parameters(), self.train_config.grad_clip)
                    
                    # Optimizer step
                    self.optimizer.step()
                    self.optimizer.zero_grad(set_to_none=True)
            
            # Update learning rate scheduler
            if (iter_num + 1) % self.train_config.gradient_accumulation_steps == 0:
                self.lr_scheduler.step()
            
            # Logging
            if iter_num % self.train_config.log_interval == 0 and self.is_main_process:
                # Get current learning rate
                current_lr = self.lr_scheduler.get_last_lr()[0]
                
                # Log to console
                logger.info(f"Iter {iter_num}: loss {loss.item() * self.train_config.gradient_accumulation_steps:.4f}, lr {current_lr:.4e}")
                
                # Log to TensorBoard
                self.writer.add_scalar("train/loss", loss.item() * self.train_config.gradient_accumulation_steps, iter_num)
                self.writer.add_scalar("train/lr", current_lr, iter_num)
            
            # Validation
            if iter_num % self.train_config.eval_interval == 0:
                avg_val_loss = self.evaluate()
                
                if self.is_main_process:
                    # Log validation loss
                    logger.info(f"Iter {iter_num}: val_loss {avg_val_loss:.4f}")
                    self.writer.add_scalar("val/loss", avg_val_loss, iter_num)
                    
                    # Save checkpoint if validation loss improved
                    if avg_val_loss < self.best_val_loss:
                        self.best_val_loss = avg_val_loss
                        self.save_checkpoint(is_best=True)
                
                # Synchronize processes
                if self.distributed:
                    dist.barrier()
            
            # Save regular checkpoint
            if iter_num % (self.train_config.eval_interval * 10) == 0 and self.is_main_process:
                self.save_checkpoint()
                
                # Synchronize processes
                if self.distributed:
                    dist.barrier()
        
        # Save final model
        if self.is_main_process:
            logger.info("Training complete!")
            self.save_checkpoint()
            
            # Generate ONNX converter
            generate_onnx_converter(self.model_config, self.train_config.output_dir)
        
        # Clean up
        if self.distributed:
            dist.destroy_process_group()
        
        if self.is_main_process and self.writer is not None:
            self.writer.close()

def main():
    parser = argparse.ArgumentParser(description="Train a terminal command model with multi-GPU support")
    
    # Data arguments
    parser.add_argument("--data_dir", type=str, default="/data/processed", 
                        help="Directory with processed tokenized commands")
    parser.add_argument("--output_dir", type=str, default="/checkpoints",
                        help="Directory to save checkpoints")
                        
    # Model arguments
    parser.add_argument("--model_size", type=str, default="small", choices=["small", "medium", "large"],
                       help="Model size to use")
                       
    # Training arguments
    parser.add_argument("--batch_size", type=int, default=32,
                       help="Batch size for training (per GPU)")
    parser.add_argument("--learning_rate", type=float, default=3e-4,
                       help="Learning rate")
    parser.add_argument("--max_iters", type=int, default=30000,
                       help="Maximum number of iterations")
    parser.add_argument("--gradient_accumulation_steps", type=int, default=4,
                       help="Number of gradient accumulation steps")
    parser.add_argument("--resume", type=str, default=None,
                       help="Resume training from checkpoint")
                       
    args = parser.parse_args()
    
    # Set model configuration based on size
    if args.model_size == "small":
        model_config = ModelConfig(
            n_layer=6, 
            n_head=8, 
            n_embd=384,
        )
    elif args.model_size == "medium":
        model_config = ModelConfig(
            n_layer=8, 
            n_head=12, 
            n_embd=512,
        )
    else:  # large
        model_config = ModelConfig(
            n_layer=12, 
            n_head=16, 
            n_embd=768,
        )
    
    # Set up training configuration
    train_config = TrainingConfig(
        batch_size=args.batch_size,
        learning_rate=args.learning_rate,
        max_iters=args.max_iters,
        gradient_accumulation_steps=args.gradient_accumulation_steps,
        output_dir=args.output_dir,
        init_from="resume" if args.resume else "scratch",
        resume_checkpoint=args.resume,
    )
    
    # Set up data configuration
    data_config = DataConfig(
        data_dir=args.data_dir,
        cache_dir="/data/cache",
    )
    
    # Create trainer
    trainer = Trainer(model_config, train_config, data_config)
    
    # Start training
    trainer.train()

if __name__ == "__main__":
    main()