package gui_gen

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteProductionPackageFiles_PythonAddsRuntimeDependency(t *testing.T) {
	stageDir := t.TempDir()
	if err := writeProductionPackageFiles(stageDir, "example", "Example", "example", DeployConfig{}, "amd64", true); err != nil {
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
