package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// UpdateHistory manages comprehensive update history tracking
type UpdateHistory struct {
	mutex       sync.RWMutex
	historyPath string
	records     []*UpdateRecord
	metrics     *UpdateMetrics
}

// UpdateRecord represents a detailed record of an update operation
type UpdateRecord struct {
	ID                string                 `json:"id"`
	Timestamp         time.Time              `json:"timestamp"`
	Type              UpdateType             `json:"type"`
	FromVersion       string                 `json:"from_version"`
	ToVersion         string                 `json:"to_version"`
	Status            UpdateStatus           `json:"status"`
	Duration          time.Duration          `json:"duration"`
	DownloadSize      int64                  `json:"download_size"`
	DownloadTime      time.Duration          `json:"download_time"`
	InstallTime       time.Duration          `json:"install_time"`
	BackupPath        string                 `json:"backup_path,omitempty"`
	ErrorMessage      string                 `json:"error_message,omitempty"`
	Channel           string                 `json:"channel"`
	IsPrerelease      bool                   `json:"is_prerelease"`
	TriggerMethod     TriggerMethod          `json:"trigger_method"`
	UserID            string                 `json:"user_id,omitempty"`
	SystemInfo        *SystemInfo            `json:"system_info"`
	ValidationResults []*ValidationResult    `json:"validation_results,omitempty"`
	PerformanceData   *PerformanceData       `json:"performance_data"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateType represents the type of update operation
type UpdateType string

const (
	UpdateTypeManual    UpdateType = "manual"
	UpdateTypeScheduled UpdateType = "scheduled"
	UpdateTypeAutomatic UpdateType = "automatic"
	UpdateTypeRollback  UpdateType = "rollback"
)

// UpdateStatus represents the outcome of an update operation
type UpdateStatus string

const (
	UpdateStatusSuccess UpdateStatus = "success"
	UpdateStatusFailed  UpdateStatus = "failed"
	UpdateStatusPartial UpdateStatus = "partial"
	UpdateStatusRolledBack UpdateStatus = "rolled_back"
)

// TriggerMethod represents how the update was initiated
type TriggerMethod string

const (
	TriggerMethodCLI         TriggerMethod = "cli"
	TriggerMethodInteractive TriggerMethod = "interactive"
	TriggerMethodScheduled   TriggerMethod = "scheduled"
	TriggerMethodStartup     TriggerMethod = "startup"
	TriggerMethodAPI         TriggerMethod = "api"
)

// SystemInfo captures system information at update time
type SystemInfo struct {
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
	Platform     string `json:"platform"`
	Hostname     string `json:"hostname,omitempty"`
	Username     string `json:"username,omitempty"`
	WorkingDir   string `json:"working_dir,omitempty"`
	ShellType    string `json:"shell_type,omitempty"`
}

// ValidationResult represents post-update validation results
type ValidationResult struct {
	TestName    string        `json:"test_name"`
	Status      string        `json:"status"`
	Duration    time.Duration `json:"duration"`
	ErrorMsg    string        `json:"error_msg,omitempty"`
	Details     interface{}   `json:"details,omitempty"`
}

// PerformanceData captures performance metrics during update
type PerformanceData struct {
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage int64   `json:"memory_usage"`
	DiskUsage   int64   `json:"disk_usage"`
	NetworkIO   int64   `json:"network_io"`
	DiskIO      int64   `json:"disk_io"`
}

// UpdateMetrics tracks aggregated update statistics
type UpdateMetrics struct {
	TotalUpdates      int     `json:"total_updates"`
	SuccessfulUpdates int     `json:"successful_updates"`
	FailedUpdates     int     `json:"failed_updates"`
	SuccessRate       float64 `json:"success_rate"`
	AverageDownloadTime time.Duration `json:"average_download_time"`
	AverageInstallTime  time.Duration `json:"average_install_time"`
	TotalDownloadSize   int64   `json:"total_download_size"`
	LastUpdateTime      time.Time `json:"last_update_time"`
	FirstUpdateTime     time.Time `json:"first_update_time"`
}

// Global update history instance
var globalUpdateHistory *UpdateHistory
var historyOnce sync.Once

// GetUpdateHistory returns the global UpdateHistory instance
func GetUpdateHistory() *UpdateHistory {
	historyOnce.Do(func() {
		// Get config directory
		homeDir, _ := os.UserHomeDir()
		configDir := filepath.Join(homeDir, ".config", "delta")
		historyPath := filepath.Join(configDir, "update_history.json")
		
		globalUpdateHistory = NewUpdateHistory(historyPath)
		globalUpdateHistory.LoadHistory()
	})
	return globalUpdateHistory
}

// NewUpdateHistory creates a new update history manager
func NewUpdateHistory(historyPath string) *UpdateHistory {
	return &UpdateHistory{
		historyPath: historyPath,
		records:     make([]*UpdateRecord, 0),
		metrics:     &UpdateMetrics{},
	}
}

// LoadHistory loads update history from disk
func (uh *UpdateHistory) LoadHistory() error {
	uh.mutex.Lock()
	defer uh.mutex.Unlock()

	// Check if history file exists
	if _, err := os.Stat(uh.historyPath); os.IsNotExist(err) {
		return nil // No history file yet
	}

	// Read history file
	data, err := os.ReadFile(uh.historyPath)
	if err != nil {
		return fmt.Errorf("failed to read history file: %v", err)
	}

	// Parse JSON
	var historyData struct {
		Records []*UpdateRecord `json:"records"`
		Metrics *UpdateMetrics  `json:"metrics"`
	}

	if err := json.Unmarshal(data, &historyData); err != nil {
		return fmt.Errorf("failed to parse history file: %v", err)
	}

	uh.records = historyData.Records
	if historyData.Metrics != nil {
		uh.metrics = historyData.Metrics
	}

	return nil
}

// SaveHistory saves update history to disk
func (uh *UpdateHistory) SaveHistory() error {
	uh.mutex.RLock()
	defer uh.mutex.RUnlock()

	// Create directory if it doesn't exist
	dir := filepath.Dir(uh.historyPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create history directory: %v", err)
	}

	// Prepare data structure
	historyData := struct {
		Records []*UpdateRecord `json:"records"`
		Metrics *UpdateMetrics  `json:"metrics"`
		SavedAt time.Time       `json:"saved_at"`
	}{
		Records: uh.records,
		Metrics: uh.metrics,
		SavedAt: time.Now(),
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(historyData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal history data: %v", err)
	}

	// Write to file
	if err := os.WriteFile(uh.historyPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write history file: %v", err)
	}

	return nil
}

// RecordUpdate records a new update operation
func (uh *UpdateHistory) RecordUpdate(record *UpdateRecord) error {
	uh.mutex.Lock()
	defer uh.mutex.Unlock()

	// Generate ID if not provided
	if record.ID == "" {
		record.ID = fmt.Sprintf("update_%d", time.Now().Unix())
	}

	// Set timestamp if not provided
	if record.Timestamp.IsZero() {
		record.Timestamp = time.Now()
	}

	// Collect system info if not provided
	if record.SystemInfo == nil {
		record.SystemInfo = uh.collectSystemInfo()
	}

	// Add to records
	uh.records = append(uh.records, record)

	// Update metrics
	uh.updateMetrics(record)

	// Save to disk
	return uh.SaveHistory()
}

// collectSystemInfo gathers current system information
func (uh *UpdateHistory) collectSystemInfo() *SystemInfo {
	hostname, _ := os.Hostname()
	username := os.Getenv("USER")
	workingDir, _ := os.Getwd()
	shellType := os.Getenv("SHELL")

	return &SystemInfo{
		OS:           GetCurrentOS(),
		Architecture: GetCurrentArchitecture(),
		Platform:     GetCurrentPlatform(),
		Hostname:     hostname,
		Username:     username,
		WorkingDir:   workingDir,
		ShellType:    shellType,
	}
}

// updateMetrics updates aggregated metrics based on new record
func (uh *UpdateHistory) updateMetrics(record *UpdateRecord) {
	uh.metrics.TotalUpdates++

	if record.Status == UpdateStatusSuccess {
		uh.metrics.SuccessfulUpdates++
	} else {
		uh.metrics.FailedUpdates++
	}

	// Calculate success rate
	uh.metrics.SuccessRate = float64(uh.metrics.SuccessfulUpdates) / float64(uh.metrics.TotalUpdates) * 100

	// Update timing averages
	if record.DownloadTime > 0 {
		totalRecords := len(uh.records)
		if totalRecords == 1 {
			uh.metrics.AverageDownloadTime = record.DownloadTime
		} else {
			// Running average
			currentAvg := uh.metrics.AverageDownloadTime
			uh.metrics.AverageDownloadTime = time.Duration(
				(int64(currentAvg)*(int64(totalRecords)-1) + int64(record.DownloadTime)) / int64(totalRecords),
			)
		}
	}

	if record.InstallTime > 0 {
		totalRecords := len(uh.records)
		if totalRecords == 1 {
			uh.metrics.AverageInstallTime = record.InstallTime
		} else {
			currentAvg := uh.metrics.AverageInstallTime
			uh.metrics.AverageInstallTime = time.Duration(
				(int64(currentAvg)*(int64(totalRecords)-1) + int64(record.InstallTime)) / int64(totalRecords),
			)
		}
	}

	// Update download size
	uh.metrics.TotalDownloadSize += record.DownloadSize

	// Update timestamps
	uh.metrics.LastUpdateTime = record.Timestamp
	if uh.metrics.FirstUpdateTime.IsZero() || record.Timestamp.Before(uh.metrics.FirstUpdateTime) {
		uh.metrics.FirstUpdateTime = record.Timestamp
	}
}

// GetRecords returns all update records, optionally filtered
func (uh *UpdateHistory) GetRecords(filter *HistoryFilter) []*UpdateRecord {
	uh.mutex.RLock()
	defer uh.mutex.RUnlock()

	records := make([]*UpdateRecord, 0, len(uh.records))

	for _, record := range uh.records {
		if filter == nil || filter.Matches(record) {
			records = append(records, record)
		}
	}

	// Sort by timestamp (newest first)
	sort.Slice(records, func(i, j int) bool {
		return records[i].Timestamp.After(records[j].Timestamp)
	})

	return records
}

// HistoryFilter provides filtering options for update records
type HistoryFilter struct {
	Status       *UpdateStatus `json:"status,omitempty"`
	Type         *UpdateType   `json:"type,omitempty"`
	Channel      string        `json:"channel,omitempty"`
	FromVersion  string        `json:"from_version,omitempty"`
	ToVersion    string        `json:"to_version,omitempty"`
	After        *time.Time    `json:"after,omitempty"`
	Before       *time.Time    `json:"before,omitempty"`
	Limit        int           `json:"limit,omitempty"`
}

// Matches checks if a record matches the filter criteria
func (hf *HistoryFilter) Matches(record *UpdateRecord) bool {
	if hf.Status != nil && record.Status != *hf.Status {
		return false
	}
	if hf.Type != nil && record.Type != *hf.Type {
		return false
	}
	if hf.Channel != "" && record.Channel != hf.Channel {
		return false
	}
	if hf.FromVersion != "" && record.FromVersion != hf.FromVersion {
		return false
	}
	if hf.ToVersion != "" && record.ToVersion != hf.ToVersion {
		return false
	}
	if hf.After != nil && record.Timestamp.Before(*hf.After) {
		return false
	}
	if hf.Before != nil && record.Timestamp.After(*hf.Before) {
		return false
	}
	return true
}

// GetMetrics returns current aggregated metrics
func (uh *UpdateHistory) GetMetrics() *UpdateMetrics {
	uh.mutex.RLock()
	defer uh.mutex.RUnlock()
	
	// Return a copy to prevent external modification
	metricsCopy := *uh.metrics
	return &metricsCopy
}

// GetAuditTrail returns a formatted audit trail for compliance
func (uh *UpdateHistory) GetAuditTrail(format AuditFormat) (string, error) {
	uh.mutex.RLock()
	defer uh.mutex.RUnlock()

	switch format {
	case AuditFormatJSON:
		data, err := json.MarshalIndent(uh.records, "", "  ")
		return string(data), err
	case AuditFormatCSV:
		return uh.generateCSVAudit()
	case AuditFormatText:
		return uh.generateTextAudit()
	default:
		return "", fmt.Errorf("unsupported audit format: %v", format)
	}
}

// AuditFormat represents different audit trail formats
type AuditFormat string

const (
	AuditFormatJSON AuditFormat = "json"
	AuditFormatCSV  AuditFormat = "csv"
	AuditFormatText AuditFormat = "text"
)

// generateCSVAudit creates a CSV format audit trail
func (uh *UpdateHistory) generateCSVAudit() (string, error) {
	csv := "ID,Timestamp,Type,FromVersion,ToVersion,Status,Duration,Channel,TriggerMethod,ErrorMessage\n"
	
	for _, record := range uh.records {
		errorMsg := ""
		if record.ErrorMessage != "" {
			errorMsg = fmt.Sprintf("\"%s\"", record.ErrorMessage)
		}
		
		csv += fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%s,%s,%s\n",
			record.ID,
			record.Timestamp.Format("2006-01-02 15:04:05"),
			record.Type,
			record.FromVersion,
			record.ToVersion,
			record.Status,
			record.Duration.String(),
			record.Channel,
			record.TriggerMethod,
			errorMsg,
		)
	}
	
	return csv, nil
}

// generateTextAudit creates a human-readable text audit trail
func (uh *UpdateHistory) generateTextAudit() (string, error) {
	text := "Delta CLI Update Audit Trail\n"
	text += "============================\n\n"
	
	for _, record := range uh.records {
		text += fmt.Sprintf("Update ID: %s\n", record.ID)
		text += fmt.Sprintf("Timestamp: %s\n", record.Timestamp.Format("2006-01-02 15:04:05"))
		text += fmt.Sprintf("Type: %s\n", record.Type)
		text += fmt.Sprintf("Version Change: %s → %s\n", record.FromVersion, record.ToVersion)
		text += fmt.Sprintf("Status: %s\n", record.Status)
		text += fmt.Sprintf("Duration: %s\n", record.Duration.String())
		text += fmt.Sprintf("Channel: %s\n", record.Channel)
		text += fmt.Sprintf("Trigger Method: %s\n", record.TriggerMethod)
		
		if record.ErrorMessage != "" {
			text += fmt.Sprintf("Error: %s\n", record.ErrorMessage)
		}
		
		text += "\n"
	}
	
	return text, nil
}

// CleanupOldRecords removes records older than the specified duration
func (uh *UpdateHistory) CleanupOldRecords(maxAge time.Duration) (int, error) {
	uh.mutex.Lock()
	defer uh.mutex.Unlock()

	cutoff := time.Now().Add(-maxAge)
	var keptRecords []*UpdateRecord
	removedCount := 0

	for _, record := range uh.records {
		if record.Timestamp.After(cutoff) {
			keptRecords = append(keptRecords, record)
		} else {
			removedCount++
		}
	}

	uh.records = keptRecords

	// Recalculate metrics
	uh.recalculateMetrics()

	return removedCount, uh.SaveHistory()
}

// recalculateMetrics recalculates all metrics from scratch
func (uh *UpdateHistory) recalculateMetrics() {
	uh.metrics = &UpdateMetrics{}

	for _, record := range uh.records {
		uh.updateMetrics(record)
	}
}

// Helper functions for system information
func GetCurrentOS() string {
	return "linux" // TODO: Use runtime.GOOS or similar
}

func GetCurrentArchitecture() string {
	return "amd64" // TODO: Use runtime.GOARCH or similar  
}

func GetCurrentPlatform() string {
	return fmt.Sprintf("%s/%s", GetCurrentOS(), GetCurrentArchitecture())
}