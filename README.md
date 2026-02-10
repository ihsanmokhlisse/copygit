<p align="center">
  <h1 align="center">CopyGit</h1>
  <p align="center">
    <strong>Mirror your Git repositories across GitHub, GitLab, Gitea and any Git server — automatically.</strong>
  </p>
  <p align="center">
    <a href="#installation"><img alt="Go 1.21+" src="https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go&logoColor=white"></a>
    <a href="LICENSE"><img alt="License: MIT" src="https://img.shields.io/badge/License-MIT-yellow.svg"></a>
    <a href="#"><img alt="Platform" src="https://img.shields.io/badge/Platform-Linux%20%7C%20macOS%20%7C%20Windows-blue"></a>
  </p>
</p>

---

CopyGit keeps your Git repositories in sync across multiple hosting providers. Push once, and your code is mirrored everywhere. If one provider goes down, your work is safe on the others.

## Why CopyGit?

- **Bus-factor protection** — Don't depend on a single Git host. Mirror to GitHub, GitLab, Gitea, or your own server.
- **Zero friction** — Install a Git hook and forget about it. Every `git push` mirrors automatically.
- **Multi-repo management** — Register dozens of repos, push them all with one command.
- **Secure by default** — Credentials stored in your OS keychain, SSH keys, or encrypted files (never plaintext in config).

## Features

| Feature | Description |
|---------|-------------|
| **Multi-provider sync** | Push to GitHub, GitLab, Gitea, and generic Git servers simultaneously |
| **Peer-to-peer resilience** | Any provider can be primary — no single point of failure |
| **Git hooks** | Auto-sync on every `git push` via post-push hooks |
| **Background daemon** | Continuous polling-based sync for hands-free operation |
| **Secure credentials** | OS keychain, SSH keys, git helpers, env vars, or encrypted file |
| **Conflict detection** | Detects diverged branches with warn / merge / force-sync options |
| **Multi-repo management** | Register, list, and push all repos at once |
| **JSON output** | Machine-readable output for scripting and CI/CD |

## Installation

### From Source

```bash
git clone https://github.com/ihsanmokhlisse/copygit.git
cd copygit
make build        # Binary at ./bin/copygit
make install      # Installs to $GOPATH/bin
```

### Pre-built Binaries

Download from the [Releases](https://github.com/ihsanmokhlisse/copygit/releases) page. Binaries are available for Linux, macOS (Intel & Apple Silicon), and Windows.

### Verify

```bash
copygit version
```

## Quick Start

```bash
# 1. Add your providers
copygit config add-provider github github https://github.com
copygit config add-provider gitlab gitlab https://gitlab.com

# 2. Authenticate (stores token in OS keychain)
copygit login --provider github
copygit login --provider gitlab

# 3. Register a repo
cd ~/projects/my-app
copygit init

# 4. Push everywhere
copygit push

# 5. (Optional) Auto-sync on every git push
copygit hooks install
```

That's it. Every `git push` now mirrors to all your providers.

## Commands

```
copygit init [path]                Register a repo for syncing
copygit push [--all] [--force]     Push to all configured providers
copygit sync [--force]             Fetch + detect conflicts + push
copygit status [--all] [--json]    Show per-provider sync state
copygit list                       List all registered repos
copygit remove <path>              Unregister a repo

copygit config add-provider        Add a Git provider
copygit config list-providers      List configured providers
copygit config remove-provider     Remove a provider

copygit login --provider <name>    Authenticate with a provider
copygit hooks install              Install auto-sync hook
copygit hooks uninstall            Remove hook
copygit hooks status               Check hook state

copygit daemon start               Start background sync daemon
copygit daemon stop                Stop the daemon
copygit daemon status              Check daemon state
```

Use `--verbose` for debug output, `--json` for machine-readable output, `--quiet` for silence.

## How It Works

```
┌─────────────┐     copygit push     ┌──────────┐
│  Local Repo  │ ──────────────────► │  GitHub   │
│  (primary)   │ ──────────────────► │  GitLab   │
│              │ ──────────────────► │  Gitea    │
└─────────────┘                      └──────────┘

         copygit sync
    ◄──────────────────►
    Fetch + Conflict Detection + Push
```

CopyGit wraps native Git commands — it adds remotes to your repo and pushes/fetches to all of them. No custom protocols, no lock-in.

### Configuration Model

- **Global config** (`~/.copygit/config`) — Provider definitions shared across all repos
- **Per-repo config** (`.copygit.toml`) — Which providers this repo syncs to
- **Registry** (`~/.copygit/repos.toml`) — Tracks all registered repos

### Credential Resolution Chain

CopyGit tries each source in order until it finds a valid credential:

1. OS Keychain (macOS Keychain, Linux Secret Service, Windows Credential Manager)
2. SSH Keys (`~/.ssh/id_*`)
3. Git credential helper (`git credential-store`, `git credential-cache`)
4. Environment variable (`COPYGIT_TOKEN_<PROVIDER_NAME>`)
5. Encrypted file (`~/.copygit/credentials`, enforced 0600 permissions)

## Multi-Repository Workflow

```bash
# Register multiple repos
cd ~/projects/api    && copygit init
cd ~/projects/web    && copygit init
cd ~/projects/infra  && copygit init

# Push all at once
copygit push --all

# Check everything
copygit status --all
```

## Architecture

```
cmd/copygit/           CLI entry point + Cobra commands
internal/
  model/               Domain types (config, credential, errors, conflict)
  config/              TOML config management (global + per-repo + registry)
  credential/          Multi-source credential chain (keyring, SSH, env, file)
  git/                 Git CLI wrapper (executor, remote, branch, tag, status)
  provider/            Provider adapters (GitHub, GitLab, Gitea, generic)
  sync/                Sync orchestrator (push, fetch, status, conflict detection)
  output/              Formatters (text, JSON)
  lock/                Per-repo file locking (flock/LockFileEx)
  hook/                Git hook management (post-push with marker-based insertion)
  daemon/              Background polling daemon with PID management
scripts/               Test orchestrator and CI helpers
testdata/              Test fixtures (config files, golden files)
```

### Design Principles

- **No global state** — All dependencies wired via constructor injection
- **Interface-driven** — `GitExecutor`, `Provider`, `CredentialResolver`, `Formatter` are all interfaces with fakes for testing
- **Native Git** — Wraps `git` CLI via `os/exec`, no custom protocols
- **Per-repo locking** — File-based locks prevent concurrent syncs on the same repo
- **Cross-platform** — Build tags for Unix (flock) and Windows (LockFileEx)

## Development

```bash
make build              # Build binary
make test               # Run all unit tests
make test-race          # Race condition detection
make test-integration   # Integration tests (creates real git repos)
make lint               # Run golangci-lint
make coverage           # Coverage report with threshold check
make ci                 # Full CI pipeline (build → lint → test → race → coverage)
```

### Requirements

- Go 1.21+
- Git 2.x+
- golangci-lint (for linting)

## Contributing

1. Fork the repo
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Run the full CI suite (`make ci`)
4. Commit your changes
5. Open a Pull Request

## License

[MIT](LICENSE)

---

<p align="center">
  Built with Go. Designed for developers who value redundancy.
</p>
