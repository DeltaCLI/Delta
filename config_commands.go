package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Helper function to list Delta environment variables
func listDeltaEnvVars() map[string]string {
	envVars := make(map[string]string)
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "DELTA_") {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) == 2 {
				envVars[parts[0]] = parts[1]
			}
		}
	}
	return envVars
}

// HandleConfigCommand processes config-related commands
func HandleConfigCommand(args []string) bool {
	// Get the ConfigManager instance
	cm := GetConfigManager()
	if cm == nil {
		fmt.Println("Failed to initialize config manager")
		return true
	}

	// Initialize if not already done
	if !cm.isInitialized {
		err := cm.Initialize()
		if err != nil {
			fmt.Printf("Error initializing config manager: %v\n", err)
			return true
		}
	}

	// Handle commands
	if len(args) == 0 {
		// Show config status
		showConfigStatus(cm)
		return true
	}

	// Handle special commands
	if len(args) >= 1 {
		cmd := args[0]

		switch cmd {
		case "status":
			// Show status
			showConfigStatus(cm)
			return true

		case "list":
			// List configurations
			listConfigurations(cm)
			return true

		case "export":
			// Export configuration
			if len(args) >= 2 {
				exportConfiguration(cm, args[1])
			} else {
				fmt.Println("Usage: :config export <file_path>")
			}
			return true

		case "import":
			// Import configuration
			if len(args) >= 2 {
				importConfiguration(cm, args[1])
			} else {
				fmt.Println("Usage: :config import <file_path>")
			}
			return true

		case "edit":
			// Edit configuration
			if len(args) >= 2 {
				editConfiguration(cm, args[1:])
			} else {
				fmt.Println("Usage: :config edit <component> [setting=value]")
				fmt.Println("Available components: memory, ai, vector, embedding, inference, learning, token, agent")
			}
			return true

		case "reset":
			// Reset configuration
			if len(args) >= 2 {
				if args[1] == "confirm" {
					resetConfiguration(cm)
				} else {
					fmt.Println("Warning: This will reset all configurations to default values.")
					fmt.Println("To confirm, use: :config reset confirm")
				}
			} else {
				fmt.Println("Warning: This will reset all configurations to default values.")
				fmt.Println("To confirm, use: :config reset confirm")
			}
			return true

		case "env":
			// Handle environment variable operations
			if len(args) >= 2 {
				switch args[1] {
				case "list":
					listEnvironmentVariables()
					return true
				case "clear":
					clearEnvironmentVariables()
					return true
				case "help":
					showEnvironmentVariableHelp()
					return true
				default:
					fmt.Printf("Unknown environment variable command: %s\n", args[1])
					fmt.Println("Available commands: list, clear, help")
				}
			} else {
				listEnvironmentVariables()
			}
			return true

		case "help":
			// Show help
			showConfigHelp()
			return true
		}

		// If we get here, it's an unknown command
		fmt.Println("Unknown config command. Type :config help for available commands.")
		return true
	}

	return true
}

// showConfigStatus displays the current status of the configuration system
func showConfigStatus(cm *ConfigManager) {
	fmt.Println("Configuration System Status")
	fmt.Println("===========================")
	fmt.Printf("Configuration Path: %s\n", cm.configPath)
	fmt.Printf("Last Updated: %s\n", cm.lastSaved.Format(time.RFC1123))

	// Show component status
	fmt.Println("\nComponent Status:")
	if cm.memoryConfig != nil {
		fmt.Printf("Memory Config: %s\n", getComponentStatus(cm.memoryConfig.Enabled))
	} else {
		fmt.Println("Memory Config: Not available")
	}

	if cm.aiConfig != nil {
		fmt.Printf("AI Config: %s\n", getComponentStatus(cm.aiConfig.Enabled))
	} else {
		fmt.Println("AI Config: Not available")
	}

	if cm.vectorConfig != nil {
		fmt.Printf("Vector Config: %s\n", getComponentStatus(cm.vectorConfig.Enabled))
	} else {
		fmt.Println("Vector Config: Not available")
	}

	if cm.embeddingConfig != nil {
		fmt.Printf("Embedding Config: %s\n", getComponentStatus(cm.embeddingConfig.Enabled))
	} else {
		fmt.Println("Embedding Config: Not available")
	}

	if cm.learningConfig != nil {
		fmt.Printf("Inference Config: %s\n", getComponentStatus(cm.learningConfig.Enabled))
	} else {
		fmt.Println("Inference Config: Not available")
	}

	if cm.tokenConfig != nil {
		fmt.Printf("Tokenizer Config: %s\n", getComponentStatus(cm.tokenConfig.Enabled))
	} else {
		fmt.Println("Tokenizer Config: Not available")
	}

	if cm.agentConfig != nil {
		fmt.Printf("Agent Config: %s\n", getComponentStatus(cm.agentConfig.Enabled))
	} else {
		fmt.Println("Agent Config: Not available")
	}

	// Show active environment variables
	envVars := listDeltaEnvVars()
	if len(envVars) > 0 {
		fmt.Println("\nActive Environment Variables:")
		count := 0
		for k := range envVars {
			count++
			if count <= 3 {
				fmt.Printf("  %s=%s\n", k, envVars[k])
			}
		}
		if count > 3 {
			fmt.Printf("  ... and %d more (use ':config env list' to see all)\n", count-3)
		}
	}
}

// getComponentStatus returns a string representation of a component's status
func getComponentStatus(enabled bool) string {
	if enabled {
		return "Enabled"
	}
	return "Disabled"
}

// listConfigurations lists all configuration components
func listConfigurations(cm *ConfigManager) {
	fmt.Println("Available Configurations")
	fmt.Println("=======================")

	// Memory configuration
	if cm.memoryConfig != nil {
		fmt.Println("\nMemory Configuration:")
		fmt.Printf("  Enabled: %v\n", cm.memoryConfig.Enabled)
		fmt.Printf("  Collect Commands: %v\n", cm.memoryConfig.CollectCommands)
		fmt.Printf("  Max Entries: %d\n", cm.memoryConfig.MaxEntries)
		fmt.Printf("  Storage Path: %s\n", cm.memoryConfig.StoragePath)
		fmt.Printf("  Privacy Filters: %v\n", cm.memoryConfig.PrivacyFilter)
	}

	// AI configuration
	if cm.aiConfig != nil {
		fmt.Println("\nAI Configuration:")
		fmt.Printf("  Enabled: %v\n", cm.aiConfig.Enabled)
		fmt.Printf("  Model: %s\n", cm.aiConfig.ModelName)
		fmt.Printf("  Server URL: %s\n", cm.aiConfig.ServerURL)
	}

	// Vector configuration
	if cm.vectorConfig != nil {
		fmt.Println("\nVector Database Configuration:")
		fmt.Printf("  Enabled: %v\n", cm.vectorConfig.Enabled)
		fmt.Printf("  Database Path: %s\n", cm.vectorConfig.DBPath)
		fmt.Printf("  In-Memory Mode: %v\n", cm.vectorConfig.InMemoryMode)
	}

	// Embedding configuration
	if cm.embeddingConfig != nil {
		fmt.Println("\nEmbedding Configuration:")
		fmt.Printf("  Enabled: %v\n", cm.embeddingConfig.Enabled)
		fmt.Printf("  Embedding Dimensions: %d\n", cm.embeddingConfig.Dimensions)
		fmt.Printf("  Cache Size: %d\n", cm.embeddingConfig.CacheSize)
	}

	// Inference configuration
	if cm.inferenceConfig != nil {
		fmt.Println("\nInference Configuration:")
		fmt.Printf("  Enabled: %v\n", cm.inferenceConfig.Enabled)
		fmt.Printf("  Local Inference: %v\n", cm.inferenceConfig.UseLocalInference)
		fmt.Printf("  Max Tokens: %d\n", cm.inferenceConfig.MaxTokens)
		fmt.Printf("  Temperature: %.2f\n", cm.inferenceConfig.Temperature)
	}

	// Learning configuration
	if cm.learningConfig != nil {
		fmt.Println("\nLearning Configuration:")
		fmt.Printf("  Collect Feedback: %v\n", cm.learningConfig.CollectFeedback)
		fmt.Printf("  Use Custom Model: %v\n", cm.learningConfig.UseCustomModel)
		fmt.Printf("  Custom Model Path: %s\n", cm.learningConfig.CustomModelPath)
		fmt.Printf("  Training Threshold: %d\n", cm.learningConfig.TrainingThreshold)
	}

	// Tokenizer configuration
	if cm.tokenConfig != nil {
		fmt.Println("\nTokenizer Configuration:")
		fmt.Printf("  Enabled: %v\n", cm.tokenConfig.Enabled)
		fmt.Printf("  Vocabulary Size: %d\n", cm.tokenConfig.VocabSize)
		fmt.Printf("  Token Storage Path: %s\n", cm.tokenConfig.StoragePath)
	}

	// Agent configuration
	if cm.agentConfig != nil {
		fmt.Println("\nAgent Configuration:")
		fmt.Printf("  Enabled: %v\n", cm.agentConfig.Enabled)
		fmt.Printf("  Agent Storage Path: %s\n", cm.agentConfig.AgentStoragePath)
		fmt.Printf("  Cache Storage Path: %s\n", cm.agentConfig.CacheStoragePath)
		fmt.Printf("  Docker Builds: %v\n", cm.agentConfig.UseDockerBuilds)
		fmt.Printf("  AI Assistance: %v\n", cm.agentConfig.UseAIAssistance)
	}
}

// exportConfiguration exports the configuration to a file
func exportConfiguration(cm *ConfigManager, path string) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		fmt.Printf("Error creating directory: %v\n", err)
		return
	}

	// Export configuration
	if err := cm.ExportConfig(path); err != nil {
		fmt.Printf("Error exporting configuration: %v\n", err)
		return
	}

	fmt.Printf("Configuration exported to %s\n", path)
}

// importConfiguration imports the configuration from a file
func importConfiguration(cm *ConfigManager, path string) {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Printf("Error: File not found: %s\n", path)
		return
	}

	// Confirm import
	fmt.Println("Warning: This will replace your current configuration.")
	fmt.Print("Do you want to continue? (y/n): ")
	var confirm string
	fmt.Scanln(&confirm)
	if strings.ToLower(confirm) != "y" {
		fmt.Println("Import cancelled")
		return
	}

	// Import configuration
	if err := cm.ImportConfig(path); err != nil {
		fmt.Printf("Error importing configuration: %v\n", err)
		return
	}

	fmt.Println("Configuration imported successfully")
	fmt.Println("Please restart Delta CLI to apply the changes")
}

// editConfiguration edits a specific configuration component
func editConfiguration(cm *ConfigManager, args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: :config edit <component> [setting=value]")
		fmt.Println("Available components: memory, ai, vector, embedding, inference, learning, token, agent")
		return
	}

	component := args[0]
	
	// If no settings are provided, show the component configuration
	if len(args) == 1 {
		showComponentConfig(cm, component)
		return
	}

	// Process setting=value pairs
	for i := 1; i < len(args); i++ {
		parts := strings.SplitN(args[i], "=", 2)
		if len(parts) != 2 {
			fmt.Printf("Invalid setting format: %s (should be setting=value)\n", args[i])
			continue
		}

		setting := parts[0]
		value := parts[1]

		// Update the appropriate component
		switch component {
		case "memory":
			updateMemoryConfigSetting(cm, setting, value)
		case "ai":
			updateAIConfig(cm, setting, value)
		case "vector":
			updateVectorConfigSetting(cm, setting, value)
		case "embedding":
			updateEmbeddingConfigSetting(cm, setting, value)
		case "inference":
			updateInferenceConfig(cm, setting, value)
		case "learning":
			updateLearningConfig(cm, setting, value)
		case "token":
			updateTokenConfig(cm, setting, value)
		case "agent":
			updateAgentConfig(cm, setting, value)
		default:
			fmt.Printf("Unknown component: %s\n", component)
			fmt.Println("Available components: memory, ai, vector, embedding, inference, learning, token, agent")
			return
		}
	}
}

// showComponentConfig displays the configuration for a specific component
func showComponentConfig(cm *ConfigManager, component string) {
	switch component {
	case "memory":
		if cm.memoryConfig != nil {
			data, _ := json.MarshalIndent(cm.memoryConfig, "", "  ")
			fmt.Printf("Memory Configuration:\n%s\n", string(data))
		} else {
			fmt.Println("Memory configuration not available")
		}
	case "ai":
		if cm.aiConfig != nil {
			data, _ := json.MarshalIndent(cm.aiConfig, "", "  ")
			fmt.Printf("AI Configuration:\n%s\n", string(data))
		} else {
			fmt.Println("AI configuration not available")
		}
	case "vector":
		if cm.vectorConfig != nil {
			data, _ := json.MarshalIndent(cm.vectorConfig, "", "  ")
			fmt.Printf("Vector Database Configuration:\n%s\n", string(data))
		} else {
			fmt.Println("Vector database configuration not available")
		}
	case "embedding":
		if cm.embeddingConfig != nil {
			data, _ := json.MarshalIndent(cm.embeddingConfig, "", "  ")
			fmt.Printf("Embedding Configuration:\n%s\n", string(data))
		} else {
			fmt.Println("Embedding configuration not available")
		}
	case "inference":
		if cm.inferenceConfig != nil {
			data, _ := json.MarshalIndent(cm.inferenceConfig, "", "  ")
			fmt.Printf("Inference Configuration:\n%s\n", string(data))
		} else {
			fmt.Println("Inference configuration not available")
		}
	case "learning":
		if cm.learningConfig != nil {
			data, _ := json.MarshalIndent(cm.learningConfig, "", "  ")
			fmt.Printf("Learning Configuration:\n%s\n", string(data))
		} else {
			fmt.Println("Learning configuration not available")
		}
	case "token":
		if cm.tokenConfig != nil {
			data, _ := json.MarshalIndent(cm.tokenConfig, "", "  ")
			fmt.Printf("Tokenizer Configuration:\n%s\n", string(data))
		} else {
			fmt.Println("Tokenizer configuration not available")
		}
	case "agent":
		if cm.agentConfig != nil {
			data, _ := json.MarshalIndent(cm.agentConfig, "", "  ")
			fmt.Printf("Agent Configuration:\n%s\n", string(data))
		} else {
			fmt.Println("Agent configuration not available")
		}
	default:
		fmt.Printf("Unknown component: %s\n", component)
		fmt.Println("Available components: memory, ai, vector, embedding, inference, learning, token, agent")
	}
}

// updateMemoryConfigSetting updates a specific memory configuration setting
func updateMemoryConfigSetting(cm *ConfigManager, setting, value string) {
	if cm.memoryConfig == nil {
		fmt.Println("Memory configuration not available")
		return
	}

	config := cm.memoryConfig

	switch setting {
	case "enabled":
		config.Enabled = (value == "true" || value == "1" || value == "yes")
	case "collect_commands":
		config.CollectCommands = (value == "true" || value == "1" || value == "yes")
	case "max_entries":
		var maxEntries int
		fmt.Sscanf(value, "%d", &maxEntries)
		if maxEntries > 0 {
			config.MaxEntries = maxEntries
		} else {
			fmt.Println("Error: max_entries must be a positive number")
			return
		}
	// Add more settings as needed
	default:
		fmt.Printf("Unknown memory setting: %s\n", setting)
		return
	}

	// Update the configuration
	if err := cm.UpdateMemoryConfig(config); err != nil {
		fmt.Printf("Error updating memory configuration: %v\n", err)
	} else {
		fmt.Printf("Updated memory.%s to %s\n", setting, value)
	}

	// Update the memory manager
	mm := GetMemoryManager()
	if mm != nil {
		mm.config = *config
		mm.saveConfig()
	}
}

// updateAIConfig updates a specific AI configuration setting
func updateAIConfig(cm *ConfigManager, setting, value string) {
	if cm.aiConfig == nil {
		fmt.Println("AI configuration not available")
		return
	}

	config := cm.aiConfig

	switch setting {
	case "enabled":
		config.Enabled = (value == "true" || value == "1" || value == "yes")
	case "model":
		config.ModelName = value
	// Add more settings as needed
	default:
		fmt.Printf("Unknown AI setting: %s\n", setting)
		return
	}

	// Update the configuration
	if err := cm.UpdateAIConfig(config); err != nil {
		fmt.Printf("Error updating AI configuration: %v\n", err)
	} else {
		fmt.Printf("Updated ai.%s to %s\n", setting, value)
	}

	// Update the AI manager
	ai := GetAIManager()
	if ai != nil {
		ai.config = *config
	}
}

// updateVectorConfig updates a specific vector configuration setting
func updateVectorConfigSetting(cm *ConfigManager, setting, value string) {
	if cm.vectorConfig == nil {
		fmt.Println("Vector configuration not available")
		return
	}

	config := cm.vectorConfig

	switch setting {
	case "enabled":
		config.Enabled = (value == "true" || value == "1" || value == "yes")
	case "in_memory_mode":
		config.InMemoryMode = (value == "true" || value == "1" || value == "yes")
	// Add more settings as needed
	default:
		fmt.Printf("Unknown vector setting: %s\n", setting)
		return
	}

	// Update the configuration
	if err := cm.UpdateVectorConfig(config); err != nil {
		fmt.Printf("Error updating vector configuration: %v\n", err)
	} else {
		fmt.Printf("Updated vector.%s to %s\n", setting, value)
	}

	// Update the vector manager
	vdb := GetVectorDBManager()
	if vdb != nil {
		vdb.config = *config
		vdb.saveConfig()
	}
}

// updateEmbeddingConfig updates a specific embedding configuration setting
func updateEmbeddingConfigSetting(cm *ConfigManager, setting, value string) {
	if cm.embeddingConfig == nil {
		fmt.Println("Embedding configuration not available")
		return
	}

	config := cm.embeddingConfig

	switch setting {
	case "enabled":
		config.Enabled = (value == "true" || value == "1" || value == "yes")
	case "dimensions":
		var dimensions int
		fmt.Sscanf(value, "%d", &dimensions)
		if dimensions > 0 {
			config.Dimensions = dimensions
		} else {
			fmt.Println("Error: dimensions must be a positive number")
			return
		}
	// Add more settings as needed
	default:
		fmt.Printf("Unknown embedding setting: %s\n", setting)
		return
	}

	// Update the configuration
	if err := cm.UpdateEmbeddingConfig(config); err != nil {
		fmt.Printf("Error updating embedding configuration: %v\n", err)
	} else {
		fmt.Printf("Updated embedding.%s to %s\n", setting, value)
	}

	// Update the embedding manager
	em := GetEmbeddingManager()
	if em != nil {
		em.config = *config
		em.saveConfig()
	}
}

// updateInferenceConfig updates a specific inference configuration setting
func updateInferenceConfig(cm *ConfigManager, setting, value string) {
	if cm.inferenceConfig == nil {
		fmt.Println("Inference configuration not available")
		return
	}

	config := cm.inferenceConfig

	switch setting {
	case "enabled":
		config.Enabled = (value == "true" || value == "1" || value == "yes")
	case "use_local_inference":
		config.UseLocalInference = (value == "true" || value == "1" || value == "yes")
	case "max_tokens":
		var maxTokens int
		fmt.Sscanf(value, "%d", &maxTokens)
		if maxTokens > 0 {
			config.MaxTokens = maxTokens
		} else {
			fmt.Println("Error: max_tokens must be a positive number")
			return
		}
	case "temperature":
		var temperature float64
		fmt.Sscanf(value, "%f", &temperature)
		if temperature >= 0 && temperature <= 1 {
			config.Temperature = temperature
		} else {
			fmt.Println("Error: temperature must be between 0 and 1")
			return
		}
	// Add more settings as needed
	default:
		fmt.Printf("Unknown inference setting: %s\n", setting)
		return
	}

	// Update the configuration
	if err := cm.UpdateInferenceConfig(config); err != nil {
		fmt.Printf("Error updating inference configuration: %v\n", err)
	} else {
		fmt.Printf("Updated inference.%s to %s\n", setting, value)
	}

	// Update the inference manager
	im := GetInferenceManager()
	if im != nil {
		im.inferenceConfig = *config
		im.saveConfig()
	}
}

// updateLearningConfig updates a specific learning configuration setting
func updateLearningConfig(cm *ConfigManager, setting, value string) {
	if cm.learningConfig == nil {
		fmt.Println("Learning configuration not available")
		return
	}

	config := cm.learningConfig

	switch setting {
	case "collect_feedback":
		config.CollectFeedback = (value == "true" || value == "1" || value == "yes")
	case "use_custom_model":
		config.UseCustomModel = (value == "true" || value == "1" || value == "yes")
	case "custom_model_path":
		config.CustomModelPath = value
	// Add more settings as needed
	default:
		fmt.Printf("Unknown learning setting: %s\n", setting)
		return
	}

	// Update the configuration
	if err := cm.UpdateLearningConfig(config); err != nil {
		fmt.Printf("Error updating learning configuration: %v\n", err)
	} else {
		fmt.Printf("Updated learning.%s to %s\n", setting, value)
	}

	// Update the inference manager
	im := GetInferenceManager()
	if im != nil {
		im.learningConfig = *config
		im.saveConfig()
	}
}

// updateTokenConfig updates a specific tokenizer configuration setting
func updateTokenConfig(cm *ConfigManager, setting, value string) {
	if cm.tokenConfig == nil {
		fmt.Println("Tokenizer configuration not available")
		return
	}

	config := cm.tokenConfig

	switch setting {
	case "enabled":
		config.Enabled = (value == "true" || value == "1" || value == "yes")
	case "vocab_size":
		var vocabSize int
		fmt.Sscanf(value, "%d", &vocabSize)
		if vocabSize > 0 {
			config.VocabSize = vocabSize
		} else {
			fmt.Println("Error: vocab_size must be a positive number")
			return
		}
	// Add more settings as needed
	default:
		fmt.Printf("Unknown tokenizer setting: %s\n", setting)
		return
	}

	// Update the configuration
	if err := cm.UpdateTokenConfig(config); err != nil {
		fmt.Printf("Error updating tokenizer configuration: %v\n", err)
	} else {
		fmt.Printf("Updated token.%s to %s\n", setting, value)
	}

	// Update the tokenizer
	tk := GetTokenizer()
	if tk != nil {
		tk.Config = *config
		// No save config for tokenizer
	}
}

// updateAgentConfig updates a specific agent configuration setting
func updateAgentConfig(cm *ConfigManager, setting, value string) {
	if cm.agentConfig == nil {
		fmt.Println("Agent configuration not available")
		return
	}

	config := cm.agentConfig

	switch setting {
	case "enabled":
		config.Enabled = (value == "true" || value == "1" || value == "yes")
	case "use_docker_builds":
		config.UseDockerBuilds = (value == "true" || value == "1" || value == "yes")
	case "use_ai_assistance":
		config.UseAIAssistance = (value == "true" || value == "1" || value == "yes")
	// Add more settings as needed
	default:
		fmt.Printf("Unknown agent setting: %s\n", setting)
		return
	}

	// Update the configuration
	if err := cm.UpdateAgentConfig(config); err != nil {
		fmt.Printf("Error updating agent configuration: %v\n", err)
	} else {
		fmt.Printf("Updated agent.%s to %s\n", setting, value)
	}

	// Update the agent manager
	am := GetAgentManager()
	if am != nil {
		am.config = *config
		am.saveConfig()
	}
}

// resetConfiguration resets all configurations to their default values
func resetConfiguration(cm *ConfigManager) {
	// Confirm reset
	fmt.Println("Warning: This will reset ALL configurations to their default values.")
	fmt.Print("Do you want to continue? (y/n): ")
	var confirm string
	fmt.Scanln(&confirm)
	if strings.ToLower(confirm) != "y" {
		fmt.Println("Reset cancelled")
		return
	}

	// Remove the configuration file
	if err := os.Remove(cm.configPath); err != nil && !os.IsNotExist(err) {
		fmt.Printf("Error removing configuration file: %v\n", err)
		return
	}

	fmt.Println("Configuration reset to default values")
	fmt.Println("Please restart Delta CLI to apply the changes")
}

// listEnvironmentVariables displays all active Delta CLI environment variables
func listEnvironmentVariables() {
	vars := listDeltaEnvVars()
	
	if len(vars) == 0 {
		fmt.Println("No Delta CLI environment variables are currently set.")
		fmt.Println("Use environment variables like DELTA_MEMORY_ENABLED=true to configure Delta CLI.")
		fmt.Println("Run ':config env help' for more information.")
		return
	}
	
	fmt.Println("Active Delta CLI Environment Variables:")
	fmt.Println("=======================================")
	
	// Sort keys for consistent output
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	
	for _, k := range keys {
		fmt.Printf("%s=%s\n", k, vars[k])
	}
	
	fmt.Println("\nNote: Environment variables take precedence over configuration file settings.")
}

// clearEnvironmentVariables provides instructions on clearing environment variables
func clearEnvironmentVariables() {
	fmt.Println("Environment variables cannot be unset by Delta CLI.")
	fmt.Println("To clear environment variables, use your shell's commands:")
	fmt.Println("")
	fmt.Println("For Bash/Zsh:")
	fmt.Println("  unset DELTA_MEMORY_ENABLED")
	fmt.Println("")
	fmt.Println("For Fish:")
	fmt.Println("  set -e DELTA_MEMORY_ENABLED")
	fmt.Println("")
	fmt.Println("For PowerShell:")
	fmt.Println("  $env:DELTA_MEMORY_ENABLED = $null")
}

// showEnvironmentVariableHelp displays help for environment variable usage
func showEnvironmentVariableHelp() {
	fmt.Println("Delta CLI Environment Variables")
	fmt.Println("==============================")
	fmt.Println("Environment variables can be used to override configuration settings.")
	fmt.Println("They have higher priority than settings in the configuration file.")
	fmt.Println("")
	fmt.Println("Environment variables use the format DELTA_<COMPONENT>_<SETTING>")
	fmt.Println("")
	fmt.Println("Available Environment Variables:")
	
	// The variables should be grouped by component for better readability
	fmt.Println("\nMemory Configuration:")
	fmt.Println("  DELTA_MEMORY_ENABLED               - Enable/disable memory system (true/false)")
	fmt.Println("  DELTA_MEMORY_COLLECT_COMMANDS      - Enable/disable command collection (true/false)")
	fmt.Println("  DELTA_MEMORY_MAX_ENTRIES           - Maximum number of memory entries (integer)")
	fmt.Println("  DELTA_MEMORY_STORAGE_PATH          - Path to store memory data (string)")
	
	fmt.Println("\nAI Configuration:")
	fmt.Println("  DELTA_AI_ENABLED                   - Enable/disable AI features (true/false)")
	fmt.Println("  DELTA_AI_MODEL                     - AI model name (string)")
	fmt.Println("  DELTA_AI_SERVER_URL                - AI server URL (string)")
	
	fmt.Println("\nVector Database Configuration:")
	fmt.Println("  DELTA_VECTOR_ENABLED               - Enable/disable vector database (true/false)")
	fmt.Println("  DELTA_VECTOR_DB_PATH               - Database file path (string)")
	fmt.Println("  DELTA_VECTOR_IN_MEMORY_MODE        - Use in-memory mode (true/false)")
	
	fmt.Println("\nEmbedding Configuration:")
	fmt.Println("  DELTA_EMBEDDING_ENABLED            - Enable/disable embedding system (true/false)")
	fmt.Println("  DELTA_EMBEDDING_DIMENSIONS         - Embedding dimensions (integer)")
	fmt.Println("  DELTA_EMBEDDING_CACHE_SIZE         - Embedding cache size (integer)")
	
	fmt.Println("\nInference Configuration:")
	fmt.Println("  DELTA_INFERENCE_ENABLED            - Enable/disable inference system (true/false)")
	fmt.Println("  DELTA_INFERENCE_USE_LOCAL          - Use local inference (true/false)")
	fmt.Println("  DELTA_INFERENCE_MAX_TOKENS         - Maximum tokens for generation (integer)")
	fmt.Println("  DELTA_INFERENCE_TEMPERATURE        - Sampling temperature (float 0-1)")
	
	fmt.Println("\nLearning Configuration:")
	fmt.Println("  DELTA_LEARNING_COLLECT_FEEDBACK    - Collect feedback for learning (true/false)")
	fmt.Println("  DELTA_LEARNING_USE_CUSTOM_MODEL    - Use custom model (true/false)")
	fmt.Println("  DELTA_LEARNING_CUSTOM_MODEL_PATH   - Path to custom model (string)")
	fmt.Println("  DELTA_LEARNING_TRAINING_THRESHOLD  - Training threshold (integer)")
	
	fmt.Println("\nTokenizer Configuration:")
	fmt.Println("  DELTA_TOKEN_ENABLED                - Enable/disable tokenizer (true/false)")
	fmt.Println("  DELTA_TOKEN_VOCAB_SIZE             - Vocabulary size (integer)")
	fmt.Println("  DELTA_TOKEN_STORAGE_PATH           - Token storage path (string)")
	
	fmt.Println("\nAgent Configuration:")
	fmt.Println("  DELTA_AGENT_ENABLED                - Enable/disable agent system (true/false)")
	fmt.Println("  DELTA_AGENT_STORAGE_PATH           - Agent storage path (string)")
	fmt.Println("  DELTA_AGENT_CACHE_PATH             - Agent cache path (string)")
	fmt.Println("  DELTA_AGENT_USE_DOCKER             - Use Docker for agent builds (true/false)")
	fmt.Println("  DELTA_AGENT_USE_AI                 - Use AI assistance for agents (true/false)")
	
	fmt.Println("\nCommands:")
	fmt.Println("  :config env list                   - List all active environment variables")
	fmt.Println("  :config env help                   - Show this help message")
	
	fmt.Println("\nExample Usage:")
	fmt.Println("  export DELTA_MEMORY_ENABLED=true         # Enable memory system")
	fmt.Println("  export DELTA_INFERENCE_TEMPERATURE=0.7   # Set inference temperature")
}

// showConfigHelp displays help for configuration commands
func showConfigHelp() {
	fmt.Println("Configuration Commands")
	fmt.Println("=====================")
	fmt.Println("  :config                - Show configuration status")
	fmt.Println("  :config status         - Show configuration status")
	fmt.Println("  :config list           - List all configurations")
	fmt.Println("  :config export <path>  - Export configuration to a file")
	fmt.Println("  :config import <path>  - Import configuration from a file")
	fmt.Println("  :config edit <comp>    - Show component configuration")
	fmt.Println("  :config edit <comp> setting=value - Update configuration setting")
	fmt.Println("  :config reset          - Reset all configurations to default values")
	fmt.Println("  :config env            - List active environment variables")
	fmt.Println("  :config env list       - List active environment variables")
	fmt.Println("  :config env help       - Show environment variable help")
	fmt.Println("  :config help           - Show this help message")
	fmt.Println()
	fmt.Println("Available components:")
	fmt.Println("  memory    - Memory system configuration")
	fmt.Println("  ai        - AI prediction configuration")
	fmt.Println("  vector    - Vector database configuration")
	fmt.Println("  embedding - Embedding system configuration")
	fmt.Println("  inference - Inference system configuration")
	fmt.Println("  learning  - Learning system configuration")
	fmt.Println("  token     - Tokenizer configuration")
	fmt.Println("  agent     - Agent system configuration")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  :config edit memory enabled=true")
	fmt.Println("  :config edit ai model=phi3:latest")
	fmt.Println("  :config edit inference temperature=0.7")
	fmt.Println("  :config export ~/delta_config_backup.json")
}