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
