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
			ext := filepath.Ext(cleanPath)
			targetDir = filepath.Dir(cleanPath)
			// Use the file name (without extension) instead of the directory name
			baseName = strings.TrimSuffix(filepath.Base(cleanPath), ext)

			// Inject base name as a new parent directory for .go and .rs files
			if ext == ".go" || ext == ".rs" {
				targetDir = filepath.Join(targetDir, baseName)
			}
		} else {
			baseName = filepath.Base(cleanPath)
		}
	} else if ext := filepath.Ext(cleanPath); ext != "" {
		// Fallback: If it doesn't exist on disk but has an extension, treat it as a file
		targetDir = filepath.Dir(cleanPath)
		baseName = strings.TrimSuffix(filepath.Base(cleanPath), ext)

		// Inject base name as a new parent directory for .go and .rs files
		if ext == ".go" || ext == ".rs" {
			targetDir = filepath.Join(targetDir, baseName)
		}
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

// InsertParentFolder inserts a new directory immediately before the last element of the path,
// logs the JSON result as info, and returns it as a string.
func InsertParentFolder(originalPath, newFolder, logLevel string) (string, error) {
	// Parse the string into your internal LogLevel type
	lvl := logger.ParseLogLevel(logLevel)

	var newPath string
	if originalPath == "" {
		newPath = newFolder
	} else {
		// Extract everything EXCEPT the last element (e.g., "/var/log")
		dir := filepath.Dir(originalPath)

		// Extract ONLY the last element (e.g., "app.log")
		base := filepath.Base(originalPath)

		// Stitch them back together with the new folder in the middle
		newPath = filepath.Join(dir, newFolder, base)
	}

	// Package it into the existing struct
	result := PathResult{
		OriginalPath: originalPath,
		ModifiedPath: newPath,
	}

	// Marshal the struct into JSON bytes
	jsonData, err := json.Marshal(result)
	if err != nil {
		logger.Error(lvl, "Failed to marshal path result: %v", err)
		return "", err
	}

	jsonString := string(jsonData)

	// Output the JSON string as an INFO level log
	logger.Info(lvl, "%s", jsonString)

	return jsonString, nil
}

// GetDefaultBridgePathForScript generates a default binary path based on the script type.
// For Go scripts, it appends "_bridge.go" to the base name. For Rust scripts, it appends "_bridge.rs"
// and places the file in a "src" subdirectory. The result is returned as a JSON string.
// If scriptType is an empty string, it will attempt to derive it from the dirPath extension.
func GetDefaultBridgePathForScript(dirPath, scriptType, logLevel string) (string, error) {
	// Derive scriptType from the extension if it is empty
	if scriptType == "" {
		ext := strings.ToUpper(strings.TrimPrefix(filepath.Ext(dirPath), "."))
		switch ext {
		case "RS":
			scriptType = "RUST"
		case "GO":
			scriptType = "GO"
		}
	}

	switch scriptType {
	case "GO":
		return GeneratePathFromDir(dirPath, "_bridge", ".go", logLevel)
	case "RUST":
		var generatedJSON string
		var err error

		// 1. Generate the initial path string (which is a JSON string)
		if logLevel == "INFO" {
			// INFO returns the JSON string, but we are updating the modified_path in the next step,
			// so we don't want to log it twice. Use WARNING to suppress the INFO log.
			generatedJSON, err = GeneratePathFromDir(dirPath, "_bridge", ".rs", "WARNING")
		} else {
			generatedJSON, err = GeneratePathFromDir(dirPath, "_bridge", ".rs", logLevel)
		}

		if err != nil {
			return "", err
		}

		// 2. We only want the modified_path so unmarshal the JSON back into our struct
		// to isolate the path
		var parsedResult PathResult
		if err := json.Unmarshal([]byte(generatedJSON), &parsedResult); err != nil {
			lvl := logger.ParseLogLevel(logLevel)
			logger.Error(lvl, "Failed to unmarshal generated path JSON: %v", err)
			return "", err
		}

		// 3. Move the generated path into a "src" subdirectory and return the new JSON string
		return InsertParentFolder(parsedResult.ModifiedPath, "src", logLevel)
	default:
		return GeneratePathFromDir(dirPath, "_bridge", "", logLevel)
	}
}
