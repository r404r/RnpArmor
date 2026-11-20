//go:build !unix

package rnp

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func euid() int { return -1 }

// makeUnwritable denies the current user write and delete-child on dir.
// Windows chmod does not stop creating files in a directory.
func makeUnwritable(dir string) error {
	user := os.Getenv("USERNAME")
	if user == "" {
		return fmt.Errorf("USERNAME is empty")
	}
	cmd := exec.Command("icacls", dir, "/deny", user+":(W,DC)")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("icacls deny: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func restoreWritable(dir string) error {
	user := os.Getenv("USERNAME")
	if user == "" {
		return fmt.Errorf("USERNAME is empty")
	}
	cmd := exec.Command("icacls", dir, "/remove:d", user)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("icacls remove deny: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// Windows does not store Unix mode 0600. The file is the one CreateTemp
// created for this user; the test checks it is a non-empty regular file.
func secretRestricted(info os.FileInfo) bool {
	return info.Mode().IsRegular() && info.Size() > 0
}
