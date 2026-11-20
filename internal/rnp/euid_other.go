//go:build !unix

package rnp

func euid() int { return -1 }
