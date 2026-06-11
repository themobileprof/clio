package layer1

import "strings"

// phraseRule matches when every term appears in the tokenized query.
type phraseRule struct {
	terms []string
	entry CommandEntry
}

// PhraseCatalog handles full-sentence and idiomatic queries common in natural search.
var PhraseCatalog = []phraseRule{
	// location / navigation
	{[]string{"where", "am"}, CommandEntry{"pwd", "Show current directory"}},
	{[]string{"current", "directory"}, CommandEntry{"pwd", "Show current working directory"}},
	{[]string{"working", "directory"}, CommandEntry{"pwd", "Show current working directory"}},
	{[]string{"view", "directory"}, CommandEntry{"pwd", "Show current directory path"}},
	{[]string{"current", "folder"}, CommandEntry{"pwd", "Show current folder"}},
	{[]string{"files", "here"}, CommandEntry{"ls -la", "List files in this folder"}},
	{[]string{"files", "folder"}, CommandEntry{"ls -la", "List files in this folder"}},
	{[]string{"what", "files"}, CommandEntry{"ls -la", "List files"}},

	// system info
	{[]string{"memory", "usage"}, CommandEntry{"free -h", "Check memory usage"}},
	{[]string{"ram", "usage"}, CommandEntry{"free -h", "Check RAM usage"}},
	{[]string{"how", "much", "memory"}, CommandEntry{"free -h", "Check how much memory is in use"}},
	{[]string{"disk", "usage"}, CommandEntry{"df -h", "Check disk usage"}},
	{[]string{"disk", "space"}, CommandEntry{"df -h", "Check available disk space"}},
	{[]string{"how", "much", "space"}, CommandEntry{"df -h", "Check how much disk space is left"}},
	{[]string{"storage", "left"}, CommandEntry{"df -h", "Check remaining storage"}},
	{[]string{"running", "out", "space"}, CommandEntry{"df -h", "Check disk space"}},
	{[]string{"public", "ip"}, CommandEntry{"curl ifconfig.me", "Check your public IP address"}},
	{[]string{"my", "ip"}, CommandEntry{"curl ifconfig.me", "Check your public IP address"}},
	{[]string{"what", "ip"}, CommandEntry{"curl ifconfig.me", "Check your public IP address"}},
	{[]string{"internet", "working"}, CommandEntry{"ping -c 4 google.com", "Test internet connection"}},
	{[]string{"wifi", "working"}, CommandEntry{"ping -c 4 google.com", "Test network connection"}},

	// processes
	{[]string{"running", "process"}, CommandEntry{"ps aux", "List running processes"}},
	{[]string{"process", "running"}, CommandEntry{"ps aux", "List running processes"}},
	{[]string{"what", "running"}, CommandEntry{"ps aux", "See what is running"}},
	{[]string{"apps", "running"}, CommandEntry{"ps aux", "List running apps"}},
	{[]string{"kill", "process"}, CommandEntry{"kill", "Stop a running process"}},
	{[]string{"stop", "program"}, CommandEntry{"kill", "Stop a running program"}},
	{[]string{"stop", "process"}, CommandEntry{"kill", "Stop a running process"}},
	{[]string{"program", "stuck"}, CommandEntry{"kill", "Force-stop a stuck program"}},

	// files / logs
	{[]string{"end", "log"}, CommandEntry{"tail -f", "Watch the end of a log file"}},
	{[]string{"last", "lines"}, CommandEntry{"tail -n 20", "Show the last lines of a file"}},
	{[]string{"beginning", "file"}, CommandEntry{"head", "Show the start of a file"}},
	{[]string{"large", "file"}, CommandEntry{"find . -type f -size +100M", "Find large files"}},
	{[]string{"find", "pdf"}, CommandEntry{`find . -name "*.pdf"`, "Find PDF files"}},
	{[]string{"search", "text"}, CommandEntry{"grep -r", "Search for text inside files"}},
	{[]string{"text", "files"}, CommandEntry{"grep -r", "Search for text inside files"}},
	{[]string{"hidden", "file"}, CommandEntry{"ls -la", "List hidden files"}},

	// archives
	{[]string{"unzip"}, CommandEntry{"unzip", "Extract a zip file"}},
	{[]string{"extract", "zip"}, CommandEntry{"unzip", "Extract a zip archive"}},
	{[]string{"zip", "file"}, CommandEntry{"unzip", "Extract a zip file"}},
	{[]string{"extract", "tar"}, CommandEntry{"tar -xzvf", "Extract a tar archive"}},
	{[]string{"tar", "file"}, CommandEntry{"tar -xzvf", "Extract a tar.gz file"}},
	{[]string{"compress", "zip"}, CommandEntry{"zip -r archive.zip", "Compress files into a zip"}},

	// permissions
	{[]string{"change", "permission"}, CommandEntry{"chmod", "Change file permissions"}},
	{[]string{"file", "permission"}, CommandEntry{"chmod", "Change file permissions"}},
	{[]string{"make", "executable"}, CommandEntry{"chmod +x", "Make a file executable"}},
	{[]string{"chmod", "executable"}, CommandEntry{"chmod +x", "Make a file executable"}},

	// typos / casual phrasing
	{[]string{"chek", "disk"}, CommandEntry{"df -h", "Check disk space"}},
	{[]string{"chek", "memory"}, CommandEntry{"free -h", "Check memory usage"}},
	{[]string{"list", "everything"}, CommandEntry{"ls -la", "List everything in this folder"}},
	{[]string{"delete", "everything"}, CommandEntry{"rm -rf *", "Delete everything here (careful!)"}},
}

// MatchPhrase finds the best phrase-rule match for conversational input.
func MatchPhrase(input string) (CommandEntry, bool) {
	set := phraseTokenSet(input)
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
		// Deprioritize ultra-common words that appear in many sentences
		for _, term := range rule.terms {
			if term == "in" || term == "the" {
				score -= 4
			}
		}
		if score > bestScore {
			bestScore = score
			best = rule.entry
		}
	}
	return best, bestScore > 0
}

// phraseTokenSet keeps question words that conversationalStopwords would drop.
func phraseTokenSet(input string) map[string]bool {
	raw := stringsFieldsLower(input)
	set := make(map[string]bool, len(raw)*2)
	for _, t := range raw {
		t = trimPunct(t)
		if t == "" {
			continue
		}
		stemmed := Stem(t)
		set[t] = true
		set[stemmed] = true
		if v, ok := VerbAliases[stemmed]; ok {
			set[v] = true
		}
	}
	return set
}

func phraseTermsMatch(set map[string]bool, terms []string) bool {
	for _, term := range terms {
		stemmed := Stem(term)
		if !set[term] && !set[stemmed] {
			return false
		}
	}
	return true
}

func stringsFieldsLower(s string) []string {
	return strings.Fields(strings.ToLower(s))
}

func trimPunct(s string) string {
	return strings.Trim(s, "?!.,;:\"'()")
}
