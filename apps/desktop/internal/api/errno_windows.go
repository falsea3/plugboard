package api

import "syscall"

const (
	wsaNetUnreach  syscall.Errno = 10051
	wsaConnRefused syscall.Errno = 10061
	wsaHostUnreach syscall.Errno = 10065
)

var (
	refusedErrnos     = []error{syscall.ECONNREFUSED, wsaConnRefused}
	unreachableErrnos = []error{syscall.EHOSTUNREACH, syscall.ENETUNREACH, wsaHostUnreach, wsaNetUnreach}
)
