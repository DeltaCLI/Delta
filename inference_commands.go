package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// HandleInferenceCommand processes inference-related commands
func HandleInferenceCommand(args []string) bool {
	// Get the inference manager
	im := GetInferenceManager()
	if im == nil {
		fmt.Println("Failed to initialize inference manager")
		return true
	}

	// Handle commands
	if len(args) == 0 {
		// Show inference status
		showInferenceStatus(im)
		return true
	}

	// Handle subcommands
	if len(args) >= 1 {
		cmd := args[0]

		switch cmd {
		case "enable":
			// Enable learning system
			err := im.EnableLearning()
			if err != nil {
				fmt.Printf("Error enabling learning system: %v\n", err)
			} else {
				fmt.Println("Learning system enabled")
			}
			return true

		case "disable":
			// Disable learning system
			err := im.DisableLearning()
			if err != nil {
				fmt.Printf("Error disabling learning system: %v\n", err)
			} else {
				fmt.Println("Learning system disabled")
			}
			return true

		case "status":
			// Show status
			showInferenceStatus(im)
			return true

		case "feedback":
			// Add feedback
			if len(args) >= 3 {
				addInferenceFeedback(im, args[1], args[2])
			} else if len(args) >= 2 {
				addInferenceFeedback(im, args[1], "")
			} else {
				fmt.Println("Usage: :inference feedback <helpful|unhelpful|correction> [correction]")
			}
			return true

		case "stats":
			// Show detailed stats
			showInferenceStats(im)
			return true

		case "model":
			// Manage custom model
			if len(args) >= 2 {
				manageCustomModel(im, args[1:])
			} else {
				fmt.Println("Usage: :inference model <use|info>")
			}
			return true

		case "examples":
			// Show training examples
			showTrainingExamples(im)
			return true

		case "config":
			// Configure inference system
			if len(args) >= 3 && args[1] == "set" {
				configureInference(im, args[2:])
			} else {
				showInferenceConfig(im)
			}
			return true

		case "help":
			// Show help
			showInferenceHelp()
			return true

		default:
			fmt.Printf("Unknown inference command: %s\n", cmd)
			fmt.Println("Type :inference help for a list of available commands")
			return true
		}
	}

	return true
}

// showInferenceStatus displays current status of the inference system
func showInferenceStatus(im *InferenceManager) {
	fmt.Println("Inference System Status")
	fmt.Println("=======================")

	stats := im.GetInferenceStats()

	enabled := stats["learning_enabled"].(bool)
	fmt.Printf("Learning System: %s\n", formatStatus(enabled))

	fmt.Printf("Feedback Collection: %s\n", formatStatus(stats["feedback_collection"].(bool)))
	fmt.Printf("Feedback Count: %d\n", stats["feedback_count"].(int))
	fmt.Printf("Training Examples: %d\n", stats["training_examples"].(int))

	fmt.Printf("\nModel Information:\n")
	fmt.Printf("  Custom Model: %s\n", formatStatus(stats["custom_model_enabled"].(bool)))

	if stats["custom_model_enabled"].(bool) {
		if stats["custom_model_available"].(bool) {
			fmt.Printf("  Model Path: %s\n", stats["model_path"])
		} else {
			fmt.Printf("  Model Status: Not found\n")
		}
	}

	fmt.Printf("  Ollama Inference: %s\n", formatStatus(stats["ollama_enabled"].(bool)))
	fmt.Printf("  Local Inference: %s\n", formatStatus(stats["local_inference_enabled"].(bool)))

	fmt.Printf("\nTraining Status:\n")
	fmt.Printf("  Periodic Training: %s\n", formatStatus(stats["periodic_training"].(bool)))
	fmt.Printf("  Training Interval: %d days\n", stats["training_interval_days"].(int))
	fmt.Printf("  Last Training: %s\n", stats["last_training"])
}

// showInferenceStats displays detailed stats about the inference system
func showInferenceStats(im *InferenceManager) {
	fmt.Println("Inference System Statistics")
	fmt.Println("==========================")

	stats := im.GetInferenceStats()

	fmt.Printf("Learning System: %s\n", formatStatus(stats["learning_enabled"].(bool)))
	fmt.Printf("Feedback Collection: %s\n", formatStatus(stats["feedback_collection"].(bool)))
	fmt.Printf("Automatic Feedback: %s\n", formatStatus(stats["automatic_feedback"].(bool)))

	// Feedback data
	fmt.Printf("\nFeedback Data:\n")
	fmt.Printf("  Total Feedback Entries: %d\n", stats["feedback_count"].(int))
	fmt.Printf("  Training Examples: %d\n", stats["training_examples"].(int))
	fmt.Printf("  Accumulated Examples: %d\n", stats["accumulated_examples"].(int))

	// Recent feedback breakdown
	fmt.Printf("\nRecent Feedback Breakdown:\n")
	feedbacks, err := im.GetFeedbacks(time.Now().AddDate(0, -1, 0), time.Time{})
	if err == nil {
		helpfulCount := 0
		unhelpfulCount := 0
		correctionCount := 0

		for _, feedback := range feedbacks {
			switch feedback.FeedbackType {
			case "helpful":
				helpfulCount++
			case "unhelpful":
				unhelpfulCount++
			case "correction":
				correctionCount++
			}
		}

		fmt.Printf("  Helpful: %d\n", helpfulCount)
		fmt.Printf("  Unhelpful: %d\n", unhelpfulCount)
		fmt.Printf("  Corrections: %d\n", correctionCount)
		fmt.Printf("  Total Recent: %d\n", len(feedbacks))
	}

	// Model information
	fmt.Printf("\nModel Information:\n")
	fmt.Printf("  Custom Model: %s\n", formatStatus(stats["custom_model_enabled"].(bool)))

	if stats["custom_model_enabled"].(bool) {
		if stats["custom_model_available"].(bool) {
			fmt.Printf("  Model Path: %s\n", stats["model_path"])

			// Get model file info
			if fileInfo, err := os.Stat(stats["model_path"].(string)); err == nil {
				fmt.Printf("  Model Size: %.2f MB\n", float64(fileInfo.Size())/(1024*1024))
				fmt.Printf("  Last Modified: %s\n", fileInfo.ModTime().Format(time.RFC1123))
			}
		} else {
			fmt.Printf("  Model Status: Not found\n")
		}
	}

	fmt.Printf("  Ollama Inference: %s\n", formatStatus(stats["ollama_enabled"].(bool)))
	fmt.Printf("  Local Inference: %s\n", formatStatus(stats["local_inference_enabled"].(bool)))

	fmt.Printf("\nTraining Information:\n")
	fmt.Printf("  Periodic Training: %s\n", formatStatus(stats["periodic_training"].(bool)))
	fmt.Printf("  Training Interval: %d days\n", stats["training_interval_days"].(int))
	fmt.Printf("  Last Training: %s\n", stats["last_training"])

	// Check if training should be triggered
	if im.ShouldTrain() {
		fmt.Printf("\n* Training is due. Run ':memory train start' to start training.\n")
	}
}

// addInferenceFeedback adds user feedback for the inference system
func addInferenceFeedback(im *InferenceManager, feedbackType, correction string) {
	// Get the AI manager to get the last prediction
	ai := GetAIManager()
	if ai == nil {
		fmt.Println("AI manager not available")
		return
	}

	// Get last prediction using the enhanced tracking mechanism
	lastCommand, lastThought, timestamp := ai.GetLastPrediction()

	// Validate the prediction data
	if lastThought == "" {
		fmt.Println("No recent predictions to provide feedback for")
		return
	}

	if lastCommand == "" {
		// Fall back to command history if needed
		if len(ai.commandHistory) > 0 {
			lastCommand = ai.commandHistory[len(ai.commandHistory)-1]
		} else {
			fmt.Println("No recent commands to provide feedback for")
			return
		}
	}

	// Check if the prediction is too old (more than 1 hour)
	if !timestamp.IsZero() && time.Since(timestamp) > time.Hour {
		fmt.Println("Warning: Providing feedback for a prediction made over an hour ago")
		fmt.Printf("Prediction time: %s\n", timestamp.Format(time.RFC1123))
		fmt.Println("Command: " + lastCommand)
		fmt.Println("Thought: " + lastThought)
		fmt.Println("Do you want to continue? (y/n)")

		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			fmt.Println("Feedback cancelled")
			return
		}
	}

	// Validate feedback type
	validTypes := map[string]bool{
		"helpful": true, "good": true, "positive": true,
		"unhelpful": true, "bad": true, "negative": true,
		"correction": true, "fix": true, "wrong": true,
	}

	normalizedType := strings.ToLower(feedbackType)

	// Map various inputs to standard feedback types
	if normalizedType == "good" || normalizedType == "positive" {
		normalizedType = "helpful"
	} else if normalizedType == "bad" || normalizedType == "negative" {
		normalizedType = "unhelpful"
	} else if normalizedType == "fix" || normalizedType == "wrong" {
		normalizedType = "correction"
	}

	if !validTypes[normalizedType] {
		fmt.Printf("Invalid feedback type: %s\n", feedbackType)
		fmt.Println("Valid types: helpful, unhelpful, correction")
		return
	}

	// Get current working directory for context
	pwd, err := os.Getwd()
	if err != nil {
		pwd = "unknown"
	}

	// Display what we're giving feedback on
	fmt.Println("\nProviding feedback on:")
	fmt.Printf("Command: %s\n", lastCommand)
	fmt.Printf("AI thought: %s\n", lastThought)

	// Add feedback
	err = im.AddFeedback(lastCommand, lastThought, normalizedType, correction, pwd)
	if err != nil {
		fmt.Printf("Error adding feedback: %v\n", err)
		return
	}

	// Display confirmation
	switch normalizedType {
	case "helpful":
		fmt.Println("✅ Marked as helpful")
	case "unhelpful":
		fmt.Println("❌ Marked as unhelpful")
	case "correction":
		fmt.Println("🔄 Correction recorded")
		if correction != "" {
			fmt.Printf("Correction: %s\n", correction)
		}
	}

	// Show training status
	if im.learningConfig.PeriodicTraining {
		exampleCount := im.learningConfig.AccumulatedTrainingExamples
		fmt.Printf("Training examples collected: %d\n", exampleCount)

		if im.ShouldTrain() {
			fmt.Println("\n💡 Training is due. Run ':memory train start' to improve AI predictions.")
		} else if exampleCount > 0 {
			remaining := 100 - exampleCount // Need at least 100 examples to train
			if remaining > 0 {
				fmt.Printf("Need %d more examples before training can begin.\n", remaining)
			}
		}
	}

	fmt.Println("\nThank you for your feedback! It will help improve future predictions.")
}

// manageCustomModel manages custom model settings
func manageCustomModel(im *InferenceManager, args []string) {
	if len(args) == 0 {
		showInferenceStatus(im)
		return
	}

	subcmd := args[0]

	switch subcmd {
	case "use":
		if len(args) >= 2 {
			modelPath := args[1]

			// If relative path, convert to absolute
			if !filepath.IsAbs(modelPath) {
				homeDir, _ := os.UserHomeDir()
				fullPath := filepath.Join(homeDir, ".config", "delta", "memory", "models", modelPath)
				if _, err := os.Stat(fullPath); err == nil {
					modelPath = fullPath
				}
			}

			// Check if model exists
			if _, err := os.Stat(modelPath); err != nil {
				fmt.Printf("Model not found: %s\n", modelPath)
				return
			}

			// Update config
			config := im.inferenceConfig
			learning := im.learningConfig

			learning.UseCustomModel = true
			learning.CustomModelPath = modelPath
			config.UseLocalInference = true
			config.ModelPath = modelPath

			err := im.UpdateConfig(config, learning)
			if err != nil {
				fmt.Printf("Error updating config: %v\n", err)
				return
			}

			fmt.Printf("Now using custom model: %s\n", modelPath)
		} else {
			fmt.Println("Usage: :inference model use <model_path>")
		}

	case "disable":
		// Disable custom model
		config := im.inferenceConfig
		learning := im.learningConfig

		learning.UseCustomModel = false
		config.UseLocalInference = false

		err := im.UpdateConfig(config, learning)
		if err != nil {
			fmt.Printf("Error updating config: %v\n", err)
			return
		}

		fmt.Println("Custom model disabled")

	case "info":
		// Show model info
		stats := im.GetInferenceStats()

		fmt.Println("Model Information")
		fmt.Println("=================")
		fmt.Printf("Custom Model: %s\n", formatStatus(stats["custom_model_enabled"].(bool)))

		if stats["custom_model_enabled"].(bool) {
			fmt.Printf("Model Path: %s\n", stats["model_path"])

			// Check if model exists
			modelPath := stats["model_path"].(string)
			if _, err := os.Stat(modelPath); err != nil {
				fmt.Println("Model Status: Not found")
			} else {
				fmt.Println("Model Status: Available")

				// Get file info
				if fileInfo, err := os.Stat(modelPath); err == nil {
					fmt.Printf("Model Size: %.2f MB\n", float64(fileInfo.Size())/(1024*1024))
					fmt.Printf("Last Modified: %s\n", fileInfo.ModTime().Format(time.RFC1123))
				}
			}
		}

		// List available models
		homeDir, _ := os.UserHomeDir()
		modelsDir := filepath.Join(homeDir, ".config", "delta", "memory", "models")

		if files, err := os.ReadDir(modelsDir); err == nil {
			var models []string
			for _, file := range files {
				if !file.IsDir() && (strings.HasSuffix(file.Name(), ".onnx") ||
					strings.HasSuffix(file.Name(), ".pt") ||
					strings.HasSuffix(file.Name(), ".bin")) {
					models = append(models, file.Name())
				}
			}

			if len(models) > 0 {
				fmt.Println("\nAvailable Models:")
				for _, model := range models {
					fmt.Printf("  - %s\n", model)
				}
			} else {
				fmt.Println("\nNo models available in ~/.config/delta/memory/models")
				fmt.Println("Use ':memory train start' to train a new model")
			}
		}

	default:
		fmt.Printf("Unknown subcommand: %s\n", subcmd)
		fmt.Println("Usage: :inference model <use|disable|info>")
	}
}

// showTrainingExamples displays training examples from feedback
func showTrainingExamples(im *InferenceManager) {
	// Get examples
	examples, err := im.GetTrainingExamples(10)
	if err != nil {
		fmt.Printf("Error getting examples: %v\n", err)
		return
	}

	if len(examples) == 0 {
		fmt.Println("No training examples available")
		fmt.Println("Add feedback with ':inference feedback [helpful|unhelpful|correction]'")
		return
	}

	fmt.Println("Training Examples")
	fmt.Println("=================")

	for i, example := range examples {
		fmt.Printf("%d. Command: %s\n", i+1, example.Command)
		fmt.Printf("   Prediction: %s\n", example.Prediction)

		// Format label
		var labelStr string
		switch example.Label {
		case 1:
			labelStr = "Positive"
		case -1:
			labelStr = "Negative"
		default:
			labelStr = "Neutral"
		}

		fmt.Printf("   Label: %s (weight: %.2f)\n", labelStr, example.Weight)
		fmt.Printf("   Source: %s\n", example.Source)

		if i < len(examples)-1 {
			fmt.Println()
		}
	}

	total := im.GetInferenceStats()
	fmt.Printf("\nShowing %d of %d total examples\n", len(examples), total["training_examples"].(int))
}

// showInferenceConfig displays the inference configuration
func showInferenceConfig(im *InferenceManager) {
	fmt.Println("Inference Configuration")
	fmt.Println("======================")

	// Display inference config
	fmt.Println("\nInference Settings:")
	fmt.Printf("  Model Type: %s\n", im.inferenceConfig.ModelType)
	fmt.Printf("  Max Tokens: %d\n", im.inferenceConfig.MaxTokens)
	fmt.Printf("  Temperature: %.2f\n", im.inferenceConfig.Temperature)
	fmt.Printf("  Top-K: %d\n", im.inferenceConfig.TopK)
	fmt.Printf("  Top-P: %.2f\n", im.inferenceConfig.TopP)
	fmt.Printf("  Use Speculative Decoding: %v\n", im.inferenceConfig.UseSpeculative)
	fmt.Printf("  Batch Size: %d\n", im.inferenceConfig.BatchSize)

	// Display learning config
	fmt.Println("\nLearning Settings:")
	fmt.Printf("  Enabled: %v\n", im.learningConfig.Enabled)
	fmt.Printf("  Collect Feedback: %v\n", im.learningConfig.CollectFeedback)
	fmt.Printf("  Automatic Feedback: %v\n", im.learningConfig.AutomaticFeedback)
	fmt.Printf("  Feedback Threshold: %.2f\n", im.learningConfig.FeedbackThreshold)
	fmt.Printf("  Adaptation Rate: %.2f\n", im.learningConfig.AdaptationRate)
	fmt.Printf("  Use Custom Model: %v\n", im.learningConfig.UseCustomModel)
	fmt.Printf("  Periodic Training: %v\n", im.learningConfig.PeriodicTraining)
	fmt.Printf("  Training Interval: %d days\n", im.learningConfig.TrainingInterval)
}

// configureInference updates inference configuration settings
func configureInference(im *InferenceManager, args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: :inference config set <setting> <value>")
		return
	}

	setting := args[0]
	value := args[1]

	// Get current configs
	inferenceConfig := im.inferenceConfig
	learningConfig := im.learningConfig

	// Update setting
	switch setting {
	case "enabled":
		learningConfig.Enabled = parseBool(value)

	case "collect_feedback":
		learningConfig.CollectFeedback = parseBool(value)

	case "automatic_feedback":
		learningConfig.AutomaticFeedback = parseBool(value)

	case "periodic_training":
		learningConfig.PeriodicTraining = parseBool(value)

	case "training_interval":
		if days, err := parseInteger(value); err == nil && days > 0 {
			learningConfig.TrainingInterval = days
		} else {
			fmt.Println("Training interval must be a positive integer")
			return
		}

	case "temperature":
		if temp, err := parseFloat(value); err == nil && temp >= 0 && temp <= 1 {
			inferenceConfig.Temperature = temp
		} else {
			fmt.Println("Temperature must be between 0 and 1")
			return
		}

	case "use_ollama":
		inferenceConfig.UseOllama = parseBool(value)

	case "use_local_inference":
		inferenceConfig.UseLocalInference = parseBool(value)

	default:
		fmt.Printf("Unknown setting: %s\n", setting)
		return
	}

	// Save updated config
	err := im.UpdateConfig(inferenceConfig, learningConfig)
	if err != nil {
		fmt.Printf("Error updating configuration: %v\n", err)
		return
	}

	fmt.Printf("Updated %s to %s\n", setting, value)
}

// showInferenceHelp displays help for inference commands
func showInferenceHelp() {
	fmt.Println("Inference Commands")
	fmt.Println("=================")
	fmt.Println("  :inference              - Show current inference status")
	fmt.Println("  :inference status       - Show inference status")
	fmt.Println("  :inference enable       - Enable learning system")
	fmt.Println("  :inference disable      - Disable learning system")
	fmt.Println("  :inference feedback <type> [correction] - Add feedback for last prediction")
	fmt.Println("       <type> can be: helpful, unhelpful, correction")
	fmt.Println("  :inference stats        - Show detailed inference statistics")
	fmt.Println("  :inference model use <path> - Use custom model")
	fmt.Println("  :inference model disable - Disable custom model")
	fmt.Println("  :inference model info   - Show model information")
	fmt.Println("  :inference examples     - Show training examples")
	fmt.Println("  :inference config       - Show configuration")
	fmt.Println("  :inference config set <setting> <value> - Update configuration")
	fmt.Println("  :inference help         - Show this help message")
}

// Helper functions

// formatStatus formats a boolean as Enabled/Disabled
func formatStatus(enabled bool) string {
	if enabled {
		return "Enabled"
	}
	return "Disabled"
}

// parseBool parses a string to a boolean
func parseBool(s string) bool {
	return s == "true" || s == "1" || s == "yes" || s == "on" || s == "enable" || s == "enabled"
}

// parseInteger parses a string to an integer
func parseInteger(s string) (int, error) {
	var value int
	_, err := fmt.Sscanf(s, "%d", &value)
	return value, err
}

// parseFloat parses a string to a float
func parseFloat(s string) (float64, error) {
	var value float64
	_, err := fmt.Sscanf(s, "%f", &value)
	return value, err
}
