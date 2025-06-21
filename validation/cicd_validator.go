package validation

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// CIPlatform represents different CI/CD platforms
type CIPlatform string

const (
	GitHubActions CIPlatform = "github-actions"
	GitLabCI      CIPlatform = "gitlab-ci"
	CircleCI      CIPlatform = "circleci"
	Jenkins       CIPlatform = "jenkins"
	TravisCI      CIPlatform = "travis-ci"
	AzurePipelines CIPlatform = "azure-pipelines"
	Unknown       CIPlatform = "unknown"
)

// CICDValidator provides specialized validation for CI/CD environments
type CICDValidator struct {
	platform      CIPlatform
	secretsRegex  []string
	envRules      []EnvRule
	isCI          bool
}

// EnvRule defines rules for environment variables in CI/CD
type EnvRule struct {
	Name        string
	Pattern     string
	Required    bool
	Sensitive   bool
	Description string
}

// NewCICDValidator creates a new CI/CD validator
func NewCICDValidator() *CICDValidator {
	validator := &CICDValidator{
		platform: detectCIPlatform(),
		isCI:     isRunningInCI(),
		secretsRegex: []string{
			`(?i)(password|passwd|pwd)\s*[:=]\s*\S+`,
			`(?i)(secret|token|key|api_key|apikey)\s*[:=]\s*\S+`,
			`(?i)(aws_access_key_id|aws_secret_access_key)\s*[:=]\s*\S+`,
			`(?i)(auth|authorization|bearer)\s*[:=]\s*\S+`,
			`(?i)-----BEGIN\s+(RSA\s+)?PRIVATE\s+KEY-----`,
			`(?i)(mysql|postgres|mongodb|redis)://[^@]+:[^@]+@`,
			`(?i)sqlite://.*\.db`,
			`(?i)(github_token|gh_token|gitlab_token|gl_token)\s*[:=]\s*\S+`,
			`(?i)(npm_token|pypi_token|gem_token)\s*[:=]\s*\S+`,
		},
		envRules: defaultEnvRules(),
	}
	
	return validator
}

// detectCIPlatform detects which CI/CD platform we're running on
func detectCIPlatform() CIPlatform {
	// GitHub Actions
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		return GitHubActions
	}
	
	// GitLab CI
	if os.Getenv("GITLAB_CI") == "true" {
		return GitLabCI
	}
	
	// CircleCI
	if os.Getenv("CIRCLECI") == "true" {
		return CircleCI
	}
	
	// Jenkins
	if os.Getenv("JENKINS_URL") != "" || os.Getenv("BUILD_ID") != "" {
		return Jenkins
	}
	
	// Travis CI
	if os.Getenv("TRAVIS") == "true" {
		return TravisCI
	}
	
	// Azure Pipelines
	if os.Getenv("TF_BUILD") == "True" {
		return AzurePipelines
	}
	
	return Unknown
}

// isRunningInCI checks if we're running in any CI environment
func isRunningInCI() bool {
	ciVars := []string{
		"CI",
		"CONTINUOUS_INTEGRATION",
		"BUILD_ID",
		"BUILD_NUMBER",
		"GITHUB_ACTIONS",
		"GITLAB_CI",
		"CIRCLECI",
		"JENKINS_URL",
		"TRAVIS",
		"TF_BUILD",
	}
	
	for _, v := range ciVars {
		if os.Getenv(v) != "" {
			return true
		}
	}
	
	return false
}

// defaultEnvRules returns default environment variable rules for CI/CD
func defaultEnvRules() []EnvRule {
	return []EnvRule{
		{
			Name:        "AWS_ACCESS_KEY_ID",
			Pattern:     `^AKIA[0-9A-Z]{16}$`,
			Sensitive:   true,
			Description: "AWS Access Key ID",
		},
		{
			Name:        "AWS_SECRET_ACCESS_KEY",
			Pattern:     `^[A-Za-z0-9/+=]{40}$`,
			Sensitive:   true,
			Description: "AWS Secret Access Key",
		},
		{
			Name:        "GITHUB_TOKEN",
			Pattern:     `^gh[ps]_[a-zA-Z0-9]{36}$`,
			Sensitive:   true,
			Description: "GitHub Personal Access Token",
		},
		{
			Name:        "NPM_TOKEN",
			Pattern:     `^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$`,
			Sensitive:   true,
			Description: "NPM Registry Token",
		},
		{
			Name:        "DOCKER_PASSWORD",
			Pattern:     `.+`,
			Sensitive:   true,
			Description: "Docker Registry Password",
		},
	}
}

// ValidateCommand validates a command in CI/CD context
func (v *CICDValidator) ValidateCommand(command string) []ValidationError {
	var errors []ValidationError
	
	// Check for secrets in command
	errors = append(errors, v.checkForSecrets(command)...)
	
	// Platform-specific checks
	switch v.platform {
	case GitHubActions:
		errors = append(errors, v.checkGitHubActions(command)...)
	case GitLabCI:
		errors = append(errors, v.checkGitLabCI(command)...)
	case CircleCI:
		errors = append(errors, v.checkCircleCI(command)...)
	case Jenkins:
		errors = append(errors, v.checkJenkins(command)...)
	}
	
	// Check for environment variable exposure
	errors = append(errors, v.checkEnvVarExposure(command)...)
	
	// Check for CI-specific dangerous commands
	errors = append(errors, v.checkCIDangerousCommands(command)...)
	
	return errors
}

// checkForSecrets checks for potential secrets in the command
func (v *CICDValidator) checkForSecrets(command string) []ValidationError {
	var errors []ValidationError
	
	for _, pattern := range v.secretsRegex {
		re := regexp.MustCompile(pattern)
		if re.MatchString(command) {
			errors = append(errors, ValidationError{
				Type:       ErrorSafety,
				Severity:   SeverityError,
				Message:    "Potential secret detected in command",
				Rule:       "cicd-secret-exposure",
				Suggestion: "Use secure secret storage (e.g., GitHub Secrets, GitLab Variables)",
				RiskLevel:  RiskLevelCritical,
			})
			break
		}
	}
	
	return errors
}

// checkEnvVarExposure checks for environment variable exposure
func (v *CICDValidator) checkEnvVarExposure(command string) []ValidationError {
	var errors []ValidationError
	
	// Check for echo/print of sensitive environment variables
	echoPattern := regexp.MustCompile(`(echo|print|printf|cat)\s+.*\$[A-Z_]+`)
	if echoPattern.MatchString(command) {
		// Check if it's a sensitive variable
		for _, rule := range v.envRules {
			if rule.Sensitive && strings.Contains(command, "$"+rule.Name) {
				errors = append(errors, ValidationError{
					Type:       ErrorSafety,
					Severity:   SeverityError,
					Message:    fmt.Sprintf("Exposing sensitive environment variable: %s", rule.Name),
					Rule:       "cicd-env-exposure",
					Suggestion: "Mask or redact sensitive values in CI/CD logs",
					RiskLevel:  RiskLevelHigh,
				})
			}
		}
	}
	
	// Check for env command that might expose all variables
	if strings.Contains(command, "env") && !strings.Contains(command, "env |") {
		errors = append(errors, ValidationError{
			Type:       ErrorSafety,
			Severity:   SeverityWarning,
			Message:    "Command may expose all environment variables",
			Rule:       "cicd-env-dump",
			Suggestion: "Filter output to show only necessary variables",
			RiskLevel:  RiskLevelMedium,
		})
	}
	
	return errors
}

// checkCIDangerousCommands checks for commands dangerous in CI/CD
func (v *CICDValidator) checkCIDangerousCommands(command string) []ValidationError {
	var errors []ValidationError
	
	// Check for commands that might break CI/CD
	dangerousPatterns := map[string]struct {
		pattern    string
		message    string
		suggestion string
		riskLevel  RiskLevel
	}{
		"infinite-loop": {
			pattern:    `while\s+true|for\s*\(\s*;\s*;\s*\)`,
			message:    "Infinite loop detected - this will timeout CI/CD jobs",
			suggestion: "Add proper exit conditions or timeouts",
			riskLevel:  RiskLevelHigh,
		},
		"sleep-long": {
			pattern:    `sleep\s+([0-9]{4,}|[0-9]+[hd])`,
			message:    "Long sleep detected - this wastes CI/CD resources",
			suggestion: "Reduce sleep duration or use proper wait conditions",
			riskLevel:  RiskLevelMedium,
		},
		"modify-ci-files": {
			pattern:    `(rm|mv|echo.*>)\s+\.?(github/workflows|gitlab-ci\.yml|circleci/config|Jenkinsfile)`,
			message:    "Modifying CI/CD configuration files during build",
			suggestion: "CI/CD configs should be modified via version control",
			riskLevel:  RiskLevelHigh,
		},
		"docker-privileged": {
			pattern:    `docker\s+run\s+.*--privileged`,
			message:    "Running privileged Docker containers in CI/CD is risky",
			suggestion: "Use specific capabilities instead of --privileged",
			riskLevel:  RiskLevelHigh,
		},
	}
	
	for name, check := range dangerousPatterns {
		if matched, _ := regexp.MatchString(check.pattern, command); matched {
			errors = append(errors, ValidationError{
				Type:       ErrorSafety,
				Severity:   SeverityWarning,
				Message:    check.message,
				Rule:       "cicd-" + name,
				Suggestion: check.suggestion,
				RiskLevel:  check.riskLevel,
			})
		}
	}
	
	return errors
}

// Platform-specific checks

func (v *CICDValidator) checkGitHubActions(command string) []ValidationError {
	var errors []ValidationError
	
	// Check for hardcoded GitHub tokens
	if strings.Contains(command, "GITHUB_TOKEN=") || strings.Contains(command, "GH_TOKEN=") {
		errors = append(errors, ValidationError{
			Type:       ErrorSafety,
			Severity:   SeverityError,
			Message:    "Hardcoded GitHub token detected",
			Rule:       "github-actions-token",
			Suggestion: "Use ${{ secrets.GITHUB_TOKEN }} or ${{ github.token }}",
			RiskLevel:  RiskLevelCritical,
		})
	}
	
	// Check for deprecated set-output
	if strings.Contains(command, "::set-output") {
		errors = append(errors, ValidationError{
			Type:       ErrorDeprecated,
			Severity:   SeverityWarning,
			Message:    "set-output is deprecated in GitHub Actions",
			Rule:       "github-actions-deprecated",
			Suggestion: "Use $GITHUB_OUTPUT instead: echo \"name=value\" >> $GITHUB_OUTPUT",
			RiskLevel:  RiskLevelLow,
		})
	}
	
	return errors
}

func (v *CICDValidator) checkGitLabCI(command string) []ValidationError {
	var errors []ValidationError
	
	// Check for GitLab CI variables
	if strings.Contains(command, "CI_JOB_TOKEN") && strings.Contains(command, "echo") {
		errors = append(errors, ValidationError{
			Type:       ErrorSafety,
			Severity:   SeverityError,
			Message:    "Exposing CI_JOB_TOKEN is dangerous",
			Rule:       "gitlab-ci-token",
			Suggestion: "Never echo CI_JOB_TOKEN; use it directly in API calls",
			RiskLevel:  RiskLevelHigh,
		})
	}
	
	return errors
}

func (v *CICDValidator) checkCircleCI(command string) []ValidationError {
	var errors []ValidationError
	
	// Check for CircleCI context usage
	if strings.Contains(command, "CIRCLE_TOKEN") {
		errors = append(errors, ValidationError{
			Type:       ErrorSafety,
			Severity:   SeverityError,
			Message:    "CircleCI token detected in command",
			Rule:       "circleci-token",
			Suggestion: "Use CircleCI contexts for secure credential storage",
			RiskLevel:  RiskLevelCritical,
		})
	}
	
	return errors
}

func (v *CICDValidator) checkJenkins(command string) []ValidationError {
	var errors []ValidationError
	
	// Check for Jenkins credentials
	if strings.Contains(command, "withCredentials") && strings.Contains(command, "echo") {
		errors = append(errors, ValidationError{
			Type:       ErrorSafety,
			Severity:   SeverityWarning,
			Message:    "Echoing credentials in Jenkins pipeline",
			Rule:       "jenkins-credentials",
			Suggestion: "Avoid echoing credentials; Jenkins masks them in logs",
			RiskLevel:  RiskLevelMedium,
		})
	}
	
	return errors
}

// GetPlatform returns the detected CI/CD platform
func (v *CICDValidator) GetPlatform() CIPlatform {
	return v.platform
}

// IsCI returns whether we're running in a CI environment
func (v *CICDValidator) IsCI() bool {
	return v.isCI
}

// CreateCICDSafetyRule creates a safety rule for CI/CD validation
type CICDSafetyRule struct {
	validator *CICDValidator
}

// NewCICDSafetyRule creates a new CI/CD safety rule
func NewCICDSafetyRule() *CICDSafetyRule {
	return &CICDSafetyRule{
		validator: NewCICDValidator(),
	}
}

// Check implements the SafetyRule interface
func (r *CICDSafetyRule) Check(command string, ast *AST) []ValidationError {
	// Only check if we're in a CI environment
	if !r.validator.IsCI() {
		return []ValidationError{}
	}
	
	return r.validator.ValidateCommand(command)
}

// GetName implements the SafetyRule interface
func (r *CICDSafetyRule) GetName() string {
	return "CICDSafety"
}

// GetDescription implements the SafetyRule interface
func (r *CICDSafetyRule) GetDescription() string {
	return fmt.Sprintf("CI/CD safety checks for %s", r.validator.GetPlatform())
}

// GetRiskLevel implements the SafetyRule interface
func (r *CICDSafetyRule) GetRiskLevel() RiskLevel {
	return RiskLevelHigh
}