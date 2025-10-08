package lock

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/YangQing-Lin/cc-switch-cli/internal/portable"
)

const (
	// LockFileName is the name of the lock file
	LockFileName = ".cc-switch.lock"
	// StaleLockTimeout is the duration after which a lock is considered stale
	StaleLockTimeout = 5 * time.Minute
)

// Lock represents a file-based lock
type Lock struct {
	lockPath string
	acquired bool
}

// NewLock creates a new lock for the given config directory
func NewLock(configDir string) *Lock {
	return &Lock{
		lockPath: filepath.Join(configDir, LockFileName),
		acquired: false,
	}
}

// TryAcquire attempts to acquire the lock
// Returns true if acquired, false if another instance is running
func (l *Lock) TryAcquire() (bool, error) {
	// Skip lock in portable mode
	if portable.IsPortableMode() {
		l.acquired = true
		return true, nil
	}

	// Check if lock file exists
	if info, err := os.Stat(l.lockPath); err == nil {
		// Lock file exists, check if it's stale
		if time.Since(info.ModTime()) > StaleLockTimeout {
			// Stale lock, remove it
			os.Remove(l.lockPath)
		} else {
			// Active lock
			return false, nil
		}
	}

	// Create lock file with current PID
	pid := os.Getpid()
	if err := os.WriteFile(l.lockPath, []byte(strconv.Itoa(pid)), 0600); err != nil {
		return false, fmt.Errorf("failed to create lock file: %w", err)
	}

	l.acquired = true
	return true, nil
}

// ForceAcquire forcibly acquires the lock (kills previous lock)
func (l *Lock) ForceAcquire() error {
	// Skip lock in portable mode
	if portable.IsPortableMode() {
		l.acquired = true
		return nil
	}

	// Remove existing lock if present
	os.Remove(l.lockPath)

	// Create new lock
	pid := os.Getpid()
	if err := os.WriteFile(l.lockPath, []byte(strconv.Itoa(pid)), 0600); err != nil {
		return fmt.Errorf("failed to create lock file: %w", err)
	}

	l.acquired = true
	return nil
}

// Release releases the lock
func (l *Lock) Release() error {
	if !l.acquired {
		return nil
	}

	// Skip in portable mode
	if portable.IsPortableMode() {
		l.acquired = false
		return nil
	}

	if err := os.Remove(l.lockPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove lock file: %w", err)
	}

	l.acquired = false
	return nil
}

// Touch updates the lock file timestamp (keep-alive)
func (l *Lock) Touch() error {
	if !l.acquired || portable.IsPortableMode() {
		return nil
	}

	now := time.Now()
	return os.Chtimes(l.lockPath, now, now)
}

// GetPID returns the PID stored in the lock file
func (l *Lock) GetPID() (int, error) {
	data, err := os.ReadFile(l.lockPath)
	if err != nil {
		return 0, err
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, fmt.Errorf("invalid PID in lock file: %w", err)
	}

	return pid, nil
}
