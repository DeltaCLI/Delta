package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// SemVersion represents a semantic version
type SemVersion struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string
	Build      string
	Original   string
}

// ParseVersion parses a semantic version string
func ParseVersion(version string) (*SemVersion, error) {
	// Remove 'v' prefix if present
	version = strings.TrimPrefix(version, "v")
	
	// Regular expression for semantic versioning
	re := regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$`)
	
	matches := re.FindStringSubmatch(version)
	if matches == nil {
		return nil, fmt.Errorf("invalid semantic version: %s", version)
	}
	
	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])
	
	return &SemVersion{
		Major:      major,
		Minor:      minor,
		Patch:      patch,
		Prerelease: matches[4],
		Build:      matches[5],
		Original:   version,
	}, nil
}

// CompareVersions compares two version strings
// Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
func CompareVersions(v1, v2 string) int {
	ver1, err1 := ParseVersion(v1)
	ver2, err2 := ParseVersion(v2)
	
	if err1 != nil || err2 != nil {
		// Fall back to string comparison for invalid versions
		if v1 < v2 {
			return -1
		} else if v1 > v2 {
			return 1
		}
		return 0
	}
	
	// Compare major.minor.patch
	if ver1.Major != ver2.Major {
		if ver1.Major < ver2.Major {
			return -1
		}
		return 1
	}
	
	if ver1.Minor != ver2.Minor {
		if ver1.Minor < ver2.Minor {
			return -1
		}
		return 1
	}
	
	if ver1.Patch != ver2.Patch {
		if ver1.Patch < ver2.Patch {
			return -1
		}
		return 1
	}
	
	// Handle prerelease versions
	if ver1.Prerelease == "" && ver2.Prerelease != "" {
		return 1 // Release > prerelease
	}
	if ver1.Prerelease != "" && ver2.Prerelease == "" {
		return -1 // Prerelease < release
	}
	if ver1.Prerelease != "" && ver2.Prerelease != "" {
		if ver1.Prerelease < ver2.Prerelease {
			return -1
		} else if ver1.Prerelease > ver2.Prerelease {
			return 1
		}
	}
	
	return 0
}

// IsNewerVersion checks if candidate version is newer than current
func IsNewerVersion(current, candidate string) bool {
	return CompareVersions(current, candidate) < 0
}

// MatchesChannel checks if a version matches a release channel
func MatchesChannel(version string, channel string) bool {
	ver, err := ParseVersion(version)
	if err != nil {
		return false
	}
	
	switch channel {
	case "stable":
		return ver.Prerelease == ""
	case "alpha":
		return strings.Contains(ver.Prerelease, "alpha")
	case "beta":
		return strings.Contains(ver.Prerelease, "beta")
	default:
		return true // Allow all versions for unknown channels
	}
}

// GetVersionFromTag extracts version from a git tag
func GetVersionFromTag(tag string) string {
	// Remove common prefixes
	tag = strings.TrimPrefix(tag, "v")
	tag = strings.TrimPrefix(tag, "version-")
	tag = strings.TrimPrefix(tag, "release-")
	
	return tag
}

// IsValidVersion checks if a version string is valid semantic version
func IsValidVersion(version string) bool {
	_, err := ParseVersion(version)
	return err == nil
}

// String returns the string representation of a version
func (v *SemVersion) String() string {
	version := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	
	if v.Prerelease != "" {
		version += "-" + v.Prerelease
	}
	
	if v.Build != "" {
		version += "+" + v.Build
	}
	
	return version
}

// IsPrerelease returns true if this is a prerelease version
func (v *SemVersion) IsPrerelease() bool {
	return v.Prerelease != ""
}

// IsStable returns true if this is a stable release version
func (v *SemVersion) IsStable() bool {
	return v.Prerelease == ""
}

// GetChannel returns the channel this version belongs to
func (v *SemVersion) GetChannel() string {
	if v.Prerelease == "" {
		return "stable"
	}
	
	if strings.Contains(v.Prerelease, "alpha") {
		return "alpha"
	}
	
	if strings.Contains(v.Prerelease, "beta") {
		return "beta"
	}
	
	return "prerelease"
}