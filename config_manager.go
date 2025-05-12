package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ConfigManager provides centralized configuration management for all Delta CLI components
type ConfigManager struct {
	configDir      string
	configPath     string
	mutex          sync.RWMutex
	isInitialized  bool
	lastSaved      time.Time
	memoryConfig   *MemoryConfig
	aiConfig       *AIPredictionConfig
	vectorConfig   *VectorDBConfig
	embeddingConfig *EmbeddingConfig
	inferenceConfig *InferenceConfig
	learningConfig *LearningConfig
	tokenConfig    *TokenizerConfig
	agentConfig    *AgentManagerConfig
}

// SystemConfig contains all component configurations
type SystemConfig struct {
	ConfigVersion   string              `json:"config_version"`
	LastUpdated     time.Time           `json:"last_updated"`
	MemoryConfig    *MemoryConfig       `json:"memory_config,omitempty"`
	AIConfig        *AIPredictionConfig `json:"ai_config,omitempty"`
	VectorConfig    *VectorDBConfig     `json:"vector_config,omitempty"`
	EmbeddingConfig *EmbeddingConfig    `json:"embedding_config,omitempty"`
	InferenceConfig *InferenceConfig    `json:"inference_config,omitempty"`
	LearningConfig  *LearningConfig     `json:"learning_config,omitempty"`
	TokenConfig     *TokenizerConfig    `json:"token_config,omitempty"`
	AgentConfig     *AgentManagerConfig `json:"agent_config,omitempty"`
}

// NewConfigManager creates a new configuration manager instance
func NewConfigManager() (*ConfigManager, error) {
	// Set up config directory in user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = os.Getenv("HOME")
	}

	// Use ~/.config/delta directory for config files
	configDir := filepath.Join(homeDir, ".config", "delta")
	err = os.MkdirAll(configDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create config directory: %v", err)
	}

	configPath := filepath.Join(configDir, "system_config.json")

	// Create the config manager instance
	cm := &ConfigManager{
		configDir:     configDir,
		configPath:    configPath,
		mutex:         sync.RWMutex{},
		isInitialized: false,
		lastSaved:     time.Time{},
	}

	return cm, nil
}

// Initialize the configuration manager
func (cm *ConfigManager) Initialize() error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Try to load existing configuration
	if err := cm.loadConfig(); err != nil {
		// If loading fails, collect configs from individual components
		if err := cm.collectConfigs(); err != nil {
			return fmt.Errorf("failed to collect component configurations: %v", err)
		}

		// Save the consolidated configuration
		if err := cm.saveConfig(); err != nil {
			return fmt.Errorf("failed to save initial configuration: %v", err)
		}
	}

	cm.isInitialized = true
	return nil
}

// loadConfig loads the configuration from disk
func (cm *ConfigManager) loadConfig() error {
	// Check if config file exists
	_, err := os.Stat(cm.configPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("configuration file does not exist")
	}

	// Read the config file
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return err
	}

	// Parse the JSON data
	var config SystemConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	// Set component configurations
	cm.memoryConfig = config.MemoryConfig
	cm.aiConfig = config.AIConfig
	cm.vectorConfig = config.VectorConfig
	cm.embeddingConfig = config.EmbeddingConfig
	cm.inferenceConfig = config.InferenceConfig
	cm.learningConfig = config.LearningConfig
	cm.tokenConfig = config.TokenConfig
	cm.agentConfig = config.AgentConfig
	cm.lastSaved = config.LastUpdated

	return nil
}

// collectConfigs gathers configurations from individual components
func (cm *ConfigManager) collectConfigs() error {
	// Memory Manager
	mm := GetMemoryManager()
	if mm != nil {
		cm.memoryConfig = &mm.config
	}

	// AI Manager
	ai := GetAIManager()
	if ai != nil {
		cm.aiConfig = &ai.config
	}

	// Vector DB Manager
	vdb := GetVectorDBManager()
	if vdb != nil {
		cm.vectorConfig = &vdb.config
	}

	// Embedding Manager
	em := GetEmbeddingManager()
	if em != nil {
		cm.embeddingConfig = &em.config
	}

	// Inference Manager
	im := GetInferenceManager()
	if im != nil {
		cm.inferenceConfig = &im.inferenceConfig
		cm.learningConfig = &im.learningConfig
	}

	// Tokenizer
	tk := GetTokenizer()
	if tk != nil {
		cm.tokenConfig = &tk.Config
	}

	// Agent Manager
	am := GetAgentManager()
	if am != nil {
		cm.agentConfig = &am.config
	}

	return nil
}

// saveConfig saves the configuration to disk
func (cm *ConfigManager) saveConfig() error {
	// Create the system config object
	config := SystemConfig{
		ConfigVersion:   "1.0",
		LastUpdated:     time.Now(),
		MemoryConfig:    cm.memoryConfig,
		AIConfig:        cm.aiConfig,
		VectorConfig:    cm.vectorConfig,
		EmbeddingConfig: cm.embeddingConfig,
		InferenceConfig: cm.inferenceConfig,
		LearningConfig:  cm.learningConfig,
		TokenConfig:     cm.tokenConfig,
		AgentConfig:     cm.agentConfig,
	}

	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(cm.configPath)
	if err = os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Write to file
	if err := os.WriteFile(cm.configPath, data, 0644); err != nil {
		return err
	}

	cm.lastSaved = config.LastUpdated
	return nil
}

// GetMemoryConfig returns the memory configuration
func (cm *ConfigManager) GetMemoryConfig() *MemoryConfig {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return cm.memoryConfig
}

// GetAIConfig returns the AI prediction configuration
func (cm *ConfigManager) GetAIConfig() *AIPredictionConfig {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return cm.aiConfig
}

// GetVectorConfig returns the vector database configuration
func (cm *ConfigManager) GetVectorConfig() *VectorDBConfig {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return cm.vectorConfig
}

// GetEmbeddingConfig returns the embedding configuration
func (cm *ConfigManager) GetEmbeddingConfig() *EmbeddingConfig {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return cm.embeddingConfig
}

// GetInferenceConfig returns the inference configuration
func (cm *ConfigManager) GetInferenceConfig() *InferenceConfig {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return cm.inferenceConfig
}

// GetLearningConfig returns the learning configuration
func (cm *ConfigManager) GetLearningConfig() *LearningConfig {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return cm.learningConfig
}

// GetTokenConfig returns the tokenizer configuration
func (cm *ConfigManager) GetTokenConfig() *TokenizerConfig {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return cm.tokenConfig
}

// GetAgentConfig returns the agent manager configuration
func (cm *ConfigManager) GetAgentConfig() *AgentManagerConfig {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return cm.agentConfig
}

// UpdateMemoryConfig updates the memory configuration
func (cm *ConfigManager) UpdateMemoryConfig(config *MemoryConfig) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.memoryConfig = config
	return cm.saveConfig()
}

// UpdateAIConfig updates the AI prediction configuration
func (cm *ConfigManager) UpdateAIConfig(config *AIPredictionConfig) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.aiConfig = config
	return cm.saveConfig()
}

// UpdateVectorConfig updates the vector database configuration
func (cm *ConfigManager) UpdateVectorConfig(config *VectorDBConfig) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.vectorConfig = config
	return cm.saveConfig()
}

// UpdateEmbeddingConfig updates the embedding configuration
func (cm *ConfigManager) UpdateEmbeddingConfig(config *EmbeddingConfig) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.embeddingConfig = config
	return cm.saveConfig()
}

// UpdateInferenceConfig updates the inference configuration
func (cm *ConfigManager) UpdateInferenceConfig(config *InferenceConfig) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.inferenceConfig = config
	return cm.saveConfig()
}

// UpdateLearningConfig updates the learning configuration
func (cm *ConfigManager) UpdateLearningConfig(config *LearningConfig) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.learningConfig = config
	return cm.saveConfig()
}

// UpdateTokenConfig updates the tokenizer configuration
func (cm *ConfigManager) UpdateTokenConfig(config *TokenizerConfig) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.tokenConfig = config
	return cm.saveConfig()
}

// UpdateAgentConfig updates the agent manager configuration
func (cm *ConfigManager) UpdateAgentConfig(config *AgentManagerConfig) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.agentConfig = config
	return cm.saveConfig()
}

// ExportConfig exports the configuration to a file
func (cm *ConfigManager) ExportConfig(path string) error {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	// Create the system config object
	config := SystemConfig{
		ConfigVersion:   "1.0",
		LastUpdated:     time.Now(),
		MemoryConfig:    cm.memoryConfig,
		AIConfig:        cm.aiConfig,
		VectorConfig:    cm.vectorConfig,
		EmbeddingConfig: cm.embeddingConfig,
		InferenceConfig: cm.inferenceConfig,
		LearningConfig:  cm.learningConfig,
		TokenConfig:     cm.tokenConfig,
		AgentConfig:     cm.agentConfig,
	}

	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	// Create directory if needed
	dir := filepath.Dir(path)
	if err = os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(path, data, 0644)
}

// ImportConfig imports the configuration from a file
func (cm *ConfigManager) ImportConfig(path string) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Read the config file
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Parse the JSON data
	var config SystemConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	// Set component configurations
	cm.memoryConfig = config.MemoryConfig
	cm.aiConfig = config.AIConfig
	cm.vectorConfig = config.VectorConfig
	cm.embeddingConfig = config.EmbeddingConfig
	cm.inferenceConfig = config.InferenceConfig
	cm.learningConfig = config.LearningConfig
	cm.tokenConfig = config.TokenConfig
	cm.agentConfig = config.AgentConfig

	// Save the configuration
	return cm.saveConfig()
}

// Global ConfigManager instance
var globalConfigManager *ConfigManager

// GetConfigManager returns the global ConfigManager instance
func GetConfigManager() *ConfigManager {
	if globalConfigManager == nil {
		var err error
		globalConfigManager, err = NewConfigManager()
		if err != nil {
			fmt.Printf("Error initializing config manager: %v\n", err)
			return nil
		}
	}
	return globalConfigManager
}