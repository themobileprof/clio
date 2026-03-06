//go:build !linux

package safeexec

import (
	"os/exec"
)

// Command is a safe wrapper around exec.Command for non-Linux systems
// that ensures the path is resolved safely
func Command(name string, arg ...string) *exec.Cmd {
	// Resolve path safely to avoid faccessat2 syscall issues
	if path, err := LookPath(name); err == nil {
		return exec.Command(path, arg...)
	}
	return exec.Command(name, arg...)
}
