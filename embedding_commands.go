package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// HandleEmbeddingCommand processes embedding-related commands
func HandleEmbeddingCommand(args []string) bool {
	// Get the EmbeddingManager instance
	em := GetEmbeddingManager()
	if em == nil {
		fmt.Println("Failed to initialize embedding manager")
		return true
	}

	// Handle subcommands
	if len(args) == 0 {
		// Show status by default
		showEmbeddingStatus(em)
		return true
	}

	switch args[0] {
	case "enable":
		// Initialize and enable the embedding system
		err := em.Initialize()
		if err != nil {
			fmt.Printf("Error initializing embedding system: %v\n", err)
			return true
		}

		err = em.Enable()
		if err != nil {
			fmt.Printf("Error enabling embedding system: %v\n", err)
		} else {
			fmt.Println("Embedding system enabled")
		}
		return true

	case "disable":
		// Disable the embedding system
		err := em.Disable()
		if err != nil {
			fmt.Printf("Error disabling embedding system: %v\n", err)
		} else {
			fmt.Println("Embedding system disabled")
		}
		return true

	case "status":
		// Show embedding system status
		showEmbeddingStatus(em)
		return true

	case "stats":
		// Show detailed stats
		showEmbeddingStats(em)
		return true

	case "generate":
		// Generate an embedding for a command
		if len(args) < 2 {
			fmt.Println("Usage: :embedding generate <command>")
			return true
		}
		command := strings.Join(args[1:], " ")
		generateCommandEmbedding(em, command)
		return true

	case "config":
		// Handle configuration subcommands
		if len(args) > 1 && args[1] == "set" {
			if len(args) < 4 {
				fmt.Println("Usage: :embedding config set <setting> <value>")
				return true
			}
			updateEmbeddingConfig(em, args[2], args[3])
		} else {
			showEmbeddingConfig(em)
		}
		return true

	case "help":
		// Show help
		showEmbeddingHelp()
		return true

	case "download":
		// Download the embedding model and vocabulary
		downloadEmbeddingModel(em)
		return true

	default:
		fmt.Printf("Unknown embedding command: %s\n", args[0])
		fmt.Println("Type :embedding help for available commands")
		return true
	}
}

// showEmbeddingStatus displays current status of the embedding system
func showEmbeddingStatus(em *EmbeddingManager) {
	fmt.Println("Embedding System Status")
	fmt.Println("======================")

	stats := em.GetStats()
	isEnabled := stats["enabled"].(bool)
	isInitialized := stats["initialized"].(bool)

	fmt.Printf("Status: %s\n", getEmbeddingStatusText(isEnabled, isInitialized))

	if isInitialized {
		// Show model information
		modelPath := stats["model_path"].(string)
		modelType := stats["model_type"].(string)
		embeddingSize := stats["embedding_size"].(int)

		fmt.Printf("Model Type: %s\n", modelType)
		if stats["use_ollama"].(bool) {
			fmt.Printf("Using Ollama Model: %s\n", stats["ollama_model"].(string))
		} else {
			fmt.Printf("Model Path: %s\n", modelPath)
		}
		fmt.Printf("Embedding Size: %d\n", embeddingSize)

		// Show cache information
		cacheEnabled := stats["cache_enabled"].(bool)
		cacheEntries := stats["cache_entries"].(int)

		fmt.Printf("Cache Enabled: %v\n", cacheEnabled)
		fmt.Printf("Cache Entries: %d\n", cacheEntries)
	}
}

// getEmbeddingStatusText returns a descriptive status text
func getEmbeddingStatusText(enabled, initialized bool) string {
	if !initialized {
		return "Not initialized"
	} else if enabled {
		return "Enabled and ready"
	} else {
		return "Disabled (initialized)"
	}
}

// showEmbeddingStats displays detailed statistics about the embedding system
func showEmbeddingStats(em *EmbeddingManager) {
	fmt.Println("Embedding System Statistics")
	fmt.Println("==========================")

	stats := em.GetStats()

	fmt.Printf("Status: %s\n", getEmbeddingStatusText(
		stats["enabled"].(bool),
		stats["initialized"].(bool),
	))

	// Model information
	fmt.Println("\nModel Information:")
	fmt.Printf("  Model Type: %s\n", stats["model_type"].(string))

	if stats["use_ollama"].(bool) {
		fmt.Printf("  Using Ollama Model: %s\n", stats["ollama_model"].(string))
	} else {
		fmt.Printf("  Model Path: %s\n", stats["model_path"].(string))

		// Check if model exists
		if _, err := os.Stat(stats["model_path"].(string)); os.IsNotExist(err) {
			fmt.Println("  Model Status: Not found")
		} else {
			fmt.Println("  Model Status: Available")
		}
	}

	fmt.Printf("  Embedding Size: %d\n", stats["embedding_size"].(int))

	// Cache information
	fmt.Println("\nCache Information:")
	fmt.Printf("  Cache Enabled: %v\n", stats["cache_enabled"].(bool))
	fmt.Printf("  Cache Size: %d (max)\n", stats["cache_size"].(int))
	fmt.Printf("  Cache Entries: %d (current)\n", stats["cache_entries"].(int))

	// Runtime information
	fmt.Println("\nRuntime Information:")
	fmt.Printf("  Batch Size: %d\n", stats["batch_size"].(int))
	fmt.Printf("  Context Included: %v\n", stats["context_included"].(bool))
	fmt.Printf("  Directory Included: %v\n", stats["directory_included"].(bool))

	// Common commands
	if commonCmds, ok := stats["common_commands"].([]string); ok && len(commonCmds) > 0 {
		fmt.Println("\nTracked Command Types:")
		for _, cmd := range commonCmds {
			fmt.Printf("  - %s\n", cmd)
		}
	}
}

// generateCommandEmbedding generates an embedding for a command
func generateCommandEmbedding(em *EmbeddingManager, command string) {
	if !em.IsEnabled() {
		fmt.Println("Embedding system not enabled")
		fmt.Println("Run ':embedding enable' to enable")
		return
	}

	fmt.Printf("Generating embedding for command: %s\n", command)
	fmt.Println("------------------------------------------")

	// Get current directory for context
	pwd, err := os.Getwd()
	if err != nil {
		pwd = ""
	}

	// Create embedding request
	request := EmbeddingRequest{
		Command:   command,
		Directory: pwd,
		Context:   make(map[string]string),
		Timestamp: time.Now(),
	}

	// Add some basic context
	request.Context["shell"] = os.Getenv("SHELL")
	request.Context["term"] = os.Getenv("TERM")

	// Generate embedding
	start := time.Now()
	response, err := em.GenerateEmbedding(request)
	duration := time.Since(start)

	if err != nil {
		fmt.Printf("Error generating embedding: %v\n", err)
		return
	}

	// Display embedding info
	fmt.Printf("Command: %s\n", response.Command)
	fmt.Printf("Directory: %s\n", response.Directory)
	fmt.Printf("Embedding Size: %d dimensions\n", len(response.Embedding))
	fmt.Printf("Generation Time: %.2f ms\n", float64(duration.Microseconds())/1000.0)

	// Display a preview of the embedding (first 5 dimensions)
	fmt.Println("\nEmbedding Preview (first 5 dimensions):")
	previewLength := 5
	if len(response.Embedding) < previewLength {
		previewLength = len(response.Embedding)
	}

	for i := 0; i < previewLength; i++ {
		fmt.Printf("  [%d]: %.6f\n", i, response.Embedding[i])
	}

	fmt.Println("\nEmbedding Statistics:")
	// Calculate simple statistics
	var min, max, sum, sumSquared float64
	if len(response.Embedding) > 0 {
		min = float64(response.Embedding[0])
		max = float64(response.Embedding[0])

		for _, v := range response.Embedding {
			val := float64(v)
			if val < min {
				min = val
			}
			if val > max {
				max = val
			}
			sum += val
			sumSquared += val * val
		}
	}

	mean := 0.0
	l2Norm := 0.0
	if len(response.Embedding) > 0 {
		mean = sum / float64(len(response.Embedding))
		l2Norm = float64(len(response.Embedding)) * mean * mean
		if l2Norm > 0 {
			l2Norm = math.Sqrt(l2Norm)
		}
	}

	fmt.Printf("  Minimum: %.6f\n", min)
	fmt.Printf("  Maximum: %.6f\n", max)
	fmt.Printf("  Mean: %.6f\n", mean)
	fmt.Printf("  L2 Norm: %.6f\n", l2Norm)

	// Show what it can be used for
	fmt.Println("\nThis embedding can be used for:")
	fmt.Println("  - Semantic search for similar commands")
	fmt.Println("  - Command clustering and classification")
	fmt.Println("  - Intelligent command suggestion")
}

// Import math package at the top of the file

// showEmbeddingConfig displays the embedding configuration
func showEmbeddingConfig(em *EmbeddingManager) {
	fmt.Println("Embedding System Configuration")
	fmt.Println("=============================")

	stats := em.GetStats()

	fmt.Printf("Enabled: %t\n", stats["enabled"].(bool))
	fmt.Printf("Model Type: %s\n", stats["model_type"].(string))
	fmt.Printf("Model Path: %s\n", stats["model_path"].(string))
	fmt.Printf("Use Ollama: %t\n", stats["use_ollama"].(bool))
	fmt.Printf("Ollama Model: %s\n", stats["ollama_model"].(string))
	fmt.Printf("Embedding Size: %d\n", stats["embedding_size"].(int))
	fmt.Printf("Batch Size: %d\n", stats["batch_size"].(int))
	fmt.Printf("Cache Enabled: %t\n", stats["cache_enabled"].(bool))
	fmt.Printf("Cache Size: %d\n", stats["cache_size"].(int))
	fmt.Printf("Context Included: %t\n", stats["context_included"].(bool))
	fmt.Printf("Directory Included: %t\n", stats["directory_included"].(bool))

	fmt.Println("\nAvailable Settings:")
	fmt.Println("  model_type      - Model type (onnx, pytorch, ollama)")
	fmt.Println("  model_url       - URL for downloading the embedding model")
	fmt.Println("  vocab_url       - URL for downloading the vocabulary file")
	fmt.Println("  embedding_size  - Embedding dimension (e.g., 384, 768, 1024)")
	fmt.Println("  use_ollama      - Whether to use Ollama (true, false)")
	fmt.Println("  ollama_model    - Ollama model to use for embeddings")
	fmt.Println("  batch_size      - Batch size for embedding generation")
	fmt.Println("  cache_enabled   - Whether to use embedding cache (true, false)")
	fmt.Println("  cache_size      - Maximum number of cache entries")
	fmt.Println("  context_included - Whether to include context (true, false)")
	fmt.Println("  directory_included - Whether to include directory (true, false)")
}

// updateEmbeddingConfig updates an embedding configuration setting
func updateEmbeddingConfig(em *EmbeddingManager, setting, value string) {
	// Clone the current config
	stats := em.GetStats()
	config := EmbeddingConfig{
		Enabled:           stats["enabled"].(bool),
		ModelPath:         stats["model_path"].(string),
		ModelType:         stats["model_type"].(string),
		CacheEnabled:      stats["cache_enabled"].(bool),
		CacheSize:         stats["cache_size"].(int),
		BatchSize:         stats["batch_size"].(int),
		EmbbedingSize:     stats["embedding_size"].(int),
		UseOllama:         stats["use_ollama"].(bool),
		OllamaURL:         "http://localhost:11434", // Default if not in stats
		OllamaModel:       stats["ollama_model"].(string),
		ContextIncluded:   stats["context_included"].(bool),
		DirectoryIncluded: stats["directory_included"].(bool),
	}

	// Get common commands if available
	if commonCmds, ok := stats["common_commands"].([]string); ok {
		config.CommonCommands = commonCmds
	}

	// Update the setting
	switch setting {
	case "model_type":
		if value != "onnx" && value != "pytorch" && value != "ollama" {
			fmt.Println("Model type must be one of: onnx, pytorch, ollama")
			return
		}
		config.ModelType = value

	case "embedding_size":
		size, err := strconv.Atoi(value)
		if err != nil || size <= 0 {
			fmt.Println("Embedding size must be a positive integer")
			return
		}
		config.EmbbedingSize = size

	case "use_ollama":
		config.UseOllama = parseEmbeddingBool(value)

	case "ollama_model":
		config.OllamaModel = value

	case "batch_size":
		size, err := strconv.Atoi(value)
		if err != nil || size <= 0 {
			fmt.Println("Batch size must be a positive integer")
			return
		}
		config.BatchSize = size

	case "cache_enabled":
		config.CacheEnabled = parseEmbeddingBool(value)

	case "cache_size":
		size, err := strconv.Atoi(value)
		if err != nil || size <= 0 {
			fmt.Println("Cache size must be a positive integer")
			return
		}
		config.CacheSize = size

	case "context_included":
		config.ContextIncluded = parseEmbeddingBool(value)

	case "directory_included":
		config.DirectoryIncluded = parseEmbeddingBool(value)

	case "model_url":
		// Update model URL for downloading
		config.ModelURL = value

	case "vocab_url":
		// Update vocabulary URL for downloading
		config.VocabURL = value

	default:
		fmt.Printf("Unknown setting: %s\n", setting)
		return
	}

	// Save the updated config
	err := em.UpdateConfig(config)
	if err != nil {
		fmt.Printf("Error updating configuration: %v\n", err)
		return
	}

	fmt.Printf("Successfully updated %s to %s\n", setting, value)
}

// showEmbeddingHelp displays help for embedding commands
func showEmbeddingHelp() {
	fmt.Println("Embedding System Commands")
	fmt.Println("========================")
	fmt.Println("  :embedding              - Show embedding status")
	fmt.Println("  :embedding enable       - Initialize and enable embedding system")
	fmt.Println("  :embedding disable      - Disable embedding system")
	fmt.Println("  :embedding status       - Show embedding status")
	fmt.Println("  :embedding stats        - Show detailed statistics")
	fmt.Println("  :embedding generate <cmd> - Generate embedding for a command")
	fmt.Println("  :embedding config       - Show configuration")
	fmt.Println("  :embedding config set <setting> <value> - Update configuration")
	fmt.Println("  :embedding download     - Download the embedding model and vocabulary")
	fmt.Println("  :embedding help         - Show this help message")
	fmt.Println("")
	fmt.Println("Note: The embedding system is used for semantic search and")
	fmt.Println("intelligent command suggestions. It works in conjunction with")
	fmt.Println("the vector database system.")
}

// downloadEmbeddingModel downloads the embedding model and vocabulary files
func downloadEmbeddingModel(em *EmbeddingManager) {
	fmt.Println("Downloading Embedding Model and Vocabulary")
	fmt.Println("=========================================")

	// Check if model URLs are configured
	if em.config.ModelURL == "" {
		fmt.Println("Error: Model URL not configured")
		fmt.Println("Please set the model URL with:")
		fmt.Println("  :embedding config set model_url <url>")
		return
	}

	if em.config.VocabURL == "" {
		fmt.Println("Error: Vocabulary URL not configured")
		fmt.Println("Please set the vocabulary URL with:")
		fmt.Println("  :embedding config set vocab_url <url>")
		return
	}

	// Create models directory if it doesn't exist
	modelsDir := filepath.Dir(em.config.ModelPath)
	err := os.MkdirAll(modelsDir, 0755)
	if err != nil {
		fmt.Printf("Error creating models directory: %v\n", err)
		return
	}

	// Define the vocab path
	vocabPath := filepath.Join(modelsDir, "vocab.txt")

	// Download the model
	fmt.Printf("Downloading model from %s\n", em.config.ModelURL)
	fmt.Printf("To: %s\n", em.config.ModelPath)
	err = DownloadONNXEmbeddingModel(em.config.ModelPath, em.config.ModelURL)
	if err != nil {
		fmt.Printf("Error downloading model: %v\n", err)
		return
	}
	fmt.Println("Model downloaded successfully")

	// Download the vocabulary
	fmt.Printf("\nDownloading vocabulary from %s\n", em.config.VocabURL)
	fmt.Printf("To: %s\n", vocabPath)
	err = DownloadONNXEmbeddingModel(vocabPath, em.config.VocabURL)
	if err != nil {
		fmt.Printf("Error downloading vocabulary: %v\n", err)
		return
	}
	fmt.Println("Vocabulary downloaded successfully")

	// Update configuration to use ONNX if currently using Ollama
	if em.config.UseOllama {
		fmt.Println("\nUpdating configuration to use ONNX model")

		// Clone the current config
		config := em.config
		config.UseOllama = false
		config.ModelType = "onnx"

		// Save the updated config
		err = em.UpdateConfig(config)
		if err != nil {
			fmt.Printf("Error updating configuration: %v\n", err)
			return
		}
		fmt.Println("Configuration updated to use ONNX model")
	}

	fmt.Println("\nDownload complete!")
	fmt.Println("You can now enable the embedding system with:")
	fmt.Println("  :embedding enable")
}

// parseBool parses a string as a boolean value
func parseEmbeddingBool(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	return value == "true" || value == "yes" || value == "1" || value == "on" || value == "enabled"
}
