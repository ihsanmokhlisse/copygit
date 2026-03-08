package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/imokhlis/copygit/internal/model"
)

// GlobalConfig is the top-level configuration structure.
// Serialized as TOML to ~/.copygit/config.
// Contains provider definitions shared across ALL repos.
type GlobalConfig struct {
	Version   string                          `toml:"version"`
	Providers map[string]model.ProviderConfig `toml:"providers"`
	Sync      SyncConfig                      `toml:"sync"`
	Daemon    DaemonConfig                    `toml:"daemon"`
	Log       LogConfig                       `toml:"log"`
}

type SyncConfig struct {
	MaxRetries     int    `toml:"max_retries"`      // Default: 5
	RetryBaseDelay string `toml:"retry_base_delay"` // Default: "5s"
	DryRunDefault  bool   `toml:"dry_run_default"`  // Default: false
	PushTags       bool   `toml:"push_tags"`        // Default: true
	PushBranches   bool   `toml:"push_branches"`    // Default: true
}

type DaemonConfig struct {
	PollInterval string `toml:"poll_interval"` // Default: "30s"
	MaxRetries   int    `toml:"max_retries"`   // Default: 10
	AutoStart    bool   `toml:"auto_start"`    // Default: false
}

type LogConfig struct {
	Level  string `toml:"level"`  // debug | info | warn | error. Default: "info"
	Format string `toml:"format"` // text | json. Default: "text"
}

type ValidationError struct {
	Field   string
	Message string
}

// LoadGlobal reads the global config from the specified path.
// Returns model.ErrConfigNotFound if file doesn't exist.
// Returns model.ErrConfigInvalid if TOML parsing fails.
func LoadGlobal(path string) (*GlobalConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, model.ErrConfigNotFound
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := &GlobalConfig{}
	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("%w: %w", model.ErrConfigInvalid, err)
	}

	if cfg.Providers == nil {
		cfg.Providers = make(map[string]model.ProviderConfig)
	}

	return cfg, nil
}

// Save writes the global config to the specified path.
// Creates parent directories with 0700 (user-only) since the directory
// may also contain credentials. Sets file permissions to 0600.
func (c *GlobalConfig) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	data, err := toml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write: %w", err)
	}

	return nil
}

// ConfigPath returns the resolved global config file path.
// Priority: flag > env var > ~/.copygit/config
func ConfigPath(flagValue string) string { //nolint:revive // ConfigPath is the established API name
	if flagValue != "" {
		return flagValue
	}
	if env := os.Getenv("COPYGIT_CONFIG"); env != "" {
		return env
	}
	return DefaultConfigPath()
}

// ProvidersByNames returns the subset of providers matching the given names.
// Returns error if any name is not found.
func (c *GlobalConfig) ProvidersByNames(names []string) (map[string]model.ProviderConfig, error) {
	result := make(map[string]model.ProviderConfig)
	for _, name := range names {
		if prov, ok := c.Providers[name]; ok {
			result[name] = prov
		} else {
			return nil, fmt.Errorf("provider %q not found", name)
		}
	}
	return result, nil
}

// Validate checks the global config for correctness.
func (c *GlobalConfig) Validate() []ValidationError {
	var errs []ValidationError

	if c.Version == "" {
		errs = append(errs, ValidationError{
			Field:   "version",
			Message: "version is required",
		})
	}

	if len(c.Providers) == 0 {
		errs = append(errs, ValidationError{
			Field:   "providers",
			Message: "at least one provider is required",
		})
	}

	preferredCount := 0
	for name, prov := range c.Providers {
		if prov.Name == "" {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("providers.%s.name", name),
				Message: "name is required",
			})
		}
		if prov.Type == "" {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("providers.%s.type", name),
				Message: "type is required",
			})
		}
		if prov.BaseURL == "" {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("providers.%s.base_url", name),
				Message: "base_url is required",
			})
		} else if urlErrs := validateBaseURL(prov.BaseURL); len(urlErrs) > 0 {
			for _, msg := range urlErrs {
				errs = append(errs, ValidationError{
					Field:   fmt.Sprintf("providers.%s.base_url", name),
					Message: msg,
				})
			}
		}
		if prov.IsPreferred {
			preferredCount++
		}
	}

	if preferredCount > 1 {
		errs = append(errs, ValidationError{
			Field:   "providers",
			Message: "at most one provider may be preferred",
		})
	}

	return errs
}

// validateBaseURL checks a provider base URL for security issues.
// Returns a list of error messages (empty if valid).
func validateBaseURL(rawURL string) []string {
	var errs []string

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return []string{fmt.Sprintf("invalid URL: %v", err)}
	}

	// Only allow http/https schemes
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "https" && scheme != "http" {
		errs = append(errs, fmt.Sprintf("unsupported scheme %q: only http and https are allowed", parsed.Scheme))
		return errs
	}

	// Warn (not block) on plain HTTP
	if scheme != "https" {
		errs = append(errs, "base_url should use HTTPS to protect API tokens")
	}

	// Block private/internal IP ranges to prevent SSRF
	hostname := parsed.Hostname()
	if ip := net.ParseIP(hostname); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			errs = append(errs, fmt.Sprintf("base_url must not point to a private/internal IP address (%s)", hostname))
		}
	}

	// Block known metadata endpoints
	if hostname == "metadata.google.internal" {
		errs = append(errs, "base_url must not point to cloud metadata endpoints")
	}

	return errs
}
