package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ART2Config contains configuration for the ART-2 algorithm
type ART2Config struct {
	Enabled        bool    `json:"enabled"`         // Whether ART-2 learning is enabled
	Alpha          float64 `json:"alpha"`           // Choice parameter (default: 0.5)
	Beta           float64 `json:"beta"`            // Learning rate (default: 0.5)
	Rho            float64 `json:"rho"`             // Vigilance parameter (default: 0.9)
	Theta          float64 `json:"theta"`           // Activity threshold (default: 0.1)
	MaxCategories  int     `json:"max_categories"`  // Maximum number of categories
	VectorSize     int     `json:"vector_size"`     // Size of input vectors
	DecayRate      float64 `json:"decay_rate"`      // Weight decay rate
	MinActivation  float64 `json:"min_activation"`  // Minimum activation threshold
	UpdateInterval int     `json:"update_interval"` // Update interval in seconds
}

// ART2Category represents a learned category in ART-2
type ART2Category struct {
	ID              int       `json:"id"`
	Weights         []float64 `json:"weights"`          // Bottom-up weights
	TopDownWeights  []float64 `json:"top_down_weights"` // Top-down weights
	ActivationCount int       `json:"activation_count"` // Number of times activated
	LastActivation  time.Time `json:"last_activation"`  // Last time this category was activated
	CommandPatterns []string  `json:"command_patterns"` // Associated command patterns
	ContextPatterns []string  `json:"context_patterns"` // Associated context patterns
	SuccessRate     float64   `json:"success_rate"`     // Success rate for predictions
	CreatedAt       time.Time `json:"created_at"`       // When the category was created
}

// ART2Input represents preprocessed input for ART-2
type ART2Input struct {
	Vector         []float64 `json:"vector"`
	Command        string    `json:"command"`
	Context        string    `json:"context"`
	Timestamp      time.Time `json:"timestamp"`
	UserFeedback   int       `json:"user_feedback"`   // 1: positive, 0: neutral, -1: negative
	PredictionMade string    `json:"prediction_made"` // What prediction was made
	ActualOutcome  string    `json:"actual_outcome"`  // What actually happened
}

// ART2Layer represents the processing layers in ART-2
type ART2Layer struct {
	F1Activations []float64 // F1 layer activations
	F2Activations []float64 // F2 layer activations
	Reset         []bool    // Reset signals
	Chosen        int       // Chosen category (-1 if none)
}

// ART2Manager implements the Adaptive Resonance Theory-2 algorithm
type ART2Manager struct {
	config         ART2Config
	categories     []*ART2Category
	configPath     string
	categoriesPath string
	inputBuffer    []ART2Input
	layer          ART2Layer
	mutex          sync.RWMutex
	isInitialized  bool
	lastUpdate     time.Time
	stats          ART2Stats
}

// ART2Stats tracks statistics for the ART-2 system
type ART2Stats struct {
	TotalInputs          int       `json:"total_inputs"`
	CategoriesLearned    int       `json:"categories_learned"`
	CorrectPredictions   int       `json:"correct_predictions"`
	IncorrectPredictions int       `json:"incorrect_predictions"`
	AccuracyRate         float64   `json:"accuracy_rate"`
	LastTrainingTime     time.Time `json:"last_training_time"`
	MemoryEfficiency     float64   `json:"memory_efficiency"`
}

// NewART2Manager creates a new ART-2 manager
func NewART2Manager() (*ART2Manager, error) {
	// Set up config directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = os.Getenv("HOME")
	}

	configDir := filepath.Join(homeDir, ".config", "delta", "memory", "art2")
	err = os.MkdirAll(configDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create ART-2 directory: %v", err)
	}

	configPath := filepath.Join(configDir, "art2_config.json")
	categoriesPath := filepath.Join(configDir, "categories.json")

	// Default configuration based on Carpenter & Grossberg ART-2
	defaultConfig := ART2Config{
		Enabled:        true,
		Alpha:          0.5,  // Choice parameter
		Beta:           0.5,  // Learning rate
		Rho:            0.9,  // Vigilance parameter (high for precise matching)
		Theta:          0.1,  // Activity threshold
		MaxCategories:  100,  // Reasonable limit for command patterns
		VectorSize:     128,  // Will be adjusted based on embedding size
		DecayRate:      0.01, // Slow decay to maintain long-term memory
		MinActivation:  0.05, // Minimum activation to consider
		UpdateInterval: 30,   // Update every 30 seconds
	}

	am := &ART2Manager{
		config:         defaultConfig,
		categories:     make([]*ART2Category, 0),
		configPath:     configPath,
		categoriesPath: categoriesPath,
		inputBuffer:    make([]ART2Input, 0),
		layer: ART2Layer{
			F1Activations: make([]float64, defaultConfig.VectorSize),
			F2Activations: make([]float64, defaultConfig.MaxCategories),
			Reset:         make([]bool, defaultConfig.MaxCategories),
			Chosen:        -1,
		},
		isInitialized: false,
		lastUpdate:    time.Now(),
		stats:         ART2Stats{},
	}

	// Try to load existing configuration and categories
	am.loadConfig()
	am.loadCategories()

	return am, nil
}

// Initialize initializes the ART-2 manager
func (am *ART2Manager) Initialize() error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	// Initialize F1 and F2 layers
	am.layer.F1Activations = make([]float64, am.config.VectorSize)
	am.layer.F2Activations = make([]float64, am.config.MaxCategories)
	am.layer.Reset = make([]bool, am.config.MaxCategories)

	// Initialize statistics
	am.updateStats()

	am.isInitialized = true
	return nil
}

// IsEnabled returns whether ART-2 processing is enabled
func (am *ART2Manager) IsEnabled() bool {
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	return am.isInitialized && am.config.Enabled
}

// ProcessInput processes a new input through the ART-2 algorithm
func (am *ART2Manager) ProcessInput(input ART2Input) (*ART2Category, bool, error) {
	if !am.IsEnabled() {
		return nil, false, nil
	}

	am.mutex.Lock()
	defer am.mutex.Unlock()

	// Normalize input vector
	am.normalizeVector(input.Vector)

	// Reset the network state
	am.resetNetwork()

	// Set F1 activations to the input
	copy(am.layer.F1Activations, input.Vector)

	// Find the best matching category
	bestCategory, isNewCategory := am.findBestMatch(input)

	// Update statistics
	am.stats.TotalInputs++

	// Store input for learning
	am.inputBuffer = append(am.inputBuffer, input)

	// Perform learning if we have feedback
	if input.UserFeedback != 0 {
		am.learnFromFeedback(input, bestCategory)
	}

	// Periodic updates
	if time.Since(am.lastUpdate).Seconds() > float64(am.config.UpdateInterval) {
		am.performPeriodicUpdate()
		am.lastUpdate = time.Now()
	}

	return bestCategory, isNewCategory, nil
}

// findBestMatch finds the best matching category using ART-2 dynamics
func (am *ART2Manager) findBestMatch(input ART2Input) (*ART2Category, bool) {
	maxActivation := 0.0
	bestCategoryIndex := -1

	// Calculate F2 activations for all categories
	for i, category := range am.categories {
		if len(category.Weights) != len(input.Vector) {
			continue // Skip if dimensions don't match
		}

		// Calculate bottom-up activation
		activation := am.calculateBottomUpActivation(input.Vector, category.Weights)

		// Apply choice function with alpha parameter
		choiceValue := activation / (am.config.Alpha + am.calculateNorm(category.Weights))

		am.layer.F2Activations[i] = choiceValue

		if choiceValue > maxActivation && choiceValue > am.config.MinActivation {
			maxActivation = choiceValue
			bestCategoryIndex = i
		}
	}

	// Check vigilance criterion if we found a match
	if bestCategoryIndex >= 0 {
		category := am.categories[bestCategoryIndex]

		// Calculate top-down activation
		topDownActivation := am.calculateTopDownActivation(input.Vector, category.TopDownWeights)

		// Calculate vigilance test
		vigilanceTest := topDownActivation / am.calculateNorm(input.Vector)

		if vigilanceTest >= am.config.Rho {
			// Accept the match and update the category
			am.updateCategory(category, input)
			am.layer.Chosen = bestCategoryIndex
			return category, false
		} else {
			// Reset this category and try the next best
			am.layer.Reset[bestCategoryIndex] = true
		}
	}

	// No suitable match found, create new category if possible
	if len(am.categories) < am.config.MaxCategories {
		newCategory := am.createNewCategory(input)
		am.categories = append(am.categories, newCategory)
		am.stats.CategoriesLearned++
		am.layer.Chosen = len(am.categories) - 1
		return newCategory, true
	}

	// Maximum categories reached, return nil
	return nil, false
}

// calculateBottomUpActivation calculates bottom-up activation
func (am *ART2Manager) calculateBottomUpActivation(input, weights []float64) float64 {
	activation := 0.0
	for i := 0; i < len(input) && i < len(weights); i++ {
		activation += input[i] * weights[i]
	}
	return activation
}

// calculateTopDownActivation calculates top-down activation
func (am *ART2Manager) calculateTopDownActivation(input, topDownWeights []float64) float64 {
	activation := 0.0
	for i := 0; i < len(input) && i < len(topDownWeights); i++ {
		activation += math.Min(input[i], topDownWeights[i])
	}
	return activation
}

// calculateNorm calculates the norm of a vector
func (am *ART2Manager) calculateNorm(vector []float64) float64 {
	sum := 0.0
	for _, v := range vector {
		sum += v * v
	}
	return math.Sqrt(sum)
}

// normalizeVector normalizes a vector to unit length
func (am *ART2Manager) normalizeVector(vector []float64) {
	norm := am.calculateNorm(vector)
	if norm > 0 {
		for i := range vector {
			vector[i] /= norm
		}
	}
}

// createNewCategory creates a new category from input
func (am *ART2Manager) createNewCategory(input ART2Input) *ART2Category {
	category := &ART2Category{
		ID:              len(am.categories),
		Weights:         make([]float64, len(input.Vector)),
		TopDownWeights:  make([]float64, len(input.Vector)),
		ActivationCount: 1,
		LastActivation:  time.Now(),
		CommandPatterns: []string{input.Command},
		ContextPatterns: []string{input.Context},
		SuccessRate:     0.5, // Start neutral
		CreatedAt:       time.Now(),
	}

	// Initialize weights with the input pattern
	copy(category.Weights, input.Vector)
	copy(category.TopDownWeights, input.Vector)

	return category
}

// updateCategory updates an existing category with new input
func (am *ART2Manager) updateCategory(category *ART2Category, input ART2Input) {
	// Update weights using ART-2 learning rule
	for i := 0; i < len(category.Weights) && i < len(input.Vector); i++ {
		// Bottom-up weights
		category.Weights[i] = (1-am.config.Beta)*category.Weights[i] +
			am.config.Beta*input.Vector[i]

		// Top-down weights
		category.TopDownWeights[i] = (1-am.config.Beta)*category.TopDownWeights[i] +
			am.config.Beta*math.Min(input.Vector[i], category.TopDownWeights[i])
	}

	// Update category metadata
	category.ActivationCount++
	category.LastActivation = time.Now()

	// Add command and context patterns if not already present
	if !art2Contains(category.CommandPatterns, input.Command) {
		category.CommandPatterns = append(category.CommandPatterns, input.Command)
	}
	if !art2Contains(category.ContextPatterns, input.Context) {
		category.ContextPatterns = append(category.ContextPatterns, input.Context)
	}
}

// learnFromFeedback updates categories based on user feedback
func (am *ART2Manager) learnFromFeedback(input ART2Input, category *ART2Category) {
	if category == nil {
		return
	}

	// Update success rate based on feedback
	totalFeedback := float64(category.ActivationCount)
	currentSuccess := category.SuccessRate * (totalFeedback - 1)

	if input.UserFeedback > 0 {
		currentSuccess += 1.0
		am.stats.CorrectPredictions++
	} else if input.UserFeedback < 0 {
		am.stats.IncorrectPredictions++
	}

	category.SuccessRate = currentSuccess / totalFeedback

	// Adjust learning rate based on feedback quality
	feedbackFactor := 1.0
	if input.UserFeedback > 0 {
		feedbackFactor = 1.2 // Increase learning for positive feedback
	} else if input.UserFeedback < 0 {
		feedbackFactor = 0.8 // Decrease learning for negative feedback
	}

	// Apply feedback-modulated learning
	adjustedBeta := am.config.Beta * feedbackFactor
	for i := 0; i < len(category.Weights) && i < len(input.Vector); i++ {
		category.Weights[i] = (1-adjustedBeta)*category.Weights[i] +
			adjustedBeta*input.Vector[i]
	}
}

// resetNetwork resets the network state for new input processing
func (am *ART2Manager) resetNetwork() {
	// Clear F2 activations
	for i := range am.layer.F2Activations {
		am.layer.F2Activations[i] = 0.0
	}

	// Clear reset signals
	for i := range am.layer.Reset {
		am.layer.Reset[i] = false
	}

	am.layer.Chosen = -1
}

// performPeriodicUpdate performs periodic maintenance and learning
func (am *ART2Manager) performPeriodicUpdate() {
	// Apply weight decay to prevent runaway growth
	for _, category := range am.categories {
		for i := range category.Weights {
			category.Weights[i] *= (1.0 - am.config.DecayRate)
			category.TopDownWeights[i] *= (1.0 - am.config.DecayRate)
		}
	}

	// Update statistics
	am.updateStats()

	// Save categories periodically
	am.saveCategories()
}

// updateStats updates internal statistics
func (am *ART2Manager) updateStats() {
	total := float64(am.stats.CorrectPredictions + am.stats.IncorrectPredictions)
	if total > 0 {
		am.stats.AccuracyRate = float64(am.stats.CorrectPredictions) / total
	}

	am.stats.CategoriesLearned = len(am.categories)
	am.stats.MemoryEfficiency = am.calculateMemoryEfficiency()
	am.stats.LastTrainingTime = time.Now()
}

// calculateMemoryEfficiency calculates how efficiently memory is being used
func (am *ART2Manager) calculateMemoryEfficiency() float64 {
	if len(am.categories) == 0 {
		return 0.0
	}

	activeCategoriesCount := 0
	for _, category := range am.categories {
		if category.ActivationCount > 1 {
			activeCategoriesCount++
		}
	}

	return float64(activeCategoriesCount) / float64(len(am.categories))
}

// GetPrediction returns a prediction based on current input
func (am *ART2Manager) GetPrediction(inputVector []float64, command, context string) (string, float64, error) {
	if !am.IsEnabled() {
		return "", 0.0, nil
	}

	input := ART2Input{
		Vector:    inputVector,
		Command:   command,
		Context:   context,
		Timestamp: time.Now(),
	}

	category, _, err := am.ProcessInput(input)
	if err != nil {
		return "", 0.0, err
	}

	if category == nil {
		return "", 0.0, nil
	}

	// Generate prediction based on category patterns
	prediction := am.generatePredictionFromCategory(category, command, context)
	confidence := category.SuccessRate

	return prediction, confidence, nil
}

// generatePredictionFromCategory generates a prediction from a category
func (am *ART2Manager) generatePredictionFromCategory(category *ART2Category, command, context string) string {
	// Simple pattern matching for now - in a full implementation this would be more sophisticated
	mostRelevantPattern := ""
	maxRelevance := 0.0

	for _, pattern := range category.CommandPatterns {
		relevance := am.calculatePatternRelevance(pattern, command)
		if relevance > maxRelevance {
			maxRelevance = relevance
			mostRelevantPattern = pattern
		}
	}

	if mostRelevantPattern != "" {
		return fmt.Sprintf("Based on pattern analysis: %s (confidence: %.2f)",
			mostRelevantPattern, category.SuccessRate)
	}

	return "No clear pattern detected"
}

// calculatePatternRelevance calculates how relevant a pattern is to current command
func (am *ART2Manager) calculatePatternRelevance(pattern, command string) float64 {
	// Simple word overlap calculation
	patternWords := strings.Fields(strings.ToLower(pattern))
	commandWords := strings.Fields(strings.ToLower(command))

	overlap := 0
	for _, pw := range patternWords {
		for _, cw := range commandWords {
			if pw == cw {
				overlap++
				break
			}
		}
	}

	if len(patternWords) == 0 {
		return 0.0
	}

	return float64(overlap) / float64(len(patternWords))
}

// GetStats returns current ART-2 statistics
func (am *ART2Manager) GetStats() ART2Stats {
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	return am.stats
}

// GetCategories returns information about learned categories
func (am *ART2Manager) GetCategories() []*ART2Category {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	// Return a copy to prevent external modification
	categories := make([]*ART2Category, len(am.categories))
	copy(categories, am.categories)
	return categories
}

// Helper functions for configuration management
func (am *ART2Manager) loadConfig() error {
	if !fileExists(am.configPath) {
		return am.saveConfig()
	}

	data, err := os.ReadFile(am.configPath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &am.config)
}

func (am *ART2Manager) saveConfig() error {
	data, err := json.MarshalIndent(am.config, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(am.configPath, data, 0644)
}

func (am *ART2Manager) loadCategories() error {
	if !fileExists(am.categoriesPath) {
		return nil // No categories file yet
	}

	data, err := os.ReadFile(am.categoriesPath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, &am.categories)
}

func (am *ART2Manager) saveCategories() error {
	data, err := json.MarshalIndent(am.categories, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(am.categoriesPath, data, 0644)
}

// Helper function to check if slice contains string
func art2Contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Global ART2Manager instance
var globalART2Manager *ART2Manager

// GetART2Manager returns the global ART2Manager instance
func GetART2Manager() *ART2Manager {
	if globalART2Manager == nil {
		var err error
		globalART2Manager, err = NewART2Manager()
		if err != nil {
			fmt.Printf("Error initializing ART-2 manager: %v\n", err)
			return nil
		}
		globalART2Manager.Initialize()
	}
	return globalART2Manager
}
