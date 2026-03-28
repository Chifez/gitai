package git

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestGitOperations(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	ctx := context.Background()

	if _, err := runGit(ctx, "init"); err != nil {
		t.Fatalf("git init failed: %v", err)
	}

	if err := os.WriteFile("test.txt", []byte("hello"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	files, err := GetChangedFiles(ctx)
	if err != nil {
		t.Fatalf("GetChangedFiles failed: %v", err)
	}
	if len(files) != 1 || files[0].Status != "untracked" {
		t.Errorf("expected 1 untracked file, got %v", files)
	}

	if err := StageFiles(ctx, []string{"test.txt"}); err != nil {
		t.Fatalf("StageFiles failed: %v", err)
	}

	diff, err := GetStagedDiff(ctx)
	if err != nil {
		t.Fatalf("GetStagedDiff failed: %v", err)
	}
	if diff == "" {
		t.Error("expected a diff string, got empty")
	}

	runGit(ctx, "config", "user.name", "Test User")
	runGit(ctx, "config", "user.email", "test@example.com")
	
	if err := Commit(ctx, "feat: test message"); err != nil {
		t.Fatalf("Commit failed: %v", err)
	}

	branch, err := GetCurrentBranch(ctx)
	if err != nil {
		t.Fatalf("GetCurrentBranch failed: %v", err)
	}
	if branch != "master" && branch != "main" {
		t.Errorf("unexpected branch: %s", branch)
	}

	hasUpstream, err := HasUpstream(ctx)
	if err != nil {
		t.Fatalf("HasUpstream errored: %v", err)
	}
	if hasUpstream {
		t.Error("expected false for hasUpstream on fresh branch")
	}

	hasRemote, err := HasRemote(ctx)
	if err != nil {
		t.Fatalf("HasRemote errored: %v", err)
	}
	if hasRemote {
		t.Error("expected false for HasRemote on fresh repo")
	}

	if err := AddRemote(ctx, "origin", "https://github.com/test/repo.git"); err != nil {
		t.Fatalf("AddRemote failed: %v", err)
	}

	hasRemote, _ = HasRemote(ctx)
	if !hasRemote {
		t.Error("expected true for HasRemote after AddRemote")
	}
}

func TestStageAll_TrackedOnly(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	ctx := context.Background()
	runGit(ctx, "init")
	runGit(ctx, "config", "user.name", "Test")
	runGit(ctx, "config", "user.email", "test@test.com")

	// Create and commit a file
	if err := os.WriteFile("tracked.txt", []byte("v1"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	runGit(ctx, "add", "tracked.txt")
	runGit(ctx, "commit", "-m", "initial")

	// Modify tracked file and create an untracked file
	if err := os.WriteFile("tracked.txt", []byte("v2"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := os.WriteFile("untracked.txt", []byte("new"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	// Stage tracked only
	if err := StageAll(ctx, false); err != nil {
		t.Fatalf("StageAll(false) failed: %v", err)
	}

	diff, _ := GetStagedDiff(ctx)
	if diff == "" {
		t.Error("expected staged diff for tracked file")
	}
}

func TestStageAll_IncludingUntracked(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	ctx := context.Background()
	runGit(ctx, "init")
	runGit(ctx, "config", "user.name", "Test")
	runGit(ctx, "config", "user.email", "test@test.com")

	if err := os.WriteFile("file.txt", []byte("hello"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	if err := StageAll(ctx, true); err != nil {
		t.Fatalf("StageAll(true) failed: %v", err)
	}

	diff, _ := GetStagedDiff(ctx)
	if diff == "" {
		t.Error("expected staged diff including untracked files")
	}
}

func TestGetChangedFiles_Clean(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	ctx := context.Background()
	runGit(ctx, "init")
	runGit(ctx, "config", "user.name", "Test")
	runGit(ctx, "config", "user.email", "test@test.com")

	// Create and commit a file
	if err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("content"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	runGit(ctx, "add", ".")
	runGit(ctx, "commit", "-m", "init")

	files, err := GetChangedFiles(ctx)
	if err != nil {
		t.Fatalf("GetChangedFiles failed: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files on clean tree, got %d", len(files))
	}
}

func TestGetChangedFiles_AllStatuses(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	ctx := context.Background()
	runGit(ctx, "init")
	runGit(ctx, "config", "user.name", "Test")
	runGit(ctx, "config", "user.email", "test@test.com")

	// Create files and commit
	if err := os.WriteFile("modify.txt", []byte("v1"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	if err := os.WriteFile("delete.txt", []byte("v1"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	runGit(ctx, "add", ".")
	runGit(ctx, "commit", "-m", "init")

	// Create different status scenarios
	if err := os.WriteFile("modify.txt", []byte("v2"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	// Stage the deletion so git shows "D " (index deletion)
	runGit(ctx, "rm", "delete.txt")
	if err := os.WriteFile("new.txt", []byte("new"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	files, err := GetChangedFiles(ctx)
	if err != nil {
		t.Fatalf("GetChangedFiles failed: %v", err)
	}

	statuses := make(map[string]string)
	for _, f := range files {
		statuses[f.Path] = f.Status
	}

	if statuses["modify.txt"] != "modified" {
		t.Errorf("expected modified, got %s", statuses["modify.txt"])
	}
	if statuses["delete.txt"] != "deleted" {
		t.Errorf("expected deleted, got %s", statuses["delete.txt"])
	}
	if statuses["new.txt"] != "untracked" {
		t.Errorf("expected untracked, got %s", statuses["new.txt"])
	}
}

func TestGetStagedDiff_Empty(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	ctx := context.Background()
	runGit(ctx, "init")

	diff, err := GetStagedDiff(ctx)
	if err != nil {
		t.Fatalf("GetStagedDiff failed: %v", err)
	}
	if diff != "" {
		t.Errorf("expected empty diff, got '%s'", diff)
	}
}

func TestPush_NoRemote(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	ctx := context.Background()
	runGit(ctx, "init")
	runGit(ctx, "config", "user.name", "Test")
	runGit(ctx, "config", "user.email", "test@test.com")

	if err := os.WriteFile("test.txt", []byte("hello"), 0644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	runGit(ctx, "add", ".")
	runGit(ctx, "commit", "-m", "init")

	err := Push(ctx, PushOptions{})
	if err == nil {
		t.Fatal("expected error when pushing with no remote")
	}
}

func TestSanitizeDiff_Binary(t *testing.T) {
	diff := "--- a/file.go\n+++ b/file.go\n+ hello\nBinary files a/image.png and b/image.png differ\n--- a/other.go"
	result, truncated := SanitizeDiff(diff)
	if truncated {
		t.Error("expected no truncation")
	}
	if result == diff {
		t.Error("expected binary line to be replaced")
	}
	expected := "--- a/file.go\n+++ b/file.go\n+ hello\nBinary file modified: image.png\n--- a/other.go"
	if result != expected {
		t.Errorf("unexpected result:\n%s", result)
	}
}

func TestSanitizeDiff_Truncation(t *testing.T) {
	// Create a diff larger than maxChars
	longDiff := ""
	for len(longDiff) < maxChars+100 {
		longDiff += "diff --git a/file.go b/file.go\n+ some change\n"
	}
	result, truncated := SanitizeDiff(longDiff)
	if !truncated {
		t.Error("expected truncation")
	}
	if len(result) != maxChars {
		t.Errorf("expected length %d, got %d", maxChars, len(result))
	}
}

func TestSanitizeDiff_Clean(t *testing.T) {
	diff := "--- a/file.go\n+++ b/file.go\n+ hello world"
	result, truncated := SanitizeDiff(diff)
	if truncated {
		t.Error("expected no truncation")
	}
	if result != diff {
		t.Error("expected unchanged diff")
	}
}
