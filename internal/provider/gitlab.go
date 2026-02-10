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

// GitLabProvider implements Provider for GitLab.com and self-hosted.
type GitLabProvider struct {
	config model.ProviderConfig
	logger *slog.Logger
}

// NewGitLabProvider creates a new GitLab provider.
func NewGitLabProvider(config model.ProviderConfig, logger *slog.Logger) *GitLabProvider {
	return &GitLabProvider{config: config, logger: logger}
}

func (p *GitLabProvider) Type() model.ProviderType { return model.ProviderGitLab }
func (p *GitLabProvider) Name() string             { return p.config.Name }

func (p *GitLabProvider) RemoteName() string {
	return "copygit-" + p.config.Name
}

func (p *GitLabProvider) RemoteURL(authMethod model.AuthMethod) string {
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

// ValidateCredentials verifies the token by calling GET /api/v4/user.
func (p *GitLabProvider) ValidateCredentials(ctx context.Context, cred *model.Credential) error {
	if cred.AuthMethod == model.AuthSSH {
		return nil
	}

	apiURL := strings.TrimSuffix(p.config.BaseURL, "/") + "/api/v4/user"

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("PRIVATE-TOKEN", cred.Token)

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

func (p *GitLabProvider) RepoExists(ctx context.Context, cred *model.Credential) (bool, error) { //nolint:revive // required by Provider interface
	return true, nil
}
