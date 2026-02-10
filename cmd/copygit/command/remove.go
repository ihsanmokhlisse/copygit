package command

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/imokhlis/copygit/internal/config"
	"github.com/spf13/cobra"
)

// NewRemoveCmdV2 creates the enhanced "remove" command (T091-T096).
func NewRemoveCmdV2() *cobra.Command {
	var (
		clean bool
	)

	cmd := &cobra.Command{
		Use:   "remove <repo-path>",
		Short: "Unregister a repository",
		Long: `Remove a repository from CopyGit's registry.

With --clean, also deletes the .copygit.toml file from the repository.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			repoPath := args[0]

			return RunRemove(ctx, repoPath, clean, logger)
		},
	}

	cmd.Flags().BoolVar(&clean, "clean", false, "Also remove .copygit.toml from the repository")

	return cmd
}

// RunRemove is the main remove entrypoint (T091).
func RunRemove(
	ctx context.Context,
	repoPath string,
	clean bool,
	logger *slog.Logger,
) error {
	// T092: Resolve repo path
	absPath, err := resolveRepoPath(repoPath)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	// T093: Load registry
	registryPath := config.DefaultRepoRegistryPath()
	registry, err := config.LoadRepoRegistry(registryPath)
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	// T094: Check if registered
	if _, err := config.FindRepo(registry, absPath); err != nil {
		return fmt.Errorf("repo not registered: %w", err)
	}

	// T095: Remove from registry
	if err := config.UnregisterRepo(registry, absPath); err != nil {
		return fmt.Errorf("unregister: %w", err)
	}

	// Save updated registry
	if err := config.SaveRepoRegistry(registryPath, registry); err != nil {
		return fmt.Errorf("save registry: %w", err)
	}

	logger.InfoContext(ctx, "repository unregistered", "path", absPath)

	// T096: Optionally clean .copygit.toml
	if clean {
		repoConfigPath := config.RepoConfigPath(absPath)
		if err := os.Remove(repoConfigPath); err != nil && !os.IsNotExist(err) {
			logger.WarnContext(ctx, "could not remove .copygit.toml", "error", err)
		} else if err == nil {
			logger.InfoContext(ctx, ".copygit.toml removed", "path", repoConfigPath)
		}
	}

	return nil
}
