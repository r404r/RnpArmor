//go:build linux

package rnp

import "golang.org/x/sys/unix"

func publish(tmp, dst string) error {
	err := unix.Renameat2(unix.AT_FDCWD, tmp, unix.AT_FDCWD, dst, unix.RENAME_NOREPLACE)
	if err == unix.EEXIST {
		return ErrExists
	}
	return err
}
