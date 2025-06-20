# Changelog

All notable changes to Delta CLI will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added - Command Validation Phase 4: Interactive Safety (2025-06-20)

#### Overview
Implemented comprehensive interactive safety features for command validation, providing smart confirmation prompts, educational content, and safer alternative suggestions for dangerous commands.

#### Features

##### 1. Interactive Safety Checker (`validation/interactive_safety.go`)
- Smart confirmation prompts for risky operations with multiple user options:
  - [y] Yes, proceed with the command
  - [n] No, cancel the command  
  - [m] Modify the command
  - [s] Mark as safe for future (proceed without prompts)
  - [?] Show more details
- Command safety history tracking with decision recording
- Trusted path detection to bypass prompts in safe directories

##### 2. Risk-Based Educational Content
- Different educational content for each risk level:
  - **Critical**: System destruction warnings with severe consequences
  - **High**: Dangerous operation alerts with security implications
  - **Medium**: Caution advisories with potential side effects
  - **Low**: No prompts (safe operations)
- Shows potential consequences of dangerous commands
- Provides safer alternatives with practical examples
- Links to documentation for additional learning

##### 3. Configuration Options
New configuration keys available via `:validation config set <key> <value>`:
- `enabled` - Enable/disable validation (default: true)
- `interactive_safety` - Enable interactive prompts (default: true)
- `educational_info` - Show educational content (default: true)
- `auto_deny_critical` - Auto-deny critical commands (default: true)
- `bypass_trusted_paths` - Skip prompts in trusted directories (default: true)

##### 4. Command Execution Integration
- Commands are automatically validated before execution
- Both interactive shell and direct command mode (`-c` flag) are covered
- Validation respects configuration settings
- Non-blocking for safe commands (low risk)

##### 5. New Commands
- `:validation stats` - View safety decision statistics
- `:validation history` - View recent safety decisions  
- `:validation config set/get/reset` - Manage configuration

##### 6. Trusted Path Detection
- Automatically bypasses prompts in trusted directories:
  - `~/projects`
  - `~/dev`
  - `~/code`
  - `~/src`
- Configurable to add more trusted paths

#### Example Interactive Session
When a user tries to run `rm -rf /`, they see:
```
===============================================
⚠️ CRITICAL RISK: System Destruction Warning
===============================================

This command could permanently damage your system or delete critical data.

⚠️  Potential Consequences:
  • Complete system failure requiring reinstallation
  • Permanent loss of all data
  • Corruption of system files
  • Loss of user accounts and configurations

💡 Safer Alternatives:
  
  Use specific paths instead of /
  Example: rm -rf /tmp/specific-folder
  Why safer: Limits damage to intended targets only
  
  Use trash command
  Example: trash /path/to/file
  Why safer: Allows recovery if mistakes are made

📚 Learn more: https://wiki.deltacli.com/safety/critical-commands
------------------------------------------------------------

🔴 Command Risk Summary
Command: rm -rf /

Risk Factors:
  • CRITICAL: This command will recursively delete your entire system!

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
⚠️  Do you want to proceed with this CRITICAL risk command?
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Options:
  [y] Yes, proceed with the command
  [n] No, cancel the command
  [m] Modify the command
  [s] Mark as safe for future (proceed without prompts)
  [?] Show more details

Your choice [y/n/m/s/?]: 
```

#### Technical Implementation
- **Files Added:**
  - `validation/interactive_safety.go` - Core interactive safety implementation
  - `command_validator.go` - Command validation integration
  - `test_interactive_safety.sh` - Test script for safety features
  
- **Files Modified:**
  - `validation/engine.go` - Added interactive safety checker integration
  - `cli.go` - Integrated validation into command execution flow
  - `validation_commands.go` - Added stats, history, and config commands
  - `config_manager.go` - Added custom config storage for validation settings
  - `Makefile` - Added command_validator.go to build sources
  - `CLAUDE.md` - Updated documentation
  - `TASKS.md` - Marked Phase 4 as completed

This completes Phase 4 of the Command Validation system, providing a user-friendly and educational safety layer that helps prevent accidental damage while teaching users about command risks.

## [0.4.4-alpha] - 2025-06-20

### Added
- Command Validation System - Phase 1-3 implementation
  - Syntax validation engine for multiple shells
  - 17 built-in safety rules with pattern matching
  - Context-aware risk assessment
  - Permission requirement detection
  - Environmental context analysis
  - Risk mitigation suggestions

### Fixed
- Direct command mode (`-c` flag) now includes training pipeline for AI/ML features
- Commands execute immediately with async training in background

## [0.4.3-alpha] - 2025-06-20

### Fixed
- i18n feature now works outside source directory by downloading from GitHub releases
- Added SHA256 checksum verification for secure i18n file downloads

## [0.4.2-alpha] - 2025-06-17

### Added
- **Auto-Update System Phase 4**: Advanced Features
  - Update scheduling with cron-like functionality
  - Comprehensive update history tracking
  - Post-update validation with automatic rollback
  - Interactive update UI with user choices
- **AI System Improvements**:
  - AI enabled/disabled state now persists between sessions
  - Added `:ai enable` and `:ai disable` aliases
- **i18n System Improvements**:
  - Automatic installation of language files
  - Built-in English translations as fallback
  - Robust path detection for i18n files
  - Clear startup notice if translations are missing

### Fixed
- AI configuration persistence issues
- i18n file installation and detection

## [0.4.1-alpha] - 2025-06-08

### Added
- Automatic version injection at build time
- Git integration for version and commit detection
- Smart development vs release build detection

## [0.4.0-alpha] - 2025-06-08

### Added
- **Auto-Update System Phase 3**: Download, Installation & Rollback
  - Secure update downloads with SHA256 verification
  - Automatic backup before updates
  - Rollback capability on failure
  - Interactive update prompts

## [0.3.1-alpha] - 2025-06-07

### Added
- **Auto-Update System Phase 2**: GitHub Integration
  - Update detection from GitHub releases
  - Version comparison and notifications

## [0.3.0-alpha] - 2025-06-06

### Added
- **Auto-Update System Phase 1**: Foundation Infrastructure
  - Configuration management for updates
  - Version management system
  - CLI commands for update control

## [0.2.0-alpha] - 2025-06-06

### Added
- Internationalization (i18n) support for 11 languages
- ART-2 machine learning system for pattern recognition
- Memory collection system for command history
- Jump command for directory navigation
- Inference system for learning capabilities

## [0.1.0-alpha] - 2025-05-15

### Added
- Initial release
- AI-powered command suggestions with Ollama integration
- Encrypted command history
- Shell compatibility (bash, zsh, fish)
- Internal command system with `:` prefix
- Tab completion support