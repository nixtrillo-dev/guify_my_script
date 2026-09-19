package config_gen

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateFieldIncludesBoundsOnlyForSupportedFieldTypes(t *testing.T) {
	boundsSupported := map[string]bool{
		"MCB":  true,
		"MFL":  true,
		"TIME": true,
		"DATE": true,
		"NUM":  true,
	}

	for index, fieldType := range uiFieldTypes {
		field := createField("example", index)
		encoded, err := json.Marshal(field)
		if err != nil {
			t.Fatalf("marshal %s field: %v", fieldType, err)
		}

		var object map[string]json.RawMessage
		if err := json.Unmarshal(encoded, &object); err != nil {
			t.Fatalf("unmarshal %s field: %v", fieldType, err)
		}
		_, hasMinimum := object["minimum"]
		_, hasMaximum := object["maximum"]

		if boundsSupported[fieldType] {
			if !hasMinimum || !hasMaximum {
				t.Errorf("%s fields must include minimum and maximum", fieldType)
			}
		} else if hasMinimum || hasMaximum {
			t.Errorf("%s fields must not include minimum or maximum", fieldType)
		}
	}
}

func TestGenerateGUIConfig_GoSource(t *testing.T) {
	tempDir := t.TempDir()
	goFile := filepath.Join(tempDir, "sample.go")
	goContent := `package main

func DoWork(name string) {}
`
	if err := os.WriteFile(goFile, []byte(goContent), 0644); err != nil {
		t.Fatalf("Failed creating dummy go file: %v", err)
	}

	outputJSON := filepath.Join(tempDir, "config.json")
	err := GenerateGUIConfig(goFile, outputJSON, "my_binary", true, "DEBUG", false)
	if err != nil {
		t.Fatalf("GenerateGUIConfig failed: %v", err)
	}

	data, err := os.ReadFile(outputJSON)
	if err != nil {
		t.Fatalf("Failed reading generated JSON: %v", err)
	}

	var root RootGUIConfig
	if err := json.Unmarshal(data, &root); err != nil {
		t.Fatalf("Failed unmarshaling config JSON: %v", err)
	}

	expectedBase, _ := filepath.Abs("my_binary")
	expectedLinuxAMD64 := expectedBase + "_linux_amd64"
	expectedLinuxAArch64 := expectedBase + "_linux_aarch64"
	expectedAlias := "my_binary"

	if root.Config.ScriptAlias != expectedAlias {
		t.Errorf("Expected script_alias to be %q, got %q", expectedAlias, root.Config.ScriptAlias)
	}
	if root.Config.SourceScriptLinuxAMD64 != expectedLinuxAMD64 {
		t.Errorf("Expected source_script_linux_amd64 to be %q, got %q", expectedLinuxAMD64, root.Config.SourceScriptLinuxAMD64)
	}
	if root.Config.SourceScriptLinuxAArch64 != expectedLinuxAArch64 {
		t.Errorf("Expected source_script_linux_aarch64 to be %q, got %q", expectedLinuxAArch64, root.Config.SourceScriptLinuxAArch64)
	}
	if root.Config.SourceType != "go" {
		t.Errorf("Expected source_type to be 'go', got %q", root.Config.SourceType)
	}

	// Verify data_type on generated fields
	for _, action := range root.Actions {
		for _, field := range action.Fields {
			if field.DataType == "" {
				t.Errorf("Field %q in action %q is missing data_type", field.Name, action.Label)
			}
		}
	}
}

func TestGenerateGUIConfig_ExistingOutputRefreshesSchema(t *testing.T) {
	tempDir := t.TempDir()
	goFile := filepath.Join(tempDir, "sample.go")
	if err := os.WriteFile(goFile, []byte("package main\n\nfunc DoWork() {}\n"), 0644); err != nil {
		t.Fatalf("Failed creating dummy go file: %v", err)
	}

	outputJSON := filepath.Join(tempDir, "config.json")
	if err := os.WriteFile(outputJSON, []byte("existing config"), 0644); err != nil {
		t.Fatalf("Failed creating existing config: %v", err)
	}
	schemaPath := filepath.Join(tempDir, "guify_my_script_config_schema.json")
	if err := os.WriteFile(schemaPath, []byte("stale schema"), 0644); err != nil {
		t.Fatalf("Failed creating stale schema: %v", err)
	}

	if err := GenerateGUIConfig(goFile, outputJSON, "", false, "DEBUG", false); err != nil {
		t.Fatalf("GenerateGUIConfig failed: %v", err)
	}

	actualSchema, err := os.ReadFile(schemaPath)
	if err != nil {
		t.Fatalf("Failed reading refreshed schema: %v", err)
	}
	expectedSchema, err := schemaFile.ReadFile("guify_my_script_config_schema.json")
	if err != nil {
		t.Fatalf("Failed reading embedded schema: %v", err)
	}
	if string(actualSchema) != string(expectedSchema) {
		t.Error("Existing output should still refresh the schema copy")
	}
}

func TestGenerateGUIConfig_BashSource(t *testing.T) {
	tempDir := t.TempDir()
	shFile := filepath.Join(tempDir, "script.sh")
	shContent := `#!/bin/bash
function my_action() {
    echo "hello"
}
`
	if err := os.WriteFile(shFile, []byte(shContent), 0644); err != nil {
		t.Fatalf("Failed creating dummy bash file: %v", err)
	}

	outputJSON := filepath.Join(tempDir, "config.json")
	err := GenerateGUIConfig(shFile, outputJSON, "", true, "DEBUG", false)
	if err != nil {
		t.Fatalf("GenerateGUIConfig failed: %v", err)
	}

	data, err := os.ReadFile(outputJSON)
	if err != nil {
		t.Fatalf("Failed reading generated JSON: %v", err)
	}

	var root RootGUIConfig
	if err := json.Unmarshal(data, &root); err != nil {
		t.Fatalf("Failed unmarshaling config JSON: %v", err)
	}

	expectedPath, _ := filepath.Abs(shFile)
	expectedAlias := "script.sh"

	if root.Config.ScriptAlias != expectedAlias {
		t.Errorf("Expected script_alias to be %q, got %q", expectedAlias, root.Config.ScriptAlias)
	}
	if root.Config.SourceScriptLinuxAMD64 != expectedPath {
		t.Errorf("Expected source_script_linux_amd64 for bash to match full script path (%q), got %q", expectedPath, root.Config.SourceScriptLinuxAMD64)
	}
	if root.Config.SourceScriptLinuxAArch64 != expectedPath {
		t.Errorf("Expected source_script_linux_aarch64 for bash to match full script path (%q), got %q", expectedPath, root.Config.SourceScriptLinuxAArch64)
	}
	if root.Config.SourceType != "bash" {
		t.Errorf("Expected source_type to be 'bash', got %q", root.Config.SourceType)
	}
}

func TestGenerateGUIConfig_PythonSource(t *testing.T) {
	tempDir := t.TempDir()
	pyFile := filepath.Join(tempDir, "test_pyscript.py")
	pyContent := `def say_hello(name: str) -> str:
    print(f"Hello, {name}!")
    return f"Hello, {name}!"

def calculate_total(price: float, tax: float) -> float:
    print(price + (price * tax))
    return price + (price * tax)

def greet_user(name: str, message: str) -> None:
    print("--- Starting Greeting Process ---")
    print(f"Hello, {name}! Welcome to the script.")
    print(f"Custom message: {message}")
    print("--- Process Complete ---")
`
	if err := os.WriteFile(pyFile, []byte(pyContent), 0644); err != nil {
		t.Fatalf("Failed creating dummy python file: %v", err)
	}

	outputJSON := filepath.Join(tempDir, "config.json")
	err := GenerateGUIConfig(pyFile, outputJSON, "", true, "DEBUG", false)
	if err != nil {
		t.Fatalf("GenerateGUIConfig failed: %v", err)
	}

	data, err := os.ReadFile(outputJSON)
	if err != nil {
		t.Fatalf("Failed reading generated JSON: %v", err)
	}

	var root RootGUIConfig
	if err := json.Unmarshal(data, &root); err != nil {
		t.Fatalf("Failed unmarshaling config JSON: %v", err)
	}

	expectedPath, _ := filepath.Abs(pyFile)
	expectedAlias := "test_pyscript.py"

	if root.Config.ScriptAlias != expectedAlias {
		t.Errorf("Expected script_alias to be %q, got %q", expectedAlias, root.Config.ScriptAlias)
	}
	if root.Config.SourceScriptLinuxAMD64 != expectedPath {
		t.Errorf("Expected source_script_linux_amd64 for python to match full script path (%q), got %q", expectedPath, root.Config.SourceScriptLinuxAMD64)
	}
	if root.Config.SourceScriptLinuxAArch64 != expectedPath {
		t.Errorf("Expected source_script_linux_aarch64 for python to match full script path (%q), got %q", expectedPath, root.Config.SourceScriptLinuxAArch64)
	}
	if root.Config.SourceType != "python" {
		t.Errorf("Expected source_type to be 'python', got %q", root.Config.SourceType)
	}

	if len(root.Actions) != 3 {
		t.Fatalf("Expected 3 actions parsed from python file, got %d", len(root.Actions))
	}

	// Action 1: Say Hello
	if root.Actions[0].Func != "say_hello" || len(root.Actions[0].Fields) != 1 {
		t.Errorf("Action 0 mismatch: func=%s, fields=%d", root.Actions[0].Func, len(root.Actions[0].Fields))
	} else {
		f := root.Actions[0].Fields[0]
		if f.Name != "Name" || f.DataType != "string" || f.Type != "TXT" {
			t.Errorf("Action 0 field mismatch: name=%s, data_type=%s, type=%s", f.Name, f.DataType, f.Type)
		}
	}

	// Action 2: Calculate Total
	if root.Actions[1].Func != "calculate_total" || len(root.Actions[1].Fields) != 2 {
		t.Errorf("Action 1 mismatch: func=%s, fields=%d", root.Actions[1].Func, len(root.Actions[1].Fields))
	} else {
		f0 := root.Actions[1].Fields[0]
		if f0.Name != "Price" || f0.DataType != "float" || f0.Type != "NUM" {
			t.Errorf("Action 1 field 0 mismatch: name=%s, data_type=%s, type=%s", f0.Name, f0.DataType, f0.Type)
		}
		f1 := root.Actions[1].Fields[1]
		if f1.Name != "Tax" || f1.DataType != "float" || f1.Type != "NUM" {
			t.Errorf("Action 1 field 1 mismatch: name=%s, data_type=%s, type=%s", f1.Name, f1.DataType, f1.Type)
		}
	}

	// Action 3: Greet User
	if root.Actions[2].Func != "greet_user" || len(root.Actions[2].Fields) != 2 {
		t.Errorf("Action 2 mismatch: func=%s, fields=%d", root.Actions[2].Func, len(root.Actions[2].Fields))
	} else {
		f0 := root.Actions[2].Fields[0]
		if f0.Name != "Name" || f0.DataType != "string" || f0.Type != "TXT" {
			t.Errorf("Action 2 field 0 mismatch: name=%s, data_type=%s, type=%s", f0.Name, f0.DataType, f0.Type)
		}
		f1 := root.Actions[2].Fields[1]
		if f1.Name != "Message" || f1.DataType != "string" || f1.Type != "TXT" {
			t.Errorf("Action 2 field 1 mismatch: name=%s, data_type=%s, type=%s", f1.Name, f1.DataType, f1.Type)
		}
	}
}

func TestGenerateGUIConfig_PythonUntypedFunctionDefaultsToString(t *testing.T) {
	tempDir := t.TempDir()
	pyFile := filepath.Join(tempDir, "module.py")
	pyContent := `def untyped(value, count=2):
    pass

class Internal:
    def method(self, ignored: int):
        pass
`
	if err := os.WriteFile(pyFile, []byte(pyContent), 0644); err != nil {
		t.Fatalf("Failed creating dummy python file: %v", err)
	}

	outputJSON := filepath.Join(tempDir, "config.json")
	if err := GenerateGUIConfig(pyFile, outputJSON, "custom-app", true, "DEBUG", false); err != nil {
		t.Fatalf("GenerateGUIConfig failed: %v", err)
	}

	data, err := os.ReadFile(outputJSON)
	if err != nil {
		t.Fatalf("Failed reading generated JSON: %v", err)
	}
	var root RootGUIConfig
	if err := json.Unmarshal(data, &root); err != nil {
		t.Fatalf("Failed unmarshaling config JSON: %v", err)
	}

	if root.Config.SourceScriptAMD64 != pyFile || root.Config.SourceScriptARM64 != pyFile {
		t.Fatalf("Python source paths must remain the module path when binaryName is supplied: %#v", root.Config)
	}
	if len(root.Actions) != 1 || root.Actions[0].Func != "untyped" {
		t.Fatalf("Expected only the module-level untyped function, got %#v", root.Actions)
	}
	for _, field := range root.Actions[0].Fields {
		if field.DataType != "string" || field.Type != "TXT" {
			t.Errorf("Untyped Python field %#v should default to a string text field", field)
		}
	}
}

func TestSchemaContainsRequiredProperties(t *testing.T) {
	schemaData, err := schemaFile.ReadFile("guify_my_script_config_schema.json")
	if err != nil {
		t.Fatalf("Failed reading embedded schema: %v", err)
	}

	var schema map[string]interface{}
	if err := json.Unmarshal(schemaData, &schema); err != nil {
		t.Fatalf("Failed unmarshaling schema JSON: %v", err)
	}

	defs, ok := schema["$defs"].(map[string]interface{})
	if !ok {
		t.Fatalf("Missing $defs in schema")
	}

	uiConfig, ok := defs["uiConfig"].(map[string]interface{})
	if !ok {
		t.Fatalf("Missing uiConfig in $defs")
	}

	uiConfigProps, ok := uiConfig["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("Missing properties in uiConfig")
	}

	sourceTypeProp, ok := uiConfigProps["source_type"].(map[string]interface{})
	if !ok {
		t.Fatalf("Missing source_type in uiConfig properties")
	}
	enums, ok := sourceTypeProp["enum"].([]interface{})
	if !ok {
		t.Fatalf("Missing enum in source_type property")
	}
	hasPython := false
	hasJavascript := false
	hasPowerShell := false
	for _, e := range enums {
		if e == "python" {
			hasPython = true
		}
		if e == "javascript" {
			hasJavascript = true
		}
		if e == "powershell" {
			hasPowerShell = true
		}
	}
	if !hasPython {
		t.Errorf("source_type enum does not include 'python'")
	}
	if !hasJavascript {
		t.Errorf("source_type enum does not include 'javascript'")
	}
	if !hasPowerShell {
		t.Errorf("source_type enum does not include 'powershell'")
	}

	uiAction, ok := defs["uiAction"].(map[string]interface{})
	if !ok {
		t.Fatalf("Missing uiAction in $defs")
	}
	actionProps, ok := uiAction["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("Missing properties in uiAction")
	}
	if _, exists := actionProps["invoke_as_object"]; !exists {
		t.Errorf("Field 'invoke_as_object' is missing from uiAction properties (required for JavaScript object destructuring)")
	}
	execModeProp, exists := actionProps["execution_mode"].(map[string]interface{})
	if !exists {
		t.Errorf("Property 'execution_mode' is missing from uiAction properties")
	} else {
		if defVal, ok := execModeProp["default"].(string); !ok || defVal != "GUI" {
			t.Errorf("Expected 'execution_mode' default to be 'GUI', got %v", execModeProp["default"])
		}
	}
	actionRequired, ok := uiAction["required"].([]interface{})
	if !ok {
		t.Fatalf("Missing required in uiAction")
	}
	hasExecModeReq := false
	for _, r := range actionRequired {
		if r == "execution_mode" {
			hasExecModeReq = true
			break
		}
	}
	if !hasExecModeReq {
		t.Errorf("Expected 'execution_mode' to be in uiAction required list")
	}

	uiField, ok := defs["uiField"].(map[string]interface{})
	if !ok {
		t.Fatalf("Missing uiField in $defs")
	}

	required, ok := uiField["required"].([]interface{})
	if !ok {
		t.Fatalf("Missing required in uiField")
	}

	reqMap := make(map[string]bool)
	for _, r := range required {
		if s, ok := r.(string); ok {
			reqMap[s] = true
		}
	}

	for _, field := range []string{"name", "data_type", "enabled", "advanced_menu", "json_endpoints", "refresh_fields_on_change"} {
		if !reqMap[field] {
			t.Errorf("Field %q is missing from uiField required list", field)
		}
	}

	anyOf, ok := uiField["anyOf"].([]interface{})
	if !ok || len(anyOf) != 2 {
		t.Fatalf("Expected uiField to define anyOf with 2 branches for field_type and type, got %v", uiField["anyOf"])
	}
	hasFieldTypeAnyOf := false
	hasTypeAnyOf := false
	for _, branch := range anyOf {
		if bMap, ok := branch.(map[string]interface{}); ok {
			if bReq, ok := bMap["required"].([]interface{}); ok {
				for _, r := range bReq {
					if r == "field_type" {
						hasFieldTypeAnyOf = true
					}
					if r == "type" {
						hasTypeAnyOf = true
					}
				}
			}
		}
	}
	if !hasFieldTypeAnyOf || !hasTypeAnyOf {
		t.Errorf("Expected anyOf to require field_type or type, got %v", anyOf)
	}

	notClause, ok := uiField["not"].(map[string]interface{})
	if !ok {
		t.Errorf("Expected uiField to define 'not' clause for mutual exclusivity")
	} else if notReq, ok := notClause["required"].([]interface{}); !ok || len(notReq) != 2 {
		t.Errorf("Expected 'not' clause to specify required: ['type', 'field_type'], got %v", notClause)
	}

	props, ok := uiField["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("Missing properties in uiField")
	}

	fieldTypeProp, exists := props["field_type"].(map[string]interface{})
	if !exists {
		t.Errorf("Field 'field_type' is missing from uiField properties")
	} else {
		desc, hasDesc := fieldTypeProp["description"].(string)
		if !hasDesc || desc == "" {
			t.Errorf("Expected 'field_type' to have a non-empty description")
		}
		enums, hasEnums := fieldTypeProp["enum"].([]interface{})
		if !hasEnums || len(enums) != 13 {
			t.Errorf("Expected 'field_type' enum to contain 13 types, got %v", enums)
		}
		enumDescs, hasEnumDescs := fieldTypeProp["enumDescriptions"].([]interface{})
		if !hasEnumDescs || len(enumDescs) != 13 {
			t.Errorf("Expected 'field_type' enumDescriptions to contain 13 items, got %v", enumDescs)
		}
	}

	typeProp, exists := props["type"].(map[string]interface{})
	if !exists {
		t.Errorf("Field 'type' is missing from uiField properties (needed for backwards compatibility)")
	} else {
		desc, hasDesc := typeProp["description"].(string)
		if !hasDesc || !strings.Contains(desc, "Deprecated") {
			t.Errorf("Expected 'type' property to be marked as deprecated alias, got description: %q", desc)
		}
	}

	if _, exists := props["data_type"]; !exists {
		t.Errorf("Field 'data_type' is missing from uiField properties")
	}
	if _, exists := props["arg_name"]; !exists {
		t.Errorf("Field 'arg_name' is missing from uiField properties (required for JavaScript/parameter naming)")
	}
}

func TestGenerateGUIConfig_RustSource(t *testing.T) {
	tempDir := t.TempDir()
	rsFile := filepath.Join(tempDir, "main.rs")
	rsContent := `
pub fn print_greeting(name: &str) {}
pub fn print_sum(a: i32, b: i32) {}
`
	if err := os.WriteFile(rsFile, []byte(rsContent), 0644); err != nil {
		t.Fatalf("Failed creating dummy rs file: %v", err)
	}

	outputJSON := filepath.Join(tempDir, "config.json")
	err := GenerateGUIConfig(rsFile, outputJSON, "my_rust_binary", true, "DEBUG", false)
	if err != nil {
		t.Fatalf("GenerateGUIConfig failed: %v", err)
	}

	data, err := os.ReadFile(outputJSON)
	if err != nil {
		t.Fatalf("Failed reading generated JSON: %v", err)
	}

	var root RootGUIConfig
	if err := json.Unmarshal(data, &root); err != nil {
		t.Fatalf("Failed unmarshaling config JSON: %v", err)
	}

	expectedBase, _ := filepath.Abs("my_rust_binary")
	expectedLinuxAMD64 := expectedBase + "_linux_amd64"
	expectedLinuxAArch64 := expectedBase + "_linux_aarch64"
	expectedAlias := "my_rust_binary"

	if root.Config.ScriptAlias != expectedAlias {
		t.Errorf("Expected script_alias to be %q, got %q", expectedAlias, root.Config.ScriptAlias)
	}
	if root.Config.SourceScriptLinuxAMD64 != expectedLinuxAMD64 {
		t.Errorf("Expected source_script_linux_amd64 to be %q, got %q", expectedLinuxAMD64, root.Config.SourceScriptLinuxAMD64)
	}
	if root.Config.SourceScriptLinuxAArch64 != expectedLinuxAArch64 {
		t.Errorf("Expected source_script_linux_aarch64 to be %q, got %q", expectedLinuxAArch64, root.Config.SourceScriptLinuxAArch64)
	}
	if root.Config.SourceType != "rust" {
		t.Errorf("Expected source_type to be 'rust', got %q", root.Config.SourceType)
	}
	if len(root.Actions) != 2 {
		t.Fatalf("Expected 2 actions, got %d", len(root.Actions))
	}
	for _, act := range root.Actions {
		if act.ExecutionMode != "GUI" {
			t.Errorf("Expected action %q execution_mode to be 'GUI', got %q", act.Label, act.ExecutionMode)
		}
	}
}

func TestUIAction_UnmarshalJSON_ExecutionModeAndLegacyMode(t *testing.T) {
	// New style with execution_mode
	newJSON := `{"label":"Test Action","func":"test_fn","execution_mode":"GUI"}`
	var act1 UIAction
	if err := json.Unmarshal([]byte(newJSON), &act1); err != nil {
		t.Fatalf("Failed unmarshaling action with execution_mode: %v", err)
	}
	if act1.ExecutionMode != "GUI" {
		t.Errorf("Expected ExecutionMode to be 'GUI', got %q", act1.ExecutionMode)
	}

	// Legacy style with mode
	legacyJSON := `{"label":"Legacy Action","func":"legacy_fn","mode":"g"}`
	var act2 UIAction
	if err := json.Unmarshal([]byte(legacyJSON), &act2); err != nil {
		t.Fatalf("Failed unmarshaling legacy action with mode: %v", err)
	}
	if act2.ExecutionMode != "g" {
		t.Errorf("Expected ExecutionMode fallback to be 'g', got %q", act2.ExecutionMode)
	}
}

func TestUIConfig_UnmarshalJSON_LegacySourceScriptFields(t *testing.T) {
	legacyJSON := `{
		"title": "Legacy App",
		"icon": "icon.png",
		"uuid": "com.test.legacy",
		"script_alias": "legacy_bin",
		"source_script_amd64": "/path/to/legacy_amd64",
		"source_script_arm64": "/path/to/legacy_arm64",
		"source_type": "go",
		"startup": "tray",
		"tray_on_minimize": true,
		"instance_mode": "replace"
	}`

	var cfg UIConfig
	if err := json.Unmarshal([]byte(legacyJSON), &cfg); err != nil {
		t.Fatalf("Failed unmarshaling legacy UIConfig: %v", err)
	}

	if cfg.SourceScriptLinuxAMD64 != "/path/to/legacy_amd64" {
		t.Errorf("Expected SourceScriptLinuxAMD64 to be populated from legacy field, got %q", cfg.SourceScriptLinuxAMD64)
	}
	if cfg.SourceScriptLinuxAArch64 != "/path/to/legacy_arm64" {
		t.Errorf("Expected SourceScriptLinuxAArch64 to be populated from legacy field, got %q", cfg.SourceScriptLinuxAArch64)
	}
}
