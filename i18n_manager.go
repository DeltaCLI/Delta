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
	currentLocale    string
	fallbackLocale   string
	translations     map[string]map[string]interface{}
	loadedFiles      map[string]bool
	mutex            sync.RWMutex
	basePath         string
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

// NewI18nManager creates a new internationalization manager
func NewI18nManager() *I18nManager {
	manager := &I18nManager{
		currentLocale:  "en",
		fallbackLocale: "en",
		translations:   make(map[string]map[string]interface{}),
		loadedFiles:    make(map[string]bool),
		basePath:       "i18n/locales",
	}
	
	// Try to load default locale
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

// SetLocale changes the current locale
func (i *I18nManager) SetLocale(locale string) error {
	// Try to load the locale if not already loaded
	if err := i.LoadLocale(locale); err != nil {
		return err
	}
	
	i.mutex.Lock()
	i.currentLocale = locale
	i.mutex.Unlock()
	
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

// TPlural handles pluralization (simplified version)
func (i *I18nManager) TPlural(key string, count int, params ...TranslationParams) string {
	// Simple English pluralization rules
	var pluralKey string
	if count == 1 {
		pluralKey = key + ".singular"
	} else {
		pluralKey = key + ".plural"
	}
	
	// Try plural key first
	if translation := i.T(pluralKey, params...); translation != pluralKey {
		return translation
	}
	
	// Fall back to base key
	return i.T(key, params...)
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
	// Try to detect system locale
	if locale := detectSystemLocale(); locale != "" {
		if err := i.SetLocale(locale); err == nil {
			return nil
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
		"current_locale":     i.currentLocale,
		"fallback_locale":    i.fallbackLocale,
		"loaded_locales":     len(i.translations),
		"available_locales":  i.GetAvailableLocales(),
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