package gui_gen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteProductionPackageFiles_PythonAddsRuntimeDependency(t *testing.T) {
	stageDir := t.TempDir()
	if err := writeProductionPackageFiles(stageDir, "example", "Example", "example", DeployConfig{}, "amd64", true, false); err != nil {
		t.Fatalf("writeProductionPackageFiles failed: %v", err)
	}

	checks := map[string]string{
		filepath.Join(stageDir, "arch", "dependencies.arch.txt"):     "python\n",
		filepath.Join(stageDir, "debian", "dependencies.debian.txt"): "python3\n",
		filepath.Join(stageDir, "fedora", "dependencies.fedora.txt"): "python3\n",
	}
	for path, expected := range checks {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("Failed reading %s: %v", path, err)
		}
		if !strings.Contains(string(data), expected) {
			t.Errorf("Expected %s to contain %q; got:\n%s", path, expected, data)
		}
	}
}

func TestWriteProductionPackageFiles_JavaScriptAddsNodeRuntimeDependency(t *testing.T) {
	stageDir := t.TempDir()
	if err := writeProductionPackageFiles(stageDir, "example", "Example", "example", DeployConfig{}, "amd64", false, true); err != nil {
		t.Fatalf("writeProductionPackageFiles failed: %v", err)
	}

	checks := map[string]string{
		filepath.Join(stageDir, "arch", "dependencies.arch.txt"):     "nodejs\n",
		filepath.Join(stageDir, "debian", "dependencies.debian.txt"): "nodejs\n",
		filepath.Join(stageDir, "fedora", "dependencies.fedora.txt"): "nodejs\n",
	}
	for path, expected := range checks {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("Failed reading %s: %v", path, err)
		}
		if !strings.Contains(string(data), expected) {
			t.Errorf("Expected %s to contain %q; got:\n%s", path, expected, data)
		}
	}

	absentChecks := []string{"python", "python3"}
	for path := range checks {
		data, _ := os.ReadFile(path)
		for _, absent := range absentChecks {
			if strings.Contains(string(data), absent+"\n") {
				t.Errorf("Expected %s NOT to contain %q for pure JavaScript app", path, absent)
			}
		}
	}
}
