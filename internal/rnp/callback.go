package rnp

import (
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/ebitengine/purego"
)

// maxPass leaves room for the NUL that rnp's callback buffer requires.
// librnp uses a 256-byte buffer (MAX_PASSWORD_LENGTH).
const maxPass = 255

// inCallback is set only while the password trampoline is on the stack.
// A librnp call from that trampoline would deadlock on the session lock.
var inCallback atomic.Bool

var passCB uintptr

type slot struct {
	mu      sync.Mutex
	pass    []byte
	cancel  bool
	calls   int
	tid     int
	lastCtx string
}

var (
	slotsMu  sync.Mutex
	slots            = map[uintptr]*slot{}
	nextSlot uintptr = 1
)

func newSlot(pass []byte) (uintptr, error) {
	if len(pass) > maxPass {
		return 0, ErrPass
	}
	slotsMu.Lock()
	defer slotsMu.Unlock()
	id := nextSlot
	nextSlot++
	slots[id] = &slot{pass: append([]byte(nil), pass...)}
	return id, nil
}

func dropSlot(id uintptr) {
	if id == 0 {
		return
	}
	slotsMu.Lock()
	sl := slots[id]
	delete(slots, id)
	slotsMu.Unlock()
	if sl == nil {
		return
	}
	sl.mu.Lock()
	wipe(sl.pass)
	sl.pass = nil
	sl.cancel = true
	sl.mu.Unlock()
}

func slotByID(id uintptr) *slot {
	slotsMu.Lock()
	sl := slots[id]
	slotsMu.Unlock()
	return sl
}

// passCallback is the only rnp_password_cb trampoline.
//
// Constraints, from purego v0.11.1 and rnp.h at v0.18.1:
//   - Arguments are uintptr-sized. Strings and slices are rejected by NewCallback.
//   - The first argument is purego.CDecl so Windows 386 uses cdecl. On
//     amd64 and arm64 Windows the platform convention is a single one;
//     that path was not executed here.
//   - app_ctx is an integer slot id. It is not a Go pointer.
//   - key is not stored. rnp destroys that handle when the callback returns.
//   - pgp_context and buf are C pointers used only before return.
//   - purego never frees the trampoline. Closing a session drops the slot.
func passCallback(_ purego.CDecl, _, app, key, pgp, buf, bufLen uintptr) bool {
	if inCallback.Swap(true) {
		return false
	}
	defer inCallback.Store(false)
	_ = key
	sl := slotByID(app)
	if sl == nil {
		return false
	}
	sl.mu.Lock()
	defer sl.mu.Unlock()
	sl.calls++
	sl.tid = currentTID()
	sl.lastCtx = readCString(pgp, 64)
	if sl.cancel || len(sl.pass) == 0 || buf == 0 || uintptr(len(sl.pass))+1 > bufLen {
		return false
	}
	dst := unsafe.Slice((*byte)(cPtr(buf)), bufLen)
	copy(dst, sl.pass)
	dst[len(sl.pass)] = 0
	for i := len(sl.pass) + 1; i < len(dst); i++ {
		dst[i] = 0
	}
	return true
}

func readCString(p uintptr, max int) string {
	if p == 0 || max <= 0 {
		return ""
	}
	// unsafe.Slice of a C pointer is rejected by checkptr, which the race
	// detector turns on. Run that detector with -d=checkptr=0; this is
	// librnp's buffer, valid only for this call.
	b := unsafe.Slice((*byte)(cPtr(p)), max)
	n := 0
	for n < len(b) && b[n] != 0 {
		n++
	}
	return string(b[:n])
}
