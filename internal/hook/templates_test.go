package hook

import (
	"strings"
	"testing"
)

func TestHookMarkerStart(t *testing.T) {
	if HookMarkerStart == "" {
		t.Error("HookMarkerStart should not be empty")
	}
	if !strings.Contains(HookMarkerStart, "COPYGIT-START") {
		t.Error("HookMarkerStart should contain COPYGIT-START")
	}
}

func TestHookMarkerEnd(t *testing.T) {
	if HookMarkerEnd == "" {
		t.Error("HookMarkerEnd should not be empty")
	}
	if !strings.Contains(HookMarkerEnd, "COPYGIT-END") {
		t.Error("HookMarkerEnd should contain COPYGIT-END")
	}
}

func TestPostPushHookContent(t *testing.T) {
	content := PostPushHookContent()

	// Should NOT contain markers (those are added by Install)
	if strings.Contains(content, HookMarkerStart) {
		t.Error("PostPushHookContent should NOT contain HookMarkerStart (added by Install)")
	}
	if strings.Contains(content, HookMarkerEnd) {
		t.Error("PostPushHookContent should NOT contain HookMarkerEnd (added by Install)")
	}

	// Should contain the copygit command
	if !strings.Contains(content, "copygit push --from-hook") {
		t.Error("PostPushHookContent should contain copygit push command")
	}

	// Should pass remote argument
	if !strings.Contains(content, "$remote") {
		t.Error("PostPushHookContent should pass remote argument")
	}
}
