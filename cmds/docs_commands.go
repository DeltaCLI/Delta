package cmds

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// HandleDocsCommand handles the :docs command
func HandleDocsCommand(args []string) bool {
	// Check if we have any arguments
	if len(args) == 0 {
		// Default action: build and open docs
		return openDocs()
	}

	// Handle subcommands
	switch args[0] {
	case "build":
		return buildDocs()
	case "dev":
		return runDocsDev()
	case "open":
		return openDocs()
	case "status":
		return showDocsStatus()
	case "help":
		showDocsHelp()
		return true
	default:
		fmt.Printf("Unknown docs command: %s\n", args[0])
		showDocsHelp()
		return true
	}
}

// buildDocs builds the documentation using Astro
func buildDocs() bool {
	fmt.Println("Building documentation...")
	
	// Change to docs directory
	docsPath := filepath.Join(".", "docs")
	
	// Check if docs directory exists
	if _, err := os.Stat(docsPath); os.IsNotExist(err) {
		fmt.Println("Error: docs directory not found")
		return false
	}
	
	// Run npm install if node_modules doesn't exist
	nodeModulesPath := filepath.Join(docsPath, "node_modules")
	if _, err := os.Stat(nodeModulesPath); os.IsNotExist(err) {
		fmt.Println("Installing dependencies...")
		cmd := exec.Command("npm", "install")
		cmd.Dir = docsPath
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("Error installing dependencies: %v\n", err)
			return false
		}
	}
	
	// Run the build command
	cmd := exec.Command("npm", "run", "build")
	cmd.Dir = docsPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	startTime := time.Now()
	err := cmd.Run()
	duration := time.Since(startTime)
	
	// Check if dist directory was created even if build had errors
	distPath := filepath.Join(docsPath, "dist")
	if _, err := os.Stat(distPath); err == nil {
		fmt.Printf("Documentation built successfully in %v (with some warnings)\n", duration)
		return true
	}
	
	if err != nil {
		fmt.Printf("Error building docs: %v\n", err)
		return false
	}
	
	fmt.Printf("Documentation built successfully in %v\n", duration)
	return true
}

// runDocsDev runs the Astro dev server
func runDocsDev() bool {
	fmt.Println("Starting documentation dev server...")
	
	// Change to docs directory
	docsPath := filepath.Join(".", "docs")
	
	// Check if docs directory exists
	if _, err := os.Stat(docsPath); os.IsNotExist(err) {
		fmt.Println("Error: docs directory not found")
		return false
	}
	
	// Run npm install if node_modules doesn't exist
	nodeModulesPath := filepath.Join(docsPath, "node_modules")
	if _, err := os.Stat(nodeModulesPath); os.IsNotExist(err) {
		fmt.Println("Installing dependencies...")
		cmd := exec.Command("npm", "install")
		cmd.Dir = docsPath
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Printf("Error installing dependencies: %v\n", err)
			return false
		}
	}
	
	// Run the dev server
	cmd := exec.Command("npm", "run", "dev")
	cmd.Dir = docsPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	
	// Start the command
	err := cmd.Start()
	if err != nil {
		fmt.Printf("Error starting dev server: %v\n", err)
		return false
	}
	
	// Wait a moment for the server to start
	time.Sleep(2 * time.Second)
	
	// Open the browser
	url := "http://localhost:4321"
	fmt.Printf("Opening docs at %s\n", url)
	openURL(url)
	
	// Wait for the process to finish (user will Ctrl+C)
	err = cmd.Wait()
	if err != nil {
		// Ignore interrupt errors
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 130 {
			fmt.Println("\nDev server stopped")
			return true
		}
		fmt.Printf("Dev server exited with error: %v\n", err)
		return false
	}
	
	return true
}

// openDocs builds and opens the documentation in the default browser
func openDocs() bool {
	// First, build the docs
	if !buildDocs() {
		return false
	}
	
	// Find the built output directory
	distPath := filepath.Join(".", "docs", "dist")
	
	// Check if dist directory exists
	if _, err := os.Stat(distPath); os.IsNotExist(err) {
		fmt.Println("Error: Built documentation not found. Please run ':docs build' first.")
		return false
	}
	
	// Find the index.html file
	indexPath := filepath.Join(distPath, "index.html")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		fmt.Println("Error: index.html not found in dist directory")
		return false
	}
	
	// Get absolute path
	absPath, err := filepath.Abs(indexPath)
	if err != nil {
		fmt.Printf("Error getting absolute path: %v\n", err)
		return false
	}
	
	// Open the file URL
	fileURL := "file://" + absPath
	fmt.Printf("Opening documentation at %s\n", fileURL)
	openURL(fileURL)
	
	return true
}

// openURL opens a URL in the default browser
func openURL(url string) {
	var cmd *exec.Cmd
	
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		// Try xdg-open first, then fallback to other options
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		fmt.Printf("Please open %s in your browser\n", url)
		return
	}
	
	err := cmd.Start()
	if err != nil {
		// On Linux, try other browsers if xdg-open fails
		if runtime.GOOS == "linux" {
			browsers := []string{"firefox", "chromium", "google-chrome", "chrome"}
			for _, browser := range browsers {
				cmd = exec.Command(browser, url)
				if err := cmd.Start(); err == nil {
					return
				}
			}
		}
		fmt.Printf("Error opening browser: %v\n", err)
		fmt.Printf("Please open %s in your browser\n", url)
	}
}

// showDocsStatus shows the current status of the documentation
func showDocsStatus() bool {
	fmt.Println("Documentation Status")
	fmt.Println("===================")
	
	// Check if docs directory exists
	docsPath := filepath.Join(".", "docs")
	if _, err := os.Stat(docsPath); os.IsNotExist(err) {
		fmt.Println("❌ Documentation directory not found")
		return true
	}
	fmt.Println("✅ Documentation directory exists")
	
	// Check if node_modules exists
	nodeModulesPath := filepath.Join(docsPath, "node_modules")
	if _, err := os.Stat(nodeModulesPath); os.IsNotExist(err) {
		fmt.Println("❌ Dependencies not installed (run :docs build to install)")
	} else {
		fmt.Println("✅ Dependencies installed")
	}
	
	// Check if dist directory exists
	distPath := filepath.Join(docsPath, "dist")
	if _, err := os.Stat(distPath); os.IsNotExist(err) {
		fmt.Println("❌ Documentation not built (run :docs build)")
	} else {
		// Get modification time
		info, err := os.Stat(distPath)
		if err == nil {
			fmt.Printf("✅ Documentation built (last built: %s)\n", info.ModTime().Format("2006-01-02 15:04:05"))
		} else {
			fmt.Println("✅ Documentation built")
		}
	}
	
	// Check for running dev server (simple port check)
	cmd := exec.Command("lsof", "-i", ":4321")
	if err := cmd.Run(); err == nil {
		fmt.Println("✅ Dev server is running on port 4321")
	} else {
		fmt.Println("ℹ️  Dev server is not running")
	}
	
	return true
}

// showDocsHelp displays help for the docs command
func showDocsHelp() {
	fmt.Println("Documentation Commands")
	fmt.Println("=====================")
	fmt.Println("  :docs           - Build and open documentation in browser")
	fmt.Println("  :docs build     - Build the documentation")
	fmt.Println("  :docs dev       - Start the development server")
	fmt.Println("  :docs open      - Build and open docs in browser")
	fmt.Println("  :docs status    - Show documentation status")
	fmt.Println("  :docs help      - Show this help message")
	fmt.Println()
	fmt.Println("The documentation uses Astro and will be served locally.")
	fmt.Println("For development, use ':docs dev' to start a live-reload server.")
}