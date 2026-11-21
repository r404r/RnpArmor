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
	verCreate    func(op *uintptr, ffi, in, out uintptr) uint32
	verSigHash   func(sig uintptr, hash *uintptr) uint32
	verSigKey    func(sig uintptr, key *uintptr) uint32
	verSigTimes  func(sig uintptr, created, expires *uint32) uint32
	verDestroy   func(op uintptr) uint32
	iterCreate   func(ffi uintptr, it *uintptr, typ string) uint32
	iterNext     func(it uintptr, id *uintptr) uint32
	iterDestroy  func(it uintptr) uint32
	locateKey    func(ffi uintptr, typ, id string, key *uintptr) uint32
	genJSON      func(ffi uintptr, json string, results *uintptr) uint32
	importKeys   func(ffi, in uintptr, flags uint32, results *uintptr) uint32
	keyExport    func(key, out uintptr, flags uint32) uint32
	keyExportRev func(key, out uintptr, flags uint32, hash, code, reason string) uint32
	keyRemove    func(key uintptr, flags uint32) uint32
	keyToJSON    func(key uintptr, flags uint32, result *uintptr) uint32
	keyFprint    func(key uintptr, fprint *uintptr) uint32
	keyKeyID     func(key uintptr, keyid *uintptr) uint32
	keyAlg       func(key uintptr, alg *uintptr) uint32
	keyPrimaryID func(key uintptr, uid *uintptr) uint32
	keyCreation  func(key uintptr, ts *uint32) uint32
	keyExpiry    func(key uintptr, sec *uint32) uint32
	keySetExpiry func(key uintptr, sec uint32) uint32
	keySecret    func(key uintptr, have *bool) uint32
	keyLocked    func(key uintptr, locked *bool) uint32
	keyRevoked   func(key uintptr, revoked *bool) uint32
	keyUsage     func(key uintptr, usage string, allowed *bool) uint32
	encPassword  func(op uintptr, pass, s2kHash string, iterations uintptr, s2kCipher string) uint32
	encSign      func(op, key uintptr, sig *uintptr) uint32
	encCompress  func(op uintptr, alg string, level int32) uint32
	signCreate   func(op *uintptr, ffi, in, out uintptr) uint32
	signClear    func(op *uintptr, ffi, in, out uintptr) uint32
	signCompress func(op uintptr, alg string, level int32) uint32
	dumpToOutput func(in, out uintptr, flags uint32) uint32
	dumpToJSON   func(in uintptr, flags uint32, result *uintptr) uint32
	features     func(typ string, result *uintptr) uint32
	supports     func(typ, name string, supported *bool) uint32
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
		reg(&bound.verCreate, "rnp_op_verify_create")
		reg(&bound.verExec, "rnp_op_verify_execute")
		reg(&bound.verCount, "rnp_op_verify_get_signature_count")
		reg(&bound.verAt, "rnp_op_verify_get_signature_at")
		reg(&bound.verStatus, "rnp_op_verify_signature_get_status")
		reg(&bound.verSigHash, "rnp_op_verify_signature_get_hash")
		reg(&bound.verSigKey, "rnp_op_verify_signature_get_key")
		reg(&bound.verSigTimes, "rnp_op_verify_signature_get_times")
		reg(&bound.verDestroy, "rnp_op_verify_destroy")
		reg(&bound.iterCreate, "rnp_identifier_iterator_create")
		reg(&bound.iterNext, "rnp_identifier_iterator_next")
		reg(&bound.iterDestroy, "rnp_identifier_iterator_destroy")
		reg(&bound.locateKey, "rnp_locate_key")
		reg(&bound.genJSON, "rnp_generate_key_json")
		reg(&bound.importKeys, "rnp_import_keys")
		reg(&bound.keyExport, "rnp_key_export")
		reg(&bound.keyExportRev, "rnp_key_export_revocation")
		reg(&bound.keyRemove, "rnp_key_remove")
		reg(&bound.keyToJSON, "rnp_key_to_json")
		reg(&bound.keyFprint, "rnp_key_get_fprint")
		reg(&bound.keyKeyID, "rnp_key_get_keyid")
		reg(&bound.keyAlg, "rnp_key_get_alg")
		reg(&bound.keyPrimaryID, "rnp_key_get_primary_uid")
		reg(&bound.keyCreation, "rnp_key_get_creation")
		reg(&bound.keyExpiry, "rnp_key_get_expiration")
		reg(&bound.keySetExpiry, "rnp_key_set_expiration")
		reg(&bound.keySecret, "rnp_key_have_secret")
		reg(&bound.keyLocked, "rnp_key_is_locked")
		reg(&bound.keyRevoked, "rnp_key_is_revoked")
		reg(&bound.keyUsage, "rnp_key_allows_usage")
		reg(&bound.encPassword, "rnp_op_encrypt_add_password")
		reg(&bound.encSign, "rnp_op_encrypt_add_signature")
		reg(&bound.encCompress, "rnp_op_encrypt_set_compression")
		reg(&bound.signCreate, "rnp_op_sign_create")
		reg(&bound.signClear, "rnp_op_sign_cleartext_create")
		reg(&bound.signCompress, "rnp_op_sign_set_compression")
		reg(&bound.dumpToOutput, "rnp_dump_packets_to_output")
		reg(&bound.dumpToJSON, "rnp_dump_packets_to_json")
		reg(&bound.features, "rnp_supported_features")
		reg(&bound.supports, "rnp_supports_feature")
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
