package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// HistoryEntry represents a parsed command from shell history
type HistoryEntry struct {
	Command   string
	Timestamp time.Time
	Duration  int64 // in seconds, for zsh extended history
	SessionID string
	Valid     bool
}

// ShellHistoryParser handles parsing of different shell history formats
type ShellHistoryParser struct {
	zshExtendedRegex *regexp.Regexp
}

// NewShellHistoryParser creates a new parser instance
func NewShellHistoryParser() *ShellHistoryParser {
	// Regex for zsh extended history format: : <timestamp>:<duration>;<command>
	zshRegex := regexp.MustCompile(`^:\s*(\d+):(\d+);(.*)$`)
	
	return &ShellHistoryParser{
		zshExtendedRegex: zshRegex,
	}
}

// ParseHistoryFile parses a shell history file based on its format
func (p *ShellHistoryParser) ParseHistoryFile(filePath, format string, maxEntries int) ([]HistoryEntry, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open history file: %w", err)
	}
	defer file.Close()
	
	switch format {
	case "zsh":
		return p.parseZshHistory(file, maxEntries)
	case "bash":
		return p.parseBashHistory(file, maxEntries)
	case "simple":
		return p.parseSimpleHistory(file, maxEntries)
	default:
		return p.parseSimpleHistory(file, maxEntries)
	}
}

// parseZshHistory parses zsh history file with extended format support
func (p *ShellHistoryParser) parseZshHistory(file *os.File, maxEntries int) ([]HistoryEntry, error) {
	var entries []HistoryEntry
	scanner := bufio.NewScanner(file)
	count := 0
	
	for scanner.Scan() && (maxEntries == 0 || count < maxEntries) {
		line := scanner.Text()
		
		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}
		
		entry := p.parseZshLine(line)
		if entry.Valid {
			entries = append(entries, entry)
			count++
		}
	}
	
	if err := scanner.Err(); err != nil {
		return entries, fmt.Errorf("error reading zsh history: %w", err)
	}
	
	return entries, nil
}

// parseZshLine parses a single line from zsh history
func (p *ShellHistoryParser) parseZshLine(line string) HistoryEntry {
	// Try to match zsh extended history format
	matches := p.zshExtendedRegex.FindStringSubmatch(line)
	if len(matches) == 4 {
		timestamp, err1 := strconv.ParseInt(matches[1], 10, 64)
		duration, err2 := strconv.ParseInt(matches[2], 10, 64)
		command := strings.TrimSpace(matches[3])
		
		if err1 == nil && err2 == nil && command != "" {
			return HistoryEntry{
				Command:   command,
				Timestamp: time.Unix(timestamp, 0),
				Duration:  duration,
				Valid:     true,
			}
		}
	}
	
	// Fall back to treating the line as a simple command
	command := strings.TrimSpace(line)
	if command != "" && !strings.HasPrefix(command, "#") {
		return HistoryEntry{
			Command: command,
			Valid:   true,
		}
	}
	
	return HistoryEntry{Valid: false}
}

// parseBashHistory parses bash history file (simple newline-separated format)
func (p *ShellHistoryParser) parseBashHistory(file *os.File, maxEntries int) ([]HistoryEntry, error) {
	var entries []HistoryEntry
	scanner := bufio.NewScanner(file)
	count := 0
	
	for scanner.Scan() && (maxEntries == 0 || count < maxEntries) {
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		entry := HistoryEntry{
			Command: line,
			Valid:   true,
		}
		
		entries = append(entries, entry)
		count++
	}
	
	if err := scanner.Err(); err != nil {
		return entries, fmt.Errorf("error reading bash history: %w", err)
	}
	
	return entries, nil
}

// parseSimpleHistory parses simple history files (one command per line)
func (p *ShellHistoryParser) parseSimpleHistory(file *os.File, maxEntries int) ([]HistoryEntry, error) {
	return p.parseBashHistory(file, maxEntries) // Same format as bash
}

// FilterEntries applies filters to history entries
func (p *ShellHistoryParser) FilterEntries(entries []HistoryEntry, filters HistoryFilters) []HistoryEntry {
	var filtered []HistoryEntry
	
	for _, entry := range entries {
		if p.shouldIncludeEntry(entry, filters) {
			filtered = append(filtered, entry)
		}
	}
	
	return filtered
}

// HistoryFilters defines criteria for filtering history entries
type HistoryFilters struct {
	ExcludePatterns  []string // Commands matching these patterns will be excluded
	MinLength        int      // Minimum command length
	ExcludeSensitive bool     // Exclude potentially sensitive commands
	StartTime        *time.Time // Only include commands after this time
	EndTime          *time.Time // Only include commands before this time
}

// shouldIncludeEntry determines if a history entry should be included based on filters
func (p *ShellHistoryParser) shouldIncludeEntry(entry HistoryEntry, filters HistoryFilters) bool {
	// Check minimum length
	if len(entry.Command) < filters.MinLength {
		return false
	}
	
	// Check time range
	if filters.StartTime != nil && !entry.Timestamp.IsZero() && entry.Timestamp.Before(*filters.StartTime) {
		return false
	}
	if filters.EndTime != nil && !entry.Timestamp.IsZero() && entry.Timestamp.After(*filters.EndTime) {
		return false
	}
	
	// Check exclude patterns
	for _, pattern := range filters.ExcludePatterns {
		if strings.Contains(strings.ToLower(entry.Command), strings.ToLower(pattern)) {
			return false
		}
	}
	
	// Check for sensitive commands
	if filters.ExcludeSensitive && p.isSensitiveCommand(entry.Command) {
		return false
	}
	
	return true
}

// isSensitiveCommand checks if a command might contain sensitive information
func (p *ShellHistoryParser) isSensitiveCommand(command string) bool {
	sensitivePatterns := []string{
		"password",
		"passwd",
		"token",
		"key",
		"secret",
		"api_key",
		"auth",
		"login",
		"sudo",
		"su ",
		"ssh-keygen",
		"gpg",
		"openssl",
		"curl.*-H.*Authorization",
		"wget.*--header.*Authorization",
		"export.*KEY",
		"export.*TOKEN",
		"export.*PASSWORD",
	}
	
	lowerCommand := strings.ToLower(command)
	
	for _, pattern := range sensitivePatterns {
		if matched, _ := regexp.MatchString(pattern, lowerCommand); matched {
			return true
		}
	}
	
	return false
}

// GetParsingStats returns statistics about parsed history entries
func (p *ShellHistoryParser) GetParsingStats(entries []HistoryEntry) map[string]interface{} {
	stats := make(map[string]interface{})
	
	totalEntries := len(entries)
	validEntries := 0
	withTimestamps := 0
	uniqueCommands := make(map[string]int)
	
	var oldestTime, newestTime time.Time
	commandLengths := make([]int, 0, totalEntries)
	
	for _, entry := range entries {
		if entry.Valid {
			validEntries++
		}
		
		if !entry.Timestamp.IsZero() {
			withTimestamps++
			
			if oldestTime.IsZero() || entry.Timestamp.Before(oldestTime) {
				oldestTime = entry.Timestamp
			}
			if newestTime.IsZero() || entry.Timestamp.After(newestTime) {
				newestTime = entry.Timestamp
			}
		}
		
		// Count unique commands
		baseCmd := strings.Fields(entry.Command)[0]
		if baseCmd != "" {
			uniqueCommands[baseCmd]++
		}
		
		commandLengths = append(commandLengths, len(entry.Command))
	}
	
	stats["total_entries"] = totalEntries
	stats["valid_entries"] = validEntries
	stats["with_timestamps"] = withTimestamps
	stats["unique_commands"] = len(uniqueCommands)
	stats["command_frequencies"] = uniqueCommands
	
	if !oldestTime.IsZero() {
		stats["oldest_entry"] = oldestTime.Format(time.RFC3339)
	}
	if !newestTime.IsZero() {
		stats["newest_entry"] = newestTime.Format(time.RFC3339)
	}
	
	// Calculate average command length
	if len(commandLengths) > 0 {
		total := 0
		for _, length := range commandLengths {
			total += length
		}
		stats["avg_command_length"] = total / len(commandLengths)
	}
	
	return stats
}

// ConvertToTrainingData converts history entries to Delta's training data format
func (p *ShellHistoryParser) ConvertToTrainingData(entries []HistoryEntry, context string) []TrainingData {
	var trainingData []TrainingData
	
	for _, entry := range entries {
		if !entry.Valid {
			continue
		}
		
		// Extract command components
		fields := strings.Fields(entry.Command)
		if len(fields) == 0 {
			continue
		}
		
		baseCommand := fields[0]
		args := ""
		if len(fields) > 1 {
			args = strings.Join(fields[1:], " ")
		}
		
		data := TrainingData{
			Command:     entry.Command,
			BaseCommand: baseCommand,
			Arguments:   args,
			Context:     context,
			Timestamp:   entry.Timestamp,
			Source:      "shell_history",
		}
		
		trainingData = append(trainingData, data)
	}
	
	return trainingData
}