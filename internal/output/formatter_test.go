package output

import (
	"bytes"
	"testing"
	"time"

	"github.com/imokhlis/copygit/internal/model"
	"github.com/imokhlis/copygit/internal/sync"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTextFormatter_PrintSyncReport(t *testing.T) {
	var buf bytes.Buffer
	f := NewTextFormatter(&buf)

	report := &sync.SyncReport{
		OperationType:   model.OpPush,
		RepoPath:        "/path/to/repo",
		TotalTargets:    2,
		SuccessCount:    1,
		FailureCount:    1,
		DurationSeconds: 3.5,
		Operations: []model.SyncOperation{
			{ProviderName: "github", Status: model.StatusCompleted},
			{ProviderName: "gitlab", Status: model.StatusFailed, Error: "timeout"},
		},
	}

	err := f.PrintSyncReport(report)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "push")
	assert.Contains(t, output, "/path/to/repo")
	assert.Contains(t, output, "3.50 seconds")
	assert.Contains(t, output, "1 success")
	assert.Contains(t, output, "github")
	assert.Contains(t, output, "timeout")
}

func TestTextFormatter_PrintStatusReport(t *testing.T) {
	var buf bytes.Buffer
	f := NewTextFormatter(&buf)

	report := &model.StatusReport{
		RepoPath:    "/path/to/repo",
		LocalHead:   "abc123def456",
		LocalBranch: "main",
		Providers: []model.ProviderStatus{
			{Name: "gh", Type: "github", RemoteHead: "abc123def456", InSync: true},
			{Name: "gl", Type: "gitlab", RemoteHead: "xyz789", InSync: false},
		},
	}

	err := f.PrintStatusReport(report)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "/path/to/repo")
	assert.Contains(t, output, "main")
	assert.Contains(t, output, "gh")
	assert.Contains(t, output, "yes")
	assert.Contains(t, output, "gl")
	assert.Contains(t, output, "no")
}

func TestTextFormatter_PrintRepoList(t *testing.T) {
	var buf bytes.Buffer
	f := NewTextFormatter(&buf)

	repos := []model.RepoRegistration{
		{Path: "/path/repo1", Alias: "repo1", LastSyncTime: time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)},
		{Path: "/path/repo2", Alias: ""},
	}

	err := f.PrintRepoList(repos)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "/path/repo1")
	assert.Contains(t, output, "repo1")
	assert.Contains(t, output, "/path/repo2")
	assert.Contains(t, output, "never")
}

func TestTextFormatter_PrintRepoList_Empty(t *testing.T) {
	var buf bytes.Buffer
	f := NewTextFormatter(&buf)

	err := f.PrintRepoList(nil)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), "No repositories registered")
}

func TestJSONFormatter_PrintSyncReport(t *testing.T) {
	var buf bytes.Buffer
	f := NewJSONFormatter(&buf)

	report := &sync.SyncReport{
		OperationType: model.OpPush,
		RepoPath:      "/repo",
		TotalTargets:  1,
		SuccessCount:  1,
	}

	err := f.PrintSyncReport(report)
	require.NoError(t, err)
	assert.Contains(t, buf.String(), `"operation_type"`)
	assert.Contains(t, buf.String(), `"push"`)
}

func TestJSONFormatter_PrintError(t *testing.T) {
	var buf bytes.Buffer
	f := NewJSONFormatter(&buf)

	f.PrintError(assert.AnError)
	assert.Contains(t, buf.String(), `"error"`)
}

func TestFakeFormatter_Captures(t *testing.T) {
	f := &FakeFormatter{}

	_ = f.PrintSyncReport(&sync.SyncReport{RepoPath: "/repo"})
	f.PrintError(assert.AnError)
	f.PrintSuccess("done")
	f.PrintWarning("watch out")

	assert.Len(t, f.SyncReports, 1)
	assert.Len(t, f.Errors, 1)
	assert.Len(t, f.Successes, 1)
	assert.Len(t, f.Warnings, 1)
}
