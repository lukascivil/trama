//go:build darwin

package runner

import "syscall"

const ioctlReadTermios = syscall.TIOCGETA
