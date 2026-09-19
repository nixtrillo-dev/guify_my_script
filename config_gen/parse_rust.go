package config_gen

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"guify_my_script/internal_utils"
)

func parseRustFile(path string) ([]UIAction, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	content := string(data)
	var actions []UIAction

	// If this is a generated bridge file with "pub extern "C" fn Rust...", parse those exports
	if strings.Contains(content, "pub extern \"C\" fn Rust") {
		bridgeFnRegex := regexp.MustCompile(`(?s)pub\s+extern\s+"C"\s+fn\s+Rust([A-Za-z0-9_]+)\s*\((.*?)\)`)
		matches := bridgeFnRegex.FindAllStringSubmatch(content, -1)

		for _, m := range matches {
			rawName := m[1]
			bashName := strings.ToLower(strings.Join(internal_utils.SplitCamelCase(rawName), "_"))
			rawParams := strings.TrimSpace(m[2])

			var paramNames []string
			if rawParams != "" {
				parts := strings.Split(rawParams, ",")
				for _, p := range parts {
					p = strings.TrimSpace(p)
					if colonIdx := strings.Index(p, ":"); colonIdx != -1 {
						name := strings.TrimSpace(p[:colonIdx])
						if name != "" {
							paramNames = append(paramNames, name)
						}
					}
				}
			}

			fields := []UIField{}
			for i, name := range paramNames {
				label := internal_utils.ToTitleCase(name)
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
				StopLabel:                 "Abort",
				Fields:                    fields,
			})
		}
		return actions, nil
	}

	// Otherwise, parse standard Rust source file pub fn declarations
	fnRegex := regexp.MustCompile(`(?s)\bpub\s+(?:async\s+|unsafe\s+|extern\s+"[^"]+"\s+)*fn\s+([A-Za-z0-9_]+)\s*\((.*?)\)`)
	matches := fnRegex.FindAllStringSubmatch(content, -1)

	for _, m := range matches {
		fnName := strings.TrimSpace(m[1])
		if fnName == "main" || fnName == "test" || strings.HasPrefix(fnName, "_") || strings.HasPrefix(fnName, "rust_") {
			continue
		}

		bashName := strings.ToLower(strings.Join(internal_utils.SplitCamelCase(fnName), "_"))
		rawParams := strings.TrimSpace(m[2])

		var paramNames []string
		if rawParams != "" {
			parts := strings.Split(rawParams, ",")
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p == "self" || p == "&self" || p == "&mut self" || p == "" {
					continue
				}
				if colonIdx := strings.Index(p, ":"); colonIdx != -1 {
					name := strings.TrimSpace(p[:colonIdx])
					if name != "" {
						paramNames = append(paramNames, name)
					}
				}
			}
		}

		fields := []UIField{}
		for i, name := range paramNames {
			label := internal_utils.ToTitleCase(name)
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
			StopLabel:                 "Abort",
			Fields:                    fields,
		})
	}

	return actions, nil
}
