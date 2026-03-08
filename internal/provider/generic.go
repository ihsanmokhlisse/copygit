package provider

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/imokhlis/copygit/internal/model"
)

// GenericProvider implements Provider for any generic Git server.
type GenericProvider struct {
	config model.ProviderConfig
	logger *slog.Logger
}

// NewGenericProvider creates a new Generic provider.
func NewGenericProvider(config model.ProviderConfig, logger *slog.Logger) *GenericProvider {
	return &GenericProvider{config: config, logger: logger}
}

func (p *GenericProvider) Type() model.ProviderType { return model.ProviderGeneric }
func (p *GenericProvider) Name() string             { return p.config.Name }

func (p *GenericProvider) RemoteName() string {
	return "copygit-" + p.config.Name
}

func (p *GenericProvider) RemoteURL(authMethod model.AuthMethod) string {
	base := strings.TrimSuffix(p.config.BaseURL, "/")
	switch authMethod {
	case model.AuthSSH:
		host := strings.TrimPrefix(base, "https://")
		host = strings.TrimPrefix(host, "http://")
		return fmt.Sprintf("git@%s:%%s.git", host)
	default:
		return base + "/%s.git"
	}
}

// ValidateCredentials is a no-op for generic providers.
func (p *GenericProvider) ValidateCredentials(ctx context.Context, cred *model.Credential) error { //nolint:revive // required by Provider interface
	return nil
}

func (p *GenericProvider) RepoExists(ctx context.Context, cred *model.Credential) (bool, error) { //nolint:revive // required by Provider interface
	return true, nil
}

// GetRepoMetadata is not supported for generic providers.
func (p *GenericProvider) GetRepoMetadata(ctx context.Context, remoteURL string, cred *model.Credential) (*model.RepoMetadata, error) { //nolint:revive // required by Provider interface
	return nil, fmt.Errorf("generic providers do not support metadata operations")
}

// CreateRepository is not supported for generic providers.
func (p *GenericProvider) CreateRepository(ctx context.Context, remoteURL string, metadata *model.RepoMetadata, cred *model.Credential) error { //nolint:revive // required by Provider interface
	return fmt.Errorf("generic providers do not support repository creation")
}

// UpdateRepoMetadata is not supported for generic providers.
func (p *GenericProvider) UpdateRepoMetadata(ctx context.Context, remoteURL string, metadata *model.RepoMetadata, cred *model.Credential) error { //nolint:revive // required by Provider interface
	return fmt.Errorf("generic providers do not support metadata updates")
}
