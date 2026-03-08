package provider

import (
	"context"

	"github.com/imokhlis/copygit/internal/model"
)

// FakeProvider is a test double for Provider.
type FakeProvider struct {
	TypeValue        model.ProviderType
	NameValue        string
	ValidateErr      error
	RepoExistsResult bool
	RepoExistsErr    error
	RemoteURLValue   string
	GetMetadataResult *model.RepoMetadata
	GetMetadataErr    error
	CreateRepoErr     error
	UpdateMetadataErr error
}

func (f *FakeProvider) Type() model.ProviderType { return f.TypeValue }
func (f *FakeProvider) Name() string             { return f.NameValue }
func (f *FakeProvider) RemoteName() string       { return "copygit-" + f.NameValue }

func (f *FakeProvider) RemoteURL(authMethod model.AuthMethod) string { //nolint:revive // required by Provider interface
	return f.RemoteURLValue
}

func (f *FakeProvider) ValidateCredentials(ctx context.Context, cred *model.Credential) error { //nolint:revive // required by Provider interface
	return f.ValidateErr
}

func (f *FakeProvider) RepoExists(ctx context.Context, cred *model.Credential) (bool, error) { //nolint:revive // required by Provider interface
	return f.RepoExistsResult, f.RepoExistsErr
}

func (f *FakeProvider) GetRepoMetadata(ctx context.Context, remoteURL string, cred *model.Credential) (*model.RepoMetadata, error) { //nolint:revive // required by Provider interface
	return f.GetMetadataResult, f.GetMetadataErr
}

func (f *FakeProvider) CreateRepository(ctx context.Context, remoteURL string, metadata *model.RepoMetadata, cred *model.Credential) error { //nolint:revive // required by Provider interface
	return f.CreateRepoErr
}

func (f *FakeProvider) UpdateRepoMetadata(ctx context.Context, remoteURL string, metadata *model.RepoMetadata, cred *model.Credential) error { //nolint:revive // required by Provider interface
	return f.UpdateMetadataErr
}

// FakeRegistry wraps a provider registry for tests.
type FakeRegistry struct {
	Providers map[string]Provider
}

func NewFakeRegistry() *FakeRegistry {
	return &FakeRegistry{Providers: make(map[string]Provider)}
}

func (r *FakeRegistry) Get(name string) (Provider, error) {
	if prov, ok := r.Providers[name]; ok {
		return prov, nil
	}
	return nil, model.ErrProviderNotFound
}
