package modules

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestBashScriptSkipsMapFlows(t *testing.T) {
	yaml := `
name: list_directory
id: org.themobileprof.list_directory
version: 1.0.0
description: List a directory
flows:
  main:
    start: list_contents
    steps:
      list_contents:
        type: action
        command: ls
`
	script, err := convertYAMLToBashScript(yaml)
	if err != nil {
		t.Fatal(err)
	}
	if script != "" {
		t.Fatalf("map-style flows should not become a wizard script:\n%s", script)
	}
}

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

	out, err := exec.Command("bash", "-c", `set -euo pipefail; source "$1"; printf '%s' "$STEP_0_COMMAND"`, "bash", path).Output()
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

func TestRunnerSkipsOptionalFlagsAndHonorsGoto(t *testing.T) {
	yaml := `
name: Demo
description: demo
version: "1"
estimated_time: "1 minute"
flows:
  - name: setup
    description: demo flow
    steps:
      - type: confirm
        prompt: "Continue?"
        default: yes
        on_no: abort
      - type: section
        title: Checks
        steps:
          - type: command
            command: echo RAN_UPDATE
            show_output: true
          - type: check_command
            command: bash
            on_exists: skip_install
          - type: command
            command: echo SHOULD_NOT_RUN
            show_output: true
          - type: label
            name: skip_install
          - type: confirm
            prompt: "Install extra?"
            default: no
            on_no: skip_extra
          - type: command
            command: echo INSTALLED_EXTRA
            show_output: true
          - type: label
            name: skip_extra
          - type: command
            command: echo AFTER_SKIP
            show_output: true
          - type: command
            command: |
              echo "unset:$NOT_A_REAL_VAR"
              yes | head -n 1
            show_output: true
      - type: file_operation
        operation: mark_complete
`
	script, err := convertYAMLToBashScript(yaml)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(script, "STEP_1_SUB_1_ON_EXISTS") || !strings.Contains(script, "STEP_1_SUB_4_ON_NO") {
		t.Fatalf("control fields were not stored:\n%s", script)
	}

	install, err := os.ReadFile(filepath.Join("..", "..", "install.sh"))
	if err != nil {
		t.Fatal(err)
	}
	const start = "cat > \"$INSTALL_DIR/clio-run-module\" <<'EOMODRUNNER'\n"
	const end = "\nEOMODRUNNER"
	raw := string(install)
	i := strings.Index(raw, start)
	j := strings.Index(raw[i+len(start):], end)
	if i < 0 || j < 0 {
		t.Fatal("could not find the module runner in install.sh")
	}
	runner := raw[i+len(start) : i+len(start)+j]

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "clio-run-module"), []byte(runner), 0o755); err != nil {
		t.Fatal(err)
	}
	scriptPath := filepath.Join(dir, "module.sh")
	if err := os.WriteFile(scriptPath, []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}
	// The runner calls: "$BIN" --print-module "$MODULE_ID"
	if err := os.WriteFile(filepath.Join(dir, "clio.bin"), []byte("#!/bin/bash\ncat \""+scriptPath+"\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(filepath.Join(dir, "clio-run-module"), "demo")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "HOME="+dir)
	cmd.Stdin = strings.NewReader("y\n\n")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("runner failed: %v\n%s", err, out)
	}
	got := string(out)
	if !strings.Contains(got, "RAN_UPDATE") || !strings.Contains(got, "AFTER_SKIP") || !strings.Contains(got, "unset:") || !strings.Contains(got, "\ny\n") && !strings.Contains(got, "y\n") {
		t.Fatalf("wizard did not finish the section:\n%s", got)
	}
	if strings.Contains(got, "SHOULD_NOT_RUN") || strings.Contains(got, "INSTALLED_EXTRA") {
		t.Fatalf("a skipped step ran:\n%s", got)
	}
	if _, err := os.Stat(filepath.Join(dir, ".clio", "termux_setup_complete")); err != nil {
		t.Fatalf("setup was not marked complete: %v", err)
	}
}
