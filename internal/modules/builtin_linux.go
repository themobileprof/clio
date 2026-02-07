//go:build linux

package modules

import (
	_ "embed"
)

//go:embed embedded/termux_setup.yaml
var termuxSetupYAML string

// hasEmbeddedModules returns true on Linux where Termux setup is relevant
func hasEmbeddedModules() bool {
	return true
}
