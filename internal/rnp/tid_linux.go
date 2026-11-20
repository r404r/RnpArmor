//go:build linux

package rnp

import "golang.org/x/sys/unix"

func currentTID() int { return unix.Gettid() }
