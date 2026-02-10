//go:build windows

package lock

import (
	"fmt"
	"os"
	"time"

	"golang.org/x/sys/windows"
)

// lockFile acquires an exclusive lock on Windows systems using LockFileEx.
func lockFile(file *os.File) error {
	handle := windows.Handle(file.Fd())
	overlapped := &windows.Overlapped{}
	return windows.LockFileEx(handle, windows.LOCKFILE_EXCLUSIVE_LOCK, 0, 1, 0, overlapped)
}

// unlockFile releases the lock on Windows systems.
func unlockFile(file *os.File) error {
	handle := windows.Handle(file.Fd())
	overlapped := &windows.Overlapped{}
	return windows.UnlockFileEx(handle, 0, 1, 0, overlapped)
}

// tryLockFileWithTimeout attempts to acquire the lock with a timeout.
func tryLockFileWithTimeout(file *os.File, timeout time.Duration) error {
	done := make(chan error, 1)
	go func() {
		handle := windows.Handle(file.Fd())
		overlapped := &windows.Overlapped{}
		done <- windows.LockFileEx(handle, windows.LOCKFILE_EXCLUSIVE_LOCK, 0, 1, 0, overlapped)
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		return fmt.Errorf("lock acquisition timeout")
	}
}
