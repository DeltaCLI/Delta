package main

// agent_types.go is included in the build and provides key type definitions for agents.
// This file is referenced from agent_manager.go, knowledge_extractor_agent_command.go, and others.

import (
	"context"
	"sync"
	"time"
)

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
	ErrorHandling   *ErrorHandlingConfig `json:"error_handling,omitempty"`
}

// AgentCommand represents a command sequence for an agent
type AgentCommand struct {
	ID              string            `json:"id"`
	Command         string            `json:"command"`
	WorkingDir      string            `json:"working_dir"`
	ExpectedOutput  string            `json:"expected_output,omitempty"`
	ErrorPatterns   []string          `json:"error_patterns,omitempty"`
	SuccessPatterns []string          `json:"success_patterns,omitempty"`
	Timeout         int               `json:"timeout"`
	RetryCount      int               `json:"retry_count"`
	RetryDelay      int               `json:"retry_delay"`
	IsInteractive   bool              `json:"is_interactive"`
	Environment     map[string]string `json:"environment,omitempty"`
	Enabled         bool              `json:"enabled"`
}

// AgentDockerConfig represents Docker configuration for an agent
type AgentDockerConfig struct {
	Enabled      bool              `json:"enabled"`
	Image        string            `json:"image"`
	Tag          string            `json:"tag"`
	BuildContext string            `json:"build_context,omitempty"`
	Dockerfile   string            `json:"dockerfile,omitempty"`
	ComposeFile  string            `json:"compose_file,omitempty"`
	Volumes      []string          `json:"volumes,omitempty"`
	Networks     []string          `json:"networks,omitempty"`
	Ports        []string          `json:"ports,omitempty"`
	Environment  map[string]string `json:"environment,omitempty"`
	CacheFrom    []string          `json:"cache_from,omitempty"`
	BuildArgs    map[string]string `json:"build_args,omitempty"`
	UseCache     bool              `json:"use_cache"`
	Waterfall    *WaterfallConfig  `json:"waterfall,omitempty"`
}

// WaterfallConfig represents a multi-stage waterfall build configuration
type WaterfallConfig struct {
	Stages       []string            `json:"stages"`
	Dependencies map[string][]string `json:"dependencies"`
	ComposeFile  string              `json:"compose_file,omitempty"`
	ProjectName  string              `json:"project_name,omitempty"`
}

// ErrorHandlingConfig represents agent-specific error handling configuration
type ErrorHandlingConfig struct {
	Patterns []ErrorPatternConfig `json:"patterns"`
}

// ErrorPatternConfig represents an error pattern and its solution
type ErrorPatternConfig struct {
	Pattern     string `json:"pattern"`
	Solution    string `json:"solution"`
	Description string `json:"description"`
	FilePattern string `json:"file_pattern,omitempty"`
}

// AgentErrorHandling contains error handling configuration for an agent
type AgentErrorHandling struct {
	AutoFix            bool                `json:"auto_fix"`
	Patterns           []AgentErrorPattern `json:"patterns"`
	AIAssisted         bool                `json:"ai_assisted"`
	FeedbackEnabled    bool                `json:"feedback_enabled"`
	LearnFromSolutions bool                `json:"learn_from_solutions"`
}

// AgentErrorPattern represents an error pattern and its solution
type AgentErrorPattern struct {
	Pattern     string `json:"pattern"`
	Solution    string `json:"solution"`
	Description string `json:"description"`
	FilePattern string `json:"file_pattern,omitempty"`
}

// AgentSchedule represents a schedule for agent execution
type AgentSchedule struct {
	Enabled bool   `json:"enabled"`
	Cron    string `json:"cron"`
	Timeout int    `json:"timeout"`
}

// AgentNotification contains notification configuration for an agent
type AgentNotification struct {
	OnSuccess bool     `json:"on_success"`
	OnFailure bool     `json:"on_failure"`
	OnStart   bool     `json:"on_start"`
	Channels  []string `json:"channels"`
}

// AgentTrigger represents a trigger for agent execution
type AgentTrigger struct {
	Patterns  []string `json:"patterns"`
	Paths     []string `json:"paths"`
	Schedules []string `json:"schedules"`
	Events    []string `json:"events"`
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

// ErrorHandler defines the interface for agent error handlers
type ErrorHandler interface {
	SolveError(ctx context.Context, output string, err error, config CommandConfig) (bool, string, error)
}

// DockerManager manages Docker operations for agents
type DockerManager struct {
	available bool
	cacheDir  string
	mutex     sync.RWMutex
}

// AgentManagerConfig contains configuration for the agent manager
type AgentManagerConfig struct {
	Enabled           bool   `json:"enabled"`
	AgentStoragePath  string `json:"agent_storage_path"`
	CacheStoragePath  string `json:"cache_storage_path"`
	MaxCacheSize      int64  `json:"max_cache_size"`
	CacheRetention    int    `json:"cache_retention"`
	MaxAgentRuns      int    `json:"max_agent_runs"`
	DefaultTimeout    int    `json:"default_timeout"`
	DefaultRetryCount int    `json:"default_retry_count"`
	DefaultRetryDelay int    `json:"default_retry_delay"`
	UseDockerBuilds   bool   `json:"use_docker_builds"`
	UseAIAssistance   bool   `json:"use_ai_assistance"`
	AIPromptTemplate  string `json:"ai_prompt_template"`
}

// AgentRunResult represents the result of an agent run
type AgentRunResult struct {
	AgentID         string             `json:"agent_id"`
	StartTime       time.Time          `json:"start_time"`
	EndTime         time.Time          `json:"end_time"`
	Success         bool               `json:"success"`
	ExitCode        int                `json:"exit_code"`
	Output          string             `json:"output"`
	Errors          []string           `json:"errors"`
	CommandsRun     int                `json:"commands_run"`
	ArtifactsPaths  []string           `json:"artifacts_paths"`
	PerformanceData map[string]float64 `json:"performance_data"`
}

// BuildCacheConfig represents cache configuration for a specific build
type BuildCacheConfig struct {
	Name        string            `json:"name"`
	Stages      []string          `json:"stages"`
	DependsOn   []string          `json:"depends_on"`
	CacheVolume string            `json:"cache_volume"`
	BuildArgs   map[string]string `json:"build_args"`
	LastBuiltAt time.Time         `json:"last_built_at"`
	CacheSize   int64             `json:"cache_size"`
	CacheHits   int               `json:"cache_hits"`
	CacheMisses int               `json:"cache_misses"`
}

// DockerBuildCache manages build caching for Docker-based agents
type DockerBuildCache struct {
	CacheDir     string                       `json:"cache_dir"`
	CacheSize    int64                        `json:"cache_size"`
	MaxCacheAge  time.Duration                `json:"max_cache_age"`
	BuildConfigs map[string]*BuildCacheConfig `json:"build_configs"`
}

// AgentManager handles the creation, execution, and management of agents
type AgentManager struct {
	// Configuration
	config     AgentManagerConfig
	configPath string
	storageDir string

	// Agent management
	agents        map[string]*Agent
	runningAgents map[string]AgentInterface
	errorHandlers map[string]ErrorHandler
	runHistory    []AgentRunResult

	// Docker support
	dockerManager *DockerManager
	dockerCache   *DockerBuildCache

	// AI integration
	aiManager          *AIPredictionManager
	knowledgeExtractor *KnowledgeExtractor

	// Runtime settings
	maxConcurrent int
	mutex         sync.RWMutex
	isInitialized bool
}

// AgentStatus represents the current status of an agent
type AgentStatus struct {
	ID             string        `json:"id"`
	IsRunning      bool          `json:"is_running"`
	StartTime      time.Time     `json:"start_time"`
	EndTime        time.Time     `json:"end_time"`
	RunDuration    time.Duration `json:"run_duration"`
	LastError      error         `json:"last_error"`
	CurrentCommand string        `json:"current_command"`
	Progress       float64       `json:"progress"`
	SuccessCount   int           `json:"success_count"`
	ErrorCount     int           `json:"error_count"`
	LastOutput     string        `json:"last_output"`
}

// AgentConfig represents the configuration for an agent
type AgentConfig struct {
	// Basic information
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	TaskTypes   []string `json:"task_types"`

	// Commands to execute
	Commands struct {
		Uproot AgentCommand   `json:"uproot"`
		Build  AgentCommand   `json:"build"`
		Custom []AgentCommand `json:"custom"`
	} `json:"commands"`

	// Docker configuration
	DockerConfig AgentDockerConfig `json:"docker_config"`

	// Error handling
	ErrorHandling AgentErrorHandling `json:"error_handling"`

	// Triggers and scheduling
	TriggerPatterns []string      `json:"trigger_patterns"`
	Schedule        AgentSchedule `json:"schedule"`

	// Notifications
	Notification AgentNotification `json:"notification"`

	// Context and metadata
	Context map[string]string `json:"context"`
	Tags    []string          `json:"tags"`

	// AI integration
	AIPrompt string `json:"ai_prompt"`

	// Status
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AgentOptions represents options for agent execution
type AgentOptions struct {
	Environment map[string]string
	Timeout     time.Duration
	DryRun      bool
	Verbose     bool
	OutputFile  string
	AutoFix     bool
}

// DockerComposeOptions represents options for Docker Compose execution
type DockerComposeOptions struct {
	File          string
	ProjectName   string
	Services      []string
	Environment   map[string]string
	Detached      bool
	RemoveOrphans bool
	ForceRecreate bool
	NoBuild       bool
	NoCache       bool
	PullImages    bool
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
