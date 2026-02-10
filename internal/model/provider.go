package model

// AuthMethod defines how CopyGit authenticates with a provider.
type AuthMethod string

const (
	AuthSSH   AuthMethod = "ssh"
	AuthHTTPS AuthMethod = "https"
	AuthToken AuthMethod = "token"
)

// ProviderType identifies a git hosting provider.
type ProviderType string

const (
	ProviderGitHub  ProviderType = "github"
	ProviderGitLab  ProviderType = "gitlab"
	ProviderGitea   ProviderType = "gitea"
	ProviderGeneric ProviderType = "generic"
)

// ProviderConfig holds the configuration for a single provider.
// Provider configs are GLOBAL — shared across all repos.
type ProviderConfig struct {
	Name        string       `toml:"name"`        // User-assigned name (e.g., "work-github")
	Type        ProviderType `toml:"type"`        // github | gitlab | gitea | generic
	BaseURL     string       `toml:"base_url"`    // e.g., "https://github.com"
	AuthMethod  AuthMethod   `toml:"auth_method"` // ssh | https | token
	IsPreferred bool         `toml:"preferred"`   // Soft primary for conflict resolution
}

// SyncTarget associates a local repository with a remote provider.
type SyncTarget struct {
	Provider     ProviderConfig
	RemoteName   string // Git remote name (e.g., "copygit-github")
	LastSyncTime string // Last successful sync timestamp (ISO 8601)
	LastSyncHash string // Commit hash at last sync
	InSync       bool   // Whether remote matches local
}
