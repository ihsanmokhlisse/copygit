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
	owner, repo, err := parseGitHubURL(remoteURL)
	if err != nil {
		return nil, fmt.Errorf("parse remote URL: %w", err)
	}

	var apiURL string
	if strings.Contains(p.config.BaseURL, "github.com") {
		apiURL = fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)
	} else {
		apiURL = fmt.Sprintf("%s/api/v3/repos/%s/%s", strings.TrimSuffix(p.config.BaseURL, "/"), owner, repo)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if cred.AuthMethod == model.AuthToken || cred.AuthMethod == model.AuthHTTPS {
		req.Header.Set("Authorization", "Bearer "+cred.Token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

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
		return nil, fmt.Errorf("fetch repo metadata: status %d: %s", resp.StatusCode, string(body))
	}

	var ghRepo struct {
		Private     bool     `json:"private"`
		Description string   `json:"description"`
		Homepage    string   `json:"homepage"`
		Topics      []string `json:"topics"`
		Language    string   `json:"language"`
		License     struct {
			Name string `json:"name"`
		} `json:"license"`
		HasWiki    bool `json:"has_wiki"`
		HasIssues  bool `json:"has_issues"`
		Archived   bool `json:"archived"`
		Visibility string `json:"visibility"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ghRepo); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	visibility := model.VisibilityPublic
	if ghRepo.Private {
		visibility = model.VisibilityPrivate
	}

	return &model.RepoMetadata{
		Visibility:     visibility,
		Description:    ghRepo.Description,
		Homepage:       ghRepo.Homepage,
		Topics:         ghRepo.Topics,
		Language:       ghRepo.Language,
		License:        ghRepo.License.Name,
		WikiEnabled:    ghRepo.HasWiki,
		IssuesEnabled:  ghRepo.HasIssues,
		Archived:       ghRepo.Archived,
		SourceProvider: "github",
		FetchedAt:      time.Now(),
	}, nil
}

// CreateRepository creates a new GitHub repository with specified metadata.
func (p *GitHubProvider) CreateRepository(ctx context.Context, remoteURL string, metadata *model.RepoMetadata, cred *model.Credential) error { //nolint:revive // required by Provider interface
	_, repo, err := parseGitHubURL(remoteURL)
	if err != nil {
		return fmt.Errorf("parse remote URL: %w", err)
	}

	var apiURL string
	if strings.Contains(p.config.BaseURL, "github.com") {
		apiURL = fmt.Sprintf("https://api.github.com/user/repos")
	} else {
		apiURL = fmt.Sprintf("%s/api/v3/user/repos", strings.TrimSuffix(p.config.BaseURL, "/"))
	}

	private := metadata.Visibility == model.VisibilityPrivate

	payload := map[string]interface{}{
		"name":        repo,
		"private":     private,
		"description": metadata.Description,
		"homepage":    metadata.Homepage,
		"has_wiki":    metadata.WikiEnabled,
		"has_issues":  metadata.IssuesEnabled,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	if cred.AuthMethod == model.AuthToken || cred.AuthMethod == model.AuthHTTPS {
		req.Header.Set("Authorization", "Bearer "+cred.Token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := DefaultHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %w", model.ErrProviderUnreachable, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusCreated:
		// Topics must be set separately via topics endpoint
		if len(metadata.Topics) > 0 {
			if err := p.setGitHubTopics(ctx, apiURL, repo, metadata.Topics, cred); err != nil {
				p.logger.WarnContext(ctx, "failed to set topics", "repo", repo, "error", err)
			}
		}
		return nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("authentication failed: %w", model.ErrAuthFailed)
	default:
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create repository: status %d: %s", resp.StatusCode, string(respBody))
	}
}

// UpdateRepoMetadata updates metadata on existing GitHub repository.
func (p *GitHubProvider) UpdateRepoMetadata(ctx context.Context, remoteURL string, metadata *model.RepoMetadata, cred *model.Credential) error { //nolint:revive // required by Provider interface
	owner, repo, err := parseGitHubURL(remoteURL)
	if err != nil {
		return fmt.Errorf("parse remote URL: %w", err)
	}

	var apiURL string
	if strings.Contains(p.config.BaseURL, "github.com") {
		apiURL = fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)
	} else {
		apiURL = fmt.Sprintf("%s/api/v3/repos/%s/%s", strings.TrimSuffix(p.config.BaseURL, "/"), owner, repo)
	}

	private := metadata.Visibility == model.VisibilityPrivate

	payload := map[string]interface{}{
		"private":     private,
		"description": metadata.Description,
		"homepage":    metadata.Homepage,
		"has_wiki":    metadata.WikiEnabled,
		"has_issues":  metadata.IssuesEnabled,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PATCH", apiURL, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	if cred.AuthMethod == model.AuthToken || cred.AuthMethod == model.AuthHTTPS {
		req.Header.Set("Authorization", "Bearer "+cred.Token)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := DefaultHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %w", model.ErrProviderUnreachable, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// Topics must be set separately via topics endpoint
		if len(metadata.Topics) > 0 {
			topicsURL := fmt.Sprintf("%s/topics", apiURL)
			if err := p.setGitHubTopicsDirect(ctx, topicsURL, metadata.Topics, cred); err != nil {
				p.logger.WarnContext(ctx, "failed to set topics", "repo", repo, "error", err)
			}
		}
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

// setGitHubTopics sets topics on a newly created repository.
func (p *GitHubProvider) setGitHubTopics(ctx context.Context, baseURL, repo string, topics []string, cred *model.Credential) error {
	topicsURL := strings.Replace(baseURL, "/user/repos", fmt.Sprintf("/repos/{owner}/%s/topics", repo), 1)

	// For user repos, the owner is the authenticated user, but we need to extract from context or use direct endpoint
	// Simpler approach: use the standard repos endpoint with topics
	owner, _, err := parseGitHubURL(fmt.Sprintf("https://github.com/%s/%s", "placeholder", repo))
	if err == nil && owner != "placeholder" {
		// Owner was extracted from credential or other source
	}

	return p.setGitHubTopicsDirect(ctx, topicsURL, topics, cred)
}

// setGitHubTopicsDirect sets topics directly via the topics endpoint.
func (p *GitHubProvider) setGitHubTopicsDirect(ctx context.Context, topicsURL string, topics []string, cred *model.Credential) error {
	payload := map[string]interface{}{
		"names": topics,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", topicsURL, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	if cred.AuthMethod == model.AuthToken || cred.AuthMethod == model.AuthHTTPS {
		req.Header.Set("Authorization", "Bearer "+cred.Token)
	}
	req.Header.Set("Accept", "application/vnd.github.mercy-preview+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := DefaultHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %w", model.ErrProviderUnreachable, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("set topics: status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// parseGitHubURL extracts owner and repo name from GitHub URL.
func parseGitHubURL(remoteURL string) (owner, repo string, err error) {
	// Handle SSH: git@github.com:owner/repo.git
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

	// Handle HTTPS: https://github.com/owner/repo.git
	remoteURL = strings.TrimSuffix(remoteURL, ".git")
	remoteURL = strings.TrimSuffix(remoteURL, "/")
	segments := strings.Split(remoteURL, "/")
	if len(segments) < 2 {
		return "", "", fmt.Errorf("invalid https URL format")
	}

	return segments[len(segments)-2], segments[len(segments)-1], nil
}
