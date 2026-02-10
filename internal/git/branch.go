package git

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

// BranchManager wraps git branch operations.
type BranchManager struct {
	exec   GitExecutor
	logger *slog.Logger
}

// NewBranchManager creates a new BranchManager.
func NewBranchManager(exec GitExecutor, logger *slog.Logger) *BranchManager {
	return &BranchManager{exec: exec, logger: logger}
}

// CurrentBranch returns the name of the current branch.
func (bm *BranchManager) CurrentBranch(ctx context.Context, repoPath string) (string, error) {
	branch, err := bm.exec.Run(ctx, repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", fmt.Errorf("current branch: %w", err)
	}
	return branch, nil
}

// GetHeadHash returns the current HEAD commit hash.
func (bm *BranchManager) GetHeadHash(ctx context.Context, repoPath string) (string, error) {
	hash, err := bm.exec.Run(ctx, repoPath, "rev-parse", "HEAD")
	if err != nil {
		return "", fmt.Errorf("head hash: %w", err)
	}
	return hash, nil
}

// ListBranches returns all local branch names.
func (bm *BranchManager) ListBranches(ctx context.Context, repoPath string) ([]string, error) {
	output, err := bm.exec.Run(ctx, repoPath, "branch", "--list", "--format=%(refname:short)")
	if err != nil {
		return nil, fmt.Errorf("list branches: %w", err)
	}

	var branches []string
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			branches = append(branches, line)
		}
	}
	return branches, nil
}

// AheadBehind returns how many commits the local branch is ahead/behind a remote ref.
func (bm *BranchManager) AheadBehind(
	ctx context.Context,
	repoPath string,
	localRef string,
	remoteRef string,
) (ahead, behind int, err error) {
	output, err := bm.exec.Run(ctx, repoPath,
		"rev-list", "--left-right", "--count",
		fmt.Sprintf("%s...%s", localRef, remoteRef))
	if err != nil {
		return 0, 0, fmt.Errorf("ahead-behind: %w", err)
	}

	parts := strings.Fields(output)
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("unexpected rev-list output: %q", output)
	}

	_, _ = fmt.Sscanf(parts[0], "%d", &ahead)
	_, _ = fmt.Sscanf(parts[1], "%d", &behind)
	return ahead, behind, nil
}
