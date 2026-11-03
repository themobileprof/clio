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
	asked := filepath.Join(dir, "asked")
	bin := "#!/bin/bash\nread -r q || true\nprintf '%s\\n' \"$q\" > \"" + asked + "\"\n" +
		"if [ \"$q\" = \"extract tar file\" ]; then printf '%s\\n' 'tar -xzvf'; exit 0; fi\nexit 1\n"
	if err := os.WriteFile(filepath.Join(dir, "clio.bin"), []byte(bin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "clio-run-module"), []byte("#!/bin/bash\necho MODULE \"$@\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	record := filepath.Join(dir, "llm-args")
	keys := filepath.Join(dir, "llm-keys")
	llm := "#!/bin/bash\nprintf '%s\\n' \"$*\" >> \"" + record + "\"\n" +
		"if [ \"$1\" = \"-m\" ] && [ -f \"$HOME/llm-fail\" ]; then exit 1; fi\n" +
		"case \"$1\" in\n" +
		"  keys)\n" +
		"    if [ \"${2:-}\" = \"\" ]; then cat \"" + keys + "\" 2>/dev/null || true; fi\n" +
		"    ;;\n" +
		"  models)\n" +
		"    printf '%s\\n' deepseek/deepseek-chat gemini-3.8-flash\n" +
		"    ;;\n" +
		"esac\n"
	if err := os.WriteFile(filepath.Join(dir, "python"), []byte("#!/bin/bash\nif [ -f \"$HOME/offline\" ]; then exit 1; fi\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "llm"), []byte(llm), 0o755); err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(dir, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}

	run := func(args ...string) string {
		t.Helper()
		os.Remove(record)
		cmd := exec.Command(filepath.Join(dir, "clio"), args...)
		cmd.Env = append(os.Environ(), "PATH="+dir+":"+os.Getenv("PATH"), "HOME="+home)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("clio %v: %v\n%s", args, err, out)
		}
		return string(out)
	}
	runFail := func(args ...string) string {
		t.Helper()
		os.Remove(record)
		cmd := exec.Command(filepath.Join(dir, "clio"), args...)
		cmd.Env = append(os.Environ(), "PATH="+dir+":"+os.Getenv("PATH"), "HOME="+home)
		out, err := cmd.CombinedOutput()
		if err == nil {
			t.Fatalf("clio %v succeeded:\n%s", args, out)
		}
		return string(out)
	}
	wrote := func() string {
		t.Helper()
		b, err := os.ReadFile(record)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}

	if err := os.WriteFile(keys, []byte("deepseek\ngemini\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := run("ask", "extract tar file"); got != "" {
		t.Fatalf("online ask printed %q", got)
	}
	if body := wrote(); !strings.Contains(body, "-m deepseek/deepseek-chat extract tar file") {
		t.Fatalf("online question did not reach the model: %q", body)
	}

	if got := run("ask", "your question"); got != "" {
		t.Fatalf("ask printed %q", got)
	}
	if body := wrote(); !strings.Contains(body, "-m deepseek/deepseek-chat your question") {
		t.Fatalf("default model args: %q", body)
	}

	if got := run("model", "gemini"); !strings.Contains(got, "Clio will ask gemini.") {
		t.Fatalf("model gemini: %q", got)
	}
	if got := run("ask", "your question"); got != "" {
		t.Fatalf("gemini ask printed %q", got)
	}
	if body := wrote(); !strings.Contains(body, "-m gemini-3.8-flash your question") {
		t.Fatalf("gemini model args: %q", body)
	}

	if err := os.Remove(filepath.Join(home, ".config", "clio", "model")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keys, []byte("gemini\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("ask", "your question")
	if body := wrote(); !strings.Contains(body, "-m gemini-3.8-flash your question") {
		t.Fatalf("available gemini args: %q", body)
	}

	if got := run("key", "deepseek"); got != "" {
		t.Fatalf("key printed %q", got)
	}
	if body := wrote(); !strings.Contains(body, "keys set deepseek") || strings.Contains(body, "install ") {
		t.Fatalf("key command: %q", body)
	}

	if got := run("termux_setup"); !strings.Contains(got, "MODULE termux_setup") {
		t.Fatalf("wizard was not run: %s", got)
	}
	if _, err := os.Stat(record); !os.IsNotExist(err) {
		t.Fatalf("wizard called the model tool")
	}

	got := runFail("what", "is", "html")
	if !strings.Contains(got, "clio ask") {
		t.Fatalf("sentence usage: %q", got)
	}
	if _, err := os.Stat(record); !os.IsNotExist(err) {
		t.Fatalf("a sentence called the model tool")
	}

	got = runFail("key")
	if !strings.Contains(got, "clio key deepseek") || !strings.Contains(got, "clio key gemini") {
		t.Fatalf("key usage: %q", got)
	}

	offline := filepath.Join(home, "offline")
	if err := os.WriteFile(offline, []byte("1"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keys, []byte("deepseek\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got = run("ask", "extract tar file")
	if !strings.Contains(got, "No internet. Offline answer:") || !strings.Contains(got, "Use: tar -xzvf") {
		t.Fatalf("offline fallback: %q", got)
	}
	if strings.Contains(wrote(), "-m ") {
		t.Fatal("no internet still called the model")
	}
	if err := os.Remove(offline); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(home, "llm-fail"), []byte("1"), 0o644); err != nil {
		t.Fatal(err)
	}
	got = run("ask", "extract tar file")
	if !strings.Contains(got, "The model did not answer. Offline answer:") || !strings.Contains(got, "Use: tar -xzvf") {
		t.Fatalf("failed model fallback: %q", got)
	}
	if !strings.Contains(wrote(), "-m deepseek/deepseek-chat extract tar file") {
		t.Fatal("failed model was not tried")
	}
}
