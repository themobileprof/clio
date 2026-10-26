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

const maxModuleBytes = 2 << 20 // 2 MiB per module — prevents OOM on small devices

var syncHTTP = &http.Client{Timeout: 30 * time.Second}

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

// Sync downloads modules using the active profile (lite on Termux skips non-essential modules).
func Sync() error {
	return SyncWithOptions(false)
}

// SyncFull downloads every module regardless of profile.
func SyncFull() error {
	return SyncWithOptions(true)
}

// SyncWithOptions controls whether the full module catalog is synced.
func SyncWithOptions(full bool) error {
	lite := !full && config.IsLiteProfile()
	if lite {
		fmt.Println("📱 Lite sync (Termux/low-memory) — essential modules only.")
		fmt.Println("   Use 'sync full' to download the complete catalog.")
	}

	if err := SyncFromRegistry(lite); err != nil {
		fmt.Printf("⚠️  Registry sync failed: %v\n", err)
		fmt.Println("📦 Falling back to GitHub...")
		return SyncFromGitHub(lite)
	}
	return nil
}

// isEssentialModule returns true for modules needed on constrained Termux devices.
func isEssentialModule(moduleID string) bool {
	switch moduleID {
	case "termux_setup", "neovim_setup", "git_setup", "devtools_setup", "database_setup":
		return true
	}
	lower := strings.ToLower(moduleID)
	return strings.Contains(lower, "termux") || strings.Contains(lower, "android") || strings.HasSuffix(lower, "_setup")
}

// SyncFromRegistry downloads modules from CLIPilot registry using delta sync
func SyncFromRegistry(lite bool) error {
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

	resp, err := syncHTTP.Get(url)
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
	skipped := 0
	for _, mod := range changedResp.ChangedModules {
		if lite && !isEssentialModule(mod.ID) {
			skipped++
			continue
		}

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

	if lite && skipped > 0 {
		fmt.Printf("✅ Lite sync complete. Updated %d modules (%d skipped).\n", count, skipped)
	} else {
		fmt.Printf("✅ Sync complete. Updated %d modules.\n", count)
	}
	return nil
}

// downloadAndSaveModule fetches a module YAML from registry and saves to local DB
func downloadAndSaveModule(registryURL, moduleID string) error {
	url := fmt.Sprintf("%s/api/v1/modules/%s/download", registryURL, moduleID)

	resp, err := syncHTTP.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxModuleBytes))
	if err != nil {
		return err
	}
	if len(body) == maxModuleBytes {
		return fmt.Errorf("module exceeds %d byte limit", maxModuleBytes)
	}

	hash := sha256.Sum256(body)
	checksum := fmt.Sprintf("%x", hash)

	// Parse YAML for metadata
	var mod ModuleYAML
	if err := yaml.Unmarshal(body, &mod); err != nil {
		return fmt.Errorf("yaml parse error: %w", err)
	}

	if mod.Name == "" {
		return fmt.Errorf("missing name in %s", moduleID)
	}

	tags := strings.Join(mod.Tags, ",")

	// Generate bash-friendly script for Termux
	bashScript, err := convertYAMLToBashScript(string(body))
	if err != nil {
		fmt.Printf("  Warning: failed to generate bash script: %v\n", err)
		bashScript = "" // Store empty on error
	}

	// Registry name is the DB key (what clio-run-module uses), not necessarily yaml id.
	return layer3.UpsertModuleWithChecksum(moduleID, mod.Name, mod.Description, tags, mod.Version, string(body), bashScript, checksum)
}

// SyncFromGitHub downloads modules from GitHub (fallback method)
func SyncFromGitHub(lite bool) error {
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
	skipped := 0
	for _, item := range contents {
		if item.Type != "file" || !strings.HasSuffix(item.Name, ".yaml") {
			continue
		}

		moduleID := strings.TrimSuffix(item.Name, ".yaml")
		if lite && !isEssentialModule(moduleID) {
			skipped++
			continue
		}

		fmt.Printf("  Processing %s...\n", item.Name)
		if err := processModuleByID(moduleID, item.DownloadURL); err != nil {
			fmt.Printf("  ❌ Failed %s: %v\n", item.Name, err)
		} else {
			count++
		}
		// Be nice to API rate limits
		time.Sleep(100 * time.Millisecond)
	}

	if lite && skipped > 0 {
		fmt.Printf("✅ Lite sync complete. Updated %d modules (%d skipped).\n", count, skipped)
	} else {
		fmt.Printf("✅ Sync complete. Updated %d modules.\n", count)
	}
	return nil
}

func processModuleByID(moduleID, url string) error {
	resp, err := syncHTTP.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxModuleBytes))
	if err != nil {
		return err
	}
	if len(body) == maxModuleBytes {
		return fmt.Errorf("module exceeds %d byte limit", maxModuleBytes)
	}

	var mod ModuleYAML
	if err := yaml.Unmarshal(body, &mod); err != nil {
		return fmt.Errorf("yaml parse error: %w", err)
	}

	if mod.Name == "" {
		return fmt.Errorf("missing name in %s", moduleID)
	}

	tags := strings.Join(mod.Tags, ",")

	bashScript, err := convertYAMLToBashScript(string(body))
	if err != nil {
		fmt.Printf("  Warning: failed to generate bash script: %v\n", err)
		bashScript = ""
	}

	hash := sha256.Sum256(body)
	checksum := fmt.Sprintf("%x", hash)

	return layer3.UpsertModuleWithChecksum(moduleID, mod.Name, mod.Description, tags, mod.Version, string(body), bashScript, checksum)
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
	script.WriteString(shellAssign("MODULE_NAME", module.Name))
	script.WriteString(shellAssign("MODULE_DESC", module.Description))
	script.WriteString(shellAssign("MODULE_VERSION", module.Version))
	script.WriteString(shellAssign("ESTIMATED_TIME", module.EstimatedTime))
	script.WriteString("\n")

	// For each step, write in simple format
	for _, flow := range module.Flows {
		script.WriteString(shellAssign("FLOW_NAME", flow.Name))
		script.WriteString(shellAssign("FLOW_DESC", flow.Description))
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
				script.WriteString(shellAssign(fmt.Sprintf("STEP_%d_CONTENT", i), step.Content))
			}
			if step.Command != "" {
				script.WriteString(shellAssign(fmt.Sprintf("STEP_%d_COMMAND", i), step.Command))
			}
			if step.Description != "" {
				script.WriteString(shellAssign(fmt.Sprintf("STEP_%d_DESCRIPTION", i), step.Description))
			}
			if step.Prompt != "" {
				script.WriteString(shellAssign(fmt.Sprintf("STEP_%d_PROMPT", i), step.Prompt))
			}
			if step.Default != "" {
				script.WriteString(shellAssign(fmt.Sprintf("STEP_%d_DEFAULT", i), step.Default))
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
				script.WriteString(shellAssign(fmt.Sprintf("STEP_%d_ON_NO", i), step.OnNo))
			}
			if step.Title != "" {
				script.WriteString(shellAssign(fmt.Sprintf("STEP_%d_TITLE", i), step.Title))
			}

			// Handle nested steps in sections
			if len(step.Steps) > 0 {
				for j, substep := range step.Steps {
					script.WriteString(fmt.Sprintf("STEP_%d_SUB_%d_TYPE=%s\n", i, j, substep.Type))
					if substep.Command != "" {
						script.WriteString(shellAssign(fmt.Sprintf("STEP_%d_SUB_%d_COMMAND", i, j), substep.Command))
					}
					if substep.Description != "" {
						script.WriteString(shellAssign(fmt.Sprintf("STEP_%d_SUB_%d_DESCRIPTION", i, j), substep.Description))
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

// shellAssign writes name=$(cat <<'DELIM' ... DELIM) so multiline commands,
// quotes, and dollar signs survive `source` on the Termux module runner.
// Go %q turns newlines into the two characters \ and n, and bash then
// executes the flattened line.
func shellAssign(name, value string) string {
	delim := "CLIO_EOF"
	for strings.Contains(value, delim) {
		delim += "_X"
	}
	body := strings.TrimRight(value, "\n")
	return fmt.Sprintf("%s=$(cat <<'%s'\n%s\n%s\n)\n", name, delim, body, delim)
}
