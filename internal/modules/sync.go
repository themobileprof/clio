package modules

import (
	"clio/internal/layer3"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	RepoOwner = "themobileprof"
	RepoName  = "clipilot"
	ModulesPath = "modules"
	GitHubAPI = "https://api.github.com/repos"
)

type GitHubContent struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	DownloadURL string `json:"download_url"`
	Type        string `json:"type"`
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

// Sync downloads modules from GitHub and updates the local database.
func Sync() error {
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
