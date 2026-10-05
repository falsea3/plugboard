package update

import (
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"time"
)

const AfterFlag = "--after"

func Restart(quit func()) error {
	cmd, err := relaunchCommand(runtime.GOOS, os.Getpid(), runningBundle(), os.Getenv("APPIMAGE"), os.Executable)
	if err != nil {
		return err
	}
	if runtime.GOOS == "darwin" {
		if err := cmd.Run(); err != nil {
			return err
		}
	} else {
		detach(cmd)
		if err := cmd.Start(); err != nil {
			return err
		}
	}
	quit()
	return nil
}

func relaunchCommand(goos string, pid int, bundle, appImage string, executable func() (string, error)) (*exec.Cmd, error) {
	after := []string{AfterFlag, strconv.Itoa(pid)}
	if goos == "darwin" && bundle != "" {
		return exec.Command("/usr/bin/open", append([]string{"-n", bundle, "--args"}, after...)...), nil
	}
	exe := appImage
	if exe == "" {
		var err error
		if exe, err = executable(); err != nil {
			return nil, err
		}
	}
	return exec.Command(exe, after...), nil
}

func AfterPID(args []string) (int, bool) {
	for i, a := range args {
		if a == AfterFlag && i+1 < len(args) {
			pid, err := strconv.Atoi(args[i+1])
			return pid, err == nil && pid > 0
		}
	}
	return 0, false
}

func WaitForExit(pid int, limit time.Duration) {
	deadline := time.Now().Add(limit)
	for running(pid) && time.Now().Before(deadline) {
		time.Sleep(100 * time.Millisecond)
	}
}
