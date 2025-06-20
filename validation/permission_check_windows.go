//go:build windows
// +build windows

package validation

import (
	"os"
	"os/user"
)

// canWrite checks if current user can write to a path (Windows implementation)
func canWrite(path string, info os.FileInfo, currentUser *user.User) bool {
	// On Windows, we'll use a simplified check
	// Try to open the file/directory for writing
	
	if info.IsDir() {
		// For directories, try to create a temp file
		testFile := path + "\\.delta_write_test"
		file, err := os.Create(testFile)
		if err != nil {
			return false
		}
		file.Close()
		os.Remove(testFile)
		return true
	}
	
	// For files, check if we can open it for writing
	file, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return false
	}
	file.Close()
	return true
}

// isRoot checks if running as root (Windows implementation)
func isRoot() bool {
	// On Windows, check if running as Administrator
	// This is a simplified check - Windows doesn't have a direct equivalent to Unix root
	// For now, we'll return false as Windows permission model is different
	return false
}