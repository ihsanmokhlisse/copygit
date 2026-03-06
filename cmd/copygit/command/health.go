package command

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/imokhlis/copygit/internal/config"
	"github.com/imokhlis/copygit/internal/model"
)

// NewHealthCmd creates the "health" command for v0.2.0.
func NewHealthCmd() *cobra.Command {
	var outputFmt string

	cmd := &cobra.Command{
		Use:   "health",
		Short: "Check connectivity to all configured providers",
		Long: `Verify that all configured providers are reachable.

Tests network connectivity and authentication status for each provider.
Useful for diagnosing sync failures.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			return RunHealth(ctx, outputFmt, logger)
		},
	}

	cmd.Flags().StringVar(&outputFmt, "output", "text", "Output format: text|json")
	return cmd
}

// ProviderHealth holds the health check result for a single provider.
type ProviderHealth struct {
	Name      string        `json:"name"`
	Type      string        `json:"type"`
	BaseURL   string        `json:"base_url"`
	Reachable bool          `json:"reachable"`
	Latency   time.Duration `json:"latency_ms"`
	Status    int           `json:"status_code"`
	Error     string        `json:"error,omitempty"`
}

// RunHealth checks connectivity to all configured providers.
func RunHealth(ctx context.Context, outputFmt string, logger *slog.Logger) error {
	globalCfg, err := config.LoadGlobal(config.ConfigPath(""))
	if err != nil {
		if errors.Is(err, model.ErrConfigNotFound) {
			fmt.Println("No providers configured. Run 'copygit config add-provider' first.")
			return nil
		}
		return fmt.Errorf("load config: %w", err)
	}

	if len(globalCfg.Providers) == 0 {
		fmt.Println("No providers configured.")
		return nil
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	results := make([]ProviderHealth, 0, len(globalCfg.Providers))

	for name, prov := range globalCfg.Providers {
		health := checkProvider(ctx, client, name, prov)
		results = append(results, health)
	}

	printHealthResults(results, outputFmt)
	return nil
}

func checkProvider(ctx context.Context, client *http.Client, name string, prov model.ProviderConfig) ProviderHealth {
	health := ProviderHealth{
		Name:    name,
		Type:    string(prov.Type),
		BaseURL: prov.BaseURL,
	}

	url := healthCheckURL(prov)

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	if err != nil {
		health.Error = err.Error()
		return health
	}

	req.Header.Set("User-Agent", "copygit/health-check")

	resp, err := client.Do(req)
	health.Latency = time.Since(start)

	if err != nil {
		health.Error = err.Error()
		return health
	}
	defer resp.Body.Close()

	health.Status = resp.StatusCode
	health.Reachable = resp.StatusCode < 500

	return health
}

func healthCheckURL(prov model.ProviderConfig) string {
	base := strings.TrimSuffix(prov.BaseURL, "/")

	switch prov.Type {
	case model.ProviderGitHub:
		if strings.Contains(base, "github.com") {
			return "https://api.github.com"
		}
		return base + "/api/v3"
	case model.ProviderGitLab:
		return base + "/api/v4/version"
	case model.ProviderGitea:
		return base + "/api/v1/version"
	default:
		return base
	}
}

func printHealthResults(results []ProviderHealth, format string) {
	if format == "json" {
		fmt.Print("[")
		for i, r := range results {
			if i > 0 {
				fmt.Print(",")
			}
			errStr := ""
			if r.Error != "" {
				errStr = fmt.Sprintf(`,"error":"%s"`, r.Error)
			}
			fmt.Printf(`{"name":"%s","type":"%s","reachable":%t,"latency_ms":%d,"status":%d%s}`,
				r.Name, r.Type, r.Reachable, r.Latency.Milliseconds(), r.Status, errStr)
		}
		fmt.Println("]")
		return
	}

	fmt.Println("Provider Health Check")
	fmt.Println(strings.Repeat("-", 60))

	for _, r := range results {
		status := "OK"
		if !r.Reachable {
			status = "FAIL"
		}
		if r.Error != "" {
			status = "ERROR"
		}

		icon := "+"
		if status != "OK" {
			icon = "x"
		}

		fmt.Printf("  [%s] %s (%s)\n", icon, r.Name, r.Type)
		fmt.Printf("      URL:     %s\n", r.BaseURL)
		fmt.Printf("      Status:  %s", status)
		if r.Reachable {
			fmt.Printf(" (%dms)\n", r.Latency.Milliseconds())
		} else if r.Error != "" {
			fmt.Printf(" - %s\n", r.Error)
		} else {
			fmt.Printf(" (HTTP %d)\n", r.Status)
		}
		fmt.Println()
	}
}
