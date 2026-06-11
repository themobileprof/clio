package modules

import (
	"clio/internal/config"
	"clio/internal/layer3"
	"clio/internal/setup"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
)

const catalogPageSize = 100

// CatalogEntry is a module from the registry catalog API.
type CatalogEntry struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Tags        []string `json:"tags"`
}

type catalogResponse struct {
	Modules []CatalogEntry `json:"modules"`
	Total   int            `json:"total"`
}

// FetchCatalog returns all modules from GET /api/v1/modules (with descriptions).
func FetchCatalog() ([]CatalogEntry, error) {
	registryURL := strings.TrimRight(config.GetRegistryURL(), "/")
	var all []CatalogEntry
	offset := 0

	for {
		url := fmt.Sprintf("%s/api/v1/modules?limit=%d&offset=%d&sort_by=name&order=asc",
			registryURL, catalogPageSize, offset)

		resp, err := syncHTTP.Get(url)
		if err != nil {
			return nil, fmt.Errorf("registry unreachable: %w", err)
		}

		body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("registry returned %d", resp.StatusCode)
		}

		var page catalogResponse
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("invalid catalog response: %w", err)
		}

		all = append(all, page.Modules...)
		offset += len(page.Modules)
		if len(page.Modules) == 0 || offset >= page.Total {
			break
		}
	}

	return all, nil
}

// FetchAutomationCatalog returns registry modules excluding first-class setup wizards.
func FetchAutomationCatalog() ([]CatalogEntry, error) {
	all, err := FetchCatalog()
	if err != nil {
		return nil, err
	}
	var out []CatalogEntry
	for _, m := range all {
		if setup.IsSetupModule(m.ID) {
			continue
		}
		out = append(out, m)
	}
	return out, nil
}

// ListCachedModules returns modules stored locally (metadata only).
func ListCachedModules() ([]layer3.ModuleMeta, error) {
	return layer3.ListModuleMeta()
}

// DownloadCommand is the REPL command to fetch a single module.
func DownloadCommand(moduleID string) string {
	return "download " + moduleID
}

// RunCommand is the shell command to execute a downloaded module.
func RunCommand(moduleID, flow string) string {
	if flow == "" {
		flow = "setup"
	}
	return "clio-run-module " + moduleID + " " + flow
}

// ShowFullCatalog prints setup wizards and automation modules in two distinct sections.
func ShowFullCatalog() {
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║  CLIO CATALOG — setup wizards + automation modules       ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("Both kinds match natural-language questions. They differ by purpose:")
	fmt.Println("  • Setup wizards    → install & configure (run once)")
	fmt.Println("  • Automation mods  → repeat tasks (copy, backup, check disk…)")
	fmt.Println()

	cached := cachedModuleSet()
	setup.ShowWizardCatalogSection(func(id string) bool { return cached[id] })

	fmt.Println("── 📦 AUTOMATION MODULES ────────────────────────────────────")
	fmt.Println("   Day-to-day task workflows — run whenever you need them")
	fmt.Println("   Ask naturally: \"check disk space\", \"copy file to backup\"")
	fmt.Println("   Commands: modules · download <id> · module <id>")
	fmt.Println()

	showAutomationEntries(cached)
	fmt.Println()
	fmt.Println("Browse one kind only:  setup  |  modules")
	fmt.Println()
}

// ShowCatalog prints only the automation module section.
func ShowCatalog() {
	fmt.Println("╔══════════════════════════════════════════════════════════╗")
	fmt.Println("║  📦 AUTOMATION MODULES — not setup wizards               ║")
	fmt.Println("╚══════════════════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println("These are repeat task workflows. For install/configure, type 'setup'.")
	fmt.Println()
	showAutomationEntries(cachedModuleSet())
}

func showAutomationEntries(cached map[string]bool) {
	entries, err := FetchAutomationCatalog()
	if err != nil {
		fmt.Printf("⚠️  Could not reach registry: %v\n", err)
		fmt.Println("Showing modules cached on this device instead:")
		fmt.Println()
		showCachedCatalogBody(cached)
		return
	}

	if len(entries) == 0 {
		fmt.Println("No automation modules on the registry yet. Try 'sync' when online.")
		return
	}

	for _, m := range entries {
		printCatalogEntry(m, cached[m.ID])
	}

	fmt.Println("────────────────────────────────────────────────────────────")
	fmt.Printf("%d automation module(s)\n", len(entries))
	fmt.Println("  get one:  download <module_id>")
	fmt.Println("  get all:  sync")
	fmt.Println("  run:      clio-run-module <module_id> setup")
	fmt.Println()
}

func showCachedCatalog() {
	showCachedCatalogBody(cachedModuleSet())
}

func showCachedCatalogBody(cached map[string]bool) {
	metas, err := ListCachedModules()
	if err != nil {
		fmt.Printf("Error reading cache: %v\n", err)
		return
	}
	var filtered []layer3.ModuleMeta
	for _, m := range metas {
		if !setup.IsSetupModule(m.ModuleID) {
			filtered = append(filtered, m)
		}
	}
	if len(filtered) == 0 {
		fmt.Println("No automation modules cached yet. Run 'sync' when online.")
		return
	}
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].ModuleID < filtered[j].ModuleID
	})
	for _, m := range filtered {
		printMetaEntry(m, cached[m.ModuleID])
	}
	fmt.Println("Download more from registry: sync")
}

func cachedModuleSet() map[string]bool {
	set := make(map[string]bool)
	metas, err := ListCachedModules()
	if err != nil {
		return set
	}
	for _, m := range metas {
		set[m.ModuleID] = true
	}
	return set
}

func printCatalogEntry(m CatalogEntry, cached bool) {
	status := "○ not downloaded"
	if cached {
		status = "✓ downloaded"
	}
	id := m.ID
	if id == "" {
		id = m.Name
	}
	desc := strings.TrimSpace(m.Description)
	if desc == "" {
		desc = "(no description)"
	}

	fmt.Printf("  [AUTOMATION] %s  [%s]\n", id, status)
	fmt.Printf("  %s\n", desc)
	if m.Version != "" {
		fmt.Printf("  version: %s\n", m.Version)
	}
	fmt.Printf("  ask:      (type what you want in plain English)\n")
	fmt.Printf("  get:      %s\n", DownloadCommand(id))
	fmt.Printf("  run:      %s\n", RunCommand(id, "setup"))
	fmt.Println()
}

func printMetaEntry(m layer3.ModuleMeta, cached bool) {
	status := "○ not downloaded"
	if cached {
		status = "✓ downloaded"
	}
	desc := strings.TrimSpace(m.Description)
	if desc == "" {
		desc = "(no description)"
	}
	fmt.Printf("  [AUTOMATION] %s  [%s]\n", m.ModuleID, status)
	fmt.Printf("  %s\n", desc)
	fmt.Printf("  ask:      (type what you want in plain English)\n")
	fmt.Printf("  get:      %s\n", DownloadCommand(m.ModuleID))
	fmt.Printf("  run:      %s\n", RunCommand(m.ModuleID, "setup"))
	fmt.Println()
}

// ShowModuleDetail prints one module by ID from registry or cache.
func ShowModuleDetail(moduleID string) error {
	moduleID = strings.TrimSpace(moduleID)
	if moduleID == "" {
		return fmt.Errorf("module id required")
	}
	if setup.IsSetupModule(moduleID) {
		fmt.Printf("[SETUP WIZARD] %s — not an automation module.\n", moduleID)
		fmt.Printf("Use: setup %s\n", wizardIDForModule(moduleID))
		return nil
	}
	fmt.Println("[AUTOMATION MODULE]")
	fmt.Println()

	registryURL := strings.TrimRight(config.GetRegistryURL(), "/")
	url := fmt.Sprintf("%s/api/v1/modules/%s", registryURL, moduleID)
	resp, err := syncHTTP.Get(url)
	if err == nil && resp.StatusCode == http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()
		var detail struct {
			ID          string   `json:"id"`
			Name        string   `json:"name"`
			Description string   `json:"description"`
			Version     string   `json:"version"`
			Tags        []string `json:"tags"`
		}
		if json.Unmarshal(body, &detail) == nil {
			cached, _ := layer3.ModuleExists(moduleID)
			printCatalogEntry(CatalogEntry{
				ID: detail.ID, Name: detail.Name,
				Description: detail.Description, Version: detail.Version, Tags: detail.Tags,
			}, cached)
			return nil
		}
	}

	metas, err := layer3.FindModuleMeta(moduleID)
	if err != nil || metas == nil {
		return fmt.Errorf("module %q not found — try 'modules' to browse", moduleID)
	}
	printMetaEntry(*metas, true)
	return nil
}

func wizardIDForModule(moduleID string) string {
	for _, w := range setup.AllWizards() {
		if strings.EqualFold(w.ModuleID, moduleID) {
			return w.ID
		}
	}
	return moduleID
}
