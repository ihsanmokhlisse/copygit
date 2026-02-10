package git

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// TagManager wraps git tag operations.
type TagManager struct {
	exec   GitExecutor
	logger *slog.Logger
}

// NewTagManager creates a new TagManager.
func NewTagManager(exec GitExecutor, logger *slog.Logger) *TagManager {
	return &TagManager{exec: exec, logger: logger}
}

// ListTags returns all tag names.
func (tm *TagManager) ListTags(ctx context.Context, repoPath string) ([]string, error) {
	output, err := tm.exec.Run(ctx, repoPath, "tag", "--list")
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}

	var tags []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			tags = append(tags, line)
		}
	}
	return tags, nil
}

// PushTags pushes all tags to a remote.
func (tm *TagManager) PushTags(ctx context.Context, repoPath, remoteName string) error {
	_, err := tm.exec.Run(ctx, repoPath, "push", remoteName, "--tags")
	if err != nil {
		return fmt.Errorf("push tags to %s: %w", remoteName, err)
	}
	return nil
}
