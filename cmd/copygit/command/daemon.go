package command

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/imokhlis/copygit/internal/config"
	"github.com/imokhlis/copygit/internal/credential"
	"github.com/imokhlis/copygit/internal/daemon"
	"github.com/imokhlis/copygit/internal/git"
)

// NewDaemonCmd creates the "daemon" parent command per cli-commands.md.
func NewDaemonCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: "Manage the background sync daemon",
	}

	cmd.AddCommand(newDaemonStartCmd())
	cmd.AddCommand(newDaemonStopCmd())
	cmd.AddCommand(newDaemonStatusCmd())

	return cmd
}

func newDaemonStartCmd() *cobra.Command {
	var foreground bool

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the background sync daemon",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			return RunDaemonStart(ctx, foreground, logger)
		},
	}

	cmd.Flags().BoolVar(&foreground, "foreground", false, "Run in foreground (don't daemonize)")
	return cmd
}

func newDaemonStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop",
		Short: "Stop the background sync daemon",
		RunE: func(_ *cobra.Command, _ []string) error {
			return RunDaemonStop()
		},
	}
}

func newDaemonStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show daemon status",
		RunE: func(_ *cobra.Command, _ []string) error {
			return RunDaemonStatus()
		},
	}
}

// RunDaemonStart starts the background sync daemon.
func RunDaemonStart(ctx context.Context, _ bool, logger *slog.Logger) error { //nolint:revive // foreground reserved for future daemonize
	globalCfg, err := config.LoadGlobal(config.ConfigPath(""))
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	registryPath := config.DefaultRepoRegistryPath()
	registry, err := config.LoadRepoRegistry(registryPath)
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	gitExec, err := git.NewExecGit(logger)
	if err != nil {
		return fmt.Errorf("init git: %w", err)
	}

	credMgr := credential.NewChainManager(logger, config.DefaultCredentialsPath())

	pollInterval := 30 * time.Second
	if globalCfg.Daemon.PollInterval != "" {
		d, err := time.ParseDuration(globalCfg.Daemon.PollInterval)
		if err == nil {
			pollInterval = d
		}
	}

	// Write PID file
	pidFile := daemon.NewPIDFile(config.DefaultPIDFilePath())
	if err := pidFile.Write(os.Getpid()); err != nil {
		return fmt.Errorf("write pid file: %w", err)
	}
	defer func() { _ = pidFile.Remove() }()

	d := daemon.NewDaemon(
		pollInterval, logger, gitExec, credMgr,
		globalCfg, registry, registryPath,
	)

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		logger.Info("received shutdown signal")
		cancel()
	}()

	fmt.Printf("Daemon started (PID %d, interval %s)\n", os.Getpid(), pollInterval)
	return d.Run(ctx)
}

// RunDaemonStop stops the running daemon by reading its PID file.
func RunDaemonStop() error {
	pidFile := daemon.NewPIDFile(config.DefaultPIDFilePath())
	pid, err := pidFile.Read()
	if err != nil {
		return errors.New("daemon not running (no pid file)")
	}

	if pidFile.IsStale() {
		_ = pidFile.Remove()
		return errors.New("daemon not running (stale pid file)")
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("find process: %w", err)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("send signal: %w", err)
	}

	_ = pidFile.Remove()
	fmt.Printf("Daemon stopped (PID %d)\n", pid)
	return nil
}

// RunDaemonStatus shows whether the daemon is running.
func RunDaemonStatus() error {
	pidFile := daemon.NewPIDFile(config.DefaultPIDFilePath())
	pid, err := pidFile.Read()
	if err != nil {
		fmt.Println("Daemon is not running")
		return nil //nolint:nilerr // no pid file = daemon not running
	}

	if pidFile.IsStale() {
		fmt.Printf("Daemon is not running (stale PID %d)\n", pid)
		return nil
	}

	fmt.Printf("Daemon is running (PID %d)\n", pid)
	return nil
}
