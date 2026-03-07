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

	// Generate bash-friendly script for Termux
	bashScript, err := convertYAMLToBashScript(string(body))
	if err != nil {
		fmt.Printf("  Warning: failed to generate bash script: %v\n", err)
		bashScript = "" // Store empty on error
	}

	// Save to DB with checksum
	return layer3.UpsertModuleWithChecksum(mod.ID, mod.Name, mod.Description, tags, mod.Version, string(body), bashScript, checksum)
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

	// Generate bash-friendly script for Termux
	bashScript, err := convertYAMLToBashScript(string(body))
	if err != nil {
		fmt.Printf("  Warning: failed to generate bash script: %v\n", err)
		bashScript = "" // Store empty on error
	}

	return layer3.UpsertModule(mod.ID, mod.Name, mod.Description, tags, mod.Version, string(body), bashScript)
}

// convertYAMLToBashScript converts module YAML to bash-friendly format
// This pre-processes the YAML so bash script doesn't need to parse it
func convertYAMLToBashScript(yamlContent string) (string, error) {
	var module FullModuleYAML
	if err := yaml.Unmarshal([]byte(yamlContent), &module); err != nil {
		return "", err
	}

	var script strings.Builder

	// Write metadata
	script.WriteString(fmt.Sprintf("MODULE_NAME=%s\n", shellEscape(module.Name)))
	script.WriteString(fmt.Sprintf("MODULE_DESC=%s\n", shellEscape(module.Description)))
	script.WriteString(fmt.Sprintf("MODULE_VERSION=%s\n", shellEscape(module.Version)))
	script.WriteString(fmt.Sprintf("ESTIMATED_TIME=%s\n", shellEscape(module.EstimatedTime)))
	script.WriteString("\n")

	// For each step, write in simple format
	for _, flow := range module.Flows {
		script.WriteString(fmt.Sprintf("FLOW_NAME=%s\n", shellEscape(flow.Name)))
		script.WriteString(fmt.Sprintf("FLOW_DESC=%s\n", shellEscape(flow.Description)))
		script.WriteString("\n")

		// Count sections for progress
		sectionCount := 0
		for _, step := range flow.Steps {
			if step.Type == "section" {
				sectionCount++
			}
		}
		script.WriteString(fmt.Sprintf("SECTION_COUNT=%d\n", sectionCount))
		script.WriteString("\n")

		// Write steps
		for i, step := range flow.Steps {
			script.WriteString(fmt.Sprintf("STEP_%d_TYPE=%s\n", i, step.Type))

			if step.Content != "" {
				script.WriteString(fmt.Sprintf("STEP_%d_CONTENT=%s\n", i, shellEscape(step.Content)))
			}
			if step.Command != "" {
				script.WriteString(fmt.Sprintf("STEP_%d_COMMAND=%s\n", i, shellEscape(step.Command)))
			}
			if step.Description != "" {
				script.WriteString(fmt.Sprintf("STEP_%d_DESCRIPTION=%s\n", i, shellEscape(step.Description)))
			}
			if step.Prompt != "" {
				script.WriteString(fmt.Sprintf("STEP_%d_PROMPT=%s\n", i, shellEscape(step.Prompt)))
			}
			if step.Default != "" {
				script.WriteString(fmt.Sprintf("STEP_%d_DEFAULT=%s\n", i, shellEscape(step.Default)))
			}
			if step.ShowOutput {
				script.WriteString(fmt.Sprintf("STEP_%d_SHOW_OUTPUT=true\n", i))
			}
			if step.Interactive {
				script.WriteString(fmt.Sprintf("STEP_%d_INTERACTIVE=true\n", i))
			}
			if step.ContinueOnError {
				script.WriteString(fmt.Sprintf("STEP_%d_CONTINUE_ON_ERROR=true\n", i))
			}
			if step.OnNo != "" {
				script.WriteString(fmt.Sprintf("STEP_%d_ON_NO=%s\n", i, shellEscape(step.OnNo)))
			}
			if step.Title != "" {
				script.WriteString(fmt.Sprintf("STEP_%d_TITLE=%s\n", i, shellEscape(step.Title)))
			}

			// Handle nested steps in sections
			if len(step.Steps) > 0 {
				for j, substep := range step.Steps {
					script.WriteString(fmt.Sprintf("STEP_%d_SUB_%d_TYPE=%s\n", i, j, substep.Type))
					if substep.Command != "" {
						script.WriteString(fmt.Sprintf("STEP_%d_SUB_%d_COMMAND=%s\n", i, j, shellEscape(substep.Command)))
					}
					if substep.Description != "" {
						script.WriteString(fmt.Sprintf("STEP_%d_SUB_%d_DESCRIPTION=%s\n", i, j, shellEscape(substep.Description)))
					}
					if substep.ShowOutput {
						script.WriteString(fmt.Sprintf("STEP_%d_SUB_%d_SHOW_OUTPUT=true\n", i, j))
					}
					if substep.Interactive {
						script.WriteString(fmt.Sprintf("STEP_%d_SUB_%d_INTERACTIVE=true\n", i, j))
					}
					if substep.ContinueOnError {
						script.WriteString(fmt.Sprintf("STEP_%d_SUB_%d_CONTINUE_ON_ERROR=true\n", i, j))
					}
				}
				script.WriteString(fmt.Sprintf("STEP_%d_SUB_COUNT=%d\n", i, len(step.Steps)))
			}

			script.WriteString("\n")
		}

		script.WriteString(fmt.Sprintf("STEP_COUNT=%d\n", len(flow.Steps)))
		script.WriteString("---FLOW_END---\n\n")
	}

	return script.String(), nil
}

// shellEscape escapes a string for safe use in bash
func shellEscape(s string) string {
	// Use printf %q format which properly escapes for shell
	return fmt.Sprintf("%q", s)
}
