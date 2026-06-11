package platform

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// IsLiteProfile reports whether the active Clio profile is lite (low-memory mode).
// Reads ~/.clio/config.yaml when present; auto resolves via device heuristics.
func IsLiteProfile() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return IsLowMemoryDevice()
	}

	data, err := os.ReadFile(filepath.Join(home, ".clio", "config.yaml"))
	if err != nil {
		return IsLowMemoryDevice()
	}

	var cfg struct {
		Profile string `yaml:"profile"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return IsLowMemoryDevice()
	}

	switch cfg.Profile {
	case "lite":
		return true
	case "full":
		return false
	default:
		return IsLowMemoryDevice()
	}
}

// DBPath returns the SQLite database path from config or the default location.
func DBPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	data, err := os.ReadFile(filepath.Join(home, ".clio", "config.yaml"))
	if err == nil {
		var cfg struct {
			DBPath string `yaml:"db_path"`
		}
		if yaml.Unmarshal(data, &cfg) == nil && cfg.DBPath != "" {
			return cfg.DBPath
		}
	}

	return filepath.Join(home, ".clio", "clio.db")
}
