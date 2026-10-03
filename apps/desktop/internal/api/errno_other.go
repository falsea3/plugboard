//go:build !windows

package api

import "syscall"

var (
	refusedErrnos     = []error{syscall.ECONNREFUSED}
	unreachableErrnos = []error{syscall.EHOSTUNREACH, syscall.ENETUNREACH}
)
