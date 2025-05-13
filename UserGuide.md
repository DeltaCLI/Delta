# Delta CLI User Guide

## Introduction

Delta CLI (∆) is an intelligent command-line tool that enhances your terminal experience with AI-powered features, memory capabilities, and advanced command management. This guide will walk you through all the features and commands available in Delta CLI.

## Getting Started

### Installation

Delta CLI can be installed using the provided installation script:

```bash
make install
```

This will install the `delta` binary to `/usr/local/bin/delta`.

### Basic Usage

Once installed, you can start Delta CLI by simply typing:

```bash
delta
```

You'll see the Delta CLI prompt:

```
Welcome to Delta! 🔼

[deltacli] ∆ 
```

### Command Syntax

Delta CLI supports two types of commands:

1. **Shell Commands**: Any regular shell command works in Delta CLI just like in your normal terminal.
2. **Internal Commands**: These start with a colon (`:`) and provide Delta-specific functionality.

Example:
```
[deltacli] ∆ ls -la                 # Regular shell command
[deltacli] ∆ :help                  # Delta internal command
```

## Core Features

### AI Assistant

Delta's AI assistant can predict and suggest commands based on your usage patterns.

```
[deltacli] ∆ :ai                    # Show AI status
[deltacli] ∆ :ai on                 # Enable AI features
[deltacli] ∆ :ai off                # Disable AI features
[deltacli] ∆ :ai model <name>       # Change the AI model
[deltacli] ∆ :ai status             # Show detailed AI status
[deltacli] ∆ :ai feedback <type>    # Provide feedback on suggestions
```

### Jump Navigation

Quickly navigate to commonly used directories:

```
[deltacli] ∆ :jump add <name> [path]   # Add a location
[deltacli] ∆ :jump <name>              # Jump to a saved location
[deltacli] ∆ :jump list                # List all saved locations
[deltacli] ∆ :jump remove <name>       # Remove a location
[deltacli] ∆ :j <name>                 # Shorthand for jump
```

### Memory System

Delta can remember and analyze your command history:

```
[deltacli] ∆ :memory status          # Show memory system status
[deltacli] ∆ :memory enable          # Enable memory collection
[deltacli] ∆ :memory disable         # Disable memory collection
[deltacli] ∆ :memory stats           # Show detailed memory statistics
[deltacli] ∆ :memory list            # List available data shards
[deltacli] ∆ :memory export          # Export memory data
[deltacli] ∆ :memory import <path>   # Import memory data
[deltacli] ∆ :mem                    # Shorthand for memory commands
```

### Tokenizer

Manage the command tokenizer for AI learning:

```
[deltacli] ∆ :tokenizer status      # Show tokenizer status
[deltacli] ∆ :tokenizer stats       # Show detailed tokenizer statistics
[deltacli] ∆ :tokenizer process     # Process command data for training
[deltacli] ∆ :tok                   # Shorthand for tokenizer
```

### Inference System

Control how Delta learns from your commands:

```
[deltacli] ∆ :inference enable       # Enable inference system
[deltacli] ∆ :inference disable      # Disable inference system
[deltacli] ∆ :inference feedback     # Provide feedback on predictions
[deltacli] ∆ :inference model        # Manage custom models
[deltacli] ∆ :inf                    # Shorthand for inference
```

### Vector Database

Search for semantically similar commands:

```
[deltacli] ∆ :vector enable          # Enable vector database
[deltacli] ∆ :vector disable         # Disable vector database
[deltacli] ∆ :vector search <cmd>    # Search for similar commands
[deltacli] ∆ :vector embed <cmd>     # Generate embedding for a command
```

### Embedding System

Manage command embeddings for semantic search:

```
[deltacli] ∆ :embedding enable       # Enable embedding system
[deltacli] ∆ :embedding disable      # Disable embedding system
[deltacli] ∆ :embedding generate     # Generate embedding for a command
```

### Speculative Decoding

Control fast prediction generation:

```
[deltacli] ∆ :speculative enable     # Enable speculative decoding
[deltacli] ∆ :speculative disable    # Disable speculative decoding
[deltacli] ∆ :speculative draft      # Test speculative drafting
[deltacli] ∆ :specd                  # Shorthand for speculative
```

### Knowledge Extraction

Manage project and environment knowledge:

```
[deltacli] ∆ :knowledge enable       # Enable knowledge extraction
[deltacli] ∆ :knowledge query <text> # Search for knowledge
[deltacli] ∆ :knowledge context      # Show current environment context
[deltacli] ∆ :knowledge scan         # Scan current directory for knowledge
[deltacli] ∆ :know                   # Shorthand for knowledge
```

### Agent System

Manage task-specific automation agents:

```
[deltacli] ∆ :agent enable           # Enable agent system
[deltacli] ∆ :agent list             # List all agents
[deltacli] ∆ :agent show <id>        # Show agent details
[deltacli] ∆ :agent run <id>         # Run an agent
[deltacli] ∆ :agent create <name>    # Create a new agent
```

### Configuration System

Manage Delta CLI settings:

```
[deltacli] ∆ :config                 # Show configuration status
[deltacli] ∆ :config list            # List all configurations
[deltacli] ∆ :config export <path>   # Export configuration
[deltacli] ∆ :config import <path>   # Import configuration
[deltacli] ∆ :config edit <comp>     # Edit specific component config
```

### Spell Checker

Detect and fix command typos:

```
[deltacli] ∆ :spellcheck enable      # Enable spell checking
[deltacli] ∆ :spellcheck disable     # Disable spell checking
[deltacli] ∆ :spellcheck add <word>  # Add word to dictionary
[deltacli] ∆ :spellcheck test <cmd>  # Test spell checking
[deltacli] ∆ :spell                  # Shorthand for spellcheck
```

### History Analysis

Analyze and suggest commands based on history:

```
[deltacli] ∆ :history                # Show recent history
[deltacli] ∆ :history search <query> # Search command history
[deltacli] ∆ :history suggest        # Show command suggestions
[deltacli] ∆ :history patterns       # Show command patterns
[deltacli] ∆ :hist                   # Shorthand for history
```

### System Commands

General system commands:

```
[deltacli] ∆ :help                   # Show help information
[deltacli] ∆ :init                   # Initialize all systems
```

## Advanced Usage

### Command Completion

Delta CLI supports tab completion for both shell commands and internal Delta commands. Simply press the Tab key to see available options.

### History Navigation

Press Up and Down arrow keys to navigate through your command history.

### Command Suggestions

When enabled, Delta will automatically suggest commands based on your history:

```
[deltacli] ∆ git status
[Suggestion: git push]
```

### Command Correction

If you make a typo in a command, Delta will suggest corrections:

```
[deltacli] ∆ :inferense
Unknown command: :inferense
Did you mean ':inference'?
```

### Context-Aware Suggestions

Delta analyzes your current directory and recent commands to provide context-aware suggestions:

```
[project/backend] ∆ npm start
[deltacli] ∆ 
[Suggestion: npm test]
```

## Configuration

### Configuration Files

Delta CLI stores its configuration in the following locations:

```
~/.config/delta/                      # Main config directory
├── system_config.json                # System-wide configuration
├── memory/                           # Memory data
├── memory_config.json                # Memory configuration
└── history/                          # History analysis data
```

### Environment Variables

Delta CLI respects the following environment variables:

- `DELTA_CONFIG_DIR`: Override the default config directory
- `DELTA_DISABLE_AI`: Disable AI features if set
- `DELTA_MODEL`: Specify the default AI model

## Troubleshooting

### Common Issues

1. **AI features not working**:
   - Ensure Ollama is installed and running
   - Check that the selected model is available

2. **Memory system errors**:
   - Check disk space and permissions on `~/.config/delta/`
   - Try running `:memory clear` to reset the memory system

3. **Performance issues**:
   - Disable heavy components like vector database if experiencing slowness
   - Use `:config edit` to adjust resource usage settings

### Getting Help

For additional help, use the `:help` command or consult the detailed documentation by running:

```
[deltacli] ∆ :help <command>
```

## Best Practices

1. **Use Jump Locations**: Add frequently visited directories to jump locations for quick navigation.

2. **Provide Feedback**: Use `:feedback` to improve AI predictions.

3. **Organize Knowledge**: Use `:knowledge scan` in project directories to build a knowledge base.

4. **Create Agents**: Use agents for repetitive tasks in your workflow.

5. **Export Your Config**: Regularly export your configuration with `:config export` for backup.

## Privacy

Delta CLI is designed with privacy in mind:

- Command history is stored locally only
- Sensitive commands with passwords, API keys, etc. are filtered out
- All AI processing happens locally (when using local models)
- You can use `:memory config` to adjust privacy filters

## Advanced Customization

### Custom Agents

Create specialized agents for your workflow:

```
[deltacli] ∆ :agent create "BuildAndTest"
[deltacli] ∆ :agent edit BuildAndTest
```

### Knowledge Management

Manage project-specific knowledge:

```
[deltacli] ∆ :knowledge project import <path>
[deltacli] ∆ :knowledge project export <path>
```

### Training Custom Models

Delta supports training custom models on your command history:

```
[deltacli] ∆ :memory train start
```

This will launch a training job in Docker (requires Docker to be installed).

## Conclusion

Delta CLI is designed to enhance your terminal experience with AI-powered features while respecting your privacy and workflow. As you use Delta CLI more, it will learn your patterns and become increasingly helpful.

For the latest updates and features, visit the Delta CLI repository.

---

*This User Guide is part of the Delta CLI documentation.*