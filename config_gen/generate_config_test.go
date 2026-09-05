package config_gen

import (
	"encoding/json"
	"os"
	"path/filepath"
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
	expectedAMD64 := expectedBase + "_amd64"
	expectedARM64 := expectedBase + "_arm64"
	expectedAlias := "my_binary"

	if root.Config.ScriptAlias != expectedAlias {
		t.Errorf("Expected script_alias to be %q, got %q", expectedAlias, root.Config.ScriptAlias)
	}
	if root.Config.SourceScriptAMD64 != expectedAMD64 {
		t.Errorf("Expected source_script_amd64 to be %q, got %q", expectedAMD64, root.Config.SourceScriptAMD64)
	}
	if root.Config.SourceScriptARM64 != expectedARM64 {
		t.Errorf("Expected source_script_arm64 to be %q, got %q", expectedARM64, root.Config.SourceScriptARM64)
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
	if root.Config.SourceScriptAMD64 != expectedPath {
		t.Errorf("Expected source_script_amd64 for bash to match full script path (%q), got %q", expectedPath, root.Config.SourceScriptAMD64)
	}
	if root.Config.SourceScriptARM64 != expectedPath {
		t.Errorf("Expected source_script_arm64 for bash to match full script path (%q), got %q", expectedPath, root.Config.SourceScriptARM64)
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
	if root.Config.SourceScriptAMD64 != expectedPath {
		t.Errorf("Expected source_script_amd64 for python to match full script path (%q), got %q", expectedPath, root.Config.SourceScriptAMD64)
	}
	if root.Config.SourceScriptARM64 != expectedPath {
		t.Errorf("Expected source_script_arm64 for python to match full script path (%q), got %q", expectedPath, root.Config.SourceScriptARM64)
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

	for _, field := range []string{"name", "type", "data_type", "enabled", "advanced_menu", "json_endpoints", "refresh_fields_on_change"} {
		if !reqMap[field] {
			t.Errorf("Field %q is missing from uiField required list", field)
		}
	}

	props, ok := uiField["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("Missing properties in uiField")
	}

	if _, exists := props["data_type"]; !exists {
		t.Errorf("Field 'data_type' is missing from uiField properties")
	}
	if _, exists := props["arg_name"]; !exists {
		t.Errorf("Field 'arg_name' is missing from uiField properties (required for JavaScript/parameter naming)")
	}
}
