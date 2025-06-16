package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// UpdateChoice represents user's choice for update actions
type UpdateChoice int

const (
	UpdateChoiceInstall UpdateChoice = iota
	UpdateChoicePostpone
	UpdateChoiceSkip
	UpdateChoiceCancel
	UpdateChoiceViewChangelog
)

// UpdateUI handles interactive update user interface
type UpdateUI struct {
	reader   *bufio.Reader
	i18nMgr  *I18nManager
	colorize bool
}

// UpdatePromptOptions configures the update prompt behavior
type UpdatePromptOptions struct {
	ShowChangelog     bool
	AllowPostpone     bool
	AllowSkip         bool
	AutoConfirm       bool
	PostponeOptions   []string
	DefaultChoice     UpdateChoice
}

// NewUpdateUI creates a new update UI handler
func NewUpdateUI() *UpdateUI {
	return &UpdateUI{
		reader:   bufio.NewReader(os.Stdin),
		i18nMgr:  GetI18nManager(),
		colorize: true, // TODO: Check if terminal supports colors
	}
}

// PromptForUpdate displays an interactive update prompt and returns user choice
func (ui *UpdateUI) PromptForUpdate(updateInfo *UpdateInfo, options *UpdatePromptOptions) UpdateChoice {
	if options == nil {
		options = &UpdatePromptOptions{
			ShowChangelog:   true,
			AllowPostpone:   true,
			AllowSkip:       true,
			AutoConfirm:     false,
			PostponeOptions: []string{"1 hour", "4 hours", "1 day", "1 week"},
			DefaultChoice:   UpdateChoiceCancel,
		}
	}

	// Check for auto-confirm mode
	if options.AutoConfirm {
		return UpdateChoiceInstall
	}

	ui.displayUpdateHeader(updateInfo)
	
	// Show changelog if requested
	if options.ShowChangelog && updateInfo.ReleaseNotes != "" {
		ui.showChangelog(updateInfo.ReleaseNotes)
	}

	for {
		choice := ui.showUpdateMenu(updateInfo, options)
		
		switch choice {
		case UpdateChoiceInstall:
			if ui.confirmInstallation(updateInfo) {
				return UpdateChoiceInstall
			}
			// If confirmation failed, continue loop
			
		case UpdateChoicePostpone:
			if options.AllowPostpone {
				duration := ui.selectPostponeDuration(options.PostponeOptions)
				if duration != "" {
					ui.schedulePostponement(updateInfo, duration)
					return UpdateChoicePostpone
				}
			}
			
		case UpdateChoiceSkip:
			if options.AllowSkip {
				if ui.confirmSkipVersion(updateInfo) {
					return UpdateChoiceSkip
				}
			}
			
		case UpdateChoiceViewChangelog:
			ui.showDetailedChangelog(updateInfo)
			
		case UpdateChoiceCancel:
			return UpdateChoiceCancel
			
		default:
			ui.printError("Invalid choice. Please try again.")
		}
	}
}

// displayUpdateHeader shows the update information header
func (ui *UpdateUI) displayUpdateHeader(updateInfo *UpdateInfo) {
	ui.printHeader("🔔 Update Available!")
	fmt.Printf("   Current Version: %s\n", ui.colorText(updateInfo.CurrentVersion, "cyan"))
	fmt.Printf("   Latest Version:  %s\n", ui.colorText(updateInfo.LatestVersion, "green"))
	
	if updateInfo.IsPrerelease {
		fmt.Printf("   Type: %s\n", ui.colorText("Prerelease", "yellow"))
	}
	
	fmt.Printf("   Published: %s\n", updateInfo.PublishedAt.Format("2006-01-02 15:04"))
	
	if updateInfo.AssetSize > 0 {
		fmt.Printf("   Download Size: %s\n", formatFileSize(updateInfo.AssetSize))
	}
	
	fmt.Println()
}

// showUpdateMenu displays the update choice menu
func (ui *UpdateUI) showUpdateMenu(updateInfo *UpdateInfo, options *UpdatePromptOptions) UpdateChoice {
	fmt.Println("What would you like to do?")
	fmt.Println()
	
	menuItems := []string{
		"1. Install update now",
	}
	
	if options.AllowPostpone {
		menuItems = append(menuItems, "2. Postpone update")
	}
	
	if options.AllowSkip {
		menuItems = append(menuItems, "3. Skip this version")
	}
	
	if options.ShowChangelog && updateInfo.ReleaseNotes != "" {
		menuItems = append(menuItems, "4. View full changelog")
	}
	
	menuItems = append(menuItems, "0. Cancel")
	
	for _, item := range menuItems {
		fmt.Printf("   %s\n", item)
	}
	
	fmt.Print("\nEnter your choice: ")
	
	input, err := ui.reader.ReadString('\n')
	if err != nil {
		return UpdateChoiceCancel
	}
	
	choice := strings.TrimSpace(input)
	
	switch choice {
	case "1":
		return UpdateChoiceInstall
	case "2":
		if options.AllowPostpone {
			return UpdateChoicePostpone
		}
	case "3":
		if options.AllowSkip {
			return UpdateChoiceSkip
		}
	case "4":
		if options.ShowChangelog {
			return UpdateChoiceViewChangelog
		}
	case "0", "":
		return UpdateChoiceCancel
	}
	
	return UpdateChoiceCancel
}

// confirmInstallation asks for final confirmation before installation
func (ui *UpdateUI) confirmInstallation(updateInfo *UpdateInfo) bool {
	ui.printWarning("⚠️  This will replace the current Delta CLI binary!")
	fmt.Println("A backup will be created automatically.")
	fmt.Printf("Install %s? (y/N): ", updateInfo.LatestVersion)
	
	input, err := ui.reader.ReadString('\n')
	if err != nil {
		return false
	}
	
	response := strings.TrimSpace(strings.ToLower(input))
	return response == "y" || response == "yes"
}

// confirmSkipVersion asks for confirmation before skipping a version
func (ui *UpdateUI) confirmSkipVersion(updateInfo *UpdateInfo) bool {
	fmt.Printf("Skip version %s permanently? You won't be notified about this version again.\n", updateInfo.LatestVersion)
	fmt.Print("Skip this version? (y/N): ")
	
	input, err := ui.reader.ReadString('\n')
	if err != nil {
		return false
	}
	
	response := strings.TrimSpace(strings.ToLower(input))
	return response == "y" || response == "yes"
}

// selectPostponeDuration lets user choose postponement duration
func (ui *UpdateUI) selectPostponeDuration(options []string) string {
	fmt.Println("\nPostpone update for how long?")
	
	for i, option := range options {
		fmt.Printf("   %d. %s\n", i+1, option)
	}
	fmt.Printf("   0. Cancel\n")
	
	fmt.Print("\nEnter your choice: ")
	
	input, err := ui.reader.ReadString('\n')
	if err != nil {
		return ""
	}
	
	choice := strings.TrimSpace(input)
	if choice == "0" || choice == "" {
		return ""
	}
	
	index, err := strconv.Atoi(choice)
	if err != nil || index < 1 || index > len(options) {
		ui.printError("Invalid choice.")
		return ""
	}
	
	return options[index-1]
}

// schedulePostponement schedules the update reminder
func (ui *UpdateUI) schedulePostponement(updateInfo *UpdateInfo, duration string) {
	// Parse duration and create reminder
	postponeTime := ui.parseDuration(duration)
	if postponeTime == nil {
		ui.printError("Invalid duration format.")
		return
	}
	
	// Store postponement in update manager
	um := GetUpdateManager()
	if um != nil {
		config := um.GetConfig()
		config.PostponedVersion = updateInfo.LatestVersion
		config.PostponedUntil = postponeTime.Format(time.RFC3339)
		um.UpdateConfig(config)
	}
	
	ui.printSuccess(fmt.Sprintf("Update postponed until %s", postponeTime.Format("2006-01-02 15:04")))
}

// parseDuration converts user-friendly duration to time
func (ui *UpdateUI) parseDuration(duration string) *time.Time {
	duration = strings.ToLower(strings.TrimSpace(duration))
	now := time.Now()
	
	switch duration {
	case "1 hour":
		result := now.Add(1 * time.Hour)
		return &result
	case "4 hours":
		result := now.Add(4 * time.Hour)
		return &result
	case "1 day":
		result := now.Add(24 * time.Hour)
		return &result
	case "1 week":
		result := now.Add(7 * 24 * time.Hour)
		return &result
	default:
		return nil
	}
}

// showChangelog displays a condensed changelog
func (ui *UpdateUI) showChangelog(releaseNotes string) {
	fmt.Println("\nRelease Notes:")
	fmt.Println("==============")
	
	// Show first few lines or up to 300 characters
	lines := strings.Split(releaseNotes, "\n")
	charCount := 0
	lineCount := 0
	
	for _, line := range lines {
		if lineCount >= 10 || charCount+len(line) > 300 {
			fmt.Println("   ... (use option 4 to view full changelog)")
			break
		}
		
		if strings.TrimSpace(line) != "" {
			fmt.Printf("   %s\n", line)
			charCount += len(line)
			lineCount++
		}
	}
	
	fmt.Println()
}

// showDetailedChangelog displays the full changelog
func (ui *UpdateUI) showDetailedChangelog(updateInfo *UpdateInfo) {
	ui.printHeader("📋 Full Release Notes")
	fmt.Printf("Version: %s\n", updateInfo.LatestVersion)
	fmt.Printf("Published: %s\n\n", updateInfo.PublishedAt.Format("2006-01-02 15:04"))
	
	if updateInfo.ReleaseNotes != "" {
		fmt.Println(updateInfo.ReleaseNotes)
	} else {
		fmt.Println("No release notes available.")
	}
	
	fmt.Println()
	fmt.Print("Press Enter to continue...")
	ui.reader.ReadString('\n')
	fmt.Println()
}

// Helper methods for colored output
func (ui *UpdateUI) colorText(text, color string) string {
	if !ui.colorize {
		return text
	}
	
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

func (ui *UpdateUI) printHeader(text string) {
	fmt.Printf("\n%s\n", ui.colorText(text, "blue"))
}

func (ui *UpdateUI) printSuccess(text string) {
	fmt.Printf("✅ %s\n", ui.colorText(text, "green"))
}

func (ui *UpdateUI) printWarning(text string) {
	fmt.Printf("%s\n", ui.colorText(text, "yellow"))
}

func (ui *UpdateUI) printError(text string) {
	fmt.Printf("❌ %s\n", ui.colorText(text, "red"))
}

// IsPostponementActive checks if there's an active postponement
func (ui *UpdateUI) IsPostponementActive(updateInfo *UpdateInfo) bool {
	um := GetUpdateManager()
	if um == nil {
		return false
	}
	
	config := um.GetConfig()
	
	// Check if this version is postponed
	if config.PostponedVersion != updateInfo.LatestVersion {
		return false
	}
	
	// Check if postponement time has passed
	if config.PostponedUntil == "" {
		return false
	}
	
	postponedUntil, err := time.Parse(time.RFC3339, config.PostponedUntil)
	if err != nil {
		return false
	}
	
	return time.Now().Before(postponedUntil)
}

// ClearPostponement clears any active postponement
func (ui *UpdateUI) ClearPostponement() {
	um := GetUpdateManager()
	if um == nil {
		return
	}
	
	config := um.GetConfig()
	config.PostponedVersion = ""
	config.PostponedUntil = ""
	um.UpdateConfig(config)
}

// ShowPostponementReminder shows a reminder about postponed updates
func (ui *UpdateUI) ShowPostponementReminder(updateInfo *UpdateInfo) {
	um := GetUpdateManager()
	if um == nil {
		return
	}
	
	config := um.GetConfig()
	if config.PostponedUntil == "" {
		return
	}
	
	postponedUntil, err := time.Parse(time.RFC3339, config.PostponedUntil)
	if err != nil {
		return
	}
	
	fmt.Printf("\n⏰ Reminder: Update to %s was postponed until %s\n", 
		updateInfo.LatestVersion, postponedUntil.Format("2006-01-02 15:04"))
	fmt.Println("Use ':update check' to install now or postpone again.")
}