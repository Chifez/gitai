// Package ui provides terminal color and styling utilities.
// Centralizes all colored output so the rest of the codebase calls helpers
// instead of using color directly.
package ui

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

var (
	greenBold   = color.New(color.FgGreen, color.Bold)
	yellowBold  = color.New(color.FgYellow, color.Bold)
	redBold     = color.New(color.FgRed, color.Bold)
	cyanBold    = color.New(color.FgCyan, color.Bold)
	magenta     = color.New(color.FgMagenta)
	bold        = color.New(color.Bold)
	dim         = color.New(color.Faint)
	white       = color.New(color.FgWhite)
)

// Success prints a green success message with a checkmark.
func Success(format string, a ...interface{}) {
	greenBold.Fprintf(os.Stdout, "✔ "+format+"\n", a...)
}

// Warn prints a yellow warning message.
func Warn(format string, a ...interface{}) {
	yellowBold.Fprintf(os.Stderr, "⚠ "+format+"\n", a...)
}

// Error prints a red error message.
func Error(format string, a ...interface{}) {
	redBold.Fprintf(os.Stderr, "✖ "+format+"\n", a...)
}

// Info prints a dim informational message.
func Info(format string, a ...interface{}) {
	dim.Fprintf(os.Stdout, format+"\n", a...)
}

// Bold prints bold text.
func Bold(format string, a ...interface{}) string {
	return bold.Sprintf(format, a...)
}

// Dim prints dim/muted text.
func Dim(format string, a ...interface{}) string {
	return dim.Sprintf(format, a...)
}

// Cyan returns cyan bold text (used for prompt keys).
func Cyan(format string, a ...interface{}) string {
	return cyanBold.Sprintf(format, a...)
}

// Magenta returns magenta text (used for config source labels).
func Magenta(format string, a ...interface{}) string {
	return magenta.Sprintf(format, a...)
}

// White returns white text.
func White(format string, a ...interface{}) string {
	return white.Sprintf(format, a...)
}

// Prompt prints an interactive prompt with colored keys.
// Example: [y] commit  [e] edit  [r] regenerate  [n] cancel
func Prompt(options []PromptOption) {
	for i, opt := range options {
		if i > 0 {
			fmt.Print("   ")
		}
		fmt.Printf("%s %s", Cyan("[%s]", opt.Key), White("%s", opt.Label))
	}
	fmt.Println()
}

// PromptOption represents a single prompt choice.
type PromptOption struct {
	Key   string
	Label string
}

// PrintMessage prints a commit message preview with bold formatting.
func PrintMessage(label, message string) {
	fmt.Println()
	dim.Println(label)
	fmt.Println()
	bold.Println(message)
	fmt.Println()
}

// SelectedFile prints a file in the interactive picker (selected state).
func SelectedFile(path, status string) string {
	return cyanBold.Sprintf("> [x] %-40s %s", path, status)
}

// UnselectedFile prints a file in the interactive picker (unselected state).
func UnselectedFile(path, status string) string {
	return fmt.Sprintf("  [ ] %-40s %s", path, status)
}

// SpinnerText returns dim text suitable for spinner/loading messages.
func SpinnerText(format string, a ...interface{}) string {
	return dim.Sprintf(format, a...)
}
