package validation

import (
	"regexp"
	"strings"
)

// BasicSafetyRule implements a basic pattern-based safety check
type BasicSafetyRule struct {
	name        string
	description string
	pattern     *regexp.Regexp
	riskLevel   RiskLevel
	message     string
	suggestion  string
}

// Check implements SafetyRule interface
func (r *BasicSafetyRule) Check(command string, ast *AST) []ValidationError {
	errors := []ValidationError{}
	
	if r.pattern.MatchString(command) {
		// Map risk level to severity
		severity := SeverityWarning
		if r.riskLevel == RiskLevelHigh || r.riskLevel == RiskLevelCritical {
			severity = SeverityError
		} else if r.riskLevel == RiskLevelLow {
			severity = SeverityInfo
		}
		
		errors = append(errors, ValidationError{
			Type:       ErrorSafety,
			Severity:   severity,
			Message:    r.message,
			Rule:       r.name,
			Suggestion: r.suggestion,
			RiskLevel:  r.riskLevel,
		})
	}
	
	return errors
}

// GetName returns the rule name
func (r *BasicSafetyRule) GetName() string {
	return r.name
}

// GetDescription returns the rule description
func (r *BasicSafetyRule) GetDescription() string {
	return r.description
}

// GetRiskLevel returns the risk level of this rule
func (r *BasicSafetyRule) GetRiskLevel() RiskLevel {
	return r.riskLevel
}

// DefaultSafetyRules returns the default set of safety rules
func DefaultSafetyRules() []SafetyRule {
	return []SafetyRule{
		&BasicSafetyRule{
			name:        "RecursiveRootDelete",
			description: "Detects recursive deletion of root directory",
			pattern:     regexp.MustCompile(`\brm\s+(-[a-zA-Z]*r[a-zA-Z]*f|-[a-zA-Z]*f[a-zA-Z]*r)\s+/(\s|$)`),
			riskLevel:   RiskLevelCritical,
			message:     "CRITICAL: This command will recursively delete your entire system!",
			suggestion:  "Never run 'rm -rf /' - it will destroy your system. If you need to clean up, specify exact paths.",
		},
		&BasicSafetyRule{
			name:        "RecursiveHomeDelete",
			description: "Detects recursive deletion of home directory",
			pattern:     regexp.MustCompile(`\brm\s+(-[a-zA-Z]*r[a-zA-Z]*f|-[a-zA-Z]*f[a-zA-Z]*r)\s+(~|(\$HOME|\${HOME}))(/|(\s|$))`),
			riskLevel:   RiskLevelCritical,
			message:     "CRITICAL: This command will recursively delete your entire home directory!",
			suggestion:  "Be extremely careful with 'rm -rf ~' - specify exact subdirectories instead.",
		},
		&BasicSafetyRule{
			name:        "RecursiveDelete",
			description: "Detects potentially dangerous recursive deletion",
			pattern:     regexp.MustCompile(`\brm\s+(-[a-zA-Z]*r[a-zA-Z]*f|-[a-zA-Z]*f[a-zA-Z]*r)`),
			riskLevel:   RiskLevelHigh,
			message:     "Warning: Recursive deletion detected. This will permanently delete files and directories.",
			suggestion:  "Consider using 'trash' command instead of 'rm -rf' for recoverable deletion.",
		},
		&BasicSafetyRule{
			name:        "CurlBash",
			description: "Detects piping curl output directly to bash",
			pattern:     regexp.MustCompile(`curl\s+[^|]+\|\s*(sudo\s+)?(bash|sh)`),
			riskLevel:   RiskLevelHigh,
			message:     "Security Risk: Executing remote scripts without verification is dangerous.",
			suggestion:  "Download the script first, review it, then execute: curl -o script.sh URL && cat script.sh",
		},
		&BasicSafetyRule{
			name:        "DDCommand",
			description: "Detects dd command which can overwrite disks",
			pattern:     regexp.MustCompile(`\bdd\s+.*of=/dev/[a-zA-Z]`),
			riskLevel:   RiskLevelCritical,
			message:     "CRITICAL: dd command targeting a device - this can destroy disk data!",
			suggestion:  "Double-check the 'of=' parameter. Consider using 'dd status=progress' to monitor.",
		},
		&BasicSafetyRule{
			name:        "ChmodRecursive777",
			description: "Detects making files world-writable",
			pattern:     regexp.MustCompile(`chmod\s+(-[a-zA-Z]*R|--recursive)\s+777`),
			riskLevel:   RiskLevelHigh,
			message:     "Security Risk: Making files world-writable (777) is a major security vulnerability.",
			suggestion:  "Use more restrictive permissions like 755 or 644. Only give write access when necessary.",
		},
		&BasicSafetyRule{
			name:        "ForkBomb",
			description: "Detects fork bomb patterns",
			pattern:     regexp.MustCompile(`:\(\)\{.*:\|:&.*\};:`),
			riskLevel:   RiskLevelCritical,
			message:     "CRITICAL: Fork bomb detected! This will crash your system.",
			suggestion:  "This is a fork bomb that creates infinite processes. Never run this command.",
		},
		&BasicSafetyRule{
			name:        "SudoPasswordPipe",
			description: "Detects piping passwords to sudo",
			pattern:     regexp.MustCompile(`echo\s+[^|]+\|\s*sudo\s+-S`),
			riskLevel:   RiskLevelHigh,
			message:     "Security Risk: Piping passwords to sudo is insecure and may be logged.",
			suggestion:  "Use 'sudo' directly or configure NOPASSWD in sudoers for automation.",
		},
		&BasicSafetyRule{
			name:        "TruncateFile",
			description: "Detects file truncation",
			pattern:     regexp.MustCompile(`>\s*/[^/\s]+`),
			riskLevel:   RiskLevelMedium,
			message:     "Warning: '>' will overwrite the file completely. Data will be lost.",
			suggestion:  "Use '>>' to append instead of '>' to overwrite, or backup the file first.",
		},
		&BasicSafetyRule{
			name:        "SystemDirectoryModification",
			description: "Detects modifications to system directories",
			pattern:     regexp.MustCompile(`(rm|mv|chmod|chown)\s+.*(/etc|/usr|/bin|/sbin|/lib|/boot|/sys|/proc)`),
			riskLevel:   RiskLevelHigh,
			message:     "System directory modification detected. This could affect system stability.",
			suggestion:  "Ensure you have backups and understand the impact. Consider using configuration management tools.",
		},
		&BasicSafetyRule{
			name:        "WildcardWithRm",
			description: "Detects rm with wildcards",
			pattern:     regexp.MustCompile(`rm\s+[^|]*\*`),
			riskLevel:   RiskLevelMedium,
			message:     "Wildcard deletion detected. This may delete more files than intended.",
			suggestion:  "Use 'ls' first to verify which files match the pattern before deletion.",
		},
		&BasicSafetyRule{
			name:        "ServiceManipulation",
			description: "Detects service start/stop/restart commands",
			pattern:     regexp.MustCompile(`(systemctl|service)\s+(stop|restart|disable|mask)`),
			riskLevel:   RiskLevelMedium,
			message:     "Service manipulation detected. This may affect system services.",
			suggestion:  "Ensure you understand the service dependencies before stopping or disabling services.",
		},
		&BasicSafetyRule{
			name:        "NetworkConfigChange",
			description: "Detects network configuration changes",
			pattern:     regexp.MustCompile(`(ifconfig|ip\s+addr|iptables|firewall-cmd)\s+`),
			riskLevel:   RiskLevelHigh,
			message:     "Network configuration change detected. This may affect connectivity.",
			suggestion:  "Have a backup access method ready in case network access is lost.",
		},
		&BasicSafetyRule{
			name:        "KernelModuleOperation",
			description: "Detects kernel module operations",
			pattern:     regexp.MustCompile(`(insmod|rmmod|modprobe)\s+`),
			riskLevel:   RiskLevelHigh,
			message:     "Kernel module operation detected. This affects core system functionality.",
			suggestion:  "Ensure the module is compatible with your kernel version.",
		},
		&BasicSafetyRule{
			name:        "GitForceOperation",
			description: "Detects git force operations",
			pattern:     regexp.MustCompile(`git\s+.*--force|git\s+push\s+.*-f`),
			riskLevel:   RiskLevelMedium,
			message:     "Git force operation detected. This may overwrite remote history.",
			suggestion:  "Consider if force push is necessary. It can cause issues for other collaborators.",
		},
		&BasicSafetyRule{
			name:        "DatabaseDropOperation",
			description: "Detects database drop operations",
			pattern:     regexp.MustCompile(`(DROP\s+(DATABASE|TABLE)|mysql.*-e.*drop|psql.*-c.*drop)`),
			riskLevel:   RiskLevelCritical,
			message:     "Database drop operation detected! This will permanently delete data.",
			suggestion:  "Create a backup before dropping databases or tables.",
		},
	}
}

// SimpleSyntaxCheck performs basic syntax validation without full parsing
func SimpleSyntaxCheck(command string) []ValidationError {
	errors := []ValidationError{}
	
	// Check for unmatched quotes
	singleQuotes := strings.Count(command, "'") % 2
	doubleQuotes := strings.Count(command, "\"") % 2
	
	if singleQuotes != 0 {
		errors = append(errors, ValidationError{
			Type:       ErrorSyntax,
			Severity:   SeverityError,
			Message:    "Unmatched single quote",
			Suggestion: "Add a closing single quote (') to match the opening quote",
		})
	}
	
	if doubleQuotes != 0 {
		errors = append(errors, ValidationError{
			Type:       ErrorSyntax,
			Severity:   SeverityError,
			Message:    "Unmatched double quote",
			Suggestion: "Add a closing double quote (\") to match the opening quote",
		})
	}
	
	// Check for trailing pipe
	trimmed := strings.TrimSpace(command)
	if strings.HasSuffix(trimmed, "|") {
		errors = append(errors, ValidationError{
			Type:       ErrorSyntax,
			Severity:   SeverityError,
			Message:    "Unexpected end of command after pipe",
			Suggestion: "Add a command after the pipe (|) or remove the pipe",
		})
	}
	
	// Check for empty command
	if trimmed == "" {
		errors = append(errors, ValidationError{
			Type:     ErrorSyntax,
			Severity: SeverityError,
			Message:  "Empty command",
		})
	}
	
	return errors
}