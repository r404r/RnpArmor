package rnp

import (
	"fmt"
	"sync"

	"github.com/ebitengine/purego"
)

// CodeError is a librnp status. Text comes from rnp_result_to_string and
// does not include key material or a passphrase.
type CodeError struct {
	Op   string
	Code uint32
	Text string
}

func (e *CodeError) Error() string {
	return fmt.Sprintf("%s: rnp %s (%d)", e.Op, e.Text, e.Code)
}

type api struct {
	version      func() string
	versionFull  func() string
	resultString func(code uint32) string
	ffiCreate    func(out *uintptr, pub, sec string) uint32
	ffiDestroy   func(ffi uintptr) uint32
	setPass      func(ffi, cb, ctx uintptr) uint32
	requestPass  func(ffi, key uintptr, context string, out *uintptr) uint32
	bufDestroy   func(p uintptr)
	bufClear     func(p, n uintptr)
	calcIter     func(hash string, msec uintptr, out *uintptr) uint32
	genCreate    func(op *uintptr, ffi uintptr, alg string) uint32
	subCreate    func(op *uintptr, ffi, primary uintptr, alg string) uint32
	genUser      func(op uintptr, userid string) uint32
	genUsage     func(op uintptr, usage string) uint32
	genHash      func(op uintptr, hash string) uint32
	genCurve     func(op uintptr, curve string) uint32
	genReqPass   func(op uintptr, request bool) uint32
	genCipher    func(op uintptr, cipher string) uint32
	genProtHash  func(op uintptr, hash string) uint32
	genMode      func(op uintptr, mode string) uint32
	genIter      func(op uintptr, iterations uint32) uint32
	genExec      func(op uintptr) uint32
	genKey       func(op uintptr, key *uintptr) uint32
	genDestroy   func(op uintptr) uint32
	keyDestroy   func(key uintptr) uint32
	pubCount     func(ffi uintptr, n *uintptr) uint32
	secCount     func(ffi uintptr, n *uintptr) uint32
	inMem        func(out *uintptr, buf *byte, n uintptr, copy bool) uint32
	inDestroy    func(in uintptr) uint32
	outMem       func(out *uintptr, max uintptr) uint32
	outGet       func(out uintptr, buf *uintptr, n *uintptr, copy bool) uint32
	outDestroy   func(out uintptr) uint32
	encCreate    func(op *uintptr, ffi, in, out uintptr) uint32
	encRecipient func(op, key uintptr) uint32
	encCipher    func(op uintptr, cipher string) uint32
	encArmor     func(op uintptr, armored bool) uint32
	encFileName  func(op uintptr, name string) uint32
	encExec      func(op uintptr) uint32
	encDestroy   func(op uintptr) uint32
	decrypt      func(ffi, in, out uintptr) uint32
	signDetached func(op *uintptr, ffi, in, sig uintptr) uint32
	signAdd      func(op, key uintptr, sig *uintptr) uint32
	signHash     func(op uintptr, hash string) uint32
	signArmor    func(op uintptr, armored bool) uint32
	signExec     func(op uintptr) uint32
	signDestroy  func(op uintptr) uint32
	verDetached  func(op *uintptr, ffi, in, sig uintptr) uint32
	verExec      func(op uintptr) uint32
	verCount     func(op uintptr, n *uintptr) uint32
	verAt        func(op uintptr, idx uintptr, sig *uintptr) uint32
	verStatus    func(sig uintptr) uint32
	verDestroy   func(op uintptr) uint32
	saveKeys     func(ffi uintptr, format string, out uintptr, flags uint32) uint32
	loadKeys     func(ffi uintptr, format string, in uintptr, flags uint32) uint32
}

var (
	bindOnce sync.Once
	bound    api
	bindErr  error
)

func bind(handle uintptr) error {
	bindOnce.Do(func() {
		defer func() {
			if r := recover(); r != nil {
				bindErr = fmt.Errorf("bind librnp: %v", r)
			}
		}()
		reg := func(fn any, name string) {
			purego.RegisterLibFunc(fn, handle, name)
		}
		reg(&bound.version, "rnp_version_string")
		reg(&bound.versionFull, "rnp_version_string_full")
		reg(&bound.resultString, "rnp_result_to_string")
		reg(&bound.ffiCreate, "rnp_ffi_create")
		reg(&bound.ffiDestroy, "rnp_ffi_destroy")
		reg(&bound.setPass, "rnp_ffi_set_pass_provider")
		reg(&bound.requestPass, "rnp_request_password")
		reg(&bound.bufDestroy, "rnp_buffer_destroy")
		reg(&bound.bufClear, "rnp_buffer_clear")
		reg(&bound.calcIter, "rnp_calculate_iterations")
		reg(&bound.genCreate, "rnp_op_generate_create")
		reg(&bound.subCreate, "rnp_op_generate_subkey_create")
		reg(&bound.genUser, "rnp_op_generate_set_userid")
		reg(&bound.genUsage, "rnp_op_generate_add_usage")
		reg(&bound.genHash, "rnp_op_generate_set_hash")
		reg(&bound.genCurve, "rnp_op_generate_set_curve")
		reg(&bound.genReqPass, "rnp_op_generate_set_request_password")
		reg(&bound.genCipher, "rnp_op_generate_set_protection_cipher")
		reg(&bound.genProtHash, "rnp_op_generate_set_protection_hash")
		reg(&bound.genMode, "rnp_op_generate_set_protection_mode")
		reg(&bound.genIter, "rnp_op_generate_set_protection_iterations")
		reg(&bound.genExec, "rnp_op_generate_execute")
		reg(&bound.genKey, "rnp_op_generate_get_key")
		reg(&bound.genDestroy, "rnp_op_generate_destroy")
		reg(&bound.keyDestroy, "rnp_key_handle_destroy")
		reg(&bound.pubCount, "rnp_get_public_key_count")
		reg(&bound.secCount, "rnp_get_secret_key_count")
		reg(&bound.inMem, "rnp_input_from_memory")
		reg(&bound.inDestroy, "rnp_input_destroy")
		reg(&bound.outMem, "rnp_output_to_memory")
		reg(&bound.outGet, "rnp_output_memory_get_buf")
		reg(&bound.outDestroy, "rnp_output_destroy")
		reg(&bound.encCreate, "rnp_op_encrypt_create")
		reg(&bound.encRecipient, "rnp_op_encrypt_add_recipient")
		reg(&bound.encCipher, "rnp_op_encrypt_set_cipher")
		reg(&bound.encArmor, "rnp_op_encrypt_set_armor")
		reg(&bound.encFileName, "rnp_op_encrypt_set_file_name")
		reg(&bound.encExec, "rnp_op_encrypt_execute")
		reg(&bound.encDestroy, "rnp_op_encrypt_destroy")
		reg(&bound.decrypt, "rnp_decrypt")
		reg(&bound.signDetached, "rnp_op_sign_detached_create")
		reg(&bound.signAdd, "rnp_op_sign_add_signature")
		reg(&bound.signHash, "rnp_op_sign_set_hash")
		reg(&bound.signArmor, "rnp_op_sign_set_armor")
		reg(&bound.signExec, "rnp_op_sign_execute")
		reg(&bound.signDestroy, "rnp_op_sign_destroy")
		reg(&bound.verDetached, "rnp_op_verify_detached_create")
		reg(&bound.verExec, "rnp_op_verify_execute")
		reg(&bound.verCount, "rnp_op_verify_get_signature_count")
		reg(&bound.verAt, "rnp_op_verify_get_signature_at")
		reg(&bound.verStatus, "rnp_op_verify_signature_get_status")
		reg(&bound.verDestroy, "rnp_op_verify_destroy")
		reg(&bound.saveKeys, "rnp_save_keys")
		reg(&bound.loadKeys, "rnp_load_keys")
		if passCB == 0 {
			// One process-lifetime trampoline. purego never releases it.
			// Sessions release their slot, passphrase, and ffi handle.
			passCB = purego.NewCallback(passCallback)
		}
	})
	return bindErr
}

func (a *api) check(op string, code uint32) error {
	if code == 0 {
		return nil
	}
	text := ""
	if a.resultString != nil {
		text = a.resultString(code)
	}
	return &CodeError{Op: op, Code: code, Text: text}
}
