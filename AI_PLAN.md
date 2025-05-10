# AI Integration Plan for Delta CLI

This document outlines the plan for integrating Ollama with LLama3.3:8b into the Delta CLI application to provide intelligent predictions and assistance during shell sessions.

## Overview

We will add AI capabilities to Delta CLI by integrating with Ollama running locally. The AI will observe user commands, build context, and provide helpful insights or predictions in a non-intrusive way.

## Core Components

1. **AI Context Window**: Display a single-line context window above the Delta prompt showing Delta's "thoughts" about the current session
2. **Command History Analysis**: Process recent commands to understand user intent
3. **Predictive Assistance**: Suggest relevant commands or actions based on context
4. **Ollama Integration**: Use Llama3.3:8b for efficient local inference

## Implementation Plan

### 1. Ollama Client Integration

```go
// ai.go

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// OllamaClient provides an interface to the Ollama API
type OllamaClient struct {
	BaseURL    string
	ModelName  string
	MaxTokens  int
	HttpClient *http.Client
}

// OllamaRequest represents a request to the Ollama API
type OllamaRequest struct {
	Model     string   `json:"model"`
	Prompt    string   `json:"prompt"`
	System    string   `json:"system,omitempty"`
	Context   []int    `json:"context,omitempty"`
	Options   map[string]interface{} `json:"options,omitempty"`
	Stream    bool     `json:"stream"`
}

// OllamaResponse represents a response from the Ollama API
type OllamaResponse struct {
	Model     string   `json:"model"`
	Response  string   `json:"response"`
	Context   []int    `json:"context,omitempty"`
	Done      bool     `json:"done"`
}

// NewOllamaClient creates a new Ollama client
func NewOllamaClient(baseURL string, modelName string) *OllamaClient {
	return &OllamaClient{
		BaseURL:   baseURL,
		ModelName: modelName,
		MaxTokens: 2048,
		HttpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Generate sends a prompt to Ollama and returns the response
func (c *OllamaClient) Generate(prompt string, systemPrompt string) (string, error) {
	reqBody := OllamaRequest{
		Model:   c.ModelName,
		Prompt:  prompt,
		System:  systemPrompt,
		Stream:  false,
		Options: map[string]interface{}{
			"temperature": 0.1,
			"num_predict": 64,
		},
	}
	
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}
	
	resp, err := c.HttpClient.Post(
		c.BaseURL+"/api/generate",
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	var result OllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	
	// Remove extra whitespace and make sure the response is a single line
	response := strings.TrimSpace(result.Response)
	response = strings.ReplaceAll(response, "\n", " ")
	
	return response, nil
}
```

### 2. AI Prediction Manager

```go
// ai_manager.go

package main

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// AIPredictionManager manages the AI predictions and context
type AIPredictionManager struct {
	ollamaClient      *OllamaClient
	commandHistory    []string
	maxHistorySize    int
	currentThought    string
	lastCommandTime   time.Time
	processingLock    sync.Mutex
	contextPrompt     string
}

// NewAIPredictionManager creates a new AI prediction manager
func NewAIPredictionManager() (*AIPredictionManager, error) {
	client := NewOllamaClient("http://localhost:11434", "llama3.3:8b")
	
	return &AIPredictionManager{
		ollamaClient:    client,
		commandHistory:  []string{},
		maxHistorySize:  10,
		currentThought:  "",
		lastCommandTime: time.Now(),
		contextPrompt:   "You are Delta, an AI assistant for the command line. Analyze the user's commands and provide a short, helpful thought or prediction.",
	}, nil
}

// AddCommand adds a command to the history and updates predictions
func (m *AIPredictionManager) AddCommand(command string) {
	m.processingLock.Lock()
	defer m.processingLock.Unlock()
	
	// Add command to history
	m.commandHistory = append(m.commandHistory, command)
	if len(m.commandHistory) > m.maxHistorySize {
		m.commandHistory = m.commandHistory[1:]
	}
	
	m.lastCommandTime = time.Now()
	
	// Process in background to not block the UI
	go m.generateThought()
}

// GetCurrentThought returns the current AI thought
func (m *AIPredictionManager) GetCurrentThought() string {
	m.processingLock.Lock()
	defer m.processingLock.Unlock()
	return m.currentThought
}

// generateThought generates a new thought based on command history
func (m *AIPredictionManager) generateThought() {
	m.processingLock.Lock()
	history := strings.Join(m.commandHistory, "\n")
	m.processingLock.Unlock()
	
	prompt := fmt.Sprintf(
		"Here are my recent commands:\n%s\n\nBased on these commands, provide a single short sentence (max 60 chars) of what you think I might be working on or what might be helpful:",
		history,
	)
	
	thought, err := m.ollamaClient.Generate(prompt, m.contextPrompt)
	if err != nil {
		thought = ""
	}
	
	// Limit thought length and ensure it's a single line
	if len(thought) > 60 {
		thought = thought[:57] + "..."
	}
	
	m.processingLock.Lock()
	m.currentThought = thought
	m.processingLock.Unlock()
}
```

### 3. UI Integration

We will modify the main CLI interface to display the AI predictions:

```go
// Update main.go to integrate AI predictions

// Initialize AI prediction manager
aiManager, err := NewAIPredictionManager()
if err != nil {
    fmt.Println("Warning: AI features not available:", err)
}

// In the main loop, before showing prompt:
if aiManager != nil {
    thought := aiManager.GetCurrentThought()
    if thought != "" {
        // Display thought above prompt in a muted color
        fmt.Printf("\033[2m[∆ thinking: %s]\033[0m\n", thought)
    }
}

// After processing a command:
if aiManager != nil && command != "" {
    aiManager.AddCommand(command)
}
```

### 4. Ollama Model Management

Ensure the required Ollama model is downloaded and available:

```go
// Check if model is available and download if needed
func ensureModelAvailable(modelName string) error {
    client := &http.Client{Timeout: 30 * time.Second}
    
    // Check if model exists
    resp, err := client.Get(fmt.Sprintf("http://localhost:11434/api/tags"))
    if err != nil {
        return fmt.Errorf("could not connect to Ollama: %v", err)
    }
    defer resp.Body.Close()
    
    var result struct {
        Models []struct {
            Name string `json:"name"`
        } `json:"models"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return fmt.Errorf("error parsing Ollama response: %v", err)
    }
    
    // Check if our model exists
    modelExists := false
    for _, model := range result.Models {
        if model.Name == modelName {
            modelExists = true
            break
        }
    }
    
    if !modelExists {
        fmt.Println("Model not found. Downloading llama3.3:8b (this may take a while)...")
        
        // Request model download
        downloadReq := struct {
            Name string `json:"name"`
        }{Name: modelName}
        
        jsonData, err := json.Marshal(downloadReq)
        if err != nil {
            return err
        }
        
        _, err = client.Post(
            "http://localhost:11434/api/pull",
            "application/json",
            bytes.NewBuffer(jsonData),
        )
        if err != nil {
            return fmt.Errorf("error downloading model: %v", err)
        }
        
        fmt.Println("Model downloaded successfully!")
    }
    
    return nil
}
```

## Initialization Process

1. Start Delta CLI
2. Check if Ollama is running and if the model is available
3. If not available, prompt user to install/start Ollama or disable AI features
4. Initialize AI prediction manager with Llama3.3:8b model

## User Experience

The AI integration will:

- Show a single line of "thinking" above the Delta prompt
- Update contextually based on recent commands
- Provide insights in a non-intrusive way
- Assist with predicting user intent

## Performance Considerations

- Llama3.3:8b is chosen for a balance of capability and performance
- AI predictions run in a background goroutine to avoid blocking the UI
- Predictions are cached and updated only when context changes
- Ollama runs locally, ensuring privacy and low latency

## Future Enhancements

1. Command autocompletion based on AI predictions
2. Context-aware help and documentation
3. Detection of common errors and suggestions
4. Support for multiple AI models and configurations
5. Session-based learning to improve predictions over time

## Implementation Timeline

1. **Phase 1**: Basic Ollama integration and command history analysis
2. **Phase 2**: UI integration with thought display
3. **Phase 3**: Performance optimizations and user feedback
4. **Phase 4**: Advanced prediction features and command suggestions