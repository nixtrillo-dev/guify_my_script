package endpoint_utils

import (
	"encoding/json"
	"path/filepath"
	"strings"

	"guify_my_script/logger"
)

// PathResult defines the JSON structure we will return.
type PathResult struct {
	OriginalPath string `json:"original_path"`
	ModifiedPath string `json:"modified_path"`
}

// ModifyPath strips the existing extension, appends a custom string,
// adds a new extension, logs the result based on the requested level,
// and returns the result as a JSON string.
func ModifyPath(filePath, appendString, newExtension, logLevel string) (string, error) {
	// Parse the string into your internal LogLevel type
	lvl := logger.ParseLogLevel(logLevel)

	// 1. Get the current extension (includes the last dot)
	ext := filepath.Ext(filePath)

	// 2. Strip only that final extension
	basePath := strings.TrimSuffix(filePath, ext)

	// 3. Ensure the new extension has a leading dot
	if newExtension != "" && !strings.HasPrefix(newExtension, ".") {
		newExtension = "." + newExtension
	}

	// 4. Construct the new path
	newPath := basePath + appendString + newExtension

	// 5. Package it into our struct
	result := PathResult{
		OriginalPath: filePath,
		ModifiedPath: newPath,
	}

	// 6. Marshal the struct into JSON bytes
	jsonData, err := json.Marshal(result)
	if err != nil {
		// Utilizing your logger for error states as well
		logger.Error(lvl, "Failed to marshal path result: %v", err)
		return "", err
	}

	jsonString := string(jsonData)

	// 7. Output the JSON string as an INFO level log
	logger.Info(lvl, "%s", jsonString)

	return jsonString, nil
}

// GeneratePathFromDir takes a directory path, extracts its base name to create a file name,
// appends an optional string and extension, and outputs the result as a JSON string.
// The new file is nested directly inside the provided directory.
func GeneratePathFromDir(dirPath, appendString, newExtension, logLevel string) (string, error) {
	// Parse the string into your internal LogLevel type
	lvl := logger.ParseLogLevel(logLevel)

	// 1. Clean the path to handle edge cases (like trailing slashes) safely
	cleanDir := filepath.Clean(dirPath)

	// 2. Extract just the directory name (e.g., "config" from "/app/config")
	baseName := filepath.Base(cleanDir)

	// 3. Ensure the new extension has a leading dot if one was provided
	if newExtension != "" && !strings.HasPrefix(newExtension, ".") {
		newExtension = "." + newExtension
	}

	// 4. Construct the new filename
	fileName := baseName + appendString + newExtension

	// 5. Join the original directory path with the new filename
	newPath := filepath.Join(dirPath, fileName)

	// 6. Package it into our existing struct
	result := PathResult{
		OriginalPath: dirPath,
		ModifiedPath: newPath,
	}

	// 7. Marshal the struct into JSON bytes
	jsonData, err := json.Marshal(result)
	if err != nil {
		logger.Error(lvl, "Failed to marshal directory path result: %v", err)
		return "", err
	}

	jsonString := string(jsonData)

	// 8. Output the JSON string as an INFO level log
	logger.Info(lvl, "%s", jsonString)

	return jsonString, nil
}
