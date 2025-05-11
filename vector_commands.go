package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// HandleVectorCommand processes vector database-related commands
func HandleVectorCommand(args []string) bool {
	// Get the VectorDBManager instance
	vm := GetVectorDBManager()
	if vm == nil {
		fmt.Println("Failed to initialize vector database manager")
		return true
	}

	// Handle subcommands
	if len(args) == 0 {
		// Show status by default
		showVectorStatus(vm)
		return true
	}

	switch args[0] {
	case "enable":
		// Initialize and enable the vector database
		err := vm.Initialize()
		if err != nil {
			fmt.Printf("Error initializing vector database: %v\n", err)
			return true
		}

		err = vm.Enable()
		if err != nil {
			fmt.Printf("Error enabling vector database: %v\n", err)
		} else {
			fmt.Println("Vector database enabled")
		}
		return true

	case "disable":
		// Disable the vector database
		err := vm.Disable()
		if err != nil {
			fmt.Printf("Error disabling vector database: %v\n", err)
		} else {
			fmt.Println("Vector database disabled")
		}
		return true

	case "status":
		// Show vector database status
		showVectorStatus(vm)
		return true

	case "stats":
		// Show detailed stats
		showVectorStats(vm)
		return true

	case "search":
		// Search for similar commands
		if len(args) < 2 {
			fmt.Println("Usage: :vector search <command>")
			return true
		}
		searchCommand := strings.Join(args[1:], " ")
		searchSimilarCommands(vm, searchCommand)
		return true

	case "embed":
		// Generate embeddings for a command
		if len(args) < 2 {
			fmt.Println("Usage: :vector embed <command>")
			return true
		}
		command := strings.Join(args[1:], " ")
		generateEmbedding(vm, command)
		return true

	case "config":
		// Handle configuration subcommands
		if len(args) > 1 && args[1] == "set" {
			if len(args) < 4 {
				fmt.Println("Usage: :vector config set <setting> <value>")
				return true
			}
			updateVectorConfig(vm, args[2], args[3])
		} else {
			showVectorConfig(vm)
		}
		return true

	case "help":
		// Show help
		showVectorHelp()
		return true

	default:
		fmt.Printf("Unknown vector command: %s\n", args[0])
		fmt.Println("Type :vector help for available commands")
		return true
	}
}

// showVectorStatus displays current status of the vector database
func showVectorStatus(vm *VectorDBManager) {
	fmt.Println("Vector Database Status")
	fmt.Println("=====================")

	isEnabled := vm.IsEnabled()
	isInitialized := vm.isInitialized

	fmt.Printf("Status: %s\n", getStatusText(isEnabled, isInitialized))

	if isInitialized {
		stats := vm.GetStats()
		vectorCount := stats["vector_count"].(int)
		fmt.Printf("Vector Count: %d\n", vectorCount)

		if size, ok := stats["db_size_mb"].(float64); ok {
			fmt.Printf("Database Size: %.2f MB\n", size)
		}

		// Show vector extension status
		if hasExt, ok := stats["has_vector_extension"].(bool); ok {
			if hasExt {
				fmt.Println("Vector Extension: Available (using SQLite with vector search)")
			} else {
				fmt.Println("Vector Extension: Not available (using in-memory search)")
			}
		}
	}

	// Show path to database
	fmt.Printf("Database Path: %s\n", vm.config.DBPath)
}

// getStatusText returns a descriptive status text
func getStatusText(enabled, initialized bool) string {
	if !initialized {
		return "Not initialized"
	} else if enabled {
		return "Enabled and ready"
	} else {
		return "Disabled (initialized)"
	}
}

// showVectorStats displays detailed statistics about the vector database
func showVectorStats(vm *VectorDBManager) {
	fmt.Println("Vector Database Statistics")
	fmt.Println("=========================")

	if !vm.isInitialized {
		fmt.Println("Vector database not initialized")
		fmt.Println("Run ':vector enable' to initialize")
		return
	}

	stats := vm.GetStats()

	fmt.Printf("Status: %s\n", getStatusText(stats["enabled"].(bool), stats["initialized"].(bool)))
	fmt.Printf("Vector Count: %d\n", stats["vector_count"].(int))

	if size, ok := stats["db_size_mb"].(float64); ok {
		fmt.Printf("Database Size: %.2f MB\n", size)
	}

	fmt.Printf("Embedding Dimension: %d\n", stats["dimension"].(int))
	fmt.Printf("Distance Metric: %s\n", stats["metric"].(string))
	fmt.Printf("Max Entries: %d\n", stats["max_entries"].(int))
	fmt.Printf("Index Rebuild Interval: %d minutes\n", stats["index_interval"].(int))

	// Show vector extension status
	if hasExt, ok := stats["has_vector_extension"].(bool); ok {
		if hasExt {
			fmt.Println("Vector Extension: Available (using SQLite with vector search)")
		} else {
			fmt.Println("Vector Extension: Not available (using in-memory search)")
		}
	}

	// Show last index build time
	if lastBuild, ok := stats["last_index_build"].(time.Time); ok && !lastBuild.IsZero() {
		fmt.Printf("Last Index Build: %s\n", lastBuild.Format(time.RFC1123))
	} else {
		fmt.Println("Last Index Build: Never")
	}

	// Show database path
	fmt.Printf("Database Path: %s\n", stats["db_path"].(string))
}

// searchSimilarCommands searches for similar commands
func searchSimilarCommands(vm *VectorDBManager, query string) {
	if !vm.IsEnabled() {
		fmt.Println("Vector database not enabled")
		fmt.Println("Run ':vector enable' to enable")
		return
	}

	fmt.Printf("Searching for commands similar to: %s\n", query)
	fmt.Println("------------------------------------------")

	// This is a placeholder since we don't have the actual embedding logic yet
	// In a real implementation, we would:
	// 1. Generate an embedding for the query
	// 2. Search for similar commands using that embedding
	fmt.Println("Note: This is a placeholder. Actual embedding search will be implemented when model integration is complete.")
	fmt.Println("")
	fmt.Println("Results based on keyword matching:")
	fmt.Println("")

	// For demonstration, just get commands containing the query
	if db := vm.db; db != nil {
		rows, err := db.Query(`
			SELECT command, directory, frequency, last_used
			FROM command_embeddings 
			WHERE command LIKE ? 
			ORDER BY frequency DESC, last_used DESC
			LIMIT 5
		`, "%"+query+"%")

		if err != nil {
			fmt.Printf("Error searching: %v\n", err)
			return
		}
		defer rows.Close()

		found := false
		for rows.Next() {
			found = true
			var (
				command    string
				directory  string
				frequency  int
				lastUsedTS int64
			)
			err = rows.Scan(&command, &directory, &frequency, &lastUsedTS)
			if err != nil {
				continue
			}

			lastUsed := time.Unix(lastUsedTS, 0)
			timeSince := formatTimeSince(time.Since(lastUsed))

			fmt.Printf("Command: %s\n", command)
			fmt.Printf("  Directory: %s\n", directory)
			fmt.Printf("  Frequency: %d times\n", frequency)
			fmt.Printf("  Last Used: %s ago\n", timeSince)
			fmt.Println()
		}

		if !found {
			fmt.Println("No similar commands found")
		}
	} else {
		fmt.Println("Database not initialized properly")
	}
}

// generateEmbedding generates an embedding for a command (placeholder)
func generateEmbedding(vm *VectorDBManager, command string) {
	if !vm.IsEnabled() {
		fmt.Println("Vector database not enabled")
		fmt.Println("Run ':vector enable' to enable")
		return
	}

	fmt.Printf("Generating embedding for command: %s\n", command)
	fmt.Println("------------------------------------------")
	fmt.Println("Note: This is a placeholder. Actual embedding generation will be implemented when model integration is complete.")

	// In a real implementation, we would:
	// 1. Generate an embedding for the command using the model
	// 2. Return information about the embedding
	
	// For demonstration, just show the planned embedding
	fmt.Printf("Command: %s\n", command)
	fmt.Printf("Embedding Dimension: %d\n", vm.config.EmbeddingDimension)
	fmt.Printf("Embedding Storage: %s\n", vm.config.DBPath)
	fmt.Println("Embedding would be stored in vector database when implemented")
}

// showVectorConfig displays the vector database configuration
func showVectorConfig(vm *VectorDBManager) {
	fmt.Println("Vector Database Configuration")
	fmt.Println("============================")

	fmt.Printf("Enabled: %t\n", vm.config.Enabled)
	fmt.Printf("Database Path: %s\n", vm.config.DBPath)
	fmt.Printf("Embedding Dimension: %d\n", vm.config.EmbeddingDimension)
	fmt.Printf("Distance Metric: %s\n", vm.config.DistanceMetric)
	fmt.Printf("Max Entries: %d\n", vm.config.MaxEntries)
	fmt.Printf("Index Rebuild Interval: %d minutes\n", vm.config.IndexBuildInterval)
	
	fmt.Println("\nCommand Types:")
	for _, cmdType := range vm.config.CommandTypes {
		fmt.Printf("  - %s\n", cmdType)
	}

	fmt.Println("\nAvailable Settings:")
	fmt.Println("  dimension      - Embedding dimension (e.g., 384, 768, 1024)")
	fmt.Println("  metric         - Distance metric (cosine, dot, euclidean)")
	fmt.Println("  max_entries    - Maximum number of entries to store")
	fmt.Println("  index_interval - Interval in minutes for index rebuilding")
}

// updateVectorConfig updates a vector database configuration setting
func updateVectorConfig(vm *VectorDBManager, setting, value string) {
	// Clone the current config
	config := vm.config

	// Update the setting
	switch setting {
	case "dimension", "embedding_dimension":
		dimension, err := strconv.Atoi(value)
		if err != nil || dimension <= 0 {
			fmt.Println("Dimension must be a positive integer")
			return
		}
		config.EmbeddingDimension = dimension

	case "metric", "distance_metric":
		if value != "cosine" && value != "dot" && value != "euclidean" {
			fmt.Println("Metric must be one of: cosine, dot, euclidean")
			return
		}
		config.DistanceMetric = value

	case "max_entries":
		maxEntries, err := strconv.Atoi(value)
		if err != nil || maxEntries <= 0 {
			fmt.Println("Max entries must be a positive integer")
			return
		}
		config.MaxEntries = maxEntries

	case "index_interval":
		interval, err := strconv.Atoi(value)
		if err != nil || interval <= 0 {
			fmt.Println("Index interval must be a positive integer")
			return
		}
		config.IndexBuildInterval = interval

	default:
		fmt.Printf("Unknown setting: %s\n", setting)
		return
	}

	// Save the updated config
	err := vm.UpdateConfig(config)
	if err != nil {
		fmt.Printf("Error updating configuration: %v\n", err)
		return
	}

	fmt.Printf("Successfully updated %s to %s\n", setting, value)
}

// showVectorHelp displays help for vector database commands
func showVectorHelp() {
	fmt.Println("Vector Database Commands")
	fmt.Println("=======================")
	fmt.Println("  :vector              - Show vector database status")
	fmt.Println("  :vector enable       - Initialize and enable vector database")
	fmt.Println("  :vector disable      - Disable vector database")
	fmt.Println("  :vector status       - Show vector database status")
	fmt.Println("  :vector stats        - Show detailed statistics")
	fmt.Println("  :vector search <cmd> - Search for similar commands")
	fmt.Println("  :vector embed <cmd>  - Generate embedding for a command")
	fmt.Println("  :vector config       - Show configuration")
	fmt.Println("  :vector config set <setting> <value> - Update configuration")
	fmt.Println("  :vector help         - Show this help message")
	fmt.Println("")
	fmt.Println("Note: Vector search requires the embedding model to be available.")
	fmt.Println("This feature will be fully functional when the inference optimization")
	fmt.Println("milestone is completed.")
}

// Helper functions

// formatTimeSince formats a duration in a user-friendly way
func formatTimeSince(d time.Duration) string {
	if d.Hours() > 24 {
		days := int(d.Hours() / 24)
		return fmt.Sprintf("%d days", days)
	} else if d.Hours() >= 1 {
		hours := int(d.Hours())
		return fmt.Sprintf("%d hours", hours)
	} else if d.Minutes() >= 1 {
		minutes := int(d.Minutes())
		return fmt.Sprintf("%d minutes", minutes)
	} else {
		seconds := int(d.Seconds())
		return fmt.Sprintf("%d seconds", seconds)
	}
}