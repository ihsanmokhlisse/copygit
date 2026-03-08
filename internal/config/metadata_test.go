package config

import (
	"testing"

	"github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/imokhlis/copygit/internal/model"
)

func TestParseRepoConfigWithMetadata(t *testing.T) {
	tomlStr := `version = '1'

[metadata]
inherit_from = 'github'
visibility = 'private'
description = 'Inherited from source'
wiki_enabled = true
issues_enabled = true

[[sync_targets]]
provider = 'github'
remote_url = 'https://github.com/user/repo.git'
enabled = true

[[sync_targets]]
provider = 'gitlab'
remote_url = 'https://gitlab.com/user/repo.git'
enabled = true
`

	var cfg model.RepoConfig
	err := toml.Unmarshal([]byte(tomlStr), &cfg)
	require.NoError(t, err)

	assert.Equal(t, "1", cfg.Version)
	assert.NotNil(t, cfg.Metadata)
	assert.Equal(t, "github", cfg.Metadata.InheritFrom)
	assert.Equal(t, "private", cfg.Metadata.Visibility)
	assert.Equal(t, 2, len(cfg.SyncTargets))
	assert.Equal(t, "github", cfg.SyncTargets[0].ProviderName)
	assert.Equal(t, "gitlab", cfg.SyncTargets[1].ProviderName)
}

func TestRepoConfigMetadataDefaults(t *testing.T) {
	tomlStr := `version = '1'

[[sync_targets]]
provider = 'github'
remote_url = 'https://github.com/user/repo.git'
enabled = true
`

	var cfg model.RepoConfig
	err := toml.Unmarshal([]byte(tomlStr), &cfg)
	require.NoError(t, err)

	// No metadata section = should be nil (optional)
	assert.Nil(t, cfg.Metadata)
	assert.Equal(t, 1, len(cfg.SyncTargets))
}

func TestRepoSyncTargetToRepoSyncTarget(t *testing.T) {
	target := model.RepoSyncTargetWithOverrides{
		ProviderName: "github",
		RemoteURL:    "https://github.com/user/repo.git",
		Enabled:      true,
	}

	base := target.ToRepoSyncTarget()
	assert.Equal(t, "github", base.ProviderName)
	assert.Equal(t, "https://github.com/user/repo.git", base.RemoteURL)
	assert.True(t, base.Enabled)
}
