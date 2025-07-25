package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// HandleI18nCommand processes internationalization commands
func HandleI18nCommand(args []string) bool {
	if len(args) == 0 {
		// Show current i18n status
		showI18nStatus()
		return true
	}

	switch args[0] {
	case "locale":
		return handleLocaleCommand(args[1:])
	case "list":
		return handleListLocalesCommand()
	case "reload":
		return handleReloadTranslationsCommand()
	case "stats":
		return handleI18nStatsCommand()
	case "install":
		return handleI18nInstallCommand()
	case "download":
		return handleI18nDownloadCommand()
	case "update":
		return handleI18nUpdateCommand()
	case "help":
		return handleI18nHelpCommand()
	default:
		fmt.Printf(T("interface.commands.unknown", TranslationParams{"command": ":i18n " + args[0]}))
		fmt.Println(T("interface.commands.available"))
		return true
	}
}

// showI18nStatus displays the current internationalization status
func showI18nStatus() {
	i18n := GetI18nManager()

	fmt.Println(T("interface.i18n.status.title"))
	fmt.Printf("- %s: %s\n", T("interface.i18n.status.current_locale"), i18n.GetCurrentLocale())

	availableLocales := i18n.GetAvailableLocales()
	fmt.Printf("- %s: %s\n", T("interface.i18n.status.available_locales"), strings.Join(availableLocales, ", "))

	stats := i18n.GetTranslationStats()
	if loadedLocales, ok := stats["loaded_locales"].(int); ok {
		fmt.Printf("- %s: %d\n", T("interface.i18n.status.loaded_locales"), loadedLocales)
	}

	if totalKeys, ok := stats["total_keys"].(int); ok {
		fmt.Printf("- %s: %d\n", T("interface.i18n.status.total_keys"), totalKeys)
	}
}

// handleLocaleCommand manages locale setting
func handleLocaleCommand(args []string) bool {
	if len(args) == 0 {
		// Show current locale
		current := GetCurrentLocale()
		fmt.Printf("%s: %s\n", T("interface.i18n.locale.current"), current)
		return true
	}

	newLocale := args[0]

	// Validate locale exists
	availableLocales := GetAvailableLocales()
	localeExists := false
	for _, locale := range availableLocales {
		if locale == newLocale {
			localeExists = true
			break
		}
	}

	if !localeExists {
		fmt.Printf(T("interface.i18n.locale.not_found", TranslationParams{"locale": newLocale}))
		fmt.Printf("%s: %s\n", T("interface.i18n.locale.available"), strings.Join(availableLocales, ", "))
		return true
	}

	// Set the new locale
	if err := SetLocale(newLocale); err != nil {
		fmt.Printf(T("interface.i18n.locale.error", TranslationParams{"error": err.Error()}))
		return true
	}

	fmt.Printf(T("interface.i18n.locale.changed", TranslationParams{"locale": newLocale}))

	// Update configuration if available
	if cm := GetConfigManager(); cm != nil {
		// This would be implemented when config integration is done
		// cm.UpdateI18nLocale(newLocale)
	}

	return true
}

// handleListLocalesCommand lists all available locales
func handleListLocalesCommand() bool {
	availableLocales := GetAvailableLocales()
	currentLocale := GetCurrentLocale()

	fmt.Println(T("interface.i18n.list.title"))

	for _, locale := range availableLocales {
		marker := "  "
		if locale == currentLocale {
			marker = "* "
		}

		// Try to get the language name from the locale's meta information
		i18n := GetI18nManager()
		languageName := locale
		if translation := i18n.getTranslation(locale, "commands.meta.language"); translation != "" {
			languageName = fmt.Sprintf("%s (%s)", translation, locale)
		}

		fmt.Printf("%s%s\n", marker, languageName)
	}

	fmt.Printf("\n%s\n", T("interface.i18n.list.current_marker"))
	return true
}

// handleReloadTranslationsCommand reloads all translation files
func handleReloadTranslationsCommand() bool {
	i18n := GetI18nManager()

	fmt.Println(T("interface.i18n.reload.starting"))

	if err := i18n.ReloadTranslations(); err != nil {
		fmt.Printf(T("interface.i18n.reload.error", TranslationParams{"error": err.Error()}))
		return true
	}

	fmt.Println(T("interface.i18n.reload.success"))
	return true
}

// handleI18nStatsCommand shows detailed i18n statistics
func handleI18nStatsCommand() bool {
	i18n := GetI18nManager()
	stats := i18n.GetTranslationStats()

	fmt.Println(T("interface.i18n.stats.title"))

	if currentLocale, ok := stats["current_locale"].(string); ok {
		fmt.Printf("- %s: %s\n", T("interface.i18n.stats.current_locale"), currentLocale)
	}

	if fallbackLocale, ok := stats["fallback_locale"].(string); ok {
		fmt.Printf("- %s: %s\n", T("interface.i18n.stats.fallback_locale"), fallbackLocale)
	}

	if loadedLocales, ok := stats["loaded_locales"].(int); ok {
		fmt.Printf("- %s: %d\n", T("interface.i18n.stats.loaded_locales"), loadedLocales)
	}

	if totalKeys, ok := stats["total_keys"].(int); ok {
		fmt.Printf("- %s: %d\n", T("interface.i18n.stats.total_keys"), totalKeys)
	}

	// Show per-locale statistics
	fmt.Printf("\n%s:\n", T("interface.i18n.stats.per_locale"))
	for key, value := range stats {
		if strings.HasSuffix(key, "_keys") {
			locale := strings.TrimSuffix(key, "_keys")
			if count, ok := value.(int); ok && count > 0 {
				fmt.Printf("  - %s: %d %s\n", locale, count, T("interface.i18n.stats.keys"))
			}
		}
	}

	return true
}

// handleI18nInstallCommand installs i18n files to user config directory
func handleI18nInstallCommand() bool {
	fmt.Println("Installing i18n files to user configuration directory...")
	
	if err := installI18nFiles(); err != nil {
		fmt.Printf("Error installing i18n files: %v\n", err)
		return true
	}
	
	// Get the installed path
	homeDir, _ := os.UserHomeDir()
	installedPath := filepath.Join(homeDir, ".config", "delta", "i18n", "locales")
	
	fmt.Printf("Successfully installed i18n files to: %s\n", installedPath)
	
	// Reload translations
	i18n := GetI18nManager()
	i18n.basePath = installedPath
	i18n.ReloadTranslations()
	
	fmt.Println("Translations reloaded successfully.")
	return true
}

// handleI18nDownloadCommand downloads i18n files from GitHub
func handleI18nDownloadCommand() bool {
	i18n := GetI18nManager()
	
	fmt.Println("Checking for latest Delta release...")
	fmt.Println("This will download translations with SHA256 verification for security.")
	fmt.Println()
	
	// Download all locales from GitHub
	if err := i18n.DownloadAllLocales(); err != nil {
		fmt.Printf("Error downloading translations: %v\n", err)
		
		// Offer alternative solutions
		fmt.Println("\nTroubleshooting tips:")
		fmt.Println("1. Check your internet connection")
		fmt.Println("2. Ensure deltacli.com is accessible from your network")
		fmt.Println("3. Ensure GitHub releases are accessible for downloads")
		fmt.Println("4. Try again in a few moments")
		
		if !i18n.IsLocalesInstalled() {
			fmt.Println("\nAlternatively, if you have the source files:")
			fmt.Println("  :i18n install")
		}
		
		return true
	}
	
	return true
}

// handleI18nUpdateCommand updates i18n files from GitHub to the latest version
func handleI18nUpdateCommand() bool {
	i18n := GetI18nManager()
	
	fmt.Println("Checking for translation updates...")
	
	// Force re-download of all locales from GitHub
	if err := i18n.DownloadAllLocales(); err != nil {
		fmt.Printf("Error updating translations: %v\n", err)
		return true
	}
	
	fmt.Println("Translations updated successfully.")
	return true
}

// handleI18nHelpCommand shows help for i18n commands
func handleI18nHelpCommand() bool {
	i18n := GetI18nManager()
	
	fmt.Println(T("interface.i18n.help.title"))
	fmt.Println(T("interface.i18n.help.separator"))
	fmt.Println(T("interface.i18n.help.status"))
	fmt.Println(T("interface.i18n.help.locale_show"))
	fmt.Println(T("interface.i18n.help.locale_set"))
	fmt.Println(T("interface.i18n.help.list"))
	fmt.Println(T("interface.i18n.help.reload"))
	fmt.Println(T("interface.i18n.help.stats"))
	
	// Show download/update commands (always available)
	fmt.Println("  :i18n download           - Download all translations from GitHub")
	fmt.Println("  :i18n update             - Update translations to the latest version")
	
	// Only show install command if locales are not installed
	if !i18n.IsLocalesInstalled() {
		fmt.Println("  :i18n install            - Install i18n files from local source")
	}
	
	fmt.Println(T("interface.i18n.help.help"))

	fmt.Printf("\n%s:\n", T("interface.i18n.help.examples"))
	fmt.Println(T("interface.i18n.help.example_status"))
	fmt.Println(T("interface.i18n.help.example_list"))
	fmt.Println(T("interface.i18n.help.example_set_chinese"))
	fmt.Println(T("interface.i18n.help.example_set_english"))
	fmt.Println("  :i18n download                - Download all translations")

	return true
}

// Auto-detect and suggest locale based on system settings
func suggestLocale() string {
	if locale := detectSystemLocale(); locale != "" {
		// Check if the detected locale is available
		availableLocales := GetAvailableLocales()
		for _, available := range availableLocales {
			if available == locale {
				return locale
			}
		}

		// Try language code only (e.g., "zh" from "zh-CN")
		if parts := strings.Split(locale, "-"); len(parts) > 1 {
			lang := parts[0]
			for _, available := range availableLocales {
				if strings.HasPrefix(available, lang) {
					return available
				}
			}
		}
	}

	return "en" // Default to English
}

// initializeI18nSystem sets up the i18n system during Delta startup
func initializeI18nSystem() {
	i18n := GetI18nManager()

	// Try to initialize with system locale
	if err := i18n.Initialize(); err != nil {
		// Fall back to English if initialization fails
		i18n.SetLocale("en")
	}

	// Suggest locale if not English and user hasn't explicitly set one
	currentLocale := i18n.GetCurrentLocale()
	suggestedLocale := suggestLocale()

	if currentLocale == "en" && suggestedLocale != "en" {
		// Silently try the suggested locale, but don't error if it fails
		i18n.SetLocale(suggestedLocale)
	}
}

// checkI18nStartup checks if locales are installed and shows a notice if not
func checkI18nStartup() {
	i18n := GetI18nManager()
	
	// Only show notice if locales are not installed
	if !i18n.IsLocalesInstalled() {
		fmt.Println()
		fmt.Println("\033[33m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m")
		fmt.Println("\033[33m[∆ NOTICE]\033[0m Translation files not found. Using built-in English translations.")
		fmt.Println()
		fmt.Println("To download all available languages from GitHub:")
		fmt.Println("\033[36m  :i18n download\033[0m")
		fmt.Println()
		fmt.Println("This will download and install translation files to: ~/.config/delta/i18n/locales")
		fmt.Println("\033[33m━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\033[0m")
		fmt.Println()
	}
}
