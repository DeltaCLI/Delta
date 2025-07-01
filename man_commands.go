package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// HandleManCommand handles the :man command for man page operations
func HandleManCommand(args []string) bool {
	// Initialize command docs if needed
	if globalCommandRegistry == nil {
		InitializeCommandDocs()
	}

	generator := NewManPageGenerator()

	if len(args) == 0 {
		showManCommandHelp()
		return true
	}

	switch args[0] {
	case "generate":
		outputDir := "./man"
		if len(args) > 1 {
			outputDir = args[1]
		}
		return generateManPages(generator, outputDir)

	case "preview":
		command := ""
		if len(args) > 1 {
			command = args[1]
		}
		return previewManPage(generator, command)

	case "install":
		manDir := ""
		if len(args) > 1 {
			manDir = args[1]
		}
		return installManPages(generator, manDir)

	case "view":
		command := "delta"
		if len(args) > 1 {
			command = "delta-" + args[1]
		}
		return viewManPage(command)

	case "completions":
		shell := "bash"
		if len(args) > 1 {
			shell = args[1]
		}
		return generateCompletions(generator, shell)

	case "help":
		showManCommandHelp()
		return true

	default:
		fmt.Printf("Unknown man command: %s\n", args[0])
		showManCommandHelp()
		return true
	}
}

func generateManPages(generator *ManPageGenerator, outputDir string) bool {
	fmt.Printf("Generating man pages to %s...\n", outputDir)
	
	if err := generator.GenerateAllManPages(outputDir); err != nil {
		fmt.Printf("Error generating man pages: %v\n", err)
		return false
	}
	
	// List generated files
	files, err := filepath.Glob(filepath.Join(outputDir, "*.1"))
	if err == nil && len(files) > 0 {
		fmt.Println("\nGenerated man pages:")
		for _, file := range files {
			fmt.Printf("  - %s\n", filepath.Base(file))
		}
	}
	
	fmt.Println("\nMan pages generated successfully!")
	fmt.Printf("To install: delta :man install %s\n", outputDir)
	return true
}

func previewManPage(generator *ManPageGenerator, command string) bool {
	// Generate to temp file
	tempDir := filepath.Join(os.TempDir(), "delta-man-preview")
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	var manFile string
	if command == "" || command == "delta" {
		content, err := generator.GenerateMainManPage()
		if err != nil {
			fmt.Printf("Error generating man page: %v\n", err)
			return false
		}
		manFile = filepath.Join(tempDir, "delta.1")
		os.WriteFile(manFile, []byte(content), 0644)
	} else {
		content, err := generator.GenerateCommandManPage(command)
		if err != nil {
			fmt.Printf("Error generating man page: %v\n", err)
			return false
		}
		manFile = filepath.Join(tempDir, fmt.Sprintf("delta-%s.1", command))
		os.WriteFile(manFile, []byte(content), 0644)
	}

	// Try to use man command to view
	if _, err := exec.LookPath("man"); err == nil {
		fmt.Printf("Viewing man page for %s...\n", command)
		cmd := exec.Command("man", manFile)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		return cmd.Run() == nil
	}

	// Fallback: show raw content
	content, _ := os.ReadFile(manFile)
	fmt.Println(string(content))
	return true
}

func installManPages(generator *ManPageGenerator, manDir string) bool {
	fmt.Println("Installing man pages...")
	
	if err := generator.InstallManPages(manDir); err != nil {
		fmt.Printf("Error installing man pages: %v\n", err)
		fmt.Println("\nTroubleshooting:")
		fmt.Println("1. Try with sudo: sudo delta :man install")
		fmt.Println("2. Specify a writable directory: delta :man install ~/man")
		fmt.Println("3. Generate and install manually:")
		fmt.Println("   delta :man generate ./man")
		fmt.Println("   sudo cp ./man/*.1 /usr/local/share/man/man1/")
		fmt.Println("   sudo mandb")
		return false
	}
	
	fmt.Println("Man pages installed successfully!")
	fmt.Println("\nYou can now use:")
	fmt.Println("  man delta              - View main Delta documentation")
	fmt.Println("  man delta-ai           - View AI command documentation")
	fmt.Println("  man delta-update       - View update command documentation")
	fmt.Println("\nRun 'sudo mandb' to update the man page database.")
	return true
}

func viewManPage(command string) bool {
	// Check if man command exists
	if _, err := exec.LookPath("man"); err != nil {
		fmt.Println("Error: 'man' command not found")
		fmt.Println("Please install man-db or similar package for your system")
		return false
	}

	fmt.Printf("Opening man page for %s...\n", command)
	cmd := exec.Command("man", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	
	if err := cmd.Run(); err != nil {
		fmt.Printf("Error viewing man page: %v\n", err)
		fmt.Printf("\nMan page for '%s' may not be installed.\n", command)
		fmt.Println("Run 'delta :man install' to install man pages.")
		return false
	}
	
	return true
}

func generateCompletions(generator *ManPageGenerator, shell string) bool {
	content, err := generator.GenerateCompletions(shell)
	if err != nil {
		fmt.Printf("Error generating completions: %v\n", err)
		return false
	}
	
	fmt.Printf("# %s completion script for Delta CLI\n", shell)
	fmt.Println(content)
	fmt.Println()
	
	switch shell {
	case "bash":
		fmt.Println("# To install:")
		fmt.Println("# Save to: ~/.delta-completion.bash")
		fmt.Println("# Add to ~/.bashrc: source ~/.delta-completion.bash")
	case "zsh":
		fmt.Println("# To install:")
		fmt.Println("# Save to: ~/.delta-completion.zsh")
		fmt.Println("# Add to ~/.zshrc: source ~/.delta-completion.zsh")
	case "fish":
		fmt.Println("# To install:")
		fmt.Println("# Save to: ~/.config/fish/completions/delta.fish")
	}
	
	return true
}

func showManCommandHelp() {
	fmt.Println("Man Page Commands")
	fmt.Println("=================")
	fmt.Println("  :man generate [dir]     - Generate man pages to directory (default: ./man)")
	fmt.Println("  :man preview [command]  - Preview a man page before installing")
	fmt.Println("  :man install [dir]      - Install man pages to system (may need sudo)")
	fmt.Println("  :man view [command]     - View installed man page")
	fmt.Println("  :man completions [shell] - Generate shell completions (bash/zsh/fish)")
	fmt.Println("  :man help               - Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  :man generate           - Generate all man pages to ./man/")
	fmt.Println("  :man preview ai         - Preview the man page for :ai command")
	fmt.Println("  :man install            - Install to /usr/local/share/man/man1/")
	fmt.Println("  :man view               - View the main delta man page")
	fmt.Println("  :man completions bash   - Generate bash completion script")
	fmt.Println()
	fmt.Println("Note: Installation may require sudo privileges.")
}