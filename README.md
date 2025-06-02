# Delta CLI

Delta is an intelligent shell wrapper that enhances your command-line experience with AI-powered suggestions, encrypted command history, and seamless shell compatibility.

## Features

- **Universal Shell Compatibility**: Works with bash, zsh, fish, and preserves your existing shell functions and aliases
- **AI-Powered Suggestions**: Context-aware predictions and insights using local Ollama models
- **Secure Command History**: Encrypted storage with privacy filtering
- **Advanced Memory System**: Learn from your command patterns and improve over time
- **Custom Model Training**: Train personalized models on your command history
- **Smart Navigation**: Quick directory jumping and path completion
- **Vector Search**: Fast semantic search through your command history

## Installation

```bash
# Clone the repository
git clone https://github.com/DeltaCLI/Delta.git
cd Delta

# Build the application
make build

# Install to your system
make install
```

### Requirements

- Go 1.16 or higher
- [Ollama](https://ollama.ai/) (for AI features)
- SQLite with vector extensions (automatically handled)

## Usage

Simply run `delta` to start the interactive shell:

```bash
delta
```

### Basic Commands

- `exit` or `quit` - Exit Delta
- `sub` - Enter subcommand mode
- `end` - Exit subcommand mode

### Internal Commands

Delta uses a colon (`:`) prefix for internal commands (similar to Vim):

- `:ai on` - Enable AI suggestions
- `:ai off` - Disable AI suggestions
- `:ai status` - Check if AI suggestions are enabled
- `:memory` - Memory system commands
- `:tokenizer` - Command tokenization utilities
- `:help` - Show all available commands
- `:jump <location>` - Quick directory navigation

### AI Features

Delta includes AI-powered contextual suggestions using Ollama with llama3.3:8b model.

#### Requirements

- [Ollama](https://ollama.ai/) installed and running locally
- llama3.3:8b model pulled (`ollama pull llama3.3:8b`)

#### How It Works

The AI analyzes your recent commands and displays a single line of "thinking" above the prompt. This provides contextual insights or suggestions based on your work. All processing happens locally via Ollama, ensuring your command data never leaves your machine.

### Memory and Learning System

Delta includes a sophisticated memory and learning system that can remember your command history and learn from your usage patterns over time.

#### Command Memory

Delta can safely store your command history with privacy filtering:

```bash
# Enable memory collection
:memory enable

# Check memory status
:memory status

# View detailed memory statistics
:memory stats

# List available data shards
:memory list

# Export data for a specific date
:memory export YYYY-MM-DD
```

#### Tokenization

Before training, commands need to be tokenized:

```bash
# Check tokenizer status
:tokenizer status

# Process command history into training data
:tokenizer process

# Test tokenization on a sample command
:tokenizer test "git commit -m 'Update README'"
```

#### Training Your Own Model

Delta supports training custom models on your command history using Docker:

##### Prerequisites
- Docker and Docker Compose installed
- NVIDIA GPU with CUDA support (optional, but recommended)
- NVIDIA Container Toolkit installed (for GPU support)

##### Training Process

1. **Collect Command Data**: Use Delta CLI regularly with memory collection enabled

2. **Process Command Data**: Convert raw commands to training data
   ```bash
   :tokenizer process
   ```

3. **Start Training**: Launch the Docker training environment
   ```bash
   :memory train start
   ```

4. **Configure Training** (Optional): Modify training parameters in `~/.config/delta/training/docker-compose.yml`:
   ```yaml
   environment:
     - MODEL_SIZE=small       # small, medium, or large
     - BATCH_SIZE=32          # batch size per GPU
     - MAX_ITERS=30000        # maximum training iterations
   ```

5. **Monitor Training**: Training logs are stored in `~/.config/delta/training/logs`

Training will automatically utilize all available GPUs with distributed training. The model will be saved to `~/.config/delta/memory/models` and used by Delta's AI system.

## Building from Source

Requirements:
- Go 1.16 or higher
- Make

```bash
# Build
make build

# Run without installing
make run

# Install to system
make install
```

## Configuration

Delta uses your existing shell's configuration files (.bashrc, .zshrc, etc.) for compatibility with your customized environment.

## Support

DeltaCLI is supported by continued investment from Source Parts Inc. ([https://source.parts](https://source.parts) / [https://sourceparts.eu](https://sourceparts.eu)).

## License

MIT License

Copyright (c) 2025 Source Parts Inc.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.

## Training Data Notice

This project does not provide GitHub or any other party without explicit consent to train on the source code contained herein.
