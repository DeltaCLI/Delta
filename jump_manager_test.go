package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestJumpManager(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "jump-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test config file path
	configPath := filepath.Join(tempDir, "test_jump_locations.json")

	// Create a test jump manager
	jm := &JumpManager{
		locations:  make(map[string]string),
		configPath: configPath,
	}

	// Test adding locations
	t.Run("AddLocation", func(t *testing.T) {
		// Add a location
		err := jm.AddLocation("test", "/test/path")
		if err != nil {
			t.Fatalf("Failed to add location: %v", err)
		}

		// Verify location was added
		if path, exists := jm.locations["test"]; !exists || path != "/test/path" {
			t.Errorf("Location not added correctly, got: %s, want: %s", path, "/test/path")
		}

		// Verify config file was created
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Errorf("Config file was not created: %v", err)
		}
	})

	// Test listing locations
	t.Run("ListLocations", func(t *testing.T) {
		// Add another location
		err := jm.AddLocation("home", "/home/user")
		if err != nil {
			t.Fatalf("Failed to add location: %v", err)
		}

		// List locations
		locations := jm.ListLocations()
		if len(locations) != 2 {
			t.Errorf("Expected 2 locations, got: %d", len(locations))
		}

		// Verify the locations are in alphabetical order
		if locations[0] != "home" || locations[1] != "test" {
			t.Errorf("Locations not in alphabetical order: %v", locations)
		}
	})

	// Test removing a location
	t.Run("RemoveLocation", func(t *testing.T) {
		// Remove a location
		err := jm.RemoveLocation("test")
		if err != nil {
			t.Fatalf("Failed to remove location: %v", err)
		}

		// Verify location was removed
		if _, exists := jm.locations["test"]; exists {
			t.Errorf("Location was not removed")
		}

		// Verify only one location remains
		locations := jm.ListLocations()
		if len(locations) != 1 || locations[0] != "home" {
			t.Errorf("Expected only 'home' location, got: %v", locations)
		}
	})

	// Test loading saved locations
	t.Run("LoadLocations", func(t *testing.T) {
		// Create a new jump manager with the same config path
		newJm := &JumpManager{
			locations:  make(map[string]string),
			configPath: configPath,
		}

		// Load locations
		err := newJm.loadLocations()
		if err != nil {
			t.Fatalf("Failed to load locations: %v", err)
		}

		// Verify the location was loaded
		if path, exists := newJm.locations["home"]; !exists || path != "/home/user" {
			t.Errorf("Location not loaded correctly, got: %s, want: %s", path, "/home/user")
		}
	})
}
