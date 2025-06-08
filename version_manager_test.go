package main

import (
	"testing"
)

// Test version parsing functionality
func TestParseVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected SemVersion
		hasError bool
	}{
		{"1.2.3", SemVersion{Major: 1, Minor: 2, Patch: 3, Original: "1.2.3"}, false},
		{"v1.2.3", SemVersion{Major: 1, Minor: 2, Patch: 3, Original: "v1.2.3"}, false},
		{"1.2.3-alpha", SemVersion{Major: 1, Minor: 2, Patch: 3, Prerelease: "alpha", Original: "1.2.3-alpha"}, false},
		{"1.2.3-alpha.1", SemVersion{Major: 1, Minor: 2, Patch: 3, Prerelease: "alpha.1", Original: "1.2.3-alpha.1"}, false},
		{"1.2.3+build", SemVersion{Major: 1, Minor: 2, Patch: 3, Build: "build", Original: "1.2.3+build"}, false},
		{"1.2.3-alpha+build", SemVersion{Major: 1, Minor: 2, Patch: 3, Prerelease: "alpha", Build: "build", Original: "1.2.3-alpha+build"}, false},
		{"invalid", SemVersion{}, true},
		{"", SemVersion{}, true},
	}

	for _, test := range tests {
		result, err := ParseVersion(test.input)
		
		if test.hasError {
			if err == nil {
				t.Errorf("Expected error for input '%s', but got none", test.input)
			}
			continue
		}
		
		if err != nil {
			t.Errorf("Unexpected error for input '%s': %v", test.input, err)
			continue
		}
		
		if result.Major != test.expected.Major || 
		   result.Minor != test.expected.Minor || 
		   result.Patch != test.expected.Patch ||
		   result.Prerelease != test.expected.Prerelease ||
		   result.Build != test.expected.Build {
			t.Errorf("For input '%s', expected %+v, got %+v", test.input, test.expected, *result)
		}
	}
}

// Test version comparison functionality
func TestCompareVersions(t *testing.T) {
	tests := []struct {
		v1       string
		v2       string
		expected int
	}{
		{"1.0.0", "1.0.1", -1}, // v1 < v2
		{"1.0.1", "1.0.0", 1},  // v1 > v2
		{"1.0.0", "1.0.0", 0},  // v1 == v2
		{"1.0.0", "2.0.0", -1}, // v1 < v2 (major)
		{"2.0.0", "1.0.0", 1},  // v1 > v2 (major)
		{"1.1.0", "1.0.0", 1},  // v1 > v2 (minor)
		{"1.0.0", "1.1.0", -1}, // v1 < v2 (minor)
		{"1.0.0", "1.0.0-alpha", 1},   // release > prerelease
		{"1.0.0-alpha", "1.0.0", -1},  // prerelease < release
		{"1.0.0-alpha", "1.0.0-beta", -1}, // alpha < beta
		{"1.0.0-beta", "1.0.0-alpha", 1},  // beta > alpha
	}

	for _, test := range tests {
		result := CompareVersions(test.v1, test.v2)
		if result != test.expected {
			t.Errorf("CompareVersions('%s', '%s') = %d, expected %d", test.v1, test.v2, result, test.expected)
		}
	}
}

// Test IsNewerVersion functionality
func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		current  string
		candidate string
		expected bool
	}{
		{"1.0.0", "1.0.1", true},   // candidate is newer
		{"1.0.1", "1.0.0", false},  // candidate is older
		{"1.0.0", "1.0.0", false},  // same version
		{"1.0.0-alpha", "1.0.0", true}, // release is newer than prerelease
		{"1.0.0", "2.0.0", true},   // major version bump
		{"1.0.0", "1.1.0", true},   // minor version bump
	}

	for _, test := range tests {
		result := IsNewerVersion(test.current, test.candidate)
		if result != test.expected {
			t.Errorf("IsNewerVersion('%s', '%s') = %t, expected %t", test.current, test.candidate, result, test.expected)
		}
	}
}

// Test MatchesChannel functionality
func TestMatchesChannel(t *testing.T) {
	tests := []struct {
		version  string
		channel  string
		expected bool
	}{
		{"1.0.0", "stable", true},
		{"1.0.0-alpha", "stable", false},
		{"1.0.0-alpha", "alpha", true},
		{"1.0.0-beta", "beta", true},
		{"1.0.0-beta", "alpha", false},
		{"1.0.0", "alpha", false},
		{"invalid", "stable", false},
	}

	for _, test := range tests {
		result := MatchesChannel(test.version, test.channel)
		if result != test.expected {
			t.Errorf("MatchesChannel('%s', '%s') = %t, expected %t", test.version, test.channel, result, test.expected)
		}
	}
}

// Test GetVersionFromTag functionality
func TestGetVersionFromTag(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"v1.2.3", "1.2.3"},
		{"version-1.2.3", "1.2.3"},
		{"release-1.2.3", "1.2.3"},
		{"1.2.3", "1.2.3"},
		{"", ""},
	}

	for _, test := range tests {
		result := GetVersionFromTag(test.input)
		if result != test.expected {
			t.Errorf("GetVersionFromTag('%s') = '%s', expected '%s'", test.input, result, test.expected)
		}
	}
}

// Test IsValidVersion functionality
func TestIsValidVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"1.2.3", true},
		{"v1.2.3", true},
		{"1.2.3-alpha", true},
		{"1.2.3+build", true},
		{"invalid", false},
		{"", false},
		{"1.2", false},
		{"1", false},
	}

	for _, test := range tests {
		result := IsValidVersion(test.input)
		if result != test.expected {
			t.Errorf("IsValidVersion('%s') = %t, expected %t", test.input, result, test.expected)
		}
	}
}

// Test SemVersion methods
func TestSemVersionMethods(t *testing.T) {
	version, _ := ParseVersion("1.2.3-alpha.1+build123")
	
	if !version.IsPrerelease() {
		t.Error("Expected version to be prerelease")
	}
	
	if version.IsStable() {
		t.Error("Expected version to not be stable")
	}
	
	if version.GetChannel() != "alpha" {
		t.Errorf("Expected channel 'alpha', got '%s'", version.GetChannel())
	}
	
	expected := "1.2.3-alpha.1+build123"
	if version.String() != expected {
		t.Errorf("Expected string '%s', got '%s'", expected, version.String())
	}
	
	// Test stable version
	stableVersion, _ := ParseVersion("2.0.0")
	if stableVersion.IsPrerelease() {
		t.Error("Expected stable version to not be prerelease")
	}
	
	if !stableVersion.IsStable() {
		t.Error("Expected stable version to be stable")
	}
	
	if stableVersion.GetChannel() != "stable" {
		t.Errorf("Expected channel 'stable', got '%s'", stableVersion.GetChannel())
	}
}