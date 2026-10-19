//go:build !darwin && !linux

package brief

import (
	"fmt"
	"os"
)

func tryLock(*os.File) (bool, error) {
	return false, fmt.Errorf("state locking is currently supported only on macOS and Linux")
}
