package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

// Color constants for terminal output
const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorCyan   = "\033[36m"
)

// HandleUpdateChannelCommand handles the :update channel subcommand
func HandleUpdateChannelCommand(args []string) bool {
	cm := GetChannelManager()
	if cm == nil {
		fmt.Println("Failed to initialize channel manager")
		return false
	}

	if len(args) == 0 {
		// Show current channel
		showCurrentChannel(cm)
		return true
	}

	switch args[0] {
	case "list", "channels":
		return showAvailableChannels(cm)
	case "set", "switch":
		if len(args) < 2 {
			fmt.Println("Usage: :update channel set <channel-name>")
			return false
		}
		return switchChannel(cm, args[1], strings.Join(args[2:], " "))
	case "info", "show":
		if len(args) < 2 {
			return showChannelInfo(cm, string(cm.GetCurrentChannel()))
		}
		return showChannelInfo(cm, args[1])
	case "policy":
		return handleChannelPolicy(cm, args[1:])
	case "access":
		return handleChannelAccess(cm, args[1:])
	case "migrate":
		return handleChannelMigration(cm, args[1:])
	case "history":
		return showChannelHistory(cm)
	case "stats":
		return showChannelStats(cm)
	case "enterprise":
		return handleEnterpriseMode(cm, args[1:])
	default:
		// Try to switch to the channel directly
		return switchChannel(cm, args[0], strings.Join(args[1:], " "))
	}
}

// HandleUpdateChannelsCommand handles the :update channels subcommand
func HandleUpdateChannelsCommand(args []string) bool {
	return HandleUpdateChannelCommand(append([]string{"list"}, args...))
}

func showCurrentChannel(cm *ChannelManager) {
	channel := cm.GetCurrentChannel()
	policy, err := cm.GetChannelPolicy(channel)
	
	fmt.Printf("Current Update Channel: %s%s%s\n", ColorCyan, channel, ColorReset)
	
	if err == nil && policy != nil {
		fmt.Printf("Description: %s\n", policy.Description)
		fmt.Printf("Update Frequency: %s\n", policy.UpdateFrequency)
		fmt.Printf("Auto-install: %v\n", policy.AutoInstall)
		
		if policy.AllowPrerelease {
			fmt.Printf("Pre-releases: Allowed\n")
		}
	}
}

func showAvailableChannels(cm *ChannelManager) bool {
	fmt.Println("Available Update Channels:")
	fmt.Println("=========================")
	
	channels := cm.GetAvailableChannels()
	currentChannel := cm.GetCurrentChannel()
	
	for _, channel := range channels {
		policy, err := cm.GetChannelPolicy(channel)
		if err != nil {
			continue
		}
		
		marker := "  "
		if channel == currentChannel {
			marker = "* "
		}
		
		fmt.Printf("%s%s%-10s%s - %s\n", 
			marker,
			ColorCyan, channel, ColorReset,
			policy.Description)
		
		// Show additional info
		if policy.AllowPrerelease {
			fmt.Printf("    Pre-releases: Yes | ")
		} else {
			fmt.Printf("    Pre-releases: No  | ")
		}
		
		fmt.Printf("Updates: %s | ", policy.UpdateFrequency)
		
		if policy.AutoInstall {
			fmt.Printf("Auto-install: Yes\n")
		} else {
			fmt.Printf("Auto-install: No\n")
		}
		
		// Show restrictions if any
		if len(policy.AllowedUsers) > 0 || len(policy.AllowedGroups) > 0 {
			fmt.Printf("    %sRestricted access%s\n", ColorYellow, ColorReset)
		}
		
		if policy.MinVersionRequired != "" || policy.MaxVersionAllowed != "" {
			fmt.Printf("    Version constraints: ")
			if policy.MinVersionRequired != "" {
				fmt.Printf(">= %s ", policy.MinVersionRequired)
			}
			if policy.MaxVersionAllowed != "" {
				fmt.Printf("<= %s", policy.MaxVersionAllowed)
			}
			fmt.Println()
		}
		
		fmt.Println()
	}
	
	// Show enterprise status
	stats := cm.GetChannelStats()
	if isEnterprise, ok := stats["enterprise_mode"].(bool); ok && isEnterprise {
		fmt.Printf("\n%sEnterprise Mode Active%s\n", ColorYellow, ColorReset)
		if accessRules, ok := stats["total_access_rules"].(int); ok && accessRules > 0 {
			fmt.Printf("Access rules configured: %d\n", accessRules)
		}
	}
	
	return true
}

func switchChannel(cm *ChannelManager, channelName string, reason string) bool {
	// Validate channel name
	if !ValidateChannelName(channelName) {
		fmt.Printf("Invalid channel name: %s\n", channelName)
		fmt.Println("Valid channels: stable, beta, alpha, nightly, custom, or custom-*")
		return false
	}
	
	channel := UpdateChannel(channelName)
	currentChannel := cm.GetCurrentChannel()
	
	if channel == currentChannel {
		fmt.Printf("Already on %s channel\n", channel)
		return true
	}
	
	// Prompt for reason if not provided
	if reason == "" {
		fmt.Print("Reason for channel change (optional): ")
		reason = readLine()
		if reason == "" {
			reason = "Manual channel switch"
		}
	}
	
	// Attempt to switch
	if err := cm.SetChannel(channel, reason); err != nil {
		fmt.Printf("%sError switching channel:%s %v\n", ColorRed, ColorReset, err)
		return false
	}
	
	fmt.Printf("%sSuccessfully switched to %s channel%s\n", ColorGreen, channel, ColorReset)
	
	// Show new channel info
	showChannelInfo(cm, channelName)
	
	// Check for updates on new channel
	fmt.Println("\nChecking for updates on new channel...")
	um := GetUpdateManager()
	if um != nil {
		updateInfo, err := um.CheckForUpdates()
		if err == nil && updateInfo.HasUpdate {
			fmt.Printf("\n%sUpdate available:%s version %s\n", ColorYellow, ColorReset, updateInfo.LatestVersion)
			fmt.Println("Run ':update install' to upgrade")
		} else {
			fmt.Println("No updates available on this channel")
		}
	}
	
	return true
}

func showChannelInfo(cm *ChannelManager, channelName string) bool {
	channel := UpdateChannel(channelName)
	policy, err := cm.GetChannelPolicy(channel)
	if err != nil {
		fmt.Printf("Error getting channel info: %v\n", err)
		return false
	}
	
	fmt.Printf("\nChannel: %s%s%s\n", ColorCyan, channel, ColorReset)
	fmt.Println(strings.Repeat("=", 40))
	fmt.Printf("Description: %s\n", policy.Description)
	fmt.Printf("Update Frequency: %s\n", policy.UpdateFrequency)
	fmt.Printf("Auto-install Updates: %v\n", policy.AutoInstall)
	fmt.Printf("Allow Downgrades: %v\n", policy.AllowDowngrade)
	fmt.Printf("Allow Pre-releases: %v\n", policy.AllowPrerelease)
	fmt.Printf("Require Approval: %v\n", policy.RequireApproval)
	
	if policy.MinVersionRequired != "" {
		fmt.Printf("Minimum Version: %s\n", policy.MinVersionRequired)
	}
	if policy.MaxVersionAllowed != "" {
		fmt.Printf("Maximum Version: %s\n", policy.MaxVersionAllowed)
	}
	
	if len(policy.AllowedUsers) > 0 {
		fmt.Printf("Allowed Users: %s\n", strings.Join(policy.AllowedUsers, ", "))
	}
	if len(policy.AllowedGroups) > 0 {
		fmt.Printf("Allowed Groups: %s\n", strings.Join(policy.AllowedGroups, ", "))
	}
	if len(policy.RestrictedRegions) > 0 {
		fmt.Printf("Restricted Regions: %s\n", strings.Join(policy.RestrictedRegions, ", "))
	}
	
	if policy.CustomURL != "" {
		fmt.Printf("Custom Update URL: %s\n", policy.CustomURL)
	}
	if policy.VerificationKey != "" {
		fmt.Printf("Custom Verification Key: %s...%s\n", 
			policy.VerificationKey[:8], 
			policy.VerificationKey[len(policy.VerificationKey)-8:])
	}
	
	return true
}

func handleChannelPolicy(cm *ChannelManager, args []string) bool {
	if len(args) == 0 {
		fmt.Println("Channel policy commands:")
		fmt.Println("  :update channel policy list              - List all policies")
		fmt.Println("  :update channel policy show <channel>    - Show specific policy")
		fmt.Println("  :update channel policy set <channel> <key> <value> - Modify policy (enterprise)")
		return false
	}
	
	switch args[0] {
	case "list":
		return listChannelPolicies(cm)
	case "show":
		if len(args) < 2 {
			fmt.Println("Usage: :update channel policy show <channel>")
			return false
		}
		return showChannelInfo(cm, args[1])
	case "set":
		if len(args) < 4 {
			fmt.Println("Usage: :update channel policy set <channel> <key> <value>")
			return false
		}
		return setChannelPolicy(cm, args[1], args[2], args[3])
	default:
		fmt.Printf("Unknown policy command: %s\n", args[0])
		return false
	}
}

func listChannelPolicies(cm *ChannelManager) bool {
	fmt.Println("Channel Policies:")
	fmt.Println("=================")
	
	channels := []UpdateChannel{
		ChannelStable, ChannelBeta, ChannelAlpha, ChannelNightly, ChannelCustom,
	}
	
	for _, channel := range channels {
		policy, err := cm.GetChannelPolicy(channel)
		if err != nil {
			continue
		}
		
		fmt.Printf("\n%s%s Channel:%s\n", ColorCyan, channel, ColorReset)
		fmt.Printf("  Update Frequency: %s\n", policy.UpdateFrequency)
		fmt.Printf("  Auto-install: %v | Downgrades: %v | Pre-releases: %v\n",
			policy.AutoInstall, policy.AllowDowngrade, policy.AllowPrerelease)
	}
	
	return true
}

func setChannelPolicy(cm *ChannelManager, channelName, key, value string) bool {
	channel := UpdateChannel(channelName)
	policy, err := cm.GetChannelPolicy(channel)
	if err != nil {
		fmt.Printf("Error getting channel policy: %v\n", err)
		return false
	}
	
	// Modify the policy based on key
	switch key {
	case "auto_install":
		policy.AutoInstall = value == "true"
	case "allow_downgrade":
		policy.AllowDowngrade = value == "true"
	case "allow_prerelease":
		policy.AllowPrerelease = value == "true"
	case "require_approval":
		policy.RequireApproval = value == "true"
	case "update_frequency":
		if value != "immediate" && value != "daily" && value != "weekly" && value != "monthly" {
			fmt.Println("Invalid frequency. Use: immediate, daily, weekly, or monthly")
			return false
		}
		policy.UpdateFrequency = value
	case "min_version":
		policy.MinVersionRequired = value
	case "max_version":
		policy.MaxVersionAllowed = value
	default:
		fmt.Printf("Unknown policy key: %s\n", key)
		fmt.Println("Valid keys: auto_install, allow_downgrade, allow_prerelease, require_approval")
		fmt.Println("           update_frequency, min_version, max_version")
		return false
	}
	
	// Apply the updated policy
	if err := cm.SetChannelPolicy(channel, policy); err != nil {
		fmt.Printf("%sError setting policy:%s %v\n", ColorRed, ColorReset, err)
		return false
	}
	
	fmt.Printf("%sPolicy updated successfully%s\n", ColorGreen, ColorReset)
	return true
}

func handleChannelAccess(cm *ChannelManager, args []string) bool {
	if len(args) == 0 {
		fmt.Println("Channel access commands:")
		fmt.Println("  :update channel access list              - List access rules")
		fmt.Println("  :update channel access set <user> <channels...> - Set user access")
		fmt.Println("  :update channel access force <user> <channel>   - Force user to channel")
		fmt.Println("  :update channel access remove <user>            - Remove access rules")
		return false
	}
	
	stats := cm.GetChannelStats()
	if isEnterprise, ok := stats["enterprise_mode"].(bool); !ok || !isEnterprise {
		fmt.Printf("%sChannel access control requires enterprise mode%s\n", ColorYellow, ColorReset)
		fmt.Println("Enable with: :update channel enterprise on")
		return false
	}
	
	switch args[0] {
	case "list":
		// For now, just show count from stats
		if count, ok := stats["total_access_rules"].(int); ok {
			fmt.Printf("Total access rules configured: %d\n", count)
		}
		return true
	case "set":
		if len(args) < 3 {
			fmt.Println("Usage: :update channel access set <user> <channels...>")
			return false
		}
		return setUserAccess(cm, args[1], args[2:], false)
	case "force":
		if len(args) < 3 {
			fmt.Println("Usage: :update channel access force <user> <channel>")
			return false
		}
		return setUserAccess(cm, args[1], []string{args[2]}, true)
	case "remove":
		if len(args) < 2 {
			fmt.Println("Usage: :update channel access remove <user>")
			return false
		}
		return removeUserAccess(cm, args[1])
	default:
		fmt.Printf("Unknown access command: %s\n", args[0])
		return false
	}
}

func setUserAccess(cm *ChannelManager, userID string, channelNames []string, forced bool) bool {
	var channels []UpdateChannel
	for _, name := range channelNames {
		if !ValidateChannelName(name) {
			fmt.Printf("Invalid channel name: %s\n", name)
			return false
		}
		channels = append(channels, UpdateChannel(name))
	}
	
	access := &ChannelAccess{
		UserID:          userID,
		AllowedChannels: channels,
	}
	
	if forced && len(channels) == 1 {
		access.ForcedChannel = &channels[0]
	}
	
	if err := cm.SetUserAccess(userID, access); err != nil {
		fmt.Printf("%sError setting user access:%s %v\n", ColorRed, ColorReset, err)
		return false
	}
	
	fmt.Printf("%sAccess rules updated for user %s%s\n", ColorGreen, userID, ColorReset)
	if forced {
		fmt.Printf("User is now forced to use %s channel\n", channels[0])
	}
	return true
}

func removeUserAccess(cm *ChannelManager, userID string) bool {
	// Setting nil access effectively removes restrictions
	if err := cm.SetUserAccess(userID, nil); err != nil {
		fmt.Printf("%sError removing user access:%s %v\n", ColorRed, ColorReset, err)
		return false
	}
	
	fmt.Printf("%sAccess restrictions removed for user %s%s\n", ColorGreen, userID, ColorReset)
	return true
}

func handleChannelMigration(cm *ChannelManager, args []string) bool {
	if len(args) == 0 {
		fmt.Println("Channel migration commands:")
		fmt.Println("  :update channel migrate schedule <from> <to> <time> - Schedule migration")
		fmt.Println("  :update channel migrate list                       - List migrations")
		fmt.Println("  :update channel migrate process                     - Process pending migrations")
		return false
	}
	
	stats := cm.GetChannelStats()
	if isEnterprise, ok := stats["enterprise_mode"].(bool); !ok || !isEnterprise {
		fmt.Printf("%sChannel migration requires enterprise mode%s\n", ColorYellow, ColorReset)
		fmt.Println("Enable with: :update channel enterprise on")
		return false
	}
	
	switch args[0] {
	case "schedule":
		if len(args) < 4 {
			fmt.Println("Usage: :update channel migrate schedule <from> <to> <time>")
			fmt.Println("Time format: 2006-01-02 15:04:05 or 'now+1h', 'now+1d'")
			return false
		}
		return scheduleMigration(cm, args[1], args[2], args[3])
	case "list":
		return listMigrations(cm)
	case "process":
		return processMigrations(cm)
	default:
		fmt.Printf("Unknown migration command: %s\n", args[0])
		return false
	}
}

func scheduleMigration(cm *ChannelManager, fromChannel, toChannel, timeStr string) bool {
	// Parse time
	var scheduledTime time.Time
	if strings.HasPrefix(timeStr, "now+") {
		duration := timeStr[4:]
		if strings.HasSuffix(duration, "h") {
			hours := 1
			fmt.Sscanf(duration, "%dh", &hours)
			scheduledTime = time.Now().Add(time.Duration(hours) * time.Hour)
		} else if strings.HasSuffix(duration, "d") {
			days := 1
			fmt.Sscanf(duration, "%dd", &days)
			scheduledTime = time.Now().Add(time.Duration(days) * 24 * time.Hour)
		} else {
			fmt.Println("Invalid time format. Use 'now+1h' or 'now+1d'")
			return false
		}
	} else {
		var err error
		scheduledTime, err = time.Parse("2006-01-02 15:04:05", timeStr)
		if err != nil {
			fmt.Printf("Invalid time format: %v\n", err)
			return false
		}
	}
	
	// Get reason
	fmt.Print("Reason for migration: ")
	reason := readLine()
	if reason == "" {
		reason = "Scheduled channel migration"
	}
	
	// Get affected users (empty means all)
	fmt.Print("Affected users (comma-separated, empty for all): ")
	userInput := readLine()
	var affectedUsers []string
	if userInput != "" {
		affectedUsers = strings.Split(userInput, ",")
		for i := range affectedUsers {
			affectedUsers[i] = strings.TrimSpace(affectedUsers[i])
		}
	}
	
	migration, err := cm.ScheduleMigration(
		UpdateChannel(fromChannel),
		UpdateChannel(toChannel),
		scheduledTime,
		reason,
		affectedUsers,
	)
	
	if err != nil {
		fmt.Printf("%sError scheduling migration:%s %v\n", ColorRed, ColorReset, err)
		return false
	}
	
	fmt.Printf("%sMigration scheduled successfully%s\n", ColorGreen, ColorReset)
	fmt.Printf("Migration ID: %s\n", migration.ID)
	fmt.Printf("From: %s → To: %s\n", fromChannel, toChannel)
	fmt.Printf("Scheduled: %s\n", scheduledTime.Format("2006-01-02 15:04:05"))
	if len(affectedUsers) > 0 {
		fmt.Printf("Affected users: %s\n", strings.Join(affectedUsers, ", "))
	} else {
		fmt.Println("Affected users: All")
	}
	
	return true
}

func listMigrations(cm *ChannelManager) bool {
	stats := cm.GetChannelStats()
	if migrations, ok := stats["migrations"].(map[string]int); ok && len(migrations) > 0 {
		fmt.Println("Migration Summary:")
		fmt.Println("==================")
		for status, count := range migrations {
			fmt.Printf("  %s: %d\n", status, count)
		}
	} else {
		fmt.Println("No migrations scheduled")
	}
	return true
}

func processMigrations(cm *ChannelManager) bool {
	fmt.Println("Processing pending migrations...")
	if err := cm.ProcessPendingMigrations(); err != nil {
		fmt.Printf("%sError processing migrations:%s %v\n", ColorRed, ColorReset, err)
		return false
	}
	fmt.Println("Migration processing complete")
	return true
}

func showChannelHistory(cm *ChannelManager) bool {
	stats := cm.GetChannelStats()
	if changes, ok := stats["channel_changes"].(int); ok && changes > 0 {
		fmt.Printf("Channel change history: %d changes recorded\n", changes)
		fmt.Println("\nNote: Full history viewing not yet implemented")
	} else {
		fmt.Println("No channel changes recorded")
	}
	return true
}

func showChannelStats(cm *ChannelManager) bool {
	stats := cm.GetChannelStats()
	
	fmt.Println("Channel Statistics:")
	fmt.Println("==================")
	
	if current, ok := stats["current_channel"].(UpdateChannel); ok {
		fmt.Printf("Current Channel: %s\n", current)
	}
	
	if isEnterprise, ok := stats["enterprise_mode"].(bool); ok {
		fmt.Printf("Enterprise Mode: %v\n", isEnterprise)
	}
	
	if policies, ok := stats["total_policies"].(int); ok {
		fmt.Printf("Total Policies: %d\n", policies)
	}
	
	if accessRules, ok := stats["total_access_rules"].(int); ok {
		fmt.Printf("Access Rules: %d\n", accessRules)
	}
	
	if changes, ok := stats["channel_changes"].(int); ok {
		fmt.Printf("Channel Changes: %d\n", changes)
	}
	
	if migrations, ok := stats["migrations"].(map[string]int); ok && len(migrations) > 0 {
		fmt.Println("\nMigration Status:")
		for status, count := range migrations {
			fmt.Printf("  %s: %d\n", status, count)
		}
	}
	
	return true
}

func handleEnterpriseMode(cm *ChannelManager, args []string) bool {
	if len(args) == 0 {
		stats := cm.GetChannelStats()
		if isEnterprise, ok := stats["enterprise_mode"].(bool); ok {
			if isEnterprise {
				fmt.Println("Enterprise mode is ENABLED")
			} else {
				fmt.Println("Enterprise mode is DISABLED")
			}
		}
		return true
	}
	
	switch args[0] {
	case "on", "enable":
		if err := cm.SetEnterpriseMode(true); err != nil {
			fmt.Printf("%sError enabling enterprise mode:%s %v\n", ColorRed, ColorReset, err)
			return false
		}
		fmt.Printf("%sEnterprise mode enabled%s\n", ColorGreen, ColorReset)
		fmt.Println("You can now:")
		fmt.Println("  - Set custom channel policies")
		fmt.Println("  - Configure user access restrictions")
		fmt.Println("  - Schedule channel migrations")
		fmt.Println("  - Force users to specific channels")
		return true
		
	case "off", "disable":
		if err := cm.SetEnterpriseMode(false); err != nil {
			fmt.Printf("%sError disabling enterprise mode:%s %v\n", ColorRed, ColorReset, err)
			return false
		}
		fmt.Printf("%sEnterprise mode disabled%s\n", ColorGreen, ColorReset)
		return true
		
	default:
		fmt.Println("Usage: :update channel enterprise [on|off]")
		return false
	}
}

// Helper function to read a line of input
func readLine() string {
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}