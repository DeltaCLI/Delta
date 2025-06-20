package validation

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SafetyPromptConfig configures interactive safety behavior
type SafetyPromptConfig struct {
	Enabled               bool
	RequireConfirmation   bool
	ShowEducationalInfo   bool
	TrackSafetyDecisions  bool
	AutoDenyLevel         RiskLevel
	BypassForTrustedPaths bool
}

// SafetyDecision represents a user's decision on a safety prompt
type SafetyDecision struct {
	Command      string
	RiskLevel    RiskLevel
	Decision     string // "proceed", "cancel", "modify"
	Timestamp    time.Time
	LearnedSafe  bool   // User marked as safe for future
}

// SafetyEducation provides educational content about risks
type SafetyEducation struct {
	RiskLevel    RiskLevel
	Title        string
	Description  string
	Consequences []string
	Alternatives []SafeAlternative
	LearnMoreURL string
}

// SafeAlternative suggests safer alternatives to dangerous commands
type SafeAlternative struct {
	Command     string
	Description string
	Example     string
	Safety      string // Why it's safer
}

// InteractiveSafetyChecker handles interactive safety prompts
type InteractiveSafetyChecker struct {
	config       SafetyPromptConfig
	history      []SafetyDecision
	trustedPaths map[string]bool
	education    map[RiskLevel]SafetyEducation
}

// NewInteractiveSafetyChecker creates a new interactive safety checker
func NewInteractiveSafetyChecker(config SafetyPromptConfig) *InteractiveSafetyChecker {
	checker := &InteractiveSafetyChecker{
		config:       config,
		history:      []SafetyDecision{},
		trustedPaths: make(map[string]bool),
		education:    make(map[RiskLevel]SafetyEducation),
	}
	
	// Initialize educational content
	checker.initializeEducation()
	
	// Load trusted paths
	checker.loadTrustedPaths()
	
	return checker
}

// initializeEducation sets up educational content for each risk level
func (c *InteractiveSafetyChecker) initializeEducation() {
	c.education[RiskLevelCritical] = SafetyEducation{
		RiskLevel:   RiskLevelCritical,
		Title:       "⚠️ CRITICAL RISK: System Destruction Warning",
		Description: "This command could permanently damage your system or delete critical data.",
		Consequences: []string{
			"Complete system failure requiring reinstallation",
			"Permanent loss of all data",
			"Corruption of system files",
			"Loss of user accounts and configurations",
		},
		Alternatives: []SafeAlternative{
			{
				Command:     "Use specific paths instead of /",
				Description: "Target specific directories rather than root",
				Example:     "rm -rf /tmp/specific-folder",
				Safety:      "Limits damage to intended targets only",
			},
			{
				Command:     "Use trash command",
				Description: "Move to trash instead of permanent deletion",
				Example:     "trash /path/to/file",
				Safety:      "Allows recovery if mistakes are made",
			},
		},
		LearnMoreURL: "https://wiki.deltacli.com/safety/critical-commands",
	}
	
	c.education[RiskLevelHigh] = SafetyEducation{
		RiskLevel:   RiskLevelHigh,
		Title:       "🟠 HIGH RISK: Dangerous Operation Detected",
		Description: "This command performs potentially harmful operations that could affect system stability or security.",
		Consequences: []string{
			"Security vulnerabilities if misconfigured",
			"Service disruptions",
			"Data exposure risks",
			"Difficult to reverse changes",
		},
		Alternatives: []SafeAlternative{
			{
				Command:     "Review before executing",
				Description: "Download and inspect scripts before running",
				Example:     "curl -o script.sh URL && cat script.sh",
				Safety:      "Allows inspection of code before execution",
			},
			{
				Command:     "Use restrictive permissions",
				Description: "Avoid world-writable permissions",
				Example:     "chmod 755 instead of chmod 777",
				Safety:      "Maintains security while providing needed access",
			},
		},
		LearnMoreURL: "https://wiki.deltacli.com/safety/high-risk-commands",
	}
	
	c.education[RiskLevelMedium] = SafetyEducation{
		RiskLevel:   RiskLevelMedium,
		Title:       "🟡 MEDIUM RISK: Caution Advised",
		Description: "This command could have unintended consequences. Please review carefully.",
		Consequences: []string{
			"Potential data loss if target is wrong",
			"May affect other users or processes",
			"Could require cleanup if mistakes occur",
		},
		Alternatives: []SafeAlternative{
			{
				Command:     "Test with dry-run first",
				Description: "Many commands support dry-run mode",
				Example:     "rsync --dry-run source/ dest/",
				Safety:      "Shows what would happen without making changes",
			},
			{
				Command:     "Create backups first",
				Description: "Backup data before modifications",
				Example:     "cp -a original original.bak",
				Safety:      "Allows restoration if needed",
			},
		},
		LearnMoreURL: "https://wiki.deltacli.com/safety/medium-risk-commands",
	}
}

// CheckInteractiveSafety performs interactive safety check
func (c *InteractiveSafetyChecker) CheckInteractiveSafety(result *ValidationResult) (bool, *SafetyDecision) {
	if !c.config.Enabled || result.RiskAssessment == nil {
		return true, nil // Allow if disabled or no risk assessment
	}
	
	risk := result.RiskAssessment.OverallRisk
	
	// Auto-deny if risk exceeds threshold
	if c.config.AutoDenyLevel != "" && isHigherRisk(risk, c.config.AutoDenyLevel) {
		decision := &SafetyDecision{
			Command:   result.Command,
			RiskLevel: risk,
			Decision:  "auto-denied",
			Timestamp: time.Now(),
		}
		c.recordDecision(decision)
		return false, decision
	}
	
	// Skip prompt for trusted paths if configured
	if c.config.BypassForTrustedPaths && c.isInTrustedPath() {
		return true, nil
	}
	
	// Skip prompt for low risk
	if risk == RiskLevelLow {
		return true, nil
	}
	
	// Show educational content if enabled
	if c.config.ShowEducationalInfo {
		c.showEducation(risk, result)
	}
	
	// Show risk summary
	c.showRiskSummary(result)
	
	// Prompt for confirmation if required
	if c.config.RequireConfirmation {
		return c.promptForConfirmation(result)
	}
	
	return true, nil
}

// showEducation displays educational content about the risk
func (c *InteractiveSafetyChecker) showEducation(risk RiskLevel, result *ValidationResult) {
	edu, ok := c.education[risk]
	if !ok {
		return
	}
	
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println(edu.Title)
	fmt.Println(strings.Repeat("=", 60))
	fmt.Printf("\n%s\n", edu.Description)
	
	if len(edu.Consequences) > 0 {
		fmt.Println("\n⚠️  Potential Consequences:")
		for _, consequence := range edu.Consequences {
			fmt.Printf("  • %s\n", consequence)
		}
	}
	
	if len(edu.Alternatives) > 0 {
		fmt.Println("\n💡 Safer Alternatives:")
		for _, alt := range edu.Alternatives {
			fmt.Printf("\n  %s\n", alt.Description)
			fmt.Printf("  Example: %s\n", alt.Example)
			fmt.Printf("  Why safer: %s\n", alt.Safety)
		}
	}
	
	if edu.LearnMoreURL != "" {
		fmt.Printf("\n📚 Learn more: %s\n", edu.LearnMoreURL)
	}
	
	fmt.Println(strings.Repeat("-", 60))
}

// showRiskSummary displays a summary of risks
func (c *InteractiveSafetyChecker) showRiskSummary(result *ValidationResult) {
	fmt.Printf("\n%s Command Risk Summary\n", GetRiskEmoji(result.RiskAssessment.OverallRisk))
	fmt.Printf("Command: %s\n", result.Command)
	
	if len(result.RiskAssessment.Factors) > 0 {
		fmt.Println("\nRisk Factors:")
		for _, factor := range result.RiskAssessment.Factors {
			fmt.Printf("  • %s\n", factor.Description)
		}
	}
}

// promptForConfirmation asks user whether to proceed
func (c *InteractiveSafetyChecker) promptForConfirmation(result *ValidationResult) (bool, *SafetyDecision) {
	reader := bufio.NewReader(os.Stdin)
	
	fmt.Println("\n" + strings.Repeat("━", 60))
	fmt.Printf("⚠️  Do you want to proceed with this %s risk command?\n", strings.ToUpper(string(result.RiskAssessment.OverallRisk)))
	fmt.Println(strings.Repeat("━", 60))
	fmt.Println("\nOptions:")
	fmt.Println("  [y] Yes, proceed with the command")
	fmt.Println("  [n] No, cancel the command")
	fmt.Println("  [m] Modify the command")
	fmt.Println("  [s] Mark as safe for future (proceed without prompts)")
	fmt.Println("  [?] Show more details")
	fmt.Print("\nYour choice [y/n/m/s/?]: ")
	
	input, err := reader.ReadString('\n')
	if err != nil {
		return false, nil
	}
	
	choice := strings.TrimSpace(strings.ToLower(input))
	
	decision := &SafetyDecision{
		Command:   result.Command,
		RiskLevel: result.RiskAssessment.OverallRisk,
		Timestamp: time.Now(),
	}
	
	switch choice {
	case "y", "yes":
		decision.Decision = "proceed"
		c.recordDecision(decision)
		return true, decision
		
	case "n", "no":
		decision.Decision = "cancel"
		c.recordDecision(decision)
		fmt.Println("\n❌ Command cancelled for safety.")
		return false, decision
		
	case "m", "modify":
		decision.Decision = "modify"
		c.recordDecision(decision)
		fmt.Println("\n✏️  Please modify your command and try again.")
		c.suggestModifications(result)
		return false, decision
		
	case "s", "safe":
		decision.Decision = "proceed"
		decision.LearnedSafe = true
		c.recordDecision(decision)
		c.markCommandAsSafe(result.Command)
		fmt.Println("\n✅ Command marked as safe for future use.")
		return true, decision
		
	case "?", "help":
		c.showDetailedHelp(result)
		// Recursive call to prompt again
		return c.promptForConfirmation(result)
		
	default:
		fmt.Println("\n❓ Invalid choice. Please try again.")
		// Recursive call to prompt again
		return c.promptForConfirmation(result)
	}
}

// suggestModifications shows how to modify the command to be safer
func (c *InteractiveSafetyChecker) suggestModifications(result *ValidationResult) {
	fmt.Println("\n💡 Suggested Modifications:")
	
	// Use suggestions from validation result
	for _, suggestion := range result.Suggestions {
		if suggestion.Alternative != "" {
			fmt.Printf("\n  Instead of: %s\n", result.Command)
			fmt.Printf("  Try: %s\n", suggestion.Alternative)
			fmt.Printf("  Reason: %s\n", suggestion.Explanation)
		}
	}
	
	// Add general safety tips
	fmt.Println("\n🛡️  General Safety Tips:")
	fmt.Println("  • Use specific paths instead of wildcards")
	fmt.Println("  • Add --dry-run or -n flags when available")
	fmt.Println("  • Create backups before destructive operations")
	fmt.Println("  • Use 'echo' to preview command expansion")
}

// showDetailedHelp displays detailed help about the risks
func (c *InteractiveSafetyChecker) showDetailedHelp(result *ValidationResult) {
	fmt.Println("\n📋 Detailed Risk Analysis:")
	
	// Show all validation errors
	if len(result.Errors) > 0 {
		fmt.Println("\n⚠️  Validation Issues:")
		for _, err := range result.Errors {
			fmt.Printf("  • %s\n", err.Message)
			if err.Suggestion != "" {
				fmt.Printf("    💡 %s\n", err.Suggestion)
			}
		}
	}
	
	// Show environment context
	if result.RiskAssessment != nil {
		ctx := result.RiskAssessment.Context
		fmt.Println("\n🌍 Environment Context:")
		fmt.Printf("  • Current Directory: %s\n", ctx.CurrentDirectory)
		fmt.Printf("  • Git Repository: %v\n", ctx.IsGitRepository)
		fmt.Printf("  • System Path: %v\n", ctx.IsSystemPath)
		fmt.Printf("  • Write Permission: %v\n", ctx.HasWritePermission)
	}
}

// recordDecision records a safety decision for tracking
func (c *InteractiveSafetyChecker) recordDecision(decision *SafetyDecision) {
	c.history = append(c.history, *decision)
	
	// TODO: Persist to disk for long-term tracking
}

// markCommandAsSafe marks a command pattern as safe
func (c *InteractiveSafetyChecker) markCommandAsSafe(command string) {
	// TODO: Implement pattern learning for safe commands
}

// isInTrustedPath checks if current directory is trusted
func (c *InteractiveSafetyChecker) isInTrustedPath() bool {
	cwd, err := os.Getwd()
	if err != nil {
		return false
	}
	
	// Check if current directory or any parent is trusted
	for path := cwd; path != "/" && path != ""; path = filepath.Dir(path) {
		if c.trustedPaths[path] {
			return true
		}
	}
	
	return false
}

// loadTrustedPaths loads trusted paths from configuration
func (c *InteractiveSafetyChecker) loadTrustedPaths() {
	// Default trusted paths (user's development directories)
	home, _ := os.UserHomeDir()
	if home != "" {
		c.trustedPaths[filepath.Join(home, "projects")] = true
		c.trustedPaths[filepath.Join(home, "dev")] = true
		c.trustedPaths[filepath.Join(home, "code")] = true
		c.trustedPaths[filepath.Join(home, "src")] = true
	}
	
	// TODO: Load from user configuration
}

// GetSafetyHistory returns recent safety decisions
func (c *InteractiveSafetyChecker) GetSafetyHistory() []SafetyDecision {
	return c.history
}

// GetSafetyStats returns statistics about safety decisions
func (c *InteractiveSafetyChecker) GetSafetyStats() map[string]int {
	stats := map[string]int{
		"total":     len(c.history),
		"proceeded": 0,
		"cancelled": 0,
		"modified":  0,
		"safe":      0,
	}
	
	for _, decision := range c.history {
		stats[decision.Decision]++
		if decision.LearnedSafe {
			stats["safe"]++
		}
	}
	
	return stats
}