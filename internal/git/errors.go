package git

import "fmt"

// GitError wraps git command failures with structured context.
type GitError struct { //nolint:revive // established API name
	Command  string
	Args     []string
	ExitCode int
	Stderr   string
}

// Error implements the error interface.
func (e *GitError) Error() string {
	return fmt.Sprintf("git %v: exit code %d: %s", e.Args, e.ExitCode, e.Stderr)
}

// Unwrap returns a generic error for the exit code.
func (e *GitError) Unwrap() error {
	return fmt.Errorf("exit code %d", e.ExitCode)
}
