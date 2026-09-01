package gui_gen

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndPrepareConfig_BashWithArchScripts(t *testing.T) {
	tempDir := t.TempDir()

	scriptPath := filepath.Join(tempDir, "test_logic.sh")
	if err := os.WriteFile(scriptPath, []byte("#!/bin/bash\necho hi\n"), 0755); err != nil {
		t.Fatalf("Failed creating dummy script: %v", err)
	}

	config := RootConfig{
		Config: AppConfig{
			Title:             "Test App",
			UUID:              "com.example.testapp",
			ScriptAlias:       "test_logic.sh",
			SourceScriptAMD64: scriptPath,
			SourceScriptARM64: scriptPath,
			SourceType:        "bash",
			Startup:           "tray",
			TrayOnMinimize:    true,
		},
		Actions: []UIAction{
			{Label: "Run Action", Func: "run_action", Mode: "g"},
		},
	}

	configData, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed marshaling config: %v", err)
	}

	configPath := filepath.Join(tempDir, "config.json")
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		t.Fatalf("Failed writing config: %v", err)
	}

	ctx, err := loadAndPrepareConfig(configPath, "production")
	if err != nil {
		t.Fatalf("loadAndPrepareConfig failed: %v", err)
	}

	if ctx.bundledSourceName != "test_logic.sh" {
		t.Errorf("Expected bundledSourceName to be 'test_logic.sh', got %q", ctx.bundledSourceName)
	}

	hasAMD64 := false
	hasARM64 := false
	for _, arch := range ctx.targetArchs {
		if arch == "amd64" {
			hasAMD64 = true
		}
		if arch == "arm64" {
			hasARM64 = true
		}
	}

	if !hasAMD64 || !hasARM64 {
		t.Errorf("Expected targetArchs to contain both amd64 and arm64, got %v", ctx.targetArchs)
	}
}

func TestLoadAndPrepareConfig_PythonWithArchScripts(t *testing.T) {
	tempDir := t.TempDir()

	scriptPath := filepath.Join(tempDir, "test_pyscript.py")
	if err := os.WriteFile(scriptPath, []byte("def greet(): pass\n"), 0755); err != nil {
		t.Fatalf("Failed creating dummy python script: %v", err)
	}

	config := RootConfig{
		Config: AppConfig{
			Title:             "Python App",
			UUID:              "com.example.pyapp",
			ScriptAlias:       "test_pyscript.py",
			SourceScriptAMD64: scriptPath,
			SourceScriptARM64: scriptPath,
			SourceType:        "python",
			Startup:           "tray",
			TrayOnMinimize:    true,
		},
		Actions: []UIAction{
			{
				Label: "Greet",
				Func:  "greet",
				Mode:  "g",
				Fields: []UIField{
					{Name: "Name", Type: "TXT", DataType: "string", Default: "Alice"},
				},
			},
		},
	}

	configData, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed marshaling config: %v", err)
	}

	configPath := filepath.Join(tempDir, "config.json")
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		t.Fatalf("Failed writing config: %v", err)
	}

	ctx, err := loadAndPrepareConfig(configPath, "production")
	if err != nil {
		t.Fatalf("loadAndPrepareConfig failed: %v", err)
	}

	if ctx.bundledSourceName != "test_pyscript.py" {
		t.Errorf("Expected bundledSourceName to be 'test_pyscript.py', got %q", ctx.bundledSourceName)
	}

	hasAMD64 := false
	hasARM64 := false
	for _, arch := range ctx.targetArchs {
		if arch == "amd64" {
			hasAMD64 = true
		}
		if arch == "arm64" {
			hasARM64 = true
		}
	}

	if !hasAMD64 || !hasARM64 {
		t.Errorf("Expected targetArchs to contain both amd64 and arm64, got %v", ctx.targetArchs)
	}
}

func TestLoadAndPrepareConfig_GoWithArchScripts(t *testing.T) {
	tempDir := t.TempDir()

	binAMD64 := filepath.Join(tempDir, "my_binary_amd64")
	binARM64 := filepath.Join(tempDir, "my_binary_arm64")
	_ = os.WriteFile(binAMD64, []byte("amd64 binary"), 0755)
	_ = os.WriteFile(binARM64, []byte("arm64 binary"), 0755)

	config := RootConfig{
		Config: AppConfig{
			Title:             "Go App",
			UUID:              "com.example.goapp",
			ScriptAlias:       "my_binary",
			SourceScriptAMD64: binAMD64,
			SourceScriptARM64: binARM64,
			SourceType:        "go",
			Startup:           "tray",
			TrayOnMinimize:    true,
		},
		Actions: []UIAction{
			{Label: "Go Action", Func: "go_action", Mode: "g"},
		},
	}

	configData, err := json.Marshal(config)
	if err != nil {
		t.Fatalf("Failed marshaling config: %v", err)
	}

	configPath := filepath.Join(tempDir, "config.json")
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		t.Fatalf("Failed writing config: %v", err)
	}

	ctx, err := loadAndPrepareConfig(configPath, "production")
	if err != nil {
		t.Fatalf("loadAndPrepareConfig failed: %v", err)
	}

	if ctx.bundledSourceName != "my_binary" {
		t.Errorf("Expected bundledSourceName to be 'my_binary', got %q", ctx.bundledSourceName)
	}

	hasAMD64 := false
	hasARM64 := false
	for _, arch := range ctx.targetArchs {
		if arch == "amd64" {
			hasAMD64 = true
		}
		if arch == "arm64" {
			hasARM64 = true
		}
	}

	if !hasAMD64 || !hasARM64 {
		t.Errorf("Expected targetArchs to contain both amd64 and arm64, got %v", ctx.targetArchs)
	}
}
