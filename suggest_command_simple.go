package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SuggestManager handles natural language command suggestions
type SuggestManager struct {
	aiManager       *AIPredictionManager
	historyAnalyzer *HistoryAnalyzer
	lastQuery       string
	lastSuggestions []CommandSuggestion
}

// NewSuggestManager creates a new suggestion manager
func NewSuggestManager() *SuggestManager {
	return &SuggestManager{}
}

// Initialize sets up the suggest manager with dependencies
func (sm *SuggestManager) Initialize() error {
	sm.aiManager = GetAIManager()
	sm.historyAnalyzer = GetHistoryAnalyzer()
	return nil
}

// GetSuggestions generates command suggestions from natural language input
func (sm *SuggestManager) GetSuggestions(query string, limit int) ([]CommandSuggestion, error) {
	suggestions := []CommandSuggestion{}
	
	// Get current context
	pwd, _ := os.Getwd()
	projectType := sm.detectProjectType(pwd)
	
	// Try pattern-based suggestions first
	patternSuggestions := sm.getPatternBasedSuggestions(query, projectType)
	suggestions = append(suggestions, patternSuggestions...)
	
	// If AI is available, get AI-powered suggestions
	if sm.aiManager != nil && sm.aiManager.IsEnabled() {
		aiSuggestions, err := sm.getAISuggestions(query, projectType)
		if err == nil {
			suggestions = append(suggestions, aiSuggestions...)
		}
	}
	
	// Get history-based suggestions if history analyzer is available
	if sm.historyAnalyzer != nil && sm.historyAnalyzer.IsEnabled() {
		historySuggestions := sm.getHistoryBasedSuggestions(query, pwd)
		suggestions = append(suggestions, historySuggestions...)
	}
	
	// Limit results
	if len(suggestions) > limit {
		suggestions = suggestions[:limit]
	}
	
	sm.lastQuery = query
	sm.lastSuggestions = suggestions
	
	return suggestions, nil
}

// detectProjectType identifies the type of project in the current directory
func (sm *SuggestManager) detectProjectType(dir string) string {
	// Check for various project indicators
	indicators := map[string]string{
		"package.json":     "nodejs",
		"go.mod":           "golang",
		"Cargo.toml":       "rust",
		"pom.xml":          "java",
		"build.gradle":     "java",
		"requirements.txt": "python",
		"Pipfile":          "python",
		"composer.json":    "php",
		"Gemfile":          "ruby",
		"Makefile":         "make",
		"Dockerfile":       "docker",
		".git":             "git",
	}
	
	for file, projectType := range indicators {
		if _, err := os.Stat(filepath.Join(dir, file)); err == nil {
			return projectType
		}
	}
	
	return "general"
}

// getPatternBasedSuggestions uses keyword patterns to suggest commands
func (sm *SuggestManager) getPatternBasedSuggestions(query string, projectType string) []CommandSuggestion {
	suggestions := []CommandSuggestion{}
	queryLower := strings.ToLower(query)
	
	// Simple keyword matching
	if strings.Contains(queryLower, "list") || strings.Contains(queryLower, "files") {
		suggestions = append(suggestions, CommandSuggestion{
			Command:    "ls -la",
			Confidence: 0.9,
			Reason:     "List all files with details",
		})
	}
	
	if strings.Contains(queryLower, "find") || strings.Contains(queryLower, "search") {
		suggestions = append(suggestions, CommandSuggestion{
			Command:    "find . -name \"*pattern*\"",
			Confidence: 0.8,
			Reason:     "Find files by name pattern",
		})
		suggestions = append(suggestions, CommandSuggestion{
			Command:    "grep -r \"text\" .",
			Confidence: 0.7,
			Reason:     "Search for text in files",
		})
	}
	
	if strings.Contains(queryLower, "git") && strings.Contains(queryLower, "commit") {
		suggestions = append(suggestions, CommandSuggestion{
			Command:    "git add . && git commit -m \"message\"",
			Confidence: 0.9,
			Reason:     "Stage and commit changes",
		})
	}
	
	if strings.Contains(queryLower, "install") {
		switch projectType {
		case "nodejs":
			suggestions = append(suggestions, CommandSuggestion{
				Command:    "npm install",
				Confidence: 0.9,
				Reason:     "Install Node.js dependencies",
			})
		case "golang":
			suggestions = append(suggestions, CommandSuggestion{
				Command:    "go mod download",
				Confidence: 0.9,
				Reason:     "Download Go dependencies",
			})
		case "python":
			suggestions = append(suggestions, CommandSuggestion{
				Command:    "pip install -r requirements.txt",
				Confidence: 0.9,
				Reason:     "Install Python dependencies",
			})
		}
	}
	
	if strings.Contains(queryLower, "build") {
		switch projectType {
		case "golang":
			suggestions = append(suggestions, CommandSuggestion{
				Command:    "go build",
				Confidence: 0.9,
				Reason:     "Build Go project",
			})
		case "make":
			suggestions = append(suggestions, CommandSuggestion{
				Command:    "make build",
				Confidence: 0.9,
				Reason:     "Run make build target",
			})
		}
	}
	
	if strings.Contains(queryLower, "test") {
		switch projectType {
		case "golang":
			suggestions = append(suggestions, CommandSuggestion{
				Command:    "go test ./...",
				Confidence: 0.9,
				Reason:     "Run all Go tests",
			})
		case "nodejs":
			suggestions = append(suggestions, CommandSuggestion{
				Command:    "npm test",
				Confidence: 0.9,
				Reason:     "Run Node.js tests",
			})
		}
	}
	
	return suggestions
}

// getAISuggestions uses AI to generate command suggestions
func (sm *SuggestManager) getAISuggestions(query string, projectType string) ([]CommandSuggestion, error) {
	if sm.aiManager == nil || !sm.aiManager.IsEnabled() {
		return []CommandSuggestion{}, nil
	}
	
	// Create a prompt for the AI
	prompt := fmt.Sprintf(`Given the user wants to: "%s"
Current project type: %s
Current directory: %s

Suggest 3 shell commands that would accomplish this task. For each command:
1. Provide the exact command
2. Brief description of what it does

Format each suggestion as:
COMMAND: <command>
DESC: <description>
---`, query, projectType, filepath.Base(getCurrentDirectory()))

	response, err := sm.aiManager.ollamaClient.Generate(prompt, "You are a command-line expert assistant. Provide practical, safe command suggestions.")
	if err != nil {
		return []CommandSuggestion{}, err
	}
	
	// Parse AI response
	suggestions := sm.parseAIResponse(response)
	
	return suggestions, nil
}

// parseAIResponse parses the AI response into command suggestions
func (sm *SuggestManager) parseAIResponse(response string) []CommandSuggestion {
	suggestions := []CommandSuggestion{}
	
	// Split by separator
	parts := strings.Split(response, "---")
	
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		
		suggestion := CommandSuggestion{}
		
		lines := strings.Split(part, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			
			if strings.HasPrefix(line, "COMMAND:") {
				suggestion.Command = strings.TrimSpace(strings.TrimPrefix(line, "COMMAND:"))
			} else if strings.HasPrefix(line, "DESC:") {
				suggestion.Reason = strings.TrimSpace(strings.TrimPrefix(line, "DESC:"))
			}
		}
		
		if suggestion.Command != "" {
			suggestion.Confidence = 0.7 // AI suggestions get slightly lower confidence
			suggestions = append(suggestions, suggestion)
		}
	}
	
	return suggestions
}

// getHistoryBasedSuggestions uses command history to suggest commands
func (sm *SuggestManager) getHistoryBasedSuggestions(query string, currentDir string) []CommandSuggestion {
	if sm.historyAnalyzer == nil || !sm.historyAnalyzer.IsEnabled() {
		return []CommandSuggestion{}
	}
	
	// Use the history analyzer's existing suggestion mechanism
	suggestions := sm.historyAnalyzer.GetSuggestions(currentDir)
	
	// Filter based on query keywords
	queryLower := strings.ToLower(query)
	filtered := []CommandSuggestion{}
	
	for _, suggestion := range suggestions {
		cmdLower := strings.ToLower(suggestion.Command)
		// Check if command contains any query keywords
		words := strings.Fields(queryLower)
		matches := false
		for _, word := range words {
			if strings.Contains(cmdLower, word) {
				matches = true
				break
			}
		}
		if matches {
			filtered = append(filtered, suggestion)
		}
	}
	
	return filtered
}

// ExplainCommand provides detailed explanation of a command
func (sm *SuggestManager) ExplainCommand(command string) (string, error) {
	if sm.aiManager != nil && sm.aiManager.IsEnabled() {
		prompt := fmt.Sprintf(`Explain this command in detail: %s

Include:
1. What the command does
2. Each flag/option explanation
3. Common use cases
4. Potential risks or warnings`, command)

		return sm.aiManager.ollamaClient.Generate(prompt, "You are a command-line expert. Provide clear, detailed explanations.")
	}
	
	// Fallback to basic explanation
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "Empty command", nil
	}
	
	explanation := fmt.Sprintf("Command: %s\n", parts[0])
	
	// Add basic explanations for common commands
	commonExplanations := map[string]string{
		"ls":     "Lists directory contents",
		"cd":     "Changes the current directory",
		"cp":     "Copies files or directories",
		"mv":     "Moves or renames files",
		"rm":     "Removes files or directories",
		"git":    "Version control system command",
		"docker": "Container management command",
		"npm":    "Node.js package manager",
		"make":   "Build automation tool",
	}
	
	if desc, ok := commonExplanations[parts[0]]; ok {
		explanation += "Description: " + desc
	}
	
	return explanation, nil
}

// getCurrentDirectory safely gets the current working directory
func getCurrentDirectory() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	return dir
}

// GetLastSuggestions returns the last generated suggestions
func (sm *SuggestManager) GetLastSuggestions() []CommandSuggestion {
	return sm.lastSuggestions
}

// ClearCache clears any cached data (simplified version has no cache)
func (sm *SuggestManager) ClearCache() {
	// No cache in simplified version
}