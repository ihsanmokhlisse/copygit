package provider

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/imokhlis/copygit/internal/model"
)

// DefaultHTTPClient is a pre-configured HTTP client with secure timeouts.
// Used by all providers for API calls.
var DefaultHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        10,
		IdleConnTimeout:     30 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	},
}

// GitHubProvider implements Provider for GitHub.com and GitHub Enterprise.
type GitHubProvider struct {
	config model.ProviderConfig
	logger *slog.Logger
}

// NewGitHubProvider creates a new GitHub provider.
func NewGitHubProvider(config model.ProviderConfig, logger *slog.Logger) *GitHubProvider {
	return &GitHubProvider{config: config, logger: logger}
}

func (p *GitHubProvider) Type() model.ProviderType { return model.ProviderGitHub }
func (p *GitHubProvider) Name() string             { return p.config.Name }

func (p *GitHubProvider) RemoteName() string {
	return "copygit-" + p.config.Name
}

func (p *GitHubProvider) RemoteURL(authMethod model.AuthMethod) string {
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

// ValidateCredentials verifies the token by calling GET /user.
func (p *GitHubProvider) ValidateCredentials(ctx context.Context, cred *model.Credential) error {
	if cred.AuthMethod == model.AuthSSH {
		return nil // SSH auth validated by git itself
	}

	var apiURL string
	if strings.Contains(p.config.BaseURL, "github.com") {
		// GitHub.com: API is at api.github.com
		apiURL = "https://api.github.com/user"
	} else {
		// GitHub Enterprise: API is at <base>/api/v3/user
		apiURL = strings.TrimSuffix(p.config.BaseURL, "/") + "/api/v3/user"
	}

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, http.NoBody)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+cred.Token)
	req.Header.Set("Accept", "application/vnd.github+json")

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

// RepoExists checks if the repository exists on GitHub.
func (p *GitHubProvider) RepoExists(ctx context.Context, cred *model.Credential) (bool, error) { //nolint:revive // required by Provider interface
	// TODO: Call GET /repos/{owner}/{repo} to verify
	return true, nil
}

// GetRepoMetadata fetches repository metadata from GitHub API.
func (p *GitHubProvider) GetRepoMetadata(ctx context.Context, remoteURL string, cred *model.Credential) (*model.RepoMetadata, error) { //nolint:revive // required by Provider interface
	// TODO: Implement GitHub metadata fetch
	return nil, fmt.Errorf("github metadata operations not yet implemented")
}

// CreateRepository creates a new GitHub repository with specified metadata.
func (p *GitHubProvider) CreateRepository(ctx context.Context, remoteURL string, metadata *model.RepoMetadata, cred *model.Credential) error { //nolint:revive // required by Provider interface
	// TODO: Implement GitHub repo creation
	return fmt.Errorf("github repository creation not yet implemented")
}

// UpdateRepoMetadata updates metadata on existing GitHub repository.
func (p *GitHubProvider) UpdateRepoMetadata(ctx context.Context, remoteURL string, metadata *model.RepoMetadata, cred *model.Credential) error { //nolint:revive // required by Provider interface
	// TODO: Implement GitHub metadata update
	return fmt.Errorf("github metadata update not yet implemented")
}
