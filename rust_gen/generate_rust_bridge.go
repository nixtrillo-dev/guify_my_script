package rust_gen

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"guify_my_script/internal_utils"
	"guify_my_script/logger"
)

// ArgMetaData tracks individual parameter identity alongside its parsed Rust type layout
type ArgMetaData struct {
	Name     string
	Type     string // "string", "bool", "int", "float", "*bool"
	RawType  string // original Rust type layout (e.g. "&str", "String", "i32", "u8")
	CastType string
}

// FuncMetaData tracks discovered Rust function metadata
type FuncMetaData struct {
	RustName       string
	GoName         string // PascalCase e.g. PrintGreeting
	BashName       string // snake_case e.g. print_greeting
	PackageName    string // crate name e.g. core_utils
	HasErrorReturn bool
	ReturnCount    int
	Args           []ArgMetaData
	CallPrefix     string // e.g. "core_utils::"
}

// TemplateWrapper packages data for rendering the Rust bridge
type TemplateWrapper struct {
	CrateName    string
	Mods         []string // inline module declarations needed when sources are siblings (sameDir layout)
	Imports      []string
	NeedsStrconv bool
	Functions    []FuncMetaData
}

// Complete cross-boundary interop boilerplate definitions for Rust
const bridgeTemplate = `// Auto-generated and maintained by guify_my_script. DO NOT EDIT.
#![allow(dead_code)]
#![allow(unused_imports)]
#![allow(unused_variables)]

{{range .Mods}}mod {{.}};
{{end}}{{range .Imports}}use {{.}}::*;
{{end}}

// =====================================================================
// AUTOMATICALLY GENERATED C++ EXPORTS
// =====================================================================
{{range .Functions}}
#[no_mangle]
pub extern "C" fn Rust{{.GoName}}({{range $i, $arg := .Args}}{{if $i}}, {{end}}{{$arg.Name}}: {{if eq $arg.Type "string"}}*const std::os::raw::c_char{{else if eq $arg.Type "bool"}}std::os::raw::c_int{{else if eq $arg.Type "*bool"}}*const std::os::raw::c_int{{else}}{{$arg.RawType}}{{end}}{{end}}) -> std::os::raw::c_int {
{{range .Args}}
{{if eq .Type "string"}}    let rust_{{.Name}} = if {{.Name}}.is_null() { "" } else { unsafe { std::ffi::CStr::from_ptr({{.Name}}) }.to_str().unwrap_or("") };
{{else if eq .Type "bool"}}    let rust_{{.Name}} = {{.Name}} != 0;
{{else if eq .Type "int"}}    let rust_{{.Name}} = {{.Name}};
{{else if eq .Type "float"}}    let rust_{{.Name}} = {{.Name}};
{{else if eq .Type "*bool"}}    let rust_{{.Name}} = if {{.Name}}.is_null() { None } else { Some(unsafe { *{{.Name}} } != 0) };
{{end}}{{end}}
{{if .HasErrorReturn}}
    let result = {{.CallPrefix}}{{.RustName}}({{range $i, $arg := .Args}}{{if $i}}, {{end}}{{if eq $arg.RawType "String"}}rust_{{$arg.Name}}.to_string(){{else if eq $arg.RawType "&String"}}&rust_{{$arg.Name}}.to_string(){{else}}rust_{{$arg.Name}}{{end}}{{end}});
    match result {
        Ok(_) => 1,
        Err(err) => {
            eprintln!("Pipeline Error in {{.RustName}}: {err}");
            0
        }
    }
{{else}}
    {{.CallPrefix}}{{.RustName}}({{range $i, $arg := .Args}}{{if $i}}, {{end}}{{if eq $arg.RawType "String"}}rust_{{$arg.Name}}.to_string(){{else if eq $arg.RawType "&String"}}&rust_{{$arg.Name}}.to_string(){{else}}rust_{{$arg.Name}}{{end}}{{end}});
    1
{{end}}
}

#[no_mangle]
pub extern "C" fn rust_{{.BashName}}({{range $i, $arg := .Args}}{{if $i}}, {{end}}{{$arg.Name}}: {{if eq $arg.Type "string"}}*const std::os::raw::c_char{{else if eq $arg.Type "bool"}}std::os::raw::c_int{{else if eq $arg.Type "*bool"}}*const std::os::raw::c_int{{else}}{{$arg.RawType}}{{end}}{{end}}) -> std::os::raw::c_int {
    Rust{{.GoName}}({{range $i, $arg := .Args}}{{if $i}}, {{end}}{{$arg.Name}}{{end}})
}
{{end}}

// =====================================================================
// AUTOMATICALLY GENERATED BASH CLI ROUTER
// =====================================================================
fn main() {
    let args: Vec<String> = std::env::args().collect();
    if args.len() < 2 {
        eprintln!("Error: Missing command function name");
        std::process::exit(1);
    }

    match args[1].as_str() {
{{range .Functions}}    "{{.BashName}}" => {
{{if .Args}}        if args.len() < {{add (len .Args) 2}} {
            eprintln!("Error: {{.BashName}} requires parameters");
            std::process::exit(1);
        }

        // Parse CLI parameters safely
{{range $idx, $arg := .Args}}
{{if eq $arg.Type "string"}}
        let cli_{{$arg.Name}} = &args[{{add $idx 2}}];
{{else if eq $arg.Type "bool"}}
        let cli_{{$arg.Name}}: bool = match args[{{add $idx 2}}].to_lowercase().as_str() {
            "true" | "1" => true,
            "false" | "0" => false,
            _ => {
                eprintln!("Error: parameter '{{$arg.Name}}' must be a valid boolean evaluation");
                std::process::exit(1);
            }
        };
{{else if eq $arg.Type "*bool"}}
        let raw_{{$arg.Name}} = &args[{{add $idx 2}}];
        let cli_{{$arg.Name}}: Option<bool> = if raw_{{$arg.Name}}.is_empty() || raw_{{$arg.Name}} == "nil" || raw_{{$arg.Name}} == "null" || raw_{{$arg.Name}} == "none" {
            None
        } else {
            match raw_{{$arg.Name}}.to_lowercase().as_str() {
                "true" | "1" => Some(true),
                "false" | "0" => Some(false),
                _ => {
                    eprintln!("Error: parameter '{{$arg.Name}}' must be a valid boolean or nil");
                    std::process::exit(1);
                }
            }
        };
{{else if eq $arg.Type "int"}}
        let raw_{{$arg.Name}} = &args[{{add $idx 2}}];
        let cli_{{$arg.Name}}: {{$arg.RawType}} = match raw_{{$arg.Name}}.parse::<{{$arg.RawType}}>() {
            Ok(val) => val,
            Err(err) => {
                eprintln!("Error: parameter '{{$arg.Name}}' must be a valid integer ({{$arg.RawType}}): {err}");
                std::process::exit(1);
            }
        };
{{else if eq $arg.Type "float"}}
        let raw_{{$arg.Name}} = &args[{{add $idx 2}}];
        let cli_{{$arg.Name}}: {{$arg.RawType}} = match raw_{{$arg.Name}}.parse::<{{$arg.RawType}}>() {
            Ok(val) => val,
            Err(err) => {
                eprintln!("Error: parameter '{{$arg.Name}}' must be a valid float ({{$arg.RawType}}): {err}");
                std::process::exit(1);
            }
        };
{{end}}{{end}}
{{if .HasErrorReturn}}
        if let Err(err) = {{.CallPrefix}}{{.RustName}}({{range $i, $arg := .Args}}{{if $i}}, {{end}}{{if eq $arg.RawType "String"}}cli_{{$arg.Name}}.to_string(){{else if eq $arg.RawType "&String"}}&cli_{{$arg.Name}}.to_string(){{else}}cli_{{$arg.Name}}{{end}}{{end}}) {
            eprintln!("Execution Failure: {err}");
            std::process::exit(1);
        }
{{else}}
        {{.CallPrefix}}{{.RustName}}({{range $i, $arg := .Args}}{{if $i}}, {{end}}{{if eq $arg.RawType "String"}}cli_{{$arg.Name}}.to_string(){{else if eq $arg.RawType "&String"}}&cli_{{$arg.Name}}.to_string(){{else}}cli_{{$arg.Name}}{{end}}{{end}});
{{end}}
{{else}}
{{if .HasErrorReturn}}
        if let Err(err) = {{.CallPrefix}}{{.RustName}}() {
            eprintln!("Execution Failure: {err}");
            std::process::exit(1);
        }
{{else}}
        {{.CallPrefix}}{{.RustName}}();
{{end}}
{{end}}
    }
{{end}}    unknown => {
        eprintln!("Error: Unknown function command '{unknown}'");
        std::process::exit(1);
    }
    }
}
`

// captureFileSnapshot captures the current state of all files under rootDir that match
// the auto-generated file patterns (Cargo.toml and src/lib.rs).
func captureFileSnapshot(rootDir string) map[string][]byte {
	snapshot := make(map[string][]byte)

	_ = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		// Only capture files that discoverLibCrates might modify
		if info.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(rootDir, path)
		fileName := filepath.Base(path)

		// Capture Cargo.toml files and src/lib.rs files
		if fileName == "Cargo.toml" || (filepath.Base(filepath.Dir(path)) == "src" && fileName == "lib.rs") {
			if content, readErr := os.ReadFile(path); readErr == nil {
				snapshot[relPath] = content
			}
		}

		return nil
	})

	return snapshot
}

// rollbackFileChanges restores files to their original state from the snapshot.
// It removes any files that were created and restores modified files.
func rollbackFileChanges(rootDir string, snapshot map[string][]byte) error {
	for relPath, originalContent := range snapshot {
		fullPath := filepath.Join(rootDir, relPath)

		// Check if file still exists
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			// File was deleted, but we only capture existing files, so this shouldn't happen
			// with our current snapshot logic. However, restore it anyway.
			if writeErr := os.WriteFile(fullPath, originalContent, 0644); writeErr != nil {
				return fmt.Errorf("failed to restore deleted file %s: %w", fullPath, writeErr)
			}
		} else {
			// Check if file was modified
			currentContent, readErr := os.ReadFile(fullPath)
			if readErr != nil {
				return fmt.Errorf("failed to read file %s for comparison: %w", fullPath, readErr)
			}

			if !bytes.Equal(currentContent, originalContent) {
				// File was modified, restore original
				if writeErr := os.WriteFile(fullPath, originalContent, 0644); writeErr != nil {
					return fmt.Errorf("failed to restore modified file %s: %w", fullPath, writeErr)
				}
			}
		}
	}

	// Now check for newly created files that weren't in the snapshot
	_ = filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(rootDir, path)
		fileName := filepath.Base(path)

		// Only consider files that discoverLibCrates creates
		isAutoGenFile := fileName == "Cargo.toml" || (filepath.Base(filepath.Dir(path)) == "src" && fileName == "lib.rs")

		if isAutoGenFile {
			// Check if this file was in the original snapshot
			if _, exists := snapshot[relPath]; !exists {
				// This is a newly created file, remove it
				if removeErr := os.Remove(path); removeErr != nil && !os.IsNotExist(removeErr) {
					return fmt.Errorf("failed to remove newly created file %s: %w", path, removeErr)
				}
			}
		}

		return nil
	})

	return nil
}

// GetRustNonMainPackagesJSON scans the Rust project directory and returns a JSON array of discovered crates/modules.
func GetRustNonMainPackagesJSON(inputPath string, logLevel string, noUpdateMode bool) (string, error) {
	lvl := logger.ParseLogLevel(logLevel)

	if inputPath == "" {
		emptyJSON := "[]"
		logger.Info(lvl, "%s", emptyJSON)
		return emptyJSON, nil
	}

	absInput, err := filepath.Abs(inputPath)
	if err != nil {
		return "", fmt.Errorf("failed to canonicalize input path: %w", err)
	}

	targetDir, err := resolveRustProjectDir(absInput)
	if err != nil {
		return "", err
	}

	// If in no-update mode, capture file state before discoverLibCrates and rollback after
	var fileSnapshot map[string][]byte
	if noUpdateMode {
		fileSnapshot = captureFileSnapshot(targetDir)
	}

	var modules []string
	seenMods := make(map[string]bool)

	addMod := func(mod string) {
		if !seenMods[mod] && mod != "" {
			seenMods[mod] = true
			modules = append(modules, mod)
		}
	}

	// Discover lib crates under targetDir
	libCrateDirs := discoverLibCrates(targetDir, "", lvl)
	if len(libCrateDirs) == 0 {
		libCrateDirs = []string{targetDir}
	}

	for _, crateDir := range libCrateDirs {
		crateName := getCrateName(crateDir)
		if crateName == "" {
			crateName = filepath.Base(crateDir)
			crateName = strings.ReplaceAll(crateName, " ", "_")
			crateName = strings.ReplaceAll(crateName, "-", "_")
		}
		addMod(crateName)

		// Also find submodules in src/
		srcDir := filepath.Join(crateDir, "src")
		if entries, err := os.ReadDir(srcDir); err == nil {
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), ".rs") {
					base := strings.TrimSuffix(e.Name(), ".rs")
					if base != "lib" && base != "main" {
						addMod(fmt.Sprintf("%s::%s", crateName, base))
					}
				}
			}
		}
	}

	jsonData, err := json.Marshal(modules)
	if err != nil {
		return "", fmt.Errorf("failed to encode crates to JSON: %w", err)
	}

	// Rollback file system changes if in no-update mode
	if noUpdateMode && fileSnapshot != nil {
		if rollbackErr := rollbackFileChanges(targetDir, fileSnapshot); rollbackErr != nil {
			logger.Info(lvl, "Warning: failed to rollback file changes: %v", rollbackErr)
		}
	}

	jsonString := string(jsonData)
	logger.Data(lvl, "%s", jsonString)
	return jsonString, nil
}

// GenerateRustBridge accepts inputPath, outputPath, excludePackages, logLevelStr, and includeFuncParams.
func GenerateRustBridge(inputPath, outputPath, excludePackages string, logLevelStr string, includeFuncParams bool) (string, string, error) {
	lvl := logger.ParseLogLevel(logLevelStr)
	logger.Info(lvl, "Generating the Rust bridge file...")

	excludeMap := make(map[string]bool)
	if excludePackages != "" {
		r := csv.NewReader(strings.NewReader(excludePackages))
		records, err := r.Read()
		if err == nil {
			for _, pkg := range records {
				excludeMap[strings.TrimSpace(pkg)] = true
			}
		} else {
			logger.Warn(lvl, "Failed to parse excludePackages as CSV: %v", err)
		}
	}

	if inputPath == "" {
		inputPath = "."
	}

	absInput, err := filepath.Abs(inputPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to canonicalize input path: %w", err)
	}

	targetDir, err := resolveRustProjectDir(absInput)
	if err != nil {
		return "", "", err
	}

	// Resolve expected root src directory
	expectedSrcDir := filepath.Join(targetDir, "src")

	// Resolve output path (default is <targetDir>/src/main.rs)
	if outputPath == "" {
		outputPath = filepath.Join(expectedSrcDir, "main.rs")
	} else if !filepath.IsAbs(outputPath) {
		outputPath, _ = filepath.Abs(outputPath)
	}

	absOutput, err := filepath.Abs(outputPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to canonicalize output path: %w", err)
	}

	// Enforce: bridge file must have root src as its parent folder
	bridgeParent := filepath.Clean(filepath.Dir(absOutput))
	if bridgeParent != filepath.Clean(expectedSrcDir) {
		return "", "", fmt.Errorf("output bridge file %q must have the root src directory %q as its parent folder", absOutput, expectedSrcDir)
	}

	libCrateDir := ""
	if _, err := os.Stat(filepath.Join(targetDir, "src", "lib.rs")); err == nil {
		libCrateDir = targetDir
	}

	// Discover all lib crates under targetDir (generates missing Cargo.toml and lib.rs files).
	allLibCrateDirs := discoverLibCrates(targetDir, libCrateDir, lvl)

	crateName := getCrateName(targetDir)
	if crateName == "" {
		crateName = filepath.Base(targetDir)
		crateName = strings.ReplaceAll(crateName, " ", "_")
		crateName = strings.ReplaceAll(crateName, "-", "_")
	}

	// Ensure root src directory exists
	if err := os.MkdirAll(expectedSrcDir, 0755); err != nil {
		return "", "", fmt.Errorf("failed creating root src directory: %w", err)
	}

	// Gather Rust source files from every discovered lib crate's src/ directory.
	var targetFiles []string
	seenFiles := make(map[string]bool)

	cratesToScan := allLibCrateDirs
	if len(cratesToScan) == 0 {
		cratesToScan = []string{targetDir}
	}

	for _, crateDir := range cratesToScan {
		srcDir := filepath.Join(crateDir, "src")
		if fi, statErr := os.Stat(srcDir); statErr == nil && fi.IsDir() {
			_ = filepath.WalkDir(srcDir, func(path string, d os.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				absPath, _ := filepath.Abs(path)
				if !d.IsDir() && filepath.Ext(absPath) == ".rs" && filepath.Base(absPath) != "main.rs" && absPath != absOutput && !seenFiles[absPath] {
					seenFiles[absPath] = true
					targetFiles = append(targetFiles, absPath)
					logger.Info(lvl, "  • Found Rust source: %s", absPath)
				}
				return nil
			})
		} else if crateDir == targetDir && filepath.Ext(absInput) == ".rs" {
			// Single-file input that was not yet wrapped in a src/ dir.
			if !seenFiles[absInput] {
				seenFiles[absInput] = true
				targetFiles = append(targetFiles, absInput)
			}
		}
	}

	// Extract functions and metadata from all collected sources.
	// extractRustMetadata resolves module paths relative to the file's owning crate
	// directory. We process each crate separately and merge the results so the
	// module paths (e.g. "core_utils::payments") are always correct.
	var (
		allDynamicImports []string
		allFunctions      []FuncMetaData
		allNeedsStrconv   bool
	)
	importSet := make(map[string]bool)

	for _, crateDir := range allLibCrateDirs {
		cName := getCrateName(crateDir)
		if cName == "" {
			cName = filepath.Base(crateDir)
			cName = strings.ReplaceAll(cName, " ", "_")
			cName = strings.ReplaceAll(cName, "-", "_")
		}

		// Collect only the files belonging to this crate.
		var crateFiles []string
		cSrc := filepath.Join(crateDir, "src")
		for _, f := range targetFiles {
			if strings.HasPrefix(f, cSrc+string(filepath.Separator)) || f == cSrc {
				crateFiles = append(crateFiles, f)
			}
		}
		if len(crateFiles) == 0 {
			continue
		}

		imports, fns, ns, err := extractRustMetadata(crateDir, cName, crateFiles, excludeMap, lvl, includeFuncParams)
		if err != nil {
			return "", "", err
		}
		for _, imp := range imports {
			if !importSet[imp] {
				importSet[imp] = true
				allDynamicImports = append(allDynamicImports, imp)
			}
		}
		allFunctions = append(allFunctions, fns...)
		if ns {
			allNeedsStrconv = true
		}
	}

	dynamicImports := allDynamicImports
	functions := allFunctions
	needsStrconv := allNeedsStrconv

	// sameDir: the source files are siblings of the bridge inside the same crate
	// (single-file promoted layout — no separate lib sub-crate).
	// In this case Rust requires `mod <stem>;` declarations in the bridge file
	// rather than external-crate `use` statements.  We also need to strip the
	// double-qualified "crateName::stem" import entries down to just "stem" so
	// the generated `use` lines match the inline module path.
	//
	// sameDir only applies when there is exactly one crate and it equals targetDir.
	var mods []string
	sameDir := len(allLibCrateDirs) == 0 || (len(allLibCrateDirs) == 1 && filepath.Clean(allLibCrateDirs[0]) == filepath.Clean(targetDir))
	if sameDir {
		// Collect the stem of every source file that is not lib.rs / main.rs.
		modSet := make(map[string]bool)
		for _, f := range targetFiles {
			relF, _ := filepath.Rel(libCrateDir, f)
			if relF == "src/lib.rs" || relF == "src/main.rs" {
				continue
			}
			stem := strings.TrimSuffix(filepath.Base(f), ".rs")
			if stem != "" {
				modSet[stem] = true
			}
		}
		for stem := range modSet {
			mods = append(mods, stem)
		}

		// Rewrite imports: replace "crateName::stem" entries with just "stem"
		// so the use lines reference the inline module, not a phantom external crate.
		prefix := crateName + "::"
		var fixedImports []string
		seenImport := make(map[string]bool)
		for _, imp := range dynamicImports {
			fixed := imp
			if strings.HasPrefix(imp, prefix) {
				fixed = strings.TrimPrefix(imp, prefix)
			}
			if !seenImport[fixed] {
				seenImport[fixed] = true
				fixedImports = append(fixedImports, fixed)
			}
		}
		dynamicImports = fixedImports

		// Also fix the CallPrefix on every function: strip the "crateName::" prefix
		// so calls use "stem::" (the inline module path) rather than "crateName::stem::".
		for i := range functions {
			if strings.HasPrefix(functions[i].PackageName, prefix) {
				functions[i].PackageName = strings.TrimPrefix(functions[i].PackageName, prefix)
				functions[i].CallPrefix = functions[i].PackageName + "::"
			}
		}
	} else {
		// Disambiguate external crates with leading :: to prevent collisions with local glob imports
		for i := range functions {
			if !strings.HasPrefix(functions[i].CallPrefix, "::") && functions[i].CallPrefix != "" {
				functions[i].CallPrefix = "::" + functions[i].CallPrefix
			}
		}
		var disambiguatedImports []string
		for _, imp := range dynamicImports {
			if !strings.HasPrefix(imp, "::") && imp != "" {
				disambiguatedImports = append(disambiguatedImports, "::"+imp)
			} else {
				disambiguatedImports = append(disambiguatedImports, imp)
			}
		}
		dynamicImports = disambiguatedImports
	}

	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}

	tmpl, err := template.New("rust_bridge").Funcs(funcMap).Parse(bridgeTemplate)
	if err != nil {
		return "", "", fmt.Errorf("template initialization error: %w", err)
	}

	outputFile, err := os.Create(absOutput)
	if err != nil {
		return "", "", fmt.Errorf("failed creating bridge file: %w", err)
	}
	defer outputFile.Close()

	wrapperData := TemplateWrapper{
		CrateName:    crateName,
		Mods:         mods,
		Imports:      dynamicImports,
		NeedsStrconv: needsStrconv,
		Functions:    functions,
	}

	if err := tmpl.Execute(outputFile, wrapperData); err != nil {
		outputFile.Close()
		return "", "", fmt.Errorf("template write error: %w", err)
	}
	outputFile.Close()

	// Now that src exists and the bridge file is created, ensure root Cargo.toml exists.
	if err := ensureRootCargo(targetDir, allLibCrateDirs, absOutput, lvl); err != nil {
		return "", "", fmt.Errorf("failed to ensure root Cargo.toml: %w", err)
	}

	logger.Info(lvl, "  • Successfully generated Rust bridge file %s\n", absOutput)
	return targetDir, absOutput, nil
}

// CompileRust compiles the runnable Rust binary and C++ shared library for the specified Rust project folder.
func CompileRust(targetDir, absBridge, binaryName string, logLevelStr string) error {
	lvl := logger.ParseLogLevel(logLevelStr)

	if targetDir == "" {
		targetDir = "."
	}
	absTargetDir, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("failed to canonicalize targetDir: %w", err)
	}

	resolvedDir, err := resolveRustProjectDir(absTargetDir)
	if err != nil {
		return err
	}
	absTargetDir = resolvedDir

	libCrateDir := ""
	if _, err := os.Stat(filepath.Join(absTargetDir, "src", "lib.rs")); err == nil {
		libCrateDir = absTargetDir
	}

	// Discover all lib crates (generates missing Cargo.toml files for new src dirs).
	allLibCrateDirs := discoverLibCrates(absTargetDir, libCrateDir, lvl)

	// Ensure the root project has a Cargo.toml that lists every lib crate as a
	// path dependency (generate one if absent).
	if absBridge != "" {
		if err := ensureRootCargo(absTargetDir, allLibCrateDirs, absBridge, lvl); err != nil {
			return fmt.Errorf("failed to scaffold root Cargo.toml: %w", err)
		}
	}

	buildDir := absTargetDir

	// Derive crate name from whichever directory we're actually building in.
	// For the root build dir the crate name is the binary package name (from its Cargo.toml);
	// for the library-only dir it's the lib crate name.
	crateName := getCrateName(buildDir)
	if crateName == "" {
		crateName = getCrateName(absTargetDir)
	}

	if binaryName == "" {
		if crateName != "" {
			binaryName = crateName
		} else {
			binaryName = filepath.Base(buildDir)
		}
	}

	dir, file := filepath.Split(binaryName)
	file = strings.ReplaceAll(file, " ", "_")
	binaryName = filepath.Join(dir, file)
	binaryName = strings.TrimSuffix(binaryName, ".so")
	binaryName = strings.TrimSuffix(binaryName, ".dll")

	// Resolve the binary destination. Bare filenames anchor to the build
	// directory; relative paths carrying directory components resolve against
	// the process CWD. The binary is a standalone deliverable — it may land
	// anywhere, so no project containment is enforced.
	binaryName, err = internal_utils.ResolveBinaryOutputPath(binaryName, buildDir)
	if err != nil {
		return err
	}

	logger.Info(lvl, "Compiling Rust package in %s (binary: %s)...", buildDir, binaryName)

	// Step 1: cargo build --release
	cmdBuild := exec.Command("cargo", "build", "--release")
	cmdBuild.Dir = buildDir

	var errBuild strings.Builder
	cmdBuild.Stdout = os.Stdout
	cmdBuild.Stderr = &errBuild

	if err := cmdBuild.Run(); err != nil {
		return fmt.Errorf("cargo build failure:\n%s\ninternal error: %w", errBuild.String(), err)
	}

	logger.Info(lvl, "Cargo build completed successfully.\n")
	logger.Info(lvl, "Building for architecture: linux_amd64...")

	// Locate release executable.
	targetReleaseDir := filepath.Join(buildDir, "target", "release")
	candidateBin := filepath.Join(targetReleaseDir, crateName)

	cliBinNameLinuxAMD64 := fmt.Sprintf("%s_linux_amd64", binaryName)
	cliBinNameDefault := binaryName

	// binaryName is always absolute here (resolved above).
	destLinuxAMD64 := cliBinNameLinuxAMD64
	destDefault := cliBinNameDefault

	if _, err := os.Stat(candidateBin); err == nil {
		if err := copyExecutable(candidateBin, destLinuxAMD64); err != nil {
			return fmt.Errorf("failed copying linux_amd64 binary: %w", err)
		}
		_ = copyExecutable(candidateBin, destDefault)
		logger.Info(lvl, "  • Built Linux AMD64 executable: %s", destLinuxAMD64)
	} else {
		// No executable was produced — this is expected for library-only crates
		// (lib.rs with no [[bin]] section).  Only log if a binary target was actually
		// declared in Cargo.toml so we don't surface noise for pure library builds.
		if hasBinaryTarget(buildDir) {
			logger.Error(lvl, "Cargo build did not produce expected binary %s: %e", candidateBin, err)
		} else {
			logger.Warn(lvl, "  • Skipping executable copy: %s is a library-only crate (no [[bin]] target)", crateName)
		}
	}

	// Step 2: Build C++ shared library (.so)
	// binaryName is absolute; append the suffix directly.
	libBinNameLinuxAMD64 := fmt.Sprintf("%s_linux_amd64.so", binaryName)
	destLibLinuxAMD64 := libBinNameLinuxAMD64

	// Check if cargo already built a cdylib in target/release/
	builtCdylib := filepath.Join(targetReleaseDir, fmt.Sprintf("lib%s.so", crateName))
	if _, err := os.Stat(builtCdylib); err == nil {
		if err := copyExecutable(builtCdylib, destLibLinuxAMD64); err != nil {
			return fmt.Errorf("failed copying shared library: %w", err)
		}
		logger.Info(lvl, "  • Built Linux AMD64 C++ shared library: %s", destLibLinuxAMD64)
	} else if absBridge != "" {
		// Compile absBridge with rustc as cdylib.
		depsDir := filepath.Join(targetReleaseDir, "deps")

		var rustcArgs []string
		rustcArgs = append(rustcArgs, "--edition", "2021", "--crate-type", "cdylib", absBridge, "-o", destLibLinuxAMD64)

		for _, cDir := range allLibCrateDirs {
			cName := getCrateName(cDir)
			if cName == "" {
				cName = filepath.Base(cDir)
			}
			cNameSafe := strings.ReplaceAll(cName, "-", "_")

			rlibPath := findRlibInDir(depsDir, cNameSafe)
			if rlibPath == "" {
				libDepsDir := filepath.Join(cDir, "target", "release", "deps")
				rlibPath = findRlibInDir(libDepsDir, cNameSafe)
				if rlibPath != "" {
					rustcArgs = append(rustcArgs, "-L", libDepsDir)
				}
			}
			if rlibPath != "" {
				rustcArgs = append(rustcArgs, "-L", depsDir, "--extern", fmt.Sprintf("%s=%s", cName, rlibPath))
			}
		}

		cmdLib := exec.Command("rustc", rustcArgs...)
		cmdLib.Dir = buildDir
		var errLib strings.Builder
		cmdLib.Stdout = os.Stdout
		cmdLib.Stderr = &errLib

		if err := cmdLib.Run(); err != nil {
			logger.Error(lvl, "rustc cdylib compile failed: %s", errLib.String())
		} else {
			_ = os.Chmod(destLibLinuxAMD64, 0755)
			logger.Info(lvl, "  • Built Linux AMD64 C++ shared library: %s", destLibLinuxAMD64)
		}
	}

	// Optional: Attempt ARM64 build if toolchain is installed
	cmdCheckTarget := exec.Command("rustup", "target", "list")
	if out, err := cmdCheckTarget.Output(); err == nil && strings.Contains(string(out), "aarch64-unknown-linux-gnu (installed)") {
		logger.Info(lvl, "\nBuilding for architecture: linux_aarch64...")
		cmdArm := exec.Command("cargo", "build", "--release", "--target", "aarch64-unknown-linux-gnu")
		cmdArm.Dir = buildDir
		cmdArm.Env = append(os.Environ(), "CARGO_TARGET_AARCH64_UNKNOWN_LINUX_GNU_LINKER=aarch64-linux-gnu-gcc")
		if err := cmdArm.Run(); err == nil {
			armReleaseDir := filepath.Join(buildDir, "target", "aarch64-unknown-linux-gnu", "release")

			// AArch64 executable
			armCandidate := filepath.Join(armReleaseDir, crateName)
			cliBinLinuxAArch64 := fmt.Sprintf("%s_linux_aarch64", binaryName)
			_ = copyExecutable(armCandidate, cliBinLinuxAArch64)
			logger.Info(lvl, "  • Built Linux AArch64 executable: %s", cliBinLinuxAArch64)

			// AArch64 .so — same glob-based rlib lookup as AMD64
			if absBridge != "" {
				armDepsDir := filepath.Join(armReleaseDir, "deps")
				destLibLinuxAArch64 := fmt.Sprintf("%s_linux_aarch64.so", binaryName)
				var rustcArgsArm []string
				rustcArgsArm = append(rustcArgsArm,
					"--edition", "2021",
					"--crate-type", "cdylib",
					"--target", "aarch64-unknown-linux-gnu",
					"-C", "linker=aarch64-linux-gnu-gcc",
					absBridge, "-o", destLibLinuxAArch64,
				)

				for _, cDir := range allLibCrateDirs {
					cName := getCrateName(cDir)
					if cName == "" {
						cName = filepath.Base(cDir)
					}
					cNameSafe := strings.ReplaceAll(cName, "-", "_")
					armRlibPath := findRlibInDir(armDepsDir, cNameSafe)
					if armRlibPath != "" {
						rustcArgsArm = append(rustcArgsArm, "-L", armDepsDir, "--extern", fmt.Sprintf("%s=%s", cName, armRlibPath))
					}
				}

				cmdLibArm := exec.Command("rustc", rustcArgsArm...)
				cmdLibArm.Dir = buildDir
				var errLibArm strings.Builder
				cmdLibArm.Stdout = os.Stdout
				cmdLibArm.Stderr = &errLibArm
				if err := cmdLibArm.Run(); err != nil {
					logger.Error(lvl, "rustc aarch64 cdylib compile failed: %s", errLibArm.String())
				} else {
					_ = os.Chmod(destLibLinuxAArch64, 0755)
					logger.Info(lvl, "  • Built Linux AArch64 C++ shared library: %s", destLibLinuxAArch64)
				}
			}
		}
	}

	// Step 3: Attempt Android AArch64 (Termux) build
	cmdCheckAndroidTarget := exec.Command("rustup", "target", "list")
	androidTargetInstalled := false
	if out, err := cmdCheckAndroidTarget.Output(); err == nil && strings.Contains(string(out), "aarch64-linux-android (installed)") {
		androidTargetInstalled = true
	}

	if androidTargetInstalled {
		linker := findAndroidAArch64Linker()
		hasCargoNdk := false
		if _, err := exec.LookPath("cargo-ndk"); err == nil {
			hasCargoNdk = true
		}

		if linker != "" || hasCargoNdk {
			logger.Info(lvl, "\nBuilding for architecture: android_aarch64...")
			var cmdAndroid *exec.Cmd
			if hasCargoNdk && linker == "" {
				cmdAndroid = exec.Command("cargo", "ndk", "-t", "aarch64-linux-android", "build", "--release")
			} else {
				cmdAndroid = exec.Command("cargo", "build", "--release", "--target", "aarch64-linux-android")
				cmdAndroid.Env = append(os.Environ(), "CARGO_TARGET_AARCH64_LINUX_ANDROID_LINKER="+linker)
			}
			cmdAndroid.Dir = buildDir
			var errAndroid strings.Builder
			cmdAndroid.Stdout = os.Stdout
			cmdAndroid.Stderr = &errAndroid

			if err := cmdAndroid.Run(); err == nil {
				androidReleaseDir := filepath.Join(buildDir, "target", "aarch64-linux-android", "release")
				androidCandidate := filepath.Join(androidReleaseDir, crateName)
				cliBinAndroidAArch64 := fmt.Sprintf("%s_android_aarch64", binaryName)
				if err := copyExecutable(androidCandidate, cliBinAndroidAArch64); err == nil {
					_ = os.Chmod(cliBinAndroidAArch64, 0755)
					logger.Info(lvl, "  • Built Android AArch64 executable: %s", cliBinAndroidAArch64)
				}
			} else {
				logger.Error(lvl, "cargo android_aarch64 build failed: %s", errAndroid.String())
			}
		} else {
			logger.Warn(lvl, "\nSkipping android_aarch64 build for Rust: Android NDK linker (aarch64-linux-android*-clang) not found. Set ANDROID_NDK_HOME, CARGO_TARGET_AARCH64_LINUX_ANDROID_LINKER, or run 'cargo install cargo-ndk' to compile Android Termux binaries.")
		}
	} else {
		logger.Warn(lvl, "\nSkipping android_aarch64 build for Rust: target aarch64-linux-android is not installed. Run 'rustup target add aarch64-linux-android' to enable.")
	}

	logger.Info(lvl, "Successfully compiled Rust deliverables in %s.", buildDir)
	return nil
}

// findAndroidAArch64Linker attempts to locate a suitable linker for aarch64-linux-android.
// It checks environment variables, PATH, and standard Android NDK installations.
func findAndroidAArch64Linker() string {
	// 1. Check explicit environment override
	if envLinker := os.Getenv("CARGO_TARGET_AARCH64_LINUX_ANDROID_LINKER"); envLinker != "" {
		if _, err := exec.LookPath(envLinker); err == nil {
			return envLinker
		}
		if _, err := os.Stat(envLinker); err == nil {
			return envLinker
		}
	}

	// 2. Check PATH for aarch64-linux-android*-clang binaries
	if p, err := exec.LookPath("aarch64-linux-android-clang"); err == nil {
		return p
	}

	pathEnv := os.Getenv("PATH")
	for _, dir := range filepath.SplitList(pathEnv) {
		matches, err := filepath.Glob(filepath.Join(dir, "aarch64-linux-android*-clang"))
		if err == nil && len(matches) > 0 {
			sort.Sort(sort.Reverse(sort.StringSlice(matches)))
			for _, m := range matches {
				if !strings.HasSuffix(m, "++") {
					return m
				}
			}
		}
	}

	// 3. Check NDK environment variables and standard locations
	var candidateNdkDirs []string
	if ndk := os.Getenv("ANDROID_NDK_HOME"); ndk != "" {
		candidateNdkDirs = append(candidateNdkDirs, ndk)
	}
	if ndk := os.Getenv("ANDROID_NDK_ROOT"); ndk != "" {
		candidateNdkDirs = append(candidateNdkDirs, ndk)
	}
	if ndk := os.Getenv("NDK_HOME"); ndk != "" {
		candidateNdkDirs = append(candidateNdkDirs, ndk)
	}
	if sdk := os.Getenv("ANDROID_HOME"); sdk != "" {
		ndkGlob := filepath.Join(sdk, "ndk", "*")
		if matches, err := filepath.Glob(ndkGlob); err == nil {
			candidateNdkDirs = append(candidateNdkDirs, matches...)
		}
		candidateNdkDirs = append(candidateNdkDirs, filepath.Join(sdk, "ndk-bundle"))
	}

	homeDir, _ := os.UserHomeDir()
	if homeDir != "" {
		candidateNdkDirs = append(candidateNdkDirs,
			filepath.Join(homeDir, "Android", "Sdk", "ndk-bundle"),
		)
		if matches, err := filepath.Glob(filepath.Join(homeDir, "Android", "Sdk", "ndk", "*")); err == nil {
			candidateNdkDirs = append(candidateNdkDirs, matches...)
		}
	}
	candidateNdkDirs = append(candidateNdkDirs,
		"/opt/android-ndk",
		"/usr/lib/android-ndk",
	)

	for _, ndkDir := range candidateNdkDirs {
		llvmBinGlob := filepath.Join(ndkDir, "toolchains", "llvm", "prebuilt", "*", "bin", "aarch64-linux-android*-clang")
		if matches, err := filepath.Glob(llvmBinGlob); err == nil && len(matches) > 0 {
			sort.Sort(sort.Reverse(sort.StringSlice(matches)))
			for _, m := range matches {
				if !strings.HasSuffix(m, "++") {
					return m
				}
			}
		}
	}

	return ""
}

// autoGenMarker is the substring used in all auto-generated files to identify them
const autoGenMarker = "Auto-generated and maintained by guify_my_script"

// fileHasAutoGenMarker checks if a file contains the auto-generation marker comment
func fileHasAutoGenMarker(filePath string) bool {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), autoGenMarker)
}

// shouldRegenerateFile returns true if the file does not exist OR if it exists and has the auto-gen marker
func shouldRegenerateFile(filePath string) bool {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return true // File doesn't exist, should create it
	}
	return fileHasAutoGenMarker(filePath) // File exists, regenerate only if it has marker
}

// resolveRustProjectDir resolves the directory to operate on as the root project directory.
// When path is a directory, it is strictly treated as the project root and will not
// descend into subdirectories. If path points into a src directory or file, its parent
// project root is resolved. If path cannot be made the root project directory, an error is returned.
func resolveRustProjectDir(path string) (string, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("input target path does not exist: %w", err)
	}

	var candidate string
	if fi.IsDir() {
		if filepath.Base(path) == "src" {
			candidate = filepath.Dir(path)
		} else {
			candidate = path
		}
	} else {
		dir := filepath.Dir(path)
		if filepath.Base(dir) == "src" {
			candidate = filepath.Dir(dir)
		} else {
			candidate = dir
		}
	}

	candidateFi, err := os.Stat(candidate)
	if err != nil {
		return "", fmt.Errorf("failed to access resolved root project directory %q: %w", candidate, err)
	}
	if !candidateFi.IsDir() {
		return "", fmt.Errorf("resolved root project path %q is not a directory", candidate)
	}

	return candidate, nil
}

// findRlibInDir globs dir for lib<name>-<hash>.rlib and returns the newest match.
// Cargo always appends a hash to rlib filenames so a fixed-name stat will never find them.
func findRlibInDir(dir, name string) string {
	matches, _ := filepath.Glob(filepath.Join(dir, fmt.Sprintf("lib%s-*.rlib", name)))
	var best string
	var bestTime int64
	for _, m := range matches {
		if fi, err := os.Stat(m); err == nil {
			if fi.ModTime().UnixNano() > bestTime {
				bestTime = fi.ModTime().UnixNano()
				best = m
			}
		}
	}
	return best
}

// ensureRootCargo creates a Cargo.toml at rootDir if one does not already exist.
// The generated manifest declares a binary target pointing at absBridge and adds
// every entry in absLibCrateDirs as a path dependency so `cargo build` can resolve
// all of them. Entries whose resolved path equals rootDir are treated as the same
// crate and are omitted from [dependencies] to avoid a cyclic self-reference.
//
// Generated layout (rootDir = rust_test_script, two lib crates):
//
//	[package]
//	name = "rust_test_script"
//	version = "0.1.0"
//	edition = "2021"
//
//	[[bin]]
//	name = "rust_test_script"
//	path = "src/rust_test_script_bridge.rs"
//
//	[dependencies]
//	core_utils  = { path = "core_utils" }
//	extra_utils = { path = "extra_utils" }
func ensureRootCargo(rootDir string, absLibCrateDirs []string, absBridge string, lvl logger.LogLevel) error {
	cargoPath := filepath.Join(rootDir, "Cargo.toml")

	// Only regenerate if file doesn't exist OR if it has the auto-gen marker
	if !shouldRegenerateFile(cargoPath) {
		logger.Info(lvl, "  • Keeping existing Cargo.toml at %s (auto-generation disabled by removing marker comment)", cargoPath)
		return nil
	}

	alreadyExisted := false
	if _, err := os.Stat(cargoPath); err == nil {
		alreadyExisted = true
	}

	pkgName := filepath.Base(rootDir)
	pkgName = strings.ReplaceAll(pkgName, " ", "_")
	pkgName = strings.ReplaceAll(pkgName, "-", "_")

	// Path of bridge file relative to rootDir (e.g. "src/rust_test_script_bridge.rs")
	bridgeRel, err := filepath.Rel(rootDir, absBridge)
	if err != nil {
		return fmt.Errorf("cannot compute relative path from %s to %s: %w", rootDir, absBridge, err)
	}
	bridgeRel = filepath.ToSlash(bridgeRel)

	// Build [dependencies] entries, one per lib crate dir that is not rootDir itself.
	type depEntry struct{ name, relPath string }
	var deps []depEntry
	seen := make(map[string]bool)

	for _, absLibCrateDir := range absLibCrateDirs {
		// Skip self-references (same-dir / single-file-promoted layout).
		if filepath.Clean(absLibCrateDir) == filepath.Clean(rootDir) {
			continue
		}

		libCrateRel, err := filepath.Rel(rootDir, absLibCrateDir)
		if err != nil {
			return fmt.Errorf("cannot compute relative path from %s to %s: %w", rootDir, absLibCrateDir, err)
		}
		libCrateRel = filepath.ToSlash(libCrateRel)

		libCrateName := getCrateName(absLibCrateDir)
		if libCrateName == "" {
			libCrateName = filepath.Base(absLibCrateDir)
			libCrateName = strings.ReplaceAll(libCrateName, "-", "_")
		}

		if !seen[libCrateName] {
			seen[libCrateName] = true
			deps = append(deps, depEntry{name: libCrateName, relPath: libCrateRel})
		}
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(`# Auto-generated and maintained by guify_my_script. To keep manual changes and disable auto-generation, remove this line.
[package]
name = "%s"
version = "0.1.0"
edition = "2021"

[[bin]]
name = "%s"
path = "%s"
`, pkgName, pkgName, bridgeRel))

	if len(deps) > 0 {
		sb.WriteString("\n[dependencies]\n")
		for _, dep := range deps {
			sb.WriteString(fmt.Sprintf("%s = { path = \"%s\" }\n", dep.name, dep.relPath))
		}
	}

	if err := os.WriteFile(cargoPath, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("failed writing Cargo.toml: %w", err)
	}
	if alreadyExisted {
		logger.Info(lvl, "  • Updated auto-generated Cargo.toml at %s", cargoPath)
	} else {
		logger.Info(lvl, "  • Generated root Cargo.toml at %s", cargoPath)
	}
	return nil
}

// discoverLibCrates scans all immediate subdirectories of rootDir for Rust source
// folders that are not already the known libCrateDir. A subdirectory qualifies when
// it contains at least one .rs file under a src/ sub-folder. For each qualifying
// directory that is missing a Cargo.toml, a minimal library manifest is generated
// so the crate can be consumed as a path dependency.
//
// The function returns a deduplicated, sorted slice of absolute paths to every lib
// crate found (including absLibCrateDir itself when it is non-empty and valid).
func discoverLibCrates(rootDir, absLibCrateDir string, lvl logger.LogLevel) []string {
	seen := make(map[string]bool)
	var result []string

	addCrate := func(dir string) {
		clean := filepath.Clean(dir)
		if !seen[clean] {
			seen[clean] = true
			result = append(result, clean)
		}
	}

	// Always include the primary lib crate when provided.
	if absLibCrateDir != "" {
		if _, err := os.Stat(absLibCrateDir); err == nil {
			addCrate(absLibCrateDir)
		}
	}

	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return result
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		sub := filepath.Join(rootDir, entry.Name())

		// Skip the bridge output dir itself (same as rootDir or already added).
		if filepath.Clean(sub) == filepath.Clean(rootDir) {
			continue
		}

		// A qualifying lib crate must have at least one .rs file under src/.
		srcDir := filepath.Join(sub, "src")
		hasRust := false
		_ = filepath.WalkDir(srcDir, func(p string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil
			}
			if !d.IsDir() && strings.HasSuffix(p, ".rs") {
				hasRust = true
				return filepath.SkipAll
			}
			return nil
		})
		if !hasRust {
			continue
		}

		// Generate Cargo.toml if one does not exist OR if existing file has marker
		cargoPath := filepath.Join(sub, "Cargo.toml")
		if shouldRegenerateFile(cargoPath) {
			alreadyExisted := false
			if _, err := os.Stat(cargoPath); err == nil {
				alreadyExisted = true
			}
			crateName := filepath.Base(sub)
			crateName = strings.ReplaceAll(crateName, " ", "_")
			crateName = strings.ReplaceAll(crateName, "-", "_")
			contents := fmt.Sprintf(`# Auto-generated and maintained by guify_my_script. To keep manual changes and disable auto-generation, remove this line.
[package]
name = "%s"
version = "0.1.0"
edition = "2021"

[lib]
name = "%s"
path = "src/lib.rs"
`, crateName, crateName)
			if writeErr := os.WriteFile(cargoPath, []byte(contents), 0644); writeErr == nil {
				if alreadyExisted {
					logger.Info(lvl, "  • Updated auto-generated Cargo.toml for discovered crate: %s", sub)
				} else {
					logger.Info(lvl, "  • Generated Cargo.toml for discovered crate: %s", sub)
				}
			} else {
				logger.Info(lvl, "  • Warning: could not generate Cargo.toml for %s: %v", sub, writeErr)
			}
		} else {
			logger.Info(lvl, "  • Keeping existing Cargo.toml for %s (auto-generation disabled by removing marker comment)", sub)
		}

		// Generate src/lib.rs if one does not exist but other .rs files do.
		generateLibRS(srcDir, sub, lvl)

		addCrate(sub)
	}

	return result
}

// generateLibRS creates a src/lib.rs module router when one is missing but sibling
// .rs files exist. The generated file declares each sibling as a public module and
// re-exports all of its public items via `pub use <module>::*;`, giving consumers
// flat access to the crate's entire public API.
func generateLibRS(srcDir, crateDir string, lvl logger.LogLevel) {
	libRS := filepath.Join(srcDir, "lib.rs")

	// Only regenerate if file doesn't exist OR if it has the auto-gen marker
	if !shouldRegenerateFile(libRS) {
		logger.Info(lvl, "  • Keeping existing lib.rs for %s (auto-generation disabled by removing marker comment)", crateDir)
		return
	}

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		logger.Info(lvl, "  • Warning: could not read %s for lib.rs generation: %v", srcDir, err)
		return
	}

	// Collect sibling .rs files (top-level only), excluding entry points.
	var modules []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".rs") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".rs")
		if name == "lib" || name == "main" {
			continue
		}
		modules = append(modules, name)
	}

	if len(modules) == 0 {
		return
	}

	sort.Strings(modules)

	var sb strings.Builder
	sb.WriteString("// Auto-generated and maintained by guify_my_script. To keep manual changes and disable auto-generation, remove this line.\n\n")

	// Module declarations.
	for _, m := range modules {
		sb.WriteString(fmt.Sprintf("pub mod %s;\n", m))
	}

	sb.WriteString("\n")

	// Re-exports for flat access.
	for _, m := range modules {
		sb.WriteString(fmt.Sprintf("pub use %s::*;\n", m))
	}

	alreadyExisted := false
	if _, err := os.Stat(libRS); err == nil {
		alreadyExisted = true
	}

	if writeErr := os.WriteFile(libRS, []byte(sb.String()), 0644); writeErr == nil {
		if alreadyExisted {
			logger.Info(lvl, "  • Updated auto-generated lib.rs module router for: %s", crateDir)
		} else {
			logger.Info(lvl, "  • Generated lib.rs module router for: %s", crateDir)
		}
	} else {
		logger.Info(lvl, "  • Warning: could not generate lib.rs for %s: %v", crateDir, writeErr)
	}
}

// hasBinaryTarget returns true if the Cargo.toml in dir declares a [[bin]] section
// or if the project has a src/main.rs (the implicit binary convention).
func hasBinaryTarget(dir string) bool {
	// Implicit: src/main.rs present means a binary target exists
	if _, err := os.Stat(filepath.Join(dir, "src", "main.rs")); err == nil {
		return true
	}
	// Explicit: [[bin]] table in Cargo.toml
	data, err := os.ReadFile(filepath.Join(dir, "Cargo.toml"))
	if err != nil {
		return false
	}
	return strings.Contains(string(data), "[[bin]]")
}

// getCrateName parses Cargo.toml to extract the crate name
func getCrateName(targetDir string) string {
	cargoPath := filepath.Join(targetDir, "Cargo.toml")
	data, err := os.ReadFile(cargoPath)
	if err != nil {
		return ""
	}

	re := regexp.MustCompile(`(?m)^\s*name\s*=\s*["']([^"']+)["']`)
	matches := re.FindStringSubmatch(string(data))
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// extractRustMetadata scans Rust source files to extract functions, types, and parameters
func extractRustMetadata(targetDir, crateName string, targetFiles []string, excludeMap map[string]bool, lvl logger.LogLevel, includeFuncParams bool) ([]string, []FuncMetaData, bool, error) {
	var functions []FuncMetaData
	importMap := make(map[string]bool)
	needsStrconv := false
	seenBashNames := make(map[string]string) // bashName → "crate::GoName" of first occurrence

	if crateName != "" {
		importMap[crateName] = true
	}

	for _, fileToParse := range targetFiles {
		contentBytes, err := os.ReadFile(fileToParse)
		if err != nil {
			continue
		}

		relPath, _ := filepath.Rel(targetDir, fileToParse)
		modulePath := crateName
		if relPath != "src/lib.rs" && relPath != "src/main.rs" && !strings.HasPrefix(relPath, "..") {
			cleanMod := strings.TrimSuffix(filepath.Base(relPath), ".rs")
			modulePath = fmt.Sprintf("%s::%s", crateName, cleanMod)
		}

		if excludeMap != nil && (excludeMap[modulePath] || excludeMap[crateName]) {
			logger.Info(lvl, "  • Skipping excluded module path: %s", modulePath)
			continue
		}

		parsedFns := parseRustFunctions(string(contentBytes), modulePath, includeFuncParams)
		for _, fn := range parsedFns {
			// Duplicate endpoint name guard: two public functions that share the same
			// bash_name would produce colliding extern "C" and CLI match entries.
			qualifiedName := fmt.Sprintf("%s::%s", fn.PackageName, fn.RustName)
			if firstSeen, collision := seenBashNames[fn.BashName]; collision {
				return nil, nil, false, fmt.Errorf(
					"duplicate endpoint name %q: %s and %s both map to the same CLI/export name — rename one of the exported functions to resolve the collision",
					fn.BashName, firstSeen, qualifiedName,
				)
			}
			seenBashNames[fn.BashName] = qualifiedName

			functions = append(functions, fn)
			importMap[fn.PackageName] = true
			for _, arg := range fn.Args {
				if arg.Type == "bool" || arg.Type == "int" || arg.Type == "float" || arg.Type == "*bool" {
					needsStrconv = true
				}
			}
		}
	}

	var dynamicImports []string
	for imp := range importMap {
		if imp != "" {
			dynamicImports = append(dynamicImports, imp)
		}
	}

	return dynamicImports, functions, needsStrconv, nil
}

// parseRustFunctions extracts public functions from Rust code text
func parseRustFunctions(code, packageName string, includeFuncParams bool) []FuncMetaData {
	var results []FuncMetaData

	// Pre-process code: remove line & block comments while ignoring inside strings
	cleaned := removeComments(code)

	// Regex to identify tests block and strip it
	cleaned = stripTestModules(cleaned)

	// Strip impl blocks so struct methods are not parsed as standalone functions
	cleaned = stripImplBlocks(cleaned)

	// Regex for public function declarations
	// Matches: pub fn name(args) [-> ret]
	fnRegex := regexp.MustCompile(`(?s)\bpub\s+(?:async\s+|unsafe\s+|extern\s+"[^"]+"\s+)*fn\s+([A-Za-z0-9_]+)\s*\((.*?)\)(?:\s*->\s*([^{;]+))?`)

	matches := fnRegex.FindAllStringSubmatch(cleaned, -1)
	for _, m := range matches {
		fnName := strings.TrimSpace(m[1])
		if fnName == "main" || fnName == "test" || strings.HasPrefix(fnName, "_") {
			continue
		}

		rawParams := strings.TrimSpace(m[2])
		rawReturn := strings.TrimSpace(m[3])

		hasErrorReturn := strings.Contains(rawReturn, "Result") || strings.Contains(rawReturn, "Error")
		returnCount := 0
		if rawReturn != "" && rawReturn != "()" {
			returnCount = 1
		}

		args, hasUnsupported, hasFunc := parseRustParameters(rawParams)
		if hasUnsupported || (hasFunc && !includeFuncParams) {
			continue
		}

		// Calculate BashName and GoName
		bashName := strings.ToLower(strings.Join(internal_utils.SplitCamelCase(fnName), "_"))
		if strings.HasPrefix(bashName, "rust_") {
			bashName = strings.TrimPrefix(bashName, "rust_")
		}

		// GoName is PascalCase
		goName := toPascalCase(fnName)
		if goName == "" {
			goName = fnName
		}

		callPrefix := ""
		if packageName != "" {
			callPrefix = packageName + "::"
		}

		results = append(results, FuncMetaData{
			RustName:       fnName,
			GoName:         goName,
			BashName:       bashName,
			PackageName:    packageName,
			HasErrorReturn: hasErrorReturn,
			ReturnCount:    returnCount,
			Args:           args,
			CallPrefix:     callPrefix,
		})
	}

	return results
}

// parseRustParameters parses parameter list e.g. "username: &str, age: u8, status: &str"
func parseRustParameters(raw string) ([]ArgMetaData, bool, bool) {
	if strings.TrimSpace(raw) == "" {
		return nil, false, false
	}

	var args []ArgMetaData
	hasUnsupported := false
	hasFunc := false

	parts := splitBalanced(raw, ',')
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if p == "self" || p == "&self" || p == "&mut self" ||
			strings.HasPrefix(p, "self:") || strings.HasPrefix(p, "&self:") || strings.HasPrefix(p, "&mut self:") {
			return nil, true, false
		}

		colonIdx := strings.Index(p, ":")
		if colonIdx == -1 {
			continue
		}

		name := strings.TrimSpace(p[:colonIdx])
		typeStr := strings.TrimSpace(p[colonIdx+1:])

		// Check for function parameter types e.g. F: Fn() or impl Fn()
		if strings.Contains(typeStr, "Fn(") || strings.Contains(typeStr, "FnMut(") || strings.Contains(typeStr, "FnOnce(") {
			hasFunc = true
		}

		typeCat, rawType, ok := classifyRustType(typeStr)
		if !ok {
			hasUnsupported = true
		}

		args = append(args, ArgMetaData{
			Name:     name,
			Type:     typeCat,
			RawType:  rawType,
			CastType: "",
		})
	}

	return args, hasUnsupported, hasFunc
}

// classifyRustType maps a Rust type string to an interoperable category
func classifyRustType(typeStr string) (category string, rawType string, ok bool) {
	clean := strings.TrimSpace(typeStr)

	// String types
	if clean == "&str" || clean == "&'static str" || clean == "& 'static str" || clean == "&'a str" || clean == "String" || clean == "&String" {
		return "string", clean, true
	}

	// Boolean types
	if clean == "bool" {
		return "bool", "bool", true
	}
	if clean == "Option<bool>" || clean == "Option < bool >" {
		return "*bool", "Option<bool>", true
	}

	// Integer types
	switch clean {
	case "i8", "i16", "i32", "i64", "isize", "u8", "u16", "u32", "u64", "usize":
		return "int", clean, true
	}

	// Float types
	switch clean {
	case "f32", "f64":
		return "float", clean, true
	}

	// Optional strings/integers
	if strings.HasPrefix(clean, "Option<") && strings.HasSuffix(clean, ">") {
		inner := strings.TrimSpace(clean[len("Option<") : len(clean)-1])
		if inner == "bool" {
			return "*bool", clean, true
		}
		return "string", clean, true
	}

	// Unrecognized / complex type
	return "string", clean, false
}

// splitBalanced splits by delimiter ignoring commas inside <...> or (...)
func splitBalanced(s string, delim rune) []string {
	var result []string
	var current strings.Builder
	angleDepth := 0
	parenDepth := 0

	for _, ch := range s {
		switch ch {
		case '<':
			angleDepth++
			current.WriteRune(ch)
		case '>':
			if angleDepth > 0 {
				angleDepth--
			}
			current.WriteRune(ch)
		case '(':
			parenDepth++
			current.WriteRune(ch)
		case ')':
			if parenDepth > 0 {
				parenDepth--
			}
			current.WriteRune(ch)
		case delim:
			if angleDepth == 0 && parenDepth == 0 {
				result = append(result, strings.TrimSpace(current.String()))
				current.Reset()
			} else {
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
	}

	if current.Len() > 0 {
		result = append(result, strings.TrimSpace(current.String()))
	}
	return result
}

// removeComments strips // and /* */ comments while preserving strings
func removeComments(s string) string {
	var out strings.Builder
	inString := false
	inLineComment := false
	inBlockComment := false
	chars := []rune(s)
	n := len(chars)

	for i := 0; i < n; i++ {
		c := chars[i]

		if inLineComment {
			if c == '\n' {
				inLineComment = false
				out.WriteRune(c)
			}
			continue
		}

		if inBlockComment {
			if c == '*' && i+1 < n && chars[i+1] == '/' {
				inBlockComment = false
				i++
			}
			continue
		}

		if inString {
			if c == '\\' && i+1 < n {
				out.WriteRune(c)
				out.WriteRune(chars[i+1])
				i++
				continue
			}
			if c == '"' {
				inString = false
			}
			out.WriteRune(c)
			continue
		}

		// Not in string or comment
		if c == '"' {
			inString = true
			out.WriteRune(c)
			continue
		}

		if c == '/' && i+1 < n && chars[i+1] == '/' {
			inLineComment = true
			i++
			continue
		}

		if c == '/' && i+1 < n && chars[i+1] == '*' {
			inBlockComment = true
			i++
			continue
		}

		out.WriteRune(c)
	}

	return out.String()
}

// stripTestModules removes #[cfg(test)] and mod tests { ... } blocks
func stripTestModules(s string) string {
	re := regexp.MustCompile(`(?s)#\[cfg\(test\)\]\s*mod\s+[A-Za-z0-9_]+\s*\{`)
	loc := re.FindStringIndex(s)
	if loc == nil {
		return s
	}

	start := loc[1] - 1
	depth := 0
	end := -1
	chars := []rune(s)

	for i := start; i < len(chars); i++ {
		if chars[i] == '{' {
			depth++
		} else if chars[i] == '}' {
			depth--
			if depth == 0 {
				end = i
				break
			}
		}
	}

	if end != -1 {
		return s[:loc[0]] + s[end+1:]
	}
	return s[:loc[0]]
}

// stripImplBlocks removes impl ... { ... } blocks so only module-level functions are parsed
func stripImplBlocks(s string) string {
	re := regexp.MustCompile(`(?s)\bimpl(?:\s+<[^>]+>)?\s+[^{]+\{`)
	for {
		loc := re.FindStringIndex(s)
		if loc == nil {
			break
		}
		start := loc[1] - 1
		depth := 0
		end := -1
		chars := []rune(s)
		for i := start; i < len(chars); i++ {
			if chars[i] == '{' {
				depth++
			} else if chars[i] == '}' {
				depth--
				if depth == 0 {
					end = i
					break
				}
			}
		}
		if end != -1 {
			s = s[:loc[0]] + s[end+1:]
		} else {
			break
		}
	}
	return s
}

// copyExecutable copies file from src to dst and sets chmod 0755
func copyExecutable(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return os.Chmod(dst, 0755)
}

// toPascalCase converts snake_case or camelCase to PascalCase
func toPascalCase(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	var out []string
	for _, part := range parts {
		words := internal_utils.SplitCamelCase(part)
		for _, w := range words {
			if len(w) > 0 {
				out = append(out, strings.ToUpper(w[:1])+strings.ToLower(w[1:]))
			}
		}
	}
	return strings.Join(out, "")
}
