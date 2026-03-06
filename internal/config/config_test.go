package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/imokhlis/copygit/internal/model"
)

func TestLoadGlobal_ValidFull(t *testing.T) {
	cfg, err := LoadGlobal("../../testdata/configs/valid_full.toml")
	require.NoError(t, err)
	assert.Equal(t, "1", cfg.Version)
	assert.Len(t, cfg.Providers, 3)
	assert.Equal(t, model.ProviderGitHub, cfg.Providers["my-github"].Type)
	assert.Equal(t, model.ProviderGitLab, cfg.Providers["work-gitlab"].Type)
	assert.Equal(t, model.ProviderGitea, cfg.Providers["self-gitea"].Type)
	assert.Equal(t, 5, cfg.Sync.MaxRetries)
	assert.True(t, cfg.Sync.PushTags)
}

func TestLoadGlobal_ValidMinimal(t *testing.T) {
	cfg, err := LoadGlobal("../../testdata/configs/valid_minimal.toml")
	require.NoError(t, err)
	assert.Len(t, cfg.Providers, 1)
	assert.Equal(t, 3, cfg.Sync.MaxRetries)
}

func TestLoadGlobal_NotFound(t *testing.T) {
	_, err := LoadGlobal("/nonexistent/path")
	assert.ErrorIs(t, err, model.ErrConfigNotFound)
}

func TestLoadGlobal_InvalidSyntax(t *testing.T) {
	_, err := LoadGlobal("../../testdata/configs/invalid_syntax.toml")
	assert.ErrorIs(t, err, model.ErrConfigInvalid)
}

func TestGlobalConfig_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	cfg := DefaultGlobalConfig()
	cfg.Providers["test"] = model.ProviderConfig{
		Name:       "test",
		Type:       model.ProviderGitHub,
		BaseURL:    "https://github.com",
		AuthMethod: model.AuthSSH,
	}

	err := cfg.Save(configPath)
	require.NoError(t, err)

	loaded, err := LoadGlobal(configPath)
	require.NoError(t, err)
	assert.Equal(t, cfg.Version, loaded.Version)
	assert.Len(t, loaded.Providers, 1)
}

func TestGlobalConfig_Validate(t *testing.T) {
	tests := []struct {
		name     string
		config   *GlobalConfig
		wantErrs int
	}{
		{
			name: "valid full config",
			config: &GlobalConfig{
				Version: "1",
				Providers: map[string]model.ProviderConfig{
					"gh": {Name: "gh", Type: model.ProviderGitHub, BaseURL: "https://github.com"},
				},
			},
			wantErrs: 0,
		},
		{
			name: "missing version",
			config: &GlobalConfig{
				Version: "",
				Providers: map[string]model.ProviderConfig{
					"gh": {Name: "gh", Type: model.ProviderGitHub, BaseURL: "https://github.com"},
				},
			},
			wantErrs: 1,
		},
		{
			name: "no providers",
			config: &GlobalConfig{
				Version:   "1",
				Providers: map[string]model.ProviderConfig{},
			},
			wantErrs: 1,
		},
		{
			name: "duplicate preferred",
			config: &GlobalConfig{
				Version: "1",
				Providers: map[string]model.ProviderConfig{
					"a": {Name: "a", Type: "github", BaseURL: "https://a.com", IsPreferred: true},
					"b": {Name: "b", Type: "gitlab", BaseURL: "https://b.com", IsPreferred: true},
				},
			},
			wantErrs: 1,
		},
		{
			name: "missing provider name",
			config: &GlobalConfig{
				Version: "1",
				Providers: map[string]model.ProviderConfig{
					"x": {Name: "", Type: "github", BaseURL: "https://x.com"},
				},
			},
			wantErrs: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := tt.config.Validate()
			assert.Len(t, errs, tt.wantErrs)
		})
	}
}

func TestProvidersByNames(t *testing.T) {
	cfg := &GlobalConfig{
		Providers: map[string]model.ProviderConfig{
			"gh": {Name: "gh"},
			"gl": {Name: "gl"},
		},
	}

	result, err := cfg.ProvidersByNames([]string{"gh"})
	require.NoError(t, err)
	assert.Len(t, result, 1)

	_, err = cfg.ProvidersByNames([]string{"nonexistent"})
	assert.Error(t, err)
}

func TestConfigPath(t *testing.T) {
	// Flag value takes priority
	assert.Equal(t, "/custom/path", ConfigPath("/custom/path"))

	// Env var
	os.Setenv("COPYGIT_CONFIG", "/env/path")
	assert.Equal(t, "/env/path", ConfigPath(""))
	os.Unsetenv("COPYGIT_CONFIG")
}

func TestRepoConfig_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()

	repoCfg := &model.RepoConfig{
		Version: "1",
		SyncTargets: []model.RepoSyncTarget{
			{ProviderName: "gh", RemoteURL: "git@github.com:u/r.git", Enabled: true},
			{ProviderName: "gl", RemoteURL: "https://gl.com/u/r.git", Enabled: false},
		},
	}

	err := SaveRepoConfig(tmpDir, repoCfg)
	require.NoError(t, err)

	loaded, err := LoadRepoConfig(tmpDir)
	require.NoError(t, err)
	assert.Equal(t, "1", loaded.Version)
	assert.Len(t, loaded.SyncTargets, 2)
	assert.True(t, loaded.SyncTargets[0].Enabled)
	assert.False(t, loaded.SyncTargets[1].Enabled)
}

func TestRepoConfig_EnabledProviderNames(t *testing.T) {
	cfg := &model.RepoConfig{
		SyncTargets: []model.RepoSyncTarget{
			{ProviderName: "gh", Enabled: true},
			{ProviderName: "gl", Enabled: false},
			{ProviderName: "gt", Enabled: true},
		},
	}

	names := cfg.EnabledProviderNames()
	assert.Len(t, names, 2)
	assert.Contains(t, names, "gh")
	assert.Contains(t, names, "gt")
}
