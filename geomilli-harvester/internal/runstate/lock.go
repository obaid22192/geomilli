package runstate

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// FileLock uses O_EXCL creation to provide a simple cross-process lock.
type FileLock struct {
	path string
}

func Acquire(lockPath string) (*FileLock, error) {
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			stale, staleErr := isStaleLock(lockPath)
			if staleErr != nil {
				return nil, staleErr
			}
			if !stale {
				return nil, fmt.Errorf("lock already held: %s", lockPath)
			}
			if err := os.Remove(lockPath); err != nil && !errors.Is(err, os.ErrNotExist) {
				return nil, err
			}
			f, err = os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
			if err != nil {
				if errors.Is(err, os.ErrExist) {
					return nil, fmt.Errorf("lock already held: %s", lockPath)
				}
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	if _, err := f.WriteString(strconv.Itoa(os.Getpid())); err != nil {
		_ = f.Close()
		_ = os.Remove(lockPath)
		return nil, err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(lockPath)
		return nil, err
	}
	return &FileLock{path: lockPath}, nil
}

func (l *FileLock) Release() error {
	if l == nil || l.path == "" {
		return nil
	}
	return os.Remove(l.path)
}

func isStaleLock(lockPath string) (bool, error) {
	data, err := os.ReadFile(lockPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return true, nil
		}
		return false, err
	}

	pidText := strings.TrimSpace(string(data))
	if pidText == "" {
		return true, nil
	}

	pid, err := strconv.Atoi(pidText)
	if err != nil || pid <= 0 {
		return true, nil
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return true, nil
	}
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		return true, nil
	}
	return false, nil
}
