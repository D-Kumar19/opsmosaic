//go:build !darwin && !linux

package brief

import "fmt"

func pickerPlatformSupported() bool {
	return false
}

func waitPickerInput(int, int) error {
	return fmt.Errorf("interactive topic picker is unsupported on this platform")
}
