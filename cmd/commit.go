package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Chifez/gitai/internal/config"
	"github.com/Chifez/gitai/pkg/editor"
	"github.com/Chifez/gitai/pkg/git"
	"github.com/Chifez/gitai/pkg/provider"
	"github.com/Chifez/gitai/pkg/ui"
	"github.com/spf13/cobra"
)

var commitCmd = &cobra.Command{
	Use:   "commit [files...]",
	Short: "Generate an AI commit message and commit",
	Long:  "Stage files, generate a commit message using an LLM, review it interactively, then commit and optionally push.",
	Args:  cobra.ArbitraryArgs,
	RunE:  runCommit,
}

func init() {
	rootCmd.AddCommand(commitCmd)

	// Staging flags
	commitCmd.Flags().BoolP("pick", "p", false, "Launch interactive file picker to select files to stage")
	commitCmd.Flags().BoolP("all", "a", false, "Stage all modified tracked files (git add -u)")
	commitCmd.Flags().Bool("include-untracked", false, "Used with --all: also stages new untracked files (git add .)")

	// LLM flags
	commitCmd.Flags().String("model", "", "LLM model to use for this commit only")
	commitCmd.Flags().String("style", "", "Commit style: conventional, simple, or emoji")
	commitCmd.Flags().String("context", "", "Natural language hint to guide message generation")
	commitCmd.Flags().Int("max-length", 0, "Maximum characters in the commit subject line")
	commitCmd.Flags().String("lang", "", "Language for the generated commit message")
	commitCmd.Flags().String("provider", "", "LLM provider to use for this commit only")

	// Push flags
	commitCmd.Flags().Bool("no-push", false, "Commit only, skip push regardless of config")
	commitCmd.Flags().Bool("push", false, "Force push even if auto_push is false in config")
	commitCmd.Flags().Bool("dry-run", false, "Generate and display message only, no commit or push")
	commitCmd.Flags().BoolP("yes", "y", false, "Skip interactive review, commit immediately")

	// Remote/branch flags
	commitCmd.Flags().String("remote", "", "Add a new remote and use it for this push")
	commitCmd.Flags().String("remote-name", "origin", "Name for the remote when using --remote")
	commitCmd.Flags().String("branch", "", "Remote branch name to push to")
	commitCmd.Flags().Bool("force-push", false, "Push with --force-with-lease (safe force push)")
}

func runCommit(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// --- Load config ---
	flagOverrides := buildFlagOverrides(cmd)
	cfg, err := config.EnsureConfig(flagOverrides)
	if err != nil {
		return err
	}

	// --- Staging ---
	pickMode, _ := cmd.Flags().GetBool("pick")
	allMode, _ := cmd.Flags().GetBool("all")
	includeUntracked, _ := cmd.Flags().GetBool("include-untracked")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if !dryRun {
		if err := handleStaging(ctx, cmd, args, pickMode, allMode, includeUntracked); err != nil {
			return err
		}
	}

	// --- Get diff ---
	diff, err := git.GetStagedDiff(ctx)
	if err != nil {
		return fmt.Errorf("failed to get staged diff: %w", err)
	}
	if diff == "" {
		ui.Warn("No staged changes found. Nothing to commit.")
		return nil
	}

	var truncated bool
	diff, truncated = git.SanitizeDiff(diff)
	if truncated {
		ui.Warn("Large diff detected, showing first %d tokens.", git.MaxTokens)
	}

	// --- Build provider ---
	p, err := cfg.BuildProvider()
	if err != nil {
		ui.Error("%v", err)
		return err
	}

	// --- Generate message ---
	opts := provider.Options{
		Style:       cfg.Style,
		MaxLength:   cfg.MaxLength,
		Lang:        cfg.Lang,
		Context:     getStringFlag(cmd, "context"),
		IncludeBody: cfg.IncludeBody,
	}

	message, err := streamMessage(ctx, p, diff, opts, "Generating commit message...")

	if err != nil {
		var apiErr *provider.APIError
		if errors.As(err, &apiErr) {
			ui.Error("%s", apiErr.UserMessage())
		} else {
			ui.Error("Failed to generate message: %v", err)
		}
		return err
	}

	// --- Dry run: just display ---
	if dryRun {
		ui.PrintMessage("Generated commit message (dry run):", message)
		return nil
	}

	// --- Review loop ---
	skipReview, _ := cmd.Flags().GetBool("yes")
	regenCount := 0
	const maxRegens = 10
	if !skipReview {
		for {
			var action editor.Action
			message, action, err = editor.ReviewMessage(ctx, message)
			if err != nil {
				return err
			}

			switch action {
			case editor.ActionCommit:
				// proceed to commit
			case editor.ActionRegenerate:
				regenCount++
				if regenCount > maxRegens {
					ui.Warn("Regeneration limit reached (%d). Please edit manually.", maxRegens)
					continue
				}

				message, err = streamMessage(ctx, p, diff, opts, "Regenerating commit message...")

				if err != nil {
					var apiErr *provider.APIError
					if errors.As(err, &apiErr) {
						ui.Error("%s", apiErr.UserMessage())
					} else {
						ui.Error("Failed to regenerate: %v", err)
					}
					return err
				}
				continue
			case editor.ActionCancel:
				ui.Info("Cancelled. Staged changes preserved.")
				return nil
			}
			break
		}
	}

	// --- Commit ---
	if err := git.Commit(ctx, message); err != nil {
		ui.Error("Commit failed: %v", err)
		return err
	}
	ui.Success("Committed: %s", firstLine(message))

	// --- Push ---
	if err := handlePush(ctx, cmd, cfg); err != nil {
		// Commit succeeded but push failed — don't return error, just warn
		ui.Warn("Commit succeeded. %v", err)
		return nil
	}

	return nil
}

// handleStaging handles all staging scenarios (A through F from the PRD).
func handleStaging(ctx context.Context, cmd *cobra.Command, args []string, pickMode, allMode, includeUntracked bool) error {

	switch {
	case allMode:
		// Scenario D: stage all
		if err := git.StageAll(ctx, includeUntracked); err != nil {
			return fmt.Errorf("failed to stage files: %w", err)
		}
		// Check if anything was actually staged
		diff, _ := git.GetStagedDiff(ctx)
		if diff == "" {
			ui.Warn("No modified tracked files found. Nothing to commit.")
			return fmt.Errorf("nothing to commit")
		}
		if !includeUntracked {
			// Check for untracked files and warn
			files, _ := git.GetChangedFiles(ctx)
			for _, f := range files {
				if f.Status == "untracked" {
					ui.Warn("Untracked files not included. Use --include-untracked to add them.")
					break
				}
			}
		}
		ui.Info("Staged all modified tracked files.")

	case pickMode:
		// Scenario C: interactive file picker
		if err := handleFilePicker(ctx); err != nil {
			return err
		}

	case len(args) > 0:
		// Scenario B (or E if already staged): stage passed files
		existingDiff, _ := git.GetStagedDiff(ctx)
		alreadyStaged := existingDiff != ""
		validPaths := validateFilePaths(args)
		if len(validPaths) == 0 {
			ui.Error("No valid changed files found. Nothing to commit.")
			return fmt.Errorf("no valid files")
		}
		if err := git.StageFiles(ctx, validPaths); err != nil {
			return fmt.Errorf("failed to stage files: %w", err)
		}
		if alreadyStaged {
			ui.Info("Using already-staged changes + %d newly staged file(s).", len(validPaths))
		} else {
			ui.Info("Staged %d file(s).", len(validPaths))
		}

	default:
		// Check if anything is already staged (Scenario A)
		diff, err := git.GetStagedDiff(ctx)
		if err != nil {
			return err
		}
		if diff != "" {
			// Scenario A: already staged
			return nil
		}

		// Scenario F: nothing staged, nothing passed — recovery flow
		return handleRecoveryFlow(ctx)
	}

	return nil
}

// handleFilePicker runs the interactive file picker.
func handleFilePicker(ctx context.Context) error {
	files, err := git.GetChangedFiles(ctx)

	if err != nil {
		return fmt.Errorf("failed to get changed files: %w", err)
	}
	selected := make(map[int]bool)
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("\033[H\033[2J")
		ui.Info("Pick files to stage (type number to toggle, press Enter to confirm):")
		fmt.Println()

		for i, f := range files {
			checkbox := "[ ]"

			if selected[i] {
				checkbox = ui.Green("[x]")
			}
			fmt.Printf(" %s %2d: %-10s %s\n", checkbox, i+1, ui.Cyan(f.Status), f.Path)
		}

		fmt.Println()
		fmt.Printf(" %s confirm and continue\n", ui.Cyan("[Enter]"))
		fmt.Printf(" %s quit\n", ui.Cyan("[q]  "))
		fmt.Println()

		fmt.Print(ui.Bold("Selection: "))

		rawInput, _ := reader.ReadString('\n')
		input := strings.TrimSpace(rawInput)

		if input == "" {
			break
		}

		if input == "q" {
			return fmt.Errorf("cancelled by user")
		}

		idx, err := strconv.Atoi(input)
		if err == nil && idx > 0 && idx <= len(files) {
			selected[idx-1] = !selected[idx-1]
		} else {
			ui.Warn("invalid selection: %s", input)
			time.Sleep(500 * time.Millisecond)
		}
	}

	var paths []string

	for i, f := range files {
		if selected[i] {
			paths = append(paths, f.Path)
		}
	}

	if len(paths) == 0 {
		return fmt.Errorf("no files selected")
	}
	return git.StageFiles(ctx, paths)
}

// handleRecoveryFlow handles Scenario F — nothing staged, nothing passed.
func handleRecoveryFlow(ctx context.Context) error {
	fmt.Println()
	ui.Warn("Nothing staged. What would you like to do?")
	fmt.Println()
	fmt.Printf("  %s stage all changes\n", ui.Cyan("[a]"))
	fmt.Printf("  %s pick files interactively\n", ui.Cyan("[p]"))
	fmt.Printf("  %s quit\n", ui.Cyan("[q]"))
	fmt.Println()

	reader := bufio.NewReader(os.Stdin)
	rawInput, _ := reader.ReadString('\n')
	input := strings.TrimSpace(rawInput)

	switch input {
	case "a":
		if err := git.StageAll(ctx, false); err != nil {
			return fmt.Errorf("failed to stage files: %w", err)
		}
		ui.Info("Staged all modified tracked files.")
		return nil
	case "p":
		return handleFilePicker(ctx)
	default:
		ui.Info("Exiting.")
		return fmt.Errorf("cancelled")
	}
}

// handlePush handles the push logic after a successful commit.
func handlePush(ctx context.Context, cmd *cobra.Command, cfg *config.Config) error {
	noPush, _ := cmd.Flags().GetBool("no-push")
	forcePush, _ := cmd.Flags().GetBool("push")
	forcePushLease, _ := cmd.Flags().GetBool("force-push")
	remoteURL := getStringFlag(cmd, "remote")
	remoteName := getStringFlag(cmd, "remote-name")
	branch := getStringFlag(cmd, "branch")

	// Determine if we should push
	shouldPush := cfg.AutoPush || forcePush
	if noPush {
		shouldPush = false
	}

	if !shouldPush {
		return nil
	}

	// Get current branch if not specified
	if branch == "" {
		var err error
		branch, err = git.GetCurrentBranch(ctx)
		if err != nil {
			return fmt.Errorf("Push failed: could not determine current branch: %w", err)
		}
	}

	// Situation 1: Brand new project — no remote
	if remoteURL != "" {
		if !isValidRemoteURL(remoteURL) {
			return fmt.Errorf("Invalid remote URL format: %s. Use https://, git@, or ssh:// URLs.", remoteURL)
		}
		if remoteName == "" {
			remoteName = cfg.DefaultRemoteName
		}
		if err := git.AddRemote(ctx, remoteName, remoteURL); err != nil {
			return fmt.Errorf("Push failed: could not add remote: %w", err)
		}
		ui.Success("Remote \"%s\" added.", remoteName)
	}

	// Check if remote exists
	hasRemote, err := git.HasRemote(ctx)
	if err != nil {
		return fmt.Errorf("Push failed: %w", err)
	}
	if !hasRemote {
		ui.Info("Commit saved locally. To push, run: gitai commit --remote <url>")
		return nil
	}

	// Determine push options
	pushOpts := git.PushOptions{
		RemoteName:     remoteName,
		Branch:         branch,
		ForceWithLease: forcePushLease,
	}

	// Situation 2: New branch — no upstream
	hasUpstream, _ := git.HasUpstream(ctx)
	if !hasUpstream && cfg.AutoSetUpstream {
		pushOpts.SetUpstream = true
	}

	if err := git.Push(ctx, pushOpts); err != nil {
		errMsg := err.Error()
		switch {
		case strings.Contains(errMsg, "Authentication") || strings.Contains(errMsg, "fatal: could not read Password"):
			return fmt.Errorf("Push failed: authentication error. Check your git credentials or SSH keys.")
		case strings.Contains(errMsg, "rejected") || strings.Contains(errMsg, "non-fast-forward"):
			return fmt.Errorf("Push failed: branch is behind remote. Run `git pull --rebase` then try again.")
		case strings.Contains(errMsg, "Could not resolve host"):
			return fmt.Errorf("Push failed: could not reach remote. Commit is saved locally.")
		default:
			return fmt.Errorf("Push failed: %w", err)
		}
	}

	if pushOpts.SetUpstream {
		ui.Success("Pushed to %s/%s. Upstream set.", remoteName, branch)
	} else {
		ui.Success("Pushed to %s/%s", remoteName, branch)
	}

	return nil
}

// --- Helpers ---

func buildFlagOverrides(cmd *cobra.Command) map[string]string {
	overrides := make(map[string]string)

	if v := getStringFlag(cmd, "model"); v != "" {
		overrides["model"] = v
	}
	if v := getStringFlag(cmd, "style"); v != "" {
		overrides["style"] = v
	}
	if v := getStringFlag(cmd, "lang"); v != "" {
		overrides["lang"] = v
	}
	if v := getStringFlag(cmd, "provider"); v != "" {
		overrides["provider"] = v
	}
	if v, _ := cmd.Flags().GetInt("max-length"); v > 0 {
		overrides["max_length"] = strconv.Itoa(v)
	}

	return overrides
}

func getStringFlag(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}

func isValidRemoteURL(url string) bool {
	return strings.HasPrefix(url, "https://") ||
		strings.HasPrefix(url, "http://") ||
		strings.HasPrefix(url, "git@") ||
		strings.HasPrefix(url, "ssh://")
}

func validateFilePaths(paths []string) []string {
	var valid []string
	for _, p := range paths {
		if _, err := os.Stat(p); os.IsNotExist(err) {
			ui.Warn("%s not found, skipping", p)
			continue
		}
		valid = append(valid, p)
	}
	return valid
}

func firstLine(s string) string {
	for i, c := range s {
		if c == '\n' {
			return s[:i]
		}
	}
	return s
}

func streamMessage(ctx context.Context, p provider.Provider, diff string, opts provider.Options, spinnerMsg string) (string, error) {
	s := ui.StartSpinnerWithContext(ctx, spinnerMsg)
	stream := p.GenerateMessageStream(ctx, diff, opts)

	var sb strings.Builder
	firstChunk := true

	for events := range stream {
		if events.Err != nil {
			s.Stop()
			return "", events.Err
		}

		if firstChunk {
			s.Stop()
			fmt.Println()

			firstChunk = false
		}
		fmt.Print(events.Text)
		sb.WriteString(events.Text)
	}

	if !firstChunk {
		fmt.Println()
	}

	return sb.String(), nil
}
