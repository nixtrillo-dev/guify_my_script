// Copyright (c) 2026 Nick Rowley
// All rights reserved.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package gui_gen

import (
	"encoding/json"
	"fmt"
	"strings"
)

// generateRuntimeState builds the metadata and default configuration maps
// for the UI components and serializes them to JSON for the C++ template.
func generateRuntimeState(root RootConfig) ([]byte, []byte, error) {
	defaultsMap := make(map[string]map[string]string)
	metadataMap := make(map[string]map[string]map[string]interface{})

	appSettings := make(map[string]string)
	appSettingsMeta := make(map[string]map[string]interface{})

	actionOptions := []string{""}
	for _, action := range root.Actions {
		actionOptions = append(actionOptions, action.Label)
	}

	appSettings["Action On Startup"] = root.Config.ActionOnStartup
	appSettingsMeta["Action On Startup"] = map[string]interface{}{
		"type":    "CB",
		"options": actionOptions,
	}

	appSettings["Default Tab On Startup"] = root.Config.DefaultTabOnStartup
	appSettingsMeta["Default Tab On Startup"] = map[string]interface{}{
		"type":    "CB",
		"options": actionOptions, // Reuses the actionOptions array
	}

	closeStartupDef := "0"
	if root.Config.CloseAfterStartupAction {
		closeStartupDef = "1"
	}
	appSettings["Close After Startup Action"] = closeStartupDef
	appSettingsMeta["Close After Startup Action"] = map[string]interface{}{
		"type":    "CK",
		"options": []string{},
	}

	themeColor := strings.ToLower(strings.TrimSpace(root.Config.Theme.Color))
	if themeColor == "" {
		themeColor = "dark"
	}
	appSettings["Theme Color"] = themeColor
	appSettingsMeta["Theme Color"] = map[string]interface{}{
		"type":    "CB",
		"options": []string{"dark", "light", "auto"},
	}

	defaultsMap["App Settings"] = appSettings
	metadataMap["App Settings"] = appSettingsMeta

	for _, action := range root.Actions {
		if len(action.Fields) > 0 {
			actMap := make(map[string]string)
			metaActMap := make(map[string]map[string]interface{})

			for _, field := range action.Fields {
				defStr := getDefaultString(field.Default)
				actMap[field.Name] = defStr

				metaActMap[field.Name] = map[string]interface{}{
					"type":           field.GetFieldType(),
					"field_type":     field.GetFieldType(),
					"options":        field.Options,
					"default":        defStr,
					"enabled":        field.IsEnabled(),
					"minimum":        field.Minimum,
					"maximum":        field.Maximum,
					"decimal_places": field.DecimalPlaces,
					"auto_height":    field.AutoHeight,
					"min_height":     field.MinHeight,
					"file_types":     field.FileTypes,
					"json_endpoints": map[string]interface{}{
						"array_endpoint":        field.JsonEndpoints.ArrayEndpoint,
						"path_to_array":         field.JsonEndpoints.PathToArray,
						"get_defaults_endpoint": field.JsonEndpoints.GetDefaultsEndpoint,
						"path_to_default":       field.JsonEndpoints.PathToDefault,
						"get_enabled_endpoint":  field.JsonEndpoints.GetEnabledEndpoint,
						"path_to_enabled":       field.JsonEndpoints.PathToEnabled,
					},
					"refresh_fields_on_change": field.RefreshFieldsOnChange,
				}
			}
			defaultsMap[action.Label] = actMap
			metadataMap[action.Label] = metaActMap
		}
	}

	defaultsJSONBytes, err := json.MarshalIndent(defaultsMap, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal defaults JSON: %w", err)
	}

	metadataJSONBytes, err := json.Marshal(metadataMap)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal metadata JSON: %w", err)
	}

	return defaultsJSONBytes, metadataJSONBytes, nil
}
