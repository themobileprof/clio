package modules

import (
	"clio/internal/layer3"
	"fmt"
	"os"
	"path/filepath"
)

// InitBuiltinModules loads built-in modules from disk into the database
func InitBuiltinModules() error {
	// Get the executable's directory or use current working directory
	modulesDir := "./modules"

	// Alternative: check in ~/.clio/modules
	home, err := os.UserHomeDir()
	if err == nil {
		altModulesDir := filepath.Join(home, ".clio", "modules")
		if _, err := os.Stat(altModulesDir); err == nil {
			modulesDir = altModulesDir
		}
	}

	// Check if modules directory exists
	if _, err := os.Stat(modulesDir); os.IsNotExist(err) {
		// No builtin modules directory found, that's okay
		return nil
	}

	entries, err := os.ReadDir(modulesDir)
	if err != nil {
		return fmt.Errorf("failed to read builtin modules: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
			continue
		}

		// Read module content
		modulePath := filepath.Join(modulesDir, entry.Name())
		content, err := os.ReadFile(modulePath)
		if err != nil {
			fmt.Printf("Warning: Failed to read %s: %v\n", entry.Name(), err)
			continue
		}

		// Parse to get metadata
		module, err := LoadModule(string(content))
		if err != nil {
			fmt.Printf("Warning: Failed to parse %s: %v\n", entry.Name(), err)
			continue
		}

		// Check if already in DB
		existing, err := layer3.GetModuleByID(module.ID)
		if err == nil && existing != "" {
			// Already exists, skip
			continue
		}

		// Insert into database
		tags := ""
		if len(module.Tags) > 0 {
			for i, tag := range module.Tags {
				if i > 0 {
					tags += ","
				}
				tags += tag
			}
		}

		if err := layer3.UpsertModule(module.ID, module.Name, module.Description, tags, module.Version, string(content)); err != nil {
			fmt.Printf("Warning: Failed to insert %s: %v\n", entry.Name(), err)
			continue
		}

		fmt.Printf("  Loaded builtin module: %s\n", module.Name)
	}

	return nil
}

// EnsureBuiltinModulesLoaded checks if builtin modules are loaded and loads them if not
func EnsureBuiltinModulesLoaded() error {
	// Check if termux_setup exists
	_, err := layer3.GetModuleByID("termux_setup")
	if err == nil {
		// Already loaded
		return nil
	}

	// Load builtin modules
	return InitBuiltinModules()
}
