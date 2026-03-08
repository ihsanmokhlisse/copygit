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

// RepoConfigMetadata defines global metadata inheritance and defaults
type RepoConfigMetadata struct {
	InheritFrom   string `toml:"inherit_from"`   // "github", "gitlab", "gitea", or "none"
	Visibility    string `toml:"visibility"`     // Default: "private"
	Description   string `toml:"description"`    // Empty = inherit from source
	Homepage      string `toml:"homepage"`
	Topics        []string `toml:"topics"`
	Language      string `toml:"language"`
	License       string `toml:"license"`
	WikiEnabled   *bool  `toml:"wiki_enabled"`
	IssuesEnabled *bool  `toml:"issues_enabled"`
	Archived      *bool  `toml:"archived"`
}

// RepoSyncTargetWithOverrides extends RepoSyncTarget with per-target overrides
type RepoSyncTargetWithOverrides struct {
	ProviderName string              `toml:"provider"`
	RemoteURL    string              `toml:"remote_url"`
	Enabled      bool                `toml:"enabled"`
	Overrides    *MetadataOverrides   `toml:"overrides,omitempty"`
}

// ToRepoSyncTarget converts to the base RepoSyncTarget (backward compatible)
func (r *RepoSyncTargetWithOverrides) ToRepoSyncTarget() RepoSyncTarget {
	return RepoSyncTarget{
		ProviderName: r.ProviderName,
		RemoteURL:    r.RemoteURL,
		Enabled:      r.Enabled,
	}
}

// RepoConfig is the per-repo configuration stored at <repo-root>/.copygit.toml.
type RepoConfig struct {
	Version     string                         `toml:"version"`
	Metadata    *RepoConfigMetadata            `toml:"metadata,omitempty"`
	SyncTargets []RepoSyncTargetWithOverrides `toml:"sync_targets"`
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

// AsRepoSyncTargets converts to base RepoSyncTarget slice (backward compatible)
func (rc *RepoConfig) AsRepoSyncTargets() []RepoSyncTarget {
	targets := make([]RepoSyncTarget, len(rc.SyncTargets))
	for i, t := range rc.SyncTargets {
		targets[i] = t.ToRepoSyncTarget()
	}
	return targets
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

// Visibility defines repository visibility level.
type Visibility string

const (
	VisibilityPublic   Visibility = "public"
	VisibilityPrivate  Visibility = "private"
	VisibilityInternal Visibility = "internal" // GitLab only
)

// RepoMetadata is a normalized representation of repository metadata
// that abstracts differences between GitHub, GitLab, and Gitea.
type RepoMetadata struct {
	Visibility     Visibility `json:"visibility"`
	Description    string     `json:"description"`
	Homepage       string     `json:"homepage"`
	Topics         []string   `json:"topics"`
	Language       string     `json:"language"`          // GitHub only
	License        string     `json:"license"`           // SPDX identifier
	WikiEnabled    bool       `json:"wiki_enabled"`
	IssuesEnabled  bool       `json:"issues_enabled"`
	Archived       bool       `json:"archived"`
	SourceProvider string     `json:"source_provider,omitempty"`
	FetchedAt      time.Time  `json:"fetched_at,omitempty"`
}

// MetadataOverrides allows per-target overrides of inherited metadata
type MetadataOverrides struct {
	Visibility    *Visibility `toml:"visibility,omitempty"`
	Description   *string     `toml:"description,omitempty"`
	Homepage      *string     `toml:"homepage,omitempty"`
	Topics        []string    `toml:"topics,omitempty"`
	WikiEnabled   *bool       `toml:"wiki_enabled,omitempty"`
	IssuesEnabled *bool       `toml:"issues_enabled,omitempty"`
	Archived      *bool       `toml:"archived,omitempty"`
}

// Apply merges overrides into base metadata (overrides take precedence)
func (m *RepoMetadata) ApplyOverrides(overrides *MetadataOverrides) {
	if overrides == nil {
		return
	}
	if overrides.Visibility != nil {
		m.Visibility = *overrides.Visibility
	}
	if overrides.Description != nil {
		m.Description = *overrides.Description
	}
	if overrides.Homepage != nil {
		m.Homepage = *overrides.Homepage
	}
	if len(overrides.Topics) > 0 {
		m.Topics = overrides.Topics
	}
	if overrides.WikiEnabled != nil {
		m.WikiEnabled = *overrides.WikiEnabled
	}
	if overrides.IssuesEnabled != nil {
		m.IssuesEnabled = *overrides.IssuesEnabled
	}
	if overrides.Archived != nil {
		m.Archived = *overrides.Archived
	}
}

// DefaultMetadata returns safe defaults for new repos (private, no description)
func DefaultMetadata() *RepoMetadata {
	return &RepoMetadata{
		Visibility:    VisibilityPrivate,
		Description:   "",
		Homepage:      "",
		Topics:        []string{},
		Language:      "",
		License:       "",
		WikiEnabled:   true,
		IssuesEnabled: true,
		Archived:      false,
	}
}
