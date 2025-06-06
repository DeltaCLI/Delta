package main

import (
	"fmt"
	"strconv"
	"strings"
)

// HandleSpellCheckCommand processes spellcheck-related commands
func HandleSpellCheckCommand(args []string) bool {
	// Get the SpellChecker instance
	sc := GetSpellChecker()
	if sc == nil {
		fmt.Println("Failed to initialize spell checker")
		return true
	}

	// Initialize if not already done
	if !sc.isInitialized {
		err := sc.Initialize()
		if err != nil {
			fmt.Printf("Error initializing spell checker: %v\n", err)
			return true
		}
	}

	// Handle commands
	if len(args) == 0 {
		// Show spellcheck status
		showSpellCheckStatus(sc)
		return true
	}

	// Handle special commands
	if len(args) >= 1 {
		cmd := args[0]

		switch cmd {
		case "status":
			// Show status
			showSpellCheckStatus(sc)
			return true

		case "enable":
			// Enable spell checking
			sc.EnableSpellChecker()
			fmt.Println("Spell checking enabled")
			return true

		case "disable":
			// Disable spell checking
			sc.DisableSpellChecker()
			fmt.Println("Spell checking disabled")
			return true

		case "config":
			// Show or edit configuration
			if len(args) < 2 {
				showSpellCheckConfig(sc)
			} else {
				updateSpellCheckConfig(sc, args[1:])
			}
			return true

		case "add":
			// Add a word to the custom dictionary
			if len(args) < 2 {
				fmt.Println("Usage: :spellcheck add <word>")
				return true
			}
			addToCustomDictionary(sc, args[1])
			return true

		case "remove":
			// Remove a word from the custom dictionary
			if len(args) < 2 {
				fmt.Println("Usage: :spellcheck remove <word>")
				return true
			}
			removeFromCustomDictionary(sc, args[1])
			return true

		case "test":
			// Test spell checking on a command
			if len(args) < 2 {
				fmt.Println("Usage: :spellcheck test <command>")
				return true
			}
			testSpellCheck(sc, args[1])
			return true

		case "help":
			// Show help
			showSpellCheckHelp()
			return true

		default:
			fmt.Println("Unknown spell check command:", cmd)
			fmt.Println("Type :spellcheck help for available commands")
			return true
		}
	}

	return true
}

// showSpellCheckStatus displays the current status of the spell checker
func showSpellCheckStatus(sc *SpellChecker) {
	fmt.Println("Spell Checker Status")
	fmt.Println("====================")
	fmt.Printf("Enabled: %v\n", sc.IsEnabled())
	fmt.Printf("Dictionary size: %d commands\n", len(sc.internalCommands))
	fmt.Printf("Custom entries: %d words\n", len(sc.config.CustomDictionary))
	fmt.Printf("Auto-correct: %v (threshold: %.2f)\n", sc.config.AutoCorrect, sc.config.AutoCorrectThreshold)
	fmt.Printf("Suggestion threshold: %.2f\n", sc.config.SuggestionThreshold)
}

// showSpellCheckConfig displays the spell checker configuration
func showSpellCheckConfig(sc *SpellChecker) {
	fmt.Println("Spell Checker Configuration")
	fmt.Println("==========================")
	fmt.Printf("Enabled: %v\n", sc.config.Enabled)
	fmt.Printf("Suggestion threshold: %.2f\n", sc.config.SuggestionThreshold)
	fmt.Printf("Max suggestions: %d\n", sc.config.MaxSuggestions)
	fmt.Printf("Auto-correct: %v\n", sc.config.AutoCorrect)
	fmt.Printf("Auto-correct threshold: %.2f\n", sc.config.AutoCorrectThreshold)
	fmt.Printf("Case sensitive: %v\n", sc.config.CaseSensitive)

	// Show custom dictionary
	if len(sc.config.CustomDictionary) > 0 {
		fmt.Println("\nCustom Dictionary:")
		for i, word := range sc.config.CustomDictionary {
			fmt.Printf("  %d. %s\n", i+1, word)
		}
	} else {
		fmt.Println("\nCustom Dictionary: (empty)")
	}
}

// updateSpellCheckConfig updates the spell checker configuration
func updateSpellCheckConfig(sc *SpellChecker, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: :spellcheck config <setting=value>")
		fmt.Println("Available settings: enabled, threshold, max_suggestions, auto_correct, auto_threshold, case_sensitive")
		return
	}

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
				sc.config.Enabled = true
			} else if value == "false" || value == "0" || value == "no" {
				sc.config.Enabled = false
			} else {
				fmt.Printf("Invalid value for %s: %s (should be true or false)\n", setting, value)
				continue
			}

		case "threshold":
			threshold, err := strconv.ParseFloat(value, 64)
			if err != nil || threshold < 0 || threshold > 1 {
				fmt.Printf("Invalid value for %s: %s (should be between 0.0 and 1.0)\n", setting, value)
				continue
			}
			sc.config.SuggestionThreshold = threshold

		case "max_suggestions":
			max, err := strconv.Atoi(value)
			if err != nil || max < 1 {
				fmt.Printf("Invalid value for %s: %s (should be a positive integer)\n", setting, value)
				continue
			}
			sc.config.MaxSuggestions = max

		case "auto_correct":
			if value == "true" || value == "1" || value == "yes" {
				sc.config.AutoCorrect = true
			} else if value == "false" || value == "0" || value == "no" {
				sc.config.AutoCorrect = false
			} else {
				fmt.Printf("Invalid value for %s: %s (should be true or false)\n", setting, value)
				continue
			}

		case "auto_threshold":
			threshold, err := strconv.ParseFloat(value, 64)
			if err != nil || threshold < 0 || threshold > 1 {
				fmt.Printf("Invalid value for %s: %s (should be between 0.0 and 1.0)\n", setting, value)
				continue
			}
			sc.config.AutoCorrectThreshold = threshold

		case "case_sensitive":
			if value == "true" || value == "1" || value == "yes" {
				sc.config.CaseSensitive = true
			} else if value == "false" || value == "0" || value == "no" {
				sc.config.CaseSensitive = false
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
	err := sc.UpdateConfig(sc.config)
	if err != nil {
		fmt.Printf("Error saving configuration: %v\n", err)
	}
}

// addToCustomDictionary adds a word to the custom dictionary
func addToCustomDictionary(sc *SpellChecker, word string) {
	// Check if word already exists in the dictionary
	for _, existingWord := range sc.config.CustomDictionary {
		if existingWord == word {
			fmt.Printf("'%s' already exists in the custom dictionary\n", word)
			return
		}
	}

	// Add the word to the dictionary
	sc.config.CustomDictionary = append(sc.config.CustomDictionary, word)
	err := sc.UpdateConfig(sc.config)
	if err != nil {
		fmt.Printf("Error saving configuration: %v\n", err)
		return
	}

	// Reinitialize commands to include the new word
	sc.initializeCommands()
	fmt.Printf("Added '%s' to the custom dictionary\n", word)
}

// removeFromCustomDictionary removes a word from the custom dictionary
func removeFromCustomDictionary(sc *SpellChecker, word string) {
	// Check if word exists in the dictionary
	found := false
	var newDictionary []string
	for _, existingWord := range sc.config.CustomDictionary {
		if existingWord == word {
			found = true
		} else {
			newDictionary = append(newDictionary, existingWord)
		}
	}

	if !found {
		fmt.Printf("'%s' does not exist in the custom dictionary\n", word)
		return
	}

	// Update the dictionary
	sc.config.CustomDictionary = newDictionary
	err := sc.UpdateConfig(sc.config)
	if err != nil {
		fmt.Printf("Error saving configuration: %v\n", err)
		return
	}

	// Reinitialize commands to exclude the removed word
	sc.initializeCommands()
	fmt.Printf("Removed '%s' from the custom dictionary\n", word)
}

// testSpellCheck tests the spell checker on a command
func testSpellCheck(sc *SpellChecker, command string) {
	// Ensure the command has a colon prefix for internal commands
	testCmd := command
	if !strings.HasPrefix(testCmd, ":") {
		testCmd = ":" + testCmd
	}

	// Check for suggestions
	suggestions := sc.CheckCommand(testCmd)
	if len(suggestions) == 0 {
		fmt.Printf("No suggestions for '%s'\n", command)
		return
	}

	// Display suggestions
	fmt.Printf("Suggestions for '%s':\n", command)
	for i, suggestion := range suggestions {
		fmt.Printf("  %d. '%s' (similarity: %.2f)\n", i+1, suggestion.Command, suggestion.Score)
	}

	// Check for auto-correction
	if sc.ShouldAutoCorrect(suggestions) {
		fmt.Printf("Auto-correct would choose: '%s'\n", suggestions[0].Command)
	}
}

// showSpellCheckHelp displays help for spell checker commands
func showSpellCheckHelp() {
	fmt.Println("Spell Checker Commands")
	fmt.Println("=====================")
	fmt.Println("  :spellcheck              - Show spell checker status")
	fmt.Println("  :spellcheck status       - Show spell checker status")
	fmt.Println("  :spellcheck enable       - Enable spell checking")
	fmt.Println("  :spellcheck disable      - Disable spell checking")
	fmt.Println("  :spellcheck config       - Show configuration")
	fmt.Println("  :spellcheck config <setting=value> - Update configuration")
	fmt.Println("  :spellcheck add <word>   - Add word to custom dictionary")
	fmt.Println("  :spellcheck remove <word> - Remove word from custom dictionary")
	fmt.Println("  :spellcheck test <command> - Test spell checking on a command")
	fmt.Println("  :spellcheck help         - Show this help message")
	fmt.Println()
	fmt.Println("Available configuration settings:")
	fmt.Println("  enabled         - Enable/disable spell checking (true/false)")
	fmt.Println("  threshold       - Suggestion threshold (0.0-1.0)")
	fmt.Println("  max_suggestions - Maximum number of suggestions to show")
	fmt.Println("  auto_correct    - Enable/disable auto-correction (true/false)")
	fmt.Println("  auto_threshold  - Auto-correction threshold (0.0-1.0)")
	fmt.Println("  case_sensitive  - Case sensitive matching (true/false)")
}
