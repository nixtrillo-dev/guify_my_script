package main

import (
	"fmt"
	"guify_my_script/bridge_gen"
	"guify_my_script/config_gen"
)

// GenerateBridgeWithGuiConfig strings together the JSON schema generation and the C++ bridge construction.
func GenerateBridgeWithGuiConfig(inputPath, outputJSON, excludePackages, outputBridge, binaryName string, compile bool, overwriteJson bool, logLevel string, includeFuncParams bool, autoOpen bool) error {
	// 1. Run the C++ bridge code generator (from generate_go_bridge.go)
	targetDir, absBridge, err := bridge_gen.GenerateBridge(inputPath, outputBridge, excludePackages, logLevel, includeFuncParams)
	if err != nil {
		return fmt.Errorf("bridge generation failed: %w", err)
	}

	// 2. Run the JSON layout config generator (from generate_config.go)
	if err := config_gen.GenerateGUIConfig(absBridge, outputJSON, binaryName, overwriteJson, logLevel, autoOpen); err != nil {
		return fmt.Errorf("JSON generation failed: %w", err)
	}

	// 3. Compile the local binary assets if requested (from generate_go_bridge.go)
	if compile {
		if err := bridge_gen.CompileLocal(targetDir, absBridge, binaryName, logLevel); err != nil {
			return fmt.Errorf("compilation failed: %w", err)
		}
	}

	return nil
}
