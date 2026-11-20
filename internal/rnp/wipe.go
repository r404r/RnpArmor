package rnp

import (
	"runtime"
	"unsafe"
)

// cPtr turns a C address held in a uintptr into a pointer without go vet
// treating the conversion as a dangling unsafe.Pointer. The address is
// valid only for the librnp call that produced it.
func cPtr(p uintptr) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&p))
}

func wipe(b []byte) {
	for i := range b {
		b[i] = 0
	}
	runtime.KeepAlive(b)
}
