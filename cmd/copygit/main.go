package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/imokhlis/copygit/cmd/copygit/command"
)

// Version is set at build time by GoReleaser.
var Version = "0.1.0-dev"

func main() {
	rootCmd := newRootCmd()
	ctx := context.Background()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
		logger.ErrorContext(ctx, "fatal error", "error", err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	var (
		configPath string
		jsonOutput bool
		verbose    bool
		quiet      bool
	)

	cmd := &cobra.Command{
		Use:     "copygit",
		Short:   "CopyGit - Multi-Provider Git Sync",
		Long:    "CopyGit automatically syncs your Git repositories across multiple providers (GitHub, GitLab, Gitea).",
		Version: Version,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			// Set up logger based on flags
			level := slog.LevelInfo
			if verbose {
				level = slog.LevelDebug
			}
			if quiet {
				level = slog.LevelError
			}

			opts := &slog.HandlerOptions{Level: level}
			var handler slog.Handler
			if jsonOutput {
				handler = slog.NewJSONHandler(os.Stderr, opts)
			} else {
				handler = slog.NewTextHandler(os.Stderr, opts)
			}

			slog.SetDefault(slog.New(handler))
			return nil
		},
	}

	// Global flags per cli-commands.md
	cmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "Override config file path")
	cmd.PersistentFlags().BoolVarP(&jsonOutput, "json", "j", false, "Machine-readable JSON output")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Debug-level logging")
	cmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-error output")

	// Register all commands
	cmd.AddCommand(command.NewInitCmdV2())
	cmd.AddCommand(command.NewLoginCmd())
	cmd.AddCommand(command.NewPushCmdV2())
	cmd.AddCommand(command.NewSyncCmd())
	cmd.AddCommand(command.NewStatusCmdV2())
	cmd.AddCommand(command.NewListCmdV2())
	cmd.AddCommand(command.NewRemoveCmdV2())
	cmd.AddCommand(command.NewConfigCmdV2())
	cmd.AddCommand(command.NewHooksCmd())
	cmd.AddCommand(command.NewDaemonCmd())
	cmd.AddCommand(command.NewCloneCmd())
	cmd.AddCommand(command.NewHealthCmd())
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newCompletionCmd())

	return cmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cmd.Printf("copygit version %s\n", Version)
			return nil
		},
	}
}

func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for CopyGit.

To load completions:

Bash:
  $ source <(copygit completion bash)
  # Or add to ~/.bashrc:
  $ copygit completion bash > /etc/bash_completion.d/copygit

Zsh:
  $ source <(copygit completion zsh)
  # Or add to fpath:
  $ copygit completion zsh > "${fpath[1]}/_copygit"

Fish:
  $ copygit completion fish | source
  # Or persist:
  $ copygit completion fish > ~/.config/fish/completions/copygit.fish

PowerShell:
  PS> copygit completion powershell | Out-String | Invoke-Expression`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(cmd.OutOrStdout())
			case "zsh":
				return cmd.Root().GenZshCompletion(cmd.OutOrStdout())
			case "fish":
				return cmd.Root().GenFishCompletion(cmd.OutOrStdout(), true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
			default:
				return nil
			}
		},
	}
	return cmd
}
