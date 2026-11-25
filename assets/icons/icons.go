// Package icons holds the embedded application icon.
package icons

import _ "embed"

// AppIconPNG is the 256x256 PNG used for window and Dock icons.
//
//go:embed rnparmor-256.png
var AppIconPNG []byte
