//go:build !windows

package update

import (
	"os/exec"
	"syscall"
)

// detach puts cmd in its own process group, so it outlives the app quitting.
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
