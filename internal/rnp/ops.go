package rnp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unsafe"
)

const (
	flagPublic uint32 = 1 << 0
	flagSecret uint32 = 1 << 1

	keyExportArmored uint32 = 1 << 0
	keyExportPublic  uint32 = 1 << 1
	keyExportSecret  uint32 = 1 << 2
	keyExportSubkeys uint32 = 1 << 3

	keyRemovePublic  uint32 = 1 << 0
	keyRemoveSecret  uint32 = 1 << 1
	keyRemoveSubkeys uint32 = 1 << 2
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
	vr, err := s.verifyResultFromOp(op)
	if err != nil {
		return res, err
	}
	res = vr
	if exec != 0 {
		return res, bound.check("verify", exec)
	}
	if !res.MathValid {
		if res.Status != 0 {
			return res, bound.check("verify", res.Status)
		}
		return res, errorsNew("no valid signature")
	}
	return res, nil
}

func (s *Session) ListKeys(ctx context.Context) ([]KeySummary, error) {
	out := make([]KeySummary, 0, 8)
	err := s.Do(ctx, func(context.Context) error {
		var it uintptr
		if err := bound.check("identifier_iterator_create", bound.iterCreate(s.ffi, &it, "fingerprint")); err != nil {
			return err
		}
		defer bound.iterDestroy(it)

		seen := map[string]bool{}
		for {
			var id uintptr
			if err := bound.check("identifier_iterator_next", bound.iterNext(it, &id)); err != nil {
				return err
			}
			if id == 0 {
				break
			}
			fp := strings.TrimSpace(readCString(id, 160))
			if fp == "" || seen[fp] {
				continue
			}
			seen[fp] = true
			key, err := s.locateKeyHandle("fingerprint", fp)
			if err != nil {
				return err
			}
			sum, err := s.keySummaryFromHandle(key, true)
			bound.keyDestroy(key)
			if err != nil {
				return err
			}
			out = append(out, sum)
		}
		return nil
	})
	return out, err
}

func (s *Session) GenerateKeyWithSpec(ctx context.Context, req KeyGenRequest) (KeySummary, error) {
	var out KeySummary
	err := s.Do(ctx, func(context.Context) error {
		primaryAlg := defaultAlg(req.PrimaryAlg, "EDDSA")
		hashAlg := defaultAlg(req.HashAlg, "SHA256")
		symm := defaultAlg(req.SymmCipher, "AES256")
		uid := strings.TrimSpace(req.UserID)
		if uid == "" {
			uid = "rnp-armor-ephemeral"
		}

		protect := map[string]any{
			"cipher": symm,
			"hash":   hashAlg,
			"mode":   "CFB",
		}
		if req.S2KIterations > 0 {
			protect["iterations"] = req.S2KIterations
		}

		primary := map[string]any{
			"type":       primaryAlg,
			"userid":     uid,
			"usage":      []string{"sign", "certify"},
			"hash":       hashAlg,
			"protection": protect,
		}
		if req.PrimaryBits > 0 && (primaryAlg == "RSA" || primaryAlg == "DSA" || primaryAlg == "ELGAMAL") {
			primary["length"] = req.PrimaryBits
		}
		if primaryAlg == "ECDSA" || primaryAlg == "ECDH" || primaryAlg == "SM2" {
			if c := normalizeCurve(req.PrimaryCurve); c != "" {
				primary["curve"] = c
			}
		}
		if req.ExpirationSec > 0 {
			primary["expiration"] = req.ExpirationSec
		}

		subAlg := defaultAlg(req.SubkeyAlg, "")
		if subAlg == "" {
			switch primaryAlg {
			case "RSA":
				subAlg = "RSA"
			case "ECDSA":
				subAlg = "ECDH"
			case "DSA":
				subAlg = "ELGAMAL"
			default:
				subAlg = "ECDH"
			}
		}
		sub := map[string]any{
			"type":       subAlg,
			"usage":      []string{"encrypt"},
			"protection": protect,
		}
		if req.SubkeyBits > 0 && (subAlg == "RSA" || subAlg == "DSA" || subAlg == "ELGAMAL") {
			sub["length"] = req.SubkeyBits
		}
		if subAlg == "ECDH" || subAlg == "ECDSA" || subAlg == "SM2" {
			if c := normalizeCurve(req.SubkeyCurve); c != "" {
				sub["curve"] = c
			} else if subAlg == "ECDH" {
				sub["curve"] = "Curve25519"
			}
		}
		if req.ExpirationSec > 0 {
			sub["expiration"] = req.ExpirationSec
		}

		desc := map[string]any{"primary": primary, "sub": sub}
		raw, jerr := json.Marshal(desc)
		if jerr != nil {
			return jerr
		}

		var resPtr uintptr
		if err := bound.check("generate_key_json", bound.genJSON(s.ffi, string(raw), &resPtr)); err != nil {
			return err
		}
		resJSON := takeAllocatedCString(resPtr)

		var gen struct {
			Primary struct {
				Grip string `json:"grip"`
			} `json:"primary"`
			Sub struct {
				Grip string `json:"grip"`
			} `json:"sub"`
		}
		if err := json.Unmarshal([]byte(resJSON), &gen); err != nil {
			return err
		}
		if gen.Primary.Grip == "" {
			return errorsNew("generate_key_json returned no primary grip")
		}

		primaryKey, err := s.locateKeyHandle("grip", gen.Primary.Grip)
		if err != nil {
			return err
		}
		encKey := uintptr(0)
		if strings.TrimSpace(gen.Sub.Grip) != "" {
			encKey, err = s.locateKeyHandle("grip", gen.Sub.Grip)
			if err != nil {
				bound.keyDestroy(primaryKey)
				return err
			}
		}

		summary, err := s.keySummaryFromHandle(primaryKey, true)
		if err != nil {
			bound.keyDestroy(primaryKey)
			if encKey != 0 {
				bound.keyDestroy(encKey)
			}
			return err
		}

		if s.primary != 0 {
			bound.keyDestroy(s.primary)
			s.primary = 0
		}
		if s.encKey != 0 {
			bound.keyDestroy(s.encKey)
			s.encKey = 0
		}
		s.primary = primaryKey
		s.encKey = encKey
		out = summary
		return nil
	})
	return out, err
}

func (s *Session) ImportKeys(ctx context.Context, data []byte) (string, error) {
	var out string
	err := s.Do(ctx, func(context.Context) error {
		in, err := memInput(data)
		if err != nil {
			return err
		}
		defer bound.inDestroy(in)
		var p uintptr
		if err := bound.check("import_keys", bound.importKeys(s.ffi, in, flagPublic|flagSecret, &p)); err != nil {
			return err
		}
		out = takeAllocatedCString(p)
		return nil
	})
	return out, err
}

func (s *Session) ExportKey(ctx context.Context, fprint string, secret bool) (string, error) {
	var out string
	err := s.Do(ctx, func(context.Context) error {
		key, err := s.locateKeyHandle("fingerprint", strings.TrimSpace(fprint))
		if err != nil {
			return err
		}
		defer bound.keyDestroy(key)

		var output uintptr
		if err := bound.check("output", bound.outMem(&output, MaxOutput)); err != nil {
			return err
		}
		defer bound.outDestroy(output)

		flags := keyExportArmored | keyExportSubkeys | keyExportPublic
		if secret {
			flags = keyExportArmored | keyExportSubkeys | keyExportSecret
		}
		if err := bound.check("key_export", bound.keyExport(key, output, flags)); err != nil {
			return err
		}
		b, err := takeOutput(output)
		if err != nil {
			return err
		}
		out = string(b)
		wipe(b)
		return nil
	})
	return out, err
}

func (s *Session) ExportRevocation(ctx context.Context, fprint, code, reason string) (string, error) {
	var out string
	err := s.Do(ctx, func(context.Context) error {
		key, err := s.locateKeyHandle("fingerprint", strings.TrimSpace(fprint))
		if err != nil {
			return err
		}
		defer bound.keyDestroy(key)

		var output uintptr
		if err := bound.check("output", bound.outMem(&output, MaxOutput)); err != nil {
			return err
		}
		defer bound.outDestroy(output)

		if strings.TrimSpace(code) == "" {
			code = "no"
		}
		if err := bound.check("key_export_revocation", bound.keyExportRev(key, output, keyExportArmored, "SHA256", code, reason)); err != nil {
			return err
		}
		b, err := takeOutput(output)
		if err != nil {
			return err
		}
		out = string(b)
		wipe(b)
		return nil
	})
	return out, err
}

func (s *Session) DeleteKey(ctx context.Context, fprint string) error {
	return s.Do(ctx, func(context.Context) error {
		key, err := s.locateKeyHandle("fingerprint", strings.TrimSpace(fprint))
		if err != nil {
			return err
		}
		defer bound.keyDestroy(key)
		flags := keyRemovePublic
		var hasSecret bool
		if bound.keySecret(key, &hasSecret) == 0 && hasSecret {
			flags |= keyRemoveSecret
		}
		rc := bound.keyRemove(key, flags|keyRemoveSubkeys)
		if rc != 0 {
			rc = bound.keyRemove(key, flags)
		}
		return bound.check("key_remove", rc)
	})
}

func (s *Session) KeyToJSON(ctx context.Context, fprint string) (string, error) {
	var out string
	err := s.Do(ctx, func(context.Context) error {
		key, err := s.locateKeyHandle("fingerprint", strings.TrimSpace(fprint))
		if err != nil {
			return err
		}
		defer bound.keyDestroy(key)
		var p uintptr
		if err := bound.check("key_to_json", bound.keyToJSON(key, 0, &p)); err != nil {
			return err
		}
		out = takeAllocatedCString(p)
		return nil
	})
	return out, err
}

func (s *Session) EncryptText(ctx context.Context, plain string, opts EncryptOptions) (string, error) {
	var out string
	err := s.Do(ctx, func(context.Context) error {
		b, err := s.encryptWithOpts([]byte(plain), opts, true, "")
		if err != nil {
			return err
		}
		out = string(b)
		wipe(b)
		return nil
	})
	return out, err
}

func (s *Session) DecryptText(ctx context.Context, armored string) (DecryptVerifyResult, error) {
	var out DecryptVerifyResult
	err := s.Do(ctx, func(context.Context) error {
		in, err := memInput([]byte(armored))
		if err != nil {
			return err
		}
		defer bound.inDestroy(in)

		var output uintptr
		if err := bound.check("output", bound.outMem(&output, MaxOutput)); err != nil {
			return err
		}
		defer bound.outDestroy(output)

		var op uintptr
		if err := bound.check("verify", bound.verCreate(&op, s.ffi, in, output)); err != nil {
			return err
		}
		defer bound.verDestroy(op)

		exec := bound.verExec(op)
		vr, ferr := s.verifyResultFromOp(op)
		if ferr != nil {
			return ferr
		}
		if exec != 0 {
			out.Verify = vr
			return bound.check("verify", exec)
		}
		plainBytes, err := takeOutput(output)
		if err != nil {
			return err
		}
		out.Plaintext = string(plainBytes)
		wipe(plainBytes)
		out.Verify = vr
		if vr.Status != 0 {
			return bound.check("verify", vr.Status)
		}
		return nil
	})
	return out, err
}

func (s *Session) SignText(ctx context.Context, plain string, signerFprint string, mode SignMode, hashAlg string) (string, error) {
	var out string
	err := s.Do(ctx, func(context.Context) error {
		b, err := s.signWithMode([]byte(plain), signerFprint, mode, hashAlg, true)
		if err != nil {
			return err
		}
		out = string(b)
		wipe(b)
		return nil
	})
	return out, err
}

func (s *Session) VerifyText(ctx context.Context, input string, detachedSig string) (VerifyResult, error) {
	var out VerifyResult
	err := s.Do(ctx, func(context.Context) error {
		if strings.TrimSpace(detachedSig) != "" {
			vr, err := s.verifyDetachedBytes([]byte(input), []byte(detachedSig))
			out = vr
			if err != nil {
				return err
			}
			if !out.MathValid {
				if out.Status != 0 {
					return bound.check("verify", out.Status)
				}
				return errorsNew("no valid signature")
			}
			return nil
		}

		in, err := memInput([]byte(input))
		if err != nil {
			return err
		}
		defer bound.inDestroy(in)
		var output uintptr
		if err := bound.check("output", bound.outMem(&output, MaxOutput)); err != nil {
			return err
		}
		defer bound.outDestroy(output)

		var op uintptr
		if err := bound.check("verify", bound.verCreate(&op, s.ffi, in, output)); err != nil {
			return err
		}
		defer bound.verDestroy(op)
		exec := bound.verExec(op)
		vr, ferr := s.verifyResultFromOp(op)
		if ferr != nil {
			return ferr
		}
		out = vr
		if exec != 0 {
			return bound.check("verify", exec)
		}
		if !out.MathValid {
			if out.Status != 0 {
				return bound.check("verify", out.Status)
			}
			return errorsNew("no valid signature")
		}
		return nil
	})
	return out, err
}

func (s *Session) EncryptFileWithOpts(ctx context.Context, src, dst string, opts EncryptOptions) error {
	return s.Do(ctx, func(ctx context.Context) error {
		if src == dst {
			return errorsNew("source and destination are the same")
		}
		plain, err := readLimited(src)
		if err != nil {
			return err
		}
		defer wipe(plain)
		out, err := s.encryptWithOpts(plain, opts, opts.Armor, filepath.Base(src))
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
	})
}

func (s *Session) SignFileWithOpts(ctx context.Context, src, dst string, signerFprint string, mode SignMode, armored bool, hashAlg string) error {
	return s.Do(ctx, func(ctx context.Context) error {
		if src == dst {
			return errorsNew("source and destination are the same")
		}
		data, err := readLimited(src)
		if err != nil {
			return err
		}
		defer wipe(data)
		sig, err := s.signWithMode(data, signerFprint, mode, hashAlg, armored)
		if err != nil {
			wipe(sig)
			return err
		}
		if err := ctx.Err(); err != nil {
			wipe(sig)
			return err
		}
		err = commitNew(dst, sig, 0o600)
		wipe(sig)
		return err
	})
}

func (s *Session) VerifyFileWithOpts(ctx context.Context, src, sigOrOutPath string, detached bool) (VerifyResult, error) {
	var out VerifyResult
	err := s.Do(ctx, func(context.Context) error {
		data, err := readLimited(src)
		if err != nil {
			return err
		}
		defer wipe(data)
		if detached {
			sig, err := readLimited(sigOrOutPath)
			if err != nil {
				return err
			}
			defer wipe(sig)
			vr, err := s.verifyDetachedBytes(data, sig)
			out = vr
			return err
		}
		in, err := memInput(data)
		if err != nil {
			return err
		}
		defer bound.inDestroy(in)
		var output uintptr
		if err := bound.check("output", bound.outMem(&output, MaxOutput)); err != nil {
			return err
		}
		defer bound.outDestroy(output)
		var op uintptr
		if err := bound.check("verify", bound.verCreate(&op, s.ffi, in, output)); err != nil {
			return err
		}
		defer bound.verDestroy(op)
		exec := bound.verExec(op)
		vr, err := s.verifyResultFromOp(op)
		if err != nil {
			return err
		}
		out = vr
		if exec != 0 {
			return bound.check("verify", exec)
		}
		if !out.MathValid {
			if out.Status != 0 {
				return bound.check("verify", out.Status)
			}
			return errorsNew("no valid signature")
		}
		return nil
	})
	return out, err
}

func (s *Session) DumpPackets(ctx context.Context, data []byte, asJSON bool) (string, error) {
	var out string
	err := s.Do(ctx, func(context.Context) error {
		in, err := memInput(data)
		if err != nil {
			return err
		}
		defer bound.inDestroy(in)
		if asJSON {
			var p uintptr
			if err := bound.check("dump_packets_to_json", bound.dumpToJSON(in, 0, &p)); err != nil {
				return err
			}
			out = takeAllocatedCString(p)
			return nil
		}
		var output uintptr
		if err := bound.check("output", bound.outMem(&output, MaxOutput)); err != nil {
			return err
		}
		defer bound.outDestroy(output)
		if err := bound.check("dump_packets_to_output", bound.dumpToOutput(in, output, 0)); err != nil {
			return err
		}
		b, err := takeOutput(output)
		if err != nil {
			return err
		}
		out = string(b)
		wipe(b)
		return nil
	})
	return out, err
}

func (s *Session) SupportedFeatures(ctx context.Context, category string) (string, error) {
	var out string
	err := s.Do(ctx, func(context.Context) error {
		cat := strings.TrimSpace(category)
		if strings.Contains(cat, ":") {
			parts := strings.SplitN(cat, ":", 2)
			var ok bool
			if err := bound.check("supports_feature", bound.supports(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), &ok)); err != nil {
				return err
			}
			m := map[string]any{"type": strings.TrimSpace(parts[0]), "name": strings.TrimSpace(parts[1]), "supported": ok}
			raw, err := json.Marshal(m)
			if err != nil {
				return err
			}
			out = string(raw)
			return nil
		}

		if cat != "" {
			var p uintptr
			if err := bound.check("supported_features", bound.features(cat, &p)); err != nil {
				return err
			}
			out = takeAllocatedCString(p)
			return nil
		}

		cats := []string{
			"symmetric algorithm",
			"hash algorithm",
			"public key algorithm",
			"elliptic curve",
			"compression algorithm",
			"aead algorithm",
		}
		all := map[string]any{}
		for _, c := range cats {
			var p uintptr
			if err := bound.check("supported_features", bound.features(c, &p)); err != nil {
				return err
			}
			arrJSON := takeAllocatedCString(p)
			var arr []string
			if uerr := json.Unmarshal([]byte(arrJSON), &arr); uerr != nil {
				all[c] = arrJSON
			} else {
				all[c] = arr
			}
		}
		raw, err := json.Marshal(all)
		if err != nil {
			return err
		}
		out = string(raw)
		return nil
	})
	return out, err
}

func (s *Session) locateKeyHandle(typ, id string) (uintptr, error) {
	var key uintptr
	if err := bound.check("locate_key", bound.locateKey(s.ffi, typ, id, &key)); err != nil {
		return 0, err
	}
	if key == 0 {
		return 0, fmt.Errorf("key not found: %s=%s", typ, id)
	}
	return key, nil
}

func (s *Session) keySummaryFromHandle(key uintptr, includeSubkeys bool) (KeySummary, error) {
	var out KeySummary
	var p uintptr
	if err := bound.check("key_get_fprint", bound.keyFprint(key, &p)); err != nil {
		return out, err
	}
	out.Fingerprint = takeAllocatedCString(p)

	if err := bound.check("key_get_keyid", bound.keyKeyID(key, &p)); err != nil {
		return out, err
	}
	out.KeyID = takeAllocatedCString(p)

	if bound.keyPrimaryID(key, &p) == 0 && p != 0 {
		out.PrimaryUID = takeAllocatedCString(p)
	}

	if err := bound.check("key_get_alg", bound.keyAlg(key, &p)); err != nil {
		return out, err
	}
	out.Algorithm = takeAllocatedCString(p)

	if err := bound.check("key_get_creation", bound.keyCreation(key, &out.CreatedUnix)); err != nil {
		return out, err
	}
	var exp uint32
	if err := bound.check("key_get_expiration", bound.keyExpiry(key, &exp)); err != nil {
		return out, err
	}
	if exp == 0 {
		out.ExpiresUnix = 0
	} else {
		out.ExpiresUnix = out.CreatedUnix + exp
	}

	if err := bound.check("key_have_secret", bound.keySecret(key, &out.HasSecret)); err != nil {
		return out, err
	}
	if out.HasSecret {
		if err := bound.check("key_is_locked", bound.keyLocked(key, &out.IsLocked)); err != nil {
			return out, err
		}
	}
	if err := bound.check("key_is_revoked", bound.keyRevoked(key, &out.IsRevoked)); err != nil {
		return out, err
	}
	if err := bound.check("key_allows_usage", bound.keyUsage(key, "sign", &out.CanSign)); err != nil {
		return out, err
	}
	if err := bound.check("key_allows_usage", bound.keyUsage(key, "encrypt", &out.CanEncrypt)); err != nil {
		return out, err
	}
	if err := bound.check("key_allows_usage", bound.keyUsage(key, "certify", &out.CanCertify)); err != nil {
		return out, err
	}
	if err := bound.check("key_is_primary", bound.keyIsPrimary(key, &out.IsPrimary)); err != nil {
		return out, err
	}
	if err := bound.check("key_is_sub", bound.keyIsSub(key, &out.IsSub)); err != nil {
		return out, err
	}
	if includeSubkeys && out.IsPrimary {
		var n uintptr
		if err := bound.check("key_get_subkey_count", bound.keySubCount(key, &n)); err != nil {
			return out, err
		}
		out.Subkeys = make([]KeySummary, 0, n)
		for i := uintptr(0); i < n; i++ {
			var sub uintptr
			if err := bound.check("key_get_subkey_at", bound.keySubAt(key, i, &sub)); err != nil {
				if sub != 0 {
					bound.keyDestroy(sub)
				}
				return out, err
			}
			sum, err := s.keySummaryFromHandle(sub, false)
			bound.keyDestroy(sub)
			if err != nil {
				return out, err
			}
			out.Subkeys = append(out.Subkeys, sum)
		}
	}
	return out, nil
}

func (s *Session) encryptWithOpts(plain []byte, opts EncryptOptions, armor bool, fileName string) ([]byte, error) {
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

	hasRecipient := false
	if len(opts.RecipientFprints) == 0 && strings.TrimSpace(opts.Password) == "" && s.encKey != 0 {
		if err := bound.check("encrypt", bound.encRecipient(op, s.encKey)); err != nil {
			return nil, err
		}
		hasRecipient = true
	}
	for _, fp := range opts.RecipientFprints {
		fp = strings.TrimSpace(fp)
		if fp == "" {
			continue
		}
		key, err := s.locateKeyHandle("fingerprint", fp)
		if err != nil {
			return nil, err
		}
		err = bound.check("encrypt", bound.encRecipient(op, key))
		bound.keyDestroy(key)
		if err != nil {
			return nil, err
		}
		hasRecipient = true
	}

	if strings.TrimSpace(opts.Password) != "" {
		cipher := defaultAlg(opts.Cipher, "AES256")
		if err := bound.check("encrypt", bound.encPassword(op, opts.Password, "SHA256", 0, cipher)); err != nil {
			return nil, err
		}
	}
	if !hasRecipient && strings.TrimSpace(opts.Password) == "" {
		return nil, ErrNoKey
	}

	if signer := strings.TrimSpace(opts.SignerFprint); signer != "" {
		key, err := s.locateKeyHandle("fingerprint", signer)
		if err != nil {
			return nil, err
		}
		var sig uintptr
		err = bound.check("encrypt", bound.encSign(op, key, &sig))
		bound.keyDestroy(key)
		if err != nil {
			return nil, err
		}
	}

	if err := bound.check("encrypt", bound.encCipher(op, defaultAlg(opts.Cipher, "AES256"))); err != nil {
		return nil, err
	}
	if comp := normalizeCompression(opts.Compression); comp != "" {
		level := int32(opts.CompressionLevel)
		if level < 0 || level > 9 {
			level = 6
		}
		if err := bound.check("encrypt", bound.encCompress(op, comp, level)); err != nil {
			return nil, err
		}
	}
	if err := bound.check("encrypt", bound.encArmor(op, armor)); err != nil {
		return nil, err
	}
	if err := bound.check("encrypt", bound.encFileName(op, fileName)); err != nil {
		return nil, err
	}
	if err := bound.check("encrypt", bound.encExec(op)); err != nil {
		return nil, err
	}
	return takeOutput(output)
}

func (s *Session) signWithMode(data []byte, signerFprint string, mode SignMode, hashAlg string, armored bool) ([]byte, error) {
	if mode == "" {
		mode = SignModeDetached
	}

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
	switch mode {
	case SignModeInline:
		err = bound.check("sign", bound.signCreate(&op, s.ffi, in, output))
	case SignModeCleartext:
		err = bound.check("sign", bound.signClear(&op, s.ffi, in, output))
	default:
		err = bound.check("sign", bound.signDetached(&op, s.ffi, in, output))
	}
	if err != nil {
		return nil, err
	}
	defer bound.signDestroy(op)

	signer := s.primary
	if fp := strings.TrimSpace(signerFprint); fp != "" {
		key, err := s.locateKeyHandle("fingerprint", fp)
		if err != nil {
			return nil, err
		}
		defer bound.keyDestroy(key)
		signer = key
	}
	if signer == 0 {
		return nil, ErrNoKey
	}

	if err := bound.check("sign", bound.signAdd(op, signer, new(uintptr))); err != nil {
		return nil, err
	}
	if err := bound.check("sign", bound.signHash(op, defaultAlg(hashAlg, "SHA256"))); err != nil {
		return nil, err
	}
	if mode != SignModeCleartext {
		if err := bound.check("sign", bound.signArmor(op, armored)); err != nil {
			return nil, err
		}
	}
	if err := bound.check("sign", bound.signExec(op)); err != nil {
		return nil, err
	}
	return takeOutput(output)
}

func (s *Session) verifyDetachedBytes(data, sig []byte) (VerifyResult, error) {
	out := VerifyResult{}
	in, err := memInput(data)
	if err != nil {
		return out, err
	}
	defer bound.inDestroy(in)
	sigIn, err := memInput(sig)
	if err != nil {
		return out, err
	}
	defer bound.inDestroy(sigIn)

	var op uintptr
	if err := bound.check("verify", bound.verDetached(&op, s.ffi, in, sigIn)); err != nil {
		return out, err
	}
	defer bound.verDestroy(op)
	exec := bound.verExec(op)
	vr, err := s.verifyResultFromOp(op)
	out = vr
	if err != nil {
		return out, err
	}
	if exec != 0 {
		return out, bound.check("verify", exec)
	}
	if !out.MathValid {
		if out.Status != 0 {
			return out, bound.check("verify", out.Status)
		}
		return out, errorsNew("no valid signature")
	}
	return out, nil
}

func (s *Session) verifyResultFromOp(op uintptr) (VerifyResult, error) {
	res := VerifyResult{IdentityTrusted: false}
	var n uintptr
	if err := bound.check("verify", bound.verCount(op, &n)); err != nil {
		return res, err
	}
	if n == 0 {
		res.MathValid = false
		return res, nil
	}

	var sig uintptr
	if err := bound.check("verify", bound.verAt(op, 0, &sig)); err != nil {
		return res, err
	}
	status := bound.verStatus(sig)
	res.Status = status
	if status != 0 && bound.resultString != nil {
		res.StatusText = bound.resultString(status)
	}
	res.MathValid = status == 0

	var p uintptr
	if bound.verSigHash != nil && bound.verSigHash(sig, &p) == 0 {
		res.HashAlg = takeAllocatedCString(p)
	}
	if bound.verSigTimes != nil {
		_ = bound.verSigTimes(sig, &res.CreatedUnix, &res.ExpiresUnix)
	}
	if bound.verSigKey != nil {
		var key uintptr
		if bound.verSigKey(sig, &key) == 0 && key != 0 {
			if bound.keyKeyID(key, &p) == 0 {
				res.SignerKeyID = takeAllocatedCString(p)
			}
			if bound.keyFprint(key, &p) == 0 {
				res.SignerFingerprint = takeAllocatedCString(p)
			}
			bound.keyDestroy(key)
		}
	}
	return res, nil
}

func takeAllocatedCString(p uintptr) string {
	if p == 0 {
		return ""
	}
	s := readCString(p, int(MaxOutput))
	if n := uintptr(len(s)); n > 0 {
		bound.bufClear(p, n)
	}
	bound.bufDestroy(p)
	return s
}

func normalizeCurve(in string) string {
	v := strings.TrimSpace(strings.ToLower(in))
	switch v {
	case "", "default":
		return ""
	case "curve25519", "x25519":
		return "Curve25519"
	case "p-256", "nist p-256", "secp256r1":
		return "NIST P-256"
	case "p-384", "nist p-384", "secp384r1":
		return "NIST P-384"
	case "p-521", "nist p-521", "secp521r1":
		return "NIST P-521"
	case "secp256k1":
		return "secp256k1"
	case "ed25519":
		return "Ed25519"
	default:
		return strings.TrimSpace(in)
	}
}

func normalizeCompression(in string) string {
	v := strings.TrimSpace(strings.ToLower(in))
	switch v {
	case "", "none":
		return ""
	case "uncompressed":
		return "Uncompressed"
	case "zip":
		return "ZIP"
	case "zlib":
		return "ZLIB"
	case "bzip2", "bzip":
		return "BZip2"
	default:
		return in
	}
}

// SaveTo writes the keyring files that this session actually holds.
// pubring.gpg is written only when there is a public key, and secring.gpg
// only when there is a secret key. A public-only keyring therefore has no
// secring.gpg. Both counts empty is ErrEmpty. A file that would be written
// must not already exist. The directory must be absolute.
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
	var pubN, secN uintptr
	if err := bound.check("count", bound.pubCount(s.ffi, &pubN)); err != nil {
		return err
	}
	if err := bound.check("count", bound.secCount(s.ffi, &secN)); err != nil {
		return err
	}
	if pubN == 0 && secN == 0 {
		return ErrEmpty
	}
	pubPath := filepath.Join(dir, "pubring.gpg")
	secPath := filepath.Join(dir, "secring.gpg")
	if pubN > 0 {
		if err := refuseExisting(pubPath); err != nil {
			return err
		}
	}
	if secN > 0 {
		if err := refuseExisting(secPath); err != nil {
			return err
		}
	}
	var pub, sec []byte
	if pubN > 0 {
		pub, err = s.exportKeys(flagPublic)
		if err != nil {
			return err
		}
	}
	if secN > 0 {
		sec, err = s.exportKeys(flagSecret)
		if err != nil {
			wipe(pub)
			return err
		}
	}
	if pubN > 0 {
		if err := commitNew(pubPath, pub, 0o644); err != nil {
			wipe(pub)
			wipe(sec)
			return err
		}
		wipe(pub)
	}
	if secN > 0 {
		if err := commitNew(secPath, sec, 0o600); err != nil {
			if pubN > 0 {
				os.Remove(pubPath)
			}
			wipe(sec)
			return err
		}
		wipe(sec)
	}
	return nil
}

func refuseExisting(path string) error {
	_, err := os.Lstat(path)
	if err == nil {
		return ErrExists
	}
	if os.IsNotExist(err) {
		return nil
	}
	return err
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

// LoadFrom reads whichever of pubring.gpg and secring.gpg exist in dir.
// A public-only directory loads the public ring alone. When neither file
// exists the error is the missing-file error from the public path. The
// directory must be absolute.
func (s *Session) LoadFrom(ctx context.Context, dir string) error {
	return s.Do(ctx, func(context.Context) error {
		return s.loadFrom(dir)
	})
}

func (s *Session) loadFrom(dir string) error {
	if !filepath.IsAbs(dir) {
		return ErrRelative
	}
	pubPath := filepath.Join(dir, "pubring.gpg")
	secPath := filepath.Join(dir, "secring.gpg")
	_, pubErr := os.Lstat(pubPath)
	_, secErr := os.Lstat(secPath)
	pubExists := pubErr == nil
	secExists := secErr == nil
	if !pubExists && !secExists {
		if pubErr != nil {
			return pubErr
		}
		return secErr
	}
	if pubErr != nil && !os.IsNotExist(pubErr) {
		return pubErr
	}
	if secErr != nil && !os.IsNotExist(secErr) {
		return secErr
	}
	if pubExists {
		pub, err := readLimited(pubPath)
		if err != nil {
			return err
		}
		defer wipe(pub)
		if err := s.importKeys(pub, flagPublic); err != nil {
			return err
		}
	}
	if secExists {
		sec, err := readLimited(secPath)
		if err != nil {
			return err
		}
		defer wipe(sec)
		if err := s.importKeys(sec, flagSecret); err != nil {
			return err
		}
	}
	return nil
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
