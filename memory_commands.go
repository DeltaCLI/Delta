package main

import (
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
			// Export data for a specific date or all data
			if len(args) >= 2 {
				date := args[1]
				exportMemoryData(mm, date)
			} else {
				fmt.Println("Usage: :memory export YYYY-MM-DD")
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
		fmt.Println("  :memory export     - Export data for a specific date")
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
		
		days := int(stats.LastEntry.Sub(stats.FirstEntry).Hours() / 24) + 1
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

// exportMemoryData exports data for a specific date
func exportMemoryData(mm *MemoryManager, date string) {
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
}

// showTrainingStatus displays the current training status
func showTrainingStatus(mm *MemoryManager) {
	stats, err := mm.GetStats()
	if err != nil {
		fmt.Printf("Error getting stats: %v\n", err)
		return
	}
	
	fmt.Println("Training Status")
	fmt.Println("==============")
	fmt.Printf("Training Enabled: %v\n", mm.config.TrainingEnabled)
	
	if !stats.LastTraining.IsZero() {
		fmt.Printf("Last Training: %s\n", stats.LastTraining.Format(time.RFC1123))
		fmt.Printf("Time Since Training: %s\n", formatDuration(time.Since(stats.LastTraining)))
	} else {
		fmt.Println("No training has been performed yet")
	}
	
	if len(stats.ModelVersions) > 0 {
		fmt.Println("\nAvailable Models:")
		for _, model := range stats.ModelVersions {
			fmt.Printf("  - %s\n", model)
		}
	} else {
		fmt.Println("\nNo models available")
	}
	
	fmt.Println("\nTraining Data:")
	fmt.Printf("  Commands Available: %d\n", stats.TotalEntries)
	
	// This will be implemented in later milestones
	fmt.Println("\nNext Steps:")
	fmt.Println("  Training functionality will be implemented in future milestones")
}

// addTrainingExample adds a manual training example
func addTrainingExample(mm *MemoryManager, pattern, explanation string) {
	// This will be implemented in a future milestone
	fmt.Println("Training example storage will be implemented in a future milestone")
	fmt.Printf("Recorded pattern: %s\n", pattern)
	fmt.Printf("Explanation: %s\n", explanation)
}

// addTrainingFeedback adds feedback for the AI
func addTrainingFeedback(mm *MemoryManager, feedback string) {
	// This will be implemented in a future milestone
	fmt.Printf("Feedback recorded: %s\n", feedback)
	fmt.Println("Feedback storage will be implemented in a future milestone")
}

// runTrainingContainer launches the Docker training environment
func runTrainingContainer(mm *MemoryManager, args []string) {
	// Check if Docker is installed
	_, err := exec.LookPath("docker")
	if err != nil {
		fmt.Println("Docker not found. Please install Docker to use the training environment.")
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

	// Run the training container
	fmt.Println("Starting training container...")
	runCmd := exec.Command("bash", "-c", fmt.Sprintf("cd %s && %s up", trainingDir, composeCommand))
	runCmd.Stdout = os.Stdout
	runCmd.Stderr = os.Stderr

	err = runCmd.Run()
	if err != nil {
		fmt.Printf("Error running Docker container: %v\n", err)
		return
	}

	fmt.Println("Training completed.")
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