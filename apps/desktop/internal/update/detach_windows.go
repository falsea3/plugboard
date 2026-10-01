package update

import (
	"os/exec"
	"syscall"
)

// detach runs cmd without a console window flashing up.
func detach(cmd *exec.Cmd) {
	const createNoWindow = 0x08000000
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
}
