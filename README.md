# Delta CLI

Delta is an enhanced shell wrapper with intelligent features, shell function compatibility, and encrypted command history.

## Features

- Support for shell functions and aliases in all shells (bash, zsh, fish)
- Encrypted command history
- Tab completion for commands and file paths
- Intelligent context-aware AI predictions powered by Ollama

## Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/deltacli.git
cd deltacli

# Build the application
make build

# Install
make install
```

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

## License

MIT