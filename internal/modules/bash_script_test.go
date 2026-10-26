package modules

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBashScriptPreservesMultilineCommand(t *testing.T) {
	yaml := `
name: Sample
id: sample
version: "1"
description: multiline check
flows:
  - name: setup
    description: write a file
    steps:
      - type: command
        command: |
          echo hello
          echo '$KEEP'
          printf '%s' "quotes"
`
	script, err := convertYAMLToBashScript(yaml)
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "mod.sh")
	if err := os.WriteFile(path, []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := exec.Command("bash", "-c", `source "$1"; printf '%s' "$STEP_0_COMMAND"`, "bash", path).Output()
	if err != nil {
		t.Fatalf("source failed: %v\n%s", err, script)
	}

	got := string(out)
	if !strings.Contains(got, "echo hello\n") {
		t.Fatalf("newline was flattened:\n%s", got)
	}
	if !strings.Contains(got, "echo '$KEEP'") {
		t.Fatalf("dollar sign was expanded or dropped:\n%s", got)
	}
	if !strings.Contains(got, `printf '%s' "quotes"`) {
		t.Fatalf("quotes were not preserved:\n%s", got)
	}
}
