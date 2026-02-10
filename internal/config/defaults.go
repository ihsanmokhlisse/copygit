package config

import (
	"os"
	"path/filepath"

	"github.com/imokhlis/copygit/internal/model"
)

// DefaultGlobalConfig returns a GlobalConfig with all default values set.
func DefaultGlobalConfig() *GlobalConfig {
	return &GlobalConfig{
		Version:   "1",
		Providers: make(map[string]model.ProviderConfig),
		Sync: SyncConfig{
			MaxRetries:     5,
			RetryBaseDelay: "5s",
			DryRunDefault:  false,
			PushTags:       true,
			PushBranches:   true,
		},
		Daemon: DaemonConfig{
			PollInterval: "30s",
			MaxRetries:   10,
			AutoStart:    false,
		},
		Log: LogConfig{
			Level:  "info",
			Format: "text",
		},
	}
}

// DefaultConfigDir returns ~/.copygit/ (or $COPYGIT_HOME if set).
func DefaultConfigDir() string {
	if home := os.Getenv("COPYGIT_HOME"); home != "" {
		return home
	}
	home, err := os.UserHomeDir()
	if err != nil {
		// Fall back to current directory as last resort; callers should
		// handle missing config gracefully.
		return ".copygit"
	}
	return filepath.Join(home, ".copygit")
}

// DefaultConfigPath returns ~/.copygit/config.
func DefaultConfigPath() string {
	return filepath.Join(DefaultConfigDir(), "config")
}

// DefaultRepoRegistryPath returns ~/.copygit/repos.toml.
func DefaultRepoRegistryPath() string {
	return filepath.Join(DefaultConfigDir(), "repos.toml")
}

// DefaultQueueDir returns ~/.copygit/queue/.
func DefaultQueueDir() string {
	return filepath.Join(DefaultConfigDir(), "queue")
}

// DefaultCredentialsPath returns ~/.copygit/credentials.
func DefaultCredentialsPath() string {
	return filepath.Join(DefaultConfigDir(), "credentials")
}

// DefaultLockPath returns ~/.copygit/lock.
func DefaultLockPath() string {
	return filepath.Join(DefaultConfigDir(), "lock")
}

// DefaultPIDFilePath returns ~/.copygit/daemon.pid.
func DefaultPIDFilePath() string {
	return filepath.Join(DefaultConfigDir(), "daemon.pid")
}

// DefaultStateDir returns ~/.copygit/state/.
func DefaultStateDir() string {
	return filepath.Join(DefaultConfigDir(), "state")
}
