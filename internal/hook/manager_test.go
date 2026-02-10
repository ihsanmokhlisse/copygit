package hook

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mockLogger returns a discard logger for tests.
func mockLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestHookManager_IsInstalled(t *testing.T) {
	// Create a temp directory for the test repo
	tmpDir, err := os.MkdirTemp("", "copygit-hook-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	hooksDir := filepath.Join(tmpDir, ".git", "hooks")
	err = os.MkdirAll(hooksDir, 0o755)
	if err != nil {
		t.Fatalf("failed to create hooks dir: %v", err)
	}

	manager := NewHookManager(mockLogger())
	repoPath := tmpDir

	// Test: no hook file exists - should return false
	installed, err := manager.IsInstalled(context.Background(), repoPath)
	if err != nil {
		t.Fatalf("IsInstalled error: %v", err)
	}
	if installed {
		t.Error("IsInstalled should return false when no hook exists")
	}

	// Test: hook file exists without our markers - should return false
	hookPath := filepath.Join(hooksDir, "post-push")
	err = os.WriteFile(hookPath, []byte("#!/bin/sh\necho hello\n"), 0o755) //nolint:gosec // test file
	if err != nil {
		t.Fatalf("failed to write hook file: %v", err)
	}

	installed, err = manager.IsInstalled(context.Background(), repoPath)
	if err != nil {
		t.Fatalf("IsInstalled error: %v", err)
	}
	if installed {
		t.Error("IsInstalled should return false when hook exists without our markers")
	}

	// Test: hook file exists with our markers - should return true
	err = os.WriteFile(hookPath, []byte("#!/bin/sh\n"+HookMarkerStart+"\n# content\n"+HookMarkerEnd+"\n"), 0o755) //nolint:gosec // test file
	if err != nil {
		t.Fatalf("failed to write hook file with markers: %v", err)
	}

	installed, err = manager.IsInstalled(context.Background(), repoPath)
	if err != nil {
		t.Fatalf("IsInstalled error: %v", err)
	}
	if !installed {
		t.Error("IsInstalled should return true when hook has our markers")
	}
}

func TestHookManager_Install(t *testing.T) { //nolint:gocyclo // table-driven test with subtests
	// Create a temp directory for the test repo
	tmpDir, err := os.MkdirTemp("", "copygit-hook-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	hooksDir := filepath.Join(tmpDir, ".git", "hooks")
	err = os.MkdirAll(hooksDir, 0o755)
	if err != nil {
		t.Fatalf("failed to create hooks dir: %v", err)
	}

	manager := NewHookManager(mockLogger())
	repoPath := tmpDir

	// Test: no hook exists - should create new hook
	err = manager.Install(context.Background(), repoPath)
	if err != nil {
		t.Fatalf("Install error: %v", err)
	}

	hookPath := filepath.Join(hooksDir, "post-push")
	content, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("failed to read hook file: %v", err)
	}

	if !strings.Contains(string(content), HookMarkerStart) {
		t.Error("Install should add start marker")
	}
	if !strings.Contains(string(content), HookMarkerEnd) {
		t.Error("Install should add end marker")
	}
	if !strings.Contains(string(content), "copygit push --from-hook") {
		t.Error("Install should add copygit command")
	}

	// Test: hook exists with our markers - should skip
	err = manager.Install(context.Background(), repoPath)
	if err != nil {
		t.Fatalf("Install second time error: %v", err)
	}

	content, err = os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("failed to read hook file: %v", err)
	}
	// Should not have duplicate markers
	count := strings.Count(string(content), HookMarkerStart)
	if count != 1 {
		t.Errorf("Install should skip when markers exist, got %d markers", count)
	}

	// Test: hook exists without our markers - should append between markers
	err = os.WriteFile(hookPath, []byte("#!/bin/sh\necho hello\n"), 0o755) //nolint:gosec // test file
	if err != nil {
		t.Fatalf("failed to write hook file: %v", err)
	}

	err = manager.Install(context.Background(), repoPath)
	if err != nil {
		t.Fatalf("Install with existing hook error: %v", err)
	}

	content, err = os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("failed to read hook file: %v", err)
	}

	// Should have both old content and new markers
	if !strings.Contains(string(content), "echo hello") {
		t.Error("Install should preserve existing content")
	}
	if !strings.Contains(string(content), HookMarkerStart) {
		t.Error("Install should add start marker")
	}
	if !strings.Contains(string(content), HookMarkerEnd) {
		t.Error("Install should add end marker")
	}
}

func TestHookManager_Uninstall(t *testing.T) {
	// Create a temp directory for the test repo
	tmpDir, err := os.MkdirTemp("", "copygit-hook-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	hooksDir := filepath.Join(tmpDir, ".git", "hooks")
	err = os.MkdirAll(hooksDir, 0o755)
	if err != nil {
		t.Fatalf("failed to create hooks dir: %v", err)
	}

	manager := NewHookManager(mockLogger())
	repoPath := tmpDir

	// First install the hook
	err = manager.Install(context.Background(), repoPath)
	if err != nil {
		t.Fatalf("Install error: %v", err)
	}

	// Test: uninstall removes our content
	err = manager.Uninstall(context.Background(), repoPath)
	if err != nil {
		t.Fatalf("Uninstall error: %v", err)
	}

	hookPath := filepath.Join(hooksDir, "post-push")
	_, err = os.Stat(hookPath)
	if !os.IsNotExist(err) {
		// If file exists, should only have shebang
		content, _ := os.ReadFile(hookPath)
		if strings.TrimSpace(string(content)) != "#!/bin/sh" {
			t.Error("Uninstall should leave only shebang when file only had our content")
		}
	}

	// Test: uninstall with mixed content - keeps other content
	err = os.WriteFile(hookPath, []byte("#!/bin/sh\necho hello\n"+HookMarkerStart+"\n# copygit\n"+HookMarkerEnd+"\necho goodbye\n"), 0o755) //nolint:gosec // test file
	if err != nil {
		t.Fatalf("failed to write hook file: %v", err)
	}

	err = manager.Uninstall(context.Background(), repoPath)
	if err != nil {
		t.Fatalf("Uninstall with mixed content error: %v", err)
	}

	content, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatalf("failed to read hook file: %v", err)
	}

	// Should preserve other content
	if !strings.Contains(string(content), "echo hello") {
		t.Error("Uninstall should preserve content")
	}
	if !strings.Contains(string(content), "echo goodbye") {
		t.Error("Uninstall should preserve content after markers")
	}
	// Should not have our markers
	if strings.Contains(string(content), HookMarkerStart) {
		t.Error("Uninstall should remove start marker")
	}
	if strings.Contains(string(content), HookMarkerEnd) {
		t.Error("Uninstall should remove end marker")
	}
}
