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

// createAgentImplementation creates a concrete agent implementation
func createAgentImplementation(agent *Agent, manager *AgentManager) (AgentInterface, error) {
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
		manager:    manager,
		isRunning:  false,
		mutex:      sync.RWMutex{},
	}, nil
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