package config

import (
	"fmt"
	"os"
	"path/filepath"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/imokhlis/copygit/internal/model"
)

// RepoConfigPath returns the expected .copygit.toml path for a repo.
func RepoConfigPath(repoRoot string) string {
	return filepath.Join(repoRoot, ".copygit.toml")
}

// LoadRepoConfig reads the per-repo configuration from repoRoot/.copygit.toml.
// Returns model.ErrConfigNotFound if the file doesn't exist.
// Returns model.ErrConfigInvalid if TOML parsing fails.
func LoadRepoConfig(repoRoot string) (*model.RepoConfig, error) {
	path := RepoConfigPath(repoRoot)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, model.ErrConfigNotFound
		}
		return nil, fmt.Errorf("read repo config: %w", err)
	}

	cfg := &model.RepoConfig{}
	if err := toml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("%w: %w", model.ErrConfigInvalid, err)
	}

	return cfg, nil
}

// SaveRepoConfig writes the per-repo configuration to repoRoot/.copygit.toml.
// Creates parent directories if needed. Sets file permissions to 0644.
func SaveRepoConfig(repoRoot string, cfg *model.RepoConfig) error {
	path := RepoConfigPath(repoRoot)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	data, err := toml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil { //nolint:gosec // config file, no secrets
		return fmt.Errorf("write: %w", err)
	}

	return nil
}

// ValidateRepoConfig checks the repo config for correctness.
func ValidateRepoConfig(cfg *model.RepoConfig) []ValidationError {
	var errs []ValidationError

	if cfg.Version == "" {
		errs = append(errs, ValidationError{
			Field:   "version",
			Message: "version is required",
		})
	}

	if len(cfg.SyncTargets) == 0 {
		errs = append(errs, ValidationError{
			Field:   "sync_targets",
			Message: "at least one sync target is required",
		})
	}

	for i, target := range cfg.SyncTargets {
		if target.ProviderName == "" {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("sync_targets[%d].provider", i),
				Message: "provider name is required",
			})
		}
		if target.RemoteURL == "" {
			errs = append(errs, ValidationError{
				Field:   fmt.Sprintf("sync_targets[%d].remote_url", i),
				Message: "remote_url is required",
			})
		}
	}

	return errs
}
