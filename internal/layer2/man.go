package layer2

import (
	"bufio"
	"clio/internal/config"
	"clio/internal/safeexec"
	"sort"
	"strings"
)

type Result struct {
	Name        string
	Description string
	Score       int
}

// Search queries the system manual pages for the given keywords.
// It returns a list of matching commands sorted by relevance.
func Search(keywords []string) []Result {
	if len(keywords) == 0 {
		return nil
	}

	// On lite profile only search the top keyword to avoid multiple subprocess spawns
	maxKeywords := len(keywords)
	if config.IsLiteProfile() && maxKeywords > 1 {
		maxKeywords = 1
	}

	matches := make(map[string]*Result)

	for _, kw := range keywords[:maxKeywords] {
		cmd := safeexec.Command("man", "-k", "-s", "1,8", kw)
		output, err := cmd.StdoutPipe()
		if err != nil {
			continue
		}
		if err := cmd.Start(); err != nil {
			continue
		}

		scanner := bufio.NewScanner(output)
		for scanner.Scan() {
			line := scanner.Text()
			parts := strings.SplitN(line, " - ", 2)
			if len(parts) < 2 {
				continue
			}

			namePart := strings.TrimSpace(parts[0])
			nameIdx := strings.Index(namePart, " (")
			rawName := namePart
			if nameIdx != -1 {
				rawName = namePart[:nameIdx]
			}
			if commaIdx := strings.Index(rawName, ","); commaIdx != -1 {
				rawName = strings.TrimSpace(rawName[:commaIdx])
			}

			desc := strings.TrimSpace(parts[1])

			if _, exists := matches[rawName]; !exists {
				matches[rawName] = &Result{
					Name:        rawName,
					Description: desc,
					Score:       0,
				}
			}
			matches[rawName].Score += 10
		}
		cmd.Wait()
	}

	results := make([]Result, 0, len(matches))
	for _, res := range matches {
		for _, kw := range keywords {
			if res.Name == kw {
				res.Score += 50
			}
			if strings.Contains(res.Description, kw) {
				res.Score += 5
			}
		}
		results = append(results, *res)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if len(results) > 5 {
		return results[:5]
	}

	return results
}

// IsInstalled checks if a given command is executable on the system
func IsInstalled(cmdName string) bool {
	_, err := safeexec.LookPath(cmdName)
	return err == nil
}
