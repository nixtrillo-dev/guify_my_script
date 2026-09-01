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
