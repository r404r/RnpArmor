//go:build darwin || linux

package loadlib

import "runtime"

var LibraryFile = map[string]string{
	"darwin": "librnp.dylib",
	"linux":  "librnp.so",
}[runtime.GOOS]
