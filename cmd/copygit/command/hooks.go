package command

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/imokhlis/copygit/internal/git"
	"github.com/imokhlis/copygit/internal/hook"
	"github.com/imokhlis/copygit/internal/model"
)

// NewHooksCmd creates the "hooks" parent command per cli-commands.md.
func NewHooksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hooks",
		Short: "Manage git hooks for automatic syncing",
	}

	cmd.AddCommand(newHooksInstallCmd())
	cmd.AddCommand(newHooksUninstallCmd())
	cmd.AddCommand(newHooksStatusCmd())

	return cmd
}

func newHooksInstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "install [repo-path]",
		Short: "Install git hook (post-push)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

			repoPath := "."
			if len(args) > 0 {
				repoPath = args[0]
			}

			return RunHooksInstall(ctx, repoPath, logger)
		},
	}
}

func newHooksUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall [repo-path]",
		Short: "Remove CopyGit git hooks",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

			repoPath := "."
			if len(args) > 0 {
				repoPath = args[0]
			}

			return RunHooksUninstall(ctx, repoPath, logger)
		},
	}
}

func newHooksStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status [repo-path]",
		Short: "Show hook installation status",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

			repoPath := "."
			if len(args) > 0 {
				repoPath = args[0]
			}

			return RunHooksStatus(ctx, repoPath, logger)
		},
	}
}

// RunHooksInstall installs the post-push hook.
func RunHooksInstall(ctx context.Context, repoPath string, logger *slog.Logger) error {
	absPath, err := resolveRepoPath(repoPath)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	gitExec, err := git.NewExecGit(logger)
	if err != nil {
		return err
	}

	if !gitExec.IsGitRepo(ctx, absPath) {
		return fmt.Errorf("%w: %s", model.ErrNotAGitRepo, absPath)
	}

	mgr := hook.NewHookManager(logger)
	if err := mgr.Install(ctx, absPath); err != nil {
		return fmt.Errorf("install hook: %w", err)
	}

	fmt.Println("Hook installed successfully.")
	return nil
}

// RunHooksUninstall removes the CopyGit post-push hook.
func RunHooksUninstall(ctx context.Context, repoPath string, logger *slog.Logger) error {
	absPath, err := resolveRepoPath(repoPath)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	mgr := hook.NewHookManager(logger)
	if err := mgr.Uninstall(ctx, absPath); err != nil {
		return fmt.Errorf("uninstall hook: %w", err)
	}

	fmt.Println("Hook uninstalled successfully.")
	return nil
}

// RunHooksStatus shows the current hook installation state.
func RunHooksStatus(ctx context.Context, repoPath string, logger *slog.Logger) error {
	absPath, err := resolveRepoPath(repoPath)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	mgr := hook.NewHookManager(logger)
	installed, err := mgr.IsInstalled(ctx, absPath)
	if err != nil {
		return fmt.Errorf("check hook status: %w", err)
	}

	if installed {
		fmt.Println("  post-push: installed (CopyGit)")
	} else {
		fmt.Println("  post-push: not installed")
	}

	return nil
}
