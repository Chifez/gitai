package cmd

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommitCmdFlags(t *testing.T) {
	cmd := commitCmd
	err := cmd.Flags().Set("dry-run", "true")
	if err != nil {
		t.Fatal(err)
	}

	if cmd.Use != "commit [files...]" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}
	
	if configCmd.Use != "config" {
		t.Errorf("unexpected Use: %s", configCmd.Use)
	}
}

func TestCommitFlow_DryRun(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(origDir)

	ctx := context.Background()

	exec.CommandContext(ctx, "git", "init").Run()
	exec.CommandContext(ctx, "git", "config", "user.name", "Test").Run()
	exec.CommandContext(ctx, "git", "config", "user.email", "test@example.com").Run()

	os.WriteFile("test.txt", []byte("hello"), 0644)
	exec.CommandContext(ctx, "git", "add", "test.txt").Run()

	home := filepath.Join(dir, "home")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)

	cmd := commitCmd
	t.Setenv("GITAI_PROVIDER", "mock")
	
	cmd.SetArgs([]string{"--dry-run"})
	if err := cmd.ExecuteContext(ctx); err != nil {
		t.Fatalf("commit --dry-run failed: %v", err)
	}

	out, err := exec.CommandContext(ctx, "git", "log", "-1", "--oneline").CombinedOutput()
	if err == nil || string(out) != "" {
		if string(out) != "" && !strings.Contains(string(out), "does not have any commits") {
			t.Errorf("commit --dry-run actually created a commit: %s", string(out))
		}
	}
}
