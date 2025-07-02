package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// UpdateManager manages the auto-update system
type UpdateManager struct {
	config        UpdateConfig
	currentVer    string
	configMgr     *ConfigManager
	i18nMgr       *I18nManager
	mutex         sync.RWMutex
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
		configMgr:  GetConfigManager(),
		i18nMgr:    GetI18nManager(),
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
				GitHubRepository:     "deltacli/delta",
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

// SetEnabled enables or disables the update system
func (um *UpdateManager) SetEnabled(enabled bool) error {
	um.mutex.Lock()
	defer um.mutex.Unlock()
	
	um.config.Enabled = enabled
	
	if um.configMgr != nil {
		return um.configMgr.UpdateUpdateConfig(&um.config)
	}
	
	return nil
}

// GetChannel returns the current update channel
func (um *UpdateManager) GetChannel() string {
	// Use ChannelManager if available
	if cm := GetChannelManager(); cm != nil {
		return string(cm.GetCurrentChannel())
	}
	
	um.mutex.RLock()
	defer um.mutex.RUnlock()
	return um.config.Channel
}

// SetChannel sets the update channel
func (um *UpdateManager) SetChannel(channel string) error {
	// Use ChannelManager if available
	if cm := GetChannelManager(); cm != nil {
		return cm.SetChannel(UpdateChannel(channel), "Manual channel change via update config")
	}
	
	um.mutex.Lock()
	defer um.mutex.Unlock()
	
	// Validate channel
	if !ValidateChannelName(channel) {
		return fmt.Errorf("invalid channel: %s", channel)
	}
	
	um.config.Channel = channel
	
	if um.configMgr != nil {
		return um.configMgr.UpdateUpdateConfig(&um.config)
	}
	
	return nil
}

// ShouldCheckOnStartup returns whether updates should be checked on startup
func (um *UpdateManager) ShouldCheckOnStartup() bool {
	um.mutex.RLock()
	defer um.mutex.RUnlock()
	return um.config.Enabled && um.config.CheckOnStartup
}

// GetNotificationLevel returns the current notification level
func (um *UpdateManager) GetNotificationLevel() string {
	um.mutex.RLock()
	defer um.mutex.RUnlock()
	return um.config.NotificationLevel
}

// SetNotificationLevel sets the notification level
func (um *UpdateManager) SetNotificationLevel(level string) error {
	um.mutex.Lock()
	defer um.mutex.Unlock()
	
	// Validate notification level
	validLevels := []string{"silent", "notify", "prompt"}
	isValid := false
	for _, valid := range validLevels {
		if level == valid {
			isValid = true
			break
		}
	}
	
	if !isValid {
		return fmt.Errorf("invalid notification level: %s (must be silent, notify, or prompt)", level)
	}
	
	um.config.NotificationLevel = level
	
	if um.configMgr != nil {
		return um.configMgr.UpdateUpdateConfig(&um.config)
	}
	
	return nil
}

// GetDownloadDirectory returns the download directory for updates
func (um *UpdateManager) GetDownloadDirectory() string {
	um.mutex.RLock()
	defer um.mutex.RUnlock()
	
	if um.config.DownloadDirectory == "" {
		return getDefaultDownloadDir()
	}
	
	return um.config.DownloadDirectory
}

// SetDownloadDirectory sets the download directory for updates
func (um *UpdateManager) SetDownloadDirectory(dir string) error {
	um.mutex.Lock()
	defer um.mutex.Unlock()
	
	um.config.DownloadDirectory = dir
	
	if um.configMgr != nil {
		return um.configMgr.UpdateUpdateConfig(&um.config)
	}
	
	return nil
}

// Helper functions
func getCurrentVersion() string {
	// This should return the actual current version
	// Using the global version from version.go
	return Version
}

func getDefaultDownloadDir() string {
	// Return platform-appropriate download directory
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".config", "delta", "updates")
}

// CheckForUpdates performs an update check using the update checker
func (um *UpdateManager) CheckForUpdates() (*UpdateInfo, error) {
	if !um.IsEnabled() {
		return nil, fmt.Errorf("update system is disabled")
	}

	checker := GetUpdateChecker()
	if checker == nil {
		return nil, fmt.Errorf("update checker not available")
	}

	return checker.CheckForUpdates()
}

// CheckForUpdatesAsync performs an asynchronous update check
func (um *UpdateManager) CheckForUpdatesAsync(callback func(*UpdateInfo, error)) {
	if !um.IsEnabled() {
		if callback != nil {
			callback(nil, fmt.Errorf("update system is disabled"))
		}
		return
	}

	checker := GetUpdateChecker()
	if checker == nil {
		if callback != nil {
			callback(nil, fmt.Errorf("update checker not available"))
		}
		return
	}

	checker.CheckForUpdatesAsync(callback)
}

// GetAvailableUpdate returns information about the latest available update
func (um *UpdateManager) GetAvailableUpdate() (*UpdateInfo, error) {
	if !um.IsEnabled() {
		return nil, fmt.Errorf("update system is disabled")
	}

	checker := GetUpdateChecker()
	if checker == nil {
		return nil, fmt.Errorf("update checker not available")
	}

	return checker.GetAvailableUpdate()
}

// ShouldPerformStartupCheck returns whether updates should be checked on startup
func (um *UpdateManager) ShouldPerformStartupCheck() bool {
	return um.config.Enabled && um.config.CheckOnStartup
}

// PerformStartupCheck performs an update check if configured for startup
func (um *UpdateManager) PerformStartupCheck() {
	if !um.ShouldPerformStartupCheck() {
		return
	}

	checker := GetUpdateChecker()
	if checker != nil {
		checker.PerformStartupCheck()
	}
}

// SetGitHubToken sets the GitHub API token for authenticated requests
func (um *UpdateManager) SetGitHubToken(token string) error {
	checker := GetUpdateChecker()
	if checker != nil {
		checker.SetGitHubToken(token)
		return nil
	}
	return fmt.Errorf("update checker not available")
}

// GetRateLimitStatus returns GitHub API rate limit information
func (um *UpdateManager) GetRateLimitStatus() map[string]interface{} {
	checker := GetUpdateChecker()
	if checker != nil {
		return checker.GetRateLimitStatus()
	}
	return nil
}

// DownloadUpdate downloads an update for the given version
func (um *UpdateManager) DownloadUpdate(version string) (*DownloadResult, error) {
	if !um.IsEnabled() {
		return nil, fmt.Errorf("update system is disabled")
	}

	// Get the release information
	checker := GetUpdateChecker()
	if checker == nil {
		return nil, fmt.Errorf("update checker not available")
	}

	// Get GitHub client from the checker
	client := checker.githubClient
	if client == nil {
		return nil, fmt.Errorf("GitHub client not available")
	}

	// Get the specific release
	release, err := client.GetReleaseByTag(version)
	if err != nil {
		return nil, fmt.Errorf("failed to get release %s: %v", version, err)
	}

	// Create downloader
	downloader := NewUpdateDownloader(um.GetDownloadDirectory())
	if downloader == nil {
		return nil, fmt.Errorf("failed to create downloader")
	}

	// Download the update
	return downloader.DownloadUpdate(release)
}

// InstallUpdate installs a downloaded update
func (um *UpdateManager) InstallUpdate(downloadResult *DownloadResult) (*InstallResult, error) {
	if !um.IsEnabled() {
		return nil, fmt.Errorf("update system is disabled")
	}

	if downloadResult == nil {
		return nil, fmt.Errorf("download result is nil")
	}

	// Create installer
	installer, err := NewUpdateInstaller()
	if err != nil {
		return nil, fmt.Errorf("failed to create installer: %v", err)
	}

	// Install the update
	return installer.InstallUpdate(downloadResult)
}

// DownloadAndInstallUpdate downloads and installs an update in one operation
func (um *UpdateManager) DownloadAndInstallUpdate(version string) (*InstallResult, error) {
	if !um.IsEnabled() {
		return nil, fmt.Errorf("update system is disabled")
	}

	// Download the update
	downloadResult, err := um.DownloadUpdate(version)
	if err != nil {
		return nil, fmt.Errorf("download failed: %v", err)
	}

	// Install the update
	return um.InstallUpdate(downloadResult)
}

// RollbackToPreviousVersion rolls back to the most recent backup
func (um *UpdateManager) RollbackToPreviousVersion() error {
	if !um.IsEnabled() {
		return fmt.Errorf("update system is disabled")
	}

	// Create installer to access rollback functionality
	installer, err := NewUpdateInstaller()
	if err != nil {
		return fmt.Errorf("failed to create installer: %v", err)
	}

	// Get available backups
	backups, err := installer.GetBackupInfo()
	if err != nil {
		return fmt.Errorf("failed to get backup info: %v", err)
	}

	if len(backups) == 0 {
		return fmt.Errorf("no backups available")
	}

	// Sort backups by time (newest first)
	for i := 0; i < len(backups)-1; i++ {
		for j := i + 1; j < len(backups); j++ {
			if backups[i].BackupTime.Before(backups[j].BackupTime) {
				backups[i], backups[j] = backups[j], backups[i]
			}
		}
	}

	// Rollback to the most recent backup
	return installer.Rollback(backups[0].BackupPath)
}

// GetBackupInfo returns information about available backups
func (um *UpdateManager) GetBackupInfo() ([]BackupInfo, error) {
	installer, err := NewUpdateInstaller()
	if err != nil {
		return nil, fmt.Errorf("failed to create installer: %v", err)
	}

	return installer.GetBackupInfo()
}

// CleanupOldBackups removes old backup files
func (um *UpdateManager) CleanupOldBackups(keepCount int) error {
	installer, err := NewUpdateInstaller()
	if err != nil {
		return fmt.Errorf("failed to create installer: %v", err)
	}

	return installer.CleanupOldBackups(keepCount)
}

// CleanupOldDownloads removes old download files
func (um *UpdateManager) CleanupOldDownloads(keepCount int) error {
	downloader := NewUpdateDownloader(um.GetDownloadDirectory())
	return downloader.CleanupOldDownloads(keepCount)
}

// GetDownloadStats returns statistics about downloads
func (um *UpdateManager) GetDownloadStats() map[string]interface{} {
	downloader := NewUpdateDownloader(um.GetDownloadDirectory())
	return downloader.GetDownloadStats()
}

// String returns a string representation of the UpdateManager status
func (um *UpdateManager) String() string {
	um.mutex.RLock()
	defer um.mutex.RUnlock()
	
	return fmt.Sprintf("UpdateManager{version: %s, enabled: %t, channel: %s, initialized: %t}", 
		um.currentVer, um.config.Enabled, um.config.Channel, um.isInitialized)
}