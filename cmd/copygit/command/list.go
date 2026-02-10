package command

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/imokhlis/copygit/internal/config"
	"github.com/imokhlis/copygit/internal/output"
	"github.com/spf13/cobra"
)

// NewListCmdV2 creates the enhanced "list" command (T086-T090).
func NewListCmdV2() *cobra.Command {
	var (
		outputFmt string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List registered repositories",
		Long:  "Display all repositories registered with CopyGit.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

			return RunList(ctx, outputFmt, logger)
		},
	}

	cmd.Flags().StringVar(&outputFmt, "output", "text", "Output format: text|json")

	return cmd
}

// RunList is the main list entrypoint (T086-T090).
func RunList(ctx context.Context, outputFmt string, logger *slog.Logger) error {
	// T087: Load repo registry
	registryPath := config.DefaultRepoRegistryPath()
	registry, err := config.LoadRepoRegistry(registryPath)
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	// T088: Validate registry integrity
	validationErrs := config.ValidateRepoRegistry(registry)
	if len(validationErrs) > 0 {
		logger.WarnContext(ctx, "registry validation issues", "count", len(validationErrs))
		for _, ve := range validationErrs {
			logger.WarnContext(ctx, "validation error", "field", ve.Field, "message", ve.Message)
		}
	}

	// T089: Format output
	var formatter output.Formatter
	switch outputFmt {
	case "json":
		formatter = output.NewJSONFormatter(os.Stdout)
	default:
		formatter = output.NewTextFormatter(os.Stdout)
	}

	// T090: Print results
	if err := formatter.PrintRepoList(registry.Repos); err != nil {
		return fmt.Errorf("format output: %w", err)
	}

	return nil
}
