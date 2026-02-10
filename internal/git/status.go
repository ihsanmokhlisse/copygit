package git

import (
	"context"
	"fmt"
	"log/slog"
)

// StatusManager wraps git status queries.
type StatusManager struct {
	exec   GitExecutor
	logger *slog.Logger
}

// NewStatusManager creates a new StatusManager.
func NewStatusManager(exec GitExecutor, logger *slog.Logger) *StatusManager {
	return &StatusManager{exec: exec, logger: logger}
}

// IsClean returns true if the working tree has no uncommitted changes.
func (sm *StatusManager) IsClean(ctx context.Context, repoPath string) (bool, error) {
	output, err := sm.exec.Run(ctx, repoPath, "status", "--porcelain")
	if err != nil {
		return false, fmt.Errorf("git status: %w", err)
	}
	return output == "", nil
}

// HasCommits returns true if the repository has at least one commit.
func (sm *StatusManager) HasCommits(ctx context.Context, repoPath string) bool {
	_, err := sm.exec.Run(ctx, repoPath, "rev-parse", "HEAD")
	return err == nil
}
