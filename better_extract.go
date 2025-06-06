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
	locations := extractLocations(string(data), homeDir)

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

func extractLocations(data string, homeDir string) map[string]string {
	// Map to store locations
	locations := make(map[string]string)

	// Map to store variable values
	variables := make(map[string]string)

	// Initialize with common variables that might be used
	variables["HOME"] = homeDir

	// Split into lines
	lines := strings.Split(data, "\n")

	// First pass: Find all variable definitions
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Look for variable definitions
		if strings.Contains(line, "=") && !strings.HasPrefix(line, "#") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				varName := strings.TrimSpace(parts[0])

				// Skip if it's a conditional assignment
				if strings.Contains(varName, "if") || strings.Contains(varName, "elif") {
					continue
				}

				// Clean up the value
				varValue := strings.Trim(strings.TrimSpace(parts[1]), "\"'")

				// Add to variables map
				variables[varName] = varValue
			}
		}
	}

	// Second pass: Resolve variable references
	changedSomething := true
	iterationCount := 0

	// Keep resolving variables until no more changes are made or max iterations reached
	for changedSomething && iterationCount < 10 {
		changedSomething = false
		iterationCount++

		for name, value := range variables {
			// Check if this value contains a reference
			if strings.Contains(value, "$") {
				resolved := value

				// Replace all variable references
				for refName, refValue := range variables {
					placeholder := "$" + refName
					if strings.Contains(resolved, placeholder) {
						resolved = strings.ReplaceAll(resolved, placeholder, refValue)
						changedSomething = true
					}
				}

				// Update the variable with resolved value
				if resolved != value {
					variables[name] = resolved
				}
			}
		}
	}

	// Now process jump aliases from the Jump Gate section
	inJumpGate := false
	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Find the Jump Gate section
		if strings.Contains(line, "Jump Gate") {
			inJumpGate = true
			continue
		}

		// Process commands in Jump Gate section
		if inJumpGate {
			// Look for commands like: "elif [[ ${query} == "delta" ]]; then"
			if (strings.Contains(line, "if [[") || strings.Contains(line, "elif [[")) &&
				strings.Contains(line, "${query}") && strings.Contains(line, "==") {

				// Extract alias name
				parts := strings.Split(line, "==")
				if len(parts) >= 2 {
					// Clean up the alias name
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
							destParts := strings.SplitN(nextLine, "jumpto", 2)
							if len(destParts) >= 2 {
								dest := strings.TrimSpace(destParts[1])

								// Remove quotes, return statement, etc.
								dest = strings.ReplaceAll(dest, "\"", "")
								dest = strings.ReplaceAll(dest, "'", "")
								dest = strings.ReplaceAll(dest, "||", " ")
								dest = strings.ReplaceAll(dest, "return 1", "")
								dest = strings.ReplaceAll(dest, "return", "")
								dest = strings.TrimSpace(dest)

								// Resolve ~ to home directory
								if dest == "~" {
									dest = homeDir
								} else if strings.HasPrefix(dest, "~/") {
									dest = filepath.Join(homeDir, dest[2:])
								}

								// Check if this is a variable reference
								if strings.HasPrefix(dest, "$") {
									varName := strings.TrimPrefix(dest, "$")
									if val, ok := variables[varName]; ok {
										dest = val
									}
								}

								// Only add if it's a valid path that doesn't contain variables
								if !strings.Contains(dest, "$") && (strings.HasPrefix(dest, "/") || dest == homeDir) {
									locations[name] = dest
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

	// Add some hard-coded common locations if they don't exist yet
	if _, exists := locations["home"]; !exists {
		locations["home"] = homeDir
	}
	if _, exists := locations["delta"]; !exists {
		locations["delta"] = "/home/bleepbloop/deltacli"
	}

	return locations
}
