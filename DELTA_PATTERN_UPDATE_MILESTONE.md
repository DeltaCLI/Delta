# Delta Pattern Update Milestone

## Overview

This milestone aims to implement an update system for error patterns and common commands. The system will periodically check for updates from a central API and download the latest pattern files to improve error detection, automated fixes, and command suggestions.

## Implementation Details

### API Endpoint Requirements

1. **Versioned Pattern Files**:
   - `/api/patterns/error_patterns.json?version={version}`
   - `/api/patterns/common_commands.json?version={version}`

2. **Version Check Endpoint**:
   - `/api/patterns/versions` - Returns the latest version numbers for all pattern files

### Implementation Tasks

1. **Pattern Update Manager**
   - Add a new component `PatternUpdateManager` responsible for checking and downloading updates
   - Implement version checking logic to minimize unnecessary downloads
   - Add configuration options for update frequency and source URL

2. **Integration with Agent Manager**
   - Modify `AgentManager` to use the `PatternUpdateManager` for pattern updates
   - Add hooks in the `Initialize` method to check for updates
   - Implement update checking during application startup

3. **User Commands**
   - Add user commands to manually trigger pattern updates:
     - `:pattern update` - Force update of pattern files
     - `:pattern versions` - Show current and latest versions of pattern files
     - `:pattern list` - List all available patterns

4. **Security Considerations**
   - Implement signature verification for downloaded patterns
   - Add checksums to ensure data integrity
   - Support for HTTPS endpoints only

### API Data Format

The version API should return JSON in the following format:

```json
{
  "patterns": {
    "error_patterns": {
      "version": "1.1",
      "updated_at": "2025-06-15",
      "size": 12345
    },
    "common_commands": {
      "version": "1.2",
      "updated_at": "2025-06-10",
      "size": 8765
    }
  }
}
```

### Code Implementation

The core update functionality will be implemented in a new file `pattern_update.go`:

```go
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
}

// NewPatternUpdateManager creates a new pattern update manager
func NewPatternUpdateManager(apiBaseURL string, checkInterval time.Duration) (*PatternUpdateManager, error) {
	// Get pattern directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}
	
	patternDir := filepath.Join(homeDir, ".delta", "patterns")
	if err := os.MkdirAll(patternDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create pattern directory: %v", err)
	}
	
	return &PatternUpdateManager{
		apiBaseURL:      apiBaseURL,
		lastUpdateCheck: time.Time{},
		checkInterval:   checkInterval,
		patternDir:      patternDir,
	}, nil
}

// CheckForUpdates checks if pattern updates are available
func (pm *PatternUpdateManager) CheckForUpdates() (bool, error) {
	// Skip if we've checked recently
	if time.Since(pm.lastUpdateCheck) < pm.checkInterval {
		return false, nil
	}
	
	pm.lastUpdateCheck = time.Now()
	
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
	commandsNeedsUpdate := latestVersions.Patterns.CommonCommands.Version != currentVersions.CommonCommands
	
	return errorPatternsNeedsUpdate || commandsNeedsUpdate, nil
}

// DownloadUpdates downloads the latest pattern files
func (pm *PatternUpdateManager) DownloadUpdates() error {
	// Get latest versions from API
	latestVersions, err := pm.getLatestVersions()
	if err != nil {
		return fmt.Errorf("failed to get latest versions: %v", err)
	}
	
	// Download error patterns if needed
	errorPatternsPath := filepath.Join(pm.patternDir, "error_patterns.json")
	currentErrorVersion, _ := pm.getFileVersion(errorPatternsPath)
	if currentErrorVersion != latestVersions.Patterns.ErrorPatterns.Version {
		if err := pm.downloadFile("error_patterns.json", latestVersions.Patterns.ErrorPatterns.Version); err != nil {
			return fmt.Errorf("failed to download error patterns: %v", err)
		}
		fmt.Println("Updated error patterns to version", latestVersions.Patterns.ErrorPatterns.Version)
	}
	
	// Download common commands if needed
	commandsPath := filepath.Join(pm.patternDir, "common_commands.json")
	currentCommandsVersion, _ := pm.getFileVersion(commandsPath)
	if currentCommandsVersion != latestVersions.Patterns.CommonCommands.Version {
		if err := pm.downloadFile("common_commands.json", latestVersions.Patterns.CommonCommands.Version); err != nil {
			return fmt.Errorf("failed to download common commands: %v", err)
		}
		fmt.Println("Updated common commands to version", latestVersions.Patterns.CommonCommands.Version)
	}
	
	return nil
}

// getCurrentVersions gets the current versions of pattern files
func (pm *PatternUpdateManager) getCurrentVersions() (struct{ ErrorPatterns, CommonCommands string }, error) {
	result := struct{ ErrorPatterns, CommonCommands string }{"0.0", "0.0"}
	
	// Get error patterns version
	errorPatternsPath := filepath.Join(pm.patternDir, "error_patterns.json")
	version, err := pm.getFileVersion(errorPatternsPath)
	if err == nil {
		result.ErrorPatterns = version
	}
	
	// Get common commands version
	commandsPath := filepath.Join(pm.patternDir, "common_commands.json")
	version, err = pm.getFileVersion(commandsPath)
	if err == nil {
		result.CommonCommands = version
	}
	
	return result, nil
}

// getFileVersion gets the version of a pattern file
func (pm *PatternUpdateManager) getFileVersion(filePath string) (string, error) {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return "", fmt.Errorf("file does not exist: %s", filePath)
	}
	
	// Read the file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %v", err)
	}
	
	// Parse JSON to get version
	var fileData struct {
		Version string `json:"version"`
	}
	
	if err := json.Unmarshal(data, &fileData); err != nil {
		return "", fmt.Errorf("failed to parse file: %v", err)
	}
	
	return fileData.Version, nil
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
```

### Timeline and Priority

This milestone should be implemented after completing the core AI-assisted error solver functionality. It enhances the system by providing a mechanism for continuous improvement through regular updates of error patterns and command definitions.

Priority: Medium
Estimated Completion Time: 2-3 days