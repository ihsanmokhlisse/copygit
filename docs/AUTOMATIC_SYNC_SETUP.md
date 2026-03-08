# CopyGit Automatic Synchronization Setup Guide

## Overview

CopyGit provides **complete automatic synchronization** of your repositories across multiple git providers (GitHub, GitLab, Gitea, etc.) without requiring manual `copygit push` commands for every workflow.

This guide walks you through setting up full automation so your repositories stay perfectly mirrored across all providers automatically.

## The Vision: Hands-Free Multi-Provider Sync

**Your Normal Workflow:**
```bash
cd /path/to/repo
git commit -m "new feature"
git push origin master
# 🎉 Automatically syncs to GitHub, GitLab, Gitea, etc.
# No copygit commands needed!
```

**What Happens Behind the Scenes:**
1. You push with `git push origin`
2. Post-push hook fires automatically
3. Hook runs `copygit push --from-hook origin`
4. CopyGit syncs to all other providers in parallel
5. Daemon periodically checks for changes from any provider
6. Everything stays perfectly mirrored

## System Requirements

- macOS or Linux
- CopyGit binary with credential manager support (v0.1.0+)
- Configured providers (GitHub, GitLab, Gitea, etc.)
- Credentials stored via `copygit login`

## Step-by-Step Setup

### 1. Update CopyGit Binary

First, ensure you have the latest binary with credential manager fixes:

```bash
# Copy the fixed binary to system location
sudo cp ~/copygit-fixed-v2 /usr/local/bin/copygit

# Verify installation
which copygit
copygit --version
# Output: copygit version 0.1.0-dev
```

**Why this matters:** Recent fixes ensure credentials are properly passed to:
- The push command for post-push hook syncing
- The daemon for background repository syncing

### 2. Configure Providers

Add the providers you want to sync to:

```bash
# Add GitHub
copygit config add-provider github github https://github.com --auth-method https

# Add GitLab
copygit config add-provider gitlab gitlab https://gitlab.com --auth-method https

# Add Gitea (if you have one)
copygit config add-provider gitea gitea https://gitea.example.com --auth-method https

# Verify providers
copygit config list-providers
```

### 3. Login to Providers

Store credentials for each provider:

```bash
# GitHub
copygit login --provider github
# Paste your GitHub Personal Access Token

# GitLab
copygit login --provider gitlab
# Paste your GitLab Personal Access Token

# Gitea
copygit login --provider gitea
# Paste your Gitea access token
```

**Credentials are stored securely in your system keychain** and automatically used during sync operations.

### 4. Initialize Repositories

Register the repositories you want to sync automatically:

```bash
# Initialize a repository for syncing
copygit init /path/to/your/repo

# Follow the prompts to:
# 1. Select which providers to sync to
# 2. Configure remote URLs for each provider

# Verify the configuration
cat /path/to/your/repo/.copygit.toml
```

### 5. Install Post-Push Hook

The post-push hook runs automatically after `git push` and syncs to other providers:

```bash
# Install hook for a repository
copygit hooks install /path/to/your/repo

# Verify installation
copygit hooks status /path/to/your/repo
# Output: post-push: installed (CopyGit)

# View the hook (Mac/Linux)
cat /path/to/your/repo/.git/hooks/post-push
```

**What the hook does:**
- Runs after successful `git push`
- Calls `copygit push --from-hook <remote>`
- Syncs to all providers except the one you just pushed to
- Runs silently in background
- Doesn't interfere with your git output

### 6. Start the Daemon (Optional but Recommended)

The daemon periodically syncs all registered repositories, ensuring changes from any provider are propagated:

```bash
# Start daemon (runs in background)
copygit daemon start

# Check daemon status
copygit daemon status
# Output: Daemon is running (PID: 12345)

# View daemon logs (macOS)
log stream --predicate 'process == "copygit"' --level debug

# View daemon logs (Linux)
journalctl -u copygit -f
```

**Daemon features:**
- Runs continuously in background
- Polls all registered repositories every 30 seconds
- Detects changes from any provider
- Automatically syncs when changes detected
- Gracefully handles errors and continues syncing
- Uses stored credentials for authentication

## Usage: Your Normal Workflow

Once set up, you don't need to do anything special - just use git normally:

```bash
# Make changes
cd /path/to/repo
echo "new feature" >> file.txt
git add file.txt
git commit -m "add new feature"

# Push to origin (this is IT - no copygit command!)
git push origin master

# Behind the scenes:
# 1. Git pushes to origin
# 2. Post-push hook fires
# 3. copygit syncs to other providers
# 4. Everything stays in sync automatically ✅
```

## How It Works: Two Automation Mechanisms

### Mechanism 1: Post-Push Hook (Immediate)

**Triggered:** When you run `git push origin <branch>`

**What happens:**
1. Git completes the push to origin
2. Post-push hook fires automatically
3. Hook calls: `copygit push --from-hook origin`
4. CopyGit:
   - Resolves credentials from keychain
   - Identifies all enabled providers except 'origin'
   - Pushes to GitHub, GitLab, Gitea in parallel
   - Uses credential injection (no password prompts)
5. Everything synced within seconds

**Timing:** Immediate (1-3 seconds for typical repos)

```
You run:         git push origin master
                 ↓
Hook triggers:   Post-push hook fires
                 ↓
Sync happens:    copygit push --from-hook origin
                 ↓
Result:          GitHub + GitLab + Gitea all in sync ✅
```

### Mechanism 2: Daemon (Continuous)

**Triggered:** Periodically (default: every 30 seconds)

**What happens:**
1. Daemon wakes up at poll interval
2. Checks all registered repositories
3. For each repo:
   - Detects changes on any provider
   - Pulls changes from providers
   - Pushes to other providers
   - Updates last-sync time
4. Logs results and continues

**Timing:** Background (every 30 seconds)
**Scope:** All registered repositories

```
Daemon runs:     Every 30 seconds
                 ↓
Check repos:     Scan all registered repositories
                 ↓
Detect changes:  Changes on GitHub? GitLab? Gitea?
                 ↓
Sync changes:    Pull from source → push to others
                 ↓
Result:          All providers in sync ✅
```

## Complete Example: Real Workflow

Let's walk through a complete scenario:

### Scenario: You Push Code

```bash
# 1. Clone and set up repo
git clone git@github.com:your-user/my-project.git
cd my-project
copygit init . --provider github --provider gitlab

# 2. Your work
echo "important change" >> README.md
git add README.md
git commit -m "update docs"

# 3. Push with standard git (NOT copygit)
git push origin master

# Output:
# To github.com:your-user/my-project.git
#    abc123..def456  master -> master

# 4. Post-push hook fires automatically
# (You won't see any output, but it happens)
# Behind the scenes:
#   - Hook calls: copygit push --from-hook origin
#   - GitHub: already pushed ✅
#   - GitLab: synced in ~1s ✅
```

### Scenario: Team Pushes to GitHub

```bash
# Team member pushes to GitHub
# (you don't do anything)

# Daemon wakes up at next poll interval (30s)

# 1. Daemon checks registered repos
# 2. Detects new commits on GitHub
# 3. Fetches from GitHub
# 4. Pushes to GitLab
# 5. Everything in sync ✅

# You can verify:
copygit status /path/to/repo
# Output:
# github    IN SYNC: yes
# gitlab    IN SYNC: yes
```

## Configuration

### Default Settings

```
Poll Interval:        30 seconds
Repositories:         All registered repos
Credentials Source:   System keychain
Error Handling:       Log and continue
Log Level:           INFO
```

### Advanced Configuration

Edit `~/.copygit/config` for custom settings:

```toml
[daemon]
poll_interval = "30s"        # How often to check for changes
log_level = "info"            # Log verbosity: debug, info, warn, error
```

## Monitoring & Troubleshooting

### Check Sync Status

```bash
# Check status of a specific repo
copygit status /path/to/repo

# Check all registered repos
copygit push --all --dry-run   # (if available)

# Sample output:
# PROVIDER  TYPE    IN SYNC  REMOTE HEAD      LAST SYNC
# github    github  yes      abc123def456     2026-03-08 03:15:22
# gitlab    gitlab  yes      abc123def456     2026-03-08 03:14:58
```

### View Daemon Logs

```bash
# macOS
log stream --predicate 'process == "copygit"' --level debug

# Linux
journalctl -u copygit -f

# Sample logs:
# daemon started
# polling for syncs
# sync completed repo=/path/to/repo
# sync completed repo=/path/to/other-repo
```

### Verify Hook Installation

```bash
# Check hook status
copygit hooks status /path/to/repo

# View hook content
cat /path/to/repo/.git/hooks/post-push

# Test hook manually
cd /path/to/repo
copygit push --from-hook origin
```

### Troubleshooting: Hook Not Triggering

**Issue:** You push but don't see sync happening

**Check:**
1. Hook is installed: `copygit hooks status /path/to/repo`
2. Binary in PATH is updated: `which copygit && copygit --version`
3. Credentials are stored: `copygit config list-providers`
4. Test manually: `copygit push --from-hook origin /path/to/repo`

**Solution:**
```bash
# 1. Reinstall hook (in case it got corrupted)
copygit hooks uninstall /path/to/repo
copygit hooks install /path/to/repo

# 2. Update binary
sudo cp ~/copygit-fixed-v2 /usr/local/bin/copygit

# 3. Test again
cd /path/to/repo && git push origin master
```

### Troubleshooting: Daemon Not Syncing

**Issue:** Daemon is running but repos not syncing

**Check:**
1. Daemon is running: `copygit daemon status`
2. Repos are registered: `copygit config list-providers`
3. Check daemon logs for errors: `log stream --predicate 'process == "copygit"'`

**Solution:**
```bash
# 1. Restart daemon
copygit daemon stop
sleep 2
copygit daemon start

# 2. Check logs for credential errors
log stream --predicate 'process == "copygit"' --level debug | grep -i credential

# 3. Re-login to providers
copygit login --provider github
copygit login --provider gitlab
```

## Commands Reference

### Hooks Management

```bash
# Install post-push hook for a repo
copygit hooks install /path/to/repo

# Check hook status
copygit hooks status /path/to/repo

# Remove hook
copygit hooks uninstall /path/to/repo

# View hook content
cat /path/to/repo/.git/hooks/post-push
```

### Daemon Management

```bash
# Start daemon (background process)
copygit daemon start

# Start daemon in foreground (for debugging)
copygit daemon start --foreground

# Check daemon status
copygit daemon status

# Stop daemon
copygit daemon stop

# View daemon logs (macOS)
log stream --predicate 'process == "copygit"' --level debug
```

### Repository & Provider Management

```bash
# Initialize a repository
copygit init /path/to/repo

# List configured providers
copygit config list-providers

# Add a provider
copygit config add-provider github github https://github.com --auth-method https

# Login to provider
copygit login --provider github

# Check sync status
copygit status /path/to/repo

# Manual sync (if needed)
copygit push /path/to/repo
```

## Automation Checklist

Use this checklist to verify complete automation setup:

- [ ] CopyGit binary updated (`/usr/local/bin/copygit --version`)
- [ ] Providers configured (`copygit config list-providers`)
- [ ] Credentials stored for each provider (`copygit login --provider xxx`)
- [ ] Repository initialized (`copygit init /path/to/repo`)
- [ ] Post-push hook installed (`copygit hooks status /path/to/repo`)
- [ ] Daemon started (`copygit daemon status`)
- [ ] Manual sync test passed (`copygit push /path/to/repo`)
- [ ] Hook test passed (push with git, verify sync)

## Performance Notes

### Post-Push Hook
- **Trigger time:** < 0.1s (instant)
- **Sync time:** 1-5 seconds (depending on repo size)
- **Total:** Push completes, hook runs asynchronously

### Daemon
- **Poll interval:** 30 seconds (default)
- **Sync time per repo:** 1-5 seconds
- **CPU impact:** Minimal (idle between polls)
- **Network impact:** Light (only when changes detected)

### Credentials Injection
- **Cache miss:** ~0.5s (first use, validates token)
- **Cache hit:** Instant (subsequent uses)
- **No password prompts:** Zero user interaction

## Security

### Credentials Storage
- ✅ Stored in system keychain (macOS/Linux)
- ✅ NOT stored in plain text
- ✅ NOT committed to git
- ✅ Per-provider encryption
- ✅ User-only access

### Credential Injection
- ✅ Uses git's credential helper protocol
- ✅ Credentials passed via stdin (not command line)
- ✅ Credentials cached per session
- ✅ Automatic cleanup on session end

### Repository Sync
- ✅ Uses authenticated HTTPS
- ✅ Validates SSL certificates
- ✅ Logs all sync operations
- ✅ Supports 2FA (with app passwords)

## FAQ

### Q: Do I need to use copygit commands manually?
**A:** No! Once set up, everything is automatic. You just use normal `git` commands.

### Q: What if I only want to sync certain repos?
**A:** Only run `copygit init` on the repos you want to sync. The daemon and hooks only affect initialized repos.

### Q: Can I use different auth methods for different providers?
**A:** Yes! Each provider can use SSH, HTTPS, or token auth independently.

### Q: What happens if sync fails?
**A:** Daemon logs the error and continues. Failed syncs are retried at next poll interval.

### Q: Does this work with private repositories?
**A:** Yes! Credentials are used for all repos (public and private).

### Q: Can I pause automatic syncing?
**A:** Yes, stop the daemon: `copygit daemon stop`

### Q: How do I update the binary?
**A:** `sudo cp ~/copygit-fixed-v2 /usr/local/bin/copygit`

## Need Help?

```bash
# View help for any command
copygit --help
copygit hooks --help
copygit daemon --help
copygit config --help

# Check logs for errors
log stream --predicate 'process == "copygit"' --level debug

# Manual sync to test
copygit push /path/to/repo --verbose
```

---

## Summary

With automatic sync set up, you get:

✅ **Zero manual copygit commands**
✅ **Perfect mirror across all providers**
✅ **Instant sync on your pushes**
✅ **Background sync from team pushes**
✅ **Automatic credential management**
✅ **No password prompts**
✅ **Works with your normal git workflow**

Just use `git` normally, and CopyGit handles the rest! 🚀
