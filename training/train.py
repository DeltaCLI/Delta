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
from torch.utils.data import Dataset, DataLoader
import numpy as np
from tqdm import tqdm
from torch.utils.tensorboard import SummaryWriter

# Configure logging
logging.basicConfig(
    format="%(asctime)s - %(levelname)s - %(name)s - %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
    level=logging.INFO,
)
logger = logging.getLogger(__name__)

# Command token type constants
TOKEN_TYPE_COMMAND = 0
TOKEN_TYPE_FLAG = 1
TOKEN_TYPE_ARGUMENT = 2
TOKEN_TYPE_PATH = 3
TOKEN_TYPE_PIPE = 4
TOKEN_TYPE_REDIRECT = 5
TOKEN_TYPE_VARIABLE = 6
TOKEN_TYPE_OPERATOR = 7
TOKEN_TYPE_UNKNOWN = 8

@dataclass
class ModelConfig:
    vocab_size: int = 10000
    block_size: int = 256
    n_layer: int = 6
    n_head: int = 8
    n_embd: int = 384
    dropout: float = 0.1
    bias: bool = False  # True: bias in Linears and LayerNorms, like GPT-2, False: a bit better and faster

@dataclass
class TrainingConfig:
    learning_rate: float = 3e-4
    weight_decay: float = 1e-1
    beta1: float = 0.9
    beta2: float = 0.95
    device: str = "auto"  # "auto", "cuda", "cpu"
    dtype: str = "bfloat16"  # "float32", "bfloat16", or "float16"
    compile: bool = True  # use PyTorch 2.0 to compile the model to be faster
    batch_size: int = 64
    max_iters: int = 30000
    lr_decay_iters: int = 30000
    min_lr: float = 3e-5
    warmup_iters: int = 1000
    grad_clip: float = 1.0
    gradient_accumulation_steps: int = 1
    eval_interval: int = 100
    eval_iters: int = 20
    log_interval: int = 10
    output_dir: str = "./checkpoints"
    run_name: Optional[str] = None
    seed: int = 1337
    init_from: str = "scratch"  # "scratch" or "resume"
    resume_checkpoint: Optional[str] = None

@dataclass
class DataConfig:
    data_dir: str = "/data/processed"
    train_split: float = 0.9
    val_split: float = 0.1
    shuffle: bool = True
    num_workers: int = 4
    cache_dir: Optional[str] = "/data/cache"
    
class CommandDataset(Dataset):
    """Dataset for terminal command sequences"""
    
    def __init__(self, data_dir, split="train", block_size=256, cache_dir=None):
        self.data_dir = Path(data_dir)
        self.split = split
        self.block_size = block_size
        self.cache_dir = Path(cache_dir) if cache_dir else None
        
        # Load all tokenized command files
        self.files = list(self.data_dir.glob("tokenized_*.bin"))
        
        # If there are no files, raise an error
        if not self.files:
            raise ValueError(f"No tokenized command files found in {self.data_dir}")
            
        logger.info(f"Found {len(self.files)} tokenized command files")
        
        # Split files into train/val
        if split == "train":
            self.files = self.files[:int(len(self.files) * 0.9)]
        else:
            self.files = self.files[int(len(self.files) * 0.9):]
            
        logger.info(f"Using {len(self.files)} files for {split} split")
        
        # Load or create cache
        self.data = []
        self._load_data()
        
    def _load_data(self):
        """Load tokenized command data from files"""
        total_tokens = 0
        total_commands = 0
        
        for file_path in tqdm(self.files, desc=f"Loading {self.split} data"):
            # Read the tokenized commands
            with open(file_path, "rb") as f:
                # Read the number of entries (first 4 bytes)
                entries_bytes = f.read(4)
                if len(entries_bytes) < 4:
                    continue
                    
                num_entries = int.from_bytes(entries_bytes, byteorder='little')
                
                # Read each entry
                for _ in range(num_entries):
                    # Read the length of the entry (4 bytes)
                    length_bytes = f.read(4)
                    if len(length_bytes) < 4:
                        break
                        
                    data_length = int.from_bytes(length_bytes, byteorder='little')
                    
                    # Read the JSON data
                    json_data = f.read(data_length)
                    if len(json_data) < data_length:
                        break
                        
                    try:
                        # Decode the JSON data
                        command_tokens = json.loads(json_data)
                        
                        # Convert tokens to tensor format
                        token_ids = []
                        for token in command_tokens.get("tokens", []):
                            # For simplicity, we're just using the token text as the ID
                            # In a real system, you'd convert these to vocabulary IDs
                            token_ids.append(hash(token["text"]) % 10000)
                            
                        if token_ids:
                            self.data.append({
                                "tokens": token_ids,
                                "command": command_tokens.get("command", ""),
                                "exit_code": command_tokens.get("exit_code", 0),
                            })
                            total_tokens += len(token_ids)
                            total_commands += 1
                    except json.JSONDecodeError:
                        continue
        
        logger.info(f"Loaded {total_commands} commands with {total_tokens} tokens for {self.split} split")
        
    def __len__(self):
        return len(self.data)
        
    def __getitem__(self, idx):
        item = self.data[idx]
        tokens = item["tokens"]
        
        # Pad or truncate to block_size
        if len(tokens) > self.block_size:
            tokens = tokens[:self.block_size]
        elif len(tokens) < self.block_size:
            tokens = tokens + [0] * (self.block_size - len(tokens))
            
        # Convert to tensors
        x = torch.tensor(tokens[:-1], dtype=torch.long)
        y = torch.tensor(tokens[1:], dtype=torch.long)
        
        return x, y
        
class CausalSelfAttention(nn.Module):
    """Multi-head self-attention with causal masking"""
    
    def __init__(self, config):
        super().__init__()
        assert config.n_embd % config.n_head == 0
        
        # Key, query, value projections
        self.c_attn = nn.Linear(config.n_embd, 3 * config.n_embd, bias=config.bias)
        # Output projection
        self.c_proj = nn.Linear(config.n_embd, config.n_embd, bias=config.bias)
        # Regularization
        self.attn_dropout = nn.Dropout(config.dropout)
        self.resid_dropout = nn.Dropout(config.dropout)
        
        self.n_head = config.n_head
        self.n_embd = config.n_embd
        self.dropout = config.dropout
        
        # Flash attention makes faster when sequence length is > 1024
        self.flash = hasattr(torch.nn.functional, 'scaled_dot_product_attention')
        if not self.flash:
            logger.warning("Flash attention not available, using slow attention")
            # Causal mask to ensure attention only to previous tokens
            self.register_buffer(
                "mask", 
                torch.tril(torch.ones(config.block_size, config.block_size))
                .view(1, 1, config.block_size, config.block_size)
            )
    
    def forward(self, x):
        B, T, C = x.size() # batch size, sequence length, embedding dimensionality
        
        # Calculate query, key, values for all heads in batch
        q, k, v = self.c_attn(x).split(self.n_embd, dim=2)
        k = k.view(B, T, self.n_head, C // self.n_head).transpose(1, 2) # (B, nh, T, hs)
        q = q.view(B, T, self.n_head, C // self.n_head).transpose(1, 2) # (B, nh, T, hs)
        v = v.view(B, T, self.n_head, C // self.n_head).transpose(1, 2) # (B, nh, T, hs)
        
        # Attention
        if self.flash:
            # Flash attention
            y = torch.nn.functional.scaled_dot_product_attention(
                q, k, v, 
                attn_mask=None, 
                dropout_p=self.dropout if self.training else 0,
                is_causal=True
            )
        else:
            # Manual attention
            att = (q @ k.transpose(-2, -1)) * (1.0 / math.sqrt(k.size(-1)))
            att = att.masked_fill(self.mask[:,:,:T,:T] == 0, float('-inf'))
            att = F.softmax(att, dim=-1)
            att = self.attn_dropout(att)
            y = att @ v # (B, nh, T, T) x (B, nh, T, hs) -> (B, nh, T, hs)
            
        # Re-assemble all head outputs side by side
        y = y.transpose(1, 2).contiguous().view(B, T, C) # (B, T, C)
        
        # Output projection
        y = self.resid_dropout(self.c_proj(y))
        return y
        
class MLP(nn.Module):
    """
    Multi-layer perceptron with GELU activation as used in Transformer architectures.
    """
    
    def __init__(self, config):
        super().__init__()
        self.c_fc = nn.Linear(config.n_embd, 4 * config.n_embd, bias=config.bias)
        self.gelu = nn.GELU()
        self.c_proj = nn.Linear(4 * config.n_embd, config.n_embd, bias=config.bias)
        self.dropout = nn.Dropout(config.dropout)
        
    def forward(self, x):
        x = self.c_fc(x)
        x = self.gelu(x)
        x = self.c_proj(x)
        x = self.dropout(x)
        return x
        
class Block(nn.Module):
    """Transformer block with self-attention, MLP, and layer normalization"""
    
    def __init__(self, config):
        super().__init__()
        self.ln_1 = nn.LayerNorm(config.n_embd, bias=config.bias)
        self.attn = CausalSelfAttention(config)
        self.ln_2 = nn.LayerNorm(config.n_embd, bias=config.bias)
        self.mlp = MLP(config)
        
    def forward(self, x):
        x = x + self.attn(self.ln_1(x))
        x = x + self.mlp(self.ln_2(x))
        return x
        
class DeltaGPT(nn.Module):
    """Delta GPT model for terminal command prediction"""
    
    def __init__(self, config):
        super().__init__()
        self.config = config
        
        # Token embeddings
        self.tok_emb = nn.Embedding(config.vocab_size, config.n_embd)
        # Position embeddings
        self.pos_emb = nn.Embedding(config.block_size, config.n_embd)
        # Dropout after embeddings
        self.drop = nn.Dropout(config.dropout)
        # Transformer blocks
        self.blocks = nn.ModuleList([Block(config) for _ in range(config.n_layer)])
        # Final layer norm
        self.ln_f = nn.LayerNorm(config.n_embd, bias=config.bias)
        # Output projection
        self.head = nn.Linear(config.n_embd, config.vocab_size, bias=False)
        
        # Initialize weights
        self.apply(self._init_weights)
        
        logger.info(f"Number of parameters: {sum(p.numel() for p in self.parameters())}")
        
    def _init_weights(self, module):
        if isinstance(module, nn.Linear):
            # Small init for better training stability
            torch.nn.init.normal_(module.weight, mean=0.0, std=0.02)
            if module.bias is not None:
                torch.nn.init.zeros_(module.bias)
        elif isinstance(module, nn.Embedding):
            torch.nn.init.normal_(module.weight, mean=0.0, std=0.02)
            
    def forward(self, idx, targets=None):
        B, T = idx.size()
        assert T <= self.config.block_size, f"Input sequence length ({T}) exceeds model's context size ({self.config.block_size})"
        
        # Get token embeddings
        tok_emb = self.tok_emb(idx) # (B, T, C)
        
        # Get position embeddings
        pos = torch.arange(0, T, dtype=torch.long, device=idx.device).unsqueeze(0) # (1, T)
        pos_emb = self.pos_emb(pos) # (1, T, C)
        
        # Add token and position embeddings
        x = self.drop(tok_emb + pos_emb) # (B, T, C)
        
        # Apply transformer blocks
        for block in self.blocks:
            x = block(x)
            
        # Apply final layer norm
        x = self.ln_f(x)
        
        # Get logits
        logits = self.head(x)
        
        # Calculate loss if targets are provided
        loss = None
        if targets is not None:
            loss = F.cross_entropy(logits.view(-1, logits.size(-1)), targets.view(-1))
            
        return logits, loss
    
    def generate(self, idx, max_new_tokens, temperature=1.0, top_k=None, top_p=None):
        """Generate text given a context"""
        B, T = idx.size()
        
        for _ in range(max_new_tokens):
            # If context is too long, truncate it
            idx_cond = idx if idx.size(1) <= self.config.block_size else idx[:, -self.config.block_size:]
            
            # Get predictions
            logits, _ = self(idx_cond)
            
            # Focus only on the last time step
            logits = logits[:, -1, :] / temperature # (B, C)
            
            # Optional filtering
            if top_k is not None:
                v, _ = torch.topk(logits, top_k)
                logits[logits < v[:, [-1]]] = -float('Inf')
                
            if top_p is not None:
                sorted_logits, sorted_indices = torch.sort(logits, descending=True)
                cumulative_probs = torch.cumsum(F.softmax(sorted_logits, dim=-1), dim=-1)
                
                # Remove tokens with cumulative probability above the threshold
                sorted_indices_to_remove = cumulative_probs > top_p
                # Shift the indices to the right to keep also the first token above the threshold
                sorted_indices_to_remove[..., 1:] = sorted_indices_to_remove[..., :-1].clone()
                sorted_indices_to_remove[..., 0] = 0
                
                # Scatter sorted tensors to original indexing
                indices_to_remove = sorted_indices_to_remove.scatter(1, sorted_indices, sorted_indices_to_remove)
                logits[indices_to_remove] = -float('Inf')
                
            # Apply softmax to get probabilities
            probs = F.softmax(logits, dim=-1)
            
            # Sample from the distribution
            idx_next = torch.multinomial(probs, num_samples=1)
            
            # Append to the sequence
            idx = torch.cat((idx, idx_next), dim=1)
            
        return idx
    
    def configure_optimizers(self, weight_decay, learning_rate, betas, device_type):
        """Configure optimizer with weight decay"""
        # Separate parameters that should have weight decay and those that shouldn't
        decay = set()
        no_decay = set()
        
        # All parameters with "bias" or "ln" or "layernorm" or "embedding" in their name don't use weight decay
        whitelist_weight_modules = (nn.Linear, )
        blacklist_weight_modules = (nn.LayerNorm, nn.Embedding)
        
        for mn, m in self.named_modules():
            for pn, p in m.named_parameters():
                fpn = f"{mn}.{pn}" if mn else pn
                
                if pn.endswith('bias'):
                    # All biases don't use weight decay
                    no_decay.add(fpn)
                elif pn.endswith('weight') and isinstance(m, whitelist_weight_modules):
                    # Weights of linear layers use weight decay
                    decay.add(fpn)
                elif pn.endswith('weight') and isinstance(m, blacklist_weight_modules):
                    # Weights of embedding and layer norm don't use weight decay
                    no_decay.add(fpn)
                    
        # Validate that we've considered all parameters
        param_dict = {pn: p for pn, p in self.named_parameters()}
        inter_params = decay & no_decay
        union_params = decay | no_decay
        assert len(inter_params) == 0, f"Parameters {inter_params} made it into both decay and no_decay sets!"
        assert len(param_dict.keys() - union_params) == 0, f"Parameters {param_dict.keys() - union_params} were not separated into either decay or no_decay set!"
        
        # Create optimizer with two groups
        optim_groups = [
            {"params": [param_dict[pn] for pn in sorted(list(decay))], "weight_decay": weight_decay},
            {"params": [param_dict[pn] for pn in sorted(list(no_decay))], "weight_decay": 0.0},
        ]
        
        # Use AdamW optimizer
        optimizer = torch.optim.AdamW(optim_groups, lr=learning_rate, betas=betas)
        return optimizer
        
def get_cosine_lr_schedule(optimizer, warmup_iters, lr_decay_iters, min_lr):
    """
    Cosine learning rate schedule with warmup.
    """
    def lr_lambda(step):
        # Warmup phase
        if step < warmup_iters:
            return float(step) / float(max(1, warmup_iters))
        
        # Cosine decay phase
        decay_ratio = min(1.0, float(step - warmup_iters) / float(max(1, lr_decay_iters - warmup_iters)))
        coeff = 0.5 * (1.0 + math.cos(math.pi * decay_ratio))
        return min_lr + coeff * (1.0 - min_lr)
    
    return torch.optim.lr_scheduler.LambdaLR(optimizer, lr_lambda)

def train():
    parser = argparse.ArgumentParser(description="Train a terminal command model")
    
    # Data arguments
    parser.add_argument("--data_dir", type=str, default="/data/processed", 
                        help="Directory with processed tokenized commands")
    parser.add_argument("--output_dir", type=str, default="/checkpoints",
                        help="Directory to save checkpoints")
    parser.add_argument("--log_dir", type=str, default="/logs",
                        help="Directory to save logs")
                        
    # Model arguments
    parser.add_argument("--model_size", type=str, default="small", choices=["small", "medium", "large"],
                       help="Model size to use")
                       
    # Training arguments
    parser.add_argument("--batch_size", type=int, default=32,
                       help="Batch size for training")
    parser.add_argument("--learning_rate", type=float, default=3e-4,
                       help="Learning rate")
    parser.add_argument("--max_iters", type=int, default=30000,
                       help="Maximum number of iterations")
    parser.add_argument("--gradient_accumulation_steps", type=int, default=1,
                       help="Number of gradient accumulation steps")
                       
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
    )
    
    # Check if CUDA is available
    if torch.cuda.is_available():
        device = torch.device("cuda")
        logger.info(f"Using GPU: {torch.cuda.get_device_name(0)}")
        train_config.device = "cuda"
    else:
        device = torch.device("cpu")
        logger.info("Using CPU")
        train_config.device = "cpu"
    
    # Create output directories
    os.makedirs(args.output_dir, exist_ok=True)
    os.makedirs(args.log_dir, exist_ok=True)
    
    # Set up TensorBoard writer
    writer = SummaryWriter(log_dir=args.log_dir)
    
    # Initialize datasets and data loaders
    logger.info("Loading datasets...")
    try:
        train_dataset = CommandDataset(args.data_dir, split="train", block_size=model_config.block_size)
        val_dataset = CommandDataset(args.data_dir, split="val", block_size=model_config.block_size)
        
        train_loader = DataLoader(
            train_dataset, 
            batch_size=train_config.batch_size, 
            shuffle=True, 
            num_workers=4,
            pin_memory=True,
        )
        
        val_loader = DataLoader(
            val_dataset, 
            batch_size=train_config.batch_size, 
            shuffle=False, 
            num_workers=4,
            pin_memory=True,
        )
        
        logger.info(f"Train dataset size: {len(train_dataset)}")
        logger.info(f"Validation dataset size: {len(val_dataset)}")
    except Exception as e:
        logger.error(f"Error loading datasets: {e}")
        logger.error("No training data found. Please process command data first.")
        return
    
    # Initialize model
    logger.info(f"Initializing {args.model_size} model...")
    model = DeltaGPT(model_config).to(device)
    
    # Optionally compile the model with PyTorch 2.0
    if train_config.compile and torch.__version__ >= "2.0.0":
        logger.info("Compiling model with PyTorch 2.0...")
        model = torch.compile(model)
    
    # Initialize optimizer
    optimizer = model.configure_optimizers(
        train_config.weight_decay,
        train_config.learning_rate,
        (train_config.beta1, train_config.beta2),
        train_config.device
    )
    
    # Set up learning rate scheduler
    lr_scheduler = get_cosine_lr_schedule(
        optimizer,
        train_config.warmup_iters,
        train_config.lr_decay_iters,
        train_config.min_lr
    )
    
    # Training loop
    logger.info("Starting training...")
    model.train()
    best_val_loss = float('inf')
    
    # Get data iterator
    train_iter = iter(train_loader)
    
    for iter_num in range(train_config.max_iters):
        # Get batch
        try:
            x, y = next(train_iter)
        except StopIteration:
            # Reset iterator if we've gone through the dataset
            train_iter = iter(train_loader)
            x, y = next(train_iter)
            
        x = x.to(device)
        y = y.to(device)
        
        # Forward pass
        logits, loss = model(x, y)
        
        # Loss scaling for gradient accumulation
        loss = loss / train_config.gradient_accumulation_steps
        
        # Backward pass
        loss.backward()
        
        # Only update weights after accumulating gradients
        if (iter_num + 1) % train_config.gradient_accumulation_steps == 0:
            # Clip gradients
            if train_config.grad_clip != 0.0:
                torch.nn.utils.clip_grad_norm_(model.parameters(), train_config.grad_clip)
                
            # Optimizer step
            optimizer.step()
            optimizer.zero_grad(set_to_none=True)
            
            # Update learning rate
            lr_scheduler.step()
        
        # Logging
        if iter_num % train_config.log_interval == 0:
            # Get current learning rate
            lr = lr_scheduler.get_last_lr()[0]
            
            # Log to console
            logger.info(f"Iter {iter_num}: loss {loss.item() * train_config.gradient_accumulation_steps:.4f}, lr {lr:.4e}")
            
            # Log to TensorBoard
            writer.add_scalar("train/loss", loss.item() * train_config.gradient_accumulation_steps, iter_num)
            writer.add_scalar("train/lr", lr, iter_num)
        
        # Validation
        if iter_num % train_config.eval_interval == 0:
            model.eval()
            val_losses = []
            
            with torch.no_grad():
                for eval_iter, (x_val, y_val) in enumerate(val_loader):
                    if eval_iter >= train_config.eval_iters:
                        break
                        
                    x_val = x_val.to(device)
                    y_val = y_val.to(device)
                    
                    _, val_loss = model(x_val, y_val)
                    val_losses.append(val_loss.item())
            
            # Calculate average validation loss
            avg_val_loss = sum(val_losses) / len(val_losses)
            
            # Log validation loss
            logger.info(f"Iter {iter_num}: val_loss {avg_val_loss:.4f}")
            writer.add_scalar("val/loss", avg_val_loss, iter_num)
            
            # Save checkpoint if validation loss improved
            if avg_val_loss < best_val_loss:
                best_val_loss = avg_val_loss
                
                # Save checkpoint
                checkpoint = {
                    "model": model.state_dict(),
                    "optimizer": optimizer.state_dict(),
                    "model_config": vars(model_config),
                    "iter_num": iter_num,
                    "best_val_loss": best_val_loss,
                }
                
                checkpoint_path = os.path.join(train_config.output_dir, f"checkpoint_{iter_num}.pt")
                logger.info(f"Saving checkpoint to {checkpoint_path}")
                torch.save(checkpoint, checkpoint_path)
                
                # Also save as latest
                latest_path = os.path.join(train_config.output_dir, "checkpoint_latest.pt")
                torch.save(checkpoint, latest_path)
            
            # Switch back to training mode
            model.train()
            
    # Save final model
    final_checkpoint = {
        "model": model.state_dict(),
        "optimizer": optimizer.state_dict(),
        "model_config": vars(model_config),
        "iter_num": train_config.max_iters,
        "best_val_loss": best_val_loss,
    }
    
    final_path = os.path.join(train_config.output_dir, "checkpoint_final.pt")
    logger.info(f"Saving final checkpoint to {final_path}")
    torch.save(final_checkpoint, final_path)
    
    # Generate converter script for ONNX
    generate_onnx_converter(model_config, train_config.output_dir)
    
    # Close TensorBoard writer
    writer.close()
    
    logger.info("Training complete!")
    
def generate_onnx_converter(model_config, output_dir):
    """Generate a script to convert PyTorch model to ONNX"""
    script_path = os.path.join(output_dir, "convert_to_onnx.py")
    
    script_content = f"""
import torch
import sys
from pathlib import Path
sys.path.append(str(Path(__file__).parent.parent))
from train import DeltaGPT, ModelConfig

def convert_to_onnx(checkpoint_path, output_path):
    # Load model configuration
    model_config = ModelConfig(
        vocab_size={model_config.vocab_size},
        block_size={model_config.block_size},
        n_layer={model_config.n_layer},
        n_head={model_config.n_head},
        n_embd={model_config.n_embd},
        dropout=0.0,  # Set to 0 for inference
        bias={str(model_config.bias).lower()},
    )
    
    # Create model
    model = DeltaGPT(model_config)
    
    # Load weights
    checkpoint = torch.load(checkpoint_path, map_location="cpu")
    model.load_state_dict(checkpoint["model"])
    
    # Set to eval mode
    model.eval()
    
    # Create dummy input
    dummy_input = torch.zeros(1, model_config.block_size // 2, dtype=torch.long)
    
    # Export to ONNX
    torch.onnx.export(
        model,
        (dummy_input, None),  # model inputs
        output_path,
        export_params=True,
        opset_version=15,
        do_constant_folding=True,
        input_names=["input_ids"],
        output_names=["logits"],
        dynamic_axes={{
            "input_ids": {{0: "batch_size", 1: "sequence_length"}},
            "logits": {{0: "batch_size", 1: "sequence_length"}}
        }}
    )
    
    print(f"Model exported to {{output_path}}")

if __name__ == "__main__":
    import argparse
    
    parser = argparse.ArgumentParser(description="Convert PyTorch model to ONNX")
    parser.add_argument("--checkpoint", type=str, default="checkpoint_latest.pt", help="Path to checkpoint file")
    parser.add_argument("--output", type=str, default="model.onnx", help="Output ONNX file path")
    
    args = parser.parse_args()
    
    checkpoint_path = args.checkpoint
    if not Path(checkpoint_path).is_file():
        checkpoint_path = str(Path(__file__).parent / args.checkpoint)
    
    output_path = args.output
    if not Path(output_path).is_absolute():
        output_path = str(Path(__file__).parent / args.output)
    
    convert_to_onnx(checkpoint_path, output_path)
"""
    
    with open(script_path, "w") as f:
        f.write(script_content)
    
    logger.info(f"Generated ONNX converter script at {script_path}")

if __name__ == "__main__":
    train()