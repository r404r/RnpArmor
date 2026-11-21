package rnp

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"runtime"
	"sync"

	"github.com/r404r/RnpArmor/internal/loadlib"
)

var (
	ErrClosed  = errors.New("session closed")
	ErrReenter = errors.New("librnp call from the password callback")
	ErrPass    = errors.New("passphrase unavailable")
	ErrNoKey   = errors.New("session has no generated key")
)

// VerifyResult is the outcome of a detached signature check.
// MathValid means librnp reported RNP_SUCCESS for that signature against
// a key in this session. IdentityTrusted is always false: a signature
// check is not an identity or trust decision.
type VerifyResult struct {
	MathValid       bool
	Status          uint32
	StatusText      string
	IdentityTrusted bool
	SignerKeyID     string
	SignerFingerprint string
	CreatedUnix     uint32
	ExpiresUnix     uint32
	HashAlg         string
}

type KeySummary struct {
	Fingerprint string
	KeyID       string
	PrimaryUID  string
	Algorithm   string
	CreatedUnix uint32
	ExpiresUnix uint32
	HasSecret   bool
	IsLocked    bool
	IsRevoked   bool
	CanSign     bool
	CanEncrypt  bool
	CanCertify  bool
}

type KeyGenRequest struct {
	UserID        string
	PrimaryAlg    string
	PrimaryBits   int
	PrimaryCurve  string
	SubkeyAlg     string
	SubkeyBits    int
	SubkeyCurve   string
	HashAlg       string
	SymmCipher    string
	S2KIterations uint32
	ExpirationSec uint32
}

type EncryptOptions struct {
	RecipientFprints []string
	Password         string
	SignerFprint     string
	Cipher           string
	Compression      string
	CompressionLevel int
	Armor            bool
}

type SignMode string

const (
	SignModeCleartext SignMode = "cleartext"
	SignModeDetached  SignMode = "detached"
	SignModeInline    SignMode = "inline"
)

type DecryptVerifyResult struct {
	Plaintext string
	Verify    VerifyResult
}

func defaultAlg(v, d string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return d
	}
	return strings.ToUpper(v)
}

// Session is the only owner of one rnp_ffi_t. Do and the methods that call
// it take s.mu and then the process-wide apiMu, so two sessions cannot
// enter librnp at once. The password callback runs on that same OS thread
// and must not call back into the session.
type Session struct {
	mu         sync.Mutex
	ffi        uintptr
	primary    uintptr
	encKey     uintptr
	slot       uintptr
	closed     bool
	enteredTID int
	callsAt    int
}

var apiMu sync.Mutex

// DefaultLibrary is the absolute librnp path from RNPARMOR_LIBRNP or the
// executable directory. It does not look at the working directory.
func DefaultLibrary() (string, error) { return loadlib.Resolve() }

// Probe loads librnp and returns rnp_version_string. It does not create
// a keyring and does not read a passphrase.
func Probe(path string) (string, error) {
	if err := ensureLib(path); err != nil {
		return "", err
	}
	apiMu.Lock()
	defer apiMu.Unlock()
	if bound.version == nil {
		return "", errors.New("rnp_version_string was not bound")
	}
	v := bound.version()
	if v == "" {
		return "", errors.New("empty librnp version")
	}
	return v, nil
}

var (
	loadOnce   sync.Once
	loadErr    error
	loadedPath string
)

func ensureLib(path string) error {
	loadOnce.Do(func() {
		h, err := loadlib.Open(path)
		if err != nil {
			loadErr = err
			return
		}
		loadErr = bind(h)
		loadedPath = path
	})
	if loadErr != nil {
		return loadErr
	}
	if loadedPath != path {
		return fmt.Errorf("librnp already loaded from %s", loadedPath)
	}
	return nil
}

// Open creates an empty GPG-format ffi and registers the password slot.
// pass is copied. The caller can wipe its own buffer. Open does not read
// or write the default keyring.
func Open(libPath string, pass []byte) (*Session, error) {
	if inCallback.Load() {
		return nil, ErrReenter
	}
	if len(pass) == 0 || len(pass) > maxPass {
		return nil, ErrPass
	}
	if err := ensureLib(libPath); err != nil {
		return nil, err
	}
	id, err := newSlot(pass)
	if err != nil {
		return nil, err
	}
	apiMu.Lock()
	defer apiMu.Unlock()
	var ffi uintptr
	if err := bound.check("ffi_create", bound.ffiCreate(&ffi, "GPG", "GPG")); err != nil {
		dropSlot(id)
		return nil, err
	}
	if err := bound.check("set_pass_provider", bound.setPass(ffi, passCB, id)); err != nil {
		bound.ffiDestroy(ffi)
		dropSlot(id)
		return nil, err
	}
	return &Session{ffi: ffi, slot: id}, nil
}

// Close destroys key handles, the ffi, and the password slot. The purego
// trampoline stays allocated for the process; the slot and passphrase do not.
func (s *Session) Close() error {
	if inCallback.Load() {
		return ErrReenter
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	apiMu.Lock()
	defer apiMu.Unlock()
	if s.primary != 0 {
		bound.keyDestroy(s.primary)
		s.primary = 0
	}
	if s.encKey != 0 {
		bound.keyDestroy(s.encKey)
		s.encKey = 0
	}
	if s.ffi != 0 {
		bound.ffiDestroy(s.ffi)
		s.ffi = 0
	}
	dropSlot(s.slot)
	s.slot = 0
	return nil
}

// Do runs fn on the caller's goroutine while holding the session. Call it
// from a worker, not from the MyGo UI thread. The password callback is
// synchronous on this same OS thread.
func (s *Session) Do(ctx context.Context, fn func(context.Context) error) error {
	if inCallback.Load() {
		return ErrReenter
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return ErrClosed
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	s.enteredTID = currentTID()
	apiMu.Lock()
	defer apiMu.Unlock()
	return fn(ctx)
}

// Go starts Do on another goroutine and returns before fn finishes.
// The UI thread uses this so a librnp call does not block the window.
func (s *Session) Go(ctx context.Context, fn func(context.Context) error) <-chan error {
	ch := make(chan error, 1)
	go func() {
		ch <- s.Do(ctx, fn)
	}()
	return ch
}

func (s *Session) Version(ctx context.Context) (string, error) {
	var v string
	err := s.Do(ctx, func(context.Context) error {
		v = bound.version()
		return nil
	})
	return v, err
}

// SetPassphrase replaces the slot's copy. It does not log pass.
func (s *Session) SetPassphrase(pass []byte) error {
	if inCallback.Load() {
		return ErrReenter
	}
	if len(pass) == 0 || len(pass) > maxPass {
		return ErrPass
	}
	sl := slotByID(s.slot)
	if sl == nil {
		return ErrClosed
	}
	sl.mu.Lock()
	defer sl.mu.Unlock()
	wipe(sl.pass)
	sl.pass = append([]byte(nil), pass...)
	sl.cancel = false
	return nil
}

// SetPassCancel makes the callback refuse without writing a passphrase.
func (s *Session) SetPassCancel(on bool) error {
	if inCallback.Load() {
		return ErrReenter
	}
	sl := slotByID(s.slot)
	if sl == nil {
		return ErrClosed
	}
	sl.mu.Lock()
	sl.cancel = on
	sl.mu.Unlock()
	return nil
}

func (s *Session) PasswordCalls() int {
	sl := slotByID(s.slot)
	if sl == nil {
		return 0
	}
	sl.mu.Lock()
	n := sl.calls
	sl.mu.Unlock()
	return n
}

func (s *Session) callbackTID() int {
	sl := slotByID(s.slot)
	if sl == nil {
		return 0
	}
	sl.mu.Lock()
	n := sl.tid
	sl.mu.Unlock()
	return n
}

// RequestPassword asks librnp to invoke the provider and checks that the
// returned buffer matches the slot. The bytes are wiped and not logged.
func (s *Session) RequestPassword(ctx context.Context) error {
	return s.Do(ctx, func(context.Context) error {
		return s.requestPassword()
	})
}

func (s *Session) requestPassword() error {
	var p uintptr
	if err := bound.check("request_password", bound.requestPass(s.ffi, 0, "decrypt", &p)); err != nil {
		if p != 0 {
			bound.bufClear(p, uintptr(cStringLen(p)))
			bound.bufDestroy(p)
		}
		return err
	}
	if p == 0 {
		return ErrPass
	}
	got := append([]byte(nil), readCString(p, maxPass)...)
	n := uintptr(len(got))
	bound.bufClear(p, n+1)
	bound.bufDestroy(p)
	sl := slotByID(s.slot)
	if sl == nil {
		wipe(got)
		return ErrClosed
	}
	sl.mu.Lock()
	same := len(got) == len(sl.pass) && subtleEqual(got, sl.pass)
	sl.mu.Unlock()
	wipe(got)
	if !same {
		return ErrPass
	}
	if s.callbackTID() != s.enteredTID {
		return fmt.Errorf("password callback thread %d, caller thread %d", s.callbackTID(), s.enteredTID)
	}
	return nil
}

func cStringLen(p uintptr) int {
	return len(readCString(p, maxPass+1))
}

func subtleEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var d byte
	for i := range a {
		d |= a[i] ^ b[i]
	}
	return d == 0
}

// Burn runs rnp_calculate_iterations on the session thread. It is the
// stand-in for a long librnp call: cancellation is observed only after
// the call returns, and nothing is written.
func (s *Session) Burn(ctx context.Context, msec int) error {
	return s.Do(ctx, func(ctx context.Context) error {
		var n uintptr
		if err := bound.check("calculate_iterations", bound.calcIter("SHA256", uintptr(msec), &n)); err != nil {
			return err
		}
		return ctx.Err()
	})
}
