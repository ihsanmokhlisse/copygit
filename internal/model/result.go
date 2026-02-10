package model

// ProviderResult captures the outcome of a single provider operation.
type ProviderResult struct {
	ProviderName string          `json:"provider_name"`
	Type         OperationType   `json:"type"`
	Status       OperationStatus `json:"status"`
	DurationMs   int64           `json:"duration_ms"`
	Error        string          `json:"error,omitempty"`
	Refs         []string        `json:"refs,omitempty"`
}

// ProviderStatus captures the sync state of a single provider.
type ProviderStatus struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	RemoteHead   string `json:"remote_head"`
	InSync       bool   `json:"in_sync"`
	LastSyncTime string `json:"last_sync_time,omitempty"`
	Error        string `json:"error,omitempty"`
}

// StatusReport aggregates status across all providers for a repo.
type StatusReport struct {
	RepoPath    string           `json:"repo_path"`
	LocalHead   string           `json:"local_head"`
	LocalBranch string           `json:"local_branch"`
	Providers   []ProviderStatus `json:"providers"`
	QueuedOps   int              `json:"queued_ops"`
}
