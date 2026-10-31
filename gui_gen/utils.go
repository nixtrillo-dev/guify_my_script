package gui_gen

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func sanitizeIdentifier(text string) string {
	reg := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	return reg.ReplaceAllString(text, "_")
}

func formatArgsForQt(input string) string {
	parts := strings.Split(input, " ")
	var formatted []string
	for _, p := range parts {
		cleanArg := strings.Trim(p, "'\"")
		if cleanArg != "" {
			formatted = append(formatted, fmt.Sprintf("\"%s\"", cleanArg))
		}
	}
	if len(formatted) == 0 {
		return "QStringList()"
	}
	return "QStringList() << " + strings.Join(formatted, " << ")
}

func cppStringLiteral(input string) string {
	escaped := strings.ReplaceAll(input, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, `"`, `\"`)
	escaped = strings.ReplaceAll(escaped, "\n", `\n`)
	escaped = strings.ReplaceAll(escaped, "\r", `\r`)
	return escaped
}

func sanitizePackageName(input string) string {
	name := strings.ToLower(strings.TrimSpace(input))
	reg := regexp.MustCompile(`[^a-z0-9._+-]+`)
	name = reg.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-._+")
	if name == "" {
		return "default-app"
	}
	return name
}

func copyFile(src, dst string, mode os.FileMode) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	return os.WriteFile(dst, data, mode)
}

func quoteDesktopExec(path string) string {
	return "'" + strings.ReplaceAll(path, "'", "'\\''") + "'"
}

func getDefaultString(val interface{}) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	case []interface{}:
		var parts []string
		for _, item := range v {
			parts = append(parts, fmt.Sprintf("%v", item))
		}
		return joinCSVValues(parts)
	case []string:
		return joinCSVValues(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func joinCSVValues(values []string) string {
	encoded := make([]string, len(values))
	for i, value := range values {
		encoded[i] = `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
	}
	return strings.Join(encoded, ",")
}
