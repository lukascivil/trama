//go:build linux

package runner

import "syscall"

const ioctlReadTermios = syscall.TCGETS
