# CopyGit Automatic Sync - Quick Start (5 Minutes)

## TL;DR Setup

Get automatic multi-provider sync working in 5 minutes:

```bash
# 1. Update binary (1 minute)
sudo cp ~/copygit-fixed-v2 /usr/local/bin/copygit

# 2. Configure providers (1 minute)
copygit config add-provider github github https://github.com --auth-method https
copygit config add-provider gitlab gitlab https://gitlab.com --auth-method https

# 3. Login (2 minutes)
copygit login --provider github
copygit login --provider gitlab

# 4. Initialize repo (30 seconds)
copygit init /path/to/repo

# 5. Install hook & start daemon (30 seconds)
copygit hooks install /path/to/repo
copygit daemon start

# Done! ✅
```

## Your New Workflow

```bash
# Just use git normally
cd /path/to/repo
git commit -m "new feature"
git push origin master

# 🎉 Automatically syncs to all providers!
# No copygit commands needed!
```

## Verification

```bash
# Verify setup
copygit hooks status /path/to/repo
# Output: post-push: installed (CopyGit)

copygit daemon status
# Output: Daemon is running (PID: xxxx)

copygit status /path/to/repo
# Output: All providers IN SYNC: yes
```

## What Just Happened?

| Mechanism | When | What | Sync Time |
|-----------|------|------|-----------|
| **Post-Push Hook** | After `git push` | Auto-syncs to other providers | 1-5s |
| **Daemon** | Every 30s | Syncs changes from any provider | Background |

## Common Commands

```bash
# View status of a repo
copygit status /path/to/repo

# Check hook status
copygit hooks status /path/to/repo

# Check daemon status
copygit daemon status

# Stop daemon if needed
copygit daemon stop

# View daemon logs (macOS)
log stream --predicate 'process == "copygit"'
```

## Troubleshooting

**Issue: Binary not found**
```bash
sudo cp ~/copygit-fixed-v2 /usr/local/bin/copygit
which copygit  # Should show /usr/local/bin/copygit
```

**Issue: Hook not triggering**
```bash
# Reinstall hook
copygit hooks uninstall /path/to/repo
copygit hooks install /path/to/repo

# Test manually
copygit push --from-hook origin /path/to/repo
```

**Issue: Daemon not syncing**
```bash
# Restart daemon
copygit daemon stop
sleep 2
copygit daemon start

# Check logs
log stream --predicate 'process == "copygit"' --level debug
```

## Next Steps

- Read [AUTOMATIC_SYNC_SETUP.md](./AUTOMATIC_SYNC_SETUP.md) for detailed configuration
- See [USAGE.md](./USAGE.md) for all available commands
- Check [ADDING_PROVIDERS.md](./ADDING_PROVIDERS.md) for provider-specific setup

---

**That's it!** Your repositories now sync automatically across all providers. Just use git normally! 🚀
