// Package git provides pure git operations using os/exec.
// No LLM logic, no UI — just git commands.
package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// FileStatus represents the status of a changed file.
type FileStatus struct {
	Path   string
	Status string
}

// PushOptions configures the git push command.
type PushOptions struct {
	RemoteName     string
	Branch         string
	SetUpstream    bool
	ForceWithLease bool
}

// GetStagedDiff returns the diff of staged changes.
func GetStagedDiff(ctx context.Context) (string, error) {
	return runGit(ctx, "diff", "--staged")
}

// GetChangedFiles returns all modified, added, deleted, and untracked files.
func GetChangedFiles(ctx context.Context) ([]FileStatus, error) {
	out, err := runGit(ctx, "status", "--porcelain")
	if err != nil {
		return nil, err
	}

	var files []FileStatus
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if len(line) < 4 {
			continue
		}
		code := strings.TrimSpace(line[:2])
		path := strings.TrimSpace(line[3:])

		var status string
		switch {
		case code == "??":
			status = "untracked"
		case strings.Contains(code, "A"):
			status = "added"
		case strings.Contains(code, "D"):
			status = "deleted"
		default:
			status = "modified"
		}

		files = append(files, FileStatus{Path: path, Status: status})
	}

	return files, nil
}

// StageFiles stages the given file paths.
func StageFiles(ctx context.Context, paths []string) error {
	args := append([]string{"add"}, paths...)
	_, err := runGit(ctx, args...)
	return err
}

// StageAll stages all changes. If includeUntracked is true, stages everything
// including new files (git add .). Otherwise only tracked files (git add -u).
func StageAll(ctx context.Context, includeUntracked bool) error {
	if includeUntracked {
		_, err := runGit(ctx, "add", ".")
		return err
	}
	_, err := runGit(ctx, "add", "-u")
	return err
}

// Commit creates a commit with the given message.
func Commit(ctx context.Context, message string) error {
	_, err := runGit(ctx, "commit", "-m", message)
	return err
}

// Push pushes commits to the remote.
func Push(ctx context.Context, opts PushOptions) error {
	args := []string{"push"}

	if opts.ForceWithLease {
		args = append(args, "--force-with-lease")
	}

	if opts.SetUpstream {
		args = append(args, "-u")
	}

	if opts.RemoteName != "" {
		args = append(args, opts.RemoteName)
	}

	if opts.Branch != "" {
		args = append(args, opts.Branch)
	}

	_, err := runGit(ctx, args...)
	return err
}

// GetCurrentBranch returns the name of the current branch.
func GetCurrentBranch(ctx context.Context) (string, error) {
	out, err := runGit(ctx, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// HasUpstream checks if the current branch has a tracked upstream.
func HasUpstream(ctx context.Context) (bool, error) {
	_, err := runGit(ctx, "rev-parse", "--abbrev-ref", "@{upstream}")
	if err != nil {
		return false, nil
	}
	return true, nil
}

// HasRemote checks if any remote is configured.
func HasRemote(ctx context.Context) (bool, error) {
	out, err := runGit(ctx, "remote", "-v")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

// AddRemote adds a new git remote.
func AddRemote(ctx context.Context, name, url string) error {
	_, err := runGit(ctx, "remote", "add", name, url)
	return err
}

// runGit executes a git command and returns its stdout.
// Commands use exec.CommandContext with separate args — never shell strings.
func runGit(ctx context.Context, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s failed: %w\nstderr: %s", args[0], err, stderr.String())
	}

	return stdout.String(), nil
}
