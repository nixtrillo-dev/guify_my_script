// Copyright (c) 2026 Nick Rowley
// All rights reserved.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package gui_gen

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// configContext holds the validated configuration and derived paths/states required
// for scaffolding the UI and building the final application.
type configContext struct {
	root                   RootConfig
	advancedTools          []AdvancedToolFile
	compileMode            string
	baseDir                string
	packageName            string
	sourcePathLinuxAMD64   string
	sourcePathLinuxAArch64 string
	sourcePathWindowsAMD64 string
	sourcePathAMD64        string // Deprecated: Kept for backwards compatibility
	sourcePathARM64        string // Deprecated: Kept for backwards compatibility
	bundledSourceName      string
	targetArchs            []string
	trayMinimizeState      string
	startupTrayState       string
	iconPath               string
	iconName               string
}

// loadAndPrepareConfig reads the JSON configuration, sanitizes inputs, derives
// compilation modes, and auto-detects target architectures.
func loadAndPrepareConfig(configPath, compileMode string) (configContext, error) {
	var ctx configContext

	ctx.compileMode = strings.ToLower(strings.TrimSpace(compileMode))
	if ctx.compileMode == "" {
		ctx.compileMode = "debug"
	}
	if ctx.compileMode != "debug" && ctx.compileMode != "production" {
		return ctx, fmt.Errorf("unsupported compile mode %q: expected debug or production", ctx.compileMode)
	}

	fileData, err := os.ReadFile(configPath)
	if err != nil {
		return ctx, fmt.Errorf("failed to open config: %w", err)
	}

	if err := json.Unmarshal(fileData, &ctx.root); err != nil {
		return ctx, fmt.Errorf("failed to load configuration: %w", err)
	}

	// --- VALIDATE UNIQUE ACTION LABELS ---
	seenLabels := make(map[string]bool)
	var duplicates []string

	for _, action := range ctx.root.Actions {
		if seenLabels[action.Label] {
			// Check if we've already recorded this duplicate to keep the error clean
			alreadyNoted := false
			for _, dup := range duplicates {
				if dup == action.Label {
					alreadyNoted = true
					break
				}
			}
			if !alreadyNoted {
				duplicates = append(duplicates, action.Label)
			}
		} else {
			seenLabels[action.Label] = true
		}
	}

	if len(duplicates) > 0 {
		return ctx, fmt.Errorf("failed to load configuration: duplicate action labels found: ['%s']. Action labels must be entirely unique across the JSON file", strings.Join(duplicates, "', '"))
	}
	if err := validateAdvancedActions(ctx.root.Actions); err != nil {
		return ctx, err
	}
	// -------------------------------------

	ctx.baseDir = filepath.Dir(configPath)

	ctx.trayMinimizeState = "false"
	if ctx.root.Config.TrayOnMinimize {
		ctx.trayMinimizeState = "true"
	}

	ctx.startupTrayState = "false"
	if ctx.root.Config.Startup == "tray" {
		ctx.startupTrayState = "true"
	}

	uuid := ctx.root.Config.UUID
	if uuid == "" {
		uuid = "default_app"
	}
	ctx.packageName = sanitizePackageName(uuid)

	ctx.sourcePathLinuxAMD64 = ctx.root.Config.SourceScriptLinuxAMD64
	if ctx.sourcePathLinuxAMD64 == "" {
		ctx.sourcePathLinuxAMD64 = ctx.root.Config.SourceScriptAMD64
	}
	if ctx.sourcePathLinuxAMD64 != "" && !filepath.IsAbs(ctx.sourcePathLinuxAMD64) {
		ctx.sourcePathLinuxAMD64 = filepath.Join(ctx.baseDir, ctx.sourcePathLinuxAMD64)
	}
	if ctx.sourcePathLinuxAMD64 != "" {
		if _, err := os.Stat(ctx.sourcePathLinuxAMD64); err != nil {
			legacyAMD64 := strings.Replace(ctx.sourcePathLinuxAMD64, "_linux_amd64", "_amd64", 1)
			if _, errLegacy := os.Stat(legacyAMD64); errLegacy == nil {
				ctx.sourcePathLinuxAMD64 = legacyAMD64
			}
		}
	}
	ctx.sourcePathAMD64 = ctx.sourcePathLinuxAMD64

	ctx.sourcePathLinuxAArch64 = ctx.root.Config.SourceScriptLinuxAArch64
	if ctx.sourcePathLinuxAArch64 == "" {
		ctx.sourcePathLinuxAArch64 = ctx.root.Config.SourceScriptARM64
	}
	if ctx.sourcePathLinuxAArch64 != "" && !filepath.IsAbs(ctx.sourcePathLinuxAArch64) {
		ctx.sourcePathLinuxAArch64 = filepath.Join(ctx.baseDir, ctx.sourcePathLinuxAArch64)
	}
	if ctx.sourcePathLinuxAArch64 != "" {
		if _, err := os.Stat(ctx.sourcePathLinuxAArch64); err != nil {
			legacyARM64 := strings.Replace(ctx.sourcePathLinuxAArch64, "_linux_aarch64", "_arm64", 1)
			if _, errLegacy := os.Stat(legacyARM64); errLegacy == nil {
				ctx.sourcePathLinuxAArch64 = legacyARM64
			}
		}
	}
	ctx.sourcePathARM64 = ctx.sourcePathLinuxAArch64

	ctx.sourcePathWindowsAMD64 = ctx.root.Config.SourceScriptWindowsAMD64
	if ctx.sourcePathWindowsAMD64 == "" && !isWindowsCompiledSourceType(ctx.root.Config.SourceType) {
		ctx.sourcePathWindowsAMD64 = ctx.sourcePathLinuxAMD64
	}
	if ctx.sourcePathWindowsAMD64 != "" && !filepath.IsAbs(ctx.sourcePathWindowsAMD64) {
		ctx.sourcePathWindowsAMD64 = filepath.Join(ctx.baseDir, ctx.sourcePathWindowsAMD64)
	}
	if strings.EqualFold(strings.TrimSpace(ctx.root.Config.SourceType), "rust") && strings.HasSuffix(ctx.sourcePathWindowsAMD64, "_windows_amd64_cli.exe") {
		legacyPath := strings.TrimSuffix(ctx.sourcePathWindowsAMD64, "_windows_amd64_cli.exe") + "_windows_amd64.exe"
		if _, err := os.Stat(legacyPath); err == nil {
			ctx.sourcePathWindowsAMD64 = legacyPath
		}
	}

	// Handle icon configuration
	// iconName is the theme-able icon name (e.g., package name or a system icon name)
	// iconPath is the absolute filesystem path to the icon file (for debug mode)
	ctx.iconName = ctx.root.Config.Icon
	if ctx.iconName == "" {
		ctx.iconName = "preferences-desktop"
	}

	// For non-system icons (not starting with a known theme icon), extract the filename without extension
	if ctx.iconName != "preferences-desktop" && !filepath.IsAbs(ctx.iconName) && ctx.iconName != "" {
		// This is a relative path to an icon file, use the package name as the icon name
		ctx.iconName = ctx.packageName
	}

	// iconPath is the absolute path for debug mode
	ctx.iconPath = ctx.root.Config.Icon
	if ctx.iconPath != "" && !filepath.IsAbs(ctx.iconPath) && ctx.iconPath != "preferences-desktop" {
		ctx.iconPath = filepath.Join(ctx.baseDir, ctx.iconPath)
	}

	ctx.bundledSourceName = strings.TrimSpace(ctx.root.Config.ScriptAlias)
	if ctx.bundledSourceName == "." || ctx.bundledSourceName == string(filepath.Separator) || ctx.bundledSourceName == "" {
		if ctx.sourcePathLinuxAMD64 != "" {
			ctx.bundledSourceName = filepath.Base(ctx.sourcePathLinuxAMD64)
		} else if ctx.sourcePathLinuxAArch64 != "" {
			ctx.bundledSourceName = filepath.Base(ctx.sourcePathLinuxAArch64)
		} else {
			ctx.bundledSourceName = "script_alias"
		}
	}
	if ctx.bundledSourceName == ctx.packageName {
		ctx.bundledSourceName = ctx.packageName + "-source"
	}

	// --- AUTO-DETECT ARCHITECTURES ---
	if ctx.compileMode == "production" {
		// Detect Linux AMD64
		if ctx.sourcePathLinuxAMD64 != "" {
			if _, err := os.Stat(ctx.sourcePathLinuxAMD64); err == nil {
				ctx.targetArchs = append(ctx.targetArchs, "linux_amd64")
			}
		}

		// Detect Linux AArch64
		if ctx.sourcePathLinuxAArch64 != "" {
			if _, err := os.Stat(ctx.sourcePathLinuxAArch64); err == nil {
				ctx.targetArchs = append(ctx.targetArchs, "linux_aarch64")
			}
		}
		if ctx.sourcePathWindowsAMD64 != "" {
			if _, err := os.Stat(ctx.sourcePathWindowsAMD64); err == nil {
				ctx.targetArchs = append(ctx.targetArchs, "windows_amd64")
			}
		}
	} else if os.Getenv("WINDOWS_QT6_PREFIX") != "" && ctx.sourcePathWindowsAMD64 != "" {
		if _, err := os.Stat(ctx.sourcePathWindowsAMD64); err == nil {
			ctx.targetArchs = append(ctx.targetArchs, "windows_amd64")
		}
	}

	// Fallback if no specific suffixes are found, or if running in debug mode
	if len(ctx.targetArchs) == 0 {
		ctx.targetArchs = []string{"linux_amd64"}
	}

	return ctx, nil
}

// loadAdvancedTools scans the /advanced_tools/ directory and parses all tool JSONs.
func loadAdvancedTools(baseDir string) ([]AdvancedToolFile, error) {
	_, currentFile, _, _ := runtime.Caller(0)
	guiGenDir := filepath.Dir(currentFile) // Gets the directory of this .go file
	toolsDir := filepath.Join(guiGenDir, "advanced_tools")

	if _, err := os.Stat(toolsDir); os.IsNotExist(err) {
		// Directory is optional; return empty slice if not present
		return nil, nil
	}

	files, err := os.ReadDir(toolsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read advanced_tools directory: %w", err)
	}

	var tools []AdvancedToolFile
	for _, f := range files {
		if f.IsDir() || filepath.Ext(f.Name()) != ".json" {
			continue
		}

		fullPath := filepath.Join(toolsDir, f.Name())
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read advanced tool file %s: %w", f.Name(), err)
		}

		var tool AdvancedToolFile
		if err := json.Unmarshal(data, &tool); err != nil {
			return nil, fmt.Errorf("failed to load advanced tool %s: %w", f.Name(), err)
		}

		tool.FilePath = fullPath
		tools = append(tools, tool)
	}

	return tools, nil
}

func validateAdvancedMenuConfiguration(actions []UIAction, tools []AdvancedToolFile) error {
	if err := validateAdvancedActions(actions); err != nil {
		return err
	}
	toolTitles := make(map[string]string)
	for _, tool := range tools {
		title := strings.TrimSpace(tool.Config.TabTitle)
		if title == "" {
			return fmt.Errorf("advanced tool %q has an empty tab_title", tool.FilePath)
		}
		key := strings.ToLower(title)
		if previous, exists := toolTitles[key]; exists {
			return fmt.Errorf("advanced tab title %q is duplicated by %q and %q", title, previous, tool.FilePath)
		}
		toolTitles[key] = tool.FilePath
	}

	for _, action := range actions {
		if !action.AdvancedMenu {
			continue
		}
		title := strings.TrimSpace(action.AdvancedMenuTabTitle)
		key := strings.ToLower(title)
		if toolPath, exists := toolTitles[key]; exists {
			return fmt.Errorf("action %q advanced_menu_tab_title %q conflicts with advanced tool %q", action.Label, title, toolPath)
		}
	}
	return nil
}

func validateAdvancedActions(actions []UIAction) error {
	for _, action := range actions {
		if action.AdvancedMenu && strings.TrimSpace(action.AdvancedMenuTabTitle) == "" {
			return fmt.Errorf("action %q must define a non-empty advanced_menu_tab_title when advanced_menu is true", action.Label)
		}
	}
	return nil
}
