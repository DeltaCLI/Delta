package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// HandlePatternCommand processes pattern-related commands
func HandlePatternCommand(args []string) bool {
	// Get the pattern update manager
	pm := GetPatternUpdateManager()
	if pm == nil {
		fmt.Println("Pattern update manager is not available")
		return true
	}

	// If no arguments, show pattern status
	if len(args) == 0 {
		showPatternStatus(pm)
		return true
	}

	// Handle subcommands
	cmd := args[0]
	switch cmd {
	case "enable":
		err := pm.Enable()
		if err != nil {
			fmt.Printf("Error enabling pattern updates: %v\n", err)
			return true
		}
		fmt.Println("Pattern updates enabled")
		return true

	case "disable":
		err := pm.Disable()
		if err != nil {
			fmt.Printf("Error disabling pattern updates: %v\n", err)
			return true
		}
		fmt.Println("Pattern updates disabled")
		return true

	case "auto":
		if len(args) < 2 {
			fmt.Println("Please specify 'on' or 'off'")
			return true
		}
		return handleAutoUpdate(pm, args[1])

	case "update":
		return handlePatternUpdate(pm, args[1:])

	case "versions":
		showPatternVersions(pm)
		return true

	case "list":
		listPatterns(pm)
		return true

	case "check":
		checkForUpdates(pm)
		return true

	case "interval":
		if len(args) < 2 {
			fmt.Println("Please specify an interval in hours")
			fmt.Println("Usage: :pattern interval <hours>")
			return true
		}
		return setUpdateInterval(pm, args[1])

	case "status":
		showPatternStatus(pm)
		return true

	case "stats":
		showPatternStats(pm)
		return true

	case "help":
		showPatternCommandHelp()
		return true

	default:
		fmt.Printf("Unknown pattern command: %s\n", cmd)
		fmt.Println("Use ':pattern help' for a list of available commands")
		return true
	}
}

// showPatternStatus displays the current status of pattern updates
func showPatternStatus(pm *PatternUpdateManager) {
	fmt.Println("Pattern Update Status")
	fmt.Println("====================")

	if !pm.isInitialized {
		fmt.Println("Pattern update manager is not initialized")
		return
	}

	stats, err := pm.GetStats()
	if err != nil {
		fmt.Printf("Error getting pattern stats: %v\n", err)
		return
	}

	enabled := stats["enabled"].(bool)
	autoUpdate := stats["auto_update"].(bool)
	lastCheck := stats["last_update_check"].(string)

	fmt.Printf("Enabled: %v\n", enabled)
	fmt.Printf("Auto-update: %v\n", autoUpdate)

	// Parse and format the last check time
	lastCheckTime, err := time.Parse(time.RFC3339, lastCheck)
	if err == nil {
		fmt.Printf("Last update check: %s\n", lastCheckTime.Format("2006-01-02 15:04:05"))
		fmt.Printf("Update interval: %d hours\n", stats["update_check_interval"].(int))
	}

	// Show current versions
	fmt.Println("\nCurrent Versions:")
	fmt.Printf("  Error Patterns: %s\n", stats["error_patterns_version"])
	fmt.Printf("  Common Commands: %s\n", stats["commands_version"])

	// Check if update is available (in the background to avoid blocking)
	if enabled {
		fmt.Println("\nChecking for updates...")
		go func() {
			available, err := pm.CheckForUpdates()
			if err != nil {
				fmt.Printf("Error checking for updates: %v\n", err)
				return
			}

			if available {
				fmt.Println("✓ Updates are available. Run ':pattern update' to download.")
			} else {
				fmt.Println("✓ Patterns are up to date.")
			}
		}()
	}
}

// showPatternStats displays detailed statistics about patterns
func showPatternStats(pm *PatternUpdateManager) {
	fmt.Println("Pattern Statistics")
	fmt.Println("=================")

	stats, err := pm.GetStats()
	if err != nil {
		fmt.Printf("Error getting pattern stats: %v\n", err)
		return
	}

	// Show pattern counts if available
	if count, ok := stats["error_patterns_count"]; ok {
		fmt.Printf("Error Patterns: %d patterns\n", count)
	} else {
		fmt.Println("Error Patterns: Not available")
	}

	if count, ok := stats["commands_count"]; ok {
		fmt.Printf("Common Commands: %d commands\n", count)
	} else {
		fmt.Println("Common Commands: Not available")
	}

	// Show file sizes if available
	if size, ok := stats["error_patterns_size"]; ok {
		fmt.Printf("Error Patterns Size: %.2f KB\n", float64(size.(int64))/1024)
	}

	if size, ok := stats["commands_size"]; ok {
		fmt.Printf("Common Commands Size: %.2f KB\n", float64(size.(int64))/1024)
	}

	// Show configuration
	fmt.Println("\nConfiguration:")
	fmt.Printf("  API URL: %s\n", stats["api_base_url"])
	fmt.Printf("  Update interval: %d hours\n", stats["update_check_interval"])

	// Parse and format the last check time
	lastCheck := stats["last_update_check"].(string)
	lastCheckTime, err := time.Parse(time.RFC3339, lastCheck)
	if err == nil {
		fmt.Printf("  Last update check: %s\n", lastCheckTime.Format("2006-01-02 15:04:05"))
		timeSince := time.Since(lastCheckTime)
		fmt.Printf("  Time since last check: %s\n", formatDuration(timeSince))
	}

	// Show next scheduled check
	if err == nil {
		interval := time.Duration(stats["update_check_interval"].(int)) * time.Hour
		nextCheck := lastCheckTime.Add(interval)
		if time.Now().After(nextCheck) {
			fmt.Println("  Next scheduled check: Due now")
		} else {
			timeUntil := time.Until(nextCheck)
			fmt.Printf("  Next scheduled check: %s (in %s)\n",
				nextCheck.Format("2006-01-02 15:04:05"),
				formatDuration(timeUntil))
		}
	}
}

// handleAutoUpdate enables or disables automatic updates
func handleAutoUpdate(pm *PatternUpdateManager, arg string) bool {
	switch strings.ToLower(arg) {
	case "on", "enable", "true", "yes", "1":
		err := pm.EnableAutoUpdate()
		if err != nil {
			fmt.Printf("Error enabling auto-update: %v\n", err)
			return true
		}
		fmt.Println("Automatic pattern updates enabled")
		return true

	case "off", "disable", "false", "no", "0":
		err := pm.DisableAutoUpdate()
		if err != nil {
			fmt.Printf("Error disabling auto-update: %v\n", err)
			return true
		}
		fmt.Println("Automatic pattern updates disabled")
		return true

	default:
		fmt.Println("Invalid auto-update option. Use 'on' or 'off'.")
		return true
	}
}

// handlePatternUpdate updates pattern files
func handlePatternUpdate(pm *PatternUpdateManager, args []string) bool {
	// If force flag is specified, download updates without checking
	force := false
	if len(args) > 0 && (args[0] == "--force" || args[0] == "-f") {
		force = true
	}

	if !force {
		// Check if updates are available
		fmt.Println("Checking for updates...")
		available, err := pm.CheckForUpdates()
		if err != nil {
			fmt.Printf("Error checking for updates: %v\n", err)
			return true
		}

		if !available {
			fmt.Println("Patterns are already up to date. Use '--force' to update anyway.")
			return true
		}
	}

	// Download updates
	fmt.Println("Downloading pattern updates...")
	err := pm.DownloadUpdates()
	if err != nil {
		fmt.Printf("Error downloading updates: %v\n", err)
		return true
	}

	fmt.Println("Pattern updates completed successfully.")
	return true
}

// showPatternVersions displays version information for pattern files
func showPatternVersions(pm *PatternUpdateManager) {
	fmt.Println("Pattern Versions")
	fmt.Println("===============")

	// Get current versions
	stats, err := pm.GetStats()
	if err != nil {
		fmt.Printf("Error getting pattern stats: %v\n", err)
		return
	}

	fmt.Println("Current Versions:")
	fmt.Printf("  Error Patterns: %s\n", stats["error_patterns_version"])
	fmt.Printf("  Common Commands: %s\n", stats["commands_version"])

	// Check for updates in the background
	fmt.Println("\nChecking for latest versions...")
	go func() {
		latestVersions, err := pm.getLatestVersions()
		if err != nil {
			fmt.Printf("Error checking latest versions: %v\n", err)
			return
		}

		fmt.Println("\nLatest Available Versions:")
		fmt.Printf("  Error Patterns: %s (updated on %s)\n",
			latestVersions.Patterns.ErrorPatterns.Version,
			latestVersions.Patterns.ErrorPatterns.UpdatedAt)
		fmt.Printf("  Common Commands: %s (updated on %s)\n",
			latestVersions.Patterns.CommonCommands.Version,
			latestVersions.Patterns.CommonCommands.UpdatedAt)

		// Show update status
		errorNeedsUpdate := latestVersions.Patterns.ErrorPatterns.Version != stats["error_patterns_version"]
		commandsNeedsUpdate := latestVersions.Patterns.CommonCommands.Version != stats["commands_version"]

		if errorNeedsUpdate || commandsNeedsUpdate {
			fmt.Println("\n✓ Updates are available. Run ':pattern update' to download.")
		} else {
			fmt.Println("\n✓ All patterns are up to date.")
		}
	}()

	return
}

// listPatterns lists all available patterns
func listPatterns(pm *PatternUpdateManager) {
	fmt.Println("Available Patterns")
	fmt.Println("=================")

	stats, err := pm.GetStats()
	if err != nil {
		fmt.Printf("Error getting pattern stats: %v\n", err)
		return
	}

	// Show error patterns
	if !stats["error_patterns_exists"].(bool) {
		fmt.Println("Error Patterns: Not available")
	} else {
		// Read and parse the patterns file
		errorPatternsPath := pm.patternDir + "/error_patterns.json"
		data, err := readAndParsePatternFile(errorPatternsPath)
		if err != nil {
			fmt.Printf("Error reading error patterns: %v\n", err)
		} else {
			if patterns, ok := data["patterns"].([]interface{}); ok {
				fmt.Printf("Error Patterns (%d patterns):\n", len(patterns))
				for i, pattern := range patterns {
					if i >= 10 {
						fmt.Printf("  ... and %d more patterns\n", len(patterns)-10)
						break
					}

					p := pattern.(map[string]interface{})
					fmt.Printf("  - %s: %s\n", p["category"], p["description"])
				}
			}
		}
	}

	// Show common commands
	if !stats["commands_exists"].(bool) {
		fmt.Println("\nCommon Commands: Not available")
	} else {
		// Read and parse the commands file
		commandsPath := pm.patternDir + "/common_commands.json"
		data, err := readAndParsePatternFile(commandsPath)
		if err != nil {
			fmt.Printf("Error reading common commands: %v\n", err)
		} else {
			if commands, ok := data["commands"].([]interface{}); ok {
				fmt.Printf("\nCommon Commands (%d commands):\n", len(commands))
				for i, command := range commands {
					if i >= 10 {
						fmt.Printf("  ... and %d more commands\n", len(commands)-10)
						break
					}

					c := command.(map[string]interface{})
					fmt.Printf("  - %s: %s\n", c["category"], c["description"])
				}
			}
		}
	}

	fmt.Println("\nUse ':pattern list error' or ':pattern list commands' for more details.")
}

// readAndParsePatternFile reads and parses a pattern file
func readAndParsePatternFile(filePath string) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// checkForUpdates checks if pattern updates are available
func checkForUpdates(pm *PatternUpdateManager) {
	fmt.Println("Checking for pattern updates...")

	available, err := pm.CheckForUpdates()
	if err != nil {
		fmt.Printf("Error checking for updates: %v\n", err)
		return
	}

	if available {
		fmt.Println("✓ Updates are available!")
		fmt.Println("Run ':pattern update' to download and install updates.")
	} else {
		fmt.Println("✓ All patterns are up to date.")
	}
}

// setUpdateInterval sets the update check interval
func setUpdateInterval(pm *PatternUpdateManager, arg string) bool {
	hours, err := strconv.Atoi(arg)
	if err != nil || hours <= 0 {
		fmt.Println("Invalid interval. Please specify a positive number of hours.")
		fmt.Println("Usage: :pattern interval <hours>")
		return true
	}

	err = pm.UpdateCheckInterval(hours)
	if err != nil {
		fmt.Printf("Error updating check interval: %v\n", err)
		return true
	}

	fmt.Printf("Update check interval set to %d hours.\n", hours)
	return true
}

// showPatternCommandHelp displays help for pattern commands
func showPatternCommandHelp() {
	fmt.Println("Pattern Update Commands")
	fmt.Println("=====================")
	fmt.Println("  :pattern                - Show pattern update status")
	fmt.Println("  :pattern enable         - Enable pattern updates")
	fmt.Println("  :pattern disable        - Disable pattern updates")
	fmt.Println("  :pattern auto on|off    - Enable/disable automatic updates")
	fmt.Println("  :pattern update [--force] - Download and install pattern updates")
	fmt.Println("  :pattern versions       - Show current and latest pattern versions")
	fmt.Println("  :pattern list           - List available patterns")
	fmt.Println("  :pattern check          - Check for pattern updates")
	fmt.Println("  :pattern interval <hours> - Set update check interval")
	fmt.Println("  :pattern status         - Show pattern update status")
	fmt.Println("  :pattern stats          - Show detailed pattern statistics")
	fmt.Println("  :pattern help           - Show this help message")
}
