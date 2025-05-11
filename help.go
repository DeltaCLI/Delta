package main

import "fmt"

// showEnhancedHelp displays an enhanced help message with all available Delta commands
func showEnhancedHelp() {
	fmt.Println("Delta CLI Internal Commands:")
	
	// AI Commands
	fmt.Println("  AI Assistant:")
	fmt.Println("  :ai [on|off]      - Enable or disable AI assistant")
	fmt.Println("  :ai model <n>     - Change AI model (e.g., phi4:latest)")
	fmt.Println("  :ai status        - Show AI assistant status")
	
	// Jump Commands
	fmt.Println("")
	fmt.Println("  Navigation:")
	fmt.Println("  :jump <location>  - Jump to predefined location")
	fmt.Println("  :jump add <n> [path] - Add a new jump location")
	fmt.Println("  :jump remove <n>     - Remove a jump location")
	fmt.Println("  :jump import jumpsh  - Import locations from jump.sh")
	fmt.Println("  :j <location>     - Shorthand for jump")
	
	// Memory Commands
	fmt.Println("")
	fmt.Println("  Memory System:")
	fmt.Println("  :memory status    - Show memory system status")
	fmt.Println("  :memory enable    - Enable memory collection")
	fmt.Println("  :memory disable   - Disable memory collection")
	fmt.Println("  :memory stats     - Show detailed memory statistics")
	fmt.Println("  :memory config    - View or modify memory configuration")
	fmt.Println("  :memory list      - List available data shards")
	fmt.Println("  :memory export    - Export data for a specific date")
	fmt.Println("  :memory clear     - Clear all collected data (requires confirmation)")
	fmt.Println("  :memory train     - Memory training commands")
	fmt.Println("  :mem              - Shorthand for memory commands")
	
	// Other Commands
	fmt.Println("")
	fmt.Println("  System:")
	fmt.Println("  :init             - Initialize configuration files")
	fmt.Println("  :help             - Show this help message")
	
	// Shell Navigation
	fmt.Println("")
	fmt.Println("Shell Navigation:")
	fmt.Println("  cd [directory]    - Change current directory")
	fmt.Println("  pwd               - Print current working directory")
	fmt.Println("  exit, quit        - Exit Delta shell")
	fmt.Println("  sub               - Enter subcommand mode")
	fmt.Println("  end               - Exit subcommand mode")
}