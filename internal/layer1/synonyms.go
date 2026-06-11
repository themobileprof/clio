package layer1

// slangExpansions maps casual / regional words to intent tokens (one-way to limit false positives).
var slangExpansions = map[string][]string{
	"misbehaving": {"stuck", "jam"},
	"misbehave":   {"stuck", "jam"},
	"acting":      {"stuck", "jam"},
	"frozen":      {"stuck", "jam"},
	"freeze":      {"stuck"},
	"lagging":     {"slow", "memory"},
	"lag":         {"slow"},
	"sluggish":    {"slow", "memory"},
	"filled":      {"full", "space"},
	"exhausted":   {"full", "space"},
	"connectivity": {"network", "internet"},
	"offline":     {"network"},
	"online":      {"network", "internet"},
	"screenshot":  {"file"},
	"picture":     {"file"},
	"pic":         {"file"},
	"homework":    {"assignment", "file"},
	"slides":      {"pdf", "file"},
	"repo":        {"git"},
	"github":      {"git"},
	"somehow":     {}, // stripped — too vague alone
}

func applySlangExpansions(set map[string]bool) {
	for word, targets := range slangExpansions {
		if !set[word] {
			continue
		}
		for _, t := range targets {
			if t != "" {
				set[t] = true
			}
		}
	}
}
