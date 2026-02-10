package lock

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFileLock_Success(t *testing.T) {
	fl, err := NewFileLock("/some/repo/path")
	require.NoError(t, err)
	assert.NotNil(t, fl)
	assert.Contains(t, fl.lockFile, ".copygit")
	assert.Contains(t, fl.lockFile, "locks")
	assert.True(t, filepath.IsAbs(fl.lockFile))
}

func TestNewFileLock_DeterministicHash(t *testing.T) {
	fl1, _ := NewFileLock("/repo/path")
	fl2, _ := NewFileLock("/repo/path")
	assert.Equal(t, fl1.lockFile, fl2.lockFile, "same repo path should produce same lock file")

	fl3, _ := NewFileLock("/different/repo")
	assert.NotEqual(t, fl1.lockFile, fl3.lockFile, "different repo paths should produce different lock files")
}

func TestForRepo(t *testing.T) {
	tmpDir := t.TempDir()
	fl := ForRepo("/my/repo", tmpDir)

	assert.NotNil(t, fl)
	assert.Contains(t, fl.lockFile, tmpDir)
	assert.True(t, filepath.IsAbs(fl.lockFile))
}

func TestFileLock_LockAndUnlock(t *testing.T) {
	tmpDir := t.TempDir()
	fl := ForRepo("/test/repo", tmpDir)

	// Lock should succeed
	err := fl.Lock()
	require.NoError(t, err)
	assert.True(t, fl.IsLocked())

	// Unlock should succeed
	err = fl.Unlock()
	require.NoError(t, err)
	assert.False(t, fl.IsLocked())
}

func TestFileLock_DoubleUnlock(t *testing.T) {
	tmpDir := t.TempDir()
	fl := ForRepo("/test/repo", tmpDir)

	err := fl.Lock()
	require.NoError(t, err)

	// First unlock succeeds
	err = fl.Unlock()
	require.NoError(t, err)

	// Second unlock is a no-op (file is nil)
	err = fl.Unlock()
	require.NoError(t, err)
}

func TestFileLock_TryLock_Success(t *testing.T) {
	tmpDir := t.TempDir()
	fl := ForRepo("/test/repo", tmpDir)

	acquired, err := fl.TryLock(1 * time.Second)
	require.NoError(t, err)
	assert.True(t, acquired)
	assert.True(t, fl.IsLocked())

	err = fl.Unlock()
	require.NoError(t, err)
}

func TestFileLock_TryLock_Contention(t *testing.T) {
	tmpDir := t.TempDir()

	// First lock succeeds
	fl1 := ForRepo("/test/repo", tmpDir)
	err := fl1.Lock()
	require.NoError(t, err)

	// Second lock on same repo should timeout
	fl2 := ForRepo("/test/repo", tmpDir)
	acquired, err := fl2.TryLock(200 * time.Millisecond)
	require.NoError(t, err)
	assert.False(t, acquired, "should not acquire lock while first lock is held")

	// Release first lock
	err = fl1.Unlock()
	require.NoError(t, err)

	// Now second lock should succeed
	acquired, err = fl2.TryLock(1 * time.Second)
	require.NoError(t, err)
	assert.True(t, acquired)
	_ = fl2.Unlock()
}

func TestFileLock_IsLocked_InitiallyFalse(t *testing.T) {
	tmpDir := t.TempDir()
	fl := ForRepo("/test/repo", tmpDir)
	assert.False(t, fl.IsLocked())
}
