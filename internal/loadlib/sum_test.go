package loadlib

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyRejectsRelativeAndTamper(t *testing.T) {
	if err := Verify("librnp.so"); err != ErrRelative {
		t.Fatalf("relative: %v", err)
	}
	dir := t.TempDir()
	lib := filepath.Join(dir, "librnp.so")
	other := filepath.Join(dir, "libbotan-3.so")
	if err := os.WriteFile(lib, []byte("librnp-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(other, []byte("botan-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeSums(t, dir, lib, other)
	if err := Verify(lib); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(other, []byte("replaced"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Verify(lib); err == nil {
		t.Fatal("tampered dependency was accepted")
	}
}

func TestResolveRejectsRelativeEnv(t *testing.T) {
	t.Setenv("RNPARMOR_LIBRNP", "librnp.so")
	if _, err := Resolve(); !errors.Is(err, ErrRelative) {
		t.Fatalf("got %v", err)
	}
}

func writeSums(t *testing.T, dir string, files ...string) {
	t.Helper()
	var b []byte
	for _, f := range files {
		sum := sha256.Sum256(mustRead(t, f))
		b = append(b, []byte(hex.EncodeToString(sum[:])+"  "+filepath.Base(f)+"\n")...)
	}
	if err := os.WriteFile(filepath.Join(dir, SumFile), b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
