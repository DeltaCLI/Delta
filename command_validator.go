package main

import (
	"context"
	"fmt"
	"strings"
	"time"
	
	"delta/validation"
)

// Global validation engine instance
var globalValidationEngine *validation.Engine

// GetValidationEngine returns the global validation engine instance
func GetValidationEngine() *validation.Engine {
	if globalValidationEngine == nil {
		// Create default configuration
		config := validation.ValidationConfig{
			EnableSyntaxCheck:  true,
			EnableSafetyCheck:  true,
			EnableCustomRules:  true,
			EnableObfuscationDetection: true,
			StrictMode:        false,
			RealTimeValidation: false,
			MaxValidationTime:  5 * time.Second,
			SafetyPromptConfig: validation.SafetyPromptConfig{
				Enabled:               true,
				RequireConfirmation:   true,
				ShowEducationalInfo:   true,
				TrackSafetyDecisions:  true,
				AutoDenyLevel:         validation.RiskLevelCritical,
				BypassForTrustedPaths: true,
			},
		}
		globalValidationEngine = validation.NewEngine(config)
	}
	return globalValidationEngine
}

// ValidateBeforeExecution validates a command before execution
// Returns true if the command should be executed, false if it should be cancelled
func ValidateBeforeExecution(command string) bool {
	// Skip validation for internal commands
	if strings.HasPrefix(command, ":") {
		return true
	}
	
	// Get configuration
	config := GetConfigManager()
	if config != nil && config.GetConfig("validation.enabled") == "false" {
		return true // Skip validation if disabled
	}
	
	// Get validation engine
	engine := GetValidationEngine()
	
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	
	// Validate the command
	result, err := engine.Validate(ctx, command)
	if err != nil {
		// If validation fails, allow execution but warn
		fmt.Printf("⚠️  Validation error: %v\n", err)
		return true
	}
	
	// Check if we need interactive safety confirmation
	if result.RiskAssessment != nil && result.RiskAssessment.OverallRisk != validation.RiskLevelLow {
		// Check configuration for interactive safety
		if config == nil || config.GetConfig("validation.interactive_safety") != "false" {
			// Perform interactive safety check
			proceed, decision := engine.CheckInteractiveSafety(result)
			
			// Log the decision if tracking is enabled
			if decision != nil && decision.Decision != "" {
				// TODO: Save decision to history for future analysis
			}
			
			return proceed
		}
	}
	
	// If there are critical errors, show them but allow execution
	if !result.Valid {
		hasBlockingErrors := false
		for _, err := range result.Errors {
			if err.RiskLevel == validation.RiskLevelCritical {
				hasBlockingErrors = true
				break
			}
		}
		
		if hasBlockingErrors {
			fmt.Println("\n🚨 Critical safety issues detected!")
			displayValidationErrors(result)
			
			// Auto-deny critical commands if configured
			if engine.GetConfig().SafetyPromptConfig.AutoDenyLevel == validation.RiskLevelCritical {
				fmt.Println("\n❌ Command blocked due to critical safety risks.")
				return false
			}
		}
	}
	
	return true
}

// displayValidationErrors shows validation errors in a concise format
func displayValidationErrors(result *validation.ValidationResult) {
	for _, err := range result.Errors {
		if err.Type == validation.ErrorSafety {
			fmt.Printf("⚠️  %s\n", err.Message)
			if err.Suggestion != "" {
				fmt.Printf("   💡 %s\n", err.Suggestion)
			}
		}
	}
}

// UpdateValidationConfig updates the validation configuration
func UpdateValidationConfig(key string, value string) {
	engine := GetValidationEngine()
	config := engine.GetConfig()
	
	switch key {
	case "enabled":
		// This is handled by the config manager
	case "syntax_check":
		config.EnableSyntaxCheck = value == "true"
	case "safety_check":
		config.EnableSafetyCheck = value == "true"
	case "strict_mode":
		config.StrictMode = value == "true"
	case "interactive_safety":
		config.SafetyPromptConfig.Enabled = value == "true"
	case "educational_info":
		config.SafetyPromptConfig.ShowEducationalInfo = value == "true"
	case "auto_deny_critical":
		if value == "true" {
			config.SafetyPromptConfig.AutoDenyLevel = validation.RiskLevelCritical
		} else {
			config.SafetyPromptConfig.AutoDenyLevel = ""
		}
	case "bypass_trusted_paths":
		config.SafetyPromptConfig.BypassForTrustedPaths = value == "true"
	case "custom_rules":
		config.EnableCustomRules = value == "true"
		// Reinitialize engine with new config to load custom rule engine
		engine.SetConfig(config)
		if value == "true" {
			*engine = *validation.NewEngine(config)
		}
	case "obfuscation_detection":
		config.EnableObfuscationDetection = value == "true"
		// Reinitialize engine with new config
		engine.SetConfig(config)
		if value == "true" {
			*engine = *validation.NewEngine(config)
		}
	}
	
	engine.SetConfig(config)
}