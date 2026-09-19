package gui_gen

import (
	"encoding/json"
	"fmt"
	"strings"
)

// IsValidExecutionMode checks whether the given mode string (case-insensitive) is valid.
func IsValidExecutionMode(mode string) bool {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "g", "gui",
		"gc", "gui_commandline",
		"s", "silent",
		"d", "detached_commandline",
		"c", "commandline":
		return true
	default:
		return false
	}
}

// NormalizeExecutionMode normalizes an execution mode string to its canonical shorthand
// representation ("g", "gc", "s", "d", "c"), case-insensitively. Defaults to "c".
func NormalizeExecutionMode(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "g", "gui":
		return "g"
	case "gc", "gui_commandline":
		return "gc"
	case "s", "silent":
		return "s"
	case "d", "detached_commandline":
		return "d"
	case "c", "commandline", "":
		return "c"
	default:
		return "c"
	}
}

// NormalizeExecutionModeLonghand normalizes an execution mode string to its canonical
// longhand representation ("GUI", "GUI_COMMANDLINE", "SILENT", "DETACHED_COMMANDLINE", "COMMANDLINE"),
// case-insensitively. Defaults to "COMMANDLINE".
func NormalizeExecutionModeLonghand(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "g", "gui":
		return "GUI"
	case "gc", "gui_commandline":
		return "GUI_COMMANDLINE"
	case "s", "silent":
		return "SILENT"
	case "d", "detached_commandline":
		return "DETACHED_COMMANDLINE"
	case "c", "commandline", "":
		return "COMMANDLINE"
	default:
		return "COMMANDLINE"
	}
}

type JsonEndpoints struct {
	ArrayEndpoint       []string `json:"array_endpoint,omitempty"`
	PathToArray         string   `json:"path_to_array,omitempty"`
	GetDefaultsEndpoint []string `json:"get_defaults_endpoint,omitempty"`
	PathToDefault       string   `json:"path_to_default,omitempty"`
	GetEnabledEndpoint  []string `json:"get_enabled_endpoint,omitempty"`
	PathToEnabled       string   `json:"path_to_enabled,omitempty"`
}

type UIField struct {
	Name                  string        `json:"name"`
	FieldType             string        `json:"field_type,omitempty"`
	Type                  string        `json:"type,omitempty"`
	DataType              string        `json:"data_type"`
	ArgName               string        `json:"arg_name,omitempty"`
	Options               []interface{} `json:"options,omitempty"`
	Default               interface{}   `json:"default"`
	Minimum               string        `json:"minimum,omitempty"`
	Maximum               string        `json:"maximum,omitempty"`
	DecimalPlaces         string        `json:"decimal_places,omitempty"`
	AdvancedMenu          bool          `json:"advanced_menu,omitempty"`
	Enabled               *bool         `json:"enabled,omitempty"`
	AutoHeight            bool          `json:"auto_height,omitempty"`
	MinHeight             int           `json:"min_height,omitempty"`
	FileTypes             []string      `json:"file_types,omitempty"`
	JsonEndpoints         JsonEndpoints `json:"json_endpoints,omitempty"`
	RefreshFieldsOnChange []string      `json:"refresh_fields_on_change,omitempty"`
}

// GetFieldType returns the field type, preferring FieldType over Type.
func (f UIField) GetFieldType() string {
	if f.FieldType != "" {
		return f.FieldType
	}
	return f.Type
}

// IsEnabled returns true if Enabled is not set (nil) or set to true.
func (f UIField) IsEnabled() bool {
	if f.Enabled == nil {
		return true
	}
	return *f.Enabled
}

func (f *UIField) UnmarshalJSON(data []byte) error {
	type Alias UIField
	aux := &struct {
		RawType      *string `json:"type"`
		RawFieldType *string `json:"field_type"`
		*Alias
	}{
		Alias: (*Alias)(f),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	fieldIdentifier := f.Name
	if fieldIdentifier == "" {
		fieldIdentifier = f.ArgName
	}

	// Fail early if a field tries to define both "type" and "field_type" simultaneously
	if aux.RawType != nil && aux.RawFieldType != nil {
		return fmt.Errorf("field %q cannot define both 'type' and 'field_type'; specify only one", fieldIdentifier)
	}

	if aux.RawFieldType != nil {
		f.FieldType = *aux.RawFieldType
		f.Type = *aux.RawFieldType
	} else if aux.RawType != nil {
		f.FieldType = *aux.RawType
		f.Type = *aux.RawType
	} else if f.FieldType != "" {
		f.Type = f.FieldType
	} else if f.Type != "" {
		f.FieldType = f.Type
	}

	return nil
}

func (f UIField) MarshalJSON() ([]byte, error) {
	type Alias UIField
	copy := f
	if copy.FieldType != "" {
		copy.Type = ""
	} else if copy.Type != "" {
		copy.FieldType = copy.Type
		copy.Type = ""
	}
	return json.Marshal((*Alias)(&copy))
}

type UIAction struct {
	Label                     string    `json:"label"`
	Func                      string    `json:"func"`
	ExecutionMode             string    `json:"execution_mode,omitempty"`
	Mode                      string    `json:"mode,omitempty"`
	Width                     int       `json:"width"`
	QuickLaunch               bool      `json:"quick_launch,omitempty"`
	Sudo                      bool      `json:"sudo,omitempty"`
	ExecLabel                 string    `json:"exec_label,omitempty"`
	ExecInProgressLabel       string    `json:"exec_in_progress_label,omitempty"`
	StopLabel                 string    `json:"stop_label,omitempty"`
	RefreshFieldsOnTabOpen    []string  `json:"refresh_fields_on_tab_open,omitempty"`
	RefreshFieldsOnCompletion []string  `json:"refresh_fields_on_completion,omitempty"`
	InvokeAsObject            bool      `json:"invoke_as_object,omitempty"`
	Fields                    []UIField `json:"fields"`
}

// GetMode returns the canonical shorthand mode ("g", "gc", "s", "d", "c") for this action.
func (a UIAction) GetMode() string {
	if a.ExecutionMode != "" {
		return NormalizeExecutionMode(a.ExecutionMode)
	}
	return NormalizeExecutionMode(a.Mode)
}

// GetExecutionMode returns the canonical longhand execution mode ("GUI", "GUI_COMMANDLINE", etc.) for this action.
func (a UIAction) GetExecutionMode() string {
	if a.ExecutionMode != "" {
		return NormalizeExecutionModeLonghand(a.ExecutionMode)
	}
	return NormalizeExecutionModeLonghand(a.Mode)
}

func (a *UIAction) UnmarshalJSON(data []byte) error {
	type Alias UIAction
	aux := &struct {
		RawMode          *string `json:"mode"`
		RawExecutionMode *string `json:"execution_mode"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}
	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	actionIdentifier := a.Label
	if actionIdentifier == "" {
		actionIdentifier = a.Func
	}

	// Fail early if someone tries to define both properties under a single action
	if aux.RawMode != nil && aux.RawExecutionMode != nil {
		return fmt.Errorf("action %q cannot define both 'mode' and 'execution_mode'; specify only one", actionIdentifier)
	}

	if aux.RawExecutionMode != nil {
		val := *aux.RawExecutionMode
		if !IsValidExecutionMode(val) {
			return fmt.Errorf("action %q has invalid execution_mode %q: allowed values are GUI (g), GUI_COMMANDLINE (gc), SILENT (s), DETACHED_COMMANDLINE (d), COMMANDLINE (c)", actionIdentifier, val)
		}
		a.ExecutionMode = NormalizeExecutionModeLonghand(val)
		a.Mode = NormalizeExecutionMode(val)
	} else if aux.RawMode != nil {
		val := *aux.RawMode
		if !IsValidExecutionMode(val) {
			return fmt.Errorf("action %q has invalid mode %q: allowed values are GUI (g), GUI_COMMANDLINE (gc), SILENT (s), DETACHED_COMMANDLINE (d), COMMANDLINE (c)", actionIdentifier, val)
		}
		a.ExecutionMode = NormalizeExecutionModeLonghand(val)
		a.Mode = NormalizeExecutionMode(val)
	} else if a.ExecutionMode != "" {
		a.Mode = NormalizeExecutionMode(a.ExecutionMode)
		a.ExecutionMode = NormalizeExecutionModeLonghand(a.ExecutionMode)
	} else if a.Mode != "" {
		a.ExecutionMode = NormalizeExecutionModeLonghand(a.Mode)
		a.Mode = NormalizeExecutionMode(a.Mode)
	} else {
		a.ExecutionMode = "COMMANDLINE"
		a.Mode = "c"
	}

	return nil
}

type MainMenuConfig struct {
	TabLabel            string `json:"tab_label,omitempty"`
	WelcomeHeadingLabel string `json:"welcome_heading_label,omitempty"`
	InfoLabel           string `json:"info_label,omitempty"`
}

type ThemeConfig struct {
	Color  string `json:"color,omitempty"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type AppConfig struct {
	Title                    string      `json:"title"`
	Icon                     string      `json:"icon"`
	UUID                     string      `json:"uuid"`
	ScriptAlias              string      `json:"script_alias"`
	SourceScriptLinuxAMD64   string      `json:"source_script_linux_amd64,omitempty"`
	SourceScriptLinuxAArch64 string      `json:"source_script_linux_aarch64,omitempty"`
	SourceScriptAMD64        string      `json:"source_script_amd64,omitempty"`
	SourceScriptARM64        string      `json:"source_script_arm64,omitempty"`
	SourceType               string      `json:"source_type"`
	Startup                 string      `json:"startup"`
	TrayOnMinimize          bool        `json:"tray_on_minimize"`
	InstanceMode            string      `json:"instance_mode"`
	ActionOnStartup         string      `json:"action_on_startup,omitempty"`
	DefaultTabOnStartup     string      `json:"default_tab_on_startup,omitempty"`
	CloseAfterStartupAction bool        `json:"close_after_startup_action,omitempty"`
	Theme                   ThemeConfig `json:"theme"`
}

// DeployDependency represents custom package dependencies for various distros.
type DeployDependency struct {
	Pacman []string `json:"pacman,omitempty"`
	Arch   []string `json:"arch,omitempty"`
	Apt    []string `json:"apt,omitempty"`
	Debian []string `json:"debian,omitempty"`
	Dnf    []string `json:"dnf,omitempty"`
	Fedora []string `json:"fedora,omitempty"`
}

// DeployConfig holds packaging and deployment metadata.
type DeployConfig struct {
	Description   string `json:"description,omitempty"`
	Version       string `json:"version,omitempty"`
	ReleaseNumber string `json:"release_number,omitempty"`
	Summary       string `json:"summary,omitempty"`
	License       string `json:"license,omitempty"`
	Maintainer    string `json:"maintainer,omitempty"`
	SupportEmail  string `json:"support_email,omitempty"`
	Dependencies   []DeployDependency `json:"dependencies,omitempty"`
}

type RootConfig struct {
	Config   AppConfig      `json:"config"`
	MainMenu MainMenuConfig `json:"main_menu,omitempty"`
	Deploy   DeployConfig   `json:"deploy,omitempty"`
	Actions  []UIAction     `json:"actions"`
}

// AdvancedToolConfig holds the metadata for an advanced tool plugin.
type AdvancedToolConfig struct {
	TabTitle       string `json:"tab_title"`
	PackageUUID    string `json:"package_uuid,omitempty"`
	PackageVersion string `json:"package_version,omitempty"`
	SourceScript   string `json:"source_script"`
	SourceType     string `json:"source_type"`
	SourceGUI      string `json:"source_gui,omitempty"`
}

// AdvancedToolFile represents the root structure of an advanced tool JSON config.
type AdvancedToolFile struct {
	FilePath string             `json:"-"` // Internal tracking (ignored by JSON parser)
	Config   AdvancedToolConfig `json:"config"`
	Actions  []UIAction         `json:"actions"` // Reuses your existing UIAction struct
}
