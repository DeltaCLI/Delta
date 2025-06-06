package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// CommandPattern represents a recognized pattern in command usage
type CommandPattern struct {
	Type        string   `json:"type"`        // Pattern type (sequence, workflow, prefix, etc.)
	Commands    []string `json:"commands"`    // Commands in the pattern
	Frequency   int      `json:"frequency"`   // How often this pattern occurs
	Confidence  float64  `json:"confidence"`  // Confidence score (0.0-1.0)
	Description string   `json:"description"` // Human-readable description
	Tags        []string `json:"tags"`        // Tags for categorization
}

// TaskWorkflow represents a higher-level task composed of command sequences
type TaskWorkflow struct {
	Name        string           `json:"name"`        // Workflow name
	Description string           `json:"description"` // Human-readable description
	Patterns    []CommandPattern `json:"patterns"`    // Command patterns in this workflow
	Frequency   int              `json:"frequency"`   // How often this workflow is executed
	LastUsed    string           `json:"last_used"`   // When this workflow was last used
}

// findCommandPatterns analyzes command history to find patterns
func (ha *HistoryAnalyzer) findCommandPatterns() []CommandPattern {
	ha.historyLock.RLock()
	defer ha.historyLock.RUnlock()

	var patterns []CommandPattern

	// Skip if not enough history
	if len(ha.history) < 10 {
		return patterns
	}

	// Find command sequences (commands that frequently follow each other)
	sequencePatterns := ha.findSequencePatterns()
	patterns = append(patterns, sequencePatterns...)

	// Find prefix patterns (commands that share common prefixes by context)
	prefixPatterns := ha.findPrefixPatterns()
	patterns = append(patterns, prefixPatterns...)

	// Find timing patterns (commands frequently used at certain times)
	timingPatterns := ha.findTimingPatterns()
	patterns = append(patterns, timingPatterns...)

	// Sort patterns by frequency
	sort.Slice(patterns, func(i, j int) bool {
		return patterns[i].Frequency > patterns[j].Frequency
	})

	return patterns
}

// findSequencePatterns finds sequences of commands that frequently occur together
func (ha *HistoryAnalyzer) findSequencePatterns() []CommandPattern {
	var patterns []CommandPattern

	// Create a map of command pairs and their frequency
	// Key format: "command1 -> command2"
	pairFrequency := make(map[string]int)

	// Track which commands precede others
	predecessors := make(map[string]map[string]int)

	// Analyze command history to build sequence information
	sortedHistory := make([]EnhancedHistoryEntry, len(ha.history))
	copy(sortedHistory, ha.history)

	// Sort by timestamp
	sort.Slice(sortedHistory, func(i, j int) bool {
		return sortedHistory[i].Context.Timestamp.Before(sortedHistory[j].Context.Timestamp)
	})

	// Analyze sequences
	for i := 1; i < len(sortedHistory); i++ {
		current := sortedHistory[i].Command
		previous := sortedHistory[i-1].Command

		// Extract base commands (remove arguments)
		currentBase := strings.Fields(current)[0]
		previousBase := strings.Fields(previous)[0]

		// Skip if the same command is repeated
		if currentBase == previousBase {
			continue
		}

		// Create the pair key
		pairKey := fmt.Sprintf("%s -> %s", previousBase, currentBase)
		pairFrequency[pairKey]++

		// Update predecessors map
		if _, ok := predecessors[currentBase]; !ok {
			predecessors[currentBase] = make(map[string]int)
		}
		predecessors[currentBase][previousBase]++
	}

	// Create pattern objects for significant sequences
	for pairKey, frequency := range pairFrequency {
		// Only include sequences that occur at least 3 times
		if frequency >= 3 {
			parts := strings.Split(pairKey, " -> ")
			if len(parts) == 2 {
				commands := []string{parts[0], parts[1]}

				// Calculate confidence based on how often second command follows first
				totalFirstCommandUsage := ha.commandFrequency[commands[0]]
				confidence := float64(frequency) / float64(totalFirstCommandUsage)

				// Only include patterns with reasonable confidence
				if confidence >= 0.3 {
					pattern := CommandPattern{
						Type:        "sequence",
						Commands:    commands,
						Frequency:   frequency,
						Confidence:  confidence,
						Description: fmt.Sprintf("Command '%s' often follows '%s'", commands[1], commands[0]),
						Tags:        []string{"sequence", "automation"},
					}
					patterns = append(patterns, pattern)
				}
			}
		}
	}

	return patterns
}

// findPrefixPatterns finds commands that share common prefixes by context
func (ha *HistoryAnalyzer) findPrefixPatterns() []CommandPattern {
	var patterns []CommandPattern

	// Group commands by directory
	dirCommands := make(map[string]map[string]int)

	for _, entry := range ha.history {
		dir := entry.Context.Directory
		command := entry.Command

		// Skip commands without directories
		if dir == "" {
			continue
		}

		// Initialize map for directory if not exists
		if _, ok := dirCommands[dir]; !ok {
			dirCommands[dir] = make(map[string]int)
		}

		// Increment command count for this directory
		dirCommands[dir][command]++
	}

	// Find common command prefixes by directory
	for dir, commands := range dirCommands {
		// Skip directories with too few commands
		if len(commands) < 3 {
			continue
		}

		// Find common prefix patterns
		prefixCounts := make(map[string]int)
		for cmd := range commands {
			parts := strings.Fields(cmd)
			if len(parts) >= 2 {
				prefix := parts[0]
				prefixCounts[prefix]++
			}
		}

		// Create patterns for common prefixes
		for prefix, count := range prefixCounts {
			if count >= 3 {
				// Calculate confidence as percentage of commands in this directory with this prefix
				confidence := float64(count) / float64(len(commands))

				if confidence >= 0.3 {
					dirName := dir
					if strings.HasPrefix(dirName, "/home/") {
						// Shorten home dir paths for readability
						dirName = "~" + dirName[strings.LastIndex(dirName, "/"):]
					}

					pattern := CommandPattern{
						Type:        "prefix",
						Commands:    []string{prefix + " *"},
						Frequency:   count,
						Confidence:  confidence,
						Description: fmt.Sprintf("Commands with prefix '%s' are common in directory '%s'", prefix, dirName),
						Tags:        []string{"prefix", "directory", dir},
					}
					patterns = append(patterns, pattern)
				}
			}
		}
	}

	return patterns
}

// findTimingPatterns finds commands frequently used at certain times
func (ha *HistoryAnalyzer) findTimingPatterns() []CommandPattern {
	var patterns []CommandPattern

	// Group commands by hour of day
	hourCommands := make(map[int]map[string]int)

	for _, entry := range ha.history {
		hour := entry.Context.Timestamp.Hour()
		command := entry.Command

		// Initialize map for hour if not exists
		if _, ok := hourCommands[hour]; !ok {
			hourCommands[hour] = make(map[string]int)
		}

		// Increment command count for this hour
		hourCommands[hour][command]++
	}

	// Time of day descriptions
	timeDescriptions := map[int]string{
		0: "late night", 1: "late night", 2: "late night", 3: "late night",
		4: "early morning", 5: "early morning", 6: "early morning", 7: "early morning",
		8: "morning", 9: "morning", 10: "morning", 11: "morning",
		12: "midday", 13: "afternoon", 14: "afternoon", 15: "afternoon",
		16: "late afternoon", 17: "early evening", 18: "evening", 19: "evening",
		20: "night", 21: "night", 22: "night", 23: "late night",
	}

	// Find time-specific commands
	for hour, commands := range hourCommands {
		// Skip hours with too few commands
		if len(commands) < 5 {
			continue
		}

		// Find commands with unusual frequency at this hour
		for cmd, count := range commands {
			// Skip commands used fewer than 3 times
			if count < 3 {
				continue
			}

			// Calculate overall usage of this command
			totalUsage := ha.commandFrequency[cmd]
			if totalUsage == 0 {
				continue // Skip if no data
			}

			// Calculate what percentage of this command's usage is at this hour
			hourPercentage := float64(count) / float64(totalUsage)

			// If more than 30% of the command's usage is at this hour, it's a pattern
			if hourPercentage >= 0.3 {
				timeDesc := timeDescriptions[hour]
				pattern := CommandPattern{
					Type:        "timing",
					Commands:    []string{cmd},
					Frequency:   count,
					Confidence:  hourPercentage,
					Description: fmt.Sprintf("Command '%s' is frequently used during %s hours (%d:00)", cmd, timeDesc, hour),
					Tags:        []string{"timing", "time-of-day", timeDesc},
				}
				patterns = append(patterns, pattern)
			}
		}
	}

	return patterns
}

// identifyTaskWorkflows groups command patterns into higher-level workflows
func (ha *HistoryAnalyzer) identifyTaskWorkflows() []TaskWorkflow {
	// Get command patterns
	patterns := ha.findCommandPatterns()

	// Skip if not enough patterns
	if len(patterns) < 5 {
		return []TaskWorkflow{}
	}

	// Group patterns by directory
	dirPatterns := make(map[string][]CommandPattern)

	for _, pattern := range patterns {
		// Extract directories from pattern tags
		for _, tag := range pattern.Tags {
			if strings.HasPrefix(tag, "/") {
				// It's a directory path
				dirPatterns[tag] = append(dirPatterns[tag], pattern)
				break
			}
		}
	}

	// Create workflows for directories with enough patterns
	var workflows []TaskWorkflow

	for dir, patterns := range dirPatterns {
		if len(patterns) >= 3 {
			// Directory has enough patterns to constitute a workflow
			dirName := dir
			if strings.HasPrefix(dirName, "/home/") {
				// Shorten home dir paths for readability
				parts := strings.Split(dirName, "/")
				if len(parts) > 3 {
					dirName = parts[len(parts)-1] // Just the last directory name
				}
			}

			workflow := TaskWorkflow{
				Name:        fmt.Sprintf("%s workflow", dirName),
				Description: fmt.Sprintf("Common command patterns in %s directory", dir),
				Patterns:    patterns,
				Frequency:   calculatePatternGroupFrequency(patterns),
				LastUsed:    "recently", // This would be replaced with actual timestamp
			}

			workflows = append(workflows, workflow)
		}
	}

	// Also group patterns by type
	typePatterns := make(map[string][]CommandPattern)

	for _, pattern := range patterns {
		typePatterns[pattern.Type] = append(typePatterns[pattern.Type], pattern)
	}

	// Create workflows for pattern types with enough patterns
	for patternType, patterns := range typePatterns {
		if len(patterns) >= 3 {
			typeName := strings.Title(patternType) // Capitalize

			workflow := TaskWorkflow{
				Name:        fmt.Sprintf("%s patterns", typeName),
				Description: fmt.Sprintf("Collection of %s command patterns", patternType),
				Patterns:    patterns,
				Frequency:   calculatePatternGroupFrequency(patterns),
				LastUsed:    "recently", // This would be replaced with actual timestamp
			}

			workflows = append(workflows, workflow)
		}
	}

	// Sort workflows by frequency
	sort.Slice(workflows, func(i, j int) bool {
		return workflows[i].Frequency > workflows[j].Frequency
	})

	return workflows
}

// calculatePatternGroupFrequency calculates the combined frequency of a group of patterns
func calculatePatternGroupFrequency(patterns []CommandPattern) int {
	total := 0
	for _, pattern := range patterns {
		total += pattern.Frequency
	}
	return total
}

// getNextCommandSuggestions gets suggested next commands based on the current command
func (ha *HistoryAnalyzer) getNextCommandSuggestions(currentCommand string) []CommandSuggestion {
	var suggestions []CommandSuggestion

	// Skip if not enough history
	if len(ha.history) < 10 {
		return suggestions
	}

	// Get command patterns
	patterns := ha.findCommandPatterns()

	// Find sequence patterns where current command is the first command
	for _, pattern := range patterns {
		if pattern.Type == "sequence" && len(pattern.Commands) >= 2 {
			// Check if this pattern starts with the current command
			if pattern.Commands[0] == currentCommand {
				// Suggest the next command in the sequence
				suggestion := CommandSuggestion{
					Command:      pattern.Commands[1],
					Confidence:   pattern.Confidence,
					Reason:       fmt.Sprintf("Frequently follows '%s'", currentCommand),
					IsSequence:   true,
					SequenceName: fmt.Sprintf("%s -> %s", pattern.Commands[0], pattern.Commands[1]),
				}
				suggestions = append(suggestions, suggestion)
			}
		}
	}

	// Sort suggestions by confidence
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Confidence > suggestions[j].Confidence
	})

	return suggestions
}

// enhancedNaturalLanguageSearch provides better natural language search for history
func (ha *HistoryAnalyzer) enhancedNaturalLanguageSearch(query string, maxResults int) []EnhancedHistoryEntry {
	ha.historyLock.RLock()
	defer ha.historyLock.RUnlock()

	// Extract keywords and any special operators from the query
	keywords := strings.Fields(query)

	// Check for special search operators
	dirFilter := ""
	timeFilter := ""
	categoryFilter := ""
	exitCodeFilter := -1

	processedKeywords := []string{}

	for _, keyword := range keywords {
		if strings.HasPrefix(keyword, "dir:") {
			dirFilter = strings.TrimPrefix(keyword, "dir:")
		} else if strings.HasPrefix(keyword, "time:") {
			timeFilter = strings.TrimPrefix(keyword, "time:")
		} else if strings.HasPrefix(keyword, "category:") {
			categoryFilter = strings.TrimPrefix(keyword, "category:")
		} else if strings.HasPrefix(keyword, "exit:") {
			codeStr := strings.TrimPrefix(keyword, "exit:")
			if code, err := strconv.Atoi(codeStr); err == nil {
				exitCodeFilter = code
			}
		} else {
			processedKeywords = append(processedKeywords, keyword)
		}
	}

	// Score each entry based on keyword matches and filters
	type ScoredEntry struct {
		entry EnhancedHistoryEntry
		score float64
	}

	var scoredEntries []ScoredEntry

	for _, entry := range ha.history {
		// Apply filters
		if dirFilter != "" && !strings.Contains(entry.Context.Directory, dirFilter) {
			continue
		}

		if categoryFilter != "" && entry.Category != categoryFilter {
			continue
		}

		if exitCodeFilter != -1 && entry.Context.ExitCode != exitCodeFilter {
			continue
		}

		if timeFilter != "" {
			// Parse time filter - this is a simplified version
			hour := entry.Context.Timestamp.Hour()
			switch timeFilter {
			case "morning":
				if hour < 6 || hour > 11 {
					continue
				}
			case "afternoon":
				if hour < 12 || hour > 17 {
					continue
				}
			case "evening":
				if hour < 18 || hour > 21 {
					continue
				}
			case "night":
				if hour < 22 && hour > 5 {
					continue
				}
			}
		}

		// Calculate match score
		score := 0.0

		// Check command text
		commandLower := strings.ToLower(entry.Command)
		for _, keyword := range processedKeywords {
			if strings.Contains(commandLower, strings.ToLower(keyword)) {
				score += 1.0
			}
		}

		// Check tags
		for _, tag := range entry.Tags {
			for _, keyword := range processedKeywords {
				if strings.Contains(strings.ToLower(tag), strings.ToLower(keyword)) {
					score += 0.5
				}
			}
		}

		// Add bonus for frequent or recent commands
		score += float64(entry.Frequency) * 0.05

		// Add bonus for important commands
		if entry.IsImportant {
			score += 0.5
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
	for i := 0; i < min(len(scoredEntries), maxResults); i++ {
		result = append(result, scoredEntries[i].entry)
	}

	return result
}
