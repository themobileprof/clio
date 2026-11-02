package repl

import (
	"bytes"
	"clio/internal/layer3"
	"io"
	"os"
	"strings"
	"testing"
)

func TestWelcomeWithNoModulesOnlySaysSync(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("TERMUX_VERSION", "")

	got := captureStdout(t, printWelcome)
	if strings.TrimSpace(got) != "Type sync to download modules. When it finishes, type exit." {
		t.Fatalf("empty device welcome:\n%s", got)
	}

	if err := layer3.UpsertModule("termux_setup", "Termux", "setup", "termux", "1.2.3", "yaml", "bash"); err != nil {
		t.Fatal(err)
	}
	got = captureStdout(t, printWelcome)
	if strings.Contains(got, "Type sync to download modules") {
		t.Fatalf("welcome still asked for sync after a module was stored:\n%s", got)
	}
	if !strings.Contains(got, "CLIPilot Client") {
		t.Fatalf("welcome after sync:\n%s", got)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdout
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}
