package rnp

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ebitengine/purego"
)

func libPath(t *testing.T) string {
	t.Helper()
	p := os.Getenv("RNPARMOR_LIBRNP")
	if p == "" {
		t.Fatal("RNPARMOR_LIBRNP is not set")
	}
	if !filepath.IsAbs(p) {
		t.Fatalf("RNPARMOR_LIBRNP is not absolute")
	}
	return p
}

func newPass(t *testing.T) []byte {
	t.Helper()
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		t.Fatal(err)
	}
	const abc = "abcdefghijklmnopqrstuvwxyz0123456789"
	out := make([]byte, len(raw))
	for i := range out {
		out[i] = abc[int(raw[i])%len(abc)]
	}
	wipe(raw)
	t.Cleanup(func() { wipe(out) })
	return out
}

func openSession(t *testing.T, pass []byte) *Session {
	t.Helper()
	s, err := Open(libPath(t), pass)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestProbeAndVersion(t *testing.T) {
	v, err := Probe(libPath(t))
	if err != nil {
		t.Fatal(err)
	}
	if v == "" || !bytes.Contains([]byte(v), []byte("0.18.1")) {
		t.Fatalf("version %q", v)
	}
	pass := newPass(t)
	s := openSession(t, pass)
	got, err := s.Version(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got != v {
		t.Fatalf("session version %q, probe %q", got, v)
	}
}

func TestPassphraseLifecycleAndThread(t *testing.T) {
	pass := newPass(t)
	s := openSession(t, pass)
	ctx := context.Background()
	before := s.PasswordCalls()
	if err := s.RequestPassword(ctx); err != nil {
		t.Fatal(err)
	}
	if err := s.RequestPassword(ctx); err != nil {
		t.Fatal(err)
	}
	if s.PasswordCalls() < before+2 {
		t.Fatalf("calls %d, want at least %d", s.PasswordCalls(), before+2)
	}
	if currentTID() != 0 && s.callbackTID() != s.enteredTID {
		t.Fatalf("callback tid %d entered %d", s.callbackTID(), s.enteredTID)
	}
	if err := s.SetPassCancel(true); err != nil {
		t.Fatal(err)
	}
	if err := s.RequestPassword(ctx); err == nil {
		t.Fatal("cancel returned a passphrase")
	}
	if err := s.SetPassCancel(false); err != nil {
		t.Fatal(err)
	}
	other := newPass(t)
	if err := s.SetPassphrase(other); err != nil {
		t.Fatal(err)
	}
	if err := s.RequestPassword(ctx); err != nil {
		t.Fatal(err)
	}
	slot := s.slot
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if err := s.RequestPassword(ctx); !errors.Is(err, ErrClosed) {
		t.Fatalf("after close: %v", err)
	}
	if passCallback(purego.CDecl{}, 0, slot, 0, 0, 0, 0) {
		t.Fatal("callback succeeded after the slot was dropped")
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestGenerateEncryptDecryptAndTamper(t *testing.T) {
	pass := newPass(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	s := openSession(t, pass)
	ctx := context.Background()
	if err := s.Generate(ctx); err != nil {
		t.Fatal(err)
	}
	if s.PasswordCalls() < 2 {
		t.Fatalf("generate calls %d, want at least 2", s.PasswordCalls())
	}
	pub, sec, err := s.Counts(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if pub < 2 || sec < 2 {
		t.Fatalf("keys public %d secret %d", pub, sec)
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "plain.bin")
	body := []byte("rnp-armor vertical slice plaintext")
	if err := os.WriteFile(src, body, 0o600); err != nil {
		t.Fatal(err)
	}
	enc := filepath.Join(dir, "message.asc")
	if err := s.EncryptFile(ctx, src, enc); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out.bin")
	if err := s.DecryptFile(ctx, enc, out); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, body) {
		t.Fatal("decrypted bytes differ")
	}

	wrong := newPass(t)
	if err := s.SetPassphrase(wrong); err != nil {
		t.Fatal(err)
	}
	bad := filepath.Join(dir, "wrong.bin")
	if err := s.DecryptFile(ctx, enc, bad); err == nil {
		t.Fatal("wrong passphrase decrypted")
	}
	if _, err := os.Lstat(bad); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("wrong passphrase left an output")
	}
	if err := s.SetPassphrase(pass); err != nil {
		t.Fatal(err)
	}
	if err := s.SetPassCancel(true); err != nil {
		t.Fatal(err)
	}
	cancelled := filepath.Join(dir, "cancel.bin")
	if err := s.DecryptFile(ctx, enc, cancelled); err == nil {
		t.Fatal("cancelled decrypt succeeded")
	}
	if _, err := os.Lstat(cancelled); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("cancelled decrypt left an output")
	}
	if err := s.SetPassCancel(false); err != nil {
		t.Fatal(err)
	}

	cipher, err := os.ReadFile(enc)
	if err != nil {
		t.Fatal(err)
	}
	cipher[len(cipher)/2] ^= 0x4
	tampered := filepath.Join(dir, "tampered.asc")
	if err := os.WriteFile(tampered, cipher, 0o600); err != nil {
		t.Fatal(err)
	}
	plain2 := filepath.Join(dir, "tampered.bin")
	if err := s.DecryptFile(ctx, tampered, plain2); err == nil {
		t.Fatal("tampered ciphertext decrypted")
	}
	if _, err := os.Lstat(plain2); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("tampered decrypt left plaintext")
	}
	entries, err := os.ReadDir(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("HOME gained %d entries", len(entries))
	}
}

func TestSignVerifyIsNotIdentityTrust(t *testing.T) {
	pass := newPass(t)
	s := openSession(t, pass)
	ctx := context.Background()
	if err := s.Generate(ctx); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "data.bin")
	if err := os.WriteFile(src, []byte("signed-message"), 0o600); err != nil {
		t.Fatal(err)
	}
	sig := filepath.Join(dir, "data.sig")
	if err := s.SignFile(ctx, src, sig); err != nil {
		t.Fatal(err)
	}
	res, err := s.VerifyFile(ctx, src, sig)
	if err != nil {
		t.Fatal(err)
	}
	if !res.MathValid || res.IdentityTrusted {
		t.Fatalf("result %+v", res)
	}
	raw, err := os.ReadFile(sig)
	if err != nil {
		t.Fatal(err)
	}
	raw[len(raw)/2] ^= 0x2
	bad := filepath.Join(dir, "bad.sig")
	if err := os.WriteFile(bad, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	res, err = s.VerifyFile(ctx, src, bad)
	if res.MathValid || res.IdentityTrusted {
		t.Fatalf("tampered signature %+v err %v", res, err)
	}
	msg, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	msg[0] ^= 0x1
	src2 := filepath.Join(dir, "changed.bin")
	if err := os.WriteFile(src2, msg, 0o600); err != nil {
		t.Fatal(err)
	}
	fresh, err := os.ReadFile(sig)
	if err != nil {
		t.Fatal(err)
	}
	// sig was tampered in memory only; the file is still the original.
	_ = fresh
	res, err = s.VerifyFile(ctx, src2, sig)
	if res.MathValid || res.IdentityTrusted {
		t.Fatalf("tampered message %+v err %v", res, err)
	}
}

func TestCommitRefusesOverwriteAndUnusableDir(t *testing.T) {
	pass := newPass(t)
	s := openSession(t, pass)
	ctx := context.Background()
	if err := s.Generate(ctx); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "in.bin")
	if err := os.WriteFile(src, []byte("keep-existing"), 0o600); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "exists.asc")
	if err := os.WriteFile(dst, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	err := s.EncryptFile(ctx, src, dst)
	if !errors.Is(err, ErrExists) {
		t.Fatalf("overwrite: %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "original" {
		t.Fatalf("existing file changed to %q", got)
	}
	if err := s.EncryptFile(ctx, "relative.bin", filepath.Join(dir, "x.asc")); !errors.Is(err, ErrRelative) {
		t.Fatalf("relative: %v", err)
	}
	empty := filepath.Join(dir, "empty.bin")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := s.EncryptFile(ctx, empty, filepath.Join(dir, "empty.asc")); !errors.Is(err, ErrEmpty) {
		t.Fatalf("empty: %v", err)
	}
	huge := filepath.Join(dir, "huge.bin")
	f, err := os.Create(huge)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.Truncate(MaxInput + 1); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if err := s.EncryptFile(ctx, huge, filepath.Join(dir, "huge.asc")); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("huge: %v", err)
	}

	ro := t.TempDir()
	if err := os.Chmod(ro, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(ro, 0o755) })
	roDst := filepath.Join(ro, "out.asc")
	err = s.EncryptFile(ctx, src, roDst)
	if err == nil {
		if euid() == 0 {
			t.Log("root bypasses directory mode bits; checking a non-directory parent instead")
		} else {
			t.Fatal("read-only directory accepted an output")
		}
	}
	if _, statErr := os.Lstat(roDst); err != nil && !errors.Is(statErr, os.ErrNotExist) && euid() != 0 {
		t.Fatal(statErr)
	}
	parent := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(parent, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := commitNew(filepath.Join(parent, "child"), []byte("z"), 0o600); err == nil {
		t.Fatal("commit into a file succeeded")
	}
}

func TestGoDoesNotBlockCallerAndCancelSkipsCommit(t *testing.T) {
	pass := newPass(t)
	s := openSession(t, pass)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	start := time.Now()
	ch := s.Go(ctx, func(ctx context.Context) error {
		var n uintptr
		if err := bound.check("calculate_iterations", bound.calcIter("SHA256", 300, &n)); err != nil {
			return err
		}
		return ctx.Err()
	})
	if time.Since(start) > 80*time.Millisecond {
		t.Fatalf("Go blocked the caller for %s", time.Since(start))
	}
	select {
	case err := <-ch:
		t.Fatalf("work finished before the caller continued: %v", err)
	case <-time.After(40 * time.Millisecond):
	}
	if err := <-ch; err != nil {
		t.Fatal(err)
	}

	ctx2, cancel2 := context.WithCancel(context.Background())
	dst := filepath.Join(t.TempDir(), "should-not-exist")
	ch = s.Go(ctx2, func(ctx context.Context) error {
		var n uintptr
		if err := bound.check("calculate_iterations", bound.calcIter("SHA256", 300, &n)); err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		return commitNew(dst, []byte("late"), 0o600)
	})
	time.Sleep(40 * time.Millisecond)
	cancel2()
	err := <-ch
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel: %v", err)
	}
	if _, err := os.Lstat(dst); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("cancelled operation published output")
	}
	cancelEarly, cancelFn := context.WithCancel(context.Background())
	cancelFn()
	if err := s.Burn(cancelEarly, 50); !errors.Is(err, context.Canceled) {
		t.Fatalf("already cancelled: %v", err)
	}
}

func TestPersistIsExplicit(t *testing.T) {
	pass := newPass(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	s := openSession(t, pass)
	ctx := context.Background()
	if err := s.Generate(ctx); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "in.bin")
	body := []byte("persist-check")
	if err := os.WriteFile(src, body, 0o600); err != nil {
		t.Fatal(err)
	}
	enc := filepath.Join(dir, "in.asc")
	if err := s.EncryptFile(ctx, src, enc); err != nil {
		t.Fatal(err)
	}
	keys := t.TempDir()
	if err := s.SaveTo(ctx, keys); err != nil {
		t.Fatal(err)
	}
	secInfo, err := os.Stat(filepath.Join(keys, "secring.gpg"))
	if err != nil {
		t.Fatal(err)
	}
	if secInfo.Mode().Perm() != 0o600 {
		t.Fatalf("secring mode %o", secInfo.Mode().Perm())
	}
	if err := s.SaveTo(ctx, keys); !errors.Is(err, ErrExists) {
		t.Fatalf("second save: %v", err)
	}
	if err := s.SaveTo(ctx, ""); !errors.Is(err, ErrRelative) {
		t.Fatalf("empty dir: %v", err)
	}
	s2 := openSession(t, pass)
	if err := s2.LoadFrom(ctx, keys); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "from-loaded.bin")
	if err := s2.DecryptFile(ctx, enc, out); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, body) {
		t.Fatal("loaded key did not decrypt")
	}
	entries, err := os.ReadDir(home)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Fatalf("default home was used, %d entries", len(entries))
	}
}

func TestCallbackReenterIsRejected(t *testing.T) {
	pass := newPass(t)
	s := openSession(t, pass)
	inCallback.Store(true)
	defer inCallback.Store(false)
	err := s.RequestPassword(context.Background())
	if !errors.Is(err, ErrReenter) {
		t.Fatalf("got %v", err)
	}
}
