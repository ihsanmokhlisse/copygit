package command

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/imokhlis/copygit/internal/config"
	"github.com/imokhlis/copygit/internal/credential"
	"github.com/imokhlis/copygit/internal/git"
	"github.com/imokhlis/copygit/internal/lock"
	"github.com/imokhlis/copygit/internal/output"
	"github.com/imokhlis/copygit/internal/provider"
	"github.com/imokhlis/copygit/internal/sync"
)

// NewPushCmdV2 creates the enhanced "push" command (T066-T075).
func NewPushCmdV2() *cobra.Command {
	var (
		all          bool
		dryRun       bool
		outputFmt    string
		conflictMode string
		fromHook     string
	)

	cmd := &cobra.Command{
		Use:   "push [repo-path]",
		Short: "Sync local repository to configured providers",
		Long: `Push all branches and tags from a local repository to all configured providers.

If no path is specified, uses current working directory.
With --all, syncs all registered repositories.
Conflict modes: warn (default), merge, force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

			var repoPath string
			if len(args) > 0 {
				repoPath = args[0]
			} else {
				wd, err := os.Getwd()
				if err != nil {
					return err
				}
				repoPath = wd
			}

			return RunPush(ctx, repoPath, all, dryRun, outputFmt, conflictMode, fromHook, logger)
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "Push all registered repositories")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be synced without making changes")
	cmd.Flags().StringVar(&outputFmt, "output", "text", "Output format: text|json")
	cmd.Flags().StringVar(&conflictMode, "conflict", "warn", "Conflict strategy: warn|merge|force")
	cmd.Flags().StringVar(&fromHook, "from-hook", "", "Skip the provider matching this remote name (for hook-triggered pushes)")
	_ = cmd.Flags().MarkHidden("from-hook")

	return cmd
}

// RunPush is the main push entrypoint (T066).
func RunPush(
	ctx context.Context,
	repoPath string,
	all bool,
	_ bool, // dryRun - reserved for future conflict handling
	outputFmt string,
	_ string, // conflictMode - reserved for future conflict handling
	fromHook string,
	logger *slog.Logger,
) error {
	// T067: Resolve repo path
	absPath, err := resolveRepoPath(repoPath)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	// T068: Handle --all flag
	var repos []string
	if all {
		registryPath := config.DefaultRepoRegistryPath()
		registry, err := config.LoadRepoRegistry(registryPath)
		if err != nil {
			return fmt.Errorf("load registry: %w", err)
		}
		for _, reg := range registry.Repos {
			repos = append(repos, reg.Path)
		}
	} else {
		repos = []string{absPath}
	}

	if len(repos) == 0 {
		return errors.New("no repositories to push")
	}

	// T069: Process each repo
	gitExec, err := git.NewExecGit(logger)
	if err != nil {
		return fmt.Errorf("initialize git: %w", err)
	}

	credMgr := credential.NewChainManager(logger, config.DefaultCredentialsPath())
	globalCfg, err := config.LoadGlobal(config.ConfigPath(""))
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	reports := make([]*sync.SyncReport, 0, len(repos))

	for _, repo := range repos {
		report, err := pushSingleRepo(ctx, repo, globalCfg, credMgr, gitExec, fromHook, logger)
		if err != nil {
			logger.ErrorContext(ctx, "push failed", "repo", repo, "error", err)
			continue
		}
		reports = append(reports, report)
	}

	// T075: Format and output results
	var formatter output.Formatter
	switch outputFmt {
	case "json":
		formatter = output.NewJSONFormatter(os.Stdout)
	default:
		formatter = output.NewTextFormatter(os.Stdout)
	}

	if len(reports) == 1 {
		_ = formatter.PrintSyncReport(reports[0])
	} else {
		_ = formatter.PrintMultiRepoSyncReport(reports)
	}

	return nil
}

// pushSingleRepo handles push for one repository with proper lock scoping.
// The lock is acquired and released within this function (not deferred in a loop).
func pushSingleRepo(
	ctx context.Context,
	repo string,
	globalCfg *config.GlobalConfig,
	credMgr credential.Manager,
	gitExec git.GitExecutor,
	fromHook string,
	logger *slog.Logger,
) (*sync.SyncReport, error) {
	// Load repo config
	repoCfg, err := config.LoadRepoConfig(repo)
	if err != nil {
		return nil, fmt.Errorf("load repo config: %w", err)
	}

	// Resolve provider instances
	enabledProviders := make([]string, 0)
	for _, target := range repoCfg.SyncTargets {
		if target.Enabled {
			enabledProviders = append(enabledProviders, target.ProviderName)
		}
	}

	providerConfigs, err := globalCfg.ProvidersByNames(enabledProviders)
	if err != nil {
		return nil, fmt.Errorf("resolve providers: %w", err)
	}

	providerRegistry, err := provider.BuildRegistry(ctx, providerConfigs, credMgr, gitExec, logger)
	if err != nil {
		return nil, fmt.Errorf("build provider registry: %w", err)
	}

	// Acquire lock — defer runs when this function returns, not the outer loop
	fileLock, err := lock.NewFileLock(repo)
	if err != nil {
		return nil, fmt.Errorf("create lock: %w", err)
	}
	if err := fileLock.Lock(); err != nil {
		return nil, fmt.Errorf("acquire lock: %w", err)
	}
	defer func() { _ = fileLock.Unlock() }()

	// Execute sync
	orchestrator := sync.NewOrchestrator(gitExec, logger).WithCredentialManager(credMgr)
	providerMap := make(map[string]provider.Provider)
	for _, name := range enabledProviders {
		if prov, err := providerRegistry.Get(name); err == nil {
			if fromHook != "" && prov.RemoteName() == fromHook {
				logger.InfoContext(ctx, "skipping provider due to --from-hook", "provider", name, "remote", prov.RemoteName())
				continue
			}
			providerMap[name] = prov
		}
	}

	report, err := orchestrator.Push(ctx, repo, providerMap, repoCfg.AsRepoSyncTargets())
	if err != nil {
		return nil, fmt.Errorf("sync: %w", err)
	}

	// Update registry
	registryPath := config.DefaultRepoRegistryPath()
	registry, loadErr := config.LoadRepoRegistry(registryPath)
	if loadErr == nil {
		_ = config.UpdateLastSync(registry, repo)
		_ = config.SaveRepoRegistry(registryPath, registry)
	}

	return report, nil
}
