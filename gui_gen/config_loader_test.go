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
			SourceScriptLinuxAMD64:   scriptPath,
			SourceScriptLinuxAArch64: scriptPath,
			SourceType:               "bash",
			Startup:                  "tray",
			TrayOnMinimize:           true,
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
		if arch == "linux_amd64" {
			hasAMD64 = true
		}
		if arch == "linux_aarch64" {
			hasARM64 = true
		}
	}

	if !hasAMD64 || !hasARM64 {
		t.Errorf("Expected targetArchs to contain both linux_amd64 and linux_aarch64, got %v", ctx.targetArchs)
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
			SourceScriptLinuxAMD64:   scriptPath,
			SourceScriptLinuxAArch64: scriptPath,
			SourceType:               "python",
			Startup:                  "tray",
			TrayOnMinimize:           true,
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
		if arch == "linux_amd64" {
			hasAMD64 = true
		}
		if arch == "linux_aarch64" {
			hasARM64 = true
		}
	}

	if !hasAMD64 || !hasARM64 {
		t.Errorf("Expected targetArchs to contain both linux_amd64 and linux_aarch64, got %v", ctx.targetArchs)
	}
}

func TestLoadAndPrepareConfig_GoWithArchScripts(t *testing.T) {
	tempDir := t.TempDir()

	binAMD64 := filepath.Join(tempDir, "my_binary_linux_amd64")
	binARM64 := filepath.Join(tempDir, "my_binary_linux_aarch64")
	_ = os.WriteFile(binAMD64, []byte("linux_amd64 binary"), 0755)
	_ = os.WriteFile(binARM64, []byte("linux_aarch64 binary"), 0755)

	config := RootConfig{
		Config: AppConfig{
			Title:                    "Go App",
			UUID:                     "com.example.goapp",
			ScriptAlias:              "my_binary",
			SourceScriptLinuxAMD64:   binAMD64,
			SourceScriptLinuxAArch64: binARM64,
			SourceType:               "go",
			Startup:                  "tray",
			TrayOnMinimize:           true,
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
		if arch == "linux_amd64" {
			hasAMD64 = true
		}
		if arch == "linux_aarch64" {
			hasARM64 = true
		}
	}

	if !hasAMD64 || !hasARM64 {
		t.Errorf("Expected targetArchs to contain both linux_amd64 and linux_aarch64, got %v", ctx.targetArchs)
	}
}

func TestLoadAndPrepareConfig_LegacyBackwardsCompatibility(t *testing.T) {
	tempDir := t.TempDir()

	binLegacyAMD64 := filepath.Join(tempDir, "legacy_binary_amd64")
	binLegacyARM64 := filepath.Join(tempDir, "legacy_binary_arm64")
	_ = os.WriteFile(binLegacyAMD64, []byte("legacy amd64 binary"), 0755)
	_ = os.WriteFile(binLegacyARM64, []byte("legacy arm64 binary"), 0755)

	configJSON := fmt.Sprintf(`{
		"config": {
			"title": "Legacy Go App",
			"uuid": "com.example.legacygoapp",
			"script_alias": "legacy_binary",
			"source_script_amd64": %q,
			"source_script_arm64": %q,
			"source_type": "go",
			"startup": "tray",
			"tray_on_minimize": true
		},
		"actions": [
			{"label": "Action", "func": "act", "execution_mode": "GUI"}
		]
	}`, binLegacyAMD64, binLegacyARM64)

	configPath := filepath.Join(tempDir, "config.json")
	if err := os.WriteFile(configPath, []byte(configJSON), 0644); err != nil {
		t.Fatalf("Failed writing config: %v", err)
	}

	ctx, err := loadAndPrepareConfig(configPath, "production")
	if err != nil {
		t.Fatalf("loadAndPrepareConfig failed: %v", err)
	}

	if ctx.sourcePathLinuxAMD64 != binLegacyAMD64 {
		t.Errorf("Expected sourcePathLinuxAMD64 to be %q, got %q", binLegacyAMD64, ctx.sourcePathLinuxAMD64)
	}
	if ctx.sourcePathLinuxAArch64 != binLegacyARM64 {
		t.Errorf("Expected sourcePathLinuxAArch64 to be %q, got %q", binLegacyARM64, ctx.sourcePathLinuxAArch64)
	}

	hasAMD64 := false
	hasARM64 := false
	for _, arch := range ctx.targetArchs {
		if arch == "linux_amd64" {
			hasAMD64 = true
		}
		if arch == "linux_aarch64" {
			hasARM64 = true
		}
	}
	if !hasAMD64 || !hasARM64 {
		t.Errorf("Expected targetArchs to contain both linux_amd64 and linux_aarch64 from legacy paths, got %v", ctx.targetArchs)
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

func TestUIField_FieldTypeAndLegacyType(t *testing.T) {
	fieldTypes := []string{
		"TXT", "DIR", "FL", "MFL", "FS", "CB", "CK", "FD", "MCB", "LTXT", "TIME", "DATE", "NUM",
	}

	for _, ft := range fieldTypes {
		// 1. Test "field_type" property
		jsonFieldType := fmt.Sprintf(`{"name":"test_field","field_type":%q,"data_type":"string"}`, ft)
		var f1 UIField
		if err := json.Unmarshal([]byte(jsonFieldType), &f1); err != nil {
			t.Fatalf("Failed unmarshaling field_type=%q: %v", ft, err)
		}
		if f1.FieldType != ft {
			t.Errorf("Expected FieldType=%q, got %q", ft, f1.FieldType)
		}
		if f1.Type != ft {
			t.Errorf("Expected backwards-compatible Type=%q, got %q", ft, f1.Type)
		}
		if f1.GetFieldType() != ft {
			t.Errorf("Expected GetFieldType()=%q, got %q", ft, f1.GetFieldType())
		}

		// 2. Test legacy "type" property
		jsonLegacyType := fmt.Sprintf(`{"name":"test_field","type":%q,"data_type":"string"}`, ft)
		var f2 UIField
		if err := json.Unmarshal([]byte(jsonLegacyType), &f2); err != nil {
			t.Fatalf("Failed unmarshaling legacy type=%q: %v", ft, err)
		}
		if f2.FieldType != ft {
			t.Errorf("Expected FieldType=%q from legacy type, got %q", ft, f2.FieldType)
		}
		if f2.Type != ft {
			t.Errorf("Expected Type=%q from legacy type, got %q", ft, f2.Type)
		}
		if f2.GetFieldType() != ft {
			t.Errorf("Expected GetFieldType()=%q from legacy type, got %q", ft, f2.GetFieldType())
		}
	}
}

func TestUIField_MutualExclusivityFailsEarly(t *testing.T) {
	// Attempt to define both "type" and "field_type" on the same field
	jsonConflict := `{"name":"ConflictField","type":"TXT","field_type":"TXT","data_type":"string"}`
	var f UIField
	err := json.Unmarshal([]byte(jsonConflict), &f)
	if err == nil {
		t.Fatal("Expected error when both 'type' and 'field_type' are defined on the same field, got nil")
	}
	expectedSub := "cannot define both 'type' and 'field_type'"
	if !strings.Contains(err.Error(), expectedSub) {
		t.Errorf("Expected error to contain %q, got %q", expectedSub, err.Error())
	}

	// Test with different values
	jsonConflictDiff := `{"name":"DiffConflict","type":"CB","field_type":"TXT","data_type":"string"}`
	var fDiff UIField
	errDiff := json.Unmarshal([]byte(jsonConflictDiff), &fDiff)
	if errDiff == nil {
		t.Fatal("Expected error when both 'type' and 'field_type' are defined on the same field, got nil")
	}
	if !strings.Contains(errDiff.Error(), expectedSub) {
		t.Errorf("Expected error to contain %q, got %q", expectedSub, errDiff.Error())
	}

	// Test that loading a config with conflict fails early in loadAndPrepareConfig
	tempDir := t.TempDir()
	configJSON := `{
		"config": {
			"title": "Conflict App",
			"uuid": "com.example.conflict",
			"script_alias": "test.sh",
			"source_script_amd64": "/tmp/test.sh",
			"source_type": "bash"
		},
		"actions": [
			{
				"label": "Test Action",
				"func": "test_fn",
				"execution_mode": "GUI",
				"fields": [
					{
						"name": "BadField",
						"type": "TXT",
						"field_type": "TXT",
						"data_type": "string"
					}
				]
			}
		]
	}`
	configPath := filepath.Join(tempDir, "config.json")
	if writeErr := os.WriteFile(configPath, []byte(configJSON), 0644); writeErr != nil {
		t.Fatalf("Failed writing temp config: %v", writeErr)
	}

	_, configErr := loadAndPrepareConfig(configPath, "debug")
	if configErr == nil {
		t.Fatal("Expected loadAndPrepareConfig to fail early when a field defines both 'type' and 'field_type', got nil")
	}
	if !strings.Contains(configErr.Error(), expectedSub) {
		t.Errorf("Expected error to contain %q, got %q", expectedSub, configErr.Error())
	}
}

