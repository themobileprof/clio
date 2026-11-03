package setup

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestClioAskWrapsTheModelTool(t *testing.T) {
	install, err := os.ReadFile(filepath.Join("..", "..", "install.sh"))
	if err != nil {
		t.Fatal(err)
	}
	const start = "cat > \"$INSTALL_DIR/clio\" <<'EOWRAPPER'\n"
	const end = "\nEOWRAPPER"
	raw := string(install)
	i := strings.Index(raw, start)
	j := strings.Index(raw[i+len(start):], end)
	if i < 0 || j < 0 {
		t.Fatal("could not find the clio wrapper in install.sh")
	}
	wrapper := raw[i+len(start) : i+len(start)+j]

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "clio"), []byte(wrapper), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "clio.bin"), []byte("#!/bin/bash\necho BIN\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "clio-run-module"), []byte("#!/bin/bash\necho MODULE \"$@\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	record := filepath.Join(dir, "llm-args")
	llm := "#!/bin/bash\nprintf '%s\n' \"$*\" > \"" + record + "\"\n"
	if err := os.WriteFile(filepath.Join(dir, "llm"), []byte(llm), 0o755); err != nil {
		t.Fatal(err)
	}

	run := func(args ...string) string {
		t.Helper()
		os.Remove(record)
		cmd := exec.Command(filepath.Join(dir, "clio"), args...)
		cmd.Env = append(os.Environ(), "PATH="+dir+":"+os.Getenv("PATH"))
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("clio %v: %v\n%s", args, err, out)
		}
		return string(out)
	}

	if got := run("ask", "your question"); got != "" {
		t.Fatalf("ask printed %q", got)
	}
	body, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "deepseek/deepseek-chat your question\n" && !strings.Contains(string(body), "-m deepseek/deepseek-chat your question") {
		t.Fatalf("model args: %q", body)
	}

	if got := run("key"); !strings.Contains(mustRead(t, record), "keys set deepseek") {
		t.Fatalf("key command: %s", got)
	}
	if got := run("termux_setup"); !strings.Contains(got, "MODULE termux_setup") {
		t.Fatalf("wizard was sent to the model: %s", got)
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
