package model

import "errors"

// Domain errors — all sentinel errors for the application.
// Callers MUST match with errors.Is(). Wrapping with %w preserves the chain.
var (
	// Provider errors
	ErrProviderUnreachable = errors.New("provider is unreachable")
	ErrProviderNotFound    = errors.New("provider not found in configuration")
	ErrAuthFailed          = errors.New("authentication failed")
	ErrCredentialNotFound  = errors.New("credential not found for provider")
	ErrNoProviders         = errors.New("no providers configured")

	// Config errors
	ErrConfigNotFound     = errors.New("config file not found")
	ErrConfigInvalid      = errors.New("config file has invalid syntax")
	ErrRepoConfigNotFound = errors.New("per-repo .copygit.toml not found")

	// Sync errors
	ErrConflictDetected  = errors.New("provider has diverged from local")
	ErrLockAcquireFailed = errors.New("failed to acquire sync lock")
	ErrSyncInProgress    = errors.New("another sync operation is in progress")

	// Repository errors
	ErrEmptyRepository       = errors.New("repository has no commits")
	ErrRemoteRepoMissing     = errors.New("remote repository does not exist")
	ErrNotGitRepo            = errors.New("not a git repository")
	ErrRepoNotRegistered     = errors.New("repository is not registered")
	ErrRepoAlreadyRegistered = errors.New("repository is already registered")
	ErrRepoPathMissing       = errors.New("registered repo path no longer exists")

	// Aliases for backward compatibility and naming consistency
	ErrNotAGitRepo  = ErrNotGitRepo
	ErrRepoNotFound = ErrRepoNotRegistered
)
