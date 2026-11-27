// Command seeddemo writes a disposable demo keyring for screenshot CI.
// It is not part of the release build.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/r404r/RnpArmor/internal/rnp"
)

const (
	demoPassphrase = "demo-pass"
	demoNoteBody   = "RnpArmor portable OpenPGP demo message.\n"
	demoNoteName   = "demo-note.txt"
	demoSignedName = "demo-signed.asc"
	demoEncName    = "demo-encrypted.pgp"
)

func main() {
	dir, err := parseSeedArgs(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2)
	}
	if err := seedDemo(dir); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func parseSeedArgs(args []string) (string, error) {
	fs := flag.NewFlagSet("seeddemo", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	dir := fs.String("dir", "", "workspace directory")
	if err := fs.Parse(args); err != nil {
		return "", err
	}
	if strings.TrimSpace(*dir) == "" {
		return "", errors.New("--dir is required")
	}
	abs, err := filepath.Abs(strings.TrimSpace(*dir))
	if err != nil {
		return "", err
	}
	return abs, nil
}

// seedPlans is the fixed demo key set. Alice keeps a protected secret and an
// extra signing subkey. Bob is generated aside and imported as public only.
// Carol is an unprotected secret so the UI can show an empty passphrase.
func seedPlans() (alice, carol, bob rnp.KeyGenRequest, aliceSign rnp.SubkeyGenRequest) {
	alice = rnp.KeyGenRequest{
		UserID:       "Alice Chen <alice@example.com>",
		PrimaryAlg:   "EDDSA",
		PrimaryCurve: "Ed25519",
		SubkeyAlg:    "ECDH",
		SubkeyCurve:  "Curve25519",
		HashAlg:      "SHA256",
		SymmCipher:   "AES256",
	}
	aliceSign = rnp.SubkeyGenRequest{
		Usage:       "sign",
		Algorithm:   "EDDSA",
		Curve:       "Ed25519",
		HashAlg:     "SHA256",
		Unprotected: false,
	}
	carol = alice
	carol.UserID = "Carol Smith <carol@example.com>"
	carol.Unprotected = true
	bob = rnp.KeyGenRequest{
		UserID:      "Bob Tanaka <bob@example.org>",
		PrimaryAlg:  "RSA",
		PrimaryBits: 3072,
		SubkeyAlg:   "RSA",
		SubkeyBits:  3072,
		HashAlg:     "SHA256",
		SymmCipher:  "AES256",
	}
	return alice, carol, bob, aliceSign
}

func seedDemo(dir string) error {
	if !filepath.IsAbs(dir) {
		return errors.New("workspace must be absolute")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	for _, name := range []string{"pubring.gpg", "secring.gpg", demoEncName} {
		path := filepath.Join(dir, name)
		if _, err := os.Lstat(path); err == nil {
			return fmt.Errorf("%s already exists", name)
		} else if !os.IsNotExist(err) {
			return err
		}
	}

	lib, err := rnp.DefaultLibrary()
	if err != nil {
		return err
	}
	ctx := context.Background()
	pass := []byte(demoPassphrase)
	defer zero(pass)
	sess, err := rnp.Open(lib, pass)
	if err != nil {
		return err
	}
	defer sess.Close()

	aliceReq, carolReq, bobReq, signReq := seedPlans()
	alice, err := sess.GenerateKeyWithSpec(ctx, aliceReq)
	if err != nil {
		return err
	}
	if _, err := sess.AddSubkey(ctx, alice.Fingerprint, signReq); err != nil {
		return err
	}
	carol, err := sess.GenerateKeyWithSpec(ctx, carolReq)
	if err != nil {
		return err
	}
	bobFP, armored, err := generatePublicOnly(ctx, lib, bobReq)
	if err != nil {
		return err
	}
	if _, err := sess.ImportKeys(ctx, []byte(armored)); err != nil {
		return err
	}

	notePath := filepath.Join(dir, demoNoteName)
	if err := os.WriteFile(notePath, []byte(demoNoteBody), 0o644); err != nil {
		return err
	}
	signed, err := sess.SignText(ctx, demoNoteBody, carol.Fingerprint, rnp.SignModeCleartext, "SHA256")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, demoSignedName), []byte(signed), 0o644); err != nil {
		return err
	}
	if err := sess.EncryptFileWithOpts(ctx, notePath, filepath.Join(dir, demoEncName), rnp.EncryptOptions{
		RecipientFprints: []string{alice.Fingerprint, bobFP},
		Cipher:           "AES256",
		Compression:      "zlib",
		Armor:            false,
	}); err != nil {
		return err
	}
	return sess.SaveTo(ctx, dir)
}

// generatePublicOnly builds Bob in a side session and returns armored public
// key material. The side session is closed without SaveTo, so the secret
// never lands in the demo secring.
func generatePublicOnly(ctx context.Context, lib string, req rnp.KeyGenRequest) (string, string, error) {
	pass := []byte("bob-demo-pass")
	defer zero(pass)
	side, err := rnp.Open(lib, pass)
	if err != nil {
		return "", "", err
	}
	defer side.Close()
	sum, err := side.GenerateKeyWithSpec(ctx, req)
	if err != nil {
		return "", "", err
	}
	armored, err := side.ExportKey(ctx, sum.Fingerprint, false)
	if err != nil {
		return "", "", err
	}
	return sum.Fingerprint, armored, nil
}

func zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
