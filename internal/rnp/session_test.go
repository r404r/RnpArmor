package rnp

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"strings"
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
	if passCallback(purego.CDecl{}, 0, slot, 0, 0, 0, 0) != 0 {
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
	if err := makeUnwritable(ro); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = restoreWritable(ro) })
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
	if !secretRestricted(secInfo) {
		t.Fatalf("secring mode %o size %d", secInfo.Mode().Perm(), secInfo.Size())
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

func TestListKeysGenerateImportExportJSONAndDelete(t *testing.T) {
	pass := newPass(t)
	s := openSession(t, pass)
	ctx := context.Background()

	if _, err := s.GenerateKeyWithSpec(ctx, KeyGenRequest{UserID: "Alice <alice@example.com>", PrimaryAlg: "RSA", PrimaryBits: 2048, SubkeyAlg: "RSA", SubkeyBits: 2048}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GenerateKeyWithSpec(ctx, KeyGenRequest{UserID: "Bob <bob@example.com>", PrimaryAlg: "ECDSA", PrimaryCurve: "NIST P-256", SubkeyAlg: "ECDH", SubkeyCurve: "Curve25519"}); err != nil {
		t.Fatal(err)
	}

	list, err := s.ListKeys(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) < 2 {
		t.Fatalf("list len=%d", len(list))
	}
	for _, k := range list {
		if k.Fingerprint == "" || k.KeyID == "" {
			t.Fatalf("bad key summary: %+v", k)
		}
	}

	var primary KeySummary
	found := false
	for _, k := range list {
		if k.HasSecret && k.CanCertify {
			primary = k
			found = true
			break
		}
	}
	if !found {
		primary = list[0]
	}

	pub, err := s.ExportKey(ctx, primary.Fingerprint, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(pub, "BEGIN PGP PUBLIC KEY BLOCK") {
		t.Fatalf("unexpected public key export: %q", pub)
	}

	sec, err := s.ExportKey(ctx, primary.Fingerprint, true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(sec, "BEGIN PGP PRIVATE KEY BLOCK") {
		t.Fatalf("unexpected secret key export: %q", sec)
	}

	jsonText, err := s.KeyToJSON(ctx, primary.Fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.ToLower(jsonText), strings.ToLower(primary.KeyID)) {
		t.Fatalf("key json missing keyid: %s", jsonText)
	}

	rev, err := s.ExportRevocation(ctx, primary.Fingerprint, "retired", "test")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(rev, "BEGIN PGP PUBLIC KEY BLOCK") && !strings.Contains(rev, "BEGIN PGP SIGNATURE") {
		t.Fatalf("unexpected revocation export: %q", rev)
	}

	victim := ""
	for _, k := range list {
		if k.Fingerprint != primary.Fingerprint {
			victim = k.Fingerprint
			break
		}
	}
	if victim == "" {
		victim = primary.Fingerprint
	}
	if err := s.DeleteKey(ctx, victim); err != nil {
		t.Fatal(err)
	}
	left, err := s.ListKeys(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(left) >= len(list) {
		t.Fatalf("delete did not reduce list: before=%d after=%d", len(list), len(left))
	}

	s2 := openSession(t, pass)
	if _, err := s2.ImportKeys(ctx, []byte(pub)); err != nil {
		t.Fatal(err)
	}
	l2, err := s2.ListKeys(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(l2) == 0 {
		t.Fatal("imported list is empty")
	}
}

func TestTextEncryptDecryptAndSignVerifyModes(t *testing.T) {
	pass := newPass(t)
	s := openSession(t, pass)
	ctx := context.Background()
	key, err := s.GenerateKeyWithSpec(ctx, KeyGenRequest{UserID: "Text Tester <text@example.com>"})
	if err != nil {
		t.Fatal(err)
	}

	plain := "hello text api"
	cipher, err := s.EncryptText(ctx, plain, EncryptOptions{RecipientFprints: []string{key.Fingerprint}, Cipher: "AES256", Compression: "zlib", Armor: true})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(cipher, "BEGIN PGP MESSAGE") {
		t.Fatalf("cipher not armored: %q", cipher)
	}
	dec, err := s.DecryptText(ctx, cipher)
	if err != nil {
		t.Fatal(err)
	}
	if dec.Plaintext != plain {
		t.Fatalf("decrypt mismatch: %q", dec.Plaintext)
	}

	cipher2, err := s.EncryptText(ctx, plain, EncryptOptions{Password: "secret-pass", Cipher: "AES192", Compression: "zip", Armor: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetPassphrase([]byte("secret-pass")); err != nil {
		t.Fatal(err)
	}
	dec2, err := s.DecryptText(ctx, cipher2)
	if err != nil {
		t.Fatal(err)
	}
	if dec2.Plaintext != plain {
		t.Fatalf("password decrypt mismatch: %q", dec2.Plaintext)
	}
	if err := s.SetPassphrase(pass); err != nil {
		t.Fatal(err)
	}

	clear, err := s.SignText(ctx, plain, key.Fingerprint, SignModeCleartext, "SHA256")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(clear, "BEGIN PGP SIGNED MESSAGE") {
		t.Fatalf("cleartext sign mismatch: %q", clear)
	}
	vr, err := s.VerifyText(ctx, clear, "")
	if err != nil {
		t.Fatal(err)
	}
	if !vr.MathValid {
		t.Fatalf("clearsign invalid: %+v", vr)
	}

	det, err := s.SignText(ctx, plain, key.Fingerprint, SignModeDetached, "SHA256")
	if err != nil {
		t.Fatal(err)
	}
	vr, err = s.VerifyText(ctx, plain, det)
	if err != nil {
		t.Fatal(err)
	}
	if !vr.MathValid {
		t.Fatalf("detached invalid: %+v", vr)
	}

	inline, err := s.SignText(ctx, plain, key.Fingerprint, SignModeInline, "SHA512")
	if err != nil {
		t.Fatal(err)
	}
	vr, err = s.VerifyText(ctx, inline, "")
	if err != nil {
		t.Fatal(err)
	}
	if !vr.MathValid {
		t.Fatalf("inline invalid: %+v", vr)
	}
}

func TestFileOptsDumpAndFeatures(t *testing.T) {
	pass := newPass(t)
	s := openSession(t, pass)
	ctx := context.Background()
	key, err := s.GenerateKeyWithSpec(ctx, KeyGenRequest{UserID: "File Tester <file@example.com>"})
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	src := filepath.Join(dir, "in.txt")
	if err := os.WriteFile(src, []byte("file payload"), 0o600); err != nil {
		t.Fatal(err)
	}

	enc := filepath.Join(dir, "enc.asc")
	if err := s.EncryptFileWithOpts(ctx, src, enc, EncryptOptions{RecipientFprints: []string{key.Fingerprint}, SignerFprint: key.Fingerprint, Cipher: "AES256", Compression: "bzip2", CompressionLevel: 6, Armor: true}); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(dir, "out.txt")
	if err := s.DecryptFile(ctx, enc, out); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "file payload" {
		t.Fatalf("decrypt got %q", string(got))
	}

	sig := filepath.Join(dir, "det.sig")
	if err := s.SignFileWithOpts(ctx, src, sig, key.Fingerprint, SignModeDetached, true, "SHA256"); err != nil {
		t.Fatal(err)
	}
	v, err := s.VerifyFileWithOpts(ctx, src, sig, true)
	if err != nil {
		t.Fatal(err)
	}
	if !v.MathValid {
		t.Fatalf("detached file invalid: %+v", v)
	}

	inline := filepath.Join(dir, "inline.pgp")
	if err := s.SignFileWithOpts(ctx, src, inline, key.Fingerprint, SignModeInline, true, "SHA256"); err != nil {
		t.Fatal(err)
	}
	v, err = s.VerifyFileWithOpts(ctx, inline, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if !v.MathValid {
		t.Fatalf("inline file invalid: %+v", v)
	}

	encBytes, err := os.ReadFile(enc)
	if err != nil {
		t.Fatal(err)
	}
	dump, err := s.DumpPackets(ctx, encBytes, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(strings.TrimSpace(dump)) == 0 {
		t.Fatal("empty packet dump")
	}
	dumpJSON, err := s.DumpPackets(ctx, encBytes, true)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(dumpJSON, "header") || !strings.Contains(dumpJSON, "tag") {
		t.Fatalf("unexpected packet json: %s", dumpJSON)
	}

	feat, err := s.SupportedFeatures(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(feat, "symmetric algorithm") || !strings.Contains(feat, "hash algorithm") {
		t.Fatalf("feature matrix missing categories: %s", feat)
	}
	one, err := s.SupportedFeatures(ctx, "public key algorithm:RSA")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(one, "supported") {
		t.Fatalf("single support result malformed: %s", one)
	}
}

func TestOpenOptionalPassPublicOnlyRoundTrip(t *testing.T) {
	lib := libPath(t)
	ctx := context.Background()

	if _, err := Open(lib, nil); !errors.Is(err, ErrPass) {
		t.Fatalf("Open(nil) = %v, want ErrPass", err)
	}
	if _, err := OpenOptionalPass(lib, bytes.Repeat([]byte("a"), maxPass+1)); !errors.Is(err, ErrPass) {
		t.Fatalf("OpenOptionalPass(too long) = %v", err)
	}

	empty, err := OpenOptionalPass(lib, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = empty.Close() })
	if _, err := empty.ListKeys(ctx); err != nil {
		t.Fatal(err)
	}
	if err := empty.SaveTo(ctx, t.TempDir()); !errors.Is(err, ErrEmpty) {
		t.Fatalf("empty SaveTo = %v, want ErrEmpty", err)
	}
	if err := empty.SaveTo(ctx, "relative"); !errors.Is(err, ErrRelative) {
		t.Fatalf("relative SaveTo = %v", err)
	}

	pass := newPass(t)
	full := openSession(t, pass)
	sum, err := full.GenerateKeyWithSpec(ctx, KeyGenRequest{UserID: "Pub Only <pub@example.com>"})
	if err != nil {
		t.Fatal(err)
	}
	if !sum.IsPrimary || sum.IsSub {
		t.Fatalf("generated primary flags: IsPrimary=%v IsSub=%v", sum.IsPrimary, sum.IsSub)
	}
	if len(sum.Subkeys) == 0 {
		t.Fatal("generated primary has no subkeys")
	}
	for _, sub := range sum.Subkeys {
		if sub.IsPrimary || !sub.IsSub || len(sub.Subkeys) != 0 {
			t.Fatalf("subkey summary: %+v", sub)
		}
	}
	pub, err := full.ExportKey(ctx, sum.Fingerprint, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(pub, "BEGIN PGP PUBLIC KEY BLOCK") {
		t.Fatalf("public export: %q", pub)
	}

	ro, err := OpenOptionalPass(lib, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ro.Close() })
	if _, err := ro.ImportKeys(ctx, []byte(pub)); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	if err := ro.SaveTo(ctx, dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "pubring.gpg")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "secring.gpg")); !os.IsNotExist(err) {
		t.Fatalf("public-only secring: %v", err)
	}
	if err := ro.SaveTo(ctx, dir); !errors.Is(err, ErrExists) {
		t.Fatalf("second public save = %v", err)
	}

	ro2, err := OpenOptionalPass(lib, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ro2.Close() })
	if err := ro2.LoadFrom(ctx, dir); err != nil {
		t.Fatal(err)
	}
	list, err := ro2.ListKeys(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var loaded *KeySummary
	for i := range list {
		if strings.EqualFold(list[i].Fingerprint, sum.Fingerprint) && list[i].IsPrimary {
			loaded = &list[i]
			break
		}
	}
	if loaded == nil {
		t.Fatalf("reloaded primary missing from %d keys", len(list))
	}
	if loaded.HasSecret {
		t.Fatal("public-only reload has a secret")
	}
	if len(loaded.Subkeys) == 0 {
		t.Fatal("reloaded primary has no subkeys")
	}
	if err := ro2.LoadFrom(ctx, t.TempDir()); !os.IsNotExist(err) {
		t.Fatalf("missing keyring LoadFrom = %v", err)
	}
	if err := ro2.LoadFrom(ctx, "relative"); !errors.Is(err, ErrRelative) {
		t.Fatalf("relative LoadFrom = %v", err)
	}
}

func TestSubkeyLifecycle(t *testing.T) {
	pass := newPass(t)
	s := openSession(t, pass)
	ctx := context.Background()
	primary, err := s.GenerateKeyWithSpec(ctx, KeyGenRequest{
		UserID:       "Sub <sub@example.com>",
		PrimaryAlg:   "EDDSA",
		PrimaryCurve: "Ed25519",
		SubkeyAlg:    "ECDH",
		SubkeyCurve:  "Curve25519",
		HashAlg:      "SHA256",
		SymmCipher:   "AES256",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !primary.IsPrimary || !primary.HasSecret || primary.CanAuth {
		t.Fatalf("primary %+v", primary)
	}
	original := map[string]bool{}
	for _, sk := range primary.Subkeys {
		original[sk.Fingerprint] = true
	}

	enc, err := s.AddSubkey(ctx, primary.Fingerprint, SubkeyGenRequest{
		Usage: "encrypt", Algorithm: "ECDH", Curve: "Curve25519", HashAlg: "SHA256", Cipher: "AES256",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !enc.IsSub || !enc.CanEncrypt || enc.CanSign || enc.CanAuth || enc.Fingerprint == "" {
		t.Fatalf("encrypt subkey %+v", enc)
	}
	sig, err := s.AddSubkey(ctx, primary.Fingerprint, SubkeyGenRequest{
		Usage: "sign", Algorithm: "EDDSA", Curve: "Ed25519", HashAlg: "SHA256", Cipher: "AES256",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !sig.IsSub || !sig.CanSign || sig.CanEncrypt || sig.CanAuth {
		t.Fatalf("sign subkey %+v", sig)
	}
	auth, err := s.AddSubkey(ctx, primary.Fingerprint, SubkeyGenRequest{
		Usage: "authenticate", Algorithm: "EDDSA", Curve: "Ed25519", HashAlg: "SHA256", Cipher: "AES256",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !auth.IsSub || !auth.CanAuth || auth.CanEncrypt {
		t.Fatalf("auth subkey %+v", auth)
	}
	if _, err := s.AddSubkey(ctx, primary.Fingerprint, SubkeyGenRequest{Usage: "encrypt", Algorithm: "EDDSA"}); err == nil {
		t.Fatal("eddsa encrypt subkey was accepted")
	}
	if _, err := s.AddSubkey(ctx, primary.Fingerprint, SubkeyGenRequest{Usage: "sign", Algorithm: "ECDH", Curve: "Curve25519"}); err == nil {
		t.Fatal("ecdh sign subkey was accepted")
	}

	const year = uint32(365 * 24 * 60 * 60)
	if err := s.SetKeyExpiration(ctx, sig.Fingerprint, year); err != nil {
		t.Fatal(err)
	}
	if err := s.RevokeKey(ctx, auth.Fingerprint, "compromised", "lifecycle"); err != nil {
		t.Fatal(err)
	}
	if err := s.RevokeKey(ctx, enc.Fingerprint, "not-a-code", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteSubkey(ctx, primary.Fingerprint); err == nil {
		t.Fatal("deleted the primary key")
	}
	if err := s.DeleteSubkey(ctx, enc.Fingerprint); err != nil {
		t.Fatal(err)
	}

	list, err := s.ListKeys(ctx)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := primaryIn(list, primary.Fingerprint)
	if !ok {
		t.Fatal("primary missing after lifecycle")
	}
	if got.IsRevoked || !got.HasSecret {
		t.Fatalf("primary damaged %+v", got)
	}
	if subIn(got, enc.Fingerprint) {
		t.Fatal("deleted encrypt subkey still present")
	}
	sigNow, ok := subOf(got, sig.Fingerprint)
	if !ok || sigNow.IsRevoked || sigNow.ExpiresUnix != sigNow.CreatedUnix+year {
		t.Fatalf("sign subkey %+v", sigNow)
	}
	authNow, ok := subOf(got, auth.Fingerprint)
	if !ok || !authNow.IsRevoked || !authNow.CanAuth {
		t.Fatalf("auth subkey %+v", authNow)
	}
	for fp := range original {
		if !subIn(got, fp) {
			t.Fatalf("original subkey %s missing", fp)
		}
		if sk, ok := subOf(got, fp); ok && sk.IsRevoked {
			t.Fatalf("original subkey revoked %+v", sk)
		}
	}

	dir := t.TempDir()
	if err := s.SaveTo(ctx, dir); err != nil {
		t.Fatal(err)
	}
	s2 := openSession(t, pass)
	if err := s2.LoadFrom(ctx, dir); err != nil {
		t.Fatal(err)
	}
	loaded, err := s2.ListKeys(ctx)
	if err != nil {
		t.Fatal(err)
	}
	back, ok := primaryIn(loaded, primary.Fingerprint)
	if !ok || back.IsRevoked {
		t.Fatalf("reloaded primary %+v", back)
	}
	if subIn(back, enc.Fingerprint) {
		t.Fatal("deleted subkey survived save/load")
	}
	sigBack, ok := subOf(back, sig.Fingerprint)
	if !ok || sigBack.IsRevoked || sigBack.ExpiresUnix != sigNow.ExpiresUnix {
		t.Fatalf("reloaded sign subkey %+v", sigBack)
	}
	authBack, ok := subOf(back, auth.Fingerprint)
	if !ok || !authBack.IsRevoked || !authBack.CanAuth {
		t.Fatalf("reloaded auth subkey %+v", authBack)
	}
}

func primaryIn(list []KeySummary, fp string) (KeySummary, bool) {
	for _, k := range list {
		if k.IsPrimary && strings.EqualFold(k.Fingerprint, fp) {
			return k, true
		}
	}
	return KeySummary{}, false
}

func subOf(primary KeySummary, fp string) (KeySummary, bool) {
	for _, sk := range primary.Subkeys {
		if strings.EqualFold(sk.Fingerprint, fp) {
			return sk, true
		}
	}
	return KeySummary{}, false
}

func subIn(primary KeySummary, fp string) bool {
	_, ok := subOf(primary, fp)
	return ok
}
