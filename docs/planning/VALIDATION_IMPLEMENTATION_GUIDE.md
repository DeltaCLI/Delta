# Command Validation Implementation Guide

## Phase 1: Foundation Implementation

### File Structure
```
validation/
├── engine.go           # Core validation engine
├── syntax/
│   ├── parser.go       # Base parser interface
│   ├── bash.go         # Bash-specific parser
│   ├── zsh.go          # Zsh-specific parser
│   ├── fish.go         # Fish-specific parser
│   └── posix.go        # POSIX-compliant parser
├── rules/
│   ├── rules.go        # Rule definitions
│   ├── patterns.go     # Regex patterns
│   └── builtin.go      # Built-in rule sets
├── safety/
│   ├── analyzer.go     # Safety analysis engine
│   ├── risks.go        # Risk categorization
│   └── patterns.go     # Dangerous pattern definitions
└── commands.go         # CLI command handlers
```

### Core Interfaces

```go
// validation/engine.go
package validation

import (
    "context"
    "fmt"
)

// Validator is the main interface for command validation
type Validator interface {
    Validate(ctx context.Context, command string) (*ValidationResult, error)
    ValidateWithShell(ctx context.Context, command string, shell ShellType) (*ValidationResult, error)
}

// ValidationResult contains the results of validation
type ValidationResult struct {
    Valid       bool
    Command     string
    Shell       ShellType
    Errors      []ValidationError
    Warnings    []ValidationWarning
    Suggestions []Suggestion
    Metadata    map[string]interface{}
}

// ValidationError represents a syntax or safety error
type ValidationError struct {
    Type        ErrorType
    Severity    Severity
    Position    Position
    Message     string
    Rule        string
    Suggestion  string
}

// Position in the command string
type Position struct {
    Line   int
    Column int
    Offset int
    Length int
}

// ShellType represents different shell types
type ShellType string

const (
    ShellBash   ShellType = "bash"
    ShellZsh    ShellType = "zsh"
    ShellFish   ShellType = "fish"
    ShellPOSIX  ShellType = "posix"
    ShellAuto   ShellType = "auto"
)

// ErrorType categorizes validation errors
type ErrorType string

const (
    ErrorSyntax      ErrorType = "syntax"
    ErrorSafety      ErrorType = "safety"
    ErrorPermission  ErrorType = "permission"
    ErrorDeprecated  ErrorType = "deprecated"
    ErrorCustom      ErrorType = "custom"
)
```

### Syntax Parser Implementation

```go
// validation/syntax/parser.go
package syntax

import (
    "strings"
    "unicode"
)

// Parser interface for shell-specific parsing
type Parser interface {
    Parse(command string) (*AST, error)
    Validate(ast *AST) []ValidationError
    GetShellType() ShellType
}

// AST represents the abstract syntax tree of a command
type AST struct {
    Root     Node
    Metadata map[string]interface{}
}

// Node represents a node in the AST
type Node interface {
    Type() NodeType
    Position() Position
    Children() []Node
    Validate() []ValidationError
}

// NodeType categorizes AST nodes
type NodeType string

const (
    NodeCommand     NodeType = "command"
    NodePipeline    NodeType = "pipeline"
    NodeRedirect    NodeType = "redirect"
    NodeSubshell    NodeType = "subshell"
    NodeVariable    NodeType = "variable"
    NodeString      NodeType = "string"
    NodeGlob        NodeType = "glob"
)

// BaseParser provides common parsing functionality
type BaseParser struct {
    input    string
    position int
    tokens   []Token
}

// Token represents a lexical token
type Token struct {
    Type     TokenType
    Value    string
    Position Position
}

// TokenType categorizes tokens
type TokenType string

const (
    TokenWord       TokenType = "word"
    TokenString     TokenType = "string"
    TokenPipe       TokenType = "pipe"
    TokenRedirect   TokenType = "redirect"
    TokenSemicolon  TokenType = "semicolon"
    TokenAmpersand  TokenType = "ampersand"
    TokenVariable   TokenType = "variable"
    TokenGlob       TokenType = "glob"
)
```

### Bash Parser Example

```go
// validation/syntax/bash.go
package syntax

import (
    "fmt"
    "strings"
)

// BashParser implements Parser for Bash syntax
type BashParser struct {
    BaseParser
    strictMode bool
}

// NewBashParser creates a new Bash parser
func NewBashParser(strictMode bool) *BashParser {
    return &BashParser{
        strictMode: strictMode,
    }
}

// Parse parses a Bash command into an AST
func (p *BashParser) Parse(command string) (*AST, error) {
    p.input = command
    p.position = 0
    
    // Tokenize
    tokens, err := p.tokenize()
    if err != nil {
        return nil, fmt.Errorf("tokenization failed: %w", err)
    }
    p.tokens = tokens
    
    // Parse tokens into AST
    root, err := p.parseCommand()
    if err != nil {
        return nil, fmt.Errorf("parsing failed: %w", err)
    }
    
    return &AST{
        Root: root,
        Metadata: map[string]interface{}{
            "shell": "bash",
            "strict": p.strictMode,
        },
    }, nil
}

// tokenize breaks the input into tokens
func (p *BashParser) tokenize() ([]Token, error) {
    var tokens []Token
    
    for p.position < len(p.input) {
        p.skipWhitespace()
        if p.position >= len(p.input) {
            break
        }
        
        token, err := p.nextToken()
        if err != nil {
            return nil, err
        }
        tokens = append(tokens, token)
    }
    
    return tokens, nil
}

// Validate checks for Bash-specific syntax errors
func (p *BashParser) Validate(ast *AST) []ValidationError {
    var errors []ValidationError
    
    // Walk the AST and validate each node
    walkAST(ast.Root, func(node Node) {
        nodeErrors := node.Validate()
        errors = append(errors, nodeErrors...)
        
        // Bash-specific validations
        switch node.Type() {
        case NodeVariable:
            errors = append(errors, p.validateVariable(node)...)
        case NodeString:
            errors = append(errors, p.validateQuoting(node)...)
        case NodeRedirect:
            errors = append(errors, p.validateRedirection(node)...)
        }
    })
    
    return errors
}

// validateQuoting checks for quote matching
func (p *BashParser) validateQuoting(node Node) []ValidationError {
    var errors []ValidationError
    
    // Implementation for quote validation
    // Check for unmatched quotes, proper escaping, etc.
    
    return errors
}
```

### Safety Analyzer

```go
// validation/safety/analyzer.go
package safety

import (
    "context"
    "regexp"
    "strings"
)

// Analyzer performs safety analysis on commands
type Analyzer struct {
    patterns []DangerousPattern
    config   AnalyzerConfig
}

// DangerousPattern defines a pattern to check for
type DangerousPattern struct {
    Name        string
    Pattern     *regexp.Regexp
    Risk        RiskLevel
    Category    string
    Description string
    Mitigation  string
    Examples    []string
}

// RiskLevel categorizes the severity of risk
type RiskLevel int

const (
    RiskLow RiskLevel = iota
    RiskMedium
    RiskHigh
    RiskCritical
)

// AnalyzerConfig configures the safety analyzer
type AnalyzerConfig struct {
    EnabledCategories []string
    CustomPatterns    []DangerousPattern
    Strictness        StrictnessLevel
}

// Analyze performs safety analysis on a command
func (a *Analyzer) Analyze(ctx context.Context, command string) (*SafetyReport, error) {
    report := &SafetyReport{
        Command: command,
        Risks:   []Risk{},
    }
    
    // Normalize command for analysis
    normalized := a.normalizeCommand(command)
    
    // Check against all patterns
    for _, pattern := range a.patterns {
        if pattern.Pattern.MatchString(normalized) {
            risk := Risk{
                Pattern:     pattern.Name,
                Risk:        pattern.Risk,
                Description: pattern.Description,
                Mitigation:  pattern.Mitigation,
                Matches:     pattern.Pattern.FindAllStringIndex(normalized, -1),
            }
            report.Risks = append(report.Risks, risk)
        }
    }
    
    // Calculate overall risk
    report.OverallRisk = a.calculateOverallRisk(report.Risks)
    
    return report, nil
}

// Built-in dangerous patterns
var DefaultPatterns = []DangerousPattern{
    {
        Name:        "RecursiveDelete",
        Pattern:     regexp.MustCompile(`\brm\s+(-[a-zA-Z]*r[a-zA-Z]*f|--recursive.*--force)`),
        Risk:        RiskCritical,
        Category:    "filesystem",
        Description: "Recursive deletion can permanently remove entire directory trees",
        Mitigation:  "Use 'trash' command or create backup before deletion",
    },
    {
        Name:        "RootOperation",
        Pattern:     regexp.MustCompile(`\s+/\s*$|\s+/\s+`),
        Risk:        RiskCritical,
        Category:    "filesystem",
        Description: "Operation on root directory can damage system",
        Mitigation:  "Specify exact path instead of root directory",
    },
    {
        Name:        "SudoPassword",
        Pattern:     regexp.MustCompile(`echo\s+[^|]+\|\s*sudo\s+-S`),
        Risk:        RiskHigh,
        Category:    "security",
        Description: "Passing password to sudo via echo is insecure",
        Mitigation:  "Use sudo without -S flag or configure NOPASSWD",
    },
    {
        Name:        "CurlBash",
        Pattern:     regexp.MustCompile(`curl\s+[^|]+\|\s*(sudo\s+)?bash`),
        Risk:        RiskHigh,
        Category:    "security",
        Description: "Executing remote scripts without verification is dangerous",
        Mitigation:  "Download script first, review it, then execute",
    },
}
```

### CLI Integration

```go
// validation/commands.go
package validation

import (
    "fmt"
    "strings"
)

// HandleValidateCommand handles the :validate command
func HandleValidateCommand(args []string) bool {
    if len(args) == 0 {
        fmt.Println("Usage: :validate <command>")
        return true
    }
    
    command := strings.Join(args, " ")
    validator := NewValidator()
    
    result, err := validator.Validate(context.Background(), command)
    if err != nil {
        fmt.Printf("Validation error: %v\n", err)
        return true
    }
    
    // Display results
    displayValidationResult(result)
    
    return true
}

// HandleSafetyCommand handles the :safety command
func HandleSafetyCommand(args []string) bool {
    if len(args) == 0 {
        fmt.Println("Usage: :safety <command>")
        return true
    }
    
    command := strings.Join(args, " ")
    analyzer := safety.NewAnalyzer()
    
    report, err := analyzer.Analyze(context.Background(), command)
    if err != nil {
        fmt.Printf("Analysis error: %v\n", err)
        return true
    }
    
    // Display safety report
    displaySafetyReport(report)
    
    return true
}

// displayValidationResult shows validation results to the user
func displayValidationResult(result *ValidationResult) {
    if result.Valid {
        fmt.Println("✓ Command syntax is valid")
    } else {
        fmt.Println("✗ Command has validation errors:")
        for _, err := range result.Errors {
            fmt.Printf("  - %s: %s at position %d\n", 
                err.Type, err.Message, err.Position.Offset)
            if err.Suggestion != "" {
                fmt.Printf("    Suggestion: %s\n", err.Suggestion)
            }
        }
    }
    
    if len(result.Warnings) > 0 {
        fmt.Println("\n⚠ Warnings:")
        for _, warn := range result.Warnings {
            fmt.Printf("  - %s\n", warn.Message)
        }
    }
}

// displaySafetyReport shows safety analysis results
func displaySafetyReport(report *SafetyReport) {
    riskEmoji := map[RiskLevel]string{
        RiskLow:      "🟢",
        RiskMedium:   "🟡",
        RiskHigh:     "🟠",
        RiskCritical: "🔴",
    }
    
    fmt.Printf("\nSafety Analysis: %s %s Risk\n", 
        riskEmoji[report.OverallRisk], 
        report.OverallRisk.String())
    
    if len(report.Risks) == 0 {
        fmt.Println("No safety concerns detected.")
        return
    }
    
    fmt.Println("\nDetected Risks:")
    for _, risk := range report.Risks {
        fmt.Printf("\n%s %s: %s\n", 
            riskEmoji[risk.Risk], 
            risk.Pattern, 
            risk.Description)
        fmt.Printf("   Mitigation: %s\n", risk.Mitigation)
    }
}
```

### Configuration

```go
// validation_config.go
type ValidationConfig struct {
    Enabled           bool              `json:"enabled"`
    Level             StrictnessLevel   `json:"level"`
    RealTime          bool              `json:"real_time"`
    SyntaxChecks      SyntaxCheckConfig `json:"syntax_checks"`
    SafetyConfig      SafetyConfig      `json:"safety"`
    EducationConfig   EducationConfig   `json:"education"`
}

type SyntaxCheckConfig struct {
    Quotes      bool `json:"quotes"`
    Pipes       bool `json:"pipes"`
    Redirects   bool `json:"redirects"`
    Expansions  bool `json:"expansions"`
    Subshells   bool `json:"subshells"`
}

type SafetyConfig struct {
    Enabled             bool       `json:"enabled"`
    RiskTolerance       RiskLevel  `json:"risk_tolerance"`
    RequireConfirmation []RiskLevel `json:"require_confirmation"`
    CustomRulesPath     string     `json:"custom_rules_path"`
}
```

## Testing Examples

```go
// validation_test.go
func TestBashSyntaxValidation(t *testing.T) {
    tests := []struct {
        name     string
        command  string
        valid    bool
        errorMsg string
    }{
        {
            name:     "unmatched quote",
            command:  `echo "hello world`,
            valid:    false,
            errorMsg: "unmatched double quote",
        },
        {
            name:     "invalid pipe",
            command:  `ls |`,
            valid:    false,
            errorMsg: "unexpected end of command after pipe",
        },
        {
            name:    "valid command",
            command: `ls -la | grep "test" | wc -l`,
            valid:   true,
        },
    }
    
    validator := NewValidator()
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := validator.Validate(context.Background(), tt.command)
            if err != nil {
                t.Fatalf("validation error: %v", err)
            }
            
            if result.Valid != tt.valid {
                t.Errorf("expected valid=%v, got %v", tt.valid, result.Valid)
            }
        })
    }
}

func TestSafetyAnalysis(t *testing.T) {
    tests := []struct {
        name         string
        command      string
        expectedRisk RiskLevel
    }{
        {
            name:         "safe read command",
            command:      "ls -la",
            expectedRisk: RiskLow,
        },
        {
            name:         "dangerous rm -rf",
            command:      "rm -rf /important/data",
            expectedRisk: RiskCritical,
        },
        {
            name:         "curl pipe bash",
            command:      "curl https://example.com/script.sh | bash",
            expectedRisk: RiskHigh,
        },
    }
    
    analyzer := safety.NewAnalyzer()
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            report, err := analyzer.Analyze(context.Background(), tt.command)
            if err != nil {
                t.Fatalf("analysis error: %v", err)
            }
            
            if report.OverallRisk != tt.expectedRisk {
                t.Errorf("expected risk=%v, got %v", 
                    tt.expectedRisk, report.OverallRisk)
            }
        })
    }
}
```

## Next Steps

1. Implement the base validation engine
2. Create shell-specific parsers starting with Bash
3. Build the safety pattern database
4. Integrate with Delta's command execution flow
5. Add configuration management
6. Create comprehensive test suite
7. Document user-facing features

This implementation guide provides a solid foundation for building the command validation system in Delta CLI.