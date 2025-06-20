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
	riskLevel   string
	message     string
	suggestion  string
}

// Check implements SafetyRule interface
func (r *BasicSafetyRule) Check(command string, ast *AST) []ValidationError {
	errors := []ValidationError{}
	
	if r.pattern.MatchString(command) {
		errors = append(errors, ValidationError{
			Type:       ErrorSafety,
			Severity:   SeverityError,
			Message:    r.message,
			Rule:       r.name,
			Suggestion: r.suggestion,
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

// DefaultSafetyRules returns the default set of safety rules
func DefaultSafetyRules() []SafetyRule {
	return []SafetyRule{
		&BasicSafetyRule{
			name:        "RecursiveRootDelete",
			description: "Detects recursive deletion of root directory",
			pattern:     regexp.MustCompile(`\brm\s+(-[a-zA-Z]*r[a-zA-Z]*f|-[a-zA-Z]*f[a-zA-Z]*r)\s+/(\s|$)`),
			riskLevel:   "critical",
			message:     "CRITICAL: This command will recursively delete your entire system!",
			suggestion:  "Never run 'rm -rf /' - it will destroy your system. If you need to clean up, specify exact paths.",
		},
		&BasicSafetyRule{
			name:        "RecursiveHomeDelete",
			description: "Detects recursive deletion of home directory",
			pattern:     regexp.MustCompile(`\brm\s+(-[a-zA-Z]*r[a-zA-Z]*f|-[a-zA-Z]*f[a-zA-Z]*r)\s+(~|(\$HOME|\${HOME}))(/|(\s|$))`),
			riskLevel:   "critical",
			message:     "CRITICAL: This command will recursively delete your entire home directory!",
			suggestion:  "Be extremely careful with 'rm -rf ~' - specify exact subdirectories instead.",
		},
		&BasicSafetyRule{
			name:        "RecursiveDelete",
			description: "Detects potentially dangerous recursive deletion",
			pattern:     regexp.MustCompile(`\brm\s+(-[a-zA-Z]*r[a-zA-Z]*f|-[a-zA-Z]*f[a-zA-Z]*r)`),
			riskLevel:   "high",
			message:     "Warning: Recursive deletion detected. This will permanently delete files and directories.",
			suggestion:  "Consider using 'trash' command instead of 'rm -rf' for recoverable deletion.",
		},
		&BasicSafetyRule{
			name:        "CurlBash",
			description: "Detects piping curl output directly to bash",
			pattern:     regexp.MustCompile(`curl\s+[^|]+\|\s*(sudo\s+)?(bash|sh)`),
			riskLevel:   "high",
			message:     "Security Risk: Executing remote scripts without verification is dangerous.",
			suggestion:  "Download the script first, review it, then execute: curl -o script.sh URL && cat script.sh",
		},
		&BasicSafetyRule{
			name:        "DDCommand",
			description: "Detects dd command which can overwrite disks",
			pattern:     regexp.MustCompile(`\bdd\s+.*of=/dev/[a-zA-Z]`),
			riskLevel:   "critical",
			message:     "CRITICAL: dd command targeting a device - this can destroy disk data!",
			suggestion:  "Double-check the 'of=' parameter. Consider using 'dd status=progress' to monitor.",
		},
		&BasicSafetyRule{
			name:        "ChmodRecursive777",
			description: "Detects making files world-writable",
			pattern:     regexp.MustCompile(`chmod\s+(-[a-zA-Z]*R|--recursive)\s+777`),
			riskLevel:   "high",
			message:     "Security Risk: Making files world-writable (777) is a major security vulnerability.",
			suggestion:  "Use more restrictive permissions like 755 or 644. Only give write access when necessary.",
		},
		&BasicSafetyRule{
			name:        "ForkBomb",
			description: "Detects fork bomb patterns",
			pattern:     regexp.MustCompile(`:\(\)\{.*:\|:&.*\};:`),
			riskLevel:   "critical",
			message:     "CRITICAL: Fork bomb detected! This will crash your system.",
			suggestion:  "This is a fork bomb that creates infinite processes. Never run this command.",
		},
		&BasicSafetyRule{
			name:        "SudoPasswordPipe",
			description: "Detects piping passwords to sudo",
			pattern:     regexp.MustCompile(`echo\s+[^|]+\|\s*sudo\s+-S`),
			riskLevel:   "high",
			message:     "Security Risk: Piping passwords to sudo is insecure and may be logged.",
			suggestion:  "Use 'sudo' directly or configure NOPASSWD in sudoers for automation.",
		},
		&BasicSafetyRule{
			name:        "TruncateFile",
			description: "Detects file truncation",
			pattern:     regexp.MustCompile(`>\s*/[^/\s]+`),
			riskLevel:   "medium",
			message:     "Warning: '>' will overwrite the file completely. Data will be lost.",
			suggestion:  "Use '>>' to append instead of '>' to overwrite, or backup the file first.",
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