package internal_utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveBinaryOutputPath(t *testing.T) {
	tempDir := t.TempDir()

	// 1. Bare filename anchors to the anchor directory.
	resolved, err := ResolveBinaryOutputPath("my_binary", tempDir)
	if err != nil {
		t.Fatalf("bare filename failed: %v", err)
	}
	if resolved != filepath.Join(tempDir, "my_binary") {
		t.Errorf("Expected %s, got %s", filepath.Join(tempDir, "my_binary"), resolved)
	}

	// 2. "./"-prefixed bare filename also anchors.
	resolved, err = ResolveBinaryOutputPath("./my_binary", tempDir)
	if err != nil {
		t.Fatalf("./-prefixed bare filename failed: %v", err)
	}
	if resolved != filepath.Join(tempDir, "my_binary") {
		t.Errorf("Expected %s, got %s", filepath.Join(tempDir, "my_binary"), resolved)
	}

	// 3. Absolute paths are preserved, even outside the anchor directory —
	// the binary is standalone and may land anywhere.
	absOutside := filepath.Join(t.TempDir(), "elsewhere", "bin")
	resolved, err = ResolveBinaryOutputPath(absOutside, tempDir)
	if err != nil {
		t.Fatalf("absolute outside path failed: %v", err)
	}
	if resolved != absOutside {
		t.Errorf("Expected %s, got %s", absOutside, resolved)
	}

	// 4. Empty path is rejected.
	if _, err := ResolveBinaryOutputPath("", tempDir); err == nil {
		t.Error("Expected error for empty path, got nil")
	}

	// 5. Relative paths resolve against the CWD, not the anchor directory.
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	// Run from a subdirectory so CWD and anchorDir differ; joining the relative
	// path onto anchorDir would land it one level too high.
	subDir := filepath.Join(tempDir, "sub")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.Chdir(subDir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	resolved, err = ResolveBinaryOutputPath("../my_binary", tempDir)
	if err != nil {
		t.Fatalf("CWD-relative path failed: %v", err)
	}
	if resolved != filepath.Join(tempDir, "my_binary") {
		t.Errorf("Expected %s (CWD-resolved), got %s", filepath.Join(tempDir, "my_binary"), resolved)
	}
}
