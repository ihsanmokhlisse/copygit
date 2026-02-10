package credential

import (
	"context"
	"log/slog"
	"strings"

	"github.com/imokhlis/copygit/internal/git"
	"github.com/imokhlis/copygit/internal/model"
)

// GitHelperResolver resolves credentials using git's credential helpers.
type GitHelperResolver struct {
	logger  *slog.Logger
	gitExec git.GitExecutor
}

// NewGitHelperResolver creates a new git credential helper resolver.
func NewGitHelperResolver(logger *slog.Logger) *GitHelperResolver {
	return &GitHelperResolver{logger: logger}
}

// SetGitExecutor sets the git executor (needed after construction due to
// circular dependency — credential depends on git, which is created later).
func (r *GitHelperResolver) SetGitExecutor(exec git.GitExecutor) {
	r.gitExec = exec
}

func (r *GitHelperResolver) Name() string { return "githelper" }

func (r *GitHelperResolver) IsAvailable() bool {
	return r.gitExec != nil
}

func (r *GitHelperResolver) Resolve(ctx context.Context, provider model.ProviderConfig) (*model.Credential, error) {
	if r.gitExec == nil {
		return nil, model.ErrCredentialNotFound
	}

	if provider.AuthMethod != model.AuthHTTPS && provider.AuthMethod != model.AuthToken {
		return nil, model.ErrCredentialNotFound
	}

	// Build credential fill input
	input := "protocol=https\nhost=" + extractHost(provider.BaseURL) + "\n\n"

	output, err := r.gitExec.RunWithStdin(ctx, ".", input, "credential", "fill")
	if err != nil {
		return nil, model.ErrCredentialNotFound
	}

	// Parse output
	cred := &model.Credential{
		ProviderName: provider.Name,
		AuthMethod:   provider.AuthMethod,
	}

	for _, line := range strings.Split(output, "\n") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		switch parts[0] {
		case "username":
			cred.Username = parts[1]
		case "password":
			cred.Token = parts[1]
		}
	}

	if cred.Token == "" && cred.Username == "" {
		return nil, model.ErrCredentialNotFound
	}

	return cred, nil
}

func (r *GitHelperResolver) Store(ctx context.Context, provider model.ProviderConfig, cred *model.Credential) error {
	if r.gitExec == nil {
		return model.ErrCredentialNotFound
	}

	input := "protocol=https\nhost=" + extractHost(provider.BaseURL) +
		"\nusername=" + cred.Username +
		"\npassword=" + cred.Token + "\n\n"

	_, err := r.gitExec.RunWithStdin(ctx, ".", input, "credential", "approve")
	return err
}

func (r *GitHelperResolver) Delete(ctx context.Context, providerName string) error { //nolint:revive // required by CredentialResolver interface
	// Cannot easily delete from git credential helper without the full URL
	return nil
}

// extractHost extracts the hostname from a URL.
func extractHost(url string) string {
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	parts := strings.SplitN(url, "/", 2)
	return parts[0]
}
