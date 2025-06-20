package validation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RiskLevel represents the severity of a command's potential impact
type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

// RiskFactor represents a specific risk associated with a command
type RiskFactor struct {
	Type        string
	Description string
	Level       RiskLevel
	Mitigation  string
}

// RiskAssessment contains the complete risk analysis for a command
type RiskAssessment struct {
	OverallRisk   RiskLevel
	Factors       []RiskFactor
	RequiresRoot  bool
	AffectsSystem bool
	IsIrreversible bool
	Context       EnvironmentContext
}

// EnvironmentContext provides context about the current environment
type EnvironmentContext struct {
	CurrentDirectory string
	IsGitRepository  bool
	IsHomeDirectory  bool
	IsSystemPath     bool
	HasWritePermission bool
	ImportantPaths   []string
}

// GetEnvironmentContext analyzes the current environment
func GetEnvironmentContext() EnvironmentContext {
	ctx := EnvironmentContext{
		ImportantPaths: []string{
			"/etc", "/usr", "/bin", "/sbin", "/boot", "/dev", "/proc", "/sys",
			"/lib", "/lib64", "/var", "/opt", "/root",
		},
	}
	
	// Get current directory
	if cwd, err := os.Getwd(); err == nil {
		ctx.CurrentDirectory = cwd
		
		// Check if in home directory
		if home, err := os.UserHomeDir(); err == nil {
			ctx.IsHomeDirectory = strings.HasPrefix(cwd, home)
		}
		
		// Check if in system path
		for _, sysPath := range ctx.ImportantPaths {
			if strings.HasPrefix(cwd, sysPath) {
				ctx.IsSystemPath = true
				break
			}
		}
		
		// Check write permission
		testFile := filepath.Join(cwd, ".delta_test_write")
		if file, err := os.Create(testFile); err == nil {
			ctx.HasWritePermission = true
			file.Close()
			os.Remove(testFile)
		}
		
		// Check if git repository
		gitDir := filepath.Join(cwd, ".git")
		if stat, err := os.Stat(gitDir); err == nil && stat.IsDir() {
			ctx.IsGitRepository = true
		} else {
			// Check parent directories for .git
			dir := cwd
			for i := 0; i < 5 && dir != "/" && dir != ""; i++ {
				dir = filepath.Dir(dir)
				gitDir = filepath.Join(dir, ".git")
				if stat, err := os.Stat(gitDir); err == nil && stat.IsDir() {
					ctx.IsGitRepository = true
					break
				}
			}
		}
	}
	
	return ctx
}

// AssessCommandRisk performs comprehensive risk assessment
func AssessCommandRisk(command string, errors []ValidationError, ctx EnvironmentContext) RiskAssessment {
	assessment := RiskAssessment{
		OverallRisk: RiskLevelLow,
		Context:     ctx,
		Factors:     []RiskFactor{},
	}
	
	// Check permission requirements
	permReq := CheckPermissionRequirements(command, ctx)
	
	// Add permission-related risk factors
	if permReq.RequiresRoot {
		assessment.RequiresRoot = true
		if len(permReq.MissingPermissions) > 0 {
			assessment.Factors = append(assessment.Factors, RiskFactor{
				Type:        "permission",
				Description: "Command requires elevated privileges that are not available",
				Level:       RiskLevelHigh,
				Mitigation:  "Run with sudo or as root user",
			})
		}
	}
	
	// Add risk factors for missing permissions
	for _, missing := range permReq.MissingPermissions {
		assessment.Factors = append(assessment.Factors, RiskFactor{
			Type:        "permission",
			Description: missing,
			Level:       RiskLevelMedium,
			Mitigation:  "Ensure you have necessary permissions or use sudo",
		})
	}
	
	// Analyze validation errors for risk factors
	for _, err := range errors {
		if err.Type == ErrorSafety {
			// Use the risk level from the error if available
			factor := RiskFactor{
				Type:        string(err.Type),
				Description: err.Message,
				Mitigation:  err.Suggestion,
				Level:       err.RiskLevel,
			}
			
			// Check if operation is irreversible
			if err.RiskLevel == RiskLevelCritical {
				assessment.IsIrreversible = true
			}
			
			assessment.Factors = append(assessment.Factors, factor)
		}
	}
	
	// Check for sudo/root requirements
	if strings.Contains(command, "sudo") || strings.Contains(command, "su ") {
		assessment.RequiresRoot = true
		assessment.Factors = append(assessment.Factors, RiskFactor{
			Type:        "permission",
			Description: "Command requires elevated privileges",
			Level:       RiskLevelMedium,
			Mitigation:  "Ensure you understand why root access is needed",
		})
	}
	
	// Check if command affects system directories
	for _, sysPath := range ctx.ImportantPaths {
		if strings.Contains(command, sysPath) {
			assessment.AffectsSystem = true
			assessment.Factors = append(assessment.Factors, RiskFactor{
				Type:        "system",
				Description: fmt.Sprintf("Command affects system directory: %s", sysPath),
				Level:       RiskLevelHigh,
				Mitigation:  "Be extra careful when modifying system directories",
			})
			break
		}
	}
	
	// Context-based risk adjustments
	if ctx.IsGitRepository {
		// Check for git-dangerous operations
		if strings.Contains(command, "git reset --hard") || strings.Contains(command, "git clean -fd") {
			assessment.Factors = append(assessment.Factors, RiskFactor{
				Type:        "git",
				Description: "Command will permanently discard git changes",
				Level:       RiskLevelMedium,
				Mitigation:  "Consider using 'git stash' to save changes first",
			})
		}
		
		if strings.Contains(command, "force") || strings.Contains(command, " -f") {
			assessment.Factors = append(assessment.Factors, RiskFactor{
				Type:        "force",
				Description: "Force flag detected - bypasses safety checks",
				Level:       RiskLevelMedium,
				Mitigation:  "Remove force flag unless absolutely necessary",
			})
		}
	}
	
	// Determine overall risk level
	maxRisk := RiskLevelLow
	for _, factor := range assessment.Factors {
		if isHigherRisk(factor.Level, maxRisk) {
			maxRisk = factor.Level
		}
	}
	assessment.OverallRisk = maxRisk
	
	return assessment
}

// isHigherRisk compares two risk levels
func isHigherRisk(a, b RiskLevel) bool {
	riskOrder := map[RiskLevel]int{
		RiskLevelLow:      1,
		RiskLevelMedium:   2,
		RiskLevelHigh:     3,
		RiskLevelCritical: 4,
	}
	return riskOrder[a] > riskOrder[b]
}

// GetRiskEmoji returns an emoji representation of the risk level
func GetRiskEmoji(level RiskLevel) string {
	switch level {
	case RiskLevelLow:
		return "🟢"
	case RiskLevelMedium:
		return "🟡"
	case RiskLevelHigh:
		return "🟠"
	case RiskLevelCritical:
		return "🔴"
	default:
		return "⚪"
	}
}

// GetRiskColor returns ANSI color code for risk level
func GetRiskColor(level RiskLevel) string {
	switch level {
	case RiskLevelLow:
		return "\033[32m" // Green
	case RiskLevelMedium:
		return "\033[33m" // Yellow
	case RiskLevelHigh:
		return "\033[38;5;208m" // Orange
	case RiskLevelCritical:
		return "\033[31m" // Red
	default:
		return "\033[0m" // Reset
	}
}

// FormatRiskLevel returns a formatted string for the risk level
func FormatRiskLevel(level RiskLevel) string {
	emoji := GetRiskEmoji(level)
	color := GetRiskColor(level)
	reset := "\033[0m"
	
	levelStr := strings.Title(string(level))
	return fmt.Sprintf("%s %s%s Risk%s", emoji, color, levelStr, reset)
}