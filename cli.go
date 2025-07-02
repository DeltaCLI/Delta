package main

import (
	"context"
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
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"delta/cmds"

	"github.com/chzyer/readline"
	"github.com/spf13/cobra"
)

// Global variable to store the original terminal title
var originalTerminalTitle string

// getOriginalTerminalTitle attempts to determine what the terminal title should be restored to
func getOriginalTerminalTitle() string {
	// Only attempt if we're connected to a terminal
	if !isTerminal() {
		return ""
	}

	// Method 1: Check if any environment variables give us hints about the original title
	envVars := []string{
		"TERM_PROGRAM_TITLE", // Some terminal programs set this
		"TERMINAL_TITLE",     // Custom env var if set by user
	}

	for _, envVar := range envVars {
		if value := os.Getenv(envVar); value != "" {
			return value
		}
	}

	// Method 2: Get the shell name - this is most likely what the title was
	shell := os.Getenv("SHELL")
	if shell != "" {
		return filepath.Base(shell)
	}

	// Method 3: Try to determine from parent process
	if ppid := os.Getppid(); ppid > 0 {
		// Read parent process info from /proc on Linux
		cmdlineFile := fmt.Sprintf("/proc/%d/cmdline", ppid)
		if data, err := os.ReadFile(cmdlineFile); err == nil {
			cmdline := string(data)
			// Replace null bytes with spaces and clean up
			cmdline = strings.ReplaceAll(cmdline, "\x00", " ")
			cmdline = strings.TrimSpace(cmdline)

			if cmdline != "" {
				parts := strings.Fields(cmdline)
				if len(parts) > 0 {
					baseName := filepath.Base(parts[0])
					// Return the parent process name if it looks like a shell/terminal
					terminalNames := []string{"bash", "zsh", "fish", "sh", "gnome-terminal", "konsole", "xterm", "alacritty", "kitty", "terminal"}
					for _, term := range terminalNames {
						if strings.Contains(baseName, term) {
							return baseName
						}
					}
				}
			}
		}
	}

	// Fallback: return empty string - we'll just reset to empty title
	return ""
}

// isTerminal checks if we're running in a terminal
func isTerminal() bool {
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// setTerminalTitle sets the terminal title using ANSI escape sequence
func setTerminalTitle(title string) {
	fmt.Printf("\033]0;%s\007", title)
}

// setDeltaTitle sets the terminal title to show delta with the triangle symbol
func setDeltaTitle() {
	setTerminalTitle("∆ delta")
}

// setProgramTitle sets the terminal title to show delta symbol followed by program name
func setProgramTitle(programName string) {
	setTerminalTitle(fmt.Sprintf("∆ %s", programName))
}

// resetTerminalTitle resets the terminal title to the original title
func resetTerminalTitle() {
	if originalTerminalTitle != "" {
		// Restore the original title
		setTerminalTitle(originalTerminalTitle)
	} else {
		// Fallback: Get the shell name to use as the default title
		shell := os.Getenv("SHELL")
		if shell != "" {
			shellName := filepath.Base(shell)
			setTerminalTitle(shellName)
		} else {
			// Reset to empty title if no shell is set
			setTerminalTitle("")
		}
	}
}

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
	historyHandler *EncryptedHistoryHandler // For history-based completion
	cmdCache       map[string]bool          // Cache of executable commands
	cmdCacheMutex  sync.RWMutex             // Mutex for thread-safe access to cmdCache
	cmdCacheInit   sync.Once                // Used to initialize the command cache once
	cmdDirs        []string                 // Directories in PATH

	// Special command completions
	internalCmds map[string][]string // Map of internal commands to their subcommands
}

// NewDeltaCompleter creates a new tab completer with the given history handler
func NewDeltaCompleter(historyHandler *EncryptedHistoryHandler) *DeltaCompleter {
	// Initialize internal commands for completion
	internalCmds := map[string][]string{
		"ai":        {"on", "off", "model", "status"},
		"help":      {},
		"jump":      {"add", "remove", "rm", "import", "list"},
		"j":         {},
		"memory":    {"enable", "disable", "status", "stats", "clear", "config", "list", "export", "train"},
		"mem":       {"enable", "disable", "status", "stats", "clear", "config", "list", "export", "train"},
		"tokenizer": {"status", "stats", "process", "vocab", "test", "help"},
		"tok":       {"status", "stats", "process", "vocab", "test", "help"},
		"inference": {"enable", "disable", "status", "stats", "feedback", "model", "examples", "config", "help"},
		"inf":       {"enable", "disable", "status", "stats", "feedback", "model", "examples", "config", "help"},
		"training":  {"extract", "stats", "evaluate", "help"},
		"train":     {"extract", "stats", "evaluate", "help"},
		"learning":  {"status", "enable", "disable", "feedback", "train", "patterns", "process", "stats", "config", "help"},
		"learn":     {"status", "enable", "disable", "feedback", "train", "patterns", "process", "stats", "config", "help"},
		"feedback":  {"helpful", "unhelpful", "correction"},
		"init":      {},
		"validate":  {},
		"v":         {},
		"validation": {"check", "safety", "config", "help"},
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

		// Special handling for multi-level commands
		if len(parts) >= 3 && (cmd == "knowledge" || cmd == "know") && subCmd == "agent" {
			// We're completing a third-level command
			subSubCmd := parts[2]

			// Check if the multi-level command exists
			multiCmd := cmd + " " + subCmd
			if subSubCmds, ok := c.internalCmds[multiCmd]; ok {
				// Find matching subcommands
				var matches []string
				matchPrefix := ":" + cmd + " " + subCmd + " "

				for _, ssc := range subSubCmds {
					if strings.HasPrefix(ssc, subSubCmd) {
						matches = append(matches, matchPrefix+ssc)
					}
				}
				sort.Strings(matches)
				return matches
			}
		}

		// Special handling for jump commands
		if cmd == "jump" || cmd == "j" {
			// Get locations from JumpManager
			jm := GetJumpManager()
			if jm != nil {
				var matches []string
				matchPrefix := ":" + cmd + " "

				for _, loc := range jm.ListLocations() {
					if strings.HasPrefix(loc, subCmd) {
						matches = append(matches, matchPrefix+loc)
					}
				}
				sort.Strings(matches)
				return matches
			}
		}

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

// Show help for internal commands
func showHelp() {
	// Call the enhanced help function from help.go
	showEnhancedHelp()
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
	case "art2":
		return HandleART2Command(args)
	case "help":
		showHelp()
		return true
	case "i18n", "lang", "locale":
		return HandleI18nCommand(args)
	case "jump", "j":
		return HandleJumpCommand(args)
	case "memory", "mem":
		return HandleMemoryCommand(args)
	case "tokenizer", "tok":
		return HandleTokenizerCommand(args)
	case "inference", "inf":
		return HandleInferenceCommand(args)
	case "training", "train":
		return HandleTrainingCommand(args)
	case "learning", "learn":
		return HandleLearningCommand(args)
	case "vector":
		return HandleVectorCommand(args)
	case "embedding":
		return HandleEmbeddingCommand(args)
	case "speculative", "specd":
		return HandleSpeculativeCommand(args)
	case "knowledge", "know":
		return HandleKnowledgeCommand(args)
	case "agent":
		return HandleAgentCommand(args)
	case "config":
		return HandleConfigCommand(args)
	case "pattern", "pat":
		return HandlePatternCommand(args)
	case "spellcheck", "spell":
		return HandleSpellCheckCommand(args)
	case "history", "hist":
		return HandleHistoryCommand(args)
	case "docs":
		return cmds.HandleDocsCommand(args)
	case "man":
		return HandleManCommand(args)
	case "update":
		return HandleUpdateCommand(args)
	case "feedback":
		// Shorthand for inference feedback
		if im := GetInferenceManager(); im != nil {
			feedbackType := "helpful"
			correction := ""

			if len(args) > 0 {
				feedbackType = args[0]
			}

			// Combine all remaining arguments as the correction text
			if len(args) > 1 {
				correction = strings.Join(args[1:], " ")
			}

			addInferenceFeedback(im, feedbackType, correction)
			return true
		}
		fmt.Println("Inference system not available. Enable it with ':inference enable'")
		return true
	case "init":
		return handleInitCommand()
	case "suggest", "s":
		return handleSuggestCommand(args)
	case "validate", "v":
		return HandleValidationCommand(args)
	case "validation":
		return HandleValidationCommand(args)
	default:
		// Check for typos and suggest corrections
		if sc := GetSpellChecker(); sc != nil && sc.IsEnabled() {
			// Check for spelling errors and get suggestions
			suggestions := sc.CheckCommand(command)

			if len(suggestions) > 0 {
				fmt.Printf("Unknown command: %s\n", command)
				fmt.Println(sc.GetCorrectionText(command, suggestions))

				// Auto-correct if enabled and confidence is high enough
				if sc.ShouldAutoCorrect(suggestions) {
					correctedCmd := ":" + suggestions[0].Command
					fmt.Printf("Auto-correcting to: %s\n", correctedCmd)

					// Record the correction
					sc.RecordCorrection(command, correctedCmd)

					// Execute the corrected command
					return handleInternalCommand(correctedCmd)
				}
			} else {
				fmt.Printf("Unknown command: %s\n", command)
				fmt.Println("Type :help for a list of available commands.")
			}
		} else {
			fmt.Printf("Unknown command: %s\n", command)
			fmt.Println("Type :help for a list of available commands.")
		}
		return true
	}
}

// handleInitCommand initializes all required configuration files
func handleInitCommand() bool {
	fmt.Println("Initializing Delta CLI configuration...")

	// Initialize JumpManager (creates config directory and jump_locations.json)
	jm := GetJumpManager()
	if jm != nil {
		fmt.Printf("Jump locations config created at: %s\n", jm.configPath)
	}

	// Initialize AI Manager
	ai := GetAIManager()
	if ai != nil {
		fmt.Println("AI assistant initialized")
	}

	// Initialize Memory Manager
	mm := GetMemoryManager()
	if mm != nil {
		err := mm.Initialize()
		if err == nil {
			fmt.Printf("Memory system initialized at: %s\n", mm.storagePath)
		} else {
			fmt.Printf("Warning: Failed to initialize memory system: %v\n", err)
		}
	}

	// Initialize Tokenizer
	tok := GetTokenizer()
	if tok != nil {
		fmt.Printf("Tokenizer initialized with %d vocabulary tokens\n", tok.GetVocabularySize())
	}

	// Initialize Inference Manager
	inf := GetInferenceManager()
	if inf != nil {
		err := inf.Initialize()
		if err == nil {
			fmt.Println("Inference system initialized")
			if inf.learningConfig.CollectFeedback {
				fmt.Println("Learning feedback collection is enabled")
			} else {
				fmt.Println("Learning feedback collection is disabled")
			}
		} else {
			fmt.Printf("Warning: Failed to initialize inference system: %v\n", err)
		}
	}

	// Initialize Vector Database
	vdb := GetVectorDBManager()
	if vdb != nil {
		err := vdb.Initialize()
		if err == nil {
			fmt.Println("Vector database initialized")
		} else {
			fmt.Printf("Warning: Failed to initialize vector database: %v\n", err)
		}
	}

	// Initialize Embedding Manager
	em := GetEmbeddingManager()
	if em != nil {
		err := em.Initialize()
		if err == nil {
			fmt.Println("Embedding system initialized")
		} else {
			fmt.Printf("Warning: Failed to initialize embedding system: %v\n", err)
		}
	}

	// Initialize Speculative Decoder
	sd := GetSpeculativeDecoder()
	if sd != nil {
		err := sd.Initialize()
		if err == nil {
			fmt.Println("Speculative decoding initialized")
		} else {
			fmt.Printf("Warning: Failed to initialize speculative decoding: %v\n", err)
		}
	}

	// Initialize Knowledge Extractor
	ke := GetKnowledgeExtractor()
	if ke != nil {
		err := ke.Initialize()
		if err == nil {
			fmt.Println("Knowledge extractor initialized")
		} else {
			fmt.Printf("Warning: Failed to initialize knowledge extractor: %v\n", err)
		}
	}

	// Initialize Agent Manager
	am := GetAgentManager()
	if am != nil {
		err := am.Initialize()
		if err == nil {
			fmt.Println("Agent manager initialized")
		} else {
			fmt.Printf("Warning: Failed to initialize agent manager: %v\n", err)
		}
	}

	// Initialize Config Manager
	cm := GetConfigManager()
	if cm != nil {
		err := cm.Initialize()
		if err == nil {
			fmt.Println("Configuration manager initialized")
		} else {
			fmt.Printf("Warning: Failed to initialize configuration manager: %v\n", err)
		}
	}

	// Initialize spell checker
	sc := GetSpellChecker()
	if sc != nil {
		err := sc.Initialize()
		if err == nil {
			fmt.Println("Spell checker initialized")
		} else {
			fmt.Printf("Warning: Failed to initialize spell checker: %v\n", err)
		}
	}

	// Initialize history analyzer
	ha := GetHistoryAnalyzer()
	if ha != nil {
		err := ha.Initialize()
		if err == nil {
			fmt.Println("History analyzer initialized")
		} else {
			fmt.Printf("Warning: Failed to initialize history analyzer: %v\n", err)
		}
	}

	// Initialize ART-2 machine learning system
	art2Mgr := GetART2Manager()
	if art2Mgr != nil {
		err := art2Mgr.Initialize()
		if err == nil {
			fmt.Println("ART-2 pattern recognition system initialized")
		} else {
			fmt.Printf("Warning: Failed to initialize ART-2 system: %v\n", err)
		}
	}

	// Initialize ART-2 preprocessor
	art2Preprocessor := GetART2Preprocessor()
	if art2Preprocessor != nil {
		stats := art2Preprocessor.GetVocabularyStats()
		fmt.Printf("ART-2 preprocessor initialized with %d vocabulary terms\n", stats["vocabulary_size"].(int))
	}

	// Initialize Update Manager
	um := GetUpdateManager()
	if um != nil {
		err := um.Initialize()
		if err == nil {
			fmt.Println("Update system initialized")
			// Perform startup update check if configured
			um.PerformStartupCheck()
		} else {
			fmt.Printf("Warning: Failed to initialize update system: %v\n", err)
		}
	}

	// Initialize history file
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = os.Getenv("HOME")
	}
	historyFile := filepath.Join(homeDir, ".delta_history")

	// Create an empty history file if it doesn't exist
	if _, err := os.Stat(historyFile); os.IsNotExist(err) {
		// Create directory if it doesn't exist
		dir := filepath.Dir(historyFile)
		if err := os.MkdirAll(dir, 0755); err == nil {
			// Initialize empty history
			historyHandler := NewEncryptedHistory(historyFile)
			historyHandler.WriteHistory([]string{})
			fmt.Printf("History file created at: %s\n", historyFile)
		}
	} else {
		fmt.Printf("History file already exists at: %s\n", historyFile)
	}

	fmt.Println("Delta CLI configuration initialized successfully!")
	return true
}

// GetAIManager returns the global AI manager instance
func GetAIManager() *AIPredictionManager {
	// Initialize AI manager if needed
	if globalAIManager == nil {
		var err error
		
		// Check if we have a persisted configuration
		cm := GetConfigManager()
		if cm != nil && cm.GetAIConfig() != nil {
			aiConfig := cm.GetAIConfig()
			globalAIManager, err = NewAIPredictionManager(aiConfig.ServerURL, aiConfig.ModelName)
			if err == nil && globalAIManager != nil {
				// Apply the persisted configuration
				globalAIManager.config = *aiConfig
				globalAIManager.predictionEnabled = aiConfig.Enabled
				if aiConfig.MaxHistory > 0 {
					globalAIManager.maxHistorySize = aiConfig.MaxHistory
				}
				if aiConfig.ContextPrompt != "" {
					globalAIManager.contextPrompt = aiConfig.ContextPrompt
				}
			}
		} else {
			// Use defaults if no config exists
			globalAIManager, err = NewAIPredictionManager("http://localhost:11434", "phi4:latest")
		}
		
		// Initialize the AI manager
		if err == nil && globalAIManager != nil {
			globalAIManager.Initialize()
		}
	}
	return globalAIManager
}

// Handle AI-specific commands
func handleAICommand(args []string) bool {
	// Get a reference to the global AI manager
	ai := GetAIManager()
	if ai == nil {
		fmt.Println("AI features unavailable - failed to initialize AI manager")
		return true
	}

	if len(args) == 0 {
		fmt.Println("AI assistant is currently", getAIStatusText())
		return true
	}

	switch args[0] {
	case "on", "enable":
		ai.EnablePredictions()
		// Persist the configuration
		cm := GetConfigManager()
		if cm != nil {
			cm.UpdateAIConfig(&ai.config)
		}
		// Try to initialize if not already initialized
		if !ai.isInitialized {
			ai.Initialize()
		}
		fmt.Println("AI assistant enabled")
		return true

	case "off", "disable":
		ai.DisablePredictions()
		// Persist the configuration
		cm := GetConfigManager()
		if cm != nil {
			cm.UpdateAIConfig(&ai.config)
		}
		fmt.Println("AI assistant disabled")
		return true

	case "model":
		if len(args) < 2 {
			// Show current model information
			fmt.Println("Current AI model:", ai.ollamaClient.ModelName)

			// Check if using a custom model from inference system
			infMgr := GetInferenceManager()
			if infMgr != nil && infMgr.IsEnabled() && infMgr.learningConfig.UseCustomModel {
				fmt.Printf("Custom trained model: %s\n", infMgr.learningConfig.CustomModelPath)
			}

			// Show model options
			fmt.Println("\nAvailable commands:")
			fmt.Println("  :ai model <model_name>    - Use specified Ollama model")
			fmt.Println("  :ai model custom <path>   - Use custom trained model")
			fmt.Println("  :ai model default         - Use default Ollama model")
			return true
		}

		if args[1] == "custom" {
			// Use custom trained model
			if len(args) < 3 {
				fmt.Println("Please specify the path to the custom model")
				fmt.Println("Usage: :ai model custom <path_to_model>")
				return true
			}

			// Check inference manager availability
			infMgr := GetInferenceManager()
			if infMgr == nil {
				fmt.Println("Inference system not available")
				return true
			}

			modelPath := args[2]

			// Update to use custom model
			err := ai.UpdateModel(modelPath, true)
			if err != nil {
				fmt.Printf("Error setting custom model: %v\n", err)
				return true
			}

			fmt.Printf("Now using custom trained model: %s\n", modelPath)
			return true

		} else if args[1] == "default" {
			// Revert to default Ollama model
			infMgr := GetInferenceManager()
			if infMgr != nil && infMgr.IsEnabled() {
				// Disable custom model in inference config
				inferenceConfig := infMgr.inferenceConfig
				learningConfig := infMgr.learningConfig

				learningConfig.UseCustomModel = false
				inferenceConfig.UseLocalInference = false

				infMgr.UpdateConfig(inferenceConfig, learningConfig)
				fmt.Println("Reverted to default Ollama model")
			}

			// Make sure Ollama model is set
			ai.ollamaClient.ModelName = "phi4:latest" // Default model
			fmt.Println("Using Ollama model:", ai.ollamaClient.ModelName)
			return true
		} else {
			// Set a new Ollama model
			newModel := args[1]

			// Disable any custom model first
			infMgr := GetInferenceManager()
			if infMgr != nil && infMgr.IsEnabled() && infMgr.learningConfig.UseCustomModel {
				// Disable custom model in inference config
				inferenceConfig := infMgr.inferenceConfig
				learningConfig := infMgr.learningConfig

				learningConfig.UseCustomModel = false
				inferenceConfig.UseLocalInference = false

				infMgr.UpdateConfig(inferenceConfig, learningConfig)
			}

			// Update Ollama model
			err := ai.UpdateModel(newModel, false)
			if err != nil {
				fmt.Printf("Error: %v\n", err)
				fmt.Println("You can pull the model with: ollama pull " + newModel)
				return true
			}

			fmt.Println("AI model set to:", newModel)
			return true
		}

	case "status":
		fmt.Println("AI Status:")
		fmt.Println("- Enabled:", ai.IsEnabled())
		fmt.Println("- Model:", ai.ollamaClient.ModelName)
		fmt.Println("- Available:", ai.isInitialized)

		// Show health monitor status
		if ai.healthMonitor != nil {
			isRunning, lastCheck, lastStatus := ai.healthMonitor.GetStatus()
			fmt.Println("\nHealth Monitor:")
			fmt.Println("- Monitor enabled:", ai.config.HealthMonitorEnabled)
			fmt.Println("- Running:", isRunning)
			if !lastCheck.IsZero() {
				fmt.Printf("- Last check: %s ago\n", time.Since(lastCheck).Round(time.Second))
			}
			fmt.Println("- Ollama status:", func() string {
				if lastStatus {
					return "Available"
				}
				return "Unavailable"
			}())
			fmt.Printf("- Check interval: %d seconds\n", ai.config.HealthCheckInterval)
			fmt.Println("- Notifications:", ai.config.HealthNotificationEnabled)
		}

		// Check for custom trained model
		infMgr := GetInferenceManager()
		if infMgr != nil && infMgr.IsEnabled() {
			fmt.Println("\nLearning System:")
			fmt.Println("- Enabled:", infMgr.IsEnabled())
			if infMgr.learningConfig.UseCustomModel {
				fmt.Println("- Using custom trained model:", infMgr.learningConfig.CustomModelPath)
			}

			stats := infMgr.GetInferenceStats()
			fmt.Printf("- Training examples: %d\n", stats["training_examples"].(int))
			fmt.Printf("- Feedback entries: %d\n", stats["feedback_count"].(int))

			if infMgr.ShouldTrain() {
				fmt.Println("- Training status: Due for training")
			} else {
				fmt.Printf("- Training status: %d examples collected since last training\n",
					infMgr.learningConfig.AccumulatedTrainingExamples)
			}
		}
		return true

	case "health":
		// Health monitoring configuration
		if len(args) < 2 {
			// Show current health monitoring status
			if ai.healthMonitor != nil {
				isRunning, lastCheck, lastStatus := ai.healthMonitor.GetStatus()
				fmt.Println("Health Monitoring Configuration:")
				fmt.Println("- Enabled:", ai.config.HealthMonitorEnabled)
				fmt.Println("- Monitor running:", isRunning)
				fmt.Printf("- Check interval: %d seconds\n", ai.config.HealthCheckInterval)
				fmt.Println("- Notifications enabled:", ai.config.HealthNotificationEnabled)
				if !lastCheck.IsZero() {
					fmt.Printf("- Last checked: %s ago\n", time.Since(lastCheck).Round(time.Second))
					fmt.Printf("- Ollama status: %s\n", func() string {
						if lastStatus {
							return "Available"
						}
						return "Unavailable"
					}())
				}
			}
			fmt.Println("\nUsage:")
			fmt.Println("  :ai health monitor <on|off>     - Enable/disable health monitoring")
			fmt.Println("  :ai health interval <seconds>   - Set check interval")
			fmt.Println("  :ai health notify <on|off>      - Enable/disable notifications")
			return true
		}

		switch args[1] {
		case "monitor":
			if len(args) < 3 {
				fmt.Println("Please specify 'on' or 'off'")
				return true
			}
			enabled := args[2] == "on"
			ai.config.HealthMonitorEnabled = enabled
			
			// Update the monitor state
			if ai.healthMonitor != nil {
				if enabled && !ai.IsEnabled() {
					ai.healthMonitor.SetMonitorEnabled(true)
					ai.healthMonitor.Start()
					fmt.Println("Health monitoring enabled")
				} else {
					ai.healthMonitor.Stop()
					fmt.Println("Health monitoring disabled")
				}
			}
			
			// Persist configuration
			cm := GetConfigManager()
			if cm != nil {
				cm.UpdateAIConfig(&ai.config)
			}
			return true

		case "interval":
			if len(args) < 3 {
				fmt.Printf("Current interval: %d seconds\n", ai.config.HealthCheckInterval)
				fmt.Println("Usage: :ai health interval <seconds>")
				return true
			}
			interval, err := strconv.Atoi(args[2])
			if err != nil || interval < 10 {
				fmt.Println("Invalid interval. Please specify a value of at least 10 seconds")
				return true
			}
			ai.config.HealthCheckInterval = interval
			
			// Update the monitor
			if ai.healthMonitor != nil {
				ai.healthMonitor.SetCheckInterval(time.Duration(interval) * time.Second)
			}
			
			// Persist configuration
			cm := GetConfigManager()
			if cm != nil {
				cm.UpdateAIConfig(&ai.config)
			}
			fmt.Printf("Health check interval set to %d seconds\n", interval)
			return true

		case "notify":
			if len(args) < 3 {
				fmt.Println("Please specify 'on' or 'off'")
				return true
			}
			enabled := args[2] == "on"
			ai.config.HealthNotificationEnabled = enabled
			
			// Update the monitor
			if ai.healthMonitor != nil {
				ai.healthMonitor.SetEnabled(enabled)
			}
			
			// Persist configuration
			cm := GetConfigManager()
			if cm != nil {
				cm.UpdateAIConfig(&ai.config)
			}
			
			if enabled {
				fmt.Println("Health notifications enabled")
			} else {
				fmt.Println("Health notifications disabled")
			}
			return true

		default:
			fmt.Println("Unknown health command:", args[1])
			fmt.Println("Use ':ai health' to see available options")
			return true
		}

	case "feedback":
		// Shorthand for inference feedback
		if len(args) < 2 {
			fmt.Println("Please specify feedback type: helpful, unhelpful, or correction")
			fmt.Println("Usage: :ai feedback <helpful|unhelpful|correction> [correction]")
			return true
		}

		// Get the inference manager
		infMgr := GetInferenceManager()
		if infMgr == nil {
			fmt.Println("Inference system not available")
			return true
		}

		feedbackType := args[1]
		correction := ""
		if len(args) > 2 {
			correction = args[2]
		}

		// Add feedback
		addInferenceFeedback(infMgr, feedbackType, correction)
		return true

	case "help":
		fmt.Println("AI Assistant Commands")
		fmt.Println("====================")
		fmt.Println("  :ai              - Show AI status")
		fmt.Println("  :ai on           - Enable AI assistant (alias: :ai enable)")
		fmt.Println("  :ai off          - Disable AI assistant (alias: :ai disable)")
		fmt.Println("  :ai model        - Show current model")
		fmt.Println("  :ai model <name> - Switch to specified Ollama model")
		fmt.Println("  :ai model custom <path> - Use custom trained model")
		fmt.Println("  :ai model default - Revert to default model")
		fmt.Println("  :ai status       - Show detailed AI status")
		fmt.Println("  :ai health       - Show health monitoring status")
		fmt.Println("  :ai health monitor <on|off> - Enable/disable health monitoring")
		fmt.Println("  :ai health interval <sec>   - Set check interval (min 10s)")
		fmt.Println("  :ai health notify <on|off>  - Enable/disable notifications")
		fmt.Println("  :ai feedback <helpful|unhelpful|correction> [correction]")
		fmt.Println("                   - Provide feedback on last prediction")
		fmt.Println("  :ai help         - Show this help message")
		return true

	default:
		fmt.Printf("Unknown AI command: %s\n", args[0])
		fmt.Println("Type :ai help for available commands")
		return true
	}
}

// Get the AI status as a formatted text
func getAIStatusText() string {
	ai := GetAIManager()
	if ai == nil {
		return "unavailable"
	}

	if ai.IsEnabled() {
		return fmt.Sprintf("enabled (using %s)", ai.ollamaClient.ModelName)
	}
	return "disabled"
}

// formatThought formats an AI thought with appropriate styling
func formatThought(thought string) string {
	// Process double-asterisk highlighted sections
	var result strings.Builder
	parts := strings.Split(thought, "**")

	// If we have asterisk-marked sections
	if len(parts) > 1 {
		// Start with the text before first **
		result.WriteString(parts[0])

		// Process each part
		for i := 1; i < len(parts); i++ {
			if i%2 == 1 { // This is text inside ** **
				// Get appropriate emoji for the content
				emoji := chooseEmoji(parts[i])
				// Add on a new line with emoji
				result.WriteString("\n" + emoji + " " + parts[i])
			} else { // This is text after a ** ** section
				result.WriteString(parts[i])
			}
		}

		return fmt.Sprintf("\033[32m[∆ thinking: %s]\033[0m", result.String())
	}

	// If no ** sections, return the original (in green color)
	return fmt.Sprintf("\033[32m[∆ thinking: %s]\033[0m", thought)
}

// chooseEmoji selects an appropriate emoji based on text content
func chooseEmoji(text string) string {
	text = strings.ToLower(text)

	// Map keywords to emojis
	if strings.Contains(text, "error") || strings.Contains(text, "fail") {
		return "⚠️"
	} else if strings.Contains(text, "success") || strings.Contains(text, "complete") {
		return "✅"
	} else if strings.Contains(text, "warning") {
		return "⚡"
	} else if strings.Contains(text, "good thought") {
		return "💭"
	} else if strings.Contains(text, "helpful thought") {
		return "✨"
	} else if strings.Contains(text, "tip") || strings.Contains(text, "hint") || strings.Contains(text, "suggestion") {
		return "💡"
	} else if strings.Contains(text, "info") || strings.Contains(text, "note") {
		return "ℹ️"
	} else if strings.Contains(text, "question") {
		return "❓"
	} else if strings.Contains(text, "important") {
		return "‼️"
	} else if strings.Contains(text, "todo") || strings.Contains(text, "task") {
		return "📋"
	} else if strings.Contains(text, "file") || strings.Contains(text, "document") {
		return "📄"
	} else if strings.Contains(text, "folder") || strings.Contains(text, "directory") {
		return "📁"
	} else if strings.Contains(text, "github") || strings.Contains(text, "git") {
		return "🔄"
	} else if strings.Contains(text, "configuration") || strings.Contains(text, "config") || strings.Contains(text, "settings") {
		return "🔬"
	}

	// Default emoji
	return "🔹"
}

// Global variable for AI manager
var globalAIManager *AIPredictionManager

func runInteractiveShell() {
	// Determine what the original terminal title should be restored to
	originalTerminalTitle = getOriginalTerminalTitle()

	// Set the initial terminal title
	setDeltaTitle()

	// Initialize i18n system first
	initializeI18nSystem()

	fmt.Println(T("interface.welcome.message"))
	fmt.Println()

	// Now that all components have been created with default configs,
	// we can update them with the configuration from the config manager
	configManager := GetConfigManager()
	if configManager != nil && configManager.isInitialized {
		// Full initialization to update component configs
		if err := configManager.Initialize(); err != nil {
			fmt.Printf("Warning: Error updating component configurations: %v\n", err)
		}
	}

	// Initialize AI features
	// Use GetAIManager to ensure we have a single instance
	ai := GetAIManager()

	// Try to initialize AI with the updated config, showing enabled message only if successful
	if ai != nil {
		// Initialize will now handle displaying errors only once
		if ai.Initialize() {
			fmt.Printf("\033[33m%s\033[0m\n", T("interface.welcome.features_enabled", TranslationParams{
				"feature": "AI features",
				"details": T("interface.ai.features_enabled", TranslationParams{"model": ai.ollamaClient.ModelName}),
			}))
		}
	}

	// Initialize inference system for learning capabilities
	infMgr := GetInferenceManager()
	if infMgr != nil && infMgr.IsEnabled() {
		infMgr.Initialize()
		fmt.Println("\033[33m[∆ Learning system enabled: " +
			fmt.Sprintf("%d training examples collected]\033[0m",
				infMgr.learningConfig.AccumulatedTrainingExamples))
	}

	// Initialize agent manager
	am := GetAgentManager()
	if am != nil && am.IsEnabled() {
		am.Initialize()
		fmt.Println("\033[33m[∆ Agent system enabled: Task automation active]\033[0m")
	}

	// Initialize all components first with default configurations
	// Then we'll update them with configs from the config manager

	// Initialize config manager (base only) to prevent circular dependencies
	cm := GetConfigManager()
	if cm != nil {
		// Initialize config manager without updating components
		if err := cm.InitializeBase(); err != nil {
			fmt.Printf("Warning: Could not initialize configuration system: %v\n", err)
		} else {
			fmt.Println("\033[33m[∆ Configuration system enabled: Centralized settings management]\033[0m")
		}
	}

	// Initialize spell checker with default config
	sc := GetSpellChecker()
	if sc != nil {
		sc.Initialize()
		if sc.config.Enabled {
			fmt.Println("\033[33m[∆ Spell checking enabled: Command correction active]\033[0m")
		}
	}

	// Initialize history analyzer
	ha := GetHistoryAnalyzer()
	if ha != nil {
		ha.Initialize()
		if ha.config.Enabled {
			fmt.Println("\033[33m[∆ History analysis enabled: Command suggestion active]\033[0m")
		}
	}

	// Initialize ART-2 machine learning system
	art2Mgr := GetART2Manager()
	if art2Mgr != nil {
		art2Mgr.Initialize()
		if art2Mgr.IsEnabled() {
			fmt.Println("\033[33m[∆ ART-2 learning enabled: Adaptive pattern recognition active]\033[0m")
		}
	}
	
	// Check i18n installation status and show notice if needed
	checkI18nStartup()

	// Set up cleanup for AI resources on exit
	defer func() {
		if ai != nil && ai.cancelFunc != nil {
			ai.cancelFunc() // Cancel any pending AI requests
		}
	}()

	historyFile := os.Getenv("HOME") + "/.delta_history"
	historyLimit := 500

	// Initialize our encrypted history handler
	historyHandler, err := NewEncryptedHistoryHandler(historyFile, historyLimit)
	if err != nil {
		fmt.Println("Error initializing history:", err)
	}

	// Create our completer with extended commands
	internalCmds := map[string][]string{
		"ai":              {"on", "off", "model", "custom", "default", "status", "health", "feedback", "help"},
		"art2":            {"enable", "disable", "status", "stats", "categories", "predict", "feedback", "config", "help"},
		"help":            {},
		"jump":            {"add", "remove", "rm", "import", "list"},
		"j":               {},
		"memory":          {"enable", "disable", "status", "stats", "clear", "config", "list", "export", "train"},
		"mem":             {"enable", "disable", "status", "stats", "clear", "config", "list", "export", "train"},
		"tokenizer":       {"status", "stats", "process", "vocab", "test", "help"},
		"tok":             {"status", "stats", "process", "vocab", "test", "help"},
		"inference":       {"enable", "disable", "status", "stats", "feedback", "model", "examples", "config", "help"},
		"inf":             {"enable", "disable", "status", "stats", "feedback", "model", "examples", "config", "help"},
		"feedback":        {"helpful", "unhelpful", "correction"},
		"vector":          {"enable", "disable", "status", "stats", "search", "embed", "config", "help"},
		"embedding":       {"enable", "disable", "status", "stats", "generate", "config", "help"},
		"speculative":     {"enable", "disable", "status", "stats", "draft", "reset", "config", "help"},
		"specd":           {"enable", "disable", "status", "stats", "draft", "reset", "config", "help"},
		"knowledge":       {"enable", "disable", "status", "stats", "query", "context", "scan", "project", "extract", "clear", "export", "import", "agent", "help"},
		"know":            {"enable", "disable", "status", "stats", "query", "context", "scan", "project", "extract", "clear", "export", "import", "agent", "help"},
		"knowledge agent": {"suggest", "learn", "optimize", "create", "extract", "context", "triggers", "discover", "help"},
		"know agent":      {"suggest", "learn", "optimize", "create", "extract", "context", "triggers", "discover", "help"},
		"agent":           {"enable", "disable", "list", "show", "run", "create", "edit", "delete", "learn", "docker", "stats", "help"},
		"config":          {"status", "list", "export", "import", "edit", "reset", "help"},
		"pattern":         {"enable", "disable", "update", "versions", "list", "check", "auto", "interval", "status", "stats", "help"},
		"pat":             {"enable", "disable", "update", "versions", "list", "check", "auto", "interval", "status", "stats", "help"},
		"spellcheck":      {"enable", "disable", "status", "config", "add", "remove", "test", "help"},
		"spell":           {"enable", "disable", "status", "config", "add", "remove", "test", "help"},
		"history":         {"import", "show", "status", "stats", "enable", "disable", "search", "find", "suggest", "config", "mark", "patterns", "info", "help"},
		"hist":            {"import", "show", "status", "stats", "enable", "disable", "search", "find", "suggest", "config", "mark", "patterns", "info", "help"},
		"docs":            {"build", "dev", "open", "status", "help"},
		"update":          {"status", "config", "version", "help"},
		"init":            {},
		"suggest":         {"help", "explain", "last", "clear"},
		"s":               {},
		"validate":        {},
		"v":               {},
		"validation":      {"check", "safety", "config", "help"},
	}

	completer := NewDeltaCompleter(historyHandler)
	completer.internalCmds = internalCmds

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

	// Set up cleanup for AI manager on exit
	defer func() {
		if ai := GetAIManager(); ai != nil {
			ai.Cleanup()
		}
	}()

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

	// Set up signal handling for Ctrl+C and termination
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	// Set up cleanup on signal termination
	go func() {
		sig := <-c
		if sig == syscall.SIGTERM {
			fmt.Println("\nTerminating...")
			resetTerminalTitle()
			os.Exit(0)
		}
		// For SIGINT (Ctrl+C), just continue - it's handled in the main loop
	}()

	for {
		var prompt string
		if inSubCommand {
			prompt = "⬠ "
		} else {
			// Get current directory for the prompt
			pwd, err := os.Getwd()
			if err == nil {
				// Get the home directory to replace with ~
				home, err := os.UserHomeDir()
				if err == nil && strings.HasPrefix(pwd, home) {
					// Replace home directory with ~
					pwd = "~" + pwd[len(home):]
				}

				// Get just the last part of the path for brevity
				lastPart := filepath.Base(pwd)

				// Display the current directory in the prompt
				prompt = fmt.Sprintf("[%s] ∆ ", lastPart)
			} else {
				// Default prompt if we can't get the directory
				prompt = "∆ "
			}
		}
		rl.SetPrompt(prompt)

		// Display AI thought if available
		ai := GetAIManager()
		if ai != nil && ai.IsEnabled() {
			thought := ai.GetCurrentThought()
			if thought != "" {
				// Display thought above prompt using formatThought function
				fmt.Println(formatThought(thought))
				
				// Record the prediction for feedback collection
				if fc := GetFeedbackCollector(); fc != nil && fc.IsEnabled() {
					// Get the last command for context
					lastCmd, _, _ := ai.GetLastPrediction()
					if lastCmd != "" {
						fc.RecordPrediction(lastCmd, thought)
					}
				}
			}
		}

		// Wait for any background AI prediction tasks to complete
		if ai != nil {
			ai.Wait()
		}

		// Read input from the user with history support
		command, err := readInputWithContinuation(rl)
		if err != nil {
			if err == readline.ErrInterrupt {
				// Ctrl+C at prompt just clears the line
				continue
			} else if err == io.EOF {
				// Ctrl+D exits
				fmt.Printf("\n%s\n", T("interface.goodbye.message"))
				resetTerminalTitle()
				break
			}
			fmt.Println(T("interface.errors.reading_input", TranslationParams{"error": err.Error()}))
			continue
		}

		// Save command to encrypted history
		if command != "" && historyHandler != nil {
			historyHandler.Write(command)
		}

		// Process command with AI if enabled
		// We've already defined ai above, so just reuse it here
		if ai != nil && ai.IsEnabled() && command != "" {
			// Submit command to AI for analysis in the background
			go func(cmd string) {
				ai.AddCommand(cmd)
			}(command)
		}

		// Process command with ART-2 learning if enabled
		if art2Mgr := GetART2Manager(); art2Mgr != nil && art2Mgr.IsEnabled() && command != "" {
			// Submit command to ART-2 for pattern learning in the background
			go func(cmd string) {
				// Get current context
				dir, _ := os.Getwd()

				// Preprocess the command into a feature vector
				preprocessor := GetART2Preprocessor()
				if preprocessor != nil {
					featureVector, err := preprocessor.PreprocessCommand(cmd, "", dir)
					if err == nil {
						// Create ART-2 input
						art2Input := ART2Input{
							Vector:    featureVector.Values,
							Command:   cmd,
							Context:   dir,
							Timestamp: time.Now(),
						}

						// Process with ART-2 algorithm
						art2Mgr.ProcessInput(art2Input)
					}
				}
			}(command)
		}

		// Handle the exit command
		if command == "exit" || command == "quit" {
			fmt.Println(T("interface.goodbye.message"))
			resetTerminalTitle()
			break
		}

		// Check for subcommand mode
		if command == "sub" {
			inSubCommand = true
			fmt.Println(T("interface.navigation.entering_subcommand"))
			continue
		} else if command == "end" {
			inSubCommand = false
			fmt.Println(T("interface.navigation.exiting_subcommand"))
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
		exitCode, duration := runCommand(command, c)

		// Get the current directory for context
		dir, err := os.Getwd()
		if err != nil {
			dir = ""
		}

		// Record command in history analyzer if enabled
		if ha := GetHistoryAnalyzer(); ha != nil && ha.IsEnabled() {
			// Create command context
			ctx := CommandContext{
				Directory:   dir,
				Environment: map[string]string{}, // Minimal environment info for privacy
				Timestamp:   time.Now(),
				ExitCode:    exitCode,
				Duration:    duration,
			}

			// Record the command (this happens asynchronously)
			go ha.AddCommand(command, ctx)

			// Check if auto-suggest is enabled and show suggestions
			if ha.config.AutoSuggest {
				// Get suggestions for next command
				suggestions := ha.GetSuggestions(dir)
				if len(suggestions) > 0 && suggestions[0].Confidence > ha.config.MinConfidenceThreshold {
					// Format suggestion
					suggestion := suggestions[0]
					// Only display suggestion if it's not empty
					if suggestion.Command != "" {
						fmt.Printf("\n\033[2m∆ 💡 Next command suggestion: %s\033[0m\n", suggestion.Command)
					}
				}
			}
		}

		// Record command in memory manager if enabled
		if mm := GetMemoryManager(); mm != nil && mm.IsEnabled() {
			mm.AddCommand(command, dir, exitCode, duration.Milliseconds())
		}

		// Process with learning engine if enabled
		if le := GetLearningEngine(); le != nil && le.isEnabled {
			entry := CommandEntry{
				Command:     command,
				Directory:   dir,
				Timestamp:   time.Now(),
				ExitCode:    exitCode,
				Duration:    duration.Milliseconds(),
				Environment: map[string]string{},
			}
			go le.LearnFromCommand(entry)
		}

		// Collect implicit feedback if enabled
		if fc := GetFeedbackCollector(); fc != nil && fc.IsEnabled() {
			go fc.CollectImplicitFeedback(command, exitCode, duration)
		}
	}
}

// initializeManagers initializes essential managers for command execution
func initializeManagers() {
	// Initialize i18n system first (required for translations)
	initializeI18nSystem()
	
	// Initialize config manager base
	cm := GetConfigManager()
	if cm != nil {
		cm.InitializeBase()
	}
	
	// Initialize AI features (silent initialization for non-interactive)
	ai := GetAIManager()
	if ai != nil {
		ai.Initialize()
	}
	
	// Initialize inference system for learning capabilities
	infMgr := GetInferenceManager()
	if infMgr != nil && infMgr.IsEnabled() {
		infMgr.Initialize()
	}
	
	// Initialize agent manager
	am := GetAgentManager()
	if am != nil && am.IsEnabled() {
		am.Initialize()
	}
	
	// Initialize spell checker
	sc := GetSpellChecker()
	if sc != nil {
		sc.Initialize()
	}
	
	// Initialize history analyzer
	ha := GetHistoryAnalyzer()
	if ha != nil {
		ha.Initialize()
	}
	
	// Initialize ART-2 machine learning system
	art2Mgr := GetART2Manager()
	if art2Mgr != nil {
		art2Mgr.Initialize()
	}
}

// executeDirectCommand executes a single command and exits with appropriate code
func executeDirectCommand(command string) {
	// Initialize managers for command execution
	initializeManagers()
	
	var exitCode int
	var duration time.Duration
	
	// Check if it's an internal command (starts with :)
	if strings.HasPrefix(command, ":") {
		// Handle internal command
		startTime := time.Now()
		if handleInternalCommand(command) {
			exitCode = 0
		} else {
			exitCode = 1
		}
		duration = time.Since(startTime)
	} else {
		// Set up signal handling for external commands
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		
		// Execute external command IMMEDIATELY
		exitCode, duration = runCommand(command, sigChan)
	}
	
	// Launch background goroutine for all training/recording operations
	// This allows delta to exit immediately with the proper exit code
	go func() {
		// Process command with AI if enabled
		ai := GetAIManager()
		if ai != nil && ai.IsEnabled() && command != "" && !strings.HasPrefix(command, ":") {
			// Submit command to AI for analysis
			ai.AddCommand(command)
		}
		
		// Process command with ART-2 learning if enabled
		if art2Mgr := GetART2Manager(); art2Mgr != nil && art2Mgr.IsEnabled() && command != "" && !strings.HasPrefix(command, ":") {
			// Get current context
			dir, _ := os.Getwd()
			
			// Preprocess the command into a feature vector
			preprocessor := GetART2Preprocessor()
			if preprocessor != nil {
				featureVector, err := preprocessor.PreprocessCommand(command, "", dir)
				if err == nil {
					// Create ART-2 input
					art2Input := ART2Input{
						Vector:    featureVector.Values,
						Command:   command,
						Context:   dir,
						Timestamp: time.Now(),
					}
					
					// Process with ART-2 algorithm
					art2Mgr.ProcessInput(art2Input)
				}
			}
		}
		
		// Record command in history analyzer if enabled
		if ha := GetHistoryAnalyzer(); ha != nil && ha.IsEnabled() {
			// Get the current directory for context
			dir, err := os.Getwd()
			if err != nil {
				dir = ""
			}
			
			// Create command context
			ctx := CommandContext{
				Directory:   dir,
				Environment: map[string]string{}, // Minimal environment info for privacy
				Timestamp:   time.Now(),
				ExitCode:    exitCode,
				Duration:    duration,
			}
			
			// Record the command
			ha.AddCommand(command, ctx)
		}

		// Get directory for other managers
		dir, err := os.Getwd()
		if err != nil {
			dir = ""
		}

		// Record command in memory manager if enabled
		if mm := GetMemoryManager(); mm != nil && mm.IsEnabled() {
			mm.AddCommand(command, dir, exitCode, duration.Milliseconds())
		}

		// Process with learning engine if enabled
		if le := GetLearningEngine(); le != nil && le.isEnabled {
			entry := CommandEntry{
				Command:     command,
				Directory:   dir,
				Timestamp:   time.Now(),
				ExitCode:    exitCode,
				Duration:    duration.Milliseconds(),
				Environment: map[string]string{},
			}
			le.LearnFromCommand(entry)
		}

		// Collect implicit feedback if enabled
		if fc := GetFeedbackCollector(); fc != nil && fc.IsEnabled() {
			fc.CollectImplicitFeedback(command, exitCode, duration)
		}
	}()
	
	// Exit immediately with the command's exit code
	// Background goroutine will continue processing
	os.Exit(exitCode)
}

func main() {
	var cmdToRun string
	
	var rootCmd = &cobra.Command{
		Use:   "delta",
		Short: "Delta CLI - AI-powered shell enhancement",
		Long: `Delta CLI is an intelligent shell wrapper that enhances your command-line experience 
with AI-powered suggestions, encrypted command history, and seamless shell compatibility.

🌍 Multilingual Support: Available in 6 languages
💡 AI-Powered Shell Enhancement with Local Privacy`,
		Version: GetVersionInfo(),
		Run: func(cmd *cobra.Command, args []string) {
			// If a command was provided via flag, execute it directly
			if cmdToRun != "" {
				executeDirectCommand(cmdToRun)
				return
			}
			// Otherwise, run interactive shell
			runInteractiveShell()
		},
	}

	// Add command execution flags
	rootCmd.Flags().StringVarP(&cmdToRun, "command", "c", "", "Execute a single command and exit")
	rootCmd.Flags().StringVar(&cmdToRun, "cmd", "", "Execute a single command and exit (alternative)")

	rootCmd.SetVersionTemplate(GetVersionInfo() + "\n")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runCommand(command string, sigChan chan os.Signal) (int, time.Duration) {
	// Validate command before execution
	if !ValidateBeforeExecution(command) {
		// Command was cancelled by user or blocked by safety rules
		return 1, 0
	}
	
	// Parse the command to get the executable
	cmdParts := strings.Fields(command)
	if len(cmdParts) == 0 {
		return 0, 0
	}

	// Extract the program name for the terminal title
	programName := cmdParts[0]
	// Handle cases where the program is a path
	if strings.Contains(programName, "/") {
		programName = filepath.Base(programName)
	}

	// Set the terminal title to show the running program
	setProgramTitle(programName)

	// Check for built-in `jump` command to override external jump.sh
	if cmdParts[0] == "jump" {
		// Use our internal jump command instead
		args := []string{}
		if len(cmdParts) > 1 {
			args = cmdParts[1:]
		}
		// Start timing
		startTime := time.Now()
		result := HandleJumpCommand(args)
		duration := time.Since(startTime)

		// Return 0 for success, 1 for failure
		exitCode := 0
		if !result {
			exitCode = 1
		}

		// Restore the delta title after jump command completion
		setDeltaTitle()

		return exitCode, duration
	}

	// Handle cd command directly to change our own working directory
	if cmdParts[0] == "cd" {
		startTime := time.Now()

		// Default to home directory if no argument is given
		targetDir := ""
		if len(cmdParts) > 1 {
			targetDir = cmdParts[1]
		} else {
			var err error
			targetDir, err = os.UserHomeDir()
			if err != nil {
				fmt.Println("Error getting home directory:", err)
				return 1, time.Since(startTime)
			}
		}

		// Expand ~ to home directory
		if targetDir == "~" {
			var err error
			targetDir, err = os.UserHomeDir()
			if err != nil {
				fmt.Println("Error getting home directory:", err)
				return 1, time.Since(startTime)
			}
		} else if strings.HasPrefix(targetDir, "~/") {
			home, err := os.UserHomeDir()
			if err != nil {
				fmt.Println("Error getting home directory:", err)
				return 1, time.Since(startTime)
			}
			targetDir = filepath.Join(home, targetDir[2:])
		}

		// Handle special case for ..
		if targetDir == ".." {
			// Get current directory and go up one level
			pwd, err := os.Getwd()
			if err != nil {
				fmt.Println("Error getting current directory:", err)
				return 1, time.Since(startTime)
			}
			targetDir = filepath.Dir(pwd)
		}

		// Handle relative paths that don't start with ./ or ../
		if !filepath.IsAbs(targetDir) && !strings.HasPrefix(targetDir, "./") && !strings.HasPrefix(targetDir, "../") {
			// Combine with current directory
			pwd, err := os.Getwd()
			if err != nil {
				fmt.Println("Error getting current directory:", err)
				return 1, time.Since(startTime)
			}
			targetDir = filepath.Join(pwd, targetDir)
		}

		// Change the working directory
		err := os.Chdir(targetDir)
		if err != nil {
			fmt.Printf("cd: %v\n", err)
			// Restore the delta title after cd command failure
			setDeltaTitle()
			return 1, time.Since(startTime)
		}

		// Restore the delta title after cd command completion
		setDeltaTitle()
		return 0, time.Since(startTime)
	}

	// Handle pwd command directly
	if cmdParts[0] == "pwd" {
		startTime := time.Now()

		pwd, err := os.Getwd()
		if err != nil {
			fmt.Println("Error getting current directory:", err)
			// Restore the delta title after pwd command failure
			setDeltaTitle()
			return 1, time.Since(startTime)
		}
		fmt.Println(pwd)
		// Restore the delta title after pwd command completion
		setDeltaTitle()
		return 0, time.Since(startTime)
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
			exitCode, duration := runDirectCommand(shell, command, sigChan)
			// Restore the delta title after direct command completion
			setDeltaTitle()
			return exitCode, duration
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
	exitCode, duration := runShellCommand(shell, shellCmd, sigChan)

	// Restore the delta title after command completion
	setDeltaTitle()

	return exitCode, duration
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
func runDirectCommand(shell string, command string, sigChan chan os.Signal) (int, time.Duration) {
	// Create the command with default arguments
	shellArgs := []string{"-c", command}
	cmd := exec.Command(shell, shellArgs...)

	// Connect standard I/O
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	startTime := time.Now()
	err := executeCommand(cmd, sigChan)
	duration := time.Since(startTime)

	// Extract exit code if available
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1 // Generic error
		}
	}

	return exitCode, duration
}

// Execute a shell command with profile sourcing
func runShellCommand(shell string, shellCmd string, sigChan chan os.Signal) (int, time.Duration) {
	// Create command with the -c flag for all shells
	shellArgs := []string{"-c", shellCmd}
	cmd := exec.Command(shell, shellArgs...)

	// Connect standard I/O
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	startTime := time.Now()
	err := executeCommand(cmd, sigChan)
	duration := time.Since(startTime)

	// Extract exit code if available
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1 // Generic error
		}
	}

	return exitCode, duration
}

// readInputWithContinuation reads user input and handles backslash continuation for multi-line commands
func readInputWithContinuation(rl *readline.Instance) (string, error) {
	var lines []string
	var err error

	// Save the original prompt to restore it later
	originalPrompt := rl.Config.Prompt

	// Read the first line
	line, err := rl.Readline()
	if err != nil {
		return "", err
	}

	// Check if the line ends with a backslash (continuation)
	continuationMode := false
	for strings.HasSuffix(strings.TrimSpace(line), "\\") {
		// We're now in continuation mode
		continuationMode = true

		// Remove the trailing backslash
		line = strings.TrimSpace(line)
		line = line[:len(line)-1]

		// Add the line to our collection
		lines = append(lines, line)

		// Change prompt to indicate continuation using the pentagon symbol
		rl.SetPrompt("⬠ ")

		// Read the next line
		nextLine, nextErr := rl.Readline()
		if nextErr != nil {
			// If there's an error, return what we have so far
			if len(lines) > 0 {
				// Make sure to restore the original prompt before returning
				rl.SetPrompt(originalPrompt)
				return strings.Join(lines, " "), nil
			}
			return "", nextErr
		}

		// Update line for the next iteration
		line = nextLine
	}

	// Add the final line
	lines = append(lines, strings.TrimSpace(line))

	// Restore the original prompt if we were in continuation mode
	if continuationMode {
		rl.SetPrompt(originalPrompt)
	}

	// Join all lines with spaces
	fullCommand := strings.Join(lines, " ")

	// Trim any whitespace from the input
	return strings.TrimSpace(fullCommand), nil
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
