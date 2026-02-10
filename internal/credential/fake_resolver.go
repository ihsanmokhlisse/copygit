package credential

import (
	"context"

	"github.com/imokhlis/copygit/internal/model"
)

// FakeCredentialResolver is a test double for CredentialResolver.
type FakeCredentialResolver struct {
	NameValue   string
	Available   bool
	ResolveFunc func(ctx context.Context, p model.ProviderConfig) (*model.Credential, error)
	StoreFunc   func(ctx context.Context, p model.ProviderConfig, c *model.Credential) error
	DeleteFunc  func(ctx context.Context, name string) error
}

func (f *FakeCredentialResolver) Name() string      { return f.NameValue }
func (f *FakeCredentialResolver) IsAvailable() bool { return f.Available }

func (f *FakeCredentialResolver) Resolve(ctx context.Context, p model.ProviderConfig) (*model.Credential, error) {
	if f.ResolveFunc != nil {
		return f.ResolveFunc(ctx, p)
	}
	return nil, model.ErrCredentialNotFound
}

func (f *FakeCredentialResolver) Store(ctx context.Context, p model.ProviderConfig, c *model.Credential) error {
	if f.StoreFunc != nil {
		return f.StoreFunc(ctx, p, c)
	}
	return nil
}

func (f *FakeCredentialResolver) Delete(ctx context.Context, name string) error {
	if f.DeleteFunc != nil {
		return f.DeleteFunc(ctx, name)
	}
	return nil
}
