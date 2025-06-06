package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// HandleMemoryCommand processes the memory command with arguments
func HandleMemoryCommand(args []string) bool {
	// Get the MemoryManager instance
	mm := GetMemoryManager()

	// Initialize if not already done
	if !mm.isInitialized {
		err := mm.Initialize()
		if err != nil {
			fmt.Printf("Error initializing memory manager: %v\n", err)
			return true
		}
	}

	// Handle commands
	if len(args) == 0 {
		// Show memory status
		showMemoryStatus(mm)
		return true
	}

	// Handle special commands
	if len(args) >= 1 {
		cmd := args[0]

		switch cmd {
		case "enable":
			// Enable memory collection
			err := mm.Enable()
			if err != nil {
				fmt.Printf("Error enabling memory system: %v\n", err)
			} else {
				fmt.Println("Memory system enabled")
			}
			return true

		case "disable":
			// Disable memory collection
			err := mm.Disable()
			if err != nil {
				fmt.Printf("Error disabling memory system: %v\n", err)
			} else {
				fmt.Println("Memory system disabled")
			}
			return true

		case "status":
			// Show status
			showMemoryStatus(mm)
			return true

		case "stats":
			// Show detailed stats
			showMemoryStats(mm)
			return true

		case "clear":
			// Clear all data
			if len(args) >= 2 && args[1] == "confirm" {
				err := mm.ClearData()
				if err != nil {
					fmt.Printf("Error clearing memory data: %v\n", err)
				} else {
					fmt.Println("Memory data cleared")
				}
			} else {
				fmt.Println("Warning: This will delete all collected command data.")
				fmt.Println("To confirm, use: :memory clear confirm")
			}
			return true

		case "config":
			// Show or update configuration
			if len(args) >= 3 && args[1] == "set" {
				// Update config
				updateMemoryConfig(mm, args[2:])
			} else {
				// Show config
				showMemoryConfig(mm)
			}
			return true

		case "list":
			// List shards of data
			listMemoryShards(mm)
			return true

		case "export":
			// Enhanced export with options
			if len(args) >= 2 && args[1] == "help" {
				showExportHelp()
				return true
			}

			// Parse export options
			exportOptions := parseExportOptions(args[1:])
			exportMemoryData(mm, exportOptions)
			return true

		case "import":
			// Import memory data
			if len(args) >= 2 {
				if args[1] == "help" {
					showImportHelp()
					return true
				}
				importMemoryData(mm, args[1:])
			} else {
				fmt.Println("Usage: :memory import <path> [--with-config]")
				fmt.Println("Use ':memory import help' for more information")
			}
			return true

		case "train":
			// Handle training commands
			if len(args) >= 2 {
				switch args[1] {
				case "start":
					fmt.Println("Starting Docker training environment...")
					runTrainingContainer(mm, args[2:])
					return true

				case "status":
					showTrainingStatus(mm)
					return true

				case "add":
					if len(args) >= 4 {
						addTrainingExample(mm, args[2], args[3])
					} else {
						fmt.Println("Usage: :memory train add <pattern> <explanation>")
					}
					return true

				case "feedback":
					if len(args) >= 3 {
						addTrainingFeedback(mm, args[2])
					} else {
						fmt.Println("Usage: :memory train feedback <helpful|unhelpful>")
					}
					return true

				case "docker":
					fmt.Println("Setting up Docker training environment...")
					runTrainingContainer(mm, args[2:])
					return true
				}
			}
			fmt.Println("Usage: :memory train [start|docker|status|add|feedback]")
			return true
		}

		// If we get here, it's an unknown command
		fmt.Println("Unknown memory command. Available commands:")
		fmt.Println("  :memory status     - Show memory system status")
		fmt.Println("  :memory enable     - Enable memory collection")
		fmt.Println("  :memory disable    - Disable memory collection")
		fmt.Println("  :memory stats      - Show detailed stats")
		fmt.Println("  :memory clear      - Clear all data")
		fmt.Println("  :memory config     - Show configuration")
		fmt.Println("  :memory list       - List available data shards")
		fmt.Println("  :memory export     - Export data with options")
		fmt.Println("  :memory import     - Import data from an export")
		fmt.Println("  :memory train      - Training commands")
		return true
	}

	return true
}

// showMemoryStatus displays the current status of the memory system
func showMemoryStatus(mm *MemoryManager) {
	fmt.Println("Memory System Status")
	fmt.Println("===================")
	fmt.Printf("Enabled: %v\n", mm.IsEnabled())
	fmt.Printf("Command Collection: %v\n", mm.config.CollectCommands)

	stats, err := mm.GetStats()
	if err != nil {
		fmt.Printf("Error getting stats: %v\n", err)
		return
	}

	fmt.Printf("Total Commands: %d\n", stats.TotalEntries)
	if !stats.FirstEntry.IsZero() {
		fmt.Printf("First Command: %s\n", stats.FirstEntry.Format(time.RFC1123))
		fmt.Printf("Last Command: %s\n", stats.LastEntry.Format(time.RFC1123))
	} else {
		fmt.Println("No commands collected yet")
	}

	fmt.Printf("Storage Used: %.2f MB\n", float64(stats.DiskUsage)/(1024*1024))
}

// showMemoryStats displays detailed statistics about collected data
func showMemoryStats(mm *MemoryManager) {
	stats, err := mm.GetStats()
	if err != nil {
		fmt.Printf("Error getting stats: %v\n", err)
		return
	}

	fmt.Println("Memory System Statistics")
	fmt.Println("=======================")
	fmt.Printf("Total Commands: %d\n", stats.TotalEntries)

	if !stats.FirstEntry.IsZero() {
		fmt.Printf("Data Span: %s to %s\n",
			stats.FirstEntry.Format("2006-01-02"),
			stats.LastEntry.Format("2006-01-02"))

		days := int(stats.LastEntry.Sub(stats.FirstEntry).Hours()/24) + 1
		fmt.Printf("Days Covered: %d\n", days)
		fmt.Printf("Average Commands/Day: %.1f\n", float64(stats.TotalEntries)/float64(days))
	} else {
		fmt.Println("No commands collected yet")
	}

	fmt.Printf("Storage Used: %.2f MB\n", float64(stats.DiskUsage)/(1024*1024))

	// Model information
	fmt.Println("\nModel Information")
	fmt.Println("-----------------")
	if !stats.LastTraining.IsZero() {
		fmt.Printf("Last Training: %s\n", stats.LastTraining.Format(time.RFC1123))
		fmt.Printf("Days Since Training: %d\n",
			int(time.Since(stats.LastTraining).Hours()/24))
	} else {
		fmt.Println("No training has been performed yet")
	}

	if len(stats.ModelVersions) > 0 {
		fmt.Println("Available Models:")
		for _, model := range stats.ModelVersions {
			fmt.Printf("  - %s\n", model)
		}
	} else {
		fmt.Println("No models available")
	}
}

// showMemoryConfig displays the current configuration of the memory system
func showMemoryConfig(mm *MemoryManager) {
	fmt.Println("Memory System Configuration")
	fmt.Println("==========================")
	fmt.Printf("Enabled: %v\n", mm.config.Enabled)
	fmt.Printf("Collect Commands: %v\n", mm.config.CollectCommands)
	fmt.Printf("Max Entries: %d\n", mm.config.MaxEntries)
	fmt.Printf("Storage Path: %s\n", mm.config.StoragePath)

	fmt.Println("\nPrivacy Settings")
	fmt.Println("----------------")
	fmt.Printf("Privacy Filters: %s\n", strings.Join(mm.config.PrivacyFilter, ", "))
	fmt.Printf("Collect Environment: %v\n", mm.config.CollectEnvironment)
	if mm.config.CollectEnvironment {
		fmt.Printf("Environment Whitelist: %s\n", strings.Join(mm.config.EnvWhitelist, ", "))
	}

	fmt.Println("\nTraining Settings")
	fmt.Println("----------------")
	fmt.Printf("Training Enabled: %v\n", mm.config.TrainingEnabled)
	fmt.Printf("Model Path: %s\n", mm.config.ModelPath)
}

// updateMemoryConfig updates a specific configuration setting
func updateMemoryConfig(mm *MemoryManager, args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: :memory config set <setting> <value>")
		return
	}

	setting := args[0]
	value := args[1]

	config := mm.config

	switch setting {
	case "collect_commands":
		config.CollectCommands = (value == "true" || value == "1" || value == "yes")

	case "collect_environment":
		config.CollectEnvironment = (value == "true" || value == "1" || value == "yes")

	case "training_enabled":
		config.TrainingEnabled = (value == "true" || value == "1" || value == "yes")

	case "max_entries":
		var maxEntries int
		fmt.Sscanf(value, "%d", &maxEntries)
		if maxEntries > 0 {
			config.MaxEntries = maxEntries
		} else {
			fmt.Println("Error: max_entries must be a positive number")
			return
		}

	case "add_privacy_filter":
		if !contains(config.PrivacyFilter, value) {
			config.PrivacyFilter = append(config.PrivacyFilter, value)
		}

	case "remove_privacy_filter":
		config.PrivacyFilter = removeString(config.PrivacyFilter, value)

	case "add_env_whitelist":
		if !contains(config.EnvWhitelist, value) {
			config.EnvWhitelist = append(config.EnvWhitelist, value)
		}

	case "remove_env_whitelist":
		config.EnvWhitelist = removeString(config.EnvWhitelist, value)

	default:
		fmt.Printf("Unknown setting: %s\n", setting)
		return
	}

	err := mm.UpdateConfig(config)
	if err != nil {
		fmt.Printf("Error updating configuration: %v\n", err)
	} else {
		fmt.Printf("Updated %s to %s\n", setting, value)
	}
}

// listMemoryShards lists all available data shards
func listMemoryShards(mm *MemoryManager) {
	// Get list of all shard files
	entries, err := os.ReadDir(mm.config.StoragePath)
	if err != nil {
		fmt.Printf("Error reading storage directory: %v\n", err)
		return
	}

	fmt.Println("Available Data Shards")
	fmt.Println("====================")

	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "commands_") {
			continue
		}

		// Extract date from filename
		date := strings.TrimPrefix(entry.Name(), "commands_")
		date = strings.TrimSuffix(date, ".bin")

		// Get file size
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Get entry count
		shardPath := filepath.Join(mm.config.StoragePath, entry.Name())
		entriesCount, _, _, err := mm.getShardStats(shardPath)

		fmt.Printf("%s: %d commands (%.2f MB)\n",
			date, entriesCount, float64(info.Size())/(1024*1024))
		count++
	}

	if count == 0 {
		fmt.Println("No data shards available")
	}
}

// Parse export options from command-line arguments
func parseExportOptions(args []string) ExportOptions {
	options := ExportOptions{
		Format:      "binary", // Default format
		IncludeAll:  false,
		Destination: "",
	}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			options.Format = "json"
		case "--binary":
			options.Format = "binary"
		case "--all":
			options.IncludeAll = true
		case "--output":
			if i+1 < len(args) {
				options.Destination = args[i+1]
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
		default:
			// Check if it's a date in YYYY-MM-DD format
			if date, err := time.Parse("2006-01-02", args[i]); err == nil {
				// If both start and end are not set, use as start date
				// If start is set but end isn't, use as end date
				if options.StartDate.IsZero() {
					options.StartDate = date
				} else if options.EndDate.IsZero() {
					options.EndDate = date
				}
			}
		}
	}

	return options
}

// showExportHelp displays help for the export command
func showExportHelp() {
	fmt.Println("Memory Export Help")
	fmt.Println("=================")
	fmt.Println("Export memory data to a specified format and location.")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  :memory export [options]")
	fmt.Println("  :memory export <date>")
	fmt.Println("  :memory export <start_date> <end_date>")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --json           Export in JSON format (default is binary)")
	fmt.Println("  --binary         Export in binary format (default)")
	fmt.Println("  --all            Include configuration in export")
	fmt.Println("  --output <dir>   Specify output directory")
	fmt.Println("  --from <date>    Start date (YYYY-MM-DD)")
	fmt.Println("  --to <date>      End date (YYYY-MM-DD)")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  :memory export                    # Export all data in binary format")
	fmt.Println("  :memory export 2023-01-15         # Export data for specific date")
	fmt.Println("  :memory export --json             # Export all data in JSON format")
	fmt.Println("  :memory export --all --json       # Export all data and config in JSON format")
	fmt.Println("  :memory export 2023-01-01 2023-01-31 # Export date range")
	fmt.Println("  :memory export --from 2023-01-01 --to 2023-01-31 # Same as above")
}

// showImportHelp displays help for the import command
func showImportHelp() {
	fmt.Println("Memory Import Help")
	fmt.Println("=================")
	fmt.Println("Import memory data from a previously exported backup.")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  :memory import <path> [options]")
	fmt.Println("")
	fmt.Println("Options:")
	fmt.Println("  --with-config    Import configuration (if available in export)")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  :memory import ~/.config/delta/memory/exports/export_20230115_120000")
	fmt.Println("  :memory import /tmp/delta_backup --with-config")
	fmt.Println("")
	fmt.Println("Notes:")
	fmt.Println("  - Import path must be a directory containing metadata.json")
	fmt.Println("  - Existing data for imported dates will be overwritten")
	fmt.Println("  - Configuration is not imported by default")
}

// exportMemoryData exports data using the new ExportMemory function
func exportMemoryData(mm *MemoryManager, options ExportOptions) {
	// Legacy mode for simpler date-based export
	if !options.StartDate.IsZero() && options.EndDate.IsZero() {
		// If only start date is set, export a single day
		date := options.StartDate.Format("2006-01-02")

		// Check if we should use the legacy export
		if options.Format == "txt" && options.Destination == "" && !options.IncludeAll {
			entries, err := mm.ReadCommands(date)
			if err != nil {
				fmt.Printf("Error reading commands: %v\n", err)
				return
			}

			if len(entries) == 0 {
				fmt.Printf("No commands found for date: %s\n", date)
				return
			}

			// Create export directory if it doesn't exist
			exportDir := filepath.Join(mm.config.StoragePath, "exports")
			err = os.MkdirAll(exportDir, 0755)
			if err != nil {
				fmt.Printf("Error creating export directory: %v\n", err)
				return
			}

			// Create export file
			exportPath := filepath.Join(exportDir, "export_"+date+".txt")
			file, err := os.Create(exportPath)
			if err != nil {
				fmt.Printf("Error creating export file: %v\n", err)
				return
			}
			defer file.Close()

			// Write header
			file.WriteString(fmt.Sprintf("# Command Export for %s\n", date))
			file.WriteString(fmt.Sprintf("# Total Commands: %d\n\n", len(entries)))

			// Write each command
			for i, entry := range entries {
				file.WriteString(fmt.Sprintf("## Command %d\n", i+1))
				file.WriteString(fmt.Sprintf("Time: %s\n", entry.Timestamp.Format(time.RFC3339)))
				file.WriteString(fmt.Sprintf("Directory: %s\n", entry.Directory))
				file.WriteString(fmt.Sprintf("Command: %s\n", entry.Command))
				file.WriteString(fmt.Sprintf("Exit Code: %d\n", entry.ExitCode))
				file.WriteString(fmt.Sprintf("Duration: %d ms\n", entry.Duration))

				if entry.PrevCommand != "" {
					file.WriteString(fmt.Sprintf("Previous Command: %s\n", entry.PrevCommand))
				}

				if len(entry.Environment) > 0 {
					file.WriteString("Environment:\n")
					for k, v := range entry.Environment {
						file.WriteString(fmt.Sprintf("  %s: %s\n", k, v))
					}
				}

				file.WriteString("\n")
			}

			fmt.Printf("Exported %d commands to %s\n", len(entries), exportPath)
			return
		}
	}

	// Use the enhanced export system for everything else
	exportPath, err := mm.ExportMemory(options)
	if err != nil {
		fmt.Printf("Error exporting memory data: %v\n", err)
		return
	}

	fmt.Printf("Memory data exported to: %s\n", exportPath)
}

// importMemoryData imports memory data from an export
func importMemoryData(mm *MemoryManager, args []string) {
	if len(args) == 0 {
		fmt.Println("Please specify the path to the export directory")
		return
	}

	importPath := args[0]
	options := make(map[string]bool)

	// Parse options
	for _, arg := range args[1:] {
		if arg == "--with-config" {
			options["import_config"] = true
		}
	}

	// Validate import path
	if _, err := os.Stat(importPath); os.IsNotExist(err) {
		fmt.Printf("Error: Import path does not exist: %s\n", importPath)
		return
	}

	// Confirm import
	fmt.Println("Warning: Importing may overwrite existing data.")
	fmt.Print("Do you want to continue? (y/n): ")
	var confirm string
	fmt.Scanln(&confirm)
	if strings.ToLower(confirm) != "y" {
		fmt.Println("Import cancelled")
		return
	}

	// Perform import
	err := mm.ImportMemory(importPath, options)
	if err != nil {
		fmt.Printf("Error importing memory data: %v\n", err)
		return
	}

	fmt.Println("Memory data imported successfully")
}

// showTrainingStatus displays the current training status
func showTrainingStatus(mm *MemoryManager) {
	// Get memory stats
	memStats, err := mm.GetStats()
	if err != nil {
		fmt.Printf("Error getting memory stats: %v\n", err)
		return
	}

	// Get inference manager to get training status
	im := GetInferenceManager()
	if im == nil {
		fmt.Println("Inference manager not available")
		return
	}

	// Get inference stats
	infStats := im.GetInferenceStats()

	fmt.Println("Training Status")
	fmt.Println("==============")
	fmt.Printf("Training Enabled: %v\n", mm.config.TrainingEnabled)

	// Show learning system status
	fmt.Printf("Learning System: %s\n", getBoolStatus(infStats["learning_enabled"].(bool)))
	fmt.Printf("Feedback Collection: %s\n", getBoolStatus(infStats["feedback_collection"].(bool)))

	// Show last training info
	if !memStats.LastTraining.IsZero() {
		fmt.Printf("Last Training: %s\n", memStats.LastTraining.Format(time.RFC1123))
		fmt.Printf("Time Since Training: %s\n", formatDuration(time.Since(memStats.LastTraining)))
	} else {
		fmt.Println("No training has been performed yet")
	}

	// Show model information
	fmt.Println("\nModel Information:")
	fmt.Printf("  Custom Model: %s\n", getBoolStatus(infStats["custom_model_enabled"].(bool)))
	if infStats["custom_model_enabled"].(bool) {
		fmt.Printf("  Model Path: %s\n", infStats["model_path"])

		if infStats["custom_model_available"].(bool) {
			fmt.Println("  Model Status: Available")
		} else {
			fmt.Println("  Model Status: Not found")
		}
	}

	// Show available models
	if len(memStats.ModelVersions) > 0 {
		fmt.Println("\nAvailable Models:")
		for _, model := range memStats.ModelVersions {
			fmt.Printf("  - %s\n", model)
		}
	} else {
		fmt.Println("\nNo models available")
	}

	// Show training data
	fmt.Println("\nTraining Data:")
	fmt.Printf("  Commands Available: %d\n", memStats.TotalEntries)
	fmt.Printf("  Training Examples: %d\n", infStats["training_examples"].(int))
	fmt.Printf("  Feedback Entries: %d\n", infStats["feedback_count"].(int))
	fmt.Printf("  Accumulated Examples: %d\n", infStats["accumulated_examples"].(int))

	// Show next steps based on training status
	fmt.Println("\nNext Steps:")
	if im.ShouldTrain() {
		fmt.Println("  ✓ Ready for training! Run ':memory train start' to train a new model")
		fmt.Println("  ✓ Enough data has been collected for effective training")
	} else if infStats["accumulated_examples"].(int) < 100 {
		remaining := 100 - infStats["accumulated_examples"].(int)
		fmt.Printf("  ✗ Need %d more examples before training can begin\n", remaining)
		fmt.Println("  ✓ Use ':memory train add' or ':inference feedback' to add more data")
	} else {
		// We have enough examples but not yet due for periodic training
		if infStats["periodic_training"].(bool) {
			fmt.Println("  ✓ Enough data has been collected for training")
			fmt.Println("  ✗ Not yet due for periodic training (it's too soon since last training)")
			fmt.Println("  ✓ Use ':memory train start --force' to train anyway")
		} else {
			fmt.Println("  ✓ Enough data has been collected for training")
			fmt.Println("  ✓ Run ':memory train start' to train a new model")
		}
	}
}

// getBoolStatus returns a string representation of a boolean status
func getBoolStatus(enabled bool) string {
	if enabled {
		return "Enabled"
	}
	return "Disabled"
}

// addTrainingExample adds a manual training example
func addTrainingExample(mm *MemoryManager, pattern, explanation string) {
	// Get the inference manager to store the training example
	im := GetInferenceManager()
	if im == nil {
		fmt.Println("Inference manager not available")
		return
	}

	if !im.IsEnabled() {
		fmt.Println("Learning system is not enabled")
		fmt.Println("Enable with ':inference enable'")
		return
	}

	// Create a synthetic feedback entry for the training example
	err := im.AddFeedback(pattern, explanation, "helpful", "", mm.config.StoragePath)
	if err != nil {
		fmt.Printf("Error adding training example: %v\n", err)
		return
	}

	fmt.Println("✅ Training example added successfully")
	fmt.Printf("Command Pattern: %s\n", pattern)
	fmt.Printf("Explanation: %s\n", explanation)

	// Show training data stats
	stats := im.GetInferenceStats()
	fmt.Printf("\nTotal training examples: %d\n", stats["training_examples"].(int))
	fmt.Printf("Accumulated examples: %d\n", stats["accumulated_examples"].(int))

	// Suggest training if enough examples accumulated
	if im.ShouldTrain() {
		fmt.Println("\n💡 Training is due. Run ':memory train start' to improve AI predictions.")
	}
}

// addTrainingFeedback adds feedback for the AI
func addTrainingFeedback(mm *MemoryManager, feedback string) {
	// Get the inference manager to store the feedback
	im := GetInferenceManager()
	if im == nil {
		fmt.Println("Inference manager not available")
		return
	}

	if !im.IsEnabled() {
		fmt.Println("Learning system is not enabled")
		fmt.Println("Enable with ':inference enable'")
		return
	}

	// Get the AI manager to get the last prediction
	ai := GetAIManager()
	if ai == nil {
		fmt.Println("AI manager not available")
		return
	}

	// Get last prediction
	lastCommand, lastThought, _ := ai.GetLastPrediction()

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

	// Normalize feedback type
	feedbackType := strings.ToLower(feedback)
	if feedbackType == "good" || feedbackType == "positive" {
		feedbackType = "helpful"
	} else if feedbackType == "bad" || feedbackType == "negative" {
		feedbackType = "unhelpful"
	}

	// Validate feedback type
	if feedbackType != "helpful" && feedbackType != "unhelpful" {
		fmt.Printf("Invalid feedback type: %s\n", feedback)
		fmt.Println("Valid types: helpful, unhelpful")
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
	err = im.AddFeedback(lastCommand, lastThought, feedbackType, "", pwd)
	if err != nil {
		fmt.Printf("Error adding feedback: %v\n", err)
		return
	}

	// Display confirmation
	switch feedbackType {
	case "helpful":
		fmt.Println("✅ Marked as helpful")
	case "unhelpful":
		fmt.Println("❌ Marked as unhelpful")
	}

	// Show training status
	stats := im.GetInferenceStats()
	fmt.Printf("\nTotal training examples: %d\n", stats["training_examples"].(int))
	fmt.Printf("Accumulated examples: %d\n", stats["accumulated_examples"].(int))

	// Suggest training if enough examples accumulated
	if im.ShouldTrain() {
		fmt.Println("\n💡 Training is due. Run ':memory train start' to improve AI predictions.")
	}
}

// runTrainingContainer launches the Docker training environment
func runTrainingContainer(mm *MemoryManager, args []string) {
	// Check if Docker is installed
	_, err := exec.LookPath("docker")
	if err != nil {
		fmt.Println("Docker not found. Please install Docker to use the training environment.")
		return
	}

	// Get inference manager to track training progress
	im := GetInferenceManager()
	if im == nil {
		fmt.Println("Inference manager not available")
		return
	}

	// Parse options from args
	forceTraining := false
	for _, arg := range args {
		if arg == "--force" || arg == "-f" {
			forceTraining = true
		}
	}

	// Check if training is due, unless forcing
	if !forceTraining && !im.ShouldTrain() && im.learningConfig.AccumulatedTrainingExamples < 100 {
		fmt.Println("⚠️ Training is not yet due. Not enough training examples collected.")
		fmt.Printf("Currently have %d examples, need at least 100.\n", im.learningConfig.AccumulatedTrainingExamples)
		fmt.Println("Use ':memory train start --force' to train anyway.")
		return
	}

	// Check if training directory exists
	homeDir, _ := os.UserHomeDir()
	trainingDir := filepath.Join(homeDir, ".config", "delta", "training")

	// Create training directory if it doesn't exist
	err = os.MkdirAll(trainingDir, 0755)
	if err != nil {
		fmt.Printf("Error creating training directory: %v\n", err)
		return
	}

	// Create training data directory if it doesn't exist
	trainingDataDir := filepath.Join(trainingDir, "data")
	err = os.MkdirAll(trainingDataDir, 0755)
	if err != nil {
		fmt.Printf("Error creating training data directory: %v\n", err)
		return
	}

	// Prepare training data
	fmt.Println("Preparing training data...")

	// Export training examples from inference manager
	examples, err := im.GetTrainingExamples(0) // Get all examples
	if err != nil {
		fmt.Printf("Error getting training examples: %v\n", err)
		return
	}

	if len(examples) == 0 {
		fmt.Println("No training examples found. Add examples using:")
		fmt.Println("  :memory train add <pattern> <explanation>")
		fmt.Println("  :inference feedback <helpful|unhelpful|correction>")
		return
	}

	// Write training examples to files
	trainingDataPath := filepath.Join(trainingDataDir, "training_examples.json")
	examplesJSON, err := json.MarshalIndent(examples, "", "  ")
	if err != nil {
		fmt.Printf("Error encoding training examples: %v\n", err)
		return
	}

	err = os.WriteFile(trainingDataPath, examplesJSON, 0644)
	if err != nil {
		fmt.Printf("Error writing training examples: %v\n", err)
		return
	}

	// Export command history data for context
	fmt.Println("Exporting command history data...")
	// Get last 30 days of commands
	startDate := time.Now().AddDate(0, 0, -30)
	endDate := time.Now()

	// Create command entries file to export the data
	exportOptions := ExportOptions{
		Format:      "json",
		StartDate:   startDate,
		EndDate:     endDate,
		IncludeAll:  false,
		Destination: trainingDataDir,
	}

	// Export memory data to the training data directory
	_, err = mm.ExportMemory(exportOptions)
	if err != nil {
		fmt.Printf("Error exporting memory data: %v\n", err)
		return
	}

	// Check if the Docker files are installed
	dockerfilePath := filepath.Join(trainingDir, "Dockerfile")
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		// Copy the Docker files from the installation directory
		installDir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
		srcDir := filepath.Join(installDir, "training")

		// Alternative paths to check if the first one doesn't exist
		alternativePaths := []string{
			"/home/bleepbloop/deltacli/training",
			"/usr/local/share/delta/training",
			"/usr/share/delta/training",
		}

		// Check if the source directory exists
		if _, err := os.Stat(srcDir); os.IsNotExist(err) {
			// Try alternative paths
			for _, path := range alternativePaths {
				if _, err := os.Stat(path); err == nil {
					srcDir = path
					break
				}
			}
		}

		// If we found a valid source directory, copy the files
		if _, err := os.Stat(srcDir); err == nil {
			fmt.Printf("Copying training files from %s to %s...\n", srcDir, trainingDir)

			filesToCopy := []string{
				"Dockerfile",
				"docker-compose.yml",
				"docker-entrypoint.sh",
				"requirements.txt",
				"train.py",
				"train_multi.py",
				"run_training.sh",
			}

			for _, file := range filesToCopy {
				srcFile := filepath.Join(srcDir, file)
				dstFile := filepath.Join(trainingDir, file)

				// Check if source file exists
				if _, err := os.Stat(srcFile); os.IsNotExist(err) {
					continue
				}

				// Copy the file
				data, err := os.ReadFile(srcFile)
				if err != nil {
					fmt.Printf("Error reading %s: %v\n", file, err)
					continue
				}

				err = os.WriteFile(dstFile, data, 0644)
				if err != nil {
					fmt.Printf("Error writing %s: %v\n", file, err)
					continue
				}

				// Make scripts executable
				if strings.HasSuffix(file, ".sh") {
					os.Chmod(dstFile, 0755)
				}
			}
		} else {
			fmt.Println("Error: Training files not found. Please install Delta properly.")
			return
		}
	}

	// Create configuration for the training run
	trainingConfig := map[string]interface{}{
		"model_type":        "onnx",
		"training_epochs":   10,
		"batch_size":        32,
		"learning_rate":     5e-5,
		"input_data_path":   "/data",
		"examples_file":     "training_examples.json",
		"output_model_name": fmt.Sprintf("delta_model_%s", time.Now().Format("20060102")),
		"use_gpu":           true,
	}

	// Write configuration to file
	configPath := filepath.Join(trainingDataDir, "config.json")
	configJSON, err := json.MarshalIndent(trainingConfig, "", "  ")
	if err != nil {
		fmt.Printf("Error encoding training configuration: %v\n", err)
		return
	}

	err = os.WriteFile(configPath, configJSON, 0644)
	if err != nil {
		fmt.Printf("Error writing training configuration: %v\n", err)
		return
	}

	// Check if Docker Compose is available
	composeCommand := "docker-compose"
	_, err = exec.LookPath(composeCommand)
	if err != nil {
		composeCommand = "docker compose" // Try alternative syntax
	}

	// Build the Docker container
	fmt.Println("Building Docker training environment...")
	buildCmd := exec.Command("bash", "-c", fmt.Sprintf("cd %s && %s build", trainingDir, composeCommand))
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	err = buildCmd.Run()
	if err != nil {
		fmt.Printf("Error building Docker container: %v\n", err)
		return
	}

	// Run the training container with data volume mapped
	fmt.Println("Starting training container...")
	runCmd := exec.Command("bash", "-c", fmt.Sprintf("cd %s && %s up", trainingDir, composeCommand))
	runCmd.Stdout = os.Stdout
	runCmd.Stderr = os.Stderr

	err = runCmd.Run()
	if err != nil {
		fmt.Printf("Error running Docker container: %v\n", err)
		return
	}

	// Check if training produced a model file
	modelsDir := filepath.Join(trainingDir, "models")
	modelsFound := false

	if files, err := os.ReadDir(modelsDir); err == nil {
		for _, file := range files {
			if !file.IsDir() && (strings.HasSuffix(file.Name(), ".onnx") || strings.HasSuffix(file.Name(), ".bin")) {
				modelsFound = true

				// Copy the model to the memory models directory
				srcPath := filepath.Join(modelsDir, file.Name())
				dstPath := filepath.Join(mm.config.ModelPath, file.Name())

				data, err := os.ReadFile(srcPath)
				if err != nil {
					fmt.Printf("Error reading model file: %v\n", err)
					continue
				}

				err = os.WriteFile(dstPath, data, 0644)
				if err != nil {
					fmt.Printf("Error copying model file: %v\n", err)
					continue
				}

				fmt.Printf("Model saved to: %s\n", dstPath)
			}
		}
	}

	if modelsFound {
		// Update last training timestamp in memory stats
		lastTrainingFile := filepath.Join(mm.config.ModelPath, "last_training.txt")
		nowTime := time.Now().Format(time.RFC3339)
		os.WriteFile(lastTrainingFile, []byte(nowTime), 0644)

		// Update inference manager
		im.RecordTrainingCompletion()

		fmt.Println("✅ Training completed successfully!")
		fmt.Println("Use ':inference model use <model_name>' to use the new model")
	} else {
		fmt.Println("⚠️ Training completed but no model files were produced.")
		fmt.Println("Check the training logs for errors.")
	}
}

// Helper functions

// formatDuration formats a duration in a human-readable format
func formatDuration(d time.Duration) string {
	days := int(d.Hours() / 24)
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%d days, %d hours, %d minutes", days, hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%d hours, %d minutes", hours, minutes)
	} else {
		return fmt.Sprintf("%d minutes", minutes)
	}
}

// contains checks if a string slice contains a value
func contains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// removeString removes a string from a slice
func removeString(slice []string, value string) []string {
	result := make([]string, 0, len(slice))
	for _, item := range slice {
		if item != value {
			result = append(result, item)
		}
	}
	return result
}
