package credential

import (
	"context"
	"errors"
	"log/slog"

	keyring "github.com/zalando/go-keyring"

	"github.com/imokhlis/copygit/internal/model"
)

const keyringService = "copygit"

// KeyringResolver resolves credentials from the OS keychain.
type KeyringResolver struct {
	logger *slog.Logger
}

// NewKeyringResolver creates a new keyring resolver.
func NewKeyringResolver(logger *slog.Logger) *KeyringResolver {
	return &KeyringResolver{logger: logger}
}

func (r *KeyringResolver) Name() string { return "keyring" }

func (r *KeyringResolver) IsAvailable() bool {
	// Test if keyring is accessible by attempting a no-op
	_, err := keyring.Get(keyringService, "__copygit_test__")
	// ErrNotFound means keyring works, just no entry
	return errors.Is(err, keyring.ErrNotFound) || err == nil
}

func (r *KeyringResolver) Resolve(ctx context.Context, provider model.ProviderConfig) (*model.Credential, error) { //nolint:revive // required by CredentialResolver interface
	token, err := keyring.Get(keyringService, provider.Name)
	if err != nil {
		return nil, model.ErrCredentialNotFound
	}

	return &model.Credential{
		ProviderName: provider.Name,
		AuthMethod:   provider.AuthMethod,
		Token:        token,
	}, nil
}

func (r *KeyringResolver) Store(ctx context.Context, provider model.ProviderConfig, cred *model.Credential) error { //nolint:revive // required by CredentialResolver interface
	return keyring.Set(keyringService, provider.Name, cred.Token)
}

func (r *KeyringResolver) Delete(ctx context.Context, providerName string) error { //nolint:revive // required by CredentialResolver interface
	return keyring.Delete(keyringService, providerName)
}
