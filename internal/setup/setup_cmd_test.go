package setup

import "testing"

func TestIsSetupRequest(t *testing.T) {
	cases := []struct {
		input string
		want  bool
	}{
		{"setup", true},
		{"termux setup", true},
		{"how do I setup termux for coding", true},
		{"abeg help me setup termux", true},
		{"configure my dev environment", true},
		{"list files", false},
		{"check disk space", false},
	}
	for _, c := range cases {
		if got := IsSetupRequest(c.input); got != c.want {
			t.Errorf("IsSetupRequest(%q) = %v, want %v", c.input, got, c.want)
		}
	}
}

func TestRunCommand(t *testing.T) {
	want := "clio-run-module termux_setup setup"
	if got := RunCommand(); got != want {
		t.Fatalf("RunCommand() = %q, want %q", got, want)
	}
}
