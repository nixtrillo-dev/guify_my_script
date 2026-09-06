package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"guify_my_script/config_gen"
	"guify_my_script/endpoint_utils"
	"guify_my_script/go_gen"
	"guify_my_script/logger"
	"guify_my_script/rust_gen"
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

		// Directory fallback: if extension is empty and inputPath is a directory, inspect contents
		if scriptType == "" {
			if fi, err := os.Stat(inputPath); err == nil && fi.IsDir() {
				cargoToml := filepath.Join(inputPath, "Cargo.toml")
				if _, err := os.Stat(cargoToml); err == nil {
					scriptType = "RUST"
				} else {
					_ = filepath.WalkDir(inputPath, func(path string, d os.DirEntry, err error) error {
						if err == nil && !d.IsDir() && filepath.Ext(path) == ".rs" {
							scriptType = "RUST"
							return filepath.SkipAll
						}
						return nil
					})
				}
			}
		}
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
	} else if strings.ToUpper(scriptType) == "RS" || strings.ToUpper(scriptType) == "RUST" {
		// Enforce: outputBridge must live inside inputPath/src/ for RUST projects.
		if outputBridge != "" {
			absInput, err := filepath.Abs(inputPath)
			if err != nil {
				return fmt.Errorf("RUST: failed to resolve inputPath: %w", err)
			}
			absOutBridge, err := filepath.Abs(outputBridge)
			if err != nil {
				return fmt.Errorf("RUST: failed to resolve outputBridge: %w", err)
			}
			// Rule 1: bridge must be inside inputPath (when inputPath is a directory)
			if fi, err := os.Stat(absInput); err == nil && fi.IsDir() {
				if !strings.HasPrefix(absOutBridge, absInput+string(filepath.Separator)) {
					return fmt.Errorf("RUST: outputBridge %q must be located inside inputPath %q", outputBridge, inputPath)
				}
			}
			// Rule 2: bridge's immediate parent must be a directory named "src"
			if filepath.Base(filepath.Dir(absOutBridge)) != "src" {
				return fmt.Errorf("RUST: outputBridge %q must be placed inside a 'src' directory (got parent %q)", outputBridge, filepath.Base(filepath.Dir(absOutBridge)))
			}
		}

		// 1. Run the Rust code generator (from generate_rust_bridge.go)
		targetDir, absBridge, err := rust_gen.GenerateRustBridge(inputPath, outputBridge, excludePackages, logLevel, includeFuncParams)
		if err != nil {
			return fmt.Errorf("RUST: bridge generation failed: %w", err)
		}

		// 2. Run the JSON layout config generator (from generate_config.go)
		if err := config_gen.GenerateGUIConfig(absBridge, outputJSON, binaryName, overwriteJson, logLevel, autoOpen); err != nil {
			return fmt.Errorf("RUST: JSON generation failed: %w", err)
		}

		// 3. Compile the local RUST binary assets if requested (from generate_rust_bridge.go)
		if compile {
			if err := rust_gen.CompileRust(targetDir, absBridge, binaryName, logLevel); err != nil {
				return fmt.Errorf("RUST: compilation failed: %w", err)
			}
		}
	} else {
		// Non-GO / Non-RUST script (e.g., "SH" (BASH))
		// Run only the JSON layout config generator directly on the script inputPath
		if err := config_gen.GenerateGUIConfig(inputPath, outputJSON, binaryName, overwriteJson, logLevel, autoOpen); err != nil {
			return fmt.Errorf("JSON generation failed: %w", err)
		}
	}

	return nil
}
