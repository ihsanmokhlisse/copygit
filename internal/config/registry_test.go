package config

import (
	"path/filepath"
	"testing"

	"github.com/imokhlis/copygit/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterAndFindRepo(t *testing.T) {
	registry := &model.RepoRegistry{
		Version: "1",
		Repos:   []model.RepoRegistration{},
	}

	reg, err := RegisterRepo(registry, "/path/to/repo", "my-repo")
	require.NoError(t, err)
	assert.Equal(t, "/path/to/repo", reg.Path)
	assert.Equal(t, "my-repo", reg.Alias)

	found, err := FindRepo(registry, "/path/to/repo")
	require.NoError(t, err)
	assert.Equal(t, "my-repo", found.Alias)

	found, err = FindRepoByAlias(registry, "my-repo")
	require.NoError(t, err)
	assert.Equal(t, "/path/to/repo", found.Path)
}

func TestUnregisterRepo(t *testing.T) {
	registry := &model.RepoRegistry{
		Version: "1",
		Repos:   []model.RepoRegistration{},
	}

	_, err := RegisterRepo(registry, "/path/to/repo", "")
	require.NoError(t, err)
	assert.Len(t, registry.Repos, 1)

	err = UnregisterRepo(registry, "/path/to/repo")
	require.NoError(t, err)
	assert.Empty(t, registry.Repos)

	err = UnregisterRepo(registry, "/nonexistent")
	assert.ErrorIs(t, err, model.ErrRepoNotFound)
}

func TestDuplicateRegistration(t *testing.T) {
	registry := &model.RepoRegistry{
		Version: "1",
		Repos:   []model.RepoRegistration{},
	}

	_, err := RegisterRepo(registry, "/path/to/repo", "")
	require.NoError(t, err)
	_, err = RegisterRepo(registry, "/path/to/repo", "")
	assert.ErrorIs(t, err, model.ErrRepoAlreadyRegistered)
}

func TestRegistrySaveAndLoad(t *testing.T) {
	tmpdir := t.TempDir()
	registryPath := filepath.Join(tmpdir, "repos.toml")

	registry := &model.RepoRegistry{
		Version: "1",
		Repos:   []model.RepoRegistration{},
	}

	_, err := RegisterRepo(registry, "/path1", "alias1")
	require.NoError(t, err)
	_, err = RegisterRepo(registry, "/path2", "alias2")
	require.NoError(t, err)

	err = SaveRepoRegistry(registryPath, registry)
	require.NoError(t, err)

	loaded, err := LoadRepoRegistry(registryPath)
	require.NoError(t, err)
	assert.Len(t, loaded.Repos, 2)
	assert.Equal(t, "alias1", loaded.Repos[0].Alias)
}

func TestValidateRepoRegistry(t *testing.T) {
	registry := &model.RepoRegistry{
		Version: "1",
		Repos: []model.RepoRegistration{
			{Path: "/path1", Alias: "alias1"},
			{Path: "/path2", Alias: "alias2"},
		},
	}

	errs := ValidateRepoRegistry(registry)
	assert.Empty(t, errs)

	badRegistry := &model.RepoRegistry{
		Version: "1",
		Repos: []model.RepoRegistration{
			{Path: "/path1"},
			{Path: "/path1"},
		},
	}

	errs = ValidateRepoRegistry(badRegistry)
	assert.True(t, len(errs) > 0)
}
