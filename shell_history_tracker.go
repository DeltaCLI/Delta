package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// ImportRecord tracks when and what was imported from shell history files
type ImportRecord struct {
	FilePath         string    `json:"file_path"`
	LastImported     time.Time `json:"last_imported"`
	FileSize         int64     `json:"file_size"`
	FileModTime      time.Time `json:"file_mod_time"`
	FileChecksum     string    `json:"file_checksum"`
	CommandsImported int       `json:"commands_imported"`
	ImportedRanges   []ImportRange `json:"imported_ranges"`
}

// ImportRange tracks specific ranges of commands that were imported
type ImportRange struct {
	StartOffset int       `json:"start_offset"`
	EndOffset   int       `json:"end_offset"`
	StartTime   time.Time `json:"start_time,omitempty"`
	EndTime     time.Time `json:"end_time,omitempty"`
	Count       int       `json:"count"`
}

// ShellHistoryTracker manages tracking of imported shell history
type ShellHistoryTracker struct {
	configDir   string
	recordsFile string
	records     map[string]ImportRecord
}

// NewShellHistoryTracker creates a new history import tracker
func NewShellHistoryTracker() (*ShellHistoryTracker, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", "delta", "history_import")
	recordsFile := filepath.Join(configDir, "import_records.json")

	tracker := &ShellHistoryTracker{
		configDir:   configDir,
		recordsFile: recordsFile,
		records:     make(map[string]ImportRecord),
	}

	// Create config directory if it doesn't exist
	err = os.MkdirAll(configDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Load existing records
	err = tracker.loadRecords()
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to load import records: %w", err)
	}

	return tracker, nil
}

// loadRecords loads import records from the JSON file
func (t *ShellHistoryTracker) loadRecords() error {
	data, err := os.ReadFile(t.recordsFile)
	if err != nil {
		return err
	}

	if len(data) == 0 {
		return nil // Empty file
	}

	return json.Unmarshal(data, &t.records)
}

// saveRecords saves import records to the JSON file
func (t *ShellHistoryTracker) saveRecords() error {
	data, err := json.MarshalIndent(t.records, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal records: %w", err)
	}

	return os.WriteFile(t.recordsFile, data, 0644)
}

// calculateFileChecksum calculates SHA256 checksum of a file
func (t *ShellHistoryTracker) calculateFileChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	_, err = io.Copy(hash, file)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// getFileInfo gets file size and modification time
func (t *ShellHistoryTracker) getFileInfo(filePath string) (int64, time.Time, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0, time.Time{}, err
	}

	return info.Size(), info.ModTime(), nil
}

// HasBeenImported checks if a file has been imported recently
func (t *ShellHistoryTracker) HasBeenImported(filePath string) (bool, *ImportRecord, error) {
	// Get current file info
	size, modTime, err := t.getFileInfo(filePath)
	if err != nil {
		return false, nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Check if we have a record for this file
	record, exists := t.records[filePath]
	if !exists {
		return false, nil, nil
	}

	// Check if file has been modified since last import
	if modTime.After(record.FileModTime) || size != record.FileSize {
		return false, &record, nil
	}

	// File hasn't changed since last import
	return true, &record, nil
}

// GetNewCommandsOnly returns only commands that haven't been imported before
func (t *ShellHistoryTracker) GetNewCommandsOnly(filePath string, entries []HistoryEntry) ([]HistoryEntry, error) {
	imported, record, err := t.HasBeenImported(filePath)
	if err != nil {
		return nil, err
	}

	if !imported || record == nil {
		// No previous import, return all entries
		return entries, nil
	}

	// Calculate checksum to see if content has changed
	currentChecksum, err := t.calculateFileChecksum(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %w", err)
	}

	if currentChecksum == record.FileChecksum {
		// File content is identical, no new commands
		return []HistoryEntry{}, nil
	}

	// File has changed, try to find new commands
	// For now, use a simple heuristic: if file is larger, assume new commands were appended
	currentSize, _, err := t.getFileInfo(filePath)
	if err != nil {
		return nil, err
	}

	if currentSize > record.FileSize {
		// File grew, likely new commands appended
		// Return commands that would be beyond the previously imported range
		if len(record.ImportedRanges) > 0 {
			lastRange := record.ImportedRanges[len(record.ImportedRanges)-1]
			
			// Simple approach: if we have more entries than were previously imported,
			// return the excess entries
			if len(entries) > lastRange.Count {
				newEntries := entries[lastRange.Count:]
				return newEntries, nil
			}
		}
	}

	// Fallback: return all entries and let deduplication handle it
	return entries, nil
}

// RecordImport records a successful import operation
func (t *ShellHistoryTracker) RecordImport(filePath string, entries []HistoryEntry, importedCount int) error {
	// Get current file info
	size, modTime, err := t.getFileInfo(filePath)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}

	// Calculate checksum
	checksum, err := t.calculateFileChecksum(filePath)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	// Create import range
	importRange := ImportRange{
		StartOffset: 0,
		EndOffset:   len(entries),
		Count:       importedCount,
	}

	// Set time range if available
	if len(entries) > 0 {
		// Find earliest and latest timestamps
		var earliest, latest time.Time
		for _, entry := range entries {
			if !entry.Timestamp.IsZero() {
				if earliest.IsZero() || entry.Timestamp.Before(earliest) {
					earliest = entry.Timestamp
				}
				if latest.IsZero() || entry.Timestamp.After(latest) {
					latest = entry.Timestamp
				}
			}
		}
		importRange.StartTime = earliest
		importRange.EndTime = latest
	}

	// Update or create record
	existingRecord, exists := t.records[filePath]
	if exists {
		// Update existing record
		existingRecord.LastImported = time.Now()
		existingRecord.FileSize = size
		existingRecord.FileModTime = modTime
		existingRecord.FileChecksum = checksum
		existingRecord.CommandsImported += importedCount
		existingRecord.ImportedRanges = append(existingRecord.ImportedRanges, importRange)
		t.records[filePath] = existingRecord
	} else {
		// Create new record
		t.records[filePath] = ImportRecord{
			FilePath:         filePath,
			LastImported:     time.Now(),
			FileSize:         size,
			FileModTime:      modTime,
			FileChecksum:     checksum,
			CommandsImported: importedCount,
			ImportedRanges:   []ImportRange{importRange},
		}
	}

	// Save records to disk
	return t.saveRecords()
}

// GetImportSummary returns a summary of all imports
func (t *ShellHistoryTracker) GetImportSummary() map[string]interface{} {
	totalFiles := len(t.records)
	totalCommands := 0
	
	recentImports := make([]ImportRecord, 0)
	oldestImport := time.Now()
	newestImport := time.Time{}

	for _, record := range t.records {
		totalCommands += record.CommandsImported
		
		if record.LastImported.Before(oldestImport) {
			oldestImport = record.LastImported
		}
		if record.LastImported.After(newestImport) {
			newestImport = record.LastImported
		}

		// Include imports from last 7 days
		weekAgo := time.Now().AddDate(0, 0, -7)
		if record.LastImported.After(weekAgo) {
			recentImports = append(recentImports, record)
		}
	}

	summary := map[string]interface{}{
		"total_files_imported":     totalFiles,
		"total_commands_imported":  totalCommands,
		"recent_imports_count":     len(recentImports),
	}

	if totalFiles > 0 {
		summary["oldest_import"] = oldestImport.Format(time.RFC3339)
		summary["newest_import"] = newestImport.Format(time.RFC3339)
		summary["recent_imports"] = recentImports
	}

	return summary
}

// CheckForUpdates checks if any tracked files have been updated
func (t *ShellHistoryTracker) CheckForUpdates() ([]string, error) {
	var updatedFiles []string

	for filePath := range t.records {
		imported, _, err := t.HasBeenImported(filePath)
		if err != nil {
			continue // Skip files that can't be accessed
		}

		if !imported {
			updatedFiles = append(updatedFiles, filePath)
		}
	}

	return updatedFiles, nil
}

// ResetImportRecord removes the import record for a specific file
func (t *ShellHistoryTracker) ResetImportRecord(filePath string) error {
	delete(t.records, filePath)
	return t.saveRecords()
}

// ClearAllRecords removes all import records
func (t *ShellHistoryTracker) ClearAllRecords() error {
	t.records = make(map[string]ImportRecord)
	return t.saveRecords()
}