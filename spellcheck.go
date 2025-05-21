package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// SpellCheckConfig holds configuration for spell checking
type SpellCheckConfig struct {
	Enabled             bool    `json:"enabled"`               // Whether spell checking is enabled
	SuggestionThreshold float64 `json:"suggestion_threshold"`  // Threshold for suggesting corrections (0.0-1.0)
	MaxSuggestions      int     `json:"max_suggestions"`       // Maximum number of suggestions to provide
	AutoCorrect         bool    `json:"auto_correct"`          // Whether to auto-correct high confidence matches
	AutoCorrectThreshold float64 `json:"auto_correct_threshold"` // Threshold for auto-correction (0.0-1.0)
	CaseSensitive       bool    `json:"case_sensitive"`        // Whether matching is case sensitive
	CustomDictionary    []string `json:"custom_dictionary"`    // Additional valid commands
}

// CommandSimilarity represents a possible correction with similarity score
type CommandSimilarity struct {
	Command   string  // The command name
	Score     float64 // Similarity score (0.0-1.0)
	Full      string  // Full command representation (with colon prefix if applicable)
	IsInternal bool   // Whether it's an internal command
}

// SpellChecker provides spell checking and correction for commands
type SpellChecker struct {
	config          SpellCheckConfig
	configPath      string
	mutex           sync.RWMutex
	internalCommands map[string]bool // Map of valid internal commands
	popularCommands map[string]int   // Map of commands to usage count
	isInitialized   bool
}

// NewSpellChecker creates a new spell checker
func NewSpellChecker() (*SpellChecker, error) {
	// Set up config directory in user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = os.Getenv("HOME")
	}

	// Use ~/.config/delta directory for config file
	configDir := filepath.Join(homeDir, ".config", "delta")
	err = os.MkdirAll(configDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create config directory: %v", err)
	}

	configPath := filepath.Join(configDir, "spellcheck_config.json")

	// Create default SpellChecker instance
	sc := &SpellChecker{
		config: SpellCheckConfig{
			Enabled:             true,
			SuggestionThreshold: 0.7,  // 70% similarity for suggestions
			MaxSuggestions:      3,
			AutoCorrect:         false,
			AutoCorrectThreshold: 0.9, // 90% similarity for auto-correction
			CaseSensitive:       false,
			CustomDictionary:    []string{},
		},
		configPath:       configPath,
		internalCommands: make(map[string]bool),
		popularCommands:  make(map[string]int),
		isInitialized:    false,
	}

	return sc, nil
}

// Initialize initializes the spell checker system
func (sc *SpellChecker) Initialize() error {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	// Try to load existing configuration
	err := sc.loadConfig()
	if err != nil {
		// If loading fails, save the default configuration
		err = sc.saveConfig()
		if err != nil {
			return fmt.Errorf("failed to save default configuration: %v", err)
		}
	}

	// Initialize command dictionary
	sc.initializeCommands()

	sc.isInitialized = true
	return nil
}

// loadConfig loads the configuration from disk
func (sc *SpellChecker) loadConfig() error {
	// Check if config file exists
	_, err := os.Stat(sc.configPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("configuration file does not exist")
	}

	// Read the config file
	data, err := os.ReadFile(sc.configPath)
	if err != nil {
		return err
	}

	// Parse the JSON data
	var config SpellCheckConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	sc.config = config
	return nil
}

// saveConfig saves the configuration to disk
func (sc *SpellChecker) saveConfig() error {
	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(sc.config, "", "  ")
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(sc.configPath)
	if err = os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(sc.configPath, data, 0644)
}

// UpdateConfig updates the spell checker configuration
func (sc *SpellChecker) UpdateConfig(config SpellCheckConfig) error {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	sc.config = config
	return sc.saveConfig()
}

// initializeCommands populates the internal commands dictionary
func (sc *SpellChecker) initializeCommands() {
	// Add known internal commands to the dictionary
	// We can't directly access the internalCmds map from cli.go,
	// so we need to maintain our own list here
	knownCommands := []string{
		"ai", "help", "jump", "j", "memory", "mem", "tokenizer", "tok",
		"inference", "inf", "feedback", "vector", "embedding", "speculative",
		"specd", "knowledge", "know", "agent", "config", "init",
			"pattern", "pat",
		"spellcheck", "spell",
	}

	for _, cmd := range knownCommands {
		sc.internalCommands[cmd] = true
	}
	
	// Add command aliases
	sc.internalCommands["inference"] = true
	sc.internalCommands["inf"] = true
	sc.internalCommands["memory"] = true
	sc.internalCommands["mem"] = true
	sc.internalCommands["tokenizer"] = true
	sc.internalCommands["tok"] = true
	sc.internalCommands["jump"] = true
	sc.internalCommands["j"] = true
	sc.internalCommands["config"] = true
	sc.internalCommands["help"] = true
	sc.internalCommands["agent"] = true
	sc.internalCommands["vector"] = true
	sc.internalCommands["embedding"] = true
	sc.internalCommands["knowledge"] = true
	sc.internalCommands["know"] = true
	sc.internalCommands["speculative"] = true
	sc.internalCommands["specd"] = true
	sc.internalCommands["feedback"] = true
	sc.internalCommands["init"] = true
	sc.internalCommands["ai"] = true
		sc.internalCommands["pattern"] = true
		sc.internalCommands["pat"] = true
	
	// Add custom dictionary entries
	for _, cmd := range sc.config.CustomDictionary {
		sc.internalCommands[cmd] = true
	}
}

// CheckCommand checks a command for spelling errors and returns suggestions
func (sc *SpellChecker) CheckCommand(input string) []CommandSimilarity {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()
	
	if !sc.isInitialized || !sc.config.Enabled {
		return nil
	}
	
	// Handle internal commands (those that start with :)
	if strings.HasPrefix(input, ":") {
		cmdWithoutColon := strings.TrimPrefix(input, ":")
		firstWord := strings.Fields(cmdWithoutColon)[0]
		
		// If it's a valid command, no need for spell checking
		if sc.internalCommands[firstWord] {
			return nil
		}
		
		// Generate suggestions for internal commands
		return sc.findSimilarInternalCommands(firstWord)
	}
	
	// For now, we only check internal commands
	return nil
}

// GetCorrectionText returns a formatted string with correction suggestions
func (sc *SpellChecker) GetCorrectionText(cmd string, suggestions []CommandSimilarity) string {
	if len(suggestions) == 0 {
		return ""
	}
	
	// Format based on number of suggestions
	if len(suggestions) == 1 {
		return fmt.Sprintf("Did you mean ':%s'?", suggestions[0].Command)
	}
	
	// Multiple suggestions
	var sb strings.Builder
	sb.WriteString("Did you mean one of these: ")
	
	for i, suggestion := range suggestions {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("':%s'", suggestion.Command))
	}
	sb.WriteString("?")
	
	return sb.String()
}

// ShouldAutoCorrect determines if we should auto-correct a command
func (sc *SpellChecker) ShouldAutoCorrect(suggestions []CommandSimilarity) bool {
	if !sc.config.AutoCorrect || len(suggestions) == 0 {
		return false
	}
	
	// Only auto-correct if the top suggestion has a high enough confidence
	return suggestions[0].Score >= sc.config.AutoCorrectThreshold
}

// findSimilarInternalCommands finds internal commands similar to the input
func (sc *SpellChecker) findSimilarInternalCommands(input string) []CommandSimilarity {
	var similarities []CommandSimilarity
	
	// Get lowercase version if case insensitive
	searchInput := input
	if !sc.config.CaseSensitive {
		searchInput = strings.ToLower(input)
	}
	
	// Compare with all known internal commands
	for cmd := range sc.internalCommands {
		cmdCompare := cmd
		if !sc.config.CaseSensitive {
			cmdCompare = strings.ToLower(cmd)
		}
		
		// Skip exact matches (though this shouldn't happen)
		if cmdCompare == searchInput {
			continue
		}
		
		// Calculate similarity score
		score := calculateSimilarity(searchInput, cmdCompare)
		
		// Only keep suggestions above threshold
		if score >= sc.config.SuggestionThreshold {
			similarities = append(similarities, CommandSimilarity{
				Command:   cmd,
				Score:     score,
				Full:      ":" + cmd,
				IsInternal: true,
			})
		}
	}
	
	// Sort by score (highest first)
	sort.Slice(similarities, func(i, j int) bool {
		return similarities[i].Score > similarities[j].Score
	})
	
	// Limit to max suggestions
	if len(similarities) > sc.config.MaxSuggestions {
		similarities = similarities[:sc.config.MaxSuggestions]
	}
	
	return similarities
}

// calculateSimilarity calculates the similarity between two strings
// Returns a value between 0.0 (completely different) and 1.0 (identical)
func calculateSimilarity(s1, s2 string) float64 {
	// For very short strings, use special handling
	if len(s1) <= 3 || len(s2) <= 3 {
		if s1 == s2 {
			return 1.0
		}
		
		// For very short strings, if they share the first letter, give them a boost
		if len(s1) > 0 && len(s2) > 0 && s1[0] == s2[0] {
			// Calculate normalized edit distance for the rest
			editDistance := levenshteinDistance(s1[1:], s2[1:])
			maxLen := float64(max(len(s1), len(s2)) - 1)
			if maxLen == 0 {
				return 0.8 // They're both single-character strings with the same first letter
			}
			return 0.5 + (0.5 * (1.0 - float64(editDistance)/maxLen))
		}
	}
	
	// For longer strings, use Levenshtein distance
	editDistance := levenshteinDistance(s1, s2)
	maxLen := float64(max(len(s1), len(s2)))
	if maxLen == 0 {
		return 1.0 // Both strings are empty
	}
	
	// Normalize and invert so higher is more similar
	return 1.0 - float64(editDistance)/maxLen
}

// levenshteinDistance calculates the edit distance between two strings
func levenshteinDistance(s1, s2 string) int {
	// Convert to runes to handle UTF-8 correctly
	r1 := []rune(s1)
	r2 := []rune(s2)
	
	// Create a matrix of size (len(r1)+1) x (len(r2)+1)
	rows, cols := len(r1)+1, len(r2)+1
	matrix := make([][]int, rows)
	for i := 0; i < rows; i++ {
		matrix[i] = make([]int, cols)
		matrix[i][0] = i // Initialize first column
	}
	
	// Initialize first row
	for j := 1; j < cols; j++ {
		matrix[0][j] = j
	}
	
	// Fill the matrix
	for i := 1; i < rows; i++ {
		for j := 1; j < cols; j++ {
			cost := 1
			if r1[i-1] == r2[j-1] {
				cost = 0
			}
			
			// The min of three operations: deletion, insertion, substitution
			matrix[i][j] = min(
				matrix[i-1][j]+1,           // Deletion
				matrix[i][j-1]+1,           // Insertion
				matrix[i-1][j-1]+cost,      // Substitution
			)
			
			// Handle transposition (optional, for Damerau-Levenshtein distance)
			if i > 1 && j > 1 && r1[i-1] == r2[j-2] && r1[i-2] == r2[j-1] {
				matrix[i][j] = min(matrix[i][j], matrix[i-2][j-2]+cost)
			}
		}
	}
	
	// The distance is the bottom-right cell
	return matrix[rows-1][cols-1]
}

// min returns the minimum of multiple integers
func min(values ...int) int {
	result := values[0]
	for _, v := range values[1:] {
		if v < result {
			result = v
		}
	}
	return result
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// TrackCommandUsage increases the usage count for a command
func (sc *SpellChecker) TrackCommandUsage(cmd string) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	
	// Strip colon prefix if present
	if strings.HasPrefix(cmd, ":") {
		cmd = strings.TrimPrefix(cmd, ":")
	}
	
	// Extract the base command (without arguments)
	baseCmd := strings.Fields(cmd)[0]
	
	// Increment usage count
	sc.popularCommands[baseCmd]++
}

// RecordCorrection records when a user accepts a correction
func (sc *SpellChecker) RecordCorrection(original, correction string) {
	// For future machine learning/improvement of the suggestion algorithm
	// This is a placeholder for now
}

// IsEnabled returns whether spell checking is enabled
func (sc *SpellChecker) IsEnabled() bool {
	sc.mutex.RLock()
	defer sc.mutex.RUnlock()
	return sc.isInitialized && sc.config.Enabled
}

// EnableSpellChecker enables spell checking
func (sc *SpellChecker) EnableSpellChecker() {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	
	sc.config.Enabled = true
	sc.saveConfig()
}

// DisableSpellChecker disables spell checking
func (sc *SpellChecker) DisableSpellChecker() {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	
	sc.config.Enabled = false
	sc.saveConfig()
}

// Global SpellChecker instance
var globalSpellChecker *SpellChecker

// GetSpellChecker returns the global SpellChecker instance
func GetSpellChecker() *SpellChecker {
	if globalSpellChecker == nil {
		var err error
		globalSpellChecker, err = NewSpellChecker()
		if err != nil {
			fmt.Printf("Error initializing spell checker: %v\n", err)
			return nil
		}
		
		// Initialize the spell checker
		globalSpellChecker.Initialize()
	}
	return globalSpellChecker
}