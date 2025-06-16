package main

import (
	"fmt"
	"runtime"
)

// Version information - injected at build time via ldflags
var (
	Version   = "v0.4.1-alpha"  // Overridden by -ldflags "-X main.Version=..."
	GitCommit = "unknown"  // Overridden by -ldflags "-X main.GitCommit=..."
	BuildDate = "unknown"  // Overridden by -ldflags "-X main.BuildDate=..."
	GoVersion = runtime.Version()
	IsDirty   = "true"     // Overridden by -ldflags "-X main.IsDirty=..." (default to dev build)
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

// IsDevelopmentBuild returns true if this is a development build
func IsDevelopmentBuild() bool {
	return IsDirty == "true" || GitCommit == "unknown" || BuildDate == "unknown"
}

// GetDevelopmentStatus returns information about the build status
func GetDevelopmentStatus() map[string]interface{} {
	return map[string]interface{}{
		"is_development": IsDevelopmentBuild(),
		"is_dirty":       IsDirty == "true",
		"git_commit":     GitCommit,
		"build_date":     BuildDate,
		"version":        Version,
	}
}
