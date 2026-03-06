package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/imokhlis/copygit/internal/model"
)

func TestFullConfigWorkflow(t *testing.T) {
	tmpdir := t.TempDir()

	// 1. Create global config with provider
	globalPath := filepath.Join(tmpdir, "config.toml")
	globalCfg := DefaultGlobalConfig()
	globalCfg.Providers["github"] = model.ProviderConfig{
		Name:       "github",
		Type:       model.ProviderGitHub,
		BaseURL:    "https://github.com",
		AuthMethod: model.AuthSSH,
	}

	err := globalCfg.Save(globalPath)
	assert.NoError(t, err)

	// 2. Create repo config
	repoPath := filepath.Join(tmpdir, "my-repo")
	err = os.MkdirAll(repoPath, 0o755)
	assert.NoError(t, err)

	repoCfg := &model.RepoConfig{
		Version: "1",
		SyncTargets: []model.RepoSyncTarget{
			{
				ProviderName: "github",
				RemoteURL:    "https://github.com/user/my-repo.git",
				Enabled:      true,
			},
		},
	}

	err = SaveRepoConfig(repoPath, repoCfg)
	assert.NoError(t, err)

	// 3. Register in global registry
	registryPath := filepath.Join(tmpdir, "repos.toml")
	registry, err := LoadRepoRegistry(registryPath)
	assert.NoError(t, err)
	_, err = RegisterRepo(registry, repoPath, "my-repo-alias")
	assert.NoError(t, err)
	err = SaveRepoRegistry(registryPath, registry)
	assert.NoError(t, err)
	assert.NoError(t, err)

	// 4. Load everything back and verify
	loadedGlobal, err := LoadGlobal(globalPath)
	assert.NoError(t, err)
	assert.Len(t, loadedGlobal.Providers, 1)

	loadedRepo, err := LoadRepoConfig(repoPath)
	assert.NoError(t, err)
	assert.Len(t, loadedRepo.SyncTargets, 1)

	loadedRegistry, err := LoadRepoRegistry(registryPath)
	assert.NoError(t, err)
	assert.Len(t, loadedRegistry.Repos, 1)
	assert.Equal(t, "my-repo-alias", loadedRegistry.Repos[0].Alias)
}

func TestEmptyRegistryOnFirstLoad(t *testing.T) {
	tmpdir := t.TempDir()
	registryPath := filepath.Join(tmpdir, "repos.toml")

	registry, err := LoadRepoRegistry(registryPath)
	assert.NoError(t, err)
	assert.Empty(t, registry.Repos)
	assert.Equal(t, "1", registry.Version)
}

func TestConfigVersionHandling(t *testing.T) {
	tmpdir := t.TempDir()
	configPath := filepath.Join(tmpdir, "config.toml")

	cfg := &GlobalConfig{
		Version:   "1",
		Providers: make(map[string]model.ProviderConfig),
	}

	err := cfg.Save(configPath)
	assert.NoError(t, err)

	loaded, err := LoadGlobal(configPath)
	assert.NoError(t, err)
	assert.Equal(t, "1", loaded.Version)
}
