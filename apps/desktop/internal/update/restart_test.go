package update

import (
	"os"
	"os/exec"
	"reflect"
	"testing"
	"time"
)

func TestRelaunchStartsTheNewAppWaitingForThisOne(t *testing.T) {
	exe := func() (string, error) { return "/opt/plugboard/plugboard", nil }
	cases := []struct {
		goos, bundle, appImage string
		want                   []string
	}{
		{"darwin", "/Applications/Plugboard.app", "", []string{"/usr/bin/open", "-n", "/Applications/Plugboard.app", "--args", "--after", "42"}},
		{"darwin", "", "", []string{"/opt/plugboard/plugboard", "--after", "42"}},
		{"linux", "", "/home/u/Plugboard.AppImage", []string{"/home/u/Plugboard.AppImage", "--after", "42"}},
		{"windows", "", "", []string{"/opt/plugboard/plugboard", "--after", "42"}},
	}
	for _, tc := range cases {
		cmd, err := relaunchCommand(tc.goos, 42, tc.bundle, tc.appImage, exe)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(cmd.Args, tc.want) {
			t.Errorf("%s: %q, want %q", tc.goos, cmd.Args, tc.want)
		}
	}
}

func TestAfterPID(t *testing.T) {
	if pid, ok := AfterPID([]string{"plugboard", "-psn_0_1", "--after", "123"}); !ok || pid != 123 {
		t.Errorf("got %d %v", pid, ok)
	}
	for _, args := range [][]string{{"plugboard"}, {"plugboard", "--after"}, {"plugboard", "--after", "x"}, {"plugboard", "--after", "0"}} {
		if _, ok := AfterPID(args); ok {
			t.Errorf("%q gave a pid", args)
		}
	}
}

func TestWaitForExit(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	go cmd.Wait()
	start := time.Now()
	WaitForExit(cmd.Process.Pid, 10*time.Second)
	if time.Since(start) > 9*time.Second || running(cmd.Process.Pid) {
		t.Fatal("still waiting for a process that ended")
	}
	start = time.Now()
	WaitForExit(os.Getpid(), 300*time.Millisecond)
	if time.Since(start) < 250*time.Millisecond {
		t.Fatal("stopped waiting while the process still runs")
	}
}
