package command

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveRepoPath(t *testing.T) {
	t.Run("absolute path", func(t *testing.T) {
		got, err := resolveRepoPath("/tmp/test-repo")
		if err != nil {
			t.Fatal(err)
		}
		if got != "/tmp/test-repo" {
			t.Errorf("resolveRepoPath() = %q, want /tmp/test-repo", got)
		}
	})

	t.Run("relative path", func(t *testing.T) {
		got, err := resolveRepoPath("relative/path")
		if err != nil {
			t.Fatal(err)
		}
		if !filepath.IsAbs(got) {
			t.Errorf("resolveRepoPath() returned non-absolute path: %q", got)
		}
	})

	t.Run("home directory expansion", func(t *testing.T) {
		home, err := os.UserHomeDir()
		if err != nil {
			t.Skip("cannot get home dir")
		}
		got, err := resolveRepoPath("~/projects")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(got, home) {
			t.Errorf("resolveRepoPath(~/projects) = %q, should start with %q", got, home)
		}
		if !strings.HasSuffix(got, "projects") {
			t.Errorf("resolveRepoPath(~/projects) = %q, should end with 'projects'", got)
		}
	})

	t.Run("dot path resolves to cwd", func(t *testing.T) {
		cwd, _ := os.Getwd()
		got, err := resolveRepoPath(".")
		if err != nil {
			t.Fatal(err)
		}
		if got != cwd {
			t.Errorf("resolveRepoPath(.) = %q, want %q", got, cwd)
		}
	})
}
