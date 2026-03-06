package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/imokhlis/copygit/internal/model"
)

// LoadRepoRegistry reads the global repository registry from the given path.
// Returns an empty registry (not an error) if the file doesn't exist.
func LoadRepoRegistry(path string) (*model.RepoRegistry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &model.RepoRegistry{
				Version: "1",
				Repos:   []model.RepoRegistration{},
			}, nil
		}
		return nil, fmt.Errorf("read repo registry: %w", err)
	}

	reg := &model.RepoRegistry{}
	if err := toml.Unmarshal(data, reg); err != nil {
		return nil, fmt.Errorf("%w: %w", model.ErrConfigInvalid, err)
	}

	return reg, nil
}

// SaveRepoRegistry writes the repository registry to disk.
func SaveRepoRegistry(path string, reg *model.RepoRegistry) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	data, err := toml.Marshal(reg)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil { //nolint:gosec // config file, no secrets
		return fmt.Errorf("write: %w", err)
	}

	return nil
}

// RegisterRepo adds a new repository to the registry.
// Returns model.ErrRepoAlreadyRegistered if the path is already registered.
func RegisterRepo(reg *model.RepoRegistry, path, alias string) (*model.RepoRegistration, error) {
	for _, existing := range reg.Repos {
		if existing.Path == path {
			return nil, model.ErrRepoAlreadyRegistered
		}
	}

	registration := model.RepoRegistration{
		Path:         path,
		Alias:        alias,
		RegisteredAt: time.Now(),
	}

	reg.Repos = append(reg.Repos, registration)
	return &registration, nil
}

// UnregisterRepo removes a repository from the registry by path.
// Returns model.ErrRepoNotFound if the repo is not registered.
func UnregisterRepo(reg *model.RepoRegistry, path string) error {
	for i, r := range reg.Repos {
		if r.Path == path {
			reg.Repos = append(reg.Repos[:i], reg.Repos[i+1:]...)
			return nil
		}
	}
	return model.ErrRepoNotFound
}

// FindRepo returns the registration for the given path.
// Returns model.ErrRepoNotFound if not registered.
func FindRepo(reg *model.RepoRegistry, path string) (*model.RepoRegistration, error) {
	for i, r := range reg.Repos {
		if r.Path == path {
			return &reg.Repos[i], nil
		}
	}
	return nil, model.ErrRepoNotFound
}

// FindRepoByAlias returns the registration for the given alias.
// Returns model.ErrRepoNotFound if not found.
func FindRepoByAlias(reg *model.RepoRegistry, alias string) (*model.RepoRegistration, error) {
	for i, r := range reg.Repos {
		if r.Alias == alias {
			return &reg.Repos[i], nil
		}
	}
	return nil, model.ErrRepoNotFound
}

// UpdateLastSync updates the LastSyncTime for a repo.
// Returns model.ErrRepoNotFound if the repo is not registered.
func UpdateLastSync(reg *model.RepoRegistry, path string) error {
	for i, r := range reg.Repos {
		if r.Path == path {
			reg.Repos[i].LastSyncTime = time.Now()
			return nil
		}
	}
	return model.ErrRepoNotFound
}

// ValidRepos returns only repos whose paths still exist on disk.
func ValidRepos(reg *model.RepoRegistry) []model.RepoRegistration {
	var valid []model.RepoRegistration
	for _, r := range reg.Repos {
		if _, err := os.Stat(r.Path); err == nil {
			valid = append(valid, r)
		}
	}
	return valid
}

// ValidateRepoRegistry checks the registry for correctness.
func ValidateRepoRegistry(reg *model.RepoRegistry) []ValidationError {
	var errs []ValidationError

	if reg.Version == "" {
		errs = append(errs, ValidationError{
			Field: "version", Message: "version is required",
		})
	}

	paths := make(map[string]bool)
	aliases := make(map[string]bool)
	for i, repo := range reg.Repos {
		if repo.Path == "" {
			errs = append(errs, ValidationError{
				Field: fmt.Sprintf("repos[%d].path", i), Message: "path is required",
			})
		}
		if paths[repo.Path] {
			errs = append(errs, ValidationError{
				Field: fmt.Sprintf("repos[%d].path", i), Message: "duplicate path",
			})
		}
		paths[repo.Path] = true

		if repo.Alias != "" {
			if aliases[repo.Alias] {
				errs = append(errs, ValidationError{
					Field: fmt.Sprintf("repos[%d].alias", i), Message: "duplicate alias",
				})
			}
			aliases[repo.Alias] = true
		}
	}

	return errs
}
