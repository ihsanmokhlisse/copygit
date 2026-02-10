package git

import (
	"context"
)

// FakeGitExecutor is a test double for GitExecutor.
// Set RunFunc, RunWithStdinFunc, IsGitRepoFunc to customize per-test behavior.
type FakeGitExecutor struct {
	RunFunc          func(ctx context.Context, repoPath string, args ...string) (string, error)
	RunWithStdinFunc func(ctx context.Context, repoPath, stdin string, args ...string) (string, error)
	IsGitRepoFunc    func(ctx context.Context, path string) bool

	// Track calls for assertions
	RunCalls          [][]string
	RunWithStdinCalls []string
	IsGitRepoCalls    []string
}

func (f *FakeGitExecutor) Run(ctx context.Context, repoPath string, args ...string) (string, error) {
	if f.RunFunc != nil {
		return f.RunFunc(ctx, repoPath, args...)
	}
	return "", nil
}

func (f *FakeGitExecutor) RunWithStdin(ctx context.Context, repoPath, stdin string, args ...string) (string, error) {
	if f.RunWithStdinFunc != nil {
		return f.RunWithStdinFunc(ctx, repoPath, stdin, args...)
	}
	return "", nil
}

func (f *FakeGitExecutor) IsGitRepo(ctx context.Context, path string) bool {
	if f.IsGitRepoFunc != nil {
		return f.IsGitRepoFunc(ctx, path)
	}
	return true
}
