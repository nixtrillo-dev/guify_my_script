package gui_gen

import (
	"strings"
	"testing"
)

func TestBuildExecutionLambda_PythonUsesTypedAndShellSafeRunner(t *testing.T) {
	cpp := buildExecutionLambda(UIAction{Label: "Greet", Func: "greet_user", Mode: "g"}, "Greet", "0", "fields_greet_0", "python")

	for _, expected := range []string{
		"formatPythonArg(rawVal, dataType)",
		"escapePythonString(SOURCE_SCRIPT)",
		"shellQuote(pyCode)",
		"python3 -u -c %1",
		"import sys, os; sys.path.insert(0, os.path.dirname(%1)); import %2; %2.greet_user(%3)",
	} {
		if !strings.Contains(cpp, expected) {
			t.Errorf("Python execution lambda is missing %q", expected)
		}
	}
}

func TestBuildExecutionLambda_JavaScriptUsesTypedJsonAndNodeRunner(t *testing.T) {
	cpp := buildExecutionLambda(UIAction{Label: "Multiply", Func: "multiply", Mode: "g"}, "Multiply", "0", "fields_multiply_0", "javascript")

	for _, expected := range []string{
		"formatJsonArg(rawVal, dataType)",
		"shellQuote(jsRunner)",
		"node --input-type=module -e %1",
		"pathToFileURL",
		"import(pathToFileURL(script).href)",
		"JSON.parse(s)",
	} {
		if !strings.Contains(cpp, expected) {
			t.Errorf("JavaScript execution lambda is missing %q", expected)
		}
	}
}

func TestBuildExecutionLambda_PowerShellUsesLiteralImportAndJsonArgs(t *testing.T) {
	cpp := buildExecutionLambda(UIAction{Label: "Filter", Func: "Invoke-CustomFilter", Mode: "g"}, "Filter", "0", "fields_filter_0", "powershell")
	for _, expected := range []string{
		"formatPowerShellArg(rawVal, dataType)",
		"Import-Module -Name $scriptPath -Force",
		"FromBase64String",
		"ConvertFrom-Json",
		"$callArgs = @{}",
		"$callArgs[$parameterName]",
		"pwsh -NoProfile -NonInteractive -Command",
		"shellQuote(psRunner)",
		"& $functionName @callArgs",
	} {
		if !strings.Contains(cpp, expected) {
			t.Errorf("PowerShell execution lambda is missing %q", expected)
		}
	}
}

func TestBuildExecutionLambda_GUICommandlineWindowsShowsConsoleAndCapturesOutput(t *testing.T) {
	cpp := buildExecutionLambda(UIAction{Label: "Run", Func: "run", Mode: "gc"}, "Run", "0", "fields_run_0", "rust")
	for _, expected := range []string{
		"configureTerminalProcess(proc, consoleCommand, false, false, true)",
		"createCommandOutputLogPath()",
		"drainCommandLog()",
		"QProcess::MergedChannels",
		"configureActionProcess(proc, shell_cmd",
	} {
		if !strings.Contains(cpp, expected) {
			t.Errorf("GUI_COMMANDLINE execution lambda is missing %q", expected)
		}
	}
	if !strings.Contains(cppTemplateUtilsH, "QProcess::ForwardedChannels") {
		t.Error("Windows terminal helper must forward console output while the GUI tails its output log")
	}
}

func TestBuildExecutionLambda_CommandlineWindowsStartsUntrackedConsole(t *testing.T) {
	cpp := buildExecutionLambda(UIAction{Label: "Run", Func: "run", Mode: "c"}, "Run", "0", "fields_run_0", "python")
	for _, expected := range []string{
		"configureTerminalProcess(proc, shell_cmd, false, true)",
		"if (!proc->startDetached())",
		"proc->deleteLater();",
	} {
		if !strings.Contains(cpp, expected) {
			t.Errorf("COMMANDLINE execution lambda is missing %q", expected)
		}
	}
}

func TestMultiValueFieldsUseQuotedCSV(t *testing.T) {
	want := `"one, two","say ""hello"""`
	got := joinCSVValues([]string{"one, two", `say "hello"`})
	if got != want {
		t.Fatalf("joinCSVValues() = %q, want %q", got, want)
	}
	if gotDefault := getDefaultString([]interface{}{"one, two", `say "hello"`}); gotDefault != want {
		t.Fatalf("getDefaultString() = %q, want %q", gotDefault, want)
	}
	for _, expected := range []string{
		"QStringList parseCsvValues(const QString& rawVal)",
		"QString joinCsvValues(const QStringList& values)",
		"QStringList parts = parseCsvValues(rawVal)",
	} {
		if !strings.Contains(cppTemplateUtilsH, expected) {
			t.Errorf("Qt CSV helpers are missing %q", expected)
		}
	}
}

func TestBuildExecutionLambda_JavaScriptInvokeAsObject(t *testing.T) {
	fields := []UIField{
		{Name: "A", ArgName: "a", Type: "TXT", DataType: "string", Default: "hello"},
		{Name: "B", ArgName: "b", Type: "NUM", DataType: "float", Default: "14"},
	}
	action := UIAction{
		Label:          "Invoke With Object Test",
		Func:           "invokeWithObjectTest",
		Mode:           "g",
		InvokeAsObject: true,
		Fields:         fields,
	}
	cpp := buildExecutionLambda(action, "Invoke With Object Test", "0", "fields_obj_0", "javascript")

	expectedSubstrings := []string{
		"formatJsonArg(rawVal, dataType)",
		"jsParts.append",
		"jsKeys",
		"args.at(i)",
		"args = QStringList() << (\"{\" + jsParts.join(\",\") + \"}\");",
		"invokeWithObjectTest",
	}
	for _, expected := range expectedSubstrings {
		if !strings.Contains(cpp, expected) {
			t.Errorf("JavaScript invoke-as-object lambda is missing %q\n\nSnippet around jsParts:\n%s",
				expected, extractSnippet(cpp, "jsParts"))
		}
	}

	keyChecks := []string{"a", "b"}
	for _, k := range keyChecks {
		if !strings.Contains(cpp, "QStringLiteral") || !strings.Contains(cpp, k) {
			t.Errorf("Expected key %q to be referenced with QStringLiteral in generated C++", k)
		}
	}

	if strings.Contains(cpp, "buildJavaScriptObjectWrapCPP") {
		t.Error("Expected buildJavaScriptObjectWrapCPP call was inlined, should not appear literally in generated C++")
	}
}

func extractSnippet(s, marker string) string {
	i := strings.Index(s, marker)
	if i < 0 {
		return "(marker not found)"
	}
	start := i - 100
	if start < 0 {
		start = 0
	}
	end := i + 300
	if end > len(s) {
		end = len(s)
	}
	return s[start:end]
}

func TestBuildJavaScriptShellCmdCPP_Modes(t *testing.T) {
	cppGui := buildJavaScriptShellCmdCPP("g", "myFunc", "", "SOURCE_SCRIPT", true)
	if !strings.Contains(cppGui, `"g" == "g" || "g" == "s" || "g" == "d"`) {
		t.Error("GUI mode condition should include g/s/d")
	}

	cppAdv := buildJavaScriptShellCmdCPP("g", "myFunc", "", "targetScript", false)
	if !strings.Contains(cppAdv, `"g" == "s" || "g" == "d"`) {
		t.Error("Advanced mode condition should include only s/d")
	}
	if strings.Contains(cppAdv, `"g" == "g" ||`) {
		t.Error("Advanced mode should not include GUI-only modes")
	}

	for _, expected := range []string{
		"node --input-type=module -e",
		"shellQuote(SOURCE_SCRIPT)",
		"shellQuote(QStringLiteral(\"myFunc\"))",
	} {
		if !strings.Contains(cppGui, expected) {
			t.Errorf("JavaScript shell cmd missing %q", expected)
		}
	}
}

func TestBuildJavaScriptObjectWrapCPP_OnlyActiveForJavaScriptWithObject(t *testing.T) {
	empty := buildJavaScriptObjectWrapCPP(UIAction{Label: "X", Func: "x"}, "python")
	if empty != "" {
		t.Error("Non-JavaScript should produce empty object wrap")
	}

	jsNoFlag := buildJavaScriptObjectWrapCPP(UIAction{Label: "X", Func: "x", Fields: []UIField{{Name: "Y"}}}, "javascript")
	if jsNoFlag != "" {
		t.Error("JavaScript without InvokeAsObject should produce empty object wrap")
	}

	jsObjNoFields := buildJavaScriptObjectWrapCPP(UIAction{Label: "X", Func: "x", InvokeAsObject: true}, "javascript")
	if jsObjNoFields != "" {
		t.Error("JavaScript with object but no fields should produce empty wrap")
	}

	jsWithFields := buildJavaScriptObjectWrapCPP(UIAction{
		Label:          "X",
		Func:           "x",
		InvokeAsObject: true,
		Fields:         []UIField{{Name: "First", ArgName: "first"}},
	}, "javascript")
	if jsWithFields == "" {
		t.Error("Expected non-empty object wrap code for JavaScript with InvokeAsObject + fields")
	}
	if !strings.Contains(jsWithFields, "first") {
		t.Error("Expected arg_name 'first' in generated object wrap code")
	}
}

func TestBuildJavaScriptShellCmdCPP_LonghandAndCaseInsensitive(t *testing.T) {
	for _, mode := range []string{"GUI", "gui", "Gui", "g", "G"} {
		cpp := buildJavaScriptShellCmdCPP(mode, "testFn", "", "SOURCE_SCRIPT", true)
		if !strings.Contains(cpp, `"g" == "g" || "g" == "s" || "g" == "d"`) {
			t.Errorf("Mode %q should normalize to 'g' condition in GUI mode", mode)
		}
	}

	for _, mode := range []string{"SILENT", "silent", "Silent", "s", "S"} {
		cpp := buildJavaScriptShellCmdCPP(mode, "testFn", "", "SOURCE_SCRIPT", false)
		if !strings.Contains(cpp, `"s" == "s" || "s" == "d"`) {
			t.Errorf("Mode %q should normalize to 's' condition in non-GUI mode", mode)
		}
	}
}

func TestBuildExecutionLambda_ExecutionModeLonghand(t *testing.T) {
	// Test with longhand ExecutionMode: "GUI"
	cppGui := buildExecutionLambda(UIAction{Label: "Test", Func: "test_fn", ExecutionMode: "GUI"}, "Test", "0", "fields_0", "bash")
	if !strings.Contains(cppGui, "logOutputWindow_0") {
		t.Errorf("ExecutionMode 'GUI' should set up logOutputWindow")
	}

	// Test with longhand ExecutionMode: "SILENT"
	cppSilent := buildExecutionLambda(UIAction{Label: "Test", Func: "test_fn", ExecutionMode: "SILENT"}, "Test", "0", "fields_0", "bash")
	if strings.Contains(cppSilent, "logOutputWindow_0") {
		t.Errorf("ExecutionMode 'SILENT' should not set up logOutputWindow")
	}

	// Test with longhand ExecutionMode: "DETACHED_COMMANDLINE"
	cppDetached := buildExecutionLambda(UIAction{Label: "Test", Func: "test_fn", ExecutionMode: "DETACHED_COMMANDLINE"}, "Test", "0", "fields_0", "bash")
	if !strings.Contains(cppDetached, "proc->startDetached();") {
		t.Errorf("ExecutionMode 'DETACHED_COMMANDLINE' should call startDetached")
	}
}
