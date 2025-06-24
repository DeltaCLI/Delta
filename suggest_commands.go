package main

import (
	"fmt"
	"strconv"
	"strings"
)

var globalSuggestManager *SuggestManager

// GetSuggestManager returns the global suggest manager instance
func GetSuggestManager() *SuggestManager {
	if globalSuggestManager == nil {
		globalSuggestManager = NewSuggestManager()
		globalSuggestManager.Initialize()
	}
	return globalSuggestManager
}

// handleSuggestCommand processes the suggest command
func handleSuggestCommand(args []string) bool {
	sm := GetSuggestManager()
	
	// If no args, show help
	if len(args) == 0 {
		showSuggestHelp()
		return true
	}
	
	// Handle subcommands
	switch args[0] {
	case "help":
		showSuggestHelp()
		return true
		
	case "clear":
		sm.ClearCache()
		fmt.Println("Suggestion cache cleared")
		return true
		
	case "last":
		showLastSuggestions()
		return true
		
	case "explain":
		if len(args) < 2 {
			fmt.Println("Please provide a command to explain")
			fmt.Println("Usage: :suggest explain <command>")
			return true
		}
		
		command := strings.Join(args[1:], " ")
		explanation, err := sm.ExplainCommand(command)
		if err != nil {
			fmt.Printf("Error explaining command: %v\n", err)
			return true
		}
		
		fmt.Println("\n" + explanation)
		return true
		
	default:
		// Treat everything as a natural language query
		query := strings.Join(args, " ")
		return handleSuggestionQuery(query)
	}
}

// handleSuggestionQuery processes a natural language query
func handleSuggestionQuery(query string) bool {
	sm := GetSuggestManager()
	
	// Show thinking message
	fmt.Printf("\n🤔 Analyzing: \"%s\"...\n", query)
	
	// Get suggestions
	suggestions, err := sm.GetSuggestions(query, 5)
	if err != nil {
		fmt.Printf("Error generating suggestions: %v\n", err)
		return true
	}
	
	if len(suggestions) == 0 {
		fmt.Println("\nNo suggestions found. Try rephrasing your request or use more specific keywords.")
		fmt.Println("Examples:")
		fmt.Println("  - \"list all files in current directory\"")
		fmt.Println("  - \"find files containing text\"")
		fmt.Println("  - \"install npm packages\"")
		return true
	}
	
	// Display suggestions
	fmt.Printf("\n💡 Found %d suggestions:\n\n", len(suggestions))
	
	for i, suggestion := range suggestions {
		// Show number for selection
		fmt.Printf("%d. ", i+1)
		
		// Show safety indicator
		switch suggestion.Safety {
		case "dangerous":
			fmt.Print("⚠️  ")
		case "caution":
			fmt.Print("⚡ ")
		default:
			fmt.Print("✓  ")
		}
		
		// Show command in color
		fmt.Printf("\033[36m%s\033[0m\n", suggestion.Command)
		
		// Show description
		fmt.Printf("   %s", suggestion.Description)
		
		// Show confidence and category
		fmt.Printf(" \033[2m(%.0f%% confidence, %s)\033[0m\n", 
			suggestion.Confidence*100, suggestion.Category)
		
		// Add extra warning for dangerous commands
		if suggestion.Safety == "dangerous" {
			fmt.Printf("   \033[31m⚠️  This command is potentially dangerous!\033[0m\n")
		}
		
		fmt.Println()
	}
	
	// Show interactive options
	fmt.Println("Options:")
	fmt.Println("  • Type a number (1-5) to execute the command")
	fmt.Println("  • Type 'e<number>' to explain (e.g., 'e1' for first command)")
	fmt.Println("  • Type 'c' to copy a command to clipboard")
	fmt.Println("  • Type 'n' for none/cancel")
	fmt.Println("  • Press Enter to cancel")
	
	// Read user choice
	fmt.Print("\nYour choice: ")
	var choice string
	fmt.Scanln(&choice)
	
	// Handle choice
	choice = strings.TrimSpace(strings.ToLower(choice))
	
	if choice == "" || choice == "n" {
		fmt.Println("Cancelled")
		return true
	}
	
	// Handle explain command
	if strings.HasPrefix(choice, "e") {
		numStr := strings.TrimPrefix(choice, "e")
		if num, err := strconv.Atoi(numStr); err == nil && num >= 1 && num <= len(suggestions) {
			command := suggestions[num-1].Command
			fmt.Printf("\nExplaining: %s\n", command)
			
			explanation, err := sm.ExplainCommand(command)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
			} else {
				fmt.Println("\n" + explanation)
			}
		} else {
			fmt.Println("Invalid selection")
		}
		return true
	}
	
	// Handle copy command
	if choice == "c" {
		fmt.Print("Which command to copy? (1-5): ")
		var copyChoice string
		fmt.Scanln(&copyChoice)
		
		if num, err := strconv.Atoi(copyChoice); err == nil && num >= 1 && num <= len(suggestions) {
			command := suggestions[num-1].Command
			// Note: Actual clipboard functionality would require platform-specific implementation
			fmt.Printf("\nCommand ready to paste: %s\n", command)
			fmt.Println("(Clipboard functionality not implemented in this version)")
		} else {
			fmt.Println("Invalid selection")
		}
		return true
	}
	
	// Handle execution
	if num, err := strconv.Atoi(choice); err == nil && num >= 1 && num <= len(suggestions) {
		suggestion := suggestions[num-1]
		
		// Safety check for dangerous commands
		if suggestion.Safety == "dangerous" {
			fmt.Printf("\n\033[31m⚠️  WARNING: This command is potentially dangerous!\033[0m\n")
			fmt.Printf("Command: %s\n", suggestion.Command)
			fmt.Print("Are you sure you want to execute it? (yes/no): ")
			
			var confirm string
			fmt.Scanln(&confirm)
			
			if strings.ToLower(confirm) != "yes" {
				fmt.Println("Command execution cancelled")
				return true
			}
		}
		
		// Execute the command
		fmt.Printf("\nExecuting: %s\n\n", suggestion.Command)
		
		// Note: In the actual implementation, this would execute through Delta's
		// command execution system. For now, we'll just show what would happen
		fmt.Println("(Command execution would happen here)")
		
		// In real implementation:
		// exitCode, duration := runCommand(suggestion.Command, nil)
		// fmt.Printf("\nCommand completed with exit code %d (took %s)\n", exitCode, duration)
	} else {
		fmt.Println("Invalid selection")
	}
	
	return true
}

// showSuggestHelp displays help for the suggest command
func showSuggestHelp() {
	fmt.Println("Command Suggestion System")
	fmt.Println("========================")
	fmt.Println()
	fmt.Println("Natural language command suggestions powered by pattern matching and AI.")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  :suggest <description>     - Get command suggestions")
	fmt.Println("  :suggest explain <command> - Explain what a command does")
	fmt.Println("  :suggest last              - Show last suggestions")
	fmt.Println("  :suggest clear             - Clear suggestion cache")
	fmt.Println("  :suggest help              - Show this help")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  :suggest list all files")
	fmt.Println("  :suggest find text in files") 
	fmt.Println("  :suggest install dependencies")
	fmt.Println("  :suggest run tests")
	fmt.Println("  :suggest create new branch")
	fmt.Println("  :suggest show running processes")
	fmt.Println()
	fmt.Println("Tips:")
	fmt.Println("  • Be specific about what you want to do")
	fmt.Println("  • Suggestions are context-aware (project type, directory)")
	fmt.Println("  • History-based suggestions improve over time")
	fmt.Println("  • AI suggestions available when Ollama is enabled")
	fmt.Println()
	fmt.Println("Safety Indicators:")
	fmt.Println("  ✓  Safe command")
	fmt.Println("  ⚡ Use with caution")
	fmt.Println("  ⚠️  Potentially dangerous")
}

// showLastSuggestions displays the last generated suggestions
func showLastSuggestions() {
	sm := GetSuggestManager()
	suggestions := sm.GetLastSuggestions()
	
	if len(suggestions) == 0 {
		fmt.Println("No previous suggestions found")
		return
	}
	
	fmt.Printf("\nLast %d suggestions:\n\n", len(suggestions))
	
	for i, suggestion := range suggestions {
		fmt.Printf("%d. %s\n", i+1, suggestion.Command)
		fmt.Printf("   %s\n", suggestion.Description)
	}
}

// Configuration for suggest feature
type SuggestConfig struct {
	Enabled           bool   `json:"enabled"`
	UseAI             bool   `json:"use_ai"`
	UseHistory        bool   `json:"use_history"`
	CacheSize         int    `json:"cache_size"`
	MaxSuggestions    int    `json:"max_suggestions"`
	SafetyPrompts     bool   `json:"safety_prompts"`
	ExplainByDefault  bool   `json:"explain_by_default"`
}