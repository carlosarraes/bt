package pr

import (
	"regexp"
	"strconv"
	"strings"
)

var hunkHeaderRe = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

// ParseAddedLinesByFile parses a unified diff and returns added line numbers
// per file. The map key is the post-image path (b/...) without the b/ prefix.
// Removed and context lines are not included.
func ParseAddedLinesByFile(diff string) map[string]map[int]bool {
	result := make(map[string]map[int]bool)
	if diff == "" {
		return result
	}

	var currentFile string
	var newLine int
	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "diff --git"):
			currentFile = ""
		case strings.HasPrefix(line, "+++ b/"):
			currentFile = strings.TrimPrefix(line, "+++ b/")
			if _, ok := result[currentFile]; !ok {
				result[currentFile] = make(map[int]bool)
			}
		case strings.HasPrefix(line, "+++ "):
			currentFile = ""
		case strings.HasPrefix(line, "@@"):
			m := hunkHeaderRe.FindStringSubmatch(line)
			if len(m) >= 2 {
				if n, err := strconv.Atoi(m[1]); err == nil {
					newLine = n
				}
			}
		case currentFile != "" && strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			result[currentFile][newLine] = true
			newLine++
		case currentFile != "" && strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
			// removed line: do not advance newLine
		case currentFile != "" && (strings.HasPrefix(line, " ") || line == ""):
			newLine++
		}
	}
	return result
}
