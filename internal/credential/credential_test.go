package credential

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/imokhlis/copygit/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testLogger returns a silent logger for tests.
func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestCredentialChain_Resolve_FirstWins(t *testing.T) {
	ctx := context.Background()

	resolver1 := &FakeCredentialResolver{
		NameValue: "first",
		Available: true,
		ResolveFunc: func(_ context.Context, _ model.ProviderConfig) (*model.Credential, error) {
			return &model.Credential{Token: "from-first"}, nil
		},
	}
	resolver2 := &FakeCredentialResolver{
		NameValue: "second",
		Available: true,
		ResolveFunc: func(_ context.Context, _ model.ProviderConfig) (*model.Credential, error) {
			return &model.Credential{Token: "from-second"}, nil
		},
	}

	chain := NewCredentialChain(testLogger(), resolver1, resolver2)
	cred, err := chain.Resolve(ctx, model.ProviderConfig{Name: "test"})
	require.NoError(t, err)
	assert.Equal(t, "from-first", cred.Token)
}

func TestCredentialChain_Resolve_FallsThrough(t *testing.T) {
	ctx := context.Background()

	resolver1 := &FakeCredentialResolver{
		NameValue: "fail",
		Available: true,
		ResolveFunc: func(_ context.Context, _ model.ProviderConfig) (*model.Credential, error) {
			return nil, model.ErrCredentialNotFound
		},
	}
	resolver2 := &FakeCredentialResolver{
		NameValue: "success",
		Available: true,
		ResolveFunc: func(_ context.Context, _ model.ProviderConfig) (*model.Credential, error) {
			return &model.Credential{Token: "from-fallback"}, nil
		},
	}

	chain := NewCredentialChain(testLogger(), resolver1, resolver2)
	cred, err := chain.Resolve(ctx, model.ProviderConfig{Name: "test"})
	require.NoError(t, err)
	assert.Equal(t, "from-fallback", cred.Token)
}

func TestCredentialChain_Resolve_AllFail(t *testing.T) {
	ctx := context.Background()

	resolver := &FakeCredentialResolver{
		NameValue: "fail",
		Available: true,
		ResolveFunc: func(_ context.Context, _ model.ProviderConfig) (*model.Credential, error) {
			return nil, model.ErrCredentialNotFound
		},
	}

	chain := NewCredentialChain(testLogger(), resolver)
	_, err := chain.Resolve(ctx, model.ProviderConfig{Name: "test"})
	assert.ErrorIs(t, err, model.ErrCredentialNotFound)
}

func TestCredentialChain_Store_FirstAvailable(t *testing.T) {
	ctx := context.Background()
	stored := false

	resolver := &FakeCredentialResolver{
		NameValue: "store",
		Available: true,
		StoreFunc: func(_ context.Context, _ model.ProviderConfig, _ *model.Credential) error {
			stored = true
			return nil
		},
	}

	chain := NewCredentialChain(testLogger(), resolver)
	err := chain.Store(ctx, model.ProviderConfig{Name: "test"}, &model.Credential{Token: "t"})
	require.NoError(t, err)
	assert.True(t, stored)
}

func TestCredentialChain_Delete_AllResolvers(t *testing.T) {
	ctx := context.Background()
	deleteCount := 0

	resolver1 := &FakeCredentialResolver{
		NameValue: "r1",
		Available: true,
		DeleteFunc: func(_ context.Context, _ string) error {
			deleteCount++
			return nil
		},
	}
	resolver2 := &FakeCredentialResolver{
		NameValue: "r2",
		Available: true,
		DeleteFunc: func(_ context.Context, _ string) error {
			deleteCount++
			return nil
		},
	}

	chain := NewCredentialChain(testLogger(), resolver1, resolver2)
	err := chain.Delete(ctx, "test")
	require.NoError(t, err)
	assert.Equal(t, 2, deleteCount, "Delete should be called on all resolvers")
}

func TestDefaultChain_FiltersUnavailable(t *testing.T) {
	logger := testLogger()
	chain := DefaultChain(logger, "/nonexistent/path")

	// Should not panic and should have at least the env resolver (always available)
	assert.NotNil(t, chain)
	assert.Greater(t, len(chain.resolvers), 0)
}

func TestEnvResolver_Resolve(t *testing.T) {
	ctx := context.Background()
	resolver := NewEnvResolver(testLogger())

	t.Run("token found", func(t *testing.T) {
		t.Setenv("COPYGIT_TOKEN_MY_GITHUB", "test-token")

		cred, err := resolver.Resolve(ctx, model.ProviderConfig{Name: "my-github"})
		require.NoError(t, err)
		assert.Equal(t, "test-token", cred.Token)
		assert.Equal(t, model.AuthToken, cred.AuthMethod)
	})

	t.Run("token not found", func(t *testing.T) {
		_, err := resolver.Resolve(ctx, model.ProviderConfig{Name: "nonexistent"})
		assert.ErrorIs(t, err, model.ErrCredentialNotFound)
	})
}

func TestEnvResolver_StoreReturnsError(t *testing.T) {
	ctx := context.Background()
	resolver := NewEnvResolver(testLogger())

	err := resolver.Store(ctx, model.ProviderConfig{}, &model.Credential{})
	assert.ErrorIs(t, err, model.ErrCredentialNotFound)
}

func TestEnvVarName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"my-github", "COPYGIT_TOKEN_MY_GITHUB"},
		{"gitlab", "COPYGIT_TOKEN_GITLAB"},
		{"self-hosted-gitea", "COPYGIT_TOKEN_SELF_HOSTED_GITEA"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.want, envVarName(tt.input))
		})
	}
}

func TestFileResolver_ResolveAndStore(t *testing.T) {
	tmpDir := t.TempDir()
	credPath := filepath.Join(tmpDir, "credentials")

	resolver := NewFileResolver(testLogger(), credPath)
	ctx := context.Background()

	// Initially not available (file doesn't exist)
	assert.False(t, resolver.IsAvailable())

	// Store a credential
	provider := model.ProviderConfig{Name: "test", AuthMethod: model.AuthToken}
	cred := &model.Credential{Token: "my-secret-token"}
	err := resolver.Store(ctx, provider, cred)
	require.NoError(t, err)

	// Fix permissions for IsAvailable check
	require.NoError(t, os.Chmod(credPath, 0o600))
	assert.True(t, resolver.IsAvailable())

	// Resolve it back
	loaded, err := resolver.Resolve(ctx, provider)
	require.NoError(t, err)
	assert.Equal(t, "my-secret-token", loaded.Token)

	// Delete
	err = resolver.Delete(ctx, "test")
	require.NoError(t, err)

	// Should not find it anymore
	_, err = resolver.Resolve(ctx, provider)
	assert.ErrorIs(t, err, model.ErrCredentialNotFound)
}

func TestFileResolver_InsecurePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	credPath := filepath.Join(tmpDir, "credentials")

	// Create file with insecure permissions (world-readable)
	require.NoError(t, os.WriteFile(credPath, []byte("[test]\ntoken = \"abc\"\n"), 0o644)) //nolint:gosec // test file

	resolver := NewFileResolver(testLogger(), credPath)
	ctx := context.Background()

	// IsAvailable should return false
	assert.False(t, resolver.IsAvailable())

	// Resolve should fail with permission error
	_, err := resolver.Resolve(ctx, model.ProviderConfig{Name: "test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "0600")
}

func TestFakeManager(t *testing.T) {
	ctx := context.Background()
	mgr := NewFakeManager()

	provider := model.ProviderConfig{Name: "test"}
	cred := &model.Credential{Token: "fake-token"}

	// Store
	err := mgr.Store(ctx, provider, cred)
	require.NoError(t, err)

	// Resolve
	loaded, err := mgr.Resolve(ctx, provider)
	require.NoError(t, err)
	assert.Equal(t, "fake-token", loaded.Token)

	// Delete
	err = mgr.Delete(ctx, "test")
	require.NoError(t, err)

	// Not found
	_, err = mgr.Resolve(ctx, provider)
	assert.ErrorIs(t, err, model.ErrCredentialNotFound)
}

func TestChainManager_DelegatesToChain(t *testing.T) {
	ctx := context.Background()
	logger := testLogger()

	resolver := &FakeCredentialResolver{
		NameValue: "test",
		Available: true,
		ResolveFunc: func(_ context.Context, _ model.ProviderConfig) (*model.Credential, error) {
			return &model.Credential{Token: "chain-token"}, nil
		},
	}

	mgr := NewChainManagerWithResolvers(logger, resolver)

	cred, err := mgr.Resolve(ctx, model.ProviderConfig{Name: "test"})
	require.NoError(t, err)
	assert.Equal(t, "chain-token", cred.Token)
}
