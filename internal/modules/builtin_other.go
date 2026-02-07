package modules
//go:build !linux

package modules

// termuxSetupYAML is empty on non-Linux platforms
var termuxSetupYAML string = ""

// hasEmbeddedModules returns false on non-Linux platforms
func hasEmbeddedModules() bool {
	return false
}
