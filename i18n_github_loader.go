package main

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// I18nGitHubLoader handles downloading i18n files from GitHub releases
type I18nGitHubLoader struct {
	httpClient   *http.Client
	deltaAPIURL  string
	cacheDir     string
	tempDir      string
}

// I18nReleaseInfo contains information about a release with i18n files
type I18nReleaseInfo struct {
	Version         string
	I18nAssetURL    string
	ChecksumURL     string
	AssetName       string
	AssetSize       int64
	ExpectedSHA256  string
}

// I18nDownloadResult contains the result of an i18n download operation
type I18nDownloadResult struct {
	DownloadedLocales []string
	TotalFiles        int
	DownloadTime      time.Duration
	InstallPath       string
	Error             error
}

// NewI18nGitHubLoader creates a new GitHub-based i18n loader
func NewI18nGitHubLoader() *I18nGitHubLoader {
	// Get cache directory
	cacheDir, _ := os.UserCacheDir()
	if cacheDir == "" {
		cacheDir = "/tmp"
	}
	cacheDir = filepath.Join(cacheDir, "delta", "i18n")

	// Get temp directory
	tempDir := filepath.Join(os.TempDir(), "delta-i18n-github")

	return &I18nGitHubLoader{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		deltaAPIURL: "https://deltacli.com/api/github/latest-version",
		cacheDir:    cacheDir,
		tempDir:     tempDir,
	}
}

// GetLatestI18nRelease fetches information about the latest release with i18n files
func (gl *I18nGitHubLoader) GetLatestI18nRelease() (*I18nReleaseInfo, error) {
	// Get latest version from deltacli.com API
	version, err := gl.getLatestVersionFromDeltaAPI()
	if err != nil {
		return nil, fmt.Errorf("failed to get latest version: %v", err)
	}

	// Construct the release info based on the version
	// GitHub releases follow a predictable pattern
	assetName := fmt.Sprintf("delta-i18n-%s.tar.gz", version)
	
	return &I18nReleaseInfo{
		Version:      version,
		I18nAssetURL: fmt.Sprintf("https://github.com/deltacli/delta/releases/download/%s/%s", version, assetName),
		ChecksumURL:  fmt.Sprintf("https://github.com/deltacli/delta/releases/download/%s/checksums.sha256", version),
		AssetName:    assetName,
		AssetSize:    0, // We don't know the size without hitting GitHub API
	}, nil
}

// getLatestVersionFromDeltaAPI uses deltacli.com API to get latest version
func (gl *I18nGitHubLoader) getLatestVersionFromDeltaAPI() (string, error) {
	resp, err := gl.httpClient.Get(gl.deltaAPIURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("deltacli.com API returned status %d", resp.StatusCode)
	}

	var result struct {
		Version string `json:"version"`
		URL     string `json:"url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Version, nil
}


// DownloadI18nFilesFromRelease downloads i18n files from a specific release
func (gl *I18nGitHubLoader) DownloadI18nFilesFromRelease() (*I18nDownloadResult, error) {
	startTime := time.Now()
	result := &I18nDownloadResult{}

	// Get latest release info
	releaseInfo, err := gl.GetLatestI18nRelease()
	if err != nil {
		result.Error = fmt.Errorf("failed to get release info: %v", err)
		return result, result.Error
	}

	fmt.Printf("Found i18n files for Delta %s\n", releaseInfo.Version)

	// Clean up temp directory if it exists
	if err := os.RemoveAll(gl.tempDir); err != nil {
		// Non-fatal, continue
	}

	// Create temp directory
	if err := os.MkdirAll(gl.tempDir, 0755); err != nil {
		result.Error = fmt.Errorf("failed to create temp directory: %v", err)
		return result, result.Error
	}
	defer os.RemoveAll(gl.tempDir) // Clean up when done

	// Download checksums if available
	if releaseInfo.ChecksumURL != "" {
		fmt.Println("Downloading checksums for verification...")
		expectedSHA, err := gl.downloadAndParseChecksums(releaseInfo)
		if err != nil {
			fmt.Printf("Warning: Could not download checksums: %v\n", err)
			fmt.Println("Proceeding without checksum verification...")
		} else {
			releaseInfo.ExpectedSHA256 = expectedSHA
		}
	}

	// Download the i18n archive
	archivePath, err := gl.downloadI18nArchive(releaseInfo)
	if err != nil {
		result.Error = fmt.Errorf("failed to download i18n archive: %v", err)
		return result, result.Error
	}
	
	// Verify checksum if available
	if releaseInfo.ExpectedSHA256 != "" {
		fmt.Println("Verifying file integrity...")
		if err := gl.verifyChecksum(archivePath, releaseInfo.ExpectedSHA256); err != nil {
			result.Error = fmt.Errorf("checksum verification failed: %v", err)
			return result, result.Error
		}
		fmt.Println("✓ Checksum verified successfully")
	}

	// Extract i18n files
	extractedPath, err := gl.extractI18nArchive(archivePath)
	if err != nil {
		result.Error = fmt.Errorf("failed to extract i18n files: %v", err)
		return result, result.Error
	}

	// Get install path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		result.Error = fmt.Errorf("failed to get home directory: %v", err)
		return result, result.Error
	}

	installPath := filepath.Join(homeDir, ".config", "delta", "i18n", "locales")

	// Install the files
	locales, fileCount, err := gl.installExtractedFiles(extractedPath, installPath)
	if err != nil {
		result.Error = fmt.Errorf("failed to install i18n files: %v", err)
		return result, result.Error
	}

	result.DownloadedLocales = locales
	result.TotalFiles = fileCount
	result.DownloadTime = time.Since(startTime)
	result.InstallPath = installPath

	return result, nil
}

// downloadI18nArchive downloads the i18n archive from the release
func (gl *I18nGitHubLoader) downloadI18nArchive(releaseInfo *I18nReleaseInfo) (string, error) {
	req, err := http.NewRequest("GET", releaseInfo.I18nAssetURL, nil)
	if err != nil {
		return "", err
	}

	// Set user agent
	req.Header.Set("User-Agent", "Delta-CLI/"+GetVersionShort())

	// Download the file
	resp, err := gl.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	// Create temp file for archive
	archivePath := filepath.Join(gl.tempDir, releaseInfo.AssetName)
	out, err := os.Create(archivePath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	// Copy with progress reporting
	written, err := io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}

	fmt.Printf("Downloaded %d bytes\n", written)
	return archivePath, nil
}

// extractI18nArchive extracts the i18n archive
func (gl *I18nGitHubLoader) extractI18nArchive(archivePath string) (string, error) {
	// Open the archive
	file, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Create gzip reader
	gzr, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer gzr.Close()

	// Create tar reader
	tr := tar.NewReader(gzr)

	// Extract directory path
	extractPath := filepath.Join(gl.tempDir, "extracted")
	if err := os.MkdirAll(extractPath, 0755); err != nil {
		return "", err
	}

	// Extract files
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		// Skip if not a regular file
		if header.Typeflag != tar.TypeReg {
			continue
		}

		// Create the full path
		fullPath := filepath.Join(extractPath, header.Name)

		// Create directory if needed
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return "", err
		}

		// Create the file
		outFile, err := os.Create(fullPath)
		if err != nil {
			return "", err
		}

		// Copy file contents
		if _, err := io.Copy(outFile, tr); err != nil {
			outFile.Close()
			return "", err
		}
		outFile.Close()

		// Set file permissions
		if err := os.Chmod(fullPath, 0644); err != nil {
			// Non-fatal, continue
		}
	}

	return extractPath, nil
}

// installExtractedFiles installs extracted files to the target directory
func (gl *I18nGitHubLoader) installExtractedFiles(sourcePath, targetPath string) ([]string, int, error) {
	// Look for the locales directory in the extracted files
	localesPath := filepath.Join(sourcePath, "locales")
	if _, err := os.Stat(localesPath); os.IsNotExist(err) {
		// Try without locales prefix
		localesPath = sourcePath
	}

	// Create target directory
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		return nil, 0, err
	}

	locales := []string{}
	fileCount := 0

	// Walk through the extracted files
	err := filepath.Walk(localesPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get relative path from source
		relPath, err := filepath.Rel(localesPath, path)
		if err != nil {
			return err
		}

		// Create target path
		targetFile := filepath.Join(targetPath, relPath)

		// Create target directory if needed
		targetDir := filepath.Dir(targetFile)
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return err
		}

		// Copy the file
		if err := gl.copyFile(path, targetFile); err != nil {
			return err
		}

		fileCount++

		// Track locale
		parts := strings.Split(relPath, string(os.PathSeparator))
		if len(parts) > 0 && !i18nContains(locales, parts[0]) {
			locales = append(locales, parts[0])
		}

		return nil
	})

	return locales, fileCount, err
}

// copyFile copies a file from source to destination
func (gl *I18nGitHubLoader) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// LoadSingleLocaleFromGitHub loads a specific locale from GitHub
func (gl *I18nGitHubLoader) LoadSingleLocaleFromGitHub(locale string) (map[string]interface{}, error) {
	// Download all i18n files
	result, err := gl.DownloadI18nFilesFromRelease()
	if err != nil {
		return nil, err
	}

	// Check if the requested locale was downloaded
	if !i18nContains(result.DownloadedLocales, locale) {
		return nil, fmt.Errorf("locale %s not found in release", locale)
	}

	// Load the translation files for the locale
	localePath := filepath.Join(result.InstallPath, locale)
	translations := make(map[string]interface{})

	// Read all JSON files in the locale directory
	files, err := ioutil.ReadDir(localePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read locale directory: %v", err)
	}

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		filePath := filepath.Join(localePath, file.Name())
		content, err := ioutil.ReadFile(filePath)
		if err != nil {
			continue
		}

		var data map[string]interface{}
		if err := json.Unmarshal(content, &data); err != nil {
			continue
		}

		// Add to translations
		fileKey := strings.TrimSuffix(file.Name(), ".json")
		translations[fileKey] = data
	}

	return translations, nil
}

// downloadAndParseChecksums downloads and parses the checksums file
func (gl *I18nGitHubLoader) downloadAndParseChecksums(releaseInfo *I18nReleaseInfo) (string, error) {
	req, err := http.NewRequest("GET", releaseInfo.ChecksumURL, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", "Delta-CLI/"+GetVersionShort())

	resp, err := gl.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download checksums: status %d", resp.StatusCode)
	}

	// Parse checksums file
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) == 2 {
			checksum, filename := parts[0], parts[1]
			// Look for our i18n file
			if filename == releaseInfo.AssetName || 
			   filename == "./"+releaseInfo.AssetName ||
			   strings.HasSuffix(filename, "/"+releaseInfo.AssetName) {
				return checksum, nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", fmt.Errorf("checksum not found for %s", releaseInfo.AssetName)
}

// verifyChecksum verifies the SHA256 checksum of a file
func (gl *I18nGitHubLoader) verifyChecksum(filepath string, expectedSHA256 string) error {
	file, err := os.Open(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return err
	}

	actualSHA256 := hex.EncodeToString(hasher.Sum(nil))
	if actualSHA256 != expectedSHA256 {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedSHA256, actualSHA256)
	}

	return nil
}

// calculateSHA256 calculates the SHA256 checksum of a file
func (gl *I18nGitHubLoader) calculateSHA256(filepath string) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// i18nContains checks if a string slice contains a value
func i18nContains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}