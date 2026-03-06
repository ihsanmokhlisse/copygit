package command

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/imokhlis/copygit/internal/config"
	"github.com/imokhlis/copygit/internal/credential"
	"github.com/imokhlis/copygit/internal/git"
	"github.com/imokhlis/copygit/internal/lock"
	"github.com/imokhlis/copygit/internal/model"
	"github.com/imokhlis/copygit/internal/output"
	"github.com/imokhlis/copygit/internal/provider"
	"github.com/imokhlis/copygit/internal/sync"
)

// NewSyncCmd creates the "sync" command per cli-commands.md.
func NewSyncCmd() *cobra.Command {
	var (
		dryRun    bool
		force     bool
		outputFmt string
	)

	cmd := &cobra.Command{
		Use:   "sync [repo-path]",
		Short: "Full bidirectional sync (fetch + detect conflicts + push)",
		Long: `Performs a complete sync: fetch all → detect conflicts → resolve → push.

Unlike 'push', this fetches remote changes first and detects divergences.
If conflicts are found, prompts for resolution (unless --force).`,
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

			return RunSync(ctx, repoPath, dryRun, force, outputFmt, logger)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be synced")
	cmd.Flags().BoolVar(&force, "force", false, "Force push if diverged")
	cmd.Flags().StringVar(&outputFmt, "output", "text", "Output format: text|json")

	return cmd
}

// RunSync performs a full fetch + conflict detection + push cycle.
func RunSync( //nolint:gocognit,gocyclo,funlen // full sync cycle is inherently complex
	ctx context.Context,
	repoPath string,
	dryRun, force bool,
	outputFmt string,
	logger *slog.Logger,
) error { // full sync cycle is inherently complex
	absPath, err := resolveRepoPath(repoPath)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	gitExec, err := git.NewExecGit(logger)
	if err != nil {
		return fmt.Errorf("init git: %w", err)
	}

	if !gitExec.IsGitRepo(ctx, absPath) {
		return fmt.Errorf("%w: %s", model.ErrNotAGitRepo, absPath)
	}

	// Load configs
	repoCfg, err := config.LoadRepoConfig(absPath)
	if err != nil {
		return fmt.Errorf("load repo config: %w", err)
	}

	globalCfg, err := config.LoadGlobal(config.ConfigPath(""))
	if err != nil {
		return fmt.Errorf("load global config: %w", err)
	}

	credMgr := credential.NewChainManager(logger, config.DefaultCredentialsPath())

	// Build providers
	enabledNames := repoCfg.EnabledProviderNames()
	providerConfigs, err := globalCfg.ProvidersByNames(enabledNames)
	if err != nil {
		return fmt.Errorf("resolve providers: %w", err)
	}

	providerRegistry, err := provider.BuildRegistry(ctx, providerConfigs, credMgr, gitExec, logger)
	if err != nil {
		return fmt.Errorf("build providers: %w", err)
	}

	// Acquire lock
	fileLock, err := lock.NewFileLock(absPath)
	if err != nil {
		return fmt.Errorf("create lock: %w", err)
	}
	if err := fileLock.Lock(); err != nil {
		return fmt.Errorf("%w: %w", model.ErrLockAcquireFailed, err)
	}
	defer func() { _ = fileLock.Unlock() }()

	// Step 1: Fetch from all remotes
	remoteMgr := git.NewRemoteManager(gitExec, logger)
	branchMgr := git.NewBranchManager(gitExec, logger)
	currentBranch, err := branchMgr.CurrentBranch(ctx, absPath)
	if err != nil {
		return fmt.Errorf("get current branch: %w", err)
	}

	for _, target := range repoCfg.SyncTargets {
		if !target.Enabled {
			continue
		}
		prov, err := providerRegistry.Get(target.ProviderName)
		if err != nil {
			continue
		}
		remoteName := prov.RemoteName()
		if err := remoteMgr.Fetch(ctx, absPath, remoteName); err != nil {
			logger.WarnContext(ctx, "fetch failed", "remote", remoteName, "error", err)
		}
	}

	// Step 2: Detect conflicts
	localHead, err := branchMgr.GetHeadHash(ctx, absPath)
	if err != nil {
		return fmt.Errorf("get local head: %w", err)
	}

	var conflicts []model.ConflictInfo
	for _, target := range repoCfg.SyncTargets {
		if !target.Enabled {
			continue
		}
		prov, err := providerRegistry.Get(target.ProviderName)
		if err != nil {
			continue
		}
		remoteName := prov.RemoteName()
		remoteHead, err := remoteMgr.GetRemoteHeadHash(ctx, absPath, remoteName, currentBranch)
		if err != nil {
			continue
		}

		if remoteHead != localHead {
			ahead, behind, err := branchMgr.AheadBehind(ctx, absPath, "HEAD",
				fmt.Sprintf("%s/%s", remoteName, currentBranch))
			if err != nil {
				continue
			}

			conflictType := model.ConflictFastForward
			if behind > 0 {
				conflictType = model.ConflictDiverged
			}

			conflicts = append(conflicts, model.ConflictInfo{
				ProviderName: target.ProviderName,
				Type:         conflictType,
				LocalHead:    localHead,
				RemoteHead:   remoteHead,
				AheadBy:      ahead,
				BehindBy:     behind,
			})
		}
	}

	// Step 3: Handle conflicts
	if len(conflicts) > 0 && !force {
		for _, c := range conflicts {
			fmt.Printf("Conflict: %s is %s (ahead: %d, behind: %d)\n",
				c.ProviderName, c.Type, c.AheadBy, c.BehindBy)
		}
		if !dryRun {
			fmt.Println("Use --force to force push, or resolve manually.")
			return model.ErrConflictDetected
		}
	}

	if dryRun {
		fmt.Println("Dry run — no changes made.")
		return nil
	}

	// Step 4: Push
	providerMap := make(map[string]provider.Provider)
	for _, name := range enabledNames {
		if prov, err := providerRegistry.Get(name); err == nil {
			providerMap[name] = prov
		}
	}

	orchestrator := sync.NewOrchestrator(gitExec, logger)
	report, err := orchestrator.Push(ctx, absPath, providerMap, repoCfg.SyncTargets)
	if err != nil {
		return fmt.Errorf("push: %w", err)
	}

	// Output
	var formatter output.Formatter
	switch outputFmt {
	case "json":
		formatter = output.NewJSONFormatter(os.Stdout)
	default:
		formatter = output.NewTextFormatter(os.Stdout)
	}

	_ = formatter.PrintSyncReport(report)
	return nil
}
