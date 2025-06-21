package validation

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// ShellType represents different shell types
type ShellType string

const (
	ShellBash   ShellType = "bash"
	ShellZsh    ShellType = "zsh"
	ShellFish   ShellType = "fish"
	ShellPOSIX  ShellType = "posix"
	ShellAuto   ShellType = "auto"
)

// ErrorType categorizes validation errors
type ErrorType string

const (
	ErrorSyntax      ErrorType = "syntax"
	ErrorSafety      ErrorType = "safety"
	ErrorPermission  ErrorType = "permission"
	ErrorDeprecated  ErrorType = "deprecated"
	ErrorCustom      ErrorType = "custom"
)

// Severity levels for validation issues
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// Position in the command string
type Position struct {
	Line   int
	Column int
	Offset int
	Length int
}

// ValidationError represents a syntax or safety error
type ValidationError struct {
	Type        ErrorType
	Severity    Severity
	Position    Position
	Message     string
	Rule        string
	Suggestion  string
	RiskLevel   RiskLevel // Added for risk assessment
}

// ValidationWarning represents a non-critical issue
type ValidationWarning struct {
	Message    string
	Suggestion string
	Position   Position
}

// Suggestion for command improvement
type Suggestion struct {
	Message     string
	Alternative string
	Explanation string
}

// ValidationResult contains the results of validation
type ValidationResult struct {
	Valid          bool
	Command        string
	Shell          ShellType
	Errors         []ValidationError
	Warnings       []ValidationWarning
	Suggestions    []Suggestion
	RiskAssessment *RiskAssessment // Added for comprehensive risk analysis
	Timestamp      time.Time
	Duration       time.Duration
	Metadata       map[string]interface{}
}

// Validator is the main interface for command validation
type Validator interface {
	Validate(ctx context.Context, command string) (*ValidationResult, error)
	ValidateWithShell(ctx context.Context, command string, shell ShellType) (*ValidationResult, error)
}

// Engine is the main validation engine
type Engine struct {
	shellType            ShellType
	parsers              map[ShellType]Parser
	safetyRules          []SafetyRule
	customRules          []ICustomRule
	config               ValidationConfig
	safetyChecker        *InteractiveSafetyChecker
	obfuscationDetector  *ObfuscationDetector
	customRuleEngine     *CustomRuleEngine
}

// Parser interface for shell-specific parsing
type Parser interface {
	Parse(command string) (*AST, error)
	Validate(ast *AST) []ValidationError
	GetShellType() ShellType
}

// SafetyRule defines a safety check
type SafetyRule interface {
	Check(command string, ast *AST) []ValidationError
	GetName() string
	GetDescription() string
	GetRiskLevel() RiskLevel
}

// ICustomRule allows user-defined validation rules (interface)
type ICustomRule interface {
	Validate(command string, ast *AST) []ValidationError
	GetName() string
	IsEnabled() bool
}

// ValidationConfig configures the validation engine
type ValidationConfig struct {
	EnableSyntaxCheck          bool
	EnableSafetyCheck          bool
	EnableCustomRules          bool
	EnableObfuscationDetection bool
	StrictMode                 bool
	RealTimeValidation         bool
	MaxValidationTime          time.Duration
	SafetyPromptConfig         SafetyPromptConfig // Configuration for interactive safety
}

// NewEngine creates a new validation engine
func NewEngine(config ValidationConfig) *Engine {
	engine := &Engine{
		shellType:   ShellAuto,
		parsers:     make(map[ShellType]Parser),
		safetyRules: DefaultSafetyRules(),
		customRules: []ICustomRule{},
		config:      config,
	}
	
	// Initialize parsers
	engine.initializeParsers()
	
	// Initialize interactive safety checker if enabled
	if config.SafetyPromptConfig.Enabled {
		engine.safetyChecker = NewInteractiveSafetyChecker(config.SafetyPromptConfig)
	}
	
	// Initialize obfuscation detector if enabled
	if config.EnableObfuscationDetection {
		engine.obfuscationDetector = NewObfuscationDetector()
	}
	
	// Initialize custom rule engine if enabled
	if config.EnableCustomRules {
		// Use default path if not specified
		customRulesPath := "~/.config/delta/validation_rules.yaml"
		engine.customRuleEngine = NewCustomRuleEngine(customRulesPath)
	}
	
	return engine
}

// initializeParsers sets up shell-specific parsers
func (e *Engine) initializeParsers() {
	// For now, we'll use a simplified approach to avoid circular imports
	// TODO: Properly initialize parsers without circular dependencies
}

// Validate validates a command with auto-detected shell
func (e *Engine) Validate(ctx context.Context, command string) (*ValidationResult, error) {
	shell := e.detectShell(command)
	return e.ValidateWithShell(ctx, command, shell)
}

// ValidateWithShell validates a command with a specific shell
func (e *Engine) ValidateWithShell(ctx context.Context, command string, shell ShellType) (*ValidationResult, error) {
	start := time.Now()
	
	result := &ValidationResult{
		Valid:       true,
		Command:     command,
		Shell:       shell,
		Errors:      []ValidationError{},
		Warnings:    []ValidationWarning{},
		Suggestions: []Suggestion{},
		Timestamp:   start,
		Metadata:    make(map[string]interface{}),
	}
	
	// Check context timeout
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("validation cancelled: %w", err)
	}
	
	// Obfuscation detection (do this first to potentially deobfuscate)
	if e.config.EnableObfuscationDetection && e.obfuscationDetector != nil {
		obfResult := e.obfuscationDetector.DetectObfuscation(command)
		if obfResult.IsObfuscated {
			// Add obfuscation error
			result.Errors = append(result.Errors, ValidationError{
				Type:     ErrorSafety,
				Severity: SeverityError,
				Message:  fmt.Sprintf("Command appears to be obfuscated: %s", obfResult.Explanation),
				Rule:     "ObfuscationDetection",
				RiskLevel: obfResult.RiskLevel,
				Suggestion: "Obfuscated commands are often used to hide malicious intent. Review the deobfuscated command carefully.",
			})
			
			// Store obfuscation info in metadata
			result.Metadata["obfuscation"] = obfResult
			result.Metadata["deobfuscated"] = obfResult.Deobfuscated
		}
	}
	
	// For now, use simple syntax checking without full parsing
	if e.config.EnableSyntaxCheck {
		syntaxErrors := SimpleSyntaxCheck(command)
		result.Errors = append(result.Errors, syntaxErrors...)
	}
	
	// Safety validation using pattern matching
	if e.config.EnableSafetyCheck {
		for _, rule := range e.safetyRules {
			safetyErrors := rule.Check(command, nil) // Pass nil AST for now
			result.Errors = append(result.Errors, safetyErrors...)
		}
	}
	
	// Custom rules validation
	if e.config.EnableCustomRules && e.customRuleEngine != nil {
		customErrors := e.customRuleEngine.ValidateCommand(command)
		result.Errors = append(result.Errors, customErrors...)
	}
	
	// Set valid flag based on errors
	result.Valid = len(result.Errors) == 0
	
	// Add suggestions
	result.Suggestions = e.generateSuggestions(command, nil, result.Errors)
	
	// Perform risk assessment
	envContext := GetEnvironmentContext()
	result.RiskAssessment = &RiskAssessment{}
	*result.RiskAssessment = AssessCommandRisk(command, result.Errors, envContext)
	
	result.Duration = time.Since(start)
	return result, nil
}

// detectShell attempts to detect the shell type from the command
func (e *Engine) detectShell(command string) ShellType {
	// Simple heuristics for shell detection
	if strings.Contains(command, "[[") && strings.Contains(command, "]]") {
		return ShellBash
	}
	if strings.Contains(command, "setopt") || strings.Contains(command, "zstyle") {
		return ShellZsh
	}
	if strings.Contains(command, "set -x") || strings.Contains(command, "set -l") {
		return ShellFish
	}
	
	// Default to POSIX
	return ShellPOSIX
}

// generateSuggestions creates helpful suggestions based on errors
func (e *Engine) generateSuggestions(command string, ast *AST, errors []ValidationError) []Suggestion {
	suggestions := []Suggestion{}
	
	// Generate suggestions based on common patterns
	for _, err := range errors {
		switch err.Type {
		case ErrorSyntax:
			if strings.Contains(err.Message, "quote") {
				suggestions = append(suggestions, Suggestion{
					Message:     "Consider using single quotes for literal strings",
					Alternative: strings.ReplaceAll(command, `"`, `'`),
					Explanation: "Single quotes prevent variable expansion and special character interpretation",
				})
			}
		case ErrorSafety:
			if strings.Contains(command, "rm -rf") {
				suggestions = append(suggestions, Suggestion{
					Message:     "Use 'trash' command instead of 'rm -rf'",
					Alternative: strings.ReplaceAll(command, "rm -rf", "trash"),
					Explanation: "The trash command moves files to a recoverable location instead of permanent deletion",
				})
			}
		}
	}
	
	return suggestions
}

// AddCustomRule adds a custom validation rule
func (e *Engine) AddCustomRule(rule ICustomRule) {
	e.customRules = append(e.customRules, rule)
}

// SetSafetyRules replaces the safety rules
func (e *Engine) SetSafetyRules(rules []SafetyRule) {
	e.safetyRules = rules
}

// GetConfig returns the current configuration
func (e *Engine) GetConfig() ValidationConfig {
	return e.config
}

// SetConfig updates the configuration
func (e *Engine) SetConfig(config ValidationConfig) {
	e.config = config
	
	// Update interactive safety checker if needed
	if config.SafetyPromptConfig.Enabled && e.safetyChecker == nil {
		e.safetyChecker = NewInteractiveSafetyChecker(config.SafetyPromptConfig)
	} else if !config.SafetyPromptConfig.Enabled {
		e.safetyChecker = nil
	}
}

// CheckInteractiveSafety performs interactive safety check on validation result
func (e *Engine) CheckInteractiveSafety(result *ValidationResult) (bool, *SafetyDecision) {
	if e.safetyChecker == nil {
		return true, nil
	}
	
	return e.safetyChecker.CheckInteractiveSafety(result)
}

// GetSafetyChecker returns the interactive safety checker
func (e *Engine) GetSafetyChecker() *InteractiveSafetyChecker {
	return e.safetyChecker
}

// GetCustomRuleEngine returns the custom rule engine
func (e *Engine) GetCustomRuleEngine() *CustomRuleEngine {
	return e.customRuleEngine
}

