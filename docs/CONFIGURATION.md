# CopyGit Configuration Guide

## Overview

CopyGit uses a **centralized configuration approach** for security and simplicity:

- **Global Config**: `~/.copygit/config` - All providers and global settings
- **Per-Repository Config**: `.copygit.toml` - Repository-specific sync targets (LOCAL ONLY, NOT COMMITTED)
- **Credentials**: Stored securely in system keychain (not in git)

## Why Not Commit .copygit.toml?

**Security Concerns:**
- Personal sync target configuration
- User-specific metadata preferences
- Potentially sensitive override settings
- Local state (last sync times)

**Solution:**
- `.copygit.toml` is ONLY for local use
- Never committed to version control
- Each developer creates their own locally via `copygit init`

## Configuration Hierarchy

### 1. Global Config (`~/.copygit/config`)

Defines all available providers and global settings that apply to every repository:

```toml
[providers.github]
name = 'github'
type = 'github'
base_url = 'https://github.com'
auth_method = 'https'

[providers.gitlab]
name = 'gitlab'
type = 'gitlab'
base_url = 'https://gitlab.com'
auth_method = 'https'

[sync]
max_retries = 5
retry_base_delay = '5s'

[daemon]
poll_interval = '30s'
auto_start = false
```

**Used for:**
- ✅ Defining available providers
- ✅ Global sync settings
- ✅ Daemon configuration
- ✅ Logging preferences

**Shared across:** All repositories and developers

---

### 2. Repository Config (`.copygit.toml` - LOCAL ONLY)

Defines sync targets and metadata preferences for a single repository:

```toml
version = '1'

[[sync_targets]]
provider = 'github'
remote_url = 'https://github.com/owner/repo.git'
enabled = true

[[sync_targets]]
provider = 'gitlab'
remote_url = 'https://gitlab.com/owner/repo.git'
enabled = true

[metadata]
inherit_from = 'github'  # Inherit from GitHub when creating on other providers
visibility = 'private'   # Default to private repos

# Per-target overrides
[[sync_targets]]
provider = 'gitlab'
[sync_targets.overrides]
visibility = 'public'    # Override: make GitLab public
topics = ['mirror']
```

**Used for:**
- ✅ Repository-specific sync targets
- ✅ Metadata inheritance rules
- ✅ Per-provider visibility/metadata overrides
- ✅ Remote URL mappings

**Scope:** Single repository only

**Security:** Never committed to git (in `.gitignore`)

---

## Setup Process

### For Repository Owners (First Time)

1. **Configure providers globally** (once per machine):
```bash
copygit config add-provider github --auth-method token
copygit config add-provider gitlab --auth-method token
```

2. **Authenticate** with each provider:
```bash
copygit login github
copygit login gitlab
```

3. **Initialize the repository** (creates local `.copygit.toml`):
```bash
cd /path/to/my-project
copygit init .
```

4. **Define sync targets** in the interactive prompt

Result:
- `~/.copygit/config` - Global provider definitions
- `.copygit.toml` - Your local sync targets (not shared)

### For Contributors (Cloning Existing Repo)

1. **Clone the repository**:
```bash
git clone https://github.com/owner/repo.git
cd repo
```

2. **Initialize CopyGit** (creates your own `.copygit.toml`):
```bash
copygit init .
```

3. **Your config is local** and never shared with others

---

## Workflow Example

### Complete Setup
```bash
# 1. Configure providers globally (one-time per machine)
copygit config add-provider github --auth-method token
copygit config add-provider gitlab --auth-method token
copygit config add-provider gitea --auth-method token

# 2. Authenticate
copygit login github
copygit login gitlab
copygit login gitea

# 3. Initialize repository with CopyGit
cd ~/my-awesome-project
copygit init .

# 4. Follow the prompts to configure sync targets
# This creates ~/.copygit/config (global)
#         and .copygit.toml (local)
```

### Daily Usage
```bash
# Push with automatic repo creation and metadata inheritance
copygit push

# Check status across all providers
copygit status

# Full bidirectional sync
copygit sync

# View help
copygit help
```

---

## Files Overview

```
HOME DIRECTORY
~/.copygit/
├── config              ← Global provider definitions (SHARED)
├── repos.toml          ← Global repository registry (LOCAL)
├── locks/              ← Lock files (LOCAL)
└── (credentials in macOS keychain, NOT in files)

PROJECT DIRECTORY
~/my-project/
├── .copygit.toml       ← Local sync configuration (NOT committed to git)
├── .gitignore          ← Includes .copygit.toml
└── (other project files)
```

---

## Metadata Inheritance

The `[metadata]` section in `.copygit.toml` controls how repositories are created on target providers:

### Basic Configuration
```toml
[metadata]
# Which provider to inherit metadata from
inherit_from = 'github'

# Fallback if no source metadata found
visibility = 'private'
description = 'My project'
topics = ['golang', 'mirror']
```

### With Per-Provider Overrides
```toml
[metadata]
inherit_from = 'github'
visibility = 'private'
description = 'My project mirror'

# GitHub: use inherited settings
[[sync_targets]]
provider = 'github'
remote_url = 'https://github.com/owner/repo.git'
enabled = true

# GitLab: override visibility to public
[[sync_targets]]
provider = 'gitlab'
remote_url = 'https://gitlab.com/owner/repo.git'
enabled = true
[sync_targets.overrides]
visibility = 'public'
topics = ['mirror', 'gitlab']

# Gitea: minimal config
[[sync_targets]]
provider = 'gitea'
remote_url = 'https://gitea.example.com/owner/repo.git'
enabled = true
[sync_targets.overrides]
visibility = 'private'
```

### Inheritance Flow

When creating a repository on a target provider:

1. **Check explicit `inherit_from`**
   - If set, fetch metadata from that provider
   - If "none", skip inheritance

2. **Fallback to discovery order**
   - github → gitlab → gitea
   - Uses first provider that has the repo

3. **Apply global overrides**
   - `[metadata]` section values override defaults

4. **Apply per-target overrides**
   - `[sync_targets.overrides]` take final precedence

5. **Handle unsupported fields gracefully**
   - Warn about platform limitations
   - Don't fail the operation

---

## Best Practices

### DO Commit to Git
- ✅ `.gitignore` entry for `.copygit.toml`
- ✅ Configuration documentation
- ✅ Example `.copygit.toml.example` (without credentials)

### DO NOT Commit to Git
- ❌ `.copygit.toml` (personal sync targets)
- ❌ `~/.copygit/` directory (local state)
- ❌ Any credentials or tokens

### DO Store Globally
- ✅ Provider definitions in `~/.copygit/config`
- ✅ Credentials in system keychain
- ✅ Repository registry in `~/.copygit/repos.toml`

### DO Customize Locally
- ✅ Sync targets in `.copygit.toml`
- ✅ Metadata overrides per repository
- ✅ Personal preferences

---

## Example `.copygit.toml.example`

Include this in your repository as a template (without sensitive values):

```toml
# Example CopyGit Repository Configuration
# Copy to .copygit.toml and customize for your setup

version = '1'

# Define sync targets for this repository
# Each target represents a git remote on a different provider

[[sync_targets]]
provider = 'github'
# Replace with your GitHub repository URL
remote_url = 'https://github.com/your-username/your-repo.git'
enabled = true

[[sync_targets]]
provider = 'gitlab'
# Replace with your GitLab repository URL
remote_url = 'https://gitlab.com/your-username/your-repo.git'
enabled = true

[[sync_targets]]
provider = 'gitea'
# Replace with your Gitea repository URL
remote_url = 'https://gitea.example.com/your-username/your-repo.git'
enabled = true

# Optional: Configure metadata inheritance and defaults
[metadata]
# Where to inherit metadata from: 'github', 'gitlab', 'gitea', or 'none'
inherit_from = 'github'

# Default visibility if no source metadata found
visibility = 'private'

# Optional metadata defaults
description = 'Project description'
topics = ['golang', 'mirror']
wiki_enabled = false
issues_enabled = true

# Per-provider overrides (optional)
[[sync_targets]]
provider = 'gitlab'
[sync_targets.overrides]
visibility = 'public'
topics = ['mirror', 'public-copy']
```

---

## Troubleshooting

### Configuration not being applied
- Check `~/.copygit/config` exists and is valid TOML
- Verify `.copygit.toml` is in the repository root
- Run `copygit config list-providers` to verify setup

### Credentials not working
- Re-authenticate: `copygit login github`
- Check keychain for stored credentials
- Verify tokens haven't expired

### Repository creation failing
- Ensure you have credentials for the target provider
- Check that the repository name doesn't already exist
- Verify you have permission to create repositories

---

## Next Steps

- [Usage Guide](./USAGE.md) - Daily workflow and commands
- [API Documentation](./API.md) - Provider interface details
- [Contributing](./CONTRIBUTING.md) - Development setup
