package provider

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/imokhlis/copygit/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestGitHubProvider_ValidateCredentials(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantErr    error
	}{
		{
			name:       "valid token",
			statusCode: 200,
			body:       `{"login":"testuser"}`,
			wantErr:    nil,
		},
		{
			name:       "invalid token",
			statusCode: 401,
			body:       `{"message":"Bad credentials"}`,
			wantErr:    model.ErrAuthFailed,
		},
		{
			name:       "server error",
			statusCode: 500,
			body:       `{"message":"Internal Server Error"}`,
			wantErr:    model.ErrProviderUnreachable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Non-github.com URLs use /api/v3/user path
				assert.Equal(t, "/api/v3/user", r.URL.Path)
				assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			p := NewGitHubProvider(model.ProviderConfig{
				Name:    "test",
				Type:    model.ProviderGitHub,
				BaseURL: server.URL,
			}, testLogger())

			ctx := context.Background()
			err := p.ValidateCredentials(ctx, &model.Credential{Token: "test-token"})
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGitHubProvider_RemoteURL(t *testing.T) {
	p := NewGitHubProvider(model.ProviderConfig{
		Name:    "test",
		BaseURL: "https://github.com",
	}, testLogger())

	t.Run("HTTPS", func(t *testing.T) {
		url := p.RemoteURL(model.AuthHTTPS)
		assert.Equal(t, "https://github.com/%s.git", url)
	})

	t.Run("SSH", func(t *testing.T) {
		url := p.RemoteURL(model.AuthSSH)
		assert.Equal(t, "git@github.com:%s.git", url)
	})

	t.Run("token defaults to HTTPS", func(t *testing.T) {
		url := p.RemoteURL(model.AuthToken)
		assert.Equal(t, "https://github.com/%s.git", url)
	})
}

func TestGitLabProvider_ValidateCredentials(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    error
	}{
		{"valid token", 200, nil},
		{"invalid token", 401, model.ErrAuthFailed},
		{"forbidden", 403, model.ErrAuthFailed},
		{"server error", 500, model.ErrProviderUnreachable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v4/user", r.URL.Path)
				assert.Equal(t, "test-token", r.Header.Get("PRIVATE-TOKEN"))
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			p := NewGitLabProvider(model.ProviderConfig{
				Name:    "test",
				Type:    model.ProviderGitLab,
				BaseURL: server.URL,
			}, testLogger())

			ctx := context.Background()
			err := p.ValidateCredentials(ctx, &model.Credential{Token: "test-token"})
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGiteaProvider_ValidateCredentials(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    error
	}{
		{"valid token", 200, nil},
		{"invalid token", 401, model.ErrAuthFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/v1/user", r.URL.Path)
				assert.Equal(t, "token test-token", r.Header.Get("Authorization"))
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			p := NewGiteaProvider(model.ProviderConfig{
				Name:    "test",
				Type:    model.ProviderGitea,
				BaseURL: server.URL,
			}, testLogger())

			ctx := context.Background()
			err := p.ValidateCredentials(ctx, &model.Credential{Token: "test-token"})
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestProvider_SSHSkipsValidation(t *testing.T) {
	ctx := context.Background()
	logger := testLogger()

	providers := []Provider{
		NewGitHubProvider(model.ProviderConfig{Name: "gh"}, logger),
		NewGitLabProvider(model.ProviderConfig{Name: "gl"}, logger),
		NewGiteaProvider(model.ProviderConfig{Name: "gt"}, logger),
		NewGenericProvider(model.ProviderConfig{Name: "gen"}, logger),
	}

	for _, prov := range providers {
		err := prov.ValidateCredentials(ctx, &model.Credential{AuthMethod: model.AuthSSH})
		assert.NoError(t, err, "provider %s should skip SSH validation", prov.Name())
	}
}

func TestGenericProvider_ValidateCredentials_NoOp(t *testing.T) {
	p := NewGenericProvider(model.ProviderConfig{Name: "gen"}, testLogger())
	err := p.ValidateCredentials(context.Background(), &model.Credential{Token: "any"})
	assert.NoError(t, err)
}

func TestProvider_RemoteName(t *testing.T) {
	tests := []struct {
		name     string
		provider Provider
		want     string
	}{
		{"github", NewGitHubProvider(model.ProviderConfig{Name: "my-github"}, testLogger()), "copygit-my-github"},
		{"gitlab", NewGitLabProvider(model.ProviderConfig{Name: "work-gl"}, testLogger()), "copygit-work-gl"},
		{"gitea", NewGiteaProvider(model.ProviderConfig{Name: "self"}, testLogger()), "copygit-self"},
		{"generic", NewGenericProvider(model.ProviderConfig{Name: "bare"}, testLogger()), "copygit-bare"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.provider.RemoteName())
		})
	}
}

func TestBuildRegistry(t *testing.T) {
	ctx := context.Background()
	logger := testLogger()
	configs := map[string]model.ProviderConfig{
		"gh":  {Name: "gh", Type: model.ProviderGitHub, BaseURL: "https://github.com"},
		"gl":  {Name: "gl", Type: model.ProviderGitLab, BaseURL: "https://gitlab.com"},
		"gt":  {Name: "gt", Type: model.ProviderGitea, BaseURL: "https://gitea.io"},
		"gen": {Name: "gen", Type: model.ProviderGeneric, BaseURL: "https://git.example.com"},
	}

	reg, err := BuildRegistry(ctx, configs, nil, nil, logger)
	require.NoError(t, err)

	gh, err := reg.Get("gh")
	require.NoError(t, err)
	assert.Equal(t, model.ProviderGitHub, gh.Type())

	gl, err := reg.Get("gl")
	require.NoError(t, err)
	assert.Equal(t, model.ProviderGitLab, gl.Type())

	gt, err := reg.Get("gt")
	require.NoError(t, err)
	assert.Equal(t, model.ProviderGitea, gt.Type())

	gen, err := reg.Get("gen")
	require.NoError(t, err)
	assert.Equal(t, model.ProviderGeneric, gen.Type())

	_, err = reg.Get("nonexistent")
	assert.ErrorIs(t, err, model.ErrProviderNotFound)
}

func TestBuildRegistry_UnknownType(t *testing.T) {
	ctx := context.Background()
	configs := map[string]model.ProviderConfig{
		"bad": {Name: "bad", Type: "bitbucket", BaseURL: "https://bb.org"},
	}

	_, err := BuildRegistry(ctx, configs, nil, nil, testLogger())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown provider type")
}

func TestRegistry_All(t *testing.T) {
	reg := NewRegistry()
	reg.Register("a", &FakeProvider{NameValue: "a"})
	reg.Register("b", &FakeProvider{NameValue: "b"})

	all := reg.All()
	assert.Len(t, all, 2)
}
