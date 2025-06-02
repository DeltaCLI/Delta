package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// HandleART2Command handles ART-2 related commands
func HandleART2Command(args []string) bool {
	if len(args) == 0 {
		showART2Help()
		return true
	}

	art2Mgr := GetART2Manager()
	if art2Mgr == nil {
		fmt.Println("ART-2 system not available")
		return true
	}

	switch args[0] {
	case "enable":
		return handleART2Enable(art2Mgr)
	case "disable":
		return handleART2Disable(art2Mgr)
	case "status":
		return handleART2Status(art2Mgr)
	case "stats":
		return handleART2Stats(art2Mgr)
	case "categories":
		return handleART2Categories(art2Mgr, args[1:])
	case "predict":
		return handleART2Predict(art2Mgr, args[1:])
	case "feedback":
		return handleART2Feedback(art2Mgr, args[1:])
	case "config":
		return handleART2Config(art2Mgr, args[1:])
	case "help":
		showART2Help()
		return true
	default:
		fmt.Printf("Unknown ART-2 command: %s\n", args[0])
		showART2Help()
		return true
	}
}

// handleART2Enable enables the ART-2 system
func handleART2Enable(art2Mgr *ART2Manager) bool {
	art2Mgr.config.Enabled = true
	err := art2Mgr.saveConfig()
	if err != nil {
		fmt.Printf("Error saving ART-2 configuration: %v\n", err)
		return true
	}
	
	fmt.Println("ART-2 adaptive pattern recognition enabled")
	return true
}

// handleART2Disable disables the ART-2 system
func handleART2Disable(art2Mgr *ART2Manager) bool {
	art2Mgr.config.Enabled = false
	err := art2Mgr.saveConfig()
	if err != nil {
		fmt.Printf("Error saving ART-2 configuration: %v\n", err)
		return true
	}
	
	fmt.Println("ART-2 adaptive pattern recognition disabled")
	return true
}

// handleART2Status shows the current status of the ART-2 system
func handleART2Status(art2Mgr *ART2Manager) bool {
	fmt.Println("ART-2 Adaptive Resonance Theory System Status:")
	fmt.Println("==============================================")
	
	if art2Mgr.IsEnabled() {
		fmt.Println("Status: ENABLED ✓")
	} else {
		fmt.Println("Status: DISABLED ✗")
	}
	
	fmt.Printf("Initialized: %v\n", art2Mgr.isInitialized)
	fmt.Printf("Categories learned: %d\n", len(art2Mgr.categories))
	fmt.Printf("Maximum categories: %d\n", art2Mgr.config.MaxCategories)
	fmt.Printf("Vector size: %d\n", art2Mgr.config.VectorSize)
	
	// Algorithm parameters
	fmt.Println("\nAlgorithm Parameters:")
	fmt.Printf("- Vigilance (ρ): %.3f\n", art2Mgr.config.Rho)
	fmt.Printf("- Learning rate (β): %.3f\n", art2Mgr.config.Beta)
	fmt.Printf("- Choice parameter (α): %.3f\n", art2Mgr.config.Alpha)
	fmt.Printf("- Activity threshold (θ): %.3f\n", art2Mgr.config.Theta)
	
	// Memory efficiency
	stats := art2Mgr.GetStats()
	fmt.Printf("\nMemory efficiency: %.2f%%\n", stats.MemoryEfficiency*100)
	fmt.Printf("Accuracy rate: %.2f%%\n", stats.AccuracyRate*100)
	
	return true
}

// handleART2Stats shows detailed statistics about the ART-2 system
func handleART2Stats(art2Mgr *ART2Manager) bool {
	stats := art2Mgr.GetStats()
	
	fmt.Println("ART-2 Learning Statistics:")
	fmt.Println("==========================")
	fmt.Printf("Total inputs processed: %d\n", stats.TotalInputs)
	fmt.Printf("Categories learned: %d\n", stats.CategoriesLearned)
	fmt.Printf("Correct predictions: %d\n", stats.CorrectPredictions)
	fmt.Printf("Incorrect predictions: %d\n", stats.IncorrectPredictions)
	fmt.Printf("Accuracy rate: %.2f%%\n", stats.AccuracyRate*100)
	fmt.Printf("Memory efficiency: %.2f%%\n", stats.MemoryEfficiency*100)
	
	if !stats.LastTrainingTime.IsZero() {
		fmt.Printf("Last training: %s\n", stats.LastTrainingTime.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Println("Last training: Never")
	}
	
	// Show preprocessor statistics
	preprocessor := GetART2Preprocessor()
	if preprocessor != nil {
		fmt.Println("\nPreprocessor Statistics:")
		vocabStats := preprocessor.GetVocabularyStats()
		fmt.Printf("Vocabulary size: %d\n", vocabStats["vocabulary_size"].(int))
		fmt.Printf("Feature count: %d\n", vocabStats["feature_count"].(int))
		fmt.Printf("Vector size: %d\n", vocabStats["vector_size"].(int))
		
		if topTokens, ok := vocabStats["top_tokens"].([]string); ok && len(topTokens) > 0 {
			fmt.Printf("Top tokens: %s\n", strings.Join(topTokens[:art2Min(5, len(topTokens))], ", "))
		}
	}
	
	return true
}

// handleART2Categories shows information about learned categories
func handleART2Categories(art2Mgr *ART2Manager, args []string) bool {
	categories := art2Mgr.GetCategories()
	
	if len(categories) == 0 {
		fmt.Println("No categories learned yet")
		return true
	}
	
	// Show summary by default
	if len(args) == 0 || args[0] == "list" {
		fmt.Printf("ART-2 Learned Categories (%d total):\n", len(categories))
		fmt.Println("=====================================")
		
		for i, category := range categories {
			fmt.Printf("Category %d:\n", category.ID)
			fmt.Printf("  Activations: %d\n", category.ActivationCount)
			fmt.Printf("  Success rate: %.2f%%\n", category.SuccessRate*100)
			fmt.Printf("  Created: %s\n", category.CreatedAt.Format("2006-01-02 15:04:05"))
			fmt.Printf("  Last used: %s\n", category.LastActivation.Format("2006-01-02 15:04:05"))
			
			if len(category.CommandPatterns) > 0 {
				fmt.Printf("  Command patterns: %s\n", 
					strings.Join(category.CommandPatterns[:art2Min(3, len(category.CommandPatterns))], ", "))
			}
			
			if i < len(categories)-1 {
				fmt.Println()
			}
		}
		return true
	}
	
	// Show detailed information for a specific category
	if len(args) > 0 {
		categoryID, err := strconv.Atoi(args[0])
		if err != nil {
			fmt.Printf("Invalid category ID: %s\n", args[0])
			return true
		}
		
		var targetCategory *ART2Category
		for _, category := range categories {
			if category.ID == categoryID {
				targetCategory = category
				break
			}
		}
		
		if targetCategory == nil {
			fmt.Printf("Category %d not found\n", categoryID)
			return true
		}
		
		fmt.Printf("Category %d Details:\n", targetCategory.ID)
		fmt.Println("===================")
		fmt.Printf("Activation count: %d\n", targetCategory.ActivationCount)
		fmt.Printf("Success rate: %.2f%%\n", targetCategory.SuccessRate*100)
		fmt.Printf("Created: %s\n", targetCategory.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("Last activation: %s\n", targetCategory.LastActivation.Format("2006-01-02 15:04:05"))
		
		fmt.Printf("Command patterns (%d):\n", len(targetCategory.CommandPatterns))
		for i, pattern := range targetCategory.CommandPatterns {
			fmt.Printf("  %d. %s\n", i+1, pattern)
		}
		
		fmt.Printf("Context patterns (%d):\n", len(targetCategory.ContextPatterns))
		for i, pattern := range targetCategory.ContextPatterns {
			fmt.Printf("  %d. %s\n", i+1, pattern)
		}
		
		fmt.Printf("Weight vector size: %d\n", len(targetCategory.Weights))
	}
	
	return true
}

// handleART2Predict generates a prediction for a command
func handleART2Predict(art2Mgr *ART2Manager, args []string) bool {
	if len(args) == 0 {
		fmt.Println("Usage: :art2 predict <command>")
		return true
	}
	
	if !art2Mgr.IsEnabled() {
		fmt.Println("ART-2 system is disabled. Enable with ':art2 enable'")
		return true
	}
	
	command := strings.Join(args, " ")
	dir, _ := os.Getwd()
	
	// Preprocess the command
	preprocessor := GetART2Preprocessor()
	if preprocessor == nil {
		fmt.Println("ART-2 preprocessor not available")
		return true
	}
	
	featureVector, err := preprocessor.PreprocessCommand(command, "", dir)
	if err != nil {
		fmt.Printf("Error preprocessing command: %v\n", err)
		return true
	}
	
	// Get prediction
	prediction, confidence, err := art2Mgr.GetPrediction(featureVector.Values, command, dir)
	if err != nil {
		fmt.Printf("Error generating prediction: %v\n", err)
		return true
	}
	
	if prediction == "" {
		fmt.Println("No prediction available for this command pattern")
		return true
	}
	
	fmt.Printf("Command: %s\n", command)
	fmt.Printf("Prediction: %s\n", prediction)
	fmt.Printf("Confidence: %.2f%%\n", confidence*100)
	
	return true
}

// handleART2Feedback provides feedback for ART-2 learning
func handleART2Feedback(art2Mgr *ART2Manager, args []string) bool {
	if len(args) < 2 {
		fmt.Println("Usage: :art2 feedback <helpful|unhelpful|correction> <command> [prediction]")
		fmt.Println("Examples:")
		fmt.Println("  :art2 feedback helpful 'ls -la'")
		fmt.Println("  :art2 feedback unhelpful 'git status'")
		fmt.Println("  :art2 feedback correction 'make' 'try make clean first'")
		return true
	}
	
	feedbackType := args[0]
	command := args[1]
	
	if feedbackType != "helpful" && feedbackType != "unhelpful" && feedbackType != "correction" {
		fmt.Println("Feedback type must be 'helpful', 'unhelpful', or 'correction'")
		return true
	}
	
	// Get current context
	dir, _ := os.Getwd()
	
	// Process with ART-2 to get current prediction
	preprocessor := GetART2Preprocessor()
	if preprocessor == nil {
		fmt.Println("ART-2 preprocessor not available")
		return true
	}
	
	featureVector, err := preprocessor.PreprocessCommand(command, "", dir)
	if err != nil {
		fmt.Printf("Error preprocessing command: %v\n", err)
		return true
	}
	
	// Create feedback input
	userFeedback := 0
	if feedbackType == "helpful" {
		userFeedback = 1
	} else if feedbackType == "unhelpful" {
		userFeedback = -1
	}
	
	actualOutcome := ""
	if len(args) > 2 {
		actualOutcome = strings.Join(args[2:], " ")
	}
	
	art2Input := ART2Input{
		Vector:         featureVector.Values,
		Command:        command,
		Context:        dir,
		Timestamp:      time.Now(),
		UserFeedback:   userFeedback,
		ActualOutcome:  actualOutcome,
	}
	
	// Process the feedback
	category, isNew, err := art2Mgr.ProcessInput(art2Input)
	if err != nil {
		fmt.Printf("Error processing feedback: %v\n", err)
		return true
	}
	
	if category != nil {
		if isNew {
			fmt.Printf("Created new category %d based on feedback\n", category.ID)
		} else {
			fmt.Printf("Updated category %d with feedback\n", category.ID)
		}
		fmt.Printf("Category now has %.2f%% success rate\n", category.SuccessRate*100)
	} else {
		fmt.Println("Feedback processed (no category match)")
	}
	
	return true
}

// handleART2Config manages ART-2 configuration
func handleART2Config(art2Mgr *ART2Manager, args []string) bool {
	if len(args) == 0 {
		// Show current configuration
		fmt.Println("ART-2 Configuration:")
		fmt.Println("===================")
		fmt.Printf("Enabled: %v\n", art2Mgr.config.Enabled)
		fmt.Printf("Vigilance (ρ): %.3f\n", art2Mgr.config.Rho)
		fmt.Printf("Learning rate (β): %.3f\n", art2Mgr.config.Beta)
		fmt.Printf("Choice parameter (α): %.3f\n", art2Mgr.config.Alpha)
		fmt.Printf("Activity threshold (θ): %.3f\n", art2Mgr.config.Theta)
		fmt.Printf("Max categories: %d\n", art2Mgr.config.MaxCategories)
		fmt.Printf("Vector size: %d\n", art2Mgr.config.VectorSize)
		fmt.Printf("Decay rate: %.3f\n", art2Mgr.config.DecayRate)
		fmt.Printf("Min activation: %.3f\n", art2Mgr.config.MinActivation)
		fmt.Printf("Update interval: %d seconds\n", art2Mgr.config.UpdateInterval)
		return true
	}
	
	if len(args) < 2 {
		fmt.Println("Usage: :art2 config <parameter> <value>")
		fmt.Println("Parameters: rho, beta, alpha, theta, max_categories, decay_rate, min_activation")
		return true
	}
	
	parameter := args[0]
	valueStr := args[1]
	
	switch parameter {
	case "rho", "vigilance":
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil || value < 0 || value > 1 {
			fmt.Println("Vigilance must be a number between 0 and 1")
			return true
		}
		art2Mgr.config.Rho = value
		fmt.Printf("Vigilance parameter set to %.3f\n", value)
		
	case "beta", "learning_rate":
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil || value < 0 || value > 1 {
			fmt.Println("Learning rate must be a number between 0 and 1")
			return true
		}
		art2Mgr.config.Beta = value
		fmt.Printf("Learning rate set to %.3f\n", value)
		
	case "alpha", "choice":
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil || value < 0 {
			fmt.Println("Choice parameter must be a positive number")
			return true
		}
		art2Mgr.config.Alpha = value
		fmt.Printf("Choice parameter set to %.3f\n", value)
		
	case "theta", "threshold":
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil || value < 0 || value > 1 {
			fmt.Println("Activity threshold must be a number between 0 and 1")
			return true
		}
		art2Mgr.config.Theta = value
		fmt.Printf("Activity threshold set to %.3f\n", value)
		
	case "max_categories":
		value, err := strconv.Atoi(valueStr)
		if err != nil || value < 1 || value > 1000 {
			fmt.Println("Max categories must be a number between 1 and 1000")
			return true
		}
		art2Mgr.config.MaxCategories = value
		fmt.Printf("Maximum categories set to %d\n", value)
		
	case "decay_rate":
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil || value < 0 || value > 1 {
			fmt.Println("Decay rate must be a number between 0 and 1")
			return true
		}
		art2Mgr.config.DecayRate = value
		fmt.Printf("Decay rate set to %.3f\n", value)
		
	case "min_activation":
		value, err := strconv.ParseFloat(valueStr, 64)
		if err != nil || value < 0 || value > 1 {
			fmt.Println("Minimum activation must be a number between 0 and 1")
			return true
		}
		art2Mgr.config.MinActivation = value
		fmt.Printf("Minimum activation set to %.3f\n", value)
		
	default:
		fmt.Printf("Unknown parameter: %s\n", parameter)
		fmt.Println("Available parameters: rho, beta, alpha, theta, max_categories, decay_rate, min_activation")
		return true
	}
	
	// Save the configuration
	err := art2Mgr.saveConfig()
	if err != nil {
		fmt.Printf("Error saving configuration: %v\n", err)
	} else {
		fmt.Println("Configuration saved successfully")
	}
	
	return true
}

// showART2Help displays help for ART-2 commands
func showART2Help() {
	fmt.Println("ART-2 Adaptive Resonance Theory Commands:")
	fmt.Println("=========================================")
	fmt.Println()
	fmt.Println("Basic Commands:")
	fmt.Println("  :art2 enable              - Enable ART-2 pattern learning")
	fmt.Println("  :art2 disable             - Disable ART-2 pattern learning")
	fmt.Println("  :art2 status              - Show system status and parameters")
	fmt.Println("  :art2 stats               - Show detailed learning statistics")
	fmt.Println()
	fmt.Println("Learning Commands:")
	fmt.Println("  :art2 categories          - List all learned categories")
	fmt.Println("  :art2 categories <id>     - Show details for category <id>")
	fmt.Println("  :art2 predict <command>   - Generate prediction for command")
	fmt.Println("  :art2 feedback <type> <command> [outcome]")
	fmt.Println("                            - Provide learning feedback")
	fmt.Println("                              Types: helpful, unhelpful, correction")
	fmt.Println()
	fmt.Println("Configuration:")
	fmt.Println("  :art2 config              - Show current configuration")
	fmt.Println("  :art2 config <param> <val>- Set configuration parameter")
	fmt.Println("                              Params: rho, beta, alpha, theta,")
	fmt.Println("                                     max_categories, decay_rate")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  :art2 predict ls -la")
	fmt.Println("  :art2 feedback helpful 'git status'")
	fmt.Println("  :art2 feedback correction 'make' 'try make clean first'")
	fmt.Println("  :art2 config rho 0.85")
	fmt.Println()
	fmt.Println("The ART-2 algorithm learns command patterns and contexts")
	fmt.Println("to provide intelligent predictions and suggestions.")
}

// Helper function to get minimum of two integers
func art2Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}