package main

import (
	"fmt"
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