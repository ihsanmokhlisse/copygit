package command

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/imokhlis/copygit/internal/config"
	"github.com/imokhlis/copygit/internal/credential"
	"github.com/imokhlis/copygit/internal/git"
	"github.com/imokhlis/copygit/internal/model"
	"github.com/imokhlis/copygit/internal/output"
	"github.com/imokhlis/copygit/internal/provider"
	"github.com/imokhlis/copygit/internal/sync"
	"github.com/spf13/cobra"
)

// NewStatusCmdV2 creates the "status" command per cli-commands.md.
func NewStatusCmdV2() *cobra.Command {
	var (
		all       bool
		outputFmt string
	)

	cmd := &cobra.Command{
		Use:   "status [repo-path]",
		Short: "Show sync status of a repository",
		Long: `Display which providers are in sync and last sync times.

Shows whether each configured provider has the same commits as the local repo.`,
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

			return RunStatus(ctx, repoPath, all, outputFmt, logger)
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "Show status of all registered repositories")
	cmd.Flags().StringVar(&outputFmt, "output", "text", "Output format: text|json")

	return cmd
}

// RunStatus checks sync state for one or all repos.
func RunStatus( //nolint:gocognit,gocyclo,funlen // status reporting requires gathering multiple data sources
	ctx context.Context,
	repoPath string,
	all bool,
	outputFmt string,
	logger *slog.Logger,
) error { // status reporting requires gathering multiple data sources
	absPath, err := resolveRepoPath(repoPath)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	// Collect repos
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
		return errors.New("no repositories registered")
	}

	// Initialize dependencies
	gitExec, err := git.NewExecGit(logger)
	if err != nil {
		return fmt.Errorf("init git: %w", err)
	}

	credMgr := credential.NewChainManager(logger, config.DefaultCredentialsPath())
	globalCfg, err := config.LoadGlobal(config.ConfigPath(""))
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Select formatter
	var formatter output.Formatter
	switch outputFmt {
	case "json":
		formatter = output.NewJSONFormatter(os.Stdout)
	default:
		formatter = output.NewTextFormatter(os.Stdout)
	}

	orchestrator := sync.NewOrchestrator(gitExec, logger)

	for _, repo := range repos {
		if !gitExec.IsGitRepo(ctx, repo) {
			formatter.PrintWarning(fmt.Sprintf("%s: %v", repo, model.ErrNotGitRepo))
			continue
		}

		repoCfg, err := config.LoadRepoConfig(repo)
		if err != nil {
			formatter.PrintWarning(fmt.Sprintf("%s: %v", repo, err))
			continue
		}

		enabledNames := repoCfg.EnabledProviderNames()
		providerConfigs, err := globalCfg.ProvidersByNames(enabledNames)
		if err != nil {
			formatter.PrintWarning(fmt.Sprintf("%s: %v", repo, err))
			continue
		}

		providerRegistry, err := provider.BuildRegistry(ctx, providerConfigs, credMgr, gitExec, logger)
		if err != nil {
			formatter.PrintWarning(fmt.Sprintf("%s: %v", repo, err))
			continue
		}

		providerMap := make(map[string]provider.Provider)
		for _, name := range enabledNames {
			if prov, err := providerRegistry.Get(name); err == nil {
				providerMap[name] = prov
			}
		}

		report, err := orchestrator.Status(ctx, repo, providerMap, repoCfg.SyncTargets)
		if err != nil {
			formatter.PrintError(fmt.Errorf("%s: %w", repo, err))
			continue
		}

		_ = formatter.PrintStatusReport(report)
	}

	return nil
}
