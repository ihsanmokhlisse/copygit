package command

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/imokhlis/copygit/internal/config"
	"github.com/imokhlis/copygit/internal/git"
	"github.com/imokhlis/copygit/internal/model"
	"github.com/spf13/cobra"
)

// NewInitCmd creates the enhanced "init" command (T056-T065).
func NewInitCmdV2() *cobra.Command {
	return &cobra.Command{
		Use:   "init <repo-path>",
		Short: "Initialize a repository for syncing",
		Long: `Register a Git repository with CopyGit and configure its sync targets.

This command will:
1. Validate the repository
2. Auto-detect existing remotes
3. Prompt for providers to sync to
4. Create .copygit.toml
5. Register in global repository registry`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			repoPath := args[0]

			return RunInit(ctx, repoPath, logger)
		},
	}
}

// InitHandler encapsulates init logic.
type InitHandler struct {
	repoPath   string
	configPath string
	logger     *slog.Logger
	gitExec    git.GitExecutor
	reader     *bufio.Reader
}

// RunInit is the main init entrypoint (T056).
func RunInit(ctx context.Context, repoPath string, logger *slog.Logger) error {
	// 1. Resolve absolute path (T057)
	absPath, err := resolveRepoPath(repoPath)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	// 2. Validate it's a git repo (T058)
	gitExec, err := git.NewExecGit(logger)
	if err != nil {
		return fmt.Errorf("initialize git executor: %w", err)
	}

	if !gitExec.IsGitRepo(ctx, absPath) {
		return fmt.Errorf("%w: %s", model.ErrNotAGitRepo, absPath)
	}

	// 3. Check if already registered (T059)
	registryPath := config.DefaultRepoRegistryPath()
	registry, err := config.LoadRepoRegistry(registryPath)
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	if _, err := config.FindRepo(registry, absPath); err == nil {
		return fmt.Errorf("repository already registered: %s", absPath)
	}

	// 4. Load global config (T060)
	globalConfigPath := config.ConfigPath("")
	globalCfg, err := config.LoadGlobal(globalConfigPath)
	if err != nil {
		if !errors.Is(err, model.ErrConfigNotFound) {
			return fmt.Errorf("load global config: %w", err)
		}
		// Create default if not found
		globalCfg = config.DefaultGlobalConfig()
	}

	// 5. Auto-detect existing remotes (T061)
	existingRemotes, err := DetectExistingRemotes(ctx, absPath, gitExec)
	if err != nil {
		logger.WarnContext(ctx, "could not detect remotes", "error", err)
	}

	logger.InfoContext(ctx, "detected existing remotes", "count", len(existingRemotes))

	// 6. Interactive provider selection (T062)
	handler := &InitHandler{
		repoPath:   absPath,
		configPath: globalConfigPath,
		logger:     logger,
		gitExec:    gitExec,
		reader:     bufio.NewReader(os.Stdin),
	}

	syncTargets, err := handler.InteractiveProviderSelection(ctx, globalCfg)
	if err != nil {
		return fmt.Errorf("provider selection: %w", err)
	}

	// 7. Create repo config (T063)
	repoConfig := &model.RepoConfig{
		Version:     "1",
		SyncTargets: syncTargets,
	}

	// 8. Save repo config (T064)
	if err := config.SaveRepoConfig(absPath, repoConfig); err != nil {
		return fmt.Errorf("save repo config: %w", err)
	}

	// 9. Register repo globally (T065)
	if _, err := config.RegisterRepo(registry, absPath, ""); err != nil {
		return fmt.Errorf("register repo: %w", err)
	}

	if err := config.SaveRepoRegistry(registryPath, registry); err != nil {
		return fmt.Errorf("save registry: %w", err)
	}

	logger.InfoContext(ctx, "repository initialized successfully",
		"path", absPath,
		"targets", len(syncTargets))

	return nil
}

// resolveRepoPath converts relative or home-relative path to absolute.
func resolveRepoPath(repoPath string) (string, error) {
	if strings.HasPrefix(repoPath, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		repoPath = filepath.Join(home, repoPath[1:])
	}

	return filepath.Abs(repoPath)
}

// DetectExistingRemotes extracts remotes from git config.
func DetectExistingRemotes(ctx context.Context, repoPath string, gitExec git.GitExecutor) (map[string]string, error) {
	output, err := gitExec.Run(ctx, repoPath, "config", "--get-regexp", "^remote\\..*\\.url$")
	if err != nil {
		return nil, err
	}

	remotes := make(map[string]string)
	for _, line := range strings.Split(output, "\n") {
		parts := strings.Fields(line)
		if len(parts) == 2 {
			key := parts[0]
			url := parts[1]
			// Extract remote name from key (remote.NAME.url -> NAME)
			nameParts := strings.Split(key, ".")
			if len(nameParts) == 3 {
				remoteName := nameParts[1]
				remotes[remoteName] = url
			}
		}
	}

	return remotes, nil
}

// InteractiveProviderSelection prompts user for sync targets.
func (h *InitHandler) InteractiveProviderSelection(
	ctx context.Context,
	globalCfg *config.GlobalConfig,
) ([]model.RepoSyncTarget, error) {
	if len(globalCfg.Providers) == 0 {
		fmt.Println("No providers configured. Run 'copygit config add-provider' first.")
		return nil, model.ErrProviderNotFound
	}

	fmt.Println("\nAvailable providers:")
	providers := make([]string, 0, len(globalCfg.Providers))
	for name := range globalCfg.Providers {
		providers = append(providers, name)
	}
	sort.Strings(providers)
	for _, name := range providers {
		fmt.Printf("  - %s (%s)\n", name, globalCfg.Providers[name].Type)
	}

	fmt.Println("\nSelect providers to sync to (comma-separated names, or 'all'):")
	input, err := h.reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	input = strings.TrimSpace(input)
	var selectedNames []string

	if input == "all" {
		for name := range globalCfg.Providers {
			selectedNames = append(selectedNames, name)
		}
	} else {
		selectedNames = strings.Split(input, ",")
		for i := range selectedNames {
			selectedNames[i] = strings.TrimSpace(selectedNames[i])
		}
	}

	// For each selected provider, prompt for remote URL
	targets := make([]model.RepoSyncTarget, 0, len(selectedNames))
	for _, name := range selectedNames {
		if _, ok := globalCfg.Providers[name]; !ok {
			h.logger.WarnContext(ctx, "provider not found", "name", name)
			continue
		}

		fmt.Printf("\nEnter remote URL for %s (or press Enter to auto-generate):\n", name)
		urlInput, err := h.reader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		urlInput = strings.TrimSpace(urlInput)
		if urlInput == "" {
			// TODO: Auto-generate from current repo info
			urlInput = fmt.Sprintf("https://example.com/%s.git", filepath.Base(h.repoPath))
		}

		targets = append(targets, model.RepoSyncTarget{
			ProviderName: name,
			RemoteURL:    urlInput,
			Enabled:      true,
		})
	}

	return targets, nil
}
