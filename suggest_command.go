package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// SuggestCommandInfo extends CommandSuggestion with additional metadata
type SuggestCommandInfo struct {
	CommandSuggestion
	Description string   // Extended description
	Category    string   // Category like "file", "git", "docker"
	Safety      string   // safe, caution, dangerous
	Examples    []string // Example usage
}

// NewSuggestInfo creates a new SuggestCommandInfo
func NewSuggestInfo(cmd string, desc string, conf float64, cat string) SuggestCommandInfo {
	return SuggestCommandInfo{
		CommandSuggestion: CommandSuggestion{
			Command:    cmd,
			Confidence: conf,
			Reason:     desc,
		},
		Description: desc,
		Category:    cat,
		Safety:      "safe",
	}
}

// NewSuggestInfoWithSafety creates a new SuggestCommandInfo with safety level
func NewSuggestInfoWithSafety(cmd string, desc string, conf float64, cat string, safety string) SuggestCommandInfo {
	return SuggestCommandInfo{
		CommandSuggestion: CommandSuggestion{
			Command:    cmd,
			Confidence: conf,
			Reason:     desc,
		},
		Description: desc,
		Category:    cat,
		Safety:      safety,
	}
}

// SuggestManager handles natural language command suggestions
type SuggestManager struct {
	aiManager        *AIPredictionManager
	historyAnalyzer  *HistoryAnalyzer
	// validator removed - not implemented yet
	contextCache     map[string][]SuggestCommandInfo
	lastQuery        string
	lastSuggestions  []SuggestCommandInfo
}

// NewSuggestManager creates a new suggestion manager
func NewSuggestManager() *SuggestManager {
	return &SuggestManager{
		contextCache: make(map[string][]SuggestCommandInfo),
	}
}

// Initialize sets up the suggest manager with dependencies
func (sm *SuggestManager) Initialize() error {
	sm.aiManager = GetAIManager()
	sm.historyAnalyzer = GetHistoryAnalyzer()
	// sm.validator = GetCommandValidator() // Not implemented yet
	return nil
}

// GetSuggestions generates command suggestions from natural language input
func (sm *SuggestManager) GetSuggestions(query string, limit int) ([]SuggestCommandInfo, error) {
	// Check cache first
	if cached, ok := sm.contextCache[query]; ok && len(cached) > 0 {
		return cached, nil
	}
	
	suggestions := []SuggestCommandInfo{}
	
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
	
	// Deduplicate and sort by confidence
	suggestions = sm.deduplicateAndSort(suggestions)
	
	// Limit results
	if len(suggestions) > limit {
		suggestions = suggestions[:limit]
	}
	
	// Validate suggestions for safety
	for i := range suggestions {
		sm.validateSuggestion(&suggestions[i])
	}
	
	// Cache results
	sm.contextCache[query] = suggestions
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
func (sm *SuggestManager) getPatternBasedSuggestions(query string, projectType string) []SuggestCommandInfo {
	suggestions := []SuggestCommandInfo{}
	queryLower := strings.ToLower(query)
	
	// Common patterns and their suggestions
	patterns := []struct {
		keywords    []string
		suggestions []SuggestCommandInfo
	}{
		// File operations
		{
			keywords: []string{"list", "show", "files", "directory", "folders"},
			suggestions: []SuggestCommandInfo{
				NewSuggestInfo("ls -la", "List all files with details", 0.9, "file"),
				NewSuggestInfo("ls -lh", "List files with human-readable sizes", 0.8, "file"),
				NewSuggestInfo("tree", "Show directory tree structure", 0.7, "file"),
			},
		},
		{
			keywords: []string{"find", "search", "locate", "where"},
			suggestions: []SuggestCommandInfo{
				NewSuggestInfo("find . -name \"*pattern*\"", "Find files by name pattern", 0.9, "search"),
				NewSuggestInfo("grep -r \"text\" .", "Search for text in files", 0.8, "search"),
				NewSuggestInfo("rg \"pattern\"", "Fast search with ripgrep", 0.8, "search"),
			},
		},
		{
			keywords: []string{"copy", "cp", "duplicate"},
			suggestions: []SuggestCommandInfo{
				NewSuggestInfo("cp source destination", "Copy file or directory", 0.9, "file"),
				NewSuggestInfo("cp -r source/ destination/", "Copy directory recursively", 0.8, "file"),
			},
		},
		{
			keywords: []string{"move", "rename", "mv"},
			suggestions: []SuggestCommandInfo{
				NewSuggestInfo("mv oldname newname", "Move or rename file", 0.9, "file"),
			},
		},
		{
			keywords: []string{"delete", "remove", "rm", "clean"},
			suggestions: []SuggestCommandInfo{
				NewSuggestInfoWithSafety("rm filename", "Remove file", 0.9, "file", "caution"),
				NewSuggestInfoWithSafety("rm -rf directory/", "Remove directory recursively", 0.8, "file", "dangerous"),
			},
		},
		// Git operations
		{
			keywords: []string{"git", "commit", "save", "changes"},
			suggestions: []SuggestCommandInfo{
				NewSuggestInfo("git add .", "Stage all changes", 0.8, "git"),
				NewSuggestInfo("git commit -m \"message\"", "Commit with message", 0.9, "git"),
				NewSuggestInfo("git status", "Check repository status", 0.9, "git"),
			},
		},
		{
			keywords: []string{"branch", "checkout", "switch"},
			suggestions: []SuggestCommandInfo{
				NewSuggestInfo("git branch", "List branches", 0.8, "git"),
				NewSuggestInfo("git checkout -b new-branch", "Create and switch to new branch", 0.9, "git"),
				NewSuggestInfo("git switch branch-name", "Switch to existing branch", 0.8, "git"),
			},
		},
		{
			keywords: []string{"push", "upload", "publish"},
			suggestions: []SuggestCommandInfo{
				NewSuggestInfo("git push origin main", "Push to main branch", 0.9, "git"),
				NewSuggestInfo("git push -u origin branch-name", "Push and set upstream", 0.8, "git"),
			},
		},
		{
			keywords: []string{"pull", "download", "fetch", "update"},
			suggestions: []SuggestCommandInfo{
				NewSuggestInfo("git pull", "Pull latest changes", 0.9, "git"),
				NewSuggestInfo("git fetch --all", "Fetch all remote branches", 0.8, "git"),
			},
		},
		// Process management
		{
			keywords: []string{"process", "running", "ps", "tasks"},
			suggestions: []SuggestCommandInfo{
				NewSuggestInfo("ps aux", "Show all running processes", 0.9, "system"),
				NewSuggestInfo("top", "Interactive process viewer", 0.8, "system"),
				NewSuggestInfo("htop", "Enhanced process viewer", 0.8, "system"),
			},
		},
		{
			keywords: []string{"kill", "stop", "terminate"},
			suggestions: []SuggestCommandInfo{
				NewSuggestInfoWithSafety("kill PID", "Terminate process by ID", 0.8, "system", "caution"),
				NewSuggestInfoWithSafety("killall process-name", "Kill all processes by name", 0.7, "system", "caution"),
			},
		},
		// Network operations
		{
			keywords: []string{"network", "connection", "port", "listen"},
			suggestions: []SuggestCommandInfo{
				NewSuggestInfo("netstat -tlnp", "Show listening ports", 0.8, "network"),
				NewSuggestInfo("ss -tlnp", "Modern socket statistics", 0.8, "network"),
				NewSuggestInfo("lsof -i :PORT", "Check what's using a port", 0.7, "network"),
			},
		},
		{
			keywords: []string{"download", "wget", "curl", "fetch"},
			suggestions: []SuggestCommandInfo{
				NewSuggestInfo("curl -O URL", "Download file from URL", 0.9, "network"),
				NewSuggestInfo("wget URL", "Download with wget", 0.8, "network"),
			},
		},
		// Docker operations
		{
			keywords: []string{"docker", "container", "image"},
			suggestions: []SuggestCommandInfo{
				NewSuggestInfo("docker ps", "List running containers", 0.9, "docker"),
				NewSuggestInfo("docker images", "List docker images", 0.8, "docker"),
				NewSuggestInfo("docker-compose up", "Start services", 0.8, "docker"),
			},
		},
		// Package management based on project type
		{
			keywords: []string{"install", "package", "dependency"},
			suggestions: sm.getPackageManagerSuggestions(projectType),
		},
		{
			keywords: []string{"build", "compile", "make"},
			suggestions: sm.getBuildSuggestions(projectType),
		},
		{
			keywords: []string{"test", "check", "verify"},
			suggestions: sm.getTestSuggestions(projectType),
		},
		{
			keywords: []string{"run", "start", "execute", "launch"},
			suggestions: sm.getRunSuggestions(projectType),
		},
	}
	
	// Check which patterns match the query
	for _, pattern := range patterns {
		matched := false
		for _, keyword := range pattern.keywords {
			if strings.Contains(queryLower, keyword) {
				matched = true
				break
			}
		}
		
		if matched {
			for _, suggestion := range pattern.suggestions {
				// Adjust confidence based on keyword match strength
				suggestion.Confidence *= sm.calculateMatchStrength(query, pattern.keywords)
				suggestions = append(suggestions, suggestion)
			}
		}
	}
	
	return suggestions
}

// getPackageManagerSuggestions returns package manager commands based on project type
func (sm *SuggestManager) getPackageManagerSuggestions(projectType string) []SuggestCommandInfo {
	switch projectType {
	case "nodejs":
		return []SuggestCommandInfo{
			NewSuggestInfo("npm install", "Install all dependencies", 0.9, "package"),
			NewSuggestInfo("npm install package-name", "Install specific package", 0.8, "package"),
			NewSuggestInfo("npm update", "Update dependencies", 0.7, "package"),
		}
	case "golang":
		return []SuggestCommandInfo{
			NewSuggestInfo("go mod download", "Download dependencies", 0.9, "package"),
			NewSuggestInfo("go get package-name", "Install specific package", 0.8, "package"),
			NewSuggestInfo("go mod tidy", "Clean up dependencies", 0.7, "package"),
		}
	case "python":
		return []SuggestCommandInfo{
			NewSuggestInfo("pip install -r requirements.txt", "Install from requirements", 0.9, "package"),
			NewSuggestInfo("pip install package-name", "Install specific package", 0.8, "package"),
			NewSuggestInfo("pipenv install", "Install with pipenv", 0.7, "package"),
		}
	case "rust":
		return []SuggestCommandInfo{
			NewSuggestInfo("cargo build", "Build dependencies", 0.9, "package"),
			NewSuggestInfo("cargo add package-name", "Add dependency", 0.8, "package"),
		}
	case "java":
		return []SuggestCommandInfo{
			NewSuggestInfo("mvn install", "Install Maven dependencies", 0.8, "package"),
			NewSuggestInfo("gradle build", "Build with Gradle", 0.8, "package"),
		}
	default:
		return []SuggestCommandInfo{
			NewSuggestInfo("make install", "Run make install", 0.6, "package"),
		}
	}
}

// getBuildSuggestions returns build commands based on project type
func (sm *SuggestManager) getBuildSuggestions(projectType string) []SuggestCommandInfo {
	switch projectType {
	case "nodejs":
		return []SuggestCommandInfo{
			NewSuggestInfo("npm run build", "Build the project", 0.9, "build"),
			NewSuggestInfo("npm run dev", "Run development build", 0.8, "build"),
		}
	case "golang":
		return []SuggestCommandInfo{
			NewSuggestInfo("go build", "Build Go project", 0.9, "build"),
			NewSuggestInfo("go build -o output", "Build with output name", 0.8, "build"),
		}
	case "rust":
		return []SuggestCommandInfo{
			NewSuggestInfo("cargo build", "Build Rust project", 0.9, "build"),
			NewSuggestInfo("cargo build --release", "Build optimized", 0.8, "build"),
		}
	case "make":
		return []SuggestCommandInfo{
			NewSuggestInfo("make", "Run default make target", 0.9, "build"),
			NewSuggestInfo("make build", "Run build target", 0.8, "build"),
			NewSuggestInfo("make clean", "Clean build artifacts", 0.7, "build"),
		}
	default:
		return []SuggestCommandInfo{
			NewSuggestInfo("make", "Try make build", 0.5, "build"),
		}
	}
}

// getTestSuggestions returns test commands based on project type
func (sm *SuggestManager) getTestSuggestions(projectType string) []SuggestCommandInfo {
	switch projectType {
	case "nodejs":
		return []SuggestCommandInfo{
			NewSuggestInfo("npm test", "Run tests", 0.9, "test"),
			NewSuggestInfo("npm run test:watch", "Run tests in watch mode", 0.8, "test"),
		}
	case "golang":
		return []SuggestCommandInfo{
			NewSuggestInfo("go test ./...", "Run all tests", 0.9, "test"),
			NewSuggestInfo("go test -v", "Run tests verbosely", 0.8, "test"),
		}
	case "python":
		return []SuggestCommandInfo{
			NewSuggestInfo("pytest", "Run tests with pytest", 0.9, "test"),
			NewSuggestInfo("python -m unittest", "Run unittest", 0.8, "test"),
		}
	case "rust":
		return []SuggestCommandInfo{
			NewSuggestInfo("cargo test", "Run Rust tests", 0.9, "test"),
		}
	default:
		return []SuggestCommandInfo{
			NewSuggestInfo("make test", "Run make test", 0.6, "test"),
		}
	}
}

// getRunSuggestions returns run commands based on project type
func (sm *SuggestManager) getRunSuggestions(projectType string) []SuggestCommandInfo {
	switch projectType {
	case "nodejs":
		return []SuggestCommandInfo{
			NewSuggestInfo("npm start", "Start the application", 0.9, "run"),
			NewSuggestInfo("node index.js", "Run with node", 0.8, "run"),
			NewSuggestInfo("npm run dev", "Run in development mode", 0.8, "run"),
		}
	case "golang":
		return []SuggestCommandInfo{
			NewSuggestInfo("go run .", "Run Go application", 0.9, "run"),
			NewSuggestInfo("go run main.go", "Run main.go", 0.8, "run"),
		}
	case "python":
		return []SuggestCommandInfo{
			NewSuggestInfo("python main.py", "Run Python script", 0.8, "run"),
			NewSuggestInfo("python -m module", "Run as module", 0.7, "run"),
		}
	case "rust":
		return []SuggestCommandInfo{
			NewSuggestInfo("cargo run", "Run Rust application", 0.9, "run"),
		}
	case "docker":
		return []SuggestCommandInfo{
			NewSuggestInfo("docker-compose up", "Start Docker services", 0.9, "run"),
			NewSuggestInfo("docker run image", "Run Docker container", 0.8, "run"),
		}
	default:
		return []SuggestCommandInfo{
			NewSuggestInfo("./run.sh", "Run shell script", 0.5, "run"),
			NewSuggestInfo("make run", "Run make target", 0.5, "run"),
		}
	}
}

// calculateMatchStrength calculates how well keywords match the query
func (sm *SuggestManager) calculateMatchStrength(query string, keywords []string) float64 {
	queryLower := strings.ToLower(query)
	matchCount := 0
	
	for _, keyword := range keywords {
		if strings.Contains(queryLower, keyword) {
			matchCount++
		}
	}
	
	return float64(matchCount) / float64(len(keywords))
}

// getAISuggestions uses AI to generate command suggestions
func (sm *SuggestManager) getAISuggestions(query string, projectType string) ([]SuggestCommandInfo, error) {
	if sm.aiManager == nil || !sm.aiManager.IsEnabled() {
		return []SuggestCommandInfo{}, nil
	}
	
	// Create a prompt for the AI
	prompt := fmt.Sprintf(`Given the user wants to: "%s"
Current project type: %s
Current directory: %s

Suggest 3 shell commands that would accomplish this task. For each command:
1. Provide the exact command
2. Brief description of what it does
3. Mark safety as: safe, caution, or dangerous

Format each suggestion as:
COMMAND: <command>
DESC: <description>
SAFETY: <safety level>
---`, query, projectType, filepath.Base(getCurrentDirectory()))

	response, err := sm.aiManager.ollamaClient.Generate(prompt, "You are a command-line expert assistant. Provide practical, safe command suggestions.")
	if err != nil {
		return []SuggestCommandInfo{}, err
	}
	
	// Parse AI response
	suggestions := sm.parseAIResponse(response)
	
	// Set confidence based on AI response
	for i := range suggestions {
		suggestions[i].Confidence = 0.85 - (float64(i) * 0.1) // Decreasing confidence
		suggestions[i].Category = "ai-suggested"
	}
	
	return suggestions, nil
}

// parseAIResponse parses the AI response into command suggestions
func (sm *SuggestManager) parseAIResponse(response string) []SuggestCommandInfo {
	suggestions := []SuggestCommandInfo{}
	
	// Split by separator
	parts := strings.Split(response, "---")
	
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		
		var cmd, desc, safety string
		safety = "safe" // Default
		
		lines := strings.Split(part, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			
			if strings.HasPrefix(line, "COMMAND:") {
				cmd = strings.TrimSpace(strings.TrimPrefix(line, "COMMAND:"))
			} else if strings.HasPrefix(line, "DESC:") {
				desc = strings.TrimSpace(strings.TrimPrefix(line, "DESC:"))
			} else if strings.HasPrefix(line, "SAFETY:") {
				s := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(line, "SAFETY:")))
				if s == "caution" || s == "dangerous" {
					safety = s
				}
			}
		}
		
		if cmd != "" {
			suggestion := NewSuggestInfoWithSafety(cmd, desc, 0.7, "ai-suggested", safety)
			suggestions = append(suggestions, suggestion)
		}
	}
	
	return suggestions
}

// getHistoryBasedSuggestions uses command history to suggest commands
func (sm *SuggestManager) getHistoryBasedSuggestions(query string, currentDir string) []SuggestCommandInfo {
	if sm.historyAnalyzer == nil || !sm.historyAnalyzer.IsEnabled() {
		return []SuggestCommandInfo{}
	}
	
	suggestions := []SuggestCommandInfo{}
	queryLower := strings.ToLower(query)
	
	// Get history entries by searching with empty query
	history := sm.historyAnalyzer.SearchHistory("", 100)
	
	// Score commands based on relevance
	type scoredCommand struct {
		command string
		score   float64
		count   int
	}
	
	commandScores := make(map[string]*scoredCommand)
	
	for _, entry := range history {
		// Skip if command doesn't contain any query keywords
		cmdLower := strings.ToLower(entry.Command)
		
		// Calculate relevance score
		score := 0.0
		
		// Check for keyword matches
		words := strings.Fields(queryLower)
		matchedWords := 0
		for _, word := range words {
			if strings.Contains(cmdLower, word) {
				matchedWords++
			}
		}
		
		if matchedWords > 0 {
			score = float64(matchedWords) / float64(len(words))
			
			// Boost score for recent commands
			age := time.Since(entry.Context.Timestamp)
			if age < 1*time.Hour {
				score *= 1.5
			} else if age < 24*time.Hour {
				score *= 1.2
			}
			
			// Boost score for commands in same directory
			if entry.Context.Directory == currentDir {
				score *= 1.3
			}
			
			// Update or create score entry
			if existing, ok := commandScores[entry.Command]; ok {
				existing.score += score
				existing.count++
			} else {
				commandScores[entry.Command] = &scoredCommand{
					command: entry.Command,
					score:   score,
					count:   1,
				}
			}
		}
	}
	
	// Convert to sorted list
	var scoredList []scoredCommand
	for _, sc := range commandScores {
		// Boost score by frequency
		sc.score *= (1.0 + float64(sc.count)*0.1)
		scoredList = append(scoredList, *sc)
	}
	
	// Sort by score
	sort.Slice(scoredList, func(i, j int) bool {
		return scoredList[i].score > scoredList[j].score
	})
	
	// Convert to suggestions
	for i, sc := range scoredList {
		if i >= 5 { // Limit history suggestions
			break
		}
		
		suggestion := SuggestCommandInfo{
			CommandSuggestion: CommandSuggestion{
				Command:    sc.command,
				Confidence: sc.score * 0.7, // Scale down confidence
				Reason:     fmt.Sprintf("Previously used (%d times)", sc.count),
			},
			Description: fmt.Sprintf("Previously used (%d times)", sc.count),
			Category:    "history",
			Safety:      "safe", // Assume previously used commands are safe
		}
		
		suggestions = append(suggestions, suggestion)
	}
	
	return suggestions
}

// validateSuggestion checks command safety
func (sm *SuggestManager) validateSuggestion(suggestion *SuggestCommandInfo) {
	// Basic safety validation without CommandValidator
	// Check for dangerous patterns
	dangerousPatterns := []string{
		"rm -rf /",
		"dd if=",
		"mkfs",
		":(){ :|:& };", // Fork bomb
		"> /dev/sda",
	}
	
	for _, pattern := range dangerousPatterns {
		if strings.Contains(suggestion.Command, pattern) {
			suggestion.Safety = "dangerous"
			suggestion.Description += " (DANGEROUS COMMAND!)"
			return
		}
	}
	
	// Check for caution patterns
	cautionPatterns := []string{
		"sudo",
		"rm -rf",
		"kill",
		"pkill",
		"chmod 777",
	}
	
	for _, pattern := range cautionPatterns {
		if strings.Contains(suggestion.Command, pattern) && suggestion.Safety != "dangerous" {
			suggestion.Safety = "caution"
		}
	}
}

// deduplicateAndSort removes duplicate suggestions and sorts by confidence
func (sm *SuggestManager) deduplicateAndSort(suggestions []SuggestCommandInfo) []SuggestCommandInfo {
	// Use map to track unique commands
	unique := make(map[string]SuggestCommandInfo)
	
	for _, suggestion := range suggestions {
		key := suggestion.Command
		
		// Keep the suggestion with highest confidence
		if existing, ok := unique[key]; !ok || suggestion.Confidence > existing.Confidence {
			unique[key] = suggestion
		}
	}
	
	// Convert back to slice
	result := []SuggestCommandInfo{}
	for _, suggestion := range unique {
		result = append(result, suggestion)
	}
	
	// Sort by confidence (highest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Confidence > result[j].Confidence
	})
	
	return result
}

// ExplainCommand provides detailed explanation of a command
func (sm *SuggestManager) ExplainCommand(command string) (string, error) {
	if sm.aiManager != nil && sm.aiManager.IsEnabled() {
		prompt := fmt.Sprintf(`Explain this command in detail: %s

Include:
1. What the command does
2. Each flag/option explanation
3. Common use cases
4. Potential risks or warnings
5. Example variations`, command)

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
func (sm *SuggestManager) GetLastSuggestions() []SuggestCommandInfo {
	return sm.lastSuggestions
}

// ClearCache clears the suggestion cache
func (sm *SuggestManager) ClearCache() {
	sm.contextCache = make(map[string][]SuggestCommandInfo)
}