//go:build darwin || linux

package brief

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func pickerPlatformSupported() bool {
	return true
}

func waitPickerInput(fd int, timeout int) error {
	descriptors := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
	for {
		ready, err := unix.Poll(descriptors, timeout)
		if err == unix.EINTR {
			continue
		}
		if err != nil {
			return fmt.Errorf("wait for terminal key: %w", err)
		}
		if ready == 0 {
			return errPickerSequenceTimeout
		}
		if descriptors[0].Revents&(unix.POLLERR|unix.POLLHUP|unix.POLLNVAL) != 0 {
			return fmt.Errorf("terminal input closed")
		}
		return nil
	}
}
