//go:build unix

package lock

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

// lockFile acquires an exclusive advisory lock on Unix systems using flock.
func lockFile(file *os.File) error {
	return unix.Flock(int(file.Fd()), unix.LOCK_EX)
}

// unlockFile releases the lock on Unix systems.
func unlockFile(file *os.File) error {
	return unix.Flock(int(file.Fd()), unix.LOCK_UN)
}

// tryLockFileWithTimeout attempts to acquire the lock with a timeout.
// Uses non-blocking flock with polling to avoid goroutine leaks.
func tryLockFileWithTimeout(file *os.File, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
		if err == nil {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("lock acquisition timeout after %s", timeout)
		}
		time.Sleep(50 * time.Millisecond)
	}
}
