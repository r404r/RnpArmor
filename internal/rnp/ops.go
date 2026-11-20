package rnp

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"unsafe"
)

const (
	flagPublic uint32 = 1 << 0
	flagSecret uint32 = 1 << 1
)

// Generate creates a new EDDSA primary key and a Curve25519 encryption
// subkey in this ffi. The passphrase is taken only from the callback.
// Nothing is written to disk.
func (s *Session) Generate(ctx context.Context) error {
	return s.Do(ctx, func(context.Context) error {
		return s.generate()
	})
}

func (s *Session) generate() error {
	sl := slotByID(s.slot)
	if sl == nil {
		return ErrClosed
	}
	sl.mu.Lock()
	empty := len(sl.pass) == 0 || sl.cancel
	sl.mu.Unlock()
	if empty {
		return ErrPass
	}
	if s.primary != 0 {
		return errorsNew("keys already generated")
	}
	var op uintptr
	if err := bound.check("generate", bound.genCreate(&op, s.ffi, "EDDSA")); err != nil {
		return err
	}
	defer func() {
		if op != 0 {
			bound.genDestroy(op)
		}
	}()
	if err := s.protect(op); err != nil {
		return err
	}
	if err := bound.check("generate", bound.genUser(op, "rnp-armor-ephemeral")); err != nil {
		return err
	}
	if err := bound.check("generate", bound.genUsage(op, "sign")); err != nil {
		return err
	}
	if err := bound.check("generate", bound.genUsage(op, "certify")); err != nil {
		return err
	}
	if err := bound.check("generate", bound.genHash(op, "SHA256")); err != nil {
		return err
	}
	if err := bound.check("generate", bound.genExec(op)); err != nil {
		return err
	}
	var primary uintptr
	if err := bound.check("generate", bound.genKey(op, &primary)); err != nil {
		return err
	}
	bound.genDestroy(op)
	op = 0
	s.primary = primary

	if err := bound.check("generate", bound.subCreate(&op, s.ffi, s.primary, "ECDH")); err != nil {
		return err
	}
	if err := bound.check("generate", bound.genCurve(op, "Curve25519")); err != nil {
		return err
	}
	if err := s.protect(op); err != nil {
		return err
	}
	if err := bound.check("generate", bound.genUsage(op, "encrypt")); err != nil {
		return err
	}
	if err := bound.check("generate", bound.genExec(op)); err != nil {
		return err
	}
	var sub uintptr
	if err := bound.check("generate", bound.genKey(op, &sub)); err != nil {
		return err
	}
	s.encKey = sub
	if s.callbackTID() != 0 && s.callbackTID() != s.enteredTID {
		return errorsNew("password callback left the caller thread")
	}
	return nil
}

func (s *Session) protect(op uintptr) error {
	if err := bound.check("generate", bound.genReqPass(op, true)); err != nil {
		return err
	}
	if err := bound.check("generate", bound.genCipher(op, "AES256")); err != nil {
		return err
	}
	if err := bound.check("generate", bound.genProtHash(op, "SHA256")); err != nil {
		return err
	}
	if err := bound.check("generate", bound.genMode(op, "cfb")); err != nil {
		return err
	}
	return bound.check("generate", bound.genIter(op, 1024))
}

func (s *Session) EncryptFile(ctx context.Context, src, dst string) error {
	return s.Do(ctx, func(ctx context.Context) error {
		return s.encryptFile(ctx, src, dst)
	})
}

func (s *Session) encryptFile(ctx context.Context, src, dst string) error {
	if s.encKey == 0 {
		return ErrNoKey
	}
	if src == dst {
		return errorsNew("source and destination are the same")
	}
	plain, err := readLimited(src)
	if err != nil {
		return err
	}
	defer wipe(plain)
	out, err := s.encrypt(plain)
	wipe(plain)
	plain = nil
	if err != nil {
		wipe(out)
		return err
	}
	if err := ctx.Err(); err != nil {
		wipe(out)
		return err
	}
	err = commitNew(dst, out, 0o600)
	wipe(out)
	return err
}

func (s *Session) encrypt(plain []byte) ([]byte, error) {
	in, err := memInput(plain)
	if err != nil {
		return nil, err
	}
	defer bound.inDestroy(in)
	var output uintptr
	if err := bound.check("output", bound.outMem(&output, MaxOutput)); err != nil {
		return nil, err
	}
	defer bound.outDestroy(output)
	var op uintptr
	if err := bound.check("encrypt", bound.encCreate(&op, s.ffi, in, output)); err != nil {
		return nil, err
	}
	defer bound.encDestroy(op)
	if err := bound.check("encrypt", bound.encRecipient(op, s.encKey)); err != nil {
		return nil, err
	}
	if err := bound.check("encrypt", bound.encCipher(op, "AES256")); err != nil {
		return nil, err
	}
	if err := bound.check("encrypt", bound.encArmor(op, true)); err != nil {
		return nil, err
	}
	if err := bound.check("encrypt", bound.encFileName(op, "")); err != nil {
		return nil, err
	}
	if err := bound.check("encrypt", bound.encExec(op)); err != nil {
		return nil, err
	}
	return takeOutput(output)
}

func (s *Session) DecryptFile(ctx context.Context, src, dst string) error {
	return s.Do(ctx, func(ctx context.Context) error {
		return s.decryptFile(ctx, src, dst)
	})
}

func (s *Session) decryptFile(ctx context.Context, src, dst string) error {
	if src == dst {
		return errorsNew("source and destination are the same")
	}
	inBytes, err := readLimited(src)
	if err != nil {
		return err
	}
	defer wipe(inBytes)
	plain, err := s.decrypt(inBytes)
	if err != nil {
		wipe(plain)
		return err
	}
	if err := ctx.Err(); err != nil {
		wipe(plain)
		return err
	}
	err = commitNew(dst, plain, 0o600)
	wipe(plain)
	return err
}

func (s *Session) decrypt(cipher []byte) ([]byte, error) {
	in, err := memInput(cipher)
	if err != nil {
		return nil, err
	}
	defer bound.inDestroy(in)
	var output uintptr
	if err := bound.check("output", bound.outMem(&output, MaxOutput)); err != nil {
		return nil, err
	}
	defer bound.outDestroy(output)
	if err := bound.check("decrypt", bound.decrypt(s.ffi, in, output)); err != nil {
		return nil, err
	}
	return takeOutput(output)
}

func (s *Session) SignFile(ctx context.Context, src, sig string) error {
	return s.Do(ctx, func(ctx context.Context) error {
		return s.signFile(ctx, src, sig)
	})
}

func (s *Session) signFile(ctx context.Context, src, sigPath string) error {
	if s.primary == 0 {
		return ErrNoKey
	}
	data, err := readLimited(src)
	if err != nil {
		return err
	}
	defer wipe(data)
	sig, err := s.sign(data)
	if err != nil {
		wipe(sig)
		return err
	}
	if err := ctx.Err(); err != nil {
		wipe(sig)
		return err
	}
	err = commitNew(sigPath, sig, 0o600)
	wipe(sig)
	return err
}

func (s *Session) sign(data []byte) ([]byte, error) {
	in, err := memInput(data)
	if err != nil {
		return nil, err
	}
	defer bound.inDestroy(in)
	var output uintptr
	if err := bound.check("output", bound.outMem(&output, MaxOutput)); err != nil {
		return nil, err
	}
	defer bound.outDestroy(output)
	var op uintptr
	if err := bound.check("sign", bound.signDetached(&op, s.ffi, in, output)); err != nil {
		return nil, err
	}
	defer bound.signDestroy(op)
	if err := bound.check("sign", bound.signAdd(op, s.primary, new(uintptr))); err != nil {
		return nil, err
	}
	if err := bound.check("sign", bound.signHash(op, "SHA256")); err != nil {
		return nil, err
	}
	if err := bound.check("sign", bound.signArmor(op, true)); err != nil {
		return nil, err
	}
	if err := bound.check("sign", bound.signExec(op)); err != nil {
		return nil, err
	}
	return takeOutput(output)
}

func (s *Session) VerifyFile(ctx context.Context, src, sig string) (VerifyResult, error) {
	var res VerifyResult
	err := s.Do(ctx, func(context.Context) error {
		var verr error
		res, verr = s.verify(src, sig)
		return verr
	})
	return res, err
}

func (s *Session) verify(src, sigPath string) (VerifyResult, error) {
	res := VerifyResult{}
	data, err := readLimited(src)
	if err != nil {
		return res, err
	}
	sig, err := readLimited(sigPath)
	if err != nil {
		return res, err
	}
	in, err := memInput(data)
	if err != nil {
		return res, err
	}
	defer bound.inDestroy(in)
	sigIn, err := memInput(sig)
	if err != nil {
		return res, err
	}
	defer bound.inDestroy(sigIn)
	var op uintptr
	if err := bound.check("verify", bound.verDetached(&op, s.ffi, in, sigIn)); err != nil {
		return res, err
	}
	defer bound.verDestroy(op)
	exec := bound.verExec(op)
	var n uintptr
	_ = bound.verCount(op, &n)
	var status uint32
	if n > 0 {
		var handle uintptr
		if bound.verAt(op, 0, &handle) == 0 && handle != 0 {
			status = bound.verStatus(handle)
		}
	}
	res.Status = status
	if bound.resultString != nil && status != 0 {
		res.StatusText = bound.resultString(status)
	}
	res.MathValid = exec == 0 && n >= 1 && status == 0
	res.IdentityTrusted = false
	if exec != 0 {
		return res, bound.check("verify", exec)
	}
	if !res.MathValid {
		return res, bound.check("verify", status)
	}
	return res, nil
}

// SaveTo writes pubring.gpg and secring.gpg into an existing directory.
// It fails if either file exists or the directory is not writable. It does
// not choose a default directory.
func (s *Session) SaveTo(ctx context.Context, dir string) error {
	return s.Do(ctx, func(context.Context) error {
		return s.saveTo(dir)
	})
}

func (s *Session) saveTo(dir string) error {
	if !filepath.IsAbs(dir) {
		return ErrRelative
	}
	info, err := os.Stat(dir)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return errorsNew("not a directory")
	}
	pubPath := filepath.Join(dir, "pubring.gpg")
	secPath := filepath.Join(dir, "secring.gpg")
	if _, err := os.Lstat(pubPath); err == nil {
		return ErrExists
	}
	if _, err := os.Lstat(secPath); err == nil {
		return ErrExists
	}
	pub, err := s.exportKeys(flagPublic)
	if err != nil {
		return err
	}
	sec, err := s.exportKeys(flagSecret)
	if err != nil {
		wipe(pub)
		return err
	}
	if err := commitNew(pubPath, pub, 0o644); err != nil {
		wipe(pub)
		wipe(sec)
		return err
	}
	wipe(pub)
	if err := commitNew(secPath, sec, 0o600); err != nil {
		os.Remove(pubPath)
		wipe(sec)
		return err
	}
	wipe(sec)
	return nil
}

func (s *Session) exportKeys(flags uint32) ([]byte, error) {
	var output uintptr
	if err := bound.check("output", bound.outMem(&output, MaxOutput)); err != nil {
		return nil, err
	}
	defer bound.outDestroy(output)
	if err := bound.check("save_keys", bound.saveKeys(s.ffi, "GPG", output, flags)); err != nil {
		return nil, err
	}
	return takeOutput(output)
}

// LoadFrom reads pubring.gpg and secring.gpg from dir and nowhere else.
func (s *Session) LoadFrom(ctx context.Context, dir string) error {
	return s.Do(ctx, func(context.Context) error {
		return s.loadFrom(dir)
	})
}

func (s *Session) loadFrom(dir string) error {
	if !filepath.IsAbs(dir) {
		return ErrRelative
	}
	pub, err := readLimited(filepath.Join(dir, "pubring.gpg"))
	if err != nil {
		return err
	}
	defer wipe(pub)
	sec, err := readLimited(filepath.Join(dir, "secring.gpg"))
	if err != nil {
		return err
	}
	defer wipe(sec)
	if err := s.importKeys(pub, flagPublic); err != nil {
		return err
	}
	return s.importKeys(sec, flagSecret)
}

func (s *Session) importKeys(b []byte, flags uint32) error {
	in, err := memInput(b)
	if err != nil {
		return err
	}
	defer bound.inDestroy(in)
	return bound.check("load_keys", bound.loadKeys(s.ffi, "GPG", in, flags))
}

func (s *Session) Counts(ctx context.Context) (public, secret int, err error) {
	err = s.Do(ctx, func(context.Context) error {
		var n uintptr
		if err := bound.check("count", bound.pubCount(s.ffi, &n)); err != nil {
			return err
		}
		public = int(n)
		if err := bound.check("count", bound.secCount(s.ffi, &n)); err != nil {
			return err
		}
		secret = int(n)
		return nil
	})
	return public, secret, err
}

func memInput(b []byte) (uintptr, error) {
	if len(b) == 0 {
		return 0, ErrEmpty
	}
	var in uintptr
	err := bound.check("input", bound.inMem(&in, &b[0], uintptr(len(b)), true))
	runtime.KeepAlive(b)
	return in, err
}

func takeOutput(output uintptr) ([]byte, error) {
	var p, n uintptr
	if err := bound.check("output", bound.outGet(output, &p, &n, true)); err != nil {
		return nil, err
	}
	if p == 0 || n == 0 {
		return nil, ErrEmpty
	}
	if n > MaxOutput {
		bound.bufClear(p, n)
		bound.bufDestroy(p)
		return nil, ErrTooLarge
	}
	b := make([]byte, n)
	copy(b, unsafe.Slice((*byte)(cPtr(p)), n))
	bound.bufClear(p, n)
	bound.bufDestroy(p)
	return b, nil
}

func errorsNew(text string) error { return &plainError{text} }

type plainError struct{ s string }

func (e *plainError) Error() string { return e.s }
