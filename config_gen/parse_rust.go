package config_gen

import (
	"fmt"
	"os"
	"path/filepath"
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
	enumOptions := rustEnumOptions(content)
	for name, variants := range rustEnumOptionsNearby(path) {
		if _, exists := enumOptions[name]; !exists {
			enumOptions[name] = variants
		}
	}

	// If this is a generated bridge file with "pub extern "C" fn Rust...", parse those exports
	if strings.Contains(content, "pub extern \"C\" fn Rust") {
		bridgeFnRegex := regexp.MustCompile(`(?s)pub\s+extern\s+"C"\s+fn\s+Rust([A-Za-z0-9_]+)\s*\((.*?)\)`)
		matches := bridgeFnRegex.FindAllStringSubmatch(content, -1)

		for _, m := range matches {
			rawName := m[1]
			bashName := strings.ToLower(strings.Join(internal_utils.SplitCamelCase(rawName), "_"))
			rawParams := strings.TrimSpace(m[2])

			type rustParam struct {
				name, enumName string
				multi          bool
			}
			var params []rustParam
			if rawParams != "" {
				parts := strings.Split(rawParams, ",")
				for _, p := range parts {
					p = strings.TrimSpace(p)
					if colonIdx := strings.Index(p, ":"); colonIdx != -1 {
						name := strings.TrimSpace(p[:colonIdx])
						if name != "" {
							enumName, multi := rustEnumParamType(strings.TrimSpace(p[colonIdx+1:]), enumOptions)
							params = append(params, rustParam{name: name, enumName: enumName, multi: multi})
						}
					}
				}
			}

			fields := []UIField{}
			for i, param := range params {
				label := internal_utils.ToTitleCase(param.name)
				if variants, ok := enumOptions[param.enumName]; ok && param.enumName != "" {
					fields = append(fields, createEnumField(label, param.name, variants, param.multi, i))
				} else {
					fields = append(fields, createField(label, i))
				}
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

		type rustParam struct {
			name, enumName string
			multi          bool
		}
		var params []rustParam
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
						enumName, multi := rustEnumParamType(strings.TrimSpace(p[colonIdx+1:]), enumOptions)
						params = append(params, rustParam{name: name, enumName: enumName, multi: multi})
					}
				}
			}
		}

		fields := []UIField{}
		for i, param := range params {
			label := internal_utils.ToTitleCase(param.name)
			if variants, ok := enumOptions[param.enumName]; ok && param.enumName != "" {
				fields = append(fields, createEnumField(label, param.name, variants, param.multi, i))
			} else {
				fields = append(fields, createField(label, i))
			}
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

func rustEnumOptionsNearby(filePath string) map[string][]string {
	out := make(map[string][]string)
	root := filepath.Dir(filePath)
	var roots []string
	for i := 0; i < 4; i++ {
		if _, err := os.Stat(filepath.Join(root, "Cargo.toml")); err == nil {
			roots = append(roots, root)
		}
		parent := filepath.Dir(root)
		if parent == root {
			break
		}
		root = parent
	}
	for _, root := range roots {
		_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil
			}
			if entry.IsDir() && path != root {
				name := entry.Name()
				if strings.HasPrefix(name, ".") || name == "target" || name == "dist" || name == "node_modules" {
					return filepath.SkipDir
				}
			}
			if entry.IsDir() || filepath.Ext(path) != ".rs" {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			for enumName, variants := range rustEnumOptions(string(data)) {
				if _, exists := out[enumName]; !exists {
					out[enumName] = variants
				}
			}
			return nil
		})
	}
	return out
}

func rustEnumOptions(content string) map[string][]string {
	out := make(map[string][]string)
	content = regexp.MustCompile(`(?m)//[^\n]*`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`(?s)/\*.*?\*/`).ReplaceAllString(content, "")
	re := regexp.MustCompile(`(?s)\benum\s+([A-Za-z_][A-Za-z0-9_]*)\s*\{([^}]*)\}`)
	for _, match := range re.FindAllStringSubmatch(content, -1) {
		var variants []string
		valid := true
		for _, part := range strings.Split(match[2], ",") {
			part = strings.TrimSpace(regexp.MustCompile(`(?m)^\s*#\[[^]]*\]\s*`).ReplaceAllString(part, ""))
			if part == "" {
				continue
			}
			parts := strings.FieldsFunc(part, func(r rune) bool { return r == '(' || r == '{' || r == '=' || r == ' ' || r == '\n' || r == '\t' })
			if len(parts) == 0 {
				valid = false
				break
			}
			name := parts[0]
			if strings.Contains(part, "(") || strings.Contains(part, "{") || !regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`).MatchString(name) {
				valid = false
				break
			}
			variants = append(variants, name)
		}
		if valid && len(variants) > 0 {
			out[match[1]] = variants
		}
	}
	return out
}

func rustEnumParamType(typeName string, enums map[string][]string) (string, bool) {
	t := strings.TrimSpace(strings.TrimPrefix(typeName, "&"))
	if strings.HasPrefix(t, "Vec<") && strings.HasSuffix(t, ">") {
		name := strings.TrimSpace(t[4 : len(t)-1])
		if _, ok := enums[name]; ok {
			return name, true
		}
		return "", false
	}
	if _, ok := enums[t]; ok {
		return t, false
	}
	return "", false
}
