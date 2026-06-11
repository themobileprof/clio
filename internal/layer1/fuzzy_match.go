package layer1

// FuzzyExpandTokens adds catalog keys that are within edit distance 1 of query tokens.
// Only runs on words >= 5 chars to avoid false positives (e.g. "ls" -> "as").
func FuzzyExpandTokens(set map[string]bool) {
	if len(set) == 0 {
		return
	}

	candidates := collectCatalogWords()
	expanded := make(map[string]bool)

	for token := range set {
		if len(token) < 5 {
			continue
		}
		for _, word := range candidates {
			if abs(len(token)-len(word)) > 1 {
				continue
			}
			if levenshtein(token, word) == 1 {
				expanded[word] = true
			}
		}
	}

	for w := range expanded {
		set[w] = true
	}
}

func collectCatalogWords() []string {
	seen := make(map[string]bool)
	add := func(w string) {
		if w != "" && len(w) >= 4 {
			seen[w] = true
		}
	}

	for verb, nouns := range VerbNounCatalog {
		add(verb)
		for noun := range nouns {
			add(noun)
		}
	}
	for w := range VerbAliases {
		add(w)
	}
	for w := range NounAliases {
		add(w)
	}
	for _, rule := range PhraseCatalog {
		for _, t := range rule.terms {
			add(t)
		}
	}

	out := make([]string, 0, len(seen))
	for w := range seen {
		out = append(out, w)
	}
	return out
}

// MatchWithFuzzy retries catalog and phrase matching after fuzzy + slang expansion.
func MatchWithFuzzy(input string) (CommandEntry, string, bool) {
	set := TokenSet(input)
	applySlangExpansions(set)
	applyPidginAliases(set)
	FuzzyExpandTokens(set)

	if entry, ok := matchPhraseFromSet(set); ok {
		return entry, "phrase", true
	}
	if entry, ok := matchCatalogFromSet(set); ok {
		return entry, "catalog", true
	}
	return CommandEntry{}, "", false
}

func matchPhraseFromSet(set map[string]bool) (CommandEntry, bool) {
	bestScore := 0
	var best CommandEntry
	for _, rule := range PhraseCatalog {
		if !phraseTermsMatch(set, rule.terms) {
			continue
		}
		score := len(rule.terms) * 10
		for _, term := range rule.terms {
			if set[term] {
				score += 5
			}
		}
		if score > bestScore {
			bestScore = score
			best = rule.entry
		}
	}
	return best, bestScore > 0
}

func matchCatalogFromSet(set map[string]bool) (CommandEntry, bool) {
	bestScore := 0
	var best CommandEntry

	for verb, nouns := range VerbNounCatalog {
		verbScore := scoreVerb(verb, set)
		if verbScore == 0 {
			continue
		}
		for noun, entry := range nouns {
			nounScore := scoreNoun(noun, set)
			if nounScore == 0 {
				continue
			}
			score := verbScore + nounScore
			if noun == "file" && hasSpecificNounInSet(set) {
				score -= 8
			}
			if score > bestScore {
				bestScore = score
				best = entry
			}
		}
	}

	if bestScore >= 12 {
		return best, true
	}
	return CommandEntry{}, false
}
