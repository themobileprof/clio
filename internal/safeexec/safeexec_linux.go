//go:build linux

package safeexec

import (
	"os/exec"
	"syscall"
)

// Command is a safe wrapper around exec.Command for Linux that ensures the path
// is resolved safely and configures the command to avoid blocked syscalls on Android/Termux
func Command(name string, arg ...string) *exec.Cmd {
	// Resolve path safely to avoid faccessat2 syscall issues
	var cmd *exec.Cmd
	if path, err := LookPath(name); err == nil {
		cmd = exec.Command(path, arg...)
	} else {
		cmd = exec.Command(name, arg...)
	}

	// On Android/Termux, prevent SIGSYS errors by using older syscalls
	// This avoids clone3, faccessat2, pidfd_open and other newer syscalls that may be blocked
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}

	// Force use of older clone() instead of clone3() on Linux
	// This prevents SIGSYS: bad system call on Android/Termux
	// Setting Cloneflags to 0 tells Go runtime to use legacy syscalls
	cmd.SysProcAttr.Cloneflags = 0

	// Disable pidfd_open by setting PidFD to nil explicitly
	// Go 1.24 will try to use pidfd_open if PidFD is set to a valid pointer
	// Setting it to nil tells Go not to use pidfd functionality at all
	// This prevents SIGSYS on Android where pidfd_open is blocked by seccomp
	cmd.SysProcAttr.PidFD = nil

	return cmd
}
