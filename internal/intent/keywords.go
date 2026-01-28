package intent

import (
	"strings"
)

// IsolateKeywords extracts the core search terms from a natural language query.
func IsolateKeywords(input string) []string {
	// 1. Tokenization
	tokens := strings.Fields(strings.ToLower(input))

	// 2. Stopword Removal
	stopwords := map[string]bool{
		"i": true, "want": true, "to": true, "how": true, "do": true,
		"can": true, "you": true, "please": true, "the": true, "a": true,
		"an": true, "is": true, "of": true, "in": true, "on": true,
		"for": true, "and": true, "or": true, "with": true, "help": true,
		"me": true,
	}

	var keywords []string
	for _, token := range tokens {
		// Clean punctuation
		token = strings.Trim(token, "?!.,;:\"'()")
		if token == "" {
			continue
		}
		if !stopwords[token] {
			keywords = append(keywords, token)
		}
	}
    
    // 3. Synonym Expansion (Basic)
    // Map common verbs to system terms if needed, 
    // although Layer 1 PreProcess also handles some of this.
    // For search, we might want to keep original or expand.
    // Let's keep it simple for now as per guide.
    
    // Limit to top keywords if too many?
    // Guide example: "How do I duplicate a directory?" -> ["duplicate", "directory"] -> ["copy", "directory"]
    
    finalKeywords := make([]string, 0, len(keywords))
    verbMap := map[string]string{
        "duplicate": "cp",
        "copy":      "cp",
        "move":      "mv",
        "rename":    "mv",
        "remove":    "rm",
        "delete":    "rm",
        "list":      "ls",
        "show":      "ls", // context dependent, but often ls
        "change":    "cd",
        "folder":    "directory",
    }
    
    for _, k := range keywords {
        if val, ok := verbMap[k]; ok {
            finalKeywords = append(finalKeywords, val)
        } else {
            finalKeywords = append(finalKeywords, k)
        }
    }

	return finalKeywords
}
