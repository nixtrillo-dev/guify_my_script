package config_gen

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"guify_my_script/internal_utils"
)

type jsDocParam struct {
	Type    string
	Default string
}

type jsParam struct {
	Name     string
	TypeHint string
	Default  string
}

var jsFunctionRe = regexp.MustCompile(`(?m)^(?:export\s+(?:default\s+)?)?(?:async\s+)?function\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*\(`)
var jsInlineTypeRe = regexp.MustCompile(`/\*\*\s*@type\s*\{([^}]+)\}\s*\*/`)

func parseJavaScriptFile(path string) ([]UIAction, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := string(data)
	var actions []UIAction

	matches := jsFunctionRe.FindAllStringSubmatchIndex(content, -1)
	for _, loc := range matches {
		name := content[loc[2]:loc[3]]
		if strings.HasPrefix(name, "_") || name == "main" || name == "usage" || name == "cleanup" {
			continue
		}

		openParen := loc[1] - 1
		paramsRaw, closeParen, ok := extractBalanced(content, openIdxOf(content, openParen, '('))
		if !ok {
			continue
		}
		_ = closeParen

		jsdoc := precedingJSDoc(content, loc[0])
		docs := parseJSDocParams(jsdoc)

		params := splitPythonParams(stripJSComments(paramsRaw))
		fields, invokeAsObject := jsFieldsFromParams(params, docs)

		actions = append(actions, UIAction{
			Label:                     internal_utils.ToTitleCase(name),
			Func:                      name,
			ExecutionMode:             "GUI",
			RefreshFieldsOnTabOpen:    emptyArrayJSON(),
			RefreshFieldsOnCompletion: emptyArrayJSON(),
			QuickLaunch:               false,
			Width:                     600,
			Sudo:                      false,
			InvokeAsObject:           invokeAsObject,
			ExecLabel:                 fmt.Sprintf("Execute %s", internal_utils.ToTitleCase(name)),
			ExecInProgressLabel:       fmt.Sprintf("Executing %s...", internal_utils.ToTitleCase(name)),
			StopLabel:                 "Stop",
			Fields:                    fields,
		})
	}

	return actions, nil
}

func openIdxOf(content string, hint int, open byte) int {
	if hint >= 0 && hint < len(content) && content[hint] == open {
		return hint
	}
	for i := hint; i >= 0 && i < len(content); i-- {
		if content[i] == open {
			return i
		}
	}
	return hint
}

func jsFieldsFromParams(params []string, docs map[string]jsDocParam) ([]UIField, bool) {
	objectProps, isObject := jsObjectProperties(params, docs)
	if isObject {
		var fields []UIField
		for _, prop := range objectProps {
			hint, def := jsDocHintAndDefault(prop.Name, docs)
			if prop.TypeHint != "" {
				hint = prop.TypeHint
			}
			if prop.Default != "" {
				def = prop.Default
			}
			f := createJSField(prop.Name, hint, def)
			if f.Name != "" {
				fields = append(fields, f)
			}
		}
		return fields, true
	}

	var fields []UIField
	for _, raw := range params {
		p := parseJSPositionalParam(raw)
		if p.Name == "" {
			continue
		}
		hint, def := jsDocHintAndDefault(p.Name, docs)
		if p.TypeHint != "" {
			hint = p.TypeHint
		}
		if p.Default != "" {
			def = p.Default
		}
		f := createJSField(p.Name, hint, def)
		if f.Name != "" {
			fields = append(fields, f)
		}
	}
	return fields, false
}

func jsObjectProperties(params []string, docs map[string]jsDocParam) ([]jsParam, bool) {
	cleaned := make([]string, 0, len(params))
	for _, p := range params {
		p = strings.TrimSpace(p)
		if p != "" {
			cleaned = append(cleaned, p)
		}
	}
	if len(cleaned) == 0 {
		return nil, false
	}

	if len(cleaned) == 1 {
		if props, ok := parseDestructuredObjectParam(cleaned[0]); ok {
			return props, true
		}
		ident := parseJSPositionalParam(cleaned[0])
		if ident.Name != "" {
			nested := jsDocNestedProps(ident.Name, docs)
			if len(nested) > 0 {
				return nested, true
			}
		}
	}
	return nil, false
}

func jsDocNestedProps(parent string, docs map[string]jsDocParam) []jsParam {
	prefix := parent + "."
	var keys []string
	for key := range docs {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	var props []jsParam
	for _, key := range keys {
		doc := docs[key]
		props = append(props, jsParam{
			Name:     strings.TrimPrefix(key, prefix),
			TypeHint: doc.Type,
			Default:  doc.Default,
		})
	}
	return props
}

func jsDocHintAndDefault(name string, docs map[string]jsDocParam) (string, string) {
	if d, ok := docs[name]; ok {
		return d.Type, d.Default
	}
	for key, d := range docs {
		if strings.HasSuffix(key, "."+name) {
			return d.Type, d.Default
		}
	}
	return "", ""
}

func parseDestructuredObjectParam(param string) ([]jsParam, bool) {
	param = strings.TrimSpace(param)
	if !strings.HasPrefix(param, "{") {
		return nil, false
	}
	inner, _, ok := extractBalanced(param, 0)
	if !ok {
		return nil, false
	}
	var props []jsParam
	for _, raw := range splitPythonParams(inner) {
		p := parseJSPositionalParam(raw)
		if p.Name == "" || p.Name == "this" {
			continue
		}
		props = append(props, p)
	}
	if len(props) == 0 {
		return nil, false
	}
	return props, true
}

func parseJSPositionalParam(raw string) jsParam {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "this" || strings.HasPrefix(raw, "...") {
		return jsParam{}
	}

	typeHint := ""
	if m := jsInlineTypeRe.FindStringSubmatch(raw); len(m) == 2 {
		typeHint = strings.TrimSpace(m[1])
		raw = strings.TrimSpace(jsInlineTypeRe.ReplaceAllString(raw, ""))
	}

	def := ""
	nameAndType := raw
	if eq := indexJSDefaultEquals(raw); eq != -1 {
		def = strings.TrimSpace(raw[eq+1:])
		nameAndType = strings.TrimSpace(raw[:eq])
	}

	parts := strings.SplitN(nameAndType, ":", 2)
	name := strings.TrimSpace(parts[0])
	if len(parts) > 1 && typeHint == "" {
		typeHint = strings.TrimSpace(parts[1])
	}
	if name == "" || name == "this" || strings.HasPrefix(name, "...") || strings.HasPrefix(name, "{") {
		return jsParam{}
	}
	if !isJSIdent(name) {
		return jsParam{}
	}
	return jsParam{Name: name, TypeHint: typeHint, Default: def}
}

func indexJSDefaultEquals(raw string) int {
	depthParen, depthBracket, depthBrace := 0, 0, 0
	inSingle, inDouble, escaped := false, false, false
	for i, r := range raw {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' && (inSingle || inDouble) {
			escaped = true
			continue
		}
		if inSingle {
			if r == '\'' {
				inSingle = false
			}
			continue
		}
		if inDouble {
			if r == '"' {
				inDouble = false
			}
			continue
		}
		switch r {
		case '\'':
			inSingle = true
		case '"':
			inDouble = true
		case '(':
			depthParen++
		case ')':
			depthParen--
		case '[':
			depthBracket++
		case ']':
			depthBracket--
		case '{':
			depthBrace++
		case '}':
			depthBrace--
		case '=':
			if depthParen == 0 && depthBracket == 0 && depthBrace == 0 {
				return i
			}
		}
	}
	return -1
}

func isJSIdent(name string) bool {
	if name == "" {
		return false
	}
	for i, r := range name {
		if i == 0 {
			if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '_' || r == '$') {
				return false
			}
			continue
		}
		if !((r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '$') {
			return false
		}
	}
	return true
}

func precedingJSDoc(content string, fnStart int) string {
	i := fnStart
	for i > 0 && (content[i-1] == ' ' || content[i-1] == '\t' || content[i-1] == '\n' || content[i-1] == '\r') {
		i--
	}
	if i < 2 || content[i-2:i] != "*/" {
		return ""
	}
	start := strings.LastIndex(content[:i], "/**")
	if start < 0 {
		return ""
	}
	return content[start:i]
}

func parseJSDocParams(block string) map[string]jsDocParam {
	docs := make(map[string]jsDocParam)
	if block == "" {
		return docs
	}
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "*")
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "@param") {
			continue
		}
		rest := strings.TrimSpace(strings.TrimPrefix(line, "@param"))
		if !strings.HasPrefix(rest, "{") {
			continue
		}
		typeHint, after, ok := extractBalanced(rest, 0)
		if !ok {
			continue
		}
		rest = strings.TrimSpace(after)
		name, def := parseJSDocNameAndDefault(rest)
		if name == "" {
			continue
		}
		docs[name] = jsDocParam{Type: strings.TrimSpace(typeHint), Default: def}
	}
	return docs
}

func parseJSDocNameAndDefault(rest string) (string, string) {
	if strings.HasPrefix(rest, "[") {
		inner, _, ok := extractBalanced(rest, 0)
		if !ok {
			return "", ""
		}
		eq := indexJSDefaultEquals(inner)
		if eq == -1 {
			return strings.TrimSpace(inner), ""
		}
		return strings.TrimSpace(inner[:eq]), strings.TrimSpace(inner[eq+1:])
	}
	fields := strings.Fields(rest)
	if len(fields) == 0 {
		return "", ""
	}
	return fields[0], ""
}

func extractBalanced(s string, openIdx int) (inner string, after string, ok bool) {
	if openIdx < 0 || openIdx >= len(s) {
		return "", "", false
	}
	open := s[openIdx]
	var close byte
	switch open {
	case '(':
		close = ')'
	case '{':
		close = '}'
	case '[':
		close = ']'
	default:
		return "", "", false
	}

	depth := 0
	inSingle, inDouble, inTemplate := false, false, false
	escaped := false
	for i := openIdx; i < len(s); i++ {
		c := s[i]
		if escaped {
			escaped = false
			continue
		}
		if c == '\\' && (inSingle || inDouble || inTemplate) {
			escaped = true
			continue
		}
		if inSingle {
			if c == '\'' {
				inSingle = false
			}
			continue
		}
		if inDouble {
			if c == '"' {
				inDouble = false
			}
			continue
		}
		if inTemplate {
			if c == '`' {
				inTemplate = false
			}
			continue
		}
		switch c {
		case '\'':
			inSingle = true
		case '"':
			inDouble = true
		case '`':
			inTemplate = true
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				return s[openIdx+1 : i], s[i+1:], true
			}
		}
	}
	return "", "", false
}

func stripJSComments(s string) string {
	var b strings.Builder
	inSingle, inDouble, inLine, inBlock, escaped := false, false, false, false, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inLine {
			if c == '\n' {
				inLine = false
				b.WriteByte(c)
			}
			continue
		}
		if inBlock {
			if c == '*' && i+1 < len(s) && s[i+1] == '/' {
				inBlock = false
				i++
			}
			continue
		}
		if escaped {
			b.WriteByte(c)
			escaped = false
			continue
		}
		if inSingle {
			b.WriteByte(c)
			if c == '\\' {
				escaped = true
			} else if c == '\'' {
				inSingle = false
			}
			continue
		}
		if inDouble {
			b.WriteByte(c)
			if c == '\\' {
				escaped = true
			} else if c == '"' {
				inDouble = false
			}
			continue
		}
		if c == '/' && i+1 < len(s) && s[i+1] == '/' {
			inLine = true
			i++
			continue
		}
		if c == '/' && i+1 < len(s) && s[i+1] == '*' {
			inBlock = true
			i++
			continue
		}
		if c == '\'' {
			inSingle = true
			b.WriteByte(c)
			continue
		}
		if c == '"' {
			inDouble = true
			b.WriteByte(c)
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

func createJSField(paramName, typeHint, defaultLiteral string) UIField {
	if paramName == "" {
		return UIField{}
	}

	dataType, fieldType, defaultVal, decPlaces := jsTypeToField(typeHint, defaultLiteral)

	field := UIField{
		Name:         internal_utils.ToTitleCase(paramName),
		Type:         fieldType,
		DataType:     dataType,
		ArgName:      paramName,
		Default:      defaultVal,
		AdvancedMenu: false,
		Enabled:      true,
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
	if fieldType == "NUM" {
		field.Minimum = nullJSON()
		field.Maximum = nullJSON()
		field.DecimalPlaces = decPlaces
	}
	return field
}

func jsTypeToField(typeHint, defaultLiteral string) (dataType, fieldType string, defaultVal any, decPlaces *string) {
	dataType = "string"
	fieldType = "TXT"
	defaultVal = "Initial Value"

	inferredFromDefault := ""
	if lit := strings.TrimSpace(defaultLiteral); lit != "" {
		defaultVal, inferredFromDefault = jsLiteralToDefault(lit)
	}

	mapped := mapJSTypeHint(typeHint)
	if mapped == "" {
		mapped = inferredFromDefault
	}
	if mapped == "" {
		mapped = "string"
	}

	switch mapped {
	case "int":
		dataType = "int"
		fieldType = "NUM"
		if strings.TrimSpace(defaultLiteral) == "" {
			defaultVal = "0"
		}
		zero := "0"
		decPlaces = &zero
	case "float":
		dataType = "float"
		fieldType = "NUM"
		if strings.TrimSpace(defaultLiteral) == "" {
			defaultVal = "0.0"
		}
		two := "2"
		decPlaces = &two
	case "bool":
		dataType = "bool"
		fieldType = "CK"
		if strings.TrimSpace(defaultLiteral) == "" {
			defaultVal = "0"
		}
	case "list":
		dataType = "list"
		fieldType = "TXT"
		if strings.TrimSpace(defaultLiteral) == "" {
			defaultVal = ""
		}
	case "dict":
		dataType = "dict"
		fieldType = "TXT"
		if strings.TrimSpace(defaultLiteral) == "" {
			defaultVal = ""
		}
	default:
		dataType = "string"
		fieldType = "TXT"
		if strings.TrimSpace(defaultLiteral) == "" {
			defaultVal = "Initial Value"
		}
	}
	return dataType, fieldType, defaultVal, decPlaces
}

func mapJSTypeHint(typeHint string) string {
	typeHint = strings.ToLower(strings.TrimSpace(typeHint))
	if typeHint == "" {
		return ""
	}
	parts := strings.Split(typeHint, "|")
	kinds := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.TrimPrefix(p, "undefined")
		p = strings.TrimSpace(strings.TrimPrefix(p, "?"))
		if p == "" || p == "undefined" || p == "null" || p == "void" {
			continue
		}
		kinds = append(kinds, classifyJSType(p))
	}
	if len(kinds) == 0 {
		return ""
	}
	uniq := map[string]bool{}
	for _, k := range kinds {
		uniq[k] = true
	}
	if uniq["string"] && len(uniq) > 1 {
		return "string"
	}
	if len(uniq) == 1 {
		for k := range uniq {
			return k
		}
	}
	return "string"
}

func classifyJSType(p string) string {
	p = strings.TrimSpace(strings.ToLower(p))
	switch {
	case p == "int" || p == "integer":
		return "int"
	case p == "number" || p == "float" || p == "double":
		return "float"
	case p == "bool" || p == "boolean":
		return "bool"
	case strings.HasPrefix(p, "array") || strings.HasPrefix(p, "array<") || strings.HasPrefix(p, "array.") || p == "list":
		return "list"
	case p == "object" || p == "record" || p == "dict" || p == "map" || strings.HasPrefix(p, "record<"):
		return "dict"
	case p == "string" || p == "str":
		return "string"
	default:
		return "string"
	}
}

func jsLiteralToDefault(lit string) (any, string) {
	lit = strings.TrimSpace(lit)
	if lit == "true" || lit == "false" {
		if lit == "true" {
			return "1", "bool"
		}
		return "0", "bool"
	}
	if strings.HasPrefix(lit, "\"") && strings.HasSuffix(lit, "\"") && len(lit) >= 2 {
		return lit[1 : len(lit)-1], "string"
	}
	if strings.HasPrefix(lit, "'") && strings.HasSuffix(lit, "'") && len(lit) >= 2 {
		return lit[1 : len(lit)-1], "string"
	}
	if strings.HasPrefix(lit, "`") && strings.HasSuffix(lit, "`") && len(lit) >= 2 {
		return lit[1 : len(lit)-1], "string"
	}
	if strings.HasPrefix(lit, "[") {
		return lit, "list"
	}
	if strings.HasPrefix(lit, "{") {
		return lit, "dict"
	}
	if strings.ContainsAny(lit, ".eE") && isJSNumber(lit) {
		return lit, "float"
	}
	if isJSNumber(lit) {
		return lit, "int"
	}
	return lit, "string"
}

func isJSNumber(s string) bool {
	if s == "" {
		return false
	}
	_, err := fmt.Sscan(s, new(float64))
	return err == nil
}
