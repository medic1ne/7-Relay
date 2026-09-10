package tokensaver

import (
	"regexp"
	"strings"
)

// RTK compresses tool output to save tokens
func RTKCompress(input string) string {
	// Detect tool output patterns
	if isGitDiff(input) {
		return compressGitDiff(input)
	}
	if isGrepOutput(input) {
		return compressGrepOutput(input)
	}
	if isLsOutput(input) {
		return compressLsOutput(input)
	}
	return input
}

func isGitDiff(s string) bool {
	return strings.Contains(s, "diff --git") || strings.Contains(s, "--- a/") || strings.Contains(s, "+++ b/")
}

func isGrepOutput(s string) bool {
	lines := strings.Split(s, "\n")
	if len(lines) < 2 {
		return false
	}
	// Check if lines follow file:line:content pattern
	for _, line := range lines[:min(3, len(lines))] {
		if matched, _ := regexp.MatchString(`^[^:]+:\d+:`, line); matched {
			return true
		}
	}
	return false
}

func isLsOutput(s string) bool {
	lines := strings.Split(s, "\n")
	if len(lines) < 3 {
		return false
	}
	// Check if most lines look like ls output (no colons, spaces between items)
	colonCount := 0
	for _, line := range lines {
		if strings.Contains(line, ":") {
			colonCount++
		}
	}
	return colonCount < len(lines)/3
}

func compressGitDiff(diff string) string {
	lines := strings.Split(diff, "\n")
	var result []string
	var skipHunks bool

	for _, line := range lines {
		// Keep file headers
		if strings.HasPrefix(line, "diff --git") || strings.HasPrefix(line, "---") || strings.HasPrefix(line, "+++") {
			result = append(result, line)
			skipHunks = false
			continue
		}

		// Skip binary files
		if strings.Contains(line, "Binary files") {
			skipHunks = true
			continue
		}

		// Keep context lines and changes, skip empty hunks
		if !skipHunks {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

func compressGrepOutput(output string) string {
	lines := strings.Split(output, "\n")
	var result []string

	// Group by file, show first 3 matches per file
	currentFile := ""
	matchCount := 0

	for _, line := range lines {
		parts := strings.SplitN(line, ":", 3)
		if len(parts) >= 3 {
			file := parts[0]
			if file != currentFile {
				currentFile = file
				matchCount = 0
				result = append(result, "\n--- "+file+" ---")
			}
			matchCount++
			if matchCount <= 3 {
				result = append(result, line)
			} else if matchCount == 4 {
				result = append(result, "  ... (more matches)")
			}
		} else {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

func compressLsOutput(output string) string {
	// ls output is already compact, just ensure single spaces
	return strings.Join(strings.Fields(output), " ")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Caveman mode makes output terse
func CavemanCompress(input string) string {
	lines := strings.Split(input, "\n")
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		result = append(result, trimmed)
	}
	return strings.Join(result, "\n")
}

// Ponytail mode injects YAGNI prompt prefix
func PonytailPrefix(level string) string {
	switch level {
	case "lite":
		return "[SYSTEM] Be concise. Build what's asked, name lazier alternative if exists.\n"
	case "full":
		return "[SYSTEM] YAGNI enforced: stdlib > native > existing deps > one-liner > minimal code. No unrequested abstractions.\n"
	case "ultra":
		return "[SYSTEM] YAGNI EXTREME: deletion first. Ship one-liner. Challenge requirement in same response. Zero abstractions.\n"
	default:
		return ""
	}
}
