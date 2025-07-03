package main

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"
)

// UpdateChecker handles checking for updates from GitHub
type UpdateChecker struct {
	githubClient  *GitHubClient
	updateManager *UpdateManager
	i18nManager   *I18nManager
	mutex         sync.RWMutex
	isChecking    bool
	lastCheck     time.Time
	lastResult    *UpdateInfo
}

// Global update checker instance
var globalUpdateChecker *UpdateChecker
var updateCheckerOnce sync.Once

// GetUpdateChecker returns the global UpdateChecker instance
func GetUpdateChecker() *UpdateChecker {
	updateCheckerOnce.Do(func() {
		um := GetUpdateManager()
		if um != nil {
			config := um.GetConfig()
			client := NewGitHubClient(config.GitHubRepository, "")
			globalUpdateChecker = NewUpdateChecker(client, um)
		}
	})
	return globalUpdateChecker
}

// NewUpdateChecker creates a new update checker instance
func NewUpdateChecker(githubClient *GitHubClient, updateManager *UpdateManager) *UpdateChecker {
	return &UpdateChecker{
		githubClient:  githubClient,
		updateManager: updateManager,
		i18nManager:   GetI18nManager(),
	}
}

// CheckForUpdates performs an update check and returns update information
func (uc *UpdateChecker) CheckForUpdates() (*UpdateInfo, error) {
	uc.mutex.Lock()
	defer uc.mutex.Unlock()

	if uc.isChecking {
		return nil, fmt.Errorf("update check already in progress")
	}

	uc.isChecking = true
	checkStartTime := time.Now()
	defer func() {
		uc.isChecking = false
		uc.lastCheck = time.Now()
	}()

	config := uc.updateManager.GetConfig()
	currentVersion := uc.updateManager.GetCurrentVersion()

	// Get the latest release for the configured channel
	latestRelease, err := uc.githubClient.GetLatestRelease(config.Channel)
	if err != nil {
		// Record failed check
		if metrics := GetUpdateMetrics(); metrics != nil {
			metrics.RecordUpdateCheck(currentVersion, false, time.Since(checkStartTime))
		}
		return nil, fmt.Errorf("failed to fetch latest release: %v", err)
	}

	// Extract version from tag
	latestVersion := GetVersionFromTag(latestRelease.TagName)
	if !IsValidVersion(latestVersion) {
		return nil, fmt.Errorf("invalid version format in latest release: %s", latestVersion)
	}

	// Check if update is available
	hasUpdate := IsNewerVersion(currentVersion, latestVersion)

	// Development build handling
	isDevelopmentBuild := IsDevelopmentBuild()
	
	// Skip version check
	if config.SkipVersion != "" && latestVersion == config.SkipVersion {
		hasUpdate = false
	}

	// Channel filtering
	if !MatchesChannel(latestVersion, config.Channel) {
		hasUpdate = false
	}

	// Prerelease filtering
	if !config.AllowPrerelease {
		if ver, err := ParseVersion(latestVersion); err == nil && ver.IsPrerelease() {
			hasUpdate = false
		}
	}

	// Development build special handling
	if isDevelopmentBuild {
		// For development builds, we're more conservative about updates
		// Only suggest updates if the latest version is significantly newer
		if currentVer, err := ParseVersion(currentVersion); err == nil {
			if latestVer, err := ParseVersion(latestVersion); err == nil {
				// Only suggest if it's a major or minor version bump, not just patch
				if latestVer.Major == currentVer.Major && latestVer.Minor == currentVer.Minor {
					hasUpdate = false
				}
			}
		}
	}

	updateInfo := &UpdateInfo{
		HasUpdate:      hasUpdate,
		CurrentVersion: currentVersion,
		LatestVersion:  latestVersion,
		LatestRelease:  latestRelease,
		ReleaseNotes:   latestRelease.Body,
		IsPrerelease:   latestRelease.Prerelease,
		PublishedAt:    latestRelease.PublishedAt,
	}

	// Find appropriate download asset
	if hasUpdate && len(latestRelease.Assets) > 0 {
		asset, err := uc.githubClient.SelectAssetForPlatform(latestRelease.Assets)
		if err == nil && asset != nil {
			updateInfo.DownloadURL = asset.BrowserDownloadURL
			updateInfo.AssetName = asset.Name
			updateInfo.AssetSize = asset.Size
		}
	}

	// Cache the result
	uc.lastResult = updateInfo

	// Update last check time in config
	uc.updateLastCheckTime()

	// Record successful check metrics
	if metrics := GetUpdateMetrics(); metrics != nil {
		metrics.RecordUpdateCheck(currentVersion, hasUpdate, time.Since(checkStartTime))
	}

	return updateInfo, nil
}

// CheckForUpdatesAsync performs an asynchronous update check
func (uc *UpdateChecker) CheckForUpdatesAsync(callback func(*UpdateInfo, error)) {
	go func() {
		updateInfo, err := uc.CheckForUpdates()
		if callback != nil {
			callback(updateInfo, err)
		}
	}()
}

// ShouldCheck determines if an update check should be performed
func (uc *UpdateChecker) ShouldCheck() bool {
	uc.mutex.RLock()
	defer uc.mutex.RUnlock()

	config := uc.updateManager.GetConfig()

	// Check if updates are enabled
	if !config.Enabled {
		return false
	}

	// Check if already checking
	if uc.isChecking {
		return false
	}

	// Parse last check time
	lastCheckTime, err := time.Parse(time.RFC3339, config.LastCheck)
	if err != nil {
		// If no valid last check time, we should check
		return true
	}

	// Check based on interval
	switch config.CheckInterval {
	case "daily":
		return time.Since(lastCheckTime) >= 24*time.Hour
	case "weekly":
		return time.Since(lastCheckTime) >= 7*24*time.Hour
	case "monthly":
		return time.Since(lastCheckTime) >= 30*24*time.Hour
	default:
		// Default to daily
		return time.Since(lastCheckTime) >= 24*time.Hour
	}
}

// GetAvailableUpdate returns information about the latest available update
func (uc *UpdateChecker) GetAvailableUpdate() (*UpdateInfo, error) {
	uc.mutex.RLock()
	
	// Return cached result if available and recent
	if uc.lastResult != nil && time.Since(uc.lastCheck) < 30*time.Minute {
		result := uc.lastResult
		uc.mutex.RUnlock()
		return result, nil
	}
	
	// Release the read lock before calling CheckForUpdates (which needs write lock)
	uc.mutex.RUnlock()
	
	// Perform a new check
	return uc.CheckForUpdates()
}

// NotifyUpdateAvailable handles user notification for available updates
func (uc *UpdateChecker) NotifyUpdateAvailable(updateInfo *UpdateInfo) {
	if updateInfo == nil || !updateInfo.HasUpdate {
		return
	}

	config := uc.updateManager.GetConfig()

	// Check if this version is skipped
	if config.SkipVersion == updateInfo.LatestVersion {
		return // Don't notify about skipped versions
	}

	// Check if this version is postponed
	ui := NewUpdateUI()
	if ui.IsPostponementActive(updateInfo) {
		return // Don't notify during active postponement
	}

	switch config.NotificationLevel {
	case "silent":
		// No notification
		return
	case "notify":
		uc.showUpdateNotification(updateInfo)
	case "prompt":
		uc.showInteractiveUpdatePrompt(updateInfo)
	default:
		uc.showUpdateNotification(updateInfo)
	}
}

// showUpdateNotification displays a simple update notification
func (uc *UpdateChecker) showUpdateNotification(updateInfo *UpdateInfo) {
	fmt.Printf("\n📢 Update Available: Delta CLI %s → %s\n", 
		updateInfo.CurrentVersion, updateInfo.LatestVersion)
	
	if updateInfo.IsPrerelease {
		fmt.Printf("   ⚠️  This is a prerelease version\n")
	}
	
	fmt.Printf("   Published: %s\n", updateInfo.PublishedAt.Format("2006-01-02"))
	fmt.Printf("   Use ':update' to manage updates\n\n")
}

// showInteractiveUpdatePrompt displays an interactive update prompt with choices
func (uc *UpdateChecker) showInteractiveUpdatePrompt(updateInfo *UpdateInfo) {
	ui := NewUpdateUI()
	
	// Configure prompt options based on update config
	config := uc.updateManager.GetConfig()
	options := &UpdatePromptOptions{
		ShowChangelog:   true,
		AllowPostpone:   true,
		AllowSkip:       true,
		AutoConfirm:     config.AutoInstall,
		PostponeOptions: []string{"1 hour", "4 hours", "1 day", "1 week"},
		DefaultChoice:   UpdateChoiceCancel,
	}
	
	choice := ui.PromptForUpdate(updateInfo, options)
	
	switch choice {
	case UpdateChoiceInstall:
		uc.handleInteractiveInstall(updateInfo)
	case UpdateChoiceSkip:
		uc.handleSkipVersion(updateInfo)
	case UpdateChoicePostpone:
		// Postponement is handled in the UI
		fmt.Println("Update postponed.")
	case UpdateChoiceCancel:
		fmt.Println("Update cancelled.")
	}
}

// showUpdatePrompt displays a basic update prompt (for backward compatibility)
func (uc *UpdateChecker) showUpdatePrompt(updateInfo *UpdateInfo) {
	fmt.Printf("\n🔔 Update Available!\n")
	fmt.Printf("   Current Version: %s\n", updateInfo.CurrentVersion)
	fmt.Printf("   Latest Version:  %s\n", updateInfo.LatestVersion)
	
	if updateInfo.IsPrerelease {
		fmt.Printf("   Type: Prerelease\n")
	}
	
	fmt.Printf("   Published: %s\n", updateInfo.PublishedAt.Format("2006-01-02 15:04"))
	
	if updateInfo.AssetSize > 0 {
		fmt.Printf("   Download Size: %s\n", formatFileSize(updateInfo.AssetSize))
	}
	
	// Show condensed release notes
	if updateInfo.ReleaseNotes != "" {
		notes := uc.condenseReleaseNotes(updateInfo.ReleaseNotes)
		if notes != "" {
			fmt.Printf("   Release Notes: %s\n", notes)
		}
	}
	
	fmt.Printf("\n   Use ':update config' to manage update settings\n")
	fmt.Printf("   Use ':update help' for more options\n\n")
}

// condenseReleaseNotes creates a brief summary of release notes
func (uc *UpdateChecker) condenseReleaseNotes(notes string) string {
	if len(notes) <= 100 {
		return strings.TrimSpace(notes)
	}
	
	// Take first line or first 100 characters
	lines := strings.Split(notes, "\n")
	firstLine := strings.TrimSpace(lines[0])
	
	if len(firstLine) <= 100 {
		return firstLine
	}
	
	// Truncate to 97 characters and add ellipsis
	return strings.TrimSpace(firstLine[:97]) + "..."
}

// updateLastCheckTime updates the last check time in configuration
func (uc *UpdateChecker) updateLastCheckTime() {
	config := uc.updateManager.GetConfig()
	config.LastCheck = time.Now().Format(time.RFC3339)
	uc.updateManager.UpdateConfig(config)
}

// IsChecking returns whether an update check is currently in progress
func (uc *UpdateChecker) IsChecking() bool {
	uc.mutex.RLock()
	defer uc.mutex.RUnlock()
	return uc.isChecking
}

// GetLastCheck returns the time of the last update check
func (uc *UpdateChecker) GetLastCheck() time.Time {
	uc.mutex.RLock()
	defer uc.mutex.RUnlock()
	return uc.lastCheck
}

// GetCachedResult returns the last cached update check result
func (uc *UpdateChecker) GetCachedResult() *UpdateInfo {
	uc.mutex.RLock()
	defer uc.mutex.RUnlock()
	return uc.lastResult
}

// SetGitHubToken sets the GitHub API token for authenticated requests
func (uc *UpdateChecker) SetGitHubToken(token string) {
	if uc.githubClient != nil {
		uc.githubClient.SetToken(token)
	}
}

// GetRateLimitStatus returns GitHub API rate limit information
func (uc *UpdateChecker) GetRateLimitStatus() map[string]interface{} {
	if uc.githubClient != nil {
		return uc.githubClient.GetRateLimitStatus()
	}
	return nil
}

// PerformStartupCheck performs an update check if configured for startup checking
func (uc *UpdateChecker) PerformStartupCheck() {
	config := uc.updateManager.GetConfig()
	
	if !config.Enabled || !config.CheckOnStartup {
		return
	}

	if !uc.ShouldCheck() {
		return
	}

	// Perform async check to avoid blocking startup
	uc.CheckForUpdatesAsync(func(updateInfo *UpdateInfo, err error) {
		if err != nil {
			// Silently fail for startup checks
			return
		}
		
		if updateInfo != nil && updateInfo.HasUpdate {
			uc.NotifyUpdateAvailable(updateInfo)
		}
	})
}

// Helper functions

// formatFileSize is implemented in update_commands.go

// getCurrentPlatformString returns a platform string for asset selection
func getCurrentPlatformString() string {
	return fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
}

// validateUpdateInfo performs basic validation on update information
func validateUpdateInfo(updateInfo *UpdateInfo) error {
	if updateInfo == nil {
		return fmt.Errorf("update info is nil")
	}
	
	if updateInfo.HasUpdate {
		if updateInfo.LatestVersion == "" {
			return fmt.Errorf("latest version is empty")
		}
		
		if !IsValidVersion(updateInfo.LatestVersion) {
			return fmt.Errorf("latest version is invalid: %s", updateInfo.LatestVersion)
		}
		
		if updateInfo.LatestRelease == nil {
			return fmt.Errorf("latest release is nil")
		}
	}
	
	return nil
}

// handleInteractiveInstall processes the install choice from interactive prompt
func (uc *UpdateChecker) handleInteractiveInstall(updateInfo *UpdateInfo) {
	um := uc.updateManager
	if um == nil {
		fmt.Printf("❌ Update manager not available\n")
		return
	}

	fmt.Printf("Installing update %s...\n", updateInfo.LatestVersion)
	
	installResult, err := um.DownloadAndInstallUpdate(updateInfo.LatestVersion)
	if err != nil {
		fmt.Printf("❌ Update failed: %v\n", err)
		if installResult != nil && installResult.BackupPath != "" {
			fmt.Printf("Backup available at: %s\n", installResult.BackupPath)
			fmt.Printf("Use ':update rollback' to restore if needed\n")
		}
		return
	}

	fmt.Printf("✅ Update completed successfully!\n")
	fmt.Printf("   Old Version: %s\n", installResult.OldVersion)
	fmt.Printf("   New Version: %s\n", installResult.NewVersion)
	fmt.Printf("   Installation Time: %s\n", installResult.InstallTime.Truncate(time.Millisecond))
	fmt.Printf("\n🔄 Please restart Delta CLI to use the new version\n")
}

// handleSkipVersion processes the skip choice from interactive prompt
func (uc *UpdateChecker) handleSkipVersion(updateInfo *UpdateInfo) {
	config := uc.updateManager.GetConfig()
	config.SkipVersion = updateInfo.LatestVersion
	
	err := uc.updateManager.UpdateConfig(config)
	if err != nil {
		fmt.Printf("❌ Failed to skip version: %v\n", err)
		return
	}
	
	fmt.Printf("✅ Version %s will be skipped. You won't be notified about this version again.\n", updateInfo.LatestVersion)
	fmt.Printf("Use ':update config skip_version \"\"' to re-enable notifications for this version.\n")
}

// CheckPostponementReminders checks if any postponed updates should be reminded
func (uc *UpdateChecker) CheckPostponementReminders() {
	config := uc.updateManager.GetConfig()
	
	if config.PostponedVersion == "" || config.PostponedUntil == "" {
		return
	}
	
	postponedUntil, err := time.Parse(time.RFC3339, config.PostponedUntil)
	if err != nil {
		return
	}
	
	// If postponement has expired, show reminder
	if time.Now().After(postponedUntil) {
		updateInfo, err := uc.GetAvailableUpdate()
		if err == nil && updateInfo != nil && updateInfo.LatestVersion == config.PostponedVersion {
			ui := NewUpdateUI()
			ui.ShowPostponementReminder(updateInfo)
			ui.ClearPostponement() // Clear expired postponement
		}
	}
}