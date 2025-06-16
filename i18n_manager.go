package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// I18nManager handles internationalization for Delta CLI
type I18nManager struct {
	currentLocale  string
	fallbackLocale string
	translations   map[string]map[string]interface{}
	loadedFiles    map[string]bool
	mutex          sync.RWMutex
	basePath       string
	localesFound   bool // Track if locale files were found
}

// TranslationParams holds parameters for string interpolation
type TranslationParams map[string]interface{}

// Global i18n manager instance
var globalI18nManager *I18nManager
var i18nOnce sync.Once

// GetI18nManager returns the global i18n manager instance
func GetI18nManager() *I18nManager {
	i18nOnce.Do(func() {
		globalI18nManager = NewI18nManager()
	})
	return globalI18nManager
}

// getBuiltInEnglishTranslations returns essential built-in English translations
func getBuiltInEnglishTranslations() map[string]interface{} {
	return map[string]interface{}{
		"interface": map[string]interface{}{
			"welcome": map[string]interface{}{
				"message": "Welcome to Delta CLI - Your Context-Aware Shell Enhancement",
				"features_enabled": "[∆ {{feature}}: {{details}}]",
			},
			"goodbye": map[string]interface{}{
				"message": "\nThank you for using Delta CLI. Goodbye!",
			},
			"ai": map[string]interface{}{
				"features_enabled": "AI features enabled with {{model}}",
			},
			"commands": map[string]interface{}{
				"unknown": "Unknown command: {{command}}",
				"available": "Type :help for available commands",
			},
			"error": map[string]interface{}{
				"command_failed": "Command failed: {{error}}",
			},
		},
		"commands": map[string]interface{}{
			"help": map[string]interface{}{
				"title": "Available Commands",
				"description": "Delta CLI Help - Type any command to execute it",
			},
		},
	}
}

// findI18nBasePath attempts to find the i18n directory in various locations
func findI18nBasePath() string {
	// Method 1: Check user config directory first (installed locales)
	homeDir, err := os.UserHomeDir()
	if err == nil {
		userI18nPath := filepath.Join(homeDir, ".config", "delta", "i18n", "locales")
		if _, err := os.Stat(userI18nPath); err == nil {
			return userI18nPath
		}
	}
	
	// Method 2: Check relative to executable location
	execPath, err := os.Executable()
	if err == nil {
		// Resolve any symlinks
		execPath, _ = filepath.EvalSymlinks(execPath)
		execDir := filepath.Dir(execPath)
		
		// Check if i18n directory exists relative to executable
		i18nPath := filepath.Join(execDir, "i18n", "locales")
		if _, err := os.Stat(i18nPath); err == nil {
			return i18nPath
		}
	}
	
	// Method 3: Check current working directory (for development)
	cwd, err := os.Getwd()
	if err == nil {
		i18nPath := filepath.Join(cwd, "i18n", "locales")
		if _, err := os.Stat(i18nPath); err == nil {
			return i18nPath
		}
	}
	
	// Method 4: Check environment variable
	if deltaHome := os.Getenv("DELTA_HOME"); deltaHome != "" {
		i18nPath := filepath.Join(deltaHome, "i18n", "locales")
		if _, err := os.Stat(i18nPath); err == nil {
			return i18nPath
		}
	}
	
	// Fallback to relative path
	return "i18n/locales"
}

// installI18nFiles copies i18n files from source to user config directory
func installI18nFiles() error {
	// Find source i18n directory
	sourcePaths := []string{
		"i18n/locales", // Development
	}
	
	// Add executable relative paths
	if execPath, err := os.Executable(); err == nil {
		execPath, _ = filepath.EvalSymlinks(execPath)
		execDir := filepath.Dir(execPath)
		sourcePaths = append(sourcePaths, filepath.Join(execDir, "i18n", "locales"))
	}
	
	// Find a valid source directory
	var sourceDir string
	for _, path := range sourcePaths {
		if _, err := os.Stat(path); err == nil {
			sourceDir = path
			break
		}
	}
	
	if sourceDir == "" {
		return fmt.Errorf("could not find i18n source files")
	}
	
	// Create destination directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not determine home directory: %v", err)
	}
	
	destDir := filepath.Join(homeDir, ".config", "delta", "i18n", "locales")
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("could not create i18n directory: %v", err)
	}
	
	// Copy locale directories
	entries, err := ioutil.ReadDir(sourceDir)
	if err != nil {
		return fmt.Errorf("could not read source directory: %v", err)
	}
	
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		
		localeSource := filepath.Join(sourceDir, entry.Name())
		localeDest := filepath.Join(destDir, entry.Name())
		
		// Create locale directory
		if err := os.MkdirAll(localeDest, 0755); err != nil {
			continue
		}
		
		// Copy JSON files
		files, err := ioutil.ReadDir(localeSource)
		if err != nil {
			continue
		}
		
		for _, file := range files {
			if !strings.HasSuffix(file.Name(), ".json") {
				continue
			}
			
			srcFile := filepath.Join(localeSource, file.Name())
			dstFile := filepath.Join(localeDest, file.Name())
			
			content, err := ioutil.ReadFile(srcFile)
			if err != nil {
				continue
			}
			
			if err := ioutil.WriteFile(dstFile, content, 0644); err != nil {
				continue
			}
		}
	}
	
	return nil
}

// NewI18nManager creates a new internationalization manager
func NewI18nManager() *I18nManager {
	basePath := findI18nBasePath()
	localesFound := true
	
	// Check if we actually found locale files
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		localesFound = false
		// Try to install files to user config
		if err := installI18nFiles(); err == nil {
			// Update base path to installed location
			if homeDir, err := os.UserHomeDir(); err == nil {
				installedPath := filepath.Join(homeDir, ".config", "delta", "i18n", "locales")
				if _, err := os.Stat(installedPath); err == nil {
					basePath = installedPath
					localesFound = true
				}
			}
		}
	}
	
	manager := &I18nManager{
		currentLocale:  "en",
		fallbackLocale: "en",
		translations:   make(map[string]map[string]interface{}),
		loadedFiles:    make(map[string]bool),
		basePath:       basePath,
		localesFound:   localesFound,
	}
	
	// Load built-in English translations as fallback
	manager.translations["en"] = getBuiltInEnglishTranslations()
	manager.loadedFiles["en"] = true

	// Try to load default locale from files (will merge with built-ins)
	manager.LoadLocale("en")

	return manager
}

// LoadLocale loads translation files for a specific locale
func (i *I18nManager) LoadLocale(locale string) error {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	// Check if already loaded
	if i.loadedFiles[locale] {
		return nil
	}

	localePath := filepath.Join(i.basePath, locale)

	// Check if locale directory exists
	if _, err := os.Stat(localePath); os.IsNotExist(err) {
		return fmt.Errorf("locale %s not found", locale)
	}

	// Initialize locale map if it doesn't exist
	if i.translations[locale] == nil {
		i.translations[locale] = make(map[string]interface{})
	}

	// Load all JSON files in the locale directory
	files, err := ioutil.ReadDir(localePath)
	if err != nil {
		return fmt.Errorf("failed to read locale directory: %v", err)
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(localePath, file.Name())
		content, err := ioutil.ReadFile(filePath)
		if err != nil {
			continue // Skip files that can't be read
		}

		var data map[string]interface{}
		if err := json.Unmarshal(content, &data); err != nil {
			continue // Skip invalid JSON files
		}

		// Merge the data into the locale translations
		fileKey := strings.TrimSuffix(file.Name(), ".json")
		i.translations[locale][fileKey] = data
	}

	i.loadedFiles[locale] = true
	return nil
}

// IsLocalesInstalled returns whether locale files are properly installed
func (i *I18nManager) IsLocalesInstalled() bool {
	return i.localesFound
}

// SetLocale changes the current locale
func (i *I18nManager) SetLocale(locale string) error {
	// Try to load the locale if not already loaded
	if err := i.LoadLocale(locale); err != nil {
		return err
	}

	i.mutex.Lock()
	i.currentLocale = locale
	i.mutex.Unlock()

	// Persist the locale change to configuration
	cm := GetConfigManager()
	if cm != nil {
		config := cm.GetI18nConfig()
		if config != nil {
			config.Locale = locale
			cm.UpdateI18nConfig(config)
		}
	}

	return nil
}

// GetCurrentLocale returns the current locale
func (i *I18nManager) GetCurrentLocale() string {
	i.mutex.RLock()
	defer i.mutex.RUnlock()
	return i.currentLocale
}

// GetAvailableLocales returns a list of available locales
func (i *I18nManager) GetAvailableLocales() []string {
	locales := []string{}

	// Read the locales directory
	files, err := ioutil.ReadDir(i.basePath)
	if err != nil {
		return []string{"en"} // Return default if can't read directory
	}

	for _, file := range files {
		if file.IsDir() {
			locales = append(locales, file.Name())
		}
	}

	return locales
}

// T translates a key with optional parameters
func (i *I18nManager) T(key string, params ...TranslationParams) string {
	i.mutex.RLock()
	defer i.mutex.RUnlock()

	// Try current locale first
	if translation := i.getTranslation(i.currentLocale, key); translation != "" {
		return i.interpolate(translation, params...)
	}

	// Fall back to fallback locale
	if i.currentLocale != i.fallbackLocale {
		if translation := i.getTranslation(i.fallbackLocale, key); translation != "" {
			return i.interpolate(translation, params...)
		}
	}

	// Return the key itself if no translation found
	return key
}

// getTranslation retrieves a translation for a specific locale and key
func (i *I18nManager) getTranslation(locale, key string) string {
	if i.translations[locale] == nil {
		return ""
	}

	// Split the key by dots to navigate nested structure
	keys := strings.Split(key, ".")
	var current interface{} = i.translations[locale]

	for _, k := range keys {
		if current == nil {
			return ""
		}

		switch v := current.(type) {
		case map[string]interface{}:
			current = v[k]
		case string:
			if len(keys) == 1 {
				return v
			}
			return ""
		default:
			return ""
		}
	}

	if str, ok := current.(string); ok {
		return str
	}

	return ""
}

// interpolate replaces variables in a translation string
func (i *I18nManager) interpolate(translation string, params ...TranslationParams) string {
	if len(params) == 0 {
		return translation
	}

	result := translation
	for _, param := range params {
		for key, value := range param {
			placeholder := fmt.Sprintf("{{%s}}", key)
			replacement := fmt.Sprintf("%v", value)
			result = strings.ReplaceAll(result, placeholder, replacement)
		}
	}

	return result
}

// PluralRule defines the function signature for pluralization rules
type PluralRule func(count int) string

// TPlural handles pluralization with advanced grammar rules
func (i *I18nManager) TPlural(key string, count int, params ...TranslationParams) string {
	i.mutex.RLock()
	locale := i.currentLocale
	i.mutex.RUnlock()

	// Get the appropriate plural form for the locale
	pluralForm := i.getPluralForm(locale, count)
	pluralKey := key + "." + pluralForm

	// Add count to params if not already present
	allParams := make([]TranslationParams, len(params)+1)
	copy(allParams, params)
	allParams[len(params)] = TranslationParams{"count": count}

	// Try plural key first
	if translation := i.T(pluralKey, allParams...); translation != pluralKey {
		return translation
	}

	// Fall back to base key
	return i.T(key, allParams...)
}

// getPluralForm returns the plural form identifier for a given locale and count
func (i *I18nManager) getPluralForm(locale string, count int) string {
	rule := i.getPluralRule(locale)
	return rule(count)
}

// getPluralRule returns the pluralization rule for a given locale
func (i *I18nManager) getPluralRule(locale string) PluralRule {
	switch locale {
	case "ru": // Russian - 3 forms
		return russianPluralRule
	case "pl": // Polish - 3 forms
		return polishPluralRule
	case "cs", "sk": // Czech, Slovak - 3 forms
		return czechSlovakPluralRule
	case "lt": // Lithuanian - 3 forms
		return lithuanianPluralRule
	case "lv": // Latvian - 3 forms
		return latvianPluralRule
	case "ar": // Arabic - 6 forms
		return arabicPluralRule
	case "ga": // Irish - 5 forms
		return irishPluralRule
	case "gd": // Scottish Gaelic - 4 forms
		return scottishGaelicPluralRule
	case "cy": // Welsh - 6 forms
		return welshPluralRule
	case "br": // Breton - 5 forms
		return bretonPluralRule
	case "mt": // Maltese - 4 forms
		return maltesePluralRule
	case "ro": // Romanian - 3 forms
		return romanianPluralRule
	case "hr", "sr", "bs": // Croatian, Serbian, Bosnian - 3 forms
		return serboCroatianPluralRule
	case "uk": // Ukrainian - 3 forms
		return ukrainianPluralRule
	case "be": // Belarusian - 3 forms
		return belarusianPluralRule
	case "mk": // Macedonian - 2 forms with special rule
		return macedonianPluralRule
	case "sl": // Slovenian - 4 forms
		return slovenianPluralRule
	case "hu": // Hungarian - 2 forms
		return hungarianPluralRule
	case "fi": // Finnish - 2 forms
		return finnishPluralRule
	case "et": // Estonian - 2 forms
		return estonianPluralRule
	case "tr": // Turkish - 2 forms
		return turkishPluralRule
	case "ja", "ko", "zh", "zh-CN", "zh-TW", "th", "vi", "id", "ms": // No pluralization
		return noPluralRule
	default: // English and similar (2 forms)
		return englishPluralRule
	}
}

// Pluralization rules for different languages

// noPluralRule - Languages with no plural distinction
func noPluralRule(count int) string {
	return "other"
}

// englishPluralRule - English, German, Dutch, Swedish, etc.
func englishPluralRule(count int) string {
	if count == 1 {
		return "one"
	}
	return "other"
}

// russianPluralRule - Russian pluralization
func russianPluralRule(count int) string {
	n := count % 100
	if n >= 11 && n <= 14 {
		return "many"
	}
	switch count % 10 {
	case 1:
		return "one"
	case 2, 3, 4:
		return "few"
	default:
		return "many"
	}
}

// polishPluralRule - Polish pluralization
func polishPluralRule(count int) string {
	if count == 1 {
		return "one"
	}
	n := count % 100
	if n >= 10 && n <= 20 {
		return "many"
	}
	switch count % 10 {
	case 2, 3, 4:
		return "few"
	default:
		return "many"
	}
}

// czechSlovakPluralRule - Czech and Slovak
func czechSlovakPluralRule(count int) string {
	if count == 1 {
		return "one"
	}
	if count >= 2 && count <= 4 {
		return "few"
	}
	return "other"
}

// lithuanianPluralRule - Lithuanian
func lithuanianPluralRule(count int) string {
	n := count % 100
	if n >= 11 && n <= 19 {
		return "other"
	}
	switch count % 10 {
	case 1:
		return "one"
	case 2, 3, 4, 5, 6, 7, 8, 9:
		return "few"
	default:
		return "other"
	}
}

// latvianPluralRule - Latvian
func latvianPluralRule(count int) string {
	if count%10 == 1 && count%100 != 11 {
		return "one"
	}
	if count != 0 {
		return "other"
	}
	return "zero"
}

// arabicPluralRule - Arabic (complex 6-form system)
func arabicPluralRule(count int) string {
	if count == 0 {
		return "zero"
	}
	if count == 1 {
		return "one"
	}
	if count == 2 {
		return "two"
	}
	if count%100 >= 3 && count%100 <= 10 {
		return "few"
	}
	if count%100 >= 11 && count%100 <= 99 {
		return "many"
	}
	return "other"
}

// irishPluralRule - Irish Gaelic
func irishPluralRule(count int) string {
	if count == 1 {
		return "one"
	}
	if count == 2 {
		return "two"
	}
	if count >= 3 && count <= 6 {
		return "few"
	}
	if count >= 7 && count <= 10 {
		return "many"
	}
	return "other"
}

// scottishGaelicPluralRule - Scottish Gaelic
func scottishGaelicPluralRule(count int) string {
	if count == 1 || count == 11 {
		return "one"
	}
	if count == 2 || count == 12 {
		return "two"
	}
	if (count >= 3 && count <= 10) || (count >= 13 && count <= 19) {
		return "few"
	}
	return "other"
}

// welshPluralRule - Welsh
func welshPluralRule(count int) string {
	if count == 0 {
		return "zero"
	}
	if count == 1 {
		return "one"
	}
	if count == 2 {
		return "two"
	}
	if count == 3 {
		return "few"
	}
	if count == 6 {
		return "many"
	}
	return "other"
}

// bretonPluralRule - Breton
func bretonPluralRule(count int) string {
	if count == 1 {
		return "one"
	}
	if count == 2 {
		return "two"
	}
	if count == 3 {
		return "few"
	}
	if count == 6 {
		return "many"
	}
	return "other"
}

// maltesePluralRule - Maltese
func maltesePluralRule(count int) string {
	if count == 1 {
		return "one"
	}
	if count == 0 || (count%100 >= 2 && count%100 <= 10) {
		return "few"
	}
	if count%100 >= 11 && count%100 <= 19 {
		return "many"
	}
	return "other"
}

// romanianPluralRule - Romanian
func romanianPluralRule(count int) string {
	if count == 1 {
		return "one"
	}
	if count == 0 || (count%100 >= 1 && count%100 <= 19) {
		return "few"
	}
	return "other"
}

// serboCroatianPluralRule - Serbian, Croatian, Bosnian
func serboCroatianPluralRule(count int) string {
	n := count % 100
	if n >= 11 && n <= 14 {
		return "other"
	}
	switch count % 10 {
	case 1:
		return "one"
	case 2, 3, 4:
		return "few"
	default:
		return "other"
	}
}

// ukrainianPluralRule - Ukrainian
func ukrainianPluralRule(count int) string {
	n := count % 100
	if n >= 11 && n <= 14 {
		return "many"
	}
	switch count % 10 {
	case 1:
		return "one"
	case 2, 3, 4:
		return "few"
	default:
		return "many"
	}
}

// belarusianPluralRule - Belarusian
func belarusianPluralRule(count int) string {
	n := count % 100
	if n >= 11 && n <= 14 {
		return "many"
	}
	switch count % 10 {
	case 1:
		return "one"
	case 2, 3, 4:
		return "few"
	default:
		return "many"
	}
}

// macedonianPluralRule - Macedonian
func macedonianPluralRule(count int) string {
	if count%10 == 1 && count != 11 {
		return "one"
	}
	return "other"
}

// slovenianPluralRule - Slovenian
func slovenianPluralRule(count int) string {
	n := count % 100
	if n == 1 {
		return "one"
	}
	if n == 2 {
		return "two"
	}
	if n == 3 || n == 4 {
		return "few"
	}
	return "other"
}

// hungarianPluralRule - Hungarian
func hungarianPluralRule(count int) string {
	if count == 1 {
		return "one"
	}
	return "other"
}

// finnishPluralRule - Finnish
func finnishPluralRule(count int) string {
	if count == 1 {
		return "one"
	}
	return "other"
}

// estonianPluralRule - Estonian
func estonianPluralRule(count int) string {
	if count == 1 {
		return "one"
	}
	return "other"
}

// turkishPluralRule - Turkish
func turkishPluralRule(count int) string {
	if count == 1 {
		return "one"
	}
	return "other"
}

// Helper functions for common use cases

// T is a global function for easy translation access
func T(key string, params ...TranslationParams) string {
	manager := GetI18nManager()
	return manager.T(key, params...)
}

// TPlural is a global function for pluralization
func TPlural(key string, count int, params ...TranslationParams) string {
	manager := GetI18nManager()
	return manager.TPlural(key, count, params...)
}

// SetLocale is a global function to change locale
func SetLocale(locale string) error {
	manager := GetI18nManager()
	return manager.SetLocale(locale)
}

// GetCurrentLocale is a global function to get current locale
func GetCurrentLocale() string {
	manager := GetI18nManager()
	return manager.GetCurrentLocale()
}

// GetAvailableLocales is a global function to get available locales
func GetAvailableLocales() []string {
	manager := GetI18nManager()
	return manager.GetAvailableLocales()
}

// Initialize sets up the i18n system
func (i *I18nManager) Initialize() error {
	// Check for persistent configuration first
	cm := GetConfigManager()
	if cm != nil {
		if config := cm.GetI18nConfig(); config != nil {
			if config.Locale != "" {
				if err := i.SetLocale(config.Locale); err == nil {
					i.fallbackLocale = config.FallbackLocale
					return nil
				}
			}
		}
	}

	// Try to detect system locale if auto-detection is enabled
	cm = GetConfigManager()
	if cm != nil {
		if config := cm.GetI18nConfig(); config != nil && config.AutoDetectLanguage {
			if locale := detectSystemLocale(); locale != "" {
				if err := i.SetLocale(locale); err == nil {
					return nil
				}
			}
		}
	} else {
		// Fallback: try system detection if no config available
		if locale := detectSystemLocale(); locale != "" {
			if err := i.SetLocale(locale); err == nil {
				return nil
			}
		}
	}

	// Fall back to English
	return i.SetLocale("en")
}

// detectSystemLocale attempts to detect the system locale
func detectSystemLocale() string {
	// Check common environment variables
	localeVars := []string{"LC_ALL", "LC_MESSAGES", "LANG", "LANGUAGE"}

	for _, envVar := range localeVars {
		if locale := os.Getenv(envVar); locale != "" {
			// Extract just the language code (e.g., "en_US.UTF-8" -> "en")
			if parts := strings.Split(locale, "_"); len(parts) > 0 {
				lang := parts[0]
				if lang != "" && lang != "C" && lang != "POSIX" {
					return lang
				}
			}
		}
	}

	return ""
}

// ReloadTranslations reloads all translation files
func (i *I18nManager) ReloadTranslations() error {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	// Clear loaded files cache
	i.loadedFiles = make(map[string]bool)
	i.translations = make(map[string]map[string]interface{})

	// Reload current locale
	return i.LoadLocale(i.currentLocale)
}

// GetTranslationStats returns statistics about loaded translations
func (i *I18nManager) GetTranslationStats() map[string]interface{} {
	i.mutex.RLock()
	defer i.mutex.RUnlock()

	stats := map[string]interface{}{
		"current_locale":    i.currentLocale,
		"fallback_locale":   i.fallbackLocale,
		"loaded_locales":    len(i.translations),
		"available_locales": i.GetAvailableLocales(),
	}

	// Count total translation keys
	totalKeys := 0
	for locale, data := range i.translations {
		count := i.countKeys(data)
		stats[fmt.Sprintf("%s_keys", locale)] = count
		totalKeys += count
	}

	stats["total_keys"] = totalKeys
	return stats
}

// countKeys recursively counts the number of translation keys
func (i *I18nManager) countKeys(data map[string]interface{}) int {
	count := 0
	for _, value := range data {
		switch v := value.(type) {
		case map[string]interface{}:
			count += i.countKeys(v)
		case string:
			count++
		}
	}
	return count
}
