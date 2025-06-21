# Delta CLI v0.4.6-alpha Release Notes

## Overview

Delta CLI v0.4.6-alpha completes Phase 5 of the Command Validation System, introducing advanced security features that protect users from sophisticated threats and provide powerful customization options for validation rules.

## Major Features

### 🔍 Obfuscation Detection

Delta can now detect and analyze obfuscated commands that attempt to hide malicious intent:

- **Base64 Detection**: Identifies base64 encoded commands like `echo "cm0gLXJmIC8=" | base64 -d`
- **Hex Encoding**: Catches hex-encoded commands and Unicode escapes
- **Variable Tricks**: Detects variable substitution abuse like `a="r"; b="m"; $a$b -rf /`
- **Character Substitution**: Identifies tricks like `rm${IFS}-rf${IFS}/`
- **Deobfuscation**: Shows the actual command being executed after deobfuscation

### 📝 Custom Rule Engine with YAML DSL

Create your own validation rules using a simple YAML format:

```yaml
rules:
  - name: no-force-push-main
    description: "Prevent force push to main branch"
    pattern: "git\\s+push\\s+.*--force.*\\s+(main|master)"
    risk: high
    message: "Force pushing to main is dangerous"
    suggest: "Use --force-with-lease instead"
```

**CLI Commands**:
- `:validation rules list` - View all custom rules
- `:validation rules add` - Create new rules interactively
- `:validation rules test <cmd>` - Test commands against rules
- `:validation rules enable/disable <name>` - Toggle rules on/off

### 🔐 Git-Aware Safety Checks

Specialized protection for Git operations:

- Force push warnings on protected branches (main, master, production)
- Hard reset alerts when uncommitted changes exist
- Sensitive file detection (.env, .key, .pem files)
- Aggressive clean operation warnings
- Smart branch protection based on repository context

### 🚀 CI/CD Pipeline Integration

Automatic detection and protection for CI/CD environments:

**Supported Platforms**:
- GitHub Actions
- GitLab CI
- CircleCI
- Jenkins
- Travis CI
- Azure Pipelines

**Features**:
- Secret exposure prevention
- Platform-specific deprecated command warnings
- Environment variable protection
- CI-specific dangerous pattern detection

## Usage Examples

### Detecting Obfuscated Commands

```bash
# Check for obfuscation
:validation obfuscation 'echo "bHMgLWxh" | base64 -d'

# Deobfuscate and analyze
:validation deobfuscate 'r${IFS}m${IFS}-rf${IFS}/'
```

### Managing Custom Rules

```bash
# List all rules
:validation rules list

# Test a command
:validation rules test 'git push --force origin main'

# Enable a specific rule
:validation rules enable no-curl-pipe-bash
```

### Configuration

New configuration options:

```bash
# Enable/disable features
:validation config set custom_rules true
:validation config set obfuscation_detection true
```

## Security Improvements

1. **Multi-layer Protection**: Combines syntax checking, safety analysis, obfuscation detection, and custom rules
2. **Context-Aware**: Understands Git repositories and CI/CD environments
3. **User Education**: Provides detailed explanations and safer alternatives
4. **Customizable**: Create rules specific to your organization's needs

## Default Custom Rules

The system comes with pre-configured rules for common security concerns:

- **no-force-push-main**: Prevents force pushing to main branches
- **no-curl-pipe-bash**: Blocks `curl | bash` patterns
- **no-password-in-command**: Detects passwords in commands
- **docker-privileged**: Warns about privileged containers
- **aws-credentials-exposed**: Prevents AWS credential exposure
- **drop-database-prod**: Protects production databases

## Technical Details

- **Language**: Go
- **New Files**: 
  - `validation/obfuscation_detector.go`
  - `validation/custom_rule_engine.go`
  - `validation/git_safety.go`
  - `validation/cicd_validator.go`
- **Configuration**: `~/.config/delta/validation_rules.yaml`
- **Testing**: Comprehensive unit and integration tests

## Coming Next

- Machine learning for pattern detection
- Community rule sharing platform
- Integration with external security scanners
- Real-time rule updates

## Feedback

We welcome your feedback! Please report issues or suggestions at:
https://github.com/xorobabel/delta

---

**Full Changelog**: https://github.com/xorobabel/delta/compare/v0.4.5-alpha...v0.4.6-alpha