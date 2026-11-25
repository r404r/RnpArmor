package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"

	"github.com/r404r/RnpArmor/internal/rnp"
)

func TestValidateCatalogIncludesMenuAndAboutKeys(t *testing.T) {
	if err := ValidateCatalog(); err != nil {
		t.Fatalf("ValidateCatalog() error = %v", err)
	}

	keys := []string{
		"menu.file",
		"menu.workspace",
		"menu.language",
		"menu.help",
		"menu.help.about",
		"btn.close",
		"about.no_menu_hint",
	}
	langs := []Lang{LangEN, LangZH, LangJA}
	for _, k := range keys {
		for _, l := range langs {
			v := strings.TrimSpace(Tr(l, k))
			if v == "" || v == k {
				t.Fatalf("translation missing or fallback key for %s/%s: %q", k, l, v)
			}
		}
	}
}

func TestMenuTemplateLayoutAndRadioLanguage(t *testing.T) {
	s := &uiState{langIdx: LangIndex(LangEN)}
	items := s.menuTemplate()
	if got, want := len(items), 4; got != want {
		t.Fatalf("top menu count = %d, want %d", got, want)
	}

	if got := items[0].Label; got != Tr(LangEN, "menu.file") {
		t.Fatalf("menu[0] label = %q", got)
	}
	if got := len(items[0].Submenu); got != 5 {
		t.Fatalf("file submenu count = %d, want 5", got)
	}
	if items[0].Submenu[0].Label != Tr(LangEN, "btn.open_session") {
		t.Fatalf("file[0] label = %q", items[0].Submenu[0].Label)
	}
	if items[0].Submenu[1].Label != Tr(LangEN, "btn.load_keyring") {
		t.Fatalf("file[1] label = %q", items[0].Submenu[1].Label)
	}
	if items[0].Submenu[2].Label != Tr(LangEN, "btn.save_keyring") {
		t.Fatalf("file[2] label = %q", items[0].Submenu[2].Label)
	}
	if items[0].Submenu[3].Type != mygo.MenuItemSeparator {
		t.Fatalf("file[3] type = %q, want separator", items[0].Submenu[3].Type)
	}
	if items[0].Submenu[4].Label != Tr(LangEN, "menu.file.quit") {
		t.Fatalf("file[4] label = %q", items[0].Submenu[4].Label)
	}

	if got := len(items[1].Submenu); got != 5 {
		t.Fatalf("workspace submenu count = %d, want 5", got)
	}
	for i, it := range items[1].Submenu {
		if it.Type != mygo.MenuItemRadio {
			t.Fatalf("workspace submenu item %d type = %q, want radio", i, it.Type)
		}
		wantChecked := i == 0
		if it.Checked != wantChecked {
			t.Fatalf("workspace submenu item %d checked = %v, want %v at default tab 0", i, it.Checked, wantChecked)
		}
	}

	if got := len(items[2].Submenu); got != 3 {
		t.Fatalf("language submenu count = %d, want 3", got)
	}
	for i, it := range items[2].Submenu {
		if it.Type != mygo.MenuItemRadio {
			t.Fatalf("language submenu item %d type = %q, want radio", i, it.Type)
		}
		wantChecked := i == LangIndex(LangEN)
		if it.Checked != wantChecked {
			t.Fatalf("language submenu item %d checked = %v, want %v", i, it.Checked, wantChecked)
		}
	}
	if got := len(items[3].Submenu); got != 1 {
		t.Fatalf("help submenu count = %d, want 1", got)
	}

	s.tab = 3
	s.langIdx = LangIndex(LangJA)
	items = s.menuTemplate()
	for i, it := range items[1].Submenu {
		wantChecked := i == 3
		if it.Checked != wantChecked {
			t.Fatalf("workspace submenu item %d checked = %v, want %v when tab=3", i, it.Checked, wantChecked)
		}
	}
	for i, it := range items[2].Submenu {
		wantChecked := i == LangIndex(LangJA)
		if it.Checked != wantChecked {
			t.Fatalf("language submenu item %d checked = %v, want %v when lang=ja", i, it.Checked, wantChecked)
		}
	}
}

func TestBuildSmokeOutputAndWriteSmokeOutput(t *testing.T) {
	got := buildSmokeOutput("v1.2.3", `C:\\RnpArmor\\rnp.dll`, "0.18.1")
	for _, wantLine := range []string{
		"status=ok",
		"app_version=v1.2.3",
		`library=C:\\RnpArmor\\rnp.dll`,
		"version=0.18.1",
		"i18n=ok",
	} {
		if !strings.Contains(got, wantLine+"\n") {
			t.Fatalf("smoke output missing line %q in %q", wantLine, got)
		}
	}

	if err := writeSmokeOutput("", got); err != nil {
		t.Fatalf("writeSmokeOutput(empty) error = %v", err)
	}

	root := t.TempDir()
	out := filepath.Join(root, "nested", "smoke", "smoke-out.txt")
	if err := writeSmokeOutput(out, got); err != nil {
		t.Fatalf("writeSmokeOutput(%q) error = %v", out, err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", out, err)
	}
	if string(b) != got {
		t.Fatalf("smoke file content mismatch: got %q want %q", string(b), got)
	}
}

func TestResolveDecryptInput(t *testing.T) {
	const msg = "-----BEGIN PGP MESSAGE-----\nA\n-----END PGP MESSAGE-----"
	const other = "-----BEGIN PGP MESSAGE-----\nB\n-----END PGP MESSAGE-----"
	cases := []struct {
		name, in, verify, out, want string
	}{
		{name: "armored input wins", in: msg, verify: "hidden", out: other, want: msg},
		{name: "verify buffer", in: "hello", verify: msg, out: other, want: msg},
		{name: "output when verify empty", in: "hello", verify: "", out: other, want: other},
		{name: "blank verify is empty", in: "hello", verify: "  \n", out: other, want: other},
		{name: "plain fallback", in: "hello", verify: "", out: "plain", want: "hello"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveDecryptInput(tc.in, tc.verify, tc.out)
			if got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestResolveVerifyInput(t *testing.T) {
	const signed = "-----BEGIN PGP SIGNED MESSAGE-----\n\nhi\n-----BEGIN PGP SIGNATURE-----\nS\n-----END PGP SIGNATURE-----"
	const sig = "-----BEGIN PGP SIGNATURE-----\nD\n-----END PGP SIGNATURE-----"
	const inline = "-----BEGIN PGP MESSAGE-----\nI\n-----END PGP MESSAGE-----"

	got, det := resolveVerifyInput("hello", "other", signed, sig)
	if got != "hello" || det != sig {
		t.Fatalf("detached input: got (%q, %q)", got, det)
	}
	got, det = resolveVerifyInput("  ", "other", signed, sig)
	if got != "other" || det != sig {
		t.Fatalf("detached fallback: got (%q, %q)", got, det)
	}
	got, det = resolveVerifyInput("", "", signed, sig)
	if got != "" || det != sig {
		t.Fatalf("detached empty plain: got (%q, %q)", got, det)
	}
	got, det = resolveVerifyInput(signed, "verify", inline, "")
	if got != signed || det != "" {
		t.Fatalf("armored input: got (%q, %q)", got, det)
	}
	got, det = resolveVerifyInput("hello", signed, inline, "")
	if got != signed || det != "" {
		t.Fatalf("verify buffer: got (%q, %q)", got, det)
	}
	got, det = resolveVerifyInput("hello", "  ", inline, " \t")
	if got != inline || det != "" {
		t.Fatalf("output armor: got (%q, %q)", got, det)
	}
	got, det = resolveVerifyInput("hello", "", "plain", "")
	if got != "hello" || det != "" {
		t.Fatalf("plain fallback: got (%q, %q)", got, det)
	}
}

func TestRecipientFingerprintsFallback(t *testing.T) {
	in := actionInput{selected: "ABC"}
	got := recipientFingerprints(in)
	if len(got) != 1 || got[0] != "ABC" {
		t.Fatalf("fallback %#v", got)
	}
	in.encRecipients = " DEF , GHI "
	got = recipientFingerprints(in)
	if len(got) != 2 || got[0] != "DEF" || got[1] != "GHI" {
		t.Fatalf("explicit %#v", got)
	}
	in.encRecipients = ""
	in.encPassword = "pw"
	got = recipientFingerprints(in)
	if len(got) != 0 {
		t.Fatalf("password %#v", got)
	}
	in.encPassword = ""
	in.selected = "  "
	got = recipientFingerprints(in)
	if len(got) != 0 {
		t.Fatalf("blank selected %#v", got)
	}
}

func TestUIStateActionsEndToEnd(t *testing.T) {
	if strings.TrimSpace(os.Getenv("RNPARMOR_LIBRNP")) == "" {
		t.Skip("RNPARMOR_LIBRNP is not set")
	}
	lib, err := rnp.DefaultLibrary()
	if err != nil {
		t.Skip(err.Error())
	}
	ws := t.TempDir()
	errCh := make(chan error, 1)
	go func() {
		errCh <- exerciseUIActions(lib, ws)
	}()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(20 * time.Second):
		t.Fatal("ui actions timed out")
	}
}

func exerciseUIActions(lib, ws string) error {
	const plain = "hello from RnpArmor"
	s := &uiState{
		libPath:      lib,
		workspace:    ws,
		passphrase:   "change-me",
		encCipher:    "AES256",
		encCompress:  "zlib",
		textMode:     string(rnp.SignModeDetached),
		textInput:    plain,
		featureQuery: "",
		langIdx:      LangIndex(LangEN),
		keyReq: rnp.KeyGenRequest{
			UserID:       "User <user@example.com>",
			PrimaryAlg:   "EDDSA",
			PrimaryCurve: "Ed25519",
			SubkeyAlg:    "ECDH",
			SubkeyCurve:  "Curve25519",
			HashAlg:      "SHA256",
			SymmCipher:   "AES256",
		},
	}

	if err := runUI(s, "action.open_session", s.openSession); err != nil {
		return err
	}
	if err := assertNoKeyring(ws); err != nil {
		return fmt.Errorf("open session: %w", err)
	}
	if err := runUI(s, "action.query_features", s.queryFeatures); err != nil {
		return err
	}
	if !strings.Contains(s.features, "{") {
		return fmt.Errorf("features not json: %q", s.features)
	}
	if err := assertNoKeyring(ws); err != nil {
		return fmt.Errorf("query features: %w", err)
	}

	s.run("action.save_keyring", s.saveKeyring)
	if s.busy {
		return fmt.Errorf("empty save left busy set")
	}
	if !strings.Contains(s.lastErr, rnp.ErrNoKey.Error()) {
		return fmt.Errorf("empty save lastErr=%q", s.lastErr)
	}
	if err := assertNoKeyring(ws); err != nil {
		return fmt.Errorf("empty save: %w", err)
	}

	if err := runUI(s, "action.generate_key", s.generateKey); err != nil {
		return err
	}
	if len(s.keys) == 0 || s.selected == "" {
		return fmt.Errorf("generate keys=%d selected=%q", len(s.keys), s.selected)
	}
	selectedOK := false
	certify := ""
	for _, k := range s.keys {
		if k.Fingerprint == s.selected {
			selectedOK = true
		}
		if certify == "" && k.CanCertify && k.HasSecret {
			certify = k.Fingerprint
		}
	}
	if !selectedOK {
		return fmt.Errorf("selected %q is not in the key list", s.selected)
	}
	if certify == "" {
		return fmt.Errorf("no secret certify key after generate")
	}
	s.selected = certify
	pub, sec, err := readKeyring(ws)
	if err != nil {
		return fmt.Errorf("generate keyring: %w", err)
	}

	if certify != "" {
		s.selected = certify
	}
	if err := runUI(s, "action.key_json", s.loadKeyJSON); err != nil {
		return err
	}
	if strings.TrimSpace(s.keyJSON) == "" || !strings.Contains(s.keyJSON, "{") {
		return fmt.Errorf("key json missing identity: %s", s.keyJSON)
	}
	if err := runUI(s, "action.export_public", s.exportPublic); err != nil {
		return err
	}
	if !strings.Contains(s.keyJSON, "BEGIN PGP PUBLIC KEY BLOCK") {
		return fmt.Errorf("public export: %s", s.keyJSON)
	}
	if err := runUI(s, "action.export_secret", s.exportSecret); err != nil {
		return err
	}
	if !strings.Contains(s.keyJSON, "BEGIN PGP PRIVATE KEY BLOCK") {
		return fmt.Errorf("secret export: %s", s.keyJSON)
	}
	if err := sameKeyring(ws, pub, sec); err != nil {
		return fmt.Errorf("export rewrote keyring: %w", err)
	}

	s.textInput = plain
	s.encRecipients = ""
	s.encPassword = ""
	s.encSign = false
	if err := runUI(s, "action.encrypt_text", s.encryptText); err != nil {
		return err
	}
	cipher := s.textOutput
	if !strings.Contains(cipher, "-----BEGIN PGP MESSAGE-----") {
		return fmt.Errorf("ciphertext: %q", cipher)
	}
	if err := runUI(s, "action.decrypt_text", s.decryptText); err != nil {
		return err
	}
	if s.textOutput != plain {
		return fmt.Errorf("decrypt got %q", s.textOutput)
	}
	s.textInput = cipher
	s.textVerifyInput = ""
	s.textOutput = ""
	if err := runUI(s, "action.decrypt_text", s.decryptText); err != nil {
		return err
	}
	if s.textOutput != plain {
		return fmt.Errorf("pasted decrypt got %q", s.textOutput)
	}

	s.textInput = plain
	s.textOutput = ""
	s.textVerifyInput = ""
	s.textVerifyDetached = ""
	s.textSigner = ""
	s.textMode = string(rnp.SignModeDetached)
	if err := runUI(s, "action.sign_text", s.signText); err != nil {
		return err
	}
	if !strings.Contains(s.textOutput, "-----BEGIN PGP SIGNATURE-----") {
		return fmt.Errorf("detached signature: %q", s.textOutput)
	}
	if err := runUI(s, "action.verify_text", s.verifyText); err != nil {
		return err
	}
	if !strings.Contains(s.textOutput, "[Math: VALID]") {
		return fmt.Errorf("detached verify: %q", s.textOutput)
	}

	s.textInput = plain
	s.textMode = string(rnp.SignModeCleartext)
	if err := runUI(s, "action.sign_text", s.signText); err != nil {
		return err
	}
	if !strings.Contains(s.textOutput, "-----BEGIN PGP SIGNED MESSAGE-----") {
		return fmt.Errorf("cleartext signature: %q", s.textOutput)
	}
	if err := runUI(s, "action.verify_text", s.verifyText); err != nil {
		return err
	}
	if !strings.Contains(s.textOutput, "[Math: VALID]") {
		return fmt.Errorf("cleartext verify: %q", s.textOutput)
	}
	if err := sameKeyring(ws, pub, sec); err != nil {
		return fmt.Errorf("text ops rewrote keyring: %w", err)
	}

	body := []byte("file payload")
	src := filepath.Join(ws, "plain.txt")
	enc := filepath.Join(ws, "plain.asc")
	out := filepath.Join(ws, "plain.out")
	sig := filepath.Join(ws, "plain.sig")
	if err := os.WriteFile(src, body, 0o600); err != nil {
		return err
	}
	s.fileSrc = src
	s.fileDst = enc
	s.encRecipients = ""
	s.encPassword = ""
	s.encSign = false
	if err := runUI(s, "action.encrypt_file", s.encryptFile); err != nil {
		return err
	}
	s.fileSrc = enc
	s.fileDst = out
	if err := runUI(s, "action.decrypt_file", s.decryptFile); err != nil {
		return err
	}
	got, err := os.ReadFile(out)
	if err != nil {
		return err
	}
	if !bytes.Equal(got, body) {
		return fmt.Errorf("file decrypt got %q", got)
	}
	s.fileSrc = src
	s.fileSig = sig
	s.textMode = string(rnp.SignModeDetached)
	s.textSigner = ""
	if err := runUI(s, "action.sign_file", s.signFile); err != nil {
		return err
	}
	s.fileDetached = true
	if err := runUI(s, "action.verify_file", s.verifyFile); err != nil {
		return err
	}
	if !strings.Contains(s.textOutput, "[Math: VALID]") {
		return fmt.Errorf("file verify: %q", s.textOutput)
	}
	s.packetInput = enc
	s.packetAsJSON = true
	if err := runUI(s, "action.dump_packets", s.dumpPackets); err != nil {
		return err
	}
	if !strings.Contains(s.packetOut, "header") || !strings.Contains(s.packetOut, "tag") {
		return fmt.Errorf("packet dump: %s", s.packetOut)
	}
	if err := sameKeyring(ws, pub, sec); err != nil {
		return fmt.Errorf("file ops rewrote keyring: %w", err)
	}

	for i := 0; i < 16; i++ {
		if len(s.keys) == 0 {
			return fmt.Errorf("no key left to delete")
		}
		n := len(s.keys)
		if err := runUI(s, "action.delete_key", s.deleteKey); err != nil {
			return err
		}
		if strings.Contains(s.lastErr, "input is empty") {
			return fmt.Errorf("delete reported empty input: %s", s.lastErr)
		}
		if len(s.keys) >= n {
			return fmt.Errorf("delete left %d keys, had %d", len(s.keys), n)
		}
		if len(s.keys) == 0 {
			if s.selected != "" {
				return fmt.Errorf("selected %q after last delete", s.selected)
			}
			return assertNoKeyring(ws)
		}
		if s.selected == "" {
			return fmt.Errorf("selected cleared while %d keys remain", len(s.keys))
		}
	}
	return fmt.Errorf("keys still present after repeated delete")
}

func runUI(s *uiState, title string, fn sessionAction) error {
	s.run(title, fn)
	if s.busy {
		return fmt.Errorf("%s: still busy", title)
	}
	if s.lastErr != "" {
		return fmt.Errorf("%s: %s", title, s.lastErr)
	}
	return nil
}

func assertNoKeyring(ws string) error {
	for _, name := range []string{"pubring.gpg", "secring.gpg"} {
		p := filepath.Join(ws, name)
		_, err := os.Stat(p)
		if err == nil {
			return fmt.Errorf("%s still exists", name)
		}
		if !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func readKeyring(ws string) (pub, sec []byte, err error) {
	pub, err = os.ReadFile(filepath.Join(ws, "pubring.gpg"))
	if err != nil {
		return nil, nil, err
	}
	sec, err = os.ReadFile(filepath.Join(ws, "secring.gpg"))
	if err != nil {
		return nil, nil, err
	}
	if len(pub) == 0 || len(sec) == 0 {
		return nil, nil, fmt.Errorf("empty keyring file")
	}
	return pub, sec, nil
}

func sameKeyring(ws string, pub, sec []byte) error {
	pub2, sec2, err := readKeyring(ws)
	if err != nil {
		return err
	}
	if !bytes.Equal(pub, pub2) || !bytes.Equal(sec, sec2) {
		return fmt.Errorf("keyring bytes changed")
	}
	return nil
}
