# CopyGit User Guide

CopyGit is a CLI tool that syncs your Git repositories across multiple providers (GitHub, GitLab, Gitea) automatically, with peer-to-peer resilience — if one provider goes down, your work stays safe on the others.

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Provider Setup](#provider-setup)
- [Authentication](#authentication)
- [Repository Management](#repository-management)
- [Sync Operations](#sync-operations)
- [Git Hooks (Automatic Sync)](#git-hooks-automatic-sync)
- [Background Daemon](#background-daemon)
- [Multi-Repository Workflow](#multi-repository-workflow)
- [Configuration Files](#configuration-files)
- [Conflict Resolution](#conflict-resolution)
- [Troubleshooting](#troubleshooting)

---

## Installation

### Build from Source

Requires Go 1.21+.

```bash
git clone https://github.com/imokhlis/copygit.git
cd copygit

# Build the binary to ./bin/copygit
make build

# Or install to $GOPATH/bin (adds to PATH)
make install
```

### Verify Installation

```bash
copygit version
```

### Pre-built Binaries

GoReleaser produces cross-platform binaries for Linux, macOS (amd64/arm64), and Windows. Check the GitHub Releases page.

---

## Quick Start

```bash
# 1. Add a provider (e.g., GitHub)
copygit config add-provider my-github github https://github.com

# 2. Authenticate — enter your personal access token when prompted
copygit login --provider my-github

# 3. cd into an existing git repo and register it
cd ~/projects/my-app
copygit init

# 4. Push to all configured providers
copygit push

# 5. Check sync status
copygit status
```

That's it. Your repo now syncs to GitHub. Add more providers to push everywhere.

---

## Provider Setup

### Add a Provider

```bash
copygit config add-provider <name> <type> <url>
```

Supported types: `github`, `gitlab`, `gitea`, `generic`.

```bash
copygit config add-provider work-gh    github  https://github.com
copygit config add-provider personal   gitlab  https://gitlab.com
copygit config add-provider selfhost   gitea   https://git.myserver.com
copygit config add-provider bare-srv   generic ssh://git@mybox.local
```

### List Providers

```bash
copygit config list-providers
```

### Remove a Provider

```bash
copygit config remove-provider <name>
```

---

## Authentication

### Interactive Login

```bash
copygit login --provider <name>
```

You'll be prompted to enter your token. The token is stored securely using the first available method from this resolution chain:

1. **OS Keychain** — macOS Keychain, Linux Secret Service, Windows Credential Manager
2. **SSH Keys** — detected from `~/.ssh/id_*` (read-only, CopyGit won't create keys)
3. **Git Credential Helper** — your existing `git credential-store` or `git credential-cache`
4. **Environment Variable** — `COPYGIT_TOKEN_<PROVIDER_NAME>` (uppercased)
5. **Credential File** — `~/.copygit/credentials` (TOML, enforced 0600 permissions)

### Environment Variables

For CI/CD or scripting, set tokens via environment:

```bash
export COPYGIT_TOKEN_WORK_GH=ghp_xxxxxxxxxxxxxxxxxxxx
export COPYGIT_TOKEN_PERSONAL=glpat-xxxxxxxxxxxxxxxx
```

### Token Requirements

| Provider | Token Type | Required Scopes |
|----------|-----------|-----------------|
| GitHub   | Personal Access Token | `repo` |
| GitLab   | Personal Access Token | `api`, `write_repository` |
| Gitea    | API Token | Read/Write access |
| Generic  | Depends on server | — |

---

## Repository Management

### Register a Repository

From inside any git repository:

```bash
copygit init
```

Or specify a path:

```bash
copygit init /path/to/repo
```

This registers the repo in your global registry (`~/.copygit/repos.toml`) and creates a per-repo config (`.copygit.toml`) with sync targets matching your configured providers.

### List Registered Repositories

```bash
copygit list
```

Shows all repos, their paths, aliases, and last sync times.

### Remove a Repository

```bash
copygit remove /path/to/repo
```

Unregisters the repo. Does not delete any git data or remotes.

---

## Sync Operations

### Push

Push branches and tags to all configured providers:

```bash
copygit push
```

Options:

```bash
copygit push --providers github,gitlab   # Push to specific providers only
copygit push --force                      # Force-push (overwrite remote)
copygit push --dry-run                    # Preview without pushing
copygit push --all                        # Push all registered repos
```

### Full Sync (Fetch + Push)

Fetch from remotes, detect conflicts, then push:

```bash
copygit sync
```

Options:

```bash
copygit sync --force            # Force-push, overwriting diverged branches
copygit sync --conflict warn    # Warn on conflicts (default)
copygit sync --conflict merge   # Attempt auto-merge
```

### Status

Check per-provider sync state:

```bash
copygit status                # Current repo
copygit status --all          # All registered repos
copygit status --json         # Machine-readable output
```

---

## Git Hooks (Automatic Sync)

Install a `post-push` hook so CopyGit syncs automatically after every `git push`:

```bash
# Install hook
copygit hooks install

# Check if hooks are installed
copygit hooks status

# Remove hooks
copygit hooks uninstall
```

The hook uses marker-based insertion, so it won't overwrite existing hooks. If you already have a `post-push` hook, CopyGit's section is inserted between `### COPYGIT START ###` and `### COPYGIT END ###` markers.

After installation, the workflow is:

```bash
git add -A && git commit -m "feature: add thing"
git push origin main
# → CopyGit automatically pushes to all other providers
```

---

## Background Daemon

Run CopyGit as a background polling service that periodically syncs all registered repos:

```bash
# Start the daemon
copygit daemon start

# Check daemon status (running/stopped, PID)
copygit daemon status

# Stop the daemon gracefully
copygit daemon stop
```

The daemon writes its PID to `~/.copygit/copygit.pid` and responds to SIGINT/SIGTERM for graceful shutdown. The default polling interval syncs all repos in the registry on each tick.

---

## Multi-Repository Workflow

CopyGit uses a global provider model with per-repo sync targets. You configure providers once, then register any number of repos.

### Setup: One-Time Provider Configuration

```bash
copygit config add-provider work    github  https://github.com
copygit config add-provider backup  gitlab  https://gitlab.com
copygit login --provider work
copygit login --provider backup
```

### Register Multiple Repos

```bash
cd ~/projects/app1 && copygit init
cd ~/projects/app2 && copygit init
cd ~/projects/library && copygit init
```

### Operate Across All Repos

```bash
# Push all repos to all providers
copygit push --all

# Check status of everything
copygit status --all
```

### Per-Repo Overrides

Each repo gets a `.copygit.toml` that controls which providers it syncs to. Edit it to disable specific providers for a repo:

```toml
[[sync_targets]]
provider = "work"
enabled = true

[[sync_targets]]
provider = "backup"
enabled = false   # Skip this provider for this repo
```

---

## Configuration Files

CopyGit uses three TOML files:

### 1. Global Config (`~/.copygit/config`)

Provider definitions and global preferences.

```toml
[[providers]]
name = "work"
type = "github"
base_url = "https://github.com"
auth_method = "token"

[[providers]]
name = "backup"
type = "gitlab"
base_url = "https://gitlab.com"
auth_method = "token"
```

### 2. Per-Repo Config (`<repo>/.copygit.toml`)

Which providers this repo syncs to, with optional remote URL overrides.

```toml
[[sync_targets]]
provider = "work"
enabled = true
remote_url = "git@github.com:user/repo.git"

[[sync_targets]]
provider = "backup"
enabled = true
```

### 3. Registry (`~/.copygit/repos.toml`)

Tracks all registered repositories.

```toml
[[repos]]
path = "/Users/me/projects/app1"
alias = "app1"
last_sync = "2026-02-10T14:30:00Z"

[[repos]]
path = "/Users/me/projects/app2"
alias = "app2"
```

### 4. Credentials (`~/.copygit/credentials`)

Fallback credential file with enforced 0600 permissions. Only used when keychain/SSH/env are unavailable.

```toml
[work]
token = "ghp_xxxxxxxxx"

[backup]
token = "glpat-xxxxxxxxx"
```

---

## Conflict Resolution

When `copygit sync` detects that remote and local branches have diverged:

| Mode | Flag | Behavior |
|------|------|----------|
| Warn (default) | `--conflict warn` | Reports the conflict, skips the push |
| Merge | `--conflict merge` | Attempts a fast-forward merge |
| Force | `--force` | Force-pushes local state to remote |

Example output:

```
⚠ Conflict detected on branch main for provider "backup":
  Local:  abc1234 (ahead 2, behind 1)
  Remote: def5678
  → Skipping push. Use --force to overwrite or --conflict merge to attempt merge.
```

---

## Troubleshooting

### "git not found"

CopyGit requires Git to be installed and in your `PATH`:

```bash
which git
git --version
```

### "authentication failed"

1. Re-enter credentials: `copygit login --provider <name>`
2. Verify your token has the required scopes (see [Token Requirements](#token-requirements))
3. Check if the provider URL is correct: `copygit config list-providers`

### "failed to acquire lock"

Another CopyGit process is already operating on this repo. CopyGit uses per-repo file locking to prevent concurrent syncs.

```bash
# Check for running CopyGit processes
ps aux | grep copygit

# If a process is stuck, kill it
kill <pid>

# Lock files are in ~/.copygit/locks/ and auto-clean on next run
```

### Hook not triggering

1. Verify you're using `git push` from the command line, not a GUI client
2. Check hook status: `copygit hooks status`
3. Verify the hook file exists: `ls -la .git/hooks/post-push`
4. Check hook content for the CopyGit markers: `cat .git/hooks/post-push`

### Daemon not starting

```bash
# Check if already running
copygit daemon status

# If stale PID file exists, stop cleans it up
copygit daemon stop

# Check logs for errors
copygit daemon start  # runs in foreground for debugging
```

### "repo not registered"

Make sure you've run `copygit init` in the repo directory (or with the repo path):

```bash
copygit init /path/to/repo
copygit list  # verify it appears
```

### Verbose Output

Add `--verbose` to any command for debug-level logging:

```bash
copygit push --verbose
copygit sync --verbose
```

### JSON Output

Use `--json` for machine-readable output, useful for scripting:

```bash
copygit status --json | jq '.providers[].name'
```

---

## Global Flags

| Flag | Description |
|------|-------------|
| `--config <path>` | Override global config file path |
| `--verbose` | Enable debug logging |
| `--quiet` | Suppress non-error output |
| `--json` | Output in JSON format |

---

## Development

See the [README](../README.md) for build targets and architecture overview.
