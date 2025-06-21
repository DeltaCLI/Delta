package validation

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// CustomRuleEngine manages user-defined validation rules
type CustomRuleEngine struct {
	rules      []CustomRule
	dslPath    string
	compiled   map[string]*regexp.Regexp
}

// CustomRule represents a user-defined validation rule
type CustomRule struct {
	Name        string    `yaml:"name"`
	Description string    `yaml:"description"`
	Pattern     string    `yaml:"pattern"`
	Risk        string    `yaml:"risk"`
	Message     string    `yaml:"message"`
	Suggest     string    `yaml:"suggest"`
	Enabled     bool      `yaml:"enabled"`
	Tags        []string  `yaml:"tags"`
}

// CustomRuleSet represents a collection of custom rules
type CustomRuleSet struct {
	Rules []CustomRule `yaml:"rules"`
}

// NewCustomRuleEngine creates a new custom rule engine
func NewCustomRuleEngine(dslPath string) *CustomRuleEngine {
	engine := &CustomRuleEngine{
		rules:    []CustomRule{},
		dslPath:  dslPath,
		compiled: make(map[string]*regexp.Regexp),
	}
	
	// Try to load rules from file
	if dslPath != "" {
		engine.LoadRules()
	}
	
	return engine
}

// LoadRules loads rules from the DSL file
func (e *CustomRuleEngine) LoadRules() error {
	// Expand home directory if needed
	if strings.HasPrefix(e.dslPath, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		e.dslPath = filepath.Join(home, e.dslPath[2:])
	}
	
	// Check if file exists
	if _, err := os.Stat(e.dslPath); os.IsNotExist(err) {
		// Create default rules file
		return e.createDefaultRulesFile()
	}
	
	// Read the file
	data, err := ioutil.ReadFile(e.dslPath)
	if err != nil {
		return fmt.Errorf("failed to read rules file: %w", err)
	}
	
	// Parse YAML
	var ruleSet CustomRuleSet
	if err := yaml.Unmarshal(data, &ruleSet); err != nil {
		return fmt.Errorf("failed to parse rules YAML: %w", err)
	}
	
	// Compile patterns and store rules
	e.rules = []CustomRule{}
	e.compiled = make(map[string]*regexp.Regexp)
	
	for _, rule := range ruleSet.Rules {
		// Default to enabled if not specified
		if rule.Name != "" && rule.Pattern != "" {
			if !rule.Enabled && rule.Risk == "" {
				rule.Enabled = true
			}
			
			// Compile the pattern
			pattern, err := regexp.Compile(rule.Pattern)
			if err != nil {
				return fmt.Errorf("invalid pattern in rule '%s': %w", rule.Name, err)
			}
			
			e.compiled[rule.Name] = pattern
			e.rules = append(e.rules, rule)
		}
	}
	
	return nil
}

// createDefaultRulesFile creates a default rules file with examples
func (e *CustomRuleEngine) createDefaultRulesFile() error {
	defaultRules := `# Delta Custom Validation Rules
# 
# Define custom validation rules using this YAML format.
# Each rule should have:
#   - name: unique identifier for the rule
#   - description: what the rule checks for
#   - pattern: regular expression to match
#   - risk: low, medium, high, or critical
#   - message: error message to show when rule matches
#   - suggest: (optional) suggestion for fixing the issue
#   - enabled: (optional) whether rule is active (default: true)
#   - tags: (optional) list of tags for categorization

rules:
  # Security Rules
  - name: no-force-push-main
    description: "Prevent force push to main branch"
    pattern: "git\\s+push\\s+.*--force.*\\s+(origin\\s+)?(main|master)"
    risk: high
    message: "Force pushing to main branch is dangerous and can cause data loss"
    suggest: "Create a feature branch instead or use --force-with-lease"
    tags: [git, security]
    
  - name: no-curl-pipe-bash
    description: "Prevent curl | bash pattern"
    pattern: "curl\\s+.*\\|\\s*(sudo\\s+)?bash"
    risk: critical
    message: "Piping curl output directly to bash is extremely dangerous"
    suggest: "Download the script first, review it, then execute"
    tags: [security, download]
    
  - name: no-password-in-command
    description: "Prevent passwords in command line"
    pattern: "--password[= ]|PASS(WORD)?=|-p\\s+\\S+"
    risk: critical
    message: "Never include passwords directly in commands"
    suggest: "Use environment variables or secure credential storage"
    tags: [security, credentials]
    
  # Development Rules
  - name: no-npm-force
    description: "Warn about npm --force flag"
    pattern: "npm\\s+.*--force"
    risk: medium
    message: "Using --force with npm can lead to broken dependencies"
    suggest: "Try to resolve conflicts without --force first"
    tags: [npm, development]
    
  - name: docker-privileged
    description: "Warn about privileged Docker containers"
    pattern: "docker\\s+run\\s+.*--privileged"
    risk: high
    message: "Running privileged containers bypasses Docker security"
    suggest: "Use specific capabilities instead of --privileged"
    tags: [docker, security]
    
  # System Rules
  - name: recursive-chmod-777
    description: "Prevent chmod 777 on directories"
    pattern: "chmod\\s+.*\\b777\\b.*-R|chmod\\s+.*-R.*\\b777\\b"
    risk: high
    message: "Setting 777 permissions recursively is a security risk"
    suggest: "Use more restrictive permissions like 755 or 644"
    tags: [permissions, security]
    
  # AWS Rules
  - name: aws-credentials-exposed
    description: "Prevent AWS credential exposure"
    pattern: "AWS_(SECRET_)?ACCESS_KEY|aws_access_key_id|aws_secret_access_key"
    risk: critical
    message: "Command may expose AWS credentials"
    suggest: "Use AWS credential file or IAM roles"
    tags: [aws, security, credentials]
    
  # Database Rules  
  - name: drop-database-prod
    description: "Prevent dropping production databases"
    pattern: "DROP\\s+DATABASE.*(prod|production)"
    risk: critical
    message: "Attempting to drop what appears to be a production database"
    suggest: "Double-check the database name and consider a backup first"
    tags: [database, destructive]
`

	// Create directory if it doesn't exist
	dir := filepath.Dir(e.dslPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	
	// Write the default rules
	if err := ioutil.WriteFile(e.dslPath, []byte(defaultRules), 0644); err != nil {
		return fmt.Errorf("failed to write default rules: %w", err)
	}
	
	// Now load the rules we just created
	return e.LoadRules()
}

// ValidateCommand checks a command against all enabled custom rules
func (e *CustomRuleEngine) ValidateCommand(command string) []ValidationError {
	var errors []ValidationError
	
	for _, rule := range e.rules {
		if !rule.Enabled {
			continue
		}
		
		pattern, ok := e.compiled[rule.Name]
		if !ok {
			continue
		}
		
		// Check if pattern matches
		if pattern.MatchString(command) {
			errors = append(errors, ValidationError{
				Type:       ErrorCustom,
				Severity:   SeverityError,
				Message:    rule.Message,
				Rule:       rule.Name,
				Suggestion: rule.Suggest,
				RiskLevel:  parseRiskLevel(rule.Risk),
			})
		}
	}
	
	return errors
}

// GetRules returns all loaded rules
func (e *CustomRuleEngine) GetRules() []CustomRule {
	return e.rules
}

// GetRule returns a specific rule by name
func (e *CustomRuleEngine) GetRule(name string) (*CustomRule, bool) {
	for _, rule := range e.rules {
		if rule.Name == name {
			return &rule, true
		}
	}
	return nil, false
}

// AddRule adds a new rule to the engine
func (e *CustomRuleEngine) AddRule(rule CustomRule) error {
	// Validate rule
	if rule.Name == "" {
		return fmt.Errorf("rule name is required")
	}
	if rule.Pattern == "" {
		return fmt.Errorf("rule pattern is required")
	}
	
	// Check for duplicate
	if _, exists := e.GetRule(rule.Name); exists {
		return fmt.Errorf("rule with name '%s' already exists", rule.Name)
	}
	
	// Compile pattern
	pattern, err := regexp.Compile(rule.Pattern)
	if err != nil {
		return fmt.Errorf("invalid pattern: %w", err)
	}
	
	// Add to engine
	e.compiled[rule.Name] = pattern
	e.rules = append(e.rules, rule)
	
	// Save to file
	return e.SaveRules()
}

// UpdateRule updates an existing rule
func (e *CustomRuleEngine) UpdateRule(name string, rule CustomRule) error {
	// Find the rule
	index := -1
	for i, r := range e.rules {
		if r.Name == name {
			index = i
			break
		}
	}
	
	if index == -1 {
		return fmt.Errorf("rule '%s' not found", name)
	}
	
	// Compile new pattern if changed
	if rule.Pattern != "" && rule.Pattern != e.rules[index].Pattern {
		pattern, err := regexp.Compile(rule.Pattern)
		if err != nil {
			return fmt.Errorf("invalid pattern: %w", err)
		}
		e.compiled[name] = pattern
	}
	
	// Update rule
	rule.Name = name // Preserve original name
	e.rules[index] = rule
	
	// Save to file
	return e.SaveRules()
}

// DeleteRule removes a rule
func (e *CustomRuleEngine) DeleteRule(name string) error {
	// Find and remove the rule
	found := false
	newRules := []CustomRule{}
	
	for _, rule := range e.rules {
		if rule.Name != name {
			newRules = append(newRules, rule)
		} else {
			found = true
			delete(e.compiled, name)
		}
	}
	
	if !found {
		return fmt.Errorf("rule '%s' not found", name)
	}
	
	e.rules = newRules
	return e.SaveRules()
}

// EnableRule enables a rule
func (e *CustomRuleEngine) EnableRule(name string) error {
	for i, rule := range e.rules {
		if rule.Name == name {
			e.rules[i].Enabled = true
			return e.SaveRules()
		}
	}
	return fmt.Errorf("rule '%s' not found", name)
}

// DisableRule disables a rule
func (e *CustomRuleEngine) DisableRule(name string) error {
	for i, rule := range e.rules {
		if rule.Name == name {
			e.rules[i].Enabled = false
			return e.SaveRules()
		}
	}
	return fmt.Errorf("rule '%s' not found", name)
}

// SaveRules saves the current rules to the DSL file
func (e *CustomRuleEngine) SaveRules() error {
	ruleSet := CustomRuleSet{
		Rules: e.rules,
	}
	
	data, err := yaml.Marshal(&ruleSet)
	if err != nil {
		return fmt.Errorf("failed to marshal rules: %w", err)
	}
	
	// Add header comment
	header := `# Delta Custom Validation Rules
# 
# Define custom validation rules using this YAML format.
# Each rule should have:
#   - name: unique identifier for the rule
#   - description: what the rule checks for
#   - pattern: regular expression to match
#   - risk: low, medium, high, or critical
#   - message: error message to show when rule matches
#   - suggest: (optional) suggestion for fixing the issue
#   - enabled: (optional) whether rule is active (default: true)
#   - tags: (optional) list of tags for categorization

`
	
	fullData := append([]byte(header), data...)
	
	return ioutil.WriteFile(e.dslPath, fullData, 0644)
}

// TestCommand tests a command against custom rules without enforcing
func (e *CustomRuleEngine) TestCommand(command string) []CustomRule {
	var matchedRules []CustomRule
	
	for _, rule := range e.rules {
		pattern, ok := e.compiled[rule.Name]
		if !ok {
			continue
		}
		
		if pattern.MatchString(command) {
			matchedRules = append(matchedRules, rule)
		}
	}
	
	return matchedRules
}

// parseRiskLevel converts string risk level to RiskLevel type
func parseRiskLevel(risk string) RiskLevel {
	switch strings.ToLower(risk) {
	case "low":
		return RiskLevelLow
	case "medium":
		return RiskLevelMedium
	case "high":
		return RiskLevelHigh
	case "critical":
		return RiskLevelCritical
	default:
		return RiskLevelMedium
	}
}