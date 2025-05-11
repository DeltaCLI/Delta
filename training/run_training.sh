#!/bin/bash

# Set up environment variables
export DATA_DIR=${1:-/data/processed}
export OUTPUT_DIR=${2:-/checkpoints}
export MODEL_SIZE=${3:-small}
export BATCH_SIZE=${4:-32}
export MAX_ITERS=${5:-30000}

# Display configuration
echo "Delta CLI Training"
echo "=================="
echo "Data directory: $DATA_DIR"
echo "Output directory: $OUTPUT_DIR"
echo "Model size: $MODEL_SIZE"
echo "Batch size: $BATCH_SIZE"
echo "Max iterations: $MAX_ITERS"
echo ""

# Check if we have data
if [ ! -d "$DATA_DIR" ] || [ -z "$(ls -A $DATA_DIR)" ]; then
    echo "Error: No training data found in $DATA_DIR"
    echo "Please run tokenizer processing first using:"
    echo "  :tokenizer process"
    exit 1
fi

# Create output directories
mkdir -p "$OUTPUT_DIR"
mkdir -p "$OUTPUT_DIR/logs"

# Check for available GPUs
NUM_GPUS=$(nvidia-smi --list-gpus | wc -l 2>/dev/null || echo 0)

echo "Found $NUM_GPUS GPUs"

if [ "$NUM_GPUS" -gt 1 ]; then
    echo "Using distributed training on $NUM_GPUS GPUs"
    # Use torchrun for multi-GPU training
    python3 -m torch.distributed.run \
        --nproc_per_node=$NUM_GPUS \
        --nnodes=1 \
        --node_rank=0 \
        --master_addr="127.0.0.1" \
        --master_port=29500 \
        train_multi.py \
        --data_dir="$DATA_DIR" \
        --output_dir="$OUTPUT_DIR" \
        --model_size="$MODEL_SIZE" \
        --batch_size="$BATCH_SIZE" \
        --max_iters="$MAX_ITERS"
elif [ "$NUM_GPUS" -eq 1 ]; then
    echo "Using single GPU training"
    python3 train.py \
        --data_dir="$DATA_DIR" \
        --output_dir="$OUTPUT_DIR" \
        --model_size="$MODEL_SIZE" \
        --batch_size="$BATCH_SIZE" \
        --max_iters="$MAX_ITERS"
else
    echo "No GPUs detected, using CPU training (this will be very slow)"
    python3 train.py \
        --data_dir="$DATA_DIR" \
        --output_dir="$OUTPUT_DIR" \
        --model_size="$MODEL_SIZE" \
        --batch_size=$((BATCH_SIZE / 4)) \
        --max_iters="$MAX_ITERS"
fi

# Check if training was successful
if [ -f "$OUTPUT_DIR/checkpoint_latest.pt" ]; then
    echo "Training completed successfully!"
    echo "Model checkpoints saved to $OUTPUT_DIR"
    
    # Generate ONNX model
    echo "Converting model to ONNX format..."
    cd "$OUTPUT_DIR" && python3 convert_to_onnx.py
    
    # Copy model to Delta models directory
    echo "Copying model to Delta..."
    DELTA_MODELS_DIR="$HOME/.config/delta/memory/models"
    mkdir -p "$DELTA_MODELS_DIR"
    cp "$OUTPUT_DIR/model.onnx" "$DELTA_MODELS_DIR/delta_model_$(date +%Y%m%d).onnx"
    cp "$OUTPUT_DIR/model.onnx" "$DELTA_MODELS_DIR/delta_model_latest.onnx"
    echo "Model copied to $DELTA_MODELS_DIR"
else
    echo "Training failed or was interrupted."
    exit 1
fi