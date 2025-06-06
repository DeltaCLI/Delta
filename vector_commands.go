package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

	case "export":
		// Export vector database to a file
		if len(args) < 2 {
			fmt.Println("Usage: :vector export <file_path>")
			return true
		}
		exportVectorDatabase(vm, args[1])
		return true

	case "import":
		// Import vector database from a file
		if len(args) < 2 {
			fmt.Println("Usage: :vector import <file_path> [merge_strategy]")
			fmt.Println("Available merge strategies: replace, merge, keep_newer")
			return true
		}

		// Default merge strategy is "merge"
		mergeStrategy := "merge"
		if len(args) >= 3 {
			mergeStrategy = args[2]
		}

		importVectorDatabase(vm, args[1], mergeStrategy)
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

			// Show available vector functions
			if functions, ok := stats["vectorx_functions"].(map[string]interface{}); ok {
				availableMetrics := []string{}

				// Check which metrics are available
				if fn, ok := functions["vectorx_cosine_similarity"].(bool); ok && fn {
					availableMetrics = append(availableMetrics, "cosine")
				}
				if fn, ok := functions["vectorx_dot_product"].(bool); ok && fn {
					availableMetrics = append(availableMetrics, "dot")
				}
				if fn, ok := functions["vectorx_euclidean_distance"].(bool); ok && fn {
					availableMetrics = append(availableMetrics, "euclidean")
				}
				if fn, ok := functions["vectorx_manhattan_distance"].(bool); ok && fn {
					availableMetrics = append(availableMetrics, "manhattan")
				}
				if fn, ok := functions["vectorx_jaccard_similarity"].(bool); ok && fn {
					availableMetrics = append(availableMetrics, "jaccard")
				}

				if len(availableMetrics) > 0 {
					fmt.Printf("Available Metrics: %s\n", strings.Join(availableMetrics, ", "))
				}
			}
		} else {
			fmt.Println("Vector Extension: Not available (using in-memory search)")
			fmt.Println("Available Metrics: cosine, dot, euclidean, manhattan, jaccard (all computed in memory)")
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

	// Check if query has a special format to specify metric
	// Format: "metric:xxx query" e.g. "metric:manhattan git commit"
	metric := vm.config.DistanceMetric
	searchQuery := query

	if strings.HasPrefix(query, "metric:") {
		parts := strings.SplitN(query, " ", 2)
		if len(parts) == 2 {
			metricSpec := strings.TrimPrefix(parts[0], "metric:")
			searchQuery = parts[1]

			// Validate metric
			validMetrics := map[string]bool{
				"cosine":    true,
				"dot":       true,
				"euclidean": true,
				"manhattan": true,
				"jaccard":   true,
			}

			if validMetrics[metricSpec] {
				metric = metricSpec
				fmt.Printf("Using %s distance metric for this search\n", metric)
			} else {
				fmt.Printf("Invalid metric: %s, using default (%s)\n", metricSpec, metric)
			}
		}
	}

	fmt.Printf("Searching for commands similar to: %s\n", searchQuery)
	fmt.Println("------------------------------------------")

	// Get the AI manager for embedding generation
	ai := GetAIManager()
	if ai == nil {
		fmt.Println("AI manager not initialized. Using keyword search fallback.")
		keywordSearch(vm, searchQuery)
		return
	}

	// Generate embedding for the query
	embedding, err := ai.GenerateEmbedding(searchQuery)
	if err != nil {
		fmt.Printf("Error generating embedding: %v\n", err)
		fmt.Println("Using keyword search fallback.")
		keywordSearch(vm, searchQuery)
		return
	}

	// Store original metric
	originalMetric := vm.config.DistanceMetric

	// Temporarily set the metric for this search if different
	if metric != originalMetric {
		config := vm.config
		config.DistanceMetric = metric
		vm.UpdateConfig(config)
		defer func() {
			// Restore original metric after search
			config := vm.config
			config.DistanceMetric = originalMetric
			vm.UpdateConfig(config)
		}()
	}

	// Search for similar commands
	results, err := vm.SearchSimilarCommands(embedding, "", 10) // Empty context to search all
	if err != nil {
		fmt.Printf("Error searching for similar commands: %v\n", err)
		fmt.Println("Using keyword search fallback.")
		keywordSearch(vm, searchQuery)
		return
	}

	// Display results
	if len(results) == 0 {
		fmt.Println("No similar commands found")
		return
	}

	fmt.Printf("Found %d similar commands (metric: %s):\n\n", len(results), metric)

	for i, result := range results {
		// Parse metadata to get similarity score if available
		similarityStr := ""
		if result.Metadata != "" {
			var metadata map[string]interface{}
			if err := json.Unmarshal([]byte(result.Metadata), &metadata); err == nil {
				if sim, ok := metadata["similarity"].(float64); ok {
					similarityStr = fmt.Sprintf(" (score: %.4f)", sim)
				}
			}
		}

		// Calculate time since last used
		timeSince := formatTimeSince(time.Since(result.LastUsed))

		fmt.Printf("%d. Command: %s%s\n", i+1, result.Command, similarityStr)
		fmt.Printf("   Directory: %s\n", result.Directory)
		fmt.Printf("   Used: %d times, last used %s ago\n", result.Frequency, timeSince)
		if result.ExitCode != 0 {
			fmt.Printf("   Note: Last exit code was %d\n", result.ExitCode)
		}
		fmt.Println()
	}
}

// keywordSearch is a fallback search method that uses SQL LIKE for finding commands
func keywordSearch(vm *VectorDBManager, query string) {
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

	// Show Jaccard threshold if using Jaccard similarity
	if vm.config.DistanceMetric == "jaccard" {
		fmt.Printf("Jaccard Similarity Threshold: %.2f\n", vm.config.JaccardThreshold)
	}

	fmt.Println("\nCommand Types:")
	for _, cmdType := range vm.config.CommandTypes {
		fmt.Printf("  - %s\n", cmdType)
	}

	fmt.Println("\nAvailable Settings:")
	fmt.Println("  dimension        - Embedding dimension (e.g., 384, 768, 1024)")
	fmt.Println("  metric           - Distance metric (cosine, dot, euclidean, manhattan, jaccard)")
	fmt.Println("  max_entries      - Maximum number of entries to store")
	fmt.Println("  index_interval   - Interval in minutes for index rebuilding")
	fmt.Println("  jaccard_threshold - Threshold for Jaccard similarity (default 0.1)")
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
		validMetrics := map[string]bool{
			"cosine":    true,
			"dot":       true,
			"euclidean": true,
			"manhattan": true,
			"jaccard":   true,
		}

		if !validMetrics[value] {
			fmt.Println("Metric must be one of: cosine, dot, euclidean, manhattan, jaccard")
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

	case "jaccard_threshold":
		threshold, err := strconv.ParseFloat(value, 32)
		if err != nil || threshold < 0 || threshold > 1 {
			fmt.Println("Jaccard threshold must be a float between 0 and 1")
			return
		}
		config.JaccardThreshold = float32(threshold)

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
	fmt.Println("  :vector search metric:<metric> <cmd> - Search with specified metric")
	fmt.Println("                         Available metrics: cosine, dot, euclidean, manhattan, jaccard")
	fmt.Println("  :vector embed <cmd>  - Generate embedding for a command")
	fmt.Println("  :vector export <file> - Export vector database to a file")
	fmt.Println("  :vector import <file> [strategy] - Import vector database from a file")
	fmt.Println("                         Available strategies: replace, merge, keep_newer")
	fmt.Println("  :vector config       - Show configuration")
	fmt.Println("  :vector config set <setting> <value> - Update configuration")
	fmt.Println("  :vector help         - Show this help message")
	fmt.Println("")
	fmt.Println("Note: Vector search requires the embedding model to be available.")
	fmt.Println("This feature will be fully functional when the inference optimization")
	fmt.Println("milestone is completed.")
}

// Helper functions

// exportVectorDatabase exports vector database to a file
func exportVectorDatabase(vm *VectorDBManager, filePath string) {
	if !vm.IsEnabled() {
		fmt.Println("Vector database not enabled")
		fmt.Println("Run ':vector enable' to enable")
		return
	}

	fmt.Printf("Exporting vector database to: %s\n", filePath)

	// Expand ~ in file path if present
	if strings.HasPrefix(filePath, "~") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			filePath = filepath.Join(homeDir, filePath[1:])
		}
	}

	// Ensure the directory exists
	dir := filepath.Dir(filePath)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		fmt.Printf("Error creating directory %s: %v\n", dir, err)
		return
	}

	// Export the data
	startTime := time.Now()
	err = vm.ExportData(filePath)
	if err != nil {
		fmt.Printf("Error exporting data: %v\n", err)
		return
	}

	// Get stats about the export
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		fmt.Printf("Export completed successfully to %s\n", filePath)
		return
	}

	fileSizeMB := float64(fileInfo.Size()) / 1024 / 1024
	duration := time.Since(startTime)

	fmt.Printf("Export completed successfully to %s\n", filePath)
	fmt.Printf("  Size: %.2f MB\n", fileSizeMB)
	fmt.Printf("  Duration: %s\n", formatVectorDuration(duration))

	// Get the number of embeddings
	stats := vm.GetStats()
	if vectorCount, ok := stats["vector_count"].(int); ok {
		fmt.Printf("  Exported %d embeddings\n", vectorCount)
	}
}

// importVectorDatabase imports vector database from a file
func importVectorDatabase(vm *VectorDBManager, filePath string, mergeStrategy string) {
	if !vm.IsEnabled() {
		fmt.Println("Vector database not enabled")
		fmt.Println("Run ':vector enable' to enable")
		return
	}

	// Validate merge strategy
	validStrategies := map[string]bool{
		"replace":    true,
		"merge":      true,
		"keep_newer": true,
	}

	if !validStrategies[mergeStrategy] {
		fmt.Printf("Invalid merge strategy: %s\n", mergeStrategy)
		fmt.Println("Available strategies: replace, merge, keep_newer")
		return
	}

	fmt.Printf("Importing vector database from: %s\n", filePath)
	fmt.Printf("Using merge strategy: %s\n", mergeStrategy)

	// Expand ~ in file path if present
	if strings.HasPrefix(filePath, "~") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			filePath = filepath.Join(homeDir, filePath[1:])
		}
	}

	// Check if the file exists
	_, err := os.Stat(filePath)
	if err != nil {
		fmt.Printf("Error accessing file: %v\n", err)
		return
	}

	// Get stats before import
	statsBefore := vm.GetStats()
	countBefore := 0
	if vectorCount, ok := statsBefore["vector_count"].(int); ok {
		countBefore = vectorCount
	}

	// Import the data
	startTime := time.Now()
	err = vm.ImportData(filePath, mergeStrategy)
	if err != nil {
		fmt.Printf("Error importing data: %v\n", err)
		return
	}

	// Get stats after import
	statsAfter := vm.GetStats()
	countAfter := 0
	if vectorCount, ok := statsAfter["vector_count"].(int); ok {
		countAfter = vectorCount
	}

	duration := time.Since(startTime)

	fmt.Printf("Import completed successfully\n")
	fmt.Printf("  Duration: %s\n", formatVectorDuration(duration))
	fmt.Printf("  Embeddings before: %d\n", countBefore)
	fmt.Printf("  Embeddings after: %d\n", countAfter)
	fmt.Printf("  Net change: %+d\n", countAfter-countBefore)
}

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

// formatVectorDuration formats a duration in a more precise way
func formatVectorDuration(d time.Duration) string {
	if d.Hours() >= 1 {
		return fmt.Sprintf("%.1f hours", d.Hours())
	} else if d.Minutes() >= 1 {
		return fmt.Sprintf("%.1f minutes", d.Minutes())
	} else if d.Seconds() >= 1 {
		return fmt.Sprintf("%.1f seconds", d.Seconds())
	} else {
		return fmt.Sprintf("%d milliseconds", d.Milliseconds())
	}
}
