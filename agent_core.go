package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// AgentManager manages agent lifecycle and execution
type AgentManager struct {
	agents          map[string]*Agent
	runningAgents   map[string]AgentInterface
	errorHandlers   map[string]ErrorHandler
	dockerManager   *DockerManager
	configPath      string
	storageDir      string
	aiManager       *AIPredictionManager
	maxConcurrent   int
	mutex           sync.RWMutex
	isInitialized   bool
}

// ErrorHandler defines the interface for agent error handlers
type ErrorHandler interface {
	SolveError(ctx context.Context, output string, err error, config CommandConfig) (bool, string, error)
}

// NewAgentManager creates a new agent manager
func NewAgentManager() (*AgentManager, error) {
	// Set up environment
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}

	// Set up agent storage directory
	storageDir := filepath.Join(homeDir, ".config", "delta", "agents")
	err = os.MkdirAll(storageDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create agent storage directory: %v", err)
	}

	// Create agent manager
	am := &AgentManager{
		agents:        make(map[string]*Agent),
		runningAgents: make(map[string]AgentInterface),
		errorHandlers: make(map[string]ErrorHandler),
		configPath:    filepath.Join(storageDir, "agents.json"),
		storageDir:    storageDir,
		aiManager:     GetAIManager(),
		maxConcurrent: 5,
		mutex:         sync.RWMutex{},
		isInitialized: false,
	}

	// Initialize Docker manager
	dockerManager, err := NewDockerManager()
	if err == nil {
		am.dockerManager = dockerManager
	}

	return am, nil
}

// Initialize initializes the agent manager
func (am *AgentManager) Initialize() error {
	// Load agents from storage
	err := am.loadAgents()
	if err != nil {
		return fmt.Errorf("failed to load agents: %v", err)
	}

	// Initialize Docker manager if available
	if am.dockerManager != nil {
		err := am.dockerManager.Initialize()
		if err != nil {
			fmt.Printf("Warning: failed to initialize Docker manager: %v\n", err)
		}
	}

	am.isInitialized = true
	return nil
}

// loadAgents loads agents from storage
func (am *AgentManager) loadAgents() error {
	// Check if agents file exists
	if _, err := os.Stat(am.configPath); os.IsNotExist(err) {
		// No agents file, nothing to load
		return nil
	}

	// Read agents file
	data, err := os.ReadFile(am.configPath)
	if err != nil {
		return fmt.Errorf("failed to read agents file: %v", err)
	}

	// Parse agents
	var agents []*Agent
	err = json.Unmarshal(data, &agents)
	if err != nil {
		return fmt.Errorf("failed to parse agents file: %v", err)
	}

	// Add agents to map
	for _, agent := range agents {
		am.agents[agent.ID] = agent
	}

	return nil
}

// saveAgents saves agents to storage
func (am *AgentManager) saveAgents() error {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	// Convert agents map to slice
	agents := make([]*Agent, 0, len(am.agents))
	for _, agent := range am.agents {
		agents = append(agents, agent)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(agents, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal agents: %v", err)
	}

	// Write to file
	err = os.WriteFile(am.configPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write agents file: %v", err)
	}

	return nil
}

// CreateAgent creates a new agent
func (am *AgentManager) CreateAgent(agent *Agent) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	// Validate agent
	if agent.ID == "" {
		return fmt.Errorf("agent ID cannot be empty")
	}

	// Check if agent already exists
	if _, exists := am.agents[agent.ID]; exists {
		return fmt.Errorf("agent with ID %s already exists", agent.ID)
	}

	// Set timestamps
	now := time.Now()
	agent.CreatedAt = now
	agent.UpdatedAt = now

	// Add agent to map
	am.agents[agent.ID] = agent

	// Save agents
	return am.saveAgents()
}

// UpdateAgent updates an existing agent
func (am *AgentManager) UpdateAgent(agent *Agent) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	// Check if agent exists
	if _, exists := am.agents[agent.ID]; !exists {
		return fmt.Errorf("agent with ID %s does not exist", agent.ID)
	}

	// Update timestamp
	agent.UpdatedAt = time.Now()

	// Update agent in map
	am.agents[agent.ID] = agent

	// Save agents
	return am.saveAgents()
}

// DeleteAgent deletes an agent
func (am *AgentManager) DeleteAgent(id string) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	// Check if agent exists
	if _, exists := am.agents[id]; !exists {
		return fmt.Errorf("agent with ID %s does not exist", id)
	}

	// Check if agent is running
	if _, running := am.runningAgents[id]; running {
		return fmt.Errorf("cannot delete agent %s, it is currently running", id)
	}

	// Delete agent from map
	delete(am.agents, id)

	// Save agents
	return am.saveAgents()
}

// GetAgent returns an agent by ID
func (am *AgentManager) GetAgent(id string) (*Agent, error) {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	// Check if agent exists
	agent, exists := am.agents[id]
	if !exists {
		return nil, fmt.Errorf("agent with ID %s does not exist", id)
	}

	return agent, nil
}

// ListAgents returns all agents
func (am *AgentManager) ListAgents() ([]*Agent, error) {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	// Convert agents map to slice
	agents := make([]*Agent, 0, len(am.agents))
	for _, agent := range am.agents {
		agents = append(agents, agent)
	}

	return agents, nil
}

// RunAgent runs an agent
func (am *AgentManager) RunAgent(ctx context.Context, id string, options map[string]string) error {
	// Get agent
	agent, err := am.GetAgent(id)
	if err != nil {
		return err
	}

	// Check if agent is enabled
	if !agent.Enabled {
		return fmt.Errorf("agent %s is disabled", id)
	}

	// Create agent implementation
	impl, err := am.createAgentImplementation(agent)
	if err != nil {
		return fmt.Errorf("failed to create agent implementation: %v", err)
	}

	// Add to running agents
	am.mutex.Lock()
	am.runningAgents[id] = impl
	am.mutex.Unlock()

	// Execute agent in a goroutine
	go func() {
		// Run agent
		err := impl.Run(ctx)

		// Remove from running agents when done
		am.mutex.Lock()
		delete(am.runningAgents, id)
		am.mutex.Unlock()

		// Update agent metrics
		am.mutex.Lock()
		agent.LastRunAt = time.Now()
		agent.RunCount++
		// Update success rate
		if err == nil {
			successRuns := float64(agent.RunCount) * agent.SuccessRate
			agent.SuccessRate = (successRuns + 1) / float64(agent.RunCount)
		} else {
			successRuns := float64(agent.RunCount-1) * agent.SuccessRate
			agent.SuccessRate = successRuns / float64(agent.RunCount)
		}
		am.saveAgents()
		am.mutex.Unlock()
	}()

	return nil
}

// StopAgent stops a running agent
func (am *AgentManager) StopAgent(id string) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	// Check if agent is running
	impl, running := am.runningAgents[id]
	if !running {
		return fmt.Errorf("agent %s is not running", id)
	}

	// Stop agent
	err := impl.Stop()
	if err != nil {
		return fmt.Errorf("failed to stop agent: %v", err)
	}

	// Remove from running agents
	delete(am.runningAgents, id)

	return nil
}

// GetAgentStatus returns the status of an agent
func (am *AgentManager) GetAgentStatus(id string) (AgentStatus, error) {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	// Check if agent exists
	if _, exists := am.agents[id]; !exists {
		return AgentStatus{}, fmt.Errorf("agent with ID %s does not exist", id)
	}

	// Check if agent is running
	impl, running := am.runningAgents[id]
	if !running {
		// Return static status for non-running agent
		return AgentStatus{
			ID:        id,
			IsRunning: false,
		}, nil
	}

	// Get status from implementation
	return impl.GetStatus(), nil
}

// EnableAgent enables an agent
func (am *AgentManager) EnableAgent(id string) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	// Check if agent exists
	agent, exists := am.agents[id]
	if !exists {
		return fmt.Errorf("agent with ID %s does not exist", id)
	}

	// Enable agent
	agent.Enabled = true
	agent.UpdatedAt = time.Now()

	// Save agents
	return am.saveAgents()
}

// DisableAgent disables an agent
func (am *AgentManager) DisableAgent(id string) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	// Check if agent exists
	agent, exists := am.agents[id]
	if !exists {
		return fmt.Errorf("agent with ID %s does not exist", id)
	}

	// Check if agent is running
	if _, running := am.runningAgents[id]; running {
		return fmt.Errorf("cannot disable agent %s, it is currently running", id)
	}

	// Disable agent
	agent.Enabled = false
	agent.UpdatedAt = time.Now()

	// Save agents
	return am.saveAgents()
}

// FindAgentsByContext finds agents by context key/value
func (am *AgentManager) FindAgentsByContext(key, value string) []*Agent {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	// Filter agents by context
	var result []*Agent
	for _, agent := range am.agents {
		if v, exists := agent.Context[key]; exists && v == value {
			result = append(result, agent)
		}
	}

	return result
}

// HasAgents returns true if there are any agents
func (am *AgentManager) HasAgents() bool {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	return len(am.agents) > 0
}

// CreateAgentFromTemplate creates a new agent from a template
func (am *AgentManager) CreateAgentFromTemplate(id string, template []byte, vars map[string]string) (*Agent, error) {
	// Parse template
	var agent Agent
	err := json.Unmarshal(template, &agent)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %v", err)
	}

	// Update ID
	agent.ID = id

	// Process variables
	for k, v := range vars {
		// Replace variables in strings
		varPattern := fmt.Sprintf("${%s}", k)
		agent.Name = strings.ReplaceAll(agent.Name, varPattern, v)
		agent.Description = strings.ReplaceAll(agent.Description, varPattern, v)
		
		// Process context
		for ck, cv := range agent.Context {
			agent.Context[ck] = strings.ReplaceAll(cv, varPattern, v)
		}
		
		// Process commands
		for i, cmd := range agent.Commands {
			cmd.Command = strings.ReplaceAll(cmd.Command, varPattern, v)
			cmd.WorkingDir = strings.ReplaceAll(cmd.WorkingDir, varPattern, v)
			agent.Commands[i] = cmd
		}
		
		// Process Docker config
		if agent.DockerConfig != nil {
			agent.DockerConfig.BuildContext = strings.ReplaceAll(agent.DockerConfig.BuildContext, varPattern, v)
			agent.DockerConfig.ComposeFile = strings.ReplaceAll(agent.DockerConfig.ComposeFile, varPattern, v)
		}
	}

	// Create agent
	err = am.CreateAgent(&agent)
	if err != nil {
		return nil, err
	}

	return &agent, nil
}

// createAgentImplementation creates a concrete agent implementation
func (am *AgentManager) createAgentImplementation(agent *Agent) (AgentInterface, error) {
	// Check if agent has Docker configuration
	if agent.DockerConfig != nil && agent.DockerConfig.Enabled {
		// Create Docker agent
		dockerAgent, err := createDockerAgentImplementation(agent)
		if err != nil {
			return nil, fmt.Errorf("failed to create Docker agent: %v", err)
		}
		return dockerAgent, nil
	}
	
	// Default to generic agent
	return &GenericAgent{
		config:     agent,
		manager:    am,
		isRunning:  false,
		mutex:      sync.RWMutex{},
	}, nil
}

// DockerManager manages Docker operations for agents
type DockerManager struct {
	available       bool
	cacheDir        string
	mutex           sync.RWMutex
}

// NewDockerManager creates a new Docker manager
func NewDockerManager() (*DockerManager, error) {
	// Check if Docker is available
	_, err := exec.LookPath("docker")
	if err != nil {
		return &DockerManager{
			available: false,
		}, fmt.Errorf("Docker not found: %v", err)
	}

	// Set up cache directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}

	cacheDir := filepath.Join(homeDir, ".config", "delta", "docker", "cache")
	err = os.MkdirAll(cacheDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker cache directory: %v", err)
	}

	return &DockerManager{
		available: true,
		cacheDir:  cacheDir,
		mutex:     sync.RWMutex{},
	}, nil
}

// Initialize initializes the Docker manager
func (dm *DockerManager) Initialize() error {
	// Check Docker version
	cmd := exec.Command("docker", "version", "--format", "{{.Server.Version}}")
	output, err := cmd.CombinedOutput()
	if err != nil {
		dm.available = false
		return fmt.Errorf("failed to get Docker version: %v", err)
	}

	version := strings.TrimSpace(string(output))
	fmt.Printf("Docker version: %s\n", version)

	dm.available = true
	return nil
}

// IsAvailable returns true if Docker is available
func (dm *DockerManager) IsAvailable() bool {
	return dm.available
}

// RunDockerCompose runs Docker Compose
func (dm *DockerManager) RunDockerCompose(ctx context.Context, options DockerComposeOptions) error {
	if !dm.available {
		return fmt.Errorf("Docker is not available")
	}

	// Build Docker Compose command
	args := []string{"compose"}
	
	// Add file if specified
	if options.File != "" {
		args = append(args, "-f", options.File)
	}
	
	// Add project name if specified
	if options.ProjectName != "" {
		args = append(args, "-p", options.ProjectName)
	}
	
	// Add up command
	args = append(args, "up")
	
	// Add options
	if options.Detached {
		args = append(args, "-d")
	}
	
	if options.RemoveOrphans {
		args = append(args, "--remove-orphans")
	}
	
	if options.ForceRecreate {
		args = append(args, "--force-recreate")
	}
	
	if options.NoBuild {
		args = append(args, "--no-build")
	}
	
	// Add services
	args = append(args, options.Services...)
	
	// Create environment
	env := os.Environ()
	for k, v := range options.Environment {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	
	// Create command
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	// Run command
	return cmd.Run()
}

// CleanupDockerCompose cleans up Docker Compose resources
func (dm *DockerManager) CleanupDockerCompose(ctx context.Context, options DockerComposeOptions) error {
	if !dm.available {
		return fmt.Errorf("Docker is not available")
	}

	// Build Docker Compose command
	args := []string{"compose"}
	
	// Add file if specified
	if options.File != "" {
		args = append(args, "-f", options.File)
	}
	
	// Add project name if specified
	if options.ProjectName != "" {
		args = append(args, "-p", options.ProjectName)
	}
	
	// Add down command
	args = append(args, "down")
	
	// Add options
	if options.RemoveOrphans {
		args = append(args, "--remove-orphans")
	}
	
	// Create command
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	// Run command
	return cmd.Run()
}

// ExecuteCommand executes a command
func ExecuteCommand(ctx context.Context, config CommandConfig) (string, error) {
	// Parse command
	cmdParts := strings.Fields(config.Command)
	if len(cmdParts) == 0 {
		return "", fmt.Errorf("empty command")
	}

	// Build command
	cmd := exec.CommandContext(ctx, cmdParts[0], cmdParts[1:]...)
	
	// Set working directory
	if config.WorkingDir != "" {
		cmd.Dir = config.WorkingDir
	}
	
	// Set environment
	if config.Environment != nil {
		env := os.Environ()
		for k, v := range config.Environment {
			env = append(env, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = env
	}
	
	// Run command
	output, err := cmd.CombinedOutput()
	
	// Check for timeout
	if ctx.Err() == context.DeadlineExceeded {
		return string(output), fmt.Errorf("command timed out after %d seconds", config.Timeout)
	}
	
	// Check for error
	if err != nil {
		// Check error patterns
		if isErrorMatch(string(output), config.ErrorPatterns) {
			return string(output), fmt.Errorf("command failed: %v", err)
		}
	}
	
	// Check success patterns
	if isSuccessMatch(string(output), config.SuccessPatterns) {
		return string(output), nil
	}
	
	return string(output), err
}

// isErrorMatch checks if output matches any error patterns
func isErrorMatch(output string, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}
	
	for _, pattern := range patterns {
		match, err := regexp.MatchString(pattern, output)
		if err == nil && match {
			return true
		}
	}
	
	return false
}

// isSuccessMatch checks if output matches any success patterns
func isSuccessMatch(output string, patterns []string) bool {
	if len(patterns) == 0 {
		return true
	}
	
	for _, pattern := range patterns {
		match, err := regexp.MatchString(pattern, output)
		if err == nil && match {
			return true
		}
	}
	
	return false
}

// GenericAgent implements a generic agent
type GenericAgent struct {
	config          *Agent
	manager         *AgentManager
	isRunning       bool
	startTime       time.Time
	endTime         time.Time
	currentCommand  string
	lastError       error
	progress        float64
	successCount    int
	errorCount      int
	lastOutput      string
	cancelFunc      context.CancelFunc
	mutex           sync.RWMutex
}

// Initialize initializes the agent
func (a *GenericAgent) Initialize() error {
	return nil
}

// Run executes the agent
func (a *GenericAgent) Run(ctx context.Context) error {
	a.mutex.Lock()
	if a.isRunning {
		a.mutex.Unlock()
		return fmt.Errorf("agent is already running")
	}

	// Create a cancelable context
	runCtx, cancel := context.WithCancel(ctx)
	a.cancelFunc = cancel

	a.isRunning = true
	a.startTime = time.Now()
	a.progress = 0
	a.successCount = 0
	a.errorCount = 0
	a.lastOutput = ""
	a.lastError = nil
	a.mutex.Unlock()

	// Ensure cleanup when done
	defer func() {
		a.mutex.Lock()
		a.isRunning = false
		a.endTime = time.Now()
		if a.cancelFunc != nil {
			a.cancelFunc = nil
		}
		a.mutex.Unlock()
	}()

	// Execute commands
	for i, cmd := range a.config.Commands {
		// Skip disabled commands
		if !cmd.Enabled {
			continue
		}

		// Update current command
		a.mutex.Lock()
		a.currentCommand = cmd.Command
		a.progress = float64(i) / float64(len(a.config.Commands))
		a.mutex.Unlock()

		// Create command configuration
		cmdConfig := CommandConfig{
			Command:         cmd.Command,
			WorkingDir:      cmd.WorkingDir,
			Timeout:         cmd.Timeout,
			RetryCount:      cmd.RetryCount,
			RetryDelay:      cmd.RetryDelay,
			ErrorPatterns:   cmd.ErrorPatterns,
			SuccessPatterns: cmd.SuccessPatterns,
			Environment:     cmd.Environment,
			IsInteractive:   cmd.IsInteractive,
		}

		// Execute command
		output, err := ExecuteCommand(runCtx, cmdConfig)

		// Update last output
		a.mutex.Lock()
		a.lastOutput = output
		a.mutex.Unlock()

		// Handle error
		if err != nil {
			a.mutex.Lock()
			a.lastError = err
			a.errorCount++
			a.mutex.Unlock()

			return fmt.Errorf("command %s failed: %v", cmd.Command, err)
		}

		// Increment success count
		a.mutex.Lock()
		a.successCount++
		a.mutex.Unlock()
	}

	// Set progress to 100%
	a.mutex.Lock()
	a.progress = 1.0
	a.mutex.Unlock()

	return nil
}

// Stop stops the agent execution
func (a *GenericAgent) Stop() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if !a.isRunning {
		return nil
	}

	// Cancel context
	if a.cancelFunc != nil {
		a.cancelFunc()
	}

	a.isRunning = false
	a.endTime = time.Now()

	return nil
}

// GetStatus returns the current status of the agent
func (a *GenericAgent) GetStatus() AgentStatus {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	status := AgentStatus{
		ID:             a.config.ID,
		IsRunning:      a.isRunning,
		StartTime:      a.startTime,
		EndTime:        a.endTime,
		LastError:      a.lastError,
		CurrentCommand: a.currentCommand,
		Progress:       a.progress,
		SuccessCount:   a.successCount,
		ErrorCount:     a.errorCount,
		LastOutput:     a.lastOutput,
	}

	if a.isRunning {
		status.RunDuration = time.Since(a.startTime)
	} else if !a.endTime.IsZero() {
		status.RunDuration = a.endTime.Sub(a.startTime)
	}

	return status
}

// GetID returns the agent ID
func (a *GenericAgent) GetID() string {
	return a.config.ID
}

// Global agent manager instance
var globalAgentManager *AgentManager

// GetAgentManager returns the global agent manager instance
func GetAgentManager() *AgentManager {
	if globalAgentManager == nil {
		var err error
		globalAgentManager, err = NewAgentManager()
		if err != nil {
			fmt.Printf("Error initializing agent manager: %v\n", err)
			return nil
		}

		err = globalAgentManager.Initialize()
		if err != nil {
			fmt.Printf("Error initializing agent manager: %v\n", err)
		}
	}

	return globalAgentManager
}