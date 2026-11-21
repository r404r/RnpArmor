//go:build !linux

package rnp

func currentTID() int { return 0 }
