package hook

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// HookManager manages git hooks for CopyGit.
type HookManager struct { //nolint:revive // established API name
	logger *slog.Logger
}

// NewHookManager creates a new HookManager.
func NewHookManager(logger *slog.Logger) *HookManager {
	return &HookManager{logger: logger}
}

// hookPath returns the path to the post-push hook for the given repo.
func (hm *HookManager) hookPath(repoPath string) string {
	return filepath.Join(repoPath, ".git", "hooks", "post-push")
}

// IsInstalled checks if the post-push hook is installed with our markers.
func (hm *HookManager) IsInstalled(ctx context.Context, repoPath string) (bool, error) { //nolint:revive // ctx reserved for future cancellation
	hookPath := hm.hookPath(repoPath)

	content, err := os.ReadFile(hookPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("read hook file: %w", err)
	}

	return strings.Contains(string(content), HookMarkerStart) &&
		strings.Contains(string(content), HookMarkerEnd), nil
}

// Install installs the post-push hook using marker-based insertion.
// If hook exists and has our markers, it skips (already installed).
// If hook exists without our markers, it appends between COPYGIT-START/END markers.
// If no hook exists, it creates a new one with markers.
func (hm *HookManager) Install(ctx context.Context, repoPath string) error { //nolint:nestif // existing hook logic requires nested conditions
	hookPath := hm.hookPath(repoPath)

	// Check if already installed
	installed, err := hm.IsInstalled(ctx, repoPath)
	if err != nil {
		return err
	}
	if installed {
		hm.logger.Debug("hook already installed", "path", hookPath)
		return nil
	}

	// Read existing content or start fresh
	var newContent string
	existingContent, err := os.ReadFile(hookPath)
	if err != nil { //nolint:nestif // existing hook logic requires nested conditions
		if os.IsNotExist(err) {
			// No hook exists, create new one with shebang + markers + content
			newContent = "#!/bin/sh\n" + HookMarkerStart + "\n" + PostPushHookContent() + "\n" + HookMarkerEnd + "\n"
		} else {
			return fmt.Errorf("read existing hook: %w", err)
		}
	} else {
		// Hook exists without our markers, insert between markers
		existing := strings.TrimSuffix(string(existingContent), "\n")
		if !strings.HasSuffix(existing, "\n") {
			existing += "\n"
		}
		newContent = existing + HookMarkerStart + "\n" + PostPushHookContent() + "\n" + HookMarkerEnd + "\n"
	}

	// Ensure directory exists
	hooksDir := filepath.Dir(hookPath)
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return fmt.Errorf("create hooks directory: %w", err)
	}

	// Write the hook file
	if err := os.WriteFile(hookPath, []byte(newContent), 0o755); err != nil { //nolint:gosec // hooks must be executable
		return fmt.Errorf("write hook file: %w", err)
	}

	hm.logger.Info("hook installed", "path", hookPath)
	return nil
}

// Uninstall removes CopyGit's content between markers from the post-push hook.
// If the file only had our content, it removes the file entirely.
// Otherwise, it keeps other content.
func (hm *HookManager) Uninstall(ctx context.Context, repoPath string) error { //nolint:gocyclo,revive // hook uninstall requires careful conditional logic; ctx reserved for future cancellation
	hookPath := hm.hookPath(repoPath)

	content, err := os.ReadFile(hookPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No hook to uninstall
			hm.logger.Debug("hook does not exist, nothing to uninstall")
			return nil
		}
		return fmt.Errorf("read hook file: %w", err)
	}

	hookContent := string(content)

	// Check if our markers exist
	if !strings.Contains(hookContent, HookMarkerStart) || !strings.Contains(hookContent, HookMarkerEnd) {
		hm.logger.Debug("hook does not have our markers, nothing to uninstall")
		return nil
	}

	// Extract content between markers
	startIdx := strings.Index(hookContent, HookMarkerStart)
	endIdx := strings.Index(hookContent, HookMarkerEnd)
	if startIdx == -1 || endIdx == -1 {
		hm.logger.Debug("markers not found, nothing to uninstall")
		return nil
	}

	// endIdx needs to account for the marker length
	endIdx += len(HookMarkerEnd)

	// Build new content: everything before start + everything after end
	beforeMarkers := hookContent[:startIdx]
	afterMarkers := hookContent[endIdx:]

	// Clean up: remove trailing newlines from beforeMarkers
	beforeMarkers = strings.TrimRight(beforeMarkers, "\n")
	// Remove leading newlines from afterMarkers
	afterMarkers = strings.TrimLeft(afterMarkers, "\n")

	var newContent string
	if beforeMarkers == "" && afterMarkers == "" {
		// Only our content exists, remove the file
		if err := os.Remove(hookPath); err != nil {
			return fmt.Errorf("remove hook file: %w", err)
		}
		hm.logger.Info("hook removed (only had our content)", "path", hookPath)
		return nil
	}

	// Reconstruct with shebang and preserve other content
	// Remove shebang from beforeMarkers if present
	beforeContent := strings.TrimPrefix(beforeMarkers, "#!/bin/sh\n")
	beforeContent = strings.TrimPrefix(beforeContent, "#!/bin/sh")

	// Build new content
	switch {
	case beforeContent != "" && afterMarkers != "":
		newContent = "#!/bin/sh\n" + beforeContent + "\n" + afterMarkers + "\n"
	case beforeContent != "":
		newContent = "#!/bin/sh\n" + beforeContent + "\n"
	case afterMarkers != "":
		newContent = "#!/bin/sh\n" + afterMarkers + "\n"
	default:
		newContent = "#!/bin/sh\n"
	}

	// Handle edge case: if there's no meaningful content left, remove file
	trimmed := strings.TrimSpace(strings.ReplaceAll(newContent, "#!/bin/sh", ""))
	if trimmed == "" {
		if err := os.Remove(hookPath); err != nil {
			return fmt.Errorf("remove hook file: %w", err)
		}
		hm.logger.Info("hook removed (only had our content)", "path", hookPath)
		return nil
	}

	if err := os.WriteFile(hookPath, []byte(newContent), 0o755); err != nil { //nolint:gosec // hooks must be executable
		return fmt.Errorf("write hook file: %w", err)
	}

	hm.logger.Info("hook uninstalled", "path", hookPath)
	return nil
}
