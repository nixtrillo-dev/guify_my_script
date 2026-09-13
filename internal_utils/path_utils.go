package internal_utils

import (
	"fmt"
	"path/filepath"
)

// ResolveBinaryOutputPath resolves a binary output path argument to an absolute
// path. Bare filenames (no directory component) are anchored to anchorDir;
// relative paths carrying directory components resolve against the process CWD;
// absolute paths are returned cleaned.
//
// The binary is a standalone deliverable — unlike a bridge file, its location
// carries no build-time significance — so no containment validation is applied
// and the result may fall anywhere on disk.
func ResolveBinaryOutputPath(name, anchorDir string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("binary output path must not be empty")
	}

	if filepath.Dir(filepath.Clean(name)) == "." {
		// Bare filename: anchor it to the given directory.
		return filepath.Clean(filepath.Join(anchorDir, name)), nil
	}

	// Resolve relative paths from the process CWD, not from anchorDir.
	// Joining a caller-relative path (e.g. "../repo/project/binary") onto
	// anchorDir produces a doubled path when the caller expressed the
	// destination relative to their own working directory.
	abs, err := filepath.Abs(name)
	if err != nil {
		return "", fmt.Errorf("failed to resolve binary output path %q: %w", name, err)
	}
	return abs, nil
}
