package daemon

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPIDFile_WriteReadRemove(t *testing.T) {
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "daemon.pid")

	pf := NewPIDFile(pidPath)

	// Write
	err := pf.Write(12345)
	require.NoError(t, err)

	// Read
	pid, err := pf.Read()
	require.NoError(t, err)
	assert.Equal(t, 12345, pid)

	// Remove
	err = pf.Remove()
	require.NoError(t, err)

	// Should fail after removal
	_, err = pf.Read()
	assert.Error(t, err)
}

func TestPIDFile_IsStale(t *testing.T) {
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "daemon.pid")
	pf := NewPIDFile(pidPath)

	// Write a PID that definitely doesn't exist
	err := pf.Write(999999999)
	require.NoError(t, err)

	assert.True(t, pf.IsStale())
}

func TestPIDFile_CurrentProcess(t *testing.T) {
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "daemon.pid")
	pf := NewPIDFile(pidPath)

	// Write our own PID
	err := pf.Write(os.Getpid())
	require.NoError(t, err)

	// Should not be stale (we're running)
	assert.False(t, pf.IsStale())

	_ = pf.Remove()
}

func TestPIDFile_ReadNonExistent(t *testing.T) {
	pf := NewPIDFile("/nonexistent/daemon.pid")
	_, err := pf.Read()
	assert.Error(t, err)
}
