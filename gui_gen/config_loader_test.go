package gui_gen

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func TestExecutionMode_ValidationAndEquivalents(t *testing.T) {
	testCases := []struct {
		input             string
		fieldJSONKey      string
		expectedShorthand string
		expectedLonghand  string
	}{
		// Longhand
		{"GUI", "execution_mode", "g", "GUI"},
		{"gui", "execution_mode", "g", "GUI"},
		{"Gui", "execution_mode", "g", "GUI"},
		{"GUI_COMMANDLINE", "execution_mode", "gc", "GUI_COMMANDLINE"},
		{"gui_commandline", "execution_mode", "gc", "GUI_COMMANDLINE"},
		{"SILENT", "execution_mode", "s", "SILENT"},
		{"silent", "execution_mode", "s", "SILENT"},
		{"DETACHED_COMMANDLINE", "execution_mode", "d", "DETACHED_COMMANDLINE"},
		{"detached_commandline", "execution_mode", "d", "DETACHED_COMMANDLINE"},
		{"COMMANDLINE", "execution_mode", "c", "COMMANDLINE"},
		{"commandline", "execution_mode", "c", "COMMANDLINE"},
		// Shorthand under execution_mode
		{"g", "execution_mode", "g", "GUI"},
		{"G", "execution_mode", "g", "GUI"},
		{"gc", "execution_mode", "gc", "GUI_COMMANDLINE"},
		{"GC", "execution_mode", "gc", "GUI_COMMANDLINE"},
		{"s", "execution_mode", "s", "SILENT"},
		{"S", "execution_mode", "s", "SILENT"},
		{"d", "execution_mode", "d", "DETACHED_COMMANDLINE"},
		{"D", "execution_mode", "d", "DETACHED_COMMANDLINE"},
		{"c", "execution_mode", "c", "COMMANDLINE"},
		{"C", "execution_mode", "c", "COMMANDLINE"},
		// Legacy mode property
		{"g", "mode", "g", "GUI"},
		{"GUI", "mode", "g", "GUI"},
		{"gc", "mode", "gc", "GUI_COMMANDLINE"},
		{"s", "mode", "s", "SILENT"},
		{"d", "mode", "d", "DETACHED_COMMANDLINE"},
		{"c", "mode", "c", "COMMANDLINE"},
	}

	for _, tc := range testCases {
		jsonStr := fmt.Sprintf(`{"label":"Action Test","func":"fn_test",%q:%q}`, tc.fieldJSONKey, tc.input)
		var act UIAction
		if err := json.Unmarshal([]byte(jsonStr), &act); err != nil {
			t.Errorf("Failed unmarshaling %s=%q: %v", tc.fieldJSONKey, tc.input, err)
			continue
		}
		if act.GetMode() != tc.expectedShorthand {
			t.Errorf("Input %s=%q: expected GetMode()=%q, got %q", tc.fieldJSONKey, tc.input, tc.expectedShorthand, act.GetMode())
		}
		if act.GetExecutionMode() != tc.expectedLonghand {
			t.Errorf("Input %s=%q: expected GetExecutionMode()=%q, got %q", tc.fieldJSONKey, tc.input, tc.expectedLonghand, act.GetExecutionMode())
		}
	}
}

func TestExecutionMode_MutualExclusivityFailsEarly(t *testing.T) {
	// Attempt to define both "mode" and "execution_mode" on the same action
	jsonStr := `{"label":"Conflict Action","func":"conflict_fn","mode":"g","execution_mode":"GUI"}`
	var act UIAction
	err := json.Unmarshal([]byte(jsonStr), &act)
	if err == nil {
		t.Fatal("Expected error when both 'mode' and 'execution_mode' are defined on the same action, got nil")
	}
	expectedSub := "cannot define both 'mode' and 'execution_mode'"
	if !strings.Contains(err.Error(), expectedSub) {
		t.Errorf("Expected error to contain %q, got %q", expectedSub, err.Error())
	}
}

func TestExecutionMode_InvalidValueFails(t *testing.T) {
	invalidJSON := `{"label":"Invalid Action","func":"bad_fn","execution_mode":"INVALID_MODE"}`
	var act UIAction
	err := json.Unmarshal([]byte(invalidJSON), &act)
	if err == nil {
		t.Fatal("Expected error when invalid execution_mode is specified, got nil")
	}
	expectedSub := "has invalid execution_mode"
	if !strings.Contains(err.Error(), expectedSub) {
		t.Errorf("Expected error to contain %q, got %q", expectedSub, err.Error())
	}

	invalidModeJSON := `{"label":"Invalid Mode Action","func":"bad_fn","mode":"xyz"}`
	var act2 UIAction
	err2 := json.Unmarshal([]byte(invalidModeJSON), &act2)
	if err2 == nil {
		t.Fatal("Expected error when invalid mode is specified, got nil")
	}
	if !strings.Contains(err2.Error(), "has invalid mode") {
		t.Errorf("Expected error to contain 'has invalid mode', got %q", err2.Error())
	}
}

func TestLoadAndPrepareConfig_RealGuifyMyScriptConfig(t *testing.T) {
	configPath := filepath.Join("..", "guify_my_script_config.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Skip("guify_my_script_config.json not found at project root")
	}

	ctx, err := loadAndPrepareConfig(configPath, "debug")
	if err != nil {
		t.Fatalf("Failed to load real guify_my_script_config.json: %v", err)
	}

	if len(ctx.root.Actions) != 3 {
		t.Fatalf("Expected 3 actions, got %d", len(ctx.root.Actions))
	}

	expectedModes := map[string]struct {
		shorthand string
		longhand  string
	}{
		"Compile My Script":  {"g", "GUI"},
		"Guify My Script":    {"g", "GUI"},
		"Install My Package": {"gc", "GUI_COMMANDLINE"},
	}

	for _, action := range ctx.root.Actions {
		exp, ok := expectedModes[action.Label]
		if !ok {
			t.Errorf("Unexpected action: %s", action.Label)
			continue
		}
		if action.GetMode() != exp.shorthand {
			t.Errorf("Action %q: expected GetMode()=%q, got %q", action.Label, exp.shorthand, action.GetMode())
		}
		if action.GetExecutionMode() != exp.longhand {
			t.Errorf("Action %q: expected GetExecutionMode()=%q, got %q", action.Label, exp.longhand, action.GetExecutionMode())
		}
	}
}
