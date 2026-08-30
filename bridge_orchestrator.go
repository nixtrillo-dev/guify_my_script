package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"guify_my_script/config_gen"
	"guify_my_script/endpoint_utils"
	"guify_my_script/go_gen"
	"guify_my_script/logger"
)

// CompileScriptWithGuiConfig strings together the JSON schema generation and the C++ bridge construction.
func CompileScriptWithGuiConfig(inputPath, scriptType, outputJSON, excludePackages, outputBridge, binaryName string, compile bool, overwriteJson bool, logLevel string, includeFuncParams bool, autoOpen bool) error {
	parsedLogLevel := logger.ParseLogLevel(logLevel)

	// If scriptType is empty, determine it dynamically from the file extension
	if scriptType == "" {
		var extJsonStr string
		var err error

		if logLevel == "INFO" {
			extJsonStr, err = endpoint_utils.GetExtension(inputPath, "WARNING") // Passing "WARNING" here as we have no interest in INFO here under the context of this function...
		} else {
			extJsonStr, err = endpoint_utils.GetExtension(inputPath, logLevel)
		}

		if err != nil {
			return fmt.Errorf("failed to extract extension: %w", err)
		}

		var extResult endpoint_utils.ExtensionResult
		if err := json.Unmarshal([]byte(extJsonStr), &extResult); err != nil {
			return fmt.Errorf("failed to unmarshal extension result: %w", err)
		}

		scriptType = extResult.Extension
	}

	logger.Debug(parsedLogLevel, "scriptType: %s", scriptType)

	if strings.ToUpper(scriptType) == "GO" {
		// 1. Run the Go code generator (from generate_go_bridge.go)
		targetDir, absBridge, err := go_gen.GenerateBridge(inputPath, outputBridge, excludePackages, logLevel, includeFuncParams)
		if err != nil {
			return fmt.Errorf("GO: bridge generation failed: %w", err)
		}

		// 2. Run the JSON layout config generator (from generate_config.go)
		if err := config_gen.GenerateGUIConfig(absBridge, outputJSON, binaryName, overwriteJson, logLevel, autoOpen); err != nil {
			return fmt.Errorf("GO: JSON generation failed: %w", err)
		}

		// 3. Compile the local GO binary assets if requested (from generate_go_bridge.go)
		if compile {
			if err := go_gen.CompileLocal(targetDir, absBridge, binaryName, logLevel); err != nil {
				return fmt.Errorf("GO: compilation failed: %w", err)
			}
		}
	} else {
		// Non-GO script (e.g., "SH" (BASH))
		// Run only the JSON layout config generator directly on the script inputPath
		if err := config_gen.GenerateGUIConfig(inputPath, outputJSON, binaryName, overwriteJson, logLevel, autoOpen); err != nil {
			return fmt.Errorf("SH: JSON generation failed: %w", err)
		}
	}

	return nil
}
