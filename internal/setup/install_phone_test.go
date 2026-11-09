package setup

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallerKeepsWhicheverPhoneBinaryStarts(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "install.sh"))
	if err != nil {
		t.Fatal(err)
	}
	const start = "# phone-loader:start\n"
	const end = "# phone-loader:end\n"
	body := string(raw)
	i := strings.Index(body, start)
	j := strings.Index(body, end)
	if i < 0 || j < i {
		t.Fatal("phone loader block missing from install.sh")
	}
	if strings.Contains(body[i:j], "candidates=(") || strings.Contains(body[i:j], "local -a") {
		t.Fatal("phone installer is piped to sh, which rejects bash arrays")
	}
	linux := strings.Index(body, "try_asset clio-linux-arm64")
	android := strings.Index(body, "try_asset clio-android-arm64")
	if linux < 0 || android < linux {
		t.Fatal("Android arm64 must try the Android 10 binary, then the Android 16 binary")
	}

	dir := t.TempDir()
	rejected := filepath.Join(dir, "rejected")
	accepted := filepath.Join(dir, "accepted")
	writeExe(t, rejected, "#!/bin/sh\necho 'CANNOT LINK EXECUTABLE \"./clio\": unexpected e_type: 2' >&2\nexit 1\n")
	writeExe(t, accepted, "#!/bin/sh\necho 'Run clio, then type sync, before running this wizard.' >&2\nexit 1\n")

	script := body[i+len(start):j] + "\n" +
		"if loader_rejected " + shellQuote(rejected) + "; then echo rejected; else echo accepted-rejected; fi\n" +
		"if loader_rejected " + shellQuote(accepted) + "; then echo rejected-accepted; else echo accepted; fi\n"
	cmd := exec.Command("sh", "-s")
	cmd.Stdin = strings.NewReader(script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("loader probe failed: %v\n%s", err, out)
	}
	got := string(out)
	if !strings.Contains(got, "rejected\n") || !strings.Contains(got, "accepted\n") || strings.Contains(got, "accepted-rejected") || strings.Contains(got, "rejected-accepted") {
		t.Fatalf("probe output:\n%s", got)
	}
}

func writeExe(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}

func shellQuote(path string) string {
	return "'" + strings.ReplaceAll(path, "'", `'\''`) + "'"
}
