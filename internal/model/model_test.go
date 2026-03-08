package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRepoConfig_EnabledProviderNames(t *testing.T) {
	tests := []struct {
		name    string
		targets []RepoSyncTargetWithOverrides
		want    []string
	}{
		{
			name: "all enabled",
			targets: []RepoSyncTargetWithOverrides{
				{ProviderName: "gh", Enabled: true},
				{ProviderName: "gl", Enabled: true},
			},
			want: []string{"gh", "gl"},
		},
		{
			name: "mixed",
			targets: []RepoSyncTargetWithOverrides{
				{ProviderName: "gh", Enabled: true},
				{ProviderName: "gl", Enabled: false},
			},
			want: []string{"gh"},
		},
		{
			name:    "none",
			targets: []RepoSyncTargetWithOverrides{},
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &RepoConfig{SyncTargets: tt.targets}
			got := cfg.EnabledProviderNames()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestOperationStatus_Constants(t *testing.T) {
	assert.Equal(t, OperationStatus("pending"), StatusPending)
	assert.Equal(t, OperationStatus("in_progress"), StatusInProgress)
	assert.Equal(t, OperationStatus("completed"), StatusCompleted)
	assert.Equal(t, OperationStatus("failed"), StatusFailed)
}

func TestProviderType_Constants(t *testing.T) {
	assert.Equal(t, ProviderType("github"), ProviderGitHub)
	assert.Equal(t, ProviderType("gitlab"), ProviderGitLab)
	assert.Equal(t, ProviderType("gitea"), ProviderGitea)
	assert.Equal(t, ProviderType("generic"), ProviderGeneric)
}

func TestAuthMethod_Constants(t *testing.T) {
	assert.Equal(t, AuthMethod("ssh"), AuthSSH)
	assert.Equal(t, AuthMethod("https"), AuthHTTPS)
	assert.Equal(t, AuthMethod("token"), AuthToken)
}

func TestConflictType_Constants(t *testing.T) {
	assert.Equal(t, ConflictType("none"), ConflictNone)
	assert.Equal(t, ConflictType("fast_forward"), ConflictFastForward)
	assert.Equal(t, ConflictType("diverged"), ConflictDiverged)
}

func TestSentinelErrors(t *testing.T) {
	// Verify all sentinel errors are distinct and non-nil
	errors := []error{
		ErrProviderUnreachable,
		ErrProviderNotFound,
		ErrAuthFailed,
		ErrCredentialNotFound,
		ErrNoProviders,
		ErrConfigNotFound,
		ErrConfigInvalid,
		ErrRepoConfigNotFound,
		ErrConflictDetected,
		ErrLockAcquireFailed,
		ErrSyncInProgress,
		ErrEmptyRepository,
		ErrRemoteRepoMissing,
		ErrNotGitRepo,
		ErrRepoNotRegistered,
		ErrRepoAlreadyRegistered,
		ErrRepoPathMissing,
	}

	for i, err := range errors {
		assert.NotNil(t, err, "error at index %d should not be nil", i)
		for j, other := range errors {
			if i != j {
				assert.NotEqual(t, err.Error(), other.Error(),
					"errors at index %d and %d should have different messages", i, j)
			}
		}
	}
}

func TestErrorAliases(t *testing.T) {
	assert.Equal(t, ErrNotGitRepo, ErrNotAGitRepo)
	assert.Equal(t, ErrRepoNotRegistered, ErrRepoNotFound)
}
