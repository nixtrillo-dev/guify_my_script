package endpoint_utils

import (
	"encoding/json"
	"os"

	"guify_my_script/logger"
)

// DirectoryStatus represents the JSON structure for the directory check.
type DirectoryStatus struct {
	IsValid bool `json:"is_valid"`
}

// IsValidDirectory checks if a path is a valid directory, logs the result, and returns a JSON string.
func IsValidDirectory(path string, logLevel logger.LogLevel) string {
	isValid := false
	info, err := os.Stat(path)

	// Check if the path exists and is specifically a directory
	if err == nil && info.IsDir() {
		isValid = true
		logger.Debug(logLevel, "Validation passed: path is a valid directory -> %s", path)
	} else {
		logger.Debug(logLevel, "Validation failed: path is invalid or not a directory -> %s", path)
	}

	status := DirectoryStatus{IsValid: isValid}
	jsonBytes, err := json.Marshal(status)

	var jsonString string
	if err != nil {
		logger.Debug(logLevel, "JSON marshaling failed for directory validation: %v", err)
		jsonString = `{"is_valid": false}`
	} else {
		jsonString = string(jsonBytes)
	}

	logger.Info(logLevel, "%s", jsonString)
	return jsonString
}

// GetDirectoryListIfValid checks if a path is a valid directory and returns a JSON array containing a list of valid directory based projects.
func GetDirectoryListIfValid(path string, logLevel logger.LogLevel) string {
	info, err := os.Stat(path)

	// Initialize an empty slice to ensure an empty JSON array "[]" instead of "null" on failure
	result := []string{}

	if err == nil && info.IsDir() {
		logger.Debug(logLevel, "Directory valid, building GO array for path -> %s", path)
		result = append(result, "GO")
	} else {
		logger.Debug(logLevel, "Directory invalid or not found, returning empty array -> %s", path)
	}

	jsonBytes, err := json.Marshal(result)

	var jsonString string
	if err != nil {
		logger.Debug(logLevel, "JSON marshaling failed for array generation: %v", err)
		jsonString = `[]`
	} else {
		jsonString = string(jsonBytes)
	}

	logger.Info(logLevel, "%s", jsonString)
	return jsonString
}
