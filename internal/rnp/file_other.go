//go:build !linux

package rnp

import "os"

// publish copies tmp onto a new dst created with O_EXCL. This is not
// atomic. The Linux path uses renameat2 and is the one exercised by tests.
func publish(tmp, dst string) error {
	in, err := os.ReadFile(tmp)
	if err != nil {
		return err
	}
	defer wipe(in)
	f, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return ErrExists
		}
		return err
	}
	_, werr := f.Write(in)
	serr := f.Sync()
	cerr := f.Close()
	os.Remove(tmp)
	if werr != nil || serr != nil || cerr != nil {
		os.Remove(dst)
		if werr != nil {
			return werr
		}
		if serr != nil {
			return serr
		}
		return cerr
	}
	return nil
}
