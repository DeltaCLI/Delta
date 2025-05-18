package main

import (
	"context"
	"time"
)

// Agent represents a task-specific autonomous agent
type Agent struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	TaskTypes       []string               `json:"task_types"`
	Commands        []AgentCommand         `json:"commands"`
	DockerConfig    *AgentDockerConfig     `json:"docker_config,omitempty"`
	TriggerPatterns []string               `json:"trigger_patterns"`
	Context         map[string]string      `json:"context"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	LastRunAt       time.Time              `json:"last_run_at"`
	RunCount        int                    `json:"run_count"`
	SuccessRate     float64                `json:"success_rate"`
	Tags            []string               `json:"tags"`
	AIPrompt        string                 `json:"ai_prompt"`
	Enabled         bool                   `json:"enabled"`
}

// AgentCommand represents a command sequence for an agent
type AgentCommand struct {
	ID              string                 `json:"id"`
	Command         string                 `json:"command"`
	WorkingDir      string                 `json:"working_dir"`
	ExpectedOutput  string                 `json:"expected_output,omitempty"`
	ErrorPatterns   []string               `json:"error_patterns,omitempty"`
	SuccessPatterns []string               `json:"success_patterns,omitempty"`
	Timeout         int                    `json:"timeout"`
	RetryCount      int                    `json:"retry_count"`
	RetryDelay      int                    `json:"retry_delay"`
	IsInteractive   bool                   `json:"is_interactive"`
	Environment     map[string]string      `json:"environment,omitempty"`
	Enabled         bool                   `json:"enabled"`
}

// AgentDockerConfig represents Docker configuration for an agent
type AgentDockerConfig struct {
	Enabled         bool                   `json:"enabled"`
	Image           string                 `json:"image"`
	Tag             string                 `json:"tag"`
	BuildContext    string                 `json:"build_context,omitempty"`
	Dockerfile      string                 `json:"dockerfile,omitempty"`
	ComposeFile     string                 `json:"compose_file,omitempty"`
	Volumes         []string               `json:"volumes,omitempty"`
	Networks        []string               `json:"networks,omitempty"`
	Ports           []string               `json:"ports,omitempty"`
	Environment     map[string]string      `json:"environment,omitempty"`
	CacheFrom       []string               `json:"cache_from,omitempty"`
	BuildArgs       map[string]string      `json:"build_args,omitempty"`
	UseCache        bool                   `json:"use_cache"`
	Waterfall       *WaterfallConfig       `json:"waterfall,omitempty"`
}

// WaterfallConfig represents a multi-stage waterfall build configuration
type WaterfallConfig struct {
	Stages          []string               `json:"stages"`
	Dependencies    map[string][]string    `json:"dependencies"`
	ComposeFile     string                 `json:"compose_file,omitempty"`
	ProjectName     string                 `json:"project_name,omitempty"`
}

// AgentErrorHandling contains error handling configuration for an agent
type AgentErrorHandling struct {
	AutoFix         bool                   `json:"auto_fix"`
	Patterns        []AgentErrorPattern    `json:"patterns"`
	AIAssisted      bool                   `json:"ai_assisted"`
	FeedbackEnabled bool                   `json:"feedback_enabled"`
	LearnFromSolutions bool                `json:"learn_from_solutions"`
}

// AgentErrorPattern represents an error pattern and its solution
type AgentErrorPattern struct {
	Pattern         string                 `json:"pattern"`
	Solution        string                 `json:"solution"`
	Description     string                 `json:"description"`
	FilePattern     string                 `json:"file_pattern,omitempty"`
}

// AgentSchedule represents a schedule for agent execution
type AgentSchedule struct {
	Enabled         bool                   `json:"enabled"`
	Cron            string                 `json:"cron"`
	Timeout         int                    `json:"timeout"`
}

// AgentNotification contains notification configuration for an agent
type AgentNotification struct {
	OnSuccess       bool                   `json:"on_success"`
	OnFailure       bool                   `json:"on_failure"`
	OnStart         bool                   `json:"on_start"`
	Channels        []string               `json:"channels"`
}

// AgentTrigger represents a trigger for agent execution
type AgentTrigger struct {
	Patterns        []string               `json:"patterns"`
	Paths           []string               `json:"paths"`
	Schedules       []string               `json:"schedules"`
	Events          []string               `json:"events"`
}

// AgentInterface defines the interface for agent implementations
type AgentInterface interface {
	// Initialize initializes the agent
	Initialize() error
	
	// Run executes the agent
	Run(ctx context.Context) error
	
	// Stop stops the agent execution
	Stop() error
	
	// GetStatus returns the current status of the agent
	GetStatus() AgentStatus
	
	// GetID returns the agent ID
	GetID() string
}

// AgentStatus represents the current status of an agent
type AgentStatus struct {
	ID              string                 `json:"id"`
	IsRunning       bool                   `json:"is_running"`
	StartTime       time.Time              `json:"start_time"`
	EndTime         time.Time              `json:"end_time"`
	RunDuration     time.Duration          `json:"run_duration"`
	LastError       error                  `json:"last_error"`
	CurrentCommand  string                 `json:"current_command"`
	Progress        float64                `json:"progress"`
	SuccessCount    int                    `json:"success_count"`
	ErrorCount      int                    `json:"error_count"`
	LastOutput      string                 `json:"last_output"`
}

// AgentConfig represents the configuration for an agent
type AgentConfig struct {
	// Basic information
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	TaskTypes       []string               `json:"task_types"`
	
	// Commands to execute
	Commands        struct {
		Uproot       AgentCommand          `json:"uproot"`
		Build        AgentCommand          `json:"build"`
		Custom       []AgentCommand        `json:"custom"`
	} `json:"commands"`
	
	// Docker configuration
	DockerConfig    AgentDockerConfig      `json:"docker_config"`
	
	// Error handling
	ErrorHandling   AgentErrorHandling     `json:"error_handling"`
	
	// Triggers and scheduling
	TriggerPatterns []string               `json:"trigger_patterns"`
	Schedule        AgentSchedule          `json:"schedule"`
	
	// Notifications
	Notification    AgentNotification      `json:"notification"`
	
	// Context and metadata
	Context         map[string]string      `json:"context"`
	Tags            []string               `json:"tags"`
	
	// AI integration
	AIPrompt        string                 `json:"ai_prompt"`
	
	// Status
	Enabled         bool                   `json:"enabled"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// AgentOptions represents options for agent execution
type AgentOptions struct {
	Environment     map[string]string
	Timeout         time.Duration
	DryRun          bool
	Verbose         bool
	OutputFile      string
	AutoFix         bool
}

// DockerComposeOptions represents options for Docker Compose execution
type DockerComposeOptions struct {
	File            string
	ProjectName     string
	Services        []string
	Environment     map[string]string
	Detached        bool
	RemoveOrphans   bool
	ForceRecreate   bool
	NoBuild         bool
	NoCache         bool
	PullImages      bool
}

// CommandConfig represents configuration for command execution
type CommandConfig struct {
	Command         string
	WorkingDir      string
	Timeout         int
	RetryCount      int
	RetryDelay      int
	ErrorPatterns   []string
	SuccessPatterns []string
	Environment     map[string]string
	IsInteractive   bool
}