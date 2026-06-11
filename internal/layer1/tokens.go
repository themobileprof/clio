package layer1

import "strings"

// conversationalStopwords are removed for catalog scoring but question words are kept.
var conversationalStopwords = map[string]bool{
	"i": true, "want": true, "to": true, "can": true, "you": true,
	"please": true, "the": true, "a": true, "an": true, "is": true,
	"of": true, "in": true, "on": true, "for": true, "and": true,
	"or": true, "with": true, "help": true, "me": true, "just": true,
	"like": true, "need": true, "tell": true, "someone": true, "something": true,
	"this": true, "that": true, "from": true, "into": true, "about": true,
	"would": true, "could": true, "should": true, "does": true, "did": true,
	"will": true, "am": true, "are": true, "was": true, "be": true, "been": true,
	"have": true, "has": true, "had": true, "get": true, "got": true,
	"way": true, "best": true, "know": true, "use": true, "using": true,
	"command": true, "terminal": true, "linux": true, "termux": true,
}

// irregularPlurals maps plural/ing forms to catalog keys.
var irregularForms = map[string]string{
	"running": "run", "listing": "list", "copying": "copy",
	"deleting": "delete", "moving": "move", "searching": "search",
	"processes": "process", "files": "file", "folders": "folder",
	"directories": "directory", "permissions": "permission",
	"archives": "archive", "images": "file", "photos": "file",
	"apps": "process", "programs": "process", "tasks": "process",
	"chek": "check", "chekc": "check", "spce": "space", "memry": "memory",
}

// specificNouns are more precise than the generic "file" fallback.
var specificNouns = map[string]bool{
	"process": true, "log": true, "directory": true, "folder": true,
	"disk": true, "memory": true, "ram": true, "space": true,
	"network": true, "port": true, "permission": true, "archive": true,
	"zip": true, "package": true, "ip": true, "cpu": true,
	"path": true, "end": true, "start": true, "hidden": true, "all": true,
}

// Tokenize splits input into normalized, stemmed tokens for matching.
func Tokenize(input string) []string {
	raw := strings.Fields(strings.ToLower(input))
	out := make([]string, 0, len(raw))
	seen := make(map[string]bool)

	add := func(tok string) {
		if tok == "" || seen[tok] {
			return
		}
		seen[tok] = true
		out = append(out, tok)
	}

	for _, t := range raw {
		t = strings.Trim(t, "?!.,;:\"'()")
		if t == "" || conversationalStopwords[t] {
			continue
		}
		stemmed := Stem(t)
		add(stemmed)
		if v, ok := VerbAliases[stemmed]; ok {
			add(v)
		}
		if n, ok := NounAliases[stemmed]; ok {
			add(n)
		}
	}
	return out
}

// TokenSet returns a set of tokens plus reverse alias hits.
func TokenSet(input string) map[string]bool {
	tokens := Tokenize(input)
	set := make(map[string]bool, len(tokens)*2)
	for _, t := range tokens {
		set[t] = true
	}
	for alias, verb := range VerbAliases {
		if set[alias] {
			set[verb] = true
		}
	}
	for alias, noun := range NounAliases {
		if set[alias] {
			set[noun] = true
		}
	}
	return set
}

func hasToken(set map[string]bool, words ...string) bool {
	for _, w := range words {
		if set[w] || set[Stem(w)] {
			return true
		}
	}
	return false
}
