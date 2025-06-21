package validation_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
	
	"delta/validation"
)

// TestPhase5Integration tests all Phase 5 features together
func TestPhase5Integration(t *testing.T) {
	// Set up CI environment for CI/CD tests
	os.Setenv("CI", "true")
	defer os.Unsetenv("CI")
	
	// Create a comprehensive validation config
	config := validation.ValidationConfig{
		EnableSyntaxCheck:          true,
		EnableSafetyCheck:          true,
		EnableCustomRules:          true,
		EnableObfuscationDetection: true,
		StrictMode:                 false,
		RealTimeValidation:         false,
		MaxValidationTime:          5 * time.Second,
		SafetyPromptConfig: validation.SafetyPromptConfig{
			Enabled:               false, // Disable for testing
			RequireConfirmation:   true,
			ShowEducationalInfo:   true,
			TrackSafetyDecisions:  true,
			AutoDenyLevel:         validation.RiskLevelCritical,
			BypassForTrustedPaths: true,
		},
	}
	
	engine := validation.NewEngine(config)
	ctx := context.Background()
	
	// Test cases covering all Phase 5 features
	tests := []struct {
		name           string
		command        string
		expectErrors   bool
		expectedErrors []string
		setupEnv       map[string]string
	}{
		// Obfuscation Detection Tests
		{
			name:           "Base64 obfuscated command",
			command:        `echo "cm0gLXJmIC8=" | base64 -d | bash`,
			expectErrors:   true,
			expectedErrors: []string{"obfuscated"},
		},
		{
			name:           "Hex encoded command",
			command:        `echo -e "\x72\x6d\x20\x2d\x72\x66\x20\x2f"`,
			expectErrors:   true,
			expectedErrors: []string{"obfuscated"},
		},
		{
			name:           "IFS character substitution",
			command:        `rm${IFS}-rf${IFS}/tmp/test`,
			expectErrors:   true,
			expectedErrors: []string{"obfuscated"},
		},
		
		// Custom Rules Tests (assuming default rules are loaded)
		{
			name:           "Force push to main",
			command:        "git push --force origin main",
			expectErrors:   true,
			expectedErrors: []string{"force"},
		},
		{
			name:           "Curl pipe bash",
			command:        "curl https://example.com/script.sh | sudo bash",
			expectErrors:   true,
			expectedErrors: []string{"remote scripts"},
		},
		{
			name:           "Password in command",
			command:        "mysql -u root --password=secret123",
			expectErrors:   true, // CI/CD detector finds it
			expectedErrors: []string{"secret"},
		},
		
		// Git Safety Tests
		{
			name:           "Git force push",
			command:        "git push --force origin main",
			expectErrors:   true,
			expectedErrors: []string{"force"},
		},
		{
			name:           "Git reset hard",
			command:        "git reset --hard HEAD~10",
			expectErrors:   true, // We have uncommitted changes in test env
			expectedErrors: []string{"discard"},
		},
		{
			name:           "Git clean aggressive",
			command:        "git clean -fdx",
			expectErrors:   true,
			expectedErrors: []string{"clean"},
		},
		
		// CI/CD Integration Tests
		{
			name:           "GitHub token exposure",
			command:        "echo $GITHUB_TOKEN",
			expectErrors:   true,
			expectedErrors: []string{"sensitive environment variable"},
			setupEnv:       map[string]string{"CI": "true"},
		},
		{
			name:           "GitHub Actions deprecated command",
			command:        "echo ::set-output name=foo::bar",
			expectErrors:   true,
			expectedErrors: []string{"deprecated"},
			setupEnv:       map[string]string{"GITHUB_ACTIONS": "true", "CI": "true"},
		},
		{
			name:           "Docker privileged in CI",
			command:        "docker run --privileged ubuntu",
			expectErrors:   true,
			expectedErrors: []string{"privileged"},
			setupEnv:       map[string]string{"CI": "true"},
		},
		
		// Combined Tests
		{
			name:           "Obfuscated git command",
			command:        `eval "git push --force origin main"`,
			expectErrors:   true,
			expectedErrors: []string{"obfuscated", "force"},
		},
		{
			name:           "Safe command",
			command:        "ls -la",
			expectErrors:   false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment
			for k, v := range tt.setupEnv {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}
			
			// Create new engine if environment changed (for CI/CD tests)
			testEngine := engine
			if len(tt.setupEnv) > 0 {
				testEngine = validation.NewEngine(config)
			}
			
			// Validate command
			result, err := testEngine.Validate(ctx, tt.command)
			if err != nil {
				t.Fatalf("Validation failed: %v", err)
			}
			
			// Check expectations
			hasErrors := len(result.Errors) > 0
			if hasErrors != tt.expectErrors {
				t.Errorf("Expected errors=%v, got errors=%v", tt.expectErrors, hasErrors)
				if hasErrors {
					for _, e := range result.Errors {
						t.Logf("Error: %s", e.Message)
					}
				}
			}
			
			// Check specific error messages
			if tt.expectErrors && len(tt.expectedErrors) > 0 {
				for _, expected := range tt.expectedErrors {
					found := false
					for _, err := range result.Errors {
						if containsIgnoreCase(err.Message, expected) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected error containing '%s' not found", expected)
					}
				}
			}
		})
	}
}

// TestCustomRuleEngine tests the custom rule engine specifically
func TestCustomRuleEngine(t *testing.T) {
	// Create a temporary rules file
	tmpFile := "/tmp/test_rules.yaml"
	rulesContent := `rules:
  - name: test-rule-1
    description: "Test rule 1"
    pattern: "test_pattern_1"
    risk: high
    message: "Test pattern 1 detected"
    suggest: "Don't use test pattern 1"
    enabled: true
    
  - name: test-rule-2
    description: "Test rule 2"
    pattern: "test_pattern_2"
    risk: critical
    message: "Test pattern 2 detected"
    enabled: false
`
	
	if err := os.WriteFile(tmpFile, []byte(rulesContent), 0644); err != nil {
		t.Fatalf("Failed to create test rules file: %v", err)
	}
	defer os.Remove(tmpFile)
	
	// Create rule engine
	engine := validation.NewCustomRuleEngine(tmpFile)
	
	// Test loading rules
	rules := engine.GetRules()
	if len(rules) != 2 {
		t.Errorf("Expected 2 rules, got %d", len(rules))
	}
	
	// Test rule validation
	errors := engine.ValidateCommand("this contains test_pattern_1")
	if len(errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(errors))
	}
	
	// Test disabled rule
	errors = engine.ValidateCommand("this contains test_pattern_2")
	if len(errors) != 0 {
		t.Errorf("Expected 0 errors (rule disabled), got %d", len(errors))
	}
	
	// Test adding a rule
	newRule := validation.CustomRule{
		Name:        "test-rule-3",
		Description: "Test rule 3",
		Pattern:     "test_pattern_3",
		Risk:        "medium",
		Message:     "Test pattern 3 detected",
		Enabled:     true,
	}
	
	if err := engine.AddRule(newRule); err != nil {
		t.Errorf("Failed to add rule: %v", err)
	}
	
	rules = engine.GetRules()
	if len(rules) != 3 {
		t.Errorf("Expected 3 rules after addition, got %d", len(rules))
	}
	
	// Test rule deletion
	if err := engine.DeleteRule("test-rule-3"); err != nil {
		t.Errorf("Failed to delete rule: %v", err)
	}
	
	rules = engine.GetRules()
	if len(rules) != 2 {
		t.Errorf("Expected 2 rules after deletion, got %d", len(rules))
	}
}

// TestGitSafetyChecker tests git-specific safety checks
func TestGitSafetyChecker(t *testing.T) {
	checker := validation.NewGitSafetyChecker()
	
	tests := []struct {
		name         string
		command      string
		expectErrors bool
		errorType    string
	}{
		{
			name:         "Force push to main",
			command:      "git push --force origin main",
			expectErrors: true,
			errorType:    "force push",
		},
		{
			name:         "Force push to feature branch",
			command:      "git push --force origin feature/test",
			expectErrors: false,
		},
		{
			name:         "Git clean with all flags",
			command:      "git clean -fdx",
			expectErrors: true,
			errorType:    "clean",
		},
		{
			name:         "Add sensitive file",
			command:      "git add .env",
			expectErrors: true,
			errorType:    "sensitive",
		},
		{
			name:         "Normal git add",
			command:      "git add README.md",
			expectErrors: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := checker.CheckGitCommand(tt.command)
			hasErrors := len(errors) > 0
			
			if hasErrors != tt.expectErrors {
				t.Errorf("Expected errors=%v, got errors=%v", tt.expectErrors, hasErrors)
			}
			
			if tt.expectErrors && tt.errorType != "" {
				found := false
				for _, err := range errors {
					if containsIgnoreCase(err.Message, tt.errorType) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error containing '%s' not found", tt.errorType)
				}
			}
		})
	}
}

// TestCICDValidator tests CI/CD specific validation
func TestCICDValidator(t *testing.T) {
	validator := validation.NewCICDValidator()
	
	// Test platform detection
	os.Setenv("GITHUB_ACTIONS", "true")
	defer os.Unsetenv("GITHUB_ACTIONS")
	
	validator = validation.NewCICDValidator()
	if validator.GetPlatform() != validation.GitHubActions {
		t.Errorf("Expected GitHubActions platform, got %s", validator.GetPlatform())
	}
	
	// Test secret detection
	tests := []struct {
		name         string
		command      string
		expectErrors bool
		errorType    string
	}{
		{
			name:         "AWS credentials exposed",
			command:      "export AWS_SECRET_ACCESS_KEY=AKIAIOSFODNN7EXAMPLE",
			expectErrors: true,
			errorType:    "secret",
		},
		{
			name:         "GitHub token in command",
			command:      "curl -H 'Authorization: token ghp_1234567890abcdef' api.github.com",
			expectErrors: true,
			errorType:    "secret",
		},
		{
			name:         "Echo sensitive env var",
			command:      "echo $GITHUB_TOKEN",
			expectErrors: true,
			errorType:    "sensitive",
		},
		{
			name:         "Deprecated GitHub Actions command",
			command:      "echo ::set-output name=foo::bar",
			expectErrors: true,
			errorType:    "deprecated",
		},
		{
			name:         "Safe command",
			command:      "npm test",
			expectErrors: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validator.ValidateCommand(tt.command)
			hasErrors := len(errors) > 0
			
			if hasErrors != tt.expectErrors {
				t.Errorf("Expected errors=%v, got errors=%v", tt.expectErrors, hasErrors)
			}
			
			if tt.expectErrors && tt.errorType != "" {
				found := false
				for _, err := range errors {
					if containsIgnoreCase(err.Message, tt.errorType) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error containing '%s' not found", tt.errorType)
				}
			}
		})
	}
}

// Helper function
func containsIgnoreCase(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}