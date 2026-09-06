package rust_gen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
)

func TestTemplateParse(t *testing.T) {
	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}
	_, err := template.New("rust_bridge").Funcs(funcMap).Parse(bridgeTemplate)
	if err != nil {
		t.Fatalf("Template parse error: %v", err)
	}
}

func TestParseRustFunctions_ExampleFile(t *testing.T) {
	examplePath := "../example/rust_support_implementation/rust_test_script/core_utils/src/lib.rs"
	content, err := os.ReadFile(examplePath)
	if err != nil {
		t.Fatalf("Failed reading example lib.rs: %v", err)
	}

	fns := parseRustFunctions(string(content), "core_utils", false)
	if len(fns) != 4 {
		t.Fatalf("Expected 4 public functions, found %d", len(fns))
	}

	expectedMap := map[string]struct {
		argCount int
		goName   string
		bashName string
	}{
		"print_version":      {argCount: 0, goName: "PrintVersion", bashName: "print_version"},
		"print_greeting":     {argCount: 1, goName: "PrintGreeting", bashName: "print_greeting"},
		"print_sum":          {argCount: 2, goName: "PrintSum", bashName: "print_sum"},
		"print_user_profile": {argCount: 3, goName: "PrintUserProfile", bashName: "print_user_profile"},
	}

	for _, fn := range fns {
		exp, ok := expectedMap[fn.RustName]
		if !ok {
			t.Errorf("Unexpected function extracted: %s", fn.RustName)
			continue
		}
		if len(fn.Args) != exp.argCount {
			t.Errorf("Function %s: expected %d args, got %d", fn.RustName, exp.argCount, len(fn.Args))
		}
		if fn.GoName != exp.goName {
			t.Errorf("Function %s: expected GoName %s, got %s", fn.RustName, exp.goName, fn.GoName)
		}
		if fn.BashName != exp.bashName {
			t.Errorf("Function %s: expected BashName %s, got %s", fn.RustName, exp.bashName, fn.BashName)
		}
	}

	// Verify specific parameter types for print_user_profile
	var profileFn *FuncMetaData
	for i := range fns {
		if fns[i].RustName == "print_user_profile" {
			profileFn = &fns[i]
			break
		}
	}
	if profileFn == nil {
		t.Fatal("print_user_profile not found")
	}

	if profileFn.Args[0].Name != "username" || profileFn.Args[0].Type != "string" {
		t.Errorf("Arg 0: expected username:string, got %s:%s", profileFn.Args[0].Name, profileFn.Args[0].Type)
	}
	if profileFn.Args[1].Name != "age" || profileFn.Args[1].Type != "int" || profileFn.Args[1].RawType != "u8" {
		t.Errorf("Arg 1: expected age:int(u8), got %s:%s(%s)", profileFn.Args[1].Name, profileFn.Args[1].Type, profileFn.Args[1].RawType)
	}
	if profileFn.Args[2].Name != "status" || profileFn.Args[2].Type != "string" {
		t.Errorf("Arg 2: expected status:string, got %s:%s", profileFn.Args[2].Name, profileFn.Args[2].Type)
	}
}

func TestGenerateRustBridge_RendersValidBridge(t *testing.T) {
	tempDir := t.TempDir()
	srcDir := filepath.Join(tempDir, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}

	cargoToml := `[package]
name = "my_rust_crate"
version = "0.1.0"
edition = "2021"
`
	if err := os.WriteFile(filepath.Join(tempDir, "Cargo.toml"), []byte(cargoToml), 0644); err != nil {
		t.Fatalf("write Cargo.toml failed: %v", err)
	}

	libRs := `pub fn greet(msg: &str) {
    println!("{}", msg);
}

#[cfg(test)]
mod tests {
    #[test]
    fn it_works() {}
}
`
	if err := os.WriteFile(filepath.Join(srcDir, "lib.rs"), []byte(libRs), 0644); err != nil {
		t.Fatalf("write lib.rs failed: %v", err)
	}

	outputBridge := filepath.Join(srcDir, "main.rs")
	targetDir, absOutput, err := GenerateRustBridge(tempDir, outputBridge, "", "DEBUG", false)
	if err != nil {
		t.Fatalf("GenerateBridge failed: %v", err)
	}

	if targetDir != tempDir {
		t.Errorf("Expected targetDir %s, got %s", tempDir, targetDir)
	}
	if absOutput != outputBridge {
		t.Errorf("Expected absOutput %s, got %s", outputBridge, absOutput)
	}

	bridgeBytes, err := os.ReadFile(absOutput)
	if err != nil {
		t.Fatalf("Failed reading generated bridge: %v", err)
	}
	bridgeContent := string(bridgeBytes)

	if !strings.Contains(bridgeContent, "pub extern \"C\" fn RustGreet") {
		t.Errorf("Bridge missing RustGreet export: %s", bridgeContent)
	}
	if !strings.Contains(bridgeContent, "pub extern \"C\" fn rust_greet") {
		t.Errorf("Bridge missing rust_greet export: %s", bridgeContent)
	}
	if !strings.Contains(bridgeContent, "match args[1].as_str()") {
		t.Errorf("Bridge missing CLI router: %s", bridgeContent)
	}
	if !strings.Contains(bridgeContent, "\"greet\" =>") {
		t.Errorf("Bridge missing 'greet' match branch: %s", bridgeContent)
	}
}
