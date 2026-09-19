package config_gen

import (
	"bufio"
	"embed"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"guify_my_script/internal_utils"
	"guify_my_script/logger"
)

//go:embed guify_my_script_config_schema.json
var schemaFile embed.FS

// Helper to conditionally force literal 'null' in JSON for string values
func nullJSON() *json.RawMessage {
	raw := json.RawMessage("null")
	return &raw
}

// Helper to force an empty array '[]' in JSON to visually hint the type
func emptyArrayJSON() *json.RawMessage {
	raw := json.RawMessage("[]")
	return &raw
}

// Helper to emit a JSON string where the schema expects a stringified value.
func stringJSON(value string) *json.RawMessage {
	raw := json.RawMessage(fmt.Sprintf("%q", value))
	return &raw
}

// JsonEndpoints maps to the new nested endpoint structure
type JsonEndpoints struct {
	ArrayEndpoint       *json.RawMessage `json:"array_endpoint"`
	PathToArray         *json.RawMessage `json:"path_to_array"`
	GetDefaultsEndpoint *json.RawMessage `json:"get_defaults_endpoint"`
	PathToDefault       *json.RawMessage `json:"path_to_default"`
	GetEnabledEndpoint  *json.RawMessage `json:"get_enabled_endpoint"`
	PathToEnabled       *json.RawMessage `json:"path_to_enabled"`
}

// UIField maps directly to your C++ application UI schema rules
type UIField struct {
	Name     string        `json:"name"`
	Type     string        `json:"type"`
	DataType string        `json:"data_type"`
	ArgName  string        `json:"arg_name,omitempty"`
	Options  []interface{} `json:"options,omitempty"`
	Default  any           `json:"default"`
	// Bounds are meaningful only for MCB, MFL, TIME, DATE, and NUM fields.
	// Omitting nil values prevents unrelated field types from receiving null bounds.
	Minimum               *json.RawMessage `json:"minimum,omitempty"`
	Maximum               *json.RawMessage `json:"maximum,omitempty"`
	DecimalPlaces         *string          `json:"decimal_places,omitempty"`
	AdvancedMenu          bool             `json:"advanced_menu"`
	Enabled               bool             `json:"enabled"`
	AutoHeight            bool             `json:"auto_height,omitempty"`
	MinHeight             int              `json:"min_height,omitempty"`
	FileTypes             []string         `json:"file_types,omitempty"`
	JsonEndpoints         JsonEndpoints    `json:"json_endpoints"`
	RefreshFieldsOnChange *json.RawMessage `json:"refresh_fields_on_change"`
}

// UIAction matches individual discovered subroutines stripped of main() mechanics
type UIAction struct {
	Label                     string           `json:"label"`
	Func                      string           `json:"func"`
	ExecutionMode             string           `json:"execution_mode"`
	Mode                      string           `json:"mode,omitempty"`
	RefreshFieldsOnTabOpen    *json.RawMessage `json:"refresh_fields_on_tab_open"`
	RefreshFieldsOnCompletion *json.RawMessage `json:"refresh_fields_on_completion"`
	QuickLaunch               bool             `json:"quick_launch"`
	Width                     int              `json:"width"`
	Sudo                      bool             `json:"sudo"`
	ExecLabel                 string           `json:"exec_label"`
	ExecInProgressLabel       string           `json:"exec_in_progress_label"`
	StopLabel                 string           `json:"stop_label"`
	AdvancedMenu              bool             `json:"advanced_menu"`
	AdvancedMenuTabTitle      string           `json:"advanced_menu_tab_title"`
	InvokeAsObject            bool             `json:"invoke_as_object,omitempty"`
	Fields                    []UIField        `json:"fields"`
}

func (a *UIAction) UnmarshalJSON(data []byte) error {
	type Alias UIAction
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	if a.ExecutionMode == "" && a.Mode != "" {
		a.ExecutionMode = a.Mode
	}
	return nil
}

// UIConfig sets up high-fidelity platform layout values
type UIConfig struct {
	Title                    string  `json:"title"`
	Icon                     string  `json:"icon"`
	UUID                     string  `json:"uuid"`
	ScriptAlias              string  `json:"script_alias"`
	SourceScriptLinuxAMD64   string  `json:"source_script_linux_amd64"`
	SourceScriptLinuxAArch64 string  `json:"source_script_linux_aarch64"`
	SourceScriptAMD64        string  `json:"source_script_amd64,omitempty"`
	SourceScriptARM64        string  `json:"source_script_arm64,omitempty"`
	SourceType               string  `json:"source_type"`
	Startup                  string  `json:"startup"`
	TrayOnMinimize           bool    `json:"tray_on_minimize"`
	InstanceMode             string  `json:"instance_mode"`
	ActionOnStartup          *string `json:"action_on_startup"`
	DefaultTabOnStartup      *string `json:"default_tab_on_startup"`
	CloseAfterStartupAction  bool    `json:"close_after_startup_action"`
	Sudo                     bool    `json:"sudo"`
	Theme                    struct {
		Color  string `json:"color"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	} `json:"theme"`
}

func (u *UIConfig) UnmarshalJSON(data []byte) error {
	type Alias UIConfig
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(u),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}
	if u.SourceScriptLinuxAMD64 == "" && u.SourceScriptAMD64 != "" {
		u.SourceScriptLinuxAMD64 = u.SourceScriptAMD64
	}
	if u.SourceScriptLinuxAArch64 == "" && u.SourceScriptARM64 != "" {
		u.SourceScriptLinuxAArch64 = u.SourceScriptARM64
	}
	if u.SourceScriptAMD64 == "" && u.SourceScriptLinuxAMD64 != "" {
		u.SourceScriptAMD64 = u.SourceScriptLinuxAMD64
	}
	if u.SourceScriptARM64 == "" && u.SourceScriptLinuxAArch64 != "" {
		u.SourceScriptARM64 = u.SourceScriptLinuxAArch64
	}
	return nil
}

// MainMenuConfig defines the generated application's landing tab and optional
// copy shown above its actions. Nil labels intentionally serialize as null.
type MainMenuConfig struct {
	TabLabel            string  `json:"tab_label"`
	WelcomeHeadingLabel *string `json:"welcome_heading_label"`
	InfoLabel           *string `json:"info_label"`
}

// DeployDependency exposes all supported package-manager dependency lists.
type DeployDependency struct {
	Pacman []string `json:"pacman"`
	Arch   []string `json:"arch"`
	Apt    []string `json:"apt"`
	Debian []string `json:"debian"`
	Dnf    []string `json:"dnf"`
	Fedora []string `json:"fedora"`
}

// DeployConfig holds metadata for packaging and deployment
type DeployConfig struct {
	Description  string             `json:"description"`
	Version      string             `json:"version"`
	ReleaseNum   string             `json:"release_number"`
	Summary      string             `json:"summary"`
	License      string             `json:"license"`
	Maintainer   string             `json:"maintainer"`
	SupportEmail string             `json:"support_email"`
	Dependencies []DeployDependency `json:"dependencies"`
}

// RootGUIConfig encapsulates the complete deliverable layout to be parsed by C++
type RootGUIConfig struct {
	Schema   string         `json:"$schema,omitempty"`
	Config   UIConfig       `json:"config"`
	MainMenu MainMenuConfig `json:"main_menu"`
	Deploy   DeployConfig   `json:"deploy"`
	Actions  []UIAction     `json:"actions"`
}

var uiFieldTypes = []string{"TXT", "DIR", "FL", "MFL", "FS", "CB", "CK", "FD", "MCB", "LTXT", "TIME", "DATE", "NUM"}
var exampleFieldNames = []string{
	"Text Input Example",
	"Directory Picker Example",
	"File Picker Example",
	"Multi-File Picker Example",
	"Save File Picker Example",
	"Combo Box Example",
	"Checkbox Example",
	"File or Directory Picker Example",
	"Multi-Combo Box Example",
	"Long Text Example",
	"Time Picker Example",
	"Date Picker Example",
	"Number Input Example",
}

func GenerateGUIConfigWithPrompt(inputPath, outputPath, binaryName, logLevel string, autoOpen bool) error {
	lvl := logger.ParseLogLevel(logLevel)

	if outputPath == "" {
		outputPath = "config.json"
	}
	overwriteFlag := false
	if _, err := os.Stat(outputPath); err == nil {
		fmt.Printf("File '%s' already exists. Overwrite? (y/N): ", outputPath)
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed reading shell confirmation: %w", err)
		}
		confirm := strings.TrimSpace(strings.ToLower(input))
		if confirm == "y" || confirm == "yes" {
			overwriteFlag = true
		} else {
			logger.Info(lvl, "Operation aborted by user.")
			return nil
		}
	}
	return GenerateGUIConfig(inputPath, outputPath, binaryName, overwriteFlag, logLevel, autoOpen)
}

func GenerateGUIConfig(inputPath, outputPath, binaryName string, overwrite bool, logLevel string, autoOpen bool) error {
	lvl := logger.ParseLogLevel(logLevel)

	if outputPath == "" {
		outputPath = "config.json"
	}

	logger.Info(lvl, "Generating JSON template for GUI...")

	if !overwrite {
		if _, err := os.Stat(outputPath); err == nil {
			logger.Warn(lvl, "Safety abort: target file '%s' already exists. Skipping JSON generation.\n", outputPath)
			if autoOpen {
				openFileInEditor(outputPath, lvl)
			} else {
				logger.Debug(lvl, "autoOpen is: False")
			}
			return nil
		}
	}

	absInput, _ := filepath.Abs(inputPath)
	inputExt := filepath.Ext(absInput)
	ext := strings.ToLower(inputExt)
	sourceType := "go"

	var targetName string
	var sourceScript string
	var scriptAlias string
	var sourceScriptLinuxAMD64 string
	var sourceScriptLinuxAArch64 string
	if strings.TrimSpace(binaryName) != "" {
		absBinary, _ := filepath.Abs(binaryName)
		targetName = filepath.Base(absBinary)
		sourceScript = absBinary
	} else {
		inputBase := filepath.Base(absInput)
		targetName = strings.TrimSuffix(inputBase, inputExt)
		sourceScript = absInput
	}

	var (
		actions []UIAction
		err     error
	)
	if ext == ".go" {
		actions, err = parseGoFile(absInput)
		if err != nil {
			return fmt.Errorf("failed to parse Go source %q: %w", absInput, err)
		}
		sourceScriptBase := strings.TrimSuffix(sourceScript, inputExt)
		scriptAlias = filepath.Base(sourceScriptBase)
		sourceScriptLinuxAMD64 = sourceScriptBase + "_linux_amd64"
		sourceScriptLinuxAArch64 = sourceScriptBase + "_linux_aarch64"
	} else if ext == ".rs" {
		actions, err = parseRustFile(absInput)
		if err != nil {
			return fmt.Errorf("failed to parse Rust source %q: %w", absInput, err)
		}
		sourceType = "rust"
		sourceScriptBase := strings.TrimSuffix(sourceScript, inputExt)
		scriptAlias = filepath.Base(sourceScriptBase)
		sourceScriptLinuxAMD64 = sourceScriptBase + "_linux_amd64"
		sourceScriptLinuxAArch64 = sourceScriptBase + "_linux_aarch64"
	} else if ext == ".py" {
		actions, err = parsePythonFile(absInput)
		if err != nil {
			return fmt.Errorf("failed to parse Python module %q: %w", absInput, err)
		}
		sourceType = "python"
		// Python modules are source files, even when binaryName customises the
		// generated application's name.
		scriptAlias = filepath.Base(absInput)
		sourceScriptLinuxAMD64 = absInput
		sourceScriptLinuxAArch64 = absInput
	} else if ext == ".js" || ext == ".mjs" {
		actions, err = parseJavaScriptFile(absInput)
		if err != nil {
			return fmt.Errorf("failed to parse JavaScript module %q: %w", absInput, err)
		}
		sourceType = "javascript"
		scriptAlias = filepath.Base(absInput)
		sourceScriptLinuxAMD64 = absInput
		sourceScriptLinuxAArch64 = absInput
	} else if ext == ".psm1" {
		actions, err = parsePowerShellFile(absInput)
		if err != nil {
			return fmt.Errorf("failed to parse PowerShell module %q: %w", absInput, err)
		}
		sourceType = "powershell"
		scriptAlias = filepath.Base(absInput)
		sourceScriptLinuxAMD64 = absInput
		sourceScriptLinuxAArch64 = absInput
	} else {
		actions, err = parseBashFile(absInput)
		if err != nil {
			return fmt.Errorf("failed to parse Bash source %q: %w", absInput, err)
		}
		sourceType = "bash"
		scriptAlias = filepath.Base(sourceScript)
		sourceScriptLinuxAMD64 = sourceScript
		sourceScriptLinuxAArch64 = sourceScript
	}

	guiConfig := RootGUIConfig{}

	// Add $schema pointing to a relative config schema path
	guiConfig.Schema = "guify_my_script_config_schema.json"

	// Config Section
	guiConfig.Config.Title = internal_utils.ToTitleCase(targetName)
	guiConfig.Config.Icon = "/usr/share/icons/hicolor/256x256/apps/com.nixtrillo.guifymyscript.png"
	guiConfig.Config.UUID = fmt.Sprintf("com.%s.%s", os.Getenv("USER"), toUuidCase(targetName))
	guiConfig.Config.ScriptAlias = scriptAlias
	guiConfig.Config.SourceScriptLinuxAMD64 = sourceScriptLinuxAMD64
	guiConfig.Config.SourceScriptLinuxAArch64 = sourceScriptLinuxAArch64
	guiConfig.Config.SourceType = sourceType
	guiConfig.Config.Startup = "tray"
	guiConfig.Config.TrayOnMinimize = true
	guiConfig.Config.InstanceMode = "replace"
	guiConfig.Config.ActionOnStartup = nil
	guiConfig.Config.DefaultTabOnStartup = nil
	guiConfig.Config.CloseAfterStartupAction = false
	guiConfig.Config.Sudo = false
	guiConfig.Config.Theme.Color = "auto"
	guiConfig.Config.Theme.Width = 600
	guiConfig.Config.Theme.Height = 400

	// Main Menu Section
	guiConfig.MainMenu.TabLabel = "Main Menu"
	guiConfig.MainMenu.WelcomeHeadingLabel = nil
	guiConfig.MainMenu.InfoLabel = nil

	// Deploy Section
	guiConfig.Deploy.Description = "Generated Qt GUI package staged by guify_my_script."
	guiConfig.Deploy.Version = "0.0.1"
	guiConfig.Deploy.ReleaseNum = "1"
	guiConfig.Deploy.Summary = fmt.Sprintf("A GUI frontend for executing commands for %s", targetName)
	guiConfig.Deploy.License = "MIT"
	guiConfig.Deploy.Maintainer = "gms"
	guiConfig.Deploy.SupportEmail = "maintainer@example.invalid"
	guiConfig.Deploy.Dependencies = []DeployDependency{{
		Pacman: []string{},
		Arch:   []string{},
		Apt:    []string{},
		Debian: []string{},
		Dnf:    []string{},
		Fedora: []string{},
	}}

	// Actions Section
	guiConfig.Actions = actions
	if len(actions) > 0 {
		defaultTab := actions[0].Label
		guiConfig.Config.DefaultTabOnStartup = &defaultTab
	}

	jsonBytes, _ := json.MarshalIndent(guiConfig, "", "  ")
	os.WriteFile(outputPath, jsonBytes, 0644)

	logger.Info(lvl, "  • Successfully generated JSON: %s", outputPath)

	if autoOpen {
		openFileInEditor(outputPath, lvl)
	} else {
		logger.Debug(lvl, "autoOpen is: False")
	}

	// Copy the embedded schema to the output directory
	if err := copySchemaToOutputDir(outputPath); err != nil {
		return fmt.Errorf("failed to copy schema to output directory: %w", err)
	}

	logger.Info(lvl, "  • Successfully copied JSON schema: %s\n", filepath.Join(filepath.Dir(outputPath), "guify_my_script_config_schema.json"))

	return nil
}

func copySchemaToOutputDir(outputPath string) error {
	schemaData, err := schemaFile.ReadFile("guify_my_script_config_schema.json")
	if err != nil {
		return fmt.Errorf("failed to read embedded schema: %w", err)
	}

	schemaPath := filepath.Join(filepath.Dir(outputPath), "guify_my_script_config_schema.json")
	if err := os.WriteFile(schemaPath, schemaData, 0644); err != nil {
		return fmt.Errorf("failed to write schema to output directory: %w", err)
	}

	return nil
}

func parseGoFile(path string) ([]UIAction, error) {
	var actions []UIAction
	fset := token.NewFileSet()
	node, _ := parser.ParseFile(fset, path, nil, parser.ParseComments)

	for _, decl := range node.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || fn.Name.Name == "main" || !ast.IsExported(fn.Name.Name) {
			continue
		}

		bashName := strings.ToLower(strings.Join(internal_utils.SplitCamelCase(fn.Name.Name), "_"))
		if strings.HasPrefix(bashName, "go_") {
			bashName = strings.TrimPrefix(bashName, "go_")
		}

		var paramNames []string
		if fn.Type.Params != nil {
			for _, fieldList := range fn.Type.Params.List {
				for _, name := range fieldList.Names {
					paramNames = append(paramNames, name.Name)
				}
			}
		}

		fields := []UIField{}
		total := len(paramNames)

		for i := 0; i < total; i++ {
			label := internal_utils.ToTitleCase(paramNames[i])
			fields = append(fields, createField(label, i))
		}
		actions = append(actions, UIAction{
			Label:                     internal_utils.ToTitleCase(bashName),
			Func:                      bashName,
			ExecutionMode:             "GUI",
			RefreshFieldsOnTabOpen:    emptyArrayJSON(),
			RefreshFieldsOnCompletion: emptyArrayJSON(),
			QuickLaunch:               false,
			Width:                     600,
			Sudo:                      false,
			ExecLabel:                 fmt.Sprintf("Execute %s", internal_utils.ToTitleCase(bashName)),
			ExecInProgressLabel:       fmt.Sprintf("Executing %s...", internal_utils.ToTitleCase(bashName)),
			StopLabel:                 "Stop",
			AdvancedMenu:              false,
			AdvancedMenuTabTitle:      "",
			Fields:                    fields,
		})
	}
	return actions, nil
}

func parseBashFile(path string) ([]UIAction, error) {
	var actions []UIAction
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	regex := regexp.MustCompile(`^[[:space:]]*(function[[:space:]]+)?([a-zA-Z_][a-zA-Z0-9_]*)[[:space:]]*\(\)`)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		matches := regex.FindStringSubmatch(scanner.Text())
		if len(matches) >= 3 {
			name := matches[2]
			if name == "main" || name == "usage" || name == "cleanup" {
				continue
			}
			if strings.HasPrefix(name, "go_") {
				name = strings.TrimPrefix(name, "go_")
			}

			fields := []UIField{}
			// Bash functions don't have parameter information from the simple regex
			// Set to 0 fields since we can't determine parameters

			actions = append(actions, UIAction{
				Label:                     internal_utils.ToTitleCase(name),
				Func:                      name,
				ExecutionMode:             "GUI",
				RefreshFieldsOnTabOpen:    emptyArrayJSON(),
				RefreshFieldsOnCompletion: emptyArrayJSON(),
				QuickLaunch:               false,
				Width:                     600,
				Sudo:                      false,
				ExecLabel:                 fmt.Sprintf("Execute %s", internal_utils.ToTitleCase(name)),
				ExecInProgressLabel:       fmt.Sprintf("Executing %s...", internal_utils.ToTitleCase(name)),
				StopLabel:                 "Stop",
				AdvancedMenu:              false,
				AdvancedMenuTabTitle:      "",
				Fields:                    fields,
			})
		}
	}
	return actions, scanner.Err()
}

func createField(displayName string, index int) UIField {
	fieldType := uiFieldTypes[index%len(uiFieldTypes)]

	// Unconditionally apply the empty endpoints object so it is visible in the JSON
	field := UIField{
		Name:         internal_utils.ToTitleCase(displayName),
		Type:         fieldType,
		DataType:     "string",
		Default:      "",
		AdvancedMenu: false,
		Enabled:      true,
		JsonEndpoints: JsonEndpoints{
			ArrayEndpoint:       emptyArrayJSON(),
			PathToArray:         nullJSON(),
			GetDefaultsEndpoint: emptyArrayJSON(),
			PathToDefault:       nullJSON(),
			GetEnabledEndpoint:  emptyArrayJSON(),
			PathToEnabled:       nullJSON(),
		},
		RefreshFieldsOnChange: emptyArrayJSON(),
	}

	switch fieldType {
	case "CB":
		field.Options = []interface{}{"Option1", "Option2", "Option3"}
		field.Default = "Option1"
	case "MCB":
		field.Options = []interface{}{"Option1", "Option2", "Option3"}
		field.Default = []string{"Option2", "Option3"}
		// "0" means optional/unlimited. Keeping both fields in the template
		// makes it straightforward to change the selection bounds per field.
		field.Minimum = stringJSON("0")
		field.Maximum = stringJSON("0")
	case "DIR":
		field.Default = "/home"
	case "TXT":
		field.Default = "Initial Value"
	case "CK":
		field.Default = "0"
	case "FD", "FL", "FS":
		field.Default = ""
		field.FileTypes = []string{"txt"}
	case "MFL":
		field.Default = ""
		field.FileTypes = []string{"txt"}
		field.Minimum = nullJSON()
		field.Maximum = nullJSON()
	case "LTXT":
		field.Default = "Initial Long Text Value"
		field.AutoHeight = true
		field.MinHeight = 240
	case "TIME":
		field.Default = "12:00:00"
		field.Minimum = nullJSON()
		field.Maximum = nullJSON()
	case "DATE":
		field.Default = "2026-01-01"
		field.Minimum = nullJSON()
		field.Maximum = nullJSON()
	case "NUM":
		field.Default = "0"
		field.Minimum = nullJSON()
		field.Maximum = nullJSON()
		zeroStr := "0"
		field.DecimalPlaces = &zeroStr
	}
	return field
}

func toUuidCase(input string) string {
	replacer := strings.NewReplacer("_", "", "-", "", " ", "")
	unspaced := replacer.Replace(input)
	return strings.ToLower(unspaced)
}

func getNativeOpenCommand(path string) *exec.Cmd {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("cmd", "/c", "start", "", path)
	case "darwin":
		return exec.Command("open", path)
	default: // "linux", "freebsd", "openbsd", "netbsd" etc.
		return exec.Command("xdg-open", path)
	}
}

func openFileInEditor(outputPath string, lvl logger.LogLevel) {
	logger.Debug(lvl, "autoOpen is: True")
	editor := os.Getenv("EDITOR")
	logger.Debug(lvl, "editor is: %s", editor)
	if editor == "" {
		// Try to find the system default handler for JSON (like xdg-open on Linux, open on macOS, start on Windows)
		// This handles Kate, VSCode, or whatever is assigned to .json files.
		logger.Debug(lvl, "EDITOR env var not set, attempting native desktop open...")
		var cmd *exec.Cmd
		// Check runtime
		// If on Linux/BSD, xdg-open works great and respects the system default (e.g. org.kde.kate.desktop).
		// If on macOS, "open" launches the default app.
		// If on Windows, "rundll32.exe url.dll,FileProtocolHandler" or start command launches the default.
		// We can also fallback to searching common visual/CLI editors if xdg-open fails.
		cmd = getNativeOpenCommand(outputPath)
		if err := cmd.Start(); err != nil {
			// Fallback to common terminal editors
			logger.Debug(lvl, "Native open failed, falling back to terminal editor search")
			for _, fallbackEditor := range []string{"nano", "vim", "vi"} {
				if path, err := exec.LookPath(fallbackEditor); err == nil {
					editor = path
					break
				}
			}
			if editor != "" {
				cmd = exec.Command(editor, outputPath)
				_ = cmd.Start()
			} else {
				logger.Warn(lvl, "Failed to find any viable editor to open JSON config.")
			}
		}
	} else {
		cmd := exec.Command(editor, outputPath)
		if err := cmd.Start(); err != nil {
			logger.Warn(lvl, "Failed to open JSON file in $EDITOR: %w", err)
		}
	}
}

func parsePythonFile(path string) ([]UIAction, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := string(data)
	var actions []UIAction

	// Match module-level def statements.  Restricting this to column zero avoids
	// accidentally exposing class methods or nested implementation helpers.
	// Matches `def func_name(params) -> return_type:` or `async def func_name(params):`
	re := regexp.MustCompile(`(?m)^(?:async\s+)?def\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\(([\s\S]*?)\)\s*(?:->\s*[^:]+)?\s*:`)
	matches := re.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		name := match[1]
		paramsRaw := match[2]

		// Skip internal, dunder, or standard utility functions
		if strings.HasPrefix(name, "_") || name == "main" || name == "usage" || name == "cleanup" {
			continue
		}

		params := splitPythonParams(paramsRaw)
		var fields []UIField
		for i, param := range params {
			f := createPythonField(param, i)
			if f.Name != "" {
				fields = append(fields, f)
			}
		}

		actions = append(actions, UIAction{
			Label:                     internal_utils.ToTitleCase(name),
			Func:                      name,
			ExecutionMode:             "GUI",
			RefreshFieldsOnTabOpen:    emptyArrayJSON(),
			RefreshFieldsOnCompletion: emptyArrayJSON(),
			QuickLaunch:               false,
			Width:                     600,
			Sudo:                      false,
			ExecLabel:                 fmt.Sprintf("Execute %s", internal_utils.ToTitleCase(name)),
			ExecInProgressLabel:       fmt.Sprintf("Executing %s...", internal_utils.ToTitleCase(name)),
			StopLabel:                 "Stop",
			AdvancedMenu:              false,
			AdvancedMenuTabTitle:      "",
			Fields:                    fields,
		})
	}

	return actions, nil
}

func splitPythonParams(raw string) []string {
	var params []string
	var current strings.Builder
	depthParen := 0
	depthBracket := 0
	depthBrace := 0
	inSingleQuote := false
	inDoubleQuote := false
	escaped := false

	for _, r := range raw {
		if escaped {
			current.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' {
			current.WriteRune(r)
			escaped = true
			continue
		}
		if inSingleQuote {
			current.WriteRune(r)
			if r == '\'' {
				inSingleQuote = false
			}
			continue
		}
		if inDoubleQuote {
			current.WriteRune(r)
			if r == '"' {
				inDoubleQuote = false
			}
			continue
		}

		switch r {
		case '\'':
			inSingleQuote = true
			current.WriteRune(r)
		case '"':
			inDoubleQuote = true
			current.WriteRune(r)
		case '(':
			depthParen++
			current.WriteRune(r)
		case ')':
			depthParen--
			current.WriteRune(r)
		case '[':
			depthBracket++
			current.WriteRune(r)
		case ']':
			depthBracket--
			current.WriteRune(r)
		case '{':
			depthBrace++
			current.WriteRune(r)
		case '}':
			depthBrace--
			current.WriteRune(r)
		case ',':
			if depthParen == 0 && depthBracket == 0 && depthBrace == 0 {
				trimmed := strings.TrimSpace(current.String())
				if trimmed != "" {
					params = append(params, trimmed)
				}
				current.Reset()
			} else {
				current.WriteRune(r)
			}
		default:
			current.WriteRune(r)
		}
	}
	trimmed := strings.TrimSpace(current.String())
	if trimmed != "" {
		params = append(params, trimmed)
	}
	return params
}

func createPythonField(param string, index int) UIField {
	param = strings.TrimSpace(param)
	if param == "" || param == "self" || param == "cls" || strings.HasPrefix(param, "*") {
		return UIField{}
	}

	// Remove default value if present e.g. "foo: int = 10" -> "foo: int"
	nameAndType := param
	if idx := strings.Index(nameAndType, "="); idx != -1 {
		nameAndType = strings.TrimSpace(nameAndType[:idx])
	}

	parts := strings.SplitN(nameAndType, ":", 2)
	paramName := strings.TrimSpace(parts[0])
	typeHint := ""
	if len(parts) > 1 {
		typeHint = strings.ToLower(strings.TrimSpace(parts[1]))
	}

	dataType := "string"
	fieldType := "TXT"
	var defaultVal any = "Initial Value"
	var decPlaces *string

	if typeHint == "int" || typeHint == "integer" {
		dataType = "int"
		fieldType = "NUM"
		defaultVal = "0"
		zeroStr := "0"
		decPlaces = &zeroStr
	} else if typeHint == "float" || typeHint == "double" {
		dataType = "float"
		fieldType = "NUM"
		defaultVal = "0.0"
		twoStr := "2"
		decPlaces = &twoStr
	} else if typeHint == "bool" || typeHint == "boolean" {
		dataType = "bool"
		fieldType = "CK"
		defaultVal = "0"
	} else if strings.HasPrefix(typeHint, "list") || strings.HasPrefix(typeHint, "array") {
		dataType = "list"
		fieldType = "TXT"
		defaultVal = ""
	} else if strings.HasPrefix(typeHint, "dict") || strings.HasPrefix(typeHint, "map") {
		dataType = "dict"
		fieldType = "TXT"
		defaultVal = ""
	}

	field := UIField{
		Name:         internal_utils.ToTitleCase(paramName),
		Type:         fieldType,
		DataType:     dataType,
		Default:      defaultVal,
		AdvancedMenu: false,
		Enabled:      true,
		JsonEndpoints: JsonEndpoints{
			ArrayEndpoint:       emptyArrayJSON(),
			PathToArray:         nullJSON(),
			GetDefaultsEndpoint: emptyArrayJSON(),
			PathToDefault:       nullJSON(),
			GetEnabledEndpoint:  emptyArrayJSON(),
			PathToEnabled:       nullJSON(),
		},
		RefreshFieldsOnChange: emptyArrayJSON(),
	}

	if fieldType == "NUM" {
		field.Minimum = nullJSON()
		field.Maximum = nullJSON()
		field.DecimalPlaces = decPlaces
	}

	return field
}
