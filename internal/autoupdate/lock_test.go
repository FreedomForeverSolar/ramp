package autoupdate

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAcquireLock_Success(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "update.lock")

	lock, err := AcquireLock(lockPath)
	if err != nil {
		t.Fatalf("AcquireLock() error: %v", err)
	}
	defer lock.Release()

	// Verify lock file was created
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Error("Lock file was not created")
	}
}

func TestAcquireLock_AlreadyLocked(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "update.lock")

	// Acquire first lock
	lock1, err := AcquireLock(lockPath)
	if err != nil {
		t.Fatalf("First AcquireLock() error: %v", err)
	}
	defer lock1.Release()

	// Try to acquire second lock (should fail)
	lock2, err := AcquireLock(lockPath)
	if err == nil {
		lock2.Release()
		t.Fatal("Second AcquireLock() should have failed, but succeeded")
	}

	// Error message should indicate lock is held
	if err.Error() == "" {
		t.Error("Error message should not be empty")
	}
}

func TestLockRelease(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "update.lock")

	// Acquire lock
	lock1, err := AcquireLock(lockPath)
	if err != nil {
		t.Fatalf("AcquireLock() error: %v", err)
	}

	// Release it
	lock1.Release()

	// Should be able to acquire again
	lock2, err := AcquireLock(lockPath)
	if err != nil {
		t.Fatalf("AcquireLock() after release error: %v", err)
	}
	defer lock2.Release()
}

func TestAcquireLock_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	// Use nested path that doesn't exist
	lockPath := filepath.Join(tmpDir, "nested", "dir", "update.lock")

	lock, err := AcquireLock(lockPath)
	if err != nil {
		t.Fatalf("AcquireLock() should create directories, got error: %v", err)
	}
	defer lock.Release()

	// Verify file exists
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Error("Lock file was not created in nested directory")
	}
}

func TestMultipleReleaseCalls(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "update.lock")

	lock, err := AcquireLock(lockPath)
	if err != nil {
		t.Fatalf("AcquireLock() error: %v", err)
	}

	// Multiple releases should not panic
	lock.Release()
	lock.Release() // Should be safe to call multiple times
}

func TestConcurrentLockAttempts(t *testing.T) {
	tmpDir := t.TempDir()
	lockPath := filepath.Join(tmpDir, "update.lock")

	// Acquire lock in main goroutine
	lock, err := AcquireLock(lockPath)
	if err != nil {
		t.Fatalf("AcquireLock() error: %v", err)
	}

	// Try to acquire in goroutine
	done := make(chan bool)
	go func() {
		lock2, err := AcquireLock(lockPath)
		if err == nil {
			lock2.Release()
			t.Error("Concurrent lock acquisition should have failed")
		}
		done <- true
	}()

	// Wait for goroutine
	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Goroutine timed out")
	}

	lock.Release()
}
