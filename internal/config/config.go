package config

import (
	"clio/internal/setup"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// Profile controls resource usage: auto (default), lite, or full.
type Profile string

const (
	ProfileAuto Profile = "auto"
	ProfileLite Profile = "lite"
	ProfileFull Profile = "full"
)

// Config holds Clio configuration settings.
type Config struct {
	Profile      Profile `yaml:"profile"`
	RegistryURL  string  `yaml:"registry_url"`
	CacheTTL     string  `yaml:"cache_ttl"`
	SyncInterval string  `yaml:"sync_interval"`
	DBPath       string  `yaml:"db_path"`
	// MemoryLimit sets the Go runtime soft memory cap (e.g. "48MiB"). Empty = default per profile.
	MemoryLimit string `yaml:"memory_limit"`
}

var defaultConfig = Config{
	Profile:      ProfileAuto,
	RegistryURL:  "https://clipilot.themobileprof.com",
	CacheTTL:     "24h",
	SyncInterval: "168h",
}

var (
	cached     Config
	cachedOnce sync.Once
)

// Load reads config from ~/.clio/config.yaml or returns defaults.
func Load() Config {
	cachedOnce.Do(func() {
		cached = loadFromDisk()
	})
	return cached
}

// ResetCache clears the cached config (for tests).
func ResetCache() {
	cachedOnce = sync.Once{}
}

func loadFromDisk() Config {
	cfg := defaultConfig

	home, err := os.UserHomeDir()
	if err != nil {
		return applyDefaults(cfg)
	}

	configPath := filepath.Join(home, ".clio", "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return applyDefaults(cfg)
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return applyDefaults(defaultConfig)
	}

	return applyDefaults(cfg)
}

func applyDefaults(cfg Config) Config {
	if cfg.RegistryURL == "" {
		cfg.RegistryURL = defaultConfig.RegistryURL
	}
	if cfg.CacheTTL == "" {
		cfg.CacheTTL = defaultConfig.CacheTTL
	}
	if cfg.SyncInterval == "" {
		cfg.SyncInterval = defaultConfig.SyncInterval
	}
	if cfg.Profile == "" {
		cfg.Profile = ProfileAuto
	}
	if cfg.DBPath == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			cfg.DBPath = filepath.Join(home, ".clio", "clio.db")
		}
	}
	return cfg
}

// EffectiveProfile resolves auto to lite or full based on the device.
func EffectiveProfile() Profile {
	cfg := Load()
	switch cfg.Profile {
	case ProfileLite:
		return ProfileLite
	case ProfileFull:
		return ProfileFull
	default:
		if setup.IsLowMemoryDevice() {
			return ProfileLite
		}
		return ProfileFull
	}
}

// IsLiteProfile is true when man search, remote API, and full module sync are skipped.
func IsLiteProfile() bool {
	return EffectiveProfile() == ProfileLite
}

// GetRegistryURL returns the configured registry URL.
func GetRegistryURL() string {
	return Load().RegistryURL
}

// GetDBPath returns the SQLite database path.
func GetDBPath() string {
	cfg := Load()
	if cfg.DBPath != "" {
		return cfg.DBPath
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".clio", "clio.db")
}

// GetMemoryLimit returns the Go runtime memory limit for the active profile.
func GetMemoryLimit() int64 {
	cfg := Load()
	if cfg.MemoryLimit != "" {
		return parseMemoryLimit(cfg.MemoryLimit)
	}
	if IsLiteProfile() {
		return 48 << 20 // 48 MiB — conservative for 2 GB Termux hosts
	}
	return 0
}

func parseMemoryLimit(s string) int64 {
	s = strings.TrimSpace(s)
	multipliers := []struct {
		suffix string
		mult   int64
	}{
		{"GiB", 1 << 30},
		{"MiB", 1 << 20},
		{"KiB", 1 << 10},
		{"GB", 1000 * 1000 * 1000},
		{"MB", 1000 * 1000},
		{"KB", 1000},
	}
	for _, m := range multipliers {
		if strings.HasSuffix(s, m.suffix) {
			numStr := strings.TrimSpace(s[:len(s)-len(m.suffix)])
			if n, err := strconv.ParseFloat(numStr, 64); err == nil {
				return int64(n * float64(m.mult))
			}
		}
	}
	return 0
}
