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
	cmd.AddCommand(newVersionCmd())

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
