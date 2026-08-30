package endpoint_utils

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"guify_my_script/logger"
)

// PathResult defines the JSON structure we will return.
type PathResult struct {
	OriginalPath string `json:"original_path"`
	ModifiedPath string `json:"modified_path"`
}

// ExtensionResult defines the JSON structure for extension extraction.
type ExtensionResult struct {
	OriginalPath string `json:"original_path"`
	Extension    string `json:"extension"`
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

// GeneratePathFromDir takes a directory or file path. If it's a directory,
// it extracts its base name to create a file name. If it's a file, it uses
// the file's base name (without extension). It then appends an optional string
// and extension, placing the new file in the target directory, and outputs
// the result as a JSON string.
func GeneratePathFromDir(dirPath, appendString, newExtension, logLevel string) (string, error) {
	// Parse the string into your internal LogLevel type
	lvl := logger.ParseLogLevel(logLevel)

	// 1. Clean the path to handle edge cases (like trailing slashes) safely
	cleanPath := filepath.Clean(dirPath)

	targetDir := cleanPath
	var baseName string

	// 2. Determine if the path provided is actually a file or directory.
	if info, err := os.Stat(cleanPath); err == nil {
		if !info.IsDir() {
			targetDir = filepath.Dir(cleanPath)
			// Use the file name (without extension) instead of the directory name
			baseName = strings.TrimSuffix(filepath.Base(cleanPath), filepath.Ext(cleanPath))
		} else {
			baseName = filepath.Base(cleanPath)
		}
	} else if ext := filepath.Ext(cleanPath); ext != "" {
		// Fallback: If it doesn't exist on disk but has an extension, treat it as a file
		targetDir = filepath.Dir(cleanPath)
		baseName = strings.TrimSuffix(filepath.Base(cleanPath), ext)
	} else {
		// Fallback: Assume it's a directory
		baseName = filepath.Base(cleanPath)
	}

	// 3. Ensure the new extension has a leading dot if one was provided
	if newExtension != "" && !strings.HasPrefix(newExtension, ".") {
		newExtension = "." + newExtension
	}

	// 4. Construct the new filename
	fileName := baseName + appendString + newExtension

	// 5. Join the resolved directory path with the new filename
	newPath := filepath.Join(targetDir, fileName)

	// 6. Package it into our existing struct, keeping the original raw input
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

// GetExtension parses the input file path string, extracts its extension
// (without the leading dot), converts it to uppercase, logs the result,
// and returns it as a JSON string.
func GetExtension(filePath, logLevel string) (string, error) {
	lvl := logger.ParseLogLevel(logLevel)

	// 1. Get the current extension (includes the last dot)
	ext := filepath.Ext(filePath)

	// 2. Remove the leading dot if it exists and convert to uppercase
	ext = strings.ToUpper(strings.TrimPrefix(ext, "."))

	// 3. Package it into the new struct
	result := ExtensionResult{
		OriginalPath: filePath,
		Extension:    ext,
	}

	// 4. Marshal the struct into JSON bytes
	jsonData, err := json.Marshal(result)
	if err != nil {
		logger.Error(lvl, "Failed to marshal extension result: %v", err)
		return "", err
	}

	jsonString := string(jsonData)

	// 5. Output the JSON string as an INFO level log
	logger.Info(lvl, "%s", jsonString)

	return jsonString, nil
}
