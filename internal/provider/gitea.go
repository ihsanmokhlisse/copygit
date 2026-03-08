package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

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
	owner, repo, err := parseGiteaURL(remoteURL)
	if err != nil {
		return nil, fmt.Errorf("parse remote URL: %w", err)
	}

	apiURL := fmt.Sprintf("%s/api/v1/repos/%s/%s", strings.TrimSuffix(p.config.BaseURL, "/"), owner, repo)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "token "+cred.Token)

	resp, err := DefaultHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", model.ErrProviderUnreachable, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotFound:
		return nil, model.ErrRepositoryNotFound
	case http.StatusOK:
		break
	default:
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fetch repository metadata: status %d: %s", resp.StatusCode, string(body))
	}

	var giteaRepo struct {
		Private     bool   `json:"private"`
		Description string `json:"description"`
		Topics      []string `json:"topics"`
		Archived    bool   `json:"archived"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&giteaRepo); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	visibility := model.VisibilityPublic
	if giteaRepo.Private {
		visibility = model.VisibilityPrivate
	}

	return &model.RepoMetadata{
		Visibility:     visibility,
		Description:    giteaRepo.Description,
		Homepage:       "",
		Topics:         giteaRepo.Topics,
		Language:       "",
		License:        "",
		WikiEnabled:    false,
		IssuesEnabled:  false,
		Archived:       giteaRepo.Archived,
		SourceProvider: "gitea",
		FetchedAt:      time.Now(),
	}, nil
}

// CreateRepository creates a new Gitea repository with specified metadata.
func (p *GiteaProvider) CreateRepository(ctx context.Context, remoteURL string, metadata *model.RepoMetadata, cred *model.Credential) error { //nolint:revive // required by Provider interface
	_, repo, err := parseGiteaURL(remoteURL)
	if err != nil {
		return fmt.Errorf("parse remote URL: %w", err)
	}

	apiURL := fmt.Sprintf("%s/api/v1/user/repos", strings.TrimSuffix(p.config.BaseURL, "/"))

	private := metadata.Visibility == model.VisibilityPrivate

	payload := map[string]interface{}{
		"name":        repo,
		"private":     private,
		"description": metadata.Description,
		"topics":      metadata.Topics,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "token "+cred.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := DefaultHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %w", model.ErrProviderUnreachable, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusCreated:
		return nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("authentication failed: %w", model.ErrAuthFailed)
	default:
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create repository: status %d: %s", resp.StatusCode, string(respBody))
	}
}

// UpdateRepoMetadata updates metadata on existing Gitea repository.
func (p *GiteaProvider) UpdateRepoMetadata(ctx context.Context, remoteURL string, metadata *model.RepoMetadata, cred *model.Credential) error { //nolint:revive // required by Provider interface
	owner, repo, err := parseGiteaURL(remoteURL)
	if err != nil {
		return fmt.Errorf("parse remote URL: %w", err)
	}

	apiURL := fmt.Sprintf("%s/api/v1/repos/%s/%s", strings.TrimSuffix(p.config.BaseURL, "/"), owner, repo)

	private := metadata.Visibility == model.VisibilityPrivate

	payload := map[string]interface{}{
		"private":     private,
		"description": metadata.Description,
		"topics":      metadata.Topics,
		"archived":    metadata.Archived,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PATCH", apiURL, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "token "+cred.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := DefaultHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %w", model.ErrProviderUnreachable, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusNotFound:
		return model.ErrRepositoryNotFound
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("authentication failed: %w", model.ErrAuthFailed)
	default:
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update repository: status %d: %s", resp.StatusCode, string(respBody))
	}
}

// parseGiteaURL extracts owner and repo name from Gitea URL.
func parseGiteaURL(remoteURL string) (owner, repo string, err error) {
	// Handle SSH: git@gitea.com:owner/repo.git
	if strings.Contains(remoteURL, "git@") {
		parts := strings.Split(remoteURL, ":")
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid ssh URL format")
		}
		path := strings.TrimSuffix(parts[1], ".git")
		segments := strings.Split(path, "/")
		if len(segments) != 2 {
			return "", "", fmt.Errorf("invalid ssh URL format")
		}
		return segments[0], segments[1], nil
	}

	// Handle HTTPS: https://gitea.com/owner/repo.git
	remoteURL = strings.TrimSuffix(remoteURL, ".git")
	remoteURL = strings.TrimSuffix(remoteURL, "/")
	segments := strings.Split(remoteURL, "/")
	if len(segments) < 2 {
		return "", "", fmt.Errorf("invalid https URL format")
	}

	return segments[len(segments)-2], segments[len(segments)-1], nil
}
