package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// AIPredictionConfig contains configuration for AI prediction system
type AIPredictionConfig struct {
	Enabled                   bool   `json:"enabled"`                      // Whether AI prediction is enabled
	ModelName                 string `json:"model_name"`                   // Model name to use (e.g., phi4:latest)
	ServerURL                 string `json:"server_url"`                   // Ollama server URL
	MaxHistory                int    `json:"max_history"`                  // Maximum history entries to keep
	ContextPrompt             string `json:"context_prompt"`               // Base prompt for AI
	HealthMonitorEnabled      bool   `json:"health_monitor_enabled"`       // Whether to monitor Ollama health
	HealthCheckInterval       int    `json:"health_check_interval"`        // Health check interval in seconds (default: 30)
	HealthNotificationEnabled bool   `json:"health_notification_enabled"`  // Whether to show notifications when Ollama becomes available
}

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
	errorPrinted      bool               // Track whether error messages have been printed
	cancelFunc        context.CancelFunc // Used to cancel pending requests
	lastPrediction    struct {           // Tracking for feedback
		command    string
		prediction string
		timestamp  time.Time
	}
	config        AIPredictionConfig   // Configuration for AI prediction
	healthMonitor *OllamaHealthMonitor // Health monitoring for Ollama
}

// NewAIPredictionManager creates a new AI prediction manager
func NewAIPredictionManager(ollamaURL string, modelName string) (*AIPredictionManager, error) {
	client := NewOllamaClient(ollamaURL, modelName)

	// Create a cancellable context
	_, cancel := context.WithCancel(context.Background())

	manager := &AIPredictionManager{
		ollamaClient:      client,
		commandHistory:    []string{},
		maxHistorySize:    10,
		currentThought:    "",
		lastCommandTime:   time.Now(),
		contextPrompt:     "You are Delta, an AI assistant for the command line. Analyze the user's commands and provide a short, helpful thought or prediction.",
		isInitialized:     false,
		predictionEnabled: false,
		cancelFunc:        cancel,
		config: AIPredictionConfig{
			Enabled:                   false,
			ModelName:                 modelName,
			ServerURL:                 ollamaURL,
			MaxHistory:                10,
			ContextPrompt:             "You are Delta, an AI assistant for the command line. Analyze the user's commands and provide a short, helpful thought or prediction.",
			HealthMonitorEnabled:      true,
			HealthCheckInterval:       30,
			HealthNotificationEnabled: true,
		},
	}

	// Create health monitor
	manager.healthMonitor = NewOllamaHealthMonitor(manager)

	return manager, nil
}

// Initialize initializes the AI manager and checks Ollama availability
func (m *AIPredictionManager) Initialize() bool {
	// Check if already initialized
	if m.isInitialized {
		return m.predictionEnabled
	}

	// Start health monitor if configured
	if m.healthMonitor != nil && m.config.HealthMonitorEnabled && !m.predictionEnabled {
		// Configure health monitor with settings
		if m.config.HealthCheckInterval > 0 {
			m.healthMonitor.SetCheckInterval(time.Duration(m.config.HealthCheckInterval) * time.Second)
		}
		m.healthMonitor.SetEnabled(m.config.HealthNotificationEnabled)
		m.healthMonitor.Start()
	}

	// Check if Ollama is available
	if !m.ollamaClient.IsAvailable() {
		if !m.errorPrinted {
			fmt.Println("\033[2m[AI features disabled: Cannot connect to Ollama server]\033[0m")
			m.errorPrinted = true
		}
		return false
	}

	// Check if the model is available
	available, err := m.ollamaClient.CheckModelAvailability()
	if err != nil {
		if !m.errorPrinted {
			fmt.Println("\033[2m[AI features disabled: Error checking model availability]\033[0m")
			m.errorPrinted = true
		}
		return false
	}

	if !available {
		if !m.errorPrinted {
			fmt.Printf("\033[2m[AI features disabled: Model %s not available. Run Ollama and download the model first.]\033[0m\n",
				m.ollamaClient.ModelName)
			m.errorPrinted = true
		}
		return false
	}

	// Check if we have a custom model from the inference system
	infMgr := GetInferenceManager()
	if infMgr != nil && infMgr.IsEnabled() && infMgr.learningConfig.UseCustomModel {
		// Try to initialize the inference manager to set up the custom model
		err := infMgr.Initialize()
		if err == nil && infMgr.inferenceConfig.UseLocalInference && infMgr.inferenceConfig.ModelPath != "" {
			fmt.Printf("\033[33m[Using custom trained model: %s]\033[0m\n",
				infMgr.inferenceConfig.ModelPath)
		}
	}

	m.isInitialized = true
	m.predictionEnabled = true

	// Stop health monitor when AI is enabled
	if m.healthMonitor != nil {
		m.healthMonitor.Stop()
	}

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
	m.config.Enabled = true
	
	// Stop health monitor when AI is enabled
	if m.healthMonitor != nil {
		m.healthMonitor.Stop()
	}
}

// DisablePredictions disables AI predictions
func (m *AIPredictionManager) DisablePredictions() {
	m.predictionEnabled = false
	m.config.Enabled = false
	
	// Start health monitor when AI is disabled
	if m.healthMonitor != nil {
		m.healthMonitor.Start()
	}
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

	// Add to memory system if available
	mm := GetMemoryManager()
	if mm != nil && mm.IsEnabled() {
		pwd, _ := os.Getwd()
		mm.AddCommand(command, pwd, 0, 0)
	}
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
	// Use WaitTimeout to prevent hanging on exit
	timeout := time.Millisecond * 100
	c := make(chan struct{})
	go func() {
		defer close(c)
		m.waitGroup.Wait()
	}()
	select {
	case <-c:
		// Completed normally
		return
	case <-time.After(timeout):
		// Timed out - continue anyway
		return
	}
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

	// Check for inference manager to use relevant feedback
	infMgr := GetInferenceManager()
	var enhancedPrompt string

	// Get current working directory for context
	pwd, _ := os.Getwd()

	// Get last command for feedback tracking
	lastCmd := ""
	if len(history) > 0 {
		lastCmd = history[len(history)-1]
	}

	if infMgr != nil && infMgr.IsEnabled() {
		// Get examples from inference manager if available
		examples, err := infMgr.GetTrainingExamples(5)

		if err == nil && len(examples) > 0 {
			// Incorporate examples into prompt
			examplesStr := ""
			for _, ex := range examples {
				if ex.Label > 0 { // Only include positive examples
					examplesStr += fmt.Sprintf("Command: %s\nGood Thought: %s\n\n", ex.Command, ex.Prediction)
				}
			}

			enhancedPrompt = fmt.Sprintf(
				"Here are my recent commands:\n%s\n\nCurrent directory: %s\n\nHere are some examples of helpful thoughts:\n%s\nProvide a helpful thought that summarizes what I might be working on or a relevant suggestion:",
				historyStr, pwd, examplesStr,
			)
		} else {
			// Use regular prompt if no examples
			enhancedPrompt = fmt.Sprintf(
				"Here are my recent commands:\n%s\n\nCurrent directory: %s\n\nProvide a helpful thought that summarizes what I might be working on or a relevant suggestion:",
				historyStr, pwd,
			)
		}
	} else {
		// Use regular prompt if inference manager is not available
		enhancedPrompt = fmt.Sprintf(
			"Here are my recent commands:\n%s\n\nCurrent directory: %s\n\nProvide a helpful thought that summarizes what I might be working on or a relevant suggestion:",
			historyStr, pwd,
		)
	}

	// Check if we should use a custom system prompt
	systemPrompt := m.contextPrompt
	if infMgr != nil && infMgr.IsEnabled() && infMgr.learningConfig.UseCustomModel {
		// Add additional context to system prompt
		systemPrompt += "\nProvide concise, specific, and relevant thoughts about what the user is working on based on their command history."

		// Check if inference system has custom instructions
		if len(infMgr.learningConfig.CustomModelPath) > 0 {
			systemPrompt += "\nPrefer brief, actionable insights over generic observations."
		}
	}

	// Try inferencing with custom model first if available
	var thought string
	var err error

	// First try using local inference if available
	if infMgr != nil && infMgr.IsEnabled() && infMgr.inferenceConfig.UseLocalInference {
		// TODO: Implement local inference using ONNX when available
		// This will be implemented in the future when the ONNX runtime is ready
	}

	// Fall back to Ollama if local inference is not available or fails
	thought, err = m.ollamaClient.Generate(enhancedPrompt, systemPrompt)
	if err != nil {
		// If we can't generate a thought, don't update the current one
		return
	}

	// Preserve newlines for properly formatting **text** sections
	thought = strings.TrimSpace(thought)

	// Store the prediction and command for feedback
	m.processingLock.Lock()
	m.currentThought = thought
	m.lastPrediction.command = lastCmd
	m.lastPrediction.prediction = thought
	m.lastPrediction.timestamp = time.Now()
	m.processingLock.Unlock()

	// Provide automatic feedback if enabled
	if infMgr != nil && infMgr.IsEnabled() && infMgr.learningConfig.AutomaticFeedback {
		// Automatic feedback criteria - add more patterns for better automatic detection
		isRelevant := false

		// Command-specific relevance checks
		if strings.Contains(lastCmd, "git") && strings.Contains(thought, "git") {
			isRelevant = true
		} else if strings.Contains(lastCmd, "make") && (strings.Contains(thought, "build") || strings.Contains(thought, "compile")) {
			isRelevant = true
		} else if strings.Contains(lastCmd, "npm") && strings.Contains(thought, "package") {
			isRelevant = true
		} else if strings.Contains(lastCmd, "docker") && strings.Contains(thought, "container") {
			isRelevant = true
		} else if strings.Contains(lastCmd, "test") && (strings.Contains(thought, "testing") || strings.Contains(thought, "tests")) {
			isRelevant = true
		} else if strings.Contains(lastCmd, "cd") && strings.Contains(thought, "directory") {
			isRelevant = true
		}

		// Directory-specific relevance
		if strings.Contains(pwd, "go") && strings.Contains(thought, "Go") {
			isRelevant = true
		} else if strings.Contains(pwd, "react") && strings.Contains(thought, "React") {
			isRelevant = true
		} else if strings.Contains(pwd, "python") && strings.Contains(thought, "Python") {
			isRelevant = true
		}

		// If relevant, add automatic positive feedback
		if isRelevant {
			infMgr.AddFeedback(lastCmd, thought, "helpful", "", pwd)
		}
	}

	// Check if we should trigger training based on accumulated feedback
	if infMgr != nil && infMgr.IsEnabled() && infMgr.learningConfig.PeriodicTraining {
		if infMgr.ShouldTrain() {
			fmt.Println("\033[33m[Delta AI training due - run ':memory train start' to improve predictions]\033[0m")
		}
	}
}

// GetLastPrediction returns the last command and prediction for feedback
func (m *AIPredictionManager) GetLastPrediction() (string, string, time.Time) {
	m.processingLock.Lock()
	defer m.processingLock.Unlock()

	return m.lastPrediction.command, m.lastPrediction.prediction, m.lastPrediction.timestamp
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

// UpdateModel switches to a different model or custom trained model
func (m *AIPredictionManager) UpdateModel(modelName string, customModel bool) error {
	if customModel {
		// Check if inference manager is available
		infMgr := GetInferenceManager()
		if infMgr == nil || !infMgr.IsEnabled() {
			return fmt.Errorf("inference system not available")
		}

		// Check if the custom model exists
		if !fileExists(modelName) {
			return fmt.Errorf("custom model not found: %s", modelName)
		}

		// Update inference config to use the custom model
		inferenceConfig := infMgr.inferenceConfig
		learningConfig := infMgr.learningConfig

		learningConfig.UseCustomModel = true
		learningConfig.CustomModelPath = modelName
		inferenceConfig.UseLocalInference = true
		inferenceConfig.ModelPath = modelName

		return infMgr.UpdateConfig(inferenceConfig, learningConfig)
	} else {
		// Update Ollama model
		m.ollamaClient.ModelName = modelName

		// Check if the model is available
		available, err := m.ollamaClient.CheckModelAvailability()
		if err != nil {
			return fmt.Errorf("error checking model availability: %v", err)
		}

		if !available {
			return fmt.Errorf("model %s not available - run 'ollama pull %s' first",
				modelName, modelName)
		}

		return nil
	}
}

// GenerateEmbedding generates an embedding for a text string using the embedding manager
func (m *AIPredictionManager) GetAIResponse(prompt string) (string, error) {
	// This is a simple wrapper around Ollama API - in a real implementation, this would
	// use the Ollama API to get a response
	if !m.IsEnabled() {
		return "", fmt.Errorf("AI prediction manager is not enabled")
	}

	// For now, just return a simple fixed response for error analysis
	// In a full implementation, this would call Ollama or another LLM service
	return "Try running 'make clean' followed by 'make build' to resolve the issue.", nil
}

func (m *AIPredictionManager) GenerateEmbedding(text string) ([]float32, error) {
	// Get the embedding manager
	em := GetEmbeddingManager()
	if em == nil {
		return nil, fmt.Errorf("embedding manager not available")
	}

	// Create an embedding request with just the text
	request := EmbeddingRequest{
		Command:   text,
		Directory: "",
		Context:   make(map[string]string),
		Timestamp: time.Now(),
	}

	// Generate the embedding
	response, err := em.GenerateEmbedding(request)
	if err != nil {
		return nil, err
	}

	return response.Embedding, nil
}

// Cleanup stops background tasks and health monitoring
func (m *AIPredictionManager) Cleanup() {
	// Stop health monitor
	if m.healthMonitor != nil {
		m.healthMonitor.Stop()
	}

	// Cancel any pending requests
	if m.cancelFunc != nil {
		m.cancelFunc()
	}

	// Wait for background tasks to complete
	m.Wait()
}
