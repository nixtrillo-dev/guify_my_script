package config_gen

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestParseJavaScriptFile(t *testing.T) {
	testJS := `#!/usr/bin/env node
// @ts-check

/**
 * Multiplies two values.
 * @param {number} a - The first number
 * @param {number} b - The second number
 * @returns {number} The product
 */
export function multiply(a, b) {
    console.log("a is: " + a + ", b is: " + b);
    return Number(a) * Number(b);
}

/**
 * Adds two values.
 * @param {number} a - The first number
 * @param {number} b - The second number
 * @returns {number} The sum
 */
export function add(a, b) {
    console.log("a is: " + a + ", b is: " + b);
    return Number(a) + Number(b);
}

/**
 * Multiplies or formats values.
 * @param {Object} [options] - The options object (optional)
 * @param {string | number} [options.a="some text"] - The text/number input
 * @param {number} [options.b=14] - The multiplier
 */
function invokeWithObjectTest({ a = "some text", b = 14 } = {}) {
    console.log("a is: " + a + ", b is: " + b);
    if (typeof a === "number" && typeof b === "number") {
        return a * b;
    }
    return a + " x " + b;
}
`
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "mathjstest.js")
	if err := os.WriteFile(testFile, []byte(testJS), 0644); err != nil {
		t.Fatalf("failed to write test JS file: %v", err)
	}

	actions, err := parseJavaScriptFile(testFile)
	if err != nil {
		t.Fatalf("parseJavaScriptFile failed: %v", err)
	}

	t.Logf("Found %d actions", len(actions))
	for i, act := range actions {
		data, _ := json.MarshalIndent(act, "", "  ")
		t.Logf("Action %d:\n%s", i, string(data))
	}

	if len(actions) != 3 {
		t.Errorf("expected 3 actions, got %d", len(actions))
	}

	multiplyFound := false
	addFound := false
	invokeObjFound := false
	for _, act := range actions {
		switch act.Func {
		case "multiply":
			multiplyFound = true
			if len(act.Fields) != 2 {
				t.Errorf("multiply: expected 2 fields, got %d", len(act.Fields))
			}
			if len(act.Fields) >= 1 {
				f := act.Fields[0]
				if f.Name != "A" {
					t.Errorf("multiply field 0 name: expected 'A', got %q", f.Name)
				}
				if f.DataType != "float" && f.DataType != "int" {
					t.Errorf("multiply field 0 data_type: expected float/int from @param {number}, got %q", f.DataType)
				}
			}
		case "add":
			addFound = true
			if len(act.Fields) != 2 {
				t.Errorf("add: expected 2 fields, got %d", len(act.Fields))
			}
		case "invokeWithObjectTest":
			invokeObjFound = true
			if !act.InvokeAsObject {
				t.Errorf("invokeWithObjectTest: expected InvokeAsObject=true")
			}
			if len(act.Fields) != 2 {
				t.Errorf("invokeWithObjectTest: expected 2 fields from destructured obj, got %d", len(act.Fields))
			}
			for _, f := range act.Fields {
				t.Logf("  invoke field: name=%s type=%s data_type=%s default=%v arg_name=%s",
					f.Name, f.Type, f.DataType, f.Default, f.ArgName)
			}
		}
	}

	if !multiplyFound {
		t.Error("multiply function not found")
	}
	if !addFound {
		t.Error("add function not found")
	}
	if !invokeObjFound {
		t.Error("invokeWithObjectTest function not found")
	}
}

func TestGenerateGUIConfigForJavaScript(t *testing.T) {
	testJS := `#!/usr/bin/env node
// @ts-check

/**
 * Multiplies two values.
 * @param {number} a - The first number
 * @param {number} b - The second number
 * @returns {number} The product
 */
export function multiply(a, b) {
    return Number(a) * Number(b);
}

/**
 * Object-param test
 * @param {Object} [options]
 * @param {string} [options.name="hello"]
 * @param {number} [options.count=5]
 */
function objectFunc({ name = "hello", count = 5 } = {}) {
    return name + count;
}
`
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "module.js")
	if err := os.WriteFile(testFile, []byte(testJS), 0644); err != nil {
		t.Fatalf("failed to write test JS file: %v", err)
	}

	outFile := filepath.Join(tmpDir, "config.json")
	err := GenerateGUIConfig(testFile, outFile, "", true, "INFO", false)
	if err != nil {
		t.Fatalf("GenerateGUIConfig failed: %v", err)
	}

	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("failed to read output config: %v", err)
	}

	var cfg RootGUIConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("failed to parse output JSON: %v\n%s", err, string(data))
	}

	if cfg.Config.SourceType != "javascript" {
		t.Errorf("expected source_type='javascript', got %q", cfg.Config.SourceType)
	}

	t.Logf("Generated config successfully, source_type=%s, actions=%d", cfg.Config.SourceType, len(cfg.Actions))
	for _, a := range cfg.Actions {
		t.Logf("  action: %s (invokeAsObject=%v, fields=%d)", a.Func, a.InvokeAsObject, len(a.Fields))
	}
}
