package setup

import "testing"

func TestParseRunModuleCommand(t *testing.T) {
	cases := []struct {
		input    string
		moduleID string
		flow     string
		ok       bool
	}{
		{"clio-run-module termux_setup setup", "termux_setup", "setup", true},
		{"clio-run-module git_setup", "git_setup", "setup", true},
		{"clio-run-module copy_file run", "copy_file", "run", true},
		{"setup termux", "", "", false},
		{"ls -la", "", "", false},
	}
	for _, c := range cases {
		mod, flow, ok := ParseRunModuleCommand(c.input)
		if ok != c.ok || mod != c.moduleID || flow != c.flow {
			t.Errorf("ParseRunModuleCommand(%q) = (%q, %q, %v), want (%q, %q, %v)",
				c.input, mod, flow, ok, c.moduleID, c.flow, c.ok)
		}
	}
}

func TestWizardByModuleID(t *testing.T) {
	w := WizardByModuleID("termux_setup")
	if w == nil || w.ID != "termux" {
		t.Fatalf("WizardByModuleID(termux_setup) = %v", w)
	}
}
