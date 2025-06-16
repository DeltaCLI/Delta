package main

import (
	"fmt"
	"strings"
	"time"
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
	case "check":
		performUpdateCheck(um)
	case "download":
		handleDownloadCommand(um, args[1:])
	case "install":
		handleInstallCommand(um, args[1:])
	case "update":
		handleUpdateCommand(um, args[1:])
	case "rollback":
		handleRollbackCommand(um)
	case "backups":
		handleBackupsCommand(um)
	case "cleanup":
		handleCleanupCommand(um, args[1:])
	case "config":
		if len(args) == 1 {
			showUpdateConfig(um)
		} else {
			updateConfigSetting(um, args[1:])
		}
	case "version":
		showVersionInfo(um)
	case "info":
		showUpdateInfo(um)
	case "rate-limit":
		showRateLimitStatus(um)
	case "interactive":
		handleInteractiveUpdateCommand(um)
	case "skip":
		handleSkipCommand(um, args[1:])
	case "postpone":
		handlePostponeCommand(um, args[1:])
	case "reminder":
		handleReminderCommand(um)
	case "schedule":
		handleScheduleCommand(um, args[1:])
	case "pending":
		handlePendingCommand(um)
	case "cancel":
		handleCancelCommand(um, args[1:])
	case "logs":
		handleLogsCommand(um, args[1:])
	case "validate":
		handleValidateCommand(um, args[1:])
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
	fmt.Printf("  Update Checking: %s\n", getUpdateStatusText(config.Enabled))
	fmt.Printf("  Check on Startup: %s\n", getUpdateStatusText(config.CheckOnStartup))
	fmt.Printf("  Auto Install: %s\n", getUpdateStatusText(config.AutoInstall))
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
	fmt.Printf("  download_directory: %s\n", um.GetDownloadDirectory())
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
		enabled := (value == "true" || value == "1" || value == "yes")
		if err := um.SetEnabled(enabled); err != nil {
			fmt.Printf("Error updating enabled setting: %v\n", err)
		} else {
			fmt.Printf("Updated enabled to %t\n", enabled)
		}
	case "check_on_startup":
		checkOnStartup := (value == "true" || value == "1" || value == "yes")
		config.CheckOnStartup = checkOnStartup
		if err := um.UpdateConfig(config); err != nil {
			fmt.Printf("Error updating check_on_startup: %v\n", err)
		} else {
			fmt.Printf("Updated check_on_startup to %t\n", checkOnStartup)
		}
	case "auto_install":
		autoInstall := (value == "true" || value == "1" || value == "yes")
		config.AutoInstall = autoInstall
		if err := um.UpdateConfig(config); err != nil {
			fmt.Printf("Error updating auto_install: %v\n", err)
		} else {
			fmt.Printf("Updated auto_install to %t\n", autoInstall)
		}
	case "channel":
		if err := um.SetChannel(value); err != nil {
			fmt.Printf("Error updating channel: %v\n", err)
		} else {
			fmt.Printf("Updated channel to %s\n", value)
		}
	case "check_interval":
		if value == "daily" || value == "weekly" || value == "monthly" {
			config.CheckInterval = value
			if err := um.UpdateConfig(config); err != nil {
				fmt.Printf("Error updating check_interval: %v\n", err)
			} else {
				fmt.Printf("Updated check_interval to %s\n", value)
			}
		} else {
			fmt.Printf("Invalid interval: %s (must be daily, weekly, or monthly)\n", value)
		}
	case "notification_level":
		if err := um.SetNotificationLevel(value); err != nil {
			fmt.Printf("Error updating notification_level: %v\n", err)
		} else {
			fmt.Printf("Updated notification_level to %s\n", value)
		}
	case "backup_before_update":
		backup := (value == "true" || value == "1" || value == "yes")
		config.BackupBeforeUpdate = backup
		if err := um.UpdateConfig(config); err != nil {
			fmt.Printf("Error updating backup_before_update: %v\n", err)
		} else {
			fmt.Printf("Updated backup_before_update to %t\n", backup)
		}
	case "allow_prerelease":
		allowPrerelease := (value == "true" || value == "1" || value == "yes")
		config.AllowPrerelease = allowPrerelease
		if err := um.UpdateConfig(config); err != nil {
			fmt.Printf("Error updating allow_prerelease: %v\n", err)
		} else {
			fmt.Printf("Updated allow_prerelease to %t\n", allowPrerelease)
		}
	case "github_repository":
		config.GitHubRepository = value
		if err := um.UpdateConfig(config); err != nil {
			fmt.Printf("Error updating github_repository: %v\n", err)
		} else {
			fmt.Printf("Updated github_repository to %s\n", value)
		}
	case "download_directory":
		if err := um.SetDownloadDirectory(value); err != nil {
			fmt.Printf("Error updating download_directory: %v\n", err)
		} else {
			fmt.Printf("Updated download_directory to %s\n", value)
		}
	default:
		fmt.Printf("Unknown setting: %s\n", setting)
		fmt.Println("Use ':update help' to see available settings")
	}
}

// showVersionInfo displays version information
func showVersionInfo(um *UpdateManager) {
	current := um.GetCurrentVersion()
	fmt.Printf("Current Version: %s\n", current)
	
	// Parse version to show additional details
	if ver, err := ParseVersion(current); err == nil {
		fmt.Printf("  Major: %d\n", ver.Major)
		fmt.Printf("  Minor: %d\n", ver.Minor)
		fmt.Printf("  Patch: %d\n", ver.Patch)
		if ver.Prerelease != "" {
			fmt.Printf("  Prerelease: %s\n", ver.Prerelease)
		}
		fmt.Printf("  Channel: %s\n", ver.GetChannel())
		fmt.Printf("  Is Stable: %t\n", ver.IsStable())
	}
}

// showUpdateHelp displays help for update commands
func showUpdateHelp() {
	fmt.Println("Update Commands")
	fmt.Println("===============")
	fmt.Println("  Status & Information:")
	fmt.Println("  :update                     - Show update status")
	fmt.Println("  :update status              - Show detailed status")
	fmt.Println("  :update check               - Check for updates manually")
	fmt.Println("  :update info                - Show comprehensive update info")
	fmt.Println("  :update version             - Show version information")
	fmt.Println("  :update rate-limit          - Show GitHub API rate limit status")
	fmt.Println()
	fmt.Println("  Download & Installation:")
	fmt.Println("  :update download <version>  - Download a specific update")
	fmt.Println("  :update install <file>      - Install from downloaded file")
	fmt.Println("  :update update <version>    - Download and install update")
	fmt.Println("  :update interactive         - Interactive update with choices")
	fmt.Println()
	fmt.Println("  Backup & Recovery:")
	fmt.Println("  :update backups             - List available backups")
	fmt.Println("  :update rollback            - Rollback to previous version")
	fmt.Println()
	fmt.Println("  Version Management:")
	fmt.Println("  :update skip [version]      - Skip current or specific version")
	fmt.Println("  :update postpone <duration> - Postpone current update")
	fmt.Println("  :update reminder            - Check for postponement reminders")
	fmt.Println()
	fmt.Println("  Scheduling:")
	fmt.Println("  :update schedule <ver> <time> - Schedule update for specific time")
	fmt.Println("  :update pending             - List pending scheduled updates")
	fmt.Println("  :update cancel <id>         - Cancel scheduled update")
	fmt.Println()
	fmt.Println("  History & Logging:")
	fmt.Println("  :update logs                - Show update history")
	fmt.Println("  :update logs --filter <type> - Filter history by type/status")
	fmt.Println("  :update logs --audit        - Generate audit trail")
	fmt.Println()
	fmt.Println("  Validation:")
	fmt.Println("  :update validate            - Run post-update validation")
	fmt.Println("  :update validate --tests    - List available validation tests")
	fmt.Println()
	fmt.Println("  Maintenance:")
	fmt.Println("  :update cleanup             - Clean old downloads and backups")
	fmt.Println("  :update cleanup downloads   - Clean old downloads only")
	fmt.Println("  :update cleanup backups     - Clean old backups only")
	fmt.Println("  :update cleanup stats       - Show download statistics")
	fmt.Println()
	fmt.Println("  Configuration:")
	fmt.Println("  :update config              - Show configuration")
	fmt.Println("  :update config <key> <val>  - Set configuration value")
	fmt.Println("  :update help                - Show this help")
	fmt.Println()
	fmt.Println("Configuration Settings:")
	fmt.Println("  enabled                     - Enable/disable update system")
	fmt.Println("  check_on_startup            - Check for updates on startup")
	fmt.Println("  auto_install                - Automatically install updates")
	fmt.Println("  channel                     - Release channel (stable/alpha/beta)")
	fmt.Println("  check_interval              - Check frequency (daily/weekly/monthly)")
	fmt.Println("  notification_level          - Notification level (silent/notify/prompt)")
	fmt.Println("  backup_before_update        - Backup before installing updates")
	fmt.Println("  allow_prerelease            - Allow prerelease versions")
	fmt.Println("  github_repository           - GitHub repository for updates")
	fmt.Println("  download_directory          - Directory for downloaded updates")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  :update config enabled true")
	fmt.Println("  :update config channel alpha")
	fmt.Println("  :update config auto_install false")
	fmt.Println("  :update config notification_level notify")
}

// Helper functions
func getUpdateStatusText(enabled bool) string {
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

// performUpdateCheck manually triggers an update check
func performUpdateCheck(um *UpdateManager) {
	fmt.Println("Checking for updates...")
	
	updateInfo, err := um.CheckForUpdates()
	if err != nil {
		fmt.Printf("Error checking for updates: %v\n", err)
		return
	}
	
	if updateInfo.HasUpdate {
		fmt.Printf("✅ Update Available!\n")
		fmt.Printf("   Current Version: %s\n", updateInfo.CurrentVersion)
		fmt.Printf("   Latest Version:  %s\n", updateInfo.LatestVersion)
		
		if updateInfo.IsPrerelease {
			fmt.Printf("   Type: Prerelease\n")
		}
		
		fmt.Printf("   Published: %s\n", updateInfo.PublishedAt.Format("2006-01-02 15:04"))
		
		if updateInfo.AssetSize > 0 {
			fmt.Printf("   Download Size: %s\n", formatFileSize(updateInfo.AssetSize))
		}
		
		if updateInfo.ReleaseNotes != "" {
			fmt.Printf("   Release Notes:\n")
			notes := updateInfo.ReleaseNotes
			if len(notes) > 200 {
				notes = notes[:197] + "..."
			}
			fmt.Printf("   %s\n", notes)
		}
		
		fmt.Printf("\n   Use ':update config' to manage update settings\n")
	} else {
		fmt.Printf("✅ You're up to date!\n")
		fmt.Printf("   Current Version: %s\n", updateInfo.CurrentVersion)
		fmt.Printf("   Latest Version:  %s\n", updateInfo.LatestVersion)
		fmt.Printf("   Channel: %s\n", um.GetChannel())
		
		// Show development build info if applicable
		if IsDevelopmentBuild() {
			fmt.Printf("   Build Type: Development\n")
			devStatus := GetDevelopmentStatus()
			if gitCommit, ok := devStatus["git_commit"].(string); ok && gitCommit != "unknown" {
				fmt.Printf("   Git Commit: %s\n", gitCommit)
			}
		}
	}
}

// showUpdateInfo displays comprehensive update information
func showUpdateInfo(um *UpdateManager) {
	fmt.Println("Update System Information:")
	fmt.Printf("  Current Version: %s\n", um.GetCurrentVersion())
	fmt.Printf("  Enabled: %t\n", um.IsEnabled())
	fmt.Printf("  Channel: %s\n", um.GetChannel())
	
	// Show development build status
	if IsDevelopmentBuild() {
		fmt.Printf("  Build Type: Development\n")
		devStatus := GetDevelopmentStatus()
		if gitCommit, ok := devStatus["git_commit"].(string); ok && gitCommit != "unknown" {
			fmt.Printf("  Git Commit: %s\n", gitCommit)
		}
		if buildDate, ok := devStatus["build_date"].(string); ok && buildDate != "unknown" {
			fmt.Printf("  Build Date: %s\n", buildDate)
		}
	} else {
		fmt.Printf("  Build Type: Release\n")
	}
	
	// Try to get available update info
	updateInfo, err := um.GetAvailableUpdate()
	if err != nil {
		fmt.Printf("  Update Check: Failed (%v)\n", err)
	} else {
		if updateInfo.HasUpdate {
			fmt.Printf("  Update Available: Yes (%s)\n", updateInfo.LatestVersion)
			fmt.Printf("  Update Type: %s\n", getUpdateType(updateInfo))
		} else {
			fmt.Printf("  Update Available: No\n")
		}
	}
	
	// Show rate limit status
	if rateLimitStatus := um.GetRateLimitStatus(); rateLimitStatus != nil {
		fmt.Printf("  API Rate Limit: %d/%d remaining\n", 
			rateLimitStatus["remaining"], rateLimitStatus["limit"])
	}
}

// showRateLimitStatus displays GitHub API rate limit information
func showRateLimitStatus(um *UpdateManager) {
	rateLimitStatus := um.GetRateLimitStatus()
	if rateLimitStatus == nil {
		fmt.Println("Rate limit information not available")
		return
	}
	
	fmt.Println("GitHub API Rate Limit Status:")
	fmt.Printf("  Limit: %d requests per hour\n", rateLimitStatus["limit"])
	fmt.Printf("  Remaining: %d requests\n", rateLimitStatus["remaining"])
	
	if resetTime, ok := rateLimitStatus["reset_time"].(time.Time); ok {
		fmt.Printf("  Reset Time: %s\n", resetTime.Format("2006-01-02 15:04:05"))
		
		if waitTime, ok := rateLimitStatus["wait_time"].(time.Duration); ok && waitTime > 0 {
			fmt.Printf("  Time Until Reset: %s\n", waitTime.Truncate(time.Second))
		}
	}
}

// Helper functions for new commands
func getUpdateType(updateInfo *UpdateInfo) string {
	if updateInfo.IsPrerelease {
		return "Prerelease"
	}
	return "Stable"
}

func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	
	units := []string{"B", "KB", "MB", "GB", "TB"}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}

// handleDownloadCommand handles the download subcommand
func handleDownloadCommand(um *UpdateManager, args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: :update download <version>")
		fmt.Println("Example: :update download v0.2.0-alpha")
		return
	}

	version := args[0]
	fmt.Printf("Downloading update %s...\n", version)
	
	downloadResult, err := um.DownloadUpdate(version)
	if err != nil {
		fmt.Printf("❌ Download failed: %v\n", err)
		return
	}

	fmt.Printf("✅ Download completed successfully!\n")
	fmt.Printf("   File: %s\n", downloadResult.FilePath)
	fmt.Printf("   Size: %s\n", formatFileSize(downloadResult.Size))
	fmt.Printf("   Verified: %t\n", downloadResult.Verified)
	fmt.Printf("   Download Time: %s\n", downloadResult.DownloadTime.Truncate(time.Millisecond))
	fmt.Printf("\nUse ':update install %s' to install this update\n", downloadResult.FilePath)
}

// handleInstallCommand handles the install subcommand
func handleInstallCommand(um *UpdateManager, args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: :update install <file_path>")
		fmt.Println("Example: :update install /path/to/downloaded/delta")
		return
	}

	filePath := args[0]
	
	// Create a dummy download result for the file
	downloadResult := &DownloadResult{
		FilePath: filePath,
		Verified: true, // Assume verified for manual install
	}
	
	fmt.Printf("Installing update from %s...\n", filePath)
	fmt.Println("⚠️  This will replace the current Delta CLI binary!")
	fmt.Println("A backup will be created automatically.")
	
	installResult, err := um.InstallUpdate(downloadResult)
	if err != nil {
		fmt.Printf("❌ Installation failed: %v\n", err)
		if installResult != nil && installResult.BackupPath != "" {
			fmt.Printf("Backup available at: %s\n", installResult.BackupPath)
			fmt.Printf("Use ':update rollback' to restore if needed\n")
		}
		return
	}

	fmt.Printf("✅ Installation completed successfully!\n")
	fmt.Printf("   Old Version: %s\n", installResult.OldVersion)
	fmt.Printf("   New Version: %s\n", installResult.NewVersion)
	fmt.Printf("   Backup Created: %s\n", installResult.BackupPath)
	fmt.Printf("   Installation Time: %s\n", installResult.InstallTime.Truncate(time.Millisecond))
	fmt.Printf("\n🔄 Please restart Delta CLI to use the new version\n")
}

// handleUpdateCommand handles the update subcommand (download + install)
func handleUpdateCommand(um *UpdateManager, args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: :update update <version>")
		fmt.Println("Example: :update update v0.2.0-alpha")
		return
	}

	version := args[0]
	fmt.Printf("Downloading and installing update %s...\n", version)
	fmt.Println("⚠️  This will replace the current Delta CLI binary!")
	fmt.Println("A backup will be created automatically.")
	
	installResult, err := um.DownloadAndInstallUpdate(version)
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
	fmt.Printf("   Backup Created: %s\n", installResult.BackupPath)
	fmt.Printf("   Installation Time: %s\n", installResult.InstallTime.Truncate(time.Millisecond))
	fmt.Printf("\n🔄 Please restart Delta CLI to use the new version\n")
}

// handleRollbackCommand handles the rollback subcommand
func handleRollbackCommand(um *UpdateManager) {
	fmt.Println("Rolling back to previous version...")
	fmt.Println("⚠️  This will replace the current Delta CLI binary with the most recent backup!")
	
	err := um.RollbackToPreviousVersion()
	if err != nil {
		fmt.Printf("❌ Rollback failed: %v\n", err)
		return
	}

	fmt.Printf("✅ Rollback completed successfully!\n")
	fmt.Printf("\n🔄 Please restart Delta CLI to use the restored version\n")
}

// handleBackupsCommand handles the backups subcommand
func handleBackupsCommand(um *UpdateManager) {
	backups, err := um.GetBackupInfo()
	if err != nil {
		fmt.Printf("❌ Failed to get backup information: %v\n", err)
		return
	}

	if len(backups) == 0 {
		fmt.Println("No backups available")
		return
	}

	fmt.Printf("Available Backups (%d total):\n", len(backups))
	fmt.Println("================================")
	
	// Sort backups by time (newest first)
	for i := 0; i < len(backups)-1; i++ {
		for j := i + 1; j < len(backups); j++ {
			if backups[i].BackupTime.Before(backups[j].BackupTime) {
				backups[i], backups[j] = backups[j], backups[i]
			}
		}
	}

	for i, backup := range backups {
		fmt.Printf("%d. Version: %s\n", i+1, backup.Version)
		fmt.Printf("   File: %s\n", backup.BackupPath)
		fmt.Printf("   Size: %s\n", formatFileSize(backup.Size))
		fmt.Printf("   Created: %s\n", backup.BackupTime.Format("2006-01-02 15:04:05"))
		fmt.Println()
	}
	
	fmt.Printf("Use ':update rollback' to restore the most recent backup\n")
}

// handleCleanupCommand handles the cleanup subcommand
func handleCleanupCommand(um *UpdateManager, args []string) {
	if len(args) == 0 {
		// Default cleanup - both downloads and backups
		fmt.Println("Cleaning up old downloads and backups...")
		
		if err := um.CleanupOldDownloads(3); err != nil {
			fmt.Printf("Warning: Failed to cleanup downloads: %v\n", err)
		} else {
			fmt.Println("✅ Old downloads cleaned up (kept 3 most recent)")
		}
		
		if err := um.CleanupOldBackups(5); err != nil {
			fmt.Printf("Warning: Failed to cleanup backups: %v\n", err)
		} else {
			fmt.Println("✅ Old backups cleaned up (kept 5 most recent)")
		}
		return
	}

	switch args[0] {
	case "downloads":
		keepCount := 3
		if len(args) > 1 {
			if count, err := parseIntSafely(args[1]); err == nil && count > 0 {
				keepCount = count
			}
		}
		
		if err := um.CleanupOldDownloads(keepCount); err != nil {
			fmt.Printf("❌ Failed to cleanup downloads: %v\n", err)
		} else {
			fmt.Printf("✅ Old downloads cleaned up (kept %d most recent)\n", keepCount)
		}

	case "backups":
		keepCount := 5
		if len(args) > 1 {
			if count, err := parseIntSafely(args[1]); err == nil && count > 0 {
				keepCount = count
			}
		}
		
		if err := um.CleanupOldBackups(keepCount); err != nil {
			fmt.Printf("❌ Failed to cleanup backups: %v\n", err)
		} else {
			fmt.Printf("✅ Old backups cleaned up (kept %d most recent)\n", keepCount)
		}

	case "stats":
		stats := um.GetDownloadStats()
		fmt.Println("Download Statistics:")
		fmt.Printf("  Directory: %s\n", stats["download_directory"])
		fmt.Printf("  File Count: %d\n", stats["file_count"])
		fmt.Printf("  Total Size: %s\n", stats["total_size_formatted"])

	default:
		fmt.Printf("Unknown cleanup target: %s\n", args[0])
		fmt.Println("Usage: :update cleanup [downloads|backups|stats] [keep_count]")
	}
}

// parseIntSafely safely parses an integer with error handling
func parseIntSafely(s string) (int, error) {
	result := 0
	for _, char := range s {
		if char < '0' || char > '9' {
			return 0, fmt.Errorf("invalid integer: %s", s)
		}
		result = result*10 + int(char-'0')
	}
	return result, nil
}

// handleInteractiveUpdateCommand launches interactive update interface
func handleInteractiveUpdateCommand(um *UpdateManager) {
	fmt.Println("Checking for updates...")
	
	updateInfo, err := um.CheckForUpdates()
	if err != nil {
		fmt.Printf("❌ Error checking for updates: %v\n", err)
		return
	}
	
	if !updateInfo.HasUpdate {
		fmt.Printf("✅ You're up to date!\n")
		fmt.Printf("   Current Version: %s\n", updateInfo.CurrentVersion)
		fmt.Printf("   Latest Version:  %s\n", updateInfo.LatestVersion)
		return
	}
	
	// Launch interactive UI
	ui := NewUpdateUI()
	options := &UpdatePromptOptions{
		ShowChangelog:   true,
		AllowPostpone:   true,
		AllowSkip:       true,
		AutoConfirm:     false,
		PostponeOptions: []string{"1 hour", "4 hours", "1 day", "1 week"},
		DefaultChoice:   UpdateChoiceCancel,
	}
	
	choice := ui.PromptForUpdate(updateInfo, options)
	
	switch choice {
	case UpdateChoiceInstall:
		// Handle installation
		handleUpdateCommand(um, []string{updateInfo.LatestVersion})
	case UpdateChoiceSkip:
		fmt.Printf("✅ Version %s will be skipped.\n", updateInfo.LatestVersion)
	case UpdateChoicePostpone:
		fmt.Println("Update postponed.")
	case UpdateChoiceCancel:
		fmt.Println("Update cancelled.")
	}
}

// handleSkipCommand manages version skipping
func handleSkipCommand(um *UpdateManager, args []string) {
	config := um.GetConfig()
	
	if len(args) == 0 {
		// Skip current available version
		updateInfo, err := um.GetAvailableUpdate()
		if err != nil {
			fmt.Printf("❌ Error checking for updates: %v\n", err)
			return
		}
		
		if !updateInfo.HasUpdate {
			fmt.Println("No updates available to skip.")
			return
		}
		
		config.SkipVersion = updateInfo.LatestVersion
		err = um.UpdateConfig(config)
		if err != nil {
			fmt.Printf("❌ Failed to skip version: %v\n", err)
			return
		}
		
		fmt.Printf("✅ Version %s will be skipped.\n", updateInfo.LatestVersion)
	} else if args[0] == "clear" || args[0] == "reset" {
		// Clear skip version
		config.SkipVersion = ""
		err := um.UpdateConfig(config)
		if err != nil {
			fmt.Printf("❌ Failed to clear skip version: %v\n", err)
			return
		}
		fmt.Println("✅ Skip version cleared. All updates will be shown.")
	} else {
		// Skip specific version
		version := args[0]
		config.SkipVersion = version
		err := um.UpdateConfig(config)
		if err != nil {
			fmt.Printf("❌ Failed to skip version: %v\n", err)
			return
		}
		fmt.Printf("✅ Version %s will be skipped.\n", version)
	}
}

// handlePostponeCommand manages update postponement
func handlePostponeCommand(um *UpdateManager, args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: :update postpone <duration>")
		fmt.Println("Examples:")
		fmt.Println("  :update postpone 1h      - Postpone for 1 hour")
		fmt.Println("  :update postpone 4h      - Postpone for 4 hours") 
		fmt.Println("  :update postpone 1d      - Postpone for 1 day")
		fmt.Println("  :update postpone 1w      - Postpone for 1 week")
		fmt.Println("  :update postpone clear   - Clear postponement")
		return
	}
	
	if args[0] == "clear" || args[0] == "reset" {
		// Clear postponement
		config := um.GetConfig()
		config.PostponedVersion = ""
		config.PostponedUntil = ""
		err := um.UpdateConfig(config)
		if err != nil {
			fmt.Printf("❌ Failed to clear postponement: %v\n", err)
			return
		}
		fmt.Println("✅ Postponement cleared.")
		return
	}
	
	// Check for available update
	updateInfo, err := um.GetAvailableUpdate()
	if err != nil {
		fmt.Printf("❌ Error checking for updates: %v\n", err)
		return
	}
	
	if !updateInfo.HasUpdate {
		fmt.Println("No updates available to postpone.")
		return
	}
	
	// Parse duration
	duration := args[0]
	var postponeUntil time.Time
	
	switch duration {
	case "1h", "1 hour":
		postponeUntil = time.Now().Add(1 * time.Hour)
	case "4h", "4 hours":
		postponeUntil = time.Now().Add(4 * time.Hour)
	case "1d", "1 day":
		postponeUntil = time.Now().Add(24 * time.Hour)
	case "1w", "1 week":
		postponeUntil = time.Now().Add(7 * 24 * time.Hour)
	default:
		fmt.Printf("❌ Invalid duration: %s\n", duration)
		fmt.Println("Valid options: 1h, 4h, 1d, 1w")
		return
	}
	
	// Set postponement
	config := um.GetConfig()
	config.PostponedVersion = updateInfo.LatestVersion
	config.PostponedUntil = postponeUntil.Format(time.RFC3339)
	
	err = um.UpdateConfig(config)
	if err != nil {
		fmt.Printf("❌ Failed to postpone update: %v\n", err)
		return
	}
	
	fmt.Printf("✅ Update to %s postponed until %s\n", 
		updateInfo.LatestVersion, postponeUntil.Format("2006-01-02 15:04"))
}

// handleReminderCommand checks and shows postponement reminders
func handleReminderCommand(um *UpdateManager) {
	checker := GetUpdateChecker()
	if checker == nil {
		fmt.Println("❌ Update checker not available")
		return
	}
	
	checker.CheckPostponementReminders()
	
	config := um.GetConfig()
	if config.PostponedVersion == "" {
		fmt.Println("No postponed updates.")
		return
	}
	
	if config.PostponedUntil == "" {
		fmt.Printf("Version %s is postponed (no expiration time set).\n", config.PostponedVersion)
		return
	}
	
	postponedUntil, err := time.Parse(time.RFC3339, config.PostponedUntil)
	if err != nil {
		fmt.Printf("Version %s is postponed (invalid expiration time).\n", config.PostponedVersion)
		return
	}
	
	if time.Now().After(postponedUntil) {
		fmt.Printf("⏰ Postponement for version %s has expired!\n", config.PostponedVersion)
		fmt.Println("Use ':update check' to install or postpone again.")
	} else {
		timeLeft := postponedUntil.Sub(time.Now())
		fmt.Printf("Version %s is postponed for %s more.\n", 
			config.PostponedVersion, timeLeft.Truncate(time.Minute))
		fmt.Printf("Postponement expires: %s\n", postponedUntil.Format("2006-01-02 15:04"))
	}
}

// handleScheduleCommand manages update scheduling
func handleScheduleCommand(um *UpdateManager, args []string) {
	if len(args) < 2 {
		fmt.Println("Usage: :update schedule <version> <time>")
		fmt.Println("Examples:")
		fmt.Println("  :update schedule v0.5.0 '2024-12-25 10:00'")
		fmt.Println("  :update schedule latest 'tomorrow 9:00'")
		fmt.Println("  :update schedule v0.5.0 '+2h'")
		fmt.Println("  :update schedule v0.5.0 '@daily'")
		return
	}

	version := args[0]
	timeStr := args[1]

	// Parse the time string
	scheduledTime, err := parseTimeString(timeStr)
	if err != nil {
		fmt.Printf("❌ Invalid time format: %v\n", err)
		fmt.Println("Supported formats:")
		fmt.Println("  - '2024-12-25 10:00'")
		fmt.Println("  - 'tomorrow 9:00'")
		fmt.Println("  - '+2h', '+30m', '+1d'")
		fmt.Println("  - '@daily', '@weekly', '@monthly'")
		return
	}

	// Get or create scheduler
	scheduler := GetUpdateScheduler()
	if scheduler == nil {
		fmt.Println("❌ Update scheduler not available")
		return
	}

	// Start scheduler if not running
	if !scheduler.IsRunning() {
		if err := scheduler.Start(); err != nil {
			fmt.Printf("❌ Failed to start scheduler: %v\n", err)
			return
		}
	}

	// Create schedule options
	options := &ScheduleOptions{
		AutoConfirm: false,
		MaxRetries:  3,
	}

	// Check if it's a cron expression
	if timeStr[0] == '@' {
		options.CronExpression = timeStr
		options.IsRecurring = true
	}

	// Schedule the update
	scheduledUpdate, err := scheduler.ScheduleUpdate(version, scheduledTime, options)
	if err != nil {
		fmt.Printf("❌ Failed to schedule update: %v\n", err)
		return
	}

	fmt.Printf("✅ Update to %s scheduled for %s\n", version, scheduledTime.Format("2006-01-02 15:04"))
	fmt.Printf("   Schedule ID: %s\n", scheduledUpdate.ID)
	
	if scheduledUpdate.IsRecurring {
		fmt.Printf("   Recurring: %s\n", scheduledUpdate.CronExpression)
	}
}

// handlePendingCommand shows pending scheduled updates
func handlePendingCommand(um *UpdateManager) {
	scheduler := GetUpdateScheduler()
	if scheduler == nil {
		fmt.Println("❌ Update scheduler not available")
		return
	}

	pending := scheduler.GetPendingUpdates()
	if len(pending) == 0 {
		fmt.Println("No pending scheduled updates.")
		return
	}

	fmt.Printf("Pending Scheduled Updates (%d total):\n", len(pending))
	fmt.Println("======================================")

	for i, update := range pending {
		fmt.Printf("%d. ID: %s\n", i+1, update.ID)
		fmt.Printf("   Version: %s\n", update.Version)
		fmt.Printf("   Scheduled: %s\n", update.ScheduledTime.Format("2006-01-02 15:04"))
		
		timeUntil := update.ScheduledTime.Sub(time.Now())
		if timeUntil > 0 {
			fmt.Printf("   Time Until: %s\n", timeUntil.Truncate(time.Minute))
		} else {
			fmt.Printf("   Status: %s\n", "Due now")
		}
		
		if update.IsRecurring {
			fmt.Printf("   Recurring: %s\n", update.CronExpression)
		}
		
		fmt.Printf("   Retries: %d/%d\n", update.RetryCount, update.MaxRetries)
		fmt.Println()
	}

	fmt.Printf("Use ':update cancel <id>' to cancel a scheduled update\n")
}

// handleCancelCommand cancels a scheduled update
func handleCancelCommand(um *UpdateManager, args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: :update cancel <schedule_id>")
		fmt.Println("Use ':update pending' to see scheduled updates")
		return
	}

	scheduleID := args[0]
	
	scheduler := GetUpdateScheduler()
	if scheduler == nil {
		fmt.Println("❌ Update scheduler not available")
		return
	}

	err := scheduler.CancelScheduledUpdate(scheduleID)
	if err != nil {
		fmt.Printf("❌ Failed to cancel scheduled update: %v\n", err)
		return
	}

	fmt.Printf("✅ Scheduled update %s has been cancelled.\n", scheduleID)
}

// parseTimeString parses various time string formats
func parseTimeString(timeStr string) (time.Time, error) {
	now := time.Now()

	// Handle cron expressions
	if timeStr[0] == '@' {
		switch timeStr {
		case "@daily":
			return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location()), nil
		case "@weekly":
			daysUntilSunday := (7 - int(now.Weekday())) % 7
			if daysUntilSunday == 0 {
				daysUntilSunday = 7
			}
			return time.Date(now.Year(), now.Month(), now.Day()+daysUntilSunday, 0, 0, 0, 0, now.Location()), nil
		case "@monthly":
			return time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location()), nil
		default:
			return time.Time{}, fmt.Errorf("unsupported cron expression: %s", timeStr)
		}
	}

	// Handle relative time (+2h, +30m, +1d)
	if timeStr[0] == '+' {
		duration, err := time.ParseDuration(timeStr[1:])
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid duration: %s", timeStr)
		}
		return now.Add(duration), nil
	}

	// Handle absolute time formats
	formats := []string{
		"2006-01-02 15:04",
		"2006-01-02 15:04:05",
		"01-02 15:04",
		"15:04",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			// If no year specified, use current year
			if format == "01-02 15:04" {
				t = t.AddDate(now.Year(), 0, 0)
			}
			// If no date specified, use tomorrow if time has passed today
			if format == "15:04" {
				t = time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, now.Location())
				if t.Before(now) {
					t = t.AddDate(0, 0, 1)
				}
			}
			return t, nil
		}
	}

	// Handle special keywords
	switch timeStr {
	case "tomorrow":
		return time.Date(now.Year(), now.Month(), now.Day()+1, 9, 0, 0, 0, now.Location()), nil
	case "next week":
		return time.Date(now.Year(), now.Month(), now.Day()+7, 9, 0, 0, 0, now.Location()), nil
	case "next month":
		return time.Date(now.Year(), now.Month()+1, now.Day(), 9, 0, 0, 0, now.Location()), nil
	}

	return time.Time{}, fmt.Errorf("unrecognized time format: %s", timeStr)
}

// handleLogsCommand manages update history and logging
func handleLogsCommand(um *UpdateManager, args []string) {
	history := GetUpdateHistory()
	if history == nil {
		fmt.Println("❌ Update history not available")
		return
	}

	// Parse arguments
	var filter *HistoryFilter
	showAudit := false
	showMetrics := false
	format := "table"

	for i, arg := range args {
		switch arg {
		case "--filter":
			if i+1 < len(args) {
				filter = parseHistoryFilter(args[i+1])
				i++ // Skip next arg as it's the filter value
			}
		case "--audit":
			showAudit = true
		case "--metrics":
			showMetrics = true
		case "--format":
			if i+1 < len(args) {
				format = args[i+1]
				i++
			}
		}
	}

	if showAudit {
		handleAuditCommand(history, format)
		return
	}

	if showMetrics {
		handleMetricsCommand(history)
		return
	}

	// Show update history
	records := history.GetRecords(filter)
	if len(records) == 0 {
		fmt.Println("No update records found.")
		return
	}

	fmt.Printf("Update History (%d records):\n", len(records))
	fmt.Println("=============================")

	for i, record := range records {
		if i >= 20 { // Limit to last 20 records
			fmt.Printf("... and %d more records (use --filter to narrow results)\n", len(records)-i)
			break
		}

		fmt.Printf("%d. %s\n", i+1, record.ID)
		fmt.Printf("   Timestamp: %s\n", record.Timestamp.Format("2006-01-02 15:04:05"))
		fmt.Printf("   Type: %s\n", record.Type)
		fmt.Printf("   Version: %s → %s\n", record.FromVersion, record.ToVersion)
		
		statusColor := "green"
		if record.Status == UpdateStatusFailed {
			statusColor = "red"
		} else if record.Status == UpdateStatusPartial {
			statusColor = "yellow"
		}
		fmt.Printf("   Status: %s\n", colorizeText(string(record.Status), statusColor))
		
		if record.Duration > 0 {
			fmt.Printf("   Duration: %s\n", record.Duration.Truncate(time.Second))
		}
		
		if record.DownloadTime > 0 {
			fmt.Printf("   Download Time: %s\n", record.DownloadTime.Truncate(time.Second))
		}
		
		if record.InstallTime > 0 {
			fmt.Printf("   Install Time: %s\n", record.InstallTime.Truncate(time.Second))
		}
		
		if record.Channel != "" {
			fmt.Printf("   Channel: %s\n", record.Channel)
		}
		
		if record.TriggerMethod != "" {
			fmt.Printf("   Triggered By: %s\n", record.TriggerMethod)
		}
		
		if record.ErrorMessage != "" {
			fmt.Printf("   Error: %s\n", record.ErrorMessage)
		}
		
		fmt.Println()
	}

	// Show summary metrics
	metrics := history.GetMetrics()
	fmt.Println("Summary:")
	fmt.Printf("  Total Updates: %d\n", metrics.TotalUpdates)
	fmt.Printf("  Success Rate: %.1f%%\n", metrics.SuccessRate)
	if metrics.AverageDownloadTime > 0 {
		fmt.Printf("  Avg Download Time: %s\n", metrics.AverageDownloadTime.Truncate(time.Second))
	}
	if metrics.AverageInstallTime > 0 {
		fmt.Printf("  Avg Install Time: %s\n", metrics.AverageInstallTime.Truncate(time.Second))
	}
}

// parseHistoryFilter parses filter arguments
func parseHistoryFilter(filterStr string) *HistoryFilter {
	filter := &HistoryFilter{}

	switch filterStr {
	case "success", "successful":
		status := UpdateStatusSuccess
		filter.Status = &status
	case "failed", "failure":
		status := UpdateStatusFailed
		filter.Status = &status
	case "manual":
		updateType := UpdateTypeManual
		filter.Type = &updateType
	case "scheduled":
		updateType := UpdateTypeScheduled
		filter.Type = &updateType
	case "automatic", "auto":
		updateType := UpdateTypeAutomatic
		filter.Type = &updateType
	case "rollback":
		updateType := UpdateTypeRollback
		filter.Type = &updateType
	case "alpha":
		filter.Channel = "alpha"
	case "beta":
		filter.Channel = "beta"
	case "stable":
		filter.Channel = "stable"
	}

	return filter
}

// handleAuditCommand generates audit trails
func handleAuditCommand(history *UpdateHistory, format string) {
	var auditFormat AuditFormat
	switch format {
	case "json":
		auditFormat = AuditFormatJSON
	case "csv":
		auditFormat = AuditFormatCSV
	case "text", "txt":
		auditFormat = AuditFormatText
	default:
		auditFormat = AuditFormatText
	}

	audit, err := history.GetAuditTrail(auditFormat)
	if err != nil {
		fmt.Printf("❌ Failed to generate audit trail: %v\n", err)
		return
	}

	fmt.Printf("Update Audit Trail (%s format):\n", format)
	fmt.Println("===============================")
	fmt.Println(audit)
}

// handleMetricsCommand shows detailed metrics
func handleMetricsCommand(history *UpdateHistory) {
	metrics := history.GetMetrics()
	
	fmt.Println("Update Metrics:")
	fmt.Println("===============")
	fmt.Printf("Total Updates: %d\n", metrics.TotalUpdates)
	fmt.Printf("Successful Updates: %d\n", metrics.SuccessfulUpdates)
	fmt.Printf("Failed Updates: %d\n", metrics.FailedUpdates)
	fmt.Printf("Success Rate: %.2f%%\n", metrics.SuccessRate)
	
	if metrics.AverageDownloadTime > 0 {
		fmt.Printf("Average Download Time: %s\n", metrics.AverageDownloadTime.Truncate(time.Millisecond))
	}
	
	if metrics.AverageInstallTime > 0 {
		fmt.Printf("Average Install Time: %s\n", metrics.AverageInstallTime.Truncate(time.Millisecond))
	}
	
	if metrics.TotalDownloadSize > 0 {
		fmt.Printf("Total Download Size: %s\n", formatFileSize(metrics.TotalDownloadSize))
	}
	
	if !metrics.FirstUpdateTime.IsZero() {
		fmt.Printf("First Update: %s\n", metrics.FirstUpdateTime.Format("2006-01-02 15:04:05"))
	}
	
	if !metrics.LastUpdateTime.IsZero() {
		fmt.Printf("Last Update: %s\n", metrics.LastUpdateTime.Format("2006-01-02 15:04:05"))
	}
}

// colorizeText adds color to text for better readability
func colorizeText(text, color string) string {
	colors := map[string]string{
		"red":    "\033[31m",
		"green":  "\033[32m",
		"yellow": "\033[33m",
		"blue":   "\033[34m",
		"cyan":   "\033[36m",
		"reset":  "\033[0m",
	}
	
	if colorCode, exists := colors[color]; exists {
		return colorCode + text + colors["reset"]
	}
	
	return text
}

// handleValidateCommand manages post-update validation
func handleValidateCommand(um *UpdateManager, args []string) {
	validator := GetUpdateValidator()
	if validator == nil {
		fmt.Println("❌ Update validator not available")
		return
	}

	// Parse arguments
	showTests := false
	runValidation := true

	for _, arg := range args {
		switch arg {
		case "--tests", "--list":
			showTests = true
			runValidation = false
		case "--enable":
			// TODO: Implement enable/disable specific tests
		case "--disable":
			// TODO: Implement enable/disable specific tests
		}
	}

	if showTests {
		handleValidationTestsCommand(validator)
		return
	}

	if runValidation {
		currentVersion := um.GetCurrentVersion()
		suite, err := validator.RunValidation(currentVersion, "manual")
		
		if err != nil {
			fmt.Printf("❌ Validation failed: %v\n", err)
			return
		}

		// Show detailed results
		showValidationResults(suite)
	}
}

// handleValidationTestsCommand shows available validation tests
func handleValidationTestsCommand(validator *UpdateValidator) {
	tests := validator.GetTests()
	config := validator.GetConfig()

	fmt.Println("Available Validation Tests:")
	fmt.Println("===========================")
	fmt.Printf("Validation Enabled: %t\n", config.Enabled)
	fmt.Printf("Auto Rollback: %t\n", config.AutoRollbackOnFailure)
	fmt.Printf("Validation Timeout: %s\n", config.ValidationTimeout)
	fmt.Println()

	for i, test := range tests {
		status := "Disabled"
		statusColor := "red"
		if test.Enabled {
			status = "Enabled"
			statusColor = "green"
		}

		critical := ""
		if test.Critical {
			critical = colorizeText(" [CRITICAL]", "yellow")
		}

		fmt.Printf("%d. %s%s\n", i+1, test.Name, critical)
		fmt.Printf("   Description: %s\n", test.Description)
		fmt.Printf("   Status: %s\n", colorizeText(status, statusColor))
		fmt.Printf("   Timeout: %s\n", test.Timeout)
		fmt.Println()
	}

	fmt.Println("Legend:")
	fmt.Printf("  %s - Test failure triggers automatic rollback\n", colorizeText("[CRITICAL]", "yellow"))
	fmt.Printf("  %s - Test is currently enabled\n", colorizeText("Enabled", "green"))
	fmt.Printf("  %s - Test is currently disabled\n", colorizeText("Disabled", "red"))
}

// showValidationResults displays detailed validation results
func showValidationResults(suite *ValidationSuite) {
	fmt.Printf("\nValidation Results for %s:\n", suite.Version)
	fmt.Println("=========================")
	fmt.Printf("Suite ID: %s\n", suite.ID)
	fmt.Printf("Started: %s\n", suite.StartTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("Duration: %s\n", suite.Duration.Truncate(time.Second))
	fmt.Printf("Overall Status: %s\n", colorizeValidationStatus(suite.OverallStatus))
	fmt.Println()

	fmt.Printf("Test Summary:\n")
	fmt.Printf("  Total: %d\n", suite.TotalTests)
	fmt.Printf("  Passed: %s\n", colorizeText(fmt.Sprintf("%d", suite.PassedTests), "green"))
	fmt.Printf("  Failed: %s\n", colorizeText(fmt.Sprintf("%d", suite.FailedTests), "red"))
	fmt.Printf("  Skipped: %s\n", colorizeText(fmt.Sprintf("%d", suite.SkippedTests), "yellow"))
	fmt.Println()

	if len(suite.Results) > 0 {
		fmt.Println("Detailed Results:")
		for i, result := range suite.Results {
			statusColor := "green"
			statusIcon := "✅"
			if result.Status == "failed" {
				statusColor = "red"
				statusIcon = "❌"
			} else if result.Status == "skipped" {
				statusColor = "yellow"
				statusIcon = "⏭️ "
			}

			fmt.Printf("%d. %s %s %s (%s)\n", 
				i+1, 
				statusIcon,
				result.TestName, 
				colorizeText(strings.ToUpper(result.Status), statusColor),
				result.Duration.Truncate(time.Millisecond),
			)
			
			if result.ErrorMsg != "" {
				fmt.Printf("   Error: %s\n", result.ErrorMsg)
			}
			
			if result.Details != nil {
				if details, ok := result.Details.(map[string]interface{}); ok {
					for key, value := range details {
						fmt.Printf("   %s: %v\n", key, value)
					}
				}
			}
		}
	}
}

// colorizeValidationStatus applies appropriate color to validation status
func colorizeValidationStatus(status ValidationStatus) string {
	switch status {
	case ValidationStatusPassed:
		return colorizeText(string(status), "green")
	case ValidationStatusFailed:
		return colorizeText(string(status), "red")
	case ValidationStatusPartial:
		return colorizeText(string(status), "yellow")
	case ValidationStatusSkipped:
		return colorizeText(string(status), "cyan")
	default:
		return string(status)
	}
}