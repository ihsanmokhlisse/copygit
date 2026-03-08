# Adding Providers to CopyGit

## Quick Start

### Add a Provider

```bash
copygit config add-provider <name> <type> <base-url> [--auth-method <method>]
```

**Parameters:**
- `<name>` - Friendly name for this provider (e.g., "github", "my-gitlab")
- `<type>` - Provider type: `github` | `gitlab` | `gitea` | `generic`
- `<base-url>` - Provider base URL
- `--auth-method` - Authentication method: `ssh` | `https` | `token` (default: `https`)

### Login with Credentials

```bash
copygit login <provider-name>
```

---

## Step-by-Step Guide for Each Provider

### GitHub

#### 1. Add GitHub Provider

```bash
copygit config add-provider github github https://github.com --auth-method https
```

**Parameters:**
- `name`: `github` (or any friendly name)
- `type`: `github`
- `base-url`: `https://github.com`
- `auth-method`: `https` (token-based)

#### 2. Create GitHub Personal Access Token

Visit: https://github.com/settings/tokens/new

**Required Scopes:**
- ✅ `repo` - Full control of repositories
- ✅ `read:org` - Read organization members and teams
- ✅ `gist` - Create gists (optional, for extra functionality)
- ✅ `user` - Access user profile data

**Example token:**
```
ghp_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

#### 3. Login to CopyGit

```bash
copygit login github
```

**Prompt:**
```
Enter GitHub token:
```

Paste your token and press Enter. It will be stored securely in your system keychain.

#### 4. Verify

```bash
copygit config list-providers
```

Should show:
```
Provider: github
  Type: github
  Base URL: https://github.com
  Auth Method: https
```

---

### GitLab

#### 1. Add GitLab Provider

```bash
copygit config add-provider gitlab gitlab https://gitlab.com --auth-method https
```

**For Self-Hosted GitLab:**
```bash
copygit config add-provider my-gitlab gitlab https://gitlab.example.com --auth-method https
```

#### 2. Create GitLab Personal Access Token

Visit: https://gitlab.com/-/user_settings/personal_access_tokens

**Required Scopes:**
- ✅ `api` - Full API access
- ✅ `read_user` - Read user profile
- ✅ `read_repository` - Read repository data
- ✅ `write_repository` - Write to repositories (for push)

**Example token:**
```
glpat-xxxxxxxxxxxxxxx
```

#### 3. Login to CopyGit

```bash
copygit login gitlab
```

**Prompt:**
```
Enter GitLab token:
```

Paste your token.

#### 4. Verify

```bash
copygit config list-providers
```

---

### Gitea

#### 1. Add Gitea Provider

```bash
copygit config add-provider gitea gitea https://gitea.example.com --auth-method https
```

#### 2. Create Gitea Access Token

In Gitea UI:
1. Login to your Gitea instance
2. Go: Settings → Applications → Generate Token
3. Name: `copygit`
4. Scopes: `repo` (read/write repositories)

**Example token:**
```
xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
```

#### 3. Login to CopyGit

```bash
copygit login gitea
```

**Prompt:**
```
Enter Gitea token:
```

Paste your token.

#### 4. Verify

```bash
copygit config list-providers
```

---

## Complete Example: Setup All Three Providers

### Step 1: Add All Providers

```bash
# Add GitHub
copygit config add-provider github github https://github.com --auth-method https

# Add GitLab
copygit config add-provider gitlab gitlab https://gitlab.com --auth-method https

# Add Gitea
copygit config add-provider gitea gitea https://gitea.example.com --auth-method https
```

### Step 2: Create Tokens on Each Platform

**GitHub:**
1. Visit https://github.com/settings/tokens/new
2. Select scopes: `repo`, `read:org`, `gist`, `user`
3. Click "Generate token"
4. Copy the token

**GitLab:**
1. Visit https://gitlab.com/-/user_settings/personal_access_tokens
2. Select scopes: `api`, `read_user`, `read_repository`, `write_repository`
3. Click "Create personal access token"
4. Copy the token

**Gitea:**
1. Login and go to Settings → Applications
2. Click "Generate Token"
3. Name: `copygit`
4. Select scope: `repo`
5. Copy the token

### Step 3: Login to Each Provider

```bash
copygit login github
# Paste: ghp_xxxxx...

copygit login gitlab
# Paste: glpat-xxxxx...

copygit login gitea
# Paste: xxxxx...
```

### Step 4: Verify Setup

```bash
copygit config list-providers
```

Expected output:
```
Configured providers:
  github   (github)      https://github.com
  gitlab   (gitlab)      https://gitlab.com
  gitea    (gitea)       https://gitea.example.com
```

---

## Managing Providers

### List All Providers

```bash
copygit config list-providers
```

### Remove a Provider

```bash
copygit config remove-provider github
```

### Update a Provider

Currently, to update a provider, remove and re-add it:

```bash
copygit config remove-provider github
copygit config add-provider github github https://github.com --auth-method https
```

---

## Authentication Methods

### HTTPS (Token-Based)

```bash
copygit config add-provider github github https://github.com --auth-method https
copygit login github
# Paste personal access token
```

**When to use:**
- ✅ Most common and recommended
- ✅ Works with 2FA
- ✅ Easier credential management
- ✅ Secure storage in system keychain

### SSH

```bash
copygit config add-provider github github https://github.com --auth-method ssh
```

**When to use:**
- ✅ You have SSH keys already configured
- ✅ You prefer key-based authentication
- ✅ Corporate security requirements
- ❌ No manual login needed (uses git SSH config)

### Token (Explicit)

```bash
copygit config add-provider github github https://github.com --auth-method token
copygit login github
# Paste token
```

**When to use:**
- ✅ Same as HTTPS but explicit naming
- ✅ Preferred for API-heavy operations

---

## Token Scopes by Provider

### GitHub Scopes

| Scope | Purpose | Required? |
|-------|---------|-----------|
| `repo` | Full control of repositories | ✅ Yes |
| `read:org` | Read organization data | ✅ Yes |
| `gist` | Create and manage gists | ❌ Optional |
| `user` | Access user profile | ❌ Optional |

### GitLab Scopes

| Scope | Purpose | Required? |
|-------|---------|-----------|
| `api` | Full API access | ✅ Yes |
| `read_user` | Read user data | ✅ Yes |
| `read_repository` | Read repository data | ✅ Yes |
| `write_repository` | Write to repositories | ✅ Yes |

### Gitea Scopes

| Scope | Purpose | Required? |
|-------|---------|-----------|
| `repo` | Read/write repositories | ✅ Yes |

---

## Troubleshooting

### Token Not Working

**Error:** "Authentication failed" or "Invalid credentials"

**Solutions:**
1. Verify token hasn't expired
2. Check token has required scopes
3. Re-login to refresh credentials:
   ```bash
   copygit login <provider-name>
   ```

### Provider Not Found

**Error:** "Provider not found"

**Solutions:**
```bash
# Check configured providers
copygit config list-providers

# Add the provider if missing
copygit config add-provider <name> <type> <url> --auth-method https
```

### Credentials Stored Incorrectly

**Solution:** Remove and re-add the provider:
```bash
copygit config remove-provider <name>
copygit config add-provider <name> <type> <url> --auth-method https
copygit login <name>
```

### Multiple Accounts

**For multiple GitHub accounts:**

```bash
# Add personal account
copygit config add-provider github-personal github https://github.com --auth-method https
copygit login github-personal

# Add work account
copygit config add-provider github-work github https://github.com --auth-method https
copygit login github-work
```

Then use different names in `.copygit.toml`:
```toml
[[sync_targets]]
provider = 'github-personal'
remote_url = 'https://github.com/personal-user/repo.git'

[[sync_targets]]
provider = 'github-work'
remote_url = 'https://github.com/work-user/repo.git'
```

---

## Security Best Practices

### ✅ DO

- ✅ Use personal access tokens (limited scope)
- ✅ Set token expiration dates
- ✅ Rotate tokens regularly
- ✅ Use HTTPS with token auth (most common)
- ✅ Store tokens in system keychain (CopyGit does this)
- ✅ Use different tokens for different machines

### ❌ DO NOT

- ❌ Share tokens with others
- ❌ Commit tokens to git repositories
- ❌ Use your main user password
- ❌ Set unlimited token scope
- ❌ Use tokens in command-line history
- ❌ Store tokens in plain text files

---

## Next Steps

After adding providers:

1. **Initialize a repository:**
   ```bash
   copygit init /path/to/repo
   ```

2. **Check status:**
   ```bash
   copygit status
   ```

3. **Push to all providers:**
   ```bash
   copygit push
   ```

4. **See configuration guide:**
   - Read [CONFIGURATION.md](./CONFIGURATION.md) for detailed setup
   - Read [USAGE.md](./USAGE.md) for daily workflows

---

## Support

For more help:
```bash
copygit config --help
copygit login --help
copygit init --help
```
