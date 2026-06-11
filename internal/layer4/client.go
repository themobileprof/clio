package layer4

import (
	"bytes"
	"clio/internal/config"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"
)

// SearchRequest matches clipilot POST /api/commands/search body.
type SearchRequest struct {
	Query string `json:"query"`
	OS    string `json:"os,omitempty"`
	Arch  string `json:"arch,omitempty"`
}

// CommandResult is the normalized client-side remote hit.
type CommandResult struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Usage       string `json:"usage"`
	Cached      bool   `json:"cached,omitempty"`
}

// clipilotSearchResponse matches server/handlers/commands.go on clipilot.
type clipilotSearchResponse struct {
	Candidates []struct {
		Name        string   `json:"name"`
		Description string   `json:"description"`
		Category    string   `json:"category"`
		UseCases    []string `json:"use_cases"`
		Keywords    []string `json:"keywords"`
		Usage       string   `json:"usage"`
	} `json:"candidates"`
	Message string `json:"message"`
	Cached  bool   `json:"cached"`
}

// Legacy response shape kept for backward compatibility.
type legacySearchResponse struct {
	Results []CommandResult `json:"results"`
}

var remoteClient = newHTTPClient(4 * time.Second)

// Search queries CLIPilot when local matching fails.
// Flow: local SQLite cache → POST /api/commands/search → cache result.
func Search(query string) ([]CommandResult, error) {
	if !config.ShouldUseRemote() {
		return nil, fmt.Errorf("remote search disabled")
	}

	if cached, ok := GetCached(query, config.RemoteCacheTTL()); ok {
		return []CommandResult{cached}, nil
	}

	results, err := searchRemote(query)
	if err != nil {
		return nil, err
	}
	if len(results) > 0 {
		_ = PutCached(query, results[0])
	}
	return results, nil
}

func searchRemote(query string) ([]CommandResult, error) {
	base := strings.TrimSuffix(config.GetRegistryURL(), "/")
	url := base + "/api/commands/search"

	body, err := json.Marshal(SearchRequest{
		Query: query,
		OS:    runtime.GOOS,
		Arch:  runtime.GOARCH,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Clio/1.0")

	resp, err := remoteClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote API error: %d", resp.StatusCode)
	}

	return parseSearchResponse(raw)
}

func parseSearchResponse(raw []byte) ([]CommandResult, error) {
	var modern clipilotSearchResponse
	if err := json.Unmarshal(raw, &modern); err == nil && len(modern.Candidates) > 0 {
		out := make([]CommandResult, 0, len(modern.Candidates))
		for _, c := range modern.Candidates {
			usage := c.Usage
			if usage == "" && len(c.UseCases) > 0 {
				usage = c.UseCases[0]
			}
			out = append(out, CommandResult{
				Name:        c.Name,
				Description: c.Description,
				Usage:       usage,
				Cached:      modern.Cached,
			})
		}
		return out, nil
	}

	var legacy legacySearchResponse
	if err := json.Unmarshal(raw, &legacy); err == nil && len(legacy.Results) > 0 {
		return legacy.Results, nil
	}

	return nil, fmt.Errorf("no remote results")
}

// Ping checks registry reachability without LLM cost.
func Ping() error {
	base := strings.TrimSuffix(config.GetRegistryURL(), "/")
	resp, err := remoteClient.Get(base + "/health")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health status %d", resp.StatusCode)
	}
	return nil
}
