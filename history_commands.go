package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// HandleHistoryCommand processes history-related commands
func HandleHistoryCommand(args []string) bool {
	// Get the HistoryAnalyzer instance
	ha := GetHistoryAnalyzer()
	if ha == nil {
		fmt.Println("Failed to initialize history analyzer")
		return true
	}

	// Initialize if not already done
	if !ha.isInitialized {
		err := ha.Initialize()
		if err != nil {
			fmt.Printf("Error initializing history analyzer: %v\n", err)
			return true
		}
	}

	// Handle commands
	if len(args) == 0 {
		// Default command: show recent history
		showRecentHistory(ha, 10)
		return true
	}

	switch args[0] {
	case "status":
		// Show history analysis status
		showHistoryStatus(ha)
		return true

	case "stats":
		// Show history statistics
		showHistoryStats(ha)
		return true
		
	case "analyze":
		// Force pattern analysis
		fmt.Println("Analyzing command history to detect patterns...")
		patterns := ha.findCommandPatterns()
		workflows := ha.identifyTaskWorkflows()
		fmt.Printf("Found %d command patterns and %d workflows\n", len(patterns), len(workflows))
		fmt.Println("Use ':history patterns' to view the results")
		return true

	case "enable":
		// Enable history analysis
		ha.EnableHistoryAnalysis()
		fmt.Println("History analysis enabled")
		return true

	case "disable":
		// Disable history analysis
		ha.DisableHistoryAnalysis()
		fmt.Println("History analysis disabled")
		return true

	case "search", "find":
		// Search history
		if len(args) < 2 {
			fmt.Println("Usage: :history search <query>")
			return true
		}
		query := strings.Join(args[1:], " ")
		searchHistory(ha, query)
		return true

	case "show":
		// Show recent history with specified limit
		limit := 10 // Default
		if len(args) > 1 {
			var err error
			limit, err = strconv.Atoi(args[1])
			if err != nil || limit <= 0 {
				fmt.Println("Invalid limit. Using default of 10.")
				limit = 10
			}
		}
		showRecentHistory(ha, limit)
		return true

	case "suggest":
		// Show command suggestions for current context
		showSuggestions(ha)
		return true

	case "config":
		// Configure history analysis
		if len(args) < 2 {
			showHistoryConfig(ha)
		} else {
			updateHistoryConfig(ha, args[1:])
		}
		return true

	case "mark":
		// Mark a command as important
		if len(args) < 3 || (args[1] != "important" && args[1] != "unimportant") {
			fmt.Println("Usage: :history mark important|unimportant <command>")
			return true
		}
		isImportant := args[1] == "important"
		command := strings.Join(args[2:], " ")
		ha.MarkCommandImportant(command, isImportant)
		fmt.Printf("Marked command '%s' as %s\n", command, args[1])
		return true

	case "patterns":
		// Show identified command patterns
		showCommandPatterns(ha)
		return true

	case "info":
		// Show info about a specific command
		if len(args) < 2 {
			fmt.Println("Usage: :history info <command>")
			return true
		}
		command := strings.Join(args[1:], " ")
		showCommandInfo(ha, command)
		return true

	case "help":
		// Show help
		showHistoryHelp()
		return true

	default:
		// Unknown command, try to interpret as a search query
		query := strings.Join(args, " ")
		searchHistory(ha, query)
		return true
	}
}

// showRecentHistory displays recent command history
func showRecentHistory(ha *HistoryAnalyzer, limit int) {
	// For a basic history display, we'll just sort by last used timestamp
	ha.historyLock.RLock()
	defer ha.historyLock.RUnlock()

	// Sort history by last used (most recent first)
	sortedHistory := make([]EnhancedHistoryEntry, len(ha.history))
	copy(sortedHistory, ha.history)
	sort.Slice(sortedHistory, func(i, j int) bool {
		return sortedHistory[i].LastUsed.After(sortedHistory[j].LastUsed)
	})

	// Display only up to the limit
	if len(sortedHistory) > limit {
		sortedHistory = sortedHistory[:limit]
	}

	// Display history
	fmt.Println("Recent Command History")
	fmt.Println("=====================")
	if len(sortedHistory) == 0 {
		fmt.Println("No history available yet.")
		return
	}

	for i, entry := range sortedHistory {
		// Format the timestamp
		timeStr := entry.LastUsed.Format("2006-01-02 15:04:05")
		
		// Mark important commands with a star
		star := " "
		if entry.IsImportant {
			star = "*"
		}
		
		// Format the command (truncate if too long)
		cmd := entry.Command
		if len(cmd) > 60 {
			cmd = cmd[:57] + "..."
		}
		
		fmt.Printf("%s %3d. [%s] %s (%s, %dx)\n", 
			star, i+1, timeStr, cmd, entry.Category, entry.Frequency)
	}
}

// showHistoryStatus displays the history analysis status
func showHistoryStatus(ha *HistoryAnalyzer) {
	ha.historyLock.RLock()
	defer ha.historyLock.RUnlock()

	fmt.Println("History Analysis Status")
	fmt.Println("======================")
	fmt.Printf("Enabled: %v\n", ha.config.Enabled)
	fmt.Printf("Total entries: %d\n", len(ha.history))
	fmt.Printf("Unique commands: %d\n", len(ha.commandFrequency))
	
	// Count total command executions
	totalExecutions := 0
	for _, freq := range ha.commandFrequency {
		totalExecutions += freq
	}
	fmt.Printf("Total command executions: %d\n", totalExecutions)
	
	// Find most popular commands
	type CommandCount struct {
		Command string
		Count   int
	}
	var topCommands []CommandCount
	for cmd, count := range ha.commandFrequency {
		topCommands = append(topCommands, CommandCount{cmd, count})
	}
	
	// Sort by frequency (most used first)
	sort.Slice(topCommands, func(i, j int) bool {
		return topCommands[i].Count > topCommands[j].Count
	})
	
	// Show top 5 most used commands
	fmt.Println("\nMost Used Commands:")
	for i := 0; i < min(5, len(topCommands)); i++ {
		cmd := topCommands[i].Command
		if len(cmd) > 60 {
			cmd = cmd[:57] + "..."
		}
		fmt.Printf("  %d. %s (%d times)\n", i+1, cmd, topCommands[i].Count)
	}
	
	// Show command sequence info
	fmt.Printf("\nCommand sequences tracked: %d\n", len(ha.commandSequences))
	
	// Show auto-suggest status
	fmt.Printf("Auto-suggestions: %v (threshold: %.2f)\n", 
		ha.config.AutoSuggest, ha.config.MinConfidenceThreshold)
}

// showHistoryStats displays detailed history statistics
func showHistoryStats(ha *HistoryAnalyzer) {
	stats := ha.GetHistoryStats()
	if stats == nil {
		fmt.Println("History statistics not available")
		return
	}

	fmt.Println("History Statistics")
	fmt.Println("=================")
	fmt.Printf("Total entries: %d\n", stats["total_entries"])
	fmt.Printf("Unique commands: %d\n", stats["unique_commands"])
	fmt.Printf("Total command executions: %d\n", stats["total_command_executions"])

	// Define CommandCount type
	type CommandCount struct {
		Command string
		Count   int
	}

	// Show top commands
	fmt.Println("\nMost Used Commands:")
	topCommands := stats["top_commands"].([]CommandCount)
	for i, cmd := range topCommands {
		cmdText := cmd.Command
		if len(cmdText) > 60 {
			cmdText = cmdText[:57] + "..."
		}
		fmt.Printf("  %d. %s (%d times)\n", i+1, cmdText, cmd.Count)
	}
	
	// Show category stats
	fmt.Println("\nCommand Categories:")
	categoryStats := stats["category_stats"].(map[string]int)
	categories := make([]string, 0, len(categoryStats))
	for category := range categoryStats {
		categories = append(categories, category)
	}
	sort.Slice(categories, func(i, j int) bool {
		return categoryStats[categories[i]] > categoryStats[categories[j]]
	})
	
	for _, category := range categories {
		fmt.Printf("  %s: %d commands\n", category, categoryStats[category])
	}
	
	// Show time distribution
	fmt.Println("\nTime Distribution:")
	timeStats := stats["time_stats"].(map[int]int)
	hours := make([]int, 0, len(timeStats))
	for hour := range timeStats {
		hours = append(hours, hour)
	}
	sort.Ints(hours)
	
	// Find the max count for scaling
	maxCount := 0
	for _, count := range timeStats {
		if count > maxCount {
			maxCount = count
		}
	}
	
	// Show a simple text-based histogram
	for _, hour := range hours {
		hourStr := fmt.Sprintf("%02d:00", hour)
		count := timeStats[hour]
		barLength := count * 40 / maxCount
		bar := strings.Repeat("■", barLength)
		fmt.Printf("  %s: %s (%d)\n", hourStr, bar, count)
	}
	
	// Show command sequences
	fmt.Printf("\nCommand sequences tracked: %d\n", stats["command_sequences"])
}

// searchHistory searches command history
func searchHistory(ha *HistoryAnalyzer, query string) {
	results := ha.SearchHistory(query, 10)
	
	fmt.Printf("Search Results for '%s'\n", query)
	fmt.Println("========================" + strings.Repeat("=", len(query)))
	
	if len(results) == 0 {
		fmt.Println("No matching commands found.")
		return
	}
	
	for i, entry := range results {
		// Format the timestamp
		timeStr := entry.LastUsed.Format("2006-01-02 15:04:05")
		
		// Mark important commands with a star
		star := " "
		if entry.IsImportant {
			star = "*"
		}
		
		// Format the command (truncate if too long)
		cmd := entry.Command
		if len(cmd) > 60 {
			cmd = cmd[:57] + "..."
		}
		
		fmt.Printf("%s %3d. [%s] %s (%s, %dx)\n", 
			star, i+1, timeStr, cmd, entry.Category, entry.Frequency)
	}
}

// showSuggestions displays command suggestions for the current context
func showSuggestions(ha *HistoryAnalyzer) {
	// Get current directory
	dir, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v\n", err)
		return
	}
	
	suggestions := ha.GetSuggestions(dir)
	
	fmt.Println("Command Suggestions")
	fmt.Println("==================")
	
	if len(suggestions) == 0 {
		fmt.Println("No suggestions available yet.")
		fmt.Println("Try using more commands to build up history data.")
		return
	}
	
	for i, suggestion := range suggestions {
		// Format the command
		cmd := suggestion.Command
		if len(cmd) > 60 {
			cmd = cmd[:57] + "..."
		}
		
		// Format confidence as percentage
		confidence := int(suggestion.Confidence * 100)
		
		// Show if it's part of a sequence
		seqInfo := ""
		if suggestion.IsSequence {
			seqInfo = fmt.Sprintf(" (part of sequence: %s)", suggestion.SequenceName)
		}
		
		fmt.Printf("%d. %s [%d%%]%s\n", i+1, cmd, confidence, seqInfo)
		fmt.Printf("   Reason: %s\n", suggestion.Reason)
	}
}

// showHistoryConfig displays the history analysis configuration
func showHistoryConfig(ha *HistoryAnalyzer) {
	ha.historyLock.RLock()
	defer ha.historyLock.RUnlock()

	fmt.Println("History Analysis Configuration")
	fmt.Println("=============================")
	fmt.Printf("Enabled: %v\n", ha.config.Enabled)
	fmt.Printf("Max history size: %d\n", ha.config.MaxHistorySize)
	fmt.Printf("Min confidence threshold: %.2f\n", ha.config.MinConfidenceThreshold)
	fmt.Printf("Max suggestions: %d\n", ha.config.MaxSuggestions)
	fmt.Printf("Auto-suggest: %v\n", ha.config.AutoSuggest)
	fmt.Printf("Track command sequences: %v\n", ha.config.TrackCommandSequences)
	fmt.Printf("Sequence max length: %d\n", ha.config.SequenceMaxLength)
	fmt.Printf("Enable natural language search: %v\n", ha.config.EnableNLSearch)
	fmt.Printf("Enable command categories: %v\n", ha.config.EnableCommandCategories)
	
	// Show weights
	fmt.Println("\nSuggestion Weights:")
	fmt.Printf("  Context weight: %.2f\n", ha.config.ContextWeight)
	fmt.Printf("  Time weight: %.2f\n", ha.config.TimeWeight)
	fmt.Printf("  Frequency weight: %.2f\n", ha.config.FrequencyWeight)
	fmt.Printf("  Recency weight: %.2f\n", ha.config.RecencyWeight)
	
	// Show privacy filters
	fmt.Println("\nPrivacy Filters:")
	for _, filter := range ha.config.PrivacyFilter {
		fmt.Printf("  - %s\n", filter)
	}
}

// updateHistoryConfig updates the history analysis configuration
func updateHistoryConfig(ha *HistoryAnalyzer, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: :history config <setting=value>")
		fmt.Println("Available settings: enabled, max_history, threshold, max_suggestions, auto_suggest, ...")
		return
	}

	// Get a copy of the current config
	ha.historyLock.RLock()
	config := ha.config
	ha.historyLock.RUnlock()

	// Process setting=value pairs
	for _, arg := range args {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			fmt.Printf("Invalid setting format: %s (should be setting=value)\n", arg)
			continue
		}

		setting := parts[0]
		value := parts[1]

		switch setting {
		case "enabled":
			if value == "true" || value == "1" || value == "yes" {
				config.Enabled = true
			} else if value == "false" || value == "0" || value == "no" {
				config.Enabled = false
			} else {
				fmt.Printf("Invalid value for %s: %s (should be true or false)\n", setting, value)
				continue
			}

		case "max_history":
			size, err := strconv.Atoi(value)
			if err != nil || size <= 0 {
				fmt.Printf("Invalid value for %s: %s (should be a positive integer)\n", setting, value)
				continue
			}
			config.MaxHistorySize = size

		case "threshold":
			threshold, err := strconv.ParseFloat(value, 64)
			if err != nil || threshold < 0 || threshold > 1 {
				fmt.Printf("Invalid value for %s: %s (should be between 0.0 and 1.0)\n", setting, value)
				continue
			}
			config.MinConfidenceThreshold = threshold

		case "max_suggestions":
			max, err := strconv.Atoi(value)
			if err != nil || max <= 0 {
				fmt.Printf("Invalid value for %s: %s (should be a positive integer)\n", setting, value)
				continue
			}
			config.MaxSuggestions = max

		case "auto_suggest":
			if value == "true" || value == "1" || value == "yes" {
				config.AutoSuggest = true
			} else if value == "false" || value == "0" || value == "no" {
				config.AutoSuggest = false
			} else {
				fmt.Printf("Invalid value for %s: %s (should be true or false)\n", setting, value)
				continue
			}

		case "track_sequences":
			if value == "true" || value == "1" || value == "yes" {
				config.TrackCommandSequences = true
			} else if value == "false" || value == "0" || value == "no" {
				config.TrackCommandSequences = false
			} else {
				fmt.Printf("Invalid value for %s: %s (should be true or false)\n", setting, value)
				continue
			}

		case "sequence_max_length":
			length, err := strconv.Atoi(value)
			if err != nil || length <= 1 {
				fmt.Printf("Invalid value for %s: %s (should be greater than 1)\n", setting, value)
				continue
			}
			config.SequenceMaxLength = length

		case "nl_search":
			if value == "true" || value == "1" || value == "yes" {
				config.EnableNLSearch = true
			} else if value == "false" || value == "0" || value == "no" {
				config.EnableNLSearch = false
			} else {
				fmt.Printf("Invalid value for %s: %s (should be true or false)\n", setting, value)
				continue
			}

		case "context_weight":
			weight, err := strconv.ParseFloat(value, 64)
			if err != nil || weight < 0 {
				fmt.Printf("Invalid value for %s: %s (should be a positive number)\n", setting, value)
				continue
			}
			config.ContextWeight = weight

		case "time_weight":
			weight, err := strconv.ParseFloat(value, 64)
			if err != nil || weight < 0 {
				fmt.Printf("Invalid value for %s: %s (should be a positive number)\n", setting, value)
				continue
			}
			config.TimeWeight = weight

		case "frequency_weight":
			weight, err := strconv.ParseFloat(value, 64)
			if err != nil || weight < 0 {
				fmt.Printf("Invalid value for %s: %s (should be a positive number)\n", setting, value)
				continue
			}
			config.FrequencyWeight = weight

		case "recency_weight":
			weight, err := strconv.ParseFloat(value, 64)
			if err != nil || weight < 0 {
				fmt.Printf("Invalid value for %s: %s (should be a positive number)\n", setting, value)
				continue
			}
			config.RecencyWeight = weight

		case "enable_categories":
			if value == "true" || value == "1" || value == "yes" {
				config.EnableCommandCategories = true
			} else if value == "false" || value == "0" || value == "no" {
				config.EnableCommandCategories = false
			} else {
				fmt.Printf("Invalid value for %s: %s (should be true or false)\n", setting, value)
				continue
			}

		default:
			fmt.Printf("Unknown setting: %s\n", setting)
			continue
		}

		fmt.Printf("Updated %s to %s\n", setting, value)
	}

	// Save the updated configuration
	err := ha.UpdateConfig(config)
	if err != nil {
		fmt.Printf("Error saving configuration: %v\n", err)
	}
}

// showCommandPatterns displays identified command patterns
func showCommandPatterns(ha *HistoryAnalyzer) {
	ha.historyLock.RLock()
	defer ha.historyLock.RUnlock()

	// Find command patterns
	patterns := ha.findCommandPatterns()
	
	// Display original command sequences
	fmt.Println("Command Sequence Patterns")
	fmt.Println("=======================")
	
	if len(ha.commandSequences) == 0 && len(patterns) == 0 {
		fmt.Println("No command patterns detected yet.")
		fmt.Println("Use more commands to build up a history, or run ':history analyze' to force pattern detection.")
		return
	}
	
	// Display basic command sequences first
	if len(ha.commandSequences) > 0 {
		// Sort sequences by frequency
		sequences := make([]CommandSequence, len(ha.commandSequences))
		copy(sequences, ha.commandSequences)
		sort.Slice(sequences, func(i, j int) bool {
			return sequences[i].Frequency > sequences[j].Frequency
		})
	
	// Display the top 10 sequences
	for i, seq := range sequences {
		if i >= 10 {
			break
		}
		
		fmt.Printf("%d. Sequence: %s (%d times, last: %s)\n", 
			i+1, seq.MeaningfulName, seq.Frequency, 
			seq.LastUsed.Format("2006-01-02 15:04:05"))
		
		fmt.Println("   Commands:")
		for j, cmd := range seq.Commands {
			if len(cmd) > 60 {
				cmd = cmd[:57] + "..."
			}
			fmt.Printf("   %d. %s\n", j+1, cmd)
		}
		fmt.Println()
	}
	}

	// Display advanced patterns if available
	if len(patterns) > 0 {
		fmt.Println("\nAdvanced Command Patterns")
		fmt.Println("=======================")
		
		// Group patterns by type
		patternsByType := make(map[string][]CommandPattern)
		for _, pattern := range patterns {
			patternsByType[pattern.Type] = append(patternsByType[pattern.Type], pattern)
		}
		
		// Display patterns by type
		for patternType, typePatterns := range patternsByType {
			fmt.Printf("\n%s Patterns:\n", strings.Title(patternType))
			
			// Sort by confidence
			sort.Slice(typePatterns, func(i, j int) bool {
				return typePatterns[i].Confidence > typePatterns[j].Confidence
			})
			
			// Display patterns
			for i, pattern := range typePatterns {
				if i >= 5 {
					fmt.Printf("  ... and %d more %s patterns\n", len(typePatterns)-5, patternType)
					break
				}
				
				fmt.Printf("  [%.0f%%] %s\n", pattern.Confidence*100, pattern.Description)
				if pattern.Type == "sequence" {
					fmt.Printf("       %s → %s (used %d times)\n", 
						pattern.Commands[0], pattern.Commands[1], pattern.Frequency)
				} else {
					commandStr := strings.Join(pattern.Commands, ", ")
					fmt.Printf("       %s (used %d times)\n", commandStr, pattern.Frequency)
				}
			}
		}
		
		// Display workflows
		workflows := ha.identifyTaskWorkflows()
		if len(workflows) > 0 {
			fmt.Println("\nIdentified Task Workflows:")
			for i, workflow := range workflows {
				if i >= 3 {
					fmt.Printf("  ... and %d more workflows\n", len(workflows)-3)
					break
				}
				
				fmt.Printf("  %s: %s\n", workflow.Name, workflow.Description)
				fmt.Printf("    Contains %d command patterns, used approximately %d times\n", 
					len(workflow.Patterns), workflow.Frequency)
			}
		}
	}
}

// showCommandInfo displays detailed information about a specific command
func showCommandInfo(ha *HistoryAnalyzer, command string) {
	stats := ha.GetCommandStats(command)
	
	if stats == nil || stats["found"] == false {
		fmt.Printf("No information found for command: %s\n", command)
		return
	}
	
	fmt.Printf("Command Information: %s\n", command)
	fmt.Println(strings.Repeat("=", 20 + len(command)))
	fmt.Printf("Category: %s\n", stats["category"])
	fmt.Printf("Used %d times\n", stats["frequency"])
	fmt.Printf("Last used: %s\n", stats["last_used"].(time.Time).Format("2006-01-02 15:04:05"))
	
	// Show tags
	fmt.Println("\nTags:")
	for _, tag := range stats["tags"].([]string) {
		fmt.Printf("  - %s\n", tag)
	}
	
	// Show directory distribution
	fmt.Println("\nDirectory Distribution:")
	dirStats := stats["directory_stats"].(map[string]int)
	dirs := make([]string, 0, len(dirStats))
	for dir := range dirStats {
		dirs = append(dirs, dir)
	}
	
	// Sort directories by usage count
	sort.Slice(dirs, func(i, j int) bool {
		return dirStats[dirs[i]] > dirStats[dirs[j]]
	})
	
	for i, dir := range dirs {
		if i >= 5 { // Show only top 5 directories
			break
		}
		fmt.Printf("  %s: %d times\n", dir, dirStats[dir])
	}
	
	// Show time distribution
	fmt.Println("\nTime Distribution:")
	timeStats := stats["time_stats"].(map[int]int)
	hours := make([]int, 0, len(timeStats))
	for hour := range timeStats {
		hours = append(hours, hour)
	}
	
	// Sort hours by usage count
	sort.Slice(hours, func(i, j int) bool {
		return timeStats[hours[i]] > timeStats[hours[j]]
	})
	
	for i, hour := range hours {
		if i >= 5 { // Show only top 5 hours
			break
		}
		fmt.Printf("  %02d:00-%02d:59: %d times\n", hour, hour, timeStats[hour])
	}
	
	// Show related commands if available
	if relatedCmds, ok := stats["related_commands"].([]string); ok && len(relatedCmds) > 0 {
		fmt.Println("\nRelated Commands:")
		for _, cmd := range relatedCmds {
			fmt.Printf("  - %s\n", cmd)
		}
	}
}

// showHistoryHelp displays help for history commands
func showHistoryHelp() {
	fmt.Println("History Analysis Commands")
	fmt.Println("=======================")
	fmt.Println("  :history                - Show recent command history")
	fmt.Println("  :history show [limit]   - Show recent history with optional limit")
	fmt.Println("  :history status         - Show history analysis status")
	fmt.Println("  :history stats          - Show detailed statistics")
	fmt.Println("  :history enable         - Enable history analysis")
	fmt.Println("  :history disable        - Disable history analysis")
	fmt.Println("  :history search <query> - Search command history")
	fmt.Println("  :history suggest        - Show command suggestions")
	fmt.Println("  :history config         - Show configuration")
	fmt.Println("  :history config <setting=value> - Update configuration")
	fmt.Println("  :history mark important|unimportant <command> - Mark a command")
	fmt.Println("  :history patterns       - Show command patterns")
	fmt.Println("  :history analyze        - Force analysis of command patterns")
	fmt.Println("  :history info <command> - Show info about a specific command")
	fmt.Println("  :history help           - Show this help message")
	fmt.Println()
	fmt.Println("Available configuration settings:")
	fmt.Println("  enabled             - Enable/disable history analysis (true/false)")
	fmt.Println("  max_history         - Maximum history entries to keep")
	fmt.Println("  threshold           - Minimum confidence for suggestions (0.0-1.0)")
	fmt.Println("  max_suggestions     - Maximum number of suggestions to show")
	fmt.Println("  auto_suggest        - Whether to automatically show suggestions (true/false)")
	fmt.Println("  track_sequences     - Whether to track command sequences (true/false)")
	fmt.Println("  sequence_max_length - Maximum length of tracked sequences")
	fmt.Println("  nl_search           - Enable natural language search (true/false)")
	fmt.Println("  context_weight      - Weight of directory context in suggestions")
	fmt.Println("  time_weight         - Weight of time patterns in suggestions")
	fmt.Println("  frequency_weight    - Weight of command frequency in suggestions")
	fmt.Println("  recency_weight      - Weight of command recency in suggestions")
	fmt.Println("  enable_categories   - Categorize commands by type (true/false)")
}
