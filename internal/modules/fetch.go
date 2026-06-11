package modules

import (
	"clio/internal/config"
	"clio/internal/layer3"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// EnsureModule downloads a module from the registry when it is not cached locally.
func EnsureModule(moduleID string) error {
	moduleID = strings.TrimSpace(moduleID)
	if moduleID == "" {
		return fmt.Errorf("empty module id")
	}

	exists, err := layer3.ModuleExists(moduleID)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	fmt.Printf("📥 Downloading module %s from registry...\n", moduleID)
	registryURL := config.GetRegistryURL()
	if err := downloadAndSaveModule(registryURL, moduleID); err == nil {
		return nil
	}

	fmt.Printf("⚠️  Registry download failed, trying GitHub fallback...\n")
	return downloadModuleFromGitHub(moduleID)
}

// EnsureModules downloads each module if missing.
func EnsureModules(moduleIDs ...string) error {
	for _, id := range moduleIDs {
		if err := EnsureModule(id); err != nil {
			return fmt.Errorf("%s: %w", id, err)
		}
	}
	return nil
}

// downloadModuleFromGitHub fetches a single module YAML by registry filename.
func downloadModuleFromGitHub(moduleID string) error {
	url := fmt.Sprintf("%s/%s/%s/contents/%s/%s.yaml",
		GitHubAPI, RepoOwner, RepoName, ModulesPath, moduleID)

	resp, err := syncHTTP.Get(url)
	if err != nil {
		return fmt.Errorf("github fetch failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("github returned %d for %s", resp.StatusCode, moduleID)
	}

	var meta GitHubContent
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return err
	}
	if meta.DownloadURL == "" {
		return fmt.Errorf("no download url for %s", moduleID)
	}

	if err := processModuleByID(moduleID, meta.DownloadURL); err != nil {
		return err
	}
	fmt.Printf("✅ Downloaded %s from GitHub\n", moduleID)
	return nil
}
