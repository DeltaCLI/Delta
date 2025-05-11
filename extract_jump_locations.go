package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	// Path to jump.sh
	jumpshPath := "/home/bleepbloop/black/bin/jump.sh"
	
	// Output file path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error getting home directory: %v\n", err)
		os.Exit(1)
	}
	
	outputPath := filepath.Join(homeDir, ".config", "delta", "jump_locations.json")
	
	// Check if jump.sh exists
	_, err = os.Stat(jumpshPath)
	if os.IsNotExist(err) {
		fmt.Printf("Error: jump.sh not found at %s\n", jumpshPath)
		os.Exit(1)
	}
	
	// Read the jump.sh file
	data, err := ioutil.ReadFile(jumpshPath)
	if err != nil {
		fmt.Printf("Error reading jump.sh: %v\n", err)
		os.Exit(1)
	}
	
	// Extract locations
	locations := extractLocations(string(data))
	
	// Write to output file
	jsonData, err := json.MarshalIndent(locations, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}
	
	err = ioutil.WriteFile(outputPath, jsonData, 0644)
	if err != nil {
		fmt.Printf("Error writing to output file: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("Successfully wrote %d locations to %s\n", len(locations), outputPath)
}

func extractLocations(data string) map[string]string {
	// Map to store locations
	locations := make(map[string]string)

	// Add default locations
	home, _ := os.UserHomeDir()
	locations["home"] = home
	locations["delta"] = "/home/bleepbloop/deltacli"

	// Split into lines
	lines := strings.Split(data, "\n")

	// Look for variable definitions first (for directories)
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Look for variable definitions like VAR="/path/to/dir"
		if strings.Contains(line, "=") && strings.Contains(line, "\"") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[0])
				path := strings.Trim(strings.TrimSpace(parts[1]), "\"'")

				// Skip if it's not a path or if it contains variables
				if strings.HasPrefix(path, "/") && !strings.Contains(path, "$") {
					// Convert name to lowercase for consistency
					name = strings.ToLower(name)
					locations[name] = path
				}
			}
		}
	}

	// Find the Jump Gate section
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
			// Look for elif statements like: elif [[ ${query} == "delta" ]]; then
			if (strings.Contains(line, "if [[") || strings.Contains(line, "elif [[")) &&
			   strings.Contains(line, "${query}") && strings.Contains(line, "==") {
				// Extract the location name
				parts := strings.Split(line, "==")
				if len(parts) >= 2 {
					// Better regex for cleaning name
					namePart := parts[1]
					namePart = strings.ReplaceAll(namePart, "\"", "")
					namePart = strings.ReplaceAll(namePart, "'", "")
					namePart = strings.ReplaceAll(namePart, "]];", "")
					namePart = strings.ReplaceAll(namePart, "then", "")
					name := strings.TrimSpace(namePart)

					// Look for the jumpto call on the next line
					if i+1 < len(lines) {
						nextLine := strings.TrimSpace(lines[i+1])
						if strings.HasPrefix(nextLine, "jumpto") {
							// Extract the path
							pathParts := strings.Split(nextLine, "jumpto")
							if len(pathParts) >= 2 {
								// Clean up the path
								pathStr := pathParts[1]
								pathStr = strings.ReplaceAll(pathStr, "\"", "")
								pathStr = strings.ReplaceAll(pathStr, "'", "")
								pathStr = strings.ReplaceAll(pathStr, "||", "")
								pathStr = strings.ReplaceAll(pathStr, "return 1", "")
								pathStr = strings.ReplaceAll(pathStr, "return", "")
								path := strings.TrimSpace(pathStr)

								// Handle special cases for home directory
								if path == "~" {
									path = home
								} else if strings.HasPrefix(path, "~/") {
									path = filepath.Join(home, path[2:])
								} else if strings.HasPrefix(path, "$HOME") {
									path = filepath.Join(home, path[6:])
								}

								// Special case for deepfry location which uses $DEEPFRY_HOME variable
								if name == "deepfry" {
									// Try to find DEEPFRY_HOME in environment variables or set a reasonable default
									deepfryHome := os.Getenv("DEEPFRY_HOME")
									if deepfryHome != "" {
										locations[name] = deepfryHome
									} else {
										// Set a default location (user's home directory + /deepfry)
										locations[name] = filepath.Join(home, "deepfry")
									}
								} else {
									// Only add if it's a valid path
									if !strings.Contains(path, "$") && (strings.HasPrefix(path, "/") || path == home) {
										locations[name] = path
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

	return locations
}