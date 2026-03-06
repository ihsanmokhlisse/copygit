package command

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/imokhlis/copygit/internal/config"
	"github.com/imokhlis/copygit/internal/model"
	"github.com/imokhlis/copygit/internal/output"
)

// NewConfigCmdV2 creates the enhanced "config" command (T097-T102).
func NewConfigCmdV2() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage global configuration",
		Long:  "Add, edit, or view global provider configurations.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	// Subcommands
	cmd.AddCommand(newConfigAddProviderCmd())
	cmd.AddCommand(newConfigListProvidersCmd())
	cmd.AddCommand(newConfigRemoveProviderCmd())

	return cmd
}

// newConfigAddProviderCmd creates "config add-provider" (T097-T098).
func newConfigAddProviderCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add-provider <name> <type> <base-url>",
		Short: "Add a provider configuration",
		Long: `Add a new Git provider to the global configuration.

Type can be: github | gitlab | gitea | generic
AuthMethod can be: ssh | https | token (default: https)`,
		Args: cobra.MinimumNArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			name := args[0]
			typeStr := args[1]
			baseURL := args[2]

			return RunConfigAddProvider(ctx, name, typeStr, baseURL, logger)
		},
	}
}

// newConfigListProvidersCmd creates "config list-providers" (T099-T100).
func newConfigListProvidersCmd() *cobra.Command {
	var outputFmt string

	cmd := &cobra.Command{
		Use:   "list-providers",
		Short: "List all configured providers",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			return RunConfigListProviders(ctx, outputFmt, logger)
		},
	}

	cmd.Flags().StringVar(&outputFmt, "output", "text", "Output format: text|json")
	return cmd
}

// newConfigRemoveProviderCmd creates "config remove-provider" (T101-T102).
func newConfigRemoveProviderCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove-provider <name>",
		Short: "Remove a provider configuration",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			name := args[0]

			return RunConfigRemoveProvider(ctx, name, logger)
		},
	}
}

// RunConfigAddProvider implements add-provider logic (T097-T098).
func RunConfigAddProvider(
	ctx context.Context,
	name string,
	typeStr string,
	baseURL string,
	logger *slog.Logger,
) error {
	// Validate type
	validTypes := map[string]bool{
		"github":  true,
		"gitlab":  true,
		"gitea":   true,
		"generic": true,
	}

	if !validTypes[typeStr] {
		return fmt.Errorf("invalid type: %s (must be github|gitlab|gitea|generic)", typeStr)
	}

	// Load or create global config
	globalConfigPath := config.ConfigPath("")
	globalCfg, err := config.LoadGlobal(globalConfigPath)
	if err != nil {
		if !errors.Is(err, model.ErrConfigNotFound) {
			return fmt.Errorf("load config: %w", err)
		}
		globalCfg = config.DefaultGlobalConfig()
	}

	// Check if provider already exists
	if _, ok := globalCfg.Providers[name]; ok {
		return fmt.Errorf("provider already exists: %s", name)
	}

	// Prompt for auth method (T098)
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Select auth method (default: https):")
	fmt.Println("  1. https")
	fmt.Println("  2. ssh")
	fmt.Println("  3. token")
	fmt.Print("Choice: ")

	choice, _ := reader.ReadString('\n')
	choice = strings.TrimSpace(choice)

	authMethod := model.AuthHTTPS
	switch choice {
	case "2":
		authMethod = model.AuthSSH
	case "3":
		authMethod = model.AuthToken
	}

	// Add provider
	globalCfg.Providers[name] = model.ProviderConfig{
		Name:        name,
		Type:        model.ProviderType(typeStr),
		BaseURL:     baseURL,
		AuthMethod:  authMethod,
		IsPreferred: false,
	}

	// Save
	if err := globalCfg.Save(globalConfigPath); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	logger.InfoContext(ctx, "provider added",
		"name", name,
		"type", typeStr,
		"auth", string(authMethod))

	return nil
}

// RunConfigListProviders implements list-providers logic (T099-T100).
func RunConfigListProviders(
	_ context.Context,
	outputFmt string,
	_ *slog.Logger,
) error {
	globalConfigPath := config.ConfigPath("")
	globalCfg, err := config.LoadGlobal(globalConfigPath)
	if err != nil {
		if errors.Is(err, model.ErrConfigNotFound) {
			fmt.Println("No providers configured.")
			return nil
		}
		return fmt.Errorf("load config: %w", err)
	}

	// Convert to slice for output
	providers := make([]model.ProviderConfig, 0, len(globalCfg.Providers))
	for _, prov := range globalCfg.Providers {
		providers = append(providers, prov)
	}

	// Format output
	var formatter output.Formatter
	switch outputFmt {
	case "json":
		formatter = output.NewJSONFormatter(os.Stdout)
	default:
		formatter = output.NewTextFormatter(os.Stdout)
	}

	return formatter.PrintProviderList(providers)
}

// RunConfigRemoveProvider implements remove-provider logic (T101-T102).
func RunConfigRemoveProvider(
	ctx context.Context,
	name string,
	logger *slog.Logger,
) error {
	globalConfigPath := config.ConfigPath("")
	globalCfg, err := config.LoadGlobal(globalConfigPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if _, ok := globalCfg.Providers[name]; !ok {
		return fmt.Errorf("provider not found: %s", name)
	}

	delete(globalCfg.Providers, name)

	if err := globalCfg.Save(globalConfigPath); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	logger.InfoContext(ctx, "provider removed", "name", name)
	return nil
}
