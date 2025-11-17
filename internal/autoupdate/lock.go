package autoupdate

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// UpdateLock represents an exclusive file lock for update operations.
type UpdateLock struct {
	file *os.File
}

// AcquireLock attempts to acquire an exclusive lock on the update lock file.
// Returns an error if the lock is already held by another process.
// The lock is automatically released when the process exits.
func AcquireLock(lockPath string) (*UpdateLock, error) {
	// Ensure parent directory exists
	dir := filepath.Dir(lockPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create lock directory: %w", err)
	}

	// Open or create lock file
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open lock file: %w", err)
	}

	// Try to acquire exclusive lock (non-blocking)
	err = syscall.Flock(int(lockFile.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		lockFile.Close()
		return nil, fmt.Errorf("lock already held by another process")
	}

	return &UpdateLock{file: lockFile}, nil
}

// Release releases the lock and closes the file.
// It's safe to call Release multiple times.
func (l *UpdateLock) Release() {
	if l.file != nil {
		syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
		l.file.Close()
		l.file = nil
	}
}
