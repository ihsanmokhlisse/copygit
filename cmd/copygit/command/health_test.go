package command

import (
	"testing"

	"github.com/imokhlis/copygit/internal/model"
)

func TestHealthCheckURL(t *testing.T) {
	tests := []struct {
		name string
		prov model.ProviderConfig
		want string
	}{
		{
			name: "github.com",
			prov: model.ProviderConfig{Type: model.ProviderGitHub, BaseURL: "https://github.com"},
			want: "https://api.github.com",
		},
		{
			name: "github enterprise",
			prov: model.ProviderConfig{Type: model.ProviderGitHub, BaseURL: "https://git.corp.com"},
			want: "https://git.corp.com/api/v3",
		},
		{
			name: "gitlab",
			prov: model.ProviderConfig{Type: model.ProviderGitLab, BaseURL: "https://gitlab.com"},
			want: "https://gitlab.com/api/v4/version",
		},
		{
			name: "gitea",
			prov: model.ProviderConfig{Type: model.ProviderGitea, BaseURL: "https://gitea.example.com"},
			want: "https://gitea.example.com/api/v1/version",
		},
		{
			name: "generic",
			prov: model.ProviderConfig{Type: model.ProviderGeneric, BaseURL: "https://git.example.com"},
			want: "https://git.example.com",
		},
		{
			name: "trailing slash",
			prov: model.ProviderConfig{Type: model.ProviderGitLab, BaseURL: "https://gitlab.com/"},
			want: "https://gitlab.com/api/v4/version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := healthCheckURL(tt.prov)
			if got != tt.want {
				t.Errorf("healthCheckURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
