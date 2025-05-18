package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// HandleEvaluationCommand processes evaluation-related commands
func HandleEvaluationCommand(args []string) bool {
	// Get the model evaluator
	evaluator := GetModelEvaluator()
	if evaluator == nil {
		fmt.Println("Failed to initialize model evaluator")
		return true
	}

	// Handle commands
	if len(args) == 0 {
		// Show evaluation status
		showEvaluationStatus(evaluator)
		return true
	}

	// Handle subcommands
	if len(args) >= 1 {
		cmd := args[0]

		switch cmd {
		case "run":
			// Run model evaluation
			runModelEvaluation(evaluator, args[1:])
			return true

		case "list":
			// List evaluations
			listEvaluations(evaluator)
			return true

		case "show":
			// Show evaluation details
			if len(args) >= 2 {
				showEvaluationDetails(evaluator, args[1])
			} else {
				fmt.Println("Usage: :evaluation show <evaluation_id>")
			}
			return true

		case "compare":
			// Compare evaluations
			if len(args) >= 2 {
				compareEvaluations(evaluator, args[1:])
			} else {
				fmt.Println("Usage: :evaluation compare <eval_id1> [eval_id2 ...]")
			}
			return true

		case "help":
			// Show help
			showEvaluationHelp()
			return true

		default:
			fmt.Printf("Unknown evaluation command: %s\n", cmd)
			fmt.Println("Type :evaluation help for a list of available commands")
			return true
		}
	}

	return true
}

// showEvaluationStatus displays the current status of model evaluations
func showEvaluationStatus(evaluator *ModelEvaluator) {
	fmt.Println("Model Evaluation Status")
	fmt.Println("======================")

	// Check if evaluator is properly initialized
	if evaluator == nil {
		fmt.Println("Error: Model evaluator not available")
		return
	}

	// List recent evaluations
	evaluations, err := evaluator.ListEvaluations()
	if err != nil {
		fmt.Printf("Error listing evaluations: %v\n", err)
		return
	}

	if len(evaluations) == 0 {
		fmt.Println("No model evaluations found")
		fmt.Println("Run ':evaluation run' to evaluate a model")
		return
	}

	// Sort evaluations by modification time (newest first)
	sort.Slice(evaluations, func(i, j int) bool {
		infoI, _ := os.Stat(evaluations[i])
		infoJ, _ := os.Stat(evaluations[j])
		return infoI.ModTime().After(infoJ.ModTime())
	})

	fmt.Printf("Found %d model evaluations\n", len(evaluations))
	fmt.Println("\nRecent Evaluations:")

	// Show the 5 most recent evaluations
	count := 5
	if len(evaluations) < count {
		count = len(evaluations)
	}

	for i := 0; i < count; i++ {
		evalPath := evaluations[i]
		filename := filepath.Base(evalPath)
		parts := strings.Split(filename, "_eval_")
		
		if len(parts) >= 2 {
			modelName := parts[0]
			dateStr := strings.TrimSuffix(parts[1], ".json")
			
			// Try to parse date
			date, err := time.Parse("20060102_150405", dateStr)
			if err == nil {
				dateStr = date.Format("2006-01-02 15:04:05")
			}
			
			fmt.Printf("  %d. Model: %s, Date: %s\n", i+1, modelName, dateStr)
		} else {
			fmt.Printf("  %d. %s\n", i+1, filename)
		}
	}

	fmt.Println("\nUse ':evaluation show <id>' to see details")
	fmt.Println("Use ':evaluation compare <id1> <id2>' to compare models")
	fmt.Println("Use ':evaluation run' to evaluate a new model")
}

// runModelEvaluation runs a model evaluation with specified options
func runModelEvaluation(evaluator *ModelEvaluator, args []string) {
	// Default evaluation configuration
	config := ModelEvaluationConfig{
		ModelPath:      "",
		TestDataPath:   "",
		ModelType:      "onnx",
		Metrics:        []ModelEvaluationMetric{MetricAccuracy, MetricPrecision, MetricRecall, MetricF1Score, MetricConfusionMatrix},
		OutputDir:      "",
		BatchSize:      16,
		ClassThreshold: 0.5,
	}

	// Parse options
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--model":
			if i+1 < len(args) {
				config.ModelPath = args[i+1]
				i++ // Skip the next argument
			}
		case "--test-data":
			if i+1 < len(args) {
				config.TestDataPath = args[i+1]
				i++ // Skip the next argument
			}
		case "--model-type":
			if i+1 < len(args) {
				config.ModelType = args[i+1]
				i++ // Skip the next argument
			}
		case "--batch-size":
			if i+1 < len(args) {
				var batchSize int
				fmt.Sscanf(args[i+1], "%d", &batchSize)
				if batchSize > 0 {
					config.BatchSize = batchSize
				}
				i++ // Skip the next argument
			}
		case "--threshold":
			if i+1 < len(args) {
				var threshold float64
				fmt.Sscanf(args[i+1], "%f", &threshold)
				if threshold > 0 && threshold <= 1.0 {
					config.ClassThreshold = threshold
				}
				i++ // Skip the next argument
			}
		case "--output":
			if i+1 < len(args) {
				config.OutputDir = args[i+1]
				i++ // Skip the next argument
			}
		}
	}

	// If no model path specified, use default model
	if config.ModelPath == "" {
		// Find the latest trained model
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("Error getting home directory: %v\n", err)
			return
		}
		
		modelsDir := filepath.Join(homeDir, ".config", "delta", "memory", "models")
		files, err := os.ReadDir(modelsDir)
		if err != nil {
			fmt.Printf("Error reading models directory: %v\n", err)
			return
		}
		
		// Find most recent model file
		var newest os.FileInfo
		var newestPath string
		for _, file := range files {
			if strings.HasSuffix(file.Name(), ".onnx") || strings.HasSuffix(file.Name(), ".bin") {
				info, err := file.Info()
				if err != nil {
					continue
				}
				
				if newest == nil || info.ModTime().After(newest.ModTime()) {
					newest = info
					newestPath = filepath.Join(modelsDir, file.Name())
				}
			}
		}
		
		if newestPath != "" {
			config.ModelPath = newestPath
		} else {
			fmt.Println("No trained models found")
			fmt.Println("Train a model first with ':memory train start'")
			return
		}
	}

	// Run the evaluation
	fmt.Println("Running model evaluation...")
	fmt.Printf("Model: %s\n", config.ModelPath)
	if config.TestDataPath != "" {
		fmt.Printf("Test Data: %s\n", config.TestDataPath)
	} else {
		fmt.Println("Test Data: Using latest available test data")
	}

	// Start evaluation
	result, err := evaluator.EvaluateModel(config)
	if err != nil {
		fmt.Printf("Evaluation failed: %v\n", err)
		return
	}

	// Show results summary
	fmt.Println("\nEvaluation Results:")
	fmt.Printf("Accuracy: %.4f\n", result.Metrics["accuracy"])
	fmt.Printf("Precision: %.4f\n", result.Metrics["precision"])
	fmt.Printf("Recall: %.4f\n", result.Metrics["recall"])
	fmt.Printf("F1 Score: %.4f\n", result.Metrics["f1_score"])

	// Show confusion matrix if available
	if result.ConfusionMatrix != nil && len(result.ConfusionMatrix) >= 2 {
		fmt.Println("\nConfusion Matrix:")
		fmt.Printf("  TN: %d | FN: %d\n", result.ConfusionMatrix[0][0], result.ConfusionMatrix[0][1])
		fmt.Printf("  FP: %d | TP: %d\n", result.ConfusionMatrix[1][0], result.ConfusionMatrix[1][1])
	}

	// Show a sample of incorrect predictions
	incorrectCount := 0
	for _, eval := range result.ExampleResults {
		if !eval.Correct {
			incorrectCount++
		}
	}

	fmt.Printf("\nExamples Evaluated: %d\n", len(result.ExampleResults))
	fmt.Printf("Correct: %d (%.1f%%)\n", len(result.ExampleResults)-incorrectCount, 
		100.0*float64(len(result.ExampleResults)-incorrectCount)/float64(len(result.ExampleResults)))
	fmt.Printf("Incorrect: %d (%.1f%%)\n", incorrectCount, 
		100.0*float64(incorrectCount)/float64(len(result.ExampleResults)))

	// Show error analysis if available
	if result.ErrorAnalysis != nil && len(result.ErrorAnalysis) > 0 {
		fmt.Println("\nError Analysis:")
		for errorType, count := range result.ErrorAnalysis {
			fmt.Printf("  %s: %d\n", errorType, count)
		}
	}

	// Show where the full results are saved
	fmt.Printf("\nDetailed evaluation results saved to:\n%s\n", evaluator.outputDir)
}

// listEvaluations lists all available evaluations
func listEvaluations(evaluator *ModelEvaluator) {
	// Get all evaluations
	evaluations, err := evaluator.ListEvaluations()
	if err != nil {
		fmt.Printf("Error listing evaluations: %v\n", err)
		return
	}

	if len(evaluations) == 0 {
		fmt.Println("No model evaluations found")
		fmt.Println("Run ':evaluation run' to evaluate a model")
		return
	}

	// Sort evaluations by modification time (newest first)
	sort.Slice(evaluations, func(i, j int) bool {
		infoI, _ := os.Stat(evaluations[i])
		infoJ, _ := os.Stat(evaluations[j])
		return infoI.ModTime().After(infoJ.ModTime())
	})

	fmt.Println("Available Model Evaluations")
	fmt.Println("==========================")

	for i, evalPath := range evaluations {
		filename := filepath.Base(evalPath)
		info, err := os.Stat(evalPath)
		
		if err == nil {
			// Extract model name and date
			parts := strings.Split(filename, "_eval_")
			if len(parts) >= 2 {
				modelName := parts[0]
				dateStr := strings.TrimSuffix(parts[1], ".json")
				
				// Try to parse date
				date, err := time.Parse("20060102_150405", dateStr)
				dateFormatted := dateStr
				if err == nil {
					dateFormatted = date.Format("2006-01-02 15:04:05")
				}
				
				fmt.Printf("%d. Model: %s\n", i+1, modelName)
				fmt.Printf("   Date: %s\n", dateFormatted)
				fmt.Printf("   Size: %.2f KB\n", float64(info.Size())/1024.0)
				fmt.Printf("   Path: %s\n", evalPath)
			} else {
				fmt.Printf("%d. %s\n", i+1, filename)
				fmt.Printf("   Date: %s\n", info.ModTime().Format("2006-01-02 15:04:05"))
				fmt.Printf("   Size: %.2f KB\n", float64(info.Size())/1024.0)
				fmt.Printf("   Path: %s\n", evalPath)
			}
			
			if i < len(evaluations)-1 {
				fmt.Println()
			}
		}
	}
	
	fmt.Println("\nUse ':evaluation show <id>' to see details of an evaluation")
	fmt.Println("For example: :evaluation show 1")
}

// showEvaluationDetails shows details of a specific evaluation
func showEvaluationDetails(evaluator *ModelEvaluator, idStr string) {
	// Get all evaluations
	evaluations, err := evaluator.ListEvaluations()
	if err != nil {
		fmt.Printf("Error listing evaluations: %v\n", err)
		return
	}

	if len(evaluations) == 0 {
		fmt.Println("No model evaluations found")
		fmt.Println("Run ':evaluation run' to evaluate a model")
		return
	}

	// Sort evaluations by modification time (newest first)
	sort.Slice(evaluations, func(i, j int) bool {
		infoI, _ := os.Stat(evaluations[i])
		infoJ, _ := os.Stat(evaluations[j])
		return infoI.ModTime().After(infoJ.ModTime())
	})

	// Parse evaluation ID
	var id int
	if idStr == "latest" {
		id = 1
	} else {
		fmt.Sscanf(idStr, "%d", &id)
	}

	// Validate ID
	if id <= 0 || id > len(evaluations) {
		fmt.Printf("Invalid evaluation ID: %s\n", idStr)
		fmt.Printf("Valid IDs are 1-%d or 'latest'\n", len(evaluations))
		return
	}

	// Get the evaluation path
	evalPath := evaluations[id-1]

	// Look for the summary file first
	summaryPath := strings.TrimSuffix(evalPath, ".json") + "_summary.txt"
	if _, err := os.Stat(summaryPath); err == nil {
		// Read the summary file
		data, err := os.ReadFile(summaryPath)
		if err == nil {
			// Display the summary
			fmt.Println(string(data))
			return
		}
	}

	// If no summary file, read the JSON file
	data, err := os.ReadFile(evalPath)
	if err != nil {
		fmt.Printf("Error reading evaluation file: %v\n", err)
		return
	}

	// Just print the first N lines to keep it manageable
	lines := strings.Split(string(data), "\n")
	lineCount := 50
	if len(lines) < lineCount {
		lineCount = len(lines)
	}

	for i := 0; i < lineCount; i++ {
		fmt.Println(lines[i])
	}

	if len(lines) > lineCount {
		fmt.Printf("\n... (%d more lines)\n", len(lines)-lineCount)
	}
}

// compareEvaluations compares multiple evaluations
func compareEvaluations(evaluator *ModelEvaluator, args []string) {
	// Get all evaluations
	evaluations, err := evaluator.ListEvaluations()
	if err != nil {
		fmt.Printf("Error listing evaluations: %v\n", err)
		return
	}

	if len(evaluations) == 0 {
		fmt.Println("No model evaluations found")
		fmt.Println("Run ':evaluation run' to evaluate a model")
		return
	}

	// Sort evaluations by modification time (newest first)
	sort.Slice(evaluations, func(i, j int) bool {
		infoI, _ := os.Stat(evaluations[i])
		infoJ, _ := os.Stat(evaluations[j])
		return infoI.ModTime().After(infoJ.ModTime())
	})

	// Parse evaluation IDs
	evalPaths := make([]string, 0)
	for _, arg := range args {
		if arg == "latest" {
			evalPaths = append(evalPaths, evaluations[0])
			continue
		}

		var id int
		fmt.Sscanf(arg, "%d", &id)

		// Validate ID
		if id <= 0 || id > len(evaluations) {
			fmt.Printf("Invalid evaluation ID: %s\n", arg)
			fmt.Printf("Valid IDs are 1-%d or 'latest'\n", len(evaluations))
			continue
		}

		// Add the evaluation path
		evalPaths = append(evalPaths, evaluations[id-1])
	}

	// Need at least 2 evaluations to compare
	if len(evalPaths) < 2 {
		fmt.Println("Need at least 2 valid evaluations to compare")
		fmt.Println("Usage: :evaluation compare <id1> <id2> [id3 ...]")
		return
	}

	// Generate comparison report
	report, err := evaluator.CompareModels(evalPaths)
	if err != nil {
		fmt.Printf("Error comparing models: %v\n", err)
		return
	}

	// Display the report
	fmt.Println(report)
}

// showEvaluationHelp displays help for evaluation commands
func showEvaluationHelp() {
	fmt.Println("Evaluation Commands")
	fmt.Println("==================")
	fmt.Println("  :evaluation                - Show evaluation status")
	fmt.Println("  :evaluation run [options]  - Run model evaluation")
	fmt.Println("  :evaluation list           - List all evaluations")
	fmt.Println("  :evaluation show <id>      - Show evaluation details")
	fmt.Println("  :evaluation compare <id1> <id2> [id3 ...] - Compare evaluations")
	fmt.Println("  :evaluation help           - Show this help message")
	
	fmt.Println("\nRun Options:")
	fmt.Println("  --model <path>      - Path to the model to evaluate")
	fmt.Println("  --test-data <path>  - Path to test data (uses latest if not specified)")
	fmt.Println("  --model-type <type> - Type of model (onnx, pytorch)")
	fmt.Println("  --batch-size <n>    - Batch size for evaluation")
	fmt.Println("  --threshold <n>     - Threshold for positive class prediction")
	fmt.Println("  --output <dir>      - Directory for evaluation results")
	
	fmt.Println("\nEvaluation IDs:")
	fmt.Println("  IDs are assigned by order, with 1 being the most recent")
	fmt.Println("  You can also use 'latest' to refer to the most recent evaluation")
}