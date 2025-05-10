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

### AI Features

Delta includes AI-powered contextual suggestions using Ollama with llama3.3:8b model.

#### Requirements

- [Ollama](https://ollama.ai/) installed and running locally
- llama3.3:8b model pulled (`ollama pull llama3.3:8b`)

#### How It Works

The AI analyzes your recent commands and displays a single line of "thinking" above the prompt. This provides contextual insights or suggestions based on your work. All processing happens locally via Ollama, ensuring your command data never leaves your machine.

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