//go:build !windows

package loadlib

import "github.com/ebitengine/purego"

func openShared(path string) (uintptr, error) {
	// RTLD_NOW resolves NEEDED entries immediately. RTLD_LOCAL keeps the
	// symbols out of the global namespace. The path is absolute, so dlopen
	// does not search LD_LIBRARY_PATH for librnp itself. Dependencies use
	// the library's RPATH ($ORIGIN), set by scripts/build-librnp.sh.
	return purego.Dlopen(path, purego.RTLD_NOW|purego.RTLD_LOCAL)
}
