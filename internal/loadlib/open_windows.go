//go:build windows

package loadlib

import (
	"fmt"

	"golang.org/x/sys/windows"
)

func openShared(path string) (uintptr, error) {
	// LOAD_LIBRARY_SEARCH_DLL_LOAD_DIR searches the directory containing
	// path for dependencies. LOAD_LIBRARY_SEARCH_SYSTEM32 allows system
	// DLLs. The current directory and PATH are not searched. path must be
	// absolute; LoadLibraryEx requires that when DLL_LOAD_DIR is set.
	const flags = windows.LOAD_LIBRARY_SEARCH_DLL_LOAD_DIR | windows.LOAD_LIBRARY_SEARCH_SYSTEM32
	h, err := windows.LoadLibraryEx(path, 0, flags)
	if err != nil {
		return 0, fmt.Errorf("LoadLibraryEx: %w", err)
	}
	return uintptr(h), nil
}
