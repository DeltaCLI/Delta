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
	isInitialized     bool
	waitGroup         sync.WaitGroup
	predictionEnabled bool
}

// NewAIPredictionManager creates a new AI prediction manager
func NewAIPredictionManager(ollamaURL string, modelName string) (*AIPredictionManager, error) {
	client := NewOllamaClient(ollamaURL, modelName)
	
	return &AIPredictionManager{
		ollamaClient:      client,
		commandHistory:    []string{},
		maxHistorySize:    10,
		currentThought:    "",
		lastCommandTime:   time.Now(),
		contextPrompt:     "You are Delta, an AI assistant for the command line. Analyze the user's commands and provide a short, helpful thought or prediction.",
		isInitialized:     false,
		predictionEnabled: false,
	}, nil
}

// Initialize initializes the AI manager and checks Ollama availability
func (m *AIPredictionManager) Initialize() bool {
	// Check if Ollama is available
	if !m.ollamaClient.IsAvailable() {
		fmt.Println("\033[2m[AI features disabled: Cannot connect to Ollama server]\033[0m")
		return false
	}
	
	// Check if the model is available
	available, err := m.ollamaClient.CheckModelAvailability()
	if err != nil {
		fmt.Println("\033[2m[AI features disabled: Error checking model availability]\033[0m")
		return false
	}
	
	if !available {
		fmt.Printf("\033[2m[AI features disabled: Model %s not available. Run Ollama and download the model first.]\033[0m\n", 
			m.ollamaClient.ModelName)
		return false
	}
	
	m.isInitialized = true
	m.predictionEnabled = true
	
	// Generate initial thought
	m.currentThought = "Ready to assist with commands and suggestions."
	
	return true
}

// IsEnabled returns whether AI predictions are enabled
func (m *AIPredictionManager) IsEnabled() bool {
	return m.isInitialized && m.predictionEnabled
}

// EnablePredictions enables AI predictions
func (m *AIPredictionManager) EnablePredictions() {
	m.predictionEnabled = true
}

// DisablePredictions disables AI predictions
func (m *AIPredictionManager) DisablePredictions() {
	m.predictionEnabled = false
}

// AddCommand adds a command to the history and updates predictions
func (m *AIPredictionManager) AddCommand(command string) {
	if !m.isInitialized || !m.predictionEnabled {
		return
	}
	
	m.processingLock.Lock()
	
	// Add command to history
	m.commandHistory = append(m.commandHistory, command)
	if len(m.commandHistory) > m.maxHistorySize {
		m.commandHistory = m.commandHistory[1:]
	}
	
	m.lastCommandTime = time.Now()
	m.processingLock.Unlock()
	
	// Process in background to not block the UI
	m.waitGroup.Add(1)
	go func() {
		defer m.waitGroup.Done()
		m.generateThought()
	}()
}

// GetCurrentThought returns the current AI thought
func (m *AIPredictionManager) GetCurrentThought() string {
	if !m.isInitialized || !m.predictionEnabled {
		return ""
	}
	
	m.processingLock.Lock()
	defer m.processingLock.Unlock()
	return m.currentThought
}

// Wait waits for background prediction tasks to complete
func (m *AIPredictionManager) Wait() {
	m.waitGroup.Wait()
}

// generateThought generates a new thought based on command history
func (m *AIPredictionManager) generateThought() {
	// Make a copy of history to avoid holding the lock
	m.processingLock.Lock()
	history := make([]string, len(m.commandHistory))
	copy(history, m.commandHistory)
	m.processingLock.Unlock()
	
	// Don't generate thoughts if history is empty
	if len(history) == 0 {
		return
	}
	
	// Format history into a prompt
	historyStr := strings.Join(history, "\n")
	
	prompt := fmt.Sprintf(
		"Here are my recent commands:\n%s\n\nProvide a single short sentence (max 60 chars) that summarizes what I might be working on or a helpful suggestion:",
		historyStr,
	)
	
	thought, err := m.ollamaClient.Generate(prompt, m.contextPrompt)
	if err != nil {
		// If we can't generate a thought, don't update the current one
		return
	}
	
	// Limit thought length and ensure it's a single line
	if len(thought) > 60 {
		thought = thought[:57] + "..."
	}
	
	m.processingLock.Lock()
	m.currentThought = thought
	m.processingLock.Unlock()
}

// DownloadModel initiates download of the Ollama model
func (m *AIPredictionManager) DownloadModel() error {
	// Check if Ollama is available
	if !m.ollamaClient.IsAvailable() {
		return fmt.Errorf("cannot connect to Ollama server")
	}
	
	// Check if the model is already available
	available, err := m.ollamaClient.CheckModelAvailability()
	if err != nil {
		return fmt.Errorf("error checking model availability: %v", err)
	}
	
	if available {
		return nil // Model already available
	}
	
	// Download the model
	fmt.Printf("Downloading model %s. This may take a while...\n", m.ollamaClient.ModelName)
	
	err = m.ollamaClient.DownloadModel()
	if err != nil {
		return fmt.Errorf("error downloading model: %v", err)
	}
	
	fmt.Println("Model downloaded successfully!")
	return nil
}