package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompileScriptWithGuiConfig_RustProject(t *testing.T) {
	projectDir := "examples/rust_test_script/core_utils"
	absProjectDir, err := filepath.Abs(projectDir)
	if err != nil {
		t.Fatalf("Failed to resolve absolute path: %v", err)
	}
	if _, err := os.Stat(absProjectDir); os.IsNotExist(err) {
		t.Skip("skipping test: examples/rust_test_script does not exist")
	}

	tempDir := t.TempDir()
	outputJSON := filepath.Join(tempDir, "core_utils_config.json")
	outputBridge := filepath.Join(absProjectDir, "src", "main.rs")

	// Clean up any existing main.rs before test
	_ = os.Remove(outputBridge)

	err = CompileScriptWithGuiConfig(
		absProjectDir,
		"RUST",
		outputJSON,
		"",           // excludePackages
		outputBridge, // outputBridge
		"core_utils", // binaryName
		true,         // compile
		true,         // overwriteJson
		"DEBUG",      // logLevel
		false,        // includeFuncParams
		false,        // autoOpen
		"",           // targetBuilds (all architectures)
	)
	if err != nil {
		t.Fatalf("CompileScriptWithGuiConfig failed for Rust: %v", err)
	}

	// 1. Verify bridge file exists
	if _, err := os.Stat(outputBridge); os.IsNotExist(err) {
		t.Fatalf("Expected bridge file %s to exist", outputBridge)
	}

	// 2. Verify config JSON exists
	if _, err := os.Stat(outputJSON); os.IsNotExist(err) {
		t.Fatalf("Expected output JSON %s to exist", outputJSON)
	}

	// 3. Verify compiled binaries exist in projectDir
	binLinuxAMD64 := filepath.Join(absProjectDir, "core_utils_linux_amd64")
	if _, err := os.Stat(binLinuxAMD64); os.IsNotExist(err) {
		t.Fatalf("Expected compiled binary %s to exist", binLinuxAMD64)
	}

	libLinuxAMD64 := filepath.Join(absProjectDir, "core_utils_linux_amd64.so")
	if _, err := os.Stat(libLinuxAMD64); os.IsNotExist(err) {
		t.Fatalf("Expected compiled shared library %s to exist", libLinuxAMD64)
	}
}

func TestCompileScriptWithGuiConfig_RustProjectAutoDetect(t *testing.T) {
	parentDir := "examples/rust_test_script"
	absParentDir, err := filepath.Abs(parentDir)
	if err != nil {
		t.Fatalf("Failed to resolve absolute path: %v", err)
	}
	if _, err := os.Stat(absParentDir); os.IsNotExist(err) {
		t.Skip("skipping test: examples/rust_test_script does not exist")
	}

	tempDir := t.TempDir()
	outputJSON := filepath.Join(tempDir, "config.json")

	err = CompileScriptWithGuiConfig(
		absParentDir,
		"", // auto-detect
		outputJSON,
		"",
		"",
		"rust_test_script",
		true,
		true,
		"DEBUG",
		false,
		false,
		"",
	)
	if err != nil {
		t.Fatalf("Auto-detect failed: %v", err)
	}

	if _, err := os.Stat(outputJSON); os.IsNotExist(err) {
		t.Fatalf("Expected output JSON %s to exist", outputJSON)
	}
}
