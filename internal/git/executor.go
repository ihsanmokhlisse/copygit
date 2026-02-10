package git

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
)

// GitExecutor abstracts git CLI operations for testability.
type GitExecutor interface { //nolint:revive // established API name
	// Run executes a git command and returns combined stdout.
	// Working directory is set to repoPath.
	Run(ctx context.Context, repoPath string, args ...string) (string, error)

	// RunWithStdin executes a git command with stdin input.
	// Used for `git credential fill` and similar commands.
	RunWithStdin(ctx context.Context, repoPath string, stdin string, args ...string) (string, error)

	// IsGitRepo checks if the given path is inside a git worktree.
	IsGitRepo(ctx context.Context, path string) bool
}

// ExecGit is the production implementation that calls the git binary.
type ExecGit struct {
	gitBinary string
	logger    *slog.Logger
}

// NewExecGit creates a new ExecGit.
// Resolves the git binary path via exec.LookPath("git").
// Returns error if git is not installed.
func NewExecGit(logger *slog.Logger) (*ExecGit, error) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return nil, fmt.Errorf("git not found in PATH: %w", err)
	}
	return &ExecGit{
		gitBinary: gitPath,
		logger:    logger,
	}, nil
}

// Run executes a git command and returns combined stdout.
func (g *ExecGit) Run(ctx context.Context, repoPath string, args ...string) (string, error) {
	g.logger.DebugContext(ctx, "git command", "path", repoPath, "args", args)

	cmd := exec.CommandContext(ctx, g.gitBinary, args...) //nolint:gosec // running git is the core purpose
	cmd.Dir = repoPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", &GitError{
			Command:  g.gitBinary,
			Args:     args,
			ExitCode: cmd.ProcessState.ExitCode(),
			Stderr:   string(output),
		}
	}

	return strings.TrimSpace(string(output)), nil
}

// RunWithStdin executes a git command with stdin input.
func (g *ExecGit) RunWithStdin(ctx context.Context, repoPath, stdin string, args ...string) (string, error) {
	g.logger.DebugContext(ctx, "git command with stdin", "path", repoPath, "args", args)

	cmd := exec.CommandContext(ctx, g.gitBinary, args...) //nolint:gosec // running git is the core purpose
	cmd.Dir = repoPath
	cmd.Stdin = strings.NewReader(stdin)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", &GitError{
			Command:  g.gitBinary,
			Args:     args,
			ExitCode: cmd.ProcessState.ExitCode(),
			Stderr:   string(output),
		}
	}

	return strings.TrimSpace(string(output)), nil
}

// IsGitRepo checks if the given path is inside a git worktree.
func (g *ExecGit) IsGitRepo(ctx context.Context, path string) bool {
	cmd := exec.CommandContext(ctx, g.gitBinary, "rev-parse", "--is-inside-work-tree") //nolint:gosec // running git is the core purpose
	cmd.Dir = path
	err := cmd.Run()
	return err == nil
}

// GitError is defined in errors.go
