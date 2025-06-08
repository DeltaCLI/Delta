package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// UpdateInstaller handles the installation of updates with backup and rollback
type UpdateInstaller struct {
	backupDir     string
	installDir    string
	currentBinary string
	tempDir       string
	mutex         sync.RWMutex
	installLog    []InstallLogEntry
}

// InstallResult contains the result of an installation operation
type InstallResult struct {
	Success        bool
	BackupPath     string
	NewBinaryPath  string
	OldVersion     string
	NewVersion     string
	InstallTime    time.Duration
	Error          error
	LogEntries     []InstallLogEntry
}

// InstallLogEntry represents a single installation step
type InstallLogEntry struct {
	Timestamp time.Time
	Step      string
	Status    string
	Message   string
	Error     error
}

// BackupInfo contains information about a backup
type BackupInfo struct {
	BackupPath    string
	OriginalPath  string
	Version       string
	BackupTime    time.Time
	Size          int64
	Checksum      string
}

// NewUpdateInstaller creates a new update installer
func NewUpdateInstaller() (*UpdateInstaller, error) {
	// Determine current binary path
	currentBinary, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to get current executable path: %v", err)
	}

	// Get installation directory
	installDir := filepath.Dir(currentBinary)

	// Set up backup directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = os.Getenv("HOME")
	}
	backupDir := filepath.Join(homeDir, ".config", "delta", "backups")

	// Set up temp directory
	tempDir := filepath.Join(homeDir, ".config", "delta", "temp")

	installer := &UpdateInstaller{
		backupDir:     backupDir,
		installDir:    installDir,
		currentBinary: currentBinary,
		tempDir:       tempDir,
		installLog:    make([]InstallLogEntry, 0),
	}

	// Create necessary directories
	for _, dir := range []string{backupDir, tempDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %v", dir, err)
		}
	}

	return installer, nil
}

// InstallUpdate installs an update from a download result
func (ui *UpdateInstaller) InstallUpdate(downloadResult *DownloadResult) (*InstallResult, error) {
	if downloadResult == nil {
		return nil, fmt.Errorf("download result is nil")
	}

	startTime := time.Now()
	result := &InstallResult{
		OldVersion: GetVersionShort(),
		NewVersion: downloadResult.Version,
	}

	ui.logStep("start", "success", "Starting installation process")

	// Step 1: Extract the downloaded file
	ui.logStep("extract", "progress", "Extracting downloaded file")
	extractedPath, err := ui.extractDownload(downloadResult)
	if err != nil {
		ui.logStep("extract", "error", fmt.Sprintf("Failed to extract: %v", err))
		result.Error = err
		return result, err
	}
	ui.logStep("extract", "success", fmt.Sprintf("Extracted to: %s", extractedPath))

	// Step 2: Validate the extracted binary
	ui.logStep("validate", "progress", "Validating extracted binary")
	if err := ui.validateBinary(extractedPath); err != nil {
		ui.logStep("validate", "error", fmt.Sprintf("Binary validation failed: %v", err))
		result.Error = err
		ui.cleanup(extractedPath)
		return result, err
	}
	ui.logStep("validate", "success", "Binary validation passed")

	// Step 3: Create backup of current binary
	ui.logStep("backup", "progress", "Creating backup of current binary")
	backupPath, err := ui.CreateBackup()
	if err != nil {
		ui.logStep("backup", "error", fmt.Sprintf("Backup failed: %v", err))
		result.Error = err
		ui.cleanup(extractedPath)
		return result, err
	}
	result.BackupPath = backupPath
	ui.logStep("backup", "success", fmt.Sprintf("Backup created: %s", backupPath))

	// Step 4: Install the new binary
	ui.logStep("install", "progress", "Installing new binary")
	newBinaryPath, err := ui.installBinary(extractedPath)
	if err != nil {
		ui.logStep("install", "error", fmt.Sprintf("Installation failed: %v", err))
		result.Error = err
		
		// Attempt to rollback
		ui.logStep("rollback", "progress", "Attempting rollback")
		if rollbackErr := ui.Rollback(backupPath); rollbackErr != nil {
			ui.logStep("rollback", "error", fmt.Sprintf("Rollback failed: %v", rollbackErr))
		} else {
			ui.logStep("rollback", "success", "Rollback completed")
		}
		
		ui.cleanup(extractedPath)
		return result, err
	}
	result.NewBinaryPath = newBinaryPath
	ui.logStep("install", "success", fmt.Sprintf("Installation completed: %s", newBinaryPath))

	// Step 5: Verify the installation
	ui.logStep("verify", "progress", "Verifying installation")
	if err := ui.verifyInstallation(newBinaryPath); err != nil {
		ui.logStep("verify", "error", fmt.Sprintf("Installation verification failed: %v", err))
		result.Error = err
		
		// Attempt to rollback
		ui.logStep("rollback", "progress", "Attempting rollback due to verification failure")
		if rollbackErr := ui.Rollback(backupPath); rollbackErr != nil {
			ui.logStep("rollback", "error", fmt.Sprintf("Rollback failed: %v", rollbackErr))
		} else {
			ui.logStep("rollback", "success", "Rollback completed")
		}
		
		ui.cleanup(extractedPath)
		return result, err
	}
	ui.logStep("verify", "success", "Installation verification passed")

	// Step 6: Cleanup
	ui.logStep("cleanup", "progress", "Cleaning up temporary files")
	ui.cleanup(extractedPath)
	ui.logStep("cleanup", "success", "Cleanup completed")

	result.Success = true
	result.InstallTime = time.Since(startTime)
	result.LogEntries = ui.getLogEntries()

	ui.logStep("complete", "success", fmt.Sprintf("Installation completed successfully in %s", result.InstallTime.Truncate(time.Millisecond)))

	return result, nil
}

// extractDownload extracts the downloaded file and returns the path to the binary
func (ui *UpdateInstaller) extractDownload(downloadResult *DownloadResult) (string, error) {
	filePath := downloadResult.FilePath
	
	// Create extraction directory
	extractDir := filepath.Join(ui.tempDir, fmt.Sprintf("extract_%d", time.Now().Unix()))
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create extraction directory: %v", err)
	}

	var extractedBinaryPath string
	var err error

	// Determine extraction method based on file extension
	switch {
	case strings.HasSuffix(filePath, ".tar.gz"):
		extractedBinaryPath, err = ui.extractTarGz(filePath, extractDir)
	case strings.HasSuffix(filePath, ".zip"):
		extractedBinaryPath, err = ui.extractZip(filePath, extractDir)
	default:
		// Assume it's a raw binary
		extractedBinaryPath = filepath.Join(extractDir, "delta")
		if runtime.GOOS == "windows" {
			extractedBinaryPath += ".exe"
		}
		err = ui.copyFile(filePath, extractedBinaryPath)
	}

	if err != nil {
		os.RemoveAll(extractDir)
		return "", err
	}

	return extractedBinaryPath, nil
}

// extractTarGz extracts a .tar.gz file
func (ui *UpdateInstaller) extractTarGz(filePath, extractDir string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)

	var binaryPath string
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		targetPath := filepath.Join(extractDir, header.Name)

		switch header.Typeflag {
		case tar.TypeReg:
			// Extract regular file
			if err := ui.extractTarFile(tarReader, targetPath, header.FileInfo().Mode()); err != nil {
				return "", err
			}
			
			// Check if this looks like our binary
			if ui.isBinaryFile(header.Name) {
				binaryPath = targetPath
			}
		case tar.TypeDir:
			// Create directory
			if err := os.MkdirAll(targetPath, header.FileInfo().Mode()); err != nil {
				return "", err
			}
		}
	}

	if binaryPath == "" {
		return "", fmt.Errorf("no binary found in archive")
	}

	return binaryPath, nil
}

// extractZip extracts a .zip file
func (ui *UpdateInstaller) extractZip(filePath, extractDir string) (string, error) {
	reader, err := zip.OpenReader(filePath)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	var binaryPath string
	for _, file := range reader.File {
		targetPath := filepath.Join(extractDir, file.Name)

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(targetPath, file.FileInfo().Mode()); err != nil {
				return "", err
			}
			continue
		}

		// Extract file
		if err := ui.extractZipFile(file, targetPath); err != nil {
			return "", err
		}

		// Check if this looks like our binary
		if ui.isBinaryFile(file.Name) {
			binaryPath = targetPath
		}
	}

	if binaryPath == "" {
		return "", fmt.Errorf("no binary found in archive")
	}

	return binaryPath, nil
}

// extractTarFile extracts a single file from a tar archive
func (ui *UpdateInstaller) extractTarFile(tarReader *tar.Reader, targetPath string, mode os.FileMode) error {
	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}

	// Create the file
	file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer file.Close()

	// Copy data
	if _, err := io.Copy(file, tarReader); err != nil {
		return err
	}

	return nil
}

// extractZipFile extracts a single file from a zip archive
func (ui *UpdateInstaller) extractZipFile(file *zip.File, targetPath string) error {
	// Create directory if needed
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}

	// Open the file in the zip
	reader, err := file.Open()
	if err != nil {
		return err
	}
	defer reader.Close()

	// Create the target file
	targetFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, file.FileInfo().Mode())
	if err != nil {
		return err
	}
	defer targetFile.Close()

	// Copy data
	if _, err := io.Copy(targetFile, reader); err != nil {
		return err
	}

	return nil
}

// isBinaryFile determines if a file looks like our binary
func (ui *UpdateInstaller) isBinaryFile(name string) bool {
	name = strings.ToLower(filepath.Base(name))
	
	// Look for "delta" in the name
	if strings.Contains(name, "delta") {
		return true
	}

	// On Windows, look for .exe files
	if runtime.GOOS == "windows" && strings.HasSuffix(name, ".exe") {
		return true
	}

	// Check for executable files without extensions (Unix-like)
	if runtime.GOOS != "windows" && !strings.Contains(name, ".") {
		return true
	}

	return false
}

// validateBinary validates that the extracted binary is executable and correct
func (ui *UpdateInstaller) validateBinary(binaryPath string) error {
	// Check if file exists
	info, err := os.Stat(binaryPath)
	if err != nil {
		return fmt.Errorf("binary not found: %v", err)
	}

	// Check if it's a regular file
	if !info.Mode().IsRegular() {
		return fmt.Errorf("binary is not a regular file")
	}

	// On Unix-like systems, check if it's executable
	if runtime.GOOS != "windows" {
		if info.Mode().Perm()&0111 == 0 {
			// Try to make it executable
			if err := os.Chmod(binaryPath, info.Mode()|0111); err != nil {
				return fmt.Errorf("binary is not executable and cannot be made executable: %v", err)
			}
		}
	}

	// Try to run the binary with --version to validate it
	cmd := exec.Command(binaryPath, "--version")
	if err := cmd.Run(); err != nil {
		// If --version fails, try with version subcommand
		cmd = exec.Command(binaryPath, "version")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("binary validation failed - not a valid Delta CLI binary: %v", err)
		}
	}

	return nil
}

// CreateBackup creates a backup of the current binary
func (ui *UpdateInstaller) CreateBackup() (string, error) {
	// Generate backup filename
	timestamp := time.Now().Format("20060102_150405")
	version := strings.ReplaceAll(GetVersionShort(), "v", "")
	backupName := fmt.Sprintf("delta_%s_%s", version, timestamp)
	if runtime.GOOS == "windows" {
		backupName += ".exe"
	}
	backupPath := filepath.Join(ui.backupDir, backupName)

	// Copy current binary to backup location
	if err := ui.copyFile(ui.currentBinary, backupPath); err != nil {
		return "", fmt.Errorf("failed to create backup: %v", err)
	}

	// Verify backup
	if err := ui.validateBinary(backupPath); err != nil {
		os.Remove(backupPath)
		return "", fmt.Errorf("backup validation failed: %v", err)
	}

	return backupPath, nil
}

// installBinary installs the new binary, replacing the current one
func (ui *UpdateInstaller) installBinary(newBinaryPath string) (string, error) {
	targetPath := ui.currentBinary

	// On Windows, we may need to rename the current binary first
	if runtime.GOOS == "windows" {
		tempOldPath := ui.currentBinary + ".old"
		if err := os.Rename(ui.currentBinary, tempOldPath); err != nil {
			return "", fmt.Errorf("failed to rename current binary: %v", err)
		}
		defer os.Remove(tempOldPath) // Clean up the old binary
	}

	// Copy new binary to target location
	if err := ui.copyFile(newBinaryPath, targetPath); err != nil {
		return "", fmt.Errorf("failed to install new binary: %v", err)
	}

	// Make sure it's executable on Unix-like systems
	if runtime.GOOS != "windows" {
		if err := os.Chmod(targetPath, 0755); err != nil {
			return "", fmt.Errorf("failed to make binary executable: %v", err)
		}
	}

	return targetPath, nil
}

// verifyInstallation verifies that the installation was successful
func (ui *UpdateInstaller) verifyInstallation(binaryPath string) error {
	// Validate the installed binary
	if err := ui.validateBinary(binaryPath); err != nil {
		return err
	}

	// Check that it's in the correct location
	if binaryPath != ui.currentBinary {
		return fmt.Errorf("binary not installed in expected location")
	}

	return nil
}

// Rollback restores a previous version from backup
func (ui *UpdateInstaller) Rollback(backupPath string) error {
	ui.logStep("rollback_start", "progress", fmt.Sprintf("Starting rollback from %s", backupPath))

	// Validate backup exists and is executable
	if err := ui.validateBinary(backupPath); err != nil {
		return fmt.Errorf("backup validation failed: %v", err)
	}

	// On Windows, rename current binary first
	var tempCurrentPath string
	if runtime.GOOS == "windows" {
		tempCurrentPath = ui.currentBinary + ".rollback"
		if err := os.Rename(ui.currentBinary, tempCurrentPath); err != nil {
			return fmt.Errorf("failed to prepare for rollback: %v", err)
		}
	}

	// Copy backup to current location
	if err := ui.copyFile(backupPath, ui.currentBinary); err != nil {
		// If rollback fails and we're on Windows, try to restore
		if runtime.GOOS == "windows" && tempCurrentPath != "" {
			os.Rename(tempCurrentPath, ui.currentBinary)
		}
		return fmt.Errorf("failed to restore backup: %v", err)
	}

	// Make executable on Unix-like systems
	if runtime.GOOS != "windows" {
		if err := os.Chmod(ui.currentBinary, 0755); err != nil {
			return fmt.Errorf("failed to make restored binary executable: %v", err)
		}
	}

	// Clean up temporary files
	if tempCurrentPath != "" {
		os.Remove(tempCurrentPath)
	}

	ui.logStep("rollback_complete", "success", "Rollback completed successfully")
	return nil
}

// copyFile copies a file from src to dst
func (ui *UpdateInstaller) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Create destination directory if needed
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	// Copy file permissions
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}

// cleanup removes temporary files and directories
func (ui *UpdateInstaller) cleanup(paths ...string) {
	for _, path := range paths {
		if path != "" {
			if info, err := os.Stat(path); err == nil {
				if info.IsDir() {
					os.RemoveAll(path)
				} else {
					os.Remove(path)
				}
			}
		}
	}
}

// logStep adds an entry to the installation log
func (ui *UpdateInstaller) logStep(step, status, message string) {
	ui.mutex.Lock()
	defer ui.mutex.Unlock()

	entry := InstallLogEntry{
		Timestamp: time.Now(),
		Step:      step,
		Status:    status,
		Message:   message,
	}

	ui.installLog = append(ui.installLog, entry)
}

// getLogEntries returns a copy of the installation log
func (ui *UpdateInstaller) getLogEntries() []InstallLogEntry {
	ui.mutex.RLock()
	defer ui.mutex.RUnlock()

	entries := make([]InstallLogEntry, len(ui.installLog))
	copy(entries, ui.installLog)
	return entries
}

// GetBackupInfo returns information about available backups
func (ui *UpdateInstaller) GetBackupInfo() ([]BackupInfo, error) {
	files, err := os.ReadDir(ui.backupDir)
	if err != nil {
		return nil, err
	}

	var backups []BackupInfo
	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), "delta_") {
			info, err := file.Info()
			if err != nil {
				continue
			}

			backupPath := filepath.Join(ui.backupDir, file.Name())
			backup := BackupInfo{
				BackupPath:   backupPath,
				OriginalPath: ui.currentBinary,
				BackupTime:   info.ModTime(),
				Size:         info.Size(),
			}

			// Try to extract version from filename
			parts := strings.Split(file.Name(), "_")
			if len(parts) >= 2 {
				backup.Version = parts[1]
			}

			backups = append(backups, backup)
		}
	}

	return backups, nil
}

// CleanupOldBackups removes old backup files, keeping only the specified number
func (ui *UpdateInstaller) CleanupOldBackups(keepCount int) error {
	if keepCount <= 0 {
		keepCount = 5 // Keep 5 backups by default
	}

	backups, err := ui.GetBackupInfo()
	if err != nil {
		return err
	}

	if len(backups) <= keepCount {
		return nil // Nothing to clean up
	}

	// Sort backups by time (newest first)
	for i := 0; i < len(backups)-1; i++ {
		for j := i + 1; j < len(backups); j++ {
			if backups[i].BackupTime.Before(backups[j].BackupTime) {
				backups[i], backups[j] = backups[j], backups[i]
			}
		}
	}

	// Remove old backups
	for i := keepCount; i < len(backups); i++ {
		if err := os.Remove(backups[i].BackupPath); err != nil {
			fmt.Printf("Warning: failed to remove old backup %s: %v\n", backups[i].BackupPath, err)
		}
	}

	return nil
}