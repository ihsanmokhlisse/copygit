package credential

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/imokhlis/copygit/internal/model"
	"github.com/pelletier/go-toml/v2"
)

// FileResolver resolves credentials from the ~/.copygit/credentials file.
// File MUST have permissions 0600.
type FileResolver struct {
	logger   *slog.Logger
	filePath string
}

// NewFileResolver creates a new file-based credential resolver.
func NewFileResolver(logger *slog.Logger, filePath string) *FileResolver {
	return &FileResolver{logger: logger, filePath: filePath}
}

func (r *FileResolver) Name() string { return "file" }

func (r *FileResolver) IsAvailable() bool {
	info, err := os.Stat(r.filePath)
	if err != nil {
		return false
	}
	// Verify file permissions are 0600
	return info.Mode().Perm() == 0o600
}

func (r *FileResolver) Resolve(ctx context.Context, provider model.ProviderConfig) (*model.Credential, error) {
	// Verify permissions before reading
	info, err := os.Stat(r.filePath)
	if err != nil {
		return nil, model.ErrCredentialNotFound
	}
	if info.Mode().Perm() != 0o600 {
		r.logger.WarnContext(ctx, "credentials file has insecure permissions",
			"path", r.filePath,
			"mode", info.Mode().Perm())
		return nil, errors.New("credentials file must have permissions 0600")
	}

	data, err := os.ReadFile(r.filePath)
	if err != nil {
		return nil, model.ErrCredentialNotFound
	}

	// Parse TOML: each section is a provider name
	var creds map[string]struct {
		Token      string `toml:"token"`
		SSHKeyPath string `toml:"ssh_key_path"`
		Username   string `toml:"username"`
	}

	if err := toml.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("parse credentials: %w", err)
	}

	entry, ok := creds[provider.Name]
	if !ok {
		return nil, model.ErrCredentialNotFound
	}

	return &model.Credential{
		ProviderName: provider.Name,
		AuthMethod:   provider.AuthMethod,
		Token:        entry.Token,
		SSHKeyPath:   entry.SSHKeyPath,
		Username:     entry.Username,
	}, nil
}

func (r *FileResolver) Store(ctx context.Context, provider model.ProviderConfig, cred *model.Credential) error { //nolint:revive // required by CredentialResolver interface
	// Read existing
	var creds map[string]interface{}
	data, err := os.ReadFile(r.filePath)
	if err == nil {
		_ = toml.Unmarshal(data, &creds)
	}
	if creds == nil {
		creds = make(map[string]interface{})
	}

	// Add/update entry
	entry := map[string]string{}
	if cred.Token != "" {
		entry["token"] = cred.Token
	}
	if cred.SSHKeyPath != "" {
		entry["ssh_key_path"] = cred.SSHKeyPath
	}
	if cred.Username != "" {
		entry["username"] = cred.Username
	}
	creds[provider.Name] = entry

	// Write back with 0600 permissions
	newData, err := toml.Marshal(creds)
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}

	return os.WriteFile(r.filePath, newData, 0o600)
}

func (r *FileResolver) Delete(ctx context.Context, providerName string) error { //nolint:revive // required by CredentialResolver interface
	data, err := os.ReadFile(r.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // nothing to delete
		}
		return fmt.Errorf("read credentials: %w", err)
	}

	var creds map[string]interface{}
	if err := toml.Unmarshal(data, &creds); err != nil {
		return fmt.Errorf("parse credentials: %w", err)
	}

	delete(creds, providerName)

	newData, err := toml.Marshal(creds)
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}

	return os.WriteFile(r.filePath, newData, 0o600)
}
