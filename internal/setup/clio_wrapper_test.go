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
		"    if [ \"${2:-}\" = \"path\" ]; then printf '%s\\n' \"$HOME/llm-keys.json\"; exit 0; fi\n" +
		"    if [ \"${2:-}\" = \"\" ]; then\n" +
		"      while IFS= read -r line; do printf '%s\\n' \"$line\"; sleep 0.05; done < \"" + keys + "\"\n" +
		"    fi\n" +
		"    ;;\n" +
		"  models)\n" +
		"    printf '%s\\n' deepseek-chat deepseek-flash gemini-3.8-flash\n" +
		"    ;;\n" +
		"esac\n"
	if err := os.WriteFile(filepath.Join(dir, "python"), []byte("#!/bin/bash\nprintf '%s\\n' \"$*\" >> \"$HOME/python-args\"\ncat >/dev/null\nif [ -f \"$HOME/key-status\" ]; then exit \"$(cat \"$HOME/key-status\")\"; fi\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "llm"), []byte(llm), 0o755); err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(dir, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}

	probe := filepath.Join(home, "python-args")
	execClio := func(in string, fail bool, args ...string) string {
		t.Helper()
		os.Remove(record)
		os.Remove(probe)
		cmd := exec.Command(filepath.Join(dir, "clio"), args...)
		cmd.Env = append(os.Environ(), "PATH="+dir+":"+os.Getenv("PATH"), "HOME="+home)
		if in != "" {
			cmd.Stdin = strings.NewReader(in)
		}
		out, err := cmd.CombinedOutput()
		if fail && err == nil {
			t.Fatalf("clio %v succeeded:\n%s", args, out)
		}
		if !fail && err != nil {
			t.Fatalf("clio %v: %v\n%s", args, err, out)
		}
		return string(out)
	}
	run := func(args ...string) string {
		t.Helper()
		return execClio("", false, args...)
	}
	runFail := func(args ...string) string {
		t.Helper()
		return execClio("", true, args...)
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
	if body := wrote(); !strings.Contains(body, "-m deepseek-flash") || !strings.Contains(body, "max_tokens 220") || !strings.Contains(body, "extract tar file") {
		t.Fatalf("online question did not reach the model: %q", body)
	}
	if _, err := os.Stat(probe); !os.IsNotExist(err) {
		t.Fatal("asking checked the key again")
	}

	if got := run("ask", "your question"); got != "" {
		t.Fatalf("ask printed %q", got)
	}
	if body := wrote(); !strings.Contains(body, "-m deepseek-flash") || !strings.Contains(body, "max_tokens 220") || !strings.Contains(body, "your question") {
		t.Fatalf("default model args: %q", body)
	}

	if got := run("model", "gemini"); !strings.Contains(got, "Clio will ask gemini.") {
		t.Fatalf("model gemini: %q", got)
	}
	if got := run("ask", "your question"); got != "" {
		t.Fatalf("gemini ask printed %q", got)
	}
	if body := wrote(); !strings.Contains(body, "-m gemini-3.8-flash") || !strings.Contains(body, "max_output_tokens 220") || !strings.Contains(body, "thinking_level low") || !strings.Contains(body, "your question") {
		t.Fatalf("gemini model args: %q", body)
	}

	if err := os.Remove(filepath.Join(home, ".config", "clio", "model")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(keys, []byte("gemini\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("ask", "your question")
	if body := wrote(); !strings.Contains(body, "-m gemini-3.8-flash") || !strings.Contains(body, "max_output_tokens 220") || !strings.Contains(body, "your question") {
		t.Fatalf("available gemini args: %q", body)
	}

	if got := execClio("sk-test\n", false, "key", "deepseek"); !strings.Contains(got, "Saved the deepseek key.") {
		t.Fatalf("key printed %q", got)
	}
	if body := wrote(); !strings.Contains(body, "keys set deepseek --value sk-test") || strings.Contains(body, "install ") {
		t.Fatalf("key command: %q", body)
	}
	if _, err := os.Stat(probe); err != nil {
		t.Fatal("saving a key did not check it")
	}
	if err := os.WriteFile(filepath.Join(home, "key-status"), []byte("2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got := execClio("sk-bad\n", true, "key", "deepseek")
	if !strings.Contains(got, "That deepseek key was rejected. It was not saved.") {
		t.Fatalf("rejected key: %q", got)
	}
	if _, err := os.Stat(record); err == nil && strings.Contains(wrote(), "keys set") {
		t.Fatal("rejected key was stored")
	}
	if err := os.Remove(filepath.Join(home, "key-status")); err != nil {
		t.Fatal(err)
	}

	if got := run("termux_setup"); !strings.Contains(got, "MODULE termux_setup") {
		t.Fatalf("wizard was not run: %s", got)
	}
	if _, err := os.Stat(record); !os.IsNotExist(err) {
		t.Fatalf("wizard called the model tool")
	}

	got = runFail("what", "is", "html")
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

	if err := os.WriteFile(keys, []byte("deepseek\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, "llm-fail"), []byte("1"), 0o644); err != nil {
		t.Fatal(err)
	}
	got = run("ask", "extract tar file")
	if !strings.Contains(got, "The model did not answer. Offline answer:") || !strings.Contains(got, "Use: tar -xzvf") {
		t.Fatalf("failed model fallback: %q", got)
	}
	if !strings.Contains(wrote(), "-m deepseek-flash") || !strings.Contains(wrote(), "extract tar file") {
		t.Fatal("failed model was not tried")
	}
}
