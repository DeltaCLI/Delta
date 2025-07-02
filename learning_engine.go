package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// PatternType defines different types of patterns we can learn
type PatternType string

const (
	PatternTypeCommand   PatternType = "command"
	PatternTypeSequence  PatternType = "sequence"
	PatternTypeDirectory PatternType = "directory"
	PatternTypeTime      PatternType = "time"
	PatternTypeError     PatternType = "error"
)

// LearnedPattern represents a pattern learned from user behavior
type LearnedPattern struct {
	Type        PatternType       `json:"type"`
	Pattern     string            `json:"pattern"`
	Context     map[string]string `json:"context"`
	Frequency   int               `json:"frequency"`
	SuccessRate float64           `json:"success_rate"`
	LastSeen    time.Time         `json:"last_seen"`
	FirstSeen   time.Time         `json:"first_seen"`
	Predictions []string          `json:"predictions"`
	Confidence  float64           `json:"confidence"`
}

// LearnedSequence represents a learned sequence of commands
type LearnedSequence struct {
	Commands    []string          `json:"commands"`
	Directory   string            `json:"directory"`
	TimeOfDay   string            `json:"time_of_day"`
	Frequency   int               `json:"frequency"`
	LastSeen    time.Time         `json:"last_seen"`
	Context     map[string]string `json:"context"`
}

// LearningEngine manages pattern learning from user behavior
type LearningEngine struct {
	patterns         map[string]*LearnedPattern
	sequences        []LearnedSequence
	memoryManager    *MemoryManager
	inferenceManager *InferenceManager
	configPath       string
	dataPath         string
	isEnabled        bool
	mutex            sync.RWMutex
	lastProcessed    time.Time
}

// NewLearningEngine creates a new learning engine
func NewLearningEngine() (*LearningEngine, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}

	// Set up directories
	learningDir := filepath.Join(homeDir, ".config", "delta", "learning")
	if err := os.MkdirAll(learningDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create learning directory: %v", err)
	}

	configPath := filepath.Join(learningDir, "learning_config.json")
	dataPath := filepath.Join(learningDir, "patterns")
	if err := os.MkdirAll(dataPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create patterns directory: %v", err)
	}

	engine := &LearningEngine{
		patterns:      make(map[string]*LearnedPattern),
		sequences:     make([]LearnedSequence, 0),
		configPath:    configPath,
		dataPath:      dataPath,
		isEnabled:     true,
		lastProcessed: time.Now().Add(-24 * time.Hour), // Process last 24 hours initially
	}

	// Load existing patterns
	if err := engine.loadPatterns(); err != nil {
		fmt.Printf("Warning: failed to load patterns: %v\n", err)
	}

	// Get managers
	engine.memoryManager = GetMemoryManager()
	engine.inferenceManager = GetInferenceManager()

	return engine, nil
}

// LearnFromCommand learns patterns from a single command execution
func (le *LearningEngine) LearnFromCommand(entry CommandEntry) {
	if !le.isEnabled {
		return
	}

	le.mutex.Lock()
	defer le.mutex.Unlock()

	// Learn command patterns
	le.learnCommandPattern(entry)

	// Learn directory-specific patterns
	le.learnDirectoryPattern(entry)

	// Learn time-based patterns
	le.learnTimePattern(entry)

	// Learn from successful commands (exit code 0)
	if entry.ExitCode == 0 {
		le.updateSuccessRate(entry.Command, true)
	} else {
		le.updateSuccessRate(entry.Command, false)
	}

	// Save patterns periodically
	if time.Since(le.lastProcessed) > 5*time.Minute {
		le.savePatterns()
	}
}

// learnCommandPattern learns patterns from individual commands
func (le *LearningEngine) learnCommandPattern(entry CommandEntry) {
	// Extract base command (first word)
	parts := strings.Fields(entry.Command)
	if len(parts) == 0 {
		return
	}

	baseCmd := parts[0]
	patternKey := fmt.Sprintf("cmd:%s", baseCmd)

	pattern, exists := le.patterns[patternKey]
	if !exists {
		pattern = &LearnedPattern{
			Type:        PatternTypeCommand,
			Pattern:     baseCmd,
			Context:     make(map[string]string),
			FirstSeen:   entry.Timestamp,
			Predictions: make([]string, 0),
			Confidence:  0.5,
		}
		le.patterns[patternKey] = pattern
	}

	// Update pattern
	pattern.Frequency++
	pattern.LastSeen = entry.Timestamp
	pattern.Context["last_directory"] = entry.Directory

	// Learn common arguments for this command
	if len(parts) > 1 {
		args := strings.Join(parts[1:], " ")
		found := false
		for _, pred := range pattern.Predictions {
			if pred == args {
				found = true
				break
			}
		}
		if !found && len(pattern.Predictions) < 5 {
			pattern.Predictions = append(pattern.Predictions, args)
		}
	}

	// Update confidence based on frequency
	pattern.Confidence = calculateConfidence(pattern.Frequency, pattern.SuccessRate)
}

// learnDirectoryPattern learns directory-specific command patterns
func (le *LearningEngine) learnDirectoryPattern(entry CommandEntry) {
	patternKey := fmt.Sprintf("dir:%s:%s", entry.Directory, extractBaseCommand(entry.Command))

	pattern, exists := le.patterns[patternKey]
	if !exists {
		pattern = &LearnedPattern{
			Type:        PatternTypeDirectory,
			Pattern:     entry.Command,
			Context:     make(map[string]string),
			FirstSeen:   entry.Timestamp,
			Predictions: make([]string, 0),
			Confidence:  0.5,
		}
		pattern.Context["directory"] = entry.Directory
		le.patterns[patternKey] = pattern
	}

	pattern.Frequency++
	pattern.LastSeen = entry.Timestamp

	// Update confidence
	pattern.Confidence = calculateConfidence(pattern.Frequency, pattern.SuccessRate)
}

// learnTimePattern learns time-based command patterns
func (le *LearningEngine) learnTimePattern(entry CommandEntry) {
	// Determine time of day category
	hour := entry.Timestamp.Hour()
	timeCategory := getTimeCategory(hour)

	patternKey := fmt.Sprintf("time:%s:%s", timeCategory, extractBaseCommand(entry.Command))

	pattern, exists := le.patterns[patternKey]
	if !exists {
		pattern = &LearnedPattern{
			Type:        PatternTypeTime,
			Pattern:     entry.Command,
			Context:     make(map[string]string),
			FirstSeen:   entry.Timestamp,
			Predictions: make([]string, 0),
			Confidence:  0.5,
		}
		pattern.Context["time_category"] = timeCategory
		le.patterns[patternKey] = pattern
	}

	pattern.Frequency++
	pattern.LastSeen = entry.Timestamp
	pattern.Context["last_hour"] = fmt.Sprintf("%d", hour)

	// Update confidence
	pattern.Confidence = calculateConfidence(pattern.Frequency, pattern.SuccessRate)
}

// LearnFromSequence learns patterns from command sequences
func (le *LearningEngine) LearnFromSequence(commands []CommandEntry) {
	if !le.isEnabled || len(commands) < 2 {
		return
	}

	le.mutex.Lock()
	defer le.mutex.Unlock()

	// Extract command strings
	cmdStrings := make([]string, len(commands))
	for i, cmd := range commands {
		cmdStrings[i] = cmd.Command
	}

	// Check if this sequence already exists
	sequenceKey := strings.Join(cmdStrings, " -> ")
	found := false

	for i, seq := range le.sequences {
		if strings.Join(seq.Commands, " -> ") == sequenceKey {
			le.sequences[i].Frequency++
			le.sequences[i].LastSeen = time.Now()
			found = true
			break
		}
	}

	if !found && len(le.sequences) < 100 { // Limit stored sequences
		newSeq := LearnedSequence{
			Commands:  cmdStrings,
			Directory: commands[0].Directory,
			TimeOfDay: getTimeCategory(commands[0].Timestamp.Hour()),
			Frequency: 1,
			LastSeen:  time.Now(),
			Context:   make(map[string]string),
		}
		le.sequences = append(le.sequences, newSeq)
	}
}

// GetPredictions returns predictions for the given context
func (le *LearningEngine) GetPredictions(currentCmd string, directory string, limit int) []string {
	le.mutex.RLock()
	defer le.mutex.RUnlock()

	predictions := make([]string, 0)
	scores := make(map[string]float64)

	// Get current hour for time-based predictions
	hour := time.Now().Hour()
	timeCategory := getTimeCategory(hour)

	// Check command patterns
	baseCmd := extractBaseCommand(currentCmd)
	if pattern, exists := le.patterns[fmt.Sprintf("cmd:%s", baseCmd)]; exists {
		for _, pred := range pattern.Predictions {
			score := pattern.Confidence * pattern.SuccessRate
			scores[fmt.Sprintf("%s %s", baseCmd, pred)] = score
		}
	}

	// Check directory-specific patterns
	if pattern, exists := le.patterns[fmt.Sprintf("dir:%s:%s", directory, baseCmd)]; exists {
		score := pattern.Confidence * pattern.SuccessRate * 1.2 // Boost directory-specific
		scores[pattern.Pattern] = score
	}

	// Check time-based patterns
	if pattern, exists := le.patterns[fmt.Sprintf("time:%s:%s", timeCategory, baseCmd)]; exists {
		score := pattern.Confidence * pattern.SuccessRate * 1.1 // Slight boost for time
		scores[pattern.Pattern] = score
	}

	// Sort by score and return top predictions
	for cmd, score := range scores {
		if score > 0.3 { // Minimum confidence threshold
			predictions = append(predictions, cmd)
		}
	}

	// Limit results
	if len(predictions) > limit && limit > 0 {
		predictions = predictions[:limit]
	}

	return predictions
}

// ProcessDailyData processes accumulated data for learning
func (le *LearningEngine) ProcessDailyData() error {
	if !le.isEnabled {
		return nil
	}

	le.mutex.Lock()
	defer le.mutex.Unlock()

	// Get memory manager
	if le.memoryManager == nil {
		return fmt.Errorf("memory manager not available")
	}

	// Process commands from the last 24 hours
	endTime := time.Now()
	startTime := le.lastProcessed

	// Get commands from memory shards
	commands, err := le.memoryManager.GetCommandsInRange(startTime, endTime)
	if err != nil {
		return fmt.Errorf("failed to get commands: %v", err)
	}

	// Learn from individual commands
	for _, cmd := range commands {
		le.learnCommandPattern(cmd)
		le.learnDirectoryPattern(cmd)
		le.learnTimePattern(cmd)
	}

	// Learn from sequences
	le.detectAndLearnSequences(commands)

	// Update last processed time
	le.lastProcessed = endTime

	// Save patterns
	return le.savePatterns()
}

// detectAndLearnSequences detects and learns from command sequences
func (le *LearningEngine) detectAndLearnSequences(commands []CommandEntry) {
	if len(commands) < 2 {
		return
	}

	// Look for sequences within 5-minute windows
	windowSize := 5 * time.Minute

	for i := 0; i < len(commands)-1; i++ {
		sequence := []CommandEntry{commands[i]}

		// Build sequence
		for j := i + 1; j < len(commands) && j < i+5; j++ {
			if commands[j].Timestamp.Sub(commands[j-1].Timestamp) <= windowSize {
				sequence = append(sequence, commands[j])
			} else {
				break
			}
		}

		// Learn from sequences of 2-5 commands
		if len(sequence) >= 2 && len(sequence) <= 5 {
			le.LearnFromSequence(sequence)
		}
	}
}

// updateSuccessRate updates the success rate for a command
func (le *LearningEngine) updateSuccessRate(command string, success bool) {
	baseCmd := extractBaseCommand(command)
	patternKey := fmt.Sprintf("cmd:%s", baseCmd)

	if pattern, exists := le.patterns[patternKey]; exists {
		total := float64(pattern.Frequency)
		currentSuccesses := pattern.SuccessRate * total

		if success {
			currentSuccesses++
		}

		pattern.SuccessRate = currentSuccesses / (total + 1)
	}
}

// Helper functions

func extractBaseCommand(command string) string {
	parts := strings.Fields(command)
	if len(parts) > 0 {
		return parts[0]
	}
	return command
}

func getTimeCategory(hour int) string {
	switch {
	case hour >= 6 && hour < 12:
		return "morning"
	case hour >= 12 && hour < 17:
		return "afternoon"
	case hour >= 17 && hour < 22:
		return "evening"
	default:
		return "night"
	}
}

func calculateConfidence(frequency int, successRate float64) float64 {
	// Base confidence on frequency and success rate
	freqScore := 1.0 - 1.0/float64(frequency+1)
	return freqScore * successRate
}

// Persistence methods

func (le *LearningEngine) savePatterns() error {
	// Save patterns
	patternsFile := filepath.Join(le.dataPath, "learned_patterns.json")
	patternsData, err := json.MarshalIndent(le.patterns, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal patterns: %v", err)
	}

	if err := os.WriteFile(patternsFile, patternsData, 0644); err != nil {
		return fmt.Errorf("failed to save patterns: %v", err)
	}

	// Save sequences
	sequencesFile := filepath.Join(le.dataPath, "command_sequences.json")
	sequencesData, err := json.MarshalIndent(le.sequences, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal sequences: %v", err)
	}

	if err := os.WriteFile(sequencesFile, sequencesData, 0644); err != nil {
		return fmt.Errorf("failed to save sequences: %v", err)
	}

	return nil
}

func (le *LearningEngine) loadPatterns() error {
	// Load patterns
	patternsFile := filepath.Join(le.dataPath, "learned_patterns.json")
	if data, err := os.ReadFile(patternsFile); err == nil {
		if err := json.Unmarshal(data, &le.patterns); err != nil {
			return fmt.Errorf("failed to unmarshal patterns: %v", err)
		}
	}

	// Load sequences
	sequencesFile := filepath.Join(le.dataPath, "command_sequences.json")
	if data, err := os.ReadFile(sequencesFile); err == nil {
		if err := json.Unmarshal(data, &le.sequences); err != nil {
			return fmt.Errorf("failed to unmarshal sequences: %v", err)
		}
	}

	return nil
}

// Global learning engine instance
var globalLearningEngine *LearningEngine
var learningEngineOnce sync.Once

// GetLearningEngine returns the global learning engine instance
func GetLearningEngine() *LearningEngine {
	learningEngineOnce.Do(func() {
		var err error
		globalLearningEngine, err = NewLearningEngine()
		if err != nil {
			fmt.Printf("Warning: failed to initialize learning engine: %v\n", err)
		}
	})
	return globalLearningEngine
}