package credential

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/imokhlis/copygit/internal/model"
)

// SSHResolver resolves credentials by checking for SSH keys.
type SSHResolver struct {
	logger *slog.Logger
}

// NewSSHResolver creates a new SSH resolver.
func NewSSHResolver(logger *slog.Logger) *SSHResolver {
	return &SSHResolver{logger: logger}
}

func (r *SSHResolver) Name() string { return "ssh" }

func (r *SSHResolver) IsAvailable() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	sshDir := filepath.Join(home, ".ssh")
	_, err = os.Stat(sshDir)
	return err == nil
}

func (r *SSHResolver) Resolve(ctx context.Context, provider model.ProviderConfig) (*model.Credential, error) { //nolint:revive // required by CredentialResolver interface
	if provider.AuthMethod != model.AuthSSH {
		return nil, model.ErrCredentialNotFound
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, model.ErrCredentialNotFound
	}

	// Check common SSH key paths
	keyPaths := []string{
		filepath.Join(home, ".ssh", "id_ed25519"),
		filepath.Join(home, ".ssh", "id_rsa"),
		filepath.Join(home, ".ssh", "id_ecdsa"),
	}

	for _, keyPath := range keyPaths {
		if _, err := os.Stat(keyPath); err == nil {
			return &model.Credential{
				ProviderName: provider.Name,
				AuthMethod:   model.AuthSSH,
				SSHKeyPath:   keyPath,
			}, nil
		}
	}

	return nil, model.ErrCredentialNotFound
}

func (r *SSHResolver) Store(ctx context.Context, provider model.ProviderConfig, cred *model.Credential) error { //nolint:revive // required by CredentialResolver interface
	// SSH keys are not managed by CopyGit; they're system-level.
	return model.ErrCredentialNotFound
}

func (r *SSHResolver) Delete(ctx context.Context, providerName string) error { //nolint:revive // required by CredentialResolver interface
	// SSH keys are not managed by CopyGit.
	return nil
}
