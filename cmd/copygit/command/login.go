package command

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/imokhlis/copygit/internal/config"
	"github.com/imokhlis/copygit/internal/credential"
	"github.com/imokhlis/copygit/internal/model"
	"github.com/imokhlis/copygit/internal/provider"
)

// NewLoginCmd creates the "login" command per cli-commands.md.
func NewLoginCmd() *cobra.Command {
	var (
		providerName string
		token        string
		authMethod   string
	)

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with a provider",
		Long: `Store credentials for a configured provider.

Prompts for token interactively if --token is not provided.
Validates the credential before storing it.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			return RunLogin(ctx, providerName, token, authMethod, logger)
		},
	}

	cmd.Flags().StringVarP(&providerName, "provider", "p", "", "Provider name (required)")
	cmd.Flags().StringVarP(&token, "token", "t", "", "Access token (reads from stdin pipe; prefer over CLI args to avoid process list exposure)")
	cmd.Flags().StringVarP(&authMethod, "method", "m", "token", "Auth method: token|ssh|https")
	_ = cmd.MarkFlagRequired("provider")

	// Mark --token as deprecated in favor of interactive prompt or stdin pipe
	_ = cmd.Flags().SetAnnotation("token", "deprecated-notice", []string{"Passing tokens via CLI args exposes them in process listings. Prefer interactive prompt or piped stdin."})

	return cmd
}

// RunLogin authenticates with a provider and stores credentials.
func RunLogin(ctx context.Context, providerName, token, authMethod string, logger *slog.Logger) error {
	// Load global config
	globalCfg, err := config.LoadGlobal(config.ConfigPath(""))
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	provCfg, ok := globalCfg.Providers[providerName]
	if !ok {
		return fmt.Errorf("%w: %s", model.ErrProviderNotFound, providerName)
	}

	// Prompt for token if not provided (with hidden input)
	if token == "" && authMethod != "ssh" {
		fmt.Fprintf(os.Stderr, "Enter token for %s: ", providerName)
		tokenBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(os.Stderr) // newline after hidden input
		if err != nil {
			return fmt.Errorf("read token: %w", err)
		}
		token = strings.TrimSpace(string(tokenBytes))
	}

	if token == "" && authMethod != "ssh" {
		return fmt.Errorf("token is required for %s auth", authMethod)
	}

	// Build credential
	cred := &model.Credential{
		ProviderName: providerName,
		AuthMethod:   model.AuthMethod(authMethod),
		Token:        token,
	}

	// Validate credentials against provider API
	prov := buildProvider(provCfg, logger)
	if err := prov.ValidateCredentials(ctx, cred); err != nil {
		return fmt.Errorf("validate credentials: %w", err)
	}

	// Store credentials
	credMgr := credential.NewChainManager(logger, config.DefaultCredentialsPath())
	if err := credMgr.Store(ctx, provCfg, cred); err != nil {
		return fmt.Errorf("store credentials: %w", err)
	}

	fmt.Printf("Successfully authenticated with %s\n", providerName)
	return nil
}

// buildProvider creates a provider instance for validation.
func buildProvider(cfg model.ProviderConfig, logger *slog.Logger) provider.Provider {
	switch cfg.Type {
	case model.ProviderGitHub:
		return provider.NewGitHubProvider(cfg, logger)
	case model.ProviderGitLab:
		return provider.NewGitLabProvider(cfg, logger)
	case model.ProviderGitea:
		return provider.NewGiteaProvider(cfg, logger)
	default:
		return provider.NewGenericProvider(cfg, logger)
	}
}
