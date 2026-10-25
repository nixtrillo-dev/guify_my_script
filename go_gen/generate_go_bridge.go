// Copyright (c) 2026 Nick Rowley
// All rights reserved.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package go_gen

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
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
	Imports   []string
	Functions []FuncMetaData
}

// Complete cross-boundary interop boilerplate definitions
const bridgeTemplate = `//go:build !android

package main

import "C"
import (
{{range .Imports}}	"{{.}}"
{{end}})

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

const androidBridgeTemplate = `//go:build android

package main

import (
{{range .Imports}}	"{{.}}"
{{end}})

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
func GetGoNonMainPackagesJSON(inputPath string, logLevel string) (string, error) {
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
	dynamicImports, _, _, _ := extractMetadata(targetDir, moduleBaseName, targetFiles, nil, lvl, false)

	jsonData, err := json.Marshal(dynamicImports)
	if err != nil {
		return "", fmt.Errorf("failed to encode packages to JSON: %w", err)
	}

	// Convert bytes to string for both logging and returning
	jsonString := string(jsonData)
	logger.Data(lvl, "%s", jsonString)

	return jsonString, nil
}

// extractMetadata isolates the AST parsing logic to discover packages, imports, and function signatures.
func extractMetadata(targetDir, moduleBaseName string, targetFiles []string, excludeMap map[string]bool, lvl logger.LogLevel, includeFuncParams bool) ([]string, []FuncMetaData, bool, error) {
	var functions []FuncMetaData
	importMap := make(map[string]bool)
	fset := token.NewFileSet()
	needsStrconv := false
	seenBashNames := make(map[string]string) // bashName → "package.GoName" of first occurrence

	// Pre-scan all target files to identify declared struct, interface, and composite types
	compositeTypes := make(map[string]bool)
	for _, fileToParse := range targetFiles {
		node, err := parser.ParseFile(fset, fileToParse, nil, 0)
		if err != nil {
			continue
		}
		for _, decl := range node.Decls {
			if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.TYPE {
				for _, spec := range genDecl.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						switch typeSpec.Type.(type) {
						case *ast.StructType, *ast.InterfaceType, *ast.MapType, *ast.ArrayType, *ast.ChanType:
							compositeTypes[typeSpec.Name.Name] = true
						}
					}
				}
			}
		}
	}

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
			// Skip non-functions, methods with struct/type receivers, and main()
			if !ok || fn.Recv != nil || fn.Name.Name == "main" {
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
			hasFuncParam := false        // Track if a func() parameter is detected
			hasUnsupportedParam := false // Track if struct/composite parameters are detected
			localNeedsStrconv := false   // Track strconv needs for this function only

			if fn.Type.Params != nil {
				for _, field := range fn.Type.Params.List {
					// A variadic parameter appears as ast.Ellipsis in the AST. Treat its
					// element type like a normal parameter and expose one value to the
					// generated wrapper; calling f(value) is valid for f(args ...T).
					paramType := field.Type
					variadic := false
					if ellipsis, ok := paramType.(*ast.Ellipsis); ok {
						variadic = true
						paramType = ellipsis.Elt
					}

					// Detect if the parameter is a function type
					if _, isFunc := paramType.(*ast.FuncType); isFunc {
						hasFuncParam = true
					}

					typeStr := "string"
					castType := ""

					// Parse base identifiers (int, string, bool)
					if ident, ok := paramType.(*ast.Ident); ok {
						if variadic && ident.Name == "any" {
							// A UI value cannot represent arbitrary Go values. Expose one
							// string value instead; it is assignable to ...any as one element.
							typeStr = "string"
						} else if compositeTypes[ident.Name] {
							hasUnsupportedParam = true
						} else {
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
								// Prepend the parsed package name to the local identifier so main can see it
								typeStr = "int"
								castType = fmt.Sprintf("%s.%s", parsedPackageName, ident.Name)
								localNeedsStrconv = true
							}
						}
						// Parse pointer expressions (*bool)
					} else if _, isEmptyInterface := paramType.(*ast.InterfaceType); variadic && isEmptyInterface {
						// Support the older equivalent of ...any with the same single-string fallback.
						typeStr = "string"
					} else if starExpr, ok := paramType.(*ast.StarExpr); ok {
						if ident, ok := starExpr.X.(*ast.Ident); ok {
							if ident.Name == "bool" {
								typeStr = "*bool"
								localNeedsStrconv = true
							} else {
								hasUnsupportedParam = true
							}
						} else {
							hasUnsupportedParam = true
						}
						// Parse selector expressions (e.g., logger.LogLevel)
					} else if selExpr, ok := paramType.(*ast.SelectorExpr); ok {
						if xIdent, ok := selExpr.X.(*ast.Ident); ok {
							typeStr = "int"
							castType = fmt.Sprintf("%s.%s", xIdent.Name, selExpr.Sel.Name)
							localNeedsStrconv = true
						} else {
							hasUnsupportedParam = true
						}
					} else {
						hasUnsupportedParam = true
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

			// Skip this function if it has unsupported params or if it has a func() param and we are not explicitly including them
			if hasUnsupportedParam || (hasFuncParam && !includeFuncParams) {
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

			// Duplicate endpoint name guard: two public functions from different packages that
			// share the same bash_name would produce colliding //export and CLI switch entries.
			qualifiedName := fmt.Sprintf("%s.%s", parsedPackageName, goName)
			if firstSeen, collision := seenBashNames[bashName]; collision {
				return nil, nil, false, fmt.Errorf(
					"duplicate endpoint name %q: %s and %s both map to the same CLI/export name — rename one of the exported functions to resolve the collision",
					bashName, firstSeen, qualifiedName,
				)
			}
			seenBashNames[bashName] = qualifiedName

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
	// Map iteration order is randomized per run — sort so generated artifacts
	// (bridge files, JSON package listings) stay byte-stable across regenerations.
	sort.Strings(dynamicImports)

	return dynamicImports, functions, needsStrconv, nil
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
			logger.Error(lvl, "failed to parse excludePackages as CSV: %v", err)
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
		// Resolve relative paths from the process CWD, not from targetDir.
		// Joining a caller-relative path (e.g. "../repo/project/bridge.go")
		// onto targetDir produces a doubled path when the caller expressed
		// the destination relative to their own working directory.
		outputPath, _ = filepath.Abs(outputPath)
	}

	absOutput, err := filepath.Abs(outputPath)
	if err != nil {
		return "", "", fmt.Errorf("failed to canonicalize output path: %w", err)
	}

	// Enforce: the bridge file must live inside the target project directory —
	// unlike the compiled binary, where it sits matters because the module
	// build consumes it from there.
	if !strings.HasPrefix(filepath.Clean(absOutput), filepath.Clean(targetDir)+string(filepath.Separator)) {
		return "", "", fmt.Errorf("output bridge file %q must be located inside the target project directory %q", absOutput, targetDir)
	}

	absOutputAndroid := strings.TrimSuffix(absOutput, ".go") + "_android.go"

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
			if !d.IsDir() && filepath.Ext(absPath) == ".go" && absPath != absOutput && absPath != absOutputAndroid {
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
	dynamicImports, functions, needsStrconv, err := extractMetadata(targetDir, moduleBaseName, targetFiles, excludeMap, lvl, includeFuncParams)
	if err != nil {
		return "", "", err
	}

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

	// Compose the full import block — fixed runtime imports plus the dynamic
	// package imports — as a single sorted set so the generated bridge is
	// byte-stable across regenerations and matches what gofmt would leave.
	bridgeImports := append([]string{"fmt", "os"}, dynamicImports...)
	if needsStrconv {
		bridgeImports = append(bridgeImports, "strconv")
	}
	sort.Strings(bridgeImports)

	wrapperData := TemplateWrapper{
		Imports:   bridgeImports,
		Functions: functions,
	}

	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, wrapperData); err != nil {
		return "", "", fmt.Errorf("template write error: %w", err)
	}

	// Normalize to canonical gofmt formatting (go/format is the engine behind
	// the gofmt command) so the generated bridge never trips `gofmt -l` —
	// template range/if actions otherwise leave stray blank lines inside
	// function bodies. If the rendered source somehow fails to parse, fall
	// back to the raw template output rather than failing generation over
	// formatting.
	content := rendered.Bytes()
	if formatted, fmtErr := format.Source(content); fmtErr == nil {
		content = formatted
	} else {
		logger.Warn(lvl, "generated bridge is not parseable for gofmt normalization (%v); writing raw template output", fmtErr)
	}

	if _, err := outputFile.Write(content); err != nil {
		return "", "", fmt.Errorf("failed writing bridge file: %w", err)
	}

	logger.Info(lvl, "  • Successfully generated bridge file %s\n", absOutput)

	// Also generate Android companion bridge file without CGO exports
	tmplAndroid, err := template.New("android_bridge").Funcs(funcMap).Parse(androidBridgeTemplate)
	if err == nil {
		var renderedAndroid bytes.Buffer
		if err := tmplAndroid.Execute(&renderedAndroid, wrapperData); err == nil {
			contentAndroid := renderedAndroid.Bytes()
			if formatted, fmtErr := format.Source(contentAndroid); fmtErr == nil {
				contentAndroid = formatted
			}
			if err := os.WriteFile(absOutputAndroid, contentAndroid, 0644); err == nil {
				logger.Info(lvl, "  • Successfully generated Android companion bridge file %s\n", absOutputAndroid)
			}
		}
	}

	return targetDir, absOutput, nil
}

func CompileLocal(targetDir, absBridge, binaryName string, logLevelStr string, targetBuilds ...string) error {
	lvl := logger.ParseLogLevel(logLevelStr)
	buildSelection := ""
	if len(targetBuilds) > 0 {
		buildSelection = targetBuilds[0]
	}
	architectures, err := internal_utils.ParseBuildArchitectures(buildSelection)
	if err != nil {
		return err
	}
	selected := make(map[string]bool, len(architectures))
	for _, architecture := range architectures {
		selected[architecture] = true
	}

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

	// Resolve the binary destination. Bare filenames anchor to the project
	// directory; relative paths carrying directory components resolve against
	// the process CWD. The binary is a standalone deliverable — it may land
	// anywhere, so no project containment is enforced. Resolving to an
	// absolute path up front also keeps `go build -o` correct regardless of
	// the cmd.Dir working directory used below.
	binaryName, err = internal_utils.ResolveBinaryOutputPath(binaryName, targetDir)
	if err != nil {
		return err
	}

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
		GOARCH     string
		ArchSuffix string
		CC         string
	}{
		{"amd64", "linux_amd64", "gcc"},                     // Standard x86_64 compiler
		{"arm64", "linux_aarch64", "aarch64-linux-gnu-gcc"}, // ARM64 cross-compiler
	}
	hostArch := runtime.GOARCH
	if hostArch == "arm64" {
		targets[0].CC = "x86_64-linux-gnu-gcc"
		targets[1].CC = "gcc"
	} else {
		targets[0].CC = "gcc"
		targets[1].CC = "aarch64-linux-gnu-gcc"
	}

	for _, target := range targets {
		if !selected[target.ArchSuffix] {
			continue
		}
		logger.Info(lvl, "Building for architecture: %s...", target.ArchSuffix)

		cliBinName := fmt.Sprintf("%s_%s", binaryName, target.ArchSuffix)
		libBinName := fmt.Sprintf("%s_%s.so", binaryName, target.ArchSuffix)

		// 1. Build the Standalone Shell Executable
		cmdCLI := exec.Command("go", "build", "-o", cliBinName, ".")
		cmdCLI.Dir = targetDir
		cmdCLI.Env = append(os.Environ(),
			"CGO_ENABLED=1", // Explicitly enable CGO for the CLI
			"GOARCH="+target.GOARCH,
			"GOOS=linux",
			"CC="+target.CC, // Supply the cross-compiler mapping
		)

		var errCLI strings.Builder
		cmdCLI.Stdout = os.Stdout
		cmdCLI.Stderr = &errCLI

		if err := cmdCLI.Run(); err != nil {
			return fmt.Errorf("CLI binary build failure for %s:\n%s\ninternal error: %w", target.ArchSuffix, errCLI.String(), err)
		}
		_ = os.Chmod(cliBinName, 0755) // cliBinName is absolute (resolved above)

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
			return fmt.Errorf("C++ shared library build failure for %s:\n%s\ninternal error: %w", target.ArchSuffix, errLib.String(), err)
		}

		logger.Info(lvl, "  • Built: %s", cliBinName)
		logger.Info(lvl, "  • Built: %s", libBinName)
		logger.Info(lvl, "")
	}

	// 3. Build for Android Termux: android_aarch64 (CLI executable only)
	if !selected["android_aarch64"] {
		logger.Info(lvl, "Skipping unrequested architecture: android_aarch64")
		logger.Info(lvl, "Successfully compiled selected Go deliverables in %s.", targetDir)
		return nil
	}
	logger.Info(lvl, "Building for architecture: android_aarch64...")
	cliBinAndroid := fmt.Sprintf("%s_android_aarch64", binaryName)

	// Ensure an Android-compatible bridge exists in targetDir if absBridge contains CGO
	if absBridge != "" {
		androidBridgeFile := strings.TrimSuffix(absBridge, ".go") + "_android.go"
		if _, err := os.Stat(androidBridgeFile); os.IsNotExist(err) {
			if bridgeBytes, readErr := os.ReadFile(absBridge); readErr == nil && bytes.Contains(bridgeBytes, []byte(`import "C"`)) {
				contentStr := string(bridgeBytes)
				cliMarker := "// AUTOMATICALLY GENERATED BASH CLI ROUTER"
				if idx := strings.Index(contentStr, cliMarker); idx != -1 {
					var importsBlock string
					if impStart := strings.Index(contentStr, "import ("); impStart != -1 {
						if impEnd := strings.Index(contentStr[impStart:], ")"); impEnd != -1 {
							importsBlock = contentStr[impStart : impStart+impEnd+1]
						}
					}
					var androidSrc bytes.Buffer
					androidSrc.WriteString("//go:build android\n\npackage main\n\n")
					if importsBlock != "" {
						androidSrc.WriteString(importsBlock)
						androidSrc.WriteString("\n\n")
					}
					androidSrc.WriteString(contentStr[idx:])
					if formattedAndroid, fmtErr := format.Source(androidSrc.Bytes()); fmtErr == nil {
						_ = os.WriteFile(androidBridgeFile, formattedAndroid, 0644)
					}
				}
			}
		}
	}

	cmdAndroid := exec.Command("go", "build", "-o", cliBinAndroid, ".")
	cmdAndroid.Dir = targetDir
	cmdAndroid.Env = append(os.Environ(),
		"CGO_ENABLED=0",
		"GOARCH=arm64",
		"GOOS=android",
	)

	var errAndroid strings.Builder
	cmdAndroid.Stdout = os.Stdout
	cmdAndroid.Stderr = &errAndroid

	if err := cmdAndroid.Run(); err != nil {
		return fmt.Errorf("CLI binary build failure for android_aarch64:\n%s\ninternal error: %w", errAndroid.String(), err)
	}
	_ = os.Chmod(cliBinAndroid, 0755)

	logger.Info(lvl, "  • Built: %s", cliBinAndroid)
	logger.Info(lvl, "")

	logger.Info(lvl, "Successfully compiled unified pipeline deliverables in %s.", targetDir)
	return nil
}
