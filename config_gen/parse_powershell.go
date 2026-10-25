package config_gen

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"guify_my_script/internal_utils"
)

var powershellFunctionRe = regexp.MustCompile(`(?mi)^\s*function\s+([A-Za-z_][A-Za-z0-9_-]*)\s*\{`)
var powershellTypeRe = regexp.MustCompile(`(?i)\[([A-Za-z][A-Za-z0-9\[\]]*)\]\s*\$([A-Za-z_][A-Za-z0-9_]*)`)
var powershellParamBlockRe = regexp.MustCompile(`(?mi)\bparam\s*\(`)

func parsePowerShellFile(path string) ([]UIAction, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := string(data)
	var actions []UIAction
	for _, match := range powershellFunctionRe.FindAllStringSubmatchIndex(content, -1) {
		name := content[match[2]:match[3]]
		if strings.HasPrefix(name, "_") || name == "main" || name == "usage" || name == "cleanup" {
			continue
		}
		paramMatch := powershellParamBlockRe.FindStringIndex(content[match[1]:])
		if paramMatch == nil {
			continue
		}
		open := match[1] + paramMatch[1] - 1
		paramsRaw, _, ok := extractBalanced(content, open)
		if !ok {
			continue
		}
		var fields []UIField
		for _, typeMatch := range powershellTypeRe.FindAllStringSubmatchIndex(paramsRaw, -1) {
			param := paramsRaw[typeMatch[0]:]
			if comma := strings.Index(param, ","); comma >= 0 {
				param = param[:comma]
			}
			if field := createPowerShellField(param); field.Name != "" {
				field.ParameterPosition = (len(fields) + 1) * 100
				fields = append(fields, field)
			}
		}
		actions = append(actions, UIAction{
			Label: internal_utils.ToTitleCase(name), Func: name, ExecutionMode: "GUI",
			RefreshFieldsOnTabOpen: emptyArrayJSON(), RefreshFieldsOnCompletion: emptyArrayJSON(),
			QuickLaunch: false, Width: 600, Sudo: false,
			ExecLabel:           fmt.Sprintf("Execute %s", internal_utils.ToTitleCase(name)),
			ExecInProgressLabel: fmt.Sprintf("Executing %s...", internal_utils.ToTitleCase(name)),
			StopLabel:           "Abort", Fields: fields,
		})
	}
	return actions, nil
}

func createPowerShellField(param string) UIField {
	param = strings.TrimSpace(param)
	if param == "" || strings.HasPrefix(param, "#") {
		return UIField{}
	}
	defaultValue := ""
	if idx := strings.Index(param, "="); idx >= 0 {
		defaultValue = strings.TrimSpace(param[idx+1:])
		param = strings.TrimSpace(param[:idx])
	}
	matches := powershellTypeRe.FindStringSubmatch(param)
	if len(matches) == 0 {
		return UIField{}
	}
	typeHint := strings.ToLower(matches[1])
	argName := matches[2]
	dataType, fieldType, defaultVal := "string", "TXT", any("Initial Value")
	var decimalPlaces *string
	switch typeHint {
	case "int", "long", "short", "byte", "uint", "uint32", "uint64":
		dataType, fieldType, defaultVal = "int", "NUM", "0"
		zero := "0"
		decimalPlaces = &zero
	case "double", "float", "decimal":
		dataType, fieldType, defaultVal = "float", "NUM", "0.0"
		two := "2"
		decimalPlaces = &two
	case "bool", "boolean", "switch":
		dataType, fieldType, defaultVal = "bool", "CK", "0"
	case "string[]", "object[]", "array", "list":
		dataType, fieldType, defaultVal = "list", "TXT", ""
	}
	if defaultValue != "" {
		defaultValue = strings.Trim(strings.ToLower(defaultValue), "'\"")
		if dataType == "bool" {
			if defaultValue == "$true" || defaultValue == "true" || defaultValue == "1" {
				defaultVal = "1"
			} else {
				defaultVal = "0"
			}
		} else {
			defaultVal = strings.Trim(defaultValue, "'\"")
		}
	}
	field := UIField{Name: internal_utils.ToTitleCase(argName), ArgName: argName, Type: fieldType, DataType: dataType,
		Default: defaultVal, DecimalPlaces: decimalPlaces, AdvancedMenu: false, Enabled: true,
		JsonEndpoints: JsonEndpoints{}, RefreshFieldsOnChange: emptyArrayJSON()}
	if fieldType == "NUM" {
		field.Minimum = nullJSON()
		field.Maximum = nullJSON()
	}
	return field
}
