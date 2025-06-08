# Auto-Update System Implementation Plan

## 🎯 Quick Start Guide

This document provides a practical implementation guide for Delta CLI's auto-update system. Follow this plan to implement the features outlined in the [Auto-Update Roadmap](AUTO_UPDATE_ROADMAP.md).

## 🚀 Phase-by-Phase Implementation

### Phase 1: Foundation (IMMEDIATE) 
**Timeline**: 2 weeks | **Target**: v0.3.0-alpha

#### Step 1: Create Configuration Structure
```bash
# Add to config_manager.go
# 1. Add UpdateConfig struct to SystemConfig
# 2. Add environment variable support
# 3. Add getter/setter methods
# 4. Update saveConfig/loadConfig methods
```

#### Step 2: Implement Version Manager
```bash
# Create version_manager.go
touch version_manager.go

# Implement core functions:
# - ParseVersion()
# - CompareVersions()  
# - IsNewerVersion()
# - MatchesChannel()
```

#### Step 3: Create Update Manager Skeleton
```bash
# Create update_manager.go
touch update_manager.go

# Implement basic structure:
# - UpdateManager struct
# - NewUpdateManager()
# - Initialize()
# - Configuration management
```

#### Step 4: Add CLI Commands
```bash
# Create update_commands.go
touch update_commands.go

# Implement commands:
# - :update status
# - :update config
# - :update help
```

#### Step 5: Integration
```bash
# Update cli.go to include update commands
# Update help.go to include update help
# Add update manager to initialization sequence
```

---

### Phase 2: GitHub Integration (NEXT)
**Timeline**: 1 week | **Target**: v0.3.1-alpha

#### Step 1: GitHub API Client
```bash
# Create github_client.go
touch github_client.go

# Implement:
# - GitHubClient struct
# - GetLatestRelease()
# - GetReleases()
# - Rate limiting
```

#### Step 2: Update Checker
```bash
# Create update_checker.go  
touch update_checker.go

# Implement:
# - CheckForUpdates()
# - ShouldCheck()
# - NotifyUpdateAvailable()
```

#### Step 3: Startup Integration
```bash
# Update cli.go startup sequence
# Add background update checking
# Implement user notifications
```

---

### Phase 3: Download & Install (FUTURE)
**Timeline**: 2 weeks | **Target**: v0.4.0-alpha

#### Implementation Overview
- Secure download system with progress bars
- Checksum verification and file integrity
- Safe installation with backup/rollback
- Platform-specific installation handling

---

### Phase 4: Advanced Features (FUTURE)
**Timeline**: 1 week | **Target**: v0.4.1-alpha

#### Implementation Overview
- Interactive update prompts and scheduling
- Update history and comprehensive logging
- Post-update validation and health checks
- Enhanced user experience features

---

### Phase 5: Enterprise Features (FUTURE)
**Timeline**: 1 week | **Target**: v0.5.0-alpha

#### Implementation Overview
- Channel management (stable/beta/alpha)
- Enterprise policies and compliance
- Metrics, reporting, and audit trails
- Silent updates and advanced deployment

## 🛠️ Detailed Implementation Guide

### Creating the Configuration System

#### 1. Update config_manager.go

Add the UpdateConfig struct to the SystemConfig:

```go
// Add to SystemConfig struct
type SystemConfig struct {
    // ... existing fields ...
    UpdateConfig    *UpdateConfig       `json:"update_config,omitempty"`
}

// Add new UpdateConfig struct  
type UpdateConfig struct {
    Enabled              bool   `json:"enabled"`
    CheckOnStartup       bool   `json:"check_on_startup"`
    AutoInstall          bool   `json:"auto_install"`
    Channel              string `json:"channel"`
    CheckInterval        string `json:"check_interval"`
    BackupBeforeUpdate   bool   `json:"backup_before_update"`
    AllowPrerelease      bool   `json:"allow_prerelease"`
    GitHubRepository     string `json:"github_repository"`
    DownloadDirectory    string `json:"download_directory"`
    LastCheck            string `json:"last_check"`
    LastVersion          string `json:"last_version"`
    SkipVersion          string `json:"skip_version"`
    NotificationLevel    string `json:"notification_level"`
}
```

#### 2. Add Getter/Setter Methods

```go
// GetUpdateConfig returns the update configuration
func (cm *ConfigManager) GetUpdateConfig() *UpdateConfig {
    cm.mutex.RLock()
    defer cm.mutex.RUnlock()
    return cm.updateConfig
}

// UpdateUpdateConfig updates the update configuration
func (cm *ConfigManager) UpdateUpdateConfig(config *UpdateConfig) error {
    cm.mutex.Lock()
    defer cm.mutex.Unlock()
    
    cm.updateConfig = config
    return cm.saveConfig()
}
```

#### 3. Environment Variable Support

Add to applyEnvironmentOverrides():

```go
// Update config overrides
if cm.updateConfig != nil {
    cm.updateConfig.Enabled = getEnvBool("DELTA_UPDATE_ENABLED", cm.updateConfig.Enabled)
    cm.updateConfig.CheckOnStartup = getEnvBool("DELTA_UPDATE_CHECK_ON_STARTUP", cm.updateConfig.CheckOnStartup)
    cm.updateConfig.AutoInstall = getEnvBool("DELTA_UPDATE_AUTO_INSTALL", cm.updateConfig.AutoInstall)
    cm.updateConfig.Channel = getEnvString("DELTA_UPDATE_CHANNEL", cm.updateConfig.Channel)
    // ... add other environment variables
}
```

### Creating Version Management

#### 1. Create version_manager.go

```go
package main

import (
    "fmt"
    "regexp"
    "strconv"
    "strings"
)

type Version struct {
    Major      int
    Minor      int
    Patch      int
    Prerelease string
    Build      string
    Original   string
}

// ParseVersion parses a semantic version string
func ParseVersion(version string) (*Version, error) {
    // Remove 'v' prefix if present
    version = strings.TrimPrefix(version, "v")
    
    // Regular expression for semantic versioning
    re := regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`)
    
    matches := re.FindStringSubmatch(version)
    if matches == nil {
        return nil, fmt.Errorf("invalid semantic version: %s", version)
    }
    
    major, _ := strconv.Atoi(matches[1])
    minor, _ := strconv.Atoi(matches[2])
    patch, _ := strconv.Atoi(matches[3])
    
    return &Version{
        Major:      major,
        Minor:      minor,
        Patch:      patch,
        Prerelease: matches[4],
        Build:      matches[5],
        Original:   version,
    }, nil
}

// CompareVersions compares two version strings
// Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
func CompareVersions(v1, v2 string) int {
    ver1, err1 := ParseVersion(v1)
    ver2, err2 := ParseVersion(v2)
    
    if err1 != nil || err2 != nil {
        // Fall back to string comparison for invalid versions
        if v1 < v2 {
            return -1
        } else if v1 > v2 {
            return 1
        }
        return 0
    }
    
    // Compare major.minor.patch
    if ver1.Major != ver2.Major {
        if ver1.Major < ver2.Major {
            return -1
        }
        return 1
    }
    
    if ver1.Minor != ver2.Minor {
        if ver1.Minor < ver2.Minor {
            return -1
        }
        return 1
    }
    
    if ver1.Patch != ver2.Patch {
        if ver1.Patch < ver2.Patch {
            return -1
        }
        return 1
    }
    
    // Handle prerelease versions
    if ver1.Prerelease == "" && ver2.Prerelease != "" {
        return 1 // Release > prerelease
    }
    if ver1.Prerelease != "" && ver2.Prerelease == "" {
        return -1 // Prerelease < release
    }
    if ver1.Prerelease != "" && ver2.Prerelease != "" {
        if ver1.Prerelease < ver2.Prerelease {
            return -1
        } else if ver1.Prerelease > ver2.Prerelease {
            return 1
        }
    }
    
    return 0
}

// IsNewerVersion checks if candidate version is newer than current
func IsNewerVersion(current, candidate string) bool {
    return CompareVersions(current, candidate) < 0
}

// MatchesChannel checks if a version matches a release channel
func MatchesChannel(version string, channel string) bool {
    ver, err := ParseVersion(version)
    if err != nil {
        return false
    }
    
    switch channel {
    case "stable":
        return ver.Prerelease == ""
    case "alpha":
        return strings.Contains(ver.Prerelease, "alpha")
    case "beta":
        return strings.Contains(ver.Prerelease, "beta")
    default:
        return true // Allow all versions for unknown channels
    }
}
```

### Creating Update Manager

#### 1. Create update_manager.go

```go
package main

import (
    "sync"
)

type UpdateManager struct {
    config       UpdateConfig
    currentVer   string
    configMgr    *ConfigManager
    i18nMgr      *I18nManager
    mutex        sync.RWMutex
    isInitialized bool
}

// Global update manager instance
var globalUpdateManager *UpdateManager
var updateOnce sync.Once

// GetUpdateManager returns the global UpdateManager instance
func GetUpdateManager() *UpdateManager {
    updateOnce.Do(func() {
        globalUpdateManager = NewUpdateManager()
    })
    return globalUpdateManager
}

// NewUpdateManager creates a new update manager
func NewUpdateManager() *UpdateManager {
    return &UpdateManager{
        configMgr: GetConfigManager(),
        i18nMgr:   GetI18nManager(),
        currentVer: getCurrentVersion(),
    }
}

// Initialize sets up the update manager
func (um *UpdateManager) Initialize() error {
    um.mutex.Lock()
    defer um.mutex.Unlock()
    
    if um.isInitialized {
        return nil
    }
    
    // Load configuration
    if um.configMgr != nil {
        if config := um.configMgr.GetUpdateConfig(); config != nil {
            um.config = *config
        } else {
            // Set default configuration
            um.config = UpdateConfig{
                Enabled:              true,
                CheckOnStartup:       true,
                AutoInstall:          false,
                Channel:              "stable",
                CheckInterval:        "daily",
                BackupBeforeUpdate:   true,
                AllowPrerelease:      false,
                GitHubRepository:     "DeltaCLI/Delta",
                DownloadDirectory:    getDefaultDownloadDir(),
                NotificationLevel:    "prompt",
            }
            
            // Save default config
            um.configMgr.UpdateUpdateConfig(&um.config)
        }
    }
    
    um.isInitialized = true
    return nil
}

// GetCurrentVersion returns the current version of Delta CLI
func (um *UpdateManager) GetCurrentVersion() string {
    um.mutex.RLock()
    defer um.mutex.RUnlock()
    return um.currentVer
}

// GetConfig returns the current update configuration
func (um *UpdateManager) GetConfig() UpdateConfig {
    um.mutex.RLock()
    defer um.mutex.RUnlock()
    return um.config
}

// UpdateConfig updates the update configuration
func (um *UpdateManager) UpdateConfig(config UpdateConfig) error {
    um.mutex.Lock()
    defer um.mutex.Unlock()
    
    um.config = config
    
    if um.configMgr != nil {
        return um.configMgr.UpdateUpdateConfig(&config)
    }
    
    return nil
}

// IsEnabled returns whether the update system is enabled
func (um *UpdateManager) IsEnabled() bool {
    um.mutex.RLock()
    defer um.mutex.RUnlock()
    return um.config.Enabled
}

// Helper functions
func getCurrentVersion() string {
    // This should return the actual current version
    // For now, return a placeholder
    return "v0.2.0-alpha"
}

func getDefaultDownloadDir() string {
    // Return platform-appropriate download directory
    homeDir, _ := os.UserHomeDir()
    return filepath.Join(homeDir, ".config", "delta", "updates")
}
```

### Creating CLI Commands

#### 1. Create update_commands.go

```go
package main

import (
    "fmt"
    "strings"
)

// HandleUpdateCommand processes update-related commands
func HandleUpdateCommand(args []string) bool {
    um := GetUpdateManager()
    if um == nil {
        fmt.Println("Update system not available")
        return true
    }
    
    // Initialize if needed
    if err := um.Initialize(); err != nil {
        fmt.Printf("Error initializing update system: %v\n", err)
        return true
    }
    
    if len(args) == 0 {
        showUpdateStatus(um)
        return true
    }
    
    switch args[0] {
    case "status":
        showUpdateStatus(um)
    case "config":
        if len(args) == 1 {
            showUpdateConfig(um)
        } else {
            updateConfigSetting(um, args[1:])
        }
    case "version":
        showVersionInfo(um)
    case "help":
        showUpdateHelp()
    default:
        fmt.Printf("Unknown update command: %s\n", args[0])
        fmt.Println("Use ':update help' for available commands")
    }
    
    return true
}

// showUpdateStatus displays the current update system status
func showUpdateStatus(um *UpdateManager) {
    config := um.GetConfig()
    
    fmt.Println("Update System Status:")
    fmt.Printf("  Current Version: %s\n", um.GetCurrentVersion())
    fmt.Printf("  Update Checking: %s\n", getStatusText(config.Enabled))
    fmt.Printf("  Check on Startup: %s\n", getStatusText(config.CheckOnStartup))
    fmt.Printf("  Auto Install: %s\n", getStatusText(config.AutoInstall))
    fmt.Printf("  Channel: %s\n", config.Channel)
    fmt.Printf("  Check Interval: %s\n", config.CheckInterval)
    fmt.Printf("  Last Check: %s\n", getLastCheckText(config.LastCheck))
    
    if config.SkipVersion != "" {
        fmt.Printf("  Skipped Version: %s\n", config.SkipVersion)
    }
}

// showUpdateConfig displays the current configuration
func showUpdateConfig(um *UpdateManager) {
    config := um.GetConfig()
    
    fmt.Println("Update Configuration:")
    fmt.Printf("  enabled: %t\n", config.Enabled)
    fmt.Printf("  check_on_startup: %t\n", config.CheckOnStartup)
    fmt.Printf("  auto_install: %t\n", config.AutoInstall)
    fmt.Printf("  channel: %s\n", config.Channel)
    fmt.Printf("  check_interval: %s\n", config.CheckInterval)
    fmt.Printf("  backup_before_update: %t\n", config.BackupBeforeUpdate)
    fmt.Printf("  allow_prerelease: %t\n", config.AllowPrerelease)
    fmt.Printf("  notification_level: %s\n", config.NotificationLevel)
    fmt.Printf("  github_repository: %s\n", config.GitHubRepository)
    fmt.Printf("  download_directory: %s\n", config.DownloadDirectory)
}

// updateConfigSetting updates a specific configuration setting
func updateConfigSetting(um *UpdateManager, args []string) {
    if len(args) < 2 {
        fmt.Println("Usage: :update config <setting> <value>")
        return
    }
    
    setting := args[0]
    value := args[1]
    config := um.GetConfig()
    
    switch setting {
    case "enabled":
        config.Enabled = (value == "true" || value == "1" || value == "yes")
    case "check_on_startup":
        config.CheckOnStartup = (value == "true" || value == "1" || value == "yes")
    case "auto_install":
        config.AutoInstall = (value == "true" || value == "1" || value == "yes")
    case "channel":
        if value == "stable" || value == "alpha" || value == "beta" {
            config.Channel = value
        } else {
            fmt.Printf("Invalid channel: %s (must be stable, alpha, or beta)\n", value)
            return
        }
    case "check_interval":
        if value == "daily" || value == "weekly" || value == "monthly" {
            config.CheckInterval = value
        } else {
            fmt.Printf("Invalid interval: %s (must be daily, weekly, or monthly)\n", value)
            return
        }
    case "notification_level":
        if value == "silent" || value == "notify" || value == "prompt" {
            config.NotificationLevel = value
        } else {
            fmt.Printf("Invalid notification level: %s (must be silent, notify, or prompt)\n", value)
            return
        }
    default:
        fmt.Printf("Unknown setting: %s\n", setting)
        return
    }
    
    if err := um.UpdateConfig(config); err != nil {
        fmt.Printf("Error updating configuration: %v\n", err)
    } else {
        fmt.Printf("Updated %s to %s\n", setting, value)
    }
}

// showVersionInfo displays version information
func showVersionInfo(um *UpdateManager) {
    fmt.Printf("Current Version: %s\n", um.GetCurrentVersion())
    // In future phases, this will show more version details
}

// showUpdateHelp displays help for update commands
func showUpdateHelp() {
    fmt.Println("Update Commands")
    fmt.Println("===============")
    fmt.Println("  :update                     - Show update status")
    fmt.Println("  :update status              - Show detailed status")
    fmt.Println("  :update config              - Show configuration")
    fmt.Println("  :update config <key> <val>  - Set configuration value")
    fmt.Println("  :update version             - Show version information")
    fmt.Println("  :update help                - Show this help")
    fmt.Println()
    fmt.Println("Configuration Settings:")
    fmt.Println("  enabled                     - Enable/disable update system")
    fmt.Println("  check_on_startup            - Check for updates on startup")
    fmt.Println("  auto_install                - Automatically install updates")
    fmt.Println("  channel                     - Release channel (stable/alpha/beta)")
    fmt.Println("  check_interval              - Check frequency (daily/weekly/monthly)")
    fmt.Println("  notification_level          - Notification level (silent/notify/prompt)")
    fmt.Println()
    fmt.Println("Examples:")
    fmt.Println("  :update config enabled true")
    fmt.Println("  :update config channel alpha")
    fmt.Println("  :update config auto_install false")
}

// Helper functions
func getStatusText(enabled bool) string {
    if enabled {
        return "Enabled"
    }
    return "Disabled"
}

func getLastCheckText(lastCheck string) string {
    if lastCheck == "" {
        return "Never"
    }
    return lastCheck
}
```

### Integration with Main CLI

#### 1. Update cli.go

Add to the command handling section:

```go
// In the command handling switch statement
case "update":
    return HandleUpdateCommand(args[1:])
```

#### 2. Update help.go

Add update commands to the help output:

```go
// Add to showEnhancedHelp()
fmt.Println("")
fmt.Println("  Update System:")
fmt.Println("  :update               - Show update status")
fmt.Println("  :update config        - Manage update configuration")
fmt.Println("  :update help          - Show update command help")
```

#### 3. Add to Initialization

Add update manager initialization to the startup sequence:

```go
// In runInteractiveShell() or initialization function
updateMgr := GetUpdateManager()
if updateMgr != nil {
    if err := updateMgr.Initialize(); err != nil {
        fmt.Printf("Warning: Failed to initialize update system: %v\n", err)
    }
}
```

## 🧪 Testing Your Implementation

### Phase 1 Testing

```bash
# Test configuration
echo "update config" | ./delta
echo "update config enabled true" | ./delta
echo "update config channel alpha" | ./delta

# Test status
echo "update status" | ./delta

# Test version management
echo "update version" | ./delta

# Test help
echo "update help" | ./delta
```

### Version Testing

Create a test file to verify version comparison:

```go
// version_manager_test.go
func TestVersionComparison(t *testing.T) {
    testCases := []struct {
        v1       string
        v2       string
        expected int
    }{
        {"v1.0.0", "v1.0.1", -1},
        {"v1.1.0", "v1.0.9", 1},
        {"v1.0.0", "v1.0.0", 0},
        {"v1.0.0", "v1.0.0-alpha.1", 1},
        {"v1.0.0-alpha.1", "v1.0.0-alpha.2", -1},
    }
    
    for _, tc := range testCases {
        result := CompareVersions(tc.v1, tc.v2)
        if result != tc.expected {
            t.Errorf("CompareVersions(%s, %s) = %d, expected %d", 
                tc.v1, tc.v2, result, tc.expected)
        }
    }
}
```

## 📋 Next Steps

After completing Phase 1:

1. **Validate Foundation**: Ensure all basic functionality works
2. **User Testing**: Get feedback on CLI interface and configuration
3. **Performance Review**: Check startup time impact
4. **Documentation**: Update user guides and API docs
5. **Phase 2 Planning**: Begin GitHub integration design

## 🎯 Success Criteria

Phase 1 is complete when:

- ✅ Update configuration integrates with existing config system
- ✅ Version management utilities handle all common cases
- ✅ CLI commands provide intuitive update management
- ✅ No performance impact on startup
- ✅ All tests pass
- ✅ Documentation is updated

This foundation enables the exciting functionality in subsequent phases: GitHub integration, automatic downloads, and enterprise features.

---

*Follow this implementation plan step-by-step to build Delta CLI's comprehensive auto-update system. Each phase builds upon the previous foundation to create a robust, secure, and user-friendly update experience.*