package credential

import (
	"context"
	"log/slog"

	"github.com/imokhlis/copygit/internal/model"
)

// CredentialChain resolves credentials by trying multiple resolvers in order.
// Resolution order: Keyring → SSH → GitHelper → Env → File.
type CredentialChain struct { //nolint:revive // established API name
	resolvers []CredentialResolver
	logger    *slog.Logger
}

// NewCredentialChain creates a chain with the given resolvers.
func NewCredentialChain(logger *slog.Logger, resolvers ...CredentialResolver) *CredentialChain {
	return &CredentialChain{
		resolvers: resolvers,
		logger:    logger,
	}
}

// DefaultChain creates the standard resolution chain.
func DefaultChain(logger *slog.Logger, credFilePath string) *CredentialChain {
	resolvers := []CredentialResolver{
		NewKeyringResolver(logger),
		NewSSHResolver(logger),
		NewGitHelperResolver(logger),
		NewEnvResolver(logger),
		NewFileResolver(logger, credFilePath),
	}

	// Filter to only available resolvers
	var available []CredentialResolver
	for _, r := range resolvers {
		if r.IsAvailable() {
			available = append(available, r)
		} else {
			logger.Debug("credential resolver unavailable", "resolver", r.Name())
		}
	}

	return NewCredentialChain(logger, available...)
}

// Resolve tries each resolver in order until one succeeds.
// Returns model.ErrCredentialNotFound if all fail.
func (c *CredentialChain) Resolve(ctx context.Context, provider model.ProviderConfig) (*model.Credential, error) {
	for _, resolver := range c.resolvers {
		cred, err := resolver.Resolve(ctx, provider)
		if err == nil {
			c.logger.DebugContext(ctx, "credential resolved",
				"provider", provider.Name,
				"resolver", resolver.Name())
			return cred, nil
		}
		c.logger.DebugContext(ctx, "resolver failed",
			"provider", provider.Name,
			"resolver", resolver.Name(),
			"error", err)
	}

	return nil, model.ErrCredentialNotFound
}

// Store attempts to store a credential using the first available resolver.
func (c *CredentialChain) Store(ctx context.Context, provider model.ProviderConfig, cred *model.Credential) error {
	for _, resolver := range c.resolvers {
		err := resolver.Store(ctx, provider, cred)
		if err == nil {
			c.logger.DebugContext(ctx, "credential stored",
				"provider", provider.Name,
				"resolver", resolver.Name())
			return nil
		}
	}
	return model.ErrCredentialNotFound
}

// Delete removes a credential from all resolvers.
func (c *CredentialChain) Delete(ctx context.Context, providerName string) error {
	var lastErr error
	for _, resolver := range c.resolvers {
		if err := resolver.Delete(ctx, providerName); err != nil {
			lastErr = err
		}
	}
	return lastErr
}
