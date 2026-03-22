// Package git provides pure git operations using os/exec.
package git

import "regexp"

const (
	// MaxTokens is the approximate maximum token budget for the diff.
	MaxTokens = 8000
	charsPerToken = 4
	maxChars      = MaxTokens * charsPerToken
)

// SanitizeDiff cleans up the diff for the LLM. It replaces binary file
// diff output with a simple summary and truncates the string if it
// exceeds the token budget. Returns the cleaned diff and a bool indicating
// whether it was truncated.
func SanitizeDiff(diff string) (string, bool) {
	// Summarise binary file changes: "Binary files a/foo and b/foo differ"
	re := regexp.MustCompile(`(?m)^Binary files a/(.+) and b/.+ differ$`)
	diff = re.ReplaceAllString(diff, "Binary file modified: $1")

	truncated := false
	if len(diff) > maxChars {
		diff = diff[:maxChars]
		truncated = true
	}
	return diff, truncated
}
