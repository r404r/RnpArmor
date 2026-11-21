//go:build unix

package rnp

import "os"

func euid() int { return os.Geteuid() }

func makeUnwritable(dir string) error { return os.Chmod(dir, 0o555) }

func restoreWritable(dir string) error { return os.Chmod(dir, 0o755) }

func secretRestricted(info os.FileInfo) bool {
	return info.Mode().IsRegular() && info.Mode().Perm() == 0o600
}
