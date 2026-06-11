package layer4

import "testing"

func TestParseSearchResponseClipilotFormat(t *testing.T) {
	raw := []byte(`{
		"candidates": [{
			"name": "cp",
			"description": "Copy files and directories",
			"category": "file",
			"use_cases": ["cp source dest"],
			"keywords": ["copy"]
		}],
		"message": "Found 1 candidates",
		"cached": false
	}`)

	results, err := parseSearchResponse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Name != "cp" {
		t.Fatalf("unexpected results: %+v", results)
	}
	if results[0].Usage != "cp source dest" {
		t.Fatalf("usage = %q", results[0].Usage)
	}
}

func TestParseSearchResponseLegacyFormat(t *testing.T) {
	raw := []byte(`{"results":[{"name":"ls","description":"list","usage":"ls -la"}]}`)
	results, err := parseSearchResponse(raw)
	if err != nil || results[0].Name != "ls" {
		t.Fatalf("legacy parse failed: %+v err=%v", results, err)
	}
}

func TestHashQueryNormalized(t *testing.T) {
	a := hashQuery("  Check Disk SPACE ")
	b := hashQuery("check disk space")
	if a != b {
		t.Fatal("hash should normalize case and whitespace")
	}
}
