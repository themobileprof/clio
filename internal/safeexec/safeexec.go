package safeexec

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// LookPath is a safe wrapper around exec.LookPath that avoids faccessat2 issues
// on older Android kernels by manually checking file permissions in PATH.
func LookPath(file string) (string, error) {
	// If it contains a slash, it's a relative or absolute path, check directly
	if strings.Contains(file, "/") {
		err := findExecutable(file)
		if err == nil {
			return file, nil
		}
		return "", &exec.Error{Name: file, Err: err}
	}

	// Get PATH
	path := os.Getenv("PATH")
	for _, dir := range filepath.SplitList(path) {
		if dir == "" {
			// Unix shell semantics: path element "" means "."
			dir = "."
		}
		path := filepath.Join(dir, file)
		if err := findExecutable(path); err == nil {
			return path, nil
		}
	}
	return "", &exec.Error{Name: file, Err: exec.ErrNotFound}
}

// Command is a safe wrapper around exec.Command that ensures the path is resolved safely
func Command(name string, arg ...string) *exec.Cmd {
	// We don't change the behavior of exec.Command itself (it calls LookPath internally),
    // but better to resolve it first to fail early or ensure validity if we were replacing
    // the internal lookup. However, exec.Command uses exec.LookPath.
    // Since we can't monkeypatch exec.LookPath, users should use
    // safeexec.LookPath explicitly or this helper which resolves the path first.

    // If name is found via safe lookup, use absolute path.
    // If not found, let exec.Command try (it might fail with the syscall issue, but at least we tried).
    if path, err := LookPath(name); err == nil {
        return exec.Command(path, arg...)
    }
	return exec.Command(name, arg...)
}

func findExecutable(file string) error {
	d, err := os.Stat(file)
	if err != nil {
		return err
	}
	if m := d.Mode(); !m.IsDir() && m&0111 != 0 {
		return nil
	}
	return os.ErrPermission
}
