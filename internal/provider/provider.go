package provider

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/imokhlis/copygit/internal/credential"
	"github.com/imokhlis/copygit/internal/git"
	"github.com/imokhlis/copygit/internal/model"
)

// Provider abstracts interactions with a Git hosting service.
// Per contracts/internal-interfaces.md.
type Provider interface {
	// Type returns the provider type (github, gitlab, gitea, generic).
	Type() model.ProviderType

	// Name returns the user-assigned provider name.
	Name() string

	// ValidateCredentials verifies authentication against the provider API.
	ValidateCredentials(ctx context.Context, cred *model.Credential) error

	// RepoExists checks if the remote repository exists.
	RepoExists(ctx context.Context, cred *model.Credential) (bool, error)

	// RemoteURL returns the full git remote URL based on auth method.
	RemoteURL(authMethod model.AuthMethod) string

	// RemoteName returns the git remote name (e.g., "copygit-github").
	RemoteName() string
}

// Registry is a container for all provider implementations.
type Registry struct {
	providers map[string]Provider
}

// NewRegistry creates an empty provider registry.
func NewRegistry() *Registry {
	return &Registry{providers: make(map[string]Provider)}
}

// Register adds a provider to the registry.
func (r *Registry) Register(name string, provider Provider) {
	r.providers[name] = provider
}

// Get retrieves a provider by name.
func (r *Registry) Get(name string) (Provider, error) {
	if prov, ok := r.providers[name]; ok {
		return prov, nil
	}
	return nil, fmt.Errorf("provider %q: %w", name, model.ErrProviderNotFound)
}

// All returns all registered providers.
func (r *Registry) All() map[string]Provider {
	return r.providers
}

// BuildRegistry constructs a registry from global provider configs.
func BuildRegistry(
	_ context.Context,
	configs map[string]model.ProviderConfig,
	_ credential.Manager,
	_ git.GitExecutor,
	logger *slog.Logger,
) (*Registry, error) {
	registry := NewRegistry()

	for name, cfg := range configs {
		var prov Provider

		switch cfg.Type {
		case model.ProviderGitHub:
			prov = NewGitHubProvider(cfg, logger)
		case model.ProviderGitLab:
			prov = NewGitLabProvider(cfg, logger)
		case model.ProviderGitea:
			prov = NewGiteaProvider(cfg, logger)
		case model.ProviderGeneric:
			prov = NewGenericProvider(cfg, logger)
		default:
			return nil, fmt.Errorf("unknown provider type: %v", cfg.Type)
		}

		registry.Register(name, prov)
	}

	return registry, nil
}
