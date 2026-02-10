package daemon

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
)

// PIDFile manages the daemon's PID file.
type PIDFile struct {
	path string
}

// NewPIDFile creates a new PID file manager.
func NewPIDFile(path string) *PIDFile {
	return &PIDFile{path: path}
}

// Write stores the given PID.
func (p *PIDFile) Write(pid int) error {
	return os.WriteFile(p.path, []byte(strconv.Itoa(pid)), 0o600)
}

// Read returns the PID from the file, or error if not found.
func (p *PIDFile) Read() (int, error) {
	data, err := os.ReadFile(p.path)
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("invalid pid: %w", err)
	}
	return pid, nil
}

// Remove deletes the PID file.
func (p *PIDFile) Remove() error {
	return os.Remove(p.path)
}

// IsStale returns true if the PID file exists but the process is not running.
func (p *PIDFile) IsStale() bool {
	pid, err := p.Read()
	if err != nil {
		return false
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return true
	}

	// On Unix, FindProcess always succeeds. Send signal 0 to check.
	err = process.Signal(syscall.Signal(0))
	return err != nil
}
