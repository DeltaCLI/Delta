package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/chzyer/readline"
)

// Simple encryption key based on machine-specific values
func getEncryptionKey() []byte {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "delta-cli"
	}
	username := os.Getenv("USER")
	if username == "" {
		username = "delta-user"
	}
	
	// Create a unique key based on hostname and username
	keyString := hostname + "-" + username + "-delta-history-key"
	
	// Hash the key string to get 32 bytes (AES-256)
	hash := sha256.Sum256([]byte(keyString))
	return hash[:]
}

// Custom history file implementation with encryption
type EncryptedHistory struct {
	filePath string
	key      []byte
}

func NewEncryptedHistory(path string) *EncryptedHistory {
	return &EncryptedHistory{
		filePath: path,
		key:      getEncryptionKey(),
	}
}

func (h *EncryptedHistory) ReadHistory() ([]string, error) {
	// Check if file exists
	if _, err := os.Stat(h.filePath); os.IsNotExist(err) {
		return []string{}, nil
	}
	
	// Read encrypted file
	data, err := ioutil.ReadFile(h.filePath)
	if err != nil {
		return nil, err
	}
	
	// If file is empty, return empty history
	if len(data) == 0 {
		return []string{}, nil
	}
	
	// Decrypt the data
	block, err := aes.NewCipher(h.key)
	if err != nil {
		return nil, err
	}
	
	// First 16 bytes are the IV
	if len(data) < aes.BlockSize {
		return []string{}, nil // Not enough data for IV
	}
	
	iv := data[:aes.BlockSize]
	data = data[aes.BlockSize:]
	
	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(data, data) // decrypt in place
	
	// Convert to string and split by lines
	history := strings.Split(string(data), "\n")
	
	// Remove empty last line if present
	if len(history) > 0 && history[len(history)-1] == "" {
		history = history[:len(history)-1]
	}
	
	return history, nil
}

func (h *EncryptedHistory) WriteHistory(history []string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(h.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	
	// Convert history to string
	data := []byte(strings.Join(history, "\n"))
	if len(data) == 0 {
		// If empty, just create empty file
		return ioutil.WriteFile(h.filePath, []byte{}, 0600)
	}
	
	// Encrypt the data
	block, err := aes.NewCipher(h.key)
	if err != nil {
		return err
	}
	
	// Create IV and prepend to output
	ciphertext := make([]byte, aes.BlockSize+len(data))
	iv := ciphertext[:aes.BlockSize]
	
	// Fill IV with random data
	if _, err := io.ReadFull(strings.NewReader(strings.Repeat("delta", 4)), iv); err != nil {
		return err
	}
	
	// Encrypt the data
	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], data)
	
	// Write encrypted data to file
	return ioutil.WriteFile(h.filePath, ciphertext, 0600)
}

// Custom readline history implementation
type EncryptedHistoryHandler struct {
	eh      *EncryptedHistory
	history []string
	maxSize int
}

func NewEncryptedHistoryHandler(filePath string, maxSize int) (*EncryptedHistoryHandler, error) {
	eh := NewEncryptedHistory(filePath)
	history, err := eh.ReadHistory()
	if err != nil {
		return nil, err
	}
	
	return &EncryptedHistoryHandler{
		eh:      eh,
		history: history,
		maxSize: maxSize,
	}, nil
}

func (h *EncryptedHistoryHandler) Write(line string) error {
	// Don't add empty lines
	if strings.TrimSpace(line) == "" {
		return nil
	}
	
	// Don't add duplicates of the most recent entry
	if len(h.history) > 0 && h.history[len(h.history)-1] == line {
		return nil
	}
	
	h.history = append(h.history, line)
	
	// Trim history if it exceeds max size
	if h.maxSize > 0 && len(h.history) > h.maxSize {
		h.history = h.history[len(h.history)-h.maxSize:]
	}
	
	return h.eh.WriteHistory(h.history)
}

func (h *EncryptedHistoryHandler) GetHistory(limit int) ([]string, error) {
	if limit <= 0 || limit > len(h.history) {
		return h.history, nil
	}
	return h.history[len(h.history)-limit:], nil
}

func main() {
	fmt.Println("Welcome to Delta! 🔼")
	fmt.Println()

	historyFile := os.Getenv("HOME") + "/.delta_history"
	historyLimit := 500
	
	// Initialize our encrypted history handler
	historyHandler, err := NewEncryptedHistoryHandler(historyFile, historyLimit)
	if err != nil {
		fmt.Println("Error initializing history:", err)
	}
	
	// Configure readline with custom history support
	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "∆ ",
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
		HistoryLimit:    historyLimit,
		HistorySearchFold: true, // Enables case-insensitive history search
	})
	if err != nil {
		fmt.Println("Error initializing readline:", err)
		return
	}
	defer rl.Close()
	
	// Load history from our encrypted file
	if historyHandler != nil {
		history, err := historyHandler.GetHistory(historyLimit)
		if err == nil {
			for _, line := range history {
				rl.SaveHistory(line)
			}
		}
	}

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
		rl.SetPrompt(prompt)

		// Read input from the user with history support
		input, err := rl.Readline()
		if err != nil {
			if err == readline.ErrInterrupt {
				// Ctrl+C at prompt just clears the line
				continue
			} else if err == io.EOF {
				// Ctrl+D exits
				fmt.Println("\nGoodbye! 👋")
				break
			}
			fmt.Println("Error reading input:", err)
			continue
		}

		// Trim any whitespace from the input
		command := strings.TrimSpace(input)
		
		// Save command to encrypted history
		if command != "" && historyHandler != nil {
			historyHandler.Write(command)
		}

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

		// Process the command in a subshell and pass our signal channel
		runCommand(command, c)
	}
}

func runCommand(command string, sigChan chan os.Signal) {
	// Parse the command to get the executable
	cmdParts := strings.Fields(command)
	if len(cmdParts) == 0 {
		return
	}

	// Set up command
	cmd := exec.Command("zsh", "-c", "source ~/.zshrc && " + command)

	// Connect standard I/O
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// For potentially interactive programs, we don't use a separate process group
	// This allows signals like Ctrl+C to be passed directly to the child process

	// Start the command
	err := cmd.Start()
	if err != nil {
		fmt.Println("Error starting command:", err)
		return
	}

	// Set up channel for command completion
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	// Set up temporary signal handling for this subprocess
	subprocSigChan := make(chan os.Signal, 1)

	// Temporarily disable our main shell signal handling
	signal.Reset(os.Interrupt, syscall.SIGTERM)

	// Set up subprocess-specific signal handling
	// This allows us to properly distinguish between signals
	signal.Notify(subprocSigChan, os.Interrupt, syscall.SIGTERM)

	// Set up a goroutine to handle signals during subprocess execution
	go func() {
		for range subprocSigChan {
			// Just let the signal pass through to the subprocess
			// Don't forward it - the OS will do that automatically
			// since we're not using Setpgid
		}
	}()

	// Wait for the command to complete
	err = <-done

	// Reset all signal handling
	signal.Reset(os.Interrupt, syscall.SIGTERM)

	// Close our subprocess signal channel by stopping notification
	signal.Stop(subprocSigChan)

	// Re-establish the main shell's signal handling
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Process completed, check for errors
	if err != nil {
		// Only show error message for non-interrupt exits
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Don't print exit code for interrupt signals (Ctrl+C)
			if exitErr.ExitCode() != 130 { // 130 is the exit code for SIGINT
				fmt.Printf("Command exited with code %d\n", exitErr.ExitCode())
			}
		} else if err != syscall.EINTR { // Don't report if interrupted by signal
			fmt.Println("Command failed:", err)
		}
	}
}