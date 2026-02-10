package git

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// --- RemoteManager tests ---

func TestRemoteManager_AddRemote(t *testing.T) {
	ctx := context.Background()
	var capturedArgs []string
	fakeGit := &FakeGitExecutor{
		RunFunc: func(_ context.Context, _ string, args ...string) (string, error) {
			capturedArgs = args
			return "", nil
		},
	}

	rm := NewRemoteManager(fakeGit, testLogger())
	err := rm.AddRemote(ctx, "/repo", "origin", "https://github.com/u/r.git")
	require.NoError(t, err)
	assert.Equal(t, []string{"remote", "add", "origin", "https://github.com/u/r.git"}, capturedArgs)
}

func TestRemoteManager_RemoveRemote(t *testing.T) {
	ctx := context.Background()
	fakeGit := &FakeGitExecutor{
		RunFunc: func(_ context.Context, _ string, _ ...string) (string, error) {
			return "", nil
		},
	}

	rm := NewRemoteManager(fakeGit, testLogger())
	err := rm.RemoveRemote(ctx, "/repo", "origin")
	require.NoError(t, err)
}

func TestRemoteManager_ListRemotes(t *testing.T) {
	ctx := context.Background()
	fakeGit := &FakeGitExecutor{
		RunFunc: func(_ context.Context, _ string, _ ...string) (string, error) {
			return "origin\thttps://github.com/u/r.git (fetch)\norigin\thttps://github.com/u/r.git (push)\nupstream\thttps://github.com/other/r.git (fetch)\nupstream\thttps://github.com/other/r.git (push)", nil
		},
	}

	rm := NewRemoteManager(fakeGit, testLogger())
	remotes, err := rm.ListRemotes(ctx, "/repo")
	require.NoError(t, err)
	assert.Len(t, remotes, 2)
	assert.Equal(t, "https://github.com/u/r.git", remotes["origin"])
	assert.Equal(t, "https://github.com/other/r.git", remotes["upstream"])
}

func TestRemoteManager_ListRemotes_Empty(t *testing.T) {
	ctx := context.Background()
	fakeGit := &FakeGitExecutor{
		RunFunc: func(_ context.Context, _ string, _ ...string) (string, error) {
			return "", nil
		},
	}

	rm := NewRemoteManager(fakeGit, testLogger())
	remotes, err := rm.ListRemotes(ctx, "/repo")
	require.NoError(t, err)
	assert.Empty(t, remotes)
}

func TestRemoteManager_HasRemote(t *testing.T) {
	ctx := context.Background()
	fakeGit := &FakeGitExecutor{
		RunFunc: func(_ context.Context, _ string, _ ...string) (string, error) {
			return "origin\thttps://example.com/r.git (push)", nil
		},
	}

	rm := NewRemoteManager(fakeGit, testLogger())
	assert.True(t, rm.HasRemote(ctx, "/repo", "origin"))
	assert.False(t, rm.HasRemote(ctx, "/repo", "nonexistent"))
}

func TestRemoteManager_Push(t *testing.T) {
	ctx := context.Background()
	var capturedArgs []string
	fakeGit := &FakeGitExecutor{
		RunFunc: func(_ context.Context, _ string, args ...string) (string, error) {
			capturedArgs = args
			return "", nil
		},
	}

	rm := NewRemoteManager(fakeGit, testLogger())

	t.Run("basic push", func(t *testing.T) {
		err := rm.Push(ctx, "/repo", "origin", []string{"main"}, false, false)
		require.NoError(t, err)
		assert.Equal(t, []string{"push", "origin", "main"}, capturedArgs)
	})

	t.Run("push with tags", func(t *testing.T) {
		err := rm.Push(ctx, "/repo", "origin", []string{"main"}, true, false)
		require.NoError(t, err)
		assert.Contains(t, capturedArgs, "--tags")
	})

	t.Run("force push", func(t *testing.T) {
		err := rm.Push(ctx, "/repo", "origin", []string{"main"}, false, true)
		require.NoError(t, err)
		assert.Contains(t, capturedArgs, "--force")
	})
}

func TestRemoteManager_Push_Error(t *testing.T) {
	ctx := context.Background()
	fakeGit := &FakeGitExecutor{
		RunFunc: func(_ context.Context, _ string, _ ...string) (string, error) {
			return "", &GitError{ExitCode: 128, Stderr: "rejected"}
		},
	}

	rm := NewRemoteManager(fakeGit, testLogger())
	err := rm.Push(ctx, "/repo", "origin", []string{"main"}, false, false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "push to origin")
}

func TestRemoteManager_Fetch(t *testing.T) {
	ctx := context.Background()
	fakeGit := &FakeGitExecutor{
		RunFunc: func(_ context.Context, _ string, _ ...string) (string, error) {
			return "", nil
		},
	}

	rm := NewRemoteManager(fakeGit, testLogger())
	err := rm.Fetch(ctx, "/repo", "origin")
	require.NoError(t, err)
}

func TestRemoteManager_GetRemoteHeadHash(t *testing.T) {
	ctx := context.Background()
	fakeGit := &FakeGitExecutor{
		RunFunc: func(_ context.Context, _ string, _ ...string) (string, error) {
			return "abc123def456", nil
		},
	}

	rm := NewRemoteManager(fakeGit, testLogger())
	hash, err := rm.GetRemoteHeadHash(ctx, "/repo", "origin", "main")
	require.NoError(t, err)
	assert.Equal(t, "abc123def456", hash)
}

// --- BranchManager tests ---

func TestBranchManager_CurrentBranch(t *testing.T) {
	ctx := context.Background()
	fakeGit := &FakeGitExecutor{
		RunFunc: func(_ context.Context, _ string, _ ...string) (string, error) {
			return "feature/my-branch", nil
		},
	}

	bm := NewBranchManager(fakeGit, testLogger())
	branch, err := bm.CurrentBranch(ctx, "/repo")
	require.NoError(t, err)
	assert.Equal(t, "feature/my-branch", branch)
}

func TestBranchManager_GetHeadHash(t *testing.T) {
	ctx := context.Background()
	fakeGit := &FakeGitExecutor{
		RunFunc: func(_ context.Context, _ string, _ ...string) (string, error) {
			return "deadbeef1234", nil
		},
	}

	bm := NewBranchManager(fakeGit, testLogger())
	hash, err := bm.GetHeadHash(ctx, "/repo")
	require.NoError(t, err)
	assert.Equal(t, "deadbeef1234", hash)
}

func TestBranchManager_ListBranches(t *testing.T) {
	ctx := context.Background()
	fakeGit := &FakeGitExecutor{
		RunFunc: func(_ context.Context, _ string, _ ...string) (string, error) {
			return "main\nfeature/a\nfeature/b", nil
		},
	}

	bm := NewBranchManager(fakeGit, testLogger())
	branches, err := bm.ListBranches(ctx, "/repo")
	require.NoError(t, err)
	assert.Equal(t, []string{"main", "feature/a", "feature/b"}, branches)
}

func TestBranchManager_ListBranches_Empty(t *testing.T) {
	ctx := context.Background()
	fakeGit := &FakeGitExecutor{
		RunFunc: func(_ context.Context, _ string, _ ...string) (string, error) {
			return "", nil
		},
	}

	bm := NewBranchManager(fakeGit, testLogger())
	branches, err := bm.ListBranches(ctx, "/repo")
	require.NoError(t, err)
	assert.Empty(t, branches)
}

func TestBranchManager_AheadBehind(t *testing.T) {
	ctx := context.Background()
	fakeGit := &FakeGitExecutor{
		RunFunc: func(_ context.Context, _ string, _ ...string) (string, error) {
			return "3\t2", nil
		},
	}

	bm := NewBranchManager(fakeGit, testLogger())
	ahead, behind, err := bm.AheadBehind(ctx, "/repo", "main", "origin/main")
	require.NoError(t, err)
	assert.Equal(t, 3, ahead)
	assert.Equal(t, 2, behind)
}

func TestBranchManager_AheadBehind_BadOutput(t *testing.T) {
	ctx := context.Background()
	fakeGit := &FakeGitExecutor{
		RunFunc: func(_ context.Context, _ string, _ ...string) (string, error) {
			return "garbage", nil
		},
	}

	bm := NewBranchManager(fakeGit, testLogger())
	_, _, err := bm.AheadBehind(ctx, "/repo", "main", "origin/main")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected rev-list output")
}

func TestBranchManager_CurrentBranch_Error(t *testing.T) {
	ctx := context.Background()
	fakeGit := &FakeGitExecutor{
		RunFunc: func(_ context.Context, _ string, _ ...string) (string, error) {
			return "", errors.New("detached HEAD")
		},
	}

	bm := NewBranchManager(fakeGit, testLogger())
	_, err := bm.CurrentBranch(ctx, "/repo")
	assert.Error(t, err)
}

// --- GitError tests ---

func TestGitError_Unwrap(t *testing.T) {
	err := &GitError{ExitCode: 1}
	inner := err.Unwrap()
	assert.Error(t, inner)
	assert.Contains(t, inner.Error(), "exit code 1")
}
