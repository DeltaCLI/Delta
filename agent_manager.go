package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// AgentCommand represents a command sequence for an agent
type AgentCommand struct {
	Command         string              `json:"command"`
	WorkingDir      string              `json:"working_dir"`
	ExpectedOutput  string              `json:"expected_output,omitempty"`
	ErrorPatterns   []string            `json:"error_patterns,omitempty"`
	SuccessPatterns []string            `json:"success_patterns,omitempty"`
	Timeout         int                 `json:"timeout"`
	RetryCount      int                 `json:"retry_count"`
	RetryDelay      int                 `json:"retry_delay"`
	IsInteractive   bool                `json:"is_interactive"`
	Environment     map[string]string   `json:"environment,omitempty"`
}

// AgentDockerConfig represents Docker configuration for an agent
type AgentDockerConfig struct {
	Image           string              `json:"image"`
	Tag             string              `json:"tag"`
	BuildContext    string              `json:"build_context,omitempty"`
	Dockerfile      string              `json:"dockerfile,omitempty"`
	Volumes         []string            `json:"volumes,omitempty"`
	Networks        []string            `json:"networks,omitempty"`
	Ports           []string            `json:"ports,omitempty"`
	Environment     map[string]string   `json:"environment,omitempty"`
	CacheFrom       []string            `json:"cache_from,omitempty"`
	BuildArgs       map[string]string   `json:"build_args,omitempty"`
	UseCache        bool                `json:"use_cache"`
}

// Agent represents a task-specific autonomous agent
type Agent struct {
	ID              string               `json:"id"`
	Name            string               `json:"name"`
	Description     string               `json:"description"`
	TaskTypes       []string             `json:"task_types"`
	Commands        []AgentCommand       `json:"commands"`
	DockerConfig    *AgentDockerConfig   `json:"docker_config,omitempty"`
	TriggerPatterns []string             `json:"trigger_patterns"`
	Context         map[string]string    `json:"context"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
	LastRunAt       time.Time            `json:"last_run_at"`
	RunCount        int                  `json:"run_count"`
	SuccessRate     float64              `json:"success_rate"`
	Tags            []string             `json:"tags"`
	AIPrompt        string               `json:"ai_prompt"`
	Enabled         bool                 `json:"enabled"`
}

// BuildCacheConfig represents cache configuration for a specific build
type BuildCacheConfig struct {
	Name            string               `json:"name"`
	Stages          []string             `json:"stages"`
	DependsOn       []string             `json:"depends_on"`
	CacheVolume     string               `json:"cache_volume"`
	BuildArgs       map[string]string    `json:"build_args"`
	LastBuiltAt     time.Time            `json:"last_built_at"`
	CacheSize       int64                `json:"cache_size"`
	CacheHits       int                  `json:"cache_hits"`
	CacheMisses     int                  `json:"cache_misses"`
}

// AgentManagerConfig contains configuration for the agent manager
type AgentManagerConfig struct {
	Enabled            bool               `json:"enabled"`
	AgentStoragePath   string             `json:"agent_storage_path"`
	CacheStoragePath   string             `json:"cache_storage_path"`
	MaxCacheSize       int64              `json:"max_cache_size"`
	CacheRetention     int                `json:"cache_retention"`
	MaxAgentRuns       int                `json:"max_agent_runs"`
	DefaultTimeout     int                `json:"default_timeout"`
	DefaultRetryCount  int                `json:"default_retry_count"`
	DefaultRetryDelay  int                `json:"default_retry_delay"`
	UseDockerBuilds    bool               `json:"use_docker_builds"`
	UseAIAssistance    bool               `json:"use_ai_assistance"`
	AIPromptTemplate   string             `json:"ai_prompt_template"`
}

// AgentRunResult represents the result of an agent run
type AgentRunResult struct {
	AgentID         string               `json:"agent_id"`
	StartTime       time.Time            `json:"start_time"`
	EndTime         time.Time            `json:"end_time"`
	Success         bool                 `json:"success"`
	ExitCode        int                  `json:"exit_code"`
	Output          string               `json:"output"`
	Errors          []string             `json:"errors"`
	CommandsRun     int                  `json:"commands_run"`
	ArtifactsPaths  []string             `json:"artifacts_paths"`
	PerformanceData map[string]float64   `json:"performance_data"`
}

// DockerBuildCache manages build caching for Docker-based agents
type DockerBuildCache struct {
	CacheDir        string               `json:"cache_dir"`
	CacheSize       int64                `json:"cache_size"`
	MaxCacheAge     time.Duration        `json:"max_cache_age"`
	BuildConfigs    map[string]*BuildCacheConfig `json:"build_configs"`
}

// AgentManager handles the creation, execution, and management of agents
type AgentManager struct {
	config           AgentManagerConfig
	configPath       string
	agents           map[string]*Agent
	dockerCache      *DockerBuildCache
	runHistory       []AgentRunResult
	mutex            sync.RWMutex
	isInitialized    bool
	knowledgeExtractor *KnowledgeExtractor
	aiManager        *AIPredictionManager
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
			DefaultTimeout:    3600,                    // 1 hour
			DefaultRetryCount: 3,
			DefaultRetryDelay: 10,                      // 10 seconds
			UseDockerBuilds:   true,
			UseAIAssistance:   true,
			AIPromptTemplate:  "You are a build assistant for the %s agent. Your task is to %s.",
		},
		configPath:       configPath,
		agents:           make(map[string]*Agent),
		runHistory:       []AgentRunResult{},
		mutex:            sync.RWMutex{},
		isInitialized:    false,
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
		AgentID:        agentID,
		StartTime:      time.Now(),
		Output:         "",
		Errors:         []string{},
		CommandsRun:    0,
		Success:        false,
		PerformanceData: make(map[string]float64),
	}

	// For now, we'll just create a placeholder result
	// In a full implementation, we would execute commands and Docker operations here
	time.Sleep(2 * time.Second) // Simulate agent execution

	result.EndTime = time.Now()
	result.Success = true
	result.CommandsRun = len(agent.Commands)
	result.Output = "Agent execution simulated successfully"
	result.ExitCode = 0

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