package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// HandleUpdateMetricsCommand handles the :update metrics subcommand
func HandleUpdateMetricsCommand(args []string) bool {
	metrics := GetUpdateMetrics()
	if metrics == nil {
		fmt.Println("Metrics system not available")
		return false
	}

	if len(args) == 0 {
		showMetricsSummary(metrics)
		return true
	}

	switch args[0] {
	case "summary":
		showMetricsSummary(metrics)
	case "report":
		return generateMetricsReport(metrics, args[1:])
	case "export":
		return exportMetrics(metrics, args[1:])
	case "channel":
		return showChannelMetrics(metrics, args[1:])
	case "version":
		return showVersionMetrics(metrics, args[1:])
	case "errors":
		return showErrorAnalysis(metrics, args[1:])
	case "performance":
		return showPerformanceMetrics(metrics, args[1:])
	case "config":
		return handleMetricsConfig(metrics, args[1:])
	case "clear":
		return clearMetrics(metrics)
	case "help":
		showMetricsHelp()
	default:
		fmt.Printf("Unknown metrics command: %s\n", args[0])
		showMetricsHelp()
	}

	return true
}

// showMetricsSummary displays a summary of metrics
func showMetricsSummary(metrics *UpdateMetricsManager) {
	summary := metrics.GetMetricsSummary()

	fmt.Println("Update Metrics Summary")
	fmt.Println("=====================")
	
	if enabled, ok := summary["metrics_enabled"].(bool); ok {
		if enabled {
			fmt.Printf("Status: %sEnabled%s\n", ColorGreen, ColorReset)
		} else {
			fmt.Printf("Status: %sDisabled%s\n", ColorYellow, ColorReset)
		}
	}
	
	fmt.Println("\nLast 7 Days:")
	if total, ok := summary["total_events_7d"].(int); ok {
		fmt.Printf("  Total Events: %d\n", total)
	}
	if rate, ok := summary["success_rate_7d"].(float64); ok {
		fmt.Printf("  Success Rate: %.1f%%\n", rate)
	}
	
	fmt.Println("\nAll Time:")
	if total, ok := summary["total_events_all"].(int); ok {
		fmt.Printf("  Total Events: %d\n", total)
	}
	if retention, ok := summary["retention_days"].(int); ok {
		fmt.Printf("  Data Retention: %d days\n", retention)
	}
	
	// Show event breakdown
	if eventTypes, ok := summary["events_by_type"].(map[MetricType]int); ok && len(eventTypes) > 0 {
		fmt.Println("\nEvent Breakdown (7 days):")
		for eventType, count := range eventTypes {
			fmt.Printf("  %-20s: %d\n", eventType, count)
		}
	}
	
	// Show data range
	if oldest, ok := summary["oldest_event"].(*time.Time); ok && oldest != nil {
		fmt.Printf("\nData Range: %s", oldest.Format("2006-01-02"))
		if newest, ok := summary["newest_event"].(*time.Time); ok && newest != nil {
			fmt.Printf(" to %s", newest.Format("2006-01-02"))
		}
		fmt.Println()
	}
}

// generateMetricsReport generates a detailed metrics report
func generateMetricsReport(metrics *UpdateMetricsManager, args []string) bool {
	// Parse time range
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -7) // Default to last 7 days
	
	for i, arg := range args {
		switch arg {
		case "--days":
			if i+1 < len(args) {
				if days, err := parseIntSafely(args[i+1]); err == nil {
					startTime = endTime.AddDate(0, 0, -days)
				}
			}
		case "--start":
			if i+1 < len(args) {
				if t, err := time.Parse("2006-01-02", args[i+1]); err == nil {
					startTime = t
				}
			}
		case "--end":
			if i+1 < len(args) {
				if t, err := time.Parse("2006-01-02", args[i+1]); err == nil {
					endTime = t
				}
			}
		}
	}
	
	// Get aggregated metrics
	agg, err := metrics.GetAggregatedMetrics(startTime, endTime)
	if err != nil {
		fmt.Printf("Error generating report: %v\n", err)
		return false
	}
	
	// Display report
	fmt.Printf("\nUpdate Metrics Report\n")
	fmt.Printf("====================\n")
	fmt.Printf("Period: %s\n", agg.Period)
	fmt.Printf("Total Events: %d\n", agg.TotalEvents)
	fmt.Printf("Success Rate: %.1f%%\n\n", agg.SuccessRate)
	
	// Event breakdown
	if len(agg.EventsByType) > 0 {
		fmt.Println("Events by Type:")
		fmt.Println("---------------")
		for eventType, count := range agg.EventsByType {
			percentage := float64(count) / float64(agg.TotalEvents) * 100
			fmt.Printf("  %-20s: %4d (%.1f%%)\n", eventType, count, percentage)
		}
		fmt.Println()
	}
	
	// Channel metrics
	if len(agg.ChannelMetrics) > 0 {
		fmt.Println("Channel Performance:")
		fmt.Println("-------------------")
		for channel, cm := range agg.ChannelMetrics {
			fmt.Printf("  %s:\n", channel)
			fmt.Printf("    Total Updates: %d\n", cm.TotalUpdates)
			if cm.AverageDownloadTime > 0 {
				fmt.Printf("    Avg Download Time: %s\n", cm.AverageDownloadTime)
			}
		}
		fmt.Println()
	}
	
	// Version metrics
	if len(agg.VersionMetrics) > 0 {
		fmt.Println("Version Statistics:")
		fmt.Println("------------------")
		for version, vm := range agg.VersionMetrics {
			fmt.Printf("  %s:\n", version)
			fmt.Printf("    Installs: %d\n", vm.InstallCount)
			if vm.RollbackCount > 0 {
				fmt.Printf("    Rollbacks: %d\n", vm.RollbackCount)
			}
			if vm.SkipCount > 0 {
				fmt.Printf("    Skips: %d\n", vm.SkipCount)
			}
		}
		fmt.Println()
	}
	
	// Performance stats
	if agg.PerformanceStats != nil {
		fmt.Println("Performance Statistics:")
		fmt.Println("----------------------")
		if agg.PerformanceStats.AverageInstallDuration > 0 {
			fmt.Printf("  Avg Install Time: %s\n", agg.PerformanceStats.AverageInstallDuration)
		}
		if agg.PerformanceStats.FastestDownload > 0 {
			fmt.Printf("  Fastest Download: %s\n", agg.PerformanceStats.FastestDownload)
			fmt.Printf("  Slowest Download: %s\n", agg.PerformanceStats.SlowestDownload)
		}
		fmt.Println()
	}
	
	// Error analysis
	if agg.ErrorAnalysis != nil && agg.ErrorAnalysis.TotalErrors > 0 {
		fmt.Printf("Error Analysis:\n")
		fmt.Printf("---------------\n")
		fmt.Printf("  Total Errors: %d\n", agg.ErrorAnalysis.TotalErrors)
		errorRate := float64(agg.ErrorAnalysis.TotalErrors) / float64(agg.TotalEvents) * 100
		fmt.Printf("  Error Rate: %.1f%%\n", errorRate)
		
		if len(agg.ErrorAnalysis.ErrorsByType) > 0 {
			fmt.Println("  Error Types:")
			for errorType, count := range agg.ErrorAnalysis.ErrorsByType {
				percentage := float64(count) / float64(agg.ErrorAnalysis.TotalErrors) * 100
				fmt.Printf("    %-15s: %3d (%.1f%%)\n", errorType, count, percentage)
			}
		}
	}
	
	return true
}

// exportMetrics exports metrics in various formats
func exportMetrics(metrics *UpdateMetricsManager, args []string) bool {
	if len(args) == 0 {
		fmt.Println("Usage: :update metrics export <format> [options]")
		fmt.Println("Formats: json, csv, prometheus")
		return false
	}
	
	format := args[0]
	outputFile := ""
	
	// Parse options
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -7)
	
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--output", "-o":
			if i+1 < len(args) {
				outputFile = args[i+1]
				i++
			}
		case "--days":
			if i+1 < len(args) {
				if days, err := parseIntSafely(args[i+1]); err == nil {
					startTime = endTime.AddDate(0, 0, -days)
					i++
				}
			}
		}
	}
	
	// Export metrics
	data, err := metrics.ExportMetrics(format, startTime, endTime)
	if err != nil {
		fmt.Printf("Error exporting metrics: %v\n", err)
		return false
	}
	
	// Output to file or stdout
	if outputFile != "" {
		if err := os.WriteFile(outputFile, data, 0644); err != nil {
			fmt.Printf("Error writing file: %v\n", err)
			return false
		}
		fmt.Printf("Metrics exported to %s\n", outputFile)
	} else {
		fmt.Println(string(data))
	}
	
	return true
}

// showChannelMetrics displays channel-specific metrics
func showChannelMetrics(metrics *UpdateMetricsManager, args []string) bool {
	// Get metrics for last 30 days
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -30)
	
	agg, err := metrics.GetAggregatedMetrics(startTime, endTime)
	if err != nil {
		fmt.Printf("Error getting channel metrics: %v\n", err)
		return false
	}
	
	fmt.Println("Channel Metrics (Last 30 Days)")
	fmt.Println("==============================")
	
	if len(agg.ChannelMetrics) == 0 {
		fmt.Println("No channel data available")
		return true
	}
	
	// Show current channel from ChannelManager
	if cm := GetChannelManager(); cm != nil {
		currentChannel := cm.GetCurrentChannel()
		fmt.Printf("Current Channel: %s%s%s\n\n", ColorCyan, currentChannel, ColorReset)
	}
	
	// Display metrics for each channel
	for channel, cm := range agg.ChannelMetrics {
		fmt.Printf("Channel: %s\n", channel)
		fmt.Printf("  Total Updates:     %d\n", cm.TotalUpdates)
		fmt.Printf("  Successful:        %d\n", cm.SuccessfulUpdates)
		fmt.Printf("  Failed:            %d\n", cm.FailedUpdates)
		
		if cm.TotalUpdates > 0 {
			successRate := float64(cm.SuccessfulUpdates) / float64(cm.TotalUpdates) * 100
			fmt.Printf("  Success Rate:      %.1f%%\n", successRate)
		}
		
		if cm.AverageDownloadTime > 0 {
			fmt.Printf("  Avg Download Time: %s\n", cm.AverageDownloadTime)
		}
		if cm.AverageInstallTime > 0 {
			fmt.Printf("  Avg Install Time:  %s\n", cm.AverageInstallTime)
		}
		if cm.TotalDownloadSize > 0 {
			fmt.Printf("  Total Downloaded:  %s\n", formatFileSize(cm.TotalDownloadSize))
		}
		fmt.Println()
	}
	
	return true
}

// showVersionMetrics displays version-specific metrics
func showVersionMetrics(metrics *UpdateMetricsManager, args []string) bool {
	// Get metrics for last 90 days
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -90)
	
	agg, err := metrics.GetAggregatedMetrics(startTime, endTime)
	if err != nil {
		fmt.Printf("Error getting version metrics: %v\n", err)
		return false
	}
	
	fmt.Println("Version Metrics (Last 90 Days)")
	fmt.Println("==============================")
	
	if len(agg.VersionMetrics) == 0 {
		fmt.Println("No version data available")
		return true
	}
	
	// Current version
	if um := GetUpdateManager(); um != nil {
		fmt.Printf("Current Version: %s%s%s\n\n", ColorCyan, um.GetCurrentVersion(), ColorReset)
	}
	
	// Display metrics for each version
	for version, vm := range agg.VersionMetrics {
		fmt.Printf("Version: %s\n", version)
		fmt.Printf("  Install Count:     %d\n", vm.InstallCount)
		
		if vm.RollbackCount > 0 {
			fmt.Printf("  Rollback Count:    %d%s\n", vm.RollbackCount, ColorRed+ColorReset)
		}
		if vm.SkipCount > 0 {
			fmt.Printf("  Skip Count:        %d\n", vm.SkipCount)
		}
		if vm.PostponeCount > 0 {
			fmt.Printf("  Postpone Count:    %d\n", vm.PostponeCount)
		}
		
		if vm.SuccessRate > 0 {
			color := ColorGreen
			if vm.SuccessRate < 90 {
				color = ColorYellow
			}
			if vm.SuccessRate < 70 {
				color = ColorRed
			}
			fmt.Printf("  Success Rate:      %s%.1f%%%s\n", color, vm.SuccessRate, ColorReset)
		}
		
		if vm.AverageInstallTime > 0 {
			fmt.Printf("  Avg Install Time:  %s\n", vm.AverageInstallTime)
		}
		fmt.Println()
	}
	
	return true
}

// showErrorAnalysis displays error analysis
func showErrorAnalysis(metrics *UpdateMetricsManager, args []string) bool {
	// Get metrics for last 30 days
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -30)
	
	agg, err := metrics.GetAggregatedMetrics(startTime, endTime)
	if err != nil {
		fmt.Printf("Error getting error analysis: %v\n", err)
		return false
	}
	
	fmt.Println("Error Analysis (Last 30 Days)")
	fmt.Println("=============================")
	
	if agg.ErrorAnalysis == nil || agg.ErrorAnalysis.TotalErrors == 0 {
		fmt.Println("No errors recorded in this period")
		return true
	}
	
	ea := agg.ErrorAnalysis
	errorRate := float64(ea.TotalErrors) / float64(agg.TotalEvents) * 100
	
	fmt.Printf("Total Errors: %d (%.1f%% error rate)\n\n", ea.TotalErrors, errorRate)
	
	// Error types breakdown
	if len(ea.ErrorsByType) > 0 {
		fmt.Println("Error Types:")
		maxType := 0
		for errorType := range ea.ErrorsByType {
			if len(errorType) > maxType {
				maxType = len(errorType)
			}
		}
		
		for errorType, count := range ea.ErrorsByType {
			percentage := float64(count) / float64(ea.TotalErrors) * 100
			bar := strings.Repeat("█", int(percentage/2))
			fmt.Printf("  %-*s: %3d (%.1f%%) %s\n", maxType, errorType, count, percentage, bar)
		}
		fmt.Println()
	}
	
	// Common error patterns
	if len(ea.CommonErrors) > 0 {
		fmt.Println("Common Error Patterns:")
		for i, pattern := range ea.CommonErrors {
			if i >= 5 { // Show top 5
				break
			}
			fmt.Printf("  %d. %s (%d occurrences)\n", i+1, pattern.Pattern, pattern.Count)
			if pattern.Suggestion != "" {
				fmt.Printf("     Suggestion: %s\n", pattern.Suggestion)
			}
		}
		fmt.Println()
	}
	
	// Recovery rate
	if ea.RecoveryRate > 0 {
		fmt.Printf("Recovery Rate: %.1f%% (errors followed by successful retry)\n", ea.RecoveryRate)
	}
	
	return true
}

// showPerformanceMetrics displays performance metrics
func showPerformanceMetrics(metrics *UpdateMetricsManager, args []string) bool {
	// Get metrics for last 30 days
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -30)
	
	agg, err := metrics.GetAggregatedMetrics(startTime, endTime)
	if err != nil {
		fmt.Printf("Error getting performance metrics: %v\n", err)
		return false
	}
	
	fmt.Println("Performance Metrics (Last 30 Days)")
	fmt.Println("==================================")
	
	if agg.PerformanceStats == nil {
		fmt.Println("No performance data available")
		return true
	}
	
	ps := agg.PerformanceStats
	
	// Download performance
	fmt.Println("Download Performance:")
	if ps.AverageDownloadSpeed > 0 {
		fmt.Printf("  Average Speed:     %.2f MB/s\n", ps.AverageDownloadSpeed)
	}
	if ps.FastestDownload > 0 {
		fmt.Printf("  Fastest Download:  %s\n", ps.FastestDownload)
		fmt.Printf("  Slowest Download:  %s\n", ps.SlowestDownload)
		
		// Show range
		rangeTime := ps.SlowestDownload - ps.FastestDownload
		fmt.Printf("  Variance:          %s\n", rangeTime)
	}
	
	// Installation performance
	fmt.Println("\nInstallation Performance:")
	if ps.AverageInstallDuration > 0 {
		fmt.Printf("  Average Duration:  %s\n", ps.AverageInstallDuration)
	}
	
	// Network reliability
	if ps.NetworkReliability > 0 {
		reliabilityColor := ColorGreen
		if ps.NetworkReliability < 95 {
			reliabilityColor = ColorYellow
		}
		if ps.NetworkReliability < 90 {
			reliabilityColor = ColorRed
		}
		fmt.Printf("\nNetwork Reliability: %s%.1f%%%s\n", reliabilityColor, ps.NetworkReliability, ColorReset)
	}
	
	// Peak usage
	if ps.PeakDownloadHour >= 0 {
		fmt.Printf("\nPeak Download Hour: %02d:00-%02d:00\n", ps.PeakDownloadHour, ps.PeakDownloadHour+1)
	}
	
	return true
}

// handleMetricsConfig handles metrics configuration
func handleMetricsConfig(metrics *UpdateMetricsManager, args []string) bool {
	if len(args) == 0 {
		// Show current config
		fmt.Println("Metrics Configuration")
		fmt.Println("====================")
		fmt.Printf("  Enabled: %v\n", metrics.IsEnabled())
		
		summary := metrics.GetMetricsSummary()
		if retention, ok := summary["retention_days"].(int); ok {
			fmt.Printf("  Data Retention: %d days\n", retention)
		}
		
		fmt.Println("\nOptions:")
		fmt.Println("  :update metrics config enable")
		fmt.Println("  :update metrics config disable")
		fmt.Println("  :update metrics config retention <days>")
		return true
	}
	
	switch args[0] {
	case "enable":
		if err := metrics.SetEnabled(true); err != nil {
			fmt.Printf("Error enabling metrics: %v\n", err)
			return false
		}
		fmt.Println("✅ Metrics collection enabled")
		
	case "disable":
		if err := metrics.SetEnabled(false); err != nil {
			fmt.Printf("Error disabling metrics: %v\n", err)
			return false
		}
		fmt.Println("✅ Metrics collection disabled")
		
	case "retention":
		if len(args) < 2 {
			fmt.Println("Usage: :update metrics config retention <days>")
			return false
		}
		// This would require adding a SetRetentionDays method
		fmt.Printf("Setting retention to %s days (not implemented)\n", args[1])
		
	default:
		fmt.Printf("Unknown config option: %s\n", args[0])
		return false
	}
	
	return true
}

// clearMetrics clears all metrics data
func clearMetrics(metrics *UpdateMetricsManager) bool {
	fmt.Print("Are you sure you want to clear all metrics data? [y/N]: ")
	var response string
	fmt.Scanln(&response)
	
	if strings.ToLower(response) != "y" {
		fmt.Println("Clear operation cancelled")
		return true
	}
	
	// This would require adding a Clear method to UpdateMetrics
	fmt.Println("Clearing metrics data... (not implemented)")
	fmt.Println("Note: This feature requires implementation in UpdateMetrics")
	
	return true
}

// showMetricsHelp displays help for metrics commands
func showMetricsHelp() {
	fmt.Println("Update Metrics Commands")
	fmt.Println("======================")
	fmt.Println("  :update metrics                    - Show metrics summary")
	fmt.Println("  :update metrics summary            - Show detailed summary")
	fmt.Println("  :update metrics report             - Generate metrics report")
	fmt.Println("  :update metrics export <format>    - Export metrics (json/csv/prometheus)")
	fmt.Println("  :update metrics channel            - Show channel-specific metrics")
	fmt.Println("  :update metrics version            - Show version-specific metrics")
	fmt.Println("  :update metrics errors             - Show error analysis")
	fmt.Println("  :update metrics performance        - Show performance metrics")
	fmt.Println("  :update metrics config             - Manage metrics configuration")
	fmt.Println("  :update metrics clear              - Clear all metrics data")
	fmt.Println("  :update metrics help               - Show this help")
	fmt.Println()
	fmt.Println("Report Options:")
	fmt.Println("  --days <n>         - Show metrics for last n days")
	fmt.Println("  --start YYYY-MM-DD - Start date for report")
	fmt.Println("  --end YYYY-MM-DD   - End date for report")
	fmt.Println()
	fmt.Println("Export Options:")
	fmt.Println("  --output <file>    - Save to file instead of stdout")
	fmt.Println("  --days <n>         - Export last n days of data")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  :update metrics report --days 30")
	fmt.Println("  :update metrics export json --output metrics.json")
	fmt.Println("  :update metrics channel")
}