package gui_gen

import (
	"os"
	"os/exec"
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

func TestWriteProductionPackageFiles_PowerShellAddsRuntimeDependency(t *testing.T) {
	stageDir := t.TempDir()
	if err := writeProductionPackageFiles(stageDir, "example", "Example", "example", DeployConfig{}, "amd64", false, false, true); err != nil {
		t.Fatalf("writeProductionPackageFiles failed: %v", err)
	}
	checks := map[string]string{
		filepath.Join(stageDir, "arch", "dependencies.arch.txt"):     "powershell-bin\n",
		filepath.Join(stageDir, "debian", "dependencies.debian.txt"): "powershell\n",
		filepath.Join(stageDir, "fedora", "dependencies.fedora.txt"): "powershell\n",
	}
	for path, expected := range checks {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(data), expected) {
			t.Errorf("expected %s to contain %q; got:\n%s", path, expected, data)
		}
	}
}

func TestWriteProductionPackageFiles_ArchBuildSeparatesOfficialAndAURDependencies(t *testing.T) {
	stageDir := t.TempDir()
	if err := writeProductionPackageFiles(stageDir, "example", "Example", "example", DeployConfig{}, "amd64", false, false, true); err != nil {
		t.Fatalf("writeProductionPackageFiles failed: %v", err)
	}

	buildScript, err := os.ReadFile(filepath.Join(stageDir, "arch", "build.sh"))
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{
		"set -e",
		"pacman -S --needed --noconfirm git",
		"pacman -Si \"$pkg\"",
		"pacman -S --needed --noconfirm \"${official_deps[@]}\"",
		"dependencies.arch.aur.txt",
		"yay-bin.git",
		"xargs yay -S --needed --noconfirm",
	} {
		if !strings.Contains(string(buildScript), expected) {
			t.Errorf("Arch build script is missing %q", expected)
		}
	}

	checkSyntax := exec.Command("bash", "-n")
	checkSyntax.Stdin = strings.NewReader(string(buildScript))
	if output, err := checkSyntax.CombinedOutput(); err != nil {
		t.Fatalf("generated Arch build script has invalid shell syntax: %v\n%s", err, output)
	}
}
