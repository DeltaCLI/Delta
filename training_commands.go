package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// HandleTrainingCommand processes training-related commands
func HandleTrainingCommand(args []string) bool {
	// Get the training data service
	tds := GetTrainingDataService()
	if tds == nil {
		fmt.Println("Failed to initialize training data service")
		return true
	}

	// Handle commands
	if len(args) == 0 {
		// Show training data status
		showTrainingDataStatus(tds)
		return true
	}

	// Handle subcommands
	if len(args) >= 1 {
		cmd := args[0]

		switch cmd {
		case "extract":
			// Extract training data
			extractTrainingData(tds, args[1:])
			return true

		case "stats":
			// Show detailed stats
			showTrainingDataStats(tds)
			return true

		case "evaluate":
			// Evaluate training data quality
			evaluateTrainingData(tds, args[1:])
			return true

		case "help":
			// Show help
			showTrainingCommandHelp()
			return true

		default:
			fmt.Printf("Unknown training command: %s\n", cmd)
			fmt.Println("Type :training help for a list of available commands")
			return true
		}
	}

	return true
}

// showTrainingDataStatus displays the current status of training data
func showTrainingDataStatus(tds *TrainingDataService) {
	// Get training data stats
	stats := tds.GetTrainingDataStats()

	fmt.Println("Training Data Status")
	fmt.Println("===================")

	fmt.Printf("Total Examples: %d\n", stats["total_examples"].(int))
	fmt.Printf("  Positive: %d\n", stats["positive_examples"].(int))
	fmt.Printf("  Negative: %d\n", stats["negative_examples"].(int))
	fmt.Printf("  Neutral: %d\n", stats["neutral_examples"].(int))

	fmt.Printf("Feedback Entries: %d\n", stats["feedback_count"].(int))
	fmt.Printf("Total Commands: %d\n", stats["total_commands"].(int))

	fmt.Println("\nTraining Readiness:")
	threshold := stats["training_threshold"].(int)
	accumulated := stats["accumulated_examples"].(int)
	isReady := stats["is_training_ready"].(bool)

	fmt.Printf("  Accumulated Examples: %d/%d\n", accumulated, threshold)
	if isReady {
		fmt.Println("  ✓ Ready for training")
		fmt.Println("  Run ':memory train start' to train a new model")
	} else {
		fmt.Printf("  ✗ Need %d more examples\n", threshold-accumulated)
		fmt.Println("  Use ':inference feedback' to add more examples")
	}
}

// showTrainingDataStats displays detailed statistics about training data
func showTrainingDataStats(tds *TrainingDataService) {
	// Get training data stats
	stats := tds.GetTrainingDataStats()

	fmt.Println("Training Data Statistics")
	fmt.Println("=======================")

	fmt.Printf("Total Examples: %d\n", stats["total_examples"].(int))
	fmt.Printf("  Positive: %d (%.1f%%)\n",
		stats["positive_examples"].(int),
		100*float64(stats["positive_examples"].(int))/float64(stats["total_examples"].(int)))
	fmt.Printf("  Negative: %d (%.1f%%)\n",
		stats["negative_examples"].(int),
		100*float64(stats["negative_examples"].(int))/float64(stats["total_examples"].(int)))
	fmt.Printf("  Neutral: %d (%.1f%%)\n",
		stats["neutral_examples"].(int),
		100*float64(stats["neutral_examples"].(int))/float64(stats["total_examples"].(int)))

	// Display source distribution
	fmt.Println("\nSource Distribution:")
	srcDist := stats["source_distribution"].(map[string]int)
	for src, count := range srcDist {
		fmt.Printf("  %s: %d (%.1f%%)\n",
			src, count,
			100*float64(count)/float64(stats["total_examples"].(int)))
	}

	// Display readiness metrics
	fmt.Println("\nTraining Readiness Metrics:")
	fmt.Printf("  Total Feedbacks: %d\n", stats["feedback_count"].(int))
	fmt.Printf("  Command Corpus Size: %d\n", stats["total_commands"].(int))
	fmt.Printf("  Accumulated Examples: %d\n", stats["accumulated_examples"].(int))
	fmt.Printf("  Training Threshold: %d\n", stats["training_threshold"].(int))

	// Status indicator
	isReady := stats["is_training_ready"].(bool)
	if isReady {
		fmt.Println("\n✅ Training data is ready")
		fmt.Println("Run ':memory train start' to train a new model")
	} else {
		remaining := stats["training_threshold"].(int) - stats["accumulated_examples"].(int)
		fmt.Printf("\n⚠️ Need %d more examples for training\n", remaining)
		fmt.Println("Use ':inference feedback' to add more examples")
	}

	// Quality assessment
	fmt.Println("\nData Quality Assessment:")

	// Calculate class balance ratio
	positive := float64(stats["positive_examples"].(int))
	negative := float64(stats["negative_examples"].(int))
	neutral := float64(stats["neutral_examples"].(int))
	total := positive + negative + neutral

	if total > 0 {
		classBalance := 0.0
		if positive > 0 && negative > 0 {
			if positive > negative {
				classBalance = negative / positive
			} else {
				classBalance = positive / negative
			}
		}

		fmt.Printf("  Class Balance: %.2f (", classBalance)
		if classBalance >= 0.8 {
			fmt.Print("Good - classes are well balanced)")
		} else if classBalance >= 0.5 {
			fmt.Print("Fair - slight class imbalance)")
		} else {
			fmt.Print("Poor - significant class imbalance)")
		}
		fmt.Println()
	}

	// Coverage assessment
	coverageRatio := float64(stats["total_examples"].(int)) / float64(stats["total_commands"].(int)+1)
	fmt.Printf("  Command Coverage: %.3f (", coverageRatio)
	if coverageRatio >= 0.1 {
		fmt.Print("Good - feedback covers many commands)")
	} else if coverageRatio >= 0.05 {
		fmt.Print("Fair - moderate command coverage)")
	} else {
		fmt.Print("Limited - feedback covers few commands)")
	}
	fmt.Println()
}

// extractTrainingData extracts training data with specified options
func extractTrainingData(tds *TrainingDataService, args []string) {
	// Default options
	options := TrainingDataOptions{
		Format:          FormatJSON,
		OutputDir:       "",
		IncludeMetadata: true,
		MaxExamples:     -1,
		FilterTypes:     []string{},
		SplitRatio:      0.8,
		BalanceClasses:  false,
		AugmentData:     false,
	}

	// Parse options
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			options.Format = FormatJSON
		case "--csv":
			options.Format = FormatCSV
		case "--tfrecord":
			options.Format = FormatTFRecord
		case "--output":
			if i+1 < len(args) {
				options.OutputDir = args[i+1]
				i++ // Skip the next argument
			}
		case "--limit":
			if i+1 < len(args) {
				limit, err := strconv.Atoi(args[i+1])
				if err == nil && limit > 0 {
					options.MaxExamples = limit
				}
				i++ // Skip the next argument
			}
		case "--split":
			if i+1 < len(args) {
				ratio, err := strconv.ParseFloat(args[i+1], 64)
				if err == nil && ratio > 0 && ratio < 1.0 {
					options.SplitRatio = ratio
				}
				i++ // Skip the next argument
			}
		case "--balance":
			options.BalanceClasses = true
		case "--augment":
			options.AugmentData = true
		case "--no-metadata":
			options.IncludeMetadata = false
		case "--filter":
			if i+1 < len(args) {
				options.FilterTypes = strings.Split(args[i+1], ",")
				i++ // Skip the next argument
			}
		case "--from":
			if i+1 < len(args) {
				date, err := time.Parse("2006-01-02", args[i+1])
				if err == nil {
					options.StartDate = date
				}
				i++ // Skip the next argument
			}
		case "--to":
			if i+1 < len(args) {
				date, err := time.Parse("2006-01-02", args[i+1])
				if err == nil {
					options.EndDate = date
				}
				i++ // Skip the next argument
			}
		}
	}

	// Extract training data
	fmt.Println("Extracting training data...")
	outputPath, err := tds.ExtractTrainingData(options)
	if err != nil {
		fmt.Printf("Error extracting training data: %v\n", err)
		return
	}

	fmt.Printf("Training data extracted successfully to: %s\n", outputPath)

	// Show summary of the extracted data
	stats := tds.GetTrainingDataStats()
	fmt.Printf("Extracted %d examples\n", stats["total_examples"].(int))
	fmt.Printf("  Positive: %d\n", stats["positive_examples"].(int))
	fmt.Printf("  Negative: %d\n", stats["negative_examples"].(int))
	fmt.Printf("  Neutral: %d\n", stats["neutral_examples"].(int))

	// Show next steps
	fmt.Println("\nNext Steps:")
	fmt.Println("1. Use this data for training with ':memory train start'")
	fmt.Println("2. Evaluate data quality with ':training evaluate'")
}

// evaluateTrainingData evaluates training data quality
func evaluateTrainingData(tds *TrainingDataService, args []string) {
	// Default options
	dataPath := ""
	verbose := false

	// Parse options
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--path":
			if i+1 < len(args) {
				dataPath = args[i+1]
				i++ // Skip the next argument
			}
		case "--verbose":
			verbose = true
		}
	}

	// If no path specified, use default location
	if dataPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			fmt.Printf("Error getting home directory: %v\n", err)
			return
		}

		// Try to find the most recent training data
		dataDir := filepath.Join(homeDir, ".config", "delta", "memory", "training_data")
		files, err := os.ReadDir(dataDir)
		if err != nil {
			fmt.Printf("Error reading training data directory: %v\n", err)
			return
		}

		// Find the most recent metadata file
		var newest os.FileInfo
		var newestPath string
		for _, file := range files {
			if strings.HasPrefix(file.Name(), "metadata_") {
				info, err := file.Info()
				if err != nil {
					continue
				}

				if newest == nil || info.ModTime().After(newest.ModTime()) {
					newest = info
					newestPath = filepath.Join(dataDir, file.Name())
				}
			}
		}

		if newestPath != "" {
			dataPath = strings.TrimSuffix(newestPath, filepath.Ext(newestPath))
		} else {
			fmt.Println("No training data found. Extract data first with ':training extract'")
			return
		}
	}

	// Check if the path exists
	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		fmt.Printf("Error: training data not found at %s\n", dataPath)
		return
	}

	fmt.Println("Training Data Evaluation")
	fmt.Println("=======================")
	fmt.Printf("Evaluating data from: %s\n", dataPath)

	// For now, show basic statistics
	// In a full implementation, we'd load and analyze the data in detail
	stats := tds.GetTrainingDataStats()

	fmt.Printf("\nData Size: %d examples\n", stats["total_examples"].(int))

	// Calculate class balance metrics
	positive := float64(stats["positive_examples"].(int))
	negative := float64(stats["negative_examples"].(int))
	neutral := float64(stats["neutral_examples"].(int))
	total := positive + negative + neutral

	if total > 0 {
		fmt.Printf("\nClass Distribution:\n")
		fmt.Printf("  Positive: %.1f%%\n", 100*positive/total)
		fmt.Printf("  Negative: %.1f%%\n", 100*negative/total)
		fmt.Printf("  Neutral: %.1f%%\n", 100*neutral/total)

		classBalance := 0.0
		if positive > 0 && negative > 0 {
			if positive > negative {
				classBalance = negative / positive
			} else {
				classBalance = positive / negative
			}
		}

		fmt.Printf("\nClass Balance Score: %.2f\n", classBalance)
		if classBalance >= 0.8 {
			fmt.Println("  ✓ Good - classes are well balanced")
		} else if classBalance >= 0.5 {
			fmt.Println("  ⚠️ Fair - slight class imbalance")
		} else {
			fmt.Println("  ✗ Poor - significant class imbalance")
		}
	}

	// Source distribution metrics
	fmt.Println("\nSource Distribution:")
	srcDist := stats["source_distribution"].(map[string]int)
	for src, count := range srcDist {
		fmt.Printf("  %s: %d (%.1f%%)\n",
			src, count,
			100*float64(count)/float64(stats["total_examples"].(int)))
	}

	// Coverage assessment
	coverageRatio := float64(stats["total_examples"].(int)) / float64(stats["total_commands"].(int)+1)
	fmt.Printf("\nCommand Coverage: %.3f\n", coverageRatio)
	if coverageRatio >= 0.1 {
		fmt.Println("  ✓ Good - feedback covers many commands")
	} else if coverageRatio >= 0.05 {
		fmt.Println("  ⚠️ Fair - moderate command coverage")
	} else {
		fmt.Println("  ✗ Limited - feedback covers few commands")
	}

	// Overall quality score
	qualityScore := (coverageRatio*5 + classBalance) / 2
	fmt.Printf("\nOverall Quality Score: %.2f/5.0\n", qualityScore)
	if qualityScore >= 3.5 {
		fmt.Println("  ✓ Good - data is ready for training")
	} else if qualityScore >= 2.0 {
		fmt.Println("  ⚠️ Fair - training may work, but results could be improved")
	} else {
		fmt.Println("  ✗ Poor - consider collecting more diverse feedback")
	}

	// Recommendations
	fmt.Println("\nRecommendations:")
	if negative < positive*0.5 {
		fmt.Println("  - Collect more negative feedback for better balance")
	}
	if neutral < total*0.1 {
		fmt.Println("  - Add more correction examples for improved quality")
	}
	if coverageRatio < 0.05 {
		fmt.Println("  - Provide feedback for a wider variety of commands")
	}

	// General recommendation
	fmt.Println("  - Continue collecting feedback with ':inference feedback'")
}

// showTrainingCommandHelp displays help for training commands
func showTrainingCommandHelp() {
	fmt.Println("Training Commands")
	fmt.Println("================")
	fmt.Println("  :training                  - Show training data status")
	fmt.Println("  :training stats            - Show detailed statistics")
	fmt.Println("  :training extract [options] - Extract training data")
	fmt.Println("  :training evaluate [options] - Evaluate training data quality")
	fmt.Println("  :training help             - Show this help message")

	fmt.Println("\nExtract Options:")
	fmt.Println("  --json                     - Use JSON format (default)")
	fmt.Println("  --csv                      - Use CSV format")
	fmt.Println("  --tfrecord                 - Use TFRecord format (not implemented)")
	fmt.Println("  --output <dir>             - Specify output directory")
	fmt.Println("  --limit <n>                - Limit to n examples")
	fmt.Println("  --split <ratio>            - Train/val split ratio (default: 0.8)")
	fmt.Println("  --balance                  - Balance positive and negative examples")
	fmt.Println("  --augment                  - Augment data with synthetic examples")
	fmt.Println("  --no-metadata              - Don't include metadata")
	fmt.Println("  --filter <types>           - Filter by feedback types (helpful,unhelpful,correction)")
	fmt.Println("  --from <date>              - Start date (YYYY-MM-DD)")
	fmt.Println("  --to <date>                - End date (YYYY-MM-DD)")

	fmt.Println("\nEvaluate Options:")
	fmt.Println("  --path <path>              - Path to training data directory")
	fmt.Println("  --verbose                  - Show detailed evaluation")
}
