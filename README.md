# gitai

**AI-powered git commit CLI** — Generate meaningful commit messages using LLMs, review them interactively, then commit and push — all in one command.

## Install

```bash
# Homebrew (Mac/Linux)
brew install gitai

# Go install
go install github.com/Chifez/gitai@latest

# From source
git clone https://github.com/Chifez/gitai.git
cd gitai
go build -o gitai .
```

## Quick Start

```bash
# Stage and commit with AI-generated message
git add .
gitai commit

# Stage specific files and commit
gitai commit src/auth.go src/middleware.go

# Stage all changes and commit
gitai commit --all

# Interactive file picker
gitai commit --pick

# Skip review, commit immediately
gitai commit --all --yes

# Dry run — see the generated message without committing
gitai commit --dry-run

# First push to a new remote
gitai commit --all --remote https://github.com/user/repo.git
```

## Commit Styles

```bash
# Conventional (default): feat(auth): add JWT refresh token rotation
gitai commit --style conventional

# Simple: Add JWT refresh token rotation
gitai commit --style simple

# Emoji: ✨ feat(auth): add JWT refresh token rotation
gitai commit --style emoji
```

## Configuration

```bash
# List all config values
gitai config list

# Get/set individual values
gitai config get model
gitai config set model gpt-4o
gitai config set style emoji
gitai config set auto_push false

# Reset to defaults
gitai config reset

# Show config file path
gitai config path
```

### Config File (`~/.gitai/config.yaml`)

```yaml
provider: openai
model: gpt-4o-mini
style: conventional
auto_push: true
max_length: 72
lang: english
include_body: true
default_remote_name: origin
auto_set_upstream: true
```

### Environment Variables

| Variable | Description |
|---|---|
| `OPENAI_API_KEY` | OpenAI API key (always checked first) |
| `GITAI_MODEL` | Override model |
| `GITAI_PROVIDER` | Override provider |
| `GITAI_STYLE` | Override commit style |
| `GITAI_AUTO_PUSH` | `true` or `false` |
| `GITAI_LANG` | Override message language |

### Priority

CLI flag > Environment variable > Config file > Built-in default

## Full CLI Reference

### `gitai commit`

| Flag | Description |
|---|---|
| `[files...]` | Files to stage before committing |
| `--pick, -p` | Interactive file picker |
| `--all, -a` | Stage all modified tracked files |
| `--include-untracked` | With `--all`, also stage untracked files |
| `--model` | LLM model for this commit |
| `--style` | Commit style: conventional, simple, emoji |
| `--context` | Natural language hint for the AI |
| `--max-length` | Max subject line characters (default: 72) |
| `--lang` | Language for the message |
| `--provider` | LLM provider for this commit |
| `--no-push` | Skip push |
| `--push` | Force push even if auto_push is off |
| `--dry-run` | Display message only, no commit |
| `--yes, -y` | Skip review, commit immediately |
| `--remote <url>` | Add remote and push (first push) |
| `--remote-name` | Remote name (default: origin) |
| `--branch` | Remote branch to push to |
| `--force-push` | Push with `--force-with-lease` |

### `gitai config`

| Command | Description |
|---|---|
| `list` | Print all config values and sources |
| `get <key>` | Print a single config value |
| `set <key> <value>` | Update a config value |
| `reset` | Reset config to defaults |
| `path` | Print config file path |

## As a Go Library

```go
import (
    "github.com/Chifez/gitai/pkg/provider"
    "github.com/Chifez/gitai/pkg/git"
    "github.com/Chifez/gitai/pkg/prompt"
)
```

All package under `pkg/` are public and importable.

## License

MIT
