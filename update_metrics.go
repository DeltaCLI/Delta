package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// MetricType represents different types of metrics
type MetricType string

const (
	MetricUpdateCheck      MetricType = "update_check"
	MetricUpdateDownload   MetricType = "update_download"
	MetricUpdateInstall    MetricType = "update_install"
	MetricUpdateRollback   MetricType = "update_rollback"
	MetricChannelSwitch    MetricType = "channel_switch"
	MetricUpdateSkip       MetricType = "update_skip"
	MetricUpdatePostpone   MetricType = "update_postpone"
	MetricValidationFailed MetricType = "validation_failed"
	MetricSystemHealth     MetricType = "system_health"
)

// MetricEvent represents a single metric event
type MetricEvent struct {
	ID          string                 `json:"id"`
	Type        MetricType             `json:"type"`
	Timestamp   time.Time              `json:"timestamp"`
	Duration    time.Duration          `json:"duration,omitempty"`
	Success     bool                   `json:"success"`
	Version     string                 `json:"version,omitempty"`
	FromVersion string                 `json:"from_version,omitempty"`
	ToVersion   string                 `json:"to_version,omitempty"`
	Channel     string                 `json:"channel,omitempty"`
	Size        int64                  `json:"size,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	SystemInfo  *SystemMetrics         `json:"system_info,omitempty"`
}

// SystemMetrics captures system state during metric events
type SystemMetrics struct {
	Platform       string  `json:"platform"`
	Architecture   string  `json:"architecture"`
	CPUCount       int     `json:"cpu_count"`
	MemoryTotal    uint64  `json:"memory_total"`
	MemoryUsed     uint64  `json:"memory_used"`
	DiskTotal      uint64  `json:"disk_total"`
	DiskAvailable  uint64  `json:"disk_available"`
	NetworkLatency float64 `json:"network_latency_ms,omitempty"`
	LoadAverage    float64 `json:"load_average,omitempty"`
}

// AggregatedMetrics represents aggregated statistics
type AggregatedMetrics struct {
	Period           string                       `json:"period"`
	StartTime        time.Time                    `json:"start_time"`
	EndTime          time.Time                    `json:"end_time"`
	TotalEvents      int                          `json:"total_events"`
	EventsByType     map[MetricType]int           `json:"events_by_type"`
	SuccessRate      float64                      `json:"success_rate"`
	ChannelMetrics   map[string]*ChannelMetrics   `json:"channel_metrics"`
	VersionMetrics   map[string]*VersionMetrics   `json:"version_metrics"`
	PerformanceStats *PerformanceStats            `json:"performance_stats"`
	ErrorAnalysis    *ErrorAnalysis               `json:"error_analysis"`
}

// ChannelMetrics represents metrics for a specific channel
type ChannelMetrics struct {
	Channel         string        `json:"channel"`
	TotalUpdates    int           `json:"total_updates"`
	SuccessfulUpdates int         `json:"successful_updates"`
	FailedUpdates   int           `json:"failed_updates"`
	AverageDownloadTime time.Duration `json:"avg_download_time"`
	AverageInstallTime  time.Duration `json:"avg_install_time"`
	TotalDownloadSize   int64         `json:"total_download_size"`
}

// VersionMetrics represents metrics for a specific version
type VersionMetrics struct {
	Version           string        `json:"version"`
	InstallCount      int           `json:"install_count"`
	RollbackCount     int           `json:"rollback_count"`
	SkipCount         int           `json:"skip_count"`
	PostponeCount     int           `json:"postpone_count"`
	AverageInstallTime time.Duration `json:"avg_install_time"`
	SuccessRate       float64       `json:"success_rate"`
}

// PerformanceStats represents performance statistics
type PerformanceStats struct {
	AverageDownloadSpeed   float64       `json:"avg_download_speed_mbps"`
	AverageInstallDuration time.Duration `json:"avg_install_duration"`
	FastestDownload        time.Duration `json:"fastest_download"`
	SlowestDownload        time.Duration `json:"slowest_download"`
	PeakDownloadHour       int           `json:"peak_download_hour"`
	NetworkReliability     float64       `json:"network_reliability"`
}

// ErrorAnalysis represents error pattern analysis
type ErrorAnalysis struct {
	TotalErrors      int                    `json:"total_errors"`
	ErrorsByType     map[string]int         `json:"errors_by_type"`
	CommonErrors     []ErrorPattern         `json:"common_errors"`
	ErrorTrends      map[string][]int       `json:"error_trends"` // Daily counts
	RecoveryRate     float64                `json:"recovery_rate"`
}

// ErrorPattern represents a common error pattern
type ErrorPattern struct {
	Pattern     string  `json:"pattern"`
	Count       int     `json:"count"`
	Percentage  float64 `json:"percentage"`
	LastSeen    time.Time `json:"last_seen"`
	Suggestion  string  `json:"suggestion,omitempty"`
}

// UpdateMetricsManager manages update system metrics
type UpdateMetricsManager struct {
	configPath   string
	dataPath     string
	events       []MetricEvent
	mutex        sync.RWMutex
	flushTicker  *time.Ticker
	retentionDays int
	isEnabled    bool
}

// MetricsConfig represents metrics configuration
type MetricsConfig struct {
	Enabled         bool          `json:"enabled"`
	RetentionDays   int           `json:"retention_days"`
	FlushInterval   time.Duration `json:"flush_interval"`
	ExportFormats   []string      `json:"export_formats"`
	AnonymizeData   bool          `json:"anonymize_data"`
	CollectSystemInfo bool        `json:"collect_system_info"`
}

// NewUpdateMetrics creates a new metrics manager
func NewUpdateMetrics() (*UpdateMetricsManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %v", err)
	}

	metricsDir := filepath.Join(homeDir, ".config", "delta", "metrics")
	if err := os.MkdirAll(metricsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create metrics directory: %v", err)
	}

	um := &UpdateMetricsManager{
		configPath:    filepath.Join(metricsDir, "metrics_config.json"),
		dataPath:      filepath.Join(metricsDir, "events.json"),
		events:        make([]MetricEvent, 0),
		retentionDays: 30,
		isEnabled:     true,
	}

	// Load configuration
	if err := um.loadConfig(); err != nil {
		// Use defaults if loading fails
		um.saveConfig()
	}

	// Load existing events
	if err := um.loadEvents(); err != nil {
		// Start fresh if loading fails
		um.events = make([]MetricEvent, 0)
	}

	// Start flush ticker
	um.startFlushTicker(5 * time.Minute)

	return um, nil
}

// RecordEvent records a new metric event
func (um *UpdateMetricsManager) RecordEvent(event MetricEvent) error {
	if !um.isEnabled {
		return nil
	}

	um.mutex.Lock()
	defer um.mutex.Unlock()

	// Generate ID if not provided
	if event.ID == "" {
		event.ID = fmt.Sprintf("%s_%d", event.Type, time.Now().UnixNano())
	}

	// Set timestamp if not provided
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Collect system info if enabled
	if um.shouldCollectSystemInfo() {
		event.SystemInfo = um.collectSystemInfo()
	}

	um.events = append(um.events, event)

	// Clean old events based on retention
	um.cleanOldEvents()

	return nil
}

// RecordUpdateCheck records an update check event
func (um *UpdateMetricsManager) RecordUpdateCheck(version string, hasUpdate bool, duration time.Duration) {
	event := MetricEvent{
		Type:      MetricUpdateCheck,
		Version:   version,
		Success:   true,
		Duration:  duration,
		Metadata: map[string]interface{}{
			"has_update": hasUpdate,
		},
	}
	um.RecordEvent(event)
}

// RecordUpdateDownload records a download event
func (um *UpdateMetricsManager) RecordUpdateDownload(version string, size int64, duration time.Duration, success bool, err error) {
	event := MetricEvent{
		Type:     MetricUpdateDownload,
		Version:  version,
		Success:  success,
		Duration: duration,
		Size:     size,
	}
	
	if err != nil {
		event.Error = err.Error()
	}
	
	um.RecordEvent(event)
}

// RecordUpdateInstall records an installation event
func (um *UpdateMetricsManager) RecordUpdateInstall(fromVersion, toVersion string, duration time.Duration, success bool, err error) {
	event := MetricEvent{
		Type:        MetricUpdateInstall,
		FromVersion: fromVersion,
		ToVersion:   toVersion,
		Success:     success,
		Duration:    duration,
	}
	
	if err != nil {
		event.Error = err.Error()
	}
	
	um.RecordEvent(event)
}

// RecordChannelSwitch records a channel switch event
func (um *UpdateMetricsManager) RecordChannelSwitch(fromChannel, toChannel, reason string) {
	event := MetricEvent{
		Type:    MetricChannelSwitch,
		Success: true,
		Metadata: map[string]interface{}{
			"from_channel": fromChannel,
			"to_channel":   toChannel,
			"reason":       reason,
		},
	}
	um.RecordEvent(event)
}

// GetAggregatedMetrics returns aggregated metrics for a time period
func (um *UpdateMetricsManager) GetAggregatedMetrics(startTime, endTime time.Time) (*AggregatedMetrics, error) {
	um.mutex.RLock()
	defer um.mutex.RUnlock()

	agg := &AggregatedMetrics{
		Period:         fmt.Sprintf("%s to %s", startTime.Format("2006-01-02"), endTime.Format("2006-01-02")),
		StartTime:      startTime,
		EndTime:        endTime,
		EventsByType:   make(map[MetricType]int),
		ChannelMetrics: make(map[string]*ChannelMetrics),
		VersionMetrics: make(map[string]*VersionMetrics),
		PerformanceStats: &PerformanceStats{},
		ErrorAnalysis:   &ErrorAnalysis{
			ErrorsByType: make(map[string]int),
			ErrorTrends:  make(map[string][]int),
		},
	}

	// Process events within time range
	successCount := 0
	for _, event := range um.events {
		if event.Timestamp.Before(startTime) || event.Timestamp.After(endTime) {
			continue
		}

		agg.TotalEvents++
		agg.EventsByType[event.Type]++

		if event.Success {
			successCount++
		}

		// Process by type
		switch event.Type {
		case MetricUpdateDownload, MetricUpdateInstall:
			um.processUpdateMetrics(&event, agg)
		case MetricChannelSwitch:
			um.processChannelMetrics(&event, agg)
		}

		// Process errors
		if event.Error != "" {
			um.processErrorMetrics(&event, agg)
		}
	}

	// Calculate success rate
	if agg.TotalEvents > 0 {
		agg.SuccessRate = float64(successCount) / float64(agg.TotalEvents) * 100
	}

	// Calculate performance stats
	um.calculatePerformanceStats(agg)

	return agg, nil
}

// ExportMetrics exports metrics in various formats
func (um *UpdateMetricsManager) ExportMetrics(format string, startTime, endTime time.Time) ([]byte, error) {
	agg, err := um.GetAggregatedMetrics(startTime, endTime)
	if err != nil {
		return nil, err
	}

	switch format {
	case "json":
		return json.MarshalIndent(agg, "", "  ")
	case "csv":
		return um.exportCSV(agg)
	case "prometheus":
		return um.exportPrometheus(agg)
	default:
		return nil, fmt.Errorf("unsupported export format: %s", format)
	}
}

// GetMetricsSummary returns a summary of current metrics
func (um *UpdateMetricsManager) GetMetricsSummary() map[string]interface{} {
	um.mutex.RLock()
	defer um.mutex.RUnlock()

	// Get last 7 days metrics
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -7)
	
	agg, _ := um.GetAggregatedMetrics(startTime, endTime)

	summary := map[string]interface{}{
		"total_events_7d":     agg.TotalEvents,
		"success_rate_7d":     agg.SuccessRate,
		"total_events_all":    len(um.events),
		"metrics_enabled":     um.isEnabled,
		"retention_days":      um.retentionDays,
		"oldest_event":        um.getOldestEventTime(),
		"newest_event":        um.getNewestEventTime(),
		"events_by_type":      agg.EventsByType,
	}

	return summary
}

// Helper methods

func (um *UpdateMetricsManager) collectSystemInfo() *SystemMetrics {
	// This would collect actual system metrics
	// For now, return placeholder data
	return &SystemMetrics{
		Platform:     "linux",
		Architecture: "amd64",
		CPUCount:     4,
		MemoryTotal:  8 * 1024 * 1024 * 1024, // 8GB
		MemoryUsed:   4 * 1024 * 1024 * 1024, // 4GB
		DiskTotal:    100 * 1024 * 1024 * 1024, // 100GB
		DiskAvailable: 50 * 1024 * 1024 * 1024, // 50GB
	}
}

func (um *UpdateMetricsManager) processUpdateMetrics(event *MetricEvent, agg *AggregatedMetrics) {
	// Process version metrics
	if event.ToVersion != "" {
		if vm, exists := agg.VersionMetrics[event.ToVersion]; exists {
			if event.Type == MetricUpdateInstall && event.Success {
				vm.InstallCount++
			}
		} else {
			agg.VersionMetrics[event.ToVersion] = &VersionMetrics{
				Version:      event.ToVersion,
				InstallCount: 1,
			}
		}
	}
}

func (um *UpdateMetricsManager) processChannelMetrics(event *MetricEvent, agg *AggregatedMetrics) {
	if event.Metadata != nil {
		if channel, ok := event.Metadata["to_channel"].(string); ok {
			if cm, exists := agg.ChannelMetrics[channel]; exists {
				cm.TotalUpdates++
			} else {
				agg.ChannelMetrics[channel] = &ChannelMetrics{
					Channel:      channel,
					TotalUpdates: 1,
				}
			}
		}
	}
}

func (um *UpdateMetricsManager) processErrorMetrics(event *MetricEvent, agg *AggregatedMetrics) {
	agg.ErrorAnalysis.TotalErrors++
	
	// Categorize error
	errorType := "unknown"
	if strings.Contains(event.Error, "network") || strings.Contains(event.Error, "connection") {
		errorType = "network"
	} else if strings.Contains(event.Error, "permission") || strings.Contains(event.Error, "access") {
		errorType = "permission"
	} else if strings.Contains(event.Error, "space") || strings.Contains(event.Error, "disk") {
		errorType = "disk_space"
	} else if strings.Contains(event.Error, "timeout") {
		errorType = "timeout"
	}
	
	agg.ErrorAnalysis.ErrorsByType[errorType]++
}

func (um *UpdateMetricsManager) calculatePerformanceStats(agg *AggregatedMetrics) {
	var totalDownloadTime time.Duration
	var downloadCount int
	
	for _, event := range um.events {
		if event.Type == MetricUpdateDownload && event.Success {
			totalDownloadTime += event.Duration
			downloadCount++
			
			if agg.PerformanceStats.FastestDownload == 0 || event.Duration < agg.PerformanceStats.FastestDownload {
				agg.PerformanceStats.FastestDownload = event.Duration
			}
			if event.Duration > agg.PerformanceStats.SlowestDownload {
				agg.PerformanceStats.SlowestDownload = event.Duration
			}
		}
	}
	
	if downloadCount > 0 {
		avgDuration := totalDownloadTime / time.Duration(downloadCount)
		agg.PerformanceStats.AverageInstallDuration = avgDuration
	}
}

func (um *UpdateMetricsManager) cleanOldEvents() {
	cutoffTime := time.Now().AddDate(0, 0, -um.retentionDays)
	
	newEvents := make([]MetricEvent, 0)
	for _, event := range um.events {
		if event.Timestamp.After(cutoffTime) {
			newEvents = append(newEvents, event)
		}
	}
	
	um.events = newEvents
}

func (um *UpdateMetricsManager) startFlushTicker(interval time.Duration) {
	um.flushTicker = time.NewTicker(interval)
	go func() {
		for range um.flushTicker.C {
			um.flush()
		}
	}()
}

func (um *UpdateMetricsManager) flush() error {
	um.mutex.Lock()
	defer um.mutex.Unlock()
	
	return um.saveEvents()
}

func (um *UpdateMetricsManager) loadConfig() error {
	data, err := os.ReadFile(um.configPath)
	if err != nil {
		return err
	}
	
	var config MetricsConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return err
	}
	
	um.isEnabled = config.Enabled
	um.retentionDays = config.RetentionDays
	
	return nil
}

func (um *UpdateMetricsManager) saveConfig() error {
	config := MetricsConfig{
		Enabled:       um.isEnabled,
		RetentionDays: um.retentionDays,
		FlushInterval: 5 * time.Minute,
		ExportFormats: []string{"json", "csv", "prometheus"},
		AnonymizeData: false,
		CollectSystemInfo: true,
	}
	
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(um.configPath, data, 0644)
}

func (um *UpdateMetricsManager) loadEvents() error {
	data, err := os.ReadFile(um.dataPath)
	if err != nil {
		return err
	}
	
	return json.Unmarshal(data, &um.events)
}

func (um *UpdateMetricsManager) saveEvents() error {
	data, err := json.MarshalIndent(um.events, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(um.dataPath, data, 0644)
}

func (um *UpdateMetricsManager) exportCSV(agg *AggregatedMetrics) ([]byte, error) {
	// Simple CSV export implementation
	csv := "Period,Total Events,Success Rate,Total Errors\n"
	csv += fmt.Sprintf("%s,%d,%.2f%%,%d\n", 
		agg.Period, 
		agg.TotalEvents, 
		agg.SuccessRate,
		agg.ErrorAnalysis.TotalErrors,
	)
	return []byte(csv), nil
}

func (um *UpdateMetricsManager) exportPrometheus(agg *AggregatedMetrics) ([]byte, error) {
	// Prometheus format export
	metrics := fmt.Sprintf(`# HELP delta_update_total Total number of update events
# TYPE delta_update_total counter
delta_update_total %d

# HELP delta_update_success_rate Update success rate percentage
# TYPE delta_update_success_rate gauge
delta_update_success_rate %.2f

# HELP delta_update_errors_total Total number of update errors
# TYPE delta_update_errors_total counter
delta_update_errors_total %d
`,
		agg.TotalEvents,
		agg.SuccessRate,
		agg.ErrorAnalysis.TotalErrors,
	)
	
	return []byte(metrics), nil
}

func (um *UpdateMetricsManager) shouldCollectSystemInfo() bool {
	// Check config for system info collection
	return true
}

func (um *UpdateMetricsManager) getOldestEventTime() *time.Time {
	if len(um.events) == 0 {
		return nil
	}
	return &um.events[0].Timestamp
}

func (um *UpdateMetricsManager) getNewestEventTime() *time.Time {
	if len(um.events) == 0 {
		return nil
	}
	return &um.events[len(um.events)-1].Timestamp
}

// SetEnabled enables or disables metrics collection
func (um *UpdateMetricsManager) SetEnabled(enabled bool) error {
	um.mutex.Lock()
	defer um.mutex.Unlock()
	
	um.isEnabled = enabled
	return um.saveConfig()
}

// IsEnabled returns whether metrics collection is enabled
func (um *UpdateMetricsManager) IsEnabled() bool {
	um.mutex.RLock()
	defer um.mutex.RUnlock()
	
	return um.isEnabled
}

// Close cleanly shuts down the metrics system
func (um *UpdateMetricsManager) Close() error {
	if um.flushTicker != nil {
		um.flushTicker.Stop()
	}
	
	return um.flush()
}

// Global metrics instance
var globalUpdateMetrics *UpdateMetricsManager
var metricsOnce sync.Once

// GetUpdateMetrics returns the global metrics instance
func GetUpdateMetrics() *UpdateMetricsManager {
	metricsOnce.Do(func() {
		var err error
		globalUpdateMetrics, err = NewUpdateMetrics()
		if err != nil {
			fmt.Printf("Warning: failed to initialize update metrics: %v\n", err)
		}
	})
	return globalUpdateMetrics
}