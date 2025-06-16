package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

// UpdateValidator handles post-update validation and health checks
type UpdateValidator struct {
	mutex           sync.RWMutex
	updateManager   *UpdateManager
	i18nManager     *I18nManager
	validationTests []ValidationTest
	config          *ValidationConfig
}

// ValidationTest defines a specific validation test
type ValidationTest struct {
	Name        string
	Description string
	TestFunc    func() *ValidationResult
	Critical    bool // If true, failure triggers automatic rollback
	Timeout     time.Duration
	Enabled     bool
}

// ValidationConfig configures the validation behavior
type ValidationConfig struct {
	Enabled                bool          `json:"enabled"`
	AutoRollbackOnFailure  bool          `json:"auto_rollback_on_failure"`
	CriticalTestsOnly      bool          `json:"critical_tests_only"`
	ValidationTimeout      time.Duration `json:"validation_timeout"`
	MaxRetries             int           `json:"max_retries"`
	SkipNonCriticalOnError bool          `json:"skip_non_critical_on_error"`
}

// ValidationSuite represents a complete validation run
type ValidationSuite struct {
	ID            string               `json:"id"`
	StartTime     time.Time            `json:"start_time"`
	EndTime       time.Time            `json:"end_time"`
	Duration      time.Duration        `json:"duration"`
	TotalTests    int                  `json:"total_tests"`
	PassedTests   int                  `json:"passed_tests"`
	FailedTests   int                  `json:"failed_tests"`
	SkippedTests  int                  `json:"skipped_tests"`
	Results       []*ValidationResult  `json:"results"`
	OverallStatus ValidationStatus     `json:"overall_status"`
	Version       string               `json:"version"`
	TriggerReason string               `json:"trigger_reason"`
}

// ValidationStatus represents the overall validation status
type ValidationStatus string

const (
	ValidationStatusPassed  ValidationStatus = "passed"
	ValidationStatusFailed  ValidationStatus = "failed"
	ValidationStatusPartial ValidationStatus = "partial"
	ValidationStatusSkipped ValidationStatus = "skipped"
)

// Global validator instance
var globalUpdateValidator *UpdateValidator
var validatorOnce sync.Once

// GetUpdateValidator returns the global UpdateValidator instance
func GetUpdateValidator() *UpdateValidator {
	validatorOnce.Do(func() {
		um := GetUpdateManager()
		if um != nil {
			globalUpdateValidator = NewUpdateValidator(um)
			globalUpdateValidator.initializeDefaultTests()
		}
	})
	return globalUpdateValidator
}

// NewUpdateValidator creates a new update validator instance
func NewUpdateValidator(updateManager *UpdateManager) *UpdateValidator {
	return &UpdateValidator{
		updateManager: updateManager,
		i18nManager:   GetI18nManager(),
		config: &ValidationConfig{
			Enabled:                true,
			AutoRollbackOnFailure:  true,
			CriticalTestsOnly:      false,
			ValidationTimeout:      5 * time.Minute,
			MaxRetries:             3,
			SkipNonCriticalOnError: false,
		},
	}
}

// initializeDefaultTests sets up the default validation tests
func (uv *UpdateValidator) initializeDefaultTests() {
	uv.validationTests = []ValidationTest{
		{
			Name:        "binary_executable",
			Description: "Verify the updated binary is executable",
			TestFunc:    uv.testBinaryExecutable,
			Critical:    true,
			Timeout:     30 * time.Second,
			Enabled:     true,
		},
		{
			Name:        "version_check",
			Description: "Verify the version has been updated correctly",
			TestFunc:    uv.testVersionCheck,
			Critical:    true,
			Timeout:     10 * time.Second,
			Enabled:     true,
		},
		{
			Name:        "basic_functionality",
			Description: "Test basic CLI functionality",
			TestFunc:    uv.testBasicFunctionality,
			Critical:    true,
			Timeout:     30 * time.Second,
			Enabled:     true,
		},
		{
			Name:        "config_compatibility",
			Description: "Verify configuration files are compatible",
			TestFunc:    uv.testConfigCompatibility,
			Critical:    false,
			Timeout:     20 * time.Second,
			Enabled:     true,
		},
		{
			Name:        "dependency_check",
			Description: "Check that required dependencies are available",
			TestFunc:    uv.testDependencyCheck,
			Critical:    false,
			Timeout:     30 * time.Second,
			Enabled:     true,
		},
		{
			Name:        "memory_usage",
			Description: "Verify memory usage is within acceptable limits",
			TestFunc:    uv.testMemoryUsage,
			Critical:    false,
			Timeout:     15 * time.Second,
			Enabled:     true,
		},
	}
}

// RunValidation runs the complete validation suite
func (uv *UpdateValidator) RunValidation(version string, triggerReason string) (*ValidationSuite, error) {
	uv.mutex.Lock()
	defer uv.mutex.Unlock()

	if !uv.config.Enabled {
		return nil, fmt.Errorf("validation is disabled")
	}

	suite := &ValidationSuite{
		ID:            fmt.Sprintf("validation_%d", time.Now().Unix()),
		StartTime:     time.Now(),
		Version:       version,
		TriggerReason: triggerReason,
		Results:       make([]*ValidationResult, 0),
	}

	fmt.Printf("🔍 Running post-update validation for version %s...\n", version)

	// Get enabled tests
	tests := uv.getEnabledTests()
	suite.TotalTests = len(tests)

	// Run tests
	for _, test := range tests {
		if uv.config.CriticalTestsOnly && !test.Critical {
			suite.SkippedTests++
			continue
		}

		fmt.Printf("   • %s... ", test.Description)

		// Run test with timeout
		result := uv.runTestWithTimeout(test)
		suite.Results = append(suite.Results, result)

		switch result.Status {
		case "passed":
			suite.PassedTests++
			fmt.Printf("✅ %s\n", colorizeText("PASSED", "green"))
		case "failed":
			suite.FailedTests++
			fmt.Printf("❌ %s", colorizeText("FAILED", "red"))
			if result.ErrorMsg != "" {
				fmt.Printf(" (%s)", result.ErrorMsg)
			}
			fmt.Println()

			// Check if this is a critical test failure
			if test.Critical && uv.config.AutoRollbackOnFailure {
				suite.OverallStatus = ValidationStatusFailed
				suite.EndTime = time.Now()
				suite.Duration = suite.EndTime.Sub(suite.StartTime)

				fmt.Printf("💥 Critical test failed! Initiating automatic rollback...\n")
				if err := uv.performAutoRollback(); err != nil {
					fmt.Printf("❌ Automatic rollback failed: %v\n", err)
				} else {
					fmt.Printf("✅ Automatic rollback completed successfully\n")
				}

				return suite, fmt.Errorf("critical validation test failed: %s", test.Name)
			}
		case "skipped":
			suite.SkippedTests++
			fmt.Printf("⏭️  %s\n", colorizeText("SKIPPED", "yellow"))
		}

		// Stop if we should skip non-critical tests on error
		if result.Status == "failed" && uv.config.SkipNonCriticalOnError {
			break
		}
	}

	// Determine overall status
	suite.EndTime = time.Now()
	suite.Duration = suite.EndTime.Sub(suite.StartTime)

	if suite.FailedTests == 0 {
		suite.OverallStatus = ValidationStatusPassed
		fmt.Printf("✅ All validation tests passed! (%s)\n", suite.Duration.Truncate(time.Second))
	} else if suite.PassedTests > 0 {
		suite.OverallStatus = ValidationStatusPartial
		fmt.Printf("⚠️  Validation completed with %d failures (%s)\n", suite.FailedTests, suite.Duration.Truncate(time.Second))
	} else {
		suite.OverallStatus = ValidationStatusFailed
		fmt.Printf("❌ Validation failed (%s)\n", suite.Duration.Truncate(time.Second))
	}

	// Record validation results in update history
	uv.recordValidationInHistory(suite)

	return suite, nil
}

// runTestWithTimeout runs a single test with timeout protection
func (uv *UpdateValidator) runTestWithTimeout(test ValidationTest) *ValidationResult {
	resultChan := make(chan *ValidationResult, 1)
	
	go func() {
		startTime := time.Now()
		result := test.TestFunc()
		result.TestName = test.Name
		result.Duration = time.Since(startTime)
		resultChan <- result
	}()

	select {
	case result := <-resultChan:
		return result
	case <-time.After(test.Timeout):
		return &ValidationResult{
			TestName: test.Name,
			Status:   "failed",
			ErrorMsg: fmt.Sprintf("test timed out after %s", test.Timeout),
			Duration: test.Timeout,
		}
	}
}

// Test implementations

func (uv *UpdateValidator) testBinaryExecutable() *ValidationResult {
	binaryPath, err := os.Executable()
	if err != nil {
		return &ValidationResult{
			Status:   "failed",
			ErrorMsg: fmt.Sprintf("cannot determine executable path: %v", err),
		}
	}

	// Check if file exists and is executable
	info, err := os.Stat(binaryPath)
	if err != nil {
		return &ValidationResult{
			Status:   "failed",
			ErrorMsg: fmt.Sprintf("binary not found: %v", err),
		}
	}

	// Check executable permissions
	if runtime.GOOS != "windows" {
		if info.Mode()&0111 == 0 {
			return &ValidationResult{
				Status:   "failed",
				ErrorMsg: "binary is not executable",
			}
		}
	}

	return &ValidationResult{
		Status: "passed",
		Details: map[string]interface{}{
			"binary_path": binaryPath,
			"size":        info.Size(),
			"mode":        info.Mode().String(),
		},
	}
}

func (uv *UpdateValidator) testVersionCheck() *ValidationResult {
	expectedVersion := uv.updateManager.GetCurrentVersion()
	
	// Run the binary with --version flag
	binaryPath, err := os.Executable()
	if err != nil {
		return &ValidationResult{
			Status:   "failed",
			ErrorMsg: fmt.Sprintf("cannot determine executable path: %v", err),
		}
	}

	cmd := exec.Command(binaryPath, "--version")
	output, err := cmd.Output()
	if err != nil {
		return &ValidationResult{
			Status:   "failed",
			ErrorMsg: fmt.Sprintf("version command failed: %v", err),
		}
	}

	versionOutput := strings.TrimSpace(string(output))
	if !strings.Contains(versionOutput, expectedVersion) {
		return &ValidationResult{
			Status:   "failed",
			ErrorMsg: fmt.Sprintf("version mismatch: expected %s, got %s", expectedVersion, versionOutput),
		}
	}

	return &ValidationResult{
		Status: "passed",
		Details: map[string]interface{}{
			"expected_version": expectedVersion,
			"actual_output":    versionOutput,
		},
	}
}

func (uv *UpdateValidator) testBasicFunctionality() *ValidationResult {
	// Test basic help command
	binaryPath, err := os.Executable()
	if err != nil {
		return &ValidationResult{
			Status:   "failed",
			ErrorMsg: fmt.Sprintf("cannot determine executable path: %v", err),
		}
	}

	cmd := exec.Command(binaryPath, "--help")
	output, err := cmd.Output()
	if err != nil {
		return &ValidationResult{
			Status:   "failed",
			ErrorMsg: fmt.Sprintf("help command failed: %v", err),
		}
	}

	helpOutput := strings.TrimSpace(string(output))
	if len(helpOutput) < 10 {
		return &ValidationResult{
			Status:   "failed",
			ErrorMsg: "help output too short or empty",
		}
	}

	return &ValidationResult{
		Status: "passed",
		Details: map[string]interface{}{
			"help_output_length": len(helpOutput),
		},
	}
}

func (uv *UpdateValidator) testConfigCompatibility() *ValidationResult {
	// Test if configuration files are readable
	configManager := GetConfigManager()
	if configManager == nil {
		return &ValidationResult{
			Status:   "failed",
			ErrorMsg: "config manager not available",
		}
	}

	// Try to initialize config
	if err := configManager.InitializeBase(); err != nil {
		return &ValidationResult{
			Status:   "failed",
			ErrorMsg: fmt.Sprintf("config initialization failed: %v", err),
		}
	}

	return &ValidationResult{
		Status: "passed",
		Details: map[string]interface{}{
			"config_initialized": true,
		},
	}
}

func (uv *UpdateValidator) testDependencyCheck() *ValidationResult {
	missingDeps := []string{}

	// Check for SQLite extension
	if _, err := os.Stat("vec0.so"); err != nil {
		missingDeps = append(missingDeps, "SQLite vector extension (vec0.so)")
	}

	if len(missingDeps) > 0 {
		return &ValidationResult{
			Status:   "failed",
			ErrorMsg: fmt.Sprintf("missing dependencies: %s", strings.Join(missingDeps, ", ")),
			Details: map[string]interface{}{
				"missing_dependencies": missingDeps,
			},
		}
	}

	return &ValidationResult{
		Status: "passed",
		Details: map[string]interface{}{
			"all_dependencies_present": true,
		},
	}
}

func (uv *UpdateValidator) testMemoryUsage() *ValidationResult {
	// Get current memory usage
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Check if memory usage is reasonable (under 100MB for basic operation)
	maxMemoryMB := 100
	currentMemoryMB := int(m.Alloc / 1024 / 1024)

	if currentMemoryMB > maxMemoryMB {
		return &ValidationResult{
			Status:   "failed",
			ErrorMsg: fmt.Sprintf("memory usage too high: %dMB (limit: %dMB)", currentMemoryMB, maxMemoryMB),
		}
	}

	return &ValidationResult{
		Status: "passed",
		Details: map[string]interface{}{
			"memory_usage_mb": currentMemoryMB,
			"memory_limit_mb": maxMemoryMB,
		},
	}
}

// Helper methods

func (uv *UpdateValidator) getEnabledTests() []ValidationTest {
	var enabled []ValidationTest
	for _, test := range uv.validationTests {
		if test.Enabled {
			enabled = append(enabled, test)
		}
	}
	return enabled
}

func (uv *UpdateValidator) performAutoRollback() error {
	return uv.updateManager.RollbackToPreviousVersion()
}

func (uv *UpdateValidator) recordValidationInHistory(suite *ValidationSuite) {
	history := GetUpdateHistory()
	if history == nil {
		return
	}

	// Create a simplified record for the update history
	record := &UpdateRecord{
		ID:                fmt.Sprintf("validation_%s", suite.ID),
		Timestamp:         suite.StartTime,
		Type:              UpdateTypeManual, // Validation records are marked as manual
		FromVersion:       "unknown",
		ToVersion:         suite.Version,
		Status:            convertValidationStatusToUpdateStatus(suite.OverallStatus),
		Duration:          suite.Duration,
		Channel:           uv.updateManager.GetChannel(),
		TriggerMethod:     TriggerMethodCLI,
		ValidationResults: suite.Results,
		Metadata: map[string]interface{}{
			"validation_id":     suite.ID,
			"trigger_reason":    suite.TriggerReason,
			"total_tests":       suite.TotalTests,
			"passed_tests":      suite.PassedTests,
			"failed_tests":      suite.FailedTests,
			"skipped_tests":     suite.SkippedTests,
		},
	}

	history.RecordUpdate(record)
}

func convertValidationStatusToUpdateStatus(status ValidationStatus) UpdateStatus {
	switch status {
	case ValidationStatusPassed:
		return UpdateStatusSuccess
	case ValidationStatusFailed:
		return UpdateStatusFailed
	case ValidationStatusPartial:
		return UpdateStatusPartial
	default:
		return UpdateStatusFailed
	}
}

// Configuration methods

func (uv *UpdateValidator) SetConfig(config *ValidationConfig) {
	uv.mutex.Lock()
	defer uv.mutex.Unlock()
	uv.config = config
}

func (uv *UpdateValidator) GetConfig() *ValidationConfig {
	uv.mutex.RLock()
	defer uv.mutex.RUnlock()
	
	// Return a copy
	configCopy := *uv.config
	return &configCopy
}

func (uv *UpdateValidator) EnableTest(testName string) error {
	uv.mutex.Lock()
	defer uv.mutex.Unlock()

	for i := range uv.validationTests {
		if uv.validationTests[i].Name == testName {
			uv.validationTests[i].Enabled = true
			return nil
		}
	}

	return fmt.Errorf("test not found: %s", testName)
}

func (uv *UpdateValidator) DisableTest(testName string) error {
	uv.mutex.Lock()
	defer uv.mutex.Unlock()

	for i := range uv.validationTests {
		if uv.validationTests[i].Name == testName {
			uv.validationTests[i].Enabled = false
			return nil
		}
	}

	return fmt.Errorf("test not found: %s", testName)
}

func (uv *UpdateValidator) GetTests() []ValidationTest {
	uv.mutex.RLock()
	defer uv.mutex.RUnlock()

	// Return a copy
	tests := make([]ValidationTest, len(uv.validationTests))
	copy(tests, uv.validationTests)
	return tests
}