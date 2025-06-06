package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// HandleSpeculativeCommand processes speculative decoding commands
func HandleSpeculativeCommand(args []string) bool {
	// Get the SpeculativeDecoder instance
	sd := GetSpeculativeDecoder()
	if sd == nil {
		fmt.Println("Failed to initialize speculative decoder")
		return true
	}

	// Handle subcommands
	if len(args) == 0 {
		// Show status by default
		showSpeculativeStatus(sd)
		return true
	}

	switch args[0] {
	case "enable":
		// Initialize and enable the speculative decoder
		err := sd.Initialize()
		if err != nil {
			fmt.Printf("Error initializing speculative decoder: %v\n", err)
			return true
		}

		err = sd.Enable()
		if err != nil {
			fmt.Printf("Error enabling speculative decoder: %v\n", err)
		} else {
			fmt.Println("Speculative decoding enabled")
		}
		return true

	case "disable":
		// Disable the speculative decoder
		err := sd.Disable()
		if err != nil {
			fmt.Printf("Error disabling speculative decoder: %v\n", err)
		} else {
			fmt.Println("Speculative decoding disabled")
		}
		return true

	case "status":
		// Show status
		showSpeculativeStatus(sd)
		return true

	case "stats":
		// Show detailed stats
		showSpeculativeStats(sd)
		return true

	case "draft":
		// Generate draft tokens for a prompt
		if len(args) < 2 {
			fmt.Println("Usage: :speculative draft <prompt>")
			return true
		}
		prompt := strings.Join(args[1:], " ")
		testSpeculativeDrafting(sd, prompt)
		return true

	case "reset":
		// Reset statistics
		sd.ResetStats()
		fmt.Println("Statistics reset")
		return true

	case "config":
		// Handle configuration subcommands
		if len(args) > 1 && args[1] == "set" {
			if len(args) < 4 {
				fmt.Println("Usage: :speculative config set <setting> <value>")
				return true
			}
			updateSpeculativeConfig(sd, args[2], args[3])
		} else {
			showSpeculativeConfig(sd)
		}
		return true

	case "help":
		// Show help
		showSpeculativeHelp()
		return true

	default:
		fmt.Printf("Unknown speculative command: %s\n", args[0])
		fmt.Println("Type :speculative help for available commands")
		return true
	}
}

// showSpeculativeStatus displays current status of the speculative decoder
func showSpeculativeStatus(sd *SpeculativeDecoder) {
	fmt.Println("Speculative Decoding Status")
	fmt.Println("==========================")

	stats := sd.GetStats()
	isEnabled := stats["enabled"].(bool)
	isInitialized := stats["initialized"].(bool)

	fmt.Printf("Status: %s\n", getSpeculativeStatusText(isEnabled, isInitialized))

	if isInitialized {
		// Show basic stats
		tokensAccepted := stats["tokens_accepted"].(int)
		tokensRejected := stats["tokens_rejected"].(int)
		totalTokens := tokensAccepted + tokensRejected

		if totalTokens > 0 {
			acceptRate := float64(tokensAccepted) / float64(totalTokens) * 100
			fmt.Printf("Acceptance Rate: %.1f%% (%d/%d tokens)\n", acceptRate, tokensAccepted, totalTokens)
		} else {
			fmt.Println("No tokens processed yet")
		}

		// Show configuration
		draftTokens := stats["draft_tokens"].(int)
		useFallback := stats["use_fallback"].(bool)
		useCache := stats["use_cache"].(bool)

		fmt.Printf("Draft Tokens: %d per prompt\n", draftTokens)
		fmt.Printf("Using N-gram Fallback: %v\n", useFallback)
		fmt.Printf("Cache Enabled: %v\n", useCache)

		if useCache {
			if cacheEntries, ok := stats["cache_entries"].(int); ok {
				fmt.Printf("Cache Entries: %d\n", cacheEntries)
			}
		}
	}
}

// getSpeculativeStatusText returns a descriptive status text
func getSpeculativeStatusText(enabled, initialized bool) string {
	if !initialized {
		return "Not initialized"
	} else if enabled {
		return "Enabled and ready"
	} else {
		return "Disabled (initialized)"
	}
}

// showSpeculativeStats displays detailed statistics about the speculative decoder
func showSpeculativeStats(sd *SpeculativeDecoder) {
	fmt.Println("Speculative Decoding Statistics")
	fmt.Println("==============================")

	stats := sd.GetStats()
	isEnabled := stats["enabled"].(bool)
	isInitialized := stats["initialized"].(bool)

	fmt.Printf("Status: %s\n", getSpeculativeStatusText(isEnabled, isInitialized))

	if isInitialized {
		// Show token statistics
		tokensAccepted := stats["tokens_accepted"].(int)
		tokensRejected := stats["tokens_rejected"].(int)
		totalTokens := tokensAccepted + tokensRejected

		fmt.Println("\nToken Statistics:")
		fmt.Printf("  Accepted Tokens: %d\n", tokensAccepted)
		fmt.Printf("  Rejected Tokens: %d\n", tokensRejected)
		fmt.Printf("  Total Tokens: %d\n", totalTokens)

		if totalTokens > 0 {
			acceptRate := float64(tokensAccepted) / float64(totalTokens) * 100
			fmt.Printf("  Acceptance Rate: %.1f%%\n", acceptRate)
		}

		// Show performance statistics
		fmt.Println("\nPerformance Statistics:")
		if val, ok := stats["tokens_per_second"].(float64); ok {
			fmt.Printf("  Tokens Per Second: %.2f\n", val)
		}
		if val, ok := stats["draft_tokens_per_prompt"].(float64); ok {
			fmt.Printf("  Average Draft Tokens Per Prompt: %.2f\n", val)
		}
		if val, ok := stats["avg_latency_ms"].(float64); ok {
			fmt.Printf("  Average Latency: %.2f ms\n", val)
		}
		if val, ok := stats["total_prompts"].(float64); ok {
			fmt.Printf("  Total Prompts Processed: %.0f\n", val)
		}

		// Show cache statistics
		if useCache, ok := stats["use_cache"].(bool); ok && useCache {
			fmt.Println("\nCache Statistics:")
			if cacheSize, ok := stats["cache_size"].(int); ok {
				fmt.Printf("  Cache Size: %d (max)\n", cacheSize)
			}
			if cacheEntries, ok := stats["cache_entries"].(int); ok {
				fmt.Printf("  Cache Entries: %d (current)\n", cacheEntries)
			}
		}

		// Show configuration
		fmt.Println("\nConfiguration:")
		fmt.Printf("  Draft Tokens: %d per prompt\n", stats["draft_tokens"].(int))
		fmt.Printf("  Accept Threshold: %.2f\n", stats["accept_threshold"].(float64))
		fmt.Printf("  Using N-gram Fallback: %v\n", stats["use_fallback"].(bool))
		if ngramLength, ok := stats["ngram_length"].(int); ok {
			fmt.Printf("  N-gram Length: %d\n", ngramLength)
		}
	}
}

// testSpeculativeDrafting tests speculative drafting for a prompt
func testSpeculativeDrafting(sd *SpeculativeDecoder, prompt string) {
	if !sd.IsEnabled() {
		fmt.Println("Speculative decoding not enabled")
		fmt.Println("Run ':speculative enable' to enable")
		return
	}

	fmt.Printf("Testing speculative drafting for prompt: \"%s\"\n", prompt)
	fmt.Println("------------------------------------------")

	// Generate draft tokens
	fmt.Println("Generating draft tokens...")
	startDraft := time.Now()
	numTokens := sd.GetStats()["draft_tokens"].(int)
	draftTokens, err := sd.GenerateDraftTokens(prompt, numTokens)
	draftDuration := time.Since(startDraft)

	if err != nil {
		fmt.Printf("Error generating draft tokens: %v\n", err)
		return
	}

	// Display draft tokens
	fmt.Printf("Generated %d draft tokens in %.2f ms:\n", len(draftTokens), float64(draftDuration.Microseconds())/1000.0)
	for i, token := range draftTokens {
		fmt.Printf("  Draft[%d]: \"%s\"\n", i, token)
	}

	// Verify draft tokens
	fmt.Println("\nVerifying draft tokens...")
	startVerify := time.Now()
	acceptedTokens, accepted, err := sd.VerifyDraftTokens(prompt, draftTokens, nil)
	verifyDuration := time.Since(startVerify)

	if err != nil {
		fmt.Printf("Error verifying draft tokens: %v\n", err)
		return
	}

	// Display verification results
	fmt.Printf("Verified %d tokens in %.2f ms:\n", len(draftTokens), float64(verifyDuration.Microseconds())/1000.0)
	for i, token := range draftTokens {
		if i < len(accepted) {
			status := "✓ Accepted"
			if !accepted[i] {
				status = "✗ Rejected"
			}
			fmt.Printf("  Token[%d]: \"%s\" - %s\n", i, token, status)
		}
	}

	fmt.Printf("\nAccepted %d/%d tokens (%.1f%%)\n", len(acceptedTokens), len(draftTokens),
		float64(len(acceptedTokens))/float64(len(draftTokens))*100)

	// Show expected speedup
	if len(draftTokens) > 0 {
		// Calculate expected speedup (theoretical)
		expectedSpeedup := 1.0 + float64(len(acceptedTokens))/float64(len(draftTokens))
		fmt.Printf("Expected Speedup: %.2fx\n", expectedSpeedup)

		// Calculate actual speedup from measurements
		// Note: This is just an estimate since we're not actually running the target model
		regularTime := float64(verifyDuration) * float64(len(draftTokens)) / float64(len(acceptedTokens)+1)
		actualSpeedup := regularTime / float64(draftDuration+verifyDuration)
		fmt.Printf("Measured Speedup: %.2fx\n", actualSpeedup)
	}

	// Show total time
	totalDuration := draftDuration + verifyDuration
	fmt.Printf("\nTotal Processing Time: %.2f ms\n", float64(totalDuration.Microseconds())/1000.0)
}

// showSpeculativeConfig displays the speculative decoding configuration
func showSpeculativeConfig(sd *SpeculativeDecoder) {
	fmt.Println("Speculative Decoding Configuration")
	fmt.Println("=================================")

	stats := sd.GetStats()

	fmt.Printf("Enabled: %t\n", stats["enabled"].(bool))
	fmt.Printf("Draft Tokens: %d\n", stats["draft_tokens"].(int))
	fmt.Printf("Accept Threshold: %.2f\n", stats["accept_threshold"].(float64))
	fmt.Printf("Use Cache: %t\n", stats["use_cache"].(bool))
	fmt.Printf("Cache Size: %d\n", stats["cache_size"].(int))
	fmt.Printf("Use N-gram Fallback: %t\n", stats["use_fallback"].(bool))
	fmt.Printf("N-gram Length: %d\n", stats["ngram_length"].(int))

	// Show available settings
	fmt.Println("\nAvailable Settings:")
	fmt.Println("  draft_tokens    - Number of tokens to predict speculatively (1-10)")
	fmt.Println("  accept_threshold - Threshold for accepting speculative tokens (0.0-1.0)")
	fmt.Println("  use_cache       - Whether to use a cache for draft tokens (true/false)")
	fmt.Println("  cache_size      - Maximum number of cache entries (100-100000)")
	fmt.Println("  use_fallback    - Whether to use n-gram fallback (true/false)")
	fmt.Println("  ngram_length    - Length of n-grams for draft model (1-5)")
}

// updateSpeculativeConfig updates a speculative decoding configuration setting
func updateSpeculativeConfig(sd *SpeculativeDecoder, setting, value string) {
	// Get current configuration from stats
	stats := sd.GetStats()

	// Create a config object with current values
	config := SpeculativeDecodingConfig{
		Enabled:          stats["enabled"].(bool),
		DraftTokens:      stats["draft_tokens"].(int),
		AcceptThreshold:  stats["accept_threshold"].(float64),
		UseCache:         stats["use_cache"].(bool),
		CacheSize:        stats["cache_size"].(int),
		UseFallback:      stats["use_fallback"].(bool),
		NGramLength:      stats["ngram_length"].(int),
		DraftModel:       filepath.Join(os.Getenv("HOME"), ".config", "delta", "inference", "models", "draft_model.onnx"),
		LogStats:         true,
		MaxBatchSize:     16,
		UseQuantization:  true,
		QuantizationBits: 8,
	}

	// Update the setting
	switch setting {
	case "draft_tokens":
		tokens, err := strconv.Atoi(value)
		if err != nil || tokens < 1 || tokens > 10 {
			fmt.Println("Draft tokens must be between 1 and 10")
			return
		}
		config.DraftTokens = tokens

	case "accept_threshold":
		threshold, err := strconv.ParseFloat(value, 64)
		if err != nil || threshold < 0 || threshold > 1 {
			fmt.Println("Accept threshold must be between 0.0 and 1.0")
			return
		}
		config.AcceptThreshold = threshold

	case "use_cache":
		config.UseCache = parseBool(value)

	case "cache_size":
		size, err := strconv.Atoi(value)
		if err != nil || size < 100 || size > 100000 {
			fmt.Println("Cache size must be between 100 and 100000")
			return
		}
		config.CacheSize = size

	case "use_fallback":
		config.UseFallback = parseBool(value)

	case "ngram_length":
		length, err := strconv.Atoi(value)
		if err != nil || length < 1 || length > 5 {
			fmt.Println("N-gram length must be between 1 and 5")
			return
		}
		config.NGramLength = length

	default:
		fmt.Printf("Unknown setting: %s\n", setting)
		return
	}

	// Save the updated config
	err := sd.UpdateConfig(config)
	if err != nil {
		fmt.Printf("Error updating configuration: %v\n", err)
		return
	}

	fmt.Printf("Successfully updated %s to %s\n", setting, value)
}

// showSpeculativeHelp displays help for speculative decoding commands
func showSpeculativeHelp() {
	fmt.Println("Speculative Decoding Commands")
	fmt.Println("============================")
	fmt.Println("  :speculative              - Show speculative decoding status")
	fmt.Println("  :speculative enable       - Initialize and enable speculative decoding")
	fmt.Println("  :speculative disable      - Disable speculative decoding")
	fmt.Println("  :speculative status       - Show current status")
	fmt.Println("  :speculative stats        - Show detailed statistics")
	fmt.Println("  :speculative draft <text> - Test speculative drafting for a prompt")
	fmt.Println("  :speculative reset        - Reset statistics")
	fmt.Println("  :speculative config       - Show configuration")
	fmt.Println("  :speculative config set <setting> <value> - Update configuration")
	fmt.Println("  :speculative help         - Show this help message")
	fmt.Println("")
	fmt.Println("Speculative decoding accelerates inference by predicting multiple")
	fmt.Println("tokens at once using a smaller draft model, then verifying them")
	fmt.Println("with the main model. This can speed up generation by 2-3x.")
}
