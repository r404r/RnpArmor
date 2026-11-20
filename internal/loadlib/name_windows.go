//go:build windows

package loadlib

// rnp.dll is the file CMake produces on Windows: the target output name is
// "rnp" and the shared-library prefix is empty. Launching the portable
// directory loads this name from beside the executable.
const LibraryFile = "rnp.dll"
