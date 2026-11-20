//go:build unix

package rnp

import "os"

func euid() int { return os.Geteuid() }
