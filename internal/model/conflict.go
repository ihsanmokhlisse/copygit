package model

// ConflictType describes the nature of a divergence between local and remote.
type ConflictType string

const (
	ConflictNone        ConflictType = "none"
	ConflictFastForward ConflictType = "fast_forward"
	ConflictDiverged    ConflictType = "diverged"
)

// ConflictInfo captures details when a provider's remote HEAD differs
// from the local HEAD unexpectedly.
type ConflictInfo struct {
	ProviderName string       `json:"provider_name"`
	Type         ConflictType `json:"type"`
	LocalHead    string       `json:"local_head"`
	RemoteHead   string       `json:"remote_head"`
	AheadBy      int          `json:"ahead_by"`
	BehindBy     int          `json:"behind_by"`
}

// Resolution describes how a conflict should be handled.
type Resolution string

const (
	ResolutionAbort     Resolution = "abort"
	ResolutionMerge     Resolution = "merge"
	ResolutionForcePush Resolution = "force_push"
)
