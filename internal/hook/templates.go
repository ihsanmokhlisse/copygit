package hook

// Hook markers to delimit the editable region in generated hooks.
const (
	HookMarkerStart = "# COPYGIT-START — do not edit between markers"
	HookMarkerEnd   = "# COPYGIT-END"
)

// PostPushHookContent returns the content between markers for a post-push hook.
// The caller is responsible for adding the shebang (#!/bin/sh) and markers.
// This content is inserted between the COPYGIT-START and COPYGIT-END markers.
func PostPushHookContent() string {
	return `# This hook is run after a successful git push.
# It invokes copygit push to sync the repository to other configured remotes.
# The remote argument is passed to avoid re-pushing to the same remote.

remote="$1"
if [ -n "$remote" ]; then
    copygit push --from-hook "$remote" 2>/dev/null || true
fi`
}
