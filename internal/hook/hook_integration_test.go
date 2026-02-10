package hook

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createGitRepo creates a real git repository using git CLI.
// It creates a temp directory, initializes git, makes an initial commit,
// and returns the repo path.
func createGitRepo(t *testing.T) string {
	tmpDir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "git init failed: %s", output)

	// Configure git user for commits
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run(), "failed to set git user email")

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run(), "failed to set git user name")

	// Create initial commit
	testFile := filepath.Join(tmpDir, "README.md")
	err = os.WriteFile(testFile, []byte("# Test Repo\n"), 0o644) //nolint:gosec // test file
	require.NoError(t, err)

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	output, err = cmd.CombinedOutput()
	require.NoError(t, err, "git commit failed: %s", output)

	return tmpDir
}

// TestHookManagerIntegration tests HookManager with a real git repository.
func TestHookManagerIntegration(t *testing.T) {
	// Create a real git repository
	repoPath := createGitRepo(t)

	// Verify .git directory exists
	gitDir := filepath.Join(repoPath, ".git")
	_, err := os.Stat(gitDir)
	require.NoError(t, err, ".git directory should exist")

	// Verify hooks directory exists
	hooksDir := filepath.Join(gitDir, "hooks")
	_, err = os.Stat(hooksDir)
	require.NoError(t, err, "hooks directory should exist")

	manager := NewHookManager(mockLogger())

	// Test 1: IsInstalled returns false before install
	t.Run("IsInstalled_before_install", func(t *testing.T) {
		installed, err := manager.IsInstalled(context.Background(), repoPath)
		require.NoError(t, err)
		assert.False(t, installed, "hook should not be installed before Install()")
	})

	// Test 2: Install creates executable hook file
	t.Run("Install_creates_executable", func(t *testing.T) {
		err := manager.Install(context.Background(), repoPath)
		require.NoError(t, err)

		hookPath := filepath.Join(hooksDir, "post-push")
		info, err := os.Stat(hookPath)
		require.NoError(t, err, "hook file should exist after Install()")

		// Verify executable bit is set (mode should have execute bits)
		assert.True(t, info.Mode()&0o111 != 0, "hook file should be executable")

		// Verify content
		content, err := os.ReadFile(hookPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), HookMarkerStart)
		assert.Contains(t, string(content), HookMarkerEnd)
		assert.Contains(t, string(content), "copygit push --from-hook")
	})

	// Test 3: IsInstalled returns true after install
	t.Run("IsInstalled_after_install", func(t *testing.T) {
		installed, err := manager.IsInstalled(context.Background(), repoPath)
		require.NoError(t, err)
		assert.True(t, installed, "hook should be installed after Install()")
	})

	// Test 4: Install is idempotent (calling again doesn't duplicate)
	t.Run("Install_idempotent", func(t *testing.T) {
		err := manager.Install(context.Background(), repoPath)
		require.NoError(t, err)

		hookPath := filepath.Join(hooksDir, "post-push")
		content, err := os.ReadFile(hookPath)
		require.NoError(t, err)

		// Should only have one set of markers
		count := 0
		for _, line := range []byte(content) {
			if line == '\n' {
				count++
			}
		}
		// Marker appears twice (start and end), so count occurrences
		markerCount := 0
		for i := 0; i < len(content)-len(HookMarkerStart); i++ {
			if string(content[i:i+len(HookMarkerStart)]) == HookMarkerStart {
				markerCount++
			}
		}
		assert.Equal(t, 1, markerCount, "should have exactly one start marker after idempotent install")
	})

	// Test 5: Uninstall removes the hook file
	t.Run("Uninstall_removes_hook", func(t *testing.T) {
		err := manager.Uninstall(context.Background(), repoPath)
		require.NoError(t, err)

		hookPath := filepath.Join(hooksDir, "post-push")
		_, err = os.Stat(hookPath)
		assert.True(t, os.IsNotExist(err), "hook file should be removed after Uninstall()")
	})

	// Test 6: IsInstalled returns false after uninstall
	t.Run("IsInstalled_after_uninstall", func(t *testing.T) {
		installed, err := manager.IsInstalled(context.Background(), repoPath)
		require.NoError(t, err)
		assert.False(t, installed, "hook should not be installed after Uninstall()")
	})
}

// TestHookManagerIntegration_Reinstall tests reinstalling the hook after uninstall.
func TestHookManagerIntegration_Reinstall(t *testing.T) {
	repoPath := createGitRepo(t)
	hooksDir := filepath.Join(repoPath, ".git", "hooks")
	manager := NewHookManager(mockLogger())

	// Install, uninstall, then reinstall
	err := manager.Install(context.Background(), repoPath)
	require.NoError(t, err)

	err = manager.Uninstall(context.Background(), repoPath)
	require.NoError(t, err)

	// Reinstall should work
	err = manager.Install(context.Background(), repoPath)
	require.NoError(t, err)

	hookPath := filepath.Join(hooksDir, "post-push")
	content, err := os.ReadFile(hookPath)
	require.NoError(t, err)

	assert.Contains(t, string(content), HookMarkerStart)
	assert.Contains(t, string(content), HookMarkerEnd)
}

// TestHookManagerIntegration_RealGitCommit tests that git commands work
// after hook installation (verifies we don't break the repo).
func TestHookManagerIntegration_RealGitCommit(t *testing.T) {
	repoPath := createGitRepo(t)
	manager := NewHookManager(mockLogger())

	// Install the hook
	err := manager.Install(context.Background(), repoPath)
	require.NoError(t, err)

	// Make another commit (should still work)
	newFile := filepath.Join(repoPath, "newfile.txt")
	err = os.WriteFile(newFile, []byte("new content\n"), 0o644) //nolint:gosec // test file
	require.NoError(t, err)

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = repoPath
	require.NoError(t, cmd.Run())

	cmd = exec.Command("git", "commit", "-m", "Second commit")
	cmd.Dir = repoPath
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "git commit should work after hook install: %s", output)

	// Verify commit was created
	cmd = exec.Command("git", "log", "--oneline")
	cmd.Dir = repoPath
	output, err = cmd.CombinedOutput()
	require.NoError(t, err)
	assert.Contains(t, string(output), "Second commit")
}
