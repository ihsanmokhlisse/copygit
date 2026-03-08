package command

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/imokhlis/copygit/internal/config"
	"github.com/imokhlis/copygit/internal/git"
	"github.com/imokhlis/copygit/internal/model"
)

// NewCloneCmd creates the "clone" command for v0.2.0.
func NewCloneCmd() *cobra.Command {
	var (
		provider  string
		outputDir string
		initSync  bool
	)

	cmd := &cobra.Command{
		Use:   "clone <repo-url> [destination]",
		Short: "Clone a repo and register it with CopyGit",
		Long: `Clone a repository and automatically register it for multi-provider sync.

This combines 'git clone' + 'copygit init' in one step.
Optionally specify --provider to tag which provider the source comes from.`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

			repoURL := args[0]
			dest := outputDir
			if len(args) > 1 {
				dest = args[1]
			}

			return RunClone(ctx, repoURL, dest, provider, initSync, logger)
		},
	}

	cmd.Flags().StringVarP(&provider, "provider", "p", "", "Source provider name (auto-detected if omitted)")
	cmd.Flags().StringVarP(&outputDir, "dir", "d", "", "Clone destination directory")
	cmd.Flags().BoolVar(&initSync, "init", true, "Automatically register the cloned repo with CopyGit")

	return cmd
}

// RunClone clones a repo and optionally registers it.
func RunClone(
	ctx context.Context,
	repoURL, dest, providerName string,
	initSync bool,
	logger *slog.Logger,
) error {
	gitExec, err := git.NewExecGit(logger)
	if err != nil {
		return fmt.Errorf("initialize git: %w", err)
	}

	// Derive destination from URL if not provided
	if dest == "" {
		dest = repoNameFromURL(repoURL)
	}

	absPath, err := filepath.Abs(dest)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	// Clone
	logger.InfoContext(ctx, "cloning repository", "url", repoURL, "dest", absPath)
	wd, _ := os.Getwd()
	if _, err := gitExec.Run(ctx, wd, "clone", repoURL, absPath); err != nil {
		return fmt.Errorf("git clone: %w", err)
	}

	fmt.Printf("Cloned %s into %s\n", repoURL, absPath)

	if !initSync {
		return nil
	}

	// Auto-detect source provider from URL
	if providerName == "" {
		providerName = detectProviderFromURL(repoURL)
	}

	// Register in CopyGit
	registryPath := config.DefaultRepoRegistryPath()
	registry, err := config.LoadRepoRegistry(registryPath)
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	if _, err := config.RegisterRepo(registry, absPath, ""); err != nil {
		return fmt.Errorf("register repo: %w", err)
	}

	if err := config.SaveRepoRegistry(registryPath, registry); err != nil {
		return fmt.Errorf("save registry: %w", err)
	}

	// Load global config to find matching providers
	globalCfg, err := config.LoadGlobal(config.ConfigPath(""))
	if err != nil {
		if errors.Is(err, model.ErrConfigNotFound) {
			logger.InfoContext(ctx, "no global config found, skipping sync target setup")
			return nil
		}
		return fmt.Errorf("load config: %w", err)
	}

	// Build sync targets from all configured providers
	targets := buildSyncTargets(repoURL, providerName, globalCfg, absPath)

	if len(targets) > 0 {
		targetsWithOverrides := make([]model.RepoSyncTargetWithOverrides, len(targets))
		for i, t := range targets {
			targetsWithOverrides[i] = model.RepoSyncTargetWithOverrides{
				ProviderName: t.ProviderName,
				RemoteURL:    t.RemoteURL,
				Enabled:      t.Enabled,
			}
		}
		repoConfig := &model.RepoConfig{
			Version:     "1",
			SyncTargets: targetsWithOverrides,
		}
		if err := config.SaveRepoConfig(absPath, repoConfig); err != nil {
			return fmt.Errorf("save repo config: %w", err)
		}
	}

	fmt.Printf("Registered with CopyGit (%d sync targets)\n", len(targets))
	return nil
}

// buildSyncTargets creates sync targets for all providers, auto-generating URLs.
func buildSyncTargets(
	sourceURL, sourceProvider string,
	globalCfg *config.GlobalConfig,
	repoPath string,
) []model.RepoSyncTarget {
	repoName := repoNameFromURL(sourceURL)
	owner := ownerFromURL(sourceURL)

	var targets []model.RepoSyncTarget
	for name, prov := range globalCfg.Providers {
		remoteURL := GenerateRemoteURL(prov, owner, repoName)

		// Skip the source provider URL if it matches
		if name == sourceProvider {
			remoteURL = sourceURL
		}

		targets = append(targets, model.RepoSyncTarget{
			ProviderName: name,
			RemoteURL:    remoteURL,
			Enabled:      true,
		})
	}

	_ = repoPath // used for context
	return targets
}

// GenerateRemoteURL creates a remote URL for a provider given owner and repo name.
func GenerateRemoteURL(prov model.ProviderConfig, owner, repoName string) string {
	base := strings.TrimSuffix(prov.BaseURL, "/")

	switch prov.AuthMethod {
	case model.AuthSSH:
		host := strings.TrimPrefix(base, "https://")
		host = strings.TrimPrefix(host, "http://")
		return fmt.Sprintf("git@%s:%s/%s.git", host, owner, repoName)
	default:
		return fmt.Sprintf("%s/%s/%s.git", base, owner, repoName)
	}
}

// repoNameFromURL extracts the repository name from a git URL.
func repoNameFromURL(url string) string {
	// Handle ssh format: git@github.com:owner/repo.git
	if strings.Contains(url, ":") && strings.Contains(url, "@") {
		parts := strings.Split(url, ":")
		if len(parts) == 2 {
			path := parts[1]
			path = strings.TrimSuffix(path, ".git")
			segments := strings.Split(path, "/")
			return segments[len(segments)-1]
		}
	}

	// Handle https format: https://github.com/owner/repo.git
	url = strings.TrimSuffix(url, ".git")
	url = strings.TrimSuffix(url, "/")
	segments := strings.Split(url, "/")
	if len(segments) > 0 {
		return segments[len(segments)-1]
	}

	return "repo"
}

// ownerFromURL extracts the owner/namespace from a git URL.
func ownerFromURL(url string) string {
	// SSH: git@github.com:owner/repo.git
	if strings.Contains(url, ":") && strings.Contains(url, "@") {
		parts := strings.Split(url, ":")
		if len(parts) == 2 {
			path := strings.TrimSuffix(parts[1], ".git")
			segments := strings.Split(path, "/")
			if len(segments) >= 2 {
				return segments[0]
			}
		}
	}

	// HTTPS: https://github.com/owner/repo.git
	url = strings.TrimSuffix(url, ".git")
	segments := strings.Split(url, "/")
	if len(segments) >= 2 {
		return segments[len(segments)-2]
	}

	return "user"
}

// detectProviderFromURL guesses the provider name from a git URL.
func detectProviderFromURL(url string) string {
	lower := strings.ToLower(url)
	switch {
	case strings.Contains(lower, "github.com"):
		return "github"
	case strings.Contains(lower, "gitlab.com"):
		return "gitlab"
	case strings.Contains(lower, "gitea"):
		return "gitea"
	default:
		return ""
	}
}
