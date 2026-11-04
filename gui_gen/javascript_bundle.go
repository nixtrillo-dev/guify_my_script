package gui_gen

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// These patterns intentionally cover static imports, re-exports, and literal
// CommonJS requires. Runtime-computed module paths cannot be discovered safely.
var jsLocalModuleRefs = regexp.MustCompile(`(?:\b(?:import|export)\s+(?:[^'";]*?\s+from\s+)?|\bimport\s*\(\s*|\brequire\s*\(\s*)['"]([^'"]+)['"]`)

// copyJavaScriptBundle copies an entrypoint and its statically referenced local
// modules to destDir. The entrypoint is written using entryName; dependencies
// retain paths relative to the entrypoint's directory so relative module paths
// continue to resolve. It returns whether local dependencies were found.
func copyJavaScriptBundle(entrypoint, destDir, entryName string) (bool, error) {
	files, root, err := discoverJavaScriptBundle(entrypoint)
	if err != nil {
		return false, err
	}
	entrypoint, err = filepath.Abs(entrypoint)
	if err != nil {
		return false, err
	}
	entryRel, err := filepath.Rel(root, entrypoint)
	if err != nil {
		return false, err
	}
	// For dependency bundles the path hierarchy is part of module resolution.
	// Keep the entrypoint in that hierarchy; for standalone scripts retain the
	// configured alias for backwards compatibility.
	if len(files) > 1 {
		entryName = entryRel
	}
	for _, file := range files {
		rel, err := filepath.Rel(root, file)
		if err != nil {
			return false, err
		}
		dstRel := rel
		if len(files) == 1 {
			dstRel = entryName
		}
		info, err := os.Stat(file)
		if err != nil {
			return false, err
		}
		if err := copyFile(file, filepath.Join(destDir, dstRel), info.Mode().Perm()); err != nil {
			return false, fmt.Errorf("copy JavaScript module %q: %w", file, err)
		}
	}
	return len(files) > 1, nil
}

// discoverJavaScriptBundle returns the transitive static local-module graph
// and its common directory. The directory is the shallowest common ancestor
// of all files, which lets ../ imports retain their original relative paths.
func discoverJavaScriptBundle(entrypoint string) ([]string, string, error) {
	entrypoint, err := filepath.Abs(entrypoint)
	if err != nil {
		return nil, "", err
	}
	visited := map[string]bool{}
	files := []string{}
	var visit func(string) error
	visit = func(file string) error {
		file, err = filepath.Abs(file)
		if err != nil {
			return err
		}
		if visited[file] {
			return nil
		}
		visited[file] = true
		data, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read JavaScript module %q: %w", file, err)
		}
		files = append(files, file)
		for _, match := range jsLocalModuleRefs.FindAllStringSubmatch(string(data), -1) {
			ref := match[1]
			if !strings.HasPrefix(ref, "./") && !strings.HasPrefix(ref, "../") {
				continue
			}
			resolved, err := resolveJavaScriptLocalModule(filepath.Dir(file), ref)
			if err != nil {
				return fmt.Errorf("resolve local import %q from %q: %w", ref, file, err)
			}
			if err := visit(resolved); err != nil {
				return err
			}
		}
		return nil
	}
	if err := visit(entrypoint); err != nil {
		return nil, "", err
	}
	root := filepath.Dir(files[0])
	for _, file := range files[1:] {
		root = commonDirectory(root, filepath.Dir(file))
	}
	return files, root, nil
}

func commonDirectory(a, b string) string {
	a, b = filepath.Clean(a), filepath.Clean(b)
	for {
		rel, err := filepath.Rel(a, b)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return a
		}
		parent := filepath.Dir(a)
		if parent == a {
			return a
		}
		a = parent
	}
}

func resolveJavaScriptLocalModule(dir, ref string) (string, error) {
	base := filepath.Join(dir, filepath.FromSlash(ref))
	candidates := []string{base}
	if filepath.Ext(base) == "" {
		candidates = append(candidates, base+".js", base+".mjs", base+".cjs", filepath.Join(base, "index.js"), filepath.Join(base, "index.mjs"), filepath.Join(base, "index.cjs"))
	}
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("module file not found (checked %s)", strings.Join(candidates, ", "))
}
