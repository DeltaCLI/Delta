package main

import (
	"fmt"
	"runtime"
)

// Version information - updated by build process
var (
	Version   = "v0.1.0-alpha"
	GitCommit = "unknown"
	BuildDate = "unknown"
	GoVersion = runtime.Version()
)

// GetVersionInfo returns formatted version information
func GetVersionInfo() string {
	return fmt.Sprintf(`Delta CLI %s
Git Commit: %s
Build Date: %s
Go Version: %s
Platform: %s/%s

🌍 Multilingual Support: 6 languages available
💡 AI-Powered Shell Enhancement with Local Privacy

Copyright (c) 2025 Source Parts Inc.
License: MIT`, Version, GitCommit, BuildDate, GoVersion, runtime.GOOS, runtime.GOARCH)
}

// GetVersionShort returns just the version string
func GetVersionShort() string {
	return Version
}
