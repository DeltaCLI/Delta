package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ART2Preprocessor handles input preprocessing for the ART-2 algorithm
type ART2Preprocessor struct {
	vocabulary      map[string]int     // Word to index mapping
	featureWeights  map[string]float64 // Feature importance weights
	commandPatterns []string           // Learned command patterns
	contextFeatures []string           // Context feature names
	vectorSize      int                // Target vector size
	vocabularyPath  string             // Path to vocabulary file
	initialized     bool               // Whether preprocessor is initialized
}

// FeatureVector represents a processed feature vector
type FeatureVector struct {
	Values   []float64          `json:"values"`
	Features map[string]float64 `json:"features"`
	Metadata map[string]string  `json:"metadata"`
}

// NewART2Preprocessor creates a new preprocessor
func NewART2Preprocessor(vectorSize int) (*ART2Preprocessor, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = os.Getenv("HOME")
	}

	configDir := filepath.Join(homeDir, ".config", "delta", "memory", "art2")
	err = os.MkdirAll(configDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create preprocessor directory: %v", err)
	}

	vocabularyPath := filepath.Join(configDir, "vocabulary.json")

	processor := &ART2Preprocessor{
		vocabulary:      make(map[string]int),
		featureWeights:  make(map[string]float64),
		commandPatterns: make([]string, 0),
		contextFeatures: []string{
			"directory_type", "file_count", "git_repo", "time_of_day",
			"command_length", "has_flags", "has_pipes", "command_complexity",
			"previous_command", "error_state", "user_activity", "session_length",
		},
		vectorSize:     vectorSize,
		vocabularyPath: vocabularyPath,
		initialized:    false,
	}

	// Initialize with common command vocabulary
	processor.initializeBasicVocabulary()

	// Load existing vocabulary if available
	processor.loadVocabulary()

	processor.initialized = true
	return processor, nil
}

// initializeBasicVocabulary initializes with common shell commands and patterns
func (p *ART2Preprocessor) initializeBasicVocabulary() {
	// Common shell commands
	commonCommands := []string{
		"ls", "cd", "pwd", "mkdir", "rmdir", "rm", "cp", "mv", "touch", "find",
		"grep", "awk", "sed", "sort", "uniq", "head", "tail", "cat", "less", "more",
		"git", "make", "npm", "yarn", "docker", "kubectl", "ssh", "scp", "rsync",
		"tar", "gzip", "zip", "curl", "wget", "ping", "ps", "top", "htop", "kill",
		"sudo", "chmod", "chown", "ln", "which", "whereis", "man", "info", "help",
		"echo", "printf", "read", "test", "if", "then", "else", "fi", "for", "while",
	}

	// Common flags and options
	commonFlags := []string{
		"-l", "-a", "-h", "-r", "-f", "-v", "-n", "-i", "-o", "-e", "-s", "-t",
		"--help", "--version", "--verbose", "--quiet", "--force", "--recursive",
		"--all", "--list", "--output", "--input", "--config", "--debug",
	}

	// Common file extensions and types
	commonExtensions := []string{
		".txt", ".log", ".conf", ".cfg", ".json", ".xml", ".yaml", ".yml",
		".sh", ".bash", ".zsh", ".py", ".js", ".go", ".c", ".cpp", ".java",
		".md", ".html", ".css", ".sql", ".dockerfile", ".makefile",
	}

	// Common directory patterns
	commonDirs := []string{
		"bin", "etc", "var", "tmp", "home", "root", "usr", "opt", "src",
		"lib", "include", "share", "docs", "config", "scripts", "test",
	}

	// Build vocabulary
	index := 0

	// Add commands
	for _, cmd := range commonCommands {
		p.vocabulary[cmd] = index
		p.featureWeights[cmd] = 1.0 // High weight for commands
		index++
	}

	// Add flags
	for _, flag := range commonFlags {
		p.vocabulary[flag] = index
		p.featureWeights[flag] = 0.8 // Medium weight for flags
		index++
	}

	// Add extensions
	for _, ext := range commonExtensions {
		p.vocabulary[ext] = index
		p.featureWeights[ext] = 0.6 // Lower weight for extensions
		index++
	}

	// Add directories
	for _, dir := range commonDirs {
		p.vocabulary[dir] = index
		p.featureWeights[dir] = 0.7 // Medium weight for directories
		index++
	}

	// Add special tokens
	specialTokens := map[string]float64{
		"<UNKNOWN>":    0.1,
		"<NUMBER>":     0.3,
		"<PATH>":       0.5,
		"<URL>":        0.4,
		"<EMAIL>":      0.4,
		"<PIPE>":       0.8,
		"<REDIRECT>":   0.8,
		"<BACKGROUND>": 0.6,
		"<SUDO>":       0.9,
		"<ERROR>":      0.9,
	}

	for token, weight := range specialTokens {
		p.vocabulary[token] = index
		p.featureWeights[token] = weight
		index++
	}
}

// PreprocessCommand converts a command string into an ART-2 input vector
func (p *ART2Preprocessor) PreprocessCommand(command, context, workingDir string) (*FeatureVector, error) {
	if !p.initialized {
		return nil, fmt.Errorf("preprocessor not initialized")
	}

	// Tokenize command
	tokens := p.tokenizeCommand(command)

	// Extract features
	features := p.extractFeatures(command, context, workingDir, tokens)

	// Create base vector from tokens
	tokenVector := p.createTokenVector(tokens)

	// Create context vector
	contextVector := p.createContextVector(features)

	// Combine vectors
	combinedVector := p.combineVectors(tokenVector, contextVector)

	// Normalize to target size
	finalVector := p.normalizeVectorSize(combinedVector)

	// Create feature vector result
	result := &FeatureVector{
		Values:   finalVector,
		Features: features,
		Metadata: map[string]string{
			"command":     command,
			"context":     context,
			"working_dir": workingDir,
			"timestamp":   time.Now().Format(time.RFC3339),
			"token_count": strconv.Itoa(len(tokens)),
		},
	}

	return result, nil
}

// tokenizeCommand breaks down a command into meaningful tokens
func (p *ART2Preprocessor) tokenizeCommand(command string) []string {
	// Clean and normalize command
	command = strings.TrimSpace(command)
	command = strings.ToLower(command)

	var tokens []string

	// Split by whitespace and special characters
	parts := regexp.MustCompile(`\s+`).Split(command, -1)

	for _, part := range parts {
		if part == "" {
			continue
		}

		// Check for special patterns
		if p.isNumber(part) {
			tokens = append(tokens, "<NUMBER>")
		} else if p.isPath(part) {
			tokens = append(tokens, "<PATH>")
			// Also add the extension if it's a file
			if ext := p.extractExtension(part); ext != "" {
				tokens = append(tokens, ext)
			}
		} else if p.isURL(part) {
			tokens = append(tokens, "<URL>")
		} else if p.isEmail(part) {
			tokens = append(tokens, "<EMAIL>")
		} else if strings.Contains(part, "|") {
			tokens = append(tokens, "<PIPE>")
			// Split around pipes and process parts
			pipeParts := strings.Split(part, "|")
			for _, pipePart := range pipeParts {
				if pipePart != "" {
					tokens = append(tokens, strings.TrimSpace(pipePart))
				}
			}
		} else if strings.Contains(part, ">") || strings.Contains(part, "<") {
			tokens = append(tokens, "<REDIRECT>")
		} else if strings.HasSuffix(part, "&") {
			tokens = append(tokens, "<BACKGROUND>")
			tokens = append(tokens, strings.TrimSuffix(part, "&"))
		} else if strings.HasPrefix(part, "sudo") {
			tokens = append(tokens, "<SUDO>")
		} else {
			// Regular token
			tokens = append(tokens, part)
		}
	}

	return tokens
}

// extractFeatures extracts contextual features from command and environment
func (p *ART2Preprocessor) extractFeatures(command, context, workingDir string, tokens []string) map[string]float64 {
	features := make(map[string]float64)

	// Command-based features
	features["command_length"] = float64(len(command)) / 100.0 // Normalize to 0-1 range
	features["token_count"] = float64(len(tokens)) / 20.0      // Normalize assuming max 20 tokens

	// Check for flags
	hasFlags := 0.0
	for _, token := range tokens {
		if strings.HasPrefix(token, "-") {
			hasFlags = 1.0
			break
		}
	}
	features["has_flags"] = hasFlags

	// Check for pipes
	features["has_pipes"] = 0.0
	if strings.Contains(command, "|") {
		features["has_pipes"] = 1.0
	}

	// Command complexity (heuristic)
	complexity := 0.0
	complexity += float64(len(tokens)) * 0.1
	if strings.Contains(command, "&&") || strings.Contains(command, "||") {
		complexity += 0.3
	}
	if strings.Contains(command, ";") {
		complexity += 0.2
	}
	if strings.Contains(command, "`") || strings.Contains(command, "$") {
		complexity += 0.4
	}
	features["command_complexity"] = math.Min(complexity, 1.0)

	// Directory-based features
	if workingDir != "" {
		features["directory_type"] = p.classifyDirectory(workingDir)

		// Check if it's a git repository
		gitDir := filepath.Join(workingDir, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			features["git_repo"] = 1.0
		} else {
			features["git_repo"] = 0.0
		}

		// Count files in directory (approximate)
		if files, err := os.ReadDir(workingDir); err == nil {
			features["file_count"] = math.Min(float64(len(files))/100.0, 1.0)
		} else {
			features["file_count"] = 0.0
		}
	}

	// Time-based features
	hour := time.Now().Hour()
	features["time_of_day"] = float64(hour) / 24.0

	// Context-based features (if provided)
	if context != "" {
		// Analyze context for error indicators
		if strings.Contains(strings.ToLower(context), "error") ||
			strings.Contains(strings.ToLower(context), "fail") {
			features["error_state"] = 1.0
		} else {
			features["error_state"] = 0.0
		}
	}

	// User activity level (placeholder - could be enhanced with actual tracking)
	features["user_activity"] = 0.5 // Default medium activity

	// Session length (placeholder)
	features["session_length"] = 0.5 // Default medium session

	return features
}

// createTokenVector creates a vector representation from tokens
func (p *ART2Preprocessor) createTokenVector(tokens []string) []float64 {
	// Create a vector that can accommodate our vocabulary
	maxVocabSize := len(p.vocabulary) + 100 // Extra space for new tokens
	vector := make([]float64, maxVocabSize)

	// Count token occurrences
	tokenCounts := make(map[string]int)
	for _, token := range tokens {
		tokenCounts[token]++
	}

	// Convert to vector using vocabulary mapping
	for token, count := range tokenCounts {
		var index int
		var weight float64
		var exists bool

		if index, exists = p.vocabulary[token]; exists {
			weight = p.featureWeights[token]
		} else {
			// Handle unknown tokens
			if unknownIndex, exists := p.vocabulary["<UNKNOWN>"]; exists {
				index = unknownIndex
				weight = p.featureWeights["<UNKNOWN>"]
			} else {
				continue // Skip if no unknown token handler
			}
		}

		// Apply TF-IDF-like weighting
		tf := float64(count) / float64(len(tokens)) // Term frequency
		vector[index] = tf * weight
	}

	return vector
}

// createContextVector creates a vector from contextual features
func (p *ART2Preprocessor) createContextVector(features map[string]float64) []float64 {
	vector := make([]float64, len(p.contextFeatures))

	for i, featureName := range p.contextFeatures {
		if value, exists := features[featureName]; exists {
			vector[i] = value
		} else {
			vector[i] = 0.0
		}
	}

	return vector
}

// combineVectors combines token and context vectors
func (p *ART2Preprocessor) combineVectors(tokenVector, contextVector []float64) []float64 {
	// Weight the combination (more weight to tokens)
	tokenWeight := 0.7
	contextWeight := 0.3

	maxLen := len(tokenVector)
	if len(contextVector) > maxLen {
		maxLen = len(contextVector)
	}

	combined := make([]float64, maxLen+len(contextVector))

	// Add weighted token vector
	for i := 0; i < len(tokenVector); i++ {
		combined[i] = tokenVector[i] * tokenWeight
	}

	// Add weighted context vector
	for i := 0; i < len(contextVector); i++ {
		combined[len(tokenVector)+i] = contextVector[i] * contextWeight
	}

	return combined
}

// normalizeVectorSize normalizes vector to target size
func (p *ART2Preprocessor) normalizeVectorSize(vector []float64) []float64 {
	if len(vector) == p.vectorSize {
		return vector
	}

	result := make([]float64, p.vectorSize)

	if len(vector) > p.vectorSize {
		// Downsample using averaging
		chunkSize := float64(len(vector)) / float64(p.vectorSize)
		for i := 0; i < p.vectorSize; i++ {
			start := int(float64(i) * chunkSize)
			end := int(float64(i+1) * chunkSize)
			if end > len(vector) {
				end = len(vector)
			}

			sum := 0.0
			count := 0
			for j := start; j < end; j++ {
				sum += vector[j]
				count++
			}
			if count > 0 {
				result[i] = sum / float64(count)
			}
		}
	} else {
		// Upsample using interpolation
		for i := 0; i < p.vectorSize; i++ {
			sourceIndex := float64(i) * float64(len(vector)) / float64(p.vectorSize)
			lowerIndex := int(sourceIndex)
			upperIndex := lowerIndex + 1

			if upperIndex >= len(vector) {
				result[i] = vector[len(vector)-1]
			} else {
				// Linear interpolation
				fraction := sourceIndex - float64(lowerIndex)
				result[i] = vector[lowerIndex]*(1-fraction) + vector[upperIndex]*fraction
			}
		}
	}

	// Normalize to unit length
	norm := 0.0
	for _, v := range result {
		norm += v * v
	}
	norm = math.Sqrt(norm)

	if norm > 0 {
		for i := range result {
			result[i] /= norm
		}
	}

	return result
}

// Helper functions for pattern recognition

func (p *ART2Preprocessor) isNumber(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

func (p *ART2Preprocessor) isPath(s string) bool {
	return strings.Contains(s, "/") || strings.Contains(s, "\\") ||
		strings.HasPrefix(s, "~") || strings.HasPrefix(s, ".")
}

func (p *ART2Preprocessor) isURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") ||
		strings.HasPrefix(s, "ftp://") || strings.HasPrefix(s, "ssh://")
}

func (p *ART2Preprocessor) isEmail(s string) bool {
	return strings.Contains(s, "@") && strings.Contains(s, ".")
}

func (p *ART2Preprocessor) extractExtension(path string) string {
	ext := filepath.Ext(path)
	if ext != "" {
		return strings.ToLower(ext)
	}
	return ""
}

func (p *ART2Preprocessor) classifyDirectory(dir string) float64 {
	dirName := strings.ToLower(filepath.Base(dir))

	// Development directories
	devDirs := []string{"src", "source", "dev", "development", "code", "projects"}
	for _, devDir := range devDirs {
		if strings.Contains(dirName, devDir) {
			return 0.9
		}
	}

	// Configuration directories
	configDirs := []string{"config", "conf", "etc", "settings"}
	for _, configDir := range configDirs {
		if strings.Contains(dirName, configDir) {
			return 0.8
		}
	}

	// Documentation directories
	docDirs := []string{"doc", "docs", "documentation", "man", "manual"}
	for _, docDir := range docDirs {
		if strings.Contains(dirName, docDir) {
			return 0.7
		}
	}

	// Test directories
	testDirs := []string{"test", "tests", "testing", "spec", "specs"}
	for _, testDir := range testDirs {
		if strings.Contains(dirName, testDir) {
			return 0.6
		}
	}

	// Temporary directories
	tempDirs := []string{"tmp", "temp", "temporary", "cache"}
	for _, tempDir := range tempDirs {
		if strings.Contains(dirName, tempDir) {
			return 0.3
		}
	}

	// System directories
	sysDirs := []string{"bin", "usr", "opt", "var", "lib"}
	for _, sysDir := range sysDirs {
		if strings.Contains(dirName, sysDir) {
			return 0.2
		}
	}

	return 0.5 // Default for unclassified directories
}

// UpdateVocabulary adds new tokens to the vocabulary
func (p *ART2Preprocessor) UpdateVocabulary(tokens []string) {
	for _, token := range tokens {
		if _, exists := p.vocabulary[token]; !exists {
			p.vocabulary[token] = len(p.vocabulary)
			// Assign default weight based on token characteristics
			if strings.HasPrefix(token, "-") {
				p.featureWeights[token] = 0.8 // Flag
			} else if p.isPath(token) {
				p.featureWeights[token] = 0.6 // Path
			} else {
				p.featureWeights[token] = 0.5 // Unknown command/word
			}
		}
	}

	// Save updated vocabulary
	p.saveVocabulary()
}

// GetVocabularyStats returns statistics about the vocabulary
func (p *ART2Preprocessor) GetVocabularyStats() map[string]interface{} {
	// Sort vocabulary by frequency/weight
	type vocabEntry struct {
		Token  string
		Index  int
		Weight float64
	}

	var entries []vocabEntry
	for token, index := range p.vocabulary {
		weight := p.featureWeights[token]
		entries = append(entries, vocabEntry{token, index, weight})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Weight > entries[j].Weight
	})

	topTokens := make([]string, 0, 10)
	for i := 0; i < len(entries) && i < 10; i++ {
		topTokens = append(topTokens, entries[i].Token)
	}

	return map[string]interface{}{
		"vocabulary_size": len(p.vocabulary),
		"feature_count":   len(p.contextFeatures),
		"vector_size":     p.vectorSize,
		"top_tokens":      topTokens,
		"initialized":     p.initialized,
		"vocabulary_path": p.vocabularyPath,
	}
}

// Save and load vocabulary
func (p *ART2Preprocessor) saveVocabulary() error {
	data := struct {
		Vocabulary     map[string]int     `json:"vocabulary"`
		FeatureWeights map[string]float64 `json:"feature_weights"`
		VectorSize     int                `json:"vector_size"`
	}{
		Vocabulary:     p.vocabulary,
		FeatureWeights: p.featureWeights,
		VectorSize:     p.vectorSize,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(p.vocabularyPath, jsonData, 0644)
}

func (p *ART2Preprocessor) loadVocabulary() error {
	if !fileExists(p.vocabularyPath) {
		return nil // No vocabulary file yet, use defaults
	}

	data, err := os.ReadFile(p.vocabularyPath)
	if err != nil {
		return err
	}

	var vocabData struct {
		Vocabulary     map[string]int     `json:"vocabulary"`
		FeatureWeights map[string]float64 `json:"feature_weights"`
		VectorSize     int                `json:"vector_size"`
	}

	if err := json.Unmarshal(data, &vocabData); err != nil {
		return err
	}

	// Merge with existing vocabulary (keep existing entries)
	for token, index := range vocabData.Vocabulary {
		if _, exists := p.vocabulary[token]; !exists {
			p.vocabulary[token] = index
		}
	}

	for token, weight := range vocabData.FeatureWeights {
		p.featureWeights[token] = weight
	}

	return nil
}

// Global preprocessor instance
var globalART2Preprocessor *ART2Preprocessor

// GetART2Preprocessor returns the global preprocessor instance
func GetART2Preprocessor() *ART2Preprocessor {
	if globalART2Preprocessor == nil {
		var err error
		globalART2Preprocessor, err = NewART2Preprocessor(128) // Default vector size
		if err != nil {
			fmt.Printf("Error initializing ART-2 preprocessor: %v\n", err)
			return nil
		}
	}
	return globalART2Preprocessor
}
