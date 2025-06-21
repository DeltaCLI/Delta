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
	case "obfuscation":
		return handleObfuscationCommand(args[1:])
	case "deobfuscate":
		return handleDeobfuscateCommand(args[1:])
	case "rules":
		return handleCustomRulesCommand(args[1:])
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
  
Obfuscation Detection:
  :validation obfuscation <command>  Check for obfuscated commands
  :validation deobfuscate <command>  Show deobfuscated version
  
Custom Rules:
  :validation rules              List all custom rules
  :validation rules add          Add a new rule interactively
  :validation rules test <cmd>   Test command against custom rules
  :validation rules help         Show custom rules help
  
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
  :validation obfuscation 'echo "cm0gLXJmIC8=" | base64 -d'
  :validation deobfuscate 'r${IFS}m${IFS}-rf${IFS}/'
  :validation config set interactive_safety true

Configuration Keys:
  enabled                - Enable/disable validation
  syntax_check          - Enable syntax checking
  safety_check          - Enable safety analysis
  custom_rules          - Enable custom validation rules
  obfuscation_detection - Enable obfuscation detection
  interactive_safety    - Enable interactive prompts
  educational_info      - Show educational content
  auto_deny_critical    - Auto-deny critical commands
  bypass_trusted_paths  - Skip prompts in trusted directories

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

// handleObfuscationCommand checks a command for obfuscation
func handleObfuscationCommand(args []string) bool {
	if len(args) == 0 {
		fmt.Println("Usage: :validation obfuscation <command>")
		fmt.Println("Example: :validation obfuscation 'echo \"cm0gLXJmIC8=\" | base64 -d'")
		return true
	}
	
	command := strings.Join(args, " ")
	fmt.Printf("Analyzing command for obfuscation: %s\n\n", command)
	
	// Create detector
	detector := validation.NewObfuscationDetector()
	result := detector.DetectObfuscation(command)
	
	// Display results
	if !result.IsObfuscated {
		fmt.Println("✅ No obfuscation detected")
		return true
	}
	
	// Show obfuscation details
	fmt.Printf("%s Obfuscation Detected!\n", validation.GetRiskEmoji(result.RiskLevel))
	fmt.Printf("Risk Level: %s\n", validation.FormatRiskLevel(result.RiskLevel))
	fmt.Printf("Confidence: %.0f%%\n\n", result.Confidence*100)
	
	fmt.Println("Techniques Used:")
	for i, technique := range result.Techniques {
		fmt.Printf("  %d. %s\n", i+1, technique)
	}
	
	if result.Deobfuscated != "" && result.Deobfuscated != command {
		fmt.Println("\n🔍 Deobfuscated Command:")
		fmt.Printf("  %s\n", result.Deobfuscated)
		
		// Run safety check on deobfuscated command
		fmt.Println("\n🛡️  Safety Analysis of Deobfuscated Command:")
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
		
		valResult, err := engine.Validate(ctx, result.Deobfuscated)
		if err == nil {
			displaySafetyResult(valResult)
		}
	}
	
	fmt.Println("\n⚠️  Warning: Obfuscated commands are often used to hide malicious intent.")
	fmt.Println("Never run obfuscated commands without understanding what they do!")
	
	return true
}

// handleDeobfuscateCommand attempts to deobfuscate a command
func handleDeobfuscateCommand(args []string) bool {
	if len(args) == 0 {
		fmt.Println("Usage: :validation deobfuscate <command>")
		fmt.Println("Example: :validation deobfuscate 'echo \"bHMgLWxh\" | base64 -d'")
		return true
	}
	
	command := strings.Join(args, " ")
	
	// Create detector
	detector := validation.NewObfuscationDetector()
	result := detector.DetectObfuscation(command)
	
	if !result.IsObfuscated {
		fmt.Println("No obfuscation detected in this command.")
		fmt.Printf("Original: %s\n", command)
		return true
	}
	
	fmt.Println("🔍 Deobfuscation Results:")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━")
	
	fmt.Printf("\nOriginal Command:\n  %s\n", command)
	
	if result.Deobfuscated != "" && result.Deobfuscated != command {
		fmt.Printf("\nDeobfuscated Command:\n  %s\n", result.Deobfuscated)
		
		fmt.Printf("\nTechniques Detected:\n")
		for _, technique := range result.Techniques {
			fmt.Printf("  • %s\n", technique)
		}
		
		fmt.Printf("\nConfidence: %.0f%%\n", result.Confidence*100)
		
		// Warn about the deobfuscated command
		fmt.Println("\n⚠️  WARNING: The deobfuscated command may be dangerous!")
		fmt.Println("Review it carefully before considering execution.")
	} else {
		fmt.Println("\nUnable to fully deobfuscate this command.")
		fmt.Println("The obfuscation technique may be too complex or unknown.")
	}
	
	return true
}

// handleCustomRulesCommand manages custom validation rules
func handleCustomRulesCommand(args []string) bool {
	engine := GetValidationEngine()
	ruleEngine := engine.GetCustomRuleEngine()
	
	if ruleEngine == nil {
		fmt.Println("Custom rules are not enabled.")
		fmt.Println("Enable them with: :validation config set custom_rules true")
		return true
	}
	
	if len(args) == 0 {
		// Default to list
		return handleRulesList()
	}
	
	switch args[0] {
	case "list":
		return handleRulesList()
	case "add":
		return handleRulesAdd(args[1:])
	case "edit":
		return handleRulesEdit(args[1:])
	case "delete", "remove":
		return handleRulesDelete(args[1:])
	case "enable":
		return handleRulesEnable(args[1:])
	case "disable":
		return handleRulesDisable(args[1:])
	case "test":
		return handleRulesTest(args[1:])
	case "reload":
		return handleRulesReload()
	case "help":
		showCustomRulesHelp()
		return true
	default:
		fmt.Printf("Unknown rules command: %s\n", args[0])
		fmt.Println("Use ':validation rules help' for available commands")
		return true
	}
}

// handleRulesList shows all custom rules
func handleRulesList() bool {
	engine := GetValidationEngine()
	ruleEngine := engine.GetCustomRuleEngine()
	
	rules := ruleEngine.GetRules()
	if len(rules) == 0 {
		fmt.Println("No custom rules defined.")
		fmt.Println("Add rules with: :validation rules add")
		return true
	}
	
	fmt.Println("Custom Validation Rules:")
	fmt.Println("═══════════════════════")
	
	for i, rule := range rules {
		status := "✓ Enabled"
		if !rule.Enabled {
			status = "✗ Disabled"
		}
		
		fmt.Printf("\n%d. %s [%s]\n", i+1, rule.Name, status)
		fmt.Printf("   Description: %s\n", rule.Description)
		fmt.Printf("   Risk Level:  %s\n", validation.FormatRiskLevel(parseRiskLevel(rule.Risk)))
		fmt.Printf("   Pattern:     %s\n", rule.Pattern)
		if rule.Suggest != "" {
			fmt.Printf("   Suggestion:  %s\n", rule.Suggest)
		}
		if len(rule.Tags) > 0 {
			fmt.Printf("   Tags:        %s\n", strings.Join(rule.Tags, ", "))
		}
	}
	
	fmt.Println("\nTo test a command against rules: :validation rules test <command>")
	
	return true
}

// handleRulesAdd adds a new custom rule
func handleRulesAdd(args []string) bool {
	// For now, we'll create rules interactively
	fmt.Println("Adding a new custom rule")
	fmt.Println("═════════════════════")
	
	rule := validation.CustomRule{
		Enabled: true,
	}
	
	// Get rule details
	fmt.Print("Rule name (unique identifier): ")
	fmt.Scanln(&rule.Name)
	
	fmt.Print("Description: ")
	fmt.Scanln(&rule.Description)
	
	fmt.Print("Pattern (regex): ")
	fmt.Scanln(&rule.Pattern)
	
	fmt.Print("Risk level (low/medium/high/critical): ")
	fmt.Scanln(&rule.Risk)
	
	fmt.Print("Error message: ")
	fmt.Scanln(&rule.Message)
	
	fmt.Print("Suggestion (optional): ")
	fmt.Scanln(&rule.Suggest)
	
	// Add the rule
	engine := GetValidationEngine()
	ruleEngine := engine.GetCustomRuleEngine()
	
	if err := ruleEngine.AddRule(rule); err != nil {
		fmt.Printf("❌ Error adding rule: %v\n", err)
		return true
	}
	
	fmt.Printf("✅ Rule '%s' added successfully\n", rule.Name)
	return true
}

// handleRulesEdit edits an existing rule
func handleRulesEdit(args []string) bool {
	if len(args) == 0 {
		fmt.Println("Usage: :validation rules edit <rule_name>")
		return true
	}
	
	ruleName := args[0]
	engine := GetValidationEngine()
	ruleEngine := engine.GetCustomRuleEngine()
	
	rule, exists := ruleEngine.GetRule(ruleName)
	if !exists {
		fmt.Printf("Rule '%s' not found\n", ruleName)
		return true
	}
	
	fmt.Printf("Editing rule: %s\n", ruleName)
	fmt.Println("(Press Enter to keep current value)")
	
	// Edit fields
	fmt.Printf("Description [%s]: ", rule.Description)
	var newDesc string
	fmt.Scanln(&newDesc)
	if newDesc != "" {
		rule.Description = newDesc
	}
	
	fmt.Printf("Pattern [%s]: ", rule.Pattern)
	var newPattern string
	fmt.Scanln(&newPattern)
	if newPattern != "" {
		rule.Pattern = newPattern
	}
	
	// Update the rule
	if err := ruleEngine.UpdateRule(ruleName, *rule); err != nil {
		fmt.Printf("❌ Error updating rule: %v\n", err)
		return true
	}
	
	fmt.Printf("✅ Rule '%s' updated successfully\n", ruleName)
	return true
}

// handleRulesDelete removes a custom rule
func handleRulesDelete(args []string) bool {
	if len(args) == 0 {
		fmt.Println("Usage: :validation rules delete <rule_name>")
		return true
	}
	
	ruleName := args[0]
	engine := GetValidationEngine()
	ruleEngine := engine.GetCustomRuleEngine()
	
	if err := ruleEngine.DeleteRule(ruleName); err != nil {
		fmt.Printf("❌ Error deleting rule: %v\n", err)
		return true
	}
	
	fmt.Printf("✅ Rule '%s' deleted successfully\n", ruleName)
	return true
}

// handleRulesEnable enables a rule
func handleRulesEnable(args []string) bool {
	if len(args) == 0 {
		fmt.Println("Usage: :validation rules enable <rule_name>")
		return true
	}
	
	ruleName := args[0]
	engine := GetValidationEngine()
	ruleEngine := engine.GetCustomRuleEngine()
	
	if err := ruleEngine.EnableRule(ruleName); err != nil {
		fmt.Printf("❌ Error enabling rule: %v\n", err)
		return true
	}
	
	fmt.Printf("✅ Rule '%s' enabled\n", ruleName)
	return true
}

// handleRulesDisable disables a rule
func handleRulesDisable(args []string) bool {
	if len(args) == 0 {
		fmt.Println("Usage: :validation rules disable <rule_name>")
		return true
	}
	
	ruleName := args[0]
	engine := GetValidationEngine()
	ruleEngine := engine.GetCustomRuleEngine()
	
	if err := ruleEngine.DisableRule(ruleName); err != nil {
		fmt.Printf("❌ Error disabling rule: %v\n", err)
		return true
	}
	
	fmt.Printf("✅ Rule '%s' disabled\n", ruleName)
	return true
}

// handleRulesTest tests a command against custom rules
func handleRulesTest(args []string) bool {
	if len(args) == 0 {
		fmt.Println("Usage: :validation rules test <command>")
		return true
	}
	
	command := strings.Join(args, " ")
	engine := GetValidationEngine()
	ruleEngine := engine.GetCustomRuleEngine()
	
	fmt.Printf("Testing command: %s\n\n", command)
	
	matchedRules := ruleEngine.TestCommand(command)
	
	if len(matchedRules) == 0 {
		fmt.Println("✅ No custom rules matched this command")
		return true
	}
	
	fmt.Printf("⚠️  %d custom rule(s) matched:\n\n", len(matchedRules))
	
	for i, rule := range matchedRules {
		fmt.Printf("%d. %s [%s]\n", i+1, rule.Name, validation.FormatRiskLevel(parseRiskLevel(rule.Risk)))
		fmt.Printf("   Message: %s\n", rule.Message)
		if rule.Suggest != "" {
			fmt.Printf("   Suggestion: %s\n", rule.Suggest)
		}
		if !rule.Enabled {
			fmt.Printf("   Note: This rule is currently disabled\n")
		}
		fmt.Println()
	}
	
	return true
}

// handleRulesReload reloads rules from file
func handleRulesReload() bool {
	engine := GetValidationEngine()
	ruleEngine := engine.GetCustomRuleEngine()
	
	if err := ruleEngine.LoadRules(); err != nil {
		fmt.Printf("❌ Error reloading rules: %v\n", err)
		return true
	}
	
	rules := ruleEngine.GetRules()
	fmt.Printf("✅ Reloaded %d rules from file\n", len(rules))
	return true
}

// showCustomRulesHelp shows help for custom rules commands
func showCustomRulesHelp() {
	fmt.Println(`Custom Rules Commands:
════════════════════

Managing Rules:
  :validation rules list              List all custom rules
  :validation rules add               Add a new rule interactively
  :validation rules edit <name>       Edit an existing rule
  :validation rules delete <name>     Delete a rule
  :validation rules enable <name>     Enable a rule
  :validation rules disable <name>    Disable a rule
  :validation rules reload            Reload rules from file
  
Testing:
  :validation rules test <command>    Test command against all rules
  
Help:
  :validation rules help             Show this help

Rule File Location:
  ~/.config/delta/validation_rules.yaml

Rule Format:
  rules:
    - name: unique-identifier
      description: "What this rule checks"
      pattern: "regex pattern to match"
      risk: low|medium|high|critical
      message: "Error message when matched"
      suggest: "How to fix the issue"
      enabled: true
      tags: [security, git]`)
}

// parseRiskLevel converts string risk level to RiskLevel type
func parseRiskLevel(risk string) validation.RiskLevel {
	switch strings.ToLower(risk) {
	case "low":
		return validation.RiskLevelLow
	case "medium":
		return validation.RiskLevelMedium
	case "high":
		return validation.RiskLevelHigh
	case "critical":
		return validation.RiskLevelCritical
	default:
		return validation.RiskLevelMedium
	}
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