# Command Validation and Safety Analysis Plan

## Overview

This document outlines the plan for implementing comprehensive command validation tools in Delta CLI, including syntax checking, safety analysis, and risk assessment for shell commands before execution.

## Goals

1. **Prevent Destructive Operations**: Catch potentially harmful commands before execution
2. **Syntax Validation**: Verify command syntax across different shells
3. **Permission Analysis**: Check if commands require elevated privileges
4. **Risk Assessment**: Categorize commands by risk level
5. **User Education**: Provide explanations for why commands might be dangerous

## Architecture

### Core Components

```
┌─────────────────────────────────────────────────────────────┐
│                    Command Input                             │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                 Syntax Validator                             │
│  - Shell-specific parsing (bash, zsh, fish, etc.)          │
│  - Quote/escape validation                                   │
│  - Pipeline validation                                       │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                 Safety Analyzer                              │
│  - Pattern matching for dangerous commands                   │
│  - File system impact analysis                               │
│  - Network operation detection                               │
│  - System modification detection                             │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                 Risk Categorizer                             │
│  - Low Risk (read-only operations)                          │
│  - Medium Risk (file modifications)                          │
│  - High Risk (system changes, deletions)                     │
│  - Critical Risk (recursive deletions, format operations)   │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│              Interactive Safety Check                        │
│  - Confirmation prompts for risky operations                 │
│  - Detailed explanations of risks                            │
│  - Suggestions for safer alternatives                        │
└─────────────────────────────────────────────────────────────┘
```

## Implementation Plan

### Phase 1: Foundation (v0.5.0-alpha)

#### 1.1 Syntax Validation Engine
```go
// validation_engine.go
type ValidationEngine struct {
    shellType    ShellType
    validators   map[ShellType]Validator
    config       ValidationConfig
}

type ValidationResult struct {
    IsValid      bool
    Errors       []SyntaxError
    Warnings     []ValidationWarning
    Suggestions  []string
}

type SyntaxError struct {
    Position    int
    Message     string
    ErrorType   string
    Suggestion  string
}
```

#### 1.2 Shell-Specific Parsers
- **Bash Parser**: Handle bash-specific syntax
- **Zsh Parser**: Support zsh extensions
- **Fish Parser**: Validate fish shell syntax
- **POSIX Parser**: Ensure POSIX compliance

#### 1.3 Basic Validation Rules
- Unmatched quotes detection
- Invalid pipe syntax
- Malformed redirections
- Command existence checking

### Phase 2: Safety Analysis (v0.5.1-alpha)

#### 2.1 Dangerous Pattern Detection
```go
// safety_patterns.go
var DangerousPatterns = []Pattern{
    {
        Name:        "RecursiveDelete",
        Regex:       `rm\s+(-rf?|-fr?)\s+`,
        Risk:        RiskCritical,
        Description: "Recursive file deletion",
        Mitigation:  "Consider using trash/recycle bin",
    },
    {
        Name:        "ForceOverwrite",
        Regex:       `>\s*[^>]`,
        Risk:        RiskMedium,
        Description: "File overwrite operation",
        Mitigation:  "Use >> for append or check file existence",
    },
    {
        Name:        "SystemModification",
        Regex:       `(sudo|doas)\s+`,
        Risk:        RiskHigh,
        Description: "Elevated privilege operation",
        Mitigation:  "Ensure you understand the command fully",
    },
}
```

#### 2.2 File System Impact Analysis
- Track which files/directories will be affected
- Detect operations on system directories
- Identify potential data loss scenarios

#### 2.3 Network Operation Detection
- Identify commands that make network requests
- Detect potential data exfiltration
- Flag insecure protocols (HTTP, FTP, Telnet)

### Phase 3: Risk Assessment (v0.5.2-alpha)

#### 3.1 Risk Scoring System
```go
type RiskAssessment struct {
    OverallRisk   RiskLevel
    Categories    []RiskCategory
    Score         int
    Explanation   string
    Mitigations   []string
}

type RiskCategory struct {
    Name          string
    Level         RiskLevel
    Contributing  []string
}

const (
    RiskLow      RiskLevel = iota
    RiskMedium
    RiskHigh
    RiskCritical
)
```

#### 3.2 Context-Aware Analysis
- Consider current directory
- Check user permissions
- Analyze command history for patterns
- Environmental variable expansion risks

### Phase 4: Interactive Safety (v0.5.3-alpha)

#### 4.1 Smart Confirmation Prompts
```go
type SafetyPrompt struct {
    Command       string
    Risk          RiskAssessment
    RequiresAuth  bool
    Alternatives  []SaferAlternative
}

type SaferAlternative struct {
    Command       string
    Description   string
    RiskReduction string
}
```

#### 4.2 Educational Explanations
- Explain why a command is risky
- Show potential consequences
- Provide learning resources
- Suggest best practices

### Phase 5: Advanced Features (v0.6.0-alpha)

#### 5.1 AI-Powered Analysis
- Use AI to understand command intent
- Detect obfuscated dangerous commands
- Provide natural language explanations
- Learn from user patterns

#### 5.2 Custom Rule Engine
```go
// custom_rules.go
type CustomRule struct {
    Name         string
    Description  string
    Condition    string // DSL for rule definition
    Action       RuleAction
    Severity     RiskLevel
}

type RuleEngine struct {
    rules        []CustomRule
    userRules    []CustomRule
    executor     RuleExecutor
}
```

#### 5.3 Integration with Version Control
- Warn about operations in git repositories
- Detect uncommitted changes
- Suggest git-safe alternatives

## Command Examples

### Validation Commands
```bash
# Check command syntax
:validate "rm -rf /important/directory"

# Analyze command safety
:safety "curl http://example.com | bash"

# Get risk assessment
:risk "sudo dd if=/dev/zero of=/dev/sda"

# Check multiple commands
:validate-pipeline "cat file | grep pattern | awk '{print $1}'"
```

### Configuration
```bash
# Set validation strictness
:config validation.level strict

# Enable/disable specific checks
:config validation.checks.sudo enabled
:config validation.checks.rm-rf disabled

# Set risk tolerance
:config safety.risk-tolerance medium
```

## Safety Rules Database

### Critical Risk Patterns
1. **Recursive Deletions**: `rm -rf`, `rm -fr`
2. **Disk Operations**: `dd`, `format`, `fdisk`
3. **System Files**: Operations on `/etc`, `/sys`, `/proc`
4. **Permission Changes**: `chmod -R 777`, `chown -R`

### High Risk Patterns
1. **Service Control**: `systemctl stop`, `service restart`
2. **Package Management**: `apt remove`, `yum erase`
3. **Network Configuration**: `ifconfig down`, `iptables -F`
4. **User Management**: `userdel`, `passwd`

### Medium Risk Patterns
1. **File Overwrites**: `>`, `cp -f`
2. **Bulk Operations**: `find -exec`, `xargs`
3. **Archive Extraction**: `tar -xf` in current directory
4. **Remote Operations**: `ssh`, `scp`, `rsync`

### Low Risk Patterns
1. **Read Operations**: `ls`, `cat`, `grep`
2. **Information Gathering**: `ps`, `df`, `du`
3. **Navigation**: `cd`, `pwd`
4. **Help Commands**: `man`, `help`, `info`

## Integration Points

### 1. Pre-Execution Hook
```go
func (cli *CLI) preExecuteHook(cmd string) error {
    result := cli.validator.Validate(cmd)
    if !result.IsValid {
        return cli.handleValidationError(result)
    }
    
    risk := cli.safetyAnalyzer.Assess(cmd)
    if risk.OverallRisk >= RiskHigh {
        return cli.promptSafetyConfirmation(risk)
    }
    
    return nil
}
```

### 2. Real-time Validation
- Syntax highlighting in prompt
- Inline error indicators
- Auto-suggestions for corrections

### 3. History Analysis
- Learn from past commands
- Detect unusual patterns
- Warn about deviations

## Configuration Schema

```yaml
validation:
  enabled: true
  level: strict  # strict, normal, permissive
  real_time: true
  syntax_checks:
    quotes: true
    pipes: true
    redirects: true
    expansions: true
  
safety:
  enabled: true
  risk_tolerance: medium  # low, medium, high
  require_confirmation:
    - critical
    - high
  patterns:
    custom_rules: ~/.config/delta/safety_rules.yaml
    
education:
  show_explanations: true
  suggest_alternatives: true
  learning_mode: true
```

## Testing Strategy

### Unit Tests
- Test each validation rule
- Verify pattern matching
- Check risk calculations

### Integration Tests
- Full command validation flow
- Multi-shell compatibility
- Configuration persistence

### Safety Tests
- Dangerous command detection
- False positive minimization
- Performance benchmarks

## Performance Considerations

1. **Caching**: Cache validation results for repeated commands
2. **Async Validation**: Perform validation in background for real-time feedback
3. **Lazy Loading**: Load validation rules on demand
4. **Optimization**: Use efficient regex engines and pattern matching

## Future Enhancements

1. **Cloud-based Rule Updates**: Fetch latest safety patterns
2. **Community Rules**: Share custom rules with other users
3. **Machine Learning**: Learn from global command patterns
4. **Integration with CI/CD**: Validate scripts in pipelines
5. **Shell Script Analysis**: Full script safety analysis

## Success Metrics

1. **Prevention Rate**: Number of dangerous commands prevented
2. **False Positive Rate**: Legitimate commands incorrectly flagged
3. **User Education**: Commands explained and alternatives suggested
4. **Performance Impact**: Validation overhead < 50ms
5. **Adoption Rate**: Percentage of users with validation enabled

## Timeline

- **Phase 1**: 4 weeks - Foundation and basic syntax validation
- **Phase 2**: 3 weeks - Safety analysis implementation
- **Phase 3**: 3 weeks - Risk assessment system
- **Phase 4**: 4 weeks - Interactive safety features
- **Phase 5**: 6 weeks - Advanced features and AI integration

Total estimated time: 20 weeks (5 months)

## Conclusion

This comprehensive validation system will make Delta CLI one of the safest shell environments available, protecting users from accidental damage while educating them about command safety. The phased approach ensures we can deliver value incrementally while building toward a complete solution.