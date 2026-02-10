package sync

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/imokhlis/copygit/internal/git"
	"github.com/imokhlis/copygit/internal/model"
	"github.com/imokhlis/copygit/internal/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestOrchestrator_Push_AllSuccess(t *testing.T) {
	ctx := context.Background()
	fakeGit := &git.FakeGitExecutor{
		RunFunc: func(_ context.Context, _ string, args ...string) (string, error) {
			if len(args) > 0 {
				switch args[0] {
				case "rev-parse":
					if len(args) > 1 && args[1] == "--abbrev-ref" {
						return "main", nil
					}
					return "abc123", nil
				case "remote":
					return "", nil
				case "push":
					return "", nil
				}
			}
			return "", nil
		},
		IsGitRepoFunc: func(_ context.Context, _ string) bool {
			return true
		},
	}

	orch := NewOrchestrator(fakeGit, testLogger())
	providers := map[string]provider.Provider{
		"gh": &provider.FakeProvider{NameValue: "gh", TypeValue: model.ProviderGitHub},
		"gl": &provider.FakeProvider{NameValue: "gl", TypeValue: model.ProviderGitLab},
	}
	targets := []model.RepoSyncTarget{
		{ProviderName: "gh", RemoteURL: "git@github.com:u/r.git", Enabled: true},
		{ProviderName: "gl", RemoteURL: "https://gl.com/u/r.git", Enabled: true},
	}

	report, err := orch.Push(ctx, "/repo", providers, targets)
	require.NoError(t, err)
	assert.Equal(t, 2, report.TotalTargets)
	assert.Equal(t, 2, report.SuccessCount)
	assert.Equal(t, 0, report.FailureCount)
	assert.Equal(t, model.OpPush, report.OperationType)
	assert.Len(t, report.Operations, 2)
	assert.True(t, report.DurationSeconds >= 0)
}

func TestOrchestrator_Push_OneFails(t *testing.T) {
	ctx := context.Background()
	pushCount := 0
	fakeGit := &git.FakeGitExecutor{
		RunFunc: func(_ context.Context, _ string, args ...string) (string, error) {
			if len(args) > 0 && args[0] == "push" {
				pushCount++
				if pushCount == 2 {
					return "", &git.GitError{ExitCode: 128, Stderr: "rejected"}
				}
				return "", nil
			}
			if len(args) > 1 && args[1] == "--abbrev-ref" {
				return "main", nil
			}
			if len(args) > 0 && args[0] == "remote" {
				return "", nil
			}
			return "abc123", nil
		},
	}

	orch := NewOrchestrator(fakeGit, testLogger())
	providers := map[string]provider.Provider{
		"gh": &provider.FakeProvider{NameValue: "gh"},
		"gl": &provider.FakeProvider{NameValue: "gl"},
	}
	targets := []model.RepoSyncTarget{
		{ProviderName: "gh", RemoteURL: "url1", Enabled: true},
		{ProviderName: "gl", RemoteURL: "url2", Enabled: true},
	}

	report, err := orch.Push(ctx, "/repo", providers, targets)
	require.NoError(t, err)
	assert.Equal(t, 1, report.SuccessCount)
	assert.Equal(t, 1, report.FailureCount)
}

func TestOrchestrator_Push_DisabledTargetSkipped(t *testing.T) {
	ctx := context.Background()
	fakeGit := &git.FakeGitExecutor{
		RunFunc: func(_ context.Context, _ string, args ...string) (string, error) {
			if len(args) > 1 && args[1] == "--abbrev-ref" {
				return "main", nil
			}
			return "", nil
		},
	}

	orch := NewOrchestrator(fakeGit, testLogger())
	targets := []model.RepoSyncTarget{
		{ProviderName: "gh", Enabled: false},
	}

	report, err := orch.Push(ctx, "/repo", map[string]provider.Provider{}, targets)
	require.NoError(t, err)
	assert.Equal(t, 0, report.SuccessCount)
	assert.Equal(t, 0, report.FailureCount)
	assert.Empty(t, report.Operations)
}

func TestOrchestrator_Push_ProviderNotFound(t *testing.T) {
	ctx := context.Background()
	fakeGit := &git.FakeGitExecutor{
		RunFunc: func(_ context.Context, _ string, args ...string) (string, error) {
			if len(args) > 1 && args[1] == "--abbrev-ref" {
				return "main", nil
			}
			return "", nil
		},
	}

	orch := NewOrchestrator(fakeGit, testLogger())
	targets := []model.RepoSyncTarget{
		{ProviderName: "nonexistent", Enabled: true},
	}

	report, err := orch.Push(ctx, "/repo", map[string]provider.Provider{}, targets)
	require.NoError(t, err)
	assert.Equal(t, 0, report.SuccessCount)
	assert.Equal(t, 1, report.FailureCount, "missing provider should count as failure")
}

func TestOrchestrator_Status_AllSynced(t *testing.T) {
	ctx := context.Background()
	fakeGit := &git.FakeGitExecutor{
		RunFunc: func(_ context.Context, _ string, args ...string) (string, error) {
			if len(args) > 1 && args[1] == "--abbrev-ref" {
				return "main", nil
			}
			return "abc123def456", nil
		},
	}

	orch := NewOrchestrator(fakeGit, testLogger())
	providers := map[string]provider.Provider{
		"gh": &provider.FakeProvider{NameValue: "gh", TypeValue: model.ProviderGitHub},
	}
	targets := []model.RepoSyncTarget{
		{ProviderName: "gh", Enabled: true},
	}

	report, err := orch.Status(ctx, "/repo", providers, targets)
	require.NoError(t, err)
	assert.Equal(t, "abc123def456", report.LocalHead)
	assert.Equal(t, "main", report.LocalBranch)
	require.Len(t, report.Providers, 1)
	assert.True(t, report.Providers[0].InSync)
}

func TestOrchestrator_Status_OutOfSync(t *testing.T) {
	ctx := context.Background()
	callCount := 0
	fakeGit := &git.FakeGitExecutor{
		RunFunc: func(_ context.Context, _ string, args ...string) (string, error) {
			callCount++
			if len(args) > 1 && args[1] == "--abbrev-ref" {
				return "main", nil
			}
			// First rev-parse = local head, second = remote head
			if callCount <= 2 {
				return "local-hash", nil
			}
			return "remote-hash", nil
		},
	}

	orch := NewOrchestrator(fakeGit, testLogger())
	providers := map[string]provider.Provider{
		"gh": &provider.FakeProvider{NameValue: "gh", TypeValue: model.ProviderGitHub},
	}
	targets := []model.RepoSyncTarget{
		{ProviderName: "gh", Enabled: true},
	}

	report, err := orch.Status(ctx, "/repo", providers, targets)
	require.NoError(t, err)
	require.Len(t, report.Providers, 1)
	assert.False(t, report.Providers[0].InSync)
}

func TestOrchestrator_Fetch_AllSuccess(t *testing.T) {
	ctx := context.Background()
	fakeGit := &git.FakeGitExecutor{
		RunFunc: func(_ context.Context, _ string, _ ...string) (string, error) {
			return "", nil
		},
	}

	orch := NewOrchestrator(fakeGit, testLogger())
	providers := map[string]provider.Provider{
		"gh": &provider.FakeProvider{NameValue: "gh"},
	}
	targets := []model.RepoSyncTarget{
		{ProviderName: "gh", Enabled: true},
	}

	report, err := orch.Fetch(ctx, "/repo", providers, targets)
	require.NoError(t, err)
	assert.Equal(t, model.OpFetch, report.OperationType)
	assert.Equal(t, 1, report.SuccessCount)
	assert.Equal(t, 0, report.FailureCount)
}

func TestOrchestrator_Fetch_DisabledSkipped(t *testing.T) {
	ctx := context.Background()
	fakeGit := &git.FakeGitExecutor{
		RunFunc: func(_ context.Context, _ string, _ ...string) (string, error) {
			return "", nil
		},
	}

	orch := NewOrchestrator(fakeGit, testLogger())
	targets := []model.RepoSyncTarget{
		{ProviderName: "gh", Enabled: false},
	}

	report, err := orch.Fetch(ctx, "/repo", map[string]provider.Provider{}, targets)
	require.NoError(t, err)
	assert.Empty(t, report.Operations)
}
