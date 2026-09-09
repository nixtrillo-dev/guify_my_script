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

// GeneratePathFromDir takes a directory or file path. It detects user intent based
// on trailing slashes before cleaning the path. It then safely appends an optional
// string and extension without risking string duplication or double slashes.
func GeneratePathFromDir(dirPath, appendString, newExtension, logLevel string, omitBaseName bool) (string, error) {
	lvl := logger.ParseLogLevel(logLevel)

	// 1. Detect explicit directory intent before filepath.Clean strips it
	hasTrailingSlash := strings.HasSuffix(dirPath, "/") || strings.HasSuffix(dirPath, string(os.PathSeparator))

	// 2. Clean the path to handle edge cases safely
	cleanPath := filepath.Clean(dirPath)

	var targetDir string
	var baseName string

	// 3. Determine if the path provided is actually a file or directory
	if info, err := os.Stat(cleanPath); err == nil {
		if !info.IsDir() {
			ext := filepath.Ext(cleanPath)
			targetDir = filepath.Dir(cleanPath)
			baseName = strings.TrimSuffix(filepath.Base(cleanPath), ext)

			// Inject base name as a new parent directory for .go and .rs files
			if ext == ".go" || ext == ".rs" {
				targetDir = filepath.Join(targetDir, baseName)
			}
		} else {
			// It is a valid directory on disk
			targetDir = cleanPath
			baseName = filepath.Base(cleanPath)
		}
	} else {
		// Fallbacks if the path does not exist on disk yet
		if hasTrailingSlash {
			// Intent: Directory (due to trailing slash)
			targetDir = cleanPath
			baseName = filepath.Base(cleanPath)
		} else if ext := filepath.Ext(cleanPath); ext != "" {
			// Intent: File with extension
			targetDir = filepath.Dir(cleanPath)
			baseName = strings.TrimSuffix(filepath.Base(cleanPath), ext)

			// Inject base name as a new parent directory for .go and .rs files
			if ext == ".go" || ext == ".rs" {
				targetDir = filepath.Join(targetDir, baseName)
			}
		} else {
			// Intent: File prefix (no extension, no trailing slash)
			targetDir = filepath.Dir(cleanPath)
			baseName = filepath.Base(cleanPath)
		}
	}

	// 4. Ensure the new extension has a leading dot
	if newExtension != "" && !strings.HasPrefix(newExtension, ".") {
		newExtension = "." + newExtension
	}

	// 5. Construct the new filename, gating the baseName if requested
	var fileName string
	if omitBaseName {
		fileName = appendString + newExtension
	} else {
		fileName = baseName + appendString + newExtension
	}

	// 6. Join the resolved directory path with the new filename (prevents double slashes)
	newPath := filepath.Join(targetDir, fileName)

	// 7. Package it into our struct
	result := PathResult{
		OriginalPath: dirPath,
		ModifiedPath: newPath,
	}

	jsonData, err := json.Marshal(result)
	if err != nil {
		logger.Error(lvl, "Failed to marshal directory path result: %v", err)
		return "", err
	}

	jsonString := string(jsonData)
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
		return GeneratePathFromDir(dirPath, "main", ".go", logLevel, true)
	case "RUST":
		var generatedJSON string
		var err error

		// 1. Generate the initial path string (which is a JSON string)
		if logLevel == "INFO" {
			// INFO returns the JSON string, but we are updating the modified_path in the next step,
			// so we don't want to log it twice. Use WARNING to suppress the INFO log.
			generatedJSON, err = GeneratePathFromDir(dirPath, "main", ".rs", "WARNING", true)
		} else {
			generatedJSON, err = GeneratePathFromDir(dirPath, "main", ".rs", logLevel, true)
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
		return GeneratePathFromDir(dirPath, "main", "", logLevel, true)
	}
}
