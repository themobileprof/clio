package layer2

import (
	"bufio"
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

	matches := make(map[string]*Result)
	
    // 1. Iterative Search through all keywords
	for _, kw := range keywords {
		// safeexec.Command wrapper would need to handle arguments. 
        // Using safeexec.LookPath to find 'man' first is good practice.
		// For simplicity, we assume safeexec.Command works if we import it.
        // We run: man -k -s 1,8 <keyword>
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
			// Format is usually: name (section) - description
			parts := strings.SplitN(line, " - ", 2)
			if len(parts) < 2 {
                // Try alternate format: name(section) ... description
                // Some man implementations vary.
                continue
			}

            // Clean up name part "grep (1)" -> "grep"
            namePart := strings.TrimSpace(parts[0])
            nameIdx := strings.Index(namePart, " (")
            rawName := namePart
            if nameIdx != -1 {
                rawName = namePart[:nameIdx]
            }
            // Remove multiple names "gzip, gunzip, zcat" -> pick first
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
			matches[rawName].Score += 10 // Base score for appearing in search
		}
        cmd.Wait() // Don't care about error return here, usually "nothing appropriate" is an error code
	}

	// 2. Post-Processing & Ranking
	results := make([]Result, 0, len(matches))
	for _, res := range matches {
        // Boost for exact keyword match in name
        for _, kw := range keywords {
            if res.Name == kw {
                res.Score += 50
            }
            if strings.Contains(res.Description, kw) {
                res.Score += 5
            }
        }
        
        // Boost for intersection overlap
        // (already partially covered by iterative frequency, but specific word overlap helps)
        
		results = append(results, *res)
	}

	// Sort by Score desc
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
    
    // Limit to top 5
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
