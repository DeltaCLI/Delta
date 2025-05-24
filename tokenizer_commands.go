package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Global tokenizer instance
var globalTokenizer *Tokenizer

// GetTokenizer returns the global tokenizer instance
func GetTokenizer() *Tokenizer {
	if globalTokenizer == nil {
		var err error
		globalTokenizer, err = NewTokenizer()
		if err != nil {
			fmt.Printf("Error initializing tokenizer: %v\n", err)
			return nil
		}
	}
	return globalTokenizer
}

// HandleTokenizerCommand processes tokenizer-related commands
func HandleTokenizerCommand(args []string) bool {
	// Get the tokenizer instance
	tokenizer := GetTokenizer()
	if tokenizer == nil {
		fmt.Println("Failed to initialize tokenizer")
		return true
	}

	// Handle commands
	if len(args) == 0 {
		// Show tokenizer status
		showTokenizerStatus(tokenizer)
		return true
	}

	// Handle subcommands
	if len(args) >= 1 {
		cmd := args[0]

		switch cmd {
		case "enable":
			// Enable tokenizer
			err := enableTokenizer(tokenizer)
			if err != nil {
				fmt.Printf("Error enabling tokenizer: %v\n", err)
			} else {
				fmt.Println("Tokenizer enabled")
			}
			return true

		case "disable":
			// Disable tokenizer
			err := disableTokenizer(tokenizer)
			if err != nil {
				fmt.Printf("Error disabling tokenizer: %v\n", err)
			} else {
				fmt.Println("Tokenizer disabled")
			}
			return true

		case "status":
			// Show status
			showTokenizerStatus(tokenizer)
			return true

		case "stats":
			// Show detailed stats
			showTokenizerStats(tokenizer)
			return true

		case "process":
			// Process command shards into training data
			processCommandShards(tokenizer, args[1:])
			return true

		case "vocab":
			// Show vocabulary information
			showVocabularyInfo(tokenizer)
			return true

		case "test":
			// Test tokenization on a sample command
			if len(args) >= 2 {
				testTokenization(tokenizer, strings.Join(args[1:], " "))
			} else {
				fmt.Println("Usage: :tokenizer test <command>")
			}
			return true

		case "help":
			// Show help
			showTokenizerHelp()
			return true

		default:
			fmt.Printf("Unknown tokenizer command: %s\n", cmd)
			fmt.Println("Type :tokenizer help for a list of available commands")
			return true
		}
	}

	return true
}

// showTokenizerStatus displays the current status of the tokenizer
func showTokenizerStatus(tokenizer *Tokenizer) {
	fmt.Println("Tokenizer Status")
	fmt.Println("===============")
	
	stats := tokenizer.GetTokenizerStats()
	
	// Show enabled/disabled status
	if tokenizer.Config.Enabled {
		fmt.Println("Status: Enabled")
	} else {
		fmt.Println("Status: Disabled")
	}
	
	fmt.Printf("Vocabulary Size: %d / %d\n", stats["vocabulary_size"], tokenizer.Config.VocabSize)
	fmt.Printf("Known Commands: %d\n", stats["command_count"])
	fmt.Printf("Special Tokens: %d\n", stats["special_tokens"])
	fmt.Printf("Max Token Length: %d\n", stats["max_token_length"])
	fmt.Printf("Config Path: %s\n", stats["config_path"])
	fmt.Printf("Data Path: %s\n", stats["data_path"])
}

// showTokenizerStats displays detailed statistics about the tokenizer
func showTokenizerStats(tokenizer *Tokenizer) {
	fmt.Println("Tokenizer Statistics")
	fmt.Println("===================")
	
	stats := tokenizer.GetTokenizerStats()
	
	fmt.Printf("Vocabulary Size: %d / %d (%.1f%% used)\n", 
		stats["vocabulary_size"], tokenizer.Config.VocabSize,
		float64(stats["vocabulary_size"].(int))/float64(tokenizer.Config.VocabSize)*100)
	
	fmt.Printf("Known Commands: %d\n", stats["command_count"])
	fmt.Printf("Special Tokens: %d\n", stats["special_tokens"])
	
	// Count processed files
	dataPath, ok := stats["data_path"].(string)
	if !ok {
		dataPath = tokenizer.dataPath
	}

	files, err := os.ReadDir(dataPath)
	if err == nil {
		tokenizedFileCount := 0
		totalSize := int64(0)

		for _, file := range files {
			if strings.HasPrefix(file.Name(), "tokenized_") && strings.HasSuffix(file.Name(), ".bin") {
				tokenizedFileCount++

				// Get file size
				fileInfo, err := file.Info()
				if err == nil {
					totalSize += fileInfo.Size()
				}
			}
		}

		fmt.Printf("Processed Files: %d\n", tokenizedFileCount)
		fmt.Printf("Total Data Size: %.2f MB\n", float64(totalSize)/(1024*1024))
	}
	
	// Try to count tokens in a few sample files
	var tokenCount int
	sampleCount := 0
	
	for _, file := range files {
		if !strings.HasPrefix(file.Name(), "tokenized_") || !strings.HasSuffix(file.Name(), ".bin") {
			continue
		}

		// Read file header to get entry count
		filePath := filepath.Join(dataPath, file.Name())
		fileData, err := os.ReadFile(filePath)
		if err != nil || len(fileData) < 4 {
			continue
		}
		
		// First 4 bytes are the entry count
		entryCount := int(fileData[0]) | int(fileData[1])<<8 | int(fileData[2])<<16 | int(fileData[3])<<24
		tokenCount += entryCount
		
		sampleCount++
		if sampleCount >= 5 {
			break
		}
	}
	
	if sampleCount > 0 {
		fmt.Printf("Estimated Command Entries: ~%d\n", tokenCount * len(files) / sampleCount)
	}
}

// processCommandShards processes command shards into training data
func processCommandShards(tokenizer *Tokenizer, args []string) {
	// Default to processing all available shards if no specific shards are provided
	if len(args) == 0 {
		// Get the memory manager to find available shards
		mm := GetMemoryManager()
		if mm == nil {
			fmt.Println("Memory manager not initialized")
			return
		}
		
		// List available shards
		entries, err := os.ReadDir(mm.config.StoragePath)
		if err != nil {
			fmt.Printf("Error reading storage directory: %v\n", err)
			return
		}
		
		// Filter for command shards
		var shardPaths []string
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasPrefix(entry.Name(), "commands_") {
				continue
			}
			
			shardPaths = append(shardPaths, filepath.Join(mm.config.StoragePath, entry.Name()))
		}
		
		if len(shardPaths) == 0 {
			fmt.Println("No command shards found to process")
			return
		}
		
		fmt.Printf("Processing %d command shards...\n", len(shardPaths))
		startTime := time.Now()
		
		count, err := tokenizer.ExtractCommandsFromShards(shardPaths)
		if err != nil {
			fmt.Printf("Error processing shards: %v\n", err)
			return
		}
		
		duration := time.Since(startTime)
		fmt.Printf("Processed %d commands in %s\n", count, duration)
		fmt.Printf("Vocabulary size is now %d tokens\n", tokenizer.GetVocabularySize())
		
	} else {
		// Process specific shards
		var shardPaths []string
		for _, shardName := range args {
			// Convert date format to filename if needed
			if len(shardName) == 10 && strings.Count(shardName, "-") == 2 {
				shardName = "commands_" + shardName + ".bin"
			}
			
			// Get memory manager for storage path
			mm := GetMemoryManager()
			if mm == nil {
				fmt.Println("Memory manager not initialized")
				return
			}
			
			shardPath := filepath.Join(mm.config.StoragePath, shardName)
			shardPaths = append(shardPaths, shardPath)
		}
		
		fmt.Printf("Processing %d command shards...\n", len(shardPaths))
		startTime := time.Now()
		
		count, err := tokenizer.ExtractCommandsFromShards(shardPaths)
		if err != nil {
			fmt.Printf("Error processing shards: %v\n", err)
			return
		}
		
		duration := time.Since(startTime)
		fmt.Printf("Processed %d commands in %s\n", count, duration)
		fmt.Printf("Vocabulary size is now %d tokens\n", tokenizer.GetVocabularySize())
	}
}

// showVocabularyInfo displays information about the tokenizer vocabulary
func showVocabularyInfo(tokenizer *Tokenizer) {
	fmt.Println("Tokenizer Vocabulary")
	fmt.Println("===================")
	
	// Special tokens
	fmt.Println("\nSpecial Tokens:")
	for _, token := range tokenizer.Config.SpecialTokens {
		fmt.Printf("  %s\n", token)
	}
	
	// Common commands
	fmt.Println("\nCommon Commands:")
	for i, cmd := range tokenizer.Config.CommonCommands {
		fmt.Printf("  %s", cmd)
		
		// Format in columns
		if (i+1) % 5 == 0 {
			fmt.Println()
		} else {
			fmt.Printf("\t")
		}
	}
	fmt.Println()
	
	// Vocabulary size
	fmt.Printf("\nTotal Vocabulary Size: %d / %d tokens\n", 
		tokenizer.GetVocabularySize(), tokenizer.Config.VocabSize)
	
	// Show vocabulary compression rate
	vocabRate := float64(tokenizer.GetVocabularySize()) / float64(tokenizer.Config.VocabSize) * 100
	fmt.Printf("Vocabulary Utilization: %.1f%%\n", vocabRate)
}

// testTokenization tests tokenization on a sample command
func testTokenization(tokenizer *Tokenizer, command string) {
	fmt.Println("Tokenizing Command: " + command)
	fmt.Println("=================================================")
	
	// Tokenize the command
	tokens, err := tokenizer.TokenizeCommand(command, "~", 0)
	if err != nil {
		fmt.Printf("Error tokenizing command: %v\n", err)
		return
	}
	
	// Print token information
	fmt.Println("\nTokens:")
	fmt.Printf("%-20s %-12s %-8s\n", "TOKEN", "TYPE", "POSITION")
	fmt.Println("--------------------------------------------------")
	for _, token := range tokens.Tokens {
		fmt.Printf("%-20s %-12s %-8d\n", 
			truncateString(token.Text, 20),
			token.Type,
			token.Position)
	}
	
	// Encode tokens
	ids, err := tokenizer.EncodeTokens(tokens)
	if err != nil {
		fmt.Printf("\nError encoding tokens: %v\n", err)
		return
	}
	
	fmt.Printf("\nEncoded Token IDs (%d): ", len(ids))
	if len(ids) > 20 {
		// Show first 10 and last 10
		for i := 0; i < 10; i++ {
			fmt.Printf("%d ", ids[i])
		}
		fmt.Printf("... ")
		for i := len(ids) - 10; i < len(ids); i++ {
			fmt.Printf("%d ", ids[i])
		}
	} else {
		for _, id := range ids {
			fmt.Printf("%d ", id)
		}
	}
	fmt.Println()
	
	// Decode back to string
	decoded, err := tokenizer.DecodeTokens(ids)
	if err != nil {
		fmt.Printf("\nError decoding tokens: %v\n", err)
		return
	}
	
	fmt.Printf("\nDecoded Command: %s\n", decoded)
	fmt.Printf("Original Command: %s\n", command)
	
	// Compare
	if decoded == command {
		fmt.Println("\nDecoding is perfect!")
	} else {
		fmt.Println("\nDecoding has differences.")
	}
}

// showTokenizerHelp displays help for tokenizer commands
func showTokenizerHelp() {
	fmt.Println("Tokenizer Commands")
	fmt.Println("=================")
	fmt.Println("  :tokenizer               - Show current tokenizer status")
	fmt.Println("  :tokenizer enable        - Enable tokenizer")
	fmt.Println("  :tokenizer disable       - Disable tokenizer")
	fmt.Println("  :tokenizer status        - Show tokenizer status")
	fmt.Println("  :tokenizer stats         - Show detailed tokenizer statistics")
	fmt.Println("  :tokenizer process       - Process all command shards into training data")
	fmt.Println("  :tokenizer process <date> - Process a specific shard (e.g., YYYY-MM-DD)")
	fmt.Println("  :tokenizer vocab         - Show vocabulary information")
	fmt.Println("  :tokenizer test <cmd>    - Test tokenization on a sample command")
	fmt.Println("  :tokenizer help          - Show this help message")
}

// Helper function to truncate a string to a maximum length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

// enableTokenizer enables the tokenizer
func enableTokenizer(tokenizer *Tokenizer) error {
	tokenizer.mutex.Lock()
	tokenizer.Config.Enabled = true
	tokenizer.mutex.Unlock()
	
	// Save local config
	if err := tokenizer.saveVocabulary(); err != nil {
		return err
	}
	
	// Update ConfigManager
	cm := GetConfigManager()
	if cm != nil {
		cm.UpdateTokenConfig(&tokenizer.Config)
	}
	
	return nil
}

// disableTokenizer disables the tokenizer
func disableTokenizer(tokenizer *Tokenizer) error {
	tokenizer.mutex.Lock()
	tokenizer.Config.Enabled = false
	tokenizer.mutex.Unlock()
	
	// Save local config
	if err := tokenizer.saveVocabulary(); err != nil {
		return err
	}
	
	// Update ConfigManager
	cm := GetConfigManager()
	if cm != nil {
		cm.UpdateTokenConfig(&tokenizer.Config)
	}
	
	return nil
}