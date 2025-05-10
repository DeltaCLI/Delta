package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// JumpManager handles directory jumping functionality
type JumpManager struct {
	// Map of location shortcuts to absolute paths
	locations map[string]string
	// Path to the config file
	configPath string
}

// NewJumpManager creates a new jump manager with location shortcuts
func NewJumpManager() *JumpManager {
	// Set up config directory in user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = os.Getenv("HOME")
	}
	
	// Use ~/.config/delta directory for config file
	configDir := filepath.Join(homeDir, ".config", "delta")
	err = os.MkdirAll(configDir, 0755)
	if err != nil {
		// Fall back to home directory if we can't create .config/delta
		configDir = homeDir
	}
	
	configPath := filepath.Join(configDir, "jump_locations.json")
	
	// Create initial JumpManager instance
	jm := &JumpManager{
		locations:  make(map[string]string),
		configPath: configPath,
	}
	
	// Try to load saved locations
	err = jm.loadLocations()
	if err != nil || len(jm.locations) == 0 {
		// Initialize with minimal defaults if loading fails or no locations saved
		jm.locations = map[string]string{
			"delta": "/home/bleepbloop/deltacli",
			"home":  homeDir,
		}
		
		// Try to import locations from jump.sh
		jm.importLocationsFromJumpSh()
		
		// Save the locations
		jm.saveLocations()
	}

	return jm
}

// loadLocations loads saved jump locations from the config file
func (jm *JumpManager) loadLocations() error {
	// Check if file exists
	_, err := os.Stat(jm.configPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("config file does not exist")
	}
	
	// Read the file
	data, err := ioutil.ReadFile(jm.configPath)
	if err != nil {
		return err
	}
	
	// Unmarshal the JSON data
	return json.Unmarshal(data, &jm.locations)
}

// saveLocations saves the jump locations to the config file
func (jm *JumpManager) saveLocations() error {
	// Marshal the locations to JSON with indentation for readability
	data, err := json.MarshalIndent(jm.locations, "", "  ")
	if err != nil {
		return err
	}
	
	// Create directory if it doesn't exist
	dir := filepath.Dir(jm.configPath)
	if err = os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	
	// Write to file
	return ioutil.WriteFile(jm.configPath, data, 0644)
}

// Jump changes the current directory to the specified location
func (jm *JumpManager) Jump(location string) (string, error) {
	// Look up the location
	path, exists := jm.locations[location]
	if !exists {
		return "", fmt.Errorf("unknown location: %s", location)
	}

	// Change directory
	err := os.Chdir(path)
	if err != nil {
		return "", fmt.Errorf("failed to jump to %s: %v", location, err)
	}

	return path, nil
}

// ListLocations returns a list of all available jump locations
func (jm *JumpManager) ListLocations() []string {
	var locations []string
	for loc := range jm.locations {
		locations = append(locations, loc)
	}
	sort.Strings(locations)
	return locations
}

// AddLocation adds a new jump location
func (jm *JumpManager) AddLocation(name, path string) error {
	jm.locations[name] = path
	return jm.saveLocations()
}

// RemoveLocation removes a jump location
func (jm *JumpManager) RemoveLocation(name string) error {
	if _, exists := jm.locations[name]; !exists {
		return fmt.Errorf("location '%s' not found", name)
	}
	
	delete(jm.locations, name)
	return jm.saveLocations()
}

// importLocationsFromJumpSh tries to import locations from the jump.sh script
func (jm *JumpManager) importLocationsFromJumpSh() {
	jumpshPath := "/home/bleepbloop/black/bin/jump.sh"
	
	// Check if jump.sh exists
	_, err := os.Stat(jumpshPath)
	if os.IsNotExist(err) {
		return
	}
	
	// Read the jump.sh file
	data, err := ioutil.ReadFile(jumpshPath)
	if err != nil {
		return
	}
	
	// Convert to string and split into lines
	content := string(data)
	lines := strings.Split(content, "\n")
	
	// Process jump.sh file
	processJumpShFile(jm, lines)
}

// processJumpShFile extracts jump locations from the jump.sh file
func processJumpShFile(jm *JumpManager, lines []string) {
	// Look for location definitions in the Jump Gate section
	inJumpGate := false
	
	for i, line := range lines {
		line = strings.TrimSpace(line)
		
		// Find the Jump Gate section
		if strings.Contains(line, "Jump Gate") {
			inJumpGate = true
			continue
		}
		
		// Process the Jump Gate section
		if inJumpGate {
			// Look for case/elif statements like: elif [[ ${query} == "delta" ]]; then
			if strings.Contains(line, "if [[") || strings.Contains(line, "elif [[") {
				if strings.Contains(line, "${query}") && strings.Contains(line, "==") {
					// Extract the location name
					parts := strings.Split(line, "==")
					if len(parts) >= 2 {
						name := strings.Trim(strings.TrimSpace(parts[1]), "\"' ];")
						
						// Look for the jumpto call on the next line
						if i+1 < len(lines) {
							nextLine := strings.TrimSpace(lines[i+1])
							if strings.HasPrefix(nextLine, "jumpto") {
								// Extract the path
								pathParts := strings.Split(nextLine, "jumpto")
								if len(pathParts) >= 2 {
									// Clean up the path
									path := strings.Trim(strings.TrimSpace(pathParts[1]), "\"' $)(||return")
									
									// Convert any home directory references
									if path == "~" || path == "$HOME" {
										home, _ := os.UserHomeDir()
										path = home
									} else if strings.HasPrefix(path, "~/") {
										home, _ := os.UserHomeDir()
										path = filepath.Join(home, path[2:])
									} else if strings.HasPrefix(path, "$HOME/") {
										home, _ := os.UserHomeDir()
										path = filepath.Join(home, path[6:])
									}
									
									// Skip if it still contains variables
									if !strings.Contains(path, "$") {
										jm.locations[name] = path
									}
								}
							}
						}
					}
				}
			}
			
			// End of Jump Gate section
			if strings.Contains(line, "###############################################################################") && 
			   i > 0 && strings.Contains(lines[i-1], "fi") {
				inJumpGate = false
			}
		}
	}
}

// HandleJumpCommand processes the jump command with arguments
func HandleJumpCommand(args []string) bool {
	// Get the JumpManager instance
	jm := GetJumpManager()

	// Handle commands
	if len(args) == 0 {
		// List available locations
		locs := jm.ListLocations()
		fmt.Println("Available jump locations:")
		for _, loc := range locs {
			fmt.Printf("  %s -> %s\n", loc, jm.locations[loc])
		}
		return true
	}

	// Handle special commands
	if len(args) >= 1 {
		cmd := args[0]
		
		// Handle add command
		if cmd == "add" {
			if len(args) >= 3 {
				// Add with specified path
				name := args[1]
				path := args[2]
				
				// Expand ~ and $HOME in paths
				if path == "~" || path == "$HOME" {
					home, _ := os.UserHomeDir()
					path = home
				} else if strings.HasPrefix(path, "~/") {
					home, _ := os.UserHomeDir()
					path = filepath.Join(home, path[2:])
				} else if strings.HasPrefix(path, "$HOME/") {
					home, _ := os.UserHomeDir()
					path = filepath.Join(home, path[6:])
				}
				
				err := jm.AddLocation(name, path)
				if err != nil {
					fmt.Printf("Error adding location: %v\n", err)
				} else {
					fmt.Printf("Added jump location: %s -> %s\n", name, path)
				}
				return true
			} else if len(args) == 2 {
				// Add current directory
				name := args[1]
				path, err := os.Getwd()
				if err != nil {
					fmt.Println("Error getting current directory:", err)
					return true
				}
				err = jm.AddLocation(name, path)
				if err != nil {
					fmt.Printf("Error adding location: %v\n", err)
				} else {
					fmt.Printf("Added jump location: %s -> %s\n", name, path)
				}
				return true
			} else {
				fmt.Println("Usage: jump add <name> [path]")
				return true
			}
		}
		
		// Handle remove command
		if cmd == "remove" || cmd == "rm" {
			if len(args) >= 2 {
				name := args[1]
				err := jm.RemoveLocation(name)
				if err != nil {
					fmt.Printf("Error: %v\n", err)
				} else {
					fmt.Printf("Removed jump location: %s\n", name)
				}
				return true
			} else {
				fmt.Println("Usage: jump remove <name>")
				return true
			}
		}
		
		// Handle import command for jump.sh
		if cmd == "import" {
			if len(args) >= 2 && args[1] == "jumpsh" {
				count := len(jm.locations)
				jm.importLocationsFromJumpSh()
				newCount := len(jm.locations)
				fmt.Printf("Imported %d locations from jump.sh\n", newCount-count)
				jm.saveLocations()
				return true
			}
		}
		
		// Normal jump operation
		location := args[0]
		path, err := jm.Jump(location)
		if err != nil {
			fmt.Println(err)
			return true
		}

		// Success message
		fmt.Printf("Jumped to: %s\n", path)
		return true
	}

	return true
}

// Global JumpManager instance
var globalJumpManager *JumpManager

// GetJumpManager returns the global JumpManager instance
func GetJumpManager() *JumpManager {
	if globalJumpManager == nil {
		globalJumpManager = NewJumpManager()
	}
	return globalJumpManager
}