package main

import (
	"os"
	"path/filepath"
	"strings"
)

// ShellHistoryFile represents a detected shell history file
type ShellHistoryFile struct {
	Path     string
	Shell    string
	Format   string
	Size     int64
	Readable bool
}

// ShellHistoryDetector handles detection of shell history files
type ShellHistoryDetector struct {
	homeDir string
}

// NewShellHistoryDetector creates a new detector instance
func NewShellHistoryDetector() (*ShellHistoryDetector, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	
	return &ShellHistoryDetector{
		homeDir: homeDir,
	}, nil
}

// DetectHistoryFiles scans for shell history files in the user's home directory
func (d *ShellHistoryDetector) DetectHistoryFiles() ([]ShellHistoryFile, error) {
	var historyFiles []ShellHistoryFile
	
	// Common history file patterns
	candidates := []struct {
		filename string
		shell    string
		format   string
	}{
		{".bash_history", "bash", "bash"},
		{".zsh_history", "zsh", "zsh"},
		{".history", "unknown", "simple"},
		{".sh_history", "sh", "simple"},
	}
	
	for _, candidate := range candidates {
		filePath := filepath.Join(d.homeDir, candidate.filename)
		if info, err := os.Stat(filePath); err == nil {
			readable := d.isFileReadable(filePath)
			
			historyFile := ShellHistoryFile{
				Path:     filePath,
				Shell:    candidate.shell,
				Format:   candidate.format,
				Size:     info.Size(),
				Readable: readable,
			}
			
			historyFiles = append(historyFiles, historyFile)
		}
	}
	
	// Check for custom HISTFILE locations from shell configs
	customFiles := d.detectCustomHistFiles()
	historyFiles = append(historyFiles, customFiles...)
	
	return historyFiles, nil
}

// isFileReadable checks if a file can be read by the current user
func (d *ShellHistoryDetector) isFileReadable(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer file.Close()
	
	// Try to read a small amount to verify readability
	buffer := make([]byte, 64)
	_, err = file.Read(buffer)
	return err == nil
}

// detectCustomHistFiles looks for custom HISTFILE settings in shell configs
func (d *ShellHistoryDetector) detectCustomHistFiles() []ShellHistoryFile {
	var customFiles []ShellHistoryFile
	
	// Shell config files to check
	configFiles := []string{
		".bashrc",
		".bash_profile",
		".zshrc",
		".profile",
	}
	
	for _, configFile := range configFiles {
		configPath := filepath.Join(d.homeDir, configFile)
		if histFile := d.extractHistFileFromConfig(configPath); histFile != "" {
			// Convert relative paths to absolute
			if !filepath.IsAbs(histFile) {
				histFile = filepath.Join(d.homeDir, histFile)
			}
			
			if info, err := os.Stat(histFile); err == nil {
				readable := d.isFileReadable(histFile)
				
				// Determine format based on file extension or content
				format := d.detectHistoryFormat(histFile)
				shell := d.detectShellFromConfig(configFile)
				
				customFile := ShellHistoryFile{
					Path:     histFile,
					Shell:    shell,
					Format:   format,
					Size:     info.Size(),
					Readable: readable,
				}
				
				customFiles = append(customFiles, customFile)
			}
		}
	}
	
	return customFiles
}

// extractHistFileFromConfig parses shell config for HISTFILE settings
func (d *ShellHistoryDetector) extractHistFileFromConfig(configPath string) string {
	content, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}
	
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Look for HISTFILE=path patterns
		if strings.HasPrefix(line, "HISTFILE=") {
			histFile := strings.TrimPrefix(line, "HISTFILE=")
			histFile = strings.Trim(histFile, "\"'")
			
			// Expand environment variables
			if strings.Contains(histFile, "$HOME") {
				histFile = strings.Replace(histFile, "$HOME", d.homeDir, -1)
			}
			if strings.Contains(histFile, "~") {
				histFile = strings.Replace(histFile, "~", d.homeDir, -1)
			}
			
			return histFile
		}
		
		// Look for export HISTFILE=path patterns
		if strings.HasPrefix(line, "export HISTFILE=") {
			histFile := strings.TrimPrefix(line, "export HISTFILE=")
			histFile = strings.Trim(histFile, "\"'")
			
			// Expand environment variables
			if strings.Contains(histFile, "$HOME") {
				histFile = strings.Replace(histFile, "$HOME", d.homeDir, -1)
			}
			if strings.Contains(histFile, "~") {
				histFile = strings.Replace(histFile, "~", d.homeDir, -1)
			}
			
			return histFile
		}
	}
	
	return ""
}

// detectHistoryFormat determines the format of a history file
func (d *ShellHistoryDetector) detectHistoryFormat(filePath string) string {
	// Check file extension first
	if strings.HasSuffix(filePath, ".zsh_history") || strings.Contains(filePath, "zsh") {
		return "zsh"
	}
	if strings.HasSuffix(filePath, ".bash_history") || strings.Contains(filePath, "bash") {
		return "bash"
	}
	
	// Check file content to determine format
	file, err := os.Open(filePath)
	if err != nil {
		return "simple"
	}
	defer file.Close()
	
	// Read first few lines to detect format
	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil {
		return "simple"
	}
	
	content := string(buffer[:n])
	lines := strings.Split(content, "\n")
	
	// Look for zsh extended history format (starts with : timestamp:duration;command)
	for _, line := range lines {
		if strings.HasPrefix(line, ":") && strings.Contains(line, ";") {
			parts := strings.SplitN(line, ";", 2)
			if len(parts) == 2 && strings.Contains(parts[0], ":") {
				return "zsh"
			}
		}
	}
	
	return "bash"
}

// detectShellFromConfig determines shell type from config filename
func (d *ShellHistoryDetector) detectShellFromConfig(configFile string) string {
	if strings.Contains(configFile, "bash") {
		return "bash"
	}
	if strings.Contains(configFile, "zsh") {
		return "zsh"
	}
	return "unknown"
}

// GetHistoryStats returns summary statistics about detected history files
func (d *ShellHistoryDetector) GetHistoryStats(files []ShellHistoryFile) map[string]interface{} {
	stats := make(map[string]interface{})
	
	totalFiles := len(files)
	readableFiles := 0
	totalSize := int64(0)
	shellCounts := make(map[string]int)
	
	for _, file := range files {
		if file.Readable {
			readableFiles++
		}
		totalSize += file.Size
		shellCounts[file.Shell]++
	}
	
	stats["total_files"] = totalFiles
	stats["readable_files"] = readableFiles
	stats["total_size_bytes"] = totalSize
	stats["shells"] = shellCounts
	
	return stats
}