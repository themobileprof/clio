package intent

import (
	"clio/internal/config"
	"clio/internal/layer1"
	"clio/internal/layer2"
	"clio/internal/layer3"
	"clio/internal/layer4"
	"fmt"
)

type DetectionResult struct {
	Command     string
	Description string
	Source      string // "static", "man", "module", "remote"
	Confidence  float64
}

// Detect determines the best command for natural-language input.
// Layer 1 tries conversational phrases first, then full-sentence catalog scoring.
func Detect(input string) (*DetectionResult, error) {
	if result, ok := detectStatic(input); ok {
		return result, nil
	}

	keywords := IsolateKeywords(input)
	if len(keywords) == 0 {
		return nil, fmt.Errorf("no keywords found")
	}

	lite := config.IsLiteProfile()

	// Layer 2: Man Pages — skipped on lite profile
	if !lite {
		manResults := layer2.Search(keywords)
		if len(manResults) > 0 {
			top := manResults[0]
			if top.Score > 10 {
				return &DetectionResult{
					Command:     top.Name,
					Description: top.Description,
					Source:      "man",
					Confidence:  0.8,
				}, nil
			}
		}
	}

	// Layer 3: Local modules
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

	// Layer 4: Remote API — skipped on lite profile
	if !lite {
		remoteResults, err := layer4.Search(input)
		if err == nil && len(remoteResults) > 0 {
			top := remoteResults[0]
			return &DetectionResult{
				Command:     top.Name,
				Description: top.Description + "\nUsage: " + top.Usage,
				Source:      "remote",
				Confidence:  0.7,
			}, nil
		}
	}

	return nil, fmt.Errorf("no match found")
}

func detectStatic(input string) (*DetectionResult, bool) {
	// 1. Idiomatic full-sentence phrases ("where am I", "how much memory")
	if entry, ok := layer1.MatchPhrase(input); ok {
		return staticResult(entry, 0.98), true
	}

	// 2. Score the whole sentence against the verb-noun catalog
	if entry, ok := layer1.MatchCatalog(input); ok {
		return staticResult(entry, 0.95), true
	}

	// 3. Classic verb+noun parse for short queries
	verb, noun := layer1.ParseIntent(input)
	if entry, ok := layer1.LookupVerbNoun(verb, noun); ok {
		return staticResult(entry, 1.0), true
	}

	return nil, false
}

func staticResult(entry layer1.CommandEntry, confidence float64) *DetectionResult {
	return &DetectionResult{
		Command:     entry.Cmd,
		Description: entry.Desc,
		Source:      "static",
		Confidence:  confidence,
	}
}
