package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

//go:embed embedded_patterns/*
var embeddedPatterns embed.FS

// AgentCommand is defined in agent_types.go

// AgentDockerConfig is defined in agent_types.go

// Agent struct is defined in agent_types.go

// BuildCacheConfig represents cache configuration for a specific build
// BuildCacheConfig is defined in agent_types.go

// AgentManagerConfig contains configuration for the agent manager
// AgentManagerConfig is defined in agent_types.go
// AgentManager is defined in agent_types.go

// AgentRunResult represents the result of an agent run
// AgentRunResult is defined in agent_types.go

// DockerBuildCache manages build caching for Docker-based agents
// DockerBuildCache is defined in agent_types.go

// AgentManager handles the creation, execution, and management of agents
// AgentManager is defined in agent_types.go

// AgentYAMLConfig represents the YAML configuration for an agent
type AgentYAMLConfig struct {
	Version string `yaml:"version"`
	Project struct {
		Name        string `yaml:"name"`
		Repository  string `yaml:"repository"`
		Description string `yaml:"description"`
		Website     string `yaml:"website,omitempty"`
		Docs        string `yaml:"docs,omitempty"`
	} `yaml:"project"`

	Settings struct {
		Docker struct {
			Enabled     bool              `yaml:"enabled"`
			CacheDir    string            `yaml:"cache_dir,omitempty"`
			Volumes     []string          `yaml:"volumes,omitempty"`
			Environment map[string]string `yaml:"environment,omitempty"`
		} `yaml:"docker,omitempty"`

		ErrorHandling struct {
			Patterns []struct {
				Pattern     string `yaml:"pattern"`
				Solution    string `yaml:"solution"`
				Description string `yaml:"description,omitempty"`
				FilePattern string `yaml:"file_pattern,omitempty"`
			} `yaml:"patterns,omitempty"`
		} `yaml:"error_handling,omitempty"`
	} `yaml:"settings,omitempty"`

	Agents []struct {
		ID          string   `yaml:"id"`
		Name        string   `yaml:"name"`
		Description string   `yaml:"description"`
		Enabled     bool     `yaml:"enabled"`
		TaskTypes   []string `yaml:"task_types"`
		Import      string   `yaml:"import,omitempty"`

		Commands []struct {
			ID              string            `yaml:"id"`
			Command         string            `yaml:"command"`
			WorkingDir      string            `yaml:"working_dir"`
			Timeout         int               `yaml:"timeout,omitempty"`
			RetryCount      int               `yaml:"retry_count,omitempty"`
			ErrorPatterns   []string          `yaml:"error_patterns,omitempty"`
			SuccessPatterns []string          `yaml:"success_patterns,omitempty"`
			IsInteractive   bool              `yaml:"is_interactive,omitempty"`
			Environment     map[string]string `yaml:"environment,omitempty"`
		} `yaml:"commands,omitempty"`

		Docker struct {
			Enabled     bool              `yaml:"enabled"`
			Image       string            `yaml:"image,omitempty"`
			Tag         string            `yaml:"tag,omitempty"`
			Dockerfile  string            `yaml:"dockerfile,omitempty"`
			ComposeFile string            `yaml:"compose_file,omitempty"`
			Volumes     []string          `yaml:"volumes,omitempty"`
			Networks    []string          `yaml:"networks,omitempty"`
			Environment map[string]string `yaml:"environment,omitempty"`
			BuildArgs   map[string]string `yaml:"build_args,omitempty"`
			UseCache    bool              `yaml:"use_cache"`

			Waterfall struct {
				Stages       []string            `yaml:"stages"`
				Dependencies map[string][]string `yaml:"dependencies"`
			} `yaml:"waterfall,omitempty"`
		} `yaml:"docker,omitempty"`

		ErrorHandling struct {
			AutoFix  bool `yaml:"auto_fix"`
			Patterns []struct {
				Pattern     string `yaml:"pattern"`
				Solution    string `yaml:"solution"`
				Description string `yaml:"description,omitempty"`
				FilePattern string `yaml:"file_pattern,omitempty"`
			} `yaml:"patterns,omitempty"`
		} `yaml:"error_handling,omitempty"`

		Metadata map[string]string `yaml:"metadata,omitempty"`

		Triggers struct {
			Patterns  []string `yaml:"patterns,omitempty"`
			Paths     []string `yaml:"paths,omitempty"`
			Schedules []string `yaml:"schedules,omitempty"`
			Events    []string `yaml:"events,omitempty"`
		} `yaml:"triggers,omitempty"`
	} `yaml:"agents"`
}

// NewAgentManager creates a new agent manager
func NewAgentManager() (*AgentManager, error) {
	// Set up config directory in user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = os.Getenv("HOME")
	}

	// Use ~/.config/delta/agents directory
	configDir := filepath.Join(homeDir, ".config", "delta", "agents")
	err = os.MkdirAll(configDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent config directory: %v", err)
	}

	configPath := filepath.Join(configDir, "agent_config.json")
	agentStoragePath := filepath.Join(configDir, "agents")
	cacheStoragePath := filepath.Join(configDir, "cache")

	// Create default agent manager
	manager := &AgentManager{
		config: AgentManagerConfig{
			Enabled:           false,
			AgentStoragePath:  agentStoragePath,
			CacheStoragePath:  cacheStoragePath,
			MaxCacheSize:      10 * 1024 * 1024 * 1024, // 10GB
			CacheRetention:    30,                      // 30 days
			MaxAgentRuns:      100,
			DefaultTimeout:    3600, // 1 hour
			DefaultRetryCount: 3,
			DefaultRetryDelay: 10, // 10 seconds
			UseDockerBuilds:   true,
			UseAIAssistance:   true,
			AIPromptTemplate:  "You are a build assistant for the %s agent. Your task is to %s.",
		},
		configPath:    configPath,
		agents:        make(map[string]*Agent),
		runHistory:    []AgentRunResult{},
		mutex:         sync.RWMutex{},
		isInitialized: false,
	}

	// Initialize the Docker build cache
	manager.dockerCache = &DockerBuildCache{
		CacheDir:     cacheStoragePath,
		CacheSize:    0,
		MaxCacheAge:  time.Duration(manager.config.CacheRetention) * 24 * time.Hour,
		BuildConfigs: make(map[string]*BuildCacheConfig),
	}

	// Try to load existing configuration
	err = manager.loadConfig()
	if err != nil {
		// Save the default configuration if loading fails
		manager.saveConfig()
	}

	return manager, nil
}

// Initialize initializes the agent manager
func (am *AgentManager) Initialize() error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	// Create agent storage directory if it doesn't exist
	err := os.MkdirAll(am.config.AgentStoragePath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create agent storage directory: %v", err)
	}

	// Create cache storage directory if it doesn't exist
	err = os.MkdirAll(am.config.CacheStoragePath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create cache storage directory: %v", err)
	}

	// Install default patterns if they don't exist
	err = am.installDefaultPatterns()
	if err != nil {
		fmt.Printf("Warning: Failed to install default patterns: %v\n", err)
		// Continue anyway
	}

	// Load existing agents
	err = am.loadAgents()
	if err != nil {
		fmt.Printf("Warning: Failed to load agents: %v\n", err)
		// Continue anyway with empty agents
	}

	// Load build cache configuration
	err = am.loadBuildCache()
	if err != nil {
		fmt.Printf("Warning: Failed to load build cache: %v\n", err)
		// Continue anyway with empty cache
	}

	// Initialize knowledge extractor
	am.knowledgeExtractor = GetKnowledgeExtractor()
	if am.knowledgeExtractor == nil {
		fmt.Printf("Warning: Failed to initialize knowledge extractor\n")
	}

	// Initialize AI manager
	am.aiManager = GetAIManager()
	if am.aiManager == nil {
		fmt.Printf("Warning: Failed to initialize AI manager\n")
	}

	// Initialize error learning manager
	errorLearningMgr := GetErrorLearningManager()
	if errorLearningMgr == nil {
		fmt.Printf("Warning: Failed to initialize error learning manager\n")
	}

	am.isInitialized = true
	return nil
}

// loadConfig loads the agent manager configuration from disk
func (am *AgentManager) loadConfig() error {
	// Check if config file exists
	_, err := os.Stat(am.configPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("config file does not exist")
	}

	// Read the config file
	data, err := os.ReadFile(am.configPath)
	if err != nil {
		return err
	}

	// Unmarshal the JSON data
	return json.Unmarshal(data, &am.config)
}

// saveConfig saves the agent manager configuration to disk
func (am *AgentManager) saveConfig() error {
	// Marshal the config to JSON with indentation for readability
	data, err := json.MarshalIndent(am.config, "", "  ")
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(am.configPath)
	if err = os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(am.configPath, data, 0644)
}

// loadAgents loads all available agents from disk
func (am *AgentManager) loadAgents() error {
	// Check if agent storage directory exists
	_, err := os.Stat(am.config.AgentStoragePath)
	if os.IsNotExist(err) {
		return nil // No agents to load
	}

	// Get list of agent files
	files, err := os.ReadDir(am.config.AgentStoragePath)
	if err != nil {
		return err
	}

	// Load each agent file
	for _, file := range files {
		if file.IsDir() || filepath.Ext(file.Name()) != ".json" {
			continue
		}

		// Read the file
		data, err := os.ReadFile(filepath.Join(am.config.AgentStoragePath, file.Name()))
		if err != nil {
			fmt.Printf("Warning: Failed to read agent file %s: %v\n", file.Name(), err)
			continue
		}

		// Unmarshal the JSON data
		var agent Agent
		err = json.Unmarshal(data, &agent)
		if err != nil {
			fmt.Printf("Warning: Failed to parse agent file %s: %v\n", file.Name(), err)
			continue
		}

		// Add to agents map
		am.agents[agent.ID] = &agent
	}

	return nil
}

// loadBuildCache loads the Docker build cache configuration
func (am *AgentManager) loadBuildCache() error {
	// Check if cache directory exists
	_, err := os.Stat(am.config.CacheStoragePath)
	if os.IsNotExist(err) {
		return nil // No cache to load
	}

	// Cache config file path
	cacheConfigPath := filepath.Join(am.config.CacheStoragePath, "cache_config.json")

	// Check if cache config file exists
	_, err = os.Stat(cacheConfigPath)
	if os.IsNotExist(err) {
		return nil // No cache config to load
	}

	// Read the cache config file
	data, err := os.ReadFile(cacheConfigPath)
	if err != nil {
		return err
	}

	// Unmarshal the JSON data
	return json.Unmarshal(data, am.dockerCache)
}

// saveBuildCache saves the Docker build cache configuration
func (am *AgentManager) saveBuildCache() error {
	// Check if cache directory exists
	_, err := os.Stat(am.config.CacheStoragePath)
	if os.IsNotExist(err) {
		// Create cache directory
		err = os.MkdirAll(am.config.CacheStoragePath, 0755)
		if err != nil {
			return err
		}
	}

	// Cache config file path
	cacheConfigPath := filepath.Join(am.config.CacheStoragePath, "cache_config.json")

	// Marshal the cache config to JSON with indentation for readability
	data, err := json.MarshalIndent(am.dockerCache, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(cacheConfigPath, data, 0644)
}

// IsEnabled returns whether the agent manager is enabled
func (am *AgentManager) IsEnabled() bool {
	am.mutex.RLock()
	defer am.mutex.RUnlock()
	return am.config.Enabled && am.isInitialized
}

// Enable enables the agent manager
func (am *AgentManager) Enable() error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	if !am.isInitialized {
		return fmt.Errorf("agent manager not initialized")
	}

	am.config.Enabled = true
	return am.saveConfig()
}

// Disable disables the agent manager
func (am *AgentManager) Disable() error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	am.config.Enabled = false
	return am.saveConfig()
}

// GetAgent returns an agent by ID
func (am *AgentManager) GetAgent(agentID string) (*Agent, error) {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	agent, ok := am.agents[agentID]
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", agentID)
	}

	return agent, nil
}

// ListAgents returns a list of all agents
func (am *AgentManager) ListAgents() []*Agent {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	agents := make([]*Agent, 0, len(am.agents))
	for _, agent := range am.agents {
		agents = append(agents, agent)
	}

	return agents
}

// CreateAgent creates a new agent
func (am *AgentManager) CreateAgent(agent Agent) error {
	if !am.IsEnabled() {
		return fmt.Errorf("agent manager not enabled")
	}

	am.mutex.Lock()
	defer am.mutex.Unlock()

	// Check if agent already exists
	if _, ok := am.agents[agent.ID]; ok {
		return fmt.Errorf("agent already exists: %s", agent.ID)
	}

	// Set metadata
	agent.CreatedAt = time.Now()
	agent.UpdatedAt = time.Now()
	agent.RunCount = 0
	agent.SuccessRate = 0.0
	agent.Enabled = true

	// Set default values for commands if needed
	for i := range agent.Commands {
		if agent.Commands[i].Timeout == 0 {
			agent.Commands[i].Timeout = am.config.DefaultTimeout
		}
		if agent.Commands[i].RetryCount == 0 {
			agent.Commands[i].RetryCount = am.config.DefaultRetryCount
		}
		if agent.Commands[i].RetryDelay == 0 {
			agent.Commands[i].RetryDelay = am.config.DefaultRetryDelay
		}
	}

	// Add to agents map
	am.agents[agent.ID] = &agent

	// Save agent to disk
	return am.saveAgent(agent)
}

// UpdateAgent updates an existing agent
func (am *AgentManager) UpdateAgent(agent Agent) error {
	if !am.IsEnabled() {
		return fmt.Errorf("agent manager not enabled")
	}

	am.mutex.Lock()
	defer am.mutex.Unlock()

	// Check if agent exists
	existingAgent, ok := am.agents[agent.ID]
	if !ok {
		return fmt.Errorf("agent not found: %s", agent.ID)
	}

	// Preserve metadata
	agent.CreatedAt = existingAgent.CreatedAt
	agent.RunCount = existingAgent.RunCount
	agent.SuccessRate = existingAgent.SuccessRate
	agent.LastRunAt = existingAgent.LastRunAt
	agent.UpdatedAt = time.Now()

	// Update agent in map
	am.agents[agent.ID] = &agent

	// Save agent to disk
	return am.saveAgent(agent)
}

// DeleteAgent deletes an agent
func (am *AgentManager) DeleteAgent(agentID string) error {
	if !am.IsEnabled() {
		return fmt.Errorf("agent manager not enabled")
	}

	am.mutex.Lock()
	defer am.mutex.Unlock()

	// Check if agent exists
	if _, ok := am.agents[agentID]; !ok {
		return fmt.Errorf("agent not found: %s", agentID)
	}

	// Remove agent from map
	delete(am.agents, agentID)

	// Remove agent file from disk
	filePath := filepath.Join(am.config.AgentStoragePath, agentID+".json")
	return os.Remove(filePath)
}

// saveAgent saves an agent to disk
func (am *AgentManager) saveAgent(agent Agent) error {
	// Marshal the agent to JSON with indentation for readability
	data, err := json.MarshalIndent(agent, "", "  ")
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist
	dir := am.config.AgentStoragePath
	if err = os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Write to file
	filePath := filepath.Join(dir, agent.ID+".json")
	return os.WriteFile(filePath, data, 0644)
}

// DockerError represents an error with Docker operations
type DockerError struct {
	Command  string
	ExitCode int
	Output   string
}

// Error implements the error interface
func (e *DockerError) Error() string {
	return fmt.Sprintf("Docker error (exit code %d): %s", e.ExitCode, e.Output)
}

// RunAgent runs an agent
func (am *AgentManager) RunAgent(agentID string, options map[string]string) (*AgentRunResult, error) {
	if !am.IsEnabled() {
		return nil, fmt.Errorf("agent manager not enabled")
	}

	// Get agent
	agent, err := am.GetAgent(agentID)
	if err != nil {
		return nil, err
	}

	// Check if agent is enabled
	if !agent.Enabled {
		return nil, fmt.Errorf("agent is disabled: %s", agentID)
	}

	// Create run result
	result := &AgentRunResult{
		AgentID:         agentID,
		StartTime:       time.Now(),
		Output:          "",
		Errors:          []string{},
		CommandsRun:     0,
		Success:         false,
		ExitCode:        0,
		PerformanceData: make(map[string]float64),
		ArtifactsPaths:  []string{},
	}

	// Check if Docker is required and available
	useDocker := agent.DockerConfig != nil && am.config.UseDockerBuilds
	if useDocker {
		// Check if Docker is available
		if err := am.checkDockerAvailability(); err != nil {
			return nil, fmt.Errorf("Docker not available: %v", err)
		}

		// Build Docker image if needed
		if err := am.prepareDockerImage(agent, result, options); err != nil {
			result.Success = false
			result.ExitCode = 1
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to prepare Docker image: %v", err))
			result.EndTime = time.Now()
			return result, nil
		}
	}

	// Execute commands
	var commandOutput strings.Builder
	commandOutput.WriteString(fmt.Sprintf("Running agent: %s (%s)\n", agent.Name, agent.ID))
	commandOutput.WriteString("=================================\n\n")

	allSuccess := true
	for i, cmd := range agent.Commands {
		// Skip if options specify a specific command and this is not it
		if cmdID, ok := options["command"]; ok && cmdID != "" {
			if _, ok := options[fmt.Sprintf("cmd_%d", i)]; !ok && cmdID != fmt.Sprintf("%d", i) {
				continue
			}
		}

		// Execute command
		commandOutput.WriteString(fmt.Sprintf("Command %d: %s\n", i+1, cmd.Command))
		commandOutput.WriteString(fmt.Sprintf("Working directory: %s\n", cmd.WorkingDir))

		var cmdErr error
		var cmdOutput string
		var exitCode int

		// Execute command either directly or in Docker
		if useDocker {
			// Execute in Docker
			cmdOutput, exitCode, cmdErr = am.executeDockerCommand(agent, cmd, options)
		} else {
			// Execute command directly
			cmdOutput, exitCode, cmdErr = am.executeCommand(cmd, options)
		}

		result.CommandsRun++

		// Check for command error
		if cmdErr != nil {
			errorMsg := fmt.Sprintf("Command %d failed with exit code %d: %v", i+1, exitCode, cmdErr)
			result.Errors = append(result.Errors, errorMsg)
			commandOutput.WriteString(fmt.Sprintf("\nError: %v (exit code: %d)\n", cmdErr, exitCode))

			// Add command output
			commandOutput.WriteString("\nCommand output:\n")
			commandOutput.WriteString(cmdOutput)
			commandOutput.WriteString("\n")

			// Handle error patterns
			errorFixed := false
			if cmd.ErrorPatterns != nil && len(cmd.ErrorPatterns) > 0 {
				// Look for error patterns in output
				for _, pattern := range cmd.ErrorPatterns {
					if strings.Contains(cmdOutput, pattern) {
						commandOutput.WriteString(fmt.Sprintf("\nDetected error pattern: %s\n", pattern))

						// Try to fix the error
						fixed, fixOutput, err := am.tryFixError(agent, cmd, pattern, cmdOutput, options)
						if err != nil {
							commandOutput.WriteString(fmt.Sprintf("Error attempting to fix: %v\n", err))
						} else {
							commandOutput.WriteString(fixOutput)

							if fixed {
								errorFixed = true
								commandOutput.WriteString("\nError has been fixed. Retrying command...\n")

								// Retry the command
								var retryCmdErr error
								if useDocker {
									cmdOutput, exitCode, retryCmdErr = am.executeDockerCommand(agent, cmd, options)
								} else {
									cmdOutput, exitCode, retryCmdErr = am.executeCommand(cmd, options)
								}

								// Check if retry succeeded
								if retryCmdErr == nil {
									commandOutput.WriteString("\nCommand completed successfully after fixing error\n")
									commandOutput.WriteString("Exit code: 0\n")

									// Add command output
									commandOutput.WriteString("\nCommand output:\n")
									commandOutput.WriteString(cmdOutput)
									commandOutput.WriteString("\n")

									// Update error status
									cmdErr = nil
									break
								} else {
									commandOutput.WriteString(fmt.Sprintf("\nCommand still failed after fixing: %v\n", retryCmdErr))
									commandOutput.WriteString("\nCommand output:\n")
									commandOutput.WriteString(cmdOutput)
									commandOutput.WriteString("\n")
								}
							}
						}
					}
				}
			}

			// If error wasn't fixed, set success to false and break
			if !errorFixed {
				allSuccess = false
				break
			}
		} else {
			// Command succeeded
			commandOutput.WriteString("\nCommand completed successfully\n")
			commandOutput.WriteString("Exit code: 0\n")

			// Add command output
			commandOutput.WriteString("\nCommand output:\n")
			commandOutput.WriteString(cmdOutput)
			commandOutput.WriteString("\n")
		}

		// Look for success patterns
		for _, pattern := range cmd.SuccessPatterns {
			if strings.Contains(cmdOutput, pattern) {
				commandOutput.WriteString(fmt.Sprintf("\nSuccess pattern detected: %s\n", pattern))
			}
		}

		commandOutput.WriteString("\n---\n\n")
	}

	// Update result
	result.Success = allSuccess
	result.Output = commandOutput.String()
	result.EndTime = time.Now()

	// Update agent metadata
	am.mutex.Lock()
	agent.LastRunAt = time.Now()
	agent.RunCount++
	if result.Success {
		// Update success rate with running average
		agent.SuccessRate = (agent.SuccessRate*float64(agent.RunCount-1) + 1.0) / float64(agent.RunCount)
	} else {
		// Update success rate with running average
		agent.SuccessRate = (agent.SuccessRate * float64(agent.RunCount-1)) / float64(agent.RunCount)
	}
	am.saveAgent(*agent)
	am.mutex.Unlock()

	// Add result to run history
	am.mutex.Lock()
	am.runHistory = append(am.runHistory, *result)
	// Trim run history if needed
	if len(am.runHistory) > am.config.MaxAgentRuns {
		am.runHistory = am.runHistory[len(am.runHistory)-am.config.MaxAgentRuns:]
	}
	am.mutex.Unlock()

	return result, nil
}

// executeCommand executes a command directly (without Docker)
func (am *AgentManager) executeCommand(cmd AgentCommand, options map[string]string) (string, int, error) {
	// Create command
	execCmd := exec.Command("bash", "-c", cmd.Command)

	// Set working directory
	if cmd.WorkingDir != "" {
		execCmd.Dir = cmd.WorkingDir
	}

	// Set environment
	if len(cmd.Environment) > 0 {
		execCmd.Env = os.Environ()
		for k, v := range cmd.Environment {
			execCmd.Env = append(execCmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	// Capture output
	var stdout, stderr bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	// Set up interactive mode if required
	if cmd.IsInteractive {
		execCmd.Stdin = os.Stdin
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr
	}

	// Execute command
	err := execCmd.Run()

	// Get exit code
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	// Combine stdout and stderr
	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += stderr.String()
	}

	// Check if command failed and should be retried
	if exitCode != 0 && cmd.RetryCount > 0 {
		// Retry logic would go here
		// For now, we'll just log that we would retry
		fmt.Printf("Command failed with exit code %d. Would retry %d times.\n", exitCode, cmd.RetryCount)
	}

	// If command succeeded, return output
	if exitCode == 0 {
		return output, exitCode, nil
	}

	// Return error
	return output, exitCode, fmt.Errorf("command failed with exit code %d", exitCode)
}

// executeDockerCommand executes a command in a Docker container
func (am *AgentManager) executeDockerCommand(agent *Agent, cmd AgentCommand, options map[string]string) (string, int, error) {
	// Ensure Docker config exists
	if agent.DockerConfig == nil {
		return "", 1, fmt.Errorf("agent does not have Docker configuration")
	}

	// Create Docker command
	dockerCmd := fmt.Sprintf("docker run --rm")

	// Add volumes
	for _, volume := range agent.DockerConfig.Volumes {
		dockerCmd += fmt.Sprintf(" -v %s", volume)
	}

	// Add networks
	for _, network := range agent.DockerConfig.Networks {
		dockerCmd += fmt.Sprintf(" --network %s", network)
	}

	// Add environment variables
	for k, v := range agent.DockerConfig.Environment {
		dockerCmd += fmt.Sprintf(" -e %s=%s", k, v)
	}

	// Add command environment variables
	for k, v := range cmd.Environment {
		dockerCmd += fmt.Sprintf(" -e %s=%s", k, v)
	}

	// Add working directory
	if cmd.WorkingDir != "" {
		dockerCmd += fmt.Sprintf(" -w %s", cmd.WorkingDir)
	}

	// Add image and command
	dockerCmd += fmt.Sprintf(" %s:%s /bin/bash -c \"%s\"", agent.DockerConfig.Image, agent.DockerConfig.Tag, cmd.Command)

	// Create command
	execCmd := exec.Command("bash", "-c", dockerCmd)

	// Capture output
	var stdout, stderr bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	// Execute command
	err := execCmd.Run()

	// Get exit code
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	// Combine stdout and stderr
	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += stderr.String()
	}

	// Check if command failed and should be retried
	if exitCode != 0 && cmd.RetryCount > 0 {
		// Retry logic would go here
		// For now, we'll just log that we would retry
		fmt.Printf("Docker command failed with exit code %d. Would retry %d times.\n", exitCode, cmd.RetryCount)
	}

	// If command succeeded, return output
	if exitCode == 0 {
		return output, exitCode, nil
	}

	// Return error
	return output, exitCode, &DockerError{
		Command:  dockerCmd,
		ExitCode: exitCode,
		Output:   output,
	}
}

// prepareDockerImage builds a Docker image for an agent if needed
func (am *AgentManager) prepareDockerImage(agent *Agent, result *AgentRunResult, options map[string]string) error {
	// Ensure Docker config exists
	if agent.DockerConfig == nil {
		return fmt.Errorf("agent does not have Docker configuration")
	}

	// Check if image exists
	imageExists, err := am.checkDockerImageExists(agent.DockerConfig.Image, agent.DockerConfig.Tag)
	if err != nil {
		return err
	}

	// Build image if it doesn't exist or if force build is specified
	forceBuild := false
	if val, ok := options["force_build"]; ok && (val == "true" || val == "1") {
		forceBuild = true
	}

	if !imageExists || forceBuild {
		// Check if agent has a build configuration
		if agent.DockerConfig.Dockerfile == "" && agent.DockerConfig.BuildContext == "" {
			// No build configuration, but required image doesn't exist
			return fmt.Errorf("Docker image %s:%s does not exist and no build configuration is provided",
				agent.DockerConfig.Image, agent.DockerConfig.Tag)
		}

		// Build Docker image
		buildOutput, err := am.buildDockerImage(agent, result, options)
		if err != nil {
			result.Output += fmt.Sprintf("\nFailed to build Docker image: %v\n", err)
			result.Output += buildOutput
			return err
		}

		result.Output += fmt.Sprintf("\nDocker image %s:%s built successfully\n",
			agent.DockerConfig.Image, agent.DockerConfig.Tag)
	}

	return nil
}

// checkDockerImageExists checks if a Docker image exists
func (am *AgentManager) checkDockerImageExists(image, tag string) (bool, error) {
	// Create command to list images
	cmd := exec.Command("docker", "images", "--format", "{{.Repository}}:{{.Tag}}")

	// Capture output
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	// Execute command
	err := cmd.Run()
	if err != nil {
		return false, fmt.Errorf("failed to list Docker images: %v", err)
	}

	// Check if image exists
	return strings.Contains(stdout.String(), fmt.Sprintf("%s:%s", image, tag)), nil
}

// WaterfallBuildStatus tracks the status of a waterfall build
type WaterfallBuildStatus struct {
	CompletedStages map[string]bool
	StageDuration   map[string]time.Duration
	StageOutput     map[string]string
	CurrentStage    string
	StartTime       time.Time
	EndTime         time.Time
	Success         bool
}

// buildDockerImage builds a Docker image for an agent
func (am *AgentManager) buildDockerImage(agent *Agent, result *AgentRunResult, options map[string]string) (string, error) {
	var buildOutput strings.Builder

	// Check if agent has waterfall build configuration
	if agent.DockerConfig.Waterfall.Stages != nil && len(agent.DockerConfig.Waterfall.Stages) > 0 {
		// Waterfall build
		buildOutput.WriteString("Using waterfall build system\n")

		// Execute waterfall build
		waterfallOutput, err := am.executeWaterfallBuild(agent, result, options)
		buildOutput.WriteString(waterfallOutput)

		if err != nil {
			return buildOutput.String(), err
		}

		return buildOutput.String(), nil
	}

	// Regular build
	// Check if we have a Dockerfile
	if agent.DockerConfig.Dockerfile == "" {
		return buildOutput.String(), fmt.Errorf("no Dockerfile specified for agent")
	}

	// Set build context
	buildContext := agent.DockerConfig.BuildContext
	if buildContext == "" {
		// Use directory containing Dockerfile as build context
		buildContext = filepath.Dir(agent.DockerConfig.Dockerfile)
	}

	// Build Docker image
	buildCmd := fmt.Sprintf("docker build -t %s:%s -f %s %s",
		agent.DockerConfig.Image, agent.DockerConfig.Tag, agent.DockerConfig.Dockerfile, buildContext)

	// Add build args
	for k, v := range agent.DockerConfig.BuildArgs {
		buildCmd += fmt.Sprintf(" --build-arg %s=%s", k, v)
	}

	// Add cache-from if specified
	if agent.DockerConfig.CacheFrom != nil && len(agent.DockerConfig.CacheFrom) > 0 {
		for _, cacheFrom := range agent.DockerConfig.CacheFrom {
			buildCmd += fmt.Sprintf(" --cache-from %s", cacheFrom)
		}
	}

	// Disable cache if specified
	if !agent.DockerConfig.UseCache {
		buildCmd += " --no-cache"
	}

	// Execute build command
	cmd := exec.Command("bash", "-c", buildCmd)

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute command
	err := cmd.Run()

	// Add output to result
	buildOutput.WriteString(stdout.String())
	if stderr.Len() > 0 {
		buildOutput.WriteString(stderr.String())
	}

	// Check if build failed
	if err != nil {
		return buildOutput.String(), fmt.Errorf("failed to build Docker image: %v", err)
	}

	return buildOutput.String(), nil
}

// executeWaterfallBuild executes a multi-stage Docker build using the waterfall pattern
func (am *AgentManager) executeWaterfallBuild(agent *Agent, result *AgentRunResult, options map[string]string) (string, error) {
	var buildOutput strings.Builder

	// Initialize build status
	status := &WaterfallBuildStatus{
		CompletedStages: make(map[string]bool),
		StageDuration:   make(map[string]time.Duration),
		StageOutput:     make(map[string]string),
		StartTime:       time.Now(),
		Success:         false,
	}

	// Create a temporary directory for the build
	tempDir, err := os.MkdirTemp("", "delta-waterfall-build-")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Generate a unique build ID based on time
	buildID := fmt.Sprintf("build-%d", time.Now().Unix())

	// Get the stages and dependencies
	stages := agent.DockerConfig.Waterfall.Stages
	dependencies := agent.DockerConfig.Waterfall.Dependencies

	// Check if a docker-compose file was specified
	composeFileSpecified := agent.DockerConfig.ComposeFile != ""

	// If no compose file was specified, generate one
	composeFilePath := agent.DockerConfig.ComposeFile
	if !composeFileSpecified {
		// Generate a docker-compose file
		composeFilePath = filepath.Join(tempDir, "docker-compose.yml")
		err = am.generateDockerComposeFile(agent, composeFilePath, buildID)
		if err != nil {
			return "", fmt.Errorf("failed to generate docker-compose file: %v", err)
		}
	}

	// Log the docker-compose file path
	buildOutput.WriteString(fmt.Sprintf("Using docker-compose file: %s\n", composeFilePath))

	// Build order will be determined by dependencies
	buildOrder, err := am.determineBuildOrder(stages, dependencies)
	if err != nil {
		return "", fmt.Errorf("failed to determine build order: %v", err)
	}

	// Log the build order
	buildOutput.WriteString("Build order:\n")
	for i, stage := range buildOrder {
		buildOutput.WriteString(fmt.Sprintf("  %d. %s\n", i+1, stage))
	}
	buildOutput.WriteString("\n")

	// Execute each stage in order
	for _, stage := range buildOrder {
		status.CurrentStage = stage
		stageStartTime := time.Now()

		buildOutput.WriteString(fmt.Sprintf("Building stage: %s\n", stage))

		// Check if all dependencies have been built
		allDepsBuilt := true
		if deps, ok := dependencies[stage]; ok {
			for _, dep := range deps {
				if !status.CompletedStages[dep] {
					allDepsBuilt = false
					buildOutput.WriteString(fmt.Sprintf("Error: Dependency %s for stage %s has not been built\n", dep, stage))
				}
			}
		}

		if !allDepsBuilt {
			return buildOutput.String(), fmt.Errorf("cannot build stage %s, dependencies not met", stage)
		}

		// Build the stage
		cmdOutput, err := am.buildDockerComposeStage(composeFilePath, stage, agent, options)

		// Record stage output
		status.StageOutput[stage] = cmdOutput
		buildOutput.WriteString(fmt.Sprintf("--- Stage %s output ---\n", stage))
		buildOutput.WriteString(cmdOutput)
		buildOutput.WriteString(fmt.Sprintf("--- End of stage %s output ---\n\n", stage))

		// Record stage duration
		stageDuration := time.Since(stageStartTime)
		status.StageDuration[stage] = stageDuration
		buildOutput.WriteString(fmt.Sprintf("Stage %s completed in %s\n", stage, formatAgentDuration(stageDuration)))

		// Check if build failed
		if err != nil {
			buildOutput.WriteString(fmt.Sprintf("Error building stage %s: %v\n", stage, err))
			// Add to errors
			result.Errors = append(result.Errors, fmt.Sprintf("Error building stage %s: %v", stage, err))
			return buildOutput.String(), err
		}

		// Mark stage as completed
		status.CompletedStages[stage] = true
		buildOutput.WriteString(fmt.Sprintf("Stage %s completed successfully\n\n", stage))

		// Update cache statistics
		am.updateCacheStats(stage, cmdOutput)
	}

	// Build completed successfully
	status.EndTime = time.Now()
	status.Success = true

	totalDuration := status.EndTime.Sub(status.StartTime)
	buildOutput.WriteString(fmt.Sprintf("Waterfall build completed in %s\n", formatAgentDuration(totalDuration)))

	// Tag the final image if required
	if agent.DockerConfig.Image != "" && agent.DockerConfig.Tag != "" {
		finalStage := stages[len(stages)-1]
		finalImageName := fmt.Sprintf("%s_%s", buildID, finalStage)

		// Tag the final image
		tagCmd := fmt.Sprintf("docker tag %s %s:%s", finalImageName, agent.DockerConfig.Image, agent.DockerConfig.Tag)
		cmd := exec.Command("bash", "-c", tagCmd)

		// Execute command
		err = cmd.Run()
		if err != nil {
			buildOutput.WriteString(fmt.Sprintf("Error tagging final image: %v\n", err))
			// This is not a critical error, so we continue
		} else {
			buildOutput.WriteString(fmt.Sprintf("Tagged final image as %s:%s\n", agent.DockerConfig.Image, agent.DockerConfig.Tag))
		}
	}

	// Add performance data to result
	result.PerformanceData["build_duration"] = totalDuration.Seconds()

	for stage, duration := range status.StageDuration {
		result.PerformanceData[fmt.Sprintf("stage_%s_duration", stage)] = duration.Seconds()
	}

	return buildOutput.String(), nil
}

// generateDockerComposeFile generates a docker-compose file for a waterfall build
func (am *AgentManager) generateDockerComposeFile(agent *Agent, outputPath string, buildID string) error {
	// Create docker-compose file structure
	composeData := map[string]interface{}{
		"version":  "3",
		"services": map[string]interface{}{},
	}

	// Add services for each stage
	services := composeData["services"].(map[string]interface{})

	for _, stage := range agent.DockerConfig.Waterfall.Stages {
		// Create service for this stage
		serviceName := stage

		// Create Dockerfile for this stage (reference the main Dockerfile with target)
		dockerfile := agent.DockerConfig.Dockerfile
		if dockerfile == "" {
			return fmt.Errorf("no Dockerfile specified for agent")
		}

		// Determine build context
		buildContext := agent.DockerConfig.BuildContext
		if buildContext == "" {
			buildContext = filepath.Dir(dockerfile)
		}

		// Service configuration
		service := map[string]interface{}{
			"image": fmt.Sprintf("%s_%s", buildID, stage),
			"build": map[string]interface{}{
				"context":    buildContext,
				"dockerfile": dockerfile,
				"target":     stage,
			},
		}

		// Add build args if specified
		if len(agent.DockerConfig.BuildArgs) > 0 {
			buildArgs := make(map[string]string)
			for k, v := range agent.DockerConfig.BuildArgs {
				buildArgs[k] = v
			}
			service["build"].(map[string]interface{})["args"] = buildArgs
		}

		// Add volumes if specified
		if len(agent.DockerConfig.Volumes) > 0 {
			service["volumes"] = agent.DockerConfig.Volumes
		}

		// Add networks if specified
		if len(agent.DockerConfig.Networks) > 0 {
			service["networks"] = agent.DockerConfig.Networks
		}

		// Add dependencies
		if deps, ok := agent.DockerConfig.Waterfall.Dependencies[stage]; ok && len(deps) > 0 {
			service["depends_on"] = deps
		}

		// Add service to services map
		services[serviceName] = service
	}

	// Marshal to YAML
	yamlData, err := yaml.Marshal(composeData)
	if err != nil {
		return fmt.Errorf("failed to marshal docker-compose data: %v", err)
	}

	// Write to file
	err = os.WriteFile(outputPath, yamlData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write docker-compose file: %v", err)
	}

	return nil
}

// determineBuildOrder calculates the build order based on dependencies
func (am *AgentManager) determineBuildOrder(stages []string, dependencies map[string][]string) ([]string, error) {
	// Build a directed graph of dependencies
	graph := make(map[string][]string)
	inDegree := make(map[string]int)

	// Initialize graph
	for _, stage := range stages {
		graph[stage] = []string{}
		inDegree[stage] = 0
	}

	// Add dependencies
	for stage, deps := range dependencies {
		for _, dep := range deps {
			// Add dependency: dep -> stage (dep is required by stage)
			graph[dep] = append(graph[dep], stage)
			inDegree[stage]++
		}
	}

	// Find stages with no dependencies
	var queue []string
	for stage, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, stage)
		}
	}

	// Perform topological sort
	var buildOrder []string

	for len(queue) > 0 {
		// Get next stage with no dependencies
		stage := queue[0]
		queue = queue[1:]

		// Add to build order
		buildOrder = append(buildOrder, stage)

		// Process stages that depend on this stage
		for _, dependent := range graph[stage] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	// Check if we have a cycle in the dependency graph
	if len(buildOrder) != len(stages) {
		return nil, fmt.Errorf("cycle detected in dependency graph, cannot determine build order")
	}

	return buildOrder, nil
}

// buildDockerComposeStage builds a specific stage using docker-compose
func (am *AgentManager) buildDockerComposeStage(composeFilePath, stage string, agent *Agent, options map[string]string) (string, error) {
	// Build the stage
	buildCmd := fmt.Sprintf("docker-compose -f %s build", composeFilePath)

	// Add cache options
	if !agent.DockerConfig.UseCache {
		buildCmd += " --no-cache"
	}

	// Add stage
	buildCmd += fmt.Sprintf(" %s", stage)

	// Execute command
	cmd := exec.Command("bash", "-c", buildCmd)

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute command
	err := cmd.Run()

	// Combine output
	output := stdout.String()
	if stderr.Len() > 0 {
		output += "\n" + stderr.String()
	}

	// Check if build failed
	if err != nil {
		return output, fmt.Errorf("failed to build stage %s: %v", stage, err)
	}

	return output, nil
}

// updateCacheStats updates the Docker build cache statistics
func (am *AgentManager) updateCacheStats(stage string, buildOutput string) {
	// Parse build output to extract cache information
	cacheHits := countPattern(buildOutput, "Using cache")
	cacheMisses := countPattern(buildOutput, "Running in") - 1 // Subtract 1 for the initial "Running in" message

	if cacheMisses < 0 {
		cacheMisses = 0
	}

	// Get the build config for this stage
	am.mutex.Lock()
	defer am.mutex.Unlock()

	// Create or update build cache config
	buildConfig, ok := am.dockerCache.BuildConfigs[stage]
	if !ok {
		buildConfig = &BuildCacheConfig{
			Name:        stage,
			Stages:      []string{stage},
			CacheHits:   0,
			CacheMisses: 0,
			LastBuiltAt: time.Now(),
		}
		am.dockerCache.BuildConfigs[stage] = buildConfig
	}

	// Update cache statistics
	buildConfig.CacheHits += cacheHits
	buildConfig.CacheMisses += cacheMisses
	buildConfig.LastBuiltAt = time.Now()

	// Save cache configuration
	am.saveBuildCache()
}

// countPattern counts occurrences of a pattern in a string
func countPattern(text, pattern string) int {
	count := 0
	pos := 0

	for {
		pos = strings.Index(text[pos:], pattern)
		if pos == -1 {
			break
		}

		count++
		pos++
	}

	return count
}

// formatDuration formats a duration in a human-readable format
func formatAgentDuration(d time.Duration) string {
	seconds := int(d.Seconds())
	minutes := seconds / 60
	hours := minutes / 60

	seconds %= 60
	minutes %= 60

	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	} else {
		return fmt.Sprintf("%ds", seconds)
	}
}

// checkDockerAvailability checks if Docker is available
func (am *AgentManager) checkDockerAvailability() error {
	// Check if Docker command is available
	_, err := exec.LookPath("docker")
	if err != nil {
		return fmt.Errorf("Docker not found: %v", err)
	}

	// Check if Docker daemon is running
	cmd := exec.Command("docker", "info")
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("Docker daemon not running: %v", err)
	}

	return nil
}

// FindAgentsByContext finds agents matching a context
func (am *AgentManager) FindAgentsByContext(key, value string) []*Agent {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	var matches []*Agent
	for _, agent := range am.agents {
		if v, ok := agent.Context[key]; ok && v == value {
			matches = append(matches, agent)
		}
	}

	return matches
}

// FindAgentsByTrigger finds agents matching a trigger pattern
func (am *AgentManager) FindAgentsByTrigger(pattern string) []*Agent {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	var matches []*Agent
	for _, agent := range am.agents {
		for _, trigger := range agent.TriggerPatterns {
			if trigger == pattern || (len(trigger) > 0 && len(pattern) > 0 && strings.Contains(strings.ToLower(pattern), strings.ToLower(trigger))) {
				matches = append(matches, agent)
				break
			}
		}
	}

	return matches
}

// GetRunHistory returns the agent run history
func (am *AgentManager) GetRunHistory(agentID string, limit int) []AgentRunResult {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	var history []AgentRunResult
	for i := len(am.runHistory) - 1; i >= 0; i-- {
		if am.runHistory[i].AgentID == agentID || agentID == "" {
			history = append(history, am.runHistory[i])
			if limit > 0 && len(history) >= limit {
				break
			}
		}
	}

	return history
}

// GetDockerCacheStats returns statistics about the Docker build cache
func (am *AgentManager) GetDockerCacheStats() map[string]interface{} {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	cacheHits := 0
	cacheMisses := 0
	cacheSizeBytes := am.dockerCache.CacheSize

	for _, config := range am.dockerCache.BuildConfigs {
		cacheHits += config.CacheHits
		cacheMisses += config.CacheMisses
	}

	cacheEfficiency := 0.0
	if cacheHits+cacheMisses > 0 {
		cacheEfficiency = float64(cacheHits) / float64(cacheHits+cacheMisses) * 100.0
	}

	return map[string]interface{}{
		"cache_size_bytes": cacheSizeBytes,
		"cache_size_mb":    float64(cacheSizeBytes) / (1024 * 1024),
		"cache_hits":       cacheHits,
		"cache_misses":     cacheMisses,
		"cache_efficiency": cacheEfficiency,
		"build_configs":    len(am.dockerCache.BuildConfigs),
		"max_cache_age":    am.dockerCache.MaxCacheAge.Hours() / 24, // in days
	}
}

// ClearDockerCache clears the Docker build cache
func (am *AgentManager) ClearDockerCache() error {
	if !am.IsEnabled() {
		return fmt.Errorf("agent manager not enabled")
	}

	am.mutex.Lock()
	defer am.mutex.Unlock()

	// Reset cache data
	am.dockerCache.BuildConfigs = make(map[string]*BuildCacheConfig)
	am.dockerCache.CacheSize = 0

	// Save cache configuration
	err := am.saveBuildCache()
	if err != nil {
		return err
	}

	// Delete cache directory contents
	err = os.RemoveAll(am.config.CacheStoragePath)
	if err != nil {
		return err
	}

	// Recreate cache directory
	return os.MkdirAll(am.config.CacheStoragePath, 0755)
}

// GetAgentStats returns statistics about agents
func (am *AgentManager) GetAgentStats() map[string]interface{} {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	totalRuns := 0
	successfulRuns := 0
	totalRunTime := int64(0)

	for _, agent := range am.agents {
		totalRuns += agent.RunCount
	}

	for _, result := range am.runHistory {
		if result.Success {
			successfulRuns++
		}
		totalRunTime += result.EndTime.Unix() - result.StartTime.Unix()
	}

	avgRunTime := 0.0
	if len(am.runHistory) > 0 {
		avgRunTime = float64(totalRunTime) / float64(len(am.runHistory))
	}

	successRate := 0.0
	if totalRuns > 0 {
		successRate = float64(successfulRuns) / float64(totalRuns) * 100.0
	}

	return map[string]interface{}{
		"total_agents":    len(am.agents),
		"enabled_agents":  countEnabledAgents(am.agents),
		"total_runs":      totalRuns,
		"successful_runs": successfulRuns,
		"success_rate":    successRate,
		"avg_run_time":    avgRunTime,
	}
}

// countEnabledAgents counts the number of enabled agents
func countEnabledAgents(agents map[string]*Agent) int {
	count := 0
	for _, agent := range agents {
		if agent.Enabled {
			count++
		}
	}
	return count
}

// ImportAgentFromYAML imports agents from a YAML file
func (am *AgentManager) ImportAgentFromYAML(yamlPath string) ([]string, error) {
	// Ensure agent manager is initialized
	if !am.isInitialized {
		return nil, fmt.Errorf("agent manager not initialized")
	}

	// Read the YAML file
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read YAML file: %v", err)
	}

	// Import the YAML configuration
	return am.importAgentYAML(data, filepath.Dir(yamlPath))
}

// importAgentYAML imports agents from YAML configuration
func (am *AgentManager) importAgentYAML(yamlData []byte, basePath string) ([]string, error) {
	// Parse YAML configuration
	var config AgentYAMLConfig
	err := yaml.Unmarshal(yamlData, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %v", err)
	}

	// Check YAML version
	if config.Version != "1.0" {
		return nil, fmt.Errorf("unsupported YAML version: %s", config.Version)
	}

	// Keep track of imported agents
	importedAgents := make([]string, 0)

	// Process each agent
	for _, yamlAgent := range config.Agents {
		// Check if agent has an import reference
		if yamlAgent.Import != "" {
			// Read the imported YAML file
			importPath := filepath.Join(basePath, yamlAgent.Import)
			importData, err := os.ReadFile(importPath)
			if err != nil {
				fmt.Printf("Warning: Failed to import agent %s from %s: %v\n", yamlAgent.ID, importPath, err)
				continue
			}

			// Parse imported YAML
			var importedConfig struct {
				ID          string   `yaml:"id"`
				Name        string   `yaml:"name"`
				Description string   `yaml:"description"`
				Enabled     bool     `yaml:"enabled"`
				TaskTypes   []string `yaml:"task_types"`
				Commands    []struct {
					ID              string            `yaml:"id"`
					Command         string            `yaml:"command"`
					WorkingDir      string            `yaml:"working_dir"`
					Timeout         int               `yaml:"timeout,omitempty"`
					RetryCount      int               `yaml:"retry_count,omitempty"`
					ErrorPatterns   []string          `yaml:"error_patterns,omitempty"`
					SuccessPatterns []string          `yaml:"success_patterns,omitempty"`
					IsInteractive   bool              `yaml:"is_interactive,omitempty"`
					Environment     map[string]string `yaml:"environment,omitempty"`
				} `yaml:"commands,omitempty"`
				Docker struct {
					Enabled     bool              `yaml:"enabled"`
					Image       string            `yaml:"image,omitempty"`
					Tag         string            `yaml:"tag,omitempty"`
					Dockerfile  string            `yaml:"dockerfile,omitempty"`
					ComposeFile string            `yaml:"compose_file,omitempty"`
					Volumes     []string          `yaml:"volumes,omitempty"`
					Networks    []string          `yaml:"networks,omitempty"`
					Environment map[string]string `yaml:"environment,omitempty"`
					BuildArgs   map[string]string `yaml:"build_args,omitempty"`
					UseCache    bool              `yaml:"use_cache"`
					Waterfall   struct {
						Stages       []string            `yaml:"stages"`
						Dependencies map[string][]string `yaml:"dependencies"`
					} `yaml:"waterfall,omitempty"`
				} `yaml:"docker,omitempty"`
				ErrorHandling struct {
					AutoFix  bool `yaml:"auto_fix"`
					Patterns []struct {
						Pattern     string `yaml:"pattern"`
						Solution    string `yaml:"solution"`
						Description string `yaml:"description,omitempty"`
						FilePattern string `yaml:"file_pattern,omitempty"`
					} `yaml:"patterns,omitempty"`
				} `yaml:"error_handling,omitempty"`
				Metadata map[string]string `yaml:"metadata,omitempty"`
				Triggers struct {
					Patterns  []string `yaml:"patterns,omitempty"`
					Paths     []string `yaml:"paths,omitempty"`
					Schedules []string `yaml:"schedules,omitempty"`
					Events    []string `yaml:"events,omitempty"`
				} `yaml:"triggers,omitempty"`
			}

			err = yaml.Unmarshal(importData, &importedConfig)
			if err != nil {
				fmt.Printf("Warning: Failed to parse imported YAML for agent %s: %v\n", yamlAgent.ID, err)
				continue
			}

			// Ensure the ID matches
			if importedConfig.ID != yamlAgent.ID {
				fmt.Printf("Warning: Imported agent ID (%s) does not match reference ID (%s)\n", importedConfig.ID, yamlAgent.ID)
				continue
			}

			// Update YAML agent with imported configuration
			yamlAgent.Name = importedConfig.Name
			yamlAgent.Description = importedConfig.Description
			yamlAgent.Enabled = importedConfig.Enabled
			yamlAgent.TaskTypes = importedConfig.TaskTypes
			yamlAgent.Commands = importedConfig.Commands
			yamlAgent.Docker = importedConfig.Docker
			yamlAgent.ErrorHandling = importedConfig.ErrorHandling
			yamlAgent.Metadata = importedConfig.Metadata
			yamlAgent.Triggers = importedConfig.Triggers
		}

		// Create agent from YAML configuration
		agent := am.createAgentFromYAML(yamlAgent, config)
		if agent != nil {
			// Add or update agent
			am.mutex.Lock()
			am.agents[agent.ID] = agent
			am.mutex.Unlock()

			// Save agent to disk
			err = am.saveAgent(*agent)
			if err != nil {
				fmt.Printf("Warning: Failed to save agent %s: %v\n", agent.ID, err)
			}

			importedAgents = append(importedAgents, agent.ID)
		}
	}

	return importedAgents, nil
}

// createAgentFromYAML creates an Agent from YAML configuration
func (am *AgentManager) createAgentFromYAML(yamlAgent struct {
	ID          string   `yaml:"id"`
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Enabled     bool     `yaml:"enabled"`
	TaskTypes   []string `yaml:"task_types"`
	Import      string   `yaml:"import,omitempty"`
	Commands    []struct {
		ID              string            `yaml:"id"`
		Command         string            `yaml:"command"`
		WorkingDir      string            `yaml:"working_dir"`
		Timeout         int               `yaml:"timeout,omitempty"`
		RetryCount      int               `yaml:"retry_count,omitempty"`
		ErrorPatterns   []string          `yaml:"error_patterns,omitempty"`
		SuccessPatterns []string          `yaml:"success_patterns,omitempty"`
		IsInteractive   bool              `yaml:"is_interactive,omitempty"`
		Environment     map[string]string `yaml:"environment,omitempty"`
	} `yaml:"commands,omitempty"`
	Docker struct {
		Enabled     bool              `yaml:"enabled"`
		Image       string            `yaml:"image,omitempty"`
		Tag         string            `yaml:"tag,omitempty"`
		Dockerfile  string            `yaml:"dockerfile,omitempty"`
		ComposeFile string            `yaml:"compose_file,omitempty"`
		Volumes     []string          `yaml:"volumes,omitempty"`
		Networks    []string          `yaml:"networks,omitempty"`
		Environment map[string]string `yaml:"environment,omitempty"`
		BuildArgs   map[string]string `yaml:"build_args,omitempty"`
		UseCache    bool              `yaml:"use_cache"`
		Waterfall   struct {
			Stages       []string            `yaml:"stages"`
			Dependencies map[string][]string `yaml:"dependencies"`
		} `yaml:"waterfall,omitempty"`
	} `yaml:"docker,omitempty"`
	ErrorHandling struct {
		AutoFix  bool `yaml:"auto_fix"`
		Patterns []struct {
			Pattern     string `yaml:"pattern"`
			Solution    string `yaml:"solution"`
			Description string `yaml:"description,omitempty"`
			FilePattern string `yaml:"file_pattern,omitempty"`
		} `yaml:"patterns,omitempty"`
	} `yaml:"error_handling,omitempty"`
	Metadata map[string]string `yaml:"metadata,omitempty"`
	Triggers struct {
		Patterns  []string `yaml:"patterns,omitempty"`
		Paths     []string `yaml:"paths,omitempty"`
		Schedules []string `yaml:"schedules,omitempty"`
		Events    []string `yaml:"events,omitempty"`
	} `yaml:"triggers,omitempty"`
}, config AgentYAMLConfig) *Agent {
	// Validate required fields
	if yamlAgent.ID == "" || yamlAgent.Name == "" || yamlAgent.Description == "" {
		fmt.Printf("Warning: Agent missing required fields (ID, Name, Description)\n")
		return nil
	}

	// Create agent
	agent := &Agent{
		ID:          yamlAgent.ID,
		Name:        yamlAgent.Name,
		Description: yamlAgent.Description,
		Enabled:     yamlAgent.Enabled,
		TaskTypes:   yamlAgent.TaskTypes,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Commands:    []AgentCommand{},
		Context:     make(map[string]string),
		Tags:        []string{},
	}

	// Add project context
	agent.Context["project"] = config.Project.Name
	agent.Context["repository"] = config.Project.Repository

	// Convert commands
	for _, cmd := range yamlAgent.Commands {
		// Apply default timeout and retry settings
		timeout := cmd.Timeout
		if timeout == 0 {
			timeout = am.config.DefaultTimeout
		}

		retryCount := cmd.RetryCount
		if retryCount == 0 {
			retryCount = am.config.DefaultRetryCount
		}

		// Create command
		agentCmd := AgentCommand{
			Command:         expandVariables(cmd.Command, agent.Context),
			WorkingDir:      expandVariables(cmd.WorkingDir, agent.Context),
			ExpectedOutput:  "",
			ErrorPatterns:   cmd.ErrorPatterns,
			SuccessPatterns: cmd.SuccessPatterns,
			Timeout:         timeout,
			RetryCount:      retryCount,
			RetryDelay:      am.config.DefaultRetryDelay,
			IsInteractive:   cmd.IsInteractive,
			Environment:     make(map[string]string),
		}

		// Copy environment variables with variable expansion
		for k, v := range cmd.Environment {
			agentCmd.Environment[k] = expandVariables(v, agent.Context)
		}

		agent.Commands = append(agent.Commands, agentCmd)
	}

	// Setup Docker configuration if enabled
	if yamlAgent.Docker.Enabled {
		dockerConfig := &AgentDockerConfig{
			Image:        yamlAgent.Docker.Image,
			Tag:          yamlAgent.Docker.Tag,
			BuildContext: "",
			Dockerfile:   expandVariables(yamlAgent.Docker.Dockerfile, agent.Context),
			UseCache:     yamlAgent.Docker.UseCache,
			Volumes:      []string{},
			Networks:     []string{},
			Environment:  make(map[string]string),
			BuildArgs:    make(map[string]string),
		}

		// Apply global Docker settings from project if not overridden
		if dockerConfig.Image == "" && config.Project.Name != "" {
			dockerConfig.Image = strings.ToLower(config.Project.Name) + "-agent"
		}

		if dockerConfig.Tag == "" {
			dockerConfig.Tag = "latest"
		}

		// Copy volumes with variable expansion
		for _, v := range yamlAgent.Docker.Volumes {
			dockerConfig.Volumes = append(dockerConfig.Volumes, expandVariables(v, agent.Context))
		}

		// Copy networks
		dockerConfig.Networks = append(dockerConfig.Networks, yamlAgent.Docker.Networks...)

		// Copy environment variables with variable expansion
		for k, v := range yamlAgent.Docker.Environment {
			dockerConfig.Environment[k] = expandVariables(v, agent.Context)
		}

		// Copy build arguments with variable expansion
		for k, v := range yamlAgent.Docker.BuildArgs {
			dockerConfig.BuildArgs[k] = expandVariables(v, agent.Context)
		}

		agent.DockerConfig = dockerConfig
	}

	// Add trigger patterns
	if len(yamlAgent.Triggers.Patterns) > 0 {
		agent.TriggerPatterns = append(agent.TriggerPatterns, yamlAgent.Triggers.Patterns...)
	}

	// Add metadata as tags
	for k, v := range yamlAgent.Metadata {
		agent.Tags = append(agent.Tags, k+":"+v)
	}

	return agent
}

// expandVariables replaces variables in a string with their values
func expandVariables(input string, context map[string]string) string {
	// Check if input contains variables
	if !strings.Contains(input, "${") {
		return input
	}

	// Replace context variables
	result := input
	for k, v := range context {
		result = strings.ReplaceAll(result, "${"+k+"}", v)
	}

	// Replace HOME variable
	homeDir, _ := os.UserHomeDir()
	result = strings.ReplaceAll(result, "${HOME}", homeDir)

	// Replace USER variable
	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME") // For Windows
	}
	result = strings.ReplaceAll(result, "${USER}", username)

	// Replace DELTA_CONFIG variable
	deltaConfig := filepath.Join(homeDir, ".config", "delta")
	result = strings.ReplaceAll(result, "${DELTA_CONFIG}", deltaConfig)

	return result
}

// DiscoverAgents discovers agents in a repository
func (am *AgentManager) DiscoverAgents(repoPath string) ([]string, error) {
	// Ensure agent manager is initialized
	if !am.isInitialized {
		return nil, fmt.Errorf("agent manager not initialized")
	}

	// Check if repository exists
	if _, err := os.Stat(repoPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("repository not found: %s", repoPath)
	}

	// Check for main agent definition file
	agentYamlPath := filepath.Join(repoPath, ".delta", "agents.yml")
	if _, err := os.Stat(agentYamlPath); os.IsNotExist(err) {
		// Check for alternate extension
		agentYamlPath = filepath.Join(repoPath, ".delta", "agents.yaml")
		if _, err := os.Stat(agentYamlPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("no agent definition file found in %s/.delta", repoPath)
		}
	}

	// Import agents from YAML file
	return am.ImportAgentFromYAML(agentYamlPath)
}

// ErrorSolution represents a solution to an error
type ErrorSolution struct {
	Pattern     string
	Solution    string
	Description string
	FilePattern string
}

// tryFixError attempts to fix an error based on its pattern
func (am *AgentManager) tryFixError(agent *Agent, cmd AgentCommand, pattern, output string, options map[string]string) (bool, string, error) {
	var fixOutput strings.Builder
	var solutions []ErrorSolution
	var learnedSolutions []SolutionEffectiveness

	// Get working directory for context
	workDir := cmd.WorkingDir
	if workDir == "" {
		workDir = "."
	}

	// First check agent's error handling patterns
	if agent.DockerConfig != nil && agent.DockerConfig.Waterfall.Stages != nil {
		// Add waterfall-specific error handling
		waterfallErrors := []ErrorSolution{
			{
				Pattern:     "no space left on device",
				Solution:    "docker system prune -af --volumes",
				Description: "No space left on device",
			},
			{
				Pattern:     "failed to solve: rpc error",
				Solution:    "docker builder prune",
				Description: "Docker builder cache error",
			},
		}
		solutions = append(solutions, waterfallErrors...)
	}

	// Check for agent-specific error handling
	if agent.ErrorHandling != nil && agent.ErrorHandling.Patterns != nil && len(agent.ErrorHandling.Patterns) > 0 {
		for _, errPattern := range agent.ErrorHandling.Patterns {
			if strings.Contains(pattern, errPattern.Pattern) || strings.Contains(output, errPattern.Pattern) {
				solutions = append(solutions, ErrorSolution{
					Pattern:     errPattern.Pattern,
					Solution:    errPattern.Solution,
					Description: errPattern.Description,
					FilePattern: errPattern.FilePattern,
				})
			}
		}
	}

	// Check for global error patterns
	for _, errPattern := range am.getGlobalErrorPatterns() {
		if strings.Contains(pattern, errPattern.Pattern) || strings.Contains(output, errPattern.Pattern) {
			solutions = append(solutions, errPattern)
		}
	}

	// Check for learned solutions from the error learning manager
	errorLearningMgr := GetErrorLearningManager()
	if errorLearningMgr != nil {
		learnedSolutions = errorLearningMgr.GetBestSolutions(pattern, 3)
		fixOutput.WriteString(fmt.Sprintf("Found %d learned solutions from previous errors\n", len(learnedSolutions)))
	}

	// If we have learned solutions, try them first
	if len(learnedSolutions) > 0 {
		for i, solution := range learnedSolutions {
			fixOutput.WriteString(fmt.Sprintf("\nTrying learned solution %d: %s\n", i+1, solution.Description))
			fixOutput.WriteString(fmt.Sprintf("Command: %s (Success rate: %d/%d)\n",
				solution.Solution,
				solution.SuccessCount,
				solution.SuccessCount+solution.FailureCount))

			// Execute the solution
			solutionCmd := solution.Solution

			// Execute the solution command
			var execCmd *exec.Cmd
			if agent.DockerConfig != nil && options["use_docker"] != "false" {
				// Execute in Docker
				dockerCmd := fmt.Sprintf("docker run --rm")

				// Add volumes
				for _, volume := range agent.DockerConfig.Volumes {
					dockerCmd += fmt.Sprintf(" -v %s", volume)
				}

				// Add working directory
				if cmd.WorkingDir != "" {
					dockerCmd += fmt.Sprintf(" -w %s", cmd.WorkingDir)
				}

				// Add image and command
				dockerCmd += fmt.Sprintf(" %s:%s /bin/bash -c \"%s\"",
					agent.DockerConfig.Image, agent.DockerConfig.Tag, solutionCmd)

				execCmd = exec.Command("bash", "-c", dockerCmd)
			} else {
				// Execute directly
				execCmd = exec.Command("bash", "-c", solutionCmd)

				// Set working directory
				if cmd.WorkingDir != "" {
					execCmd.Dir = cmd.WorkingDir
				}
			}

			// Capture output
			var stdout, stderr bytes.Buffer
			execCmd.Stdout = &stdout
			execCmd.Stderr = &stderr

			// Execute command
			err := execCmd.Run()

			// Get output
			cmdOutput := stdout.String()
			if stderr.Len() > 0 {
				cmdOutput += "\n" + stderr.String()
			}

			// Log output
			fixOutput.WriteString("Solution output:\n")
			fixOutput.WriteString(cmdOutput)

			// Check if solution succeeded
			if err != nil {
				fixOutput.WriteString(fmt.Sprintf("\nSolution failed: %v\n", err))

				// Record the failure in the learning system
				if errorLearningMgr != nil {
					errorLearningMgr.AddErrorSolution(pattern, solution.Solution, solution.Description, workDir, false, solution.Source)
				}

				continue
			}

			// Solution succeeded
			fixOutput.WriteString("\nSolution succeeded\n")

			// Record the success in the learning system
			if errorLearningMgr != nil {
				errorLearningMgr.AddErrorSolution(pattern, solution.Solution, solution.Description, workDir, true, solution.Source)
			}

			return true, fixOutput.String(), nil
		}
	}

	// Try each predefined solution in order
	for i, solution := range solutions {
		fixOutput.WriteString(fmt.Sprintf("\nTrying solution %d: %s\n", i+1, solution.Description))
		fixOutput.WriteString(fmt.Sprintf("Command: %s\n", solution.Solution))

		// Process the solution
		// Replace variables in the solution
		solutionCmd := solution.Solution

		// Replace ${FILE} with the file that matches FilePattern if specified
		if solution.FilePattern != "" {
			files, err := am.findFilesMatchingPattern(cmd.WorkingDir, solution.FilePattern)
			if err != nil {
				fixOutput.WriteString(fmt.Sprintf("Error finding files matching pattern: %v\n", err))
				continue
			}

			if len(files) == 0 {
				fixOutput.WriteString(fmt.Sprintf("No files found matching pattern: %s\n", solution.FilePattern))
				continue
			}

			// Use the first matching file
			file := files[0]
			solutionCmd = strings.ReplaceAll(solutionCmd, "${FILE}", file)
			fixOutput.WriteString(fmt.Sprintf("Found file matching pattern: %s\n", file))
		}

		// Execute the solution command
		var execCmd *exec.Cmd
		if agent.DockerConfig != nil && options["use_docker"] != "false" {
			// Execute in Docker
			dockerCmd := fmt.Sprintf("docker run --rm")

			// Add volumes
			for _, volume := range agent.DockerConfig.Volumes {
				dockerCmd += fmt.Sprintf(" -v %s", volume)
			}

			// Add working directory
			if cmd.WorkingDir != "" {
				dockerCmd += fmt.Sprintf(" -w %s", cmd.WorkingDir)
			}

			// Add image and command
			dockerCmd += fmt.Sprintf(" %s:%s /bin/bash -c \"%s\"",
				agent.DockerConfig.Image, agent.DockerConfig.Tag, solutionCmd)

			execCmd = exec.Command("bash", "-c", dockerCmd)
		} else {
			// Execute directly
			execCmd = exec.Command("bash", "-c", solutionCmd)

			// Set working directory
			if cmd.WorkingDir != "" {
				execCmd.Dir = cmd.WorkingDir
			}
		}

		// Capture output
		var stdout, stderr bytes.Buffer
		execCmd.Stdout = &stdout
		execCmd.Stderr = &stderr

		// Execute command
		err := execCmd.Run()

		// Get output
		cmdOutput := stdout.String()
		if stderr.Len() > 0 {
			cmdOutput += "\n" + stderr.String()
		}

		// Log output
		fixOutput.WriteString("Solution output:\n")
		fixOutput.WriteString(cmdOutput)

		// Check if solution succeeded
		if err != nil {
			fixOutput.WriteString(fmt.Sprintf("\nSolution failed: %v\n", err))

			// Record the failure in the learning system
			if errorLearningMgr != nil {
				errorLearningMgr.AddErrorSolution(pattern, solutionCmd, solution.Description, workDir, false, "system")
			}

			continue
		}

		// Solution succeeded
		fixOutput.WriteString("\nSolution succeeded\n")

		// Record the success in the learning system
		if errorLearningMgr != nil {
			errorLearningMgr.AddErrorSolution(pattern, solutionCmd, solution.Description, workDir, true, "system")
		}

		return true, fixOutput.String(), nil
	}

	// If we didn't find any solutions or they all failed, try AI-assisted error solving
	if (len(solutions) == 0 || solutions == nil) && am.config.UseAIAssistance && am.aiManager != nil {
		fixOutput.WriteString("No pre-defined solutions found or all failed. Looking for AI-assisted solution...\n")

		// Use AI to analyze the error and suggest solutions
		fixed, aiSolution, err := am.analyzeErrorWithAI(agent, cmd, pattern, output, options)
		if err != nil {
			fixOutput.WriteString(fmt.Sprintf("AI-assisted error analysis failed: %v\n", err))
			return false, fixOutput.String(), nil
		}

		fixOutput.WriteString(aiSolution)
		return fixed, fixOutput.String(), nil
	}

	// If we reach here, none of the solutions worked
	fixOutput.WriteString("\nAll solutions failed\n")
	return false, fixOutput.String(), nil
}

// findFilesMatchingPattern finds files matching a glob pattern
func (am *AgentManager) findFilesMatchingPattern(dir, pattern string) ([]string, error) {
	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory not found: %s", dir)
	}

	// Find files matching pattern
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return nil, err
	}

	return matches, nil
}

// analyzeErrorWithAI uses AI to analyze and fix build errors
func (am *AgentManager) analyzeErrorWithAI(agent *Agent, cmd AgentCommand, pattern, output string, options map[string]string) (bool, string, error) {
	var analysisOutput strings.Builder

	// Check if AI manager is initialized
	if am.aiManager == nil || !am.aiManager.IsEnabled() {
		return false, "AI manager not available or not enabled", fmt.Errorf("AI manager not available")
	}

	// Create context for AI analysis
	workDir := cmd.WorkingDir
	if workDir == "" {
		workDir = "."
	}

	// Extract relevant parts of the error message (last 20 lines if it's very long)
	errorContext := output
	outputLines := strings.Split(output, "\n")
	if len(outputLines) > 20 {
		// Take the last 20 lines for context
		errorContext = strings.Join(outputLines[len(outputLines)-20:], "\n")
	}

	// Create a prompt for the AI to analyze the error
	prompt := fmt.Sprintf(
		`I'm encountering an error while building or running a command in a Docker container or development environment.
Command: %s
Working directory: %s
Error pattern: %s

Error output:
%s

Please analyze this error and suggest 1-3 specific solutions. For each solution:
1. Identify the root cause
2. Provide a specific command or fix to resolve the issue
3. Explain briefly why this solution should work

Focus on practical solutions that can be executed via shell commands.`,
		cmd.Command, workDir, pattern, errorContext,
	)

	// Set up a specialized system prompt for build error analysis
	systemPrompt := `You are an expert build engineer and DevOps specialist. Your task is to diagnose build errors and suggest specific, practical solutions.
Focus on actionable solutions that can be executed via shell commands. Do not suggest general troubleshooting steps.
Format your response with a brief analysis followed by specific commands that can be executed to fix the problem.
Common fix categories include missing dependencies, configuration issues, permission problems, and code errors.`

	// Generate analysis using the AI
	analysis, err := am.aiManager.ollamaClient.Generate(prompt, systemPrompt)
	if err != nil {
		return false, "", fmt.Errorf("failed to generate AI analysis: %v", err)
	}

	// Parse the analysis to extract commands
	commands := extractCommandsFromAnalysis(analysis)

	// Log the analysis
	analysisOutput.WriteString("AI Error Analysis:\n")
	analysisOutput.WriteString("==================\n")
	analysisOutput.WriteString(analysis)
	analysisOutput.WriteString("\n\n")

	// If no commands were found, return the analysis but indicate no fix
	if len(commands) == 0 {
		analysisOutput.WriteString("No executable commands found in the AI analysis.\n")
		return false, analysisOutput.String(), nil
	}

	// Log the commands found
	analysisOutput.WriteString(fmt.Sprintf("Found %d potential fix commands:\n", len(commands)))
	for i, cmd := range commands {
		analysisOutput.WriteString(fmt.Sprintf("%d. %s\n", i+1, cmd))
	}

	// Try each command
	for i, fixCmd := range commands {
		analysisOutput.WriteString(fmt.Sprintf("\nAttempting fix %d: %s\n", i+1, fixCmd))

		// Execute the fix command
		var execCmd *exec.Cmd
		if agent.DockerConfig != nil && options["use_docker"] != "false" {
			// Execute in Docker
			dockerCmd := fmt.Sprintf("docker run --rm")

			// Add volumes
			for _, volume := range agent.DockerConfig.Volumes {
				dockerCmd += fmt.Sprintf(" -v %s", volume)
			}

			// Add working directory
			if cmd.WorkingDir != "" {
				dockerCmd += fmt.Sprintf(" -w %s", cmd.WorkingDir)
			}

			// Add image and command
			dockerCmd += fmt.Sprintf(" %s:%s /bin/bash -c \"%s\"",
				agent.DockerConfig.Image, agent.DockerConfig.Tag, fixCmd)

			execCmd = exec.Command("bash", "-c", dockerCmd)
		} else {
			// Execute directly
			execCmd = exec.Command("bash", "-c", fixCmd)

			// Set working directory
			if cmd.WorkingDir != "" {
				execCmd.Dir = cmd.WorkingDir
			}
		}

		// Capture output
		var stdout, stderr bytes.Buffer
		execCmd.Stdout = &stdout
		execCmd.Stderr = &stderr

		// Execute command
		err := execCmd.Run()

		// Get output
		cmdOutput := stdout.String()
		if stderr.Len() > 0 {
			cmdOutput += "\n" + stderr.String()
		}

		// Log output
		analysisOutput.WriteString("Command output:\n")
		analysisOutput.WriteString(cmdOutput)

		// Check if command succeeded
		if err != nil {
			analysisOutput.WriteString(fmt.Sprintf("\nFix attempt failed: %v\n", err))
			continue
		}

		// Fix command succeeded, now retry the original command
		analysisOutput.WriteString("\nFix command succeeded. Retrying original command...\n")

		// Record this as a successful AI-assisted fix if possible
		// Also add to the inference system for learning
		if am.knowledgeExtractor != nil {
			am.knowledgeExtractor.AddCommand("ai_fix "+fixCmd, workDir, 0)
		}

		// Record the success in the error learning system
		errorLearningMgr := GetErrorLearningManager()
		if errorLearningMgr != nil {
			description := "AI-suggested fix for: " + pattern
			if len(pattern) > 50 {
				description = "AI-suggested fix for: " + pattern[:50] + "..."
			}
			errorLearningMgr.AddErrorSolution(pattern, fixCmd, description, workDir, true, "ai")
		}

		return true, analysisOutput.String(), nil
	}

	// If we get here, none of the fixes worked
	analysisOutput.WriteString("\nAll AI-suggested fixes failed.\n")
	return false, analysisOutput.String(), nil
}

// extractCommandsFromAnalysis extracts shell commands from AI analysis text
func extractCommandsFromAnalysis(analysis string) []string {
	var commands []string

	// Look for commands in backticks, a common formatting for code in AI responses
	backtickPattern := "`([^`]+)`"
	backtickMatches := regexp.MustCompile(backtickPattern).FindAllStringSubmatch(analysis, -1)
	for _, match := range backtickMatches {
		if len(match) > 1 && isLikelyShellCommand(match[1]) {
			commands = append(commands, match[1])
		}
	}

	// Look for commands after "run:" or "command:" prefixes
	linePatterns := []string{
		"run:\\s*(.+)",
		"command:\\s*(.+)",
		"execute:\\s*(.+)",
		"try:\\s*(.+)",
	}

	for _, pattern := range linePatterns {
		re := regexp.MustCompile("(?i)" + pattern)
		matches := re.FindAllStringSubmatch(analysis, -1)
		for _, match := range matches {
			if len(match) > 1 && isLikelyShellCommand(match[1]) {
				commands = append(commands, strings.TrimSpace(match[1]))
			}
		}
	}

	// If we still don't have commands, try to extract code blocks
	if len(commands) == 0 {
		lines := strings.Split(analysis, "\n")
		for _, line := range lines {
			// Look for lines that look like shell commands
			if isLikelyShellCommand(line) && !strings.HasPrefix(line, "#") {
				commands = append(commands, strings.TrimSpace(line))
			}
		}
	}

	// Deduplicate commands
	commandSet := make(map[string]bool)
	var uniqueCommands []string
	for _, cmd := range commands {
		if _, exists := commandSet[cmd]; !exists {
			commandSet[cmd] = true
			uniqueCommands = append(uniqueCommands, cmd)
		}
	}

	return uniqueCommands
}

// isLikelyShellCommand determines if a string is likely a shell command
func isLikelyShellCommand(cmd string) bool {
	// Clean up the command string
	cmd = strings.TrimSpace(cmd)

	// Ignore empty strings and very short strings
	if len(cmd) < 3 {
		return false
	}

	// Ignore markdown formatting and other non-command text
	if strings.HasPrefix(cmd, "#") || strings.HasPrefix(cmd, ">") {
		return false
	}

	// Common command prefixes
	commandPrefixes := []string{
		"apt", "apt-get", "yum", "dnf", "brew", // Package managers
		"npm", "yarn", "pip", "gem", "cargo", // Language package managers
		"docker", "kubectl", "helm", // Container tools
		"git", "svn", "hg", // Version control
		"gcc", "g++", "clang", "make", "cmake", // Build tools
		"cd", "mkdir", "rm", "cp", "mv", "touch", // File operations
		"chmod", "chown", "chgrp", // Permissions
		"echo", "cat", "grep", "sed", "awk", // Text processing
		"find", "ls", "du", "df", // File system
		"curl", "wget", "ssh", "scp", // Network
		"systemctl", "service", // System services
		"./", "sh", "bash", "zsh", // Scripts and shells
		"export", "unset", // Environment variables
	}

	// Check if the command starts with any of the common prefixes
	for _, prefix := range commandPrefixes {
		if strings.HasPrefix(cmd, prefix+" ") || cmd == prefix {
			return true
		}
	}

	// Check for other common command patterns
	patterns := []string{
		"\\w+\\s+-\\w+",   // Command with options (e.g., ls -la)
		"\\w+\\.\\w+\\s+", // Script execution (e.g., script.sh)
		"\\$\\(.+\\)",     // Command substitution
		"\\w+=.+",         // Assignment (e.g., VAR=value)
		"sudo\\s+\\w+",    // Sudo commands
		"\\|\\s*\\w+",     // Pipes
		"&&\\s*\\w+",      // Command chains
		";\\s*\\w+",       // Command sequences
		"^\\./\\w+",       // Executable in current directory
	}

	for _, pattern := range patterns {
		if regexp.MustCompile(pattern).MatchString(cmd) {
			return true
		}
	}

	return false
}

// PatternLibrary represents the error pattern library structure
type PatternLibrary struct {
	Version   string         `json:"version"`
	UpdatedAt string         `json:"updated_at"`
	Patterns  []PatternEntry `json:"patterns"`
}

// PatternEntry represents an entry in the pattern library
type PatternEntry struct {
	Pattern     string `json:"pattern"`
	Solution    string `json:"solution"`
	Description string `json:"description"`
	FilePattern string `json:"file_pattern,omitempty"`
	Category    string `json:"category"`
}

// CommandLibrary represents the common commands library structure
type CommandLibrary struct {
	Version   string              `json:"version"`
	UpdatedAt string              `json:"updated_at"`
	Commands  []AgentCommandEntry `json:"commands"`
}

// AgentCommandEntry represents an entry in the command library
type AgentCommandEntry struct {
	Command     string `json:"command"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

// getGlobalErrorPatterns returns a list of common error patterns and solutions
func (am *AgentManager) getGlobalErrorPatterns() []ErrorSolution {
	// First try to load from external file
	patterns, err := am.loadPatternsFromFile()
	if err == nil && len(patterns) > 0 {
		return patterns
	}

	// Fall back to embedded patterns if external file loading fails
	embeddedPatterns, err := am.loadEmbeddedPatterns()
	if err == nil && len(embeddedPatterns) > 0 {
		return embeddedPatterns
	}

	// Fall back to hardcoded patterns if both external and embedded patterns fail
	return []ErrorSolution{
		{
			Pattern:     "fatal error: .+: No such file or directory",
			Solution:    "apt-get update && apt-get install -y build-essential",
			Description: "Missing build essentials",
		},
		{
			Pattern:     "command not found",
			Solution:    "apt-get update && apt-get install -y ${MISSING_COMMAND}",
			Description: "Missing command",
		},
		{
			Pattern:     "make\\[\\d+\\]: \\*\\*\\* \\[.+\\] Error \\d+",
			Solution:    "make clean && make",
			Description: "Make build error requiring clean",
		},
		{
			Pattern:     "error: unknown type name 'uint'",
			Solution:    "sed -i 's/uint/unsigned int/g' ${FILE}",
			Description: "Type definition error",
			FilePattern: "*.c",
		},
		{
			Pattern:     "permission denied",
			Solution:    "chmod +x ${FILE}",
			Description: "Permission denied",
			FilePattern: "*.sh",
		},
	}
}

// loadPatternsFromFile loads error patterns from the external file
func (am *AgentManager) loadPatternsFromFile() ([]ErrorSolution, error) {
	// Get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}

	// Pattern file path
	patternFilePath := filepath.Join(homeDir, ".delta", "patterns", "error_patterns.json")

	// Check if the file exists
	if _, err := os.Stat(patternFilePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("pattern file does not exist: %s", patternFilePath)
	}

	// Read the pattern file
	data, err := os.ReadFile(patternFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read pattern file: %v", err)
	}

	// Parse the JSON data
	var library PatternLibrary
	if err := json.Unmarshal(data, &library); err != nil {
		return nil, fmt.Errorf("failed to parse pattern library: %v", err)
	}

	// Convert to ErrorSolution format
	var patterns []ErrorSolution
	for _, entry := range library.Patterns {
		patterns = append(patterns, ErrorSolution{
			Pattern:     entry.Pattern,
			Solution:    entry.Solution,
			Description: entry.Description,
			FilePattern: entry.FilePattern,
		})
	}

	return patterns, nil
}

// loadEmbeddedPatterns loads error patterns from the embedded files
func (am *AgentManager) loadEmbeddedPatterns() ([]ErrorSolution, error) {
	// Read the embedded pattern file
	data, err := embeddedPatterns.ReadFile("embedded_patterns/error_patterns.json")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded pattern file: %v", err)
	}

	// Parse the JSON data
	var library PatternLibrary
	if err := json.Unmarshal(data, &library); err != nil {
		return nil, fmt.Errorf("failed to parse embedded pattern library: %v", err)
	}

	// Convert to ErrorSolution format
	var patterns []ErrorSolution
	for _, entry := range library.Patterns {
		patterns = append(patterns, ErrorSolution{
			Pattern:     entry.Pattern,
			Solution:    entry.Solution,
			Description: entry.Description,
			FilePattern: entry.FilePattern,
		})
	}

	return patterns, nil
}

// installDefaultPatterns copies the embedded patterns to the user's .delta directory if they don't exist
func (am *AgentManager) installDefaultPatterns() error {
	// Get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err)
	}

	// Create patterns directory if it doesn't exist
	patternDir := filepath.Join(homeDir, ".delta", "patterns")
	if err := os.MkdirAll(patternDir, 0755); err != nil {
		return fmt.Errorf("failed to create pattern directory: %v", err)
	}

	// Check if error_patterns.json exists
	errorPatternsPath := filepath.Join(patternDir, "error_patterns.json")
	if _, err := os.Stat(errorPatternsPath); os.IsNotExist(err) {
		// Copy embedded pattern file to user's directory
		data, err := embeddedPatterns.ReadFile("embedded_patterns/error_patterns.json")
		if err != nil {
			return fmt.Errorf("failed to read embedded pattern file: %v", err)
		}

		if err := os.WriteFile(errorPatternsPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write pattern file: %v", err)
		}

		fmt.Println("Installed default error patterns to", errorPatternsPath)
	}

	// Check if common_commands.json exists
	commandsPath := filepath.Join(patternDir, "common_commands.json")
	if _, err := os.Stat(commandsPath); os.IsNotExist(err) {
		// Copy embedded commands file to user's directory
		data, err := embeddedPatterns.ReadFile("embedded_patterns/common_commands.json")
		if err != nil {
			return fmt.Errorf("failed to read embedded commands file: %v", err)
		}

		if err := os.WriteFile(commandsPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write commands file: %v", err)
		}

		fmt.Println("Installed default common commands to", commandsPath)
	}

	return nil
}

// Global AgentManager instance
var globalAgentManager *AgentManager

// GetAgentManager returns the global AgentManager instance
func GetAgentManager() *AgentManager {
	if globalAgentManager == nil {
		var err error
		globalAgentManager, err = NewAgentManager()
		if err != nil {
			fmt.Printf("Error initializing agent manager: %v\n", err)
			return nil
		}
	}
	return globalAgentManager
}
