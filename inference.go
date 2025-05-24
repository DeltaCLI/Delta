package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
)

// InferenceConfig contains configuration for model inference
type InferenceConfig struct {
	Enabled           bool    `json:"enabled"`            // Whether inference is enabled
	ModelPath         string  `json:"model_path"`         // Path to the model file
	ModelType         string  `json:"model_type"`         // "onnx" or "pytorch"
	MaxTokens         int     `json:"max_tokens"`         // Maximum number of tokens to generate
	Temperature       float64 `json:"temperature"`        // Sampling temperature (0.0-1.0)
	TopK              int     `json:"top_k"`              // Top-k sampling parameter
	TopP              float64 `json:"top_p"`              // Top-p (nucleus) sampling parameter
	UseSpeculative    bool    `json:"use_speculative"`    // Whether to use speculative decoding
	BatchSize         int     `json:"batch_size"`         // Batch size for inference
	UseOllama         bool    `json:"use_ollama"`         // Whether to use Ollama for inference
	OllamaURL         string  `json:"ollama_url"`         // URL for Ollama API
	UseLocalInference bool    `json:"use_local_inference"` // Whether to use local inference
}

// LearningConfig contains configuration for the learning system
type LearningConfig struct {
	Enabled                bool    `json:"enabled"`                  // Whether learning is enabled
	CollectFeedback        bool    `json:"collect_feedback"`         // Whether to collect user feedback
	AutomaticFeedback      bool    `json:"automatic_feedback"`       // Whether to use automatic feedback
	FeedbackThreshold      float64 `json:"feedback_threshold"`       // Threshold for automatic feedback
	AdaptationRate         float64 `json:"adaptation_rate"`          // Rate at which model adapts to feedback
	UseCustomModel         bool    `json:"use_custom_model"`         // Whether to use a custom-trained model
	CustomModelPath        string  `json:"custom_model_path"`        // Path to custom model
	PeriodicTraining       bool    `json:"periodic_training"`        // Whether to periodically train model
	TrainingInterval       int     `json:"training_interval"`        // Interval between training sessions (days)
	LastTrainingTimestamp  int64   `json:"last_training_timestamp"`  // Timestamp of last training
	AccumulatedTrainingExamples int `json:"accumulated_training_examples"` // Number of accumulated training examples
	TrainingThreshold      int     `json:"training_threshold"`      // Number of examples before training
}

// FeedbackEntry represents a user feedback entry
type FeedbackEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	Command     string    `json:"command"`
	Prediction  string    `json:"prediction"`
	FeedbackType string   `json:"feedback_type"` // "helpful", "unhelpful", or "correction"
	Correction  string    `json:"correction,omitempty"`
	UserContext string    `json:"user_context,omitempty"`
}

// TrainingExample represents a training example derived from feedback
type TrainingExample struct {
	Command    string `json:"command"`
	Context    string `json:"context,omitempty"`
	Prediction string `json:"prediction"`
	Label      int    `json:"label"` // 1: positive, 0: neutral, -1: negative
	Weight     float64 `json:"weight"`
	Source     string `json:"source"` // "feedback", "automatic", "synthetic"
}

// InferenceManager handles model inference and learning
type InferenceManager struct {
	inferenceConfig InferenceConfig
	learningConfig  LearningConfig
	configPath      string
	feedbackPath    string
	trainingPath    string
	mutex           sync.RWMutex
	httpClient      *resty.Client
	isInitialized   bool
}

// NewInferenceManager creates a new inference manager
func NewInferenceManager() (*InferenceManager, error) {
	// Set up config directory in user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = os.Getenv("HOME")
	}

	// Use ~/.config/delta/memory/inference directory
	configDir := filepath.Join(homeDir, ".config", "delta", "memory", "inference")
	err = os.MkdirAll(configDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create inference directory: %v", err)
	}

	// Set up paths
	configPath := filepath.Join(configDir, "inference_config.json")
	feedbackPath := filepath.Join(configDir, "feedback")
	trainingPath := filepath.Join(configDir, "training_examples")

	// Create directories
	os.MkdirAll(feedbackPath, 0755)
	os.MkdirAll(trainingPath, 0755)

	// Create an HTTP client for Ollama API
	client := resty.New()
	client.SetTimeout(10 * time.Second)

	// Create inference manager
	im := &InferenceManager{
		inferenceConfig: InferenceConfig{
			ModelPath:         "",
			ModelType:         "onnx",
			MaxTokens:         100,
			Temperature:       0.7,
			TopK:              40,
			TopP:              0.9,
			UseSpeculative:    true,
			BatchSize:         1,
			UseOllama:         true,
			OllamaURL:         "http://localhost:11434",
			UseLocalInference: false,
		},
		learningConfig: LearningConfig{
			Enabled:                true,
			CollectFeedback:        true,
			AutomaticFeedback:      true,
			FeedbackThreshold:      0.8,
			AdaptationRate:         0.1,
			UseCustomModel:         false,
			CustomModelPath:        "",
			PeriodicTraining:       true,
			TrainingInterval:       7, // 7 days
			LastTrainingTimestamp:  0,
			AccumulatedTrainingExamples: 0,
		},
		configPath:    configPath,
		feedbackPath:  feedbackPath,
		trainingPath:  trainingPath,
		httpClient:    client,
		isInitialized: false,
	}

	// Try to load configuration
	err = im.loadConfig()
	if err != nil {
		// Save the default configuration if loading fails
		im.saveConfig()
	}

	return im, nil
}

// Initialize initializes the inference manager
func (im *InferenceManager) Initialize() error {
	// Check for custom model
	if im.learningConfig.UseCustomModel && im.learningConfig.CustomModelPath != "" {
		// Validate custom model path
		modelPath := im.learningConfig.CustomModelPath
		if !filepath.IsAbs(modelPath) {
			// Convert relative to absolute path
			homeDir, _ := os.UserHomeDir()
			modelPath = filepath.Join(homeDir, ".config", "delta", "memory", "models", modelPath)
		}

		// Check if model exists
		if _, err := os.Stat(modelPath); err == nil {
			im.inferenceConfig.ModelPath = modelPath
			im.inferenceConfig.UseLocalInference = true
		} else {
			// Fall back to default
			im.learningConfig.UseCustomModel = false
			im.inferenceConfig.UseLocalInference = false
		}
	}

	// Check for Ollama availability if using it
	if im.inferenceConfig.UseOllama {
		url := im.inferenceConfig.OllamaURL + "/api/tags"
		resp, err := im.httpClient.R().Get(url)
		if err != nil || resp.StatusCode() != 200 {
			// Ollama not available, disable it
			im.inferenceConfig.UseOllama = false
		}
	}

	im.isInitialized = true
	return nil
}

// loadConfig loads the inference configuration from disk
func (im *InferenceManager) loadConfig() error {
	// Check if config file exists
	_, err := os.Stat(im.configPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("config file does not exist")
	}

	// Read the config file
	data, err := os.ReadFile(im.configPath)
	if err != nil {
		return err
	}

	// Parse the JSON
	var config struct {
		Inference InferenceConfig `json:"inference"`
		Learning  LearningConfig  `json:"learning"`
	}

	err = json.Unmarshal(data, &config)
	if err != nil {
		return err
	}

	// Update configurations
	im.mutex.Lock()
	im.inferenceConfig = config.Inference
	im.learningConfig = config.Learning
	im.mutex.Unlock()

	return nil
}

// saveConfig saves the inference configuration to disk
func (im *InferenceManager) saveConfig() error {
	// Create config object
	im.mutex.RLock()
	config := struct {
		Inference InferenceConfig `json:"inference"`
		Learning  LearningConfig  `json:"learning"`
	}{
		Inference: im.inferenceConfig,
		Learning:  im.learningConfig,
	}
	im.mutex.RUnlock()

	// Marshal to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(im.configPath, data, 0644)
}

// IsEnabled returns whether the learning system is enabled
func (im *InferenceManager) IsEnabled() bool {
	im.mutex.RLock()
	defer im.mutex.RUnlock()
	return im.isInitialized && im.learningConfig.Enabled
}

// EnableLearning enables the learning system
func (im *InferenceManager) EnableLearning() error {
	im.mutex.Lock()
	im.learningConfig.Enabled = true
	im.mutex.Unlock()
	
	// Save local config
	if err := im.saveConfig(); err != nil {
		return err
	}
	
	// Update ConfigManager
	cm := GetConfigManager()
	if cm != nil {
		cm.UpdateLearningConfig(&im.learningConfig)
	}
	
	return nil
}

// DisableLearning disables the learning system
func (im *InferenceManager) DisableLearning() error {
	im.mutex.Lock()
	im.learningConfig.Enabled = false
	im.mutex.Unlock()
	
	// Save local config
	if err := im.saveConfig(); err != nil {
		return err
	}
	
	// Update ConfigManager
	cm := GetConfigManager()
	if cm != nil {
		cm.UpdateLearningConfig(&im.learningConfig)
	}
	
	return nil
}

// AddFeedback adds user feedback for a prediction
func (im *InferenceManager) AddFeedback(command, prediction, feedbackType, correction, context string) error {
	if !im.IsEnabled() || !im.learningConfig.CollectFeedback {
		return nil
	}

	// Create feedback entry
	feedback := FeedbackEntry{
		Timestamp:    time.Now(),
		Command:      command,
		Prediction:   prediction,
		FeedbackType: feedbackType,
		Correction:   correction,
		UserContext:  context,
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(feedback, "", "  ")
	if err != nil {
		return err
	}

	// Create filename based on timestamp
	filename := fmt.Sprintf("feedback_%d.json", time.Now().UnixNano())
	filepath := filepath.Join(im.feedbackPath, filename)

	// Write to file
	err = os.WriteFile(filepath, data, 0644)
	if err != nil {
		return err
	}

	// Create training example from feedback
	if err = im.createTrainingExample(feedback); err != nil {
		return err
	}

	// Increment accumulated training examples
	im.mutex.Lock()
	im.learningConfig.AccumulatedTrainingExamples++
	im.mutex.Unlock()

	// Save config
	return im.saveConfig()
}

// createTrainingExample creates a training example from feedback
func (im *InferenceManager) createTrainingExample(feedback FeedbackEntry) error {
	// Create a new training example
	var label int
	var weight float64

	// Determine label and weight based on feedback type
	switch feedback.FeedbackType {
	case "helpful":
		label = 1
		weight = 1.0
	case "unhelpful":
		label = -1
		weight = 0.8
	case "correction":
		label = 0
		weight = 0.5
	default:
		label = 0
		weight = 0.3
	}

	// Create the training example
	example := TrainingExample{
		Command:    feedback.Command,
		Context:    feedback.UserContext,
		Prediction: feedback.Prediction,
		Label:      label,
		Weight:     weight,
		Source:     "feedback",
	}

	// If there's a correction, create another example
	if feedback.FeedbackType == "correction" && feedback.Correction != "" {
		correctionExample := TrainingExample{
			Command:    feedback.Command,
			Context:    feedback.UserContext,
			Prediction: feedback.Correction,
			Label:      1,
			Weight:     1.0,
			Source:     "correction",
		}

		// Save the correction example
		if err := im.saveTrainingExample(correctionExample); err != nil {
			return err
		}
	}

	// Save the example
	return im.saveTrainingExample(example)
}

// saveTrainingExample saves a training example to disk
func (im *InferenceManager) saveTrainingExample(example TrainingExample) error {
	// Marshal to JSON
	data, err := json.MarshalIndent(example, "", "  ")
	if err != nil {
		return err
	}

	// Create filename based on timestamp and hash of command
	hash := hashString(example.Command)
	filename := fmt.Sprintf("example_%d_%d.json", time.Now().UnixNano(), hash)
	filepath := filepath.Join(im.trainingPath, filename)

	// Write to file
	return os.WriteFile(filepath, data, 0644)
}

// GetFeedbacks returns feedback entries for a given time range
func (im *InferenceManager) GetFeedbacks(startTime, endTime time.Time) ([]FeedbackEntry, error) {
	var feedbacks []FeedbackEntry

	// Get all feedback files
	files, err := os.ReadDir(im.feedbackPath)
	if err != nil {
		return nil, err
	}

	// Process each file
	for _, file := range files {
		if file.IsDir() || !strings.HasPrefix(file.Name(), "feedback_") {
			continue
		}

		// Read the file
		data, err := os.ReadFile(filepath.Join(im.feedbackPath, file.Name()))
		if err != nil {
			continue
		}

		// Parse the feedback
		var feedback FeedbackEntry
		if err := json.Unmarshal(data, &feedback); err != nil {
			continue
		}

		// Check if within time range
		if (startTime.IsZero() || feedback.Timestamp.After(startTime)) &&
			(endTime.IsZero() || feedback.Timestamp.Before(endTime)) {
			feedbacks = append(feedbacks, feedback)
		}
	}

	return feedbacks, nil
}

// GetTrainingExamples returns training examples
func (im *InferenceManager) GetTrainingExamples(limit int) ([]TrainingExample, error) {
	var examples []TrainingExample

	// Get all example files
	files, err := os.ReadDir(im.trainingPath)
	if err != nil {
		return nil, err
	}

	// Process each file
	for _, file := range files {
		if file.IsDir() || !strings.HasPrefix(file.Name(), "example_") {
			continue
		}

		// Read the file
		data, err := os.ReadFile(filepath.Join(im.trainingPath, file.Name()))
		if err != nil {
			continue
		}

		// Parse the example
		var example TrainingExample
		if err := json.Unmarshal(data, &example); err != nil {
			continue
		}

		examples = append(examples, example)

		// Check if we've reached the limit
		if limit > 0 && len(examples) >= limit {
			break
		}
	}

	return examples, nil
}

// GetInferenceStats returns statistics about the inference system
func (im *InferenceManager) GetInferenceStats() map[string]interface{} {
	im.mutex.RLock()
	defer im.mutex.RUnlock()

	// Count feedback files
	feedbackCount := 0
	if files, err := os.ReadDir(im.feedbackPath); err == nil {
		for _, file := range files {
			if !file.IsDir() && strings.HasPrefix(file.Name(), "feedback_") {
				feedbackCount++
			}
		}
	}

	// Count training examples
	exampleCount := 0
	if files, err := os.ReadDir(im.trainingPath); err == nil {
		for _, file := range files {
			if !file.IsDir() && strings.HasPrefix(file.Name(), "example_") {
				exampleCount++
			}
		}
	}

	// Determine model status
	customModelAvailable := im.learningConfig.UseCustomModel && fileExists(im.learningConfig.CustomModelPath)
	modelPath := im.inferenceConfig.ModelPath
	if modelPath == "" {
		modelPath = "Not specified"
	}

	// Calculate time since last training
	timeSinceTraining := "Never trained"
	if im.learningConfig.LastTrainingTimestamp > 0 {
		lastTrainingTime := time.Unix(im.learningConfig.LastTrainingTimestamp, 0)
		timeSinceTraining = formatInferenceDuration(time.Since(lastTrainingTime))
	}

	// Return stats
	return map[string]interface{}{
		"learning_enabled":        im.learningConfig.Enabled,
		"feedback_collection":     im.learningConfig.CollectFeedback,
		"automatic_feedback":      im.learningConfig.AutomaticFeedback,
		"feedback_count":          feedbackCount,
		"training_examples":       exampleCount,
		"accumulated_examples":    im.learningConfig.AccumulatedTrainingExamples,
		"custom_model_enabled":    im.learningConfig.UseCustomModel,
		"custom_model_available":  customModelAvailable,
		"model_path":              modelPath,
		"periodic_training":       im.learningConfig.PeriodicTraining,
		"training_interval_days":  im.learningConfig.TrainingInterval,
		"last_training":           timeSinceTraining,
		"ollama_enabled":          im.inferenceConfig.UseOllama,
		"local_inference_enabled": im.inferenceConfig.UseLocalInference,
	}
}

// UpdateConfig updates the inference configuration
func (im *InferenceManager) UpdateConfig(inference InferenceConfig, learning LearningConfig) error {
	im.mutex.Lock()
	im.inferenceConfig = inference
	im.learningConfig = learning
	im.mutex.Unlock()
	return im.saveConfig()
}

// ShouldTrain checks if training should be triggered
func (im *InferenceManager) ShouldTrain() bool {
	if !im.IsEnabled() || !im.learningConfig.PeriodicTraining {
		return false
	}

	// Check if we have enough new examples
	if im.learningConfig.AccumulatedTrainingExamples < 100 {
		return false
	}

	// Check if enough time has passed since last training
	if im.learningConfig.LastTrainingTimestamp > 0 {
		lastTrainingTime := time.Unix(im.learningConfig.LastTrainingTimestamp, 0)
		daysSinceTraining := int(time.Since(lastTrainingTime).Hours() / 24)
		return daysSinceTraining >= im.learningConfig.TrainingInterval
	}

	// If never trained, check if we have enough examples
	return im.learningConfig.AccumulatedTrainingExamples >= 500
}

// RecordTrainingCompletion updates the last training timestamp
func (im *InferenceManager) RecordTrainingCompletion() error {
	im.mutex.Lock()
	im.learningConfig.LastTrainingTimestamp = time.Now().Unix()
	im.learningConfig.AccumulatedTrainingExamples = 0
	im.mutex.Unlock()
	return im.saveConfig()
}

// Helper functions

// fileExists checks if a file exists and is accessible
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir() // Ensure it's a file, not a directory
}

// hashString creates a simple hash of a string
func hashString(s string) uint32 {
	var h uint32
	for i := 0; i < len(s); i++ {
		h = 31*h + uint32(s[i])
	}
	return h
}

// formatInferenceDuration formats a duration in a user-friendly way
func formatInferenceDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%d days, %d hours", days, hours)
	} else if hours > 0 {
		return fmt.Sprintf("%d hours, %d minutes", hours, minutes)
	} else {
		return fmt.Sprintf("%d minutes", minutes)
	}
}

// Global InferenceManager instance
var globalInferenceManager *InferenceManager

// GetInferenceManager returns the global InferenceManager instance
func GetInferenceManager() *InferenceManager {
	if globalInferenceManager == nil {
		var err error
		globalInferenceManager, err = NewInferenceManager()
		if err != nil {
			fmt.Printf("Error initializing inference manager: %v\n", err)
			return nil
		}
		globalInferenceManager.Initialize()
	}
	return globalInferenceManager
}