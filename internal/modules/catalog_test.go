package modules

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"clio/internal/config"
)

func TestFetchCatalogPaginated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/modules" {
			http.NotFound(w, r)
			return
		}
		offset := 0
		if o := r.URL.Query().Get("offset"); o != "" {
			_, _ = fmt.Sscanf(o, "%d", &offset)
		}
		page := catalogResponse{
			Total: 2,
			Modules: []CatalogEntry{
				{ID: "copy_file", Name: "copy_file", Description: "Copy a file", Version: "1.0"},
			},
		}
		if offset > 0 {
			page.Modules = []CatalogEntry{
				{ID: "list_directory", Name: "list_directory", Description: "List a folder", Version: "1.0"},
			}
		}
		_ = json.NewEncoder(w).Encode(page)
	}))
	defer srv.Close()

	old := config.GetRegistryURL()
	config.SetRegistryURLForTest(srv.URL)
	defer config.SetRegistryURLForTest(old)

	all, err := FetchCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("got %d modules, want 2", len(all))
	}
}

func TestFetchAutomationCatalogExcludesSetup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(catalogResponse{
			Total: 2,
			Modules: []CatalogEntry{
				{ID: "vim_setup", Description: "setup wizard"},
				{ID: "copy_file", Description: "copy files"},
			},
		})
	}))
	defer srv.Close()

	old := config.GetRegistryURL()
	config.SetRegistryURLForTest(srv.URL)
	defer config.SetRegistryURLForTest(old)

	out, err := FetchAutomationCatalog()
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].ID != "copy_file" {
		t.Fatalf("got %+v, want only copy_file", out)
	}
}

func TestDownloadAndRunCommands(t *testing.T) {
	if got := DownloadCommand("copy_file"); got != "download copy_file" {
		t.Fatalf("DownloadCommand = %q", got)
	}
	if got := RunCommand("copy_file", ""); got != "clio-run-module copy_file setup" {
		t.Fatalf("RunCommand = %q", got)
	}
}
