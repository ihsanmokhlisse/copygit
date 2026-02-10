package lock

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FileLock provides per-repository file-based locking to prevent concurrent syncs.
// Lock file is stored at ~/.copygit/locks/<sha256(repoPath)>.lock.
type FileLock struct {
	repoPath string
	lockFile string
	file     *os.File
	mu       sync.Mutex
}

// NewFileLock creates a lock for the given repo path.
// Returns an error if the user's home directory cannot be determined.
func NewFileLock(repoPath string) (*FileLock, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("determine home directory for lock: %w", err)
	}

	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(repoPath)))
	lockDir := filepath.Join(home, ".copygit", "locks")
	lockFile := filepath.Join(lockDir, hash+".lock")

	return &FileLock{
		repoPath: repoPath,
		lockFile: lockFile,
	}, nil
}

// ForRepo creates a FileLock for the given repoPath using a custom lock dir.
func ForRepo(repoPath, lockDir string) *FileLock {
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(repoPath)))
	lockFile := filepath.Join(lockDir, hash+".lock")

	return &FileLock{
		repoPath: repoPath,
		lockFile: lockFile,
	}
}

// Lock acquires the lock, blocking until available.
func (l *FileLock) Lock() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	lockDir := filepath.Dir(l.lockFile)
	if err := os.MkdirAll(lockDir, 0o755); err != nil {
		return fmt.Errorf("mkdir locks: %w", err)
	}

	file, err := os.OpenFile(l.lockFile, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open lock file: %w", err)
	}

	l.file = file

	if err := lockFile(l.file); err != nil {
		l.file.Close()
		return fmt.Errorf("lock file: %w", err)
	}

	return nil
}

// TryLock attempts to acquire the lock with a timeout.
// Returns true if lock was acquired, false if timeout.
func (l *FileLock) TryLock(timeout time.Duration) (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	lockDir := filepath.Dir(l.lockFile)
	if err := os.MkdirAll(lockDir, 0o755); err != nil {
		return false, fmt.Errorf("mkdir locks: %w", err)
	}

	file, err := os.OpenFile(l.lockFile, os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return false, fmt.Errorf("open lock file: %w", err)
	}

	l.file = file

	err = tryLockFileWithTimeout(l.file, timeout)
	if err != nil {
		l.file.Close()
		return false, nil
	}

	return true, nil
}

// Unlock releases the lock.
func (l *FileLock) Unlock() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file == nil {
		return nil
	}

	if err := unlockFile(l.file); err != nil {
		l.file.Close()
		return fmt.Errorf("unlock file: %w", err)
	}

	if err := l.file.Close(); err != nil {
		return fmt.Errorf("close lock file: %w", err)
	}

	l.file = nil
	return nil
}

// IsLocked reports whether the lock file exists and is locked.
func (l *FileLock) IsLocked() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.file != nil
}
