package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

// ConfigManager provides centralized configuration management for all Delta CLI components
type ConfigManager struct {
	configDir       string
	configPath      string
	mutex           sync.RWMutex
	isInitialized   bool
	lastSaved       time.Time
	i18nConfig      *I18nConfig
	updateConfig    *UpdateConfig
	memoryConfig    *MemoryConfig
	aiConfig        *AIPredictionConfig
	vectorConfig    *VectorDBConfig
	embeddingConfig *EmbeddingConfig
	inferenceConfig *InferenceConfig
	learningConfig  *LearningConfig
	tokenConfig     *TokenizerConfig
	agentConfig     *AgentManagerConfig
}

// I18nConfig contains internationalization settings
type I18nConfig struct {
	Locale             string `json:"locale"`
	FallbackLocale     string `json:"fallback_locale"`
	AutoDetectLanguage bool   `json:"auto_detect_language"`
}

// UpdateConfig contains auto-update system settings
type UpdateConfig struct {
	Enabled              bool   `json:"enabled"`
	CheckOnStartup       bool   `json:"check_on_startup"`
	AutoInstall          bool   `json:"auto_install"`
	Channel              string `json:"channel"`
	CheckInterval        string `json:"check_interval"`
	BackupBeforeUpdate   bool   `json:"backup_before_update"`
	AllowPrerelease      bool   `json:"allow_prerelease"`
	GitHubRepository     string `json:"github_repository"`
	DownloadDirectory    string `json:"download_directory"`
	LastCheck            string `json:"last_check"`
	LastVersion          string `json:"last_version"`
	SkipVersion          string `json:"skip_version"`
	NotificationLevel    string `json:"notification_level"`
}

// SystemConfig contains all component configurations
type SystemConfig struct {
	ConfigVersion   string              `json:"config_version"`
	LastUpdated     time.Time           `json:"last_updated"`
	I18nConfig      *I18nConfig         `json:"i18n_config,omitempty"`
	UpdateConfig    *UpdateConfig       `json:"update_config,omitempty"`
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
// InitializeBase initializes the configuration manager without updating components
// This avoids circular dependencies during startup
func (cm *ConfigManager) InitializeBase() error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Try to load existing configuration
	if err := cm.loadConfig(); err != nil {
		// If loading fails, create minimal config without collecting from components
		// to avoid circular dependencies

		// Save an empty configuration
		if err := cm.saveConfig(); err != nil {
			return fmt.Errorf("failed to save initial configuration: %v", err)
		}
	}

	// Apply just environment variables without updating components
	cm.applyEnvironmentVariables()

	cm.isInitialized = true
	return nil
}

// applyEnvironmentVariables applies settings from environment variables without updating components
func (cm *ConfigManager) applyEnvironmentVariables() {
	// Only apply environment variables if the component configs exist

	// I18n config overrides
	if cm.i18nConfig != nil {
		cm.i18nConfig.Locale = getEnvString("DELTA_LOCALE", cm.i18nConfig.Locale)
		cm.i18nConfig.FallbackLocale = getEnvString("DELTA_FALLBACK_LOCALE", cm.i18nConfig.FallbackLocale)
		cm.i18nConfig.AutoDetectLanguage = getEnvBool("DELTA_AUTO_DETECT_LANGUAGE", cm.i18nConfig.AutoDetectLanguage)
	}

	// Update config overrides
	if cm.updateConfig != nil {
		cm.updateConfig.Enabled = getEnvBool("DELTA_UPDATE_ENABLED", cm.updateConfig.Enabled)
		cm.updateConfig.CheckOnStartup = getEnvBool("DELTA_UPDATE_CHECK_ON_STARTUP", cm.updateConfig.CheckOnStartup)
		cm.updateConfig.AutoInstall = getEnvBool("DELTA_UPDATE_AUTO_INSTALL", cm.updateConfig.AutoInstall)
		cm.updateConfig.Channel = getEnvString("DELTA_UPDATE_CHANNEL", cm.updateConfig.Channel)
		cm.updateConfig.CheckInterval = getEnvString("DELTA_UPDATE_CHECK_INTERVAL", cm.updateConfig.CheckInterval)
		cm.updateConfig.BackupBeforeUpdate = getEnvBool("DELTA_UPDATE_BACKUP_BEFORE_UPDATE", cm.updateConfig.BackupBeforeUpdate)
		cm.updateConfig.AllowPrerelease = getEnvBool("DELTA_UPDATE_ALLOW_PRERELEASE", cm.updateConfig.AllowPrerelease)
		cm.updateConfig.GitHubRepository = getEnvString("DELTA_UPDATE_GITHUB_REPOSITORY", cm.updateConfig.GitHubRepository)
		cm.updateConfig.DownloadDirectory = getEnvString("DELTA_UPDATE_DOWNLOAD_DIRECTORY", cm.updateConfig.DownloadDirectory)
		cm.updateConfig.NotificationLevel = getEnvString("DELTA_UPDATE_NOTIFICATION_LEVEL", cm.updateConfig.NotificationLevel)
	}

	// Memory config overrides
	if cm.memoryConfig != nil {
		cm.memoryConfig.Enabled = getEnvBool("DELTA_MEMORY_ENABLED", cm.memoryConfig.Enabled)
		cm.memoryConfig.CollectCommands = getEnvBool("DELTA_MEMORY_COLLECT_COMMANDS", cm.memoryConfig.CollectCommands)
		cm.memoryConfig.MaxEntries = getEnvInt("DELTA_MEMORY_MAX_ENTRIES", cm.memoryConfig.MaxEntries)
		cm.memoryConfig.StoragePath = getEnvString("DELTA_MEMORY_STORAGE_PATH", cm.memoryConfig.StoragePath)
	}

	// AI config overrides
	if cm.aiConfig != nil {
		cm.aiConfig.Enabled = getEnvBool("DELTA_AI_ENABLED", cm.aiConfig.Enabled)
		cm.aiConfig.ModelName = getEnvString("DELTA_AI_MODEL", cm.aiConfig.ModelName)
		cm.aiConfig.ServerURL = getEnvString("DELTA_AI_SERVER_URL", cm.aiConfig.ServerURL)
	}

	// Vector DB config overrides
	if cm.vectorConfig != nil {
		cm.vectorConfig.Enabled = getEnvBool("DELTA_VECTOR_ENABLED", cm.vectorConfig.Enabled)
		cm.vectorConfig.DistanceMetric = getEnvString("DELTA_VECTOR_DISTANCE_METRIC", cm.vectorConfig.DistanceMetric)
		cm.vectorConfig.DBPath = getEnvString("DELTA_VECTOR_DB_PATH", cm.vectorConfig.DBPath)
	}

	// Embedding config overrides
	if cm.embeddingConfig != nil {
		cm.embeddingConfig.Enabled = getEnvBool("DELTA_EMBEDDING_ENABLED", cm.embeddingConfig.Enabled)
		cm.embeddingConfig.ModelPath = getEnvString("DELTA_EMBEDDING_MODEL_PATH", cm.embeddingConfig.ModelPath)
		cm.embeddingConfig.ModelURL = getEnvString("DELTA_EMBEDDING_MODEL_URL", cm.embeddingConfig.ModelURL)
	}

	// Inference config overrides
	if cm.inferenceConfig != nil {
		cm.inferenceConfig.Enabled = getEnvBool("DELTA_INFERENCE_ENABLED", cm.inferenceConfig.Enabled)
		cm.inferenceConfig.UseLocalInference = getEnvBool("DELTA_INFERENCE_USE_LOCAL", cm.inferenceConfig.UseLocalInference)
		cm.inferenceConfig.ModelPath = getEnvString("DELTA_INFERENCE_MODEL_PATH", cm.inferenceConfig.ModelPath)
		cm.inferenceConfig.Temperature = getEnvFloat("DELTA_INFERENCE_TEMPERATURE", cm.inferenceConfig.Temperature)
	}
}

func (cm *ConfigManager) Initialize() error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// If not already initialized, do the base initialization
	if !cm.isInitialized {
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

		// Apply environment variable overrides
		cm.applyEnvironmentOverrides()

		cm.isInitialized = true
	} else {
		// Already initialized, just update components
		cm.updateComponentConfigs()
	}

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
	cm.i18nConfig = config.I18nConfig
	cm.updateConfig = config.UpdateConfig
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
	// I18n Manager
	i18n := GetI18nManager()
	if i18n != nil {
		cm.i18nConfig = &I18nConfig{
			Locale:             i18n.GetCurrentLocale(),
			FallbackLocale:     i18n.fallbackLocale,
			AutoDetectLanguage: true,
		}
	} else {
		// Default i18n config
		cm.i18nConfig = &I18nConfig{
			Locale:             "en",
			FallbackLocale:     "en",
			AutoDetectLanguage: true,
		}
	}

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
		I18nConfig:      cm.i18nConfig,
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

// GetI18nConfig returns the i18n configuration
func (cm *ConfigManager) GetI18nConfig() *I18nConfig {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return cm.i18nConfig
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

// GetUpdateConfig returns the update configuration
func (cm *ConfigManager) GetUpdateConfig() *UpdateConfig {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()
	return cm.updateConfig
}

// UpdateI18nConfig updates the i18n configuration
func (cm *ConfigManager) UpdateI18nConfig(config *I18nConfig) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.i18nConfig = config
	return cm.saveConfig()
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

// UpdateUpdateConfig updates the update configuration
func (cm *ConfigManager) UpdateUpdateConfig(config *UpdateConfig) error {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.updateConfig = config
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
		I18nConfig:      cm.i18nConfig,
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
	cm.i18nConfig = config.I18nConfig
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

// applyEnvironmentOverrides applies settings from environment variables
func (cm *ConfigManager) applyEnvironmentOverrides() {
	// Only apply environment variables if the component configs exist

	// I18n config overrides
	if cm.i18nConfig != nil {
		cm.i18nConfig.Locale = getEnvString("DELTA_LOCALE", cm.i18nConfig.Locale)
		cm.i18nConfig.FallbackLocale = getEnvString("DELTA_FALLBACK_LOCALE", cm.i18nConfig.FallbackLocale)
		cm.i18nConfig.AutoDetectLanguage = getEnvBool("DELTA_AUTO_DETECT_LANGUAGE", cm.i18nConfig.AutoDetectLanguage)
	}

	// Memory config overrides
	if cm.memoryConfig != nil {
		cm.memoryConfig.Enabled = getEnvBool("DELTA_MEMORY_ENABLED", cm.memoryConfig.Enabled)
		cm.memoryConfig.CollectCommands = getEnvBool("DELTA_MEMORY_COLLECT_COMMANDS", cm.memoryConfig.CollectCommands)
		cm.memoryConfig.MaxEntries = getEnvInt("DELTA_MEMORY_MAX_ENTRIES", cm.memoryConfig.MaxEntries)
		cm.memoryConfig.StoragePath = getEnvString("DELTA_MEMORY_STORAGE_PATH", cm.memoryConfig.StoragePath)
	}

	// AI config overrides
	if cm.aiConfig != nil {
		cm.aiConfig.Enabled = getEnvBool("DELTA_AI_ENABLED", cm.aiConfig.Enabled)
		cm.aiConfig.ModelName = getEnvString("DELTA_AI_MODEL", cm.aiConfig.ModelName)
		cm.aiConfig.ServerURL = getEnvString("DELTA_AI_SERVER_URL", cm.aiConfig.ServerURL)
	}

	// Vector config overrides
	if cm.vectorConfig != nil {
		cm.vectorConfig.Enabled = getEnvBool("DELTA_VECTOR_ENABLED", cm.vectorConfig.Enabled)
		cm.vectorConfig.DBPath = getEnvString("DELTA_VECTOR_DB_PATH", cm.vectorConfig.DBPath)
		cm.vectorConfig.InMemoryMode = getEnvBool("DELTA_VECTOR_IN_MEMORY_MODE", cm.vectorConfig.InMemoryMode)
	}

	// Embedding config overrides
	if cm.embeddingConfig != nil {
		cm.embeddingConfig.Enabled = getEnvBool("DELTA_EMBEDDING_ENABLED", cm.embeddingConfig.Enabled)
		cm.embeddingConfig.Dimensions = getEnvInt("DELTA_EMBEDDING_DIMENSIONS", cm.embeddingConfig.Dimensions)
		cm.embeddingConfig.CacheSize = getEnvInt("DELTA_EMBEDDING_CACHE_SIZE", cm.embeddingConfig.CacheSize)
	}

	// Inference config overrides
	if cm.inferenceConfig != nil {
		cm.inferenceConfig.Enabled = getEnvBool("DELTA_INFERENCE_ENABLED", cm.inferenceConfig.Enabled)
		cm.inferenceConfig.UseLocalInference = getEnvBool("DELTA_INFERENCE_USE_LOCAL", cm.inferenceConfig.UseLocalInference)
		cm.inferenceConfig.MaxTokens = getEnvInt("DELTA_INFERENCE_MAX_TOKENS", cm.inferenceConfig.MaxTokens)
		cm.inferenceConfig.Temperature = getEnvFloat("DELTA_INFERENCE_TEMPERATURE", cm.inferenceConfig.Temperature)
	}

	// Learning config overrides
	if cm.learningConfig != nil {
		cm.learningConfig.CollectFeedback = getEnvBool("DELTA_LEARNING_COLLECT_FEEDBACK", cm.learningConfig.CollectFeedback)
		cm.learningConfig.UseCustomModel = getEnvBool("DELTA_LEARNING_USE_CUSTOM_MODEL", cm.learningConfig.UseCustomModel)
		cm.learningConfig.CustomModelPath = getEnvString("DELTA_LEARNING_CUSTOM_MODEL_PATH", cm.learningConfig.CustomModelPath)
		cm.learningConfig.TrainingThreshold = getEnvInt("DELTA_LEARNING_TRAINING_THRESHOLD", cm.learningConfig.TrainingThreshold)
	}

	// Tokenizer config overrides
	if cm.tokenConfig != nil {
		cm.tokenConfig.Enabled = getEnvBool("DELTA_TOKEN_ENABLED", cm.tokenConfig.Enabled)
		cm.tokenConfig.VocabSize = getEnvInt("DELTA_TOKEN_VOCAB_SIZE", cm.tokenConfig.VocabSize)
		cm.tokenConfig.StoragePath = getEnvString("DELTA_TOKEN_STORAGE_PATH", cm.tokenConfig.StoragePath)
	}

	// Agent config overrides
	if cm.agentConfig != nil {
		cm.agentConfig.Enabled = getEnvBool("DELTA_AGENT_ENABLED", cm.agentConfig.Enabled)
		cm.agentConfig.AgentStoragePath = getEnvString("DELTA_AGENT_STORAGE_PATH", cm.agentConfig.AgentStoragePath)
		cm.agentConfig.CacheStoragePath = getEnvString("DELTA_AGENT_CACHE_PATH", cm.agentConfig.CacheStoragePath)
		cm.agentConfig.UseDockerBuilds = getEnvBool("DELTA_AGENT_USE_DOCKER", cm.agentConfig.UseDockerBuilds)
		cm.agentConfig.UseAIAssistance = getEnvBool("DELTA_AGENT_USE_AI", cm.agentConfig.UseAIAssistance)
	}

	// Important: Update the components with the new settings if they're already initialized
	cm.updateComponentConfigs()
}

// updateComponentConfigs updates all component managers with the current configuration
func (cm *ConfigManager) updateComponentConfigs() {
	// I18n Manager
	i18n := GetI18nManager()
	if i18n != nil && cm.i18nConfig != nil {
		i18n.SetLocale(cm.i18nConfig.Locale)
		i18n.fallbackLocale = cm.i18nConfig.FallbackLocale
	}

	// Memory Manager
	mm := GetMemoryManager()
	if mm != nil && cm.memoryConfig != nil {
		mm.config = *cm.memoryConfig
	}

	// AI Manager
	ai := GetAIManager()
	if ai != nil && cm.aiConfig != nil {
		ai.config = *cm.aiConfig
	}

	// Vector DB Manager
	vdb := GetVectorDBManager()
	if vdb != nil && cm.vectorConfig != nil {
		vdb.config = *cm.vectorConfig
	}

	// Embedding Manager
	em := GetEmbeddingManager()
	if em != nil && cm.embeddingConfig != nil {
		em.config = *cm.embeddingConfig
	}

	// Inference Manager
	im := GetInferenceManager()
	if im != nil {
		if cm.inferenceConfig != nil {
			im.inferenceConfig = *cm.inferenceConfig
		}
		if cm.learningConfig != nil {
			im.learningConfig = *cm.learningConfig
		}
	}

	// Tokenizer
	tk := GetTokenizer()
	if tk != nil && cm.tokenConfig != nil {
		tk.Config = *cm.tokenConfig
	}

	// Agent Manager
	am := GetAgentManager()
	if am != nil && cm.agentConfig != nil {
		am.config = *cm.agentConfig
	}
}

// Global ConfigManager instance
var globalConfigManager *ConfigManager

// Helper functions for environment variables
func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	val, err := strconv.ParseBool(value)
	if err != nil {
		return defaultValue
	}

	return val
}

func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	val, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}

	return val
}

func getEnvString(key string, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	return value
}

func getEnvFloat(key string, defaultValue float64) float64 {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}

	val, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return defaultValue
	}

	return val
}

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
