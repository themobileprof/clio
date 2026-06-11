package layer1

// pidginAliases maps Nigerian Pidgin and campus slang to English tokens for phrase matching.
// Students often mix English with Pidgin when searching on Termux.
var pidginAliases = map[string][]string{
	"wetin":      {"what"},
	"abi":        {},
	"abeg":       {},
	"sha":        {},
	"sef":        {},
	"na":         {},
	"wan":        {"want"},
	"comot":      {"remove", "delete"},
	"clear":      {"delete", "remove"},
	"don":        {"already"},
	"finish":     {"full", "space"},
	"hang":       {"stuck"},
	"jam":        {"stuck"},
	"jammed":     {"stuck"},
	"gree":       {"work", "respond"},
	"slow":       {"memory"},
	"data":       {"internet", "network"},
	"sub":        {"network"},
	"send":       {"copy"},
	"receive":    {"download"},
	"inside":     {"here"},
	"put":        {"create"},
	"notes":      {"file"},
	"note":       {"file"},
	"lecture":    {"file", "pdf"},
	"assignment": {"file"},
	"coursework": {"file"},
	"project":    {"folder", "directory"},
	"coding":     {"code"},
	"programming": {"code"},
	"repo":       {"git"},
	"github":     {"git"},
	"termux":     {},
	"phone":      {},
	"scratch":    {"create"},
	"everywhere": {"all"},
	"stuck":      {"stuck"},
	"frozen":     {"stuck"},
	"respond":    {"work"},
	"connect":    {"network"},
	"offline":    {"network"},
}

// nigerianStopwords are conversational fillers stripped during catalog tokenization.
var nigerianStopwords = map[string]bool{
	"abeg": true, "sha": true, "na": true, "sef": true, "abi": true,
	"o": true, "dem": true, "una": true, "wey": true,
	"make": true, "e": true, "im": true, "em": true,
}

func applyPidginAliases(set map[string]bool) {
	for word, targets := range pidginAliases {
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
