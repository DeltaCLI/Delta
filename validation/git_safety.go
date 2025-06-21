package validation

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// GitSafetyChecker provides specialized safety checks for git operations
type GitSafetyChecker struct {
	protectedBranches []string
	sensitivePatterns []string
	maxFileSize       int64
	gitContext        *GitContext
}

// GitContext contains information about the current git repository
type GitContext struct {
	IsGitRepo          bool
	CurrentBranch      string
	HasUncommittedChanges bool
	RemoteURL          string
	RootPath           string
}

// NewGitSafetyChecker creates a new git safety checker
func NewGitSafetyChecker() *GitSafetyChecker {
	return &GitSafetyChecker{
		protectedBranches: []string{"main", "master", "production", "release"},
		sensitivePatterns: []string{
			`\.pem$`, `\.key$`, `\.pfx$`, `\.p12$`,
			`\.env$`, `\.env\.`, `\.envrc$`,
			`id_rsa`, `id_dsa`, `id_ecdsa`, `id_ed25519`,
			`\.secrets$`, `\.password`, `\.passwd$`,
			`\.aws/credentials`, `\.ssh/`,
		},
		maxFileSize: 100 * 1024 * 1024, // 100MB
	}
}

// GetGitContext retrieves information about the current git repository
func (g *GitSafetyChecker) GetGitContext() *GitContext {
	if g.gitContext != nil {
		return g.gitContext
	}
	
	ctx := &GitContext{}
	
	// Check if we're in a git repository
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	if output, err := cmd.Output(); err == nil {
		ctx.IsGitRepo = true
		gitDir := strings.TrimSpace(string(output))
		if gitDir == ".git" {
			ctx.RootPath = "."
		} else {
			ctx.RootPath = filepath.Dir(gitDir)
		}
	}
	
	if !ctx.IsGitRepo {
		return ctx
	}
	
	// Get current branch
	cmd = exec.Command("git", "branch", "--show-current")
	if output, err := cmd.Output(); err == nil {
		ctx.CurrentBranch = strings.TrimSpace(string(output))
	}
	
	// Check for uncommitted changes
	cmd = exec.Command("git", "status", "--porcelain")
	if output, err := cmd.Output(); err == nil {
		ctx.HasUncommittedChanges = len(output) > 0
	}
	
	// Get remote URL
	cmd = exec.Command("git", "remote", "get-url", "origin")
	if output, err := cmd.Output(); err == nil {
		ctx.RemoteURL = strings.TrimSpace(string(output))
	}
	
	g.gitContext = ctx
	return ctx
}

// CheckGitCommand performs git-specific safety checks
func (g *GitSafetyChecker) CheckGitCommand(command string) []ValidationError {
	var errors []ValidationError
	
	// Parse git command
	gitCmd := g.parseGitCommand(command)
	if gitCmd == nil {
		return errors
	}
	
	// Get git context
	ctx := g.GetGitContext()
	
	switch gitCmd.subcommand {
	case "push":
		errors = append(errors, g.checkGitPush(gitCmd, ctx)...)
	case "reset":
		errors = append(errors, g.checkGitReset(gitCmd, ctx)...)
	case "clean":
		errors = append(errors, g.checkGitClean(gitCmd, ctx)...)
	case "rebase":
		errors = append(errors, g.checkGitRebase(gitCmd, ctx)...)
	case "force":
		errors = append(errors, g.checkGitForce(gitCmd, ctx)...)
	case "add":
		errors = append(errors, g.checkGitAdd(gitCmd, ctx)...)
	case "commit":
		errors = append(errors, g.checkGitCommit(gitCmd, ctx)...)
	}
	
	return errors
}

// gitCommand represents a parsed git command
type gitCommand struct {
	subcommand string
	flags      []string
	args       []string
	hasForce   bool
}

// parseGitCommand parses a git command into its components
func (g *GitSafetyChecker) parseGitCommand(command string) *gitCommand {
	// Simple regex to match git commands
	gitRegex := regexp.MustCompile(`^\s*git\s+(\S+)(.*)$`)
	matches := gitRegex.FindStringSubmatch(command)
	
	if len(matches) < 2 {
		return nil
	}
	
	cmd := &gitCommand{
		subcommand: matches[1],
		flags:      []string{},
		args:       []string{},
	}
	
	// Parse flags and arguments
	remaining := strings.TrimSpace(matches[2])
	parts := strings.Fields(remaining)
	
	for _, part := range parts {
		if strings.HasPrefix(part, "-") {
			cmd.flags = append(cmd.flags, part)
			if strings.Contains(part, "force") {
				cmd.hasForce = true
			}
		} else {
			cmd.args = append(cmd.args, part)
		}
	}
	
	return cmd
}

// checkGitPush checks for dangerous push operations
func (g *GitSafetyChecker) checkGitPush(cmd *gitCommand, ctx *GitContext) []ValidationError {
	var errors []ValidationError
	
	// Check for force push to protected branches
	if cmd.hasForce {
		targetBranch := g.getPushTargetBranch(cmd, ctx)
		if g.isProtectedBranch(targetBranch) {
			errors = append(errors, ValidationError{
				Type:       ErrorSafety,
				Severity:   SeverityError,
				Message:    fmt.Sprintf("Force pushing to protected branch '%s' is dangerous", targetBranch),
				Rule:       "git-force-push-protected",
				Suggestion: "Create a feature branch or use --force-with-lease for safer force pushes",
				RiskLevel:  RiskLevelHigh,
			})
		}
	}
	
	// Check for pushing large files
	// Note: This would require more complex logic to actually check file sizes
	
	return errors
}

// checkGitReset checks for dangerous reset operations
func (g *GitSafetyChecker) checkGitReset(cmd *gitCommand, ctx *GitContext) []ValidationError {
	var errors []ValidationError
	
	// Check for hard reset with uncommitted changes
	hasHard := false
	for _, flag := range cmd.flags {
		if strings.Contains(flag, "hard") {
			hasHard = true
			break
		}
	}
	
	if hasHard && ctx.HasUncommittedChanges {
		errors = append(errors, ValidationError{
			Type:       ErrorSafety,
			Severity:   SeverityWarning,
			Message:    "Hard reset will discard uncommitted changes",
			Rule:       "git-reset-hard-uncommitted",
			Suggestion: "Stash your changes first with 'git stash' or commit them",
			RiskLevel:  RiskLevelMedium,
		})
	}
	
	return errors
}

// checkGitClean checks for dangerous clean operations
func (g *GitSafetyChecker) checkGitClean(cmd *gitCommand, ctx *GitContext) []ValidationError {
	var errors []ValidationError
	
	// Check for -fdx flags (force, directories, ignored files)
	hasF := false
	hasD := false
	hasX := false
	
	for _, flag := range cmd.flags {
		if strings.Contains(flag, "f") {
			hasF = true
		}
		if strings.Contains(flag, "d") {
			hasD = true
		}
		if strings.Contains(flag, "x") || strings.Contains(flag, "X") {
			hasX = true
		}
	}
	
	if hasF && hasD && hasX {
		errors = append(errors, ValidationError{
			Type:       ErrorSafety,
			Severity:   SeverityWarning,
			Message:    "git clean -fdx will remove all untracked files, directories, and ignored files",
			Rule:       "git-clean-aggressive",
			Suggestion: "Use 'git clean -n' first to preview what will be deleted",
			RiskLevel:  RiskLevelHigh,
		})
	}
	
	return errors
}

// checkGitRebase checks for dangerous rebase operations
func (g *GitSafetyChecker) checkGitRebase(cmd *gitCommand, ctx *GitContext) []ValidationError {
	var errors []ValidationError
	
	// Check for rebasing protected branches
	if g.isProtectedBranch(ctx.CurrentBranch) {
		errors = append(errors, ValidationError{
			Type:       ErrorSafety,
			Severity:   SeverityWarning,
			Message:    fmt.Sprintf("Rebasing protected branch '%s' can cause issues for other developers", ctx.CurrentBranch),
			Rule:       "git-rebase-protected",
			Suggestion: "Consider creating a feature branch for rebasing",
			RiskLevel:  RiskLevelMedium,
		})
	}
	
	return errors
}

// checkGitForce checks for any force operations
func (g *GitSafetyChecker) checkGitForce(cmd *gitCommand, ctx *GitContext) []ValidationError {
	// This is handled by specific subcommand checks
	return []ValidationError{}
}

// checkGitAdd checks for adding sensitive files
func (g *GitSafetyChecker) checkGitAdd(cmd *gitCommand, ctx *GitContext) []ValidationError {
	var errors []ValidationError
	
	// Check if adding files that match sensitive patterns
	for _, arg := range cmd.args {
		if g.isSensitiveFile(arg) {
			errors = append(errors, ValidationError{
				Type:       ErrorSafety,
				Severity:   SeverityWarning,
				Message:    fmt.Sprintf("Adding potentially sensitive file: %s", arg),
				Rule:       "git-add-sensitive",
				Suggestion: "Ensure this file doesn't contain secrets. Consider using .gitignore",
				RiskLevel:  RiskLevelHigh,
			})
		}
	}
	
	return errors
}

// checkGitCommit checks for commit message issues
func (g *GitSafetyChecker) checkGitCommit(cmd *gitCommand, ctx *GitContext) []ValidationError {
	// Could check for commit message patterns, secrets in messages, etc.
	return []ValidationError{}
}

// Helper methods

// isProtectedBranch checks if a branch name is protected
func (g *GitSafetyChecker) isProtectedBranch(branch string) bool {
	for _, protected := range g.protectedBranches {
		if branch == protected {
			return true
		}
	}
	return false
}

// isSensitiveFile checks if a file path matches sensitive patterns
func (g *GitSafetyChecker) isSensitiveFile(path string) bool {
	for _, pattern := range g.sensitivePatterns {
		if matched, _ := regexp.MatchString(pattern, path); matched {
			return true
		}
	}
	return false
}

// getPushTargetBranch attempts to determine the target branch for a push
func (g *GitSafetyChecker) getPushTargetBranch(cmd *gitCommand, ctx *GitContext) string {
	// Look for branch name in arguments
	for i, arg := range cmd.args {
		if arg == "origin" && i+1 < len(cmd.args) {
			return cmd.args[i+1]
		}
		if strings.Contains(arg, ":") {
			parts := strings.Split(arg, ":")
			if len(parts) == 2 {
				return parts[1]
			}
		}
	}
	
	// Default to current branch
	return ctx.CurrentBranch
}

// CreateGitSafetyRule creates a safety rule that uses the GitSafetyChecker
type GitSafetyRule struct {
	checker *GitSafetyChecker
}

// NewGitSafetyRule creates a new git safety rule
func NewGitSafetyRule() *GitSafetyRule {
	return &GitSafetyRule{
		checker: NewGitSafetyChecker(),
	}
}

// Check implements the SafetyRule interface
func (r *GitSafetyRule) Check(command string, ast *AST) []ValidationError {
	// Only check git commands
	if !strings.HasPrefix(strings.TrimSpace(command), "git") {
		return []ValidationError{}
	}
	
	return r.checker.CheckGitCommand(command)
}

// GetName implements the SafetyRule interface
func (r *GitSafetyRule) GetName() string {
	return "GitSafety"
}

// GetDescription implements the SafetyRule interface
func (r *GitSafetyRule) GetDescription() string {
	return "Checks for dangerous git operations"
}

// GetRiskLevel implements the SafetyRule interface
func (r *GitSafetyRule) GetRiskLevel() RiskLevel {
	return RiskLevelMedium
}