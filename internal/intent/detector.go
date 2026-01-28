package intent

import (
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

// Detect determines the best command for the given input using a 4-layer strategy.
func Detect(input string) (*DetectionResult, error) {
    // Layer 1: Static Offline Map (Optimized Verb-Noun)
    verb, noun := layer1.ParseIntent(input)
    if verb != "" && noun != "" {
        if nouns, ok := layer1.VerbNounCatalog[verb]; ok {
            if entry, ok := nouns[noun]; ok {
                 return &DetectionResult{
                    Command:     entry.Cmd,
                    Description: entry.Desc,
                    Source:      "static",
                    Confidence:  1.0,
                }, nil
            }
        }
    }

    // Fallback to old flat catalog for specific idioms if needed,
    // or rely on Fuzzy matching against key phrases constructed from the map?
    
    // Let's keep the old Fuzzy Match but adapt it to flattening the new map?
    // Or just skip fuzzy for now as the 80% resolution implies the map covers most.
    // We can iterate the map for fuzzy matching.
    
    // Isolate keywords for further layers
    keywords := IsolateKeywords(input)
    if len(keywords) == 0 {
        return nil, fmt.Errorf("no keywords found")
    }

    // Layer 2: Man Pages (System)
    manResults := layer2.Search(keywords)
    if len(manResults) > 0 {
        top := manResults[0]
        // If score is decent
        if top.Score > 10 {
             return &DetectionResult{
                Command:     top.Name,
                Description: top.Description,
                Source:      "man",
                Confidence:  0.8, // Variable based on score ideally
            }, nil
        }
    }

    // Layer 3: Client Modules (Local DB)
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

    // Layer 4: Remote Fallback
    remoteResults, err := layer4.Search(input) // Send full input to LLM/remote
    if err == nil && len(remoteResults) > 0 {
        top := remoteResults[0]
         return &DetectionResult{
            Command:     top.Name,
            Description: top.Description + "\nUsage: " + top.Usage,
            Source:      "remote",
            Confidence:  0.7,
        }, nil
    }

    return nil, fmt.Errorf("no match found")
}
