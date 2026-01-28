package layer1

import "strings"

// FuzzyMatch checks if the input is close enough to any key in the catalog.
// It uses a simple Levenshtein distance algorithm.
// Returns the command and true if found, empty string and false otherwise.
func FuzzyMatch(input string, catalog map[string]string) (string, bool) {
	threshold := 2 // Allow 2 edits
	minDist := 100
	closestKey := ""

	for key := range catalog {
        // Optimization: if lengths differ too much, skip
        if abs(len(key) - len(input)) > threshold {
            continue
        }

		dist := levenshtein(input, key)
		if dist < minDist {
			minDist = dist
			closestKey = key
		}
	}

	if minDist <= threshold {
		return catalog[closestKey], true
	}
	return "", false
}

func levenshtein(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

    // Optimization: run on runes to handle UTF-8 correctly, though for commands ASCII is likely
	r1 := []rune(s1)
	r2 := []rune(s2)

	rows := len(r1) + 1
	cols := len(r2) + 1

    // Use two rows to save memory instead of full matrix
    // O(min(m,n)) space complexity
	current := make([]int, cols)
    prev := make([]int, cols)

	for j := 0; j < cols; j++ {
		prev[j] = j
	}

	for i := 1; i < rows; i++ {
        current[0] = i
		for j := 1; j < cols; j++ {
			cost := 0
			if r1[i-1] != r2[j-1] {
				cost = 1
			}
			current[j] = min(
				min(current[j-1]+1, prev[j]+1), // insertion, deletion
				prev[j-1]+cost,                 // substitution
			)
		}
        // Swap arrays
        prev, current = current, prev
	}

	return prev[cols-1]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func abs(x int) int {
    if x < 0 {
        return -x
    }
    return x
}

// PreProcess applies common verb mappings and normalization
func PreProcess(input string) string {
    input = strings.ToLower(strings.TrimSpace(input))
    
    parts := strings.Fields(input)
    if len(parts) > 0 {
        if val, ok := VerbAliases[parts[0]]; ok {
            parts[0] = val
            return strings.Join(parts, " ")
        }
    }
    return input
}
