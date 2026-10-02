//go:build !darwin

package update

import "runtime"

func macArch() string { return runtime.GOARCH }
