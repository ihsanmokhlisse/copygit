package git

import (
	"context"
	"log/slog"
	"os"
	"testing"
)

// TestRemoteManager_PushWithCredential verifies that PushWithCredential
// correctly calls git credential approve before pushing.
func TestRemoteManager_PushWithCredential(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Track which commands were called
	var credentialApproveCalled bool
	var pushCalled bool

	fakeExec := &FakeGitExecutor{
		RunWithStdinFunc: func(ctx context.Context, repoPath, stdin string, args ...string) (string, error) {
			if len(args) >= 2 && args[0] == "credential" && args[1] == "approve" {
				credentialApproveCalled = true
			}
			return "", nil
		},
		RunFunc: func(ctx context.Context, repoPath string, args ...string) (string, error) {
			if len(args) > 0 && args[0] == "push" {
				pushCalled = true
			}
			return "", nil
		},
	}

	rm := NewRemoteManager(fakeExec, logger)
	tmpDir := t.TempDir()

	// Test that PushWithCredential calls storeCredentialInGit and then Push
	err := rm.PushWithCredential(
		ctx,
		tmpDir,
		"origin",
		"https://github.com/user/repo.git",
		[]string{"main"},
		true, // tags
		false, // force
		"git",
		"test_token",
	)

	if err != nil {
		t.Errorf("PushWithCredential failed: %v", err)
	}

	if !credentialApproveCalled {
		t.Error("expected git credential approve to be called")
	} else {
		t.Log("✓ git credential approve was called")
	}

	if !pushCalled {
		t.Error("expected git push to be called")
	} else {
		t.Log("✓ git push was called")
	}
}

// TestRemoteManager_StoreCredentialInGit_URLParsing verifies URL parsing
// for different git URL formats.
func TestRemoteManager_StoreCredentialInGit_URLParsing(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	testCases := []struct {
		name         string
		url          string
		expectCalled bool // should git credential approve be called?
		description  string
	}{
		{
			name:         "HTTPS GitHub",
			url:          "https://github.com/user/repo.git",
			expectCalled: true,
			description:  "Standard HTTPS URL",
		},
		{
			name:         "HTTPS with userinfo",
			url:          "https://user@gitlab.com/path/to/repo.git",
			expectCalled: true,
			description:  "HTTPS with username in URL",
		},
		{
			name:         "HTTP URL",
			url:          "http://gitea.example.com:3000/user/repo.git",
			expectCalled: true,
			description:  "HTTP URL with custom port",
		},
		{
			name:         "SSH GitHub",
			url:          "git@github.com:user/repo.git",
			expectCalled: false,
			description:  "SSH URL (should skip credential storage)",
		},
		{
			name:         "SSH GitLab",
			url:          "ssh://git@gitlab.com/user/repo.git",
			expectCalled: false,
			description:  "SSH protocol URL",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			credentialApproveCalled := false

			fakeExec := &FakeGitExecutor{
				RunWithStdinFunc: func(ctx context.Context, repoPath, stdin string, args ...string) (string, error) {
					if len(args) >= 2 && args[0] == "credential" && args[1] == "approve" {
						credentialApproveCalled = true
					}
					return "", nil
				},
			}

			rm := NewRemoteManager(fakeExec, logger)
			tmpDir := t.TempDir()

			_ = rm.storeCredentialInGit(ctx, tmpDir, tc.url, "git", "test_token")

			if tc.expectCalled && !credentialApproveCalled {
				t.Errorf("%s: expected git credential approve to be called (%s)", tc.name, tc.description)
			} else if !tc.expectCalled && credentialApproveCalled {
				t.Errorf("%s: expected git credential approve NOT to be called (%s)", tc.name, tc.description)
			}

			if tc.expectCalled {
				t.Log("✓ Credential storage triggered for HTTPS URL")
			} else {
				t.Log("✓ Credential storage skipped for non-HTTPS URL")
			}
		})
	}
}

// TestRemoteManager_PushWithCredential_NoToken verifies graceful handling
// when no token is provided.
func TestRemoteManager_PushWithCredential_NoToken(t *testing.T) {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	var credentialApproveCalled bool
	var pushCalled bool

	fakeExec := &FakeGitExecutor{
		RunWithStdinFunc: func(ctx context.Context, repoPath, stdin string, args ...string) (string, error) {
			if len(args) >= 2 && args[0] == "credential" && args[1] == "approve" {
				credentialApproveCalled = true
			}
			return "", nil
		},
		RunFunc: func(ctx context.Context, repoPath string, args ...string) (string, error) {
			if len(args) > 0 && args[0] == "push" {
				pushCalled = true
			}
			return "", nil
		},
	}

	rm := NewRemoteManager(fakeExec, logger)
	tmpDir := t.TempDir()

	// Call with empty token
	err := rm.PushWithCredential(
		ctx,
		tmpDir,
		"origin",
		"https://github.com/user/repo.git",
		[]string{"main"},
		false,
		false,
		"git",
		"", // empty token
	)

	if err != nil {
		t.Errorf("PushWithCredential with empty token failed: %v", err)
	}

	if credentialApproveCalled {
		t.Error("expected git credential approve NOT to be called for empty token")
	} else {
		t.Log("✓ Credential storage skipped for empty token")
	}

	if !pushCalled {
		t.Error("expected git push to still be called despite empty token")
	} else {
		t.Log("✓ git push was called (graceful fallback)")
	}
}
