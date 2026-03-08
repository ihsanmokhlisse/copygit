package provider

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/imokhlis/copygit/internal/model"
)

// GiteaProvider implements Provider for Gitea instances.
type GiteaProvider struct {
	config model.ProviderConfig
	logger *slog.Logger
}

// NewGiteaProvider creates a new Gitea provider.
func NewGiteaProvider(config model.ProviderConfig, logger *slog.Logger) *GiteaProvider {
	return &GiteaProvider{config: config, logger: logger}
}

func (p *GiteaProvider) Type() model.ProviderType { return model.ProviderGitea }
func (p *GiteaProvider) Name() string             { return p.config.Name }

func (p *GiteaProvider) RemoteName() string {
	return "copygit-" + p.config.Name
}

func (p *GiteaProvider) RemoteURL(authMethod model.AuthMethod) string {
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

// ValidateCredentials verifies the token by calling GET /api/v1/user.
func (p *GiteaProvider) ValidateCredentials(ctx context.Context, cred *model.Credential) error {
	if cred.AuthMethod == model.AuthSSH {
		return nil
	}

	apiURL := strings.TrimSuffix(p.config.BaseURL, "/") + "/api/v1/user"

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "token "+cred.Token)

	resp, err := DefaultHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %w", model.ErrProviderUnreachable, err)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return model.ErrAuthFailed
	default:
		return fmt.Errorf("%w: unexpected status %d", model.ErrProviderUnreachable, resp.StatusCode)
	}
}

func (p *GiteaProvider) RepoExists(ctx context.Context, cred *model.Credential) (bool, error) { //nolint:revive // required by Provider interface
	return true, nil
}

// GetRepoMetadata fetches repository metadata from Gitea API.
func (p *GiteaProvider) GetRepoMetadata(ctx context.Context, remoteURL string, cred *model.Credential) (*model.RepoMetadata, error) { //nolint:revive // required by Provider interface
	// TODO: Implement Gitea metadata fetch
	return nil, fmt.Errorf("gitea metadata operations not yet implemented")
}

// CreateRepository creates a new Gitea repository with specified metadata.
func (p *GiteaProvider) CreateRepository(ctx context.Context, remoteURL string, metadata *model.RepoMetadata, cred *model.Credential) error { //nolint:revive // required by Provider interface
	// TODO: Implement Gitea repo creation
	return fmt.Errorf("gitea repository creation not yet implemented")
}

// UpdateRepoMetadata updates metadata on existing Gitea repository.
func (p *GiteaProvider) UpdateRepoMetadata(ctx context.Context, remoteURL string, metadata *model.RepoMetadata, cred *model.Credential) error { //nolint:revive // required by Provider interface
	// TODO: Implement Gitea metadata update
	return fmt.Errorf("gitea metadata update not yet implemented")
}
