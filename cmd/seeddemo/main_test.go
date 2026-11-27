package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/r404r/RnpArmor/internal/rnp"
)

func TestParseSeedArgs(t *testing.T) {
	if _, err := parseSeedArgs(nil); err == nil {
		t.Fatal("missing --dir")
	}
	if _, err := parseSeedArgs([]string{"--dir", "   "}); err == nil {
		t.Fatal("blank --dir")
	}
	dir := t.TempDir()
	got, err := parseSeedArgs([]string{"--dir", dir})
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(got) || got != dir {
		t.Fatalf("abs dir %q", got)
	}
	rel, err := parseSeedArgs([]string{"--dir", "rel-demo"})
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(rel) || !strings.HasSuffix(rel, "rel-demo") {
		t.Fatalf("relative dir %q", rel)
	}
}

func TestSeedPlans(t *testing.T) {
	alice, carol, bob, sign := seedPlans()
	if alice.UserID != "Alice Chen <alice@example.com>" || alice.Unprotected || alice.PrimaryAlg != "EDDSA" || alice.PrimaryCurve != "Ed25519" || alice.SubkeyAlg != "ECDH" || alice.SubkeyCurve != "Curve25519" {
		t.Fatalf("alice %+v", alice)
	}
	if sign.Usage != "sign" || sign.Algorithm != "EDDSA" || sign.Curve != "Ed25519" || sign.Unprotected {
		t.Fatalf("sign subkey %+v", sign)
	}
	if carol.UserID != "Carol Smith <carol@example.com>" || !carol.Unprotected || carol.PrimaryAlg != "EDDSA" {
		t.Fatalf("carol %+v", carol)
	}
	if bob.UserID != "Bob Tanaka <bob@example.org>" || bob.PrimaryAlg != "RSA" || bob.PrimaryBits != 3072 || bob.SubkeyAlg != "RSA" || bob.SubkeyBits != 3072 || bob.Unprotected {
		t.Fatalf("bob %+v", bob)
	}
	if demoPassphrase != "demo-pass" || !strings.Contains(demoNoteBody, "RnpArmor portable OpenPGP demo message.") {
		t.Fatal("demo constants")
	}
}

func TestSeedLive(t *testing.T) {
	if os.Getenv("RNPARMOR_LIBRNP") == "" {
		t.Skip("RNPARMOR_LIBRNP is not set")
	}
	dir := t.TempDir()
	if err := seedDemo(dir); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"pubring.gpg", "secring.gpg", demoNoteName, demoSignedName, demoEncName} {
		st, err := os.Stat(filepath.Join(dir, name))
		if err != nil || st.Size() == 0 {
			t.Fatalf("%s: %v size=%d", name, err, sizeOf(st))
		}
	}
	note, err := os.ReadFile(filepath.Join(dir, demoNoteName))
	if err != nil || !bytes.Contains(note, []byte("RnpArmor portable OpenPGP demo message.")) {
		t.Fatalf("note %q %v", note, err)
	}
	asc, err := os.ReadFile(filepath.Join(dir, demoSignedName))
	if err != nil || !bytes.Contains(asc, []byte("BEGIN PGP")) {
		t.Fatalf("signed %v", err)
	}
	enc, err := os.ReadFile(filepath.Join(dir, demoEncName))
	if err != nil || bytes.HasPrefix(enc, []byte("-----")) {
		t.Fatal("encrypted file is not binary")
	}

	pass := []byte(demoPassphrase)
	defer zero(pass)
	sess, err := rnp.Open(os.Getenv("RNPARMOR_LIBRNP"), pass)
	if err != nil {
		t.Fatal(err)
	}
	defer sess.Close()
	ctx := context.Background()
	if err := sess.LoadFrom(ctx, dir); err != nil {
		t.Fatal(err)
	}
	keys, err := sess.ListKeys(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var alice, bob, carol *rnp.KeySummary
	primaries := 0
	for i := range keys {
		k := &keys[i]
		if !k.IsPrimary {
			continue
		}
		primaries++
		switch {
		case strings.Contains(k.PrimaryUID, "Alice Chen"):
			alice = k
		case strings.Contains(k.PrimaryUID, "Bob Tanaka"):
			bob = k
		case strings.Contains(k.PrimaryUID, "Carol Smith"):
			carol = k
		}
	}
	if primaries != 3 || alice == nil || bob == nil || carol == nil {
		t.Fatalf("primaries=%d alice=%v bob=%v carol=%v", primaries, alice != nil, bob != nil, carol != nil)
	}
	if !alice.HasSecret || !alice.IsProtected || len(alice.Subkeys) < 2 {
		t.Fatalf("alice %+v", alice)
	}
	signSubs := 0
	for _, sk := range alice.Subkeys {
		if sk.CanSign {
			signSubs++
		}
	}
	if signSubs == 0 {
		t.Fatalf("alice sign subkeys %+v", alice.Subkeys)
	}
	if bob.HasSecret {
		t.Fatalf("bob kept a secret %+v", bob)
	}
	if !carol.HasSecret || carol.IsProtected {
		t.Fatalf("carol %+v", carol)
	}
}

func sizeOf(st os.FileInfo) int64 {
	if st == nil {
		return 0
	}
	return st.Size()
}
