FROM nvidia/cuda:12.1.0-devel-ubuntu22.04 AS base

# Install Python and dependencies
RUN apt-get update && apt-get install -y \
    python3 python3-pip \
    python3-dev build-essential cmake \
    git curl wget unzip \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Install PyTorch with CUDA support
RUN pip3 install --no-cache-dir torch==2.1.0 torchvision torchaudio --index-url https://download.pytorch.org/whl/cu121

# Install other dependencies
COPY requirements.txt .
RUN pip3 install --no-cache-dir -r requirements.txt

# Create directories for data and models
RUN mkdir -p /data/raw /data/processed /models /checkpoints /logs

# Copy training code
COPY *.py ./

# Entry point script that supports distributed training
COPY docker-entrypoint.sh /
RUN chmod +x /docker-entrypoint.sh
ENTRYPOINT ["/docker-entrypoint.sh"]