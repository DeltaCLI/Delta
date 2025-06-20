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

	// TODO: Implement configuration management
	fmt.Println("Validation configuration management coming soon!")
	return true
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
  
Configuration:
  :validation config              Show configuration
  :validation config set <key> <value>  Update configuration
  
Help:
  :validation help               Show this help

Examples:
  :validate ls -la | grep test
  :validation safety rm -rf /
  :validation config set strict true

Shortcuts:
  :v                            Alias for :validate`)
}

// showValidationConfig displays current validation configuration
func showValidationConfig() {
	fmt.Println(`Current Validation Configuration:
═══════════════════════════════

Syntax Checking:     Enabled
Safety Analysis:     Enabled  
Custom Rules:        Disabled
Strict Mode:         Disabled
Real-time:           Disabled
Max Validation Time: 5s

To modify settings, use:
  :validation config set <key> <value>`)
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