package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// PatternVersionInfo stores version information for pattern files
type PatternVersionInfo struct {
	Version   string `json:"version"`
	UpdatedAt string `json:"updated_at"`
	Size      int    `json:"size"`
}

// PatternVersionResponse represents the API response for pattern versions
type PatternVersionResponse struct {
	Patterns struct {
		ErrorPatterns  PatternVersionInfo `json:"error_patterns"`
		CommonCommands PatternVersionInfo `json:"common_commands"`
	} `json:"patterns"`
}

// PatternUpdateManager handles updating pattern files from the API
type PatternUpdateManager struct {
	apiBaseURL      string
	lastUpdateCheck time.Time
	checkInterval   time.Duration
	patternDir      string
	embeddedDir     string
	isInitialized   bool
}

// PatternUpdateConfig holds the configuration for the pattern update system
type PatternUpdateConfig struct {
	Enabled              bool   `json:"enabled"`
	AutoUpdate           bool   `json:"auto_update"`
	APIBaseURL           string `json:"api_base_url"`
	UpdateCheckInterval  int    `json:"update_check_interval"` // In hours
	LastUpdateCheck      string `json:"last_update_check"`
	ErrorPatternsVersion string `json:"error_patterns_version"`
	CommandsVersion      string `json:"commands_version"`
}

// NewPatternUpdateManager creates a new pattern update manager
func NewPatternUpdateManager() (*PatternUpdateManager, error) {
	// Get pattern directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}
	
	patternDir := filepath.Join(homeDir, ".config", "delta", "patterns")
	if err := os.MkdirAll(patternDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create pattern directory: %v", err)
	}
	
	// Find embedded patterns directory
	execDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	embeddedDir := filepath.Join(execDir, "embedded_patterns")
	if _, err := os.Stat(embeddedDir); os.IsNotExist(err) {
		// Try alternative paths
		alternativePaths := []string{
			"/home/bleepbloop/deltacli/embedded_patterns",
			"/usr/local/share/delta/embedded_patterns",
			"/usr/share/delta/embedded_patterns",
		}
		
		for _, path := range alternativePaths {
			if _, err := os.Stat(path); err == nil {
				embeddedDir = path
				break
			}
		}
	}
	
	return &PatternUpdateManager{
		apiBaseURL:      "https://api.delta-cli.org",
		lastUpdateCheck: time.Time{},
		checkInterval:   time.Hour * 24, // Default to daily checks
		patternDir:      patternDir,
		embeddedDir:     embeddedDir,
		isInitialized:   false,
	}, nil
}

// Initialize initializes the pattern update manager
func (pm *PatternUpdateManager) Initialize() error {
	// Load configuration or create default config
	config, err := pm.loadConfig()
	if err != nil {
		// Create default config
		config = PatternUpdateConfig{
			Enabled:              true,
			AutoUpdate:           true,
			APIBaseURL:           "https://api.delta-cli.org",
			UpdateCheckInterval:  24, // Daily check
			LastUpdateCheck:      time.Now().Format(time.RFC3339),
			ErrorPatternsVersion: "1.0",
			CommandsVersion:      "1.0",
		}
		
		// Save default config
		if err := pm.saveConfig(config); err != nil {
			return fmt.Errorf("failed to save default config: %v", err)
		}
	}
	
	// Update configuration
	pm.apiBaseURL = config.APIBaseURL
	pm.checkInterval = time.Duration(config.UpdateCheckInterval) * time.Hour
	
	if t, err := time.Parse(time.RFC3339, config.LastUpdateCheck); err == nil {
		pm.lastUpdateCheck = t
	}
	
	// Copy embedded patterns to user directory if they don't exist
	if err := pm.initializePatternFiles(); err != nil {
		return fmt.Errorf("failed to initialize pattern files: %v", err)
	}
	
	// If auto update is enabled and it's time to check, do it in the background
	if config.Enabled && config.AutoUpdate && time.Since(pm.lastUpdateCheck) > pm.checkInterval {
		go func() {
			if updatesAvailable, _ := pm.CheckForUpdates(); updatesAvailable {
				pm.DownloadUpdates()
			}
		}()
	}
	
	pm.isInitialized = true
	return nil
}

// initializePatternFiles copies embedded patterns to user directory if needed
func (pm *PatternUpdateManager) initializePatternFiles() error {
	// Check for error patterns
	errorPatternsPath := filepath.Join(pm.patternDir, "error_patterns.json")
	if _, err := os.Stat(errorPatternsPath); os.IsNotExist(err) {
		// Copy from embedded
		embeddedPath := filepath.Join(pm.embeddedDir, "error_patterns.json")
		if err := pm.copyFile(embeddedPath, errorPatternsPath); err != nil {
			return fmt.Errorf("failed to copy error patterns: %v", err)
		}
	}
	
	// Check for common commands
	commandsPath := filepath.Join(pm.patternDir, "common_commands.json")
	if _, err := os.Stat(commandsPath); os.IsNotExist(err) {
		// Copy from embedded
		embeddedPath := filepath.Join(pm.embeddedDir, "common_commands.json")
		if err := pm.copyFile(embeddedPath, commandsPath); err != nil {
			return fmt.Errorf("failed to copy common commands: %v", err)
		}
	}
	
	return nil
}

// copyFile copies a file from src to dst
func (pm *PatternUpdateManager) copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	
	_, err = io.Copy(out, in)
	return err
}

// loadConfig loads the pattern update configuration
func (pm *PatternUpdateManager) loadConfig() (PatternUpdateConfig, error) {
	configPath := filepath.Join(pm.patternDir, "config.json")
	
	data, err := os.ReadFile(configPath)
	if err != nil {
		return PatternUpdateConfig{}, err
	}
	
	var config PatternUpdateConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return PatternUpdateConfig{}, err
	}
	
	return config, nil
}

// saveConfig saves the pattern update configuration
func (pm *PatternUpdateManager) saveConfig(config PatternUpdateConfig) error {
	configPath := filepath.Join(pm.patternDir, "config.json")
	
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(configPath, data, 0644)
}

// CheckForUpdates checks if pattern updates are available
func (pm *PatternUpdateManager) CheckForUpdates() (bool, error) {
	if !pm.isInitialized {
		return false, fmt.Errorf("pattern update manager not initialized")
	}
	
	// Update last check time
	pm.lastUpdateCheck = time.Now()
	
	// Update config with new check time
	config, err := pm.loadConfig()
	if err == nil {
		config.LastUpdateCheck = pm.lastUpdateCheck.Format(time.RFC3339)
		pm.saveConfig(config)
	}
	
	// Get current versions
	currentVersions, err := pm.getCurrentVersions()
	if err != nil {
		return false, fmt.Errorf("failed to get current versions: %v", err)
	}
	
	// Get latest versions from API
	latestVersions, err := pm.getLatestVersions()
	if err != nil {
		return false, fmt.Errorf("failed to get latest versions: %v", err)
	}
	
	// Check if updates are available
	errorPatternsNeedsUpdate := latestVersions.Patterns.ErrorPatterns.Version != currentVersions.ErrorPatterns
	commandsNeedsUpdate := latestVersions.Patterns.CommonCommands.Version != currentVersions.CommandsVersion
	
	return errorPatternsNeedsUpdate || commandsNeedsUpdate, nil
}

// DownloadUpdates downloads the latest pattern files
func (pm *PatternUpdateManager) DownloadUpdates() error {
	if !pm.isInitialized {
		return fmt.Errorf("pattern update manager not initialized")
	}
	
	// Get latest versions from API
	latestVersions, err := pm.getLatestVersions()
	if err != nil {
		return fmt.Errorf("failed to get latest versions: %v", err)
	}
	
	// Get current versions
	currentVersions, err := pm.getCurrentVersions()
	if err != nil {
		return fmt.Errorf("failed to get current versions: %v", err)
	}
	
	updatedFiles := 0
	
	// Download error patterns if needed
	if latestVersions.Patterns.ErrorPatterns.Version != currentVersions.ErrorPatterns {
		if err := pm.downloadFile("error_patterns.json", latestVersions.Patterns.ErrorPatterns.Version); err != nil {
			return fmt.Errorf("failed to download error patterns: %v", err)
		}
		fmt.Println("Updated error patterns to version", latestVersions.Patterns.ErrorPatterns.Version)
		updatedFiles++
		
		// Update config with new version
		config, _ := pm.loadConfig()
		config.ErrorPatternsVersion = latestVersions.Patterns.ErrorPatterns.Version
		pm.saveConfig(config)
	}
	
	// Download common commands if needed
	if latestVersions.Patterns.CommonCommands.Version != currentVersions.CommandsVersion {
		if err := pm.downloadFile("common_commands.json", latestVersions.Patterns.CommonCommands.Version); err != nil {
			return fmt.Errorf("failed to download common commands: %v", err)
		}
		fmt.Println("Updated common commands to version", latestVersions.Patterns.CommonCommands.Version)
		updatedFiles++
		
		// Update config with new version
		config, _ := pm.loadConfig()
		config.CommandsVersion = latestVersions.Patterns.CommonCommands.Version
		pm.saveConfig(config)
	}
	
	if updatedFiles == 0 {
		fmt.Println("All pattern files are already up to date.")
	}
	
	return nil
}

// getCurrentVersions gets the current versions from the config
func (pm *PatternUpdateManager) getCurrentVersions() (struct{ErrorPatterns, CommandsVersion string}, error) {
	config, err := pm.loadConfig()
	if err != nil {
		return struct{ErrorPatterns, CommandsVersion string}{}, err
	}
	
	return struct{ErrorPatterns, CommandsVersion string}{
		ErrorPatterns: config.ErrorPatternsVersion,
		CommandsVersion: config.CommandsVersion,
	}, nil
}

// getLatestVersions gets the latest versions from the API
func (pm *PatternUpdateManager) getLatestVersions() (*PatternVersionResponse, error) {
	// Build API URL
	url := fmt.Sprintf("%s/api/patterns/versions", pm.apiBaseURL)
	
	// Send request
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()
	
	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}
	
	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}
	
	// Parse response
	var versions PatternVersionResponse
	if err := json.Unmarshal(body, &versions); err != nil {
		return nil, fmt.Errorf("failed to parse response: %v", err)
	}
	
	return &versions, nil
}

// downloadFile downloads a pattern file from the API
func (pm *PatternUpdateManager) downloadFile(fileName, version string) error {
	// Build API URL
	url := fmt.Sprintf("%s/api/patterns/%s?version=%s", pm.apiBaseURL, fileName, version)
	
	// Send request
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()
	
	// Check response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}
	
	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}
	
	// Write to file
	filePath := filepath.Join(pm.patternDir, fileName)
	if err := os.WriteFile(filePath, body, 0644); err != nil {
		return fmt.Errorf("failed to write file: %v", err)
	}
	
	return nil
}

// IsEnabled returns whether pattern updates are enabled
func (pm *PatternUpdateManager) IsEnabled() bool {
	if !pm.isInitialized {
		return false
	}
	
	config, err := pm.loadConfig()
	if err != nil {
		return false
	}
	
	return config.Enabled
}

// Enable enables pattern updates
func (pm *PatternUpdateManager) Enable() error {
	if !pm.isInitialized {
		return fmt.Errorf("pattern update manager not initialized")
	}
	
	config, err := pm.loadConfig()
	if err != nil {
		return err
	}
	
	config.Enabled = true
	return pm.saveConfig(config)
}

// Disable disables pattern updates
func (pm *PatternUpdateManager) Disable() error {
	if !pm.isInitialized {
		return fmt.Errorf("pattern update manager not initialized")
	}
	
	config, err := pm.loadConfig()
	if err != nil {
		return err
	}
	
	config.Enabled = false
	return pm.saveConfig(config)
}

// EnableAutoUpdate enables automatic updates
func (pm *PatternUpdateManager) EnableAutoUpdate() error {
	if !pm.isInitialized {
		return fmt.Errorf("pattern update manager not initialized")
	}
	
	config, err := pm.loadConfig()
	if err != nil {
		return err
	}
	
	config.AutoUpdate = true
	return pm.saveConfig(config)
}

// DisableAutoUpdate disables automatic updates
func (pm *PatternUpdateManager) DisableAutoUpdate() error {
	if !pm.isInitialized {
		return fmt.Errorf("pattern update manager not initialized")
	}
	
	config, err := pm.loadConfig()
	if err != nil {
		return err
	}
	
	config.AutoUpdate = false
	return pm.saveConfig(config)
}

// UpdateCheckInterval updates the check interval
func (pm *PatternUpdateManager) UpdateCheckInterval(hours int) error {
	if !pm.isInitialized {
		return fmt.Errorf("pattern update manager not initialized")
	}
	
	if hours <= 0 {
		return fmt.Errorf("check interval must be positive")
	}
	
	config, err := pm.loadConfig()
	if err != nil {
		return err
	}
	
	config.UpdateCheckInterval = hours
	pm.checkInterval = time.Duration(hours) * time.Hour
	
	return pm.saveConfig(config)
}

// GetStats returns statistics about pattern files
func (pm *PatternUpdateManager) GetStats() (map[string]interface{}, error) {
	if !pm.isInitialized {
		return nil, fmt.Errorf("pattern update manager not initialized")
	}
	
	stats := make(map[string]interface{})
	
	// Get current versions
	config, err := pm.loadConfig()
	if err != nil {
		return nil, err
	}
	
	stats["enabled"] = config.Enabled
	stats["auto_update"] = config.AutoUpdate
	stats["api_base_url"] = config.APIBaseURL
	stats["update_check_interval"] = config.UpdateCheckInterval
	stats["last_update_check"] = config.LastUpdateCheck
	stats["error_patterns_version"] = config.ErrorPatternsVersion
	stats["commands_version"] = config.CommandsVersion
	
	// Check if files exist
	errorPatternsPath := filepath.Join(pm.patternDir, "error_patterns.json")
	errorPatternsExists := fileExists(errorPatternsPath)
	stats["error_patterns_exists"] = errorPatternsExists
	
	commandsPath := filepath.Join(pm.patternDir, "common_commands.json")
	commandsExists := fileExists(commandsPath)
	stats["commands_exists"] = commandsExists
	
	// Get file sizes
	if errorPatternsExists {
		if info, err := os.Stat(errorPatternsPath); err == nil {
			stats["error_patterns_size"] = info.Size()
		}
	}
	
	if commandsExists {
		if info, err := os.Stat(commandsPath); err == nil {
			stats["commands_size"] = info.Size()
		}
	}
	
	// Get pattern counts
	if errorPatternsExists {
		data, err := os.ReadFile(errorPatternsPath)
		if err == nil {
			var patterns struct {
				Patterns []interface{} `json:"patterns"`
			}
			if err := json.Unmarshal(data, &patterns); err == nil {
				stats["error_patterns_count"] = len(patterns.Patterns)
			}
		}
	}
	
	if commandsExists {
		data, err := os.ReadFile(commandsPath)
		if err == nil {
			var patterns struct {
				Commands []interface{} `json:"commands"`
			}
			if err := json.Unmarshal(data, &patterns); err == nil {
				stats["commands_count"] = len(patterns.Commands)
			}
		}
	}
	
	return stats, nil
}

// Global instance of the pattern update manager
var globalPatternUpdateManager *PatternUpdateManager

// GetPatternUpdateManager returns the global pattern update manager
func GetPatternUpdateManager() *PatternUpdateManager {
	if globalPatternUpdateManager == nil {
		var err error
		globalPatternUpdateManager, err = NewPatternUpdateManager()
		if err != nil {
			fmt.Printf("Error creating pattern update manager: %v\n", err)
			return nil
		}
		
		// Initialize in the background to avoid blocking
		go func() {
			if err := globalPatternUpdateManager.Initialize(); err != nil {
				fmt.Printf("Error initializing pattern update manager: %v\n", err)
			}
		}()
	}
	
	return globalPatternUpdateManager
}

// patternFileExists checks if a pattern file exists
func patternFileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}