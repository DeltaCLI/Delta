package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// HandleKnowledgeCommand processes knowledge extraction commands
func HandleKnowledgeCommand(args []string) bool {
	// Get the KnowledgeExtractor instance
	ke := GetKnowledgeExtractor()
	if ke == nil {
		fmt.Println("Failed to initialize knowledge extractor")
		return true
	}

	// Process commands
	if len(args) == 0 {
		// Show status by default
		showKnowledgeStatus(ke)
		return true
	}

	// Handle subcommands
	switch args[0] {
	case "enable":
		// Initialize and enable the knowledge extractor
		err := ke.Initialize()
		if err != nil {
			fmt.Printf("Error initializing knowledge extractor: %v\n", err)
			return true
		}

		err = ke.Enable()
		if err != nil {
			fmt.Printf("Error enabling knowledge extractor: %v\n", err)
		} else {
			fmt.Println("Knowledge extraction enabled")
		}
		return true

	case "disable":
		// Disable the knowledge extractor
		err := ke.Disable()
		if err != nil {
			fmt.Printf("Error disabling knowledge extractor: %v\n", err)
		} else {
			fmt.Println("Knowledge extraction disabled")
		}
		return true

	case "status":
		// Show knowledge extractor status
		showKnowledgeStatus(ke)
		return true

	case "stats":
		// Show detailed stats
		showKnowledgeStats(ke)
		return true

	case "query":
		// Search for knowledge
		if len(args) < 2 {
			fmt.Println("Usage: :knowledge query <query>")
			return true
		}
		query := strings.Join(args[1:], " ")
		searchKnowledge(ke, query)
		return true

	case "context":
		// Show current context
		showCurrentContext(ke)
		return true

	case "scan":
		// Scan current directory for knowledge
		scanCurrentDirectory(ke)
		return true

	case "project":
		// Show project information
		showProjectInfo(ke)
		return true

	case "extract":
		// Extract knowledge from command
		if len(args) < 2 {
			fmt.Println("Usage: :knowledge extract <command>")
			return true
		}
		command := strings.Join(args[1:], " ")
		extractFromCommand(ke, command)
		return true

	case "clear":
		// Clear knowledge entities
		clearKnowledge(ke)
		return true

	case "export":
		// Export knowledge
		if len(args) < 2 {
			fmt.Println("Usage: :knowledge export <filepath>")
			return true
		}
		exportKnowledge(ke, args[1])
		return true

	case "import":
		// Import knowledge
		if len(args) < 2 {
			fmt.Println("Usage: :knowledge import <filepath>")
			return true
		}
		importKnowledge(ke, args[1])
		return true

	case "help":
		// Show help
		showKnowledgeHelp()
		return true

	default:
		fmt.Printf("Unknown knowledge command: %s\n", args[0])
		fmt.Println("Type :knowledge help for available commands")
		return true
	}
}

// showKnowledgeStatus displays current status of the knowledge extractor
func showKnowledgeStatus(ke *KnowledgeExtractor) {
	fmt.Println("Knowledge Extractor Status")
	fmt.Println("==========================")

	stats := ke.GetStats()
	enabled := stats["enabled"].(bool)
	initialized := stats["initialized"].(bool)

	if initialized {
		if enabled {
			fmt.Println("Status: Enabled and active")
		} else {
			fmt.Println("Status: Initialized but disabled")
		}

		// Show entity counts
		entityCount := stats["total_items"].(int)
		fmt.Printf("Total Knowledge Items: %d\n", entityCount)

		// Show environment status
		fmt.Printf("Environment Awareness: %t\n", stats["environment_awareness"])
		fmt.Printf("Command Awareness: %t\n", stats["command_awareness"])
		fmt.Printf("Code Awareness: %t\n", stats["code_awareness"])
		fmt.Printf("Project Awareness: %t\n", stats["project_awareness"])

		// Show last scan info
		lastRefresh, ok := stats["last_refresh"].(time.Time)
		if ok && !lastRefresh.IsZero() {
			fmt.Printf("Last Context Refresh: %s\n", formatKnowledgeDuration(time.Since(lastRefresh)))
		} else {
			fmt.Println("Last Context Refresh: Never")
		}

		// Show current directory
		currentDir, ok := stats["current_directory"].(string)
		if ok && currentDir != "" {
			fmt.Printf("Current Directory: %s\n", currentDir)
		}

		// Show project type
		projectType, ok := stats["project_type"].(string)
		if ok && projectType != "" {
			fmt.Printf("Project Type: %s\n", projectType)
		}
	} else {
		fmt.Println("Status: Not initialized")
		fmt.Println("Run ':knowledge enable' to initialize")
	}
}

// showKnowledgeStats displays detailed stats about the knowledge extractor
func showKnowledgeStats(ke *KnowledgeExtractor) {
	fmt.Println("Knowledge Extractor Statistics")
	fmt.Println("=============================")

	stats := ke.GetStats()

	// Show status
	enabled := stats["enabled"].(bool)
	initialized := stats["initialized"].(bool)

	if initialized {
		if enabled {
			fmt.Println("Status: Enabled and active")
		} else {
			fmt.Println("Status: Initialized but disabled")
		}

		// Show entity counts
		fmt.Println("\nKnowledge Items:")
		totalItems := stats["total_items"].(int)
		fmt.Printf("Total Items: %d\n", totalItems)

		// Show items with embeddings
		itemsWithEmbeddings := stats["items_with_embeddings"].(int)
		embeddingPercent := 0.0
		if totalItems > 0 {
			embeddingPercent = float64(itemsWithEmbeddings) / float64(totalItems) * 100
		}
		fmt.Printf("Items with Embeddings: %d (%.1f%%)\n", itemsWithEmbeddings, embeddingPercent)

		// Show source counts
		fmt.Println("\nItems by Source:")
		sourceCounts, ok := stats["source_counts"].(map[string]interface{})
		if ok {
			for source, count := range sourceCounts {
				fmt.Printf("  %s: %d\n", source, count)
			}
		}

		// Show type counts
		fmt.Println("\nItems by Type:")
		typeCounts, ok := stats["type_counts"].(map[string]interface{})
		if ok {
			for itemType, count := range typeCounts {
				fmt.Printf("  %s: %d\n", itemType, count)
			}
		}

		// Show refresh info
		fmt.Println("\nContext Refresh:")
		lastRefresh, ok := stats["last_refresh"].(time.Time)
		if ok && !lastRefresh.IsZero() {
			fmt.Printf("Last Refresh: %s\n", lastRefresh.Format(time.RFC1123))
			fmt.Printf("Time Since Refresh: %s\n", formatKnowledgeDuration(time.Since(lastRefresh)))
		} else {
			fmt.Println("Last Refresh: Never")
		}
	} else {
		fmt.Println("Status: Not initialized")
		fmt.Println("Run ':knowledge enable' to initialize")
	}

	// Show configuration
	fmt.Println("\nConfiguration:")
	fmt.Printf("Environment Awareness: %t\n", stats["environment_awareness"])
	fmt.Printf("Command Awareness: %t\n", stats["command_awareness"])
	fmt.Printf("Code Awareness: %t\n", stats["code_awareness"])
	fmt.Printf("Project Awareness: %t\n", stats["project_awareness"])
	fmt.Printf("Max File Size KB: %d\n", stats["max_file_size_kb"])
	fmt.Printf("Max Scan Depth: %d\n", stats["max_scan_depth"])
	fmt.Printf("Max Extracted Items: %d\n", stats["max_extracted_items"])
	fmt.Printf("Refresh Interval: %d minutes\n", stats["refresh_interval_minutes"])
	fmt.Printf("Privacy Enabled: %t\n", stats["privacy_enabled"])
}

// searchKnowledge searches for knowledge items matching a query
func searchKnowledge(ke *KnowledgeExtractor, query string) {
	if !ke.IsEnabled() {
		fmt.Println("Knowledge extractor not enabled")
		fmt.Println("Run ':knowledge enable' to enable")
		return
	}

	fmt.Printf("Searching for knowledge matching: %s\n", query)
	fmt.Println("------------------------------------------")

	// Generate embeddings for items first
	err := ke.GenerateEmbeddings()
	if err != nil {
		fmt.Printf("Warning: Error generating embeddings: %v\n", err)
	}

	// Search for knowledge
	results, err := ke.SearchKnowledge(query, 10)
	if err != nil {
		fmt.Printf("Error searching knowledge: %v\n", err)
		return
	}

	if len(results) == 0 {
		fmt.Println("No matching knowledge found")
		return
	}

	// Display results
	fmt.Printf("Found %d matching items:\n\n", len(results))

	for i, item := range results {
		fmt.Printf("%d. [%s] %s\n", i+1, item.Type, item.Pattern)
		if len(item.Examples) > 0 {
			fmt.Printf("   Example: %s\n", item.Examples[0])
		}
		fmt.Printf("   Confidence: %.2f\n", item.Confidence)
		fmt.Println()
	}
}

// showCurrentContext displays the current environment context
func showCurrentContext(ke *KnowledgeExtractor) {
	if !ke.IsEnabled() {
		fmt.Println("Knowledge extractor not enabled")
		fmt.Println("Run ':knowledge enable' to enable")
		return
	}

	// Get current directory
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v\n", err)
		return
	}

	// Update context with current directory
	err = ke.UpdateContext(currentDir)
	if err != nil {
		fmt.Printf("Error updating context: %v\n", err)
		return
	}

	// Get current context
	context := ke.GetCurrentContext()

	fmt.Println("Current Environment Context")
	fmt.Println("==========================")
	
	fmt.Printf("OS: %s\n", context.OS)
	fmt.Printf("Architecture: %s\n", context.Arch)
	fmt.Printf("Shell: %s\n", context.Shell)
	fmt.Printf("User: %s\n", context.User)
	fmt.Printf("Hostname: %s\n", context.Hostname)
	fmt.Printf("Current Directory: %s\n", context.CurrentDir)
	fmt.Printf("Home Directory: %s\n", context.HomeDir)
	
	if context.GitBranch != "" {
		fmt.Printf("Git Branch: %s\n", context.GitBranch)
	}
	
	if context.GitRepo != "" {
		fmt.Printf("Git Repository: %s\n", context.GitRepo)
	}
	
	if context.ProjectType != "" {
		fmt.Printf("Project Type: %s\n", context.ProjectType)
	}
	
	if len(context.LastCommands) > 0 {
		fmt.Println("\nLast Commands:")
		for i, cmd := range context.LastCommands {
			fmt.Printf("  %d: %s\n", i+1, cmd)
		}
	}
	
	if len(context.ShellEnvironment) > 0 {
		fmt.Println("\nImportant Environment Variables:")
		for key, value := range context.ShellEnvironment {
			fmt.Printf("  %s=%s\n", key, value)
		}
	}
}

// scanCurrentDirectory scans the current directory for knowledge
func scanCurrentDirectory(ke *KnowledgeExtractor) {
	if !ke.IsEnabled() {
		fmt.Println("Knowledge extractor not enabled")
		fmt.Println("Run ':knowledge enable' to enable")
		return
	}

	// Get current directory
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v\n", err)
		return
	}

	fmt.Printf("Scanning directory: %s\n", currentDir)
	fmt.Println("------------------------------------------")

	// Update context with current directory
	err = ke.UpdateContext(currentDir)
	if err != nil {
		fmt.Printf("Error scanning directory: %v\n", err)
		return
	}

	// Get knowledge items
	items := ke.GetKnowledgeItems()

	// Count items by source
	sourceCounts := make(map[string]int)
	for _, item := range items {
		sourceCounts[item.Source]++
	}

	fmt.Println("Scan completed successfully!")
	fmt.Printf("Found %d knowledge items:\n", len(items))
	
	for source, count := range sourceCounts {
		fmt.Printf("  %s: %d items\n", source, count)
	}

	// Show project info
	projectInfo := ke.GetProjectInfo()
	if projectInfo.Type != "" {
		fmt.Printf("\nDetected Project: %s\n", projectInfo.Name)
		fmt.Printf("Project Type: %s\n", projectInfo.Type)
		
		if projectInfo.Version != "" {
			fmt.Printf("Version: %s\n", projectInfo.Version)
		}
		
		if len(projectInfo.Languages) > 0 {
			fmt.Printf("Languages: %s\n", strings.Join(projectInfo.Languages, ", "))
		}
		
		if len(projectInfo.Dependencies) > 0 {
			fmt.Printf("Dependencies: %d found\n", len(projectInfo.Dependencies))
		}
	}
}

// showProjectInfo displays project information
func showProjectInfo(ke *KnowledgeExtractor) {
	if !ke.IsEnabled() {
		fmt.Println("Knowledge extractor not enabled")
		fmt.Println("Run ':knowledge enable' to enable")
		return
	}

	// Get current directory
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v\n", err)
		return
	}

	// Update context with current directory
	err = ke.UpdateContext(currentDir)
	if err != nil {
		fmt.Printf("Error updating context: %v\n", err)
		return
	}

	// Get project info
	projectInfo := ke.GetProjectInfo()

	fmt.Println("Project Information")
	fmt.Println("==================")
	
	if projectInfo.Type == "" {
		fmt.Println("No project detected in current directory")
		return
	}
	
	fmt.Printf("Name: %s\n", projectInfo.Name)
	fmt.Printf("Type: %s\n", projectInfo.Type)
	
	if projectInfo.Version != "" {
		fmt.Printf("Version: %s\n", projectInfo.Version)
	}
	
	if projectInfo.BuildSystem != "" {
		fmt.Printf("Build System: %s\n", projectInfo.BuildSystem)
	}
	
	if projectInfo.TestFramework != "" {
		fmt.Printf("Test Framework: %s\n", projectInfo.TestFramework)
	}
	
	if len(projectInfo.Languages) > 0 {
		fmt.Printf("\nLanguages:\n")
		for _, lang := range projectInfo.Languages {
			fmt.Printf("  - %s\n", lang)
		}
	}
	
	if len(projectInfo.Dependencies) > 0 {
		fmt.Printf("\nDependencies:\n")
		for i, dep := range projectInfo.Dependencies {
			if i >= 10 {
				fmt.Printf("  ... and %d more\n", len(projectInfo.Dependencies)-10)
				break
			}
			fmt.Printf("  - %s\n", dep)
		}
	}
	
	if projectInfo.RepoURL != "" {
		fmt.Printf("\nRepository: %s\n", projectInfo.RepoURL)
	}
	
	if projectInfo.Branch != "" {
		fmt.Printf("Branch: %s\n", projectInfo.Branch)
	}
}

// extractFromCommand extracts knowledge from a command
func extractFromCommand(ke *KnowledgeExtractor, command string) {
	if !ke.IsEnabled() {
		fmt.Println("Knowledge extractor not enabled")
		fmt.Println("Run ':knowledge enable' to enable")
		return
	}

	fmt.Printf("Extracting knowledge from command: %s\n", command)
	fmt.Println("------------------------------------------")

	// Get current directory
	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v\n", err)
		return
	}

	// Process command
	err = ke.AddCommand(command, currentDir, 0)
	if err != nil {
		fmt.Printf("Error processing command: %v\n", err)
		return
	}

	fmt.Println("Command processed successfully!")

	// Get knowledge items
	items := ke.GetKnowledgeItems()
	fmt.Printf("Total knowledge items: %d\n", len(items))
}

// clearKnowledge clears all knowledge entities
func clearKnowledge(ke *KnowledgeExtractor) {
	if !ke.IsEnabled() {
		fmt.Println("Knowledge extractor not enabled")
		fmt.Println("Run ':knowledge enable' to enable")
		return
	}

	fmt.Println("This will clear all knowledge entities.")
	fmt.Println("Are you sure you want to continue? (y/n)")

	var response string
	fmt.Scanln(&response)

	if strings.ToLower(response) != "y" {
		fmt.Println("Operation cancelled")
		return
	}

	// Clear entities
	err := ke.ClearEntities()
	if err != nil {
		fmt.Printf("Error clearing entities: %v\n", err)
		return
	}

	fmt.Println("All knowledge entities cleared successfully!")
}

// exportKnowledge exports knowledge to a file
func exportKnowledge(ke *KnowledgeExtractor, filePath string) {
	if !ke.IsEnabled() {
		fmt.Println("Knowledge extractor not enabled")
		fmt.Println("Run ':knowledge enable' to enable")
		return
	}

	// Convert to absolute path if needed
	if !strings.HasPrefix(filePath, "/") {
		currentDir, err := os.Getwd()
		if err != nil {
			fmt.Printf("Error getting current directory: %v\n", err)
			return
		}
		filePath = filepath.Join(currentDir, filePath)
	}

	// Export entities
	err := ke.ExportEntities(filePath)
	if err != nil {
		fmt.Printf("Error exporting knowledge: %v\n", err)
		return
	}

	fmt.Printf("Knowledge exported successfully to: %s\n", filePath)
}

// importKnowledge imports knowledge from a file
func importKnowledge(ke *KnowledgeExtractor, filePath string) {
	if !ke.IsEnabled() {
		fmt.Println("Knowledge extractor not enabled")
		fmt.Println("Run ':knowledge enable' to enable")
		return
	}

	// Convert to absolute path if needed
	if !strings.HasPrefix(filePath, "/") {
		currentDir, err := os.Getwd()
		if err != nil {
			fmt.Printf("Error getting current directory: %v\n", err)
			return
		}
		filePath = filepath.Join(currentDir, filePath)
	}

	// Import entities
	err := ke.ImportEntities(filePath)
	if err != nil {
		fmt.Printf("Error importing knowledge: %v\n", err)
		return
	}

	fmt.Printf("Knowledge imported successfully from: %s\n", filePath)
}

// showKnowledgeHelp displays help for knowledge commands
func showKnowledgeHelp() {
	fmt.Println("Knowledge Extractor Commands")
	fmt.Println("===========================")
	fmt.Println("  :knowledge              - Show knowledge extractor status")
	fmt.Println("  :knowledge enable       - Initialize and enable knowledge extractor")
	fmt.Println("  :knowledge disable      - Disable knowledge extractor")
	fmt.Println("  :knowledge status       - Show status")
	fmt.Println("  :knowledge stats        - Show detailed statistics")
	fmt.Println("  :knowledge query <text> - Search for knowledge")
	fmt.Println("  :knowledge context      - Show current environment context")
	fmt.Println("  :knowledge scan         - Scan current directory for knowledge")
	fmt.Println("  :knowledge project      - Show project information")
	fmt.Println("  :knowledge extract <cmd> - Extract knowledge from command")
	fmt.Println("  :knowledge clear        - Clear all knowledge entities")
	fmt.Println("  :knowledge export <file> - Export knowledge to file")
	fmt.Println("  :knowledge import <file> - Import knowledge from file")
	fmt.Println("  :knowledge help         - Show this help message")
}

// Helper functions

// formatKnowledgeDuration formats a duration in a user-friendly way
func formatKnowledgeDuration(d time.Duration) string {
	if d.Hours() > 24 {
		days := int(d.Hours() / 24)
		return fmt.Sprintf("%d days ago", days)
	} else if d.Hours() >= 1 {
		hours := int(d.Hours())
		return fmt.Sprintf("%d hours ago", hours)
	} else if d.Minutes() >= 1 {
		minutes := int(d.Minutes())
		return fmt.Sprintf("%d minutes ago", minutes)
	} else {
		seconds := int(d.Seconds())
		return fmt.Sprintf("%d seconds ago", seconds)
	}
}