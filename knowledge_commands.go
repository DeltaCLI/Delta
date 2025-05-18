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
	case "agent":
		// Agent-related knowledge commands
		return HandleKnowledgeExtractorAgentCommand(args[1:])

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

	// Show vector database integration status
	fmt.Println("\nVector Database Integration:")
	vectorDBEnabled := false
	if enabled, ok := stats["vector_db_enabled"].(bool); ok {
		vectorDBEnabled = enabled
	}
	
	if vectorDBEnabled {
		fmt.Println("Status: Enabled and connected")
		
		// Show vector count
		if count, ok := stats["vector_db_count"].(int); ok {
			fmt.Printf("Indexed Items: %d\n", count)
		}
		
		// Show similarity metric
		if metric, ok := stats["vector_db_metric"].(string); ok {
			fmt.Printf("Similarity Metric: %s\n", metric)
		}
		
		// Show vector extension status
		if hasExt, ok := stats["vector_db_extension"].(bool); ok {
			if hasExt {
				fmt.Println("SQLite Vector Extension: Available")
			} else {
				fmt.Println("SQLite Vector Extension: Not available (using in-memory fallback)")
			}
		}
		
		// Show embedding coverage
		totalItems := stats["total_items"].(int)
		itemsWithEmbeddings := stats["items_with_embeddings"].(int)
		
		if totalItems > 0 {
			coverage := float64(itemsWithEmbeddings) / float64(totalItems) * 100
			fmt.Printf("Embedding Coverage: %.1f%% (%d/%d items)\n", 
				coverage, itemsWithEmbeddings, totalItems)
		}
	} else {
		fmt.Println("Status: Not connected")
		fmt.Println("Run ':vector enable' to enable vector database")
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

	// Initialize vector database if available
	vectorDB := GetVectorDBManager()
	if vectorDB != nil && !vectorDB.IsEnabled() {
		err := vectorDB.Initialize()
		if err == nil {
			vectorDB.Enable()
		}
	}

	// Generate embeddings for items first (if not already done)
	fmt.Println("Generating embeddings for semantic search...")
	err := ke.GenerateEmbeddings()
	if err != nil {
		fmt.Printf("Warning: Error generating embeddings: %v\n", err)
		fmt.Println("Falling back to text-based search")
	}

	// Start time to measure search duration
	startTime := time.Now()

	// Search for knowledge
	results, err := ke.SearchKnowledge(query, 10)
	
	// Calculate search duration
	searchDuration := time.Since(startTime)

	if err != nil {
		fmt.Printf("Error searching knowledge: %v\n", err)
		fmt.Println("Falling back to text-based search")
		
		// Try again with text-based search only
		startTime = time.Now()
		results = ke.searchKnowledgeWithText(query, 10)
		searchDuration = time.Since(startTime)
	}

	if len(results) == 0 {
		fmt.Println("No matching knowledge found")
		return
	}

	// Display results
	fmt.Printf("Found %d matching items in %s:\n\n", len(results), formatDuration(searchDuration))

	// Determine if results are from vector search or text search
	searchMethod := "semantic vector"
	if err != nil {
		searchMethod = "text-based"
	}
	fmt.Printf("Search method: %s search\n\n", searchMethod)

	for i, item := range results {
		// Check if item is synthetic (from vector DB but not in memory)
		isSynthetic := false
		if val, ok := item.Metadata["synthetic"]; ok && val == "true" {
			isSynthetic = true
		}

		// Show item type and pattern
		fmt.Printf("%d. [%s] %s\n", i+1, item.Type, item.Pattern)
		
		// Show examples
		if len(item.Examples) > 0 {
			if len(item.Examples) == 1 {
				fmt.Printf("   Example: %s\n", item.Examples[0])
			} else {
				fmt.Printf("   Examples: %d available\n", len(item.Examples))
				// Show first example
				fmt.Printf("     - %s\n", item.Examples[0])
				
				// Show second example if available
				if len(item.Examples) > 1 {
					fmt.Printf("     - %s\n", item.Examples[1])
				}
			}
		}
		
		// Show metadata
		fmt.Printf("   Confidence: %.2f, Usage: %d\n", item.Confidence, item.UsageCount)
		
		// Show last update time
		if !item.LastUpdated.IsZero() {
			fmt.Printf("   Last Updated: %s\n", formatTimeSince(time.Since(item.LastUpdated)))
		}
		
		// Show directory if available
		if dir, ok := item.Metadata["directory"]; ok && dir != "" {
			fmt.Printf("   Context: %s\n", dir)
		}
		
		// Show if item is synthetic
		if isSynthetic {
			fmt.Printf("   [From vector database]\n")
		}
		
		fmt.Println()
	}
	
	// Check vector database status
	if vectorDB != nil && vectorDB.IsEnabled() {
		stats := vectorDB.GetStats()
		vectorCount := 0
		if count, ok := stats["vector_count"].(int); ok {
			vectorCount = count
		}
		
		fmt.Printf("Vector database status: %d entries indexed\n", vectorCount)
		
		// Show which similarity metric was used
		if metric, ok := stats["metric"].(string); ok {
			fmt.Printf("Similarity metric: %s\n", metric)
		}
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
	
	// System information
	fmt.Println("System Information:")
	fmt.Printf("  OS: %s\n", context.OS)
	fmt.Printf("  Architecture: %s\n", context.Arch)
	fmt.Printf("  Hostname: %s\n", context.Hostname)
	fmt.Printf("  User: %s\n", context.User)
	
	// Shell information
	fmt.Println("\nShell Environment:")
	fmt.Printf("  Shell: %s\n", context.Shell)
	fmt.Printf("  Current Directory: %s\n", context.CurrentDir)
	fmt.Printf("  Home Directory: %s\n", context.HomeDir)
	
	// Git information
	if context.GitBranch != "" || context.GitRepo != "" {
		fmt.Println("\nGit Information:")
		if context.GitBranch != "" {
			fmt.Printf("  Branch: %s\n", context.GitBranch)
		}
		if context.GitRepo != "" {
			fmt.Printf("  Repository: %s\n", context.GitRepo)
		}
	}
	
	// Project information
	if context.ProjectType != "" {
		fmt.Println("\nProject Information:")
		fmt.Printf("  Type: %s\n", context.ProjectType)
		
		// Get detailed project info
		projectInfo := ke.GetProjectInfo()
		
		if len(projectInfo.Languages) > 0 {
			fmt.Printf("  Languages: %s\n", strings.Join(projectInfo.Languages, ", "))
		}
		
		if projectInfo.BuildSystem != "" && projectInfo.BuildSystem != "unknown" {
			fmt.Printf("  Build System: %s\n", projectInfo.BuildSystem)
		}
		
		if projectInfo.TestFramework != "" && projectInfo.TestFramework != "unknown" {
			fmt.Printf("  Test Framework: %s\n", projectInfo.TestFramework)
		}
		
		if projectInfo.Version != "" {
			fmt.Printf("  Version: %s\n", projectInfo.Version)
		}
		
		if len(projectInfo.Dependencies) > 0 {
			fmt.Printf("  Dependencies: %d detected\n", len(projectInfo.Dependencies))
			
			// Show top dependencies (limited to 5)
			maxShow := 5
			if len(projectInfo.Dependencies) < maxShow {
				maxShow = len(projectInfo.Dependencies)
			}
			
			fmt.Println("  Top Dependencies:")
			for i := 0; i < maxShow; i++ {
				fmt.Printf("    - %s\n", projectInfo.Dependencies[i])
			}
		}
		
		// Show file statistics
		if len(context.FileExtensions) > 0 {
			fmt.Println("\nFile Extensions:")
			
			// Get top extensions by count
			type extCount struct {
				ext   string
				count int
			}
			
			extensions := make([]extCount, 0, len(context.FileExtensions))
			for ext, count := range context.FileExtensions {
				extensions = append(extensions, extCount{ext, count})
			}
			
			// Sort by count descending
			sort.Slice(extensions, func(i, j int) bool {
				return extensions[i].count > extensions[j].count
			})
			
			// Show top extensions (limited to 8)
			maxShow := 8
			if len(extensions) < maxShow {
				maxShow = len(extensions)
			}
			
			for i := 0; i < maxShow; i++ {
				fmt.Printf("  %s: %d files\n", extensions[i].ext, extensions[i].count)
			}
		}
	}
	
	// Detected tools
	if len(context.DetectedTools) > 0 {
		fmt.Println("\nDetected Tools:")
		toolCount := 0
		for tool, version := range context.DetectedTools {
			fmt.Printf("  %s: %s\n", tool, version)
			toolCount++
			if toolCount >= 8 {
				fmt.Printf("  (and %d more...)\n", len(context.DetectedTools)-toolCount)
				break
			}
		}
	}
	
	// Package managers
	packageManagerCount := 0
	for pm, installed := range context.PackageManagers {
		if installed {
			packageManagerCount++
		}
	}
	
	if packageManagerCount > 0 {
		fmt.Println("\nPackage Managers:")
		count := 0
		for pm, installed := range context.PackageManagers {
			if installed {
				fmt.Printf("  %s\n", pm)
				count++
				if count >= 5 {
					break
				}
			}
		}
	}
	
	// Docker information
	if len(context.DockerInfo) > 0 && context.DockerInfo["installed"] == "true" {
		fmt.Println("\nDocker Information:")
		fmt.Printf("  Version: %s\n", context.DockerInfo["version"])
	}
	
	// Kubernetes information
	if len(context.KubernetesInfo) > 0 && context.KubernetesInfo["installed"] == "true" {
		fmt.Println("\nKubernetes Information:")
		fmt.Printf("  Version: %s\n", context.KubernetesInfo["version"])
	}
	
	// Runtime versions
	if len(context.RuntimeVersions) > 0 {
		fmt.Println("\nRuntime Versions:")
		for runtime, version := range context.RuntimeVersions {
			fmt.Printf("  %s: %s\n", runtime, version)
		}
	}
	
	// Last commands
	if len(context.LastCommands) > 0 {
		fmt.Println("\nRecent Commands:")
		for i, cmd := range context.LastCommands {
			fmt.Printf("  %d: %s\n", i+1, cmd)
		}
	}
	
	// Important environment variables (limited view)
	if len(context.ShellEnvironment) > 0 {
		fmt.Println("\nKey Environment Variables:")
		
		// Show only important variables
		importantVars := []string{"GOPATH", "GOROOT", "JAVA_HOME", "PYTHONPATH", "NODE_PATH", "PATH"}
		
		for _, key := range importantVars {
			if value, ok := context.ShellEnvironment[key]; ok {
				if len(value) > 50 {
					// Truncate long values
					value = value[:47] + "..."
				}
				fmt.Printf("  %s=%s\n", key, value)
			}
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

// showProjectInfo displays detailed project information
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

	// Get project info and environment context
	projectInfo := ke.GetProjectInfo()
	context := ke.GetCurrentContext()

	fmt.Println("Project Information")
	fmt.Println("==================")
	
	if projectInfo.Type == "" {
		fmt.Println("No project detected in current directory")
		return
	}
	
	// Basic project information
	fmt.Printf("Name: %s\n", projectInfo.Name)
	fmt.Printf("Path: %s\n", projectInfo.Path)
	fmt.Printf("Type: %s\n", projectInfo.Type)
	
	if projectInfo.Version != "" {
		fmt.Printf("Version: %s\n", projectInfo.Version)
	}
	
	// Repository information
	if projectInfo.RepoURL != "" || projectInfo.Branch != "" {
		fmt.Println("\nRepository Information:")
		if projectInfo.RepoURL != "" {
			fmt.Printf("  URL: %s\n", projectInfo.RepoURL)
		}
		if projectInfo.Branch != "" {
			fmt.Printf("  Branch: %s\n", projectInfo.Branch)
		}
		if len(projectInfo.Contributors) > 0 {
			fmt.Println("  Contributors:")
			for _, contributor := range projectInfo.Contributors {
				fmt.Printf("    - %s\n", contributor)
			}
		}
		fmt.Printf("  Last Modified: %s\n", projectInfo.LastModified.Format(time.RFC1123))
	}
	
	// Development information
	fmt.Println("\nDevelopment Information:")
	if projectInfo.BuildSystem != "" && projectInfo.BuildSystem != "unknown" {
		fmt.Printf("  Build System: %s\n", projectInfo.BuildSystem)
	}
	if projectInfo.TestFramework != "" && projectInfo.TestFramework != "unknown" {
		fmt.Printf("  Test Framework: %s\n", projectInfo.TestFramework)
	}
	
	// Languages
	if len(projectInfo.Languages) > 0 {
		fmt.Printf("  Languages: %s\n", strings.Join(projectInfo.Languages, ", "))
	}
	
	// Code statistics
	if len(projectInfo.CodeStats) > 0 {
		fmt.Println("\nCode Statistics:")
		
		// Get total file count by extension
		totalFiles := 0
		for _, count := range projectInfo.CodeStats {
			totalFiles += count
		}
		fmt.Printf("  Total Files: %d\n", totalFiles)
		
		// Display top extensions by count
		type extCount struct {
			ext   string
			count int
		}
		
		extensions := make([]extCount, 0, len(projectInfo.CodeStats))
		for ext, count := range projectInfo.CodeStats {
			extensions = append(extensions, extCount{ext, count})
		}
		
		// Sort by count descending
		sort.Slice(extensions, func(i, j int) bool {
			return extensions[i].count > extensions[j].count
		})
		
		// Show top extensions (limited to 8)
		fmt.Println("  File Types:")
		maxShow := 8
		if len(extensions) < maxShow {
			maxShow = len(extensions)
		}
		
		for i := 0; i < maxShow; i++ {
			percentage := float64(extensions[i].count) / float64(totalFiles) * 100
			fmt.Printf("    %s: %d files (%.1f%%)\n", extensions[i].ext, extensions[i].count, percentage)
		}
	}
	
	// Dependencies
	if len(projectInfo.Dependencies) > 0 {
		fmt.Printf("\nDependencies (%d total):\n", len(projectInfo.Dependencies))
		
		// Group dependencies by type if possible
		devDeps := []string{}
		mainDeps := []string{}
		
		for _, dep := range projectInfo.Dependencies {
			if strings.Contains(dep, "(dev)") {
				devDeps = append(devDeps, dep)
			} else {
				mainDeps = append(mainDeps, dep)
			}
		}
		
		// Show main dependencies
		fmt.Println("  Main Dependencies:")
		for i, dep := range mainDeps {
			if i >= 8 {
				fmt.Printf("    ... and %d more\n", len(mainDeps)-8)
				break
			}
			fmt.Printf("    - %s\n", dep)
		}
		
		// Show dev dependencies if any
		if len(devDeps) > 0 {
			fmt.Println("  Dev Dependencies:")
			maxShow := 5
			if len(devDeps) < maxShow {
				maxShow = len(devDeps)
			}
			
			for i := 0; i < maxShow; i++ {
				fmt.Printf("    - %s\n", devDeps[i])
			}
			
			if len(devDeps) > maxShow {
				fmt.Printf("    ... and %d more\n", len(devDeps)-maxShow)
			}
		}
	}
	
	// Environment tools
	relevantTools := map[string]bool{}
	
	// Add relevant tools based on project type
	switch projectInfo.Type {
	case "go":
		relevantTools["go"] = true
		relevantTools["make"] = true
		relevantTools["git"] = true
	case "javascript", "typescript":
		relevantTools["node"] = true
		relevantTools["npm"] = true
		relevantTools["yarn"] = true
		relevantTools["git"] = true
	case "python":
		relevantTools["python"] = true
		relevantTools["python3"] = true
		relevantTools["pip"] = true
		relevantTools["pip3"] = true
		relevantTools["git"] = true
	case "java":
		relevantTools["java"] = true
		relevantTools["javac"] = true
		relevantTools["maven"] = true
		relevantTools["gradle"] = true
		relevantTools["git"] = true
	case "rust":
		relevantTools["rust"] = true
		relevantTools["cargo"] = true
		relevantTools["git"] = true
	}
	
	// Display detected relevant tools
	detectedRelevantTools := []string{}
	for tool := range context.DetectedTools {
		if relevantTools[tool] {
			detectedRelevantTools = append(detectedRelevantTools, tool)
		}
	}
	
	if len(detectedRelevantTools) > 0 {
		fmt.Println("\nDetected Development Tools:")
		for _, tool := range detectedRelevantTools {
			fmt.Printf("  - %s: %s\n", tool, context.DetectedTools[tool])
		}
	}
	
	// Show README preview if available
	if projectInfo.Readme != "" {
		fmt.Println("\nREADME Preview:")
		
		// Split into lines and show first few
		lines := strings.Split(projectInfo.Readme, "\n")
		maxLines := 8
		if len(lines) < maxLines {
			maxLines = len(lines)
		}
		
		for i := 0; i < maxLines; i++ {
			// Truncate long lines
			line := lines[i]
			if len(line) > 70 {
				line = line[:67] + "..."
			}
			fmt.Printf("  %s\n", line)
		}
		
		if len(lines) > maxLines {
			fmt.Println("  (README preview truncated)")
		}
	}
	
	// Project-specific information based on type
	switch projectInfo.Type {
	case "go":
		fmt.Println("\nGo-specific Information:")
		fmt.Printf("  GOPATH: %s\n", os.Getenv("GOPATH"))
		fmt.Printf("  Go Version: %s\n", runtime.Version())
	case "javascript", "typescript":
		fmt.Println("\nNode-specific Information:")
		if version, ok := context.RuntimeVersions["node"]; ok {
			fmt.Printf("  Node Version: %s\n", version)
		}
	case "python":
		fmt.Println("\nPython-specific Information:")
		if version, ok := context.RuntimeVersions["python"]; ok {
			fmt.Printf("  Python Version: %s\n", version)
		}
		fmt.Printf("  Virtual Environment: %s\n", os.Getenv("VIRTUAL_ENV"))
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
	fmt.Println("  :knowledge agent        - Agent-related knowledge commands")
	fmt.Println("  :knowledge help         - Show this help message")

	fmt.Println("\nKnowledge Agent Commands:")
	fmt.Println("  :knowledge agent suggest    - Suggest agents based on knowledge")
	fmt.Println("  :knowledge agent learn      - Learn patterns from existing agents")
	fmt.Println("  :knowledge agent optimize   - Optimize agents using knowledge")
	fmt.Println("  :knowledge agent create     - Create agents from knowledge")
	fmt.Println("  :knowledge agent discover   - Discover potential agents")
	fmt.Println("  :knowledge agent help       - Show agent-specific help")
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