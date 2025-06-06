package main

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

// CommandToken represents a token in a command
type CommandToken struct {
	Text     string `json:"text"`
	Type     string `json:"type"`     // "command", "flag", "argument", "path", "pipe", "redirect", "variable", "quote", "operator"
	Position int    `json:"position"` // Position in the original command
}

// CommandTokens represents a tokenized command
type CommandTokens struct {
	Command   string         `json:"command"`   // Original command
	Directory string         `json:"directory"` // Working directory
	Tokens    []CommandToken `json:"tokens"`    // Tokenized command
	ExitCode  int            `json:"exit_code"` // Exit code of the command
}

// TokenizerConfig holds configuration for the tokenizer
type TokenizerConfig struct {
	Enabled        bool     `json:"enabled"`          // Whether tokenizer is enabled
	VocabSize      int      `json:"vocab_size"`       // Maximum vocabulary size
	MaxTokenLength int      `json:"max_token_length"` // Maximum token length
	CommonCommands []string `json:"common_commands"`  // List of common commands for special handling
	SpecialTokens  []string `json:"special_tokens"`   // Special tokens like [BOS], [EOS], [PAD]
	StoragePath    string   `json:"storage_path"`     // Path to store tokenizer data
}

// Tokenizer handles tokenization of commands
type Tokenizer struct {
	Config      TokenizerConfig     `json:"config"`       // Tokenizer configuration
	Vocabulary  map[string]int      `json:"vocabulary"`   // Maps tokens to IDs
	InvVocab    map[int]string      `json:"inv_vocab"`    // Maps IDs to tokens
	CommandList map[string]struct{} `json:"command_list"` // List of known commands
	Patterns    map[string]*regexp.Regexp
	configPath  string
	dataPath    string
	mutex       sync.RWMutex // Protects vocabulary modifications
}

// NewTokenizer creates a new tokenizer
func NewTokenizer() (*Tokenizer, error) {
	// Set up config directory in user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = os.Getenv("HOME")
	}

	// Use ~/.config/delta/memory/tokenizer directory
	configDir := filepath.Join(homeDir, ".config", "delta", "memory", "tokenizer")
	err = os.MkdirAll(configDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create tokenizer directory: %v", err)
	}

	configPath := filepath.Join(configDir, "tokenizer_config.json")
	dataPath := filepath.Join(configDir, "processed")
	err = os.MkdirAll(dataPath, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create data directory: %v", err)
	}

	// Create default tokenizer
	tokenizer := &Tokenizer{
		Config: TokenizerConfig{
			VocabSize:      10000,
			MaxTokenLength: 50,
			CommonCommands: []string{
				"ls", "cd", "grep", "cat", "echo", "find", "git", "mkdir", "rm", "mv",
				"cp", "sudo", "ssh", "docker", "python", "npm", "make", "curl", "wget",
			},
			SpecialTokens: []string{
				"[BOS]", "[EOS]", "[PAD]", "[UNK]", "[CMD]", "[ARG]", "[PATH]",
				"[FLAG]", "[PIPE]", "[VAR]", "[SUBST]", "[WILD]", "[NEWLINE]",
			},
		},
		Vocabulary:  make(map[string]int),
		InvVocab:    make(map[int]string),
		CommandList: make(map[string]struct{}),
		Patterns:    make(map[string]*regexp.Regexp),
		configPath:  configPath,
		dataPath:    dataPath,
	}

	// Initialize regex patterns
	tokenizer.initPatterns()

	// Try to load existing vocabulary
	err = tokenizer.loadVocabulary()
	if err != nil {
		// Initialize a basic vocabulary with special tokens
		tokenizer.initVocabulary()
		tokenizer.saveVocabulary()
	}

	return tokenizer, nil
}

// initPatterns initializes regex patterns for tokenization
func (t *Tokenizer) initPatterns() {
	// Command pattern
	t.Patterns["command"] = regexp.MustCompile(`^[a-zA-Z0-9_\-\.]+`)

	// Flag pattern (--flag, -f)
	t.Patterns["flag"] = regexp.MustCompile(`^-{1,2}[a-zA-Z0-9_\-]+`)

	// Path pattern
	t.Patterns["path"] = regexp.MustCompile(`^(?:[~/]|\.{1,2}/|[a-zA-Z0-9_\-\.]+/)[a-zA-Z0-9_\-\./]*`)

	// Environment variable
	t.Patterns["env_var"] = regexp.MustCompile(`^\$(?:\{[a-zA-Z0-9_]+\}|[a-zA-Z0-9_]+)`)

	// Pipe and redirections
	t.Patterns["pipe_redirect"] = regexp.MustCompile(`^[|><]{1,2}`)

	// Quote start
	t.Patterns["quote"] = regexp.MustCompile(`^["'` + "`" + `]`)

	// Operators
	t.Patterns["operator"] = regexp.MustCompile(`^(?:&&|\|\||;|\(|\)|\\|\$\(|\))`)

	// Glob patterns
	t.Patterns["glob"] = regexp.MustCompile(`^[a-zA-Z0-9_\-\./*?[\]]+`)
}

// initVocabulary initializes the basic vocabulary with special tokens
func (t *Tokenizer) initVocabulary() {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.Vocabulary = make(map[string]int)
	t.InvVocab = make(map[int]string)

	// Add special tokens
	for i, token := range t.Config.SpecialTokens {
		t.Vocabulary[token] = i
		t.InvVocab[i] = token
	}

	// Add common commands
	offset := len(t.Config.SpecialTokens)
	for i, cmd := range t.Config.CommonCommands {
		t.Vocabulary[cmd] = offset + i
		t.InvVocab[offset+i] = cmd
		t.CommandList[cmd] = struct{}{}
	}

	// Add common operators and separators
	operators := []string{"|", "||", "&&", ";", ">", ">>", "<", "2>", "2>>", "&"}
	offset = len(t.Config.SpecialTokens) + len(t.Config.CommonCommands)
	for i, op := range operators {
		t.Vocabulary[op] = offset + i
		t.InvVocab[offset+i] = op
	}
}

// loadVocabulary loads the vocabulary from disk
func (t *Tokenizer) loadVocabulary() error {
	// Check if vocabulary file exists
	if _, err := os.Stat(t.configPath); os.IsNotExist(err) {
		return fmt.Errorf("vocabulary file does not exist")
	}

	// Read the file
	data, err := os.ReadFile(t.configPath)
	if err != nil {
		return err
	}

	// Parse tokenizer data
	var tokenizer Tokenizer
	err = json.Unmarshal(data, &tokenizer)
	if err != nil {
		return err
	}

	// Update the current tokenizer
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.Config = tokenizer.Config
	t.Vocabulary = tokenizer.Vocabulary
	t.InvVocab = tokenizer.InvVocab
	t.CommandList = tokenizer.CommandList

	return nil
}

// saveVocabulary saves the vocabulary to disk
func (t *Tokenizer) saveVocabulary() error {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	// Marshal to JSON
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(t.configPath, data, 0644)
}

// TokenizeCommand tokenizes a command string
func (t *Tokenizer) TokenizeCommand(command string, directory string, exitCode int) (*CommandTokens, error) {
	tokens := &CommandTokens{
		Command:   command,
		Directory: directory,
		ExitCode:  exitCode,
		Tokens:    []CommandToken{},
	}

	// Initial normalization
	normalizedCmd := t.normalizeCommand(command)

	// Scan through the command
	remainingCmd := normalizedCmd
	position := 0

	// Track quote state
	inQuote := false
	quoteChar := rune(0)

	// First token is assumed to be the command itself
	isFirstToken := true

	for len(remainingCmd) > 0 {
		// Skip whitespace
		if strings.HasPrefix(remainingCmd, " ") || strings.HasPrefix(remainingCmd, "\t") {
			remainingCmd = strings.TrimLeft(remainingCmd, " \t")
			position += 1
			continue
		}

		// Handle quotes
		if inQuote {
			// Find the end of the quote
			endQuotePos := strings.IndexRune(remainingCmd[1:], quoteChar)
			if endQuotePos == -1 {
				// Unclosed quote, treat the rest as a single token
				endQuotePos = len(remainingCmd) - 1
			} else {
				endQuotePos += 1 // Adjust for the 1: offset in the IndexRune call
			}

			// Extract the quoted content (including the quotes)
			quotedContent := remainingCmd[:endQuotePos+1]
			tokenType := "argument"

			// Add as a token
			tokens.Tokens = append(tokens.Tokens, CommandToken{
				Text:     quotedContent,
				Type:     tokenType,
				Position: position,
			})

			// Move past the quoted content
			remainingCmd = remainingCmd[endQuotePos+1:]
			position += len(quotedContent)
			inQuote = false
			quoteChar = rune(0)
			continue
		}

		// Check for the start of a quote
		if strings.HasPrefix(remainingCmd, "\"") || strings.HasPrefix(remainingCmd, "'") || strings.HasPrefix(remainingCmd, "`") {
			inQuote = true
			quoteChar = rune(remainingCmd[0])
			continue
		}

		// Try to match patterns
		tokenFound := false
		for patternName, pattern := range t.Patterns {
			if matches := pattern.FindString(remainingCmd); matches != "" {
				tokenType := patternName

				// Determine token type based on position and pattern
				if isFirstToken && patternName == "command" {
					tokenType = "command"
					isFirstToken = false
				} else if patternName == "flag" {
					tokenType = "flag"
				} else if patternName == "path" {
					tokenType = "path"
				} else if patternName == "env_var" {
					tokenType = "variable"
				} else if patternName == "pipe_redirect" {
					tokenType = "redirect"
				} else if patternName == "operator" {
					tokenType = "operator"
				} else {
					tokenType = "argument"
				}

				// Add token
				tokens.Tokens = append(tokens.Tokens, CommandToken{
					Text:     matches,
					Type:     tokenType,
					Position: position,
				})

				// Move past the matched pattern
				remainingCmd = remainingCmd[len(matches):]
				position += len(matches)
				tokenFound = true
				break
			}
		}

		// If no pattern matched, take the next character as an individual token
		if !tokenFound {
			tokens.Tokens = append(tokens.Tokens, CommandToken{
				Text:     string(remainingCmd[0]),
				Type:     "unknown",
				Position: position,
			})
			remainingCmd = remainingCmd[1:]
			position += 1
		}
	}

	return tokens, nil
}

// normalizeCommand performs initial normalization on a command
func (t *Tokenizer) normalizeCommand(command string) string {
	// Replace tabs with spaces
	command = strings.ReplaceAll(command, "\t", " ")

	// Normalize multiple spaces to a single space
	re := regexp.MustCompile(`\s+`)
	command = re.ReplaceAllString(command, " ")

	// Normalize home directory references
	homeDir, err := os.UserHomeDir()
	if err == nil {
		command = strings.ReplaceAll(command, homeDir, "~")
	}

	return strings.TrimSpace(command)
}

// EncodeTokens converts TokenizedCommand to token IDs
func (t *Tokenizer) EncodeTokens(tokens *CommandTokens) ([]int, error) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	var ids []int

	// Add beginning of sequence token
	ids = append(ids, t.Vocabulary["[BOS]"])

	// Process each token
	for _, token := range tokens.Tokens {
		// Check if token is in vocabulary
		id, exists := t.Vocabulary[token.Text]
		if !exists {
			// Add special token based on type
			switch token.Type {
			case "command":
				ids = append(ids, t.Vocabulary["[CMD]"])
			case "argument":
				ids = append(ids, t.Vocabulary["[ARG]"])
			case "path":
				ids = append(ids, t.Vocabulary["[PATH]"])
			case "flag":
				ids = append(ids, t.Vocabulary["[FLAG]"])
			case "redirect":
				ids = append(ids, t.Vocabulary["[PIPE]"])
			case "variable":
				ids = append(ids, t.Vocabulary["[VAR]"])
			default:
				ids = append(ids, t.Vocabulary["[UNK]"])
			}

			// Try to add this token to vocabulary if it's not too large
			if len(token.Text) <= t.Config.MaxTokenLength {
				t.addToVocabulary(token.Text)
			}
		} else {
			ids = append(ids, id)
		}
	}

	// Add end of sequence token
	ids = append(ids, t.Vocabulary["[EOS]"])

	return ids, nil
}

// addToVocabulary adds a new token to the vocabulary if space allows
func (t *Tokenizer) addToVocabulary(token string) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	// Check if already in vocabulary
	if _, exists := t.Vocabulary[token]; exists {
		return
	}

	// Check if we have room in the vocabulary
	if len(t.Vocabulary) >= t.Config.VocabSize {
		// Vocabulary is full
		return
	}

	// Add to vocabulary
	id := len(t.Vocabulary)
	t.Vocabulary[token] = id
	t.InvVocab[id] = token

	// If it's a command, add to command list
	if len(token) <= 20 && !strings.ContainsAny(token, " /|<>\"'`;&") {
		t.CommandList[token] = struct{}{}
	}
}

// DecodeTokens converts token IDs back to a command string
func (t *Tokenizer) DecodeTokens(ids []int) (string, error) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	var builder strings.Builder

	// Skip BOS and EOS tokens if present
	startIdx := 0
	endIdx := len(ids)
	if len(ids) > 0 && ids[0] == t.Vocabulary["[BOS]"] {
		startIdx = 1
	}
	if len(ids) > 1 && ids[len(ids)-1] == t.Vocabulary["[EOS]"] {
		endIdx = len(ids) - 1
	}

	// Convert each token ID back to text
	for i := startIdx; i < endIdx; i++ {
		id := ids[i]

		// Get the token text
		text, exists := t.InvVocab[id]
		if !exists {
			text = "[UNK]"
		}

		// Skip special tokens in output
		if strings.HasPrefix(text, "[") && strings.HasSuffix(text, "]") {
			continue
		}

		builder.WriteString(text)

		// Add space after tokens except in certain cases
		if i < endIdx-1 && !strings.ContainsAny(text, " |<>&;") && !strings.HasSuffix(text, "/") {
			builder.WriteString(" ")
		}
	}

	return builder.String(), nil
}

// ProcessCommandBatch processes a batch of commands for training
func (t *Tokenizer) ProcessCommandBatch(commands []CommandEntry) error {
	var tokenizedCommands []CommandTokens

	// Tokenize each command
	for _, cmd := range commands {
		tokens, err := t.TokenizeCommand(cmd.Command, cmd.Directory, cmd.ExitCode)
		if err != nil {
			// Skip problematic commands but continue processing
			continue
		}

		tokenizedCommands = append(tokenizedCommands, *tokens)
	}

	// Save the processed batch
	if len(tokenizedCommands) > 0 {
		return t.saveTokenizedBatch(tokenizedCommands)
	}

	return nil
}

// saveTokenizedBatch saves tokenized commands to a file
func (t *Tokenizer) saveTokenizedBatch(tokens []CommandTokens) error {
	// Create a filename based on current time
	filename := fmt.Sprintf("tokenized_%d.bin", time.Now().Unix())
	filePath := filepath.Join(t.dataPath, filename)

	// Open file for writing
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)

	// Write number of entries
	entryCountBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(entryCountBytes, uint32(len(tokens)))
	writer.Write(entryCountBytes)

	// Write each tokenized command
	for _, tokenizedCmd := range tokens {
		// Marshal to JSON
		data, err := json.Marshal(tokenizedCmd)
		if err != nil {
			return err
		}

		// Write length of JSON data
		lenBytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(lenBytes, uint32(len(data)))
		writer.Write(lenBytes)

		// Write JSON data
		writer.Write(data)
	}

	// Save updated vocabulary
	t.saveVocabulary()

	return writer.Flush()
}

// GetVocabularySize returns the current size of the vocabulary
func (t *Tokenizer) GetVocabularySize() int {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return len(t.Vocabulary)
}

// ExtractCommandsFromShards processes command shards into training data
func (t *Tokenizer) ExtractCommandsFromShards(shardPaths []string) (int, error) {
	var processedCount int

	for _, shardPath := range shardPaths {
		// Open the shard file
		file, err := os.Open(shardPath)
		if err != nil {
			continue
		}

		// Read command entries
		var commands []CommandEntry

		// Process file in chunks
		buffer := make([]byte, 4)
		for {
			// Read entry length
			_, err := file.Read(buffer)
			if err != nil {
				break // End of file or error
			}

			// Get the length of the JSON data
			length := binary.LittleEndian.Uint32(buffer)

			// Read the JSON data
			jsonData := make([]byte, length)
			_, err = file.Read(jsonData)
			if err != nil {
				break
			}

			// Parse the JSON data
			var entry CommandEntry
			if err := json.Unmarshal(jsonData, &entry); err != nil {
				continue
			}

			commands = append(commands, entry)

			// Process in batches of 100
			if len(commands) >= 100 {
				err = t.ProcessCommandBatch(commands)
				if err == nil {
					processedCount += len(commands)
				}
				commands = commands[:0]
			}
		}

		// Process any remaining commands
		if len(commands) > 0 {
			err = t.ProcessCommandBatch(commands)
			if err == nil {
				processedCount += len(commands)
			}
		}

		file.Close()
	}

	return processedCount, nil
}

// GetTokenizerStats returns statistics about the tokenizer
func (t *Tokenizer) GetTokenizerStats() map[string]interface{} {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	stats := map[string]interface{}{
		"vocabulary_size":  len(t.Vocabulary),
		"command_count":    len(t.CommandList),
		"special_tokens":   len(t.Config.SpecialTokens),
		"max_token_length": t.Config.MaxTokenLength,
		"config_path":      t.configPath,
		"data_path":        t.dataPath,
	}

	return stats
}
