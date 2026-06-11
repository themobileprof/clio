package intent

import (
	"clio/internal/config"
	"clio/internal/layer1"
	"clio/internal/layer2"
	"clio/internal/layer3"
	"clio/internal/layer4"
	"clio/internal/setup"
	"fmt"
	"strings"
)

type DetectionResult struct {
	Command     string
	Description string
	Source      string // "setup", "static", "fuzzy", "man", "module", "remote", "remote-cached"
	Confidence  float64
}

// Detect determines the best command for natural-language input.
func Detect(input string) (*DetectionResult, error) {
	// Layer 0: Termux setup wizard (first-class command)
	if setup.IsSetupRequest(input) {
		return &DetectionResult{
			Command:     setup.RunCommand(),
			Description: setup.ShortDescription(),
			Source:      "setup",
			Confidence:  1.0,
		}, nil
	}

	if result, ok := detectStatic(input); ok {
		return result, nil
	}

	keywords := IsolateKeywords(input)
	if len(keywords) == 0 {
		return tryRemote(input)
	}

	// Layer 2: man pages — skipped on lite profile (subprocess heavy)
	if !config.IsLiteProfile() {
		manResults := layer2.Search(keywords)
		if len(manResults) > 0 && manResults[0].Score > 10 {
			top := manResults[0]
			return &DetectionResult{
				Command:     top.Name,
				Description: top.Description,
				Source:      "man",
				Confidence:  0.8,
			}, nil
		}
	}

	// Layer 3: local automation modules
	modules, err := layer3.SearchModules(keywords)
	if err == nil && len(modules) > 0 {
		top := modules[0]
		return &DetectionResult{
			Command:     top.Command,
			Description: top.Description,
			Source:      "module",
			Confidence:  0.85,
		}, nil
	}

	return tryRemote(input)
}

func detectStatic(input string) (*DetectionResult, bool) {
	if entry, ok := layer1.MatchPhrase(input); ok {
		return staticResult(entry, "static", 0.98), true
	}
	if entry, ok := layer1.MatchCatalog(input); ok {
		return staticResult(entry, "static", 0.95), true
	}
	verb, noun := layer1.ParseIntent(input)
	if entry, ok := layer1.LookupVerbNoun(verb, noun); ok {
		return staticResult(entry, "static", 1.0), true
	}

	// Fuzzy + slang expansion — conservative retry before network
	if entry, kind, ok := layer1.MatchWithFuzzy(input); ok {
		conf := 0.88
		if kind == "phrase" {
			conf = 0.92
		}
		return staticResult(entry, "fuzzy", conf), true
	}

	return nil, false
}

func tryRemote(input string) (*DetectionResult, error) {
	if !config.ShouldUseRemote() {
		return nil, fmt.Errorf("no match found")
	}

	remoteResults, err := layer4.Search(input)
	if err != nil || len(remoteResults) == 0 {
		return nil, fmt.Errorf("no match found")
	}

	top := remoteResults[0]
	if isWeakRemoteResult(top) {
		return nil, fmt.Errorf("no match found")
	}

	source := "remote"
	if top.Cached {
		source = "remote-cached"
	}

	desc := top.Description
	if top.Usage != "" {
		desc = strings.TrimSpace(desc + "\nUsage: " + top.Usage)
	}

	return &DetectionResult{
		Command:     top.Name,
		Description: desc,
		Source:      source,
		Confidence:  0.7,
	}, nil
}

// isWeakRemoteResult filters generic clipilot fallbacks that cause false positives.
func isWeakRemoteResult(r layer4.CommandResult) bool {
	name := strings.ToLower(strings.TrimSpace(r.Name))
	if name == "echo" && strings.Contains(strings.ToLower(r.Description), "line of text") {
		return true
	}
	return name == ""
}

func staticResult(entry layer1.CommandEntry, source string, confidence float64) *DetectionResult {
	return &DetectionResult{
		Command:     entry.Cmd,
		Description: entry.Desc,
		Source:      source,
		Confidence:  confidence,
	}
}
