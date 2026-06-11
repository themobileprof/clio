package intent

import (
	"clio/internal/layer1"
	"strings"
)

// IsolateKeywords extracts meaningful search terms from a natural language query.
func IsolateKeywords(input string) []string {
	tokens := layer1.Tokenize(input)
	if len(tokens) == 0 {
		return nil
	}

	// Drop extra conversational filler not already removed by Tokenize.
	extraStop := map[string]bool{
		"do": true, "how": true, "what": true, "where": true, "when": true,
		"why": true, "there": true, "here": true, "my": true, "out": true,
	}

	out := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if extraStop[token] {
			continue
		}
		out = append(out, token)
	}
	if len(out) == 0 {
		return expandKeywordSynonyms(tokens)
	}
	return expandKeywordSynonyms(out)
}

// Legacy helper kept for tests that expect verb synonym expansion in keywords.
func expandKeywordSynonyms(keywords []string) []string {
	verbMap := map[string]string{
		"duplicate": "cp",
		"copy":      "cp",
		"move":      "mv",
		"rename":    "mv",
		"remove":    "rm",
		"delete":    "rm",
		"list":      "ls",
		"show":      "ls",
		"folder":    "directory",
	}

	seen := make(map[string]bool, len(keywords))
	out := make([]string, 0, len(keywords))
	for _, k := range keywords {
		k = strings.ToLower(k)
		if val, ok := verbMap[k]; ok {
			k = val
		}
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, k)
	}
	return out
}
