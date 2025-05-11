#!/bin/bash

# Get the number of GPUs
NUM_GPUS=$(nvidia-smi --list-gpus | wc -l)

if [ "$NUM_GPUS" -gt 1 ]; then
    echo "Found $NUM_GPUS GPUs, launching distributed training"
    
    # Launch multi-GPU training using torch.distributed.run
    python3 -m torch.distributed.run \
        --nproc_per_node=$NUM_GPUS \
        --nnodes=1 \
        --node_rank=0 \
        --master_addr="127.0.0.1" \
        --master_port=29500 \
        train_multi.py "$@"
else
    echo "Found only $NUM_GPUS GPU, launching single GPU training"
    python3 train.py "$@"
fi