package gui_gen

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
	Type                  string        `json:"type"`
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

// IsEnabled returns true if Enabled is not set (nil) or set to true.
func (f UIField) IsEnabled() bool {
	if f.Enabled == nil {
		return true
	}
	return *f.Enabled
}

type UIAction struct {
	Label                     string    `json:"label"`
	Func                      string    `json:"func"`
	Mode                      string    `json:"mode"`
	Width                     int       `json:"width"`
	QuickLaunch               bool      `json:"quick_launch,omitempty"`
	Sudo                      bool      `json:"sudo,omitempty"`
	ExecLabel                 string    `json:"exec_label,omitempty"`
	ExecInProgressLabel       string    `json:"exec_in_progress_label,omitempty"`
	StopLabel                 string    `json:"stop_label,omitempty"`
	RefreshFieldsOnTabOpen    []string  `json:"refresh_fields_on_tab_open,omitempty"`
	RefreshFieldsOnCompletion []string  `json:"refresh_fields_on_completion,omitempty"`
	Fields                    []UIField `json:"fields"`
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
	Title                   string      `json:"title"`
	Icon                    string      `json:"icon"`
	UUID                    string      `json:"uuid"`
	ScriptAlias             string      `json:"script_alias"`
	SourceScriptAMD64       string      `json:"source_script_amd64,omitempty"`
	SourceScriptARM64       string      `json:"source_script_arm64,omitempty"`
	SourceType              string      `json:"source_type"`
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
