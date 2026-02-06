package setup

import (
	"os"
	"path/filepath"
)

// NOTE: The interactive setup wizard has been moved to a YAML module (modules/termux_setup.yaml)
// This file now only contains utility functions used by the YAML executor.
// See YAML_MODULE_SYSTEM.md for documentation on the new module system.

// IsTermux checks if we're running on Termux
func IsTermux() bool {
	return os.Getenv("TERMUX_VERSION") != ""
}

// IsSetupComplete checks if the setup has been completed
func IsSetupComplete() bool {
	if !IsTermux() {
		return true // Not needed on non-Termux systems
	}

	// Check for flag file
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	flagPath := filepath.Join(home, ".clio", "termux_setup_complete")
	_, err = os.Stat(flagPath)
	return err == nil
}

// MarkSetupComplete creates a flag file indicating setup is done
func MarkSetupComplete() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	clioDir := filepath.Join(home, ".clio")
	if err := os.MkdirAll(clioDir, 0755); err != nil {
		return err
	}

	flagPath := filepath.Join(clioDir, "termux_setup_complete")
	return os.WriteFile(flagPath, []byte("1"), 0644)
}
