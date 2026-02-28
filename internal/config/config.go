package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds Clio configuration settings
type Config struct {
	RegistryURL  string `yaml:"registry_url"`
	CacheTTL     string `yaml:"cache_ttl"`
	SyncInterval string `yaml:"sync_interval"`
}

var defaultConfig = Config{
	RegistryURL:  "https://clipilot.themobileprof.com",
	CacheTTL:     "24h",
	SyncInterval: "168h",
}

// Load reads config from ~/.clio/config.yaml or returns defaults
func Load() Config {
	home, err := os.UserHomeDir()
	if err != nil {
		return defaultConfig
	}

	configPath := filepath.Join(home, ".clio", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		// Config doesn't exist, return defaults
		return defaultConfig
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		// Parse error, return defaults
		return defaultConfig
	}

	// Fill in missing values with defaults
	if cfg.RegistryURL == "" {
		cfg.RegistryURL = defaultConfig.RegistryURL
	}
	if cfg.CacheTTL == "" {
		cfg.CacheTTL = defaultConfig.CacheTTL
	}
	if cfg.SyncInterval == "" {
		cfg.SyncInterval = defaultConfig.SyncInterval
	}

	return cfg
}

// GetRegistryURL returns the configured registry URL
func GetRegistryURL() string {
	return Load().RegistryURL
}
