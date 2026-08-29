package config_gen

import (
	"bufio"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"embed"

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
	Name                  string           `json:"name"`
	Type                  string           `json:"type"`
	Options               []interface{}    `json:"options,omitempty"`
	Default               any              `json:"default"`
	Minimum               *json.RawMessage `json:"minimum,omitempty"`
	Maximum               *json.RawMessage `json:"maximum,omitempty"`
	DecimalPlaces         *string          `json:"decimal_places,omitempty"`
	AdvancedMenu          bool             `json:"advanced_menu"`
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
	Mode                      string           `json:"mode"`
	RefreshFieldsOnTabOpen    *json.RawMessage `json:"refresh_fields_on_tab_open"`
	RefreshFieldsOnCompletion *json.RawMessage `json:"refresh_fields_on_completion"`
	QuickLaunch               bool             `json:"quick_launch"`
	Width                     int              `json:"width"`
	Sudo                      bool             `json:"sudo"`
	ExecLabel                 string           `json:"exec_label"`
	ExecInProgressLabel       string           `json:"exec_in_progress_label"`
	StopLabel                 string           `json:"stop_label"`
	Fields                    []UIField        `json:"fields"`
}

// UIConfig sets up high-fidelity platform layout values
type UIConfig struct {
	Title                   string  `json:"title"`
	Icon                    string  `json:"icon"`
	UUID                    string  `json:"uuid"`
	SourceScript            string  `json:"source_script"`
	SourceType              string  `json:"source_type"`
	Startup                 string  `json:"startup"`
	TrayOnMinimize          bool    `json:"tray_on_minimize"`
	InstanceMode            string  `json:"instance_mode"`
	ActionOnStartup         *string `json:"action_on_startup"`
	DefaultTabOnStartup     *string `json:"default_tab_on_startup"`
	CloseAfterStartupAction bool    `json:"close_after_startup_action"`
	Sudo                    bool    `json:"sudo"`
	Theme                   struct {
		Color  string `json:"color"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	} `json:"theme"`
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
	return GenerateGUIConfig(inputPath, outputPath, binaryName, overwriteFlag, logLevel)
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
			return nil
		}
	}

	absInput, _ := filepath.Abs(inputPath)
	absBinary, _ := filepath.Abs(binaryName)

	var actions []UIAction
	ext := strings.ToLower(filepath.Ext(absInput))
	sourceType := "go"

	if ext == ".go" {
		actions, _ = parseGoFile(absInput)
	} else {
		actions, _ = parseBashFile(absInput)
		sourceType = "bash"
	}

	guiConfig := RootGUIConfig{}
	
	// Add $schema pointing to a relative config schema path
	guiConfig.Schema = "guify_my_script_config_schema.json"

	// Config Section
	guiConfig.Config.Title = internal_utils.ToTitleCase(filepath.Base(absBinary))
	guiConfig.Config.Icon = fmt.Sprintf("/home/%s/Pictures/Icons/gms/com.nixtrillo.gms.png", os.Getenv("USER"))
	guiConfig.Config.UUID = fmt.Sprintf("com.%s.%s", os.Getenv("USER"), toUuidCase(filepath.Base(absBinary)))
	guiConfig.Config.SourceScript = absBinary
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
	guiConfig.Deploy.Summary = fmt.Sprintf("A GUI frontend for executing commands for %s", filepath.Base(absBinary))
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
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vim"
		}
		cmd := exec.Command(editor, outputPath)
		if err := cmd.Start(); err != nil {
			logger.Warn(lvl, "Failed to open JSON file in editor: %w", err)
		}
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
		if !ok || fn.Name.Name == "main" || !ast.IsExported(fn.Name.Name) {
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

		var fields []UIField
		total := len(exampleFieldNames)
		if len(paramNames) > len(exampleFieldNames) {
			total = len(paramNames)
		}

		for i := 0; i < total; i++ {
			label := exampleFieldNames[i%len(exampleFieldNames)]
			if i < len(paramNames) {
				label = internal_utils.ToTitleCase(paramNames[i])
			}
			fields = append(fields, createField(label, i))
		}

		actions = append(actions, UIAction{
			Label:                     internal_utils.ToTitleCase(bashName),
			Func:                      bashName,
			Mode:                      "g",
			RefreshFieldsOnTabOpen:    emptyArrayJSON(),
			RefreshFieldsOnCompletion: emptyArrayJSON(),
			QuickLaunch:               false,
			Width:                     600,
			Sudo:                      false,
			ExecLabel:                 fmt.Sprintf("Execute %s", internal_utils.ToTitleCase(bashName)),
			ExecInProgressLabel:       fmt.Sprintf("Executing %s...", internal_utils.ToTitleCase(bashName)),
			StopLabel:                 "Stop",
			Fields:                    fields,
		})
	}
	return actions, nil
}

func parseBashFile(path string) ([]UIAction, error) {
	var actions []UIAction
	file, _ := os.Open(path)
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

			var fields []UIField
			for i := 0; i < len(exampleFieldNames); i++ {
				fields = append(fields, createField(exampleFieldNames[i], i))
			}

			actions = append(actions, UIAction{
				Label:                     internal_utils.ToTitleCase(name),
				Func:                      name,
				Mode:                      "g",
				RefreshFieldsOnTabOpen:    emptyArrayJSON(),
				RefreshFieldsOnCompletion: emptyArrayJSON(),
				QuickLaunch:               false,
				Width:                     600,
				Sudo:                      false,
				ExecLabel:                 fmt.Sprintf("Execute %s", internal_utils.ToTitleCase(name)),
				ExecInProgressLabel:       fmt.Sprintf("Executing %s...", internal_utils.ToTitleCase(name)),
				StopLabel:                 "Stop",
				Fields:                    fields,
			})
		}
	}
	return actions, nil
}

func createField(displayName string, index int) UIField {
	fieldType := uiFieldTypes[index%len(uiFieldTypes)]

	// Unconditionally apply the empty endpoints object so it is visible in the JSON
	field := UIField{
		Name:         internal_utils.ToTitleCase(displayName),
		Type:         fieldType,
		Default:      "",
		AdvancedMenu: false,
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
	case "FD":
		field.Default = ""
	case "FL", "MFL", "FS":
		field.Default = ""
		field.FileTypes = []string{"txt"}
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
