# Delta CLI Training Pipeline

This document outlines the approach for implementing a nightly training pipeline that allows Delta to learn from your terminal usage patterns and improve its predictions over time.

## Dataset Collection System

### Command Collection

```go
// CommandEntry represents a single collected command with context
type CommandEntry struct {
    Command     string    `json:"command"`
    Directory   string    `json:"directory"`
    Timestamp   time.Time `json:"timestamp"`
    ExitCode    int       `json:"exit_code"`
    Duration    int64     `json:"duration_ms"`
    Environment map[string]string `json:"environment,omitempty"` // Selective environment variables
    PrevCommand string    `json:"prev_command,omitempty"`
    NextCommand string    `json:"next_command,omitempty"`
    SystemInfo  string    `json:"system_info,omitempty"`
}
```

1. **Passive Collection**:
   - Hook into Delta's command execution flow
   - Record each command, its context, working directory, and result
   - Store in a structured binary format with compression
   - Respect privacy by excluding sensitive commands (configurable)

2. **Active Collection**:
   - Add a `:train` command for explicit training examples
   - Example: `:train "when I do this" "I often want to do that next"`
   - Allow tagging good/bad predictions: `:feedback helpful`, `:feedback unhelpful`

### Storage Format

Create a binary dataset format similar to smolGPT's approach:

1. **Daily Shards**:
   - Each day's commands stored in a separate binary file
   - Use memory-mapped files for efficient access
   - Compress with a simple scheme to save space

2. **Data Directory Structure**:
   ```
   ~/.config/delta/training/
   ├── raw/
   │   ├── commands_2025-05-11.bin
   │   ├── commands_2025-05-12.bin
   │   └── ...
   ├── processed/
   │   ├── tokenized_2025-05.bin
   │   └── ...
   ├── models/
   │   ├── delta_model_latest.bin
   │   ├── delta_model_2025-05-15.bin
   │   └── ...
   └── config.json
   ```

## Tokenization

Adapt smolGPT's tokenization approach for terminal commands:

1. **Custom Tokenizer**:
   - Train a SentencePiece tokenizer on collected commands
   - Include special tokens for command boundaries, directories, etc.
   - Periodically update tokenizer as vocabulary evolves

2. **Command Preprocessing**:
   - Normalize paths (replace home directory with ~)
   - Replace usernames and hostnames with tokens
   - Handle special characters and control sequences

## Training Pipeline

### Nightly Training Process

```
┌───────────────┐     ┌───────────────┐     ┌───────────────┐     ┌───────────────┐
│ Data          │     │ Preprocess    │     │ Train/Finetune│     │ Evaluate      │
│ Collection    │────▶│ & Tokenize    │────▶│ Model         │────▶│ & Deploy      │
└───────────────┘     └───────────────┘     └───────────────┘     └───────────────┘
```

1. **Scheduler**:
   - Run training during idle time or at fixed intervals (e.g., 2 AM)
   - Skip if insufficient new data or system is busy
   - Use systemd timers or cron for scheduling

2. **Data Preparation**:
   - Merge daily shards into training batches
   - Apply tokenization and create context windows
   - Generate synthetic examples for rare commands

3. **Multi-GPU Training Process**:
   - Implement distributed training using PyTorch DDP (DistributedDataParallel)
   - Use the Trainer class approach from smolGPT's train-multigpu.py
   - Distribute batch processing across available GPUs using NCCL backend
   - Synchronize gradients efficiently across devices
   - Support mixed precision training (bfloat16/float16) for faster iterations
   - Implement gradient accumulation for larger effective batch sizes
   - Use cosine learning rate schedule with warmup
   - Include checkpointing to resume training

4. **Evaluation**:
   - Test on held-out command sequences
   - Compare with previous model version
   - Only deploy if clear improvements

### Model Architecture

Adapt the smolGPT architecture for terminal commands:

```python
@dataclass
class DeltaModelConfig:
    block_size: int = 256         # Max command sequence length
    vocab_size: int = 8192        # Terminal command vocabulary tends to be limited
    n_layer: int = 6              # Smaller model for faster inference
    n_head: int = 6               # Multiple attention heads
    n_embed: int = 384            # Embedding dimension
    dropout: float = 0.1          # Regularization
    bias: bool = False            # Modern GPT-like architecture
```

### Docker Integration

Create a Docker container optimized for multi-GPU training:

```dockerfile
FROM nvidia/cuda:12.1.0-devel-ubuntu22.04 AS base

# Install Python and dependencies
RUN apt-get update && apt-get install -y \
    python3 python3-pip \
    python3-dev build-essential cmake \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Install PyTorch with CUDA support and other dependencies
COPY requirements.txt .
RUN pip3 install --no-cache-dir torch==2.1.0 torchvision torchaudio --index-url https://download.pytorch.org/whl/cu121
RUN pip3 install --no-cache-dir -r requirements.txt

# Copy training code
COPY training/ .

# Create volumes for data and models
VOLUME /data
VOLUME /models

# Entry point script that supports distributed training
COPY docker-entrypoint.sh /
RUN chmod +x /docker-entrypoint.sh
ENTRYPOINT ["/docker-entrypoint.sh"]
```

Create a docker-entrypoint.sh script for multi-GPU support:

```bash
#!/bin/bash

# Get the number of GPUs
NUM_GPUS=$(nvidia-smi --list-gpus | wc -l)

if [ "$NUM_GPUS" -gt 1 ]; then
    echo "Found $NUM_GPUS GPUs, launching distributed training"

    # Launch multi-GPU training using torch.distributed.launch
    python3 -m torch.distributed.run \
        --nproc_per_node=$NUM_GPUS \
        --nnodes=1 \
        --node_rank=0 \
        --master_addr="127.0.0.1" \
        --master_port=29500 \
        train-multigpu.py "$@"
else
    echo "Found only $NUM_GPUS GPU, launching single GPU training"
    python3 train.py "$@"
fi
```

Use with a Docker Compose file optimized for multi-GPU:

```yaml
version: '3'

services:
  delta-training:
    build: .
    volumes:
      - ~/.config/delta/training/raw:/data/raw
      - ~/.config/delta/training/processed:/data/processed
      - ~/.config/delta/training/models:/models
    environment:
      - MODEL_SIZE=small
      - BATCH_SIZE=32
      - GRADIENT_ACCUMULATION_STEPS=4
      - LEARNING_RATE=3e-4
      - WARMUP_ITERS=1000
      - MAX_ITERS=30000
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: all  # Use all available GPUs
              capabilities: [gpu]
    # Ensure sufficient shared memory for multiprocessing
    shm_size: 8gb
    ulimits:
      memlock: -1  # Unlimited memory lock
```

## Advanced Training Techniques from Latest Research

### Speculative Decoding
Implement speculative decoding to speed up inference during interactive use:
- Use a smaller "draft" model to predict multiple tokens at once
- Have the primary model verify these predictions in parallel
- This can achieve 2-3x speedup with minimal quality loss
- Reference: ["Accelerating LLM Inference with Speculative Decoding"](https://arxiv.org/abs/2302.01318)

### Low-Rank Adaptation (LoRA)
Use Parameter-Efficient Fine-Tuning to reduce training costs:
- Only update a small set of adapter parameters instead of full model weights
- Reduce memory requirements by 3-10x
- Enable fine-tuning on consumer hardware
- Much faster convergence for domain adaptation
- Reference: ["LoRA: Low-Rank Adaptation of Large Language Models"](https://arxiv.org/abs/2106.09685)

### Grouped-Query Attention (GQA)
Implement more efficient attention mechanism:
- Share key and value projections across multiple query heads
- Reduces inference memory requirements
- Improves computational efficiency
- Maintains model quality with proper tuning
- Reference: ["GQA: Training Generalized Multi-Query Transformer Models from Multi-Head Checkpoints"](https://arxiv.org/abs/2305.13245)

### Flash Attention
Optimize memory usage during training:
- Implement tiled attention computation to reduce memory bandwidth
- Up to 3x faster training with the same hardware
- Uses a more efficient algorithm for attention computation
- Reference: ["FlashAttention-2: Faster Attention with Better Parallelism and Work Partitioning"](https://arxiv.org/abs/2307.08691)

### Sparse Mixture of Experts (SMoE)
Consider sparse architecture for larger model capacity:
- Train specialized "expert" modules activated selectively per input
- Only a subset of the model processes each input
- Allows scaling model capacity without proportional computation
- Reference: ["Mixture-of-Experts with Expert Choice Routing"](https://arxiv.org/abs/2202.09368)

### Continuous Batching
Implement continuous batching for higher GPU utilization:
- Process multiple requests simultaneously even with different lengths
- Maintain one model copy in GPU memory
- Dynamically schedule computations to fill GPU capacity
- Reference: ["Orca: A Distributed Serving System for Transformer-Based Generative Models"](https://www.usenix.org/conference/osdi22/presentation/yu)

## Implementation Plan

### Phase 1: Data Collection
1. Add command logging to Delta CLI
2. Create structured storage format
3. Implement privacy controls and filtering
4. Add feedback mechanisms

### Phase 2: Preprocessing Pipeline
1. Develop command tokenizer
2. Create preprocessing scripts
3. Set up automated dataset generation
4. Implement binary dataset format

### Phase 3: Training System
1. Adapt smolGPT training for terminal commands
2. Implement LoRA for efficient fine-tuning
3. Add Flash Attention for faster training
4. Create Docker container with multi-GPU support
5. Implement nightly training scheduler
6. Add model evaluation metrics

### Phase 4: Inference Optimization
1. Implement speculative decoding for faster responses
2. Add GQA attention mechanism
3. Create continuous batching for better throughput

### Phase 5: Integration
1. Update AI manager to use trained models
2. Create model versioning system
3. Add fallback mechanisms
4. Implement A/B testing for improvements

## Privacy and Security Considerations

1. **Data Containment**:
   - All data stays on the user's machine
   - No cloud uploads (unless explicitly requested)
   - Exclude sensitive commands by default

2. **Filtering Options**:
   - Allow excluding commands containing passwords/tokens
   - Provide configuration for sensitive directories
   - Option to review collected data

3. **Resource Controls**:
   - Limit training to idle periods
   - Control CPU/GPU usage
   - Pause training when on battery power

4. **Model Security**:
   - Encrypt model files at rest
   - Validate model integrity before loading
   - Restrict model to authorized users

## Example Usage

Adding a command to the training dataset:

```sh
∆ :train add "git commit -m" "Adds changes to the git repository"
```

Providing feedback on a prediction:

```sh
∆ :train feedback "That was helpful" # Positive reinforcement
∆ :train feedback "Not what I meant" # Negative feedback
```

Viewing training status:

```sh
∆ :train status
> Last training: 2025-05-10 02:15 AM
> Dataset size: 4,521 commands
> Model version: delta-v3.2
> Next scheduled training: 2025-05-11 02:00 AM
```

Manually triggering training:

```sh
∆ :train now
> Starting training process in background...
> Use :train status to monitor progress
```

## Next Steps

1. Create proof-of-concept for command collection
2. Develop simple tokenizer for terminal commands
3. Adapt smolGPT training loop for command prediction
4. Test on synthetic command sequences