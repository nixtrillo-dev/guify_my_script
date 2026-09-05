package config_gen

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParsePowerShellFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "module.psm1")
	content := `function Invoke-CustomFilter {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory = $true)] [ValidateNotNullOrEmpty()] [string[]]$InputData,
        [string]$Keyword,
        [switch]$CaseSensitive,
        [ValidateSet('Ascending', 'Descending')] [string]$SortOrder = 'Ascending'
    )
}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	actions, err := parsePowerShellFile(path)
	if err != nil {
		t.Fatalf("parsePowerShellFile failed: %v", err)
	}
	if len(actions) != 1 || actions[0].Func != "Invoke-CustomFilter" {
		t.Fatalf("unexpected actions: %#v", actions)
	}
	if len(actions[0].Fields) != 4 {
		t.Fatalf("expected 4 fields, got %d: %#v", len(actions[0].Fields), actions[0].Fields)
	}
	checks := []struct {
		name, dataType, fieldType string
	}{
		{"Input Data", "list", "TXT"},
		{"Keyword", "string", "TXT"},
		{"Case Sensitive", "bool", "CK"},
		{"Sort Order", "string", "TXT"},
	}
	for i, check := range checks {
		field := actions[0].Fields[i]
		if field.Name != check.name || field.DataType != check.dataType || field.Type != check.fieldType {
			t.Errorf("field %d mismatch: %#v", i, field)
		}
	}
	if actions[0].Fields[2].Default != "0" {
		t.Errorf("expected checkbox default to be string \"0\", got %#v", actions[0].Fields[2].Default)
	}
	encoded, err := json.Marshal(actions[0].Fields[2])
	if err != nil {
		t.Fatal(err)
	}
	if containsJSONKey(string(encoded), "minimum") || containsJSONKey(string(encoded), "maximum") {
		t.Errorf("non-range fields must not include bounds: %s", encoded)
	}
}

func TestCreatePowerShellNumericFieldIncludesNullableBounds(t *testing.T) {
	field := createPowerShellField("[int]$Age")
	if field.Type != "NUM" {
		t.Fatalf("expected NUM field, got %#v", field)
	}
	if field.Minimum == nil || field.Maximum == nil {
		t.Fatalf("NUM fields must include minimum and maximum for schema validation: %#v", field)
	}
	if string(*field.Minimum) != "null" || string(*field.Maximum) != "null" {
		t.Errorf("expected unset numeric bounds to be JSON null, got minimum=%s maximum=%s", *field.Minimum, *field.Maximum)
	}
}

func containsJSONKey(encoded, key string) bool {
	return strings.Contains(encoded, `"`+key+`"`)
}
