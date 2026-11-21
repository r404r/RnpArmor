package rnp

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	// MaxInput is the largest plaintext or detached input this slice reads.
	MaxInput = 32 << 20
	// MaxOutput is the memory cap passed to rnp_output_to_memory.
	MaxOutput = 40 << 20
)

var (
	ErrExists   = errors.New("refusing to overwrite an existing file")
	ErrTooLarge = errors.New("input exceeds the memory limit")
	ErrEmpty    = errors.New("input is empty")
	ErrRelative = errors.New("path must be absolute")
)

func readLimited(path string) ([]byte, error) {
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("%w: %s", ErrRelative, path)
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("not a regular file")
	}
	if info.Size() == 0 {
		return nil, ErrEmpty
	}
	if info.Size() > MaxInput {
		return nil, ErrTooLarge
	}
	return os.ReadFile(path)
}

// commitNew writes data to a temporary file in the destination directory
// and publishes it only if dst does not exist. On Linux the publish is
// renameat2(RENAME_NOREPLACE), so a crash leaves the temp name, not a
// truncated dst. The temp name starts with .rnparmor- and is removed on
// failure. Callers must not log data.
func commitNew(dst string, data []byte, mode os.FileMode) error {
	if !filepath.IsAbs(dst) {
		return fmt.Errorf("%w: %s", ErrRelative, dst)
	}
	if _, err := os.Lstat(dst); err == nil {
		return fmt.Errorf("%w: %s", ErrExists, filepath.Base(dst))
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	dir := filepath.Dir(dst)
	f, err := os.CreateTemp(dir, ".rnparmor-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	ok := false
	defer func() {
		if !ok {
			f.Close()
			os.Remove(tmp)
		}
	}()
	if err := f.Chmod(mode); err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		return err
	}
	if err := f.Sync(); err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := publish(tmp, dst); err != nil {
		return err
	}
	ok = true
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		d.Close()
	}
	return nil
}
