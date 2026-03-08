package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

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

// GetRepoMetadata fetches project metadata from GitLab API.
func (p *GitLabProvider) GetRepoMetadata(ctx context.Context, remoteURL string, cred *model.Credential) (*model.RepoMetadata, error) { //nolint:revive // required by Provider interface
	owner, repo, err := parseGitLabURL(remoteURL)
	if err != nil {
		return nil, fmt.Errorf("parse remote URL: %w", err)
	}

	projectID := url.QueryEscape(owner + "/" + repo)
	apiURL := fmt.Sprintf("%s/api/v4/projects/%s", strings.TrimSuffix(p.config.BaseURL, "/"), projectID)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("PRIVATE-TOKEN", cred.Token)

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
		return nil, fmt.Errorf("fetch project metadata: status %d: %s", resp.StatusCode, string(body))
	}

	var glProject struct {
		Private      bool     `json:"visibility"`
		Description  string   `json:"description"`
		WebURL       string   `json:"web_url"`
		TagList      []string `json:"tag_list"`
		License      struct {
			Name string `json:"name"`
		} `json:"license"`
		WikiEnabled    bool `json:"wiki_enabled"`
		IssuesEnabled  bool `json:"issues_enabled"`
		Archived       bool `json:"archived"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&glProject); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	visibility := model.VisibilityPublic
	if glProject.Private {
		visibility = model.VisibilityPrivate
	}

	// Try to extract homepage from web URL (not ideal but GitLab doesn't store separate homepage)
	homepage := ""

	return &model.RepoMetadata{
		Visibility:     visibility,
		Description:    glProject.Description,
		Homepage:       homepage,
		Topics:         glProject.TagList,
		Language:       "", // GitLab doesn't expose primary language
		License:        glProject.License.Name,
		WikiEnabled:    glProject.WikiEnabled,
		IssuesEnabled:  glProject.IssuesEnabled,
		Archived:       glProject.Archived,
		SourceProvider: "gitlab",
		FetchedAt:      time.Now(),
	}, nil
}

// CreateRepository creates a new GitLab project with specified metadata.
func (p *GitLabProvider) CreateRepository(ctx context.Context, remoteURL string, metadata *model.RepoMetadata, cred *model.Credential) error { //nolint:revive // required by Provider interface
	_, repo, err := parseGitLabURL(remoteURL)
	if err != nil {
		return fmt.Errorf("parse remote URL: %w", err)
	}

	apiURL := fmt.Sprintf("%s/api/v4/projects", strings.TrimSuffix(p.config.BaseURL, "/"))

	visibility := "private"
	if metadata.Visibility == model.VisibilityPublic {
		visibility = "public"
	} else if metadata.Visibility == model.VisibilityInternal {
		visibility = "internal"
	}

	payload := map[string]interface{}{
		"name":               repo,
		"visibility":         visibility,
		"description":        metadata.Description,
		"issues_enabled":     metadata.IssuesEnabled,
		"wiki_enabled":       metadata.WikiEnabled,
		"archived":           metadata.Archived,
		"tag_list":           metadata.Topics,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("PRIVATE-TOKEN", cred.Token)
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
		return fmt.Errorf("create project: status %d: %s", resp.StatusCode, string(respBody))
	}
}

// UpdateRepoMetadata updates metadata on existing GitLab project.
func (p *GitLabProvider) UpdateRepoMetadata(ctx context.Context, remoteURL string, metadata *model.RepoMetadata, cred *model.Credential) error { //nolint:revive // required by Provider interface
	owner, repo, err := parseGitLabURL(remoteURL)
	if err != nil {
		return fmt.Errorf("parse remote URL: %w", err)
	}

	projectID := url.QueryEscape(owner + "/" + repo)
	apiURL := fmt.Sprintf("%s/api/v4/projects/%s", strings.TrimSuffix(p.config.BaseURL, "/"), projectID)

	visibility := "private"
	if metadata.Visibility == model.VisibilityPublic {
		visibility = "public"
	} else if metadata.Visibility == model.VisibilityInternal {
		visibility = "internal"
	}

	payload := map[string]interface{}{
		"visibility":     visibility,
		"description":    metadata.Description,
		"issues_enabled": metadata.IssuesEnabled,
		"wiki_enabled":   metadata.WikiEnabled,
		"archived":       metadata.Archived,
		"tag_list":       metadata.Topics,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", apiURL, strings.NewReader(string(body)))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("PRIVATE-TOKEN", cred.Token)
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
		return fmt.Errorf("update project: status %d: %s", resp.StatusCode, string(respBody))
	}
}

// parseGitLabURL extracts owner and repo name from GitLab URL.
func parseGitLabURL(remoteURL string) (owner, repo string, err error) {
	// Handle SSH: git@gitlab.com:owner/repo.git
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

	// Handle HTTPS: https://gitlab.com/owner/repo.git
	remoteURL = strings.TrimSuffix(remoteURL, ".git")
	remoteURL = strings.TrimSuffix(remoteURL, "/")
	segments := strings.Split(remoteURL, "/")
	if len(segments) < 2 {
		return "", "", fmt.Errorf("invalid https URL format")
	}

	return segments[len(segments)-2], segments[len(segments)-1], nil
}
