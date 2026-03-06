package command

import (
	"testing"

	"github.com/imokhlis/copygit/internal/model"
)

func TestRepoNameFromURL(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://github.com/user/myrepo.git", "myrepo"},
		{"https://github.com/user/myrepo", "myrepo"},
		{"git@github.com:user/myrepo.git", "myrepo"},
		{"git@gitlab.com:org/sub/myrepo.git", "myrepo"},
		{"https://gitlab.com/org/sub/myrepo.git", "myrepo"},
		{"https://github.com/user/repo/", "repo"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := repoNameFromURL(tt.url)
			if got != tt.want {
				t.Errorf("repoNameFromURL(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestOwnerFromURL(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://github.com/ihsan/copygit.git", "ihsan"},
		{"git@github.com:ihsan/copygit.git", "ihsan"},
		{"https://gitlab.com/myorg/myrepo.git", "myorg"},
		{"git@gitlab.com:myorg/myrepo.git", "myorg"},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := ownerFromURL(tt.url)
			if got != tt.want {
				t.Errorf("ownerFromURL(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestDetectProviderFromURL(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"https://github.com/user/repo.git", "github"},
		{"git@github.com:user/repo.git", "github"},
		{"https://gitlab.com/user/repo.git", "gitlab"},
		{"https://gitea.example.com/user/repo.git", "gitea"},
		{"https://example.com/user/repo.git", ""},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			got := detectProviderFromURL(tt.url)
			if got != tt.want {
				t.Errorf("detectProviderFromURL(%q) = %q, want %q", tt.url, got, tt.want)
			}
		})
	}
}

func TestGenerateRemoteURL(t *testing.T) {
	tests := []struct {
		name  string
		prov  model.ProviderConfig
		owner string
		repo  string
		want  string
	}{
		{
			name:  "github https",
			prov:  model.ProviderConfig{BaseURL: "https://github.com", AuthMethod: model.AuthHTTPS},
			owner: "ihsan",
			repo:  "copygit",
			want:  "https://github.com/ihsan/copygit.git",
		},
		{
			name:  "github ssh",
			prov:  model.ProviderConfig{BaseURL: "https://github.com", AuthMethod: model.AuthSSH},
			owner: "ihsan",
			repo:  "copygit",
			want:  "git@github.com:ihsan/copygit.git",
		},
		{
			name:  "gitlab token",
			prov:  model.ProviderConfig{BaseURL: "https://gitlab.com", AuthMethod: model.AuthToken},
			owner: "myorg",
			repo:  "myrepo",
			want:  "https://gitlab.com/myorg/myrepo.git",
		},
		{
			name:  "trailing slash",
			prov:  model.ProviderConfig{BaseURL: "https://github.com/", AuthMethod: model.AuthHTTPS},
			owner: "user",
			repo:  "repo",
			want:  "https://github.com/user/repo.git",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateRemoteURL(tt.prov, tt.owner, tt.repo)
			if got != tt.want {
				t.Errorf("GenerateRemoteURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
