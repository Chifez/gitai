// Package prompt builds LLM prompts from git diffs and options.
// No HTTP calls — just string construction.
package prompt

import (
	"fmt"
	"strings"
)

// Options configures how the prompt is built.
type Options struct {
	Style       string // "conventional", "simple", "emoji"
	MaxLength   int    // max chars in subject line
	Lang        string // language for commit message
	Context     string // optional user-supplied hint
	IncludeBody bool   // include multi-line body
}

// emojiMap maps conventional commit types to emojis.
var emojiMap = map[string]string{
	"feat":     "✨",
	"fix":      "🐛",
	"docs":     "📝",
	"style":    "💅",
	"refactor": "♻️",
	"perf":     "⚡",
	"test":     "✅",
	"chore":    "🔧",
	"ci":       "🚀",
	"build":    "📦",
	"revert":   "⏪",
}

// BuildPrompt constructs the full LLM prompt from a diff and options.
func BuildPrompt(diff string, opts Options) string {
	var sb strings.Builder

	sb.WriteString("You are an expert developer writing a git commit message. ")
	sb.WriteString("Return ONLY the commit message. No preamble, no explanation, no markdown code fences.\n\n")

	switch opts.Style {
	case "simple":
		sb.WriteString("Write a single concise sentence. No type prefix, no scope, no body.\n")
	case "emoji":
		sb.WriteString("Use the Conventional Commits format: type(scope): description\n")
		sb.WriteString("Prepend the matching emoji before the type.\n")
		sb.WriteString("Emoji mapping:\n")
		for commitType, emoji := range emojiMap {
			sb.WriteString(fmt.Sprintf("  %s = %s\n", commitType, emoji))
		}
		sb.WriteString("\n")
		sb.WriteString("Valid types: feat, fix, docs, style, refactor, perf, test, chore, ci, build, revert\n")
	default:
		sb.WriteString("Use the Conventional Commits 1.0.0 format: type(scope): description\n")
		sb.WriteString("Valid types: feat, fix, docs, style, refactor, perf, test, chore, ci, build, revert\n")
	}

	if opts.MaxLength > 0 {
		sb.WriteString(fmt.Sprintf("Keep the subject line under %d characters.\n", opts.MaxLength))
	}

	if opts.IncludeBody && opts.Style != "simple" {
		sb.WriteString("Include a brief multi-line body explaining the what and why of the changes.\n")
	} else {
		sb.WriteString("Do NOT include a body — subject line only.\n")
	}

	if opts.Lang != "" && strings.ToLower(opts.Lang) != "english" {
		sb.WriteString(fmt.Sprintf("Write the commit message in %s.\n", opts.Lang))
	}

	if opts.Context != "" {
		sb.WriteString(fmt.Sprintf("\nAdditional context from the developer: %s\n", opts.Context))
	}

	sb.WriteString("\nHere is the staged diff:\n\n")
	sb.WriteString(diff)

	return sb.String()
}
