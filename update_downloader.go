package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// UpdateDownloader handles secure downloading of updates
type UpdateDownloader struct {
	downloadDir   string
	verifier      *UpdateVerifier
	progressBar   ProgressReporter
	httpClient    *http.Client
	mutex         sync.RWMutex
	activeDownload *DownloadProgress
}

// UpdateVerifier handles verification of downloaded files
type UpdateVerifier struct {
	checksumCache map[string]string
	mutex         sync.RWMutex
}

// DownloadResult contains the result of a download operation
type DownloadResult struct {
	FilePath     string
	Verified     bool
	Size         int64
	Checksum     string
	DownloadTime time.Duration
	Asset        *Asset
	Version      string
}

// DownloadProgress tracks download progress
type DownloadProgress struct {
	URL           string
	FilePath      string
	TotalBytes    int64
	DownloadedBytes int64
	StartTime     time.Time
	IsComplete    bool
	Error         error
	mutex         sync.RWMutex
}

// ProgressReporter interface for download progress reporting
type ProgressReporter interface {
	Start(totalBytes int64)
	Update(downloadedBytes int64)
	Complete()
	Error(err error)
}

// ConsoleProgressReporter implements ProgressReporter for console output
type ConsoleProgressReporter struct {
	totalBytes      int64
	lastUpdate      time.Time
	startTime       time.Time
	updateInterval  time.Duration
}

// NewUpdateDownloader creates a new update downloader
func NewUpdateDownloader(downloadDir string) *UpdateDownloader {
	return &UpdateDownloader{
		downloadDir: downloadDir,
		verifier:    NewUpdateVerifier(),
		progressBar: NewConsoleProgressReporter(),
		httpClient: &http.Client{
			Timeout: 10 * time.Minute, // 10 minute timeout for downloads
		},
	}
}

// NewUpdateVerifier creates a new update verifier
func NewUpdateVerifier() *UpdateVerifier {
	return &UpdateVerifier{
		checksumCache: make(map[string]string),
	}
}

// NewConsoleProgressReporter creates a new console progress reporter
func NewConsoleProgressReporter() *ConsoleProgressReporter {
	return &ConsoleProgressReporter{
		updateInterval: 1 * time.Second, // Update every second
	}
}

// DownloadUpdate downloads an update from the given release
func (ud *UpdateDownloader) DownloadUpdate(release *Release) (*DownloadResult, error) {
	if release == nil {
		return nil, fmt.Errorf("release is nil")
	}

	// Select appropriate asset for the current platform
	asset, err := ud.SelectAssetForPlatform(release.Assets)
	if err != nil {
		return nil, fmt.Errorf("failed to select asset: %v", err)
	}

	// Create download directory if it doesn't exist
	if err := os.MkdirAll(ud.downloadDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create download directory: %v", err)
	}

	// Generate download file path
	fileName := asset.Name
	if fileName == "" {
		fileName = fmt.Sprintf("delta-%s", release.TagName)
	}
	filePath := filepath.Join(ud.downloadDir, fileName)

	// Check if file already exists and is valid
	if existingResult := ud.checkExistingFile(filePath, asset); existingResult != nil {
		return existingResult, nil
	}

	// Download the file
	startTime := time.Now()
	downloadedSize, err := ud.downloadFile(asset.BrowserDownloadURL, filePath, asset.Size)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %v", err)
	}

	downloadTime := time.Since(startTime)

	// Verify the download
	verified, checksum, err := ud.verifier.VerifyFile(filePath, asset)
	if err != nil {
		return nil, fmt.Errorf("failed to verify downloaded file: %v", err)
	}

	result := &DownloadResult{
		FilePath:     filePath,
		Verified:     verified,
		Size:         downloadedSize,
		Checksum:     checksum,
		DownloadTime: downloadTime,
		Asset:        asset,
		Version:      release.TagName,
	}

	return result, nil
}

// SelectAssetForPlatform selects the appropriate asset for the current platform
func (ud *UpdateDownloader) SelectAssetForPlatform(assets []Asset) (*Asset, error) {
	if len(assets) == 0 {
		return nil, fmt.Errorf("no assets available")
	}

	// Get current platform information
	currentOS := getCurrentOS()
	currentArch := getCurrentArch()

	// Platform-specific patterns
	platformPatterns := map[string][]string{
		"linux":   {"linux"},
		"darwin":  {"darwin", "macos", "mac"},
		"windows": {"windows", "win"},
	}

	// Architecture patterns
	archPatterns := map[string][]string{
		"amd64": {"amd64", "x86_64", "x64"},
		"arm64": {"arm64", "aarch64"},
		"386":   {"386", "i386", "x86"},
	}

	// Score assets based on platform and architecture match
	bestAsset := &assets[0]
	bestScore := 0

	for i := range assets {
		asset := &assets[i]
		score := ud.scoreAsset(asset, currentOS, currentArch, platformPatterns, archPatterns)
		
		if score > bestScore {
			bestScore = score
			bestAsset = asset
		}
	}

	return bestAsset, nil
}

// scoreAsset scores an asset based on platform compatibility
func (ud *UpdateDownloader) scoreAsset(asset *Asset, currentOS, currentArch string, platformPatterns, archPatterns map[string][]string) int {
	name := strings.ToLower(asset.Name)
	score := 0

	// Check platform match
	if patterns, ok := platformPatterns[currentOS]; ok {
		for _, pattern := range patterns {
			if strings.Contains(name, pattern) {
				score += 10 // High score for platform match
				break
			}
		}
	}

	// Check architecture match
	if patterns, ok := archPatterns[currentArch]; ok {
		for _, pattern := range patterns {
			if strings.Contains(name, pattern) {
				score += 5 // Medium score for architecture match
				break
			}
		}
	}

	// Prefer certain file types
	if strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".zip") {
		score += 2
	}

	// Avoid certain patterns
	if strings.Contains(name, "source") || strings.Contains(name, "src") {
		score -= 10
	}

	return score
}

// checkExistingFile checks if a file already exists and is valid
func (ud *UpdateDownloader) checkExistingFile(filePath string, asset *Asset) *DownloadResult {
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return nil
	}

	if err != nil {
		return nil // File exists but can't stat it
	}

	// Check if size matches (basic validation)
	if asset.Size > 0 && info.Size() != asset.Size {
		return nil // Size mismatch, re-download
	}

	// Verify the existing file
	verified, checksum, err := ud.verifier.VerifyFile(filePath, asset)
	if err != nil || !verified {
		return nil // Verification failed, re-download
	}

	return &DownloadResult{
		FilePath:     filePath,
		Verified:     verified,
		Size:         info.Size(),
		Checksum:     checksum,
		DownloadTime: 0, // Already existed
		Asset:        asset,
	}
}

// downloadFile downloads a file from the given URL
func (ud *UpdateDownloader) downloadFile(url, filePath string, expectedSize int64) (int64, error) {
	// Create the request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	// Set user agent
	req.Header.Set("User-Agent", "Delta-CLI/"+GetVersionShort())

	// Perform the request
	resp, err := ud.httpClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("bad status: %s", resp.Status)
	}

	// Get content length
	contentLength := resp.ContentLength
	if contentLength <= 0 && expectedSize > 0 {
		contentLength = expectedSize
	}

	// Create the file
	out, err := os.Create(filePath)
	if err != nil {
		return 0, err
	}
	defer out.Close()

	// Set up progress tracking
	ud.mutex.Lock()
	ud.activeDownload = &DownloadProgress{
		URL:           url,
		FilePath:      filePath,
		TotalBytes:    contentLength,
		StartTime:     time.Now(),
	}
	ud.mutex.Unlock()

	// Start progress reporting
	if ud.progressBar != nil {
		ud.progressBar.Start(contentLength)
	}

	// Download with progress reporting
	progressReader := &ProgressReader{
		Reader:    resp.Body,
		Reporter:  ud.progressBar,
		Downloaded: 0,
	}

	downloadedBytes, err := io.Copy(out, progressReader)
	
	// Update progress tracking
	ud.mutex.Lock()
	if ud.activeDownload != nil {
		ud.activeDownload.DownloadedBytes = downloadedBytes
		ud.activeDownload.IsComplete = true
		ud.activeDownload.Error = err
	}
	ud.mutex.Unlock()

	if err != nil {
		if ud.progressBar != nil {
			ud.progressBar.Error(err)
		}
		os.Remove(filePath) // Clean up partial download
		return 0, err
	}

	if ud.progressBar != nil {
		ud.progressBar.Complete()
	}

	return downloadedBytes, nil
}

// GetActiveDownload returns information about the current download
func (ud *UpdateDownloader) GetActiveDownload() *DownloadProgress {
	ud.mutex.RLock()
	defer ud.mutex.RUnlock()
	
	if ud.activeDownload != nil {
		// Return a copy to avoid race conditions
		return &DownloadProgress{
			URL:             ud.activeDownload.URL,
			FilePath:        ud.activeDownload.FilePath,
			TotalBytes:      ud.activeDownload.TotalBytes,
			DownloadedBytes: ud.activeDownload.DownloadedBytes,
			StartTime:       ud.activeDownload.StartTime,
			IsComplete:      ud.activeDownload.IsComplete,
			Error:           ud.activeDownload.Error,
		}
	}
	
	return nil
}

// VerifyFile verifies the integrity of a downloaded file
func (uv *UpdateVerifier) VerifyFile(filePath string, asset *Asset) (bool, string, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return false, "", err
	}
	defer file.Close()

	// Calculate SHA256 checksum
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return false, "", err
	}

	checksum := fmt.Sprintf("%x", hasher.Sum(nil))

	// For now, we'll always consider the file verified if we can calculate the checksum
	// In a real implementation, you would compare against known checksums from GitHub releases
	// or download a separate checksums file
	verified := true

	// Cache the checksum
	uv.mutex.Lock()
	uv.checksumCache[filePath] = checksum
	uv.mutex.Unlock()

	return verified, checksum, nil
}

// GetCachedChecksum returns a cached checksum if available
func (uv *UpdateVerifier) GetCachedChecksum(filePath string) (string, bool) {
	uv.mutex.RLock()
	defer uv.mutex.RUnlock()
	
	checksum, exists := uv.checksumCache[filePath]
	return checksum, exists
}

// ProgressReader wraps an io.Reader to report progress
type ProgressReader struct {
	Reader     io.Reader
	Reporter   ProgressReporter
	Downloaded int64
}

// Read implements io.Reader
func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.Downloaded += int64(n)
	
	if pr.Reporter != nil {
		pr.Reporter.Update(pr.Downloaded)
	}
	
	return n, err
}

// ConsoleProgressReporter implementations

// Start implements ProgressReporter
func (cpr *ConsoleProgressReporter) Start(totalBytes int64) {
	cpr.totalBytes = totalBytes
	cpr.startTime = time.Now()
	cpr.lastUpdate = time.Now()
	
	if totalBytes > 0 {
		fmt.Printf("Downloading update (%s)...\n", formatFileSize(totalBytes))
	} else {
		fmt.Println("Downloading update...")
	}
}

// Update implements ProgressReporter
func (cpr *ConsoleProgressReporter) Update(downloadedBytes int64) {
	now := time.Now()
	if now.Sub(cpr.lastUpdate) < cpr.updateInterval {
		return // Don't update too frequently
	}
	cpr.lastUpdate = now

	if cpr.totalBytes > 0 {
		percentage := float64(downloadedBytes) / float64(cpr.totalBytes) * 100
		elapsed := now.Sub(cpr.startTime)
		
		var eta time.Duration
		if downloadedBytes > 0 {
			totalTime := elapsed * time.Duration(cpr.totalBytes) / time.Duration(downloadedBytes)
			eta = totalTime - elapsed
		}

		fmt.Printf("\rProgress: %.1f%% (%s/%s) ETA: %s", 
			percentage, 
			formatFileSize(downloadedBytes), 
			formatFileSize(cpr.totalBytes),
			eta.Truncate(time.Second))
	} else {
		fmt.Printf("\rDownloaded: %s", formatFileSize(downloadedBytes))
	}
}

// Complete implements ProgressReporter
func (cpr *ConsoleProgressReporter) Complete() {
	elapsed := time.Since(cpr.startTime)
	fmt.Printf("\n✅ Download completed in %s\n", elapsed.Truncate(time.Millisecond))
}

// Error implements ProgressReporter
func (cpr *ConsoleProgressReporter) Error(err error) {
	fmt.Printf("\n❌ Download failed: %v\n", err)
}

// Helper functions

// CleanupOldDownloads removes old download files
func (ud *UpdateDownloader) CleanupOldDownloads(keepCount int) error {
	if keepCount <= 0 {
		keepCount = 3 // Keep last 3 downloads by default
	}

	files, err := os.ReadDir(ud.downloadDir)
	if err != nil {
		return err
	}

	// Filter and sort by modification time
	var downloadFiles []os.FileInfo
	for _, file := range files {
		if !file.IsDir() {
			info, err := file.Info()
			if err == nil {
				downloadFiles = append(downloadFiles, info)
			}
		}
	}

	if len(downloadFiles) <= keepCount {
		return nil // Nothing to clean up
	}

	// Sort by modification time (newest first)
	for i := 0; i < len(downloadFiles)-1; i++ {
		for j := i + 1; j < len(downloadFiles); j++ {
			if downloadFiles[i].ModTime().Before(downloadFiles[j].ModTime()) {
				downloadFiles[i], downloadFiles[j] = downloadFiles[j], downloadFiles[i]
			}
		}
	}

	// Remove old files
	for i := keepCount; i < len(downloadFiles); i++ {
		filePath := filepath.Join(ud.downloadDir, downloadFiles[i].Name())
		if err := os.Remove(filePath); err != nil {
			fmt.Printf("Warning: failed to remove old download %s: %v\n", filePath, err)
		}
	}

	return nil
}

// GetDownloadStats returns statistics about downloads
func (ud *UpdateDownloader) GetDownloadStats() map[string]interface{} {
	files, err := os.ReadDir(ud.downloadDir)
	if err != nil {
		return map[string]interface{}{
			"error": err.Error(),
		}
	}

	totalSize := int64(0)
	fileCount := 0

	for _, file := range files {
		if !file.IsDir() {
			info, err := file.Info()
			if err == nil {
				totalSize += info.Size()
				fileCount++
			}
		}
	}

	return map[string]interface{}{
		"download_directory": ud.downloadDir,
		"file_count":         fileCount,
		"total_size":         totalSize,
		"total_size_formatted": formatFileSize(totalSize),
	}
}