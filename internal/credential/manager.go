package credential

import (
	"context"
	"log/slog"

	"github.com/imokhlis/copygit/internal/model"
)

// Manager handles credential resolution and storage.
// It delegates to CredentialChain for multi-source resolution.
type Manager interface {
	Resolve(ctx context.Context, provider model.ProviderConfig) (*model.Credential, error)
	Store(ctx context.Context, provider model.ProviderConfig, cred *model.Credential) error
	Delete(ctx context.Context, providerName string) error
}

// ChainManager wraps a CredentialChain as a Manager.
type ChainManager struct {
	chain  *CredentialChain
	logger *slog.Logger
}

// NewChainManager creates a Manager backed by the default credential chain.
func NewChainManager(logger *slog.Logger, credFilePath string) *ChainManager {
	return &ChainManager{
		chain:  DefaultChain(logger, credFilePath),
		logger: logger,
	}
}

// NewChainManagerWithResolvers creates a Manager backed by specific resolvers.
func NewChainManagerWithResolvers(logger *slog.Logger, resolvers ...CredentialResolver) *ChainManager {
	return &ChainManager{
		chain:  NewCredentialChain(logger, resolvers...),
		logger: logger,
	}
}

func (m *ChainManager) Resolve(ctx context.Context, provider model.ProviderConfig) (*model.Credential, error) {
	return m.chain.Resolve(ctx, provider)
}

func (m *ChainManager) Store(ctx context.Context, provider model.ProviderConfig, cred *model.Credential) error {
	return m.chain.Store(ctx, provider, cred)
}

func (m *ChainManager) Delete(ctx context.Context, providerName string) error {
	return m.chain.Delete(ctx, providerName)
}

// FakeManager is a test double for Manager.
type FakeManager struct {
	Credentials map[string]*model.Credential
	StoreErr    error
	DeleteErr   error
}

// NewFakeManager creates a fake credential manager for tests.
func NewFakeManager() *FakeManager {
	return &FakeManager{
		Credentials: make(map[string]*model.Credential),
	}
}

func (f *FakeManager) Resolve(ctx context.Context, provider model.ProviderConfig) (*model.Credential, error) { //nolint:revive // required by Manager interface
	if cred, ok := f.Credentials[provider.Name]; ok {
		return cred, nil
	}
	return nil, model.ErrCredentialNotFound
}

func (f *FakeManager) Store(ctx context.Context, provider model.ProviderConfig, cred *model.Credential) error { //nolint:revive // required by Manager interface
	if f.StoreErr != nil {
		return f.StoreErr
	}
	f.Credentials[provider.Name] = cred
	return nil
}

func (f *FakeManager) Delete(ctx context.Context, providerName string) error { //nolint:revive // required by Manager interface
	if f.DeleteErr != nil {
		return f.DeleteErr
	}
	delete(f.Credentials, providerName)
	return nil
}
