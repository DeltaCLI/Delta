# Delta Memory Enhancement Plan

This document outlines a comprehensive plan to implement a learning and memory system for Delta CLI, allowing it to retain knowledge about the user's terminal environment and adapt over time.

## Goals

1. Enable Delta to maintain persistent memory across sessions
2. Allow users to train Delta to understand their specific terminal environment
3. Improve predictions based on historical interactions
4. Package the solution in a way that's portable and easy to maintain
5. Create a seamless user experience for memory management

## Implementation Approach

### 1. Memory Storage System

#### Vector Database
- Implement a vector database for semantic memory storage
- Use embeddings to store and retrieve semantically similar commands and contexts
- Possible implementations:
  - Integrated: SQLite + pgvector extension
  - Standalone: Qdrant, Chroma, or Milvus
  - Lightweight: FAISS with file-based storage

```go
// Example vector store interface
type VectorStore interface {
    StoreEmbedding(text string, metadata map[string]interface{}) error
    SimilaritySearch(query string, limit int) ([]SearchResult, error)
    DeleteEmbedding(id string) error
}
```

#### Relational Database
- Use SQLite for structured data storage:
  - Command history with context
  - User feedback on predictions
  - Environment information
  - Custom prompt templates
  - Learning examples

```sql
-- Example schema
CREATE TABLE commands (
  id INTEGER PRIMARY KEY,
  command TEXT,
  timestamp DATETIME,
  directory TEXT,
  result INTEGER, -- exit code
  duration INTEGER, -- execution time in ms
  embedding_id TEXT -- reference to vector store
);

CREATE TABLE feedback (
  id INTEGER PRIMARY KEY,
  command_id INTEGER,
  feedback_type TEXT, -- "helpful", "unhelpful", "corrected"
  feedback TEXT,
  FOREIGN KEY (command_id) REFERENCES commands(id)
);

CREATE TABLE environment (
  id INTEGER PRIMARY KEY,
  snapshot_date DATETIME,
  env_data TEXT -- JSON blob of environment data
);

CREATE TABLE training_examples (
  id INTEGER PRIMARY KEY,
  pattern TEXT,
  explanation TEXT,
  timestamp DATETIME,
  embedding_id TEXT -- reference to vector store
);
```

#### Docker Integration
- Create a Dockerfile for containerized memory storage
- Expose API endpoints for interaction with the memory store
- Allow easy backup and migration of memory data

### 2. Tokenization and Embedding

- Implement a local tokenization and embedding system:
  - Use SentencePiece or BPE tokenizer for consistent tokenization
  - Utilize ONNX-based embedding models that can run locally
  - Support for incremental learning with new tokens for user-specific commands

```go
// EmbeddingGenerator handles text-to-vector conversion
type EmbeddingGenerator struct {
    tokenizer *Tokenizer
    model *EmbeddingModel
    vocabCache map[string]int
}

// NewEmbeddingGenerator creates a new embedding generator
func NewEmbeddingGenerator() (*EmbeddingGenerator, error) {
    // Initialize tokenizer and model
    // Load vocabulary cache
}

// GetEmbedding converts text to vector representation
func (eg *EmbeddingGenerator) GetEmbedding(text string) ([]float32, error) {
    // Tokenize text
    // Generate embedding
    // Return vector
}
```

### 3. Memory Manager

Following the pattern of JumpManager, create a MemoryManager:

```go
// MemoryManager handles AI memory and learning
type MemoryManager struct {
    db *sql.DB
    vectorStore VectorStore
    embedder *EmbeddingGenerator
    configPath string
    customPrompts map[string]string
    feedbackEnabled bool
    learningEnabled bool
}

// NewMemoryManager creates a new memory manager
func NewMemoryManager() *MemoryManager {
    // Initialize database connection
    // Set up vector store
    // Initialize embedding generator
    // Load custom prompts
    // Set up configurations
}
```

### 4. Environment Analysis System

- Analyze shell configuration files (.bashrc, .zshrc, etc.)
- Scan for common aliases, functions, and tools
- Track frequently used directories and applications
- Detect custom scripts and utilities
- Generate embeddings for environment components for semantic search

### 5. Learning Interface

#### Command-line Interface
Add new commands to Delta for memory management:

```
∆ :ai learn "<pattern>" "<explanation>"      # Add a training example
∆ :ai feedback "<helpful|unhelpful>"         # Provide feedback on last prediction
∆ :ai reset                                  # Reset memory (with confirmation)
∆ :ai prompt set <custom prompt>             # Set a custom base prompt
∆ :ai analyze                                # Analyze environment and learn
∆ :ai status                                 # Show learning status and statistics
∆ :ai train                                  # Fine-tune local embeddings with user data
```

#### Passive Learning
- Track command success/failure to learn patterns automatically
- Observe sequences of commands to understand workflows
- Note environment differences between failed and successful commands
- Generate embeddings for new commands to improve semantic search

### 6. Integration with Existing AI System

Modify the AI prediction system to:

1. Use vector similarity search to find relevant past experiences
2. Incorporate memory data in predictions
3. Load custom prompts from the memory store
4. Allow fine-tuning based on user feedback
5. Support learning from examples

Update in `ai_manager.go`:
```go
// Modify generateThought to use memory
func (m *AIPredictionManager) generateThought() {
    // Existing code...
    
    // Get relevant memory from MemoryManager using vector similarity
    memoryManager := GetMemoryManager()
    
    // Convert current context to embedding and find similar past experiences
    currentContext := strings.Join(history, " ")
    relevantMemories := memoryManager.GetSimilarExperiences(currentContext, 5)
    
    // Incorporate memory into the prompt
    prompt := fmt.Sprintf(
        "Here are my recent commands:\n%s\n\nRelevant past experiences:\n%s\n\nProvide a helpful thought:",
        historyStr,
        relevantMemories,
    )
    
    // Use custom prompt if available
    systemPrompt := m.contextPrompt
    customPrompt := memoryManager.GetCustomPrompt()
    if customPrompt != "" {
        systemPrompt = customPrompt
    }
    
    thought, err := m.ollamaClient.Generate(prompt, systemPrompt)
    // Existing code...
}
```

### 7. Docker Deployment Strategy

Create a Docker setup for persistent memory storage:

```dockerfile
FROM golang:1.19-alpine AS builder

WORKDIR /build
COPY . .
RUN go build -o delta

FROM alpine:latest

# Install dependencies for embedding and vector storage
RUN apk add --no-cache sqlite-dev libgomp libc6-compat

# Copy binary and necessary files
COPY --from=builder /build/delta /usr/local/bin/
COPY --from=builder /build/models /models

# Set up volume for persistent storage
VOLUME /data

# Environment variables for configuration
ENV DELTA_MEMORY_PATH=/data
ENV DELTA_EMBEDDING_MODEL=all-MiniLM-L6-v2

ENTRYPOINT ["delta"]
```

### 8. Example Docker Compose Setup

```yaml
version: '3'

services:
  delta-memory:
    build: .
    volumes:
      - ./data:/data
    environment:
      - DELTA_MEMORY_PATH=/data
      - DELTA_EMBEDDING_MODEL=all-MiniLM-L6-v2
    ports:
      - "8080:8080"  # Optional API port
```

## Implementation Timeline

1. **Phase 1: Vector Database & Embedding System**
   - Implement vector storage with SQLite + extensions or standalone vector DB
   - Create lightweight embedding system using ONNX runtime
   - Build basic vector similarity search

2. **Phase 2: Memory Storage Infrastructure**
   - Implement SQLite relational database
   - Create MemoryManager with CRUD operations
   - Add configuration options

3. **Phase 3: Learning Interface**
   - Implement CLI commands for memory management
   - Create feedback collection system
   - Build environment analysis components

4. **Phase 4: AI Integration**
   - Update AI prediction to use vector similarity search
   - Implement custom prompt loading
   - Add adaptive training based on feedback

5. **Phase 5: Docker Integration**
   - Create Dockerfile and docker-compose setup
   - Implement data persistence strategy
   - Add backup and restore functionality

6. **Phase 6: Testing and Refinement**
   - Real-world testing across different environments
   - Performance optimization
   - User experience improvements

## User Experience

The memory system should be:

1. **Transparent** - Users should understand what Delta is learning
2. **Controllable** - Easy to enable/disable or reset learning
3. **Beneficial** - Provide measurably better assistance over time
4. **Unobtrusive** - Learning should happen in the background without disrupting workflow
5. **Privacy-focused** - All data stays local by default

## Next Steps

1. Research lightweight embedding models suitable for CLI environment
2. Implement proof-of-concept vector storage with sample command data
3. Create basic memory manager with vector similarity search
4. Modify AI manager to work with the vector-based memory system
5. Create Docker container for testing