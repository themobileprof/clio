//go:build unix

package setup

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestUninstallAnswersComeFromTheTerminal(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 is required to drive the terminal")
	}
	script, err := filepath.Abs(filepath.Join("..", "..", "uninstall.sh"))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("python3", "-c", uninstallDriver, script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
}

const uninstallDriver = `
import os, pty, sys, tempfile, pathlib
script = sys.argv[1]

def run(answer, expect_removed):
    home = tempfile.mkdtemp()
    prefix = pathlib.Path(home) / "prefix"
    bindir = prefix / "bin"
    bindir.mkdir(parents=True)
    (bindir / "clio").write_text("#!/bin/sh\n")
    (bindir / "clio.bin").write_text("bin")
    (bindir / "clio-run-module").write_text("run")
    cfg = pathlib.Path(home) / ".clio"
    cfg.mkdir()
    (cfg / "clio.db").write_text("db")
    wrapper = pathlib.Path(home) / "run.sh"
    wrapper.write_text(
        "#!/bin/bash\n"
        f"export HOME={home}\n"
        "export TERMUX_VERSION=test\n"
        f"export PREFIX={prefix}\n"
        f"cat {script} | bash\n"
    )
    wrapper.chmod(0o755)
    acc = bytearray()
    sent = {"bin": False, "cfg": False}
    def master_read(fd):
        data = os.read(fd, 4096)
        acc.extend(data)
        if not sent["bin"] and b"Remove Clio binary?" in acc:
            sent["bin"] = True
            os.write(fd, answer)
        if not sent["cfg"] and b"Remove configuration directory ~/.clio?" in acc:
            sent["cfg"] = True
            os.write(fd, b"n\n")
        return data
    status = pty.spawn([str(wrapper)], master_read)
    text = acc.decode(errors="replace")
    if status != 0:
        sys.exit("uninstaller failed:\n" + text)
    gone = not (bindir / "clio").exists()
    if expect_removed:
        if not gone or not cfg.exists() or "uninstalled successfully" not in text:
            sys.exit("yes did not uninstall:\n" + text)
    else:
        if gone or "Uninstall cancelled" not in text:
            sys.exit("empty answer removed Clio:\n" + text)

run(b"yes\n", True)
run(b"\n", False)
`
