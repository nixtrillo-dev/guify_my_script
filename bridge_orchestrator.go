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

		if logLevel == "INFO" || logLevel == "DATA" { // INFO or DATA level
			extJsonStr, err = endpoint_utils.GetExtension(inputPath, "WARNING") // Passing "WARNING" here as we have no interest in DATA here under the context of this function...
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
		// Single-file .go promotion: if inputPath is a plain .go file, scaffold a minimal
		// project directory around it before any further processing.
		{
			absInput, err := filepath.Abs(inputPath)
			if err != nil {
				return fmt.Errorf("GO: failed to resolve inputPath: %w", err)
			}
			if fi, statErr := os.Stat(absInput); statErr == nil && !fi.IsDir() && strings.ToLower(filepath.Ext(absInput)) == ".go" {
				stem := strings.TrimSuffix(filepath.Base(absInput), filepath.Ext(absInput))
				projectDir := filepath.Join(filepath.Dir(absInput), stem)

				// Validate outputBridge before touching anything on disk.
				// "" is explicitly allowed — GenerateBridge will default to a file inside projectDir.
				// Any non-empty value must resolve to a file directly inside projectDir
				// with a non-empty filename; anything else is an error.
				if outputBridge != "" {
					absOutBridge, err := filepath.Abs(outputBridge)
					if err != nil {
						return fmt.Errorf("GO: single-file promotion: failed to resolve outputBridge %q: %w", outputBridge, err)
					}
					bridgeParent := filepath.Clean(filepath.Dir(absOutBridge))
					bridgeFile := filepath.Base(absOutBridge)
					if bridgeParent != filepath.Clean(projectDir) || bridgeFile == "" || bridgeFile == "." {
						return fmt.Errorf("GO: single-file promotion: outputBridge %q must be a file directly inside %q (e.g. %s)", outputBridge, projectDir, filepath.Join(projectDir, stem+"_bridge.go"))
					}
				}

				// Guard: error out before touching anything if the target directory already exists.
				if _, dirErr := os.Stat(projectDir); dirErr == nil {
					return fmt.Errorf("GO: cannot promote single-file %q: directory %q already exists", absInput, projectDir)
				}

				// Create <stem>/
				if err := os.MkdirAll(projectDir, 0755); err != nil {
					return fmt.Errorf("GO: failed to create project directory %q: %w", projectDir, err)
				}

				// Move <stem>.go → <stem>/<stem>.go
				destFile := filepath.Join(projectDir, filepath.Base(absInput))
				if err := os.Rename(absInput, destFile); err != nil {
					return fmt.Errorf("GO: failed to move %q to %q: %w", absInput, destFile, err)
				}
				logger.Info(parsedLogLevel, "GO: Promoted single file %q → %q", absInput, destFile)

				// Redirect inputPath to the scaffolded project directory.
				inputPath = projectDir
			}
		}

		// Enforce: outputBridge must live inside inputPath for GO projects.
		if outputBridge != "" {
			absInput, err := filepath.Abs(inputPath)
			if err != nil {
				return fmt.Errorf("GO: failed to resolve inputPath: %w", err)
			}
			absOutBridge, err := filepath.Abs(outputBridge)
			if err != nil {
				return fmt.Errorf("GO: failed to resolve outputBridge: %w", err)
			}
			// Bridge must be inside inputPath (when inputPath is a directory)
			if fi, err := os.Stat(absInput); err == nil && fi.IsDir() {
				if !strings.HasPrefix(absOutBridge, absInput+string(filepath.Separator)) {
					return fmt.Errorf("GO: outputBridge %q must be located inside inputPath %q", outputBridge, inputPath)
				}
			}
		}

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
		// Single-file .rs promotion: if inputPath is a plain .rs file, scaffold a minimal
		// Cargo project around it before any further processing.
		{
			absInput, err := filepath.Abs(inputPath)
			if err != nil {
				return fmt.Errorf("RUST: failed to resolve inputPath: %w", err)
			}
			if fi, statErr := os.Stat(absInput); statErr == nil && !fi.IsDir() && strings.ToLower(filepath.Ext(absInput)) == ".rs" {
				stem := strings.TrimSuffix(filepath.Base(absInput), filepath.Ext(absInput))
				projectDir := filepath.Join(filepath.Dir(absInput), stem)
				expectedSrcDir := filepath.Join(projectDir, "src")

				// Validate outputBridge before touching anything on disk.
				// "" is explicitly allowed — GenerateRustBridge will default to src/main.rs.
				// Any non-empty value must resolve to a file directly inside projectDir/src/
				// with a non-empty filename; anything else is an error.
				if outputBridge != "" {
					absOutBridge, err := filepath.Abs(outputBridge)
					if err != nil {
						return fmt.Errorf("RUST: single-file promotion: failed to resolve outputBridge %q: %w", outputBridge, err)
					}
					bridgeParent := filepath.Clean(filepath.Dir(absOutBridge))
					bridgeFile := filepath.Base(absOutBridge)
					if bridgeParent != filepath.Clean(expectedSrcDir) || bridgeFile == "" || bridgeFile == "." {
						return fmt.Errorf("RUST: single-file promotion: outputBridge %q must be a file directly inside %q (e.g. %s)", outputBridge, expectedSrcDir, filepath.Join(expectedSrcDir, stem+"_bridge.rs"))
					}
				}

				// Guard: error out before touching anything if the target directory already exists.
				if _, dirErr := os.Stat(projectDir); dirErr == nil {
					return fmt.Errorf("RUST: cannot promote single-file %q: directory %q already exists", absInput, projectDir)
				}

				// Create <stem>/src/
				srcDir := filepath.Join(projectDir, "src")
				if err := os.MkdirAll(srcDir, 0755); err != nil {
					return fmt.Errorf("RUST: failed to create project src directory %q: %w", srcDir, err)
				}

				// Move <stem>.rs → <stem>/src/<stem>.rs
				destFile := filepath.Join(srcDir, filepath.Base(absInput))
				if err := os.Rename(absInput, destFile); err != nil {
					return fmt.Errorf("RUST: failed to move %q to %q: %w", absInput, destFile, err)
				}
				logger.Info(parsedLogLevel, "RUST: Promoted single file %q → %q", absInput, destFile)

				// Do NOT write Cargo.toml here — ensureRootCargo (called during CompileRust)
				// handles generation with the correct bridge path already known. Writing one
				// now would either point at the wrong file or be overwritten anyway.

				// Redirect inputPath to the scaffolded project directory.
				// outputBridge is either "" (use default) or already validated above to be
				// correctly placed inside projectDir/src/, so no further adjustment needed.
				inputPath = projectDir
			}
		}

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
			// bridge's immediate parent must be the root 'src' directory directly inside inputPath
			if fi, err := os.Stat(absInput); err == nil && fi.IsDir() {
				expectedSrcDir := filepath.Join(absInput, "src")
				if filepath.Clean(filepath.Dir(absOutBridge)) != filepath.Clean(expectedSrcDir) {
					return fmt.Errorf("RUST: outputBridge %q must have the root src directory %q as its parent folder", outputBridge, expectedSrcDir)
				}
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
