package main

// This file contains helper functions to integrate the jump_manager.go functionality
// with the main CLI code. It helps keep the concerns separated.

import (
	"fmt"
)

// handleJumpCommand processes the jump command with arguments
// This function is called from cli.go and delegates to the JumpManager
func handleJumpCommand(args []string) bool {
	// Call the implementation in the jump_manager.go file
	return HandleJumpCommand(args)
}

// Override the external jump.sh command by checking the command in runCommand
func checkForJumpCommand(command string, args []string) bool {
	// Check if it's the jump command
	if command == "jump" && len(args) > 0 {
		// Use our internal jump command
		handleJumpCommand(args)
		return true
	}

	return false
}

// Helper function to show help for jump commands
func showJumpHelp() {
	fmt.Println("Jump Command Usage:")
	fmt.Println("  jump <location>           - Jump to a saved location")
	fmt.Println("  jump add <name> [path]    - Add a new location (uses current dir if no path)")
	fmt.Println("  jump remove <name>        - Remove a saved location")
	fmt.Println("  jump import jumpsh        - Import locations from jump.sh script")
	fmt.Println("  jump list                 - List all available locations")
	fmt.Println("")
	fmt.Println("Shortcuts:")
	fmt.Println("  :j <location>             - Shorthand for jump")
}
