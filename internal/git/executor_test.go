package git

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecGit_IsGitRepo(t *testing.T) {
	fake := &FakeGitExecutor{
		IsGitRepoFunc: func(_ context.Context, path string) bool {
			return path == "/valid/repo"
		},
	}

	assert.True(t, fake.IsGitRepo(context.Background(), "/valid/repo"))
	assert.False(t, fake.IsGitRepo(context.Background(), "/not/a/repo"))
}

func TestExecGit_Run(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		err     error
		wantErr bool
	}{
		{
			name:   "successful command",
			output: "abc123",
			err:    nil,
		},
		{
			name:    "failed command",
			output:  "",
			err:     &GitError{Command: "git", Args: []string{"push"}, ExitCode: 1, Stderr: "rejected"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := &FakeGitExecutor{
				RunFunc: func(_ context.Context, _ string, _ ...string) (string, error) {
					return tt.output, tt.err
				},
			}

			output, err := fake.Run(context.Background(), "/repo", "rev-parse", "HEAD")
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.output, output)
			}
		})
	}
}

func TestFakeGitExecutor_Defaults(t *testing.T) {
	fake := &FakeGitExecutor{}

	// Default Run returns empty string, nil
	out, err := fake.Run(context.Background(), "/repo", "status")
	assert.Equal(t, "", out)
	assert.NoError(t, err)

	// Default IsGitRepo returns true
	assert.True(t, fake.IsGitRepo(context.Background(), "/any"))

	// Default RunWithStdin returns empty string, nil
	out, err = fake.RunWithStdin(context.Background(), "/repo", "input", "credential", "fill")
	assert.Equal(t, "", out)
	assert.NoError(t, err)
}

func TestGitError_Error(t *testing.T) {
	err := &GitError{
		Command:  "git",
		Args:     []string{"push", "origin"},
		ExitCode: 128,
		Stderr:   "fatal: remote rejected",
	}

	assert.Contains(t, err.Error(), "push")
	assert.Contains(t, err.Error(), "128")
	assert.Contains(t, err.Error(), "fatal: remote rejected")
}
