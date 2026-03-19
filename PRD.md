# gitai — AI-Powered Git Commit CLI
## Product Requirements Document | v1.0

| Field | Value |
|---|---|
| Status | Draft |
| Version | 1.0 |
| Author | Nwosu Ifeanyi Emmanuel |
| Last Updated | 2026 |
| Target Platform | macOS, Linux, Windows |
| Primary Language | Go (Golang) |
| Distribution | Binary + Importable Go Library |

---

## Table of Contents

1. [Product Overview](#1-product-overview)
2. [Distribution Model](#2-distribution-model)
3. [Project Structure](#3-project-structure)
4. [Provider Interface and Extensibility](#4-provider-interface-and-extensibility)
5. [Configuration System](#5-configuration-system)
6. [Full User Flows](#6-full-user-flows)
7. [Full CLI Reference](#7-full-cli-reference)
8. [Commit Message Styles](#8-commit-message-styles)
9. [Go Architecture Patterns](#9-go-architecture-patterns)
10. [Remote and Branch Management](#10-remote-and-branch-management)
11. [CI/CD Pipeline](#11-cicd-pipeline)
12. [Testing Strategy](#12-testing-strategy)
13. [Security and Privacy](#13-security-and-privacy)
14. [Future Work (Post v1)](#14-future-work-post-v1)
15. [Success Metrics](#15-success-metrics)

---

## 1. Product Overview

### 1.1 Problem Statement

Developers write poor commit messages. Phrases like "fix stuff", "wip", "asdfgh", and "update" are endemic across every codebase. This happens not because developers are lazy, but because writing a meaningful commit message at the end of a long coding session is cognitively expensive — you have to mentally reconstruct what you changed and why, then articulate it clearly, while you are already thinking about the next task.

Existing AI commit tools (aicommits, commitgpt, cz-git) solve part of this problem, but all share significant gaps:

- They are npm packages, requiring Node.js even in non-JavaScript projects
- Most do not support inline editing — they print and copy to clipboard, nothing more
- None combine commit + push in a single intelligent flow
- None offer interactive file selection as part of the commit command
- None are extensible as a Go library for other tools to build on

### 1.2 Solution

gitai is a single-binary CLI tool written in Go that reads staged git changes, generates a well-structured commit message using an LLM, presents it for inline review and editing, then commits and optionally pushes — all in one command. It ships as both a compiled binary (for end users) and an importable Go library (for developers building on top of it).

### 1.3 Goals

- Eliminate the friction of writing commit messages without removing developer control
- Work in any project regardless of language or tech stack — zero runtime dependencies
- Install in under 60 seconds on any supported platform
- Be extensible: new LLM providers should require adding one file, not refactoring the core
- Teach idiomatic Go patterns through its own codebase structure

### 1.4 Non-Goals

- gitai will not manage branches, PRs, or anything beyond commit + push in v1
- gitai will not support commit signing in v1
- gitai will not provide a GUI or web interface
- gitai will not store or log any diff content — privacy is a hard constraint

---

## 2. Distribution Model

### 2.1 Binary (End Users)

A compiled executable for each target platform. No runtime, no package manager, no dependencies. Install once, works everywhere.

| Method | Command | Notes |
|---|---|---|
| Homebrew | `brew install gitai` | Recommended for Mac/Linux |
| go install | `go install github.com/you/gitai@latest` | Requires Go on machine |
| curl script | `curl -sSL https://gitai.sh/install \| sh` | Any platform, no Go needed |
| GitHub Releases | Download binary from releases page | Manual fallback |

### 2.2 Library (Developers)

The same codebase, structured so other Go tools can import core packages directly. Any package under `pkg/` is public and importable. The `cmd/` layer is a thin CLI wrapper that calls `pkg/`.

```go
import (
    "github.com/you/gitai/pkg/provider"
    "github.com/you/gitai/pkg/git"
    "github.com/you/gitai/pkg/prompt"
)
```

### 2.3 Release Pipeline

Releases are automated via GoReleaser triggered by a GitHub Actions workflow on version tags.

- GoReleaser compiles binaries for `darwin/amd64`, `darwin/arm64`, `linux/amd64`, `linux/arm64`, `windows/amd64`
- Binaries are attached to GitHub Releases automatically
- Homebrew tap is updated automatically via GoReleaser homebrew-tap config
- Every release includes a SHA256 checksum file for verification

---

## 3. Project Structure

### 3.1 Folder Layout

```
gitai/
├── main.go                         # Entry point. Initialises cobra root command.
├── go.mod
├── go.sum
├── .goreleaser.yaml                # GoReleaser release config
├── .github/
│   └── workflows/
│       ├── ci.yaml                 # Lint, test, build on every push
│       └── release.yaml            # GoReleaser on version tag push
├── cmd/
│   ├── root.go                     # Root cobra command, global flags
│   ├── commit.go                   # gitai commit subcommand
│   └── config.go                   # gitai config subcommand (get/set/list)
├── pkg/                            # Public — importable by other Go tools
│   ├── git/
│   │   └── git.go                  # All git operations: diff, stage, commit, push
│   ├── prompt/
│   │   └── prompt.go               # Builds LLM prompt from diff + options
│   ├── editor/
│   │   └── editor.go               # Interactive terminal review + edit UI
│   └── provider/
│       ├── provider.go             # Provider interface definition
│       └── openai/
│           └── openai.go           # OpenAI implementation of Provider
└── internal/                       # Private — CLI internals only, not importable
    └── config/
        └── config.go               # Config load, write, wizard, priority resolution
```

### 3.2 Package Responsibilities

| Package | Responsibility |
|---|---|
| `main.go` | Wires cobra commands together. Nothing else. |
| `cmd/commit.go` | Parses all commit flags, resolves config, orchestrates the full commit flow by calling `pkg/*` packages. |
| `cmd/config.go` | Handles `gitai config get/set/list` subcommands. Reads and writes config via `internal/config`. |
| `pkg/git` | Pure git operations. No LLM, no UI. Exposes: `GetStagedDiff()`, `StageFiles()`, `Commit()`, `Push()`, `GetChangedFiles()`. |
| `pkg/prompt` | Builds the LLM prompt string from a diff and a set of options (style, context hint, max length). No HTTP calls. |
| `pkg/editor` | Renders the interactive review UI: print message, accept y/e/r/n keypress, open `$EDITOR`, fallback inline editor. |
| `pkg/provider` | Defines the Provider interface. Any struct implementing it can be dropped in as an LLM backend. |
| `pkg/provider/openai` | OpenAI implementation of Provider. Handles HTTP, auth, retries, token limits. |
| `internal/config` | Config struct, YAML load/write, setup wizard, priority resolution (flag > env > config > default). |

---

## 4. Provider Interface and Extensibility

### 4.1 Interface Definition

The Provider interface is the extensibility contract. Any LLM backend — OpenAI, Anthropic, local Ollama, a mock for testing — must implement exactly two methods:

```go
// pkg/provider/provider.go

package provider

type Options struct {
    Style      string   // "conventional" | "simple" | "emoji"
    MaxLength  int      // max chars in subject line
    Lang       string   // language for commit message
    Context    string   // optional user-supplied hint
}

type Provider interface {
    // GenerateMessage sends the diff to the LLM and returns a commit message.
    // Implementations must respect ctx for cancellation.
    GenerateMessage(ctx context.Context, diff string, opts Options) (string, error)

    // Name returns a human-readable identifier used in error messages and config.
    Name() string
}
```

### 4.2 Adding a New Provider

To add Anthropic (or any other provider) in the future:

1. Create `pkg/provider/anthropic/anthropic.go`
2. Implement the Provider interface (`GenerateMessage` + `Name`)
3. Register it in the provider factory in `internal/config/config.go`
4. Add `"anthropic"` as a valid value for the `--provider` flag

Zero changes to the core commit flow, the editor, the git package, or the prompt builder.

### 4.3 OpenAI Implementation Details

- **Model:** configurable, default `gpt-4o-mini`
- **Auth:** reads from `OPENAI_API_KEY` env var first, then config file
- **Request timeout:** 30 seconds
- **Retry:** one automatic retry after 3 seconds on network error or 5xx response
- **Token budget:** diff is trimmed to fit within model context window minus prompt overhead (~500 tokens reserved for system prompt + response)
- **Streaming:** response is streamed and collected — spinner shown while waiting
- The system prompt instructs the model to return only the commit message with no preamble, no explanation, no markdown code fences

---

## 5. Configuration System

### 5.1 Config File Location and Format

The config file lives at `~/.gitai/config.yaml`. It is created automatically on first run via the setup wizard. The directory is created if it does not exist.

```yaml
# ~/.gitai/config.yaml

provider: openai              # LLM provider to use
api_key: sk-...               # Omitted entirely if OPENAI_API_KEY env var is set
model: gpt-4o-mini            # Model identifier
style: conventional           # conventional | simple | emoji
auto_push: true               # Push after successful commit
max_length: 72                # Max chars in commit subject line
lang: english                 # Language for generated message
include_body: true            # Include a multi-line body in the commit message
default_remote_name: origin   # Name used when adding a new remote via --remote
auto_set_upstream: true       # Always use -u on first push of any branch
```

### 5.2 Priority Resolution

Settings are resolved in the following order. Higher priority always wins.

| # | Source | Example |
|---|---|---|
| 1 | CLI flag | `--model gpt-4o` (overrides everything for this run) |
| 2 | Environment variable | `OPENAI_API_KEY=sk-...` `GITAI_MODEL=gpt-4o` |
| 3 | Config file | `~/.gitai/config.yaml` |
| 4 | Built-in default | `model: gpt-4o-mini`, `style: conventional`, etc. |

### 5.3 Environment Variables

| Variable | Config Key | Notes |
|---|---|---|
| `OPENAI_API_KEY` | `api_key` | Standard OpenAI env var. Always checked first. |
| `GITAI_MODEL` | `model` | Override model without editing config |
| `GITAI_PROVIDER` | `provider` | Override provider |
| `GITAI_STYLE` | `style` | Override commit style |
| `GITAI_AUTO_PUSH` | `auto_push` | `"true"` or `"false"` |
| `GITAI_LANG` | `lang` | Override message language |

### 5.4 Config Commands

```bash
gitai config list               # Print all current config values and their source
gitai config get model          # Print value of a single key
gitai config set model gpt-4o   # Update a single value in config file
gitai config set auto_push false
gitai config reset              # Reset config to defaults (prompts for confirmation)
gitai config path               # Print the full path to the config file
```

### 5.5 First-Run Setup Wizard

Triggered automatically on any command when `~/.gitai/config.yaml` does not exist. The wizard runs before the command proceeds.

```
No config found. Running first-time setup...

Enter your OpenAI API key: sk-...
Default model [gpt-4o-mini]: (press enter to accept)
Commit style (conventional/simple/emoji) [conventional]:
Auto-push after commit? (y/n) [y]:
Include commit body? (y/n) [y]:

Config saved to ~/.gitai/config.yaml
Continuing with your commit...
```

> **Edge:** If the user hits Ctrl+C during setup, no partial config is written. The process exits cleanly. The next run will trigger the wizard again.

---

## 6. Full User Flows

### 6.1 Installation Flow

**Happy Path**

1. User installs binary via preferred method (Homebrew, curl, go install)
2. Binary is placed on PATH automatically
3. User runs: `gitai --version`
4. `gitai v0.1.0` is printed — install confirmed

**Edge Cases**

| Scenario | Behaviour |
|---|---|
| `go install` used, Go not installed | User is directed to golang.org/dl with a clear message |
| curl install, no write permission to `/usr/local/bin` | Script falls back to `~/.local/bin` and adds it to PATH via shell profile |
| Homebrew not installed | Error from Homebrew itself. README documents all three methods clearly. |
| Binary already installed, version conflict | Homebrew/go install handle this. curl script detects existing binary and asks before overwriting. |

---

### 6.2 Commit Flow — All Staging Scenarios

#### Scenario A: Already staged, just commit (most common)

User has run `git add` themselves. gitai reads whatever is in the staging area.

```bash
git add src/auth.go
gitai commit
# gitai reads staged diff and generates message
```

#### Scenario B: Pass files directly as arguments

User has not staged anything. They pass file paths directly to gitai. gitai stages them, then generates the message from those files only.

```bash
gitai commit src/auth.go src/middleware.go
# gitai runs: git add src/auth.go src/middleware.go
# then reads diff of those two files
Staged 2 files. Generating message...
```

| Edge Case | Behaviour |
|---|---|
| File path does not exist | Warning per file: "src/missing.go not found, skipping". Continues with valid files. |
| File exists but has no changes | Warning: "src/clean.go has no changes, skipping". Continues with remaining files. |
| All files invalid or unchanged | Exits: "No valid changed files found. Nothing to commit." |

#### Scenario C: Interactive file picker (`--pick`)

User runs `gitai commit --pick`. gitai lists all modified tracked files and untracked files as an interactive checklist. Space toggles, Enter confirms.

```
gitai commit --pick

Select files to stage (space to toggle, enter to confirm):
 > [x] src/auth.go               modified
   [x] src/middleware.go          modified
   [ ] tests/auth_test.go         modified
   [ ] config/settings.yaml       modified

Staged 2 files. Generating message...
```

| Edge Case | Behaviour |
|---|---|
| User selects nothing and hits Enter | "No files selected. Exiting." Clean exit, nothing staged. |
| User hits Ctrl+C in picker | Clean exit. Nothing staged or committed. |
| Working tree is completely clean | "No changed files found. Nothing to commit." |

#### Scenario D: Stage all modified tracked files (`--all`)

```bash
gitai commit --all
# equivalent to: git add -u && gitai commit
Staged all modified tracked files. Generating message...
```

| Edge Case | Behaviour |
|---|---|
| Untracked files present with `--all` | Warning: "Untracked files not included. Use --include-untracked to add them." |
| `--all` + `--include-untracked` | Runs `git add .` instead of `git add -u`. Stages everything including new files. |
| No modified tracked files | "No modified tracked files found. Nothing to commit." |

#### Scenario E: Mixed — already staged + extra files passed

```bash
git add src/auth.go
gitai commit src/middleware.go
Using 1 already-staged + 1 newly staged file.
Generating message...
```

#### Scenario F: Nothing staged, no files passed (recovery flow)

```
gitai commit

Nothing staged. What would you like to do?
[a] stage all changes
[p] pick files interactively
[q] quit
```

---

### 6.3 Message Generation Flow

**Happy Path**

1. Staged diff is read via `git diff --staged`
2. Diff is trimmed if it exceeds the token budget
3. Prompt is built by `pkg/prompt` with style, context, and max_length options
4. Spinner is shown while the HTTP request is in flight
5. LLM response is streamed and assembled into a complete message string
6. Message is passed to the interactive review prompt

**Edge Cases**

| Scenario | Behaviour |
|---|---|
| API key not set | "OpenAI API key not found. Set OPENAI_API_KEY or run: `gitai config set api_key YOUR_KEY`" |
| API key invalid (401) | "Invalid API key. Check your key at platform.openai.com" |
| Network timeout | Retries once after 3s. On second failure: "Could not reach OpenAI. Check your connection." |
| OpenAI 5xx error | Same retry logic as network timeout. |
| OpenAI rate limit (429) | "Rate limited by OpenAI. Waiting 10 seconds..." then retries once. |
| Diff exceeds token budget | Diff is truncated. Warning: "Large diff detected, showing first 8000 tokens." |
| Binary files in diff | Binary file changes summarised as "Binary file modified: path/to/file" |
| Model does not exist | "Model gpt-xyz not found. Check available models or update: `gitai config set model gpt-4o-mini`" |

---

### 6.4 Interactive Review Flow

```
Generated commit message:

feat(auth): add JWT refresh token rotation

Implements sliding window refresh token strategy to reduce
re-authentication friction while maintaining session security.

[y] commit   [e] edit   [r] regenerate   [n] cancel
```

| Key | Action | Detail |
|---|---|---|
| `y` | Accept and commit | Proceeds directly to `git commit` with the generated message. |
| `e` | Edit in `$EDITOR` | Message written to temp file. `$EDITOR` opened. User saves and closes. Edited message is used. |
| `r` | Regenerate | Calls the LLM again with the same diff. New message shown. Review prompt repeats. No limit on regenerations. |
| `n` | Cancel | Exits cleanly. Staged changes remain staged. Nothing committed or pushed. |

| Edge Case | Behaviour |
|---|---|
| `$EDITOR` not set | Falls back to built-in line-by-line terminal editor. |
| `$EDITOR` exits non-zero (e.g. `:q!` in vim) | Message unchanged. Review prompt shown again. |
| Edited message is empty | "Empty message not allowed. Re-opening editor." |
| Unrecognised key pressed | Prompt re-displayed: "Press y, e, r, or n." |
| Ctrl+C at review prompt | Exits cleanly. Staged changes preserved. |

---

### 6.5 Commit and Push Flow

**Happy Path**

1. `git commit -m "message"` is run via `os/exec`. Output piped to terminal.
2. If `auto_push` is true and `--no-push` was not passed: `git push` is run.
3. Success output is printed.

```
Committed: feat(auth): add JWT refresh token rotation
Pushed to origin/main
```

| Scenario | Behaviour |
|---|---|
| pre-commit hook rejects commit | Hook output printed verbatim. gitai exits non-zero. Push not attempted. |
| Commit succeeds, push fails (no upstream) | "Commit succeeded. Push failed: no upstream branch. Run: `git push --set-upstream origin <branch>`" |
| Push fails (auth error) | "Commit succeeded. Push failed: authentication error. Check your git credentials." |
| Push fails (diverged from remote) | "Commit succeeded. Push failed: branch is behind remote. Run `git pull --rebase` first." |
| `--no-push` flag passed | Commit runs normally. Push skipped. "Committed: \<message\>" shown. |
| `auto_push: false` in config | Same as `--no-push`. Push never runs unless `--push` flag explicitly passed. |

---

## 7. Full CLI Reference

### 7.1 `gitai commit`

| Flag | Type | Default | Description |
|---|---|---|---|
| `[files...]` | positional | — | One or more file paths to stage before committing. |
| `--pick, -p` | bool | false | Launch interactive file picker to select files to stage. |
| `--all, -a` | bool | false | Stage all modified tracked files (`git add -u`). |
| `--include-untracked` | bool | false | Used with `--all`: also stages new untracked files (`git add .`). |
| `--model` | string | config value | LLM model to use for this commit only. |
| `--style` | string | config value | Commit style: `conventional`, `simple`, or `emoji`. |
| `--context` | string | `""` | A natural language hint to guide message generation. |
| `--no-push` | bool | false | Commit only. Skip push regardless of config. |
| `--push` | bool | false | Force push even if `auto_push` is false in config. |
| `--dry-run` | bool | false | Generate and display the message only. No commit, no push, no staging. |
| `--max-length` | int | 72 | Maximum characters in the commit subject line. |
| `--lang` | string | english | Language for the generated commit message. |
| `--provider` | string | config value | LLM provider to use for this commit only. |
| `--yes, -y` | bool | false | Skip interactive review and commit immediately with the generated message. |
| `--remote <url>` | string | — | Add a new remote and use it for this push. Used on first push of a brand new project. |
| `--remote-name <n>` | string | origin | Name for the remote when using `--remote`. Override when "origin" is already taken. |
| `--branch <n>` | string | current branch | Remote branch name to push to. Defaults to the current local branch name. |
| `--force-push` | bool | false | Push with `--force-with-lease`. Safe force push — fails if remote has unseen commits. Never uses bare `--force`. |

### 7.2 `gitai config`

| Command | Description |
|---|---|
| `gitai config list` | Print all config keys, current values, and which source they came from (flag/env/file/default). |
| `gitai config get <key>` | Print the resolved value of a single config key. |
| `gitai config set <key> <value>` | Update a single value in `~/.gitai/config.yaml`. |
| `gitai config reset` | Reset `~/.gitai/config.yaml` to all defaults. Prompts for confirmation first. |
| `gitai config path` | Print the full path to the config file. |

### 7.3 `gitai` (root flags)

| Flag | Description |
|---|---|
| `--version, -v` | Print gitai version and exit. |
| `--help, -h` | Print help for any command. |

---

## 8. Commit Message Styles

### 8.1 Conventional Commits (default)

Follows the Conventional Commits 1.0.0 specification. Subject line format: `type(scope): description`

```
feat(auth): add JWT refresh token rotation

Implements sliding window refresh token strategy to reduce
re-authentication friction while maintaining session security.

Closes #142
```

Valid types: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `chore`, `ci`, `build`, `revert`

### 8.2 Simple

A single concise sentence. No type prefix, no body.

```
Add JWT refresh token rotation to reduce re-auth friction
```

### 8.3 Emoji

Conventional commits with a leading emoji corresponding to the type.

```
✨ feat(auth): add JWT refresh token rotation
```

| Emoji | Type | Meaning |
|---|---|---|
| ✨ | feat | New feature |
| 🐛 | fix | Bug fix |
| 📝 | docs | Documentation |
| ♻️ | refactor | Refactor |
| ⚡ | perf | Performance |
| ✅ | test | Tests |
| 🔧 | chore | Chores/tooling |
| 🚀 | ci | CI/CD |

---

## 9. Go Architecture Patterns

### 9.1 `os/exec` Usage (`pkg/git`)

All git operations use `os/exec.Command`. Commands are never run through a shell string to avoid injection risks.

```go
// CORRECT — args as separate values
cmd := exec.CommandContext(ctx, "git", "diff", "--staged")

// WRONG — never do this
cmd := exec.Command("sh", "-c", "git diff --staged")

// Capture stdout and stderr separately
var stdout, stderr bytes.Buffer
cmd.Stdout = &stdout
cmd.Stderr = &stderr
if err := cmd.Run(); err != nil {
    return "", fmt.Errorf("git diff failed: %w\nstderr: %s", err, stderr.String())
}
```

### 9.2 Atomic Config Write (`internal/config`)

Config is written atomically: write to a temp file, then `os.Rename`. Rename is atomic on all major OSes — a crash mid-write cannot produce a corrupt config file.

```go
func writeConfig(cfg Config, path string) error {
    data, err := yaml.Marshal(cfg)
    if err != nil { return err }

    tmp := path + ".tmp"
    if err := os.WriteFile(tmp, data, 0600); err != nil { return err }

    return os.Rename(tmp, path)  // atomic on Linux, macOS, Windows
}
```

### 9.3 Context Propagation

All operations that involve I/O accept a `context.Context` as their first argument. The root command creates a context cancelled on SIGINT/SIGTERM.

```go
ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
defer cancel()

// All downstream calls receive this ctx
diff, err := git.GetStagedDiff(ctx)
message, err := provider.GenerateMessage(ctx, diff, opts)
```

### 9.4 Error Handling

Errors are wrapped with context at every layer using `fmt.Errorf` with `%w`. The top-level command handler unwraps and prints a clean user-facing message. Internal errors are never shown raw to the user.

```go
// pkg layer — wraps with technical context
return fmt.Errorf("openai: request failed: %w", err)

// cmd layer — maps to user-facing message
var apiErr *provider.APIError
if errors.As(err, &apiErr) {
    fmt.Fprintf(os.Stderr, "Error: %s\n", apiErr.UserMessage())
    os.Exit(1)
}
```

### 9.5 Interface-Driven Design

The Provider interface is the central extensibility seam. `cmd/` only ever holds a `Provider` interface value — it never imports a concrete provider package directly.

```go
// internal/config — factory function
func (c *Config) BuildProvider() (provider.Provider, error) {
    switch c.Provider {
    case "openai":
        return openai.New(c.APIKey, c.Model), nil
    default:
        return nil, fmt.Errorf("unknown provider: %s", c.Provider)
    }
}

// cmd/commit.go — only knows the interface
p, err := cfg.BuildProvider()
message, err := p.GenerateMessage(ctx, diff, opts)
```

---

## 10. Remote and Branch Management

gitai reduces all remote/branch complexity to three situations. Everything else — fork detection, detached HEAD handling, multiple remote disambiguation — is deferred to v2.

### 10.1 The Three Situations

| Situation | What gitai detects | What gitai does |
|---|---|---|
| Brand new project | `git remote -v` returns nothing | Requires `--remote` flag. Adds origin, commits, pushes with `-u`. |
| New branch in existing project | Remote exists, but current branch has no upstream | Automatically uses `git push -u origin <branch>`. No prompting. |
| All subsequent commits | Remote exists, branch has tracked upstream | Just runs `git push`. Nothing else needed. |

---

### 10.2 Situation 1 — Brand New Project (No Remote)

**Detection**

gitai runs `git remote -v` after every commit before attempting a push. If the output is empty, no remote is configured and the flow below applies.

**Happy Path**

User supplies `--remote` on the first commit. gitai adds origin, commits, and pushes with `-u` to set the upstream in one step.

```bash
# What the user runs:
gitai commit --all --remote https://github.com/user/repo.git --branch main

# What gitai runs internally:
git add -u
git commit -m "feat: initial project setup"
git remote add origin https://github.com/user/repo.git
git push -u origin main

Remote "origin" added.
Pushed to origin/main. Upstream set.
```

**Edge Cases**

| Scenario | Behaviour |
|---|---|
| `--remote` passed, `--branch` omitted | Defaults to current local branch name. Prints: "Pushing to origin/main (use --branch to override)." |
| `--remote` omitted, no remote configured | Commit succeeds. Push skipped. Prints: "Commit saved locally. To push, run: `gitai commit --remote <url>`" |
| `"origin"` remote name already taken | Prints: "A remote named origin already exists. Use --remote-name to specify a different name." |
| Remote URL is invalid format | gitai validates URL format before running `git remote add`. Prints a clear error before any git command runs. |
| Push fails (auth error) | Remote was added successfully. Prints the git auth error and suggests checking credentials or SSH keys. |

---

### 10.3 Situation 2 — New Branch in Existing Project

**Detection**

gitai runs `git rev-parse --abbrev-ref @{upstream}` after commit. If this returns an error (exit code non-zero), the branch has no upstream yet.

**Happy Path**

gitai detects no upstream and automatically adds `-u` to the push. No flags needed, no prompting.

```bash
# User is on branch feature/auth, first commit on this branch:
gitai commit

# gitai detects no upstream, runs:
git commit -m "feat(auth): scaffold auth module"
git push -u origin feature/auth

Committed: feat(auth): scaffold auth module
Pushed to origin/feature/auth. Upstream set.
```

**Edge Cases**

| Scenario | Behaviour |
|---|---|
| User wants a different remote branch name | Pass `--branch` to override: `gitai commit --branch feature/auth-v2` |
| Multiple remotes exist (origin + upstream) | gitai always defaults to the remote named "origin". Use `--remote-name` to override. |

---

### 10.4 Situation 3 — All Subsequent Commits

Remote is set, branch upstream is tracked. gitai just runs `git push`. No detection logic, no prompting, no flags needed.

```bash
gitai commit

Committed: fix(auth): handle token expiry edge case
Pushed to origin/feature/auth
```

| Edge Case | Behaviour |
|---|---|
| Push rejected — branch behind remote | "Push failed: your branch is behind origin/feature/auth. Run: `git pull --rebase` then try again. Or use: `gitai commit --force-push`" |
| Push rejected — auth failure | Raw git error printed. "Check your credentials or SSH keys for origin." |
| Remote unreachable (no internet) | "Push failed: could not reach origin. Commit is saved locally — push when you are connected." |

---

### 10.5 Remote and Branch Flags

| Flag | Type | Default | Description |
|---|---|---|---|
| `--remote <url>` | string | — | Add a new remote and use it for this push. Used on first push of a brand new project. |
| `--remote-name <n>` | string | origin | Name for the remote when using `--remote`. Override when "origin" is already taken. |
| `--branch <n>` | string | current branch | Remote branch name to push to. Defaults to the current local branch name. |
| `--force-push` | bool | false | Push with `--force-with-lease`. Safe force push — never uses bare `--force`. |
| `--no-push` | bool | false | Commit only. Skip push entirely. |

### 10.6 Config Additions

```yaml
# ~/.gitai/config.yaml
default_remote_name: origin   # name used when adding a new remote via --remote
auto_set_upstream: true       # always use -u on first push of any branch
```

> **v1 Scope:** Fork detection, detached HEAD handling, and multiple remote disambiguation are explicitly out of scope for v1. The three situations above cover 95%+ of real usage. Edge cases beyond this are handled by passing explicit `--remote-name` and `--branch` flags.

---

## 11. CI/CD Pipeline

### 11.1 CI on Every Push (`.github/workflows/ci.yaml`)

- **Trigger:** `push` and `pull_request` to main
- **Steps:** checkout → setup Go → `go vet` → `golangci-lint` → `go test ./...`
- Tests must pass on `ubuntu-latest`, `macos-latest`, `windows-latest`
- golangci-lint runs: `errcheck`, `staticcheck`, `gosimple`, `unused`, `gofmt`

### 11.2 Release on Version Tag (`.github/workflows/release.yaml`)

- **Trigger:** push of tag matching `v*.*.*`
- **Steps:** checkout → setup Go → GoReleaser release --clean
- GoReleaser builds binaries for: `darwin/amd64`, `darwin/arm64`, `linux/amd64`, `linux/arm64`, `windows/amd64`
- Binaries are uploaded to GitHub Releases automatically
- Homebrew tap formula is updated automatically
- SHA256 checksums file is generated and attached to the release

### 11.3 GoReleaser Config (`.goreleaser.yaml`)

```yaml
builds:
  - env: [CGO_ENABLED=0]
    goos: [linux, windows, darwin]
    goarch: [amd64, arm64]

archives:
  - format: tar.gz
    format_overrides:
      - goos: windows
        format: zip

checksum:
  name_template: "checksums.txt"

brews:
  - tap:
      owner: you
      name: homebrew-tap
    homepage: https://gitai.sh
    description: AI-powered git commit CLI
```

---

## 12. Testing Strategy

### 12.1 Unit Tests

- `pkg/prompt`: test that given a diff and options, the correct prompt string is produced
- `pkg/git`: test argument construction and output parsing using `exec.Command` mocks
- `internal/config`: test priority resolution, YAML read/write, atomic write
- `pkg/provider/openai`: test request construction, response parsing, retry logic using `httptest.Server`

### 12.2 Integration Tests

- `cmd/commit`: test the full flow with a mock Provider, a real temp git repo, and simulated stdin input
- Integration tests use `t.TempDir()` for isolation — no real git repos or API calls

### 12.3 Mock Provider

A `MockProvider` implementing the `Provider` interface is available in `pkg/provider/mock` for use in tests. It returns a configurable message and can be set to return errors to test error handling paths.

### 12.4 Test Coverage Target

Minimum 80% coverage on all `pkg/*` packages. `internal/config` targets 90% given how critical correct config behaviour is.

---

## 13. Security and Privacy

| Concern | Approach |
|---|---|
| API key storage | Stored in `~/.gitai/config.yaml` with `0600` permissions (user read/write only). Env var always preferred over file. |
| API key in logs | API key is never printed, logged, or included in error messages under any circumstance. |
| Diff privacy | Diff content is sent to the configured LLM provider and nowhere else. No telemetry, no logging, no caching. |
| Command injection | All git commands use `exec.Command` with args as separate values. Shell string execution is never used. |
| Config file permissions | Config directory (`~/.gitai`) and config file are created with `0700` and `0600` respectively. |
| Temp files | Temp files used for editor flow are created with `os.CreateTemp` and deleted immediately after the editor closes. |

---

## 14. Future Work (Post v1)

### 14.1 Additional Providers

- Anthropic Claude — implement Provider interface, add `"anthropic"` to factory
- Ollama (local LLMs) — no API key required, privacy-first option
- Google Gemini

### 14.2 Additional Features

- `gitai log` — regenerate or improve existing commit messages
- `gitai hook` — install gitai as a `prepare-commit-msg` git hook
- `--amend` flag — amend the last commit message using AI
- Team config — project-level `.gitai.yaml` that overrides global config for shared conventions
- `gitai diff` — summarise staged changes in plain English without committing
- Fork detection and detached HEAD handling
- GitHub/GitLab repo creation from CLI during first push

### 14.3 IDE Integrations

- VS Code extension using the importable `pkg/` library
- JetBrains plugin
- Neovim plugin

---

## 15. Success Metrics

| Metric | Target |
|---|---|
| Install to first successful commit | Under 2 minutes for any install method |
| Binary size | Under 15MB for any platform |
| Message generation time | Under 5 seconds for a typical diff on gpt-4o-mini |
| Test coverage (`pkg/*`) | Minimum 80% |
| CI build time | Under 3 minutes |
| Zero data retention | No diff or message content stored anywhere by gitai |

---

*End of Document*