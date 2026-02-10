package credential

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"github.com/imokhlis/copygit/internal/model"
)

// EnvResolver resolves credentials from environment variables.
// Pattern: COPYGIT_TOKEN_<PROVIDER_NAME> (dashes become underscores, uppercased).
type EnvResolver struct {
	logger *slog.Logger
}

// NewEnvResolver creates a new environment variable resolver.
func NewEnvResolver(logger *slog.Logger) *EnvResolver {
	return &EnvResolver{logger: logger}
}

func (r *EnvResolver) Name() string { return "env" }

func (r *EnvResolver) IsAvailable() bool { return true }

func (r *EnvResolver) Resolve(ctx context.Context, provider model.ProviderConfig) (*model.Credential, error) { //nolint:revive // required by CredentialResolver interface
	envName := envVarName(provider.Name)
	token := os.Getenv(envName)
	if token == "" {
		return nil, model.ErrCredentialNotFound
	}

	return &model.Credential{
		ProviderName: provider.Name,
		AuthMethod:   model.AuthToken,
		Token:        token,
	}, nil
}

func (r *EnvResolver) Store(ctx context.Context, provider model.ProviderConfig, cred *model.Credential) error { //nolint:revive // required by CredentialResolver interface
	// Cannot persist env vars from within the process.
	return model.ErrCredentialNotFound
}

func (r *EnvResolver) Delete(ctx context.Context, providerName string) error { //nolint:revive // required by CredentialResolver interface
	return nil
}

// envVarName converts provider name to env var.
// Dashes become underscores, result is uppercased.
// "my-github" → "COPYGIT_TOKEN_MY_GITHUB"
func envVarName(providerName string) string {
	name := strings.ReplaceAll(providerName, "-", "_")
	return "COPYGIT_TOKEN_" + strings.ToUpper(name)
}
