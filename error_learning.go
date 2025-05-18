package main

import (
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

// ErrorSolutionHistory represents a record of error solutions and their success rates
type ErrorSolutionHistory struct {
	Version      string                    `json:"version"`
	LastUpdated  string                    `json:"last_updated"`
	ErrorEntries map[string]ErrorEntryData `json:"error_entries"`
	mutex        sync.RWMutex              `json:"-"`
	filePath     string                    `json:"-"`
}

// ErrorEntryData represents data about an error pattern and its solutions
type ErrorEntryData struct {
	ErrorPattern string                 `json:"error_pattern"`
	Solutions    []SolutionEffectiveness `json:"solutions"`
	FirstSeen    string                 `json:"first_seen"`
	LastSeen     string                 `json:"last_seen"`
	Occurrences  int                    `json:"occurrences"`
}

// SolutionEffectiveness tracks the effectiveness of a particular solution
type SolutionEffectiveness struct {
	Solution     string   `json:"solution"`
	Description  string   `json:"description"`
	SuccessCount int      `json:"success_count"`
	FailureCount int      `json:"failure_count"`
	LastSuccess  string   `json:"last_success"`
	LastFailure  string   `json:"last_failure"`
	Contexts     []string `json:"contexts"`
	Source       string   `json:"source"` // "ai", "user", "system"
}

// ErrorLearningManager manages the learning of error solutions
type ErrorLearningManager struct {
	history *ErrorSolutionHistory
	isInitialized bool
}

// NewErrorLearningManager creates a new error learning manager
func NewErrorLearningManager() (*ErrorLearningManager, error) {
	// Get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}

	// Create learning directory if it doesn't exist
	learningDir := filepath.Join(homeDir, ".delta", "learning")
	if err := os.MkdirAll(learningDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create learning directory: %v", err)
	}

	// File path for solution history
	historyPath := filepath.Join(learningDir, "error_solutions.json")

	// Initialize history
	history := &ErrorSolutionHistory{
		Version:      "1.0",
		LastUpdated:  time.Now().Format(time.RFC3339),
		ErrorEntries: make(map[string]ErrorEntryData),
		filePath:     historyPath,
	}

	// Load existing history if available
	if _, err := os.Stat(historyPath); err == nil {
		data, err := os.ReadFile(historyPath)
		if err == nil {
			if err := json.Unmarshal(data, history); err == nil {
				fmt.Println("Loaded error solution history from", historyPath)
			}
		}
	}

	return &ErrorLearningManager{
		history: history,
		isInitialized: true,
	}, nil
}

// AddErrorSolution adds a new error solution or updates an existing one
func (elm *ErrorLearningManager) AddErrorSolution(errorPattern, solution, description, context string, successful bool, source string) {
	if !elm.isInitialized {
		return
	}

	elm.history.mutex.Lock()
	defer elm.history.mutex.Unlock()

	// Normalize error pattern (remove specific paths, timestamps, etc.)
	normalizedPattern := elm.normalizeErrorPattern(errorPattern)
	
	// Check if this error pattern exists
	entry, exists := elm.history.ErrorEntries[normalizedPattern]
	if !exists {
		// Create new entry
		entry = ErrorEntryData{
			ErrorPattern: normalizedPattern,
			Solutions:    []SolutionEffectiveness{},
			FirstSeen:    time.Now().Format(time.RFC3339),
			LastSeen:     time.Now().Format(time.RFC3339),
			Occurrences:  1,
		}
	} else {
		// Update existing entry
		entry.LastSeen = time.Now().Format(time.RFC3339)
		entry.Occurrences++
	}

	// Find or create solution
	solutionFound := false
	for i, sol := range entry.Solutions {
		if sol.Solution == solution {
			// Update existing solution
			if successful {
				entry.Solutions[i].SuccessCount++
				entry.Solutions[i].LastSuccess = time.Now().Format(time.RFC3339)
			} else {
				entry.Solutions[i].FailureCount++
				entry.Solutions[i].LastFailure = time.Now().Format(time.RFC3339)
			}

			// Add context if not already present
			contextExists := false
			for _, ctx := range entry.Solutions[i].Contexts {
				if ctx == context {
					contextExists = true
					break
				}
			}
			if !contextExists && context != "" {
				entry.Solutions[i].Contexts = append(entry.Solutions[i].Contexts, context)
			}

			solutionFound = true
			break
		}
	}

	if !solutionFound {
		// Add new solution
		newSolution := SolutionEffectiveness{
			Solution:    solution,
			Description: description,
			Source:      source,
			Contexts:    []string{},
		}

		if context != "" {
			newSolution.Contexts = append(newSolution.Contexts, context)
		}

		if successful {
			newSolution.SuccessCount = 1
			newSolution.LastSuccess = time.Now().Format(time.RFC3339)
		} else {
			newSolution.FailureCount = 1
			newSolution.LastFailure = time.Now().Format(time.RFC3339)
		}

		entry.Solutions = append(entry.Solutions, newSolution)
	}

	// Sort solutions by success rate
	sort.Slice(entry.Solutions, func(i, j int) bool {
		iTotal := entry.Solutions[i].SuccessCount + entry.Solutions[i].FailureCount
		jTotal := entry.Solutions[j].SuccessCount + entry.Solutions[j].FailureCount
		
		// Avoid division by zero
		if iTotal == 0 {
			return false
		}
		if jTotal == 0 {
			return true
		}
		
		iRate := float64(entry.Solutions[i].SuccessCount) / float64(iTotal)
		jRate := float64(entry.Solutions[j].SuccessCount) / float64(jTotal)
		
		return iRate > jRate
	})

	elm.history.ErrorEntries[normalizedPattern] = entry
	elm.history.LastUpdated = time.Now().Format(time.RFC3339)

	// Save history
	elm.saveHistory()
}

// GetBestSolutions returns the best solutions for a given error pattern
func (elm *ErrorLearningManager) GetBestSolutions(errorPattern string, limit int) []SolutionEffectiveness {
	if !elm.isInitialized {
		return nil
	}

	elm.history.mutex.RLock()
	defer elm.history.mutex.RUnlock()

	// Normalize error pattern
	normalizedPattern := elm.normalizeErrorPattern(errorPattern)
	
	// Find similar patterns
	var matchingSolutions []SolutionEffectiveness
	
	// First try exact match
	if entry, exists := elm.history.ErrorEntries[normalizedPattern]; exists {
		matchingSolutions = append(matchingSolutions, entry.Solutions...)
	}
	
	// Then try pattern matching for similar errors
	for pattern, entry := range elm.history.ErrorEntries {
		// Skip if this is the exact pattern we already checked
		if pattern == normalizedPattern {
			continue
		}
		
		// Check if patterns are similar
		if elm.arePatternsSimilar(normalizedPattern, pattern) {
			matchingSolutions = append(matchingSolutions, entry.Solutions...)
		}
	}
	
	// Sort solutions by success rate
	sort.Slice(matchingSolutions, func(i, j int) bool {
		iTotal := matchingSolutions[i].SuccessCount + matchingSolutions[i].FailureCount
		jTotal := matchingSolutions[j].SuccessCount + matchingSolutions[j].FailureCount
		
		// Avoid division by zero
		if iTotal == 0 {
			return false
		}
		if jTotal == 0 {
			return true
		}
		
		iRate := float64(matchingSolutions[i].SuccessCount) / float64(iTotal)
		jRate := float64(matchingSolutions[j].SuccessCount) / float64(jTotal)
		
		// If success rates are equal, prefer solutions with more data
		if iRate == jRate {
			return iTotal > jTotal
		}
		
		return iRate > jRate
	})
	
	// Limit results
	if limit > 0 && len(matchingSolutions) > limit {
		matchingSolutions = matchingSolutions[:limit]
	}
	
	return matchingSolutions
}

// ExportToErrorPatterns exports learned solutions to error patterns file
func (elm *ErrorLearningManager) ExportToErrorPatterns() error {
	if !elm.isInitialized {
		return fmt.Errorf("error learning manager not initialized")
	}

	// Get user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err)
	}

	// Pattern file path
	patternFilePath := filepath.Join(homeDir, ".delta", "patterns", "error_patterns.json")

	// Check if the file exists
	if _, err := os.Stat(patternFilePath); os.IsNotExist(err) {
		return fmt.Errorf("pattern file does not exist: %s", patternFilePath)
	}

	// Read the pattern file
	data, err := os.ReadFile(patternFilePath)
	if err != nil {
		return fmt.Errorf("failed to read pattern file: %v", err)
	}

	// Parse the JSON data
	var library PatternLibrary
	if err := json.Unmarshal(data, &library); err != nil {
		return fmt.Errorf("failed to parse pattern library: %v", err)
	}

	// Create a map of existing patterns
	existingPatterns := make(map[string]bool)
	for _, pattern := range library.Patterns {
		existingPatterns[pattern.Pattern] = true
	}

	elm.history.mutex.RLock()
	defer elm.history.mutex.RUnlock()

	// Add learned patterns that have a high success rate
	var newPatterns []PatternEntry
	for _, entry := range elm.history.ErrorEntries {
		// Only consider entries with solutions
		if len(entry.Solutions) == 0 {
			continue
		}

		// Get the best solution
		bestSolution := entry.Solutions[0]
		
		// Calculate success rate
		total := bestSolution.SuccessCount + bestSolution.FailureCount
		if total < 3 {
			// Not enough data
			continue
		}
		
		successRate := float64(bestSolution.SuccessCount) / float64(total)
		if successRate < 0.8 {
			// Not reliable enough
			continue
		}

		// Check if this pattern already exists
		if existingPatterns[entry.ErrorPattern] {
			continue
		}

		// Add new pattern
		newPattern := PatternEntry{
			Pattern:     entry.ErrorPattern,
			Solution:    bestSolution.Solution,
			Description: bestSolution.Description,
			Category:    "learned",
		}

		newPatterns = append(newPatterns, newPattern)
	}

	// Add new patterns to library
	library.Patterns = append(library.Patterns, newPatterns...)
	library.UpdatedAt = time.Now().Format("2006-01-02")

	// Write updated pattern file
	updatedData, err := json.MarshalIndent(library, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal pattern library: %v", err)
	}

	if err := os.WriteFile(patternFilePath, updatedData, 0644); err != nil {
		return fmt.Errorf("failed to write pattern file: %v", err)
	}

	fmt.Printf("Exported %d learned error patterns to %s\n", len(newPatterns), patternFilePath)
	return nil
}

// saveHistory saves the solution history to disk
func (elm *ErrorLearningManager) saveHistory() {
	// Marshal the history to JSON
	data, err := json.MarshalIndent(elm.history, "", "  ")
	if err != nil {
		fmt.Printf("Error marshalling solution history: %v\n", err)
		return
	}

	// Write to file
	if err := os.WriteFile(elm.history.filePath, data, 0644); err != nil {
		fmt.Printf("Error writing solution history: %v\n", err)
	}
}

// normalizeErrorPattern normalizes an error pattern for better matching
func (elm *ErrorLearningManager) normalizeErrorPattern(pattern string) string {
	// Trim whitespace
	normalized := strings.TrimSpace(pattern)
	
	// Remove absolute paths (replace with placeholder)
	pathRegex := regexp.MustCompile(`/[^\s:]+/([^/\s:]+)`)
	normalized = pathRegex.ReplaceAllString(normalized, "/${PATH}/$1")
	
	// Remove timestamps
	timestampRegex := regexp.MustCompile(`\d{2}:\d{2}:\d{2}`)
	normalized = timestampRegex.ReplaceAllString(normalized, "${TIME}")
	
	// Remove version numbers
	versionRegex := regexp.MustCompile(`\d+\.\d+\.\d+`)
	normalized = versionRegex.ReplaceAllString(normalized, "${VERSION}")
	
	// Remove line numbers
	lineNumRegex := regexp.MustCompile(`line \d+`)
	normalized = lineNumRegex.ReplaceAllString(normalized, "line ${LINE}")
	
	// Remove hexadecimal addresses
	hexRegex := regexp.MustCompile(`0x[0-9a-fA-F]+`)
	normalized = hexRegex.ReplaceAllString(normalized, "${HEX}")
	
	// Limit length
	if len(normalized) > 200 {
		normalized = normalized[:200]
	}
	
	return normalized
}

// arePatternsSimilar checks if two error patterns are similar
func (elm *ErrorLearningManager) arePatternsSimilar(pattern1, pattern2 string) bool {
	// Simple check for now - see if one contains the other
	return strings.Contains(pattern1, pattern2) || strings.Contains(pattern2, pattern1)
}

// GenerateErrorSolution attempts to generate a solution for an error using AI
func (elm *ErrorLearningManager) GenerateErrorSolution(errorOutput string, context string) (string, string, error) {
	// Check if AI manager is available
	aiManager := GetAIManager()
	if aiManager == nil || !aiManager.IsEnabled() {
		return "", "", fmt.Errorf("AI manager not available")
	}

	// Generate prompt
	prompt := fmt.Sprintf("I encountered the following error:\n\n%s\n\nContext: %s\n\nProvide a solution for this error in the following format:\n\nSolution: [the solution to fix the error]\n\nExplanation: [why this solution works]", errorOutput, context)
	
	// Get AI response
	response, err := aiManager.GetAIResponse(prompt)
	if err != nil {
		return "", "", fmt.Errorf("failed to get AI response: %v", err)
	}

	// Extract solution
	solutionMatch := regexp.MustCompile(`(?i)Solution:\s*(.+?)(?:\n\n|\n(?:Explanation|$))`).FindStringSubmatch(response)
	if len(solutionMatch) > 1 {
		solution := strings.TrimSpace(solutionMatch[1])
		
		// Extract explanation
		explanationMatch := regexp.MustCompile(`(?i)Explanation:\s*(.+)$`).FindStringSubmatch(response)
		var explanation string
		if len(explanationMatch) > 1 {
			explanation = strings.TrimSpace(explanationMatch[1])
		}
		
		return solution, explanation, nil
	}
	
	// If no explicit solution format, use the whole response
	return strings.TrimSpace(response), "", nil
}

// FixErrorAutomatically attempts to automatically fix an error
func (elm *ErrorLearningManager) FixErrorAutomatically(errorOutput string, context string) (bool, string, error) {
	if !elm.isInitialized {
		return false, "", fmt.Errorf("error learning manager not initialized")
	}
	
	// First try to find a solution in the history
	solutions := elm.GetBestSolutions(errorOutput, 1)
	if len(solutions) > 0 && solutions[0].SuccessCount > 0 {
		// Found a solution with success history
		return true, solutions[0].Solution, nil
	}
	
	// If no solution found, try to generate one with AI
	solution, _, err := elm.GenerateErrorSolution(errorOutput, context)
	if err != nil {
		return false, "", fmt.Errorf("failed to generate solution: %v", err)
	}
	
	return false, solution, nil
}

// Global ErrorLearningManager instance
var globalErrorLearningManager *ErrorLearningManager

// GetErrorLearningManager returns the global ErrorLearningManager instance
func GetErrorLearningManager() *ErrorLearningManager {
	if globalErrorLearningManager == nil {
		var err error
		globalErrorLearningManager, err = NewErrorLearningManager()
		if err != nil {
			fmt.Printf("Error initializing error learning manager: %v\n", err)
			return nil
		}
	}
	return globalErrorLearningManager
}