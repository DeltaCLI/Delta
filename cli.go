package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
)

func main() {
	fmt.Println("Welcome to Delta! 🔼")
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	inSubCommand := false

	// Set up signal handling for Ctrl+C
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	for {
		var prompt string
		if inSubCommand {
			prompt = "⬠ "
		} else {
			// Display the delta symbol as the prompt
			prompt = "∆ "
		}

		// Display the appropriate symbol as the prompt
		fmt.Print(prompt)
		// Read input from the user
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			continue
		}

		// Trim the newline character from the input
		command := strings.TrimSpace(input)

		// Handle the exit command
		if command == "exit" || command == "quit" {
			fmt.Println("Goodbye! 👋")
			break
		}

		// Check for subcommand mode
		if command == "sub" {
			inSubCommand = true
			fmt.Println("Entering subcommand mode.")
			continue
		} else if command == "end" {
			inSubCommand = false
			fmt.Println("Exiting subcommand mode.")
			continue
		}

		// Process the command in a subshell
		runCommand(command)
	}
}

func runCommand(command string) {
	cmd := exec.Command("zsh", "-c", "source ~/.zshrc && " + command)
	
	// Connect standard I/O
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	// Create a process group for proper signal forwarding
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	
	// Start the command without waiting for it
	err := cmd.Start()
	if err != nil {
		fmt.Println("Error starting command:", err)
		return
	}
	
	// Set up a channel to catch signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	// Start a goroutine to handle signals
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	
	// Wait for either command completion or signal
	select {
	case err = <-done:
		// Command completed normally
		signal.Reset(os.Interrupt, syscall.SIGTERM)
		if err != nil {
			// Only show error message, output is already connected to terminal
			if exitErr, ok := err.(*exec.ExitError); ok {
				// Don't exit the main program, just return the exit code
				fmt.Printf("Command exited with code %d\n", exitErr.ExitCode())
			} else {
				fmt.Println("Command failed:", err)
			}
		}
	case sig := <-sigChan:
		// Forward the signal to the process group
		syscall.Kill(-cmd.Process.Pid, sig.(syscall.Signal))
		<-done // Wait for the command to exit
		signal.Reset(os.Interrupt, syscall.SIGTERM)
	}
}
