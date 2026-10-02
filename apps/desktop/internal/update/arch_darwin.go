package update

import (
	"runtime"

	"golang.org/x/sys/unix"
)

func macArch() string {
	if runtime.GOARCH == "amd64" {
		if translated, err := unix.SysctlUint32("sysctl.proc_translated"); err == nil && translated == 1 {
			return "arm64"
		}
	}
	return runtime.GOARCH
}
