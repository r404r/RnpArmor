package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

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
	if got := len(items[0].Submenu); got != 6 {
		t.Fatalf("file submenu count = %d, want 6", got)
	}
	if items[0].Submenu[0].Label != Tr(LangEN, "btn.choose_folder") {
		t.Fatalf("file[0] label = %q", items[0].Submenu[0].Label)
	}
	if items[0].Submenu[1].Label != Tr(LangEN, "btn.open_session") {
		t.Fatalf("file[1] label = %q", items[0].Submenu[1].Label)
	}
	if items[0].Submenu[2].Label != Tr(LangEN, "btn.load_keyring") {
		t.Fatalf("file[2] label = %q", items[0].Submenu[2].Label)
	}
	if items[0].Submenu[3].Label != Tr(LangEN, "btn.save_keyring") {
		t.Fatalf("file[3] label = %q", items[0].Submenu[3].Label)
	}
	if items[0].Submenu[4].Type != mygo.MenuItemSeparator {
		t.Fatalf("file[4] type = %q, want separator", items[0].Submenu[4].Type)
	}
	if items[0].Submenu[5].Label != Tr(LangEN, "menu.file.quit") {
		t.Fatalf("file[5] label = %q", items[0].Submenu[5].Label)
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
		passphrase:   "",
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

	s.passphrase = "test-pass-1"
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
	var generated rnp.KeySummary
	for _, k := range s.keys {
		if k.Fingerprint == s.selected {
			generated = k
			break
		}
	}
	if !generated.IsPrimary || len(generated.Subkeys) == 0 {
		return fmt.Errorf("primary IsPrimary=%v subkeys=%d", generated.IsPrimary, len(generated.Subkeys))
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
	pubArmor := s.keyJSON
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

	if err := exercisePublicOnly(s, ws, pubArmor); err != nil {
		return err
	}
	revPath := filepath.Join(ws, "key.rev.asc")
	s.exportPath = revPath
	if err := runUI(s, "action.export_revocation", s.exportRevocation); err != nil {
		return err
	}
	revBytes, err := os.ReadFile(revPath)
	if err != nil {
		return err
	}
	if !strings.Contains(string(revBytes), "BEGIN PGP") {
		return fmt.Errorf("revocation file: %s", revBytes)
	}
	info, err := os.Stat(revPath)
	if err != nil {
		return err
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		return fmt.Errorf("revocation mode %o", info.Mode().Perm())
	}

	for i := 0; i < 16; i++ {
		if len(s.keys) == 0 {
			return fmt.Errorf("no key left to delete")
		}
		n := len(s.keys)
		if err := s.deleteKeyConfirmed(false); err != nil {
			return err
		}
		if len(s.keys) != n {
			return fmt.Errorf("unconfirmed delete changed %d keys to %d", n, len(s.keys))
		}
		if err := s.deleteKeyConfirmed(true); err != nil {
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

func exercisePublicOnly(s *uiState, ws, pubArmor string) error {
	if !strings.Contains(pubArmor, "BEGIN PGP PUBLIC KEY BLOCK") {
		return fmt.Errorf("public armor missing")
	}
	savedWS := s.workspace
	savedPass := s.passphrase
	savedSelected := s.selected
	dir := filepath.Join(ws, "pubonly")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	s.workspace = dir
	s.passphrase = ""
	s.importBytes = []byte(pubArmor)
	if err := runUI(s, "action.import_keys", s.importKeyBytes); err != nil {
		return fmt.Errorf("public import: %w", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "pubring.gpg")); err != nil {
		return fmt.Errorf("pubonly pubring: %w", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "secring.gpg")); !os.IsNotExist(err) {
		return fmt.Errorf("pubonly secring present: %v", err)
	}
	if err := runUI(s, "action.load_keyring", s.loadKeyring); err != nil {
		return fmt.Errorf("public reload: %w", err)
	}
	found := false
	for _, k := range s.keys {
		if isPrimaryRow(k) && !k.HasSecret && len(k.Subkeys) > 0 {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("public reload missing primary, keys=%d", len(s.keys))
	}
	s.workspace = savedWS
	s.passphrase = savedPass
	s.selected = savedSelected
	if err := runUI(s, "action.load_keyring", s.loadKeyring); err != nil {
		return fmt.Errorf("restore keyring: %w", err)
	}
	if s.selected == "" || s.passphrase != savedPass {
		return fmt.Errorf("restore selected=%q pass set=%t", s.selected, s.passphrase == savedPass)
	}
	return nil
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

func TestKeyManagerFilterSearchSortAndForms(t *testing.T) {
	if err := ValidateCatalog(); err != nil {
		t.Fatal(err)
	}
	keys := []string{
		"btn.new_key", "btn.import_file", "btn.import_text", "btn.choose_folder",
		"btn.copy_fingerprint", "btn.copy_public", "btn.export_public_asc",
		"btn.export_secret_asc", "btn.copy_secret", "btn.revocation", "btn.delete_key_ellipsis",
		"btn.generate", "btn.paste_clipboard", "btn.load_json",
		"seg.all", "seg.mine", "seg.public", "seg.id16", "seg.fp40",
		"col.type", "col.name", "col.key_id", "col.fingerprint", "col.algorithm", "col.status",
		"badge.secret", "badge.public", "badge.secret_public", "badge.valid", "badge.expired", "badge.revoked",
		"placeholder.search", "placeholder.unlock_pass", "placeholder.import_text",
		"err.name_or_email", "err.pass_empty", "err.pass_mismatch", "err.import_empty",
		"empty.keys", "empty.search", "empty.select", "empty.subkeys",
		"dlg.delete_secret", "dlg.delete_public", "dlg.choose_folder",
		"title.new_key", "title.import_text", "hint.secret_export",
		"exp.never", "exp.1y", "exp.2y", "exp.5y",
	}
	for _, k := range keys {
		for _, l := range []Lang{LangEN, LangZH, LangJA} {
			v := strings.TrimSpace(Tr(l, k))
			if v == "" || v == k {
				t.Fatalf("translation missing %s/%s: %q", k, l, v)
			}
		}
	}

	rows := []rnp.KeySummary{
		{
			Fingerprint: "AAAA BBBB CCCC DDDD EEEE FFFF 1111 2222 3333 4444",
			KeyID:       "1111222233334444",
			PrimaryUID:  "Alice <a@ex>",
			Algorithm:   "Ed25519",
			HasSecret:   true,
			IsPrimary:   true,
			CanCertify:  true,
		},
		{
			Fingerprint: "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB",
			KeyID:       "BBBBBBBBBBBBBBBB",
			PrimaryUID:  "Bob <b@ex>",
			Algorithm:   "RSA",
			HasSecret:   false,
			IsPrimary:   true,
		},
		{
			Fingerprint: "CCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC",
			KeyID:       "CCCCCCCCCCCCCCCC",
			PrimaryUID:  "Sub User",
			Algorithm:   "ECDH",
			IsSub:       true,
		},
		{
			Fingerprint: "DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD",
			KeyID:       "DDDDDDDDDDDDDDDD",
			PrimaryUID:  "Old <old@ex>",
			Algorithm:   "DSA",
			HasSecret:   true,
		},
	}
	all := filterPrimaryKeys(rows, keyFilterAll, "")
	if len(all) != 3 {
		t.Fatalf("all primaries = %d", len(all))
	}
	mine := filterPrimaryKeys(rows, keyFilterMine, "")
	if len(mine) != 2 || !mine[0].HasSecret || !mine[1].HasSecret {
		t.Fatalf("my keys = %+v", mine)
	}
	pub := filterPrimaryKeys(rows, keyFilterPublic, "")
	if len(pub) != 1 || pub[0].PrimaryUID != "Bob <b@ex>" {
		t.Fatalf("public = %+v", pub)
	}
	if got := filterPrimaryKeys(rows, keyFilterAll, "ALICE"); len(got) != 1 || got[0].Algorithm != "Ed25519" {
		t.Fatalf("uid search = %+v", got)
	}
	if got := filterPrimaryKeys(rows, keyFilterAll, "33334444"); len(got) != 1 {
		t.Fatalf("key id search = %+v", got)
	}
	if got := filterPrimaryKeys(rows, keyFilterAll, "bbbbbbbbbbbbbbbbbbbb"); len(got) != 1 || got[0].PrimaryUID != "Bob <b@ex>" {
		t.Fatalf("fingerprint search = %+v", got)
	}
	if got := filterPrimaryKeys(rows, keyFilterAll, "rsa"); len(got) != 1 {
		t.Fatalf("algorithm search = %+v", got)
	}
	if got := filterPrimaryKeys(rows, keyFilterAll, "no-such-key"); len(got) != 0 {
		t.Fatalf("empty search = %+v", got)
	}
	if got := filterPrimaryKeys(rows, keyFilterPublic, "alice"); len(got) != 0 {
		t.Fatalf("public+alice = %+v", got)
	}

	sorted := append([]rnp.KeySummary(nil), all...)
	sortKeyRows(sorted, ui.SortOrder{Column: "name"}, keyIDModeShort)
	if sorted[0].PrimaryUID != "Alice <a@ex>" || sorted[1].PrimaryUID != "Bob <b@ex>" || sorted[2].PrimaryUID != "Old <old@ex>" {
		t.Fatalf("name sort = %s, %s, %s", sorted[0].PrimaryUID, sorted[1].PrimaryUID, sorted[2].PrimaryUID)
	}
	sortKeyRows(sorted, ui.SortOrder{Column: "name", Descending: true}, keyIDModeShort)
	if sorted[0].PrimaryUID != "Old <old@ex>" || sorted[2].PrimaryUID != "Alice <a@ex>" {
		t.Fatalf("name desc = %s ... %s", sorted[0].PrimaryUID, sorted[2].PrimaryUID)
	}
	sortKeyRows(sorted, ui.SortOrder{}, keyIDModeShort)
	if sorted[0].PrimaryUID != "Old <old@ex>" {
		t.Fatalf("empty sort changed order to %s", sorted[0].PrimaryUID)
	}
	sortKeyRows(sorted, ui.SortOrder{Column: "type"}, keyIDModeShort)
	if sorted[0].HasSecret || !sorted[1].HasSecret {
		t.Fatalf("type sort = secret %v %v", sorted[0].HasSecret, sorted[1].HasSecret)
	}

	fp := "ABCD1234EFGH5678IJKL9012MNOP3456QRST7890"
	if got, want := groupFingerprint("ABCD 1234 EFGH 5678 IJKL 9012 MNOP 3456 QRST 7890"), "ABCD 1234 EFGH 5678 IJKL 9012 MNOP 3456 QRST 7890"; got != want {
		t.Fatalf("grouped = %q", got)
	}
	if got := groupFingerprint(fp); got != "ABCD 1234 EFGH 5678 IJKL 9012 MNOP 3456 QRST 7890" {
		t.Fatalf("group = %q", got)
	}
	if groupFingerprint("  ") != "" {
		t.Fatal("blank fingerprint")
	}
	if got := keyIDText(fp, "EFGH5678IJKL9012MNOP3456QRST7890", keyIDModeShort); got != "MNOP3456QRST7890" {
		t.Fatalf("short id = %q", got)
	}
	if got := keyIDText(fp, "", keyIDModeShort); got != "MNOP3456QRST7890" {
		t.Fatalf("short from fingerprint = %q", got)
	}
	if got := keyIDText("  "+fp+"  ", "ignored-short", keyIDModeFull); got != fp {
		t.Fatalf("full fingerprint = %q", got)
	}
	if got := keyIDText("", "ABCDEF0123456789", keyIDModeFull); got != "ABCDEF0123456789" {
		t.Fatalf("full fallback = %q", got)
	}
	title, width := idColumnSpec(keyIDModeShort)
	if title != "col.key_id" || width != 150 {
		t.Fatalf("short column = %s %v", title, width)
	}
	title, width = idColumnSpec(keyIDModeFull)
	if title != "col.fingerprint" || width != 340 {
		t.Fatalf("full column = %s %v", title, width)
	}

	base := rnp.KeyGenRequest{
		PrimaryAlg:   "EDDSA",
		PrimaryCurve: "Ed25519",
		SubkeyAlg:    "ECDH",
		SubkeyCurve:  "Curve25519",
		HashAlg:      "SHA256",
		SymmCipher:   "AES256",
	}
	if _, errKey := validateNewKey("  ", "", "secret", "secret", 0, base); errKey != "err.name_or_email" {
		t.Fatalf("empty name = %s", errKey)
	}
	if _, errKey := validateNewKey("Ada", "ada@ex", "", "secret", 0, base); errKey != "err.pass_empty" {
		t.Fatalf("empty pass = %s", errKey)
	}
	if _, errKey := validateNewKey("Ada", "ada@ex", "one", "two", 1, base); errKey != "err.pass_mismatch" {
		t.Fatalf("mismatch = %s", errKey)
	}
	const day = 24 * 60 * 60
	req, errKey := validateNewKey(" Ada ", " ada@ex ", "secret", "secret", 2, base)
	if errKey != "" {
		t.Fatal(errKey)
	}
	if req.UserID != "Ada <ada@ex>" || req.ExpirationSec != 2*365*day || req.PrimaryAlg != "EDDSA" || req.SubkeyCurve != "Curve25519" {
		t.Fatalf("request = %+v", req)
	}
	nameOnly, errKey := validateNewKey("Ada", "", "secret", "secret", 0, base)
	if errKey != "" || nameOnly.UserID != "Ada" || nameOnly.ExpirationSec != 0 {
		t.Fatalf("name only = %+v %s", nameOnly, errKey)
	}
	mailOnly, errKey := validateNewKey("", "ada@ex", "secret", "secret", 1, base)
	if errKey != "" || mailOnly.UserID != "ada@ex" || mailOnly.ExpirationSec != 365*day {
		t.Fatalf("mail only = %+v %s", mailOnly, errKey)
	}
	five, errKey := validateNewKey("Ada", "ada@ex", "secret", "secret", 3, base)
	if errKey != "" || five.ExpirationSec != 5*365*day {
		t.Fatalf("five years = %+v %s", five, errKey)
	}
	if expirySeconds(9) != 0 || expirySeconds(0) != 0 {
		t.Fatal("unknown expiry")
	}
	if !confirmButton(0) || confirmButton(1) {
		t.Fatal("confirm button")
	}
	if deleteWarningKey(true) != "dlg.delete_secret" || deleteWarningKey(false) != "dlg.delete_public" {
		t.Fatal("delete warning")
	}
	now := time.Date(2026, 3, 4, 15, 0, 0, 0, time.UTC)
	if formatKeyDate(0, "Never") != "Never" {
		t.Fatal("never date")
	}
	if got := formatKeyDate(uint32(now.Unix()), "Never"); got != "2026-03-04" {
		t.Fatalf("date = %s", got)
	}
	revoked := rnp.KeySummary{IsRevoked: true, ExpiresUnix: 1}
	if keyStatusCode(revoked, now) != "revoked" {
		t.Fatal("revoked")
	}
	expired := rnp.KeySummary{ExpiresUnix: uint32(now.Unix())}
	if keyStatusCode(expired, now) != "expired" {
		t.Fatal("expired")
	}
	if keyStatusCode(rnp.KeySummary{}, now) != "valid" {
		t.Fatal("valid")
	}
	if !isPrimaryRow(rnp.KeySummary{}) || !isPrimaryRow(rnp.KeySummary{IsPrimary: true, IsSub: true}) || isPrimaryRow(rnp.KeySummary{IsSub: true}) {
		t.Fatal("primary row")
	}
}

func TestStudioKeyPickersAndFileDialogPaths(t *testing.T) {
	if err := ValidateCatalog(); err != nil {
		t.Fatal(err)
	}
	newKeys := []string{
		"btn.add_recipient", "btn.clear_recipients", "btn.browse", "btn.save_as",
		"signer.use_selected", "empty.recipient_keys",
		"placeholder.recipients", "placeholder.signer", "placeholder.symm_pass",
		"dlg.browse_source", "dlg.save_destination", "dlg.browse_signature", "dlg.save_signature", "dlg.browse_packet",
		"filter.openpgp", "filter.all_files",
	}
	for _, k := range newKeys {
		for _, l := range []Lang{LangEN, LangZH, LangJA} {
			v := strings.TrimSpace(Tr(l, k))
			if v == "" || v == k {
				t.Fatalf("translation missing %s/%s: %q", k, l, v)
			}
		}
	}

	const aliceFP = "AAAAAAAABBBBBBBBCCCCCCCCDDDDDDDDEEEEEEEE"
	const bobFP = "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB"
	const subFP = "CCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCCC"
	const oldFP = "DDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDDD"
	const bareFP = "EEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEEE"
	keys := []rnp.KeySummary{
		{
			Fingerprint: "AAAA AAAA BBBB BBBB CCCC CCCC DDDD DDDD EEEE EEEE",
			KeyID:       "1111222233334444",
			PrimaryUID:  "Alice <alice@example.com>",
			Algorithm:   "Ed25519",
			HasSecret:   true,
			IsPrimary:   true,
		},
		{
			Fingerprint: bobFP,
			KeyID:       "BBBBBBBBBBBBBBBB",
			PrimaryUID:  "Bob <bob@example.com>",
			Algorithm:   "RSA",
			HasSecret:   false,
			IsPrimary:   true,
		},
		{
			Fingerprint: subFP,
			KeyID:       "CCCCCCCCCCCCCCCC",
			PrimaryUID:  "Sub User",
			Algorithm:   "ECDH",
			HasSecret:   true,
			IsSub:       true,
		},
		{
			Fingerprint: oldFP,
			KeyID:       "DDDDDDDDDDDDDDDD",
			PrimaryUID:  "Old <old@example.com>",
			Algorithm:   "DSA",
			HasSecret:   true,
		},
		{
			Fingerprint: bareFP,
			KeyID:       "EEEEEEEEEEEEEEEE",
			Algorithm:   "ECDSA",
			IsPrimary:   true,
		},
	}

	labels, fps := recipientOptions(keys)
	if len(labels) != 4 || len(fps) != 4 {
		t.Fatalf("recipient options = %d labels, %d fps", len(labels), len(fps))
	}
	if labels[0] != "Alice <alice@example.com> [1111222233334444]" {
		t.Fatalf("alice label = %q", labels[0])
	}
	if fps[0] != aliceFP {
		t.Fatalf("alice fp = %q", fps[0])
	}
	if labels[1] != "Bob <bob@example.com> [BBBBBBBBBBBBBBBB]" || fps[1] != bobFP {
		t.Fatalf("bob = %q %q", labels[1], fps[1])
	}
	for _, fp := range fps {
		if fp == subFP {
			t.Fatal("subkey listed as recipient")
		}
	}
	if labels[3] != bareFP+" ECDSA" || fps[3] != bareFP {
		t.Fatalf("no uid = %q %q", labels[3], fps[3])
	}
	noAlg := keyPickerLabel(rnp.KeySummary{Fingerprint: bareFP, IsPrimary: true})
	if noAlg != bareFP {
		t.Fatalf("fingerprint only = %q", noAlg)
	}
	idOnly := keyPickerLabel(rnp.KeySummary{KeyID: "0123456789ABCDEF", IsPrimary: true})
	if idOnly != "0123456789ABCDEF" {
		t.Fatalf("id only = %q", idOnly)
	}

	list := appendRecipientFingerprint("", "  ab cd ")
	if list != "abcd" {
		t.Fatalf("first append = %q", list)
	}
	if got := appendRecipientFingerprint(list, "AB CD"); got != list {
		t.Fatalf("case-insensitive dup = %q", got)
	}
	spaced := "ABCD 1234 EFGH 5678"
	if got := appendRecipientFingerprint(spaced, "abcd1234efgh5678"); got != spaced {
		t.Fatalf("spaced dup rewrote list: %q", got)
	}
	if got := appendRecipientFingerprint(list, ""); got != list {
		t.Fatalf("empty fp = %q", got)
	}
	if got := appendRecipientFingerprint(list, "   "); got != list {
		t.Fatalf("blank fp = %q", got)
	}
	next := appendRecipientFingerprint(" AbCd , EF ", "1111")
	if next != "AbCd, EF, 1111" {
		t.Fatalf("second recipient = %q", next)
	}
	if clearRecipientList(next) != "" {
		t.Fatal("clear")
	}
	cleared := clearRecipientList(next)
	got := recipientFingerprints(actionInput{encRecipients: cleared, selected: "SELECTED"})
	if len(got) != 1 || got[0] != "SELECTED" {
		t.Fatalf("cleared fallback = %#v", got)
	}

	def := "Use selected key (default)"
	sLabels, sFPs := signerOptions(keys, def)
	if len(sLabels) != 3 || sLabels[0] != def || sFPs[0] != "" {
		t.Fatalf("signer head = %q %q len %d", sLabels[0], sFPs[0], len(sLabels))
	}
	if sLabels[1] != labels[0] || sFPs[1] != aliceFP {
		t.Fatalf("signer alice = %q %q", sLabels[1], sFPs[1])
	}
	if sLabels[2] != "Old <old@example.com> [DDDDDDDDDDDDDDDD]" || sFPs[2] != oldFP {
		t.Fatalf("signer old = %q %q", sLabels[2], sFPs[2])
	}
	for _, fp := range sFPs {
		if fp == bobFP || fp == subFP || fp == bareFP {
			t.Fatalf("signer listed %s", fp)
		}
	}

	manual := "deadbeef"
	prev := signerLabelFor(manual, sLabels, sFPs)
	if prev != def {
		t.Fatalf("unknown signer label = %q", prev)
	}
	if got := applySignerPick(prev, prev, sLabels, sFPs, manual); got != manual {
		t.Fatalf("manual override wiped: %q", got)
	}
	if got := applySignerPick(sLabels[1], prev, sLabels, sFPs, manual); got != aliceFP {
		t.Fatalf("pick alice = %q", got)
	}
	mixed := "aaaaaaaabbbbbbbbccccccccddddddddeeeeeeee"
	if signerLabelFor(mixed, sLabels, sFPs) != sLabels[1] {
		t.Fatal("signer match is case-sensitive")
	}
	if got := applySignerPick(def, sLabels[1], sLabels, sFPs, aliceFP); got != "" {
		t.Fatalf("pick default = %q", got)
	}
	if signerLabelFor("", sLabels, sFPs) != def {
		t.Fatal("empty signer")
	}
	if signerLabelFor("  ", nil, nil) != "" {
		t.Fatal("no signer options")
	}

	ws := filepath.Join("work", "space")
	sibling := filepath.Join("docs", "in.txt")
	if got := dialogDefaultPath("", sibling, ws); got != filepath.Dir(sibling) {
		t.Fatalf("sibling dir = %q", got)
	}
	if got := dialogDefaultPath("", "in.txt", ws); got != ws {
		t.Fatalf("bare sibling = %q", got)
	}
	if got := dialogDefaultPath("  ", "  ", "  "+ws+"  "); got != ws {
		t.Fatalf("workspace = %q", got)
	}
	current := filepath.Join("out", "dst.asc")
	if got := dialogDefaultPath("  "+current+" ", sibling, ws); got != current {
		t.Fatalf("current wins = %q", got)
	}

	filters := openPGPFileFilters("OpenPGP", "All files")
	if len(filters) != 2 || filters[0].Name != "OpenPGP" || filters[1].Name != "All files" {
		t.Fatalf("filters = %+v", filters)
	}
	seenAsc, seenAll := false, false
	for _, ext := range filters[0].Extensions {
		if ext == "asc" {
			seenAsc = true
		}
	}
	for _, ext := range filters[1].Extensions {
		if ext == "*" {
			seenAll = true
		}
	}
	if !seenAsc || !seenAll {
		t.Fatalf("filter extensions = %+v %+v", filters[0].Extensions, filters[1].Extensions)
	}
	for _, key := range []string{"dlg.browse_source", "dlg.save_destination"} {
		df := fileDialogFilters(key, "OpenPGP", "All files")
		if len(df) != 2 || df[0].Name != "All files" || df[1].Name != "OpenPGP" {
			t.Fatalf("fileDialogFilters(%s) = %+v", key, df)
		}
	}
	for _, key := range []string{"dlg.browse_signature", "dlg.save_signature", "dlg.browse_packet"} {
		df := fileDialogFilters(key, "OpenPGP", "All files")
		if len(df) != 2 || df[0].Name != "OpenPGP" || df[1].Name != "All files" {
			t.Fatalf("fileDialogFilters(%s) = %+v", key, df)
		}
	}
}

func TestKeygenPresetsSyncAndTips(t *testing.T) {
	if err := ValidateCatalog(); err != nil {
		t.Fatal(err)
	}
	keys := []string{
		"label.alg_preset", "label.enc_subkey",
		"preset.modern", "preset.rsa4096", "preset.rsa3072", "preset.nist_p256", "preset.custom",
		"preset.modern.desc", "preset.rsa4096.desc", "preset.rsa3072.desc", "preset.nist_p256.desc", "preset.custom.desc",
		"tip.preset.roles",
		"tip.primary.eddsa", "tip.primary.rsa", "tip.primary.ecdsa", "tip.primary.dsa", "tip.primary.role",
		"tip.subkey.ecdh", "tip.subkey.rsa", "tip.subkey.elgamal", "tip.subkey.role",
		"tip.crypto.desc", "tip.crypto.role",
	}
	seen := map[string]bool{}
	for _, k := range keys {
		if seen[k] {
			t.Fatalf("duplicate key %s", k)
		}
		seen[k] = true
		for _, l := range []Lang{LangEN, LangZH, LangJA} {
			v := strings.TrimSpace(Tr(l, k))
			if v == "" || v == k {
				t.Fatalf("translation missing %s/%s: %q", k, l, v)
			}
		}
	}
	for k, m := range i18nCatalog {
		zh := m[LangZH]
		if strings.Contains(zh, "主键") || strings.Contains(zh, "副键") {
			t.Fatalf("simplified term %s: %s", k, zh)
		}
	}
	if Tr(LangEN, "label.primary") != "Primary Key" || Tr(LangZH, "label.primary") != "主密钥" || Tr(LangJA, "label.primary") != "主鍵" {
		t.Fatal("primary term")
	}
	if Tr(LangEN, "label.subkey") != "Subkey" || Tr(LangZH, "label.subkey") != "子密钥" || Tr(LangJA, "label.subkey") != "副鍵" {
		t.Fatal("subkey term")
	}
	if Tr(LangZH, "label.enc_subkey") != "加密子密钥" || Tr(LangJA, "label.enc_subkey") != "暗号化副鍵" {
		t.Fatal("encryption subkey term")
	}

	base := rnp.KeyGenRequest{UserID: "Ada", ExpirationSec: 9, S2KIterations: 3}
	modern := applyKeyPreset(presetModern, base)
	if modern.UserID != "Ada" || modern.ExpirationSec != 9 || modern.S2KIterations != 3 {
		t.Fatalf("identity wiped: %+v", modern)
	}
	if modern.PrimaryAlg != "EDDSA" || modern.PrimaryCurve != "Ed25519" || modern.PrimaryBits != 0 {
		t.Fatalf("modern primary: %+v", modern)
	}
	if modern.SubkeyAlg != "ECDH" || modern.SubkeyCurve != "Curve25519" || modern.SubkeyBits != 0 {
		t.Fatalf("modern subkey: %+v", modern)
	}
	if modern.HashAlg != "SHA256" || modern.SymmCipher != "AES256" {
		t.Fatalf("modern crypto: %+v", modern)
	}
	rsa4096 := applyKeyPreset(presetRSA4096, modern)
	if rsa4096.PrimaryAlg != "RSA" || rsa4096.PrimaryBits != 4096 || rsa4096.PrimaryCurve != "" {
		t.Fatalf("rsa4096 primary: %+v", rsa4096)
	}
	if rsa4096.SubkeyAlg != "RSA" || rsa4096.SubkeyBits != 4096 || rsa4096.SubkeyCurve != "" || rsa4096.UserID != "Ada" {
		t.Fatalf("rsa4096 subkey: %+v", rsa4096)
	}
	rsa3072 := applyKeyPreset(presetRSA3072, modern)
	if rsa3072.PrimaryBits != 3072 || rsa3072.SubkeyBits != 3072 || rsa3072.PrimaryCurve != "" || rsa3072.SubkeyAlg != "RSA" {
		t.Fatalf("rsa3072: %+v", rsa3072)
	}
	nist := applyKeyPreset(presetNISTP256, modern)
	if nist.PrimaryAlg != "ECDSA" || nist.PrimaryCurve != "NIST P-256" || nist.PrimaryBits != 0 {
		t.Fatalf("nist primary: %+v", nist)
	}
	if nist.SubkeyAlg != "ECDH" || nist.SubkeyCurve != "NIST P-256" || nist.SubkeyBits != 0 {
		t.Fatalf("nist subkey: %+v", nist)
	}
	if got := applyKeyPreset("nope", base); !keygenSpecEqual(got, modern) {
		t.Fatalf("unknown preset: %+v", got)
	}
	customIn := modern
	customIn.PrimaryAlg = "RSA"
	customIn.PrimaryBits = 0
	customIn.PrimaryCurve = "Ed25519"
	custom := applyKeyPreset(presetCustom, customIn)
	if custom.PrimaryAlg != "RSA" || custom.PrimaryBits != 4096 || custom.PrimaryCurve != "" || custom.UserID != "Ada" {
		t.Fatalf("custom sync: %+v", custom)
	}
	if presetForRequest(modern) != presetModern || presetForRequest(rsa4096) != presetRSA4096 || presetForRequest(rsa3072) != presetRSA3072 || presetForRequest(nist) != presetNISTP256 {
		t.Fatal("preset match")
	}
	tweaked := modern
	tweaked.HashAlg = "SHA512"
	if presetForRequest(tweaked) != presetCustom {
		t.Fatal("tweaked still matches a preset")
	}
	if id, open := notePresetDrift(presetModern, modern); id != presetModern || open {
		t.Fatalf("stable modern %s %v", id, open)
	}
	if id, open := notePresetDrift(presetModern, tweaked); id != presetCustom || !open {
		t.Fatalf("drift %s %v", id, open)
	}
	if id, open := notePresetDrift(presetCustom, tweaked); id != presetCustom || open {
		t.Fatalf("stay custom %s %v", id, open)
	}

	eddsa := syncPrimaryAlgParams(rnp.KeyGenRequest{PrimaryAlg: "EDDSA", PrimaryCurve: "NIST P-256", PrimaryBits: 4096})
	if eddsa.PrimaryCurve != "Ed25519" || eddsa.PrimaryBits != 0 || eddsa.PrimaryAlg != "EDDSA" {
		t.Fatalf("eddsa sync: %+v", eddsa)
	}
	ecdsa := syncPrimaryAlgParams(rnp.KeyGenRequest{PrimaryAlg: "ECDSA", PrimaryCurve: "Ed25519", PrimaryBits: 256})
	if ecdsa.PrimaryCurve != "NIST P-256" || ecdsa.PrimaryBits != 0 {
		t.Fatalf("ecdsa kept %s bits %d", ecdsa.PrimaryCurve, ecdsa.PrimaryBits)
	}
	keptCurve := syncPrimaryAlgParams(rnp.KeyGenRequest{PrimaryAlg: "ECDSA", PrimaryCurve: "NIST P-384"})
	if keptCurve.PrimaryCurve != "NIST P-384" {
		t.Fatalf("curve reset: %s", keptCurve.PrimaryCurve)
	}
	rsa := syncPrimaryAlgParams(rnp.KeyGenRequest{PrimaryAlg: "rsa", PrimaryBits: 0, PrimaryCurve: "Ed25519"})
	if rsa.PrimaryAlg != "RSA" || rsa.PrimaryBits != 4096 || rsa.PrimaryCurve != "" {
		t.Fatalf("rsa sync: %+v", rsa)
	}
	dsa := syncPrimaryAlgParams(rnp.KeyGenRequest{PrimaryAlg: "DSA", PrimaryBits: 4096, PrimaryCurve: "NIST P-256"})
	if dsa.PrimaryBits != 3072 || dsa.PrimaryCurve != "" {
		t.Fatalf("dsa sync: %+v", dsa)
	}
	dsaOK := syncPrimaryAlgParams(rnp.KeyGenRequest{PrimaryAlg: "DSA", PrimaryBits: 2048})
	if dsaOK.PrimaryBits != 2048 {
		t.Fatalf("dsa 2048: %d", dsaOK.PrimaryBits)
	}

	ecdh := syncSubkeyAlgParams(rnp.KeyGenRequest{SubkeyAlg: "ECDH", SubkeyCurve: "Ed25519", SubkeyBits: 4096})
	if ecdh.SubkeyAlg != "ECDH" || ecdh.SubkeyCurve != "Curve25519" || ecdh.SubkeyBits != 0 {
		t.Fatalf("ecdh sync: %+v", ecdh)
	}
	ecdhKeep := syncSubkeyAlgParams(rnp.KeyGenRequest{SubkeyAlg: "ECDH", SubkeyCurve: "NIST P-521"})
	if ecdhKeep.SubkeyCurve != "NIST P-521" || ecdhKeep.SubkeyBits != 0 {
		t.Fatalf("ecdh keep: %+v", ecdhKeep)
	}
	subRSA := syncSubkeyAlgParams(rnp.KeyGenRequest{SubkeyAlg: "", SubkeyBits: 0, SubkeyCurve: "Curve25519"})
	if subRSA.SubkeyAlg != "ECDH" || subRSA.SubkeyCurve != "Curve25519" {
		t.Fatalf("empty subkey: %+v", subRSA)
	}
	subRSA = syncSubkeyAlgParams(rnp.KeyGenRequest{SubkeyAlg: "RSA", SubkeyBits: 0, SubkeyCurve: "Curve25519"})
	if subRSA.SubkeyBits != 4096 || subRSA.SubkeyCurve != "" || subRSA.SubkeyAlg != "RSA" {
		t.Fatalf("sub rsa: %+v", subRSA)
	}
	elg := syncSubkeyAlgParams(rnp.KeyGenRequest{SubkeyAlg: "ELGAMAL", SubkeyBits: 0, SubkeyCurve: "NIST P-256"})
	if elg.SubkeyBits != 3072 || elg.SubkeyCurve != "" {
		t.Fatalf("elgamal default: %+v", elg)
	}
	elg = syncSubkeyAlgParams(rnp.KeyGenRequest{SubkeyAlg: "ELGAMAL", SubkeyBits: 4096})
	if elg.SubkeyBits != 4096 {
		t.Fatalf("elgamal keep: %d", elg.SubkeyBits)
	}

	if primaryAlgTipKey(" eddsa ") != "tip.primary.eddsa" || primaryAlgTipKey("") != "tip.primary.eddsa" {
		t.Fatal("eddsa tip")
	}
	if primaryAlgTipKey("rsa") != "tip.primary.rsa" || primaryAlgTipKey("ECDSA") != "tip.primary.ecdsa" || primaryAlgTipKey("DSA") != "tip.primary.dsa" {
		t.Fatal("primary tips")
	}
	if subkeyAlgTipKey("ecdh") != "tip.subkey.ecdh" || subkeyAlgTipKey("") != "tip.subkey.ecdh" {
		t.Fatal("ecdh tip")
	}
	if subkeyAlgTipKey("RSA") != "tip.subkey.rsa" || subkeyAlgTipKey("ELGAMAL") != "tip.subkey.elgamal" {
		t.Fatal("subkey tips")
	}
	if presetDescKey(presetModern) != "preset.modern.desc" || presetDescKey(presetRSA4096) != "preset.rsa4096.desc" || presetDescKey(presetRSA3072) != "preset.rsa3072.desc" {
		t.Fatal("preset desc")
	}
	if presetDescKey(presetNISTP256) != "preset.nist_p256.desc" || presetDescKey(presetCustom) != "preset.custom.desc" || presetDescKey("nope") != "preset.modern.desc" {
		t.Fatal("preset desc fallback")
	}
	w, h := newKeyWindowSize()
	if w != 560 || h != 640 {
		t.Fatalf("window %d x %d", w, h)
	}
	labels := make([]string, 0, 5)
	for _, id := range presetIDs() {
		labels = append(labels, Tr(LangEN, presetLabelKey(id)))
	}
	for i := range labels {
		for j := i + 1; j < len(labels); j++ {
			if labels[i] == labels[j] {
				t.Fatalf("preset labels collide: %s", labels[i])
			}
		}
	}
}
