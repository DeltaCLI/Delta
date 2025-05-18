package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// AgentConfigManager manages loading and saving agent configurations
type AgentConfigManager struct {
	configDir string
}

// NewAgentConfigManager creates a new agent config manager
func NewAgentConfigManager(configDir string) *AgentConfigManager {
	return &AgentConfigManager{
		configDir: configDir,
	}
}

// LoadAgentConfig loads an agent configuration from a YAML file
func (cm *AgentConfigManager) LoadAgentConfig(filePath string) (*Agent, error) {
	// Read YAML file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read agent config file: %v", err)
	}

	// Parse YAML
	var yamlConfig AgentYAMLConfig
	err = yaml.Unmarshal(data, &yamlConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse agent config: %v", err)
	}

	// Validate version
	if yamlConfig.Version == "" {
		return nil, fmt.Errorf("missing version in agent config")
	}

	// Create agent from YAML config
	agent, err := convertYAMLToAgent(yamlConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to convert YAML to agent: %v", err)
	}

	return agent, nil
}

// SaveAgentConfig saves an agent configuration to a YAML file
func (cm *AgentConfigManager) SaveAgentConfig(agent *Agent, filePath string) error {
	// Convert agent to YAML config
	yamlConfig, err := convertAgentToYAML(agent)
	if err != nil {
		return fmt.Errorf("failed to convert agent to YAML: %v", err)
	}

	// Marshal to YAML
	data, err := yaml.Marshal(yamlConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal agent config: %v", err)
	}

	// Write to file
	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write agent config file: %v", err)
	}

	return nil
}

// LoadAgentConfigs loads all agent configurations from a directory
func (cm *AgentConfigManager) LoadAgentConfigs() ([]*Agent, error) {
	// Check if config directory exists
	_, err := os.Stat(cm.configDir)
	if os.IsNotExist(err) {
		// Create config directory
		err = os.MkdirAll(cm.configDir, 0755)
		if err != nil {
			return nil, fmt.Errorf("failed to create config directory: %v", err)
		}
		return []*Agent{}, nil
	}

	// Read all YAML files in directory
	entries, err := os.ReadDir(cm.configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read config directory: %v", err)
	}

	// Load each config file
	var agents []*Agent
	for _, entry := range entries {
		if entry.IsDir() || (!strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml")) {
			continue
		}

		// Load agent config
		filePath := filepath.Join(cm.configDir, entry.Name())
		agent, err := cm.LoadAgentConfig(filePath)
		if err != nil {
			fmt.Printf("Warning: failed to load agent config from %s: %v\n", filePath, err)
			continue
		}

		agents = append(agents, agent)
	}

	return agents, nil
}

// convertYAMLToAgent converts a YAML configuration to an Agent
func convertYAMLToAgent(yamlConfig AgentYAMLConfig) (*Agent, error) {
	if len(yamlConfig.Agents) == 0 {
		return nil, fmt.Errorf("no agents defined in YAML config")
	}

	// Use the first agent defined in the YAML
	agentYAML := yamlConfig.Agents[0]

	// Create agent
	agent := &Agent{
		ID:              agentYAML.ID,
		Name:            agentYAML.Name,
		Description:     agentYAML.Description,
		TaskTypes:       agentYAML.TaskTypes,
		TriggerPatterns: agentYAML.Triggers.Patterns,
		Context:         agentYAML.Metadata,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		LastRunAt:       time.Time{},
		RunCount:        0,
		SuccessRate:     0,
		Tags:            []string{},
		Enabled:         agentYAML.Enabled,
	}

	// Convert commands
	agent.Commands = []AgentCommand{}
	for _, cmdYAML := range agentYAML.Commands {
		cmd := AgentCommand{
			ID:              cmdYAML.ID,
			Command:         cmdYAML.Command,
			WorkingDir:      cmdYAML.WorkingDir,
			ErrorPatterns:   cmdYAML.ErrorPatterns,
			SuccessPatterns: cmdYAML.SuccessPatterns,
			Timeout:         cmdYAML.Timeout,
			RetryCount:      cmdYAML.RetryCount,
			IsInteractive:   cmdYAML.IsInteractive,
			Environment:     cmdYAML.Environment,
			Enabled:         true,
		}
		agent.Commands = append(agent.Commands, cmd)
	}

	// Convert Docker config if present
	if agentYAML.Docker.Enabled {
		agent.DockerConfig = &AgentDockerConfig{
			Enabled:      true,
			Image:        agentYAML.Docker.Image,
			Tag:          agentYAML.Docker.Tag,
			BuildContext: "",
			Dockerfile:   agentYAML.Docker.Dockerfile,
			ComposeFile:  agentYAML.Docker.ComposeFile,
			Volumes:      agentYAML.Docker.Volumes,
			Networks:     agentYAML.Docker.Networks,
			Environment:  agentYAML.Docker.Environment,
			BuildArgs:    agentYAML.Docker.BuildArgs,
			UseCache:     agentYAML.Docker.UseCache,
		}

		// Convert waterfall config if present
		if len(agentYAML.Docker.Waterfall.Stages) > 0 {
			agent.DockerConfig.Waterfall = &WaterfallConfig{
				Stages:       agentYAML.Docker.Waterfall.Stages,
				Dependencies: agentYAML.Docker.Waterfall.Dependencies,
				ProjectName:  yamlConfig.Project.Name,
			}
		}
	}

	return agent, nil
}

// convertAgentToYAML converts an Agent to a YAML configuration
func convertAgentToYAML(agent *Agent) (AgentYAMLConfig, error) {
	// Create YAML config
	yamlConfig := AgentYAMLConfig{
		Version: "1.0",
		Project: struct {
			Name        string "yaml:\"name\""
			Repository  string "yaml:\"repository\""
			Description string "yaml:\"description\""
			Website     string "yaml:\"website,omitempty\""
			Docs        string "yaml:\"docs,omitempty\""
		}{
			Name:        agent.Name,
			Description: agent.Description,
		},
		Agents: []struct {
			ID          string   "yaml:\"id\""
			Name        string   "yaml:\"name\""
			Description string   "yaml:\"description\""
			Enabled     bool     "yaml:\"enabled\""
			TaskTypes   []string "yaml:\"task_types\""
			Import      string   "yaml:\"import,omitempty\""
			Commands    []struct {
				ID              string            "yaml:\"id\""
				Command         string            "yaml:\"command\""
				WorkingDir      string            "yaml:\"working_dir\""
				Timeout         int               "yaml:\"timeout,omitempty\""
				RetryCount      int               "yaml:\"retry_count,omitempty\""
				ErrorPatterns   []string          "yaml:\"error_patterns,omitempty\""
				SuccessPatterns []string          "yaml:\"success_patterns,omitempty\""
				IsInteractive   bool              "yaml:\"is_interactive,omitempty\""
				Environment     map[string]string "yaml:\"environment,omitempty\""
			} "yaml:\"commands,omitempty\""
			Docker struct {
				Enabled      bool              "yaml:\"enabled\""
				Image        string            "yaml:\"image,omitempty\""
				Tag          string            "yaml:\"tag,omitempty\""
				Dockerfile   string            "yaml:\"dockerfile,omitempty\""
				ComposeFile  string            "yaml:\"compose_file,omitempty\""
				Volumes      []string          "yaml:\"volumes,omitempty\""
				Networks     []string          "yaml:\"networks,omitempty\""
				Environment  map[string]string "yaml:\"environment,omitempty\""
				BuildArgs    map[string]string "yaml:\"build_args,omitempty\""
				UseCache     bool              "yaml:\"use_cache\""
				Waterfall    struct {
					Stages       []string              "yaml:\"stages\""
					Dependencies map[string][]string   "yaml:\"dependencies\""
				} "yaml:\"waterfall,omitempty\""
			} "yaml:\"docker,omitempty\""
			ErrorHandling struct {
				AutoFix  bool "yaml:\"auto_fix\""
				Patterns []struct {
					Pattern     string "yaml:\"pattern\""
					Solution    string "yaml:\"solution\""
					Description string "yaml:\"description,omitempty\""
					FilePattern string "yaml:\"file_pattern,omitempty\""
				} "yaml:\"patterns,omitempty\""
			} "yaml:\"error_handling,omitempty\""
			Metadata map[string]string "yaml:\"metadata,omitempty\""
			Triggers struct {
				Patterns  []string "yaml:\"patterns,omitempty\""
				Paths     []string "yaml:\"paths,omitempty\""
				Schedules []string "yaml:\"schedules,omitempty\""
				Events    []string "yaml:\"events,omitempty\""
			} "yaml:\"triggers,omitempty\""
		}{
			{
				ID:          agent.ID,
				Name:        agent.Name,
				Description: agent.Description,
				Enabled:     agent.Enabled,
				TaskTypes:   agent.TaskTypes,
				Metadata:    agent.Context,
			},
		},
	}

	// Convert commands
	yamlCommands := []struct {
		ID              string            "yaml:\"id\""
		Command         string            "yaml:\"command\""
		WorkingDir      string            "yaml:\"working_dir\""
		Timeout         int               "yaml:\"timeout,omitempty\""
		RetryCount      int               "yaml:\"retry_count,omitempty\""
		ErrorPatterns   []string          "yaml:\"error_patterns,omitempty\""
		SuccessPatterns []string          "yaml:\"success_patterns,omitempty\""
		IsInteractive   bool              "yaml:\"is_interactive,omitempty\""
		Environment     map[string]string "yaml:\"environment,omitempty\""
	}{}

	for _, cmd := range agent.Commands {
		yamlCmd := struct {
			ID              string            "yaml:\"id\""
			Command         string            "yaml:\"command\""
			WorkingDir      string            "yaml:\"working_dir\""
			Timeout         int               "yaml:\"timeout,omitempty\""
			RetryCount      int               "yaml:\"retry_count,omitempty\""
			ErrorPatterns   []string          "yaml:\"error_patterns,omitempty\""
			SuccessPatterns []string          "yaml:\"success_patterns,omitempty\""
			IsInteractive   bool              "yaml:\"is_interactive,omitempty\""
			Environment     map[string]string "yaml:\"environment,omitempty\""
		}{
			ID:              cmd.ID,
			Command:         cmd.Command,
			WorkingDir:      cmd.WorkingDir,
			Timeout:         cmd.Timeout,
			RetryCount:      cmd.RetryCount,
			ErrorPatterns:   cmd.ErrorPatterns,
			SuccessPatterns: cmd.SuccessPatterns,
			IsInteractive:   cmd.IsInteractive,
			Environment:     cmd.Environment,
		}
		yamlCommands = append(yamlCommands, yamlCmd)
	}
	yamlConfig.Agents[0].Commands = yamlCommands

	// Convert Docker config if present
	if agent.DockerConfig != nil {
		yamlConfig.Agents[0].Docker.Enabled = agent.DockerConfig.Enabled
		yamlConfig.Agents[0].Docker.Image = agent.DockerConfig.Image
		yamlConfig.Agents[0].Docker.Tag = agent.DockerConfig.Tag
		yamlConfig.Agents[0].Docker.Dockerfile = agent.DockerConfig.Dockerfile
		yamlConfig.Agents[0].Docker.ComposeFile = agent.DockerConfig.ComposeFile
		yamlConfig.Agents[0].Docker.Volumes = agent.DockerConfig.Volumes
		yamlConfig.Agents[0].Docker.Networks = agent.DockerConfig.Networks
		yamlConfig.Agents[0].Docker.Environment = agent.DockerConfig.Environment
		yamlConfig.Agents[0].Docker.BuildArgs = agent.DockerConfig.BuildArgs
		yamlConfig.Agents[0].Docker.UseCache = agent.DockerConfig.UseCache

		// Convert waterfall config if present
		if agent.DockerConfig.Waterfall != nil {
			yamlConfig.Agents[0].Docker.Waterfall.Stages = agent.DockerConfig.Waterfall.Stages
			yamlConfig.Agents[0].Docker.Waterfall.Dependencies = agent.DockerConfig.Waterfall.Dependencies
		}
	}

	// Convert triggers
	yamlConfig.Agents[0].Triggers.Patterns = agent.TriggerPatterns

	return yamlConfig, nil
}

// CreateAgentFromTemplate creates a new agent from a template file
func (cm *AgentConfigManager) CreateAgentFromTemplate(templatePath string, agentID string, replacements map[string]string) (*Agent, error) {
	// Load template
	agent, err := cm.LoadAgentConfig(templatePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load template: %v", err)
	}

	// Update agent ID
	agent.ID = agentID

	// Apply replacements
	agent.Name = replaceVariables(agent.Name, replacements)
	agent.Description = replaceVariables(agent.Description, replacements)

	// Replace context variables
	for k, v := range agent.Context {
		agent.Context[k] = replaceVariables(v, replacements)
	}

	// Replace command variables
	for i, cmd := range agent.Commands {
		agent.Commands[i].Command = replaceVariables(cmd.Command, replacements)
		agent.Commands[i].WorkingDir = replaceVariables(cmd.WorkingDir, replacements)
	}

	// Replace Docker config variables
	if agent.DockerConfig != nil {
		agent.DockerConfig.Image = replaceVariables(agent.DockerConfig.Image, replacements)
		agent.DockerConfig.BuildContext = replaceVariables(agent.DockerConfig.BuildContext, replacements)
		agent.DockerConfig.Dockerfile = replaceVariables(agent.DockerConfig.Dockerfile, replacements)
		agent.DockerConfig.ComposeFile = replaceVariables(agent.DockerConfig.ComposeFile, replacements)

		// Replace volume paths
		for i, volume := range agent.DockerConfig.Volumes {
			agent.DockerConfig.Volumes[i] = replaceVariables(volume, replacements)
		}

		// Replace build args
		for k, v := range agent.DockerConfig.BuildArgs {
			agent.DockerConfig.BuildArgs[k] = replaceVariables(v, replacements)
		}
	}

	return agent, nil
}

// replaceVariables replaces ${VAR} variables in a string with their values
func replaceVariables(input string, replacements map[string]string) string {
	result := input
	for k, v := range replacements {
		result = strings.ReplaceAll(result, "${"+k+"}", v)
	}
	return result
}

// LoadAgentYAML loads an agent YAML file
func LoadAgentYAML(filePath string) (*AgentYAMLConfig, error) {
	// Read YAML file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read agent YAML file: %v", err)
	}

	// Parse YAML
	var yamlConfig AgentYAMLConfig
	err = yaml.Unmarshal(data, &yamlConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse agent YAML: %v", err)
	}

	return &yamlConfig, nil
}

// SaveAgentYAML saves an agent YAML file
func SaveAgentYAML(yamlConfig *AgentYAMLConfig, filePath string) error {
	// Marshal to YAML
	data, err := yaml.Marshal(yamlConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal agent YAML: %v", err)
	}

	// Write to file
	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write agent YAML file: %v", err)
	}

	return nil
}

// GetConfigManager returns a new agent config manager
func GetConfigManager() *AgentConfigManager {
	// Set up environment
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Failed to get home directory: %v\n", err)
		return nil
	}

	// Set up config directory
	configDir := filepath.Join(homeDir, ".config", "delta", "agents")
	return NewAgentConfigManager(configDir)
}