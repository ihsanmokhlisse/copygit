package config

import (
	"testing"

	"github.com/imokhlis/copygit/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepoConfigSaveAndLoad(t *testing.T) {
	tmpdir := t.TempDir()

	// Create and save
	cfg := &model.RepoConfig{
		Version: "1",
		SyncTargets: []model.RepoSyncTarget{
			{
				ProviderName: "github",
				RemoteURL:    "https://github.com/user/repo.git",
				Enabled:      true,
			},
		},
	}

	err := SaveRepoConfig(tmpdir, cfg)
	require.NoError(t, err)

	// Load and verify
	loaded, err := LoadRepoConfig(tmpdir)
	require.NoError(t, err)
	assert.Equal(t, "1", loaded.Version)
	assert.Len(t, loaded.SyncTargets, 1)
	assert.Equal(t, "github", loaded.SyncTargets[0].ProviderName)
}

func TestLoadRepoConfigNotFound(t *testing.T) {
	tmpdir := t.TempDir()
	_, err := LoadRepoConfig(tmpdir)
	assert.Equal(t, model.ErrConfigNotFound, err)
}

func TestValidateRepoConfig(t *testing.T) {
	cfg := &model.RepoConfig{
		Version: "1",
		SyncTargets: []model.RepoSyncTarget{
			{
				ProviderName: "github",
				RemoteURL:    "https://github.com/user/repo.git",
				Enabled:      true,
			},
		},
	}

	errs := ValidateRepoConfig(cfg)
	assert.Empty(t, errs)

	// Invalid: missing remote URL
	badCfg := &model.RepoConfig{
		Version: "1",
		SyncTargets: []model.RepoSyncTarget{
			{
				ProviderName: "github",
				RemoteURL:    "",
				Enabled:      true,
			},
		},
	}

	errs = ValidateRepoConfig(badCfg)
	assert.True(t, len(errs) > 0)
}
