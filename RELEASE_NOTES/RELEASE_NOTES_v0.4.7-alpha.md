# Release Notes for v0.4.7-alpha

## 🚀 Highlights

Delta CLI v0.4.7-alpha brings major **Enterprise Update Features**, completing Phase 5 of the auto-update system with advanced channel management and comprehensive metrics & reporting capabilities. This release also introduces **man page generation** for Unix systems and fixes the startup experience.

## 📦 What's New

### Enterprise Update Features - Channel Management System

We've implemented a comprehensive channel management system that allows organizations to control how Delta CLI updates are delivered:

- **Multiple Update Channels**: Support for stable, beta, alpha, nightly, and custom channels
- **Channel Policies**: Fine-grained control over each channel with configurable:
  - Update frequency (immediate, daily, weekly, monthly)
  - Auto-install preferences
  - Downgrade permissions
  - Pre-release allowances
  - Version constraints (min/max allowed versions)
- **Enterprise Mode**: Advanced features for organizational deployments:
  - User access control with forced channel assignments
  - Scheduled channel migrations for gradual rollouts
  - Custom update URLs and verification keys
  - Regional restrictions and user/group policies
- **Channel Commands**:
  - `:update channel <name>` - Switch to a different update channel
  - `:update channels` - List available channels and their policies

### Enterprise Update Features - Metrics & Reporting System

A powerful analytics system that provides comprehensive insights into your update infrastructure:

- **Comprehensive Metrics Collection**:
  - Track all update operations (checks, downloads, installations, rollbacks)
  - Channel-specific performance metrics
  - Version adoption rates and success statistics
  - Error analysis with pattern detection
  - System resource monitoring during updates

- **Rich Reporting Capabilities**:
  - `:update metrics` - Quick summary of update system health
  - `:update metrics report` - Detailed reports with customizable time ranges
  - `:update metrics channel` - Channel-specific analytics
  - `:update metrics version` - Version adoption tracking
  - `:update metrics errors` - Error analysis and troubleshooting insights
  - `:update metrics performance` - Performance statistics and trends

- **Export Formats**:
  - JSON for detailed analysis and integration
  - CSV for spreadsheet compatibility
  - Prometheus format for monitoring system integration

- **Privacy & Configuration**:
  - Configurable data retention policies
  - Optional system information collection
  - Privacy-conscious design with anonymization options

### Man Page Generation System
- **Standard Unix Documentation**: Generate and install man pages for Delta CLI
- **`:man` Command**: New command for managing documentation
  - `:man generate` - Create man pages in troff format
  - `:man preview` - Preview before installing
  - `:man install` - Install to system (may need sudo)
  - `:man view` - View installed pages
  - `:man completions` - Generate shell completions
- **Build Integration**: `make man` and PowerShell equivalents
- **Structured Docs**: Consistent command documentation across all features

### Improved Startup Experience
- Fixed misleading "Ollama connection restored" messages on boot
- Connection status now only shows for actual state changes
- Better first-run experience with accurate status reporting

### Repository Cleanup
- Moved internal planning documents to private repository
- Cleaner public repository focused on production code

## 💻 Usage Examples

### Enterprise Channel Management
```bash
# Switch to beta channel
delta :update channel beta

# View available channels
delta :update channels

# Check current channel
delta :update status
```

### Metrics & Reporting
```bash
# View metrics summary
delta :update metrics

# Generate detailed report for last 30 days
delta :update metrics report --days 30

# Export metrics in different formats
delta :update metrics export json --output metrics.json
delta :update metrics export csv --output metrics.csv
delta :update metrics export prometheus

# View channel-specific metrics
delta :update metrics channel

# Analyze errors and get troubleshooting suggestions
delta :update metrics errors

# Configure metrics settings
delta :update metrics config enable
delta :update metrics config retention 90
```

### Generate and Install Man Pages
```bash
# Generate man pages
delta :man generate

# Preview before installing
delta :man preview ai

# Install system-wide
sudo delta :man install

# View the documentation
man delta
man delta-ai
```

### Shell Completions
```bash
# Generate bash completions
delta :man completions bash > ~/.delta-completion.bash
source ~/.delta-completion.bash
```

### Build System
```bash
# Using make
make man
sudo make install-man

# Using PowerShell
.\build.ps1 man
```

## 🔧 Technical Details

### Enterprise Features
- **Channel Management**: Thread-safe channel manager with persistent configuration
- **Metrics System**: Event-driven architecture with asynchronous collection
- **Performance**: < 1ms overhead for metrics collection
- **Storage**: Efficient data storage with automatic cleanup based on retention policies
- **Security**: Enterprise mode with access control and policy enforcement

### Documentation System
- Man pages follow standard troff format
- Compatible with `man`, `apropos`, and `whatis` commands
- Structured command documentation ensures consistency
- First-check detection prevents false connection messages

## 📋 Requirements

- For man pages: Unix-like system with man-db or similar
- For shell completions: bash (zsh/fish support coming soon)
- No changes to existing Delta CLI requirements

## 🐛 Bug Fixes

- Fixed "Ollama server connection restored" showing on every startup
- Improved connection status accuracy for AI features

## 📝 Notes

- Man page installation may require sudo privileges
- Windows users can generate man pages for use on Unix systems
- Shell completions currently support bash only
- Enterprise features are opt-in with zero impact when disabled
- Metrics data is stored locally with configurable retention

## 🔮 What's Next

In upcoming releases, we plan to focus on:
- **Enterprise Configuration & Policies**: Centralized management and compliance
- **Advanced Deployment Features**: Silent updates, custom mirrors, bandwidth management
- **Enhanced Security**: Cryptographic signing and secure update verification
- **Integration**: Support for configuration management tools

---

**Full Changelog**: https://github.com/DeltaCLI/delta/compare/v0.4.6-alpha...v0.4.7-alpha