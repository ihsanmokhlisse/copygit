package model

import "time"

// OperationStatus tracks the lifecycle of a sync operation.
type OperationStatus string

const (
	StatusPending    OperationStatus = "pending"
	StatusInProgress OperationStatus = "in_progress"
	StatusCompleted  OperationStatus = "completed"
	StatusFailed     OperationStatus = "failed"
)

// OperationType defines what kind of sync operation this is.
type OperationType string

const (
	OpPush  OperationType = "push"
	OpFetch OperationType = "fetch"
)

// SyncOperation represents a single unit of sync work.
// Persisted to ~/.copygit/queue/ as JSON files when queued for retry.
type SyncOperation struct {
	ID           string          `json:"id"`
	ProviderName string          `json:"provider_name"`
	RepoPath     string          `json:"repo_path"`
	Type         OperationType   `json:"type"`
	Branch       string          `json:"branch"`
	Status       OperationStatus `json:"status"`
	RetryCount   int             `json:"retry_count"`
	MaxRetries   int             `json:"max_retries"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
	Error        string          `json:"error,omitempty"`
}

// RepoSyncTarget associates a repo with a provider.
type RepoSyncTarget struct {
	ProviderName string `toml:"provider"`   // References ProviderConfig.Name
	RemoteURL    string `toml:"remote_url"` // Repo-specific remote URL
	Enabled      bool   `toml:"enabled"`    // Whether sync is active
}

// RepoConfig is the per-repo configuration stored at <repo-root>/.copygit.toml.
type RepoConfig struct {
	Version     string           `toml:"version"`
	SyncTargets []RepoSyncTarget `toml:"sync_targets"`
}

// EnabledProviderNames returns the names of all enabled sync targets.
func (rc *RepoConfig) EnabledProviderNames() []string {
	var names []string
	for _, t := range rc.SyncTargets {
		if t.Enabled {
			names = append(names, t.ProviderName)
		}
	}
	return names
}

// RepoRegistration tracks a registered repo in the global registry.
type RepoRegistration struct {
	Path         string    `toml:"path"`
	Alias        string    `toml:"alias"`
	RegisteredAt time.Time `toml:"registered_at"`
	LastSyncTime time.Time `toml:"last_sync"`
}

// RepoRegistry holds all registered repos.
type RepoRegistry struct {
	Version string             `toml:"version"`
	Repos   []RepoRegistration `toml:"repos"`
}
