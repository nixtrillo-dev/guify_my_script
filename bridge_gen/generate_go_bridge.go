package bridge_gen

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"guify_my_script/internal_utils"
	"guify_my_script/logger"
)

// ArgMetaData tracks individual parameter identity alongside its parsed Go type layout
type ArgMetaData struct {
	Name     string
	Type     string // "string", "bool", "int", "*bool"
	CastType string // Stores fully qualified enum types for casting (e.g., "logger.LogLevel")
}

type FuncMetaData struct {
	GoName         string
	BashName       string
	PackageName    string
	HasErrorReturn bool
	ReturnCount    int // Dynamically tracks the total number of return values
	Args           []ArgMetaData
}

type TemplateWrapper struct {
	Imports      []string
	NeedsStrconv bool
	Functions    []FuncMetaData
}

// Complete cross-boundary interop boilerplate definitions
const bridgeTemplate = `package main

import "C"
import (
	"fmt"
	"os"
	{{if .NeedsStrconv}}"strconv"{{end}}
	{{range .Imports}}"{{.}}"
	{{end}}
)

// =====================================================================
// AUTOMATICALLY GENERATED C++ EXPORTS
// =====================================================================
{{range .Functions}}
//export Go{{.GoName}}
func Go{{.GoName}}({{range $i, $arg := .Args}}{{if $i}}, {{end}}{{$arg.Name}} {{if eq $arg.Type "string"}}*C.char{{else if eq $arg.Type "bool"}}C.int{{else if eq $arg.Type "int"}}C.int{{else if eq $arg.Type "*bool"}}*C.int{{end}}{{end}}) C.int {
	{{range .Args}}
	{{if eq .Type "string"}}go_{{.Name}} := C.GoString({{.Name}})
	{{else if eq .Type "bool"}}go_{{.Name}} := {{.Name}} == 1
	{{else if eq .Type "int"}}go_{{.Name}} := {{if .CastType}}{{.CastType}}(int({{.Name}})){{else}}int({{.Name}}){{end}}
	{{else if eq .Type "*bool"}}var go_{{.Name}} *bool
	if {{.Name}} != nil {
		val_{{.Name}} := *{{.Name}} != 0
		go_{{.Name}} = &val_{{.Name}}
	}
	{{end}}{{end}}
	{{if .HasErrorReturn}}
	{{if eq .ReturnCount 1}}err := {{else if eq .ReturnCount 2}}_, err := {{else if eq .ReturnCount 3}}_, _, err := {{else if eq .ReturnCount 4}}_, _, _, err := {{end}}{{if eq .PackageName "main"}}{{.GoName}}{{else}}{{.PackageName}}.{{.GoName}}{{end}}({{range $i, $arg := .Args}}{{if $i}}, {{end}}go_{{$arg.Name}}{{end}})
	if err != nil {
		fmt.Printf("Pipeline Error in {{.GoName}}: %v\n", err)
		return 0
	}
	{{else}}
	{{if eq .PackageName "main"}}{{.GoName}}{{else}}{{.PackageName}}.{{.GoName}}{{end}}({{range $i, $arg := .Args}}{{if $i}}, {{end}}go_{{$arg.Name}}{{end}})
	{{end}}
	return 1
}
{{end}}

// =====================================================================
// AUTOMATICALLY GENERATED BASH CLI ROUTER
// =====================================================================
func main() {
	if len(os.Args) < 2 {
		fmt.Println("Error: Missing command function name")
		os.Exit(1)
	}

	switch os.Args[1] {
	{{range .Functions}}    case "{{.BashName}}":
		{{if .Args}}if len(os.Args) < {{add (len .Args) 2}} {
			fmt.Println("Error: {{.BashName}} requires parameters")
			os.Exit(1)
		}

		// Parse string command line metrics safely to real types
		{{range $idx, $arg := .Args}}
		{{if eq $arg.Type "string"}}
		cli_{{$arg.Name}} := os.Args[{{add $idx 2}}]
		{{else if eq $arg.Type "bool"}}
		cli_{{$arg.Name}}, err_{{$arg.Name}} := strconv.ParseBool(os.Args[{{add $idx 2}}])
		if err_{{$arg.Name}} != nil {
			if os.Args[{{add $idx 2}}] == "1" {
				cli_{{$arg.Name}} = true
			} else if os.Args[{{add $idx 2}}] == "0" {
				cli_{{$arg.Name}} = false
			} else {
				fmt.Printf("Error: parameter '{{$arg.Name}}' must be a valid boolean evaluation: %v\n", err_{{$arg.Name}})
				os.Exit(1)
			}
		}
		{{else if eq $arg.Type "*bool"}}
		var cli_{{$arg.Name}} *bool
		raw_{{$arg.Name}} := os.Args[{{add $idx 2}}]
		if raw_{{$arg.Name}} != "" && raw_{{$arg.Name}} != "nil" && raw_{{$arg.Name}} != "null" {
			parsed_{{$arg.Name}}, err_{{$arg.Name}} := strconv.ParseBool(raw_{{$arg.Name}})
			if err_{{$arg.Name}} != nil {
				if raw_{{$arg.Name}} == "1" {
					parsed_{{$arg.Name}} = true
				} else if raw_{{$arg.Name}} == "0" {
					parsed_{{$arg.Name}} = false
				} else {
					fmt.Printf("Error: parameter '{{$arg.Name}}' must be a valid boolean or nil: %v\n", err_{{$arg.Name}})
					os.Exit(1)
				}
			}
			cli_{{$arg.Name}} = &parsed_{{$arg.Name}}
		}
		{{else if eq $arg.Type "int"}}
		var cli_{{$arg.Name}}_raw int
		raw_{{$arg.Name}} := os.Args[{{add $idx 2}}]
		if raw_{{$arg.Name}} != "" && raw_{{$arg.Name}} != "null" && raw_{{$arg.Name}} != "nil" {
			parsed_{{$arg.Name}}, err_{{$arg.Name}} := strconv.Atoi(raw_{{$arg.Name}})
			if err_{{$arg.Name}} != nil {
				fmt.Printf("Error: parameter '{{$arg.Name}}' must be a valid integer: %v\n", err_{{$arg.Name}})
				os.Exit(1)
			}
			cli_{{$arg.Name}}_raw = parsed_{{$arg.Name}}
		}
		{{if $arg.CastType}}cli_{{$arg.Name}} := {{$arg.CastType}}(cli_{{$arg.Name}}_raw)
		{{else}}cli_{{$arg.Name}} := cli_{{$arg.Name}}_raw{{end}}
		{{end}}{{end}}
		{{if .HasErrorReturn}}
		{{if eq .ReturnCount 1}}err := {{else if eq .ReturnCount 2}}_, err := {{else if eq .ReturnCount 3}}_, _, err := {{else if eq .ReturnCount 4}}_, _, _, err := {{end}}{{if eq .PackageName "main"}}{{.GoName}}{{else}}{{.PackageName}}.{{.GoName}}{{end}}({{range $i, $arg := .Args}}{{if $i}}, {{end}}cli_{{$arg.Name}}{{end}})
		if err != nil {
			fmt.Printf("Execution Failure: %v\n", err)
			os.Exit(1)
		}
		{{else}}
		{{if eq .PackageName "main"}}{{.GoName}}{{else}}{{.PackageName}}.{{.GoName}}{{end}}({{range $i, $arg := .Args}}{{if $i}}, {{end}}cli_{{$arg.Name}}{{end}})
		{{end}}
		{{else}}
		{{if .HasErrorReturn}}
		{{if eq .ReturnCount 1}}err := {{else if eq .ReturnCount 2}}_, err := {{else if eq .ReturnCount 3}}_, _, err := {{else if eq .ReturnCount 4}}_, _, _, err := {{end}}{{if eq .PackageName "main"}}{{.GoName}}{{else}}{{.PackageName}}.{{.GoName}}{{end}}()
		if err != nil {
			fmt.Printf("Execution Failure: %v\n", err)
			os.Exit(1)
		}
		{{else}}
		{{if eq .PackageName "main"}}{{.GoName}}{{else}}{{.PackageName}}.{{.GoName}}{{end}}()
		{{end}}
		{{end}}
	{{end}}    default:
		fmt.Printf("Error: Unknown function command '%s'\n", os.Args[1])
		os.Exit(1)
	}
}
`

// GetNonMainPackagesJSON takes a Go project directory path and outputs all non-main packages as a JSON array.
func GetNonMainPackagesJSON(inputPath string, logLevel string) (string, error) {
	lvl := logger.ParseLogLevel(logLevel)

	// Guard Clause: Return empty JSON array immediately if input string is empty
	if inputPath == "" {
		emptyJSON := "[]"
		logger.Info(lvl, "%s", emptyJSON)
		return emptyJSON, nil
	}

	absInput, err := filepath.Abs(inputPath)
	if err != nil {
		return "", fmt.Errorf("failed to canonicalize input path: %w", err)
	}

	fi, err := os.Stat(absInput)
	if err != nil {
		return "", fmt.Errorf("input target path does not exist: %w", err)
	}

	var targetDir string
	if fi.IsDir() {
		targetDir = absInput
	} else {
		targetDir = filepath.Dir(absInput)
	}

	moduleBaseName := filepath.Base(targetDir)
	moduleBaseName = strings.ReplaceAll(moduleBaseName, " ", "_")
	if moduleBaseName == "." || moduleBaseName == "/" || moduleBaseName == "" {
		moduleBaseName = "go_external_bridge"
	}

	var targetFiles []string
	if fi.IsDir() {
		err = filepath.WalkDir(absInput, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			absPath, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			if !d.IsDir() && filepath.Ext(absPath) == ".go" {
				targetFiles = append(targetFiles, absPath)
			}
			return nil
		})
		if err != nil {
			return "", fmt.Errorf("error recursively traversing directories: %w", err)
		}
	} else {
		targetFiles = append(targetFiles, absInput)
	}

	// Leverage extractMetadata to scrape all the import paths logic
	dynamicImports, _, _ := extractMetadata(targetDir, moduleBaseName, targetFiles, nil, lvl, false)

	jsonData, err := json.Marshal(dynamicImports)
	if err != nil {
		return "", fmt.Errorf("failed to encode packages to JSON: %w", err)
	}

	// Convert bytes to string for both logging and returning
	jsonString := string(jsonData)
	logger.Info(lvl, "%s", jsonString)

	return jsonString, nil
}

// extractMetadata isolates the AST parsing logic to discover packages, imports, and function signatures.
func extractMetadata(targetDir, moduleBaseName string, targetFiles []string, excludeMap map[string]bool, lvl logger.LogLevel, includeFuncParams bool) ([]string, []FuncMetaData, bool) {
	var functions []FuncMetaData
	importMap := make(map[string]bool)
	fset := token.NewFileSet()
	needsStrconv := false

	for _, fileToParse := range targetFiles {
		node, err := parser.ParseFile(fset, fileToParse, nil, parser.ParseComments)
		if err != nil {
			continue
		}

		parsedPackageName := node.Name.Name

		// 1. Calculate the full import path based on the directory structure FIRST
		var importPath string
		fileDir := filepath.Dir(fileToParse)
		relPath, err := filepath.Rel(targetDir, fileDir)
		if err == nil && relPath != "." {
			relClean := strings.ReplaceAll(relPath, " ", "_")
			importPath = filepath.ToSlash(filepath.Join(moduleBaseName, relClean))
		} else {
			parsedClean := strings.ReplaceAll(parsedPackageName, " ", "_")
			importPath = filepath.ToSlash(filepath.Join(moduleBaseName, parsedClean))
		}

		// 2. Guard Clause: Skip the entire file if the FULL import path is explicitly excluded
		if excludeMap != nil && excludeMap[importPath] {
			logger.Info(lvl, "  • Skipping excluded package path: %s", importPath)
			continue
		}

		for _, decl := range node.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name.Name == "main" {
				continue
			}

			goName := fn.Name.Name
			if !ast.IsExported(goName) {
				continue
			}

			bashName := strings.ToLower(strings.Join(internal_utils.SplitCamelCase(goName), "_"))

			hasErrorReturn := false
			returnCount := 0
			if fn.Type.Results != nil {
				for _, fields := range fn.Type.Results.List {
					count := len(fields.Names)
					if count == 0 {
						count = 1
					}
					returnCount += count

					if ident, ok := fields.Type.(*ast.Ident); ok && ident.Name == "error" {
						hasErrorReturn = true
					}
				}
			}

			var args []ArgMetaData
			hasFuncParam := false // Track if a func() parameter is detected
			localNeedsStrconv := false // Track strconv needs for this function only

			if fn.Type.Params != nil {
				for _, field := range fn.Type.Params.List {

					// Detect if the parameter is a function type
					if _, isFunc := field.Type.(*ast.FuncType); isFunc {
						hasFuncParam = true
					}

					typeStr := "string"
					castType := ""

					// Parse base identifiers (int, string, bool)
					if ident, ok := field.Type.(*ast.Ident); ok {
						switch ident.Name {
						case "bool":
							typeStr = "bool"
							localNeedsStrconv = true
						case "int", "int64", "int32":
							typeStr = "int"
							localNeedsStrconv = true
						case "string":
							typeStr = "string"
						default:
							// Assume unknown local types are custom enums backed by int
							// FIX: Prepend the parsed package name to the local identifier so main can see it
							typeStr = "int"
							castType = fmt.Sprintf("%s.%s", parsedPackageName, ident.Name)
							localNeedsStrconv = true
						}
					// Parse pointer expressions (*bool)
					} else if starExpr, ok := field.Type.(*ast.StarExpr); ok {
						if ident, ok := starExpr.X.(*ast.Ident); ok {
							if ident.Name == "bool" {
								typeStr = "*bool"
								localNeedsStrconv = true
							}
						}
					// Parse selector expressions (e.g., logger.LogLevel)
					} else if selExpr, ok := field.Type.(*ast.SelectorExpr); ok {
						if xIdent, ok := selExpr.X.(*ast.Ident); ok {
							typeStr = "int"
							castType = fmt.Sprintf("%s.%s", xIdent.Name, selExpr.Sel.Name)
							localNeedsStrconv = true
						}
					}

					for _, name := range field.Names {
						args = append(args, ArgMetaData{
							Name:     name.Name,
							Type:     typeStr,
							CastType: castType,
						})
					}
				}
			}

			// Skip this function if it has a func() param and we are not explicitly including them
			if hasFuncParam && !includeFuncParams {
				continue
			}

			// ADD COMMITS
			// Now that the function is confirmed to be included, commit the imports and flags
			if parsedPackageName != "main" {
				importMap[importPath] = true
			}
			if localNeedsStrconv {
				needsStrconv = true
			}

			functions = append(functions, FuncMetaData{
				GoName:         goName,
				BashName:       bashName,
				PackageName:    parsedPackageName,
				HasErrorReturn: hasErrorReturn,
				ReturnCount:    returnCount,
				Args:           args,
			})
		}
	}

	var dynamicImports []string
	for imp := range importMap {
		dynamicImports = append(dynamicImports, imp)
	}

	return dynamicImports, functions, needsStrconv
}

// GenerateBridge accepts an 'excludePackages' comma-separated string, 'logLevelStr', and 'includeFuncParams' flag
func GenerateBridge(inputPath, outputPath, excludePackages string, logLevelStr string, includeFuncParams bool) (string, string, error) {
	lvl := logger.ParseLogLevel(logLevelStr)
	logger.Info(lvl, "Generating the bridge file...")

	// 1. Parse the comma-separated exclusion list into a lookup map using a CSV reader
	excludeMap := make(map[string]bool)
	if excludePackages != "" {
		r := csv.NewReader(strings.NewReader(excludePackages))
		records, err := r.Read()
		if err == nil {
			for _, pkg := range records {
				excludeMap[strings.TrimSpace(pkg)] = true
			}
		} else {
			logger.Info(lvl, "Warning: failed to parse excludePackages as CSV: %v", err)
		}
	}

	if inputPath == "" {
		inputPath = "."
	}

	absInput, err := filepath.Abs(inputPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to canonicalize input path: %w", err)
	}

	fi, err := os.Stat(absInput)
	if err != nil {
		return "", "", fmt.Errorf("input target path does not exist: %w", err)
	}

	var targetDir string
	if fi.IsDir() {
		targetDir = absInput
	} else {
		targetDir = filepath.Dir(absInput)
	}

	moduleBaseName := filepath.Base(targetDir)
	moduleBaseName = strings.ReplaceAll(moduleBaseName, " ", "_")
	if moduleBaseName == "." || moduleBaseName == "/" || moduleBaseName == "" {
		moduleBaseName = "go_external_bridge"
	}

	if outputPath == "" {
		outputPath = filepath.Join(targetDir, "go_external_bridge_bridge.go")
	} else if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(targetDir, outputPath)
	}

	absOutput, err := filepath.Abs(outputPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to canonicalize output path: %w", err)
	}

	var targetFiles []string
	if fi.IsDir() {
		err = filepath.WalkDir(absInput, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			absPath, err := filepath.Abs(path)
			if err != nil {
				return err
			}
			if !d.IsDir() && filepath.Ext(absPath) == ".go" && absPath != absOutput {
				targetFiles = append(targetFiles, absPath)
				logger.Info(lvl, "  • Found: %s", absPath)
			}
			return nil
		})
		if err != nil {
			return "", "", fmt.Errorf("error recursively traversing directories: %w", err)
		}
	} else {
		targetFiles = append(targetFiles, absInput)
	}

	// 2. Delegate the parsing process to the shared abstraction function with the new flag
	dynamicImports, functions, needsStrconv := extractMetadata(targetDir, moduleBaseName, targetFiles, excludeMap, lvl, includeFuncParams)

	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
	}

	tmpl, err := template.New("bridge").Funcs(funcMap).Parse(bridgeTemplate)
	if err != nil {
		return "", "", fmt.Errorf("template initialization error: %w", err)
	}

	outputFile, err := os.Create(absOutput)
	if err != nil {
		return "", "", fmt.Errorf("failed creating file target: %w", err)
	}
	defer outputFile.Close()

	wrapperData := TemplateWrapper{
		Imports:      dynamicImports,
		NeedsStrconv: needsStrconv,
		Functions:    functions,
	}

	if err := tmpl.Execute(outputFile, wrapperData); err != nil {
		return "", "", fmt.Errorf("template write error: %w", err)
	}

	logger.Info(lvl, "  • Successfully generated bridge file %s\n", absOutput)
	return targetDir, absOutput, nil
}

func CompileLocal(targetDir, absBridge, binaryName string, logLevelStr string) error {
	lvl := logger.ParseLogLevel(logLevelStr)

	if binaryName == "" {
		binaryName = filepath.Base(targetDir)
	}

	// 1. Split the path into directory and filename
	dir, file := filepath.Split(binaryName)

	// 2. Only replace spaces with underscores in the filename
	file = strings.ReplaceAll(file, " ", "_")

	// 3. Rejoin them (if no directory was passed, dir will just be empty)
	binaryName = filepath.Join(dir, file)

	binaryName = strings.TrimSuffix(binaryName, ".so")
	binaryName = strings.TrimSuffix(binaryName, ".dll")

	modPath := filepath.Join(targetDir, "go.mod")
	if _, err := os.Stat(modPath); os.IsNotExist(err) {
		moduleName := filepath.Base(targetDir)
		moduleName = strings.ReplaceAll(moduleName, " ", "_")
		if moduleName == "." || moduleName == "/" || moduleName == "" {
			moduleName = "go_external_bridge"
		}

		cmdInit := exec.Command("go", "mod", "init", moduleName)
		cmdInit.Dir = targetDir
		if out, err := cmdInit.CombinedOutput(); err != nil {
			return fmt.Errorf("failed initializing local go.mod file: %s (%w)", string(out), err)
		}
	}

	cmdTidy := exec.Command("go", "mod", "tidy")
	cmdTidy.Dir = targetDir
	if out, err := cmdTidy.CombinedOutput(); err != nil {
		return fmt.Errorf("failed running go mod tidy optimization step: %s (%w)", string(out), err)
	}

	// Define our target architectures and their corresponding C cross-compilers
	// Assuming Linux targets (GOOS=linux). Adjust if targeting macOS/Windows.
	targets := []struct {
		GOARCH string
		CC     string
	}{
		{"amd64", "gcc"},                   // Standard x86_64 compiler
		{"arm64", "aarch64-linux-gnu-gcc"}, // ARM64 cross-compiler
	}

	for _, target := range targets {
		logger.Info(lvl, "Building for architecture: %s...", target.GOARCH)

		cliBinName := fmt.Sprintf("%s_%s", binaryName, target.GOARCH)
		libBinName := fmt.Sprintf("%s_%s.so", binaryName, target.GOARCH)

		// 1. Build the Standalone Shell Executable
		cmdCLI := exec.Command("go", "build", "-o", cliBinName, ".")
		cmdCLI.Dir = targetDir
		cmdCLI.Env = append(os.Environ(),
			"CGO_ENABLED=1",         // Explicitly enable CGO for the CLI
			"GOARCH="+target.GOARCH,
			"GOOS=linux",
			"CC="+target.CC, // Supply the cross-compiler mapping
		)

		var errCLI strings.Builder
		cmdCLI.Stdout = os.Stdout
		cmdCLI.Stderr = &errCLI

		if err := cmdCLI.Run(); err != nil {
			return fmt.Errorf("CLI binary build failure for %s:\n%s\ninternal error: %w", target.GOARCH, errCLI.String(), err)
		}
		_ = os.Chmod(filepath.Join(targetDir, cliBinName), 0755)

		// 2. Build the C++ Shared Library (Requires CGO and specific CC)
		cmdLib := exec.Command("go", "build", "-o", libBinName, "-buildmode=c-shared", ".")
		cmdLib.Dir = targetDir
		cmdLib.Env = append(os.Environ(),
			"CGO_ENABLED=1",
			"GOARCH="+target.GOARCH,
			"GOOS=linux",
			"CC="+target.CC, // Critical for CGO cross-compilation
		)

		var errLib strings.Builder
		cmdLib.Stdout = os.Stdout
		cmdLib.Stderr = &errLib

		if err := cmdLib.Run(); err != nil {
			return fmt.Errorf("C++ shared library build failure for %s:\n%s\ninternal error: %w", target.GOARCH, errLib.String(), err)
		}

		logger.Info(lvl, "  • Built: %s", cliBinName)
		logger.Info(lvl, "  • Built: %s", libBinName)
		logger.Info(lvl, "")
	}

	logger.Info(lvl, "Successfully compiled unified pipeline deliverables in %s.", targetDir)
	return nil
}
