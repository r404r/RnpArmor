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
	if err := os.WriteFile(filepath.Join(dir, "SHA256SUMS"), []byte("aabbcc  ../librnp.so\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Verify(lib); err == nil {
		t.Fatal("parent path in checksum was accepted")
	}
	sub := filepath.Join(dir, "licenses")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	notice := filepath.Join(sub, "notice.txt")
	if err := os.WriteFile(notice, []byte("notice"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeSums(t, dir, lib, notice)
	// writeSums records only the base name. Rewrite the notice line with its relative path.
	sum := sha256.Sum256(mustRead(t, lib))
	nsum := sha256.Sum256(mustRead(t, notice))
	text := hex.EncodeToString(sum[:]) + "  " + filepath.Base(lib) + "\n" +
		hex.EncodeToString(nsum[:]) + "  licenses/notice.txt\n"
	if err := os.WriteFile(filepath.Join(dir, SumFile), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Verify(lib); err != nil {
		t.Fatal(err)
	}
}

func TestResolveRejectsRelativeEnv(t *testing.T) {
	t.Setenv("RNPARMOR_LIBRNP", "librnp.so")
	if _, err := Resolve(); !errors.Is(err, ErrRelative) {
		t.Fatalf("got %v", err)
	}
}

func TestVerifyMacOSBundleResourcesFallback(t *testing.T) {
	root := t.TempDir()
	macos := filepath.Join(root, "RnpArmor.app", "Contents", "MacOS")
	res := filepath.Join(root, "RnpArmor.app", "Contents", "Resources")
	if err := os.MkdirAll(macos, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(res, 0o755); err != nil {
		t.Fatal(err)
	}
	lib := filepath.Join(macos, "librnp.dylib")
	if err := os.WriteFile(lib, []byte("signed-dylib-bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(mustRead(t, lib))
	line := hex.EncodeToString(sum[:]) + "  librnp.dylib\n"
	if err := os.WriteFile(filepath.Join(res, SumFile), []byte(line), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Verify(lib); err != nil {
		t.Fatalf("fallback verify failed: %v", err)
	}

	if err := os.WriteFile(lib, []byte("tampered"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Verify(lib); err == nil {
		t.Fatal("tampered dylib was accepted with Resources/SHA256SUMS fallback")
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
