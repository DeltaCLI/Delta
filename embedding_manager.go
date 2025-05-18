package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// EmbeddingConfig contains configuration for embedding generation
type EmbeddingConfig struct {
	Enabled           bool     `json:"enabled"`
	ModelPath         string   `json:"model_path"`
	ModelType         string   `json:"model_type"` // "onnx" or "pytorch" or "ollama"
	ModelURL          string   `json:"model_url"`  // URL for downloading the model
	VocabURL          string   `json:"vocab_url"`  // URL for downloading the vocabulary
	CacheEnabled      bool     `json:"cache_enabled"`
	CacheSize         int      `json:"cache_size"`
	BatchSize         int      `json:"batch_size"`
	EmbbedingSize     int      `json:"embedding_size"`
	Dimensions        int      `json:"dimensions"` // Embedding dimension size
	UseOllama         bool     `json:"use_ollama"`
	OllamaURL         string   `json:"ollama_url"`
	OllamaModel       string   `json:"ollama_model"`
	CommonCommands    []string `json:"common_commands"`
	ContextIncluded   bool     `json:"context_included"`
	DirectoryIncluded bool     `json:"directory_included"`
	NumThreads        int      `json:"num_threads"`
	UseGPU            bool     `json:"use_gpu"`
	MaxTokenLength    int      `json:"max_token_length"`
}

// EmbeddingRequest represents a request to generate an embedding
type EmbeddingRequest struct {
	Command   string            `json:"command"`
	Directory string            `json:"directory"`
	Context   map[string]string `json:"context"`
	Timestamp time.Time         `json:"timestamp"`
}

// EmbeddingResponse represents the response from an embedding request
type EmbeddingResponse struct {
	ID         string    `json:"id"`
	Command    string    `json:"command"`
	Directory  string    `json:"directory"`
	Embedding  []float32 `json:"embedding"`
	IsFromCmd  bool      `json:"is_from_cmd"`
	IsFromCtx  bool      `json:"is_from_ctx"`
	Normalized bool      `json:"normalized"`
	Score      float32   `json:"score"`
	Timestamp  time.Time `json:"timestamp"`
}

// EmbeddingCache is a simple cache for embeddings
type EmbeddingCache struct {
	Embeddings map[string]EmbeddingResponse
	MaxSize    int
	Mutex      sync.RWMutex
}

// EmbeddingManager handles the generation of embeddings
type EmbeddingManager struct {
	config        EmbeddingConfig
	configPath    string
	cache         *EmbeddingCache
	onnxRuntime   *ONNXRuntime
	mutex         sync.RWMutex
	isInitialized bool
}

// NewEmbeddingManager creates a new embedding manager
func NewEmbeddingManager() (*EmbeddingManager, error) {
	// Set up config directory in user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = os.Getenv("HOME")
	}

	// Use ~/.config/delta/embeddings directory
	configDir := filepath.Join(homeDir, ".config", "delta", "embeddings")
	err = os.MkdirAll(configDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create embeddings directory: %v", err)
	}

	configPath := filepath.Join(configDir, "embedding_config.json")
	modelPath := filepath.Join(configDir, "models", "embedding_model.onnx")

	// Create a cache
	cache := &EmbeddingCache{
		Embeddings: make(map[string]EmbeddingResponse),
		MaxSize:    1000,
		Mutex:      sync.RWMutex{},
	}

	// Default configuration
	defaultConfig := EmbeddingConfig{
		Enabled:           false,
		ModelPath:         modelPath,
		ModelType:         "onnx",
		ModelURL:          "https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2/resolve/main/model.onnx",
		VocabURL:          "https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2/resolve/main/vocab.txt",
		CacheEnabled:      true,
		CacheSize:         1000,
		BatchSize:         4,
		EmbbedingSize:     384,
		UseOllama:         true,
		OllamaURL:         "http://localhost:11434",
		OllamaModel:       "phi4:latest",
		CommonCommands:    []string{"cd", "ls", "git", "docker", "npm", "python", "make"},
		ContextIncluded:   true,
		DirectoryIncluded: true,
		NumThreads:        4,
		UseGPU:            false,
		MaxTokenLength:    128,
	}

	// Start with default configuration
	// Note: We don't load from ConfigManager here to avoid circular dependencies
	// The ConfigManager will update this manager's config after initialization
	config := defaultConfig

	// Create the manager
	manager := &EmbeddingManager{
		config:        config,
		configPath:    configPath,
		cache:         cache,
		mutex:         sync.RWMutex{},
		isInitialized: false,
	}

	// No need to load configuration here anymore
	// The ConfigManager will handle this after initialization

	return manager, nil
}

// Initialize initializes the embedding manager
func (em *EmbeddingManager) Initialize() error {
	em.mutex.Lock()
	defer em.mutex.Unlock()

	// Setup cache
	em.cache.MaxSize = em.config.CacheSize

	// Create models directory if it doesn't exist
	modelsDir := filepath.Dir(em.config.ModelPath)
	err := os.MkdirAll(modelsDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create models directory: %v", err)
	}

	// Check if we should use ONNX
	if em.config.ModelType == "onnx" && !em.config.UseOllama {
		// Define the vocab path
		vocabPath := filepath.Join(modelsDir, "vocab.txt")

		// Check if ONNX model exists, if not, download it
		if _, err := os.Stat(em.config.ModelPath); os.IsNotExist(err) {
			fmt.Printf("Embedding model not found at %s. Attempting to download...\n", em.config.ModelPath)
			if em.config.ModelURL == "" {
				fmt.Println("No model URL specified. Falling back to Ollama.")
				em.config.UseOllama = true
			} else {
				// Download the model
				err = DownloadONNXEmbeddingModel(em.config.ModelPath, em.config.ModelURL)
				if err != nil {
					fmt.Printf("Failed to download embedding model: %v\n", err)
					fmt.Println("Falling back to Ollama for embeddings.")
					em.config.UseOllama = true
				} else {
					fmt.Printf("Successfully downloaded embedding model to %s\n", em.config.ModelPath)
				}
			}
		}

		// Check if vocab file exists, if not, download it
		if _, err := os.Stat(vocabPath); os.IsNotExist(err) && !em.config.UseOllama {
			fmt.Printf("Vocab file not found at %s. Attempting to download...\n", vocabPath)
			if em.config.VocabURL == "" {
				fmt.Println("No vocab URL specified. Falling back to Ollama.")
				em.config.UseOllama = true
			} else {
				// Download the vocab file using a simple HTTP GET request
				err = DownloadONNXEmbeddingModel(vocabPath, em.config.VocabURL)
				if err != nil {
					fmt.Printf("Failed to download vocab file: %v\n", err)
					fmt.Println("Falling back to Ollama for embeddings.")
					em.config.UseOllama = true
				} else {
					fmt.Printf("Successfully downloaded vocab file to %s\n", vocabPath)
				}
			}
		}

		// Initialize ONNX Runtime if model and vocab files exist
		if !em.config.UseOllama {
			// Create ONNX runtime configuration
			onnxConfig := ONNXRuntimeConfig{
				ModelPath:   em.config.ModelPath,
				VocabPath:   vocabPath,
				Dimension:   em.config.EmbbedingSize,
				MaxLength:   em.config.MaxTokenLength,
				BatchSize:   em.config.BatchSize,
				NumThreads:  em.config.NumThreads,
				UseGPU:      em.config.UseGPU,
				UseFP16:     false, // Default to FP32 for better accuracy
				GPUDeviceID: 0,     // Default GPU device
			}

			// Initialize the ONNX runtime
			onnxRuntime, err := NewONNXRuntime(onnxConfig)
			if err != nil {
				fmt.Printf("Failed to create ONNX runtime: %v\n", err)
				fmt.Println("Falling back to Ollama for embeddings.")
				em.config.UseOllama = true
			} else {
				// Initialize the ONNX runtime
				err = onnxRuntime.Initialize()
				if err != nil {
					fmt.Printf("Failed to initialize ONNX runtime: %v\n", err)
					fmt.Println("Falling back to Ollama for embeddings.")
					em.config.UseOllama = true
				} else {
					// Store the ONNX runtime
					em.onnxRuntime = onnxRuntime
					fmt.Println("Successfully initialized ONNX runtime for embeddings.")
				}
			}
		}
	}

	em.isInitialized = true
	return nil
}

// loadConfig loads the embedding configuration from disk
func (em *EmbeddingManager) loadConfig() error {
	// Check if config file exists
	_, err := os.Stat(em.configPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("config file does not exist")
	}

	// Read the config file
	data, err := os.ReadFile(em.configPath)
	if err != nil {
		return err
	}

	// Unmarshal the JSON data
	return json.Unmarshal(data, &em.config)
}

// saveConfig saves the embedding configuration to disk
func (em *EmbeddingManager) saveConfig() error {
	// Marshal the config to JSON with indentation for readability
	data, err := json.MarshalIndent(em.config, "", "  ")
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(em.configPath)
	if err = os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(em.configPath, data, 0644)
}

// IsEnabled returns whether embeddings are enabled
func (em *EmbeddingManager) IsEnabled() bool {
	em.mutex.RLock()
	defer em.mutex.RUnlock()
	return em.config.Enabled && em.isInitialized
}

// Enable enables embeddings
func (em *EmbeddingManager) Enable() error {
	em.mutex.Lock()
	defer em.mutex.Unlock()

	if !em.isInitialized {
		return fmt.Errorf("embedding manager not initialized")
	}

	em.config.Enabled = true
	return em.saveConfig()
}

// Disable disables embeddings
func (em *EmbeddingManager) Disable() error {
	em.mutex.Lock()
	defer em.mutex.Unlock()

	em.config.Enabled = false
	return em.saveConfig()
}

// GenerateEmbedding generates an embedding for a command
func (em *EmbeddingManager) GenerateEmbedding(request EmbeddingRequest) (*EmbeddingResponse, error) {
	if !em.IsEnabled() {
		return nil, fmt.Errorf("embedding generation not enabled")
	}

	// Check cache first
	cacheKey := em.generateCacheKey(request)
	if em.config.CacheEnabled {
		if cachedEmbedding := em.getFromCache(cacheKey); cachedEmbedding != nil {
			return cachedEmbedding, nil
		}
	}

	// Prepare the input for embedding
	var embedding []float32
	var err error

	if em.config.UseOllama {
		// Generate embedding using Ollama
		embedding, err = em.generateEmbeddingWithOllama(request)
	} else if em.config.ModelType == "onnx" {
		// Generate embedding using ONNX model
		embedding, err = em.generateEmbeddingWithONNX(request)
	} else {
		// Fallback to a placeholder embedding for demonstration
		embedding, err = em.generatePlaceholderEmbedding(request)
	}

	if err != nil {
		return nil, err
	}

	// Create the response
	response := &EmbeddingResponse{
		ID:         cacheKey,
		Command:    request.Command,
		Directory:  request.Directory,
		Embedding:  embedding,
		IsFromCmd:  true,
		IsFromCtx:  em.config.ContextIncluded,
		Normalized: true,
		Score:      1.0,
		Timestamp:  time.Now(),
	}

	// Add to cache
	if em.config.CacheEnabled {
		em.addToCache(cacheKey, *response)
	}

	return response, nil
}

// generateEmbeddingWithOllama generates an embedding using Ollama
func (em *EmbeddingManager) generateEmbeddingWithOllama(request EmbeddingRequest) ([]float32, error) {
	// Get the AI manager to use Ollama
	ai := GetAIManager()
	if ai == nil || !ai.IsEnabled() {
		return nil, fmt.Errorf("AI manager not available")
	}

	// Build the prompt for embedding
	prompt := request.Command

	// Add directory context if enabled
	if em.config.DirectoryIncluded && request.Directory != "" {
		prompt = fmt.Sprintf("Directory: %s\nCommand: %s", request.Directory, request.Command)
	}

	// Add additional context if enabled
	if em.config.ContextIncluded && len(request.Context) > 0 {
		contextStr := ""
		for key, value := range request.Context {
			contextStr += fmt.Sprintf("%s: %s\n", key, value)
		}
		prompt = fmt.Sprintf("%s\n%s", contextStr, prompt)
	}

	// Call Ollama to generate embedding
	// This is a placeholder - actual Ollama embedding API call would go here
	// In a real implementation, we would:
	// 1. Call the Ollama embedding API
	// 2. Parse the response
	// 3. Return the embedding

	// For now, return a placeholder embedding by hashing the command
	return em.generatePlaceholderEmbedding(request)
}

// generateEmbeddingWithONNX generates an embedding using ONNX model
func (em *EmbeddingManager) generateEmbeddingWithONNX(request EmbeddingRequest) ([]float32, error) {
	// Check if ONNX runtime is initialized
	if em.onnxRuntime == nil {
		return nil, fmt.Errorf("ONNX runtime not initialized")
	}

	// Build the input text for the model
	var inputText string

	// Start with the command
	inputText = request.Command

	// Add directory context if enabled
	if em.config.DirectoryIncluded && request.Directory != "" {
		inputText = fmt.Sprintf("Directory: %s\nCommand: %s", request.Directory, inputText)
	}

	// Add additional context if enabled
	if em.config.ContextIncluded && len(request.Context) > 0 {
		contextStr := ""
		for key, value := range request.Context {
			contextStr += fmt.Sprintf("%s: %s\n", key, value)
		}
		inputText = fmt.Sprintf("%s\n%s", contextStr, inputText)
	}

	// Generate the embedding using the ONNX runtime
	embedding, err := em.onnxRuntime.GenerateEmbedding(inputText)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding with ONNX: %v", err)
	}

	// Ensure the embedding is normalized
	normalizeVector(embedding)

	return embedding, nil
}

// generatePlaceholderEmbedding generates a placeholder embedding for demonstration
func (em *EmbeddingManager) generatePlaceholderEmbedding(request EmbeddingRequest) ([]float32, error) {
	// Create a deterministic embedding based on the command
	// This is just for demonstration purposes
	commandHash := sha256.Sum256([]byte(request.Command))
	dirHash := sha256.Sum256([]byte(request.Directory))

	// Create an embedding of the configured size
	embedding := make([]float32, em.config.EmbbedingSize)

	// Fill with values derived from the hash
	for i := 0; i < em.config.EmbbedingSize; i++ {
		if i < 32 {
			// Use command hash for first part
			embedding[i] = float32(commandHash[i]) / 255.0
		} else if i < 64 {
			// Use directory hash for second part
			embedding[i] = float32(dirHash[i-32]) / 255.0
		} else {
			// Use a combination for the rest
			embedding[i] = float32((commandHash[i%32] + dirHash[i%32])) / 510.0
		}
	}

	// Normalize the embedding
	normalizeVector(embedding)

	return embedding, nil
}

// generateCacheKey generates a cache key for an embedding request
func (em *EmbeddingManager) generateCacheKey(request EmbeddingRequest) string {
	// Create a string containing all components for the key
	keyComponents := []string{request.Command, request.Directory}

	// Add context values if included
	if em.config.ContextIncluded && len(request.Context) > 0 {
		contextKeys := make([]string, 0, len(request.Context))
		for key := range request.Context {
			contextKeys = append(contextKeys, key)
		}

		// Sort for deterministic ordering
		sort.Strings(contextKeys)

		for _, key := range contextKeys {
			keyComponents = append(keyComponents, key, request.Context[key])
		}
	}

	// Join all components and hash
	keyString := strings.Join(keyComponents, "|")
	hash := sha256.Sum256([]byte(keyString))
	return hex.EncodeToString(hash[:])
}

// getFromCache gets an embedding from the cache
func (em *EmbeddingManager) getFromCache(key string) *EmbeddingResponse {
	em.cache.Mutex.RLock()
	defer em.cache.Mutex.RUnlock()

	if embedding, ok := em.cache.Embeddings[key]; ok {
		// Clone the embedding to avoid modification
		clonedEmbedding := embedding
		return &clonedEmbedding
	}

	return nil
}

// addToCache adds an embedding to the cache
func (em *EmbeddingManager) addToCache(key string, embedding EmbeddingResponse) {
	em.cache.Mutex.Lock()
	defer em.cache.Mutex.Unlock()

	// Add to cache
	em.cache.Embeddings[key] = embedding

	// If cache is full, remove oldest entries
	if len(em.cache.Embeddings) > em.cache.MaxSize {
		// Find oldest entries
		type cacheEntry struct {
			key       string
			timestamp time.Time
		}

		entries := make([]cacheEntry, 0, len(em.cache.Embeddings))
		for k, v := range em.cache.Embeddings {
			entries = append(entries, cacheEntry{k, v.Timestamp})
		}

		// Sort by timestamp (oldest first)
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].timestamp.Before(entries[j].timestamp)
		})

		// Remove oldest entries until we're under the limit
		for i := 0; i < len(entries)-em.cache.MaxSize; i++ {
			delete(em.cache.Embeddings, entries[i].key)
		}
	}
}

// GetStats returns statistics about the embedding system
func (em *EmbeddingManager) GetStats() map[string]interface{} {
	em.mutex.RLock()
	defer em.mutex.RUnlock()

	cacheSize := 0
	if em.cache != nil {
		em.cache.Mutex.RLock()
		cacheSize = len(em.cache.Embeddings)
		em.cache.Mutex.RUnlock()
	}

	return map[string]interface{}{
		"enabled":            em.config.Enabled,
		"initialized":        em.isInitialized,
		"model_path":         em.config.ModelPath,
		"model_type":         em.config.ModelType,
		"embedding_size":     em.config.EmbbedingSize,
		"use_ollama":         em.config.UseOllama,
		"ollama_model":       em.config.OllamaModel,
		"cache_enabled":      em.config.CacheEnabled,
		"cache_size":         em.config.CacheSize,
		"batch_size":         em.config.BatchSize,
		"context_included":   em.config.ContextIncluded,
		"directory_included": em.config.DirectoryIncluded,
		"common_commands":    em.config.CommonCommands,
		"cache_entries":      cacheSize,
	}
}

// UpdateConfig updates the embedding configuration
func (em *EmbeddingManager) UpdateConfig(config EmbeddingConfig) error {
	em.mutex.Lock()
	defer em.mutex.Unlock()

	em.config = config

	// Update cache size
	if em.cache != nil {
		em.cache.MaxSize = config.CacheSize
	}

	// Update ConfigManager if available
	configManager := GetConfigManager()
	if configManager != nil {
		err := configManager.UpdateEmbeddingConfig(&config)
		if err != nil {
			fmt.Printf("Warning: Failed to update config manager: %v\n", err)
			// Continue anyway and try to save locally
		}
	}

	// Always save locally as well for backward compatibility
	return em.saveConfig()
}

// Close closes the embedding manager
func (em *EmbeddingManager) Close() error {
	em.mutex.Lock()
	defer em.mutex.Unlock()

	// Nothing to close for now
	return nil
}

// Helper functions

// normalizeVector normalizes a vector to unit length
func normalizeVector(vec []float32) {
	var sum float32
	for _, v := range vec {
		sum += v * v
	}

	length := float32(math.Sqrt(float64(sum)))
	if length > 0 {
		for i := range vec {
			vec[i] /= length
		}
	}
}

// Global EmbeddingManager instance
var globalEmbeddingManager *EmbeddingManager

// GetEmbeddingManager returns the global EmbeddingManager instance
func GetEmbeddingManager() *EmbeddingManager {
	if globalEmbeddingManager == nil {
		var err error
		globalEmbeddingManager, err = NewEmbeddingManager()
		if err != nil {
			fmt.Printf("Error initializing embedding manager: %v\n", err)
			return nil
		}
	}
	return globalEmbeddingManager
}
