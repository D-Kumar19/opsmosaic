//go:build darwin || linux

package brief

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

func tryLock(file *os.File) (bool, error) {
	err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) || errors.Is(err, syscall.EINTR) {
		return false, nil
	}
	return false, fmt.Errorf("acquire state lock: %w", err)
}
