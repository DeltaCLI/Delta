package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

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

// DeltaCompleter implements the readline.AutoCompleter interface
type DeltaCompleter struct {
	historyHandler *EncryptedHistoryHandler        // For history-based completion
	cmdCache       map[string]bool                 // Cache of executable commands
	cmdCacheMutex  sync.RWMutex                    // Mutex for thread-safe access to cmdCache
	cmdCacheInit   sync.Once                       // Used to initialize the command cache once
	cmdDirs        []string                        // Directories in PATH

	// Special command completions
	internalCmds map[string][]string               // Map of internal commands to their subcommands
}

// NewDeltaCompleter creates a new tab completer with the given history handler
func NewDeltaCompleter(historyHandler *EncryptedHistoryHandler) *DeltaCompleter {
	// Initialize internal commands for completion
	internalCmds := map[string][]string{
		"ai":   {"on", "off", "model", "status"},
		"help": {},
	}

	return &DeltaCompleter{
		historyHandler: historyHandler,
		cmdCache:       make(map[string]bool),
		internalCmds:   internalCmds,
	}
}

// Do implements the readline.AutoCompleter interface
func (c *DeltaCompleter) Do(line []rune, pos int) (newLine [][]rune, length int) {
	// Convert current line to string up to cursor position
	lineStr := string(line[:pos])

	// Get the word at cursor (what we're trying to complete)
	word, _ := c.getCurrentWord(lineStr)

	// Get command context - are we completing a command or an argument?
	isCommand := c.isCompletingCommand(lineStr)

	// Choose completion strategy based on context
	var completions []string

	// Check for internal command completion (starts with colon)
	if strings.HasPrefix(lineStr, ":") {
		completions = c.completeInternalCommand(lineStr)
	// Check if it's a file path (starts with ./ or / or ~/)
	} else if strings.HasPrefix(word, "./") || strings.HasPrefix(word, "/") ||
	   strings.HasPrefix(word, "~/") || strings.HasPrefix(word, "$HOME/") {
		// File path completion
		completions = c.completeFilePath(word)
	} else if isCommand {
		// Command name completion (combine executables and history)
		cmdCompletions := c.completeCommand(word)
		histCompletions := c.completeFromHistory(lineStr)

		// Merge and remove duplicates
		completionMap := make(map[string]bool)
		for _, cmd := range cmdCompletions {
			completionMap[cmd] = true
		}
		for _, hist := range histCompletions {
			completionMap[hist] = true
		}

		for comp := range completionMap {
			completions = append(completions, comp)
		}

		// Sort completions alphabetically
		sort.Strings(completions)
	} else {
		// Argument completion - currently just does file paths
		completions = c.completeFilePath(word)
	}

	// Filter by current input
	var filtered []string
	for _, comp := range completions {
		if strings.HasPrefix(comp, word) {
			// Only include the part after what user already typed
			suggestion := comp[len(word):]
			if suggestion != "" {
				filtered = append(filtered, suggestion)
			}
		}
	}

	// Convert to rune arrays for return
	for _, comp := range filtered {
		newLine = append(newLine, []rune(comp))
	}

	return newLine, len(word)
}

// getCurrentWord extracts the current word being completed from the command line
func (c *DeltaCompleter) getCurrentWord(line string) (word string, prefix string) {
	// Find the start of the current word
	start := 0
	for i := len(line) - 1; i >= 0; i-- {
		if line[i] == ' ' || line[i] == '\t' || line[i] == '=' || line[i] == ':' || line[i] == '/' {
			// If it's a path separator, include it in the word
			if line[i] == '/' {
				start = i
			} else {
				start = i + 1
			}
			break
		}
	}

	if start < len(line) {
		// Return the word and everything before it
		return line[start:], line[:start]
	}
	return line, ""
}

// isCompletingCommand determines if we're completing a command or an argument
func (c *DeltaCompleter) isCompletingCommand(line string) bool {
	trimmed := strings.TrimSpace(line)
	return !strings.Contains(trimmed, " ")
}

// completeFromHistory generates completions based on command history
func (c *DeltaCompleter) completeFromHistory(prefix string) []string {
	if c.historyHandler == nil {
		return []string{}
	}

	hist, err := c.historyHandler.GetHistory(100) // Get recent history
	if err != nil {
		return []string{}
	}

	// Get unique command prefixes from history
	uniqueCmds := make(map[string]bool)
	for _, cmd := range hist {
		// If it's a command that starts with our prefix
		if strings.HasPrefix(cmd, prefix) {
			// Split by space to get just the command
			cmdParts := strings.Fields(cmd)
			if len(cmdParts) > 0 {
				uniqueCmds[cmdParts[0]] = true
			}
		}
	}

	// Convert to slice and sort by alphabetical order
	var completions []string
	for cmd := range uniqueCmds {
		completions = append(completions, cmd)
	}
	sort.Strings(completions)

	return completions
}

// completeCommand generates completions for executable commands
func (c *DeltaCompleter) completeCommand(prefix string) []string {
	// Initialize command cache if not done yet
	c.cmdCacheInit.Do(func() {
		c.refreshCommandCache()
	})

	// Read command cache with lock
	c.cmdCacheMutex.RLock()
	defer c.cmdCacheMutex.RUnlock()

	var completions []string
	for cmd := range c.cmdCache {
		if strings.HasPrefix(cmd, prefix) {
			completions = append(completions, cmd)
		}
	}

	// Sort completions
	sort.Strings(completions)
	return completions
}

// refreshCommandCache updates the cache of executable commands by scanning PATH
func (c *DeltaCompleter) refreshCommandCache() {
	c.cmdCacheMutex.Lock()
	defer c.cmdCacheMutex.Unlock()

	// Clear existing cache
	c.cmdCache = make(map[string]bool)

	// Get PATH directories
	pathEnv := os.Getenv("PATH")
	c.cmdDirs = filepath.SplitList(pathEnv)

	// Scan each directory in PATH for executables
	for _, dir := range c.cmdDirs {
		files, err := ioutil.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, file := range files {
			// Skip directories and files without execute permission
			if file.IsDir() {
				continue
			}

			// Check if file is executable
			if file.Mode()&0111 != 0 {
				c.cmdCache[file.Name()] = true
			}
		}
	}
}

// completeInternalCommand generates completions for internal commands
func (c *DeltaCompleter) completeInternalCommand(input string) []string {
	// Remove the colon prefix
	cmdStr := strings.TrimPrefix(input, ":")

	// If empty, return all internal commands
	if cmdStr == "" {
		var cmds []string
		for cmd := range c.internalCmds {
			cmds = append(cmds, ":"+cmd)
		}
		sort.Strings(cmds)
		return cmds
	}

	// Check if we're completing a subcommand
	parts := strings.Fields(cmdStr)

	if len(parts) == 1 {
		// We're completing the main command
		cmd := parts[0]

		// Find matching commands
		var matches []string
		for internalCmd := range c.internalCmds {
			if strings.HasPrefix(internalCmd, cmd) {
				// Return with colon prefix
				matches = append(matches, ":"+internalCmd)
			}
		}
		sort.Strings(matches)
		return matches
	} else if len(parts) >= 2 {
		// We're completing a subcommand
		cmd := parts[0]
		subCmd := parts[1]

		// Check if the command exists
		if subCmds, ok := c.internalCmds[cmd]; ok {
			// Find matching subcommands
			var matches []string
			matchPrefix := ":" + cmd + " "

			for _, sc := range subCmds {
				if strings.HasPrefix(sc, subCmd) {
					matches = append(matches, matchPrefix+sc)
				}
			}
			sort.Strings(matches)
			return matches
		}
	}

	return []string{}
}

// completeFilePath generates completions for file paths
func (c *DeltaCompleter) completeFilePath(prefix string) []string {
	// Expand ~ and $HOME to home directory
	expandedPrefix := prefix
	if strings.HasPrefix(prefix, "~/") || strings.HasPrefix(prefix, "$HOME/") {
		home, err := os.UserHomeDir()
		if err == nil {
			if strings.HasPrefix(prefix, "~/") {
				expandedPrefix = home + prefix[1:]
			} else {
				expandedPrefix = home + prefix[5:]
			}
		}
	}

	// Get the directory to scan
	dir := filepath.Dir(expandedPrefix)
	if dir == "." {
		dir = "./"
	}

	// Get the filename to match
	base := filepath.Base(expandedPrefix)

	// Check if the directory exists
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return []string{}
	}

	// Read the directory
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return []string{}
	}

	// Filter and format the results
	var completions []string
	for _, file := range files {
		name := file.Name()

		// Skip files that don't match prefix
		if !strings.HasPrefix(name, base) {
			continue
		}

		// Format the completion result
		var completion string
		if prefix == "" || strings.HasSuffix(prefix, "/") {
			completion = name
		} else {
			// Only include the part after the last slash
			completion = filepath.Join(filepath.Dir(prefix), name)
		}

		// Add trailing slash for directories
		if file.IsDir() {
			completion += "/"
		}

		completions = append(completions, completion)
	}

	// Sort completions
	sort.Strings(completions)
	return completions
}

// AI system configuration
var (
	aiEnabled    = false
	aiModel      = "llama3.3:8b"
	aiClient     *OllamaClient
	aiPrediction = ""
	aiAvailable  = false
)

// OllamaClient manages communication with the Ollama server
type OllamaClient struct {
	httpClient *http.Client
	serverURL  string
}

// NewOllamaClient creates a new client for communicating with Ollama
func NewOllamaClient() *OllamaClient {
	return &OllamaClient{
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		serverURL: "http://localhost:11434",
	}
}

// CheckModelAvailability checks if the specified model is available
func (c *OllamaClient) CheckModelAvailability(model string) bool {
	// Make a simple GET request to /api/tags
	resp, err := c.httpClient.Get(c.serverURL + "/api/tags")
	if err != nil {
		fmt.Printf("Error connecting to Ollama: %v\n", err)
		return false
	}
	defer resp.Body.Close()

	// Read and parse the body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Error reading Ollama response: %v\n", err)
		return false
	}

	// Simple check - just see if the model name appears in the response
	return strings.Contains(string(body), model)
}

// Predict generates a prediction based on the given context
func (c *OllamaClient) Predict(context string) (string, error) {
	// For now, return a placeholder - in a real implementation, this would call the Ollama API
	return "Command suggestion based on context...", nil
}

// Handle internal commands that start with a colon
func handleInternalCommand(command string) bool {
	// Strip the colon
	cmdWithoutColon := strings.TrimPrefix(command, ":")
	// Split into command and arguments
	parts := strings.Fields(cmdWithoutColon)

	if len(parts) == 0 {
		return false
	}

	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "ai":
		return handleAICommand(args)
	case "help":
		showHelp()
		return true
	default:
		fmt.Printf("Unknown command: %s\n", command)
		fmt.Println("Type :help for a list of available commands.")
		return true
	}
}

// Handle AI-specific commands
func handleAICommand(args []string) bool {
	if len(args) == 0 {
		fmt.Println("AI assistant is currently", getAIStatusText())
		return true
	}

	switch args[0] {
	case "on":
		if !aiEnabled {
			// Check if we can connect to Ollama
			if aiClient == nil {
				aiClient = NewOllamaClient()
			}

			// Check if the model is available
			available := aiClient.CheckModelAvailability(aiModel)
			if !available {
				fmt.Printf("AI model %s is not available. Make sure Ollama is running and the model is pulled.\n", aiModel)
				fmt.Println("You can pull the model with: ollama pull " + aiModel)
				return true
			}

			aiEnabled = true
			aiAvailable = true
			fmt.Println("AI assistant enabled using model", aiModel)
		} else {
			fmt.Println("AI assistant is already enabled.")
		}
		return true

	case "off":
		if aiEnabled {
			aiEnabled = false
			fmt.Println("AI assistant disabled.")
		} else {
			fmt.Println("AI assistant is already disabled.")
		}
		return true

	case "model":
		if len(args) < 2 {
			fmt.Println("Current AI model:", aiModel)
			return true
		}

		// Set a new model
		newModel := args[1]

		// Check if the model is available
		if aiClient == nil {
			aiClient = NewOllamaClient()
		}

		available := aiClient.CheckModelAvailability(newModel)
		if !available {
			fmt.Printf("AI model %s is not available. Make sure Ollama is running and the model is pulled.\n", newModel)
			fmt.Println("You can pull the model with: ollama pull " + newModel)
			return true
		}

		aiModel = newModel
		fmt.Println("AI model set to:", aiModel)
		return true

	case "status":
		fmt.Println("AI Status:")
		fmt.Println("- Enabled:", aiEnabled)
		fmt.Println("- Model:", aiModel)
		fmt.Println("- Available:", aiAvailable)
		return true

	default:
		fmt.Printf("Unknown AI command: %s\n", args[0])
		fmt.Println("Available AI commands: on, off, model, status")
		return true
	}
}

// Get the AI status as a formatted text
func getAIStatusText() string {
	if aiEnabled {
		return fmt.Sprintf("enabled (using %s)", aiModel)
	}
	return "disabled"
}

// Show help for internal commands
func showHelp() {
	fmt.Println("Delta CLI Internal Commands:")
	fmt.Println("  :ai [on|off]      - Enable or disable AI assistant")
	fmt.Println("  :ai model <name>  - Change AI model (e.g., llama3.3:8b)")
	fmt.Println("  :ai status        - Show AI assistant status")
	fmt.Println("  :help             - Show this help message")
	fmt.Println("")
	fmt.Println("Shell Navigation:")
	fmt.Println("  exit, quit        - Exit Delta shell")
	fmt.Println("  sub               - Enter subcommand mode")
	fmt.Println("  end               - Exit subcommand mode")
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
	
	// Create our completer
	completer := NewDeltaCompleter(historyHandler)

	// Configure readline with custom history support and tab completion
	rl, err := readline.NewEx(&readline.Config{
		Prompt:            "∆ ",
		InterruptPrompt:   "^C",
		EOFPrompt:         "exit",
		HistoryLimit:      historyLimit,
		HistorySearchFold: true, // Enables case-insensitive history search
		AutoComplete:      completer,
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

		// Handle special commands that start with a colon
		if strings.HasPrefix(command, ":") {
			// Handle special internal commands
			if handleInternalCommand(command) {
				continue
			}
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

	// Get the user's shell from environment
	shell := os.Getenv("SHELL")
	if shell == "" {
		// Default to bash if SHELL is not set
		shell = "/bin/bash"
	}

	// Get shell name for specialized handling
	shellName := filepath.Base(shell)

	// Get home directory safely
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fall back to $HOME if os.UserHomeDir fails
		homeDir = os.Getenv("HOME")
		if homeDir == "" {
			// If we can't determine home directory, just run command directly
			err = runDirectCommand(shell, command, sigChan)
			return
		}
	}

	// Build the appropriate shell command based on shell type
	var shellCmd string

	switch shellName {
	case "zsh":
		// Special handling for ZSH to properly load aliases and functions
		shellCmd = buildZshCommand(homeDir, command)
	case "bash":
		// Bash profile loading
		shellCmd = buildBashCommand(homeDir, command)
	case "fish":
		// Fish shell handling
		shellCmd = buildFishCommand(homeDir, command)
	default:
		// Default shell handling - just run the command
		shellCmd = command
	}

	// Now run the command
	runShellCommand(shell, shellCmd, sigChan)
}

// ZSH special handling to properly load functions and aliases
func buildZshCommand(homeDir string, command string) string {
	zshrcFile := filepath.Join(homeDir, ".zshrc")
	zshenvFile := filepath.Join(homeDir, ".zshenv")
	
	// Start with an empty command
	shellCmd := ""
	
	// Source zshenv if it exists
	if _, err := os.Stat(zshenvFile); err == nil {
		shellCmd += "source " + zshenvFile + " 2>/dev/null || true; "
	}
	
	// Source zshrc if it exists - this contains most user functions and aliases
	if _, err := os.Stat(zshrcFile); err == nil {
		shellCmd += "source " + zshrcFile + " 2>/dev/null || true; "
	}
	
	// Parse the command to get just the command name (no arguments)
	cmdName := command
	cmdArgs := ""
	
	if len(strings.Fields(command)) > 0 {
		cmdName = strings.Fields(command)[0]
		
		if len(strings.Fields(command)) > 1 {
			cmdArgs = strings.Join(strings.Fields(command)[1:], " ")
		}
		
		// Special handling for common commands that might be functions or aliases
		shellCmd += "if typeset -f " + cmdName + " > /dev/null 2>&1; then\n" +
			"  # It's a shell function, run it\n" +
			"  " + cmdName + " " + cmdArgs + "\n" +
			"elif alias " + cmdName + " > /dev/null 2>&1; then\n" +
			"  # It's an alias, expand and run it\n" +
			"  EXPANDED=$(alias " + cmdName + " | sed 's/^[^=]*=//g' | sed \"s/'//g\")\n" +
			"  eval \"$EXPANDED " + cmdArgs + "\"\n" +
			"else\n" +
			"  # It's an external command, run it directly\n" +
			"  " + command + "\n" +
			"fi"
	} else {
		// Empty command, just return it
		shellCmd += command
	}
	
	return shellCmd
}

// Bash profile loading
func buildBashCommand(homeDir string, command string) string {
	profileFile := filepath.Join(homeDir, ".bash_profile")
	rcFile := filepath.Join(homeDir, ".bashrc")
	
	// Start with an empty command
	shellCmd := ""
	
	// Source bash_profile or bashrc if they exist
	if _, err := os.Stat(profileFile); err == nil {
		shellCmd += "source " + profileFile + " 2>/dev/null || true; "
	} else if _, err := os.Stat(rcFile); err == nil {
		shellCmd += "source " + rcFile + " 2>/dev/null || true; "
	}
	
	// Append the user's command
	shellCmd += command
	
	return shellCmd
}

// Fish shell profile loading
func buildFishCommand(homeDir string, command string) string {
	configFile := filepath.Join(homeDir, ".config/fish/config.fish")
	
	// If config.fish exists, source it
	if _, err := os.Stat(configFile); err == nil {
		return "source " + configFile + " 2>/dev/null || true; " + command
	}
	
	// Otherwise just return the original command
	return command
}

// Execute a command directly without any profile sourcing
func runDirectCommand(shell string, command string, sigChan chan os.Signal) error {
	// Create the command with default arguments
	shellArgs := []string{"-c", command}
	cmd := exec.Command(shell, shellArgs...)
	
	// Connect standard I/O
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	return executeCommand(cmd, sigChan)
}

// Execute a shell command with profile sourcing
func runShellCommand(shell string, shellCmd string, sigChan chan os.Signal) error {
	// Create command with the -c flag for all shells
	shellArgs := []string{"-c", shellCmd}
	cmd := exec.Command(shell, shellArgs...)
	
	// Connect standard I/O
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	return executeCommand(cmd, sigChan)
}

// Actually execute the command with proper signal handling
func executeCommand(cmd *exec.Cmd, sigChan chan os.Signal) error {
	// Start the command
	err := cmd.Start()
	if err != nil {
		fmt.Println("Error starting command:", err)
		return err
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
	signal.Notify(subprocSigChan, os.Interrupt, syscall.SIGTERM)
	
	// Create a context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Set up a goroutine to handle signals during subprocess execution
	go func() {
		select {
		case sig := <-subprocSigChan:
			// Pass the signal to the child process
			if cmd.Process != nil {
				cmd.Process.Signal(sig)
			}
		case <-ctx.Done():
			// Exit if context is cancelled
			return
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
	
	return err
}