package main

import (
	"archive/tar"
	"compress/gzip"
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

// I18nRemoteLoader handles downloading i18n files from GitHub
type I18nRemoteLoader struct {
	httpClient     *http.Client
	githubBaseURL  string
	cacheDir       string
	tempDir        string
}

// I18nDownloadResult contains the result of an i18n download operation
type I18nDownloadResult struct {
	DownloadedLocales []string
	TotalFiles        int
	DownloadTime      time.Duration
	InstallPath       string
	Error             error
}

// NewI18nRemoteLoader creates a new i18n remote loader
func NewI18nRemoteLoader() *I18nRemoteLoader {
	// Get cache directory
	cacheDir, _ := os.UserCacheDir()
	if cacheDir == "" {
		cacheDir = "/tmp"
	}
	cacheDir = filepath.Join(cacheDir, "delta", "i18n")

	// Get temp directory
	tempDir := filepath.Join(os.TempDir(), "delta-i18n-download")

	return &I18nRemoteLoader{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		githubBaseURL: "https://github.com/deltacli/delta/archive",
		cacheDir:      cacheDir,
		tempDir:       tempDir,
	}
}

// DownloadI18nFiles downloads i18n files from GitHub for the current version
func (rl *I18nRemoteLoader) DownloadI18nFiles() (*I18nDownloadResult, error) {
	startTime := time.Now()
	result := &I18nDownloadResult{}

	// Get the current version
	version := GetVersionShort()
	if version == "dev" || strings.Contains(version, "dev") {
		// For development builds, use main branch
		version = "main"
	}

	// Clean up temp directory if it exists
	if err := os.RemoveAll(rl.tempDir); err != nil {
		// Non-fatal, continue
	}

	// Create temp directory
	if err := os.MkdirAll(rl.tempDir, 0755); err != nil {
		result.Error = fmt.Errorf("failed to create temp directory: %v", err)
		return result, result.Error
	}
	defer os.RemoveAll(rl.tempDir) // Clean up when done

	// Download the archive
	archivePath, err := rl.downloadArchive(version)
	if err != nil {
		result.Error = fmt.Errorf("failed to download archive: %v", err)
		return result, result.Error
	}

	// Extract i18n files
	extractedPath, err := rl.extractI18nFiles(archivePath)
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
	locales, fileCount, err := rl.installI18nFiles(extractedPath, installPath)
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

// downloadArchive downloads the repository archive from GitHub
func (rl *I18nRemoteLoader) downloadArchive(version string) (string, error) {
	// Construct download URL
	// For tags: https://github.com/deltacli/delta/archive/refs/tags/v0.4.2-alpha.tar.gz
	// For branch: https://github.com/deltacli/delta/archive/refs/heads/main.tar.gz
	var downloadURL string
	if version == "main" {
		downloadURL = fmt.Sprintf("%s/refs/heads/main.tar.gz", rl.githubBaseURL)
	} else {
		// Assume it's a tag, add 'v' prefix if not present
		if !strings.HasPrefix(version, "v") {
			version = "v" + version
		}
		downloadURL = fmt.Sprintf("%s/refs/tags/%s.tar.gz", rl.githubBaseURL, version)
	}

	// Create request
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		return "", err
	}

	// Set user agent
	req.Header.Set("User-Agent", "Delta-CLI/"+GetVersionShort())

	// Download the file
	resp, err := rl.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}

	// Create temp file for archive
	archivePath := filepath.Join(rl.tempDir, "delta-i18n.tar.gz")
	out, err := os.Create(archivePath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	// Copy the response body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}

	return archivePath, nil
}

// extractI18nFiles extracts i18n files from the downloaded archive
func (rl *I18nRemoteLoader) extractI18nFiles(archivePath string) (string, error) {
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
	extractPath := filepath.Join(rl.tempDir, "extracted")
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

		// Check if this is an i18n file
		if !strings.Contains(header.Name, "/i18n/locales/") {
			continue
		}

		// Skip if not a regular file
		if header.Typeflag != tar.TypeReg {
			continue
		}

		// Extract the relative path after "i18n/locales/"
		parts := strings.Split(header.Name, "/i18n/locales/")
		if len(parts) < 2 {
			continue
		}
		relativePath := parts[1]

		// Create the full path
		fullPath := filepath.Join(extractPath, relativePath)

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

// installI18nFiles installs extracted i18n files to the target directory
func (rl *I18nRemoteLoader) installI18nFiles(sourcePath, targetPath string) ([]string, int, error) {
	// Create target directory
	if err := os.MkdirAll(targetPath, 0755); err != nil {
		return nil, 0, err
	}

	locales := []string{}
	fileCount := 0

	// Walk through the extracted files
	err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Get relative path from source
		relPath, err := filepath.Rel(sourcePath, path)
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
		if err := rl.copyFile(path, targetFile); err != nil {
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
func (rl *I18nRemoteLoader) copyFile(src, dst string) error {
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

// DownloadSingleLocale downloads i18n files for a specific locale
func (rl *I18nRemoteLoader) DownloadSingleLocale(locale string) error {
	// For now, we download all locales and extract the one we need
	// In a future optimization, we could download individual files
	result, err := rl.DownloadI18nFiles()
	if err != nil {
		return err
	}

	// Check if the requested locale was downloaded
	if !i18nContains(result.DownloadedLocales, locale) {
		return fmt.Errorf("locale %s not found in remote repository", locale)
	}

	return nil
}

// LoadRemoteTranslation loads a translation directly from GitHub without installing
func (rl *I18nRemoteLoader) LoadRemoteTranslation(locale string) (map[string]interface{}, error) {
	// This is a simplified version that downloads and loads in memory
	// For production, you might want to cache these
	
	// First, try to download all i18n files
	result, err := rl.DownloadI18nFiles()
	if err != nil {
		return nil, err
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

// i18nContains checks if a string slice contains a value
func i18nContains(slice []string, value string) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

// CleanupCache removes old cached i18n files
func (rl *I18nRemoteLoader) CleanupCache() error {
	return os.RemoveAll(rl.cacheDir)
}