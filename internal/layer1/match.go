package layer1

// MatchCatalog scores the full sentence against the verb-noun catalog.
// It prefers specific nouns (process, log, disk) over the generic "file".
func MatchCatalog(input string) (CommandEntry, bool) {
	set := TokenSet(input)
	if len(set) == 0 {
		return CommandEntry{}, false
	}

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
			if set["text"] && noun == "text" {
				score += 6
			}
			if set["text"] && noun == "file" {
				score -= 6
			}
			if score > bestScore {
				bestScore = score
				best = entry
			}
		}
	}

	// Noun-only queries: "memory usage", "disk space" (no explicit verb)
	if bestScore < 12 {
		if entry, ok := matchNounOnly(set); ok {
			return entry, true
		}
	}

	return best, bestScore >= 12
}

func scoreVerb(verb string, set map[string]bool) int {
	if set[verb] {
		return 10
	}
	for alias, target := range VerbAliases {
		if target == verb && set[alias] {
			return 9
		}
	}
	return 0
}

func scoreNoun(noun string, set map[string]bool) int {
	if set[noun] {
		return 10
	}
	for alias, target := range NounAliases {
		if target == noun && set[alias] {
			return 9
		}
	}
	return 0
}

func hasSpecificNounInSet(set map[string]bool) bool {
	for noun := range specificNouns {
		if set[noun] {
			return true
		}
	}
	return false
}

// nounOnlyHints maps co-occurring topic words to commands when no verb is found.
var nounOnlyHints = []struct {
	terms []string
	entry CommandEntry
}{
	{[]string{"memory"}, CommandEntry{"free -h", "Check memory usage"}},
	{[]string{"ram"}, CommandEntry{"free -h", "Check memory usage"}},
	{[]string{"disk"}, CommandEntry{"df -h", "Check disk space"}},
	{[]string{"storage"}, CommandEntry{"df -h", "Check disk space"}},
	{[]string{"space"}, CommandEntry{"df -h", "Check disk space"}},
	{[]string{"process"}, CommandEntry{"ps aux", "List running processes"}},
	{[]string{"permission"}, CommandEntry{"chmod", "Change file permissions"}},
}

func matchNounOnly(set map[string]bool) (CommandEntry, bool) {
	bestLen := 0
	var best CommandEntry
	for _, hint := range nounOnlyHints {
		if len(hint.terms) <= bestLen {
			continue
		}
		ok := true
		for _, term := range hint.terms {
			if !set[term] && !set[Stem(term)] {
				ok = false
				break
			}
		}
		if ok {
			bestLen = len(hint.terms)
			best = hint.entry
		}
	}
	return best, bestLen > 0
}

// LookupVerbNoun resolves an explicit verb+noun pair.
func LookupVerbNoun(verb, noun string) (CommandEntry, bool) {
	if verb == "" || noun == "" {
		return CommandEntry{}, false
	}
	if nouns, ok := VerbNounCatalog[verb]; ok {
		if entry, ok := nouns[noun]; ok {
			return entry, true
		}
	}
	return CommandEntry{}, false
}
