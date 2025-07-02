package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// FeedbackType represents different types of user feedback
type FeedbackType string

const (
	FeedbackImplicit    FeedbackType = "implicit"    // Derived from user actions
	FeedbackExplicit    FeedbackType = "explicit"    // Direct user feedback
	FeedbackCorrection  FeedbackType = "correction"  // User corrected a prediction
	FeedbackRejection   FeedbackType = "rejection"   // User rejected a suggestion
	FeedbackAcceptance  FeedbackType = "acceptance"  // User accepted a suggestion
)

// FeedbackCollector manages collection of user feedback for learning
type FeedbackCollector struct {
	inferenceManager *InferenceManager
	learningEngine   *LearningEngine
	aiManager        *AIPredictionManager
	dataPath         string
	isEnabled        bool
	mutex            sync.RWMutex
	recentPredictions map[string]string // Maps commands to recent predictions
}

// NewFeedbackCollector creates a new feedback collector
func NewFeedbackCollector() (*FeedbackCollector, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}

	dataPath := filepath.Join(homeDir, ".config", "delta", "feedback")
	if err := os.MkdirAll(dataPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create feedback directory: %v", err)
	}

	collector := &FeedbackCollector{
		dataPath:          dataPath,
		isEnabled:         true,
		recentPredictions: make(map[string]string),
	}

	// Get managers (they might not be available yet)
	collector.inferenceManager = GetInferenceManager()
	collector.learningEngine = GetLearningEngine()
	collector.aiManager = GetAIManager()

	return collector, nil
}

// RecordPrediction records a prediction that was shown to the user
func (fc *FeedbackCollector) RecordPrediction(command string, prediction string) {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()

	// Store the prediction for this command
	fc.recentPredictions[command] = prediction

	// Keep only recent predictions (last 50)
	if len(fc.recentPredictions) > 50 {
		// Remove oldest entries
		for k := range fc.recentPredictions {
			delete(fc.recentPredictions, k)
			if len(fc.recentPredictions) <= 50 {
				break
			}
		}
	}
}

// CollectImplicitFeedback collects feedback based on user actions
func (fc *FeedbackCollector) CollectImplicitFeedback(command string, exitCode int, duration time.Duration) {
	if !fc.isEnabled {
		return
	}

	fc.mutex.RLock()
	prediction, hasPrediction := fc.recentPredictions[command]
	fc.mutex.RUnlock()

	if !hasPrediction {
		return // No prediction was made for this command
	}

	// Determine feedback based on exit code and duration
	var feedbackType string
	var label int

	if exitCode == 0 {
		// Command succeeded
		if duration < 5*time.Second {
			// Quick successful execution suggests good prediction
			feedbackType = "helpful"
			label = 1
		} else if duration < 30*time.Second {
			// Normal execution time
			feedbackType = "neutral"
			label = 0
		} else {
			// Long execution might indicate issues
			feedbackType = "unhelpful"
			label = -1
		}
	} else {
		// Command failed - prediction was likely unhelpful
		feedbackType = "unhelpful"
		label = -1
	}

	// Record the feedback
	if fc.inferenceManager != nil {
		context := fmt.Sprintf("exit_code:%d,duration:%.2fs", exitCode, duration.Seconds())
		err := fc.inferenceManager.AddFeedback(command, prediction, feedbackType, "", context)
		if err != nil {
			fmt.Printf("Failed to record implicit feedback: %v\n", err)
		}
	}

	// Update learning engine with the feedback
	if fc.learningEngine != nil {
		entry := CommandEntry{
			Command:   command,
			ExitCode:  exitCode,
			Duration:  int64(duration.Milliseconds()),
			Timestamp: time.Now(),
		}
		fc.learningEngine.LearnFromCommand(entry)
	}

	// Create training example
	fc.createTrainingExample(command, prediction, label, "implicit")
}

// CollectExplicitFeedback collects explicit feedback from user
func (fc *FeedbackCollector) CollectExplicitFeedback(command string, prediction string) {
	if !fc.isEnabled {
		return
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\nWas this prediction helpful? (y/n/c for correct)")
	fmt.Print("Feedback: ")

	response, err := reader.ReadString('\n')
	if err != nil {
		return
	}

	response = strings.TrimSpace(strings.ToLower(response))

	var feedbackType string
	var correction string
	var label int

	switch response {
	case "y", "yes":
		feedbackType = "helpful"
		label = 1
		fmt.Println("Thank you! Your feedback helps improve predictions.")
	case "n", "no":
		feedbackType = "unhelpful"
		label = -1
		fmt.Println("Thank you! What would have been better?")
		fmt.Print("Correction (or press Enter to skip): ")
		correction, _ = reader.ReadString('\n')
		correction = strings.TrimSpace(correction)
	case "c", "correct":
		// Ask for the correct command
		fmt.Print("Please enter the correct command: ")
		correction, _ = reader.ReadString('\n')
		correction = strings.TrimSpace(correction)
		if correction != "" {
			feedbackType = "correction"
			label = 0
			fmt.Println("Thank you! This correction will improve future predictions.")
		}
	default:
		return // Invalid response, skip feedback
	}

	// Record the feedback
	if fc.inferenceManager != nil {
		err := fc.inferenceManager.AddFeedback(command, prediction, feedbackType, correction, "explicit")
		if err != nil {
			fmt.Printf("Failed to record feedback: %v\n", err)
		}
	}

	// Create training example
	fc.createTrainingExample(command, prediction, label, "explicit")

	// If there's a correction, create a positive example for it
	if correction != "" {
		fc.createTrainingExample(command, correction, 1, "correction")
	}
}

// CollectInteractiveFeedback shows an interactive prompt for feedback
func (fc *FeedbackCollector) CollectInteractiveFeedback() {
	if !fc.isEnabled {
		fmt.Println("Feedback collection is disabled.")
		return
	}

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\n=== Interactive Feedback Collection ===")
	fmt.Println("Help improve Delta's predictions by providing feedback.")
	fmt.Println("Type 'done' when finished.\n")

	for {
		fmt.Print("Enter a command you ran: ")
		command, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		command = strings.TrimSpace(command)

		if command == "done" || command == "" {
			break
		}

		fmt.Print("What prediction did you expect? ")
		expectedPrediction, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		expectedPrediction = strings.TrimSpace(expectedPrediction)

		if expectedPrediction == "" {
			continue
		}

		// Record this as a positive training example
		fc.createTrainingExample(command, expectedPrediction, 1, "interactive")

		fmt.Println("✓ Feedback recorded. Thank you!")
	}

	fmt.Println("\nThank you for your feedback! It will help improve future predictions.")
}

// createTrainingExample creates a training example from feedback
func (fc *FeedbackCollector) createTrainingExample(command string, prediction string, label int, source string) {
	example := TrainingExample{
		Command:    command,
		Prediction: prediction,
		Label:      label,
		Weight:     1.0,
		Source:     source,
	}

	// Adjust weight based on source
	switch source {
	case "explicit":
		example.Weight = 1.5 // Explicit feedback is most valuable
	case "correction":
		example.Weight = 2.0 // Corrections are extremely valuable
	case "interactive":
		example.Weight = 1.3 // Interactive feedback is valuable
	case "implicit":
		example.Weight = 0.8 // Implicit feedback is less certain
	}

	// Save the training example
	timestamp := time.Now().UnixNano()
	filename := fmt.Sprintf("training_%d_%s.json", timestamp, source)
	filepath := filepath.Join(fc.dataPath, filename)

	data, err := json.MarshalIndent(example, "", "  ")
	if err != nil {
		return
	}

	os.WriteFile(filepath, data, 0644)
}

// GetFeedbackStats returns statistics about collected feedback
func (fc *FeedbackCollector) GetFeedbackStats() map[string]interface{} {
	stats := make(map[string]interface{})

	// Count feedback files by type
	files, err := os.ReadDir(fc.dataPath)
	if err != nil {
		stats["error"] = err.Error()
		return stats
	}

	typeCount := make(map[string]int)
	totalCount := 0

	for _, file := range files {
		if strings.HasPrefix(file.Name(), "training_") {
			// Extract source from filename
			parts := strings.Split(file.Name(), "_")
			if len(parts) >= 3 {
				source := strings.TrimSuffix(parts[2], ".json")
				typeCount[source]++
				totalCount++
			}
		}
	}

	stats["total_feedback"] = totalCount
	stats["by_type"] = typeCount
	stats["recent_predictions"] = len(fc.recentPredictions)

	return stats
}

// EnableFeedbackCollection enables or disables feedback collection
func (fc *FeedbackCollector) EnableFeedbackCollection(enable bool) {
	fc.mutex.Lock()
	defer fc.mutex.Unlock()
	fc.isEnabled = enable
}

// IsEnabled returns whether feedback collection is enabled
func (fc *FeedbackCollector) IsEnabled() bool {
	fc.mutex.RLock()
	defer fc.mutex.RUnlock()
	return fc.isEnabled
}

// Global feedback collector instance
var globalFeedbackCollector *FeedbackCollector
var feedbackCollectorOnce sync.Once

// GetFeedbackCollector returns the global feedback collector instance
func GetFeedbackCollector() *FeedbackCollector {
	feedbackCollectorOnce.Do(func() {
		var err error
		globalFeedbackCollector, err = NewFeedbackCollector()
		if err != nil {
			fmt.Printf("Warning: failed to initialize feedback collector: %v\n", err)
		}
	})
	return globalFeedbackCollector
}