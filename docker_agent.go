package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// DockerAgent implements the AgentInterface for Docker-based agents
type DockerAgent struct {
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
	tempDirs        []string
	containerId     string
}

// Initialize initializes the Docker agent
func (a *DockerAgent) Initialize() error {
	// Check if Docker is available
	_, err := exec.LookPath("docker")
	if err != nil {
		return fmt.Errorf("docker not found: %v", err)
	}

	return nil
}

// Run executes the Docker agent
func (a *DockerAgent) Run(ctx context.Context) error {
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
		
		// Cleanup temporary directories
		for _, dir := range a.tempDirs {
			os.RemoveAll(dir)
		}
		a.tempDirs = nil
		
		// Remove container if it exists
		if a.containerId != "" {
			exec.Command("docker", "rm", "-f", a.containerId).Run()
			a.containerId = ""
		}
		
		a.mutex.Unlock()
	}()

	// Check if Docker config is present
	if a.config.DockerConfig == nil {
		return fmt.Errorf("agent does not have Docker configuration")
	}

	// Prepare Docker configuration
	a.mutex.Lock()
	a.currentCommand = "Preparing Docker environment"
	a.progress = 0.1
	a.mutex.Unlock()

	// Create a temporary directory for build context if needed
	if a.config.DockerConfig.BuildContext != "" {
		tempDir, err := os.MkdirTemp("", "delta-agent-docker-")
		if err != nil {
			return fmt.Errorf("failed to create temporary directory: %v", err)
		}
		a.tempDirs = append(a.tempDirs, tempDir)

		// Copy build context to temporary directory
		err = copyDirectory(a.config.DockerConfig.BuildContext, tempDir)
		if err != nil {
			return fmt.Errorf("failed to copy build context: %v", err)
		}

		// Update build context path
		a.config.DockerConfig.BuildContext = tempDir
	}

	// Build or pull the Docker image
	a.mutex.Lock()
	a.currentCommand = fmt.Sprintf("Building Docker image: %s:%s", a.config.DockerConfig.Image, a.config.DockerConfig.Tag)
	a.progress = 0.2
	a.mutex.Unlock()

	var dockerImage string
	if a.config.DockerConfig.BuildContext != "" {
		// Build the image
		buildArgs := []string{"build"}
		
		// Add build args
		if a.config.DockerConfig.BuildArgs != nil {
			for k, v := range a.config.DockerConfig.BuildArgs {
				buildArgs = append(buildArgs, "--build-arg", fmt.Sprintf("%s=%s", k, v))
			}
		}
		
		// Add cache options
		if a.config.DockerConfig.UseCache {
			// Use cache
			if len(a.config.DockerConfig.CacheFrom) > 0 {
				for _, cacheFrom := range a.config.DockerConfig.CacheFrom {
					buildArgs = append(buildArgs, "--cache-from", cacheFrom)
				}
			}
		} else {
			// No cache
			buildArgs = append(buildArgs, "--no-cache")
		}
		
		// Add tag
		dockerImage = fmt.Sprintf("%s:%s", a.config.DockerConfig.Image, a.config.DockerConfig.Tag)
		buildArgs = append(buildArgs, "-t", dockerImage)
		
		// Add Dockerfile if specified
		if a.config.DockerConfig.Dockerfile != "" {
			buildArgs = append(buildArgs, "-f", a.config.DockerConfig.Dockerfile)
		}
		
		// Add build context
		buildArgs = append(buildArgs, a.config.DockerConfig.BuildContext)
		
		// Run build command
		cmd := exec.CommandContext(runCtx, "docker", buildArgs...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			a.mutex.Lock()
			a.lastOutput = string(output)
			a.lastError = fmt.Errorf("failed to build Docker image: %v", err)
			a.errorCount++
			a.mutex.Unlock()
			return a.lastError
		}
		
		a.mutex.Lock()
		a.lastOutput = string(output)
		a.progress = 0.3
		a.successCount++
		a.mutex.Unlock()
	} else {
		// Pull the image
		dockerImage = fmt.Sprintf("%s:%s", a.config.DockerConfig.Image, a.config.DockerConfig.Tag)
		cmd := exec.CommandContext(runCtx, "docker", "pull", dockerImage)
		output, err := cmd.CombinedOutput()
		if err != nil {
			a.mutex.Lock()
			a.lastOutput = string(output)
			a.lastError = fmt.Errorf("failed to pull Docker image: %v", err)
			a.errorCount++
			a.mutex.Unlock()
			return a.lastError
		}
		
		a.mutex.Lock()
		a.lastOutput = string(output)
		a.progress = 0.3
		a.successCount++
		a.mutex.Unlock()
	}

	// Execute commands in the Docker container
	for i, cmd := range a.config.Commands {
		// Skip disabled commands
		if !cmd.Enabled {
			continue
		}

		// Update current command
		a.mutex.Lock()
		a.currentCommand = cmd.Command
		a.progress = 0.3 + (0.7 * float64(i) / float64(len(a.config.Commands)))
		a.mutex.Unlock()

		// Create docker run command
		runArgs := []string{"run", "--rm"}
		
		// Add environment variables
		if cmd.Environment != nil {
			for k, v := range cmd.Environment {
				runArgs = append(runArgs, "-e", fmt.Sprintf("%s=%s", k, v))
			}
		}
		
		// Add volumes
		if a.config.DockerConfig.Volumes != nil {
			for _, volume := range a.config.DockerConfig.Volumes {
				runArgs = append(runArgs, "-v", volume)
			}
		}
		
		// Add interactive flag if needed
		if cmd.IsInteractive {
			runArgs = append(runArgs, "-i", "-t")
		}
		
		// Add working directory
		if cmd.WorkingDir != "" {
			runArgs = append(runArgs, "-w", cmd.WorkingDir)
		}
		
		// Add image
		runArgs = append(runArgs, dockerImage)
		
		// Add command
		runArgs = append(runArgs, "/bin/sh", "-c", cmd.Command)
		
		// Create command with timeout
		var cmdCtx context.Context
		var cmdCancel context.CancelFunc
		
		if cmd.Timeout > 0 {
			cmdCtx, cmdCancel = context.WithTimeout(runCtx, time.Duration(cmd.Timeout)*time.Second)
		} else {
			cmdCtx, cmdCancel = context.WithCancel(runCtx)
		}
		
		execCmd := exec.CommandContext(cmdCtx, "docker", runArgs...)
		output, err := execCmd.CombinedOutput()

		// Cleanup
		cmdCancel()
		
		// Check for timeout
		if cmdCtx.Err() == context.DeadlineExceeded {
			a.mutex.Lock()
			a.lastOutput = string(output)
			a.lastError = fmt.Errorf("command timed out after %d seconds", cmd.Timeout)
			a.errorCount++
			a.mutex.Unlock()
			return a.lastError
		}
		
		// Check for command error
		if err != nil {
			// Check retry count
			if cmd.RetryCount > 0 {
				// Retry command
				retrySuccess := false
				for retry := 0; retry < cmd.RetryCount; retry++ {
					// Wait before retry
					if cmd.RetryDelay > 0 {
						select {
						case <-time.After(time.Duration(cmd.RetryDelay) * time.Second):
						case <-runCtx.Done():
							return runCtx.Err()
						}
					}
					
					// Create new context for retry
					var retryCtx context.Context
					var retryCancel context.CancelFunc
					
					if cmd.Timeout > 0 {
						retryCtx, retryCancel = context.WithTimeout(runCtx, time.Duration(cmd.Timeout)*time.Second)
					} else {
						retryCtx, retryCancel = context.WithCancel(runCtx)
					}
					
					// Execute retry
					retryCmd := exec.CommandContext(retryCtx, "docker", runArgs...)
					retryOutput, retryErr := retryCmd.CombinedOutput()
					
					// Cleanup
					retryCancel()
					
					// Check result
					if retryErr == nil {
						// Retry succeeded
						output = retryOutput
						err = nil
						retrySuccess = true
						break
					}
					
					// Update output for next retry
					output = retryOutput
				}
				
				// Check if retry succeeded
				if !retrySuccess {
					a.mutex.Lock()
					a.lastOutput = string(output)
					a.lastError = fmt.Errorf("command failed after %d retries: %v", cmd.RetryCount, err)
					a.errorCount++
					a.mutex.Unlock()
					return a.lastError
				}
			} else {
				// No retry, just fail
				a.mutex.Lock()
				a.lastOutput = string(output)
				a.lastError = fmt.Errorf("command failed: %v", err)
				a.errorCount++
				a.mutex.Unlock()
				return a.lastError
			}
		}
		
		// Update last output
		a.mutex.Lock()
		a.lastOutput = string(output)
		a.successCount++
		a.mutex.Unlock()
	}

	// Set progress to 100%
	a.mutex.Lock()
	a.progress = 1.0
	a.mutex.Unlock()

	return nil
}

// Stop stops the Docker agent execution
func (a *DockerAgent) Stop() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	if !a.isRunning {
		return nil
	}

	// Cancel context
	if a.cancelFunc != nil {
		a.cancelFunc()
	}

	// Stop container if running
	if a.containerId != "" {
		exec.Command("docker", "stop", a.containerId).Run()
	}

	a.isRunning = false
	a.endTime = time.Now()

	return nil
}

// GetStatus returns the current status of the Docker agent
func (a *DockerAgent) GetStatus() AgentStatus {
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

// GetID returns the Docker agent ID
func (a *DockerAgent) GetID() string {
	return a.config.ID
}

// copyDirectory copies a directory recursively
func copyDirectory(src, dst string) error {
	// Get source info
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Create destination directory
	err = os.MkdirAll(dst, srcInfo.Mode())
	if err != nil {
		return err
	}

	// Read directory entries
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	// Copy each entry
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			// Recursive copy for directory
			err = copyDirectory(srcPath, dstPath)
			if err != nil {
				return err
			}
		} else {
			// Copy file
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}

			err = os.WriteFile(dstPath, data, 0644)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// WaterfallDockerAgent implements a multi-stage waterfall build agent
type WaterfallDockerAgent struct {
	DockerAgent
	waterfallStages map[string]*waterfallStageStatus
}

// waterfallStageStatus tracks the status of a waterfall stage
type waterfallStageStatus struct {
	name       string
	status     string // "pending", "running", "completed", "failed"
	startTime  time.Time
	endTime    time.Time
	dependencies []string
	dependents []string
}

// Initialize initializes the waterfall Docker agent
func (a *WaterfallDockerAgent) Initialize() error {
	err := a.DockerAgent.Initialize()
	if err != nil {
		return err
	}

	// Initialize waterfall stages
	a.waterfallStages = make(map[string]*waterfallStageStatus)
	if a.config.DockerConfig != nil && a.config.DockerConfig.Waterfall != nil {
		for _, stage := range a.config.DockerConfig.Waterfall.Stages {
			a.waterfallStages[stage] = &waterfallStageStatus{
				name:       stage,
				status:     "pending",
				dependencies: a.config.DockerConfig.Waterfall.Dependencies[stage],
			}
		}

		// Set up dependent relationships
		for stage, status := range a.waterfallStages {
			for _, dep := range status.dependencies {
				if depStatus, exists := a.waterfallStages[dep]; exists {
					depStatus.dependents = append(depStatus.dependents, stage)
				}
			}
		}
	}

	return nil
}

// Run executes the waterfall Docker agent
func (a *WaterfallDockerAgent) Run(ctx context.Context) error {
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
		
		// Cleanup temporary directories
		for _, dir := range a.tempDirs {
			os.RemoveAll(dir)
		}
		a.tempDirs = nil
		
		a.mutex.Unlock()
	}()

	// Check if Docker config with waterfall is present
	if a.config.DockerConfig == nil || a.config.DockerConfig.Waterfall == nil {
		return fmt.Errorf("agent does not have waterfall configuration")
	}

	// Prepare all stages
	a.mutex.Lock()
	a.currentCommand = "Preparing waterfall build"
	a.progress = 0.1
	a.mutex.Unlock()

	// Build all stages
	completed := make(map[string]bool)
	stagesCompleted := 0
	totalStages := len(a.config.DockerConfig.Waterfall.Stages)

	// Process stages until all are built or an error occurs
	for stagesCompleted < totalStages {
		// Find next set of stages that can be built in parallel
		var readyStages []string
		for _, stage := range a.config.DockerConfig.Waterfall.Stages {
			if completed[stage] {
				continue
			}

			// Check if all dependencies are satisfied
			depsReady := true
			for _, dep := range a.waterfallStages[stage].dependencies {
				if !completed[dep] {
					depsReady = false
					break
				}
			}

			if depsReady {
				readyStages = append(readyStages, stage)
			}
		}

		// Check if we can make progress
		if len(readyStages) == 0 {
			return fmt.Errorf("circular dependency detected in waterfall build")
		}

		// Build each ready stage in parallel
		var wg sync.WaitGroup
		stageErrors := make(map[string]error)
		stageOutputs := make(map[string]string)

		for _, stage := range readyStages {
			wg.Add(1)
			go func(stageName string) {
				defer wg.Done()
				
				// Update stage status
				a.waterfallStages[stageName].status = "running"
				a.waterfallStages[stageName].startTime = time.Now()
				
				a.mutex.Lock()
				a.currentCommand = fmt.Sprintf("Building stage: %s", stageName)
				a.mutex.Unlock()
				
				// Build Docker image for this stage
				output, err := a.buildStage(runCtx, stageName)
				
				a.mutex.Lock()
				stageOutputs[stageName] = output
				if err != nil {
					stageErrors[stageName] = err
					a.waterfallStages[stageName].status = "failed"
					a.errorCount++
				} else {
					a.waterfallStages[stageName].status = "completed"
					a.successCount++
				}
				a.waterfallStages[stageName].endTime = time.Now()
				a.mutex.Unlock()
			}(stage)
		}

		// Wait for all stages to complete
		wg.Wait()

		// Check for errors
		var errorMsgs []string
		for stage, err := range stageErrors {
			errorMsgs = append(errorMsgs, fmt.Sprintf("Stage %s: %v", stage, err))
			a.lastOutput += fmt.Sprintf("\n--- Stage %s Output ---\n%s", stage, stageOutputs[stage])
		}

		if len(errorMsgs) > 0 {
			a.mutex.Lock()
			a.lastError = fmt.Errorf("errors in waterfall build: %s", strings.Join(errorMsgs, "; "))
			a.mutex.Unlock()
			return a.lastError
		}

		// Mark all ready stages as completed
		for _, stage := range readyStages {
			completed[stage] = true
			stagesCompleted++
		}

		// Update progress
		a.mutex.Lock()
		a.progress = 0.1 + (0.9 * float64(stagesCompleted) / float64(totalStages))
		a.mutex.Unlock()
	}

	return nil
}

// buildStage builds a single stage in the waterfall
func (a *WaterfallDockerAgent) buildStage(ctx context.Context, stageName string) (string, error) {
	// Build Docker image for this stage
	buildArgs := []string{"build"}
	
	// Add build args
	if a.config.DockerConfig.BuildArgs != nil {
		for k, v := range a.config.DockerConfig.BuildArgs {
			buildArgs = append(buildArgs, "--build-arg", fmt.Sprintf("%s=%s", k, v))
		}
	}
	
	// Add stage-specific args
	buildArgs = append(buildArgs, "--target", stageName)
	
	// Add cache options
	if a.config.DockerConfig.UseCache {
		// Use cache
		if len(a.config.DockerConfig.CacheFrom) > 0 {
			for _, cacheFrom := range a.config.DockerConfig.CacheFrom {
				buildArgs = append(buildArgs, "--cache-from", cacheFrom)
			}
		}
	} else {
		// No cache
		buildArgs = append(buildArgs, "--no-cache")
	}
	
	// Add tag
	stageTag := fmt.Sprintf("%s:%s-%s", a.config.DockerConfig.Image, a.config.DockerConfig.Tag, stageName)
	buildArgs = append(buildArgs, "-t", stageTag)
	
	// Add Dockerfile if specified
	if a.config.DockerConfig.Dockerfile != "" {
		buildArgs = append(buildArgs, "-f", a.config.DockerConfig.Dockerfile)
	}
	
	// Add build context
	buildArgs = append(buildArgs, a.config.DockerConfig.BuildContext)
	
	// Run build command
	cmd := exec.CommandContext(ctx, "docker", buildArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("failed to build stage %s: %v", stageName, err)
	}
	
	return string(output), nil
}

// createDockerAgentImplementation creates a Docker agent implementation
func createDockerAgentImplementation(agent *Agent) (AgentInterface, error) {
	// Check if Docker config is present
	if agent.DockerConfig == nil {
		return nil, fmt.Errorf("agent does not have Docker configuration")
	}

	// Check if it's a waterfall build
	if agent.DockerConfig.Waterfall != nil && len(agent.DockerConfig.Waterfall.Stages) > 0 {
		waterfall := &WaterfallDockerAgent{
			DockerAgent: DockerAgent{
				config:     agent,
				isRunning:  false,
				mutex:      sync.RWMutex{},
			},
		}
		return waterfall, nil
	}

	// Regular Docker agent
	dockerAgent := &DockerAgent{
		config:     agent,
		isRunning:  false,
		mutex:      sync.RWMutex{},
	}
	return dockerAgent, nil
}