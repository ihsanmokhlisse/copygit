package git

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// RemoteManager wraps git remote operations.
type RemoteManager struct {
	exec   GitExecutor
	logger *slog.Logger
}

// NewRemoteManager creates a new RemoteManager.
func NewRemoteManager(exec GitExecutor, logger *slog.Logger) *RemoteManager {
	return &RemoteManager{exec: exec, logger: logger}
}

// AddRemote adds a git remote to the repository.
func (rm *RemoteManager) AddRemote(ctx context.Context, repoPath, name, url string) error {
	_, err := rm.exec.Run(ctx, repoPath, "remote", "add", name, url)
	if err != nil {
		return fmt.Errorf("add remote %s: %w", name, err)
	}
	return nil
}

// RemoveRemote removes a git remote from the repository.
func (rm *RemoteManager) RemoveRemote(ctx context.Context, repoPath, name string) error {
	_, err := rm.exec.Run(ctx, repoPath, "remote", "remove", name)
	if err != nil {
		return fmt.Errorf("remove remote %s: %w", name, err)
	}
	return nil
}

// ListRemotes returns a map of remote name -> URL.
func (rm *RemoteManager) ListRemotes(ctx context.Context, repoPath string) (map[string]string, error) {
	output, err := rm.exec.Run(ctx, repoPath, "remote", "-v")
	if err != nil {
		return nil, fmt.Errorf("list remotes: %w", err)
	}

	remotes := make(map[string]string)
	for _, line := range strings.Split(output, "\n") {
		parts := strings.Fields(line)
		if len(parts) >= 2 && strings.HasSuffix(line, "(push)") {
			remotes[parts[0]] = parts[1]
		}
	}
	return remotes, nil
}

// Push pushes branches to a remote. If force is true, uses --force.
func (rm *RemoteManager) Push(
	ctx context.Context,
	repoPath string,
	remoteName string,
	branches []string,
	tags bool,
	force bool,
) error {
	args := []string{"push", remoteName}
	args = append(args, branches...)

	if tags {
		args = append(args, "--tags")
	}
	if force {
		args = append(args, "--force")
	}

	rm.logger.DebugContext(ctx, "pushing", "remote", remoteName, "branches", branches)
	_, err := rm.exec.Run(ctx, repoPath, args...)
	if err != nil {
		return fmt.Errorf("push to %s: %w", remoteName, err)
	}
	return nil
}

// Fetch fetches from a remote.
func (rm *RemoteManager) Fetch(ctx context.Context, repoPath, remoteName string) error {
	_, err := rm.exec.Run(ctx, repoPath, "fetch", remoteName)
	if err != nil {
		return fmt.Errorf("fetch from %s: %w", remoteName, err)
	}
	return nil
}

// GetRemoteHeadHash returns the HEAD hash for a branch on a remote.
func (rm *RemoteManager) GetRemoteHeadHash(
	ctx context.Context,
	repoPath string,
	remoteName string,
	branch string,
) (string, error) {
	ref := fmt.Sprintf("refs/remotes/%s/%s", remoteName, branch)
	hash, err := rm.exec.Run(ctx, repoPath, "rev-parse", ref)
	if err != nil {
		return "", fmt.Errorf("get remote head: %w", err)
	}
	return hash, nil
}

// HasRemote checks if a remote with the given name exists.
func (rm *RemoteManager) HasRemote(ctx context.Context, repoPath, name string) bool {
	remotes, err := rm.ListRemotes(ctx, repoPath)
	if err != nil {
		return false
	}
	_, ok := remotes[name]
	return ok
}
