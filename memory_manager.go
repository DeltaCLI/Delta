package main

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// CommandEntry represents a single command execution with its context
type CommandEntry struct {
	Command     string            `json:"command"`
	Directory   string            `json:"directory"`
	Timestamp   time.Time         `json:"timestamp"`
	ExitCode    int               `json:"exit_code"`
	Duration    int64             `json:"duration_ms"`
	Environment map[string]string `json:"environment,omitempty"` // Selected environment variables
	PrevCommand string            `json:"prev_command,omitempty"`
	NextCommand string            `json:"next_command,omitempty"`
}

// MemoryConfig holds configuration for the memory manager
type MemoryConfig struct {
	Enabled            bool     `json:"enabled"`
	CollectCommands    bool     `json:"collect_commands"`
	MaxEntries         int      `json:"max_entries"`
	StoragePath        string   `json:"storage_path"`
	PrivacyFilter      []string `json:"privacy_filter"`      // Patterns to filter out from stored commands
	CollectEnvironment bool     `json:"collect_environment"` // Whether to collect environment variables
	EnvWhitelist       []string `json:"env_whitelist"`       // Environment variables that are safe to store
	TrainingEnabled    bool     `json:"training_enabled"`    // Whether nightly training is enabled
	ModelPath          string   `json:"model_path"`          // Path to store/load models
}

// MemoryStats contains statistics about the collected data
type MemoryStats struct {
	TotalEntries  int       `json:"total_entries"`
	FirstEntry    time.Time `json:"first_entry"`
	LastEntry     time.Time `json:"last_entry"`
	DiskUsage     int64     `json:"disk_usage"`
	LastTraining  time.Time `json:"last_training"`
	ModelVersions []string  `json:"model_versions"`
}

// MemoryManager handles collecting, storing, and retrieving command history
// for AI training and memory functionality
type MemoryManager struct {
	config         MemoryConfig
	configPath     string
	storagePath    string
	currentShard   string
	prevCommand    string
	shardWriter    *os.File
	shardWriteLock sync.Mutex
	isInitialized  bool
}

// NewMemoryManager creates a new memory manager with default configuration
func NewMemoryManager() *MemoryManager {
	// Set up config directory in user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = os.Getenv("HOME")
	}

	// Use ~/.config/delta directory for config file
	configDir := filepath.Join(homeDir, ".config", "delta")
	err = os.MkdirAll(configDir, 0755)
	if err != nil {
		// Fall back to home directory if we can't create .config/delta
		configDir = homeDir
	}

	configPath := filepath.Join(configDir, "memory_config.json")
	storagePath := filepath.Join(configDir, "memory")

	// Create a default MemoryManager instance
	mm := &MemoryManager{
		config: MemoryConfig{
			Enabled:            false, // Disabled by default
			CollectCommands:    true,
			MaxEntries:         100000,
			StoragePath:        storagePath,
			PrivacyFilter:      []string{"password", "token", "api_key", "secret", "credential"},
			CollectEnvironment: false,
			EnvWhitelist:       []string{"SHELL", "TERM", "PWD", "USER", "HOME", "PATH", "LANG"},
			TrainingEnabled:    false,
			ModelPath:          filepath.Join(storagePath, "models"),
		},
		configPath:     configPath,
		storagePath:    storagePath,
		currentShard:   "",
		shardWriteLock: sync.Mutex{},
		isInitialized:  false,
	}

	// Try to load configuration
	err = mm.loadConfig()
	if err != nil {
		// Save the default configuration if loading fails
		mm.saveConfig()
	}

	return mm
}

// Initialize sets up the memory manager
func (mm *MemoryManager) Initialize() error {
	// Create storage directory if it doesn't exist
	err := os.MkdirAll(mm.config.StoragePath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create storage directory: %v", err)
	}

	// Create models directory if it doesn't exist
	err = os.MkdirAll(mm.config.ModelPath, 0755)
	if err != nil {
		return fmt.Errorf("failed to create models directory: %v", err)
	}

	// Set up the current shard
	today := time.Now().Format("2006-01-02")
	mm.currentShard = filepath.Join(mm.config.StoragePath, "commands_"+today+".bin")

	// Open the shard file for writing (append mode)
	file, err := os.OpenFile(mm.currentShard, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open shard file: %v", err)
	}
	mm.shardWriter = file

	mm.isInitialized = true
	return nil
}

// IsEnabled returns whether the memory system is enabled
func (mm *MemoryManager) IsEnabled() bool {
	return mm.config.Enabled && mm.isInitialized
}

// loadConfig loads the memory manager configuration from disk
func (mm *MemoryManager) loadConfig() error {
	// Check if config file exists
	_, err := os.Stat(mm.configPath)
	if os.IsNotExist(err) {
		return errors.New("config file does not exist")
	}

	// Read the config file
	data, err := os.ReadFile(mm.configPath)
	if err != nil {
		return err
	}

	// Unmarshal the JSON data
	return json.Unmarshal(data, &mm.config)
}

// saveConfig saves the memory manager configuration to disk
func (mm *MemoryManager) saveConfig() error {
	// Marshal the config to JSON with indentation for readability
	data, err := json.MarshalIndent(mm.config, "", "  ")
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(mm.configPath)
	if err = os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Write to file
	return os.WriteFile(mm.configPath, data, 0644)
}

// Enable enables the memory system
func (mm *MemoryManager) Enable() error {
	if mm.isInitialized {
		mm.config.Enabled = true
		return mm.saveConfig()
	}
	return errors.New("memory manager not initialized")
}

// Disable disables the memory system
func (mm *MemoryManager) Disable() error {
	mm.config.Enabled = false
	return mm.saveConfig()
}

// AddCommand records a command execution
func (mm *MemoryManager) AddCommand(command string, directory string, exitCode int, durationMs int64) error {
	if !mm.IsEnabled() || !mm.config.CollectCommands {
		return nil
	}

	// Check for privacy sensitive commands
	for _, pattern := range mm.config.PrivacyFilter {
		if containsInsensitive(command, pattern) {
			// Skip this command for privacy reasons
			return nil
		}
	}

	// Create a new entry
	entry := CommandEntry{
		Command:     command,
		Directory:   directory,
		Timestamp:   time.Now(),
		ExitCode:    exitCode,
		Duration:    durationMs,
		PrevCommand: mm.prevCommand,
	}

	// Collect selected environment variables if enabled
	if mm.config.CollectEnvironment {
		entry.Environment = make(map[string]string)
		for _, envVar := range mm.config.EnvWhitelist {
			if value := os.Getenv(envVar); value != "" {
				entry.Environment[envVar] = value
			}
		}
	}

	// Store the current command as previous for next time
	mm.prevCommand = command

	// Write to the shard file
	return mm.writeCommandToShard(entry)
}

// writeCommandToShard writes a command entry to the current shard file
func (mm *MemoryManager) writeCommandToShard(entry CommandEntry) error {
	// Update shard if day has changed
	today := time.Now().Format("2006-01-02")
	expectedShard := filepath.Join(mm.config.StoragePath, "commands_"+today+".bin")

	mm.shardWriteLock.Lock()
	defer mm.shardWriteLock.Unlock()

	// Check if we need to switch to a new shard file
	if expectedShard != mm.currentShard {
		// Close the current shard file
		if mm.shardWriter != nil {
			mm.shardWriter.Close()
		}

		// Open the new shard file
		file, err := os.OpenFile(expectedShard, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("failed to open new shard file: %v", err)
		}
		mm.shardWriter = file
		mm.currentShard = expectedShard
	}

	// Convert entry to JSON
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	// Write format: [4-byte length][json data]
	lenBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(lenBytes, uint32(len(data)))

	// Write length
	if _, err := mm.shardWriter.Write(lenBytes); err != nil {
		return err
	}

	// Write data
	if _, err := mm.shardWriter.Write(data); err != nil {
		return err
	}

	// Flush to ensure data is written
	return mm.shardWriter.Sync()
}

// GetStats returns statistics about the collected data
func (mm *MemoryManager) GetStats() (MemoryStats, error) {
	stats := MemoryStats{
		TotalEntries: 0,
		FirstEntry:   time.Time{},
		LastEntry:    time.Time{},
		DiskUsage:    0,
	}

	// Check if storage directory exists
	if _, err := os.Stat(mm.config.StoragePath); os.IsNotExist(err) {
		return stats, nil
	}

	// Get list of all shard files
	entries, err := os.ReadDir(mm.config.StoragePath)
	if err != nil {
		return stats, err
	}

	// Process each shard file
	for _, entry := range entries {
		if entry.IsDir() || !filepath.HasPrefix(entry.Name(), "commands_") {
			continue
		}

		// Get file info
		fileInfo, err := entry.Info()
		if err != nil {
			continue
		}

		// Add file size to disk usage
		stats.DiskUsage += fileInfo.Size()

		// Process file to count entries if needed
		shardPath := filepath.Join(mm.config.StoragePath, entry.Name())
		count, first, last, err := mm.getShardStats(shardPath)
		if err != nil {
			continue
		}

		// Update stats
		stats.TotalEntries += count
		if stats.FirstEntry.IsZero() || first.Before(stats.FirstEntry) {
			stats.FirstEntry = first
		}
		if last.After(stats.LastEntry) {
			stats.LastEntry = last
		}
	}

	// Get model versions
	modelEntries, err := os.ReadDir(mm.config.ModelPath)
	if err == nil {
		for _, modelEntry := range modelEntries {
			if modelEntry.IsDir() || !strings.HasSuffix(modelEntry.Name(), ".bin") {
				continue
			}
			stats.ModelVersions = append(stats.ModelVersions, modelEntry.Name())
		}
	}

	// Read last training time
	lastTrainingFile := filepath.Join(mm.config.ModelPath, "last_training.txt")
	if data, err := os.ReadFile(lastTrainingFile); err == nil {
		if t, err := time.Parse(time.RFC3339, string(data)); err == nil {
			stats.LastTraining = t
		}
	}

	return stats, nil
}

// getShardStats processes a shard file to count entries and get timestamps
func (mm *MemoryManager) getShardStats(shardPath string) (int, time.Time, time.Time, error) {
	file, err := os.Open(shardPath)
	if err != nil {
		return 0, time.Time{}, time.Time{}, err
	}
	defer file.Close()

	count := 0
	var firstTimestamp, lastTimestamp time.Time

	// Read file in chunks
	buffer := make([]byte, 4)
	for {
		// Read entry length
		_, err := file.Read(buffer)
		if err != nil {
			break // End of file or error
		}

		// Get the length of the JSON data
		length := binary.LittleEndian.Uint32(buffer)

		// Read the JSON data
		jsonData := make([]byte, length)
		_, err = file.Read(jsonData)
		if err != nil {
			break
		}

		// Parse the JSON data to extract timestamp
		var entry CommandEntry
		if err := json.Unmarshal(jsonData, &entry); err != nil {
			continue
		}

		// Update stats
		count++
		if firstTimestamp.IsZero() || entry.Timestamp.Before(firstTimestamp) {
			firstTimestamp = entry.Timestamp
		}
		if entry.Timestamp.After(lastTimestamp) {
			lastTimestamp = entry.Timestamp
		}
	}

	return count, firstTimestamp, lastTimestamp, nil
}

// ReadCommands reads all commands from a specific date
func (mm *MemoryManager) ReadCommands(date string) ([]CommandEntry, error) {
	var entries []CommandEntry

	// Validate date format
	_, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format (required: YYYY-MM-DD): %v", err)
	}

	// Open the shard file
	shardPath := filepath.Join(mm.config.StoragePath, "commands_"+date+".bin")
	file, err := os.Open(shardPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Read entries
	buffer := make([]byte, 4)
	for {
		// Read entry length
		_, err := file.Read(buffer)
		if err != nil {
			break // End of file or error
		}

		// Get the length of the JSON data
		length := binary.LittleEndian.Uint32(buffer)

		// Read the JSON data
		jsonData := make([]byte, length)
		_, err = file.Read(jsonData)
		if err != nil {
			break
		}

		// Parse the JSON data
		var entry CommandEntry
		if err := json.Unmarshal(jsonData, &entry); err != nil {
			continue
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// ClearData removes all collected data
func (mm *MemoryManager) ClearData() error {
	// Close the current shard file
	mm.shardWriteLock.Lock()
	if mm.shardWriter != nil {
		mm.shardWriter.Close()
		mm.shardWriter = nil
	}
	mm.shardWriteLock.Unlock()

	// Remove all files in the storage directory
	entries, err := os.ReadDir(mm.config.StoragePath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !filepath.HasPrefix(entry.Name(), "commands_") {
			continue
		}
		os.Remove(filepath.Join(mm.config.StoragePath, entry.Name()))
	}

	// Reinitialize
	return mm.Initialize()
}

// UpdateConfig updates the memory manager configuration
func (mm *MemoryManager) UpdateConfig(config MemoryConfig) error {
	mm.config = config
	return mm.saveConfig()
}

// ExportOptions represents options for memory export operation
type ExportOptions struct {
	Format      string    // "json" or "binary"
	StartDate   time.Time // Start date for export range (nil for all)
	EndDate     time.Time // End date for export range (nil for all)
	IncludeAll  bool      // Whether to include configuration in export
	Destination string    // Export destination directory
}

// ExportMetadata contains information about an export
type ExportMetadata struct {
	ExportDate   time.Time               `json:"export_date"`
	StartDate    time.Time               `json:"start_date,omitempty"`
	EndDate      time.Time               `json:"end_date,omitempty"`
	Format       string                  `json:"format"`
	EntryCount   int                     `json:"entry_count"`
	ShardCount   int                     `json:"shard_count"`
	IncludesAll  bool                    `json:"includes_all"`
	Config       *MemoryConfig           `json:"config,omitempty"`
	ShardDetails map[string]ShardDetails `json:"shard_details,omitempty"`
}

// ShardDetails contains information about an exported shard
type ShardDetails struct {
	Date       string `json:"date"`
	EntryCount int    `json:"entry_count"`
	Path       string `json:"path"`
}

// ExportMemory exports memory data according to specified options
func (mm *MemoryManager) ExportMemory(options ExportOptions) (string, error) {
	// Close current shard file to ensure all data is written
	mm.shardWriteLock.Lock()
	if mm.shardWriter != nil {
		mm.shardWriter.Close()
		mm.shardWriter = nil
	}
	mm.shardWriteLock.Unlock()

	// Create export directory
	exportDir := options.Destination
	if exportDir == "" {
		exportDir = filepath.Join(mm.config.StoragePath, "exports")
	}

	err := os.MkdirAll(exportDir, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create export directory: %v", err)
	}

	// Create timestamp-based export directory
	timestamp := time.Now().Format("20060102_150405")
	exportPath := filepath.Join(exportDir, "export_"+timestamp)
	err = os.MkdirAll(exportPath, 0755)
	if err != nil {
		return "", fmt.Errorf("failed to create export subdirectory: %v", err)
	}

	// Create metadata structure
	metadata := ExportMetadata{
		ExportDate:   time.Now(),
		Format:       options.Format,
		EntryCount:   0,
		ShardCount:   0,
		IncludesAll:  options.IncludeAll,
		ShardDetails: make(map[string]ShardDetails),
	}

	// Include dates if specified
	if !options.StartDate.IsZero() {
		metadata.StartDate = options.StartDate
	}
	if !options.EndDate.IsZero() {
		metadata.EndDate = options.EndDate
	}

	// Include configuration if requested
	if options.IncludeAll {
		metadata.Config = &mm.config
	}

	// Get list of shards to export
	entries, err := os.ReadDir(mm.config.StoragePath)
	if err != nil {
		return "", fmt.Errorf("failed to read storage directory: %v", err)
	}

	// Filter and copy shard files
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "commands_") {
			continue
		}

		// Extract date from filename
		datePart := strings.TrimPrefix(entry.Name(), "commands_")
		datePart = strings.TrimSuffix(datePart, ".bin")

		shardDate, err := time.Parse("2006-01-02", datePart)
		if err != nil {
			// Skip files with invalid date format
			continue
		}

		// Apply date range filter if specified
		if !options.StartDate.IsZero() && shardDate.Before(options.StartDate) {
			continue
		}
		if !options.EndDate.IsZero() && shardDate.After(options.EndDate) {
			continue
		}

		// Get shard stats
		shardPath := filepath.Join(mm.config.StoragePath, entry.Name())
		entriesCount, _, _, err := mm.getShardStats(shardPath)
		if err != nil {
			// Skip files with errors
			continue
		}

		// Export based on format
		var exportedPath string
		if options.Format == "json" {
			// Export as JSON
			entries, err := mm.ReadCommands(datePart)
			if err != nil {
				// Skip files with errors
				continue
			}

			// Write to JSON file
			jsonPath := filepath.Join(exportPath, "commands_"+datePart+".json")
			jsonData, err := json.MarshalIndent(entries, "", "  ")
			if err != nil {
				continue
			}

			err = os.WriteFile(jsonPath, jsonData, 0644)
			if err != nil {
				continue
			}

			exportedPath = jsonPath
		} else {
			// Binary format (direct copy)
			dstPath := filepath.Join(exportPath, entry.Name())
			data, err := os.ReadFile(shardPath)
			if err != nil {
				continue
			}

			err = os.WriteFile(dstPath, data, 0644)
			if err != nil {
				continue
			}

			exportedPath = dstPath
		}

		// Update metadata
		metadata.ShardCount++
		metadata.EntryCount += entriesCount
		metadata.ShardDetails[datePart] = ShardDetails{
			Date:       datePart,
			EntryCount: entriesCount,
			Path:       exportedPath,
		}
	}

	// Write metadata file
	metadataPath := filepath.Join(exportPath, "metadata.json")
	metadataJSON, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return exportPath, fmt.Errorf("failed to create metadata: %v", err)
	}

	err = os.WriteFile(metadataPath, metadataJSON, 0644)
	if err != nil {
		return exportPath, fmt.Errorf("failed to write metadata: %v", err)
	}

	// Reinitialize memory manager
	mm.Initialize()

	return exportPath, nil
}

// ImportMemory imports memory data from an export
func (mm *MemoryManager) ImportMemory(importPath string, options map[string]bool) error {
	// Validate import path
	metadataPath := filepath.Join(importPath, "metadata.json")
	_, err := os.Stat(metadataPath)
	if err != nil {
		return fmt.Errorf("invalid import: metadata.json not found at %s", importPath)
	}

	// Read metadata
	metadataBytes, err := os.ReadFile(metadataPath)
	if err != nil {
		return fmt.Errorf("failed to read metadata: %v", err)
	}

	var metadata ExportMetadata
	err = json.Unmarshal(metadataBytes, &metadata)
	if err != nil {
		return fmt.Errorf("failed to parse metadata: %v", err)
	}

	// Close current shard file
	mm.shardWriteLock.Lock()
	if mm.shardWriter != nil {
		mm.shardWriter.Close()
		mm.shardWriter = nil
	}
	mm.shardWriteLock.Unlock()

	// Import configuration if available and requested
	importConfig := options["import_config"]
	if importConfig && metadata.Config != nil {
		// Keep original storage paths but update other settings
		config := *metadata.Config
		config.StoragePath = mm.config.StoragePath
		config.ModelPath = mm.config.ModelPath
		mm.config = config
		mm.saveConfig()
	}

	// Import memory data
	for date, shardDetails := range metadata.ShardDetails {
		// Log info about shard being imported
		fmt.Printf("Importing shard for %s with %d entries\n", shardDetails.Date, shardDetails.EntryCount)

		// Destination shard path
		dstPath := filepath.Join(mm.config.StoragePath, "commands_"+date+".bin")

		if metadata.Format == "json" {
			// Convert JSON to binary format
			jsonPath := filepath.Join(importPath, "commands_"+date+".json")
			jsonData, err := os.ReadFile(jsonPath)
			if err != nil {
				continue // Skip this shard
			}

			var entries []CommandEntry
			err = json.Unmarshal(jsonData, &entries)
			if err != nil {
				continue // Skip this shard
			}

			// Create binary shard file
			file, err := os.Create(dstPath)
			if err != nil {
				continue // Skip this shard
			}

			// Write each entry in binary format
			for _, entry := range entries {
				data, err := json.Marshal(entry)
				if err != nil {
					continue // Skip this entry
				}

				// Write length and data
				lenBytes := make([]byte, 4)
				binary.LittleEndian.PutUint32(lenBytes, uint32(len(data)))
				file.Write(lenBytes)
				file.Write(data)
			}

			file.Close()
		} else {
			// Direct binary copy
			srcPath := filepath.Join(importPath, "commands_"+date+".bin")
			data, err := os.ReadFile(srcPath)
			if err != nil {
				continue // Skip this shard
			}

			err = os.WriteFile(dstPath, data, 0644)
			if err != nil {
				continue // Skip this shard
			}
		}
	}

	// Reinitialize memory manager
	return mm.Initialize()
}

// GetCommandsInRange returns commands within the specified time range
func (mm *MemoryManager) GetCommandsInRange(startTime, endTime time.Time) ([]CommandEntry, error) {
	var allCommands []CommandEntry

	// Get all shard files
	entries, err := os.ReadDir(mm.config.StoragePath)
	if err != nil {
		return nil, err
	}

	// Process each shard that might contain commands in the range
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "commands_") {
			continue
		}

		// Extract date from filename
		dateStr := strings.TrimPrefix(entry.Name(), "commands_")
		dateStr = strings.TrimSuffix(dateStr, ".bin")
		
		shardDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		// Check if this shard might contain relevant commands
		shardEnd := shardDate.Add(24 * time.Hour)
		if shardEnd.Before(startTime) || shardDate.After(endTime) {
			continue
		}

		// Read commands from this shard
		commands, err := mm.ReadCommands(dateStr)
		if err != nil {
			continue
		}

		// Filter commands by time range
		for _, cmd := range commands {
			if cmd.Timestamp.After(startTime) && cmd.Timestamp.Before(endTime) {
				allCommands = append(allCommands, cmd)
			}
		}
	}

	// Sort by timestamp
	sort.Slice(allCommands, func(i, j int) bool {
		return allCommands[i].Timestamp.Before(allCommands[j].Timestamp)
	})

	return allCommands, nil
}

// Close closes the memory manager and releases resources
func (mm *MemoryManager) Close() error {
	mm.shardWriteLock.Lock()
	defer mm.shardWriteLock.Unlock()

	if mm.shardWriter != nil {
		err := mm.shardWriter.Close()
		mm.shardWriter = nil
		return err
	}
	return nil
}

// containsInsensitive checks if a string contains a substring (case insensitive)
func containsInsensitive(s, substr string) bool {
	s, substr = strings.ToLower(s), strings.ToLower(substr)
	return strings.Contains(s, substr)
}

// Global MemoryManager instance
var globalMemoryManager *MemoryManager

// GetMemoryManager returns the global MemoryManager instance
func GetMemoryManager() *MemoryManager {
	if globalMemoryManager == nil {
		globalMemoryManager = NewMemoryManager()
	}
	return globalMemoryManager
}
