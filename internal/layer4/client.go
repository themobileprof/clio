package layer4

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type SearchRequest struct {
	Query string `json:"query"`
    OS    string `json:"os"`
    Arch  string `json:"arch"`
}

type CommandResult struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Usage       string `json:"usage"`
}

type SearchResponse struct {
	Results []CommandResult `json:"results"`
}

// Search queries the remote API for commands.
func Search(query string) ([]CommandResult, error) {
	apiURL := "https://clipilot.themobileprof.com/api/commands/search"
	
	reqBody := SearchRequest{
		Query: query,
        OS:    "linux", // or runtime.GOOS
        Arch:  "arm64", // or runtime.GOARCH, simplified for now
	}
	
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
    req.Header.Set("User-Agent", "CLIPilot-Client/1.0")

	// Skip if offline (simple check, or just let it fail)
    // For now we just attempt the request
    
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote API error: %d", resp.StatusCode)
	}

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

	var searchResp SearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, err
	}

	return searchResp.Results, nil
}
