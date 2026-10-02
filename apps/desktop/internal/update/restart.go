package update

import (
	"os"
	"os/exec"
	"runtime"
	"strconv"
)

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
			"Wait-Process -Id "+pid+" -ErrorAction SilentlyContinue; Start-Process -FilePath $env:PLUGBOARD_RELAUNCH")
		cmd.Env = append(os.Environ(), "PLUGBOARD_RELAUNCH="+exe)
		return cmd, nil
	}
	return exec.Command("/bin/sh", "-c", `while kill -0 "$0" 2>/dev/null; do sleep 0.2; done; exec "$1"`, pid, exe), nil
}
