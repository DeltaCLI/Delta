package main

import (
	"context"
	"fmt"
	"strings"
	"time"
	"delta/validation"
)

// HandleValidationCommand handles validation-related commands
func HandleValidationCommand(args []string) bool {
	if len(args) == 0 {
		showValidationHelp()
		return true
	}

	switch args[0] {
	case "check", "validate":
		return handleCommandValidation(args[1:])
	case "safety":
		return handleSafetyCommand(args[1:])
	case "config":
		return handleValidationConfig(args[1:])
	case "stats", "statistics":
		return handleValidationStats()
	case "history":
		return handleValidationHistory()
	case "help":
		showValidationHelp()
		return true
	default:
		// If first arg doesn't match subcommand, validate the entire input
		return handleCommandValidation(args)
	}
}

// handleCommandValidation validates a command's syntax
func handleCommandValidation(args []string) bool {
	if len(args) == 0 {
		fmt.Println("Usage: :validate <command>")
		fmt.Println("Example: :validate rm -rf /tmp/test")
		return true
	}

	command := strings.Join(args, " ")
	fmt.Printf("Validating command: %s\n\n", command)

	// Create validation engine
	config := validation.ValidationConfig{
		EnableSyntaxCheck:  true,
		EnableSafetyCheck:  true,
		EnableCustomRules:  false,
		StrictMode:        false,
		RealTimeValidation: false,
		MaxValidationTime:  5 * time.Second,
	}
	engine := validation.NewEngine(config)

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), config.MaxValidationTime)
	defer cancel()

	// Validate the command
	result, err := engine.Validate(ctx, command)
	if err != nil {
		fmt.Printf("❌ Validation error: %v\n", err)
		return true
	}

	// Display results
	displayValidationResult(result)

	return true
}

// handleSafetyCommand performs safety analysis on a command
func handleSafetyCommand(args []string) bool {
	if len(args) == 0 {
		fmt.Println("Usage: :validation safety <command>")
		fmt.Println("Example: :validation safety curl http://example.com | bash")
		return true
	}

	command := strings.Join(args, " ")
	fmt.Printf("Safety analysis for: %s\n\n", command)

	// For now, just use validation with safety focus
	config := validation.ValidationConfig{
		EnableSyntaxCheck:  false,
		EnableSafetyCheck:  true,
		EnableCustomRules:  false,
		StrictMode:        true,
		RealTimeValidation: false,
		MaxValidationTime:  5 * time.Second,
	}
	engine := validation.NewEngine(config)

	ctx, cancel := context.WithTimeout(context.Background(), config.MaxValidationTime)
	defer cancel()

	result, err := engine.Validate(ctx, command)
	if err != nil {
		fmt.Printf("❌ Safety analysis error: %v\n", err)
		return true
	}

	displaySafetyResult(result)

	return true
}

// handleValidationConfig manages validation configuration
func handleValidationConfig(args []string) bool {
	if len(args) == 0 {
		showValidationConfig()
		return true
	}

	switch args[0] {
	case "set":
		if len(args) < 3 {
			fmt.Println("Usage: :validation config set <key> <value>")
			fmt.Println("\nAvailable keys:")
			fmt.Println("  enabled              - Enable/disable validation (true/false)")
			fmt.Println("  syntax_check         - Enable syntax checking (true/false)")
			fmt.Println("  safety_check         - Enable safety analysis (true/false)")
			fmt.Println("  interactive_safety   - Enable interactive safety prompts (true/false)")
			fmt.Println("  educational_info     - Show educational content (true/false)")
			fmt.Println("  auto_deny_critical   - Auto-deny critical commands (true/false)")
			fmt.Println("  bypass_trusted_paths - Skip prompts in trusted directories (true/false)")
			return true
		}
		
		key := args[1]
		value := args[2]
		
		// Update configuration through config manager
		cm := GetConfigManager()
		if cm != nil {
			configKey := fmt.Sprintf("validation.%s", key)
			if err := cm.SetConfig(configKey, value); err != nil {
				fmt.Printf("❌ Error setting config: %v\n", err)
			} else {
				// Also update the validation engine
				UpdateValidationConfig(key, value)
				fmt.Printf("✅ Set %s = %s\n", key, value)
			}
		} else {
			// Update validation engine directly
			UpdateValidationConfig(key, value)
			fmt.Printf("✅ Set %s = %s (in memory only)\n", key, value)
		}
		
		return true
		
	case "get":
		if len(args) < 2 {
			fmt.Println("Usage: :validation config get <key>")
			return true
		}
		
		key := args[1]
		cm := GetConfigManager()
		if cm != nil {
			value := cm.GetConfig(fmt.Sprintf("validation.%s", key))
			if value != "" {
				fmt.Printf("%s = %s\n", key, value)
			} else {
				fmt.Printf("%s is not set\n", key)
			}
		}
		
		return true
		
	case "reset":
		fmt.Println("Resetting validation configuration to defaults...")
		// Reset to default configuration
		engine := GetValidationEngine()
		defaultConfig := validation.ValidationConfig{
			EnableSyntaxCheck:  true,
			EnableSafetyCheck:  true,
			EnableCustomRules:  false,
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
		engine.SetConfig(defaultConfig)
		fmt.Println("✅ Configuration reset to defaults")
		return true
		
	default:
		fmt.Printf("Unknown config command: %s\n", args[0])
		fmt.Println("Available commands: set, get, reset")
		return true
	}
}

// displayValidationResult shows validation results to the user
func displayValidationResult(result *validation.ValidationResult) {
	// Display risk assessment first if available
	if result.RiskAssessment != nil && len(result.RiskAssessment.Factors) > 0 {
		fmt.Printf("Risk Assessment: %s\n\n", validation.FormatRiskLevel(result.RiskAssessment.OverallRisk))
		
		if result.RiskAssessment.RequiresRoot {
			fmt.Println("⚠️  Command requires elevated privileges (root/sudo)")
		}
		if result.RiskAssessment.AffectsSystem {
			fmt.Println("⚠️  Command affects system directories")
		}
		if result.RiskAssessment.IsIrreversible {
			fmt.Println("⚠️  Command performs irreversible operations")
		}
		fmt.Println()
	}
	
	if result.Valid && (result.RiskAssessment == nil || result.RiskAssessment.OverallRisk == validation.RiskLevelLow) {
		fmt.Println("✅ Command syntax is valid and safe")
	} else if result.Valid {
		fmt.Println("⚠️  Command syntax is valid but has safety concerns")
	} else {
		fmt.Println("❌ Command has validation errors:")
		for i, err := range result.Errors {
			// Show risk level for safety errors
			riskInfo := ""
			if err.Type == validation.ErrorSafety && err.RiskLevel != "" {
				riskInfo = fmt.Sprintf(" [%s]", validation.FormatRiskLevel(err.RiskLevel))
			}
			
			fmt.Printf("\n%d. %s Error%s: %s\n", i+1, err.Type, riskInfo, err.Message)
			if err.Position.Offset > 0 {
				fmt.Printf("   Position: line %d, column %d\n", 
					err.Position.Line, err.Position.Column)
			}
			if err.Suggestion != "" {
				fmt.Printf("   💡 Suggestion: %s\n", err.Suggestion)
			}
		}
	}

	if len(result.Warnings) > 0 {
		fmt.Println("\n⚠️  Warnings:")
		for _, warn := range result.Warnings {
			fmt.Printf("  - %s\n", warn.Message)
			if warn.Suggestion != "" {
				fmt.Printf("    💡 %s\n", warn.Suggestion)
			}
		}
	}

	if len(result.Suggestions) > 0 {
		fmt.Println("\n💡 Suggestions:")
		for _, sug := range result.Suggestions {
			fmt.Printf("  - %s\n", sug.Message)
			if sug.Alternative != "" {
				fmt.Printf("    Alternative: %s\n", sug.Alternative)
			}
			if sug.Explanation != "" {
				fmt.Printf("    Why: %s\n", sug.Explanation)
			}
		}
	}

	fmt.Printf("\n⏱️  Validation completed in %s\n", result.Duration.Round(time.Millisecond))
}

// displaySafetyResult shows safety analysis results
func displaySafetyResult(result *validation.ValidationResult) {
	if result.RiskAssessment == nil {
		fmt.Println("⚠️  Risk assessment not available")
		return
	}
	
	// Display overall risk
	fmt.Printf("Safety Analysis: %s\n\n", validation.FormatRiskLevel(result.RiskAssessment.OverallRisk))
	
	// Display risk factors
	if len(result.RiskAssessment.Factors) > 0 {
		fmt.Println("Risk Factors:")
		for _, factor := range result.RiskAssessment.Factors {
			fmt.Printf("- %s %s\n", validation.GetRiskEmoji(factor.Level), factor.Description)
			if factor.Mitigation != "" {
				fmt.Printf("  Mitigation: %s\n", factor.Mitigation)
			}
		}
		fmt.Println()
	}
	
	// Display context information
	if result.RiskAssessment.Context.IsGitRepository {
		fmt.Println("📁 Current directory is a Git repository")
	}
	if result.RiskAssessment.Context.IsSystemPath {
		fmt.Println("⚠️  Current directory is a system path")
	}
	if result.RiskAssessment.RequiresRoot {
		fmt.Println("🔐 Command requires root/sudo privileges")
	}
	if result.RiskAssessment.AffectsSystem {
		fmt.Println("⚠️  Command affects system directories")
	}
	if result.RiskAssessment.IsIrreversible {
		fmt.Println("❌ Command performs irreversible operations")
	}
	fmt.Println()

	if len(result.Errors) == 0 {
		fmt.Println("✅ No safety concerns detected.")
	} else {
		fmt.Println("⚠️  Safety concerns found:")
		for _, err := range result.Errors {
			if err.Type == validation.ErrorSafety {
				fmt.Printf("\n- %s\n", err.Message)
				if err.Suggestion != "" {
					fmt.Printf("  Mitigation: %s\n", err.Suggestion)
				}
			}
		}
	}
}

// showValidationHelp displays help for validation commands
func showValidationHelp() {
	fmt.Println(`Validation Commands:
═══════════════════

Syntax Validation:
  :validate <command>              Check command syntax
  :validation check <command>      Same as :validate
  
Safety Analysis:
  :validation safety <command>     Analyze command safety
  
Interactive Safety:
  :validation config              Show configuration
  :validation config set <key> <value>  Update configuration
  :validation config get <key>    Get configuration value
  :validation config reset        Reset to defaults
  
Safety Statistics:
  :validation stats               Show safety decision history
  :validation history             View recent safety decisions
  
Help:
  :validation help               Show this help

Examples:
  :validate ls -la | grep test
  :validation safety rm -rf /
  :validation config set interactive_safety true
  :validation config set educational_info false

Configuration Keys:
  enabled              - Enable/disable validation
  syntax_check         - Enable syntax checking
  safety_check         - Enable safety analysis
  interactive_safety   - Enable interactive prompts
  educational_info     - Show educational content
  auto_deny_critical   - Auto-deny critical commands
  bypass_trusted_paths - Skip prompts in trusted directories

Shortcuts:
  :v                            Alias for :validate`)
}

// showValidationConfig displays current validation configuration
func showValidationConfig() {
	engine := GetValidationEngine()
	config := engine.GetConfig()
	
	fmt.Println(`Current Validation Configuration:
═══════════════════════════════`)
	
	fmt.Printf("\nCore Settings:\n")
	fmt.Printf("  Validation Enabled:      %v\n", getBoolConfigDisplay("validation.enabled", true))
	fmt.Printf("  Syntax Checking:         %v\n", config.EnableSyntaxCheck)
	fmt.Printf("  Safety Analysis:         %v\n", config.EnableSafetyCheck)
	fmt.Printf("  Custom Rules:            %v\n", config.EnableCustomRules)
	fmt.Printf("  Strict Mode:             %v\n", config.StrictMode)
	fmt.Printf("  Real-time Validation:    %v\n", config.RealTimeValidation)
	fmt.Printf("  Max Validation Time:     %s\n", config.MaxValidationTime)
	
	fmt.Printf("\nInteractive Safety:\n")
	fmt.Printf("  Interactive Prompts:     %v\n", config.SafetyPromptConfig.Enabled)
	fmt.Printf("  Require Confirmation:    %v\n", config.SafetyPromptConfig.RequireConfirmation)
	fmt.Printf("  Show Educational Info:   %v\n", config.SafetyPromptConfig.ShowEducationalInfo)
	fmt.Printf("  Track Decisions:         %v\n", config.SafetyPromptConfig.TrackSafetyDecisions)
	fmt.Printf("  Auto-deny Critical:      %v\n", config.SafetyPromptConfig.AutoDenyLevel == validation.RiskLevelCritical)
	fmt.Printf("  Bypass Trusted Paths:    %v\n", config.SafetyPromptConfig.BypassForTrustedPaths)
	
	fmt.Println(`
To modify settings, use:
  :validation config set <key> <value>
  :validation config get <key>
  :validation config reset`)
}

// getBoolConfigDisplay gets a boolean config value with default
func getBoolConfigDisplay(key string, defaultValue bool) bool {
	cm := GetConfigManager()
	if cm != nil {
		value := cm.GetConfig(key)
		if value == "false" {
			return false
		} else if value == "true" {
			return true
		}
	}
	return defaultValue
}

// handleValidationStats shows safety decision statistics
func handleValidationStats() bool {
	engine := GetValidationEngine()
	safetyChecker := engine.GetSafetyChecker()
	if safetyChecker == nil {
		fmt.Println("Interactive safety checker is not enabled.")
		fmt.Println("Enable it with: :validation config set interactive_safety true")
		return true
	}
	
	stats := safetyChecker.GetSafetyStats()
	
	fmt.Println("Safety Decision Statistics:")
	fmt.Println("═════════════════════════")
	fmt.Printf("\nTotal decisions:     %d\n", stats["total"])
	fmt.Printf("Commands proceeded:  %d\n", stats["proceeded"])
	fmt.Printf("Commands cancelled:  %d\n", stats["cancelled"])
	fmt.Printf("Commands modified:   %d\n", stats["modified"])
	fmt.Printf("Marked as safe:      %d\n", stats["safe"])
	
	if stats["total"] > 0 {
		proceedRate := float64(stats["proceeded"]) / float64(stats["total"]) * 100
		cancelRate := float64(stats["cancelled"]) / float64(stats["total"]) * 100
		fmt.Printf("\nProceed rate:        %.1f%%\n", proceedRate)
		fmt.Printf("Cancel rate:         %.1f%%\n", cancelRate)
	}
	
	return true
}

// handleValidationHistory shows recent safety decisions
func handleValidationHistory() bool {
	engine := GetValidationEngine()
	safetyChecker := engine.GetSafetyChecker()
	if safetyChecker == nil {
		fmt.Println("Interactive safety checker is not enabled.")
		fmt.Println("Enable it with: :validation config set interactive_safety true")
		return true
	}
	
	history := safetyChecker.GetSafetyHistory()
	
	if len(history) == 0 {
		fmt.Println("No safety decision history available.")
		return true
	}
	
	fmt.Println("Recent Safety Decisions:")
	fmt.Println("════════════════════════")
	
	// Show last 10 decisions
	start := 0
	if len(history) > 10 {
		start = len(history) - 10
	}
	
	for i := start; i < len(history); i++ {
		decision := history[i]
		fmt.Printf("\n[%s] %s\n", decision.Timestamp.Format("2006-01-02 15:04:05"), decision.Command)
		fmt.Printf("  Risk Level: %s\n", validation.FormatRiskLevel(decision.RiskLevel))
		fmt.Printf("  Decision:   %s\n", decision.Decision)
		if decision.LearnedSafe {
			fmt.Printf("  Note:       Marked as safe for future use\n")
		}
	}
	
	return true
}

// ValidateCommandRealTime performs real-time validation as user types
func ValidateCommandRealTime(command string) *validation.ValidationResult {
	// Quick validation for real-time feedback
	config := validation.ValidationConfig{
		EnableSyntaxCheck:  true,
		EnableSafetyCheck:  false, // Skip safety for performance
		EnableCustomRules:  false,
		StrictMode:        false,
		RealTimeValidation: true,
		MaxValidationTime:  50 * time.Millisecond, // Very short timeout
	}
	
	engine := validation.NewEngine(config)
	ctx, cancel := context.WithTimeout(context.Background(), config.MaxValidationTime)
	defer cancel()
	
	result, _ := engine.Validate(ctx, command)
	return result
}