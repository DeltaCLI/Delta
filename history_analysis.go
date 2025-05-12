package main

import (
	"container/heap"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// CommandContext represents the context in which a command was executed
type CommandContext struct {
	Directory   string            `json:"directory"`   // Working directory
	Environment map[string]string `json:"environment"` // Key environment variables
	Timestamp   time.Time         `json:"timestamp"`   // When the command was executed
	ExitCode    int               `json:"exit_code"`   // Exit status (0 = success)
	Duration    time.Duration     `json:"duration"`    // How long the command took to execute
}

// EnhancedHistoryEntry represents a single command with additional context and metadata
type EnhancedHistoryEntry struct {
	Command     string         `json:"command"`     // The actual command text
	Context     CommandContext `json:"context"`     // Command execution context
	Category    string         `json:"category"`    // Command category (file, network, process, etc.)
	Tags        []string       `json:"tags"`        // Additional tags for the command
	Frequency   int            `json:"frequency"`   // How many times this exact command has been used
	LastUsed    time.Time      `json:"last_used"`   // When this command was last used
	IsImportant bool           `json:"is_important"` // Whether this is a significant command
}

// CommandSequence represents a sequence of commands that are frequently used together
type CommandSequence struct {
	Commands    []string `json:"commands"`    // Sequence of commands
	Frequency   int      `json:"frequency"`   // How many times this sequence has been observed
	LastUsed    time.Time `json:"last_used"`  // When was this sequence last used
	MeaningfulName string `json:"name"`       // User-defined or auto-generated name for the sequence
}

// CommandSuggestion represents a suggested command based on context
type CommandSuggestion struct {
	Command     string  `json:"command"`     // The suggested command
	Confidence  float64 `json:"confidence"`  // Confidence score for the suggestion (0.0-1.0)
	Reason      string  `json:"reason"`      // Why this command is being suggested
	IsSequence  bool    `json:"is_sequence"` // Whether this is part of a sequence
	SequenceName string `json:"sequence_name"` // Name of the sequence if applicable
}

// HistoryAnalysisConfig holds configuration for the history analysis system
type HistoryAnalysisConfig struct {
	Enabled                 bool     `json:"enabled"`                   // Whether history analysis is enabled
	MaxHistorySize          int      `json:"max_history_size"`          // Maximum number of commands to store
	MinConfidenceThreshold  float64  `json:"min_confidence_threshold"`  // Minimum confidence for suggestions
	MaxSuggestions          int      `json:"max_suggestions"`           // Maximum number of suggestions to show
	AutoSuggest             bool     `json:"auto_suggest"`              // Whether to automatically show suggestions
	PrivacyFilter           []string `json:"privacy_filter"`            // Patterns to filter from history
	TrackCommandSequences   bool     `json:"track_command_sequences"`   // Whether to track command sequences
	SequenceMaxLength       int      `json:"sequence_max_length"`       // Maximum length of a command sequence
	EnableNLSearch          bool     `json:"enable_nl_search"`          // Enable natural language search
	ContextWeight           float64  `json:"context_weight"`            // Weight of directory context in suggestions
	TimeWeight              float64  `json:"time_weight"`               // Weight of time patterns in suggestions
	FrequencyWeight         float64  `json:"frequency_weight"`          // Weight of command frequency in suggestions
	RecencyWeight           float64  `json:"recency_weight"`            // Weight of command recency in suggestions
	EnableCommandCategories bool     `json:"enable_command_categories"` // Categorize commands by type
}

// HistoryAnalyzer analyzes command history and provides intelligent suggestions
type HistoryAnalyzer struct {
	config               HistoryAnalysisConfig
	configPath           string
	historyPath          string
	history              []EnhancedHistoryEntry
	commandFrequency     map[string]int
	directoryCommands    map[string]map[string]int // Directory -> Command -> Count
	timePatterns         map[int]map[string]int    // Hour -> Command -> Count
	commandCategories    map[string]string         // Command prefix -> Category
	commandSequences     []CommandSequence
	recentCommands       []string // Last N commands for sequence detection
	commandRegexes       map[string]*regexp.Regexp
	lastModified         time.Time
	historyLock          sync.RWMutex
	isInitialized        bool
}

// NewHistoryAnalyzer creates a new history analyzer
func NewHistoryAnalyzer() (*HistoryAnalyzer, error) {
	// Set up config directory in user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = os.Getenv("HOME")
	}

	// Use ~/.config/delta/history directory
	configDir := filepath.Join(homeDir, ".config", "delta", "history")
	err = os.MkdirAll(configDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create history directory: %v", err)
	}

	configPath := filepath.Join(configDir, "history_analysis_config.json")
	historyPath := filepath.Join(configDir, "enhanced_history.json")

	// Create default history analyzer
	ha := &HistoryAnalyzer{
		config: HistoryAnalysisConfig{
			Enabled:                 true,
			MaxHistorySize:          10000,
			MinConfidenceThreshold:  0.3,
			MaxSuggestions:          5,
			AutoSuggest:             true,
			PrivacyFilter:           []string{"password", "token", "key", "secret", "credential"},
			TrackCommandSequences:   true,
			SequenceMaxLength:       5,
			EnableNLSearch:          true,
			ContextWeight:           0.4,
			TimeWeight:              0.2,
			FrequencyWeight:         0.3,
			RecencyWeight:           0.1,
			EnableCommandCategories: true,
		},
		configPath:        configPath,
		historyPath:       historyPath,
		history:           []EnhancedHistoryEntry{},
		commandFrequency:  make(map[string]int),
		directoryCommands: make(map[string]map[string]int),
		timePatterns:      make(map[int]map[string]int),
		commandCategories: make(map[string]string),
		commandSequences:  []CommandSequence{},
		recentCommands:    []string{},
		commandRegexes:    make(map[string]*regexp.Regexp),
		historyLock:       sync.RWMutex{},
		isInitialized:     false,
	}

	// Initialize command categories
	ha.initializeCommandCategories()

	// Compile privacy filter regexes
	ha.compileRegexes()

	return ha, nil
}

// Initialize initializes the history analyzer
func (ha *HistoryAnalyzer) Initialize() error {
	ha.historyLock.Lock()
	defer ha.historyLock.Unlock()

	// Try to load existing configuration
	err := ha.loadConfig()
	if err != nil {
		// If loading fails, save the default configuration
		err = ha.saveConfig()
		if err != nil {
			return fmt.Errorf("failed to save default configuration: %v", err)
		}
	}

	// Try to load existing history
	err = ha.loadHistory()
	if err != nil {
		fmt.Printf("No existing history found, starting fresh: %v\n", err)
		// This is fine - we'll start with an empty history
	}

	// Rebuild indexes
	ha.rebuildIndexes()

	ha.isInitialized = true
	return nil
}

// loadConfig loads the configuration from disk
func (ha *HistoryAnalyzer) loadConfig() error {
	// Check if config file exists
	_, err := os.Stat(ha.configPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("configuration file does not exist")
	}

	// Read the config file
	data, err := os.ReadFile(ha.configPath)
	if err != nil {
		return err
	}

	// Parse the JSON data
	var config HistoryAnalysisConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}

	ha.config = config
	return nil
}

// saveConfig saves the configuration to disk
func (ha *HistoryAnalyzer) saveConfig() error {
	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(ha.config, "", "  ")
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(ha.configPath)
	if err = os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(ha.configPath, data, 0644)
}

// loadHistory loads the command history from disk
func (ha *HistoryAnalyzer) loadHistory() error {
	// Check if history file exists
	_, err := os.Stat(ha.historyPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("history file does not exist")
	}

	// Read the history file
	data, err := os.ReadFile(ha.historyPath)
	if err != nil {
		return err
	}

	// Parse the JSON data
	var history []EnhancedHistoryEntry
	if err := json.Unmarshal(data, &history); err != nil {
		return err
	}

	ha.history = history
	ha.lastModified = time.Now()
	return nil
}

// saveHistory saves the command history to disk
func (ha *HistoryAnalyzer) saveHistory() error {
	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(ha.history, "", "  ")
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(ha.historyPath)
	if err = os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(ha.historyPath, data, 0644)
}

// UpdateConfig updates the history analyzer configuration
func (ha *HistoryAnalyzer) UpdateConfig(config HistoryAnalysisConfig) error {
	ha.historyLock.Lock()
	defer ha.historyLock.Unlock()

	ha.config = config
	ha.compileRegexes() // Recompile regexes if privacy filters changed
	return ha.saveConfig()
}

// AddCommand adds a command to the history with enhanced context
func (ha *HistoryAnalyzer) AddCommand(command string, ctx CommandContext) error {
	if !ha.isInitialized || !ha.config.Enabled {
		return nil
	}

	// Skip commands that match privacy filters
	if ha.shouldFilter(command) {
		return nil
	}

	ha.historyLock.Lock()
	defer ha.historyLock.Unlock()

	// Determine command category
	category := ha.categorizeCommand(command)

	// Check if this command already exists
	existingIdx := -1
	for i, entry := range ha.history {
		if entry.Command == command {
			existingIdx = i
			break
		}
	}

	if existingIdx >= 0 {
		// Update existing entry
		ha.history[existingIdx].Frequency++
		ha.history[existingIdx].LastUsed = ctx.Timestamp
		ha.history[existingIdx].Context = ctx // Update context
	} else {
		// Create new entry
		entry := EnhancedHistoryEntry{
			Command:     command,
			Context:     ctx,
			Category:    category,
			Tags:        []string{},
			Frequency:   1,
			LastUsed:    ctx.Timestamp,
			IsImportant: false,
		}

		// Add tags based on command content
		entry.Tags = ha.generateTags(command)

		// Add to history
		ha.history = append(ha.history, entry)

		// Trim history if it exceeds maximum size
		if len(ha.history) > ha.config.MaxHistorySize {
			// Remove oldest entries
			ha.history = ha.history[len(ha.history)-ha.config.MaxHistorySize:]
		}
	}

	// Update frequency map
	ha.commandFrequency[command]++

	// Update directory commands map
	dir := ctx.Directory
	if _, ok := ha.directoryCommands[dir]; !ok {
		ha.directoryCommands[dir] = make(map[string]int)
	}
	ha.directoryCommands[dir][command]++

	// Update time patterns map
	hour := ctx.Timestamp.Hour()
	if _, ok := ha.timePatterns[hour]; !ok {
		ha.timePatterns[hour] = make(map[string]int)
	}
	ha.timePatterns[hour][command]++

	// Update recent commands for sequence detection
	if ha.config.TrackCommandSequences {
		ha.recentCommands = append(ha.recentCommands, command)
		if len(ha.recentCommands) > ha.config.SequenceMaxLength {
			ha.recentCommands = ha.recentCommands[1:]
		}
		ha.detectSequences()
	}

	// Save history periodically
	if time.Since(ha.lastModified) > 5*time.Minute {
		ha.saveHistory()
		ha.lastModified = time.Now()
	}

	return nil
}

// shouldFilter checks if a command should be filtered for privacy reasons
func (ha *HistoryAnalyzer) shouldFilter(command string) bool {
	for _, regex := range ha.commandRegexes {
		if regex.MatchString(command) {
			return true
		}
	}
	return false
}

// compileRegexes compiles the privacy filter regexes
func (ha *HistoryAnalyzer) compileRegexes() {
	ha.commandRegexes = make(map[string]*regexp.Regexp)
	for _, p := range ha.config.PrivacyFilter {
		regex, err := regexp.Compile("(?i)" + p)
		if err == nil {
			ha.commandRegexes[p] = regex
		}
	}
}

// categorizeCommand determines the category of a command
func (ha *HistoryAnalyzer) categorizeCommand(command string) string {
	if !ha.config.EnableCommandCategories {
		return "unknown"
	}

	// Extract the base command (first word)
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "unknown"
	}

	baseCmd := parts[0]

	// Check internal commands that start with :
	if strings.HasPrefix(baseCmd, ":") {
		return "internal"
	}

	// Check for known command categories
	for prefix, category := range ha.commandCategories {
		if strings.HasPrefix(baseCmd, prefix) {
			return category
		}
	}

	return "unknown"
}

// initializeCommandCategories sets up the initial command categories
func (ha *HistoryAnalyzer) initializeCommandCategories() {
	// File operations
	fileOps := []string{"ls", "cp", "mv", "rm", "mkdir", "rmdir", "touch", "cat", "more", "less",
		"head", "tail", "find", "grep", "sed", "awk", "chmod", "chown", "chgrp", "ln"}
	for _, cmd := range fileOps {
		ha.commandCategories[cmd] = "file"
	}

	// Navigation
	navOps := []string{"cd", "pwd", "pushd", "popd"}
	for _, cmd := range navOps {
		ha.commandCategories[cmd] = "navigation"
	}

	// Network operations
	netOps := []string{"ssh", "scp", "ping", "telnet", "nc", "netstat", "ifconfig", "ip",
		"curl", "wget", "dig", "nslookup", "traceroute", "whois", "host"}
	for _, cmd := range netOps {
		ha.commandCategories[cmd] = "network"
	}

	// Process management
	procOps := []string{"ps", "top", "htop", "kill", "pkill", "killall", "bg", "fg", "jobs",
		"nohup", "nice", "renice", "time", "watch"}
	for _, cmd := range procOps {
		ha.commandCategories[cmd] = "process"
	}

	// Package management
	pkgOps := []string{"apt", "apt-get", "dpkg", "yum", "rpm", "dnf", "pacman", "brew", "pip",
		"npm", "gem", "cargo", "go"}
	for _, cmd := range pkgOps {
		ha.commandCategories[cmd] = "package"
	}

	// Version control
	vcsOps := []string{"git", "svn", "hg", "bzr"}
	for _, cmd := range vcsOps {
		ha.commandCategories[cmd] = "vcs"
	}

	// Text editors
	editorOps := []string{"vim", "vi", "nano", "emacs", "ed", "pico", "code", "atom", "subl"}
	for _, cmd := range editorOps {
		ha.commandCategories[cmd] = "editor"
	}

	// Build tools
	buildOps := []string{"make", "cmake", "gcc", "g++", "javac", "mvn", "gradle", "ant", "sbt"}
	for _, cmd := range buildOps {
		ha.commandCategories[cmd] = "build"
	}

	// Docker/containers
	containerOps := []string{"docker", "podman", "kubectl", "k8s", "kubelet", "helm"}
	for _, cmd := range containerOps {
		ha.commandCategories[cmd] = "container"
	}
}

// generateTags creates tags for a command based on its content
func (ha *HistoryAnalyzer) generateTags(command string) []string {
	tags := []string{}

	// Add category as a tag
	category := ha.categorizeCommand(command)
	tags = append(tags, category)

	// Parse command for interesting patterns
	parts := strings.Fields(command)
	if len(parts) > 0 {
		baseCmd := parts[0]
		tags = append(tags, baseCmd)

		// Look for flags
		for _, part := range parts[1:] {
			if strings.HasPrefix(part, "-") {
				tags = append(tags, "flag:"+part)
			}
		}

		// Detect specific operations
		switch baseCmd {
		case "git":
			if len(parts) > 1 {
				tags = append(tags, "git:"+parts[1])
			}
		case "docker":
			if len(parts) > 1 {
				tags = append(tags, "docker:"+parts[1])
			}
		case "npm", "yarn":
			if len(parts) > 1 {
				tags = append(tags, "js:"+parts[1])
			}
		}
	}

	return tags
}

// detectSequences identifies command sequences in recent history
func (ha *HistoryAnalyzer) detectSequences() {
	if len(ha.recentCommands) < 2 {
		return
	}

	// Check for sequences of different lengths
	maxLen := historyMin(ha.config.SequenceMaxLength, len(ha.recentCommands))
	for length := 2; length <= maxLen; length++ {
		// Get the last `length` commands
		sequence := ha.recentCommands[len(ha.recentCommands)-length:]
		sequenceStr := strings.Join(sequence, " ; ")

		// Check if this sequence already exists
		found := false
		for i, seq := range ha.commandSequences {
			if strings.Join(seq.Commands, " ; ") == sequenceStr {
				// Update existing sequence
				ha.commandSequences[i].Frequency++
				ha.commandSequences[i].LastUsed = time.Now()
				found = true
				break
			}
		}

		if !found {
			// Create new sequence
			name := "seq_" + strings.ReplaceAll(strings.Join(sequence, "_"), " ", "")
			ha.commandSequences = append(ha.commandSequences, CommandSequence{
				Commands:      sequence,
				Frequency:     1,
				LastUsed:      time.Now(),
				MeaningfulName: name,
			})
		}
	}

	// Limit the number of sequences we track
	if len(ha.commandSequences) > 100 {
		// Sort by frequency descending
		sort.Slice(ha.commandSequences, func(i, j int) bool {
			return ha.commandSequences[i].Frequency > ha.commandSequences[j].Frequency
		})
		// Keep only the top 100
		ha.commandSequences = ha.commandSequences[:100]
	}
}

// rebuildIndexes rebuilds the search indexes from history
func (ha *HistoryAnalyzer) rebuildIndexes() {
	// Reset indexes
	ha.commandFrequency = make(map[string]int)
	ha.directoryCommands = make(map[string]map[string]int)
	ha.timePatterns = make(map[int]map[string]int)

	// Populate indexes from history
	for _, entry := range ha.history {
		// Update command frequency
		ha.commandFrequency[entry.Command] += entry.Frequency

		// Update directory commands
		dir := entry.Context.Directory
		if _, ok := ha.directoryCommands[dir]; !ok {
			ha.directoryCommands[dir] = make(map[string]int)
		}
		ha.directoryCommands[dir][entry.Command] += entry.Frequency

		// Update time patterns
		hour := entry.Context.Timestamp.Hour()
		if _, ok := ha.timePatterns[hour]; !ok {
			ha.timePatterns[hour] = make(map[string]int)
		}
		ha.timePatterns[hour][entry.Command] += entry.Frequency
	}
}

// GetSuggestions gets command suggestions based on current context
func (ha *HistoryAnalyzer) GetSuggestions(currentDirectory string) []CommandSuggestion {
	if !ha.isInitialized || !ha.config.Enabled {
		return nil
	}

	ha.historyLock.RLock()
	defer ha.historyLock.RUnlock()

	// Get current context
	now := time.Now()
	currentHour := now.Hour()

	// Build a priority queue of potential suggestions
	pq := make(PriorityQueue, 0)
	heap.Init(&pq)

	// Check commands run in this directory
	directoryWeight := ha.config.ContextWeight
	if dirCmds, ok := ha.directoryCommands[currentDirectory]; ok {
		for cmd, count := range dirCmds {
			confidence := float64(count) * directoryWeight
			heap.Push(&pq, &CommandSuggestionItem{
				command:    cmd,
				confidence: confidence,
				reason:     "frequently used in this directory",
			})
		}
	}

	// Check commands run at this time of day
	timeWeight := ha.config.TimeWeight
	if timeCmds, ok := ha.timePatterns[currentHour]; ok {
		for cmd, count := range timeCmds {
			confidence := float64(count) * timeWeight
			existing := pq.Find(cmd)
			
			if existing != nil {
				existing.confidence += confidence
				existing.reason = "frequently used in this directory and at this time"
			} else {
				heap.Push(&pq, &CommandSuggestionItem{
					command:    cmd,
					confidence: confidence,
					reason:     "frequently used at this time",
				})
			}
		}
	}

	// Add frequently used commands with a lower weight
	frequencyWeight := ha.config.FrequencyWeight
	for cmd, count := range ha.commandFrequency {
		confidence := float64(count) * frequencyWeight
		existing := pq.Find(cmd)
		
		if existing != nil {
			existing.confidence += confidence
		} else {
			heap.Push(&pq, &CommandSuggestionItem{
				command:    cmd,
				confidence: confidence,
				reason:     "frequently used command",
			})
		}
	}

	// Check for command sequences where the last command was used
	if len(ha.recentCommands) > 0 && ha.config.TrackCommandSequences {
		lastCmd := ha.recentCommands[len(ha.recentCommands)-1]
		
		for _, seq := range ha.commandSequences {
			if len(seq.Commands) >= 2 && seq.Commands[0] == lastCmd {
				nextCmd := seq.Commands[1]
				confidence := float64(seq.Frequency) * 0.5 // Sequence confidence is higher
				
				existing := pq.Find(nextCmd)
				if existing != nil {
					existing.confidence += confidence
					existing.reason = "frequently follows your last command"
					existing.isSequence = true
					existing.sequenceName = seq.MeaningfulName
				} else {
					heap.Push(&pq, &CommandSuggestionItem{
						command:      nextCmd,
						confidence:   confidence,
						reason:       "frequently follows your last command",
						isSequence:   true,
						sequenceName: seq.MeaningfulName,
					})
				}
			}
		}
	}

	// Build result from priority queue
	var result []CommandSuggestion
	for pq.Len() > 0 && len(result) < ha.config.MaxSuggestions {
		item := heap.Pop(&pq).(*CommandSuggestionItem)
		
		// Only include items above the confidence threshold
		if item.confidence >= ha.config.MinConfidenceThreshold {
			result = append(result, CommandSuggestion{
				Command:      item.command,
				Confidence:   item.confidence,
				Reason:       item.reason,
				IsSequence:   item.isSequence,
				SequenceName: item.sequenceName,
			})
		}
	}

	return result
}

// SearchHistory searches the command history
func (ha *HistoryAnalyzer) SearchHistory(query string, maxResults int) []EnhancedHistoryEntry {
	if !ha.isInitialized || !ha.config.Enabled {
		return nil
	}

	ha.historyLock.RLock()
	defer ha.historyLock.RUnlock()

	if maxResults <= 0 {
		maxResults = 10 // Default to 10 results
	}

	// Check if it's a natural language query
	if ha.config.EnableNLSearch && strings.Contains(query, " ") {
		return ha.naturalLanguageSearch(query, maxResults)
	}

	// Exact match search
	var matches []EnhancedHistoryEntry
	for _, entry := range ha.history {
		if strings.Contains(entry.Command, query) {
			matches = append(matches, entry)
		}
	}

	// Sort by frequency and recency
	sort.Slice(matches, func(i, j int) bool {
		// Primary sort by frequency
		if matches[i].Frequency != matches[j].Frequency {
			return matches[i].Frequency > matches[j].Frequency
		}
		// Secondary sort by recency
		return matches[i].LastUsed.After(matches[j].LastUsed)
	})

	// Limit results
	if len(matches) > maxResults {
		matches = matches[:maxResults]
	}

	return matches
}

// naturalLanguageSearch performs a more advanced search based on natural language
func (ha *HistoryAnalyzer) naturalLanguageSearch(query string, maxResults int) []EnhancedHistoryEntry {
	// Extract keywords from the query
	keywords := strings.Fields(query)
	
	// Score each entry based on keyword matches
	type ScoredEntry struct {
		entry EnhancedHistoryEntry
		score float64
	}
	
	var scoredEntries []ScoredEntry
	
	for _, entry := range ha.history {
		score := 0.0
		
		// Check command text
		for _, keyword := range keywords {
			if strings.Contains(strings.ToLower(entry.Command), strings.ToLower(keyword)) {
				score += 1.0
			}
		}
		
		// Check tags
		for _, tag := range entry.Tags {
			for _, keyword := range keywords {
				if strings.Contains(strings.ToLower(tag), strings.ToLower(keyword)) {
					score += 0.5
				}
			}
		}
		
		// Check directory
		for _, keyword := range keywords {
			if strings.Contains(strings.ToLower(entry.Context.Directory), strings.ToLower(keyword)) {
				score += 0.3
			}
		}
		
		// Only include entries with a positive score
		if score > 0 {
			scoredEntries = append(scoredEntries, ScoredEntry{
				entry: entry,
				score: score,
			})
		}
	}
	
	// Sort by score descending
	sort.Slice(scoredEntries, func(i, j int) bool {
		return scoredEntries[i].score > scoredEntries[j].score
	})
	
	// Extract the top results
	var result []EnhancedHistoryEntry
	for i := 0; i < historyMin(len(scoredEntries), maxResults); i++ {
		result = append(result, scoredEntries[i].entry)
	}
	
	return result
}

// GetCommandStats gets statistics for a specific command
func (ha *HistoryAnalyzer) GetCommandStats(command string) map[string]interface{} {
	if !ha.isInitialized || !ha.config.Enabled {
		return nil
	}

	ha.historyLock.RLock()
	defer ha.historyLock.RUnlock()

	stats := make(map[string]interface{})
	
	// Find the command in history
	var entry *EnhancedHistoryEntry
	for i := range ha.history {
		if ha.history[i].Command == command {
			entry = &ha.history[i]
			break
		}
	}
	
	if entry == nil {
		stats["found"] = false
		return stats
	}
	
	stats["found"] = true
	stats["frequency"] = entry.Frequency
	stats["category"] = entry.Category
	stats["tags"] = entry.Tags
	stats["last_used"] = entry.LastUsed
	stats["is_important"] = entry.IsImportant
	
	// Get directory distribution
	dirStats := make(map[string]int)
	total := 0
	for dir, cmds := range ha.directoryCommands {
		if count, ok := cmds[command]; ok {
			dirStats[dir] = count
			total += count
		}
	}
	stats["directory_stats"] = dirStats
	
	// Get time distribution
	timeStats := make(map[int]int)
	for hour, cmds := range ha.timePatterns {
		if count, ok := cmds[command]; ok {
			timeStats[hour] = count
		}
	}
	stats["time_stats"] = timeStats
	
	// Find related commands (commands often used before or after this one)
	// This is a placeholder for future implementation
	relatedCommands := []string{}
	stats["related_commands"] = relatedCommands
	
	return stats
}

// GetHistoryStats gets overall statistics for the command history
func (ha *HistoryAnalyzer) GetHistoryStats() map[string]interface{} {
	if !ha.isInitialized || !ha.config.Enabled {
		return nil
	}

	ha.historyLock.RLock()
	defer ha.historyLock.RUnlock()

	stats := make(map[string]interface{})
	
	// Basic stats
	stats["total_entries"] = len(ha.history)
	stats["unique_commands"] = len(ha.commandFrequency)
	stats["total_command_executions"] = 0
	for _, freq := range ha.commandFrequency {
		stats["total_command_executions"] = stats["total_command_executions"].(int) + freq
	}
	
	// Most used commands
	type CommandCount struct {
		Command string
		Count   int
	}
	
	var topCommands []CommandCount
	for cmd, count := range ha.commandFrequency {
		topCommands = append(topCommands, CommandCount{cmd, count})
	}
	
	sort.Slice(topCommands, func(i, j int) bool {
		return topCommands[i].Count > topCommands[j].Count
	})
	
	if len(topCommands) > 10 {
		topCommands = topCommands[:10]
	}
	
	stats["top_commands"] = topCommands
	
	// Category distribution
	categoryStats := make(map[string]int)
	for _, entry := range ha.history {
		categoryStats[entry.Category] += entry.Frequency
	}
	stats["category_stats"] = categoryStats
	
	// Directory distribution
	dirStats := make(map[string]int)
	for dir, cmds := range ha.directoryCommands {
		dirCount := 0
		for _, count := range cmds {
			dirCount += count
		}
		dirStats[dir] = dirCount
	}
	stats["directory_stats"] = dirStats
	
	// Time distribution
	timeStats := make(map[int]int)
	for hour, cmds := range ha.timePatterns {
		hourCount := 0
		for _, count := range cmds {
			hourCount += count
		}
		timeStats[hour] = hourCount
	}
	stats["time_stats"] = timeStats
	
	// Sequence stats
	stats["command_sequences"] = len(ha.commandSequences)
	
	return stats
}

// IsEnabled returns whether history analysis is enabled
func (ha *HistoryAnalyzer) IsEnabled() bool {
	ha.historyLock.RLock()
	defer ha.historyLock.RUnlock()
	return ha.isInitialized && ha.config.Enabled
}

// EnableHistoryAnalysis enables history analysis
func (ha *HistoryAnalyzer) EnableHistoryAnalysis() {
	ha.historyLock.Lock()
	defer ha.historyLock.Unlock()
	
	ha.config.Enabled = true
	ha.saveConfig()
}

// DisableHistoryAnalysis disables history analysis
func (ha *HistoryAnalyzer) DisableHistoryAnalysis() {
	ha.historyLock.Lock()
	defer ha.historyLock.Unlock()
	
	ha.config.Enabled = false
	ha.saveConfig()
}

// MarkCommandImportant marks a command as important
func (ha *HistoryAnalyzer) MarkCommandImportant(command string, isImportant bool) {
	ha.historyLock.Lock()
	defer ha.historyLock.Unlock()
	
	for i := range ha.history {
		if ha.history[i].Command == command {
			ha.history[i].IsImportant = isImportant
			break
		}
	}
}

// Global HistoryAnalyzer instance
var globalHistoryAnalyzer *HistoryAnalyzer

// GetHistoryAnalyzer returns the global HistoryAnalyzer instance
func GetHistoryAnalyzer() *HistoryAnalyzer {
	if globalHistoryAnalyzer == nil {
		var err error
		globalHistoryAnalyzer, err = NewHistoryAnalyzer()
		if err != nil {
			fmt.Printf("Error initializing history analyzer: %v\n", err)
			return nil
		}
		
		// Initialize the history analyzer
		globalHistoryAnalyzer.Initialize()
	}
	return globalHistoryAnalyzer
}

// CommandSuggestionItem is used for the priority queue
type CommandSuggestionItem struct {
	command      string
	confidence   float64
	reason       string
	isSequence   bool
	sequenceName string
	index        int // For heap interface
}

// PriorityQueue implements heap.Interface and holds CommandSuggestionItems
type PriorityQueue []*CommandSuggestionItem

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// We want a max heap, so we use > instead of <
	return pq[i].confidence > pq[j].confidence
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*CommandSuggestionItem)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

// Find searches for a command in the priority queue
func (pq PriorityQueue) Find(command string) *CommandSuggestionItem {
	for _, item := range pq {
		if item.command == command {
			return item
		}
	}
	return nil
}

// historyMin returns the smaller of two integers
func historyMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}