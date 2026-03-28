// Package editor provides the interactive commit message review UI.
package editor

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/Chifez/gitai/pkg/ui"
)

// Action represents the user's choice during review.
type Action int

const (
	ActionCommit     Action = iota
	ActionEdit
	ActionRegenerate
	ActionCancel
)

// ReviewMessage displays the generated commit message and prompts the user
// to accept, edit, regenerate, or cancel. Reads input from os.Stdin.
func ReviewMessage(ctx context.Context, message string) (string, Action, error) {
	return ReviewMessageWithReader(ctx, message, os.Stdin)
}

// ReviewMessageWithReader displays the generated commit message and prompts
// the user using the provided reader for input.
func ReviewMessageWithReader(ctx context.Context, message string, input io.Reader) (string, Action, error) {
	ui.PrintMessage("Generated commit message:", message)
	reader := bufio.NewReader(input)

	for {
		ui.Prompt([]ui.PromptOption{
			{Key: "y", Label: "commit"},
			{Key: "e", Label: "edit"},
			{Key: "r", Label: "regenerate"},
			{Key: "n", Label: "cancel"},
		})

		rawInput, err := reader.ReadString('\n')
		if err != nil {
			return "", ActionCancel, fmt.Errorf("failed to read input: %w", err)
		}

		switch strings.TrimSpace(strings.ToLower(rawInput)) {
		case "y":
			return message, ActionCommit, nil

		case "e":
			edited, err := editInEditor(message)
			if err != nil {
				ui.Warn("Editor error: %v", err)
				continue
			}
			if strings.TrimSpace(edited) == "" {
				ui.Warn("Empty message not allowed. Try again.")
				continue
			}
			return edited, ActionCommit, nil

		case "r":
			return "", ActionRegenerate, nil

		case "n":
			return "", ActionCancel, nil

		default:
			ui.Warn("Press y, e, r, or n.")
		}
	}
}

// editInEditor opens the message in $EDITOR or falls back to inline editing.
func editInEditor(message string) (string, error) {
	editorCmd := os.Getenv("EDITOR")
	if editorCmd == "" {
		editorCmd = os.Getenv("VISUAL")
	}

	if editorCmd == "" {
		return editInline(message, os.Stdin)
	}

	tmpFile, err := os.CreateTemp("", "gitai-commit-*.txt")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(message); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("failed to write temp file: %w", err)
	}
	tmpFile.Close()

	cmd := exec.Command(editorCmd, tmpFile.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return message, nil
	}

	edited, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return "", fmt.Errorf("failed to read edited message: %w", err)
	}

	return string(edited), nil
}

// editInline provides a simple fallback editor when $EDITOR is not set.
func editInline(message string, input io.Reader) (string, error) {
	fmt.Println()
	ui.Info("No $EDITOR set. Edit the message below (enter an empty line to finish):")
	fmt.Println()
	fmt.Println(message)
	fmt.Println()
	ui.Info("Enter new message (empty line to finish):")

	scanner := bufio.NewScanner(input)
	var lines []string
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" && len(lines) > 0 {
			break
		}
		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	return strings.Join(lines, "\n"), nil
}
