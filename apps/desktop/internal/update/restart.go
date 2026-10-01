package update

import (
	"os"
	"os/exec"
	"runtime"
	"strconv"
)

// Restart starts the app again as soon as this process has exited, and then
// calls quit, so the app shuts down as usual (closing its sessions) and the
// new copy doesn't meet the old one's single-instance lock.
func Restart(quit func()) error {
	cmd, err := relaunchCommand(strconv.Itoa(os.Getpid()))
	if err != nil {
		return err
	}
	detach(cmd)
	if err := cmd.Start(); err != nil {
		return err
	}
	quit()
	return nil
}

// relaunchCommand waits for process pid to exit, then opens the app again.
func relaunchCommand(pid string) (*exec.Cmd, error) {
	if bundle := runningBundle(); runtime.GOOS == "darwin" && bundle != "" {
		return exec.Command("/bin/sh", "-c", `while kill -0 "$0" 2>/dev/null; do sleep 0.2; done; exec open "$1"`, pid, bundle), nil
	}
	exe := os.Getenv("APPIMAGE")
	if exe == "" {
		var err error
		if exe, err = os.Executable(); err != nil {
			return nil, err
		}
	}
	if runtime.GOOS == "windows" {
		cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command",
			"Wait-Process -Id "+pid+" -ErrorAction SilentlyContinue; Start-Process -FilePath $env:RELAYDB_RELAUNCH")
		// The path goes in through the environment, so no quoting can break it.
		cmd.Env = append(os.Environ(), "RELAYDB_RELAUNCH="+exe)
		return cmd, nil
	}
	return exec.Command("/bin/sh", "-c", `while kill -0 "$0" 2>/dev/null; do sleep 0.2; done; exec "$1"`, pid, exe), nil
}
