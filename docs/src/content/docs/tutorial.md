# Delta CLI Tutorial: Advanced Features

This tutorial covers the advanced features of Delta CLI, including agent commands, embedding management, vector search, inference commands, and more.

## Table of Contents
- [Agent System](#agent-system)
- [Embedding Management](#embedding-management)
- [Vector Search](#vector-search)
- [Inference System](#inference-system)
- [Memory Management](#memory-management)
- [Knowledge Extraction](#knowledge-extraction)
- [Spellcheck](#spellcheck)
- [Jump Navigation](#jump-navigation)
- [Speculative Decoding](#speculative-decoding)

## Agent System

The agent system allows you to automate tasks with customizable agents.

```
∆ :agent status                # Show agent manager status
∆ :agent enable                # Enable the agent manager
∆ :agent disable               # Disable the agent manager
∆ :agent list                  # List all available agents
∆ :agent show <id>             # Display details about a specific agent
∆ :agent run <id>              # Execute a specific agent
∆ :agent create <name>         # Create a new agent
```

### Docker Integration

Delta CLI supports Docker integration for agents:

```
∆ :agent docker list           # List Docker-enabled agents
∆ :agent docker build <id>     # Build Docker image for agent
```

### Error Learning

Agents can learn from errors and suggest solutions:

```
∆ :agent errors list           # List learned error solutions
```

## Embedding Management

Embeddings enable semantic understanding of commands.

```
∆ :embedding status            # Show embedding system status
∆ :embedding enable            # Enable the embedding system
∆ :embedding disable           # Disable the embedding system
∆ :embedding generate <cmd>    # Generate embedding for a command
∆ :embedding download          # Download embedding model and vocabulary
∆ :embedding config set <k> <v>  # Update embedding configuration
```

## Vector Search

Search for similar commands semantically:

```
∆ :vector status               # Show vector database status
∆ :vector enable               # Enable the vector database
∆ :vector disable              # Disable the vector database
∆ :vector search <cmd>         # Find similar commands (uses cosine by default)
∆ :vector search metric:euclidean <cmd>  # Use Euclidean distance
∆ :vector search metric:dot <cmd>        # Use dot product
∆ :vector export <file>        # Export vector database
∆ :vector import <file>        # Import vector database
∆ :vector config set <k> <v>   # Update vector database configuration
```

## Inference System

The inference system learns from your usage patterns:

```
∆ :inference status            # Show inference system status
∆ :inference enable            # Enable the learning system
∆ :inference disable           # Disable the learning system
∆ :inference feedback <type>   # Add feedback for the last prediction
∆ :inference model use <path>  # Use a custom model
∆ :inference examples          # Show training examples
```

## Memory Management

Delta CLI can remember your command history and learn from it:

```
∆ :memory status               # Show memory system status
∆ :memory enable               # Enable memory collection
∆ :memory disable              # Disable memory collection
∆ :memory export               # Export memory data
∆ :memory import <path>        # Import memory data
∆ :memory train start          # Start training a model
∆ :memory train add <pattern> <explanation>  # Add training example
```

## Knowledge Extraction

Extract knowledge from your environment:

```
∆ :knowledge status            # Show knowledge extractor status
∆ :knowledge enable            # Enable knowledge extraction
∆ :knowledge disable           # Disable knowledge extraction
∆ :knowledge query <text>      # Semantic search for knowledge
∆ :knowledge context           # Show current environment context
∆ :knowledge scan              # Scan directory for knowledge
∆ :knowledge project           # Show project information
∆ :knowledge agent suggest     # Suggest agents based on knowledge
```

## Spellcheck

Delta CLI includes a spell checker for commands:

```
∆ :spellcheck                  # Show spell checker status
∆ :spellcheck enable           # Enable spell checking
∆ :spellcheck disable          # Disable spell checking
∆ :spellcheck config           # Show configuration
∆ :spellcheck config threshold=0.8  # Set threshold for suggestions
∆ :spellcheck config auto_correct=true  # Enable auto-correction
∆ :spellcheck add <word>       # Add word to custom dictionary
∆ :spellcheck remove <word>    # Remove word from dictionary
∆ :spellcheck test <command>   # Test spell checking on a command
```

## Jump Navigation

Quickly navigate between directories:

```
∆ :jump                        # List jump locations
∆ :j                           # Shorthand for :jump
∆ :jump <location>             # Jump to a saved location
∆ :j <location>                # Shorthand for jump to location
∆ :jump add <name> [path]      # Add location (current dir or path)
∆ :jump remove <name>          # Remove a location
∆ :jump import jumpsh          # Import from jump.sh
```

## Speculative Decoding

Accelerate command completion with speculative decoding:

```
∆ :speculative                 # Show current status
∆ :speculative enable          # Enable speculative decoding
∆ :speculative disable         # Disable speculative decoding
∆ :speculative status          # Display detailed status
∆ :speculative stats           # Show performance statistics
∆ :speculative config          # Show configuration
∆ :speculative config set draft_tokens 5  # Set tokens to predict
∆ :speculative config set accept_threshold 0.95  # Set acceptance threshold
```

Speculative decoding can improve performance by predicting multiple tokens at once, potentially speeding up command completion by 2-3x.

---

For more detailed information about any command, use the help option:
```
∆ :<command> help
```

Delta CLI is designed to improve your command-line productivity with AI-powered features that learn from your usage patterns and provide intelligent assistance.