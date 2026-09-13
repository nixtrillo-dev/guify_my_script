package endpoint_utils

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"guify_my_script/go_gen"
	"guify_my_script/logger"
	"guify_my_script/rust_gen"
)

// DirectoryStatus represents the JSON structure for the directory check.
type DirectoryStatus struct {
	IsValid bool `json:"is_valid"`
}

// Result represents the JSON structure returned for the script type check.
type ScriptStatus struct {
	FileType string `json:"file_type"`
}

// CompilableStatus represents the JSON structure for the compilation support check.
type CompilableStatus struct {
	IsCompilable bool `json:"is_compilable"`
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

	logger.Data(logLevel, "%s", jsonString)
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

	logger.Data(logLevel, "%s", jsonString)
	return jsonString
}

// GetNonMainPackagesJSON takes a project directory path and depending on the script type outputs all non-main packages as a JSON array.
func GetNonMainPackagesJSON(inputPath string, scriptType, logLevel string, noUpdateMode bool) (string, error) {
	lvl := logger.ParseLogLevel(logLevel)

	// Guard Clause: Return empty JSON array immediately if input string is empty
	if inputPath == "" {
		emptyJSON := "[]"
		logger.Data(lvl, "%s", emptyJSON)
		return emptyJSON, nil
	}

	switch scriptType {
	case "GO":
		return go_gen.GetGoNonMainPackagesJSON(inputPath, logLevel)
	case "RUST":
		return rust_gen.GetRustNonMainPackagesJSON(inputPath, logLevel, noUpdateMode)
	}

	emptyJSON := "[]"
	logger.Data(lvl, "%s", emptyJSON)
	return emptyJSON, nil
}

// GetMostRepresentedFileType scans the directory recursively and returns
// a JSON-marshaled string indicating whether "GO", "RUST", or tie (defaults to "GO") is most common.
// If the path is a file, it returns the file's extension in uppercase.
func GetMostRepresentedFileType(path string, logLevel logger.LogLevel) string {
	var goCount, rsCount int
	var jsonString string

	// Guard Clause: Return empty JSON array immediately if input string is empty
	if path == "" {
		logger.Debug(logLevel, "Guard Clause: Returning empty JSON array immediately as input string is empty")
		jsonString := "[]"
		logger.Data(logLevel, "%s", jsonString)
		return jsonString
	}

	// Check if path is a file instead of a directory
	info, err := os.Stat(path)
	if err == nil && !info.IsDir() {
		// Extract extension, remove the dot, and convert to uppercase
		ext := strings.ToUpper(strings.TrimPrefix(filepath.Ext(path), "."))
		res := ScriptStatus{FileType: ext}

		jsonBytes, err := json.Marshal(res)
		if err != nil {
			logger.Debug(logLevel, "JSON marshaling failed for file extension output: %v", err)
			jsonString = `[]`
		} else {
			jsonString = string(jsonBytes)
		}
		logger.Data(logLevel, "%s", jsonString)
		return jsonString
	}

	// Walk through the directory and subdirectories efficiently using WalkDir
	err = filepath.WalkDir(path, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Only check files, ignore directories
		if !d.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			switch ext {
			case ".go":
				goCount++
			case ".rs":
				rsCount++
			}
		}
		return nil
	})

	if err != nil {
		logger.Debug(logLevel, "filepath.WalkDir failed for array generation: %v", err)
		jsonString := "[]"
		logger.Data(logLevel, "%s", jsonString)
		return jsonString
	}

	// Determine the most represented file type - "GO" is the tie breaker value
	winner := "GO"
	if goCount > rsCount {
		winner = "GO"
	} else if rsCount > goCount {
		winner = "RUST"
	}

	// Marshal the result into JSON
	res := ScriptStatus{FileType: winner}

	jsonBytes, err := json.Marshal(res)

	if err != nil {
		logger.Debug(logLevel, "JSON marshaling failed for array generation: %v", err)
		jsonString = `[]`
	} else {
		jsonString = string(jsonBytes)
	}

	logger.Data(logLevel, "%s", jsonString)
	return jsonString
}

// IsCompilable checks if the most represented file type in a path (or the file itself)
// is currently supported for compiling, returning a JSON string.
func IsCompilable(path string, logLevel logger.LogLevel) string {
	isCompilable := false
	var status ScriptStatus

	// Get the file type JSON string by evaluating the path
	var scriptStatusJSON string
	if logLevel == 99 { // DATA level
		// Passing "ERROR" here as we have no interest in DATA here under the context of this function...
		scriptStatusJSON = GetMostRepresentedFileType(path, 2) // 2 corresponds to ERROR level
	} else {
		scriptStatusJSON = GetMostRepresentedFileType(path, logLevel)
	}

	// Unmarshal the JSON from GetMostRepresentedFileType
	err := json.Unmarshal([]byte(scriptStatusJSON), &status)
	if err == nil {
		// Currently only GO and RUST are supported for compiling
		if status.FileType == "GO" || status.FileType == "RUST" || status.FileType == "RS" {
			isCompilable = true
		}
		logger.Debug(logLevel, "Parsed ScriptStatus: %s, Compilable: %v", status.FileType, isCompilable)
	} else {
		logger.Debug(logLevel, "Failed to parse ScriptStatus JSON in IsCompilable: %v", err)
	}

	res := CompilableStatus{IsCompilable: isCompilable}
	jsonBytes, err := json.Marshal(res)

	var jsonString string
	if err != nil {
		logger.Debug(logLevel, "JSON marshaling failed for IsCompilable: %v", err)
		jsonString = `{"is_compilable": false}`
	} else {
		jsonString = string(jsonBytes)
	}

	logger.Data(logLevel, "%s", jsonString)
	return jsonString
}
