//go:build !windows

package update

import (
	"errors"
	"syscall"
)

func running(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}
