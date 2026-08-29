// Copyright (c) 2026 Nick Rowley
// All rights reserved.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package gui_gen

import (
	_ "embed"
	"encoding/csv"
	"fmt"
	"strconv"
	"strings"

	"guify_my_script/logger"
)

//go:embed templates/main_window.cpp
var cppTemplatePart1 string

//go:embed templates/main.cpp
var cppTemplatePart2 string

// GenerateUI orchestrates the parsing of the configuration and the generation of the C++ UI code.
func GenerateUI(configPath, compileMode, logLevelStr string, clearCache bool, targetBuilds string) error {
	logLevel := logger.ParseLogLevel(logLevelStr)

	logger.Debug(logLevel, "targetBuilds will be printed on the next line...")
	logger.Debug(logLevel, targetBuilds)

	// --- INPUT VALIDATION START ---
	// 1. Validate compileMode
	cleanMode := strings.ToLower(strings.TrimSpace(compileMode))
	validCompileModes := map[string]bool{
		"debug":      true,
		"production": true,
	}

	if !validCompileModes[cleanMode] {
		return fmt.Errorf("invalid compileMode specified: '%s'. Allowed values are 'debug' or 'production'", compileMode)
	}

	// 2. Validate targetBuilds
	// Empty string is allowed (builds all defaults)
	if targetBuilds != "" {
		reader := csv.NewReader(strings.NewReader(targetBuilds))
		reader.TrimLeadingSpace = true

		builds, err := reader.Read()
		if err != nil && err.Error() != "EOF" {
			return fmt.Errorf("failed to parse targetBuilds parameter: %w", err)
		}

		// Define the exhaustive list of valid target builds based on supported PMs and Architectures
		validBuilds := map[string]bool{
			".pkg.tar.zst_amd64": true,
			".pkg.tar.zst_arm64": true,
			".deb_amd64":         true,
			".deb_arm64":         true,
			".rpm_amd64":         true,
			".rpm_arm64":         true,
		}

		for _, b := range builds {
			cleanBuild := strings.TrimSpace(b)

			if cleanBuild == "" {
				continue
			}

			// Halt immediately if an unmapped target is requested
			if !validBuilds[cleanBuild] {
				return fmt.Errorf("invalid target build specified: '%s'. Allowed values are an empty string (compile all), or comma-separated combinations like '.deb_amd64', '.pkg.tar.zst_arm64', etc", cleanBuild)
			}
		}
	}
	// --- INPUT VALIDATION END ---

	ctx, err := loadAndPrepareConfig(configPath, compileMode)
	if err != nil {
		return err
	}

	// Load all JSON configurations under /advanced_tools/
	ctx.advancedTools, err = loadAdvancedTools(ctx.baseDir)
	if err != nil {
		return fmt.Errorf("failed to process advanced tools: %w", err)
	}

	sourceScriptExpr := fmt.Sprintf("QStringLiteral(\"%s\")", cppStringLiteral(ctx.root.Config.SourceScript))
	if ctx.compileMode == "production" {
		sourceScriptExpr = fmt.Sprintf("QCoreApplication::applicationDirPath() + QStringLiteral(\"/%s\")", cppStringLiteral(ctx.bundledSourceName))
	}

	var cpp strings.Builder
	cpp.WriteString(cppTemplatePart1)

	// Inject extracted layouts
	advancedRowsStr := buildQuickLaunchRows(ctx.root.Actions)
	actionTabsStr := buildActionTabs(ctx.root.Actions, ctx.root.Config.SourceType, ctx.root.Config.ActionOnStartup, ctx.root.Config.CloseAfterStartupAction, ctx.root.Config.DefaultTabOnStartup)

	// Generate dynamic Advanced Tool tabs
	advancedToolTabsStr := buildAdvancedToolTabs(ctx.advancedTools, ctx.root.Config.UUID)

	cpp.WriteString(actionTabsStr)
	cpp.WriteString(cppTemplatePart2)

	finalSource := cpp.String()
	finalSource = strings.ReplaceAll(finalSource, "TEMPLATE_SOURCE_SCRIPT_EXPR", sourceScriptExpr)
	finalSource = strings.ReplaceAll(finalSource, "TEMPLATE_UUID", ctx.root.Config.UUID)
	finalSource = strings.ReplaceAll(finalSource, "TEMPLATE_TRAY_ON_MINIMIZE", ctx.trayMinimizeState)
	finalSource = strings.ReplaceAll(finalSource, "TEMPLATE_STARTUP_TO_TRAY", ctx.startupTrayState)
	finalSource = strings.ReplaceAll(finalSource, "TEMPLATE_INSTANCE_MODE", ctx.root.Config.InstanceMode)
	finalSource = strings.ReplaceAll(finalSource, "TEMPLATE_TITLE", ctx.root.Config.Title)
	finalSource = strings.ReplaceAll(finalSource, "TEMPLATE_PACKAGE_NAME", ctx.packageName)

	// Set a default theme color if one is not provided in the JSON
	themeColor := strings.ToLower(strings.TrimSpace(ctx.root.Config.Theme.Color))
	if themeColor == "" {
		themeColor = "dark"
	}
	if themeColor != "dark" && themeColor != "light" && themeColor != "auto" {
		return fmt.Errorf("invalid theme color specified: '%s'. Allowed values are 'dark', 'light', or 'auto'", ctx.root.Config.Theme.Color)
	}
	finalSource = strings.ReplaceAll(finalSource, "TEMPLATE_THEME_COLOR", themeColor)

	finalSource = strings.ReplaceAll(finalSource, "TEMPLATE_WIDTH", strconv.Itoa(ctx.root.Config.Theme.Width))
	finalSource = strings.ReplaceAll(finalSource, "TEMPLATE_HEIGHT", strconv.Itoa(ctx.root.Config.Theme.Height))
	finalSource = strings.ReplaceAll(finalSource, "TEMPLATE_ADVANCED_TRIGGER_ROWS", advancedRowsStr)
	// Inject the tools directly into the Advanced Settings dialog
	finalSource = strings.ReplaceAll(finalSource, "TEMPLATE_ADVANCED_TOOLS_TABS", advancedToolTabsStr)

	// Setting up the Main Tab's Elements...
	// 1. Configure the Tab's Name (Fallback to "Main Menu" if left empty)
	tabLabel := ctx.root.MainMenu.TabLabel
	if tabLabel == "" {
		tabLabel = "Main Menu"
	}
	finalSource = strings.ReplaceAll(finalSource, "TEMPLATE_MAIN_MENU_TAB_LABEL", cppStringLiteral(tabLabel))

	// 2. Configure the Welcome Heading (No fallback - allows complete collapse if empty)
	headingLabel := ctx.root.MainMenu.WelcomeHeadingLabel
	finalSource = strings.ReplaceAll(finalSource, "TEMPLATE_MAIN_MENU_HEADING_LABEL", cppStringLiteral(headingLabel))

	// 3. Configure the Info Description (No fallback - allows complete collapse if empty)
	infoLabel := ctx.root.MainMenu.InfoLabel
	finalSource = strings.ReplaceAll(finalSource, "TEMPLATE_MAIN_MENU_INFO_LABEL", cppStringLiteral(infoLabel))

	// Icon logic
	iconVal := ctx.root.Config.Icon
	if iconVal == "" {
		iconVal = "preferences-desktop"
	}
	finalSource = strings.ReplaceAll(finalSource, "TEMPLATE_ICON_RAW_PATH", iconVal)

	// Generate JSON runtime metadata
	defaultsJSONBytes, metadataJSONBytes, err := generateRuntimeState(ctx.root)
	if err != nil {
		return err
	}

	finalSource = strings.ReplaceAll(finalSource, "TEMPLATE_JSON_DEFAULTS", string(defaultsJSONBytes))
	finalSource = strings.ReplaceAll(finalSource, "TEMPLATE_JSON_METADATA", string(metadataJSONBytes))

	// Pass control to the execution engine
	buildCtx := BuildContext{
		CompileMode:       ctx.compileMode,
		TargetArchs:       ctx.targetArchs,
		BaseDir:           ctx.baseDir,
		PackageName:       ctx.packageName,
		BundledSourceName: ctx.bundledSourceName,
		SourcePath:        ctx.sourcePath,
		FinalSourceCode:   finalSource,
		DefaultsJSON:      defaultsJSONBytes,
		RootConfig:        ctx.root,
		LogLevel:          logLevel,
		ClearCache:        clearCache,
		TargetBuilds:      targetBuilds,
	}

	return runBuildLoop(buildCtx)
}
