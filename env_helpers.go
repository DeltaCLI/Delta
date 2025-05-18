package main

import (
	"os"
	"strconv"
	"strings"
)

// envVarPrefix is the prefix for all Delta CLI environment variables
const envVarPrefix = "DELTA_"

// getEnvBool retrieves a boolean value from an environment variable
// If the variable is not set, returns the defaultValue
func getEnvBool(name string, defaultValue bool) bool {
	val := os.Getenv(name)
	if val == "" {
		return defaultValue
	}
	val = strings.ToLower(val)
	return val == "true" || val == "1" || val == "yes"
}

// getEnvInt retrieves an integer value from an environment variable
// If the variable is not set or invalid, returns the defaultValue
func getEnvInt(name string, defaultValue int) int {
	val := os.Getenv(name)
	if val == "" {
		return defaultValue
	}
	intVal, err := strconv.Atoi(val)
	if err != nil {
		return defaultValue
	}
	return intVal
}

// getEnvFloat retrieves a float value from an environment variable
// If the variable is not set or invalid, returns the defaultValue
func getEnvFloat(name string, defaultValue float64) float64 {
	val := os.Getenv(name)
	if val == "" {
		return defaultValue
	}
	floatVal, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return defaultValue
	}
	return floatVal
}

// getEnvString retrieves a string value from an environment variable
// If the variable is not set, returns the defaultValue
func getEnvString(name string, defaultValue string) string {
	val := os.Getenv(name)
	if val == "" {
		return defaultValue
	}
	return val
}

// listDeltaEnvVars returns a map of all Delta CLI environment variables
func listDeltaEnvVars() map[string]string {
	result := make(map[string]string)
	
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 && strings.HasPrefix(parts[0], envVarPrefix) {
			result[parts[0]] = parts[1]
		}
	}
	
	return result
}