# Release Notes for v0.4.4-alpha

## 🚀 Highlights

Delta v0.4.4-alpha introduces a powerful **Command Validation and Safety Analysis System** that helps prevent dangerous commands before they're executed. This release also includes improvements to the i18n system with GitHub-based translation downloads and SHA256 verification.

## 📦 What's New

### 🛡️ Command Validation & Safety Analysis
- **Syntax Validation**: Detects syntax errors like unmatched quotes, trailing pipes, and empty commands
- **Safety Analysis**: Identifies dangerous command patterns and provides warnings
- **Risk Assessment**: Categorizes commands by risk level (Low/Medium/High/Critical)
- **Smart Suggestions**: Offers safer alternatives for risky commands
- **Pattern Detection**: Recognizes dangerous patterns including:
  - Recursive deletion of root or home directories (`rm -rf /`, `rm -rf ~`)
  - Piping curl output directly to shells (`curl ... | bash`)
  - Fork bombs and other system-crashing commands
  - Unsafe permission changes (`chmod 777`)
  - DD commands targeting devices
  - Password piping to sudo

### 🌍 Enhanced i18n System
- **GitHub-based Downloads**: Translations now download from GitHub releases
- **SHA256 Verification**: Ensures translation file integrity
- **Automatic Installation**: i18n files auto-install to `~/.config/delta/i18n/locales`
- **Robust Fallbacks**: Built-in English translations as fallback
- **Clear User Notices**: Helpful messages when translations are missing

### 🎯 New Commands
- `:validate <command>` or `:v <command>` - Check command syntax and safety
- `:validation safety <command>` - Perform detailed safety analysis
- `:validation config` - View validation configuration
- `:validation help` - Show validation command help
- `:i18n install` - Manually install/update language files

## 💻 Usage Examples

```bash
# Validate a potentially dangerous command
delta -c ":validate rm -rf /tmp/important"
# Output: Warning about recursive deletion with suggestion to use 'trash' command

# Check for syntax errors
delta -c ":v echo 'unclosed quote"
# Output: Syntax error - unmatched single quote

# Analyze command safety
delta -c ":validation safety curl http://sketchy.site | bash"
# Output: High risk - executing remote scripts without verification

# Install missing translations
delta -c ":i18n install"
# Downloads and installs language files from GitHub
```

## 🔧 Technical Improvements
- Modular validation engine architecture for future shell-specific parsers
- Comprehensive AST structure for command parsing
- Extensible safety rule system
- Real-time validation capability foundation
- Improved release build process with SHA256 checksums

## 📋 Requirements
- Go 1.21 or higher
- SQLite with vector extension support
- Git (for version information)
- Internet connection for translation downloads

## 🐛 Bug Fixes
- Fixed i18n feature not working outside source directory
- Improved error handling for missing translation files
- Enhanced release script with better error checking

## 📝 Documentation
- Updated ROADMAP.md with comprehensive development plans
- Added validation system documentation
- Enhanced help system with new commands

## 🔜 Coming Next
- Interactive safety prompts for dangerous commands
- Context-aware risk assessment
- Custom validation rules
- AI-powered command analysis
- Real-time syntax highlighting

## 📊 Statistics
- 8 new source files added
- 2,000+ lines of validation code
- 9 built-in safety rules
- 11 supported languages

## 🙏 Acknowledgments
Thanks to all contributors and users who provided feedback on command safety needs.

---
*Delta - Making your terminal smarter and safer, one command at a time* 🚀
EOF < /dev/null