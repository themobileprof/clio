package modules

import (
	"clio/internal/config"
	"clio/internal/layer3"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	RepoOwner   = "themobileprof"
	RepoName    = "clipilot"
	ModulesPath = "modules"
	GitHubAPI   = "https://api.github.com/repos"
)

type GitHubContent struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	DownloadURL string `json:"download_url"`
	Type        string `json:"type"`
}

// ChangedModulesResponse represents the response from /api/v1/modules/changed
type ChangedModulesResponse struct {
	ChangedModules []struct {
		ID             string `json:"id"`
		Version        string `json:"version"`
		ChecksumSHA256 string `json:"checksum_sha256"`
		UpdatedAt      string `json:"updated_at"`
		ChangeType     string `json:"change_type"`
	} `json:"changed_modules"`
	SyncTimestamp string `json:"sync_timestamp"`
}

type ModuleYAML struct {
	Name        string   `yaml:"name"`
	ID          string   `yaml:"id"`
	Version     string   `yaml:"version"`
	Description string   `yaml:"description"`
	Tags        []string `yaml:"tags"`
	// We ignore flows/steps for metadata indexing,
	// unless we want to execute them later (we store full content anyway)
}

// Sync downloads modules from CLIPilot registry or falls back to GitHub
func Sync() error {
	// Try registry first
	if err := SyncFromRegistry(); err != nil {
		fmt.Printf("⚠️  Registry sync failed: %v\n", err)
		fmt.Println("📦 Falling back to GitHub...")
		return SyncFromGitHub()
	}
	return nil
}

// SyncFromRegistry downloads modules from CLIPilot registry using delta sync
func SyncFromRegistry() error {
	registryURL := config.GetRegistryURL()
	fmt.Println("🔄 Syncing modules from registry...")

	// Get last sync timestamp
	lastSync, err := layer3.GetLastSyncTimestamp()
	if err != nil {
		lastSync = time.Time{} // First sync
	}

	// Call delta sync endpoint
	url := fmt.Sprintf("%s/api/v1/modules/changed?since=%s",
		registryURL, lastSync.Format(time.RFC3339))

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to connect to registry: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registry returned status %d", resp.StatusCode)
	}

	var changedResp ChangedModulesResponse
	if err := json.NewDecoder(resp.Body).Decode(&changedResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	count := 0
	for _, mod := range changedResp.ChangedModules {
		// Check if we need to download (checksum differs)
		localChecksum, err := layer3.GetModuleChecksum(mod.ID)
		if err == nil && localChecksum == mod.ChecksumSHA256 {
			// Already up to date
			continue
		}

		fmt.Printf("  Downloading %s...\n", mod.ID)
		if err := downloadAndSaveModule(registryURL, mod.ID); err != nil {
			fmt.Printf("  ❌ Failed %s: %v\n", mod.ID, err)
		} else {
			count++
		}

		// Be nice to server
		time.Sleep(100 * time.Millisecond)
	}

	// Save sync timestamp
	if err := layer3.SaveLastSyncTimestamp(time.Now()); err != nil {
		fmt.Printf("Warning: failed to save sync timestamp: %v\n", err)
	}

	fmt.Printf("✅ Sync complete. Updated %d modules.\n", count)
	return nil
}

// downloadAndSaveModule fetches a module YAML from registry and saves to local DB
func downloadAndSaveModule(registryURL, moduleID string) error {
	url := fmt.Sprintf("%s/api/v1/modules/%s/download", registryURL, moduleID)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Calculate checksum
	hash := sha256.Sum256(body)
	checksum := fmt.Sprintf("%x", hash)

	// Parse YAML for metadata
	var mod ModuleYAML
	if err := yaml.Unmarshal(body, &mod); err != nil {
		return fmt.Errorf("yaml parse error: %w", err)
	}

	// Validate
	if mod.ID == "" || mod.Name == "" {
		return fmt.Errorf("missing id or name")
	}

	tags := strings.Join(mod.Tags, ",")

	// Save to DB with checksum
	return layer3.UpsertModuleWithChecksum(mod.ID, mod.Name, mod.Description, tags, mod.Version, string(body), checksum)
}

// SyncFromGitHub downloads modules from GitHub (fallback method)
func SyncFromGitHub() error {
	fmt.Println("🔄 Syncing modules from remote...")

	// 1. List modules directory
	url := fmt.Sprintf("%s/%s/%s/contents/%s", GitHubAPI, RepoOwner, RepoName, ModulesPath)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch module list: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("github api error: %d", resp.StatusCode)
	}

	var contents []GitHubContent
	if err := json.NewDecoder(resp.Body).Decode(&contents); err != nil {
		return err
	}

	count := 0
	for _, item := range contents {
		if item.Type != "file" || !strings.HasSuffix(item.Name, ".yaml") {
			continue
		}

		fmt.Printf("  Processing %s...\n", item.Name)
		if err := processModule(item.DownloadURL); err != nil {
			fmt.Printf("  ❌ Failed %s: %v\n", item.Name, err)
		} else {
			count++
		}
		// Be nice to API rate limits
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Printf("✅ Sync complete. Updated %d modules.\n", count)
	return nil
}

func processModule(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var mod ModuleYAML
	if err := yaml.Unmarshal(body, &mod); err != nil {
		return fmt.Errorf("yaml parse error: %w", err)
	}

	// Validate minimal fields
	if mod.ID == "" || mod.Name == "" {
		return fmt.Errorf("missing id or name")
	}

	tags := strings.Join(mod.Tags, ",")

	return layer3.UpsertModule(mod.ID, mod.Name, mod.Description, tags, mod.Version, string(body))
}
