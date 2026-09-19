package go_gen

import (
	"bytes"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func TestGenerateBridge_EnforcesProjectDirContainment(t *testing.T) {
	tempDir := t.TempDir()

	// Absolute outputBridge outside the project directory is rejected.
	outsideBridge := filepath.Join(filepath.Dir(tempDir), "escaped_bridge.go")
	_, _, err := GenerateBridge(tempDir, outsideBridge, "", "DEBUG", false)
	if err == nil {
		t.Fatal("Expected error when bridge is outside the project dir, got nil")
	}
	if !strings.Contains(err.Error(), "must be located inside the target project directory") {
		t.Errorf("Unexpected error message: %v", err)
	}

	// CWD-relative outputBridge escaping the project directory is rejected.
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	_, _, err = GenerateBridge(tempDir, "../escaped_bridge.go", "", "DEBUG", false)
	if err == nil {
		t.Fatal("Expected error for CWD-relative bridge outside project dir, got nil")
	}
}

func TestGenerateBridge_RelativeOutputResolvesFromCWD(t *testing.T) {
	tempDir := t.TempDir()
	mainGo := filepath.Join(tempDir, "main.go")
	source := "package main\n\nfunc greet(name string) string {\n\treturn \"hello \" + name\n}\n\nfunc main() {\n\tprintln(greet(\"world\"))\n}\n"
	if err := os.WriteFile(mainGo, []byte(source), 0644); err != nil {
		t.Fatalf("write main.go failed: %v", err)
	}

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	// CWD-relative path that loops back into the project resolves cleanly
	// (regression: this used to be joined onto targetDir, doubling the path).
	loopBack := filepath.Join("..", filepath.Base(tempDir), "custom_bridge.go")
	targetDir, absBridge, err := GenerateBridge(tempDir, loopBack, "", "DEBUG", false)
	if err != nil {
		t.Fatalf("GenerateBridge failed: %v", err)
	}
	if targetDir != tempDir {
		t.Errorf("Expected targetDir %s, got %s", tempDir, targetDir)
	}
	expected := filepath.Join(tempDir, "custom_bridge.go")
	if absBridge != expected {
		t.Errorf("Expected bridge at %s, got %s", expected, absBridge)
	}
	if _, err := os.Stat(expected); err != nil {
		t.Errorf("Bridge file not created at CWD-resolved location: %v", err)
	}
}

func writeGoBridgeFixture(t *testing.T) string {
	t.Helper()
	tempDir := t.TempDir()

	packages := []struct{ dir, name, source string }{
		{"alpha", "alpha", "package alpha\n\nfunc AlphaDo(name string) string {\n\treturn name\n}\n"},
		{"beta", "beta", "package beta\n\nfunc BetaFlag(flag *bool) int {\n\tif flag != nil && *flag {\n\t\treturn 1\n\t}\n\treturn 0\n}\n"},
		{"gamma", "gamma", "package gamma\n\nfunc GammaDo(x int) int {\n\treturn x\n}\n"},
	}
	for _, pkg := range packages {
		pkgDir := filepath.Join(tempDir, pkg.dir)
		if err := os.MkdirAll(pkgDir, 0755); err != nil {
			t.Fatalf("mkdir %s failed: %v", pkgDir, err)
		}
		if err := os.WriteFile(filepath.Join(pkgDir, "lib.go"), []byte(pkg.source), 0644); err != nil {
			t.Fatalf("write %s/lib.go failed: %v", pkgDir, err)
		}
	}
	return tempDir
}

func TestExtractMetadata_SortedDeterministicImports(t *testing.T) {
	tempDir := writeGoBridgeFixture(t)
	moduleBaseName := filepath.Base(tempDir)

	var targetFiles []string
	err := filepath.WalkDir(tempDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".go" {
			targetFiles = append(targetFiles, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk failed: %v", err)
	}

	// Map iteration order is randomized per run, so with multiple dynamic
	// imports each call would otherwise shuffle them. Compare many runs —
	// a regression to unsorted output would be caught with overwhelming
	// probability (6 orderings for 3 imports, 12 iterations).
	var first []string
	for i := 0; i < 12; i++ {
		imports, _, _, err := extractMetadata(tempDir, moduleBaseName, targetFiles, nil, 0, false)
		if err != nil {
			t.Fatalf("extractMetadata run %d failed: %v", i, err)
		}
		if len(imports) != 3 {
			t.Fatalf("Expected 3 dynamic imports, got %v", imports)
		}
		if first == nil {
			first = imports
			sorted := append([]string(nil), imports...)
			sort.Strings(sorted)
			for j := range sorted {
				if first[j] != sorted[j] {
					t.Fatalf("Imports not sorted: %v", first)
				}
			}
			continue
		}
		for j := range first {
			if first[j] != imports[j] {
				t.Fatalf("Import order differs between run 0 (%v) and run %d (%v)", first, i, imports)
			}
		}
	}
}

func TestGenerateBridge_ByteStableAcrossRuns(t *testing.T) {
	tempDir := writeGoBridgeFixture(t)

	generate := func() []byte {
		t.Helper()
		_, _, err := GenerateBridge(tempDir, "", "", "DEBUG", false)
		if err != nil {
			t.Fatalf("GenerateBridge failed: %v", err)
		}
		content, err := os.ReadFile(filepath.Join(tempDir, "go_external_bridge_bridge.go"))
		if err != nil {
			t.Fatalf("read bridge failed: %v", err)
		}
		return content
	}

	first := generate()
	second := generate()
	if !bytes.Equal(first, second) {
		t.Fatal("Bridge file differs between consecutive generations with unchanged inputs")
	}

	// The import block must be a single sorted group — stable across runs and
	// exactly what gofmt would leave behind.
	content := string(first)
	start := strings.Index(content, "import (\n")
	end := strings.Index(content, ")")
	if start < 0 || end < start {
		t.Fatalf("Could not locate import block in bridge:\n%s", content)
	}
	var importPaths []string
	for _, line := range strings.Split(content[start+len("import (\n"):end], "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		importPaths = append(importPaths, strings.Trim(line, `"`))
	}
	sorted := append([]string(nil), importPaths...)
	sort.Strings(sorted)
	for i := range sorted {
		if importPaths[i] != sorted[i] {
			t.Fatalf("Import block not sorted: %v", importPaths)
		}
	}

	// Fixed runtime imports plus every dynamic package import must be present;
	// BetaFlag's *bool parameter pulls in strconv.
	for _, expected := range []string{"fmt", "os", "strconv", "alpha", "beta", "gamma"} {
		found := false
		for _, imp := range importPaths {
			if strings.HasSuffix(imp, "/"+expected) || imp == expected {
				found = true
			}
		}
		if !found {
			t.Errorf("Expected import ending in %q, got %v", expected, importPaths)
		}
	}
}

func TestGenerateBridge_GofmtCanonical(t *testing.T) {
	tempDir := writeGoBridgeFixture(t)

	_, absBridge, err := GenerateBridge(tempDir, "", "", "DEBUG", false)
	if err != nil {
		t.Fatalf("GenerateBridge failed: %v", err)
	}
	content, err := os.ReadFile(absBridge)
	if err != nil {
		t.Fatalf("read bridge failed: %v", err)
	}

	// go/format.Source is the engine behind the gofmt command — if it makes
	// no changes, `gofmt -l` will not flag the generated bridge.
	formatted, err := format.Source(content)
	if err != nil {
		t.Fatalf("generated bridge does not parse: %v", err)
	}
	if !bytes.Equal(content, formatted) {
		a := strings.Split(string(content), "\n")
		b := strings.Split(string(formatted), "\n")
		for i := 0; i < len(a) || i < len(b); i++ {
			var la, lb string
			if i < len(a) {
				la = a[i]
			}
			if i < len(b) {
				lb = b[i]
			}
			if la != lb {
				t.Fatalf("Generated bridge is not gofmt-canonical (first diff at line %d):\n got: %q\nwant: %q", i+1, la, lb)
			}
		}
		t.Fatal("Generated bridge is not gofmt-canonical")
	}

	// No double blank lines anywhere — template range/if actions used to
	// leave them inside generated function bodies.
	if strings.Contains(string(content), "\n\n\n") {
		t.Error("Generated bridge contains double blank lines")
	}
}

func TestGenerateBridge_GeneratesAndroidCompanion(t *testing.T) {
	tempDir := writeGoBridgeFixture(t)

	targetDir, absBridge, err := GenerateBridge(tempDir, "", "", "DEBUG", false)
	if err != nil {
		t.Fatalf("GenerateBridge failed: %v", err)
	}

	// Verify main bridge has //go:build !android
	mainContent, err := os.ReadFile(absBridge)
	if err != nil {
		t.Fatalf("read main bridge failed: %v", err)
	}
	if !strings.Contains(string(mainContent), "//go:build !android") {
		t.Errorf("Expected main bridge to have //go:build !android tag")
	}
	if !strings.Contains(string(mainContent), `import "C"`) {
		t.Errorf("Expected main bridge to have import \"C\"")
	}

	// Verify companion android bridge exists
	androidBridge := strings.TrimSuffix(absBridge, ".go") + "_android.go"
	androidContent, err := os.ReadFile(androidBridge)
	if err != nil {
		t.Fatalf("read android companion bridge failed: %v", err)
	}
	if !strings.Contains(string(androidContent), "//go:build android") {
		t.Errorf("Expected android bridge to have //go:build android tag")
	}
	if strings.Contains(string(androidContent), `import "C"`) {
		t.Errorf("Android companion bridge must not have import \"C\"")
	}
	if !strings.Contains(string(androidContent), "func main()") {
		t.Errorf("Android companion bridge must have func main()")
	}

	_ = targetDir
}

func TestCompileLocal_BuildsAndroidAArch64(t *testing.T) {
	tempDir := t.TempDir()
	mainGo := filepath.Join(tempDir, "script.go")
	src := `package main

import "fmt"

func EchoMsg(msg string) {
	fmt.Println(msg)
}
`
	if err := os.WriteFile(mainGo, []byte(src), 0644); err != nil {
		t.Fatalf("failed writing script.go: %v", err)
	}

	targetDir, absBridge, err := GenerateBridge(tempDir, "", "", "DEBUG", false)
	if err != nil {
		t.Fatalf("GenerateBridge failed: %v", err)
	}

	binaryBase := filepath.Join(tempDir, "myscript")
	err = CompileLocal(targetDir, absBridge, binaryBase, "DEBUG")
	if err != nil {
		t.Fatalf("CompileLocal failed: %v", err)
	}

	// Verify the android_aarch64 executable was compiled
	androidBin := binaryBase + "_android_aarch64"
	fi, err := os.Stat(androidBin)
	if err != nil {
		t.Fatalf("Expected android_aarch64 binary to exist at %s: %v", androidBin, err)
	}
	if fi.Mode()&0111 == 0 {
		t.Errorf("Expected %s to have executable permissions, mode is %v", androidBin, fi.Mode())
	}
}

