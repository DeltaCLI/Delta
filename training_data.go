package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// TrainingDataFormat specifies the format for extracted training data
type TrainingDataFormat string

const (
	// FormatJSON outputs JSON-formatted training data
	FormatJSON TrainingDataFormat = "json"
	// FormatCSV outputs CSV-formatted training data
	FormatCSV TrainingDataFormat = "csv"
	// FormatTFRecord outputs TFRecord-formatted training data
	FormatTFRecord TrainingDataFormat = "tfrecord"
)

// TrainingDataOptions defines options for data extraction
type TrainingDataOptions struct {
	Format          TrainingDataFormat // Output format
	StartDate       time.Time          // Start date for data range
	EndDate         time.Time          // End date for data range
	OutputDir       string             // Directory for output files
	IncludeMetadata bool               // Whether to include metadata
	MaxExamples     int                // Maximum number of examples (-1 for all)
	FilterTypes     []string           // Filter by feedback types
	SplitRatio      float64            // Train/validation split ratio (0.0-1.0)
	BalanceClasses  bool               // Whether to balance positive/negative examples
	AugmentData     bool               // Whether to augment data with synthetic examples
}

// TrainingExample defines a single training example with metadata
type TrainingExtendedExample struct {
	Command        string            `json:"command"`
	Context        string            `json:"context,omitempty"`
	Prediction     string            `json:"prediction"`
	Label          int               `json:"label"`
	Weight         float64           `json:"weight"`
	Source         string            `json:"source"`
	Timestamp      time.Time         `json:"timestamp"`
	FeedbackType   string            `json:"feedback_type,omitempty"`
	Environment    map[string]string `json:"environment,omitempty"`
	Directory      string            `json:"directory,omitempty"`
	CommandHistory []string          `json:"command_history,omitempty"`
}

// TrainingDataService handles training data extraction and processing
type TrainingDataService struct {
	inferenceManager *InferenceManager
	memoryManager    *MemoryManager
	aiManager        *AIPredictionManager
}

// NewTrainingDataService creates a new training data service
func NewTrainingDataService() (*TrainingDataService, error) {
	inferenceManager := GetInferenceManager()
	if inferenceManager == nil {
		return nil, fmt.Errorf("inference manager not available")
	}

	memoryManager := GetMemoryManager()
	if memoryManager == nil {
		return nil, fmt.Errorf("memory manager not available")
	}

	aiManager := GetAIManager()
	if aiManager == nil {
		return nil, fmt.Errorf("AI manager not available")
	}

	return &TrainingDataService{
		inferenceManager: inferenceManager,
		memoryManager:    memoryManager,
		aiManager:        aiManager,
	}, nil
}

// ExtractTrainingData extracts and processes training data
func (s *TrainingDataService) ExtractTrainingData(options TrainingDataOptions) (string, error) {
	// Create output directory if it doesn't exist
	if options.OutputDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %v", err)
		}
		options.OutputDir = filepath.Join(homeDir, ".config", "delta", "memory", "training_data")
	}

	err := os.MkdirAll(options.OutputDir, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create output directory: %v", err)
	}

	// Get training examples from inference manager
	examples, err := s.inferenceManager.GetTrainingExamples(-1) // Get all examples
	if err != nil {
		return "", fmt.Errorf("failed to get training examples: %v", err)
	}

	if len(examples) == 0 {
		return "", fmt.Errorf("no training examples found")
	}

	// Filter examples by date range if specified
	if !options.StartDate.IsZero() || !options.EndDate.IsZero() {
		// Get feedbacks for date filtering
		var startTime, endTime time.Time
		if !options.StartDate.IsZero() {
			startTime = options.StartDate
		}
		if !options.EndDate.IsZero() {
			endTime = options.EndDate.Add(24 * time.Hour) // Include the full end day
		}

		feedbacks, err := s.inferenceManager.GetFeedbacks(startTime, endTime)
		if err != nil {
			return "", fmt.Errorf("failed to get feedbacks: %v", err)
		}

		// Build a map of command -> prediction -> timestamp for filtering
		feedbackMap := make(map[string]map[string]time.Time)
		for _, fb := range feedbacks {
			if _, ok := feedbackMap[fb.Command]; !ok {
				feedbackMap[fb.Command] = make(map[string]time.Time)
			}
			feedbackMap[fb.Command][fb.Prediction] = fb.Timestamp
		}

		// Filter examples by timestamp
		filteredExamples := make([]TrainingExample, 0)
		for _, ex := range examples {
			if timestamps, ok := feedbackMap[ex.Command]; ok {
				if timestamp, ok := timestamps[ex.Prediction]; ok {
					// Check date range
					if (!options.StartDate.IsZero() && timestamp.Before(options.StartDate)) ||
						(!options.EndDate.IsZero() && timestamp.After(options.EndDate)) {
						continue
					}
					filteredExamples = append(filteredExamples, ex)
				}
			}
		}
		examples = filteredExamples
	}

	// Filter by feedback type if specified
	if len(options.FilterTypes) > 0 {
		filteredExamples := make([]TrainingExample, 0)
		for _, ex := range examples {
			// Map label to feedback type
			var feedbackType string
			switch ex.Label {
			case 1:
				feedbackType = "helpful"
			case -1:
				feedbackType = "unhelpful"
			case 0:
				feedbackType = "correction"
			}

			// Check if feedback type matches any filter
			for _, filter := range options.FilterTypes {
				if feedbackType == filter {
					filteredExamples = append(filteredExamples, ex)
					break
				}
			}
		}
		examples = filteredExamples
	}

	// Limit number of examples if specified
	if options.MaxExamples > 0 && len(examples) > options.MaxExamples {
		examples = examples[:options.MaxExamples]
	}

	// Balance classes if requested
	if options.BalanceClasses {
		examples = s.balanceExamples(examples)
	}

	// Augment data if requested
	if options.AugmentData {
		examples = s.augmentData(examples)
	}

	// Enhance examples with metadata if requested
	enhancedExamples := make([]TrainingExtendedExample, 0)
	if options.IncludeMetadata {
		// Get command history from memory manager
		allCommands := make(map[string][]CommandEntry)

		// Get last 30 days of commands
		thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
		for d := thirtyDaysAgo; d.Before(time.Now()); d = d.AddDate(0, 0, 1) {
			date := d.Format("2006-01-02")
			commands, err := s.memoryManager.ReadCommands(date)
			if err == nil && len(commands) > 0 {
				for _, cmd := range commands {
					dir := cmd.Directory
					if _, ok := allCommands[dir]; !ok {
						allCommands[dir] = make([]CommandEntry, 0)
					}
					allCommands[dir] = append(allCommands[dir], cmd)
				}
			}
		}

		// Enhance each example with metadata
		for _, ex := range examples {
			enhanced := TrainingExtendedExample{
				Command:    ex.Command,
				Context:    ex.Context,
				Prediction: ex.Prediction,
				Label:      ex.Label,
				Weight:     ex.Weight,
				Source:     ex.Source,
				Timestamp:  time.Now(), // Default to current time if not found
			}

			// Set feedback type based on label
			switch ex.Label {
			case 1:
				enhanced.FeedbackType = "helpful"
			case -1:
				enhanced.FeedbackType = "unhelpful"
			case 0:
				enhanced.FeedbackType = "correction"
			}

			// Set directory from context if available
			if ex.Context != "" {
				enhanced.Directory = ex.Context
			}

			// Find command history and environment if available
			if enhanced.Directory != "" {
				if cmds, ok := allCommands[enhanced.Directory]; ok {
					// Find the command in history
					for i, cmd := range cmds {
						if cmd.Command == ex.Command {
							// Set timestamp from command entry
							enhanced.Timestamp = cmd.Timestamp

							// Get environment if available
							if len(cmd.Environment) > 0 {
								enhanced.Environment = cmd.Environment
							}

							// Get command history (up to 5 previous commands)
							history := make([]string, 0)
							start := i - 5
							if start < 0 {
								start = 0
							}
							for j := start; j < i; j++ {
								history = append(history, cmds[j].Command)
							}
							enhanced.CommandHistory = history

							break
						}
					}
				}
			}

			enhancedExamples = append(enhancedExamples, enhanced)
		}
	} else {
		// Just convert to extended format without extra metadata
		for _, ex := range examples {
			enhancedExamples = append(enhancedExamples, TrainingExtendedExample{
				Command:    ex.Command,
				Context:    ex.Context,
				Prediction: ex.Prediction,
				Label:      ex.Label,
				Weight:     ex.Weight,
				Source:     ex.Source,
				Timestamp:  time.Now(),
			})
		}
	}

	// Create train/validation split if ratio specified
	var trainExamples, valExamples []TrainingExtendedExample
	if options.SplitRatio > 0 && options.SplitRatio < 1.0 {
		splitIndex := int(float64(len(enhancedExamples)) * options.SplitRatio)
		trainExamples = enhancedExamples[:splitIndex]
		valExamples = enhancedExamples[splitIndex:]
	} else {
		trainExamples = enhancedExamples
		valExamples = nil
	}

	// Generate output based on specified format
	outputPath := ""
	switch options.Format {
	case FormatJSON:
		outputPath, err = s.writeJSONOutput(options.OutputDir, trainExamples, valExamples)
		if err != nil {
			return "", fmt.Errorf("failed to write JSON output: %v", err)
		}
	case FormatCSV:
		outputPath, err = s.writeCSVOutput(options.OutputDir, trainExamples, valExamples)
		if err != nil {
			return "", fmt.Errorf("failed to write CSV output: %v", err)
		}
	case FormatTFRecord:
		outputPath, err = s.writeTFRecordOutput(options.OutputDir, trainExamples, valExamples)
		if err != nil {
			return "", fmt.Errorf("failed to write TFRecord output: %v", err)
		}
	default:
		return "", fmt.Errorf("unsupported format: %s", options.Format)
	}

	return outputPath, nil
}

// writeJSONOutput writes training data to JSON files
func (s *TrainingDataService) writeJSONOutput(outputDir string,
	trainExamples, valExamples []TrainingExtendedExample) (string, error) {

	// Create timestamp for filenames
	timestamp := time.Now().Format("20060102_150405")

	// Write training data
	trainPath := filepath.Join(outputDir, fmt.Sprintf("train_data_%s.json", timestamp))
	trainData, err := json.MarshalIndent(trainExamples, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal training data: %v", err)
	}

	err = os.WriteFile(trainPath, trainData, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write training data: %v", err)
	}

	// Write validation data if available
	if valExamples != nil {
		valPath := filepath.Join(outputDir, fmt.Sprintf("val_data_%s.json", timestamp))
		valData, err := json.MarshalIndent(valExamples, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal validation data: %v", err)
		}

		err = os.WriteFile(valPath, valData, 0644)
		if err != nil {
			return "", fmt.Errorf("failed to write validation data: %v", err)
		}
	}

	// Write metadata
	metaPath := filepath.Join(outputDir, fmt.Sprintf("metadata_%s.json", timestamp))
	metadata := map[string]interface{}{
		"timestamp":           time.Now().Format(time.RFC3339),
		"total_examples":      len(trainExamples) + len(valExamples),
		"training_examples":   len(trainExamples),
		"validation_examples": len(valExamples),
		"format":              "json",
	}

	metaData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal metadata: %v", err)
	}

	err = os.WriteFile(metaPath, metaData, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write metadata: %v", err)
	}

	return outputDir, nil
}

// writeCSVOutput writes training data to CSV files
func (s *TrainingDataService) writeCSVOutput(outputDir string,
	trainExamples, valExamples []TrainingExtendedExample) (string, error) {

	// Create timestamp for filenames
	timestamp := time.Now().Format("20060102_150405")

	// Write training data
	trainPath := filepath.Join(outputDir, fmt.Sprintf("train_data_%s.csv", timestamp))
	trainFile, err := os.Create(trainPath)
	if err != nil {
		return "", fmt.Errorf("failed to create training data file: %v", err)
	}
	defer trainFile.Close()

	// Write header
	trainFile.WriteString("command,prediction,label,weight,source\n")

	// Write each example
	for _, ex := range trainExamples {
		// Escape commas and quotes
		command := strings.ReplaceAll(ex.Command, "\"", "\"\"")
		prediction := strings.ReplaceAll(ex.Prediction, "\"", "\"\"")

		line := fmt.Sprintf("\"%s\",\"%s\",%d,%.2f,%s\n",
			command, prediction, ex.Label, ex.Weight, ex.Source)
		trainFile.WriteString(line)
	}

	// Write validation data if available
	if valExamples != nil {
		valPath := filepath.Join(outputDir, fmt.Sprintf("val_data_%s.csv", timestamp))
		valFile, err := os.Create(valPath)
		if err != nil {
			return "", fmt.Errorf("failed to create validation data file: %v", err)
		}
		defer valFile.Close()

		// Write header
		valFile.WriteString("command,prediction,label,weight,source\n")

		// Write each example
		for _, ex := range valExamples {
			// Escape commas and quotes
			command := strings.ReplaceAll(ex.Command, "\"", "\"\"")
			prediction := strings.ReplaceAll(ex.Prediction, "\"", "\"\"")

			line := fmt.Sprintf("\"%s\",\"%s\",%d,%.2f,%s\n",
				command, prediction, ex.Label, ex.Weight, ex.Source)
			valFile.WriteString(line)
		}
	}

	// Write metadata
	metaPath := filepath.Join(outputDir, fmt.Sprintf("metadata_%s.json", timestamp))
	metadata := map[string]interface{}{
		"timestamp":           time.Now().Format(time.RFC3339),
		"total_examples":      len(trainExamples) + len(valExamples),
		"training_examples":   len(trainExamples),
		"validation_examples": len(valExamples),
		"format":              "csv",
	}

	metaData, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal metadata: %v", err)
	}

	err = os.WriteFile(metaPath, metaData, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write metadata: %v", err)
	}

	return outputDir, nil
}

// writeTFRecordOutput writes training data to TFRecord files
// This is a placeholder implementation that returns an error since
// TFRecord format requires TensorFlow-specific libraries
func (s *TrainingDataService) writeTFRecordOutput(outputDir string,
	trainExamples, valExamples []TrainingExtendedExample) (string, error) {
	return "", fmt.Errorf("TFRecord format not implemented yet")
}

// balanceExamples balances positive and negative examples
func (s *TrainingDataService) balanceExamples(examples []TrainingExample) []TrainingExample {
	// Count positive and negative examples
	positiveCount := 0
	negativeCount := 0
	neutralCount := 0

	for _, ex := range examples {
		switch ex.Label {
		case 1:
			positiveCount++
		case -1:
			negativeCount++
		default:
			neutralCount++
		}
	}

	// If already balanced, return as is
	if positiveCount == negativeCount || (positiveCount == 0 || negativeCount == 0) {
		return examples
	}

	// Determine target count (smaller of positive and negative)
	targetCount := positiveCount
	if negativeCount < positiveCount {
		targetCount = negativeCount
	}

	// Create balanced set
	balanced := make([]TrainingExample, 0)
	posAdded := 0
	negAdded := 0

	// Add all neutrals
	for _, ex := range examples {
		if ex.Label == 0 {
			balanced = append(balanced, ex)
		}
	}

	// Add positive and negative examples up to target count
	for _, ex := range examples {
		if ex.Label == 1 && posAdded < targetCount {
			balanced = append(balanced, ex)
			posAdded++
		} else if ex.Label == -1 && negAdded < targetCount {
			balanced = append(balanced, ex)
			negAdded++
		}
	}

	return balanced
}

// augmentData creates synthetic examples by combining or modifying existing ones
func (s *TrainingDataService) augmentData(examples []TrainingExample) []TrainingExample {
	if len(examples) == 0 {
		return examples
	}

	// Create a map of commands to predictions for positive examples
	positiveMap := make(map[string][]string)
	for _, ex := range examples {
		if ex.Label == 1 {
			if _, ok := positiveMap[ex.Command]; !ok {
				positiveMap[ex.Command] = make([]string, 0)
			}
			positiveMap[ex.Command] = append(positiveMap[ex.Command], ex.Prediction)
		}
	}

	// Create synthetic examples if possible
	synthetic := make([]TrainingExample, 0)

	// Try to combine patterns
	for i, ex1 := range examples {
		if i < len(examples)-1 {
			for j := i + 1; j < len(examples); j++ {
				ex2 := examples[j]

				// Only combine examples with the same label
				if ex1.Label == ex2.Label && ex1.Label != 0 {
					// Check if commands are similar but not identical
					if len(ex1.Command) > 3 && len(ex2.Command) > 3 {
						similar := false

						// Check for common command patterns
						if strings.Contains(ex1.Command, "git") && strings.Contains(ex2.Command, "git") {
							similar = true
						} else if strings.Contains(ex1.Command, "docker") && strings.Contains(ex2.Command, "docker") {
							similar = true
						} else if strings.Contains(ex1.Command, "make") && strings.Contains(ex2.Command, "make") {
							similar = true
						}

						if similar {
							// Create new synthetic example
							synthetic = append(synthetic, TrainingExample{
								Command:    ex1.Command + " && " + ex2.Command,
								Context:    ex1.Context,
								Prediction: ex1.Prediction + " " + ex2.Prediction,
								Label:      ex1.Label,
								Weight:     (ex1.Weight + ex2.Weight) / 2,
								Source:     "synthetic",
							})
						}
					}
				}
			}
		}
	}

	// Add synthetic examples to original set
	return append(examples, synthetic...)
}

// GetTrainingDataStats returns statistics about available training data
func (s *TrainingDataService) GetTrainingDataStats() map[string]interface{} {
	// Get training examples from inference manager
	examples, err := s.inferenceManager.GetTrainingExamples(-1) // Get all examples
	if err != nil {
		return map[string]interface{}{
			"error": fmt.Sprintf("failed to get training examples: %v", err),
		}
	}

	// Count by label and source
	positiveCount := 0
	negativeCount := 0
	neutralCount := 0

	sourceCounts := make(map[string]int)

	for _, ex := range examples {
		// Count by label
		switch ex.Label {
		case 1:
			positiveCount++
		case -1:
			negativeCount++
		default:
			neutralCount++
		}

		// Count by source
		sourceCounts[ex.Source]++
	}

	// Get stats from inference manager
	infStats := s.inferenceManager.GetInferenceStats()

	// Get memory manager stats
	memStats, err := s.memoryManager.GetStats()
	totalCommands := 0
	if err == nil {
		totalCommands = memStats.TotalEntries
	}

	// Return stats
	return map[string]interface{}{
		"total_examples":       len(examples),
		"positive_examples":    positiveCount,
		"negative_examples":    negativeCount,
		"neutral_examples":     neutralCount,
		"feedback_count":       infStats["feedback_count"].(int),
		"source_distribution":  sourceCounts,
		"total_commands":       totalCommands,
		"accumulated_examples": infStats["accumulated_examples"].(int),
		"training_threshold":   100, // Minimum examples needed for training
		"is_training_ready":    s.inferenceManager.ShouldTrain(),
	}
}

// Global TrainingDataService instance
var globalTrainingDataService *TrainingDataService

// GetTrainingDataService returns the global TrainingDataService instance
func GetTrainingDataService() *TrainingDataService {
	if globalTrainingDataService == nil {
		var err error
		globalTrainingDataService, err = NewTrainingDataService()
		if err != nil {
			fmt.Printf("Error initializing training data service: %v\n", err)
			return nil
		}
	}
	return globalTrainingDataService
}
