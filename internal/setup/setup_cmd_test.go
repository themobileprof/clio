package setup

import "testing"

func TestResolveSetup(t *testing.T) {
	cases := []struct {
		input    string
		wantKind MatchKind
		wantID   string
	}{
		{"setup", MatchMenu, ""},
		{"wizards", MatchMenu, ""},
		{"termux setup", MatchWizard, "termux"},
		{"setup vim", MatchWizard, "vim"},
		{"setup git", MatchWizard, "git"},
		{"how do I setup termux for coding", MatchWizard, "termux"},
		{"abeg help me setup termux", MatchWizard, "termux"},
		{"configure my dev environment", MatchWizard, "termux"},
		{"setup vim plugins", MatchWizard, "vim"},
		{"install postgres", MatchWizard, "database"},
		{"setup devtools", MatchWizard, "devtools"},
		{"list files", MatchNone, ""},
		{"check disk space", MatchNone, ""},
	}
	for _, c := range cases {
		kind, w := ResolveSetup(c.input)
		if kind != c.wantKind {
			t.Errorf("ResolveSetup(%q) kind = %v, want %v", c.input, kind, c.wantKind)
			continue
		}
		gotID := ""
		if w != nil {
			gotID = w.ID
		}
		if gotID != c.wantID {
			t.Errorf("ResolveSetup(%q) wizard = %q, want %q", c.input, gotID, c.wantID)
		}
	}
}

func TestRunCommandFor(t *testing.T) {
	w := WizardByID("vim")
	if w == nil {
		t.Fatal("vim wizard not found")
	}
	want := "clio-run-module vim_setup setup"
	if got := RunCommandFor(*w); got != want {
		t.Fatalf("RunCommandFor(vim) = %q, want %q", got, want)
	}
}

func TestWizardFromCommand(t *testing.T) {
	w := WizardFromCommand("clio-run-module git_setup setup")
	if w == nil || w.ID != "git" {
		t.Fatalf("WizardFromCommand git = %v", w)
	}
}

func TestAllWizardsCount(t *testing.T) {
	if n := len(AllWizards()); n < 5 {
		t.Fatalf("expected at least 5 wizards, got %d", n)
	}
}
