package credential

import (
	"context"

	"github.com/imokhlis/copygit/internal/model"
)

// CredentialResolver abstracts a single credential source.
// Multiple resolvers are chained by CredentialChain.
type CredentialResolver interface { //nolint:revive // established API name
	// Name returns the resolver name (e.g., "keyring", "ssh", "env").
	Name() string

	// Resolve attempts to find a credential for the given provider.
	Resolve(ctx context.Context, provider model.ProviderConfig) (*model.Credential, error)

	// Store persists a credential for the given provider.
	Store(ctx context.Context, provider model.ProviderConfig, cred *model.Credential) error

	// Delete removes a stored credential for the given provider.
	Delete(ctx context.Context, providerName string) error

	// IsAvailable returns true if this resolver is usable on the current system.
	IsAvailable() bool
}
