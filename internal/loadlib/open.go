// Package loadlib opens librnp from an absolute path after checking a
// SHA256SUMS file in the same directory. It does not search the working
// directory or PATH.
package loadlib

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const SumFile = "SHA256SUMS"

var (
	ErrRelative = errors.New("librnp path must be absolute")
	ErrChecksum = errors.New("librnp bundle checksum failed")
)

// Resolve returns the library path to open. RNPARMOR_LIBRNP must be
// absolute when it is set. Otherwise the only candidate is the platform
// library name next to the executable.
func Resolve() (string, error) {
	if p := os.Getenv("RNPARMOR_LIBRNP"); p != "" {
		if !filepath.IsAbs(p) {
			return "", fmt.Errorf("%w: RNPARMOR_LIBRNP", ErrRelative)
		}
		return filepath.Clean(p), nil
	}
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(exe), LibraryFile), nil
}

// Open checks path and every file named by the sibling SHA256SUMS, then
// loads the library. path itself must be listed.
func Open(path string) (uintptr, error) {
	if err := Verify(path); err != nil {
		return 0, err
	}
	return openShared(path)
}

// Verify checks the SHA256SUMS beside path. It does not load the library.
func Verify(path string) error {
	if !filepath.IsAbs(path) {
		return ErrRelative
	}
	path = filepath.Clean(path)
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	sums := filepath.Join(dir, SumFile)
	f, err := os.Open(sums)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrChecksum, err)
	}
	defer f.Close()
	listed := false
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		hexsum, name, ok := strings.Cut(line, " ")
		if !ok {
			return fmt.Errorf("%w: malformed line", ErrChecksum)
		}
		name = strings.TrimSpace(strings.ReplaceAll(name, `\`, "/"))
		hexsum = strings.TrimSpace(hexsum)
		full, err := bundlePath(dir, name)
		if err != nil {
			return err
		}
		if err := matchFile(full, hexsum); err != nil {
			return err
		}
		if name == base {
			listed = true
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}
	if !listed {
		return fmt.Errorf("%w: %s is not listed", ErrChecksum, base)
	}
	return nil
}

// bundlePath resolves name inside dir. Names may use slashes for files in
// subdirectories. ".." and absolute names are rejected.
func bundlePath(dir, name string) (string, error) {
	if name == "" || strings.Contains(name, ":") || strings.HasPrefix(name, "/") {
		return "", fmt.Errorf("%w: bad name %q", ErrChecksum, name)
	}
	for _, part := range strings.Split(name, "/") {
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("%w: bad name %q", ErrChecksum, name)
		}
	}
	full := filepath.Join(dir, filepath.FromSlash(name))
	rel, err := filepath.Rel(dir, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%w: bad name %q", ErrChecksum, name)
	}
	return full, nil
}

func matchFile(path, want string) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrChecksum, err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("%w: %s", ErrChecksum, filepath.Base(path))
	}
	return nil
}
