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
	"github.com/egoist/mygo/plugins/glass"
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

func TestMenuTemplateForOS(t *testing.T) {
	s := &uiState{langIdx: LangIndex(LangEN)}

	for _, goos := range []string{"linux", "windows"} {
		items := s.menuTemplateForOS(goos)
		if got := len(items); got != 4 {
			t.Fatalf("%s top menu count = %d, want 4", goos, got)
		}
		if items[0].Label != Tr(LangEN, "menu.file") {
			t.Fatalf("%s menu[0] = %q, want File", goos, items[0].Label)
		}
		quit := items[0].Submenu[5]
		if quit.Accelerator != "" || quit.Role != "" || quit.Click == nil {
			t.Fatalf("%s file quit accel=%q role=%q click-nil=%v", goos, quit.Accelerator, quit.Role, quit.Click == nil)
		}
	}

	items := s.menuTemplateForOS("darwin")
	if got := len(items); got != 5 {
		t.Fatalf("darwin top menu count = %d, want 5", got)
	}
	wantLabels := []string{
		Tr(LangEN, "title.app"),
		Tr(LangEN, "menu.file"),
		Tr(LangEN, "menu.workspace"),
		Tr(LangEN, "menu.language"),
		Tr(LangEN, "menu.help"),
	}
	for i, want := range wantLabels {
		if items[i].Label != want {
			t.Fatalf("darwin menu[%d] = %q, want %q", i, items[i].Label, want)
		}
	}

	app := items[0].Submenu
	if got := len(app); got != 9 {
		t.Fatalf("darwin app menu count = %d, want 9", got)
	}
	if app[0].Label != Tr(LangEN, "menu.help.about") || app[0].Role != "" || app[0].Click == nil {
		t.Fatalf("about = label %q role %q click-nil %v", app[0].Label, app[0].Role, app[0].Click == nil)
	}
	if app[1].Type != mygo.MenuItemSeparator || app[3].Type != mygo.MenuItemSeparator || app[7].Type != mygo.MenuItemSeparator {
		t.Fatalf("app separators = %q %q %q", app[1].Type, app[3].Type, app[7].Type)
	}
	if app[2].Role != mygo.RoleServices || app[2].Submenu == nil {
		t.Fatalf("services role=%q submenu-nil=%v", app[2].Role, app[2].Submenu == nil)
	}
	if len(app[2].Submenu) != 0 {
		t.Fatalf("services submenu len = %d, want 0", len(app[2].Submenu))
	}
	if app[4].Role != mygo.RoleHide || app[5].Role != mygo.RoleHideOthers || app[6].Role != mygo.RoleUnhide {
		t.Fatalf("hide roles = %q %q %q", app[4].Role, app[5].Role, app[6].Role)
	}
	quit := app[8]
	if quit.Label != Tr(LangEN, "menu.file.quit") || quit.Role != mygo.RoleQuit || quit.Accelerator != "Cmd+Q" || quit.Click != nil {
		t.Fatalf("app quit label=%q role=%q accel=%q click-nil=%v", quit.Label, quit.Role, quit.Accelerator, quit.Click == nil)
	}

	fileQuit := items[1].Submenu[5]
	if fileQuit.Label != Tr(LangEN, "menu.file.quit") || fileQuit.Accelerator == "Cmd+Q" || fileQuit.Role == mygo.RoleQuit || fileQuit.Click == nil {
		t.Fatalf("darwin file quit label=%q role=%q accel=%q click-nil=%v", fileQuit.Label, fileQuit.Role, fileQuit.Accelerator, fileQuit.Click == nil)
	}
	cmdQ := 0
	var walk func([]*mygo.MenuItem)
	walk = func(menu []*mygo.MenuItem) {
		for _, it := range menu {
			if it.Accelerator == "Cmd+Q" {
				cmdQ++
			}
			walk(it.Submenu)
		}
	}
	walk(items)
	if cmdQ != 1 {
		t.Fatalf("Cmd+Q bindings = %d, want 1", cmdQ)
	}
}

func TestQuitConfirmState(t *testing.T) {
	live := &mygo.Window{}
	s := &uiState{langIdx: LangIndex(LangEN), mainWin: live}
	if !s.shouldInterceptQuit("darwin", false) {
		t.Fatal("darwin with a live main window should intercept")
	}
	if s.shouldInterceptQuit("darwin", true) {
		t.Fatal("session ending must quit")
	}
	s.quitConfirmed = true
	if s.shouldInterceptQuit("darwin", false) {
		t.Fatal("confirmed quit must pass")
	}
	s.quitConfirmed = false
	s.mainClosed = true
	if s.shouldInterceptQuit("darwin", false) {
		t.Fatal("closed main window must pass")
	}
	s.mainClosed = false
	if s.shouldInterceptQuit("linux", false) || s.shouldInterceptQuit("windows", false) {
		t.Fatal("non-darwin OnBeforeQuit must not block SIGTERM")
	}
	s.mainWin = nil
	if s.shouldInterceptQuit("darwin", false) {
		t.Fatal("nil main window must pass")
	}

	// handleBeforeQuit follows the host OS. On darwin it would show the
	// dialog; on Linux it must not, or SIGTERM from the screenshot script hangs.
	if runtime.GOOS != "darwin" {
		s.mainWin = live
		e := &mygo.QuitEvent{}
		s.handleBeforeQuit(e)
		if e.DefaultPrevented() || s.quitConfirmOpen {
			t.Fatal("non-darwin OnBeforeQuit blocked quit")
		}
	}

	s.handleQuitCanceled()
	s.quitConfirmed = true
	s.handleQuitCanceled()
	if s.quitConfirmed {
		t.Fatal("handleQuitCanceled left quitConfirmed set")
	}

	for _, lang := range []Lang{LangEN, LangZH, LangJA} {
		s.langIdx = LangIndex(lang)
		s.busy = false
		if got, want := s.quitConfirmMessage(), Tr(lang, "dlg.quit_msg"); got != want {
			t.Fatalf("%s idle message = %q, want %q", lang, got, want)
		}
		s.mu.Lock()
		s.busy = true
		s.mu.Unlock()
		if got, want := s.quitConfirmMessage(), Tr(lang, "dlg.quit_busy_msg"); got != want {
			t.Fatalf("%s busy message = %q, want %q", lang, got, want)
		}
		s.mu.Lock()
		s.busy = false
		s.mu.Unlock()
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
	if got := resolveDecryptInput(msg, "plain", "hidden", other); got != msg {
		t.Fatalf("decrypt box wins: %q", got)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveDecryptInput("", tc.in, tc.verify, tc.out)
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
	if len(got) != 0 {
		t.Fatalf("selected must not be used %#v", got)
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
	case <-time.After(45 * time.Second):
		t.Fatal("ui actions timed out")
	}
}

func TestUIStateEmptyPassphraseLifecycle(t *testing.T) {
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
		errCh <- exerciseEmptyPassphraseUI(lib, ws)
	}()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(45 * time.Second):
		t.Fatal("empty passphrase ui timed out")
	}
}

func exerciseEmptyPassphraseUI(lib, ws string) error {
	const plain = "hello unprotected"
	s := &uiState{
		libPath:     lib,
		workspace:   ws,
		passphrase:  "",
		encCipher:   "AES256",
		encCompress: "zlib",
		textMode:    string(rnp.SignModeDetached),
		textInput:   plain,
		langIdx:     LangIndex(LangEN),
		keyReq: rnp.KeyGenRequest{
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
	req, errKey := validateNewKey("Open", "open@example.com", "", "", 0, s.keyReq)
	if errKey != "" || !req.Unprotected {
		return fmt.Errorf("validate empty key: %s %+v", errKey, req)
	}
	kept := s.passphrase
	s.newKeyPass = ""
	s.newKeyPass2 = ""
	s.keyReq = req
	if err := runUI(s, "action.generate_key", s.generateKey); err != nil {
		return err
	}
	if s.passphrase != kept {
		return fmt.Errorf("empty generate changed session passphrase")
	}
	parent, ok := primaryByFP(s.keys, s.selected)
	if !ok || !parent.HasSecret || parent.IsProtected || parent.IsLocked || len(parent.Subkeys) == 0 {
		return fmt.Errorf("generated %+v", parent)
	}
	for _, sk := range parent.Subkeys {
		if !sk.HasSecret || sk.IsProtected || sk.IsLocked {
			return fmt.Errorf("generated subkey %+v", sk)
		}
	}

	s.textInput = plain
	s.encRecipients = s.selected
	s.encPassword = ""
	s.encSign = false
	if err := runUI(s, "action.encrypt_text", s.encryptText); err != nil {
		return err
	}
	if !strings.Contains(s.textOutput, "-----BEGIN PGP MESSAGE-----") {
		return fmt.Errorf("ciphertext: %q", s.textOutput)
	}
	if err := runUI(s, "action.decrypt_text", s.decryptText); err != nil {
		return err
	}
	if s.textOutput != plain {
		return fmt.Errorf("decrypt got %q", s.textOutput)
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
		return fmt.Errorf("signature: %q", s.textOutput)
	}
	if err := runUI(s, "action.verify_text", s.verifyText); err != nil {
		return err
	}
	if !strings.Contains(s.textOutput, "[Math: VALID]") {
		return fmt.Errorf("verify: %q", s.textOutput)
	}
	if err := runUI(s, "action.export_secret", s.exportSecret); err != nil {
		return err
	}
	if !strings.Contains(s.keyJSON, "BEGIN PGP PRIVATE KEY BLOCK") {
		return fmt.Errorf("secret export: %s", s.keyJSON)
	}

	subReq, errKey := validateAddSubkey("sign", "EDDSA", "", "SHA256", "AES256", 0, 1, "", s.passphrase, s.primaryProtected())
	if errKey != "" || !subReq.Unprotected {
		return fmt.Errorf("validate add subkey: %s %+v protected=%v", errKey, subReq, s.primaryProtected())
	}
	before := len(parent.Subkeys)
	s.subReq = subReq
	if err := runUI(s, "action.add_subkey", s.addSubkey); err != nil {
		return err
	}
	if s.passphrase != kept {
		return fmt.Errorf("empty add subkey changed session passphrase")
	}
	parent, ok = primaryByFP(s.keys, s.selected)
	if !ok || len(parent.Subkeys) != before+1 {
		return fmt.Errorf("subkeys after add ok=%v n=%d", ok, len(parent.Subkeys))
	}
	added, ok := subByFP(s.keys, s.selected, s.selectedSubkey)
	if !ok || !added.HasSecret || added.IsProtected || added.IsLocked || !added.CanSign {
		return fmt.Errorf("added %+v", added)
	}
	s.targetFP = added.Fingerprint
	s.expirySeconds = expirationFromNow(added.CreatedUnix, time.Now().UTC(), 1)
	if s.expirySeconds == 0 {
		return fmt.Errorf("expiry is 0")
	}
	if err := runUI(s, "action.set_key_expiry", s.setKeyExpiry); err != nil {
		return err
	}
	updated, ok := subByFP(s.keys, s.selected, added.Fingerprint)
	wantExp := added.CreatedUnix + s.expirySeconds
	if added.CreatedUnix == 0 {
		wantExp = s.expirySeconds
	}
	if !ok || updated.ExpiresUnix != wantExp || updated.IsProtected {
		return fmt.Errorf("expiry %+v want %d", updated, wantExp)
	}
	s.targetFP = added.Fingerprint
	s.revokeCode = "retired"
	s.revokeReason = "empty-pass"
	if err := runUI(s, "action.revoke_key", s.revokeKey); err != nil {
		return err
	}
	updated, ok = subByFP(s.keys, s.selected, added.Fingerprint)
	parent, _ = primaryByFP(s.keys, s.selected)
	if !ok || !updated.IsRevoked || parent.IsRevoked {
		return fmt.Errorf("revoke sub=%+v primaryRevoked=%v", updated, parent.IsRevoked)
	}
	s.targetFP = added.Fingerprint
	if err := runUI(s, "action.delete_subkey", s.deleteSubkey); err != nil {
		return err
	}
	if _, ok := subByFP(s.keys, s.selected, added.Fingerprint); ok {
		return fmt.Errorf("deleted subkey still listed")
	}
	parent, ok = primaryByFP(s.keys, s.selected)
	if !ok || parent.IsRevoked || len(parent.Subkeys) != before || parent.IsProtected {
		return fmt.Errorf("primary after delete %+v", parent)
	}
	return nil
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
	s.encRecipients = s.selected
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
	s.encRecipients = s.selected
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
	if !strings.Contains(s.fileOutput, "[Math: VALID]") {
		return fmt.Errorf("file verify: %q", s.fileOutput)
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
	if err := exerciseSubkeyLifecycle(s); err != nil {
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
		"badge.secret", "badge.public", "badge.secret_public", "badge.unprotected", "badge.valid", "badge.expired", "badge.revoked",
		"placeholder.new_pass", "placeholder.confirm_pass", "desc.pass_unprotected", "desc.pass_protected",
		"tip.show_passphrase", "tip.hide_passphrase",
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
	if _, errKey := validateNewKey("Ada", "ada@ex", "", "secret", 0, base); errKey != "err.pass_mismatch" {
		t.Fatalf("confirm filled = %s", errKey)
	}
	if _, errKey := validateNewKey("Ada", "ada@ex", "secret", "", 0, base); errKey != "err.pass_mismatch" {
		t.Fatalf("pass filled = %s", errKey)
	}
	if _, errKey := validateNewKey("Ada", "ada@ex", "one", "two", 1, base); errKey != "err.pass_mismatch" {
		t.Fatalf("mismatch = %s", errKey)
	}
	openReq, errKey := validateNewKey("Ada", "ada@ex", "", "", 0, base)
	if errKey != "" || !openReq.Unprotected || openReq.UserID != "Ada <ada@ex>" {
		t.Fatalf("empty pass = %+v %s", openReq, errKey)
	}
	held := &uiState{passphrase: "other-key", newKeyPass: "", newKeyPass2: "ignored"}
	held.takeNewKeyPassphrase()
	if held.passphrase != "other-key" || held.newKeyPass != "" || held.newKeyPass2 != "" {
		t.Fatalf("empty generate clobbered passphrase %q %q %q", held.passphrase, held.newKeyPass, held.newKeyPass2)
	}
	held.newKeyPass = "fresh"
	held.newKeyPass2 = "fresh"
	held.takeNewKeyPassphrase()
	if held.passphrase != "fresh" || held.newKeyPass != "" || held.newKeyPass2 != "" {
		t.Fatalf("typed generate passphrase %q fields %q %q", held.passphrase, held.newKeyPass, held.newKeyPass2)
	}

	mismatch := &uiState{
		langIdx:     LangIndex(LangEN),
		passphrase:  "kept",
		newKeyName:  "Ada",
		newKeyEmail: "ada@example.com",
		newKeyPass:  "one",
		newKeyPass2: "two",
	}
	if !mismatch.passMismatchVisible() {
		t.Fatal("live mismatch hidden")
	}
	if mismatch.submitNewKey() {
		t.Fatal("mismatch started generate")
	}
	if mismatch.busy || mismatch.statusKey == "action.generate_key" || mismatch.passphrase != "kept" || mismatch.newKeyPass != "one" {
		t.Fatalf("generate ran pass=%q busy=%v status=%s", mismatch.passphrase, mismatch.busy, mismatch.statusKey)
	}
	if mismatch.newKeyErr != Tr(LangEN, "err.pass_mismatch") || !mismatch.newKeyChecked {
		t.Fatalf("banner %q checked %v", mismatch.newKeyErr, mismatch.newKeyChecked)
	}

	oneSide := &uiState{langIdx: LangIndex(LangEN), passphrase: "kept", newKeyName: "Ada", newKeyEmail: "ada@example.com", newKeyPass: "only"}
	if oneSide.passMismatchVisible() {
		t.Fatal("blank confirm flagged early")
	}
	if oneSide.submitNewKey() || oneSide.passphrase != "kept" || !oneSide.passMismatchVisible() {
		t.Fatal("one-sided generate")
	}
	oneSide.newKeyPass2 = "only"
	oneSide.syncNewKeyPassError()
	if oneSide.newKeyErr != "" || oneSide.passMismatchVisible() {
		t.Fatalf("cleared err=%q visible=%v", oneSide.newKeyErr, oneSide.passMismatchVisible())
	}

	oneSide.newKeyShowPass = true
	oneSide.newKeyShowPass2 = true
	flipPassVisible(&oneSide.newKeyShowPass)
	if oneSide.newKeyShowPass {
		t.Fatal("eye did not hide")
	}
	flipPassVisible(&oneSide.newKeyShowPass)
	if !oneSide.newKeyShowPass {
		t.Fatal("eye did not show")
	}
	oneSide.newKeyErr = "x"
	oneSide.newKeyChecked = true
	oneSide.closeNewKeyWindow()
	if oneSide.newKeyShowPass || oneSide.newKeyShowPass2 || oneSide.newKeyErr != "" || oneSide.newKeyChecked {
		t.Fatal("close did not reset")
	}
	oneSide.newKeyShowPass = true
	oneSide.newKeyShowPass2 = true
	oneSide.newKeyErr = "y"
	oneSide.newKeyChecked = true
	oneSide.resetNewKeyPassUI()
	if oneSide.newKeyShowPass || oneSide.newKeyShowPass2 || oneSide.newKeyErr != "" || oneSide.newKeyChecked {
		t.Fatal("reset helper")
	}
	const day = 24 * 60 * 60
	req, errKey := validateNewKey(" Ada ", " ada@ex ", "secret", "secret", 2, base)
	if errKey != "" {
		t.Fatal(errKey)
	}
	if req.UserID != "Ada <ada@ex>" || req.ExpirationSec != 2*365*day || req.PrimaryAlg != "EDDSA" || req.SubkeyCurve != "Curve25519" || req.Unprotected {
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
		"placeholder.recipients", "placeholder.signer", "placeholder.symm_pass", "placeholder.enc_password2",
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
	if len(got) != 0 {
		t.Fatalf("cleared must not fall back to selected: %#v", got)
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

func TestSubkeyTableExpiryUsageAndCatalog(t *testing.T) {
	if err := ValidateCatalog(); err != nil {
		t.Fatal(err)
	}
	keys := []string{
		"table.subkey", "table.use.encrypt", "table.use.sign", "table.use.auth",
		"badge.auth",
		"btn.add_subkey", "btn.add_subkey_do", "btn.subkey_expiry", "btn.apply_expiry",
		"btn.subkey_revoke", "btn.subkey_revoke_do", "btn.subkey_delete",
		"title.add_subkey", "title.subkey_expiry", "title.subkey_revoke",
		"label.subkey_usage", "label.subkey_alg", "label.session_pass",
		"label.revoke_code", "label.revoke_reason",
		"usage.encrypt", "usage.sign", "usage.auth",
		"usage.encrypt.desc", "usage.sign.desc", "usage.auth.desc",
		"tip.addsub.ecdh", "tip.addsub.eddsa", "tip.addsub.rsa", "tip.addsub.ecdsa", "tip.addsub.elgamal", "tip.addsub.expiry",
		"placeholder.pass_reuse", "placeholder.pass_required", "placeholder.pass_optional_unprotected",
		"badge.unprotected", "desc.pass_unprotected", "desc.pass_protected",
		"exp.never_sub", "exp.plus1", "exp.plus2", "exp.plus5",
		"reason.retired", "reason.superseded", "reason.compromised", "reason.no",
		"hint.revoke_subkey", "hint.expiry_subkey", "hint.delete_subkey",
		"dlg.revoke_subkey_title", "dlg.revoke_subkey",
		"dlg.delete_subkey_title", "dlg.delete_subkey",
		"err.subkey_alg",
		"action.add_subkey", "action.set_key_expiry", "action.revoke_key", "action.delete_subkey",
	}
	for _, k := range keys {
		for _, l := range []Lang{LangEN, LangZH, LangJA} {
			v := strings.TrimSpace(Tr(l, k))
			if v == "" || v == k {
				t.Fatalf("translation missing %s/%s: %q", k, l, v)
			}
		}
	}
	if subkeyTypeText(LangEN) != "  └ Subkey" || subkeyTypeText(LangZH) != "  └ 子密钥" || subkeyTypeText(LangJA) != "  └ 副鍵" {
		t.Fatal("subkey row label")
	}
	if subkeyUsageText(LangEN, rnp.KeySummary{CanEncrypt: true}) != "  Encrypt" || subkeyUsageText(LangEN, rnp.KeySummary{CanSign: true}) != "  Sign" || subkeyUsageText(LangEN, rnp.KeySummary{CanAuth: true}) != "  Auth" {
		t.Fatal("usage indent")
	}
	for _, k := range keys {
		zh := Tr(LangZH, k)
		if strings.Contains(zh, "主键") || strings.Contains(zh, "副键") {
			t.Fatalf("simplified term %s: %s", k, zh)
		}
	}

	enc := rnp.KeySummary{Fingerprint: "SUBENC11223344556677889900AABBCCDDEEFF", KeyID: "99AABBCCDDEEFF00", Algorithm: "ECDH", IsSub: true, CanEncrypt: true}
	sig := rnp.KeySummary{Fingerprint: "SUBSIG11223344556677889900AABBCCDDEEFF", KeyID: "88AABBCCDDEEFF00", Algorithm: "EDDSA", IsSub: true, CanSign: true}
	auth := rnp.KeySummary{Fingerprint: "SUBAUT11223344556677889900AABBCCDDEEFF", KeyID: "77AABBCCDDEEFF00", Algorithm: "EDDSA", IsSub: true, CanAuth: true}
	primary := rnp.KeySummary{
		Fingerprint: "PRIMARY11223344556677889900AABBCCDDEE",
		KeyID:       "AABBCCDDEEFF0011",
		PrimaryUID:  "Ada <ada@ex>",
		Algorithm:   "EDDSA",
		HasSecret:   true,
		IsPrimary:   true,
		Subkeys:     []rnp.KeySummary{enc, sig, auth},
	}
	other := rnp.KeySummary{Fingerprint: "OTHER11223344556677889900AABBCCDDEEFF", KeyID: "0011223344556677", PrimaryUID: "Other", Algorithm: "RSA", IsPrimary: true}
	rows := buildKeyTableRows([]rnp.KeySummary{primary, other}, nil)
	if len(rows) != 5 || rows[0].Sub || rows[0].PrimaryFP != primary.Fingerprint {
		t.Fatalf("expanded rows = %+v", rows)
	}
	if !rows[1].Sub || rows[1].PrimaryFP != primary.Fingerprint || rows[1].Key.Fingerprint != enc.Fingerprint {
		t.Fatalf("first subkey row = %+v", rows[1])
	}
	if subkeyUsageRowKey(rows[1].Key) != "table.use.encrypt" || subkeyUsageRowKey(rows[2].Key) != "table.use.sign" || subkeyUsageRowKey(rows[3].Key) != "table.use.auth" {
		t.Fatal("usage row keys")
	}
	pFP, subFP := selectionForTableRow(rows[1])
	if pFP != primary.Fingerprint || subFP != enc.Fingerprint {
		t.Fatalf("subkey click = %s %s", pFP, subFP)
	}
	pFP, subFP = selectionForTableRow(rows[0])
	if pFP != primary.Fingerprint || subFP != "" {
		t.Fatalf("primary click = %s %s", pFP, subFP)
	}
	if got := keyIDText(rows[1].Key.Fingerprint, rows[1].Key.KeyID, keyIDModeShort); got != "99AABBCCDDEEFF00" {
		t.Fatalf("subkey short id = %q", got)
	}
	if got := keyIDText(rows[1].Key.Fingerprint, rows[1].Key.KeyID, keyIDModeFull); got != enc.Fingerprint {
		t.Fatalf("subkey fingerprint = %q", got)
	}
	folded := buildKeyTableRows([]rnp.KeySummary{primary}, map[string]bool{primary.Fingerprint: true})
	if len(folded) != 1 || folded[0].Sub {
		t.Fatalf("collapsed = %+v", folded)
	}
	if keyCollapsed(nil, primary.Fingerprint) || keyCollapsed(map[string]bool{}, primary.Fingerprint) {
		t.Fatal("default expanded")
	}

	matched := filterPrimaryKeys([]rnp.KeySummary{primary, other}, keyFilterAll, "99aabbcc")
	if len(matched) != 1 || matched[0].Fingerprint != primary.Fingerprint || len(matched[0].Subkeys) != 3 {
		t.Fatalf("subkey id search = %+v", matched)
	}
	if got := filterPrimaryKeys([]rnp.KeySummary{primary, other}, keyFilterAll, "subaut"); len(got) != 1 || len(got[0].Subkeys) != 3 {
		t.Fatalf("subkey fingerprint search = %+v", got)
	}
	if got := filterPrimaryKeys([]rnp.KeySummary{other, primary}, keyFilterAll, "ecdh"); len(got) != 1 || got[0].Fingerprint != primary.Fingerprint {
		t.Fatalf("subkey algorithm search = %+v", got)
	}

	now := time.Unix(1_700_000_000, 0)
	created := uint32(now.Unix() - 100)
	if expirationFromNow(created, now, 0) != 0 || expirationFromNow(created, now, -2) != 0 {
		t.Fatal("zero years")
	}
	if got, want := expirationFromNow(created, now, 1), uint32(100+yearSeconds); got != want {
		t.Fatalf("aged expiry %d want %d", got, want)
	}
	if got := expirationFromNow(0, now, 2); got != 2*yearSeconds {
		t.Fatalf("created 0 = %d", got)
	}
	future := uint32(now.Unix() + 50)
	if got := expirationFromNow(future, now, 1); got != yearSeconds {
		t.Fatalf("now before created = %d", got)
	}
	if got := expirationFromNow(uint32(now.Unix()), now, 5); got != 5*yearSeconds {
		t.Fatalf("exact now = %d", got)
	}

	alg, curve, bits := syncAddedSubkey("encrypt", "EDDSA", "Ed25519", 256)
	if alg != "ECDH" || curve != "Curve25519" || bits != 0 || rnp.UsageAllowsAlg("encrypt", "EDDSA") {
		t.Fatalf("encrypt rejected eddsa: %s %s %d", alg, curve, bits)
	}
	alg, curve, bits = syncAddedSubkey("sign", "ECDH", "Curve25519", 0)
	if alg != "EDDSA" || curve != "Ed25519" || bits != 0 || rnp.UsageAllowsAlg("sign", "ECDH") {
		t.Fatalf("sign rejected ecdh: %s %s %d", alg, curve, bits)
	}
	alg, curve, bits = syncAddedSubkey("authenticate", "ECDH", "", 0)
	if alg != "EDDSA" || rnp.UsageAllowsAlg("authenticate", "ECDH") || !rnp.UsageAllowsAlg("authenticate", "ECDSA") {
		t.Fatalf("auth = %s", alg)
	}
	alg, curve, bits = syncAddedSubkey("encrypt", "RSA", "Curve25519", 1024)
	if alg != "RSA" || curve != "" || bits != 4096 || !rnp.UsageAllowsAlg("encrypt", "ELGAMAL") {
		t.Fatalf("encrypt rsa = %s %s %d", alg, curve, bits)
	}
	alg, curve, bits = syncAddedSubkey("sign", "ECDSA", "Curve25519", 2048)
	if alg != "ECDSA" || curve != "NIST P-256" || bits != 0 {
		t.Fatalf("sign ecdsa = %s %s %d", alg, curve, bits)
	}
	if _, errKey := validateAddSubkey("encrypt", "EDDSA", "", "SHA256", "AES256", 0, 1, "", "", true); errKey != "err.pass_empty" {
		t.Fatalf("empty pass after sync = %s", errKey)
	}
	req, errKey := validateAddSubkey("sign", "ECDH", "Curve25519", "SHA256", "AES256", 0, 2, "secret", "", true)
	if errKey != "" || req.Usage != "sign" || req.Algorithm != "EDDSA" || req.Curve != "Ed25519" || req.ExpirationSec != 2*yearSeconds || req.Unprotected {
		t.Fatalf("validated = %+v %s", req, errKey)
	}
	req, errKey = validateAddSubkey("encrypt", "ELGAMAL", "", "", "", 0, 0, "", "session", true)
	if errKey != "" || req.Algorithm != "ELGAMAL" || req.Bits != 3072 || req.HashAlg != "SHA256" || req.Cipher != "AES256" || req.ExpirationSec != 0 || req.Unprotected {
		t.Fatalf("elgamal req = %+v %s", req, errKey)
	}
	req, errKey = validateAddSubkey("sign", "EDDSA", "", "SHA256", "AES256", 0, 1, "", "", false)
	if errKey != "" || !req.Unprotected || req.Algorithm != "EDDSA" {
		t.Fatalf("unprotected empty = %+v %s", req, errKey)
	}
	req, errKey = validateAddSubkey("sign", "EDDSA", "", "SHA256", "AES256", 0, 0, "typed-pass", "session", false)
	if errKey != "" || req.Unprotected {
		t.Fatalf("typed pass on open primary = %+v %s", req, errKey)
	}
	openPrimary := &uiState{
		selected: "P",
		keys: []rnp.KeySummary{{
			Fingerprint: "P", IsPrimary: true, HasSecret: true, IsProtected: false,
		}},
	}
	if openPrimary.primaryProtected() {
		t.Fatal("unprotected primary")
	}
	openPrimary.keys[0].IsProtected = true
	if !openPrimary.primaryProtected() {
		t.Fatal("protected primary")
	}
	openPrimary.keys[0].HasSecret = false
	openPrimary.keys[0].IsProtected = false
	if !openPrimary.primaryProtected() {
		t.Fatal("public primary")
	}
	openPrimary.selected = "missing"
	if !openPrimary.primaryProtected() {
		t.Fatal("missing primary")
	}
	if legal := legalSubkeyAlgs("encrypt"); len(legal) != 3 || legal[0] != "ECDH" {
		t.Fatalf("encrypt algs %v", legal)
	}
	if legal := legalSubkeyAlgs("authenticate"); len(legal) != 3 || legal[0] != "EDDSA" {
		t.Fatalf("auth algs %v", legal)
	}
}

func exerciseSubkeyLifecycle(s *uiState) error {
	primary := s.selected
	parent, ok := primaryByFP(s.keys, primary)
	if !ok || !parent.HasSecret {
		return fmt.Errorf("subkey lifecycle: primary %q missing", primary)
	}
	before := append([]rnp.KeySummary(nil), parent.Subkeys...)
	s.subReq = rnp.SubkeyGenRequest{
		Usage:     "encrypt",
		Algorithm: "ECDH",
		Curve:     "Curve25519",
		HashAlg:   "SHA256",
		Cipher:    "AES256",
	}
	if err := runUI(s, "action.add_subkey", s.addSubkey); err != nil {
		return err
	}
	if s.selected != primary {
		return fmt.Errorf("add subkey selected %q", s.selected)
	}
	parent, ok = primaryByFP(s.keys, primary)
	if !ok {
		return fmt.Errorf("primary missing after add")
	}
	if len(parent.Subkeys) != len(before)+1 {
		return fmt.Errorf("subkeys %d -> %d", len(before), len(parent.Subkeys))
	}
	seen := map[string]bool{}
	for _, sk := range before {
		seen[sk.Fingerprint] = true
	}
	var added rnp.KeySummary
	for _, sk := range parent.Subkeys {
		if !seen[sk.Fingerprint] {
			added = sk
			break
		}
	}
	if added.Fingerprint == "" || !added.IsSub || !added.CanEncrypt {
		return fmt.Errorf("added subkey %+v", added)
	}
	if s.selectedSubkey != added.Fingerprint {
		return fmt.Errorf("selected subkey %q", s.selectedSubkey)
	}
	exp := expirationFromNow(added.CreatedUnix, time.Now().UTC(), 1)
	if exp == 0 {
		return fmt.Errorf("computed expiry is 0")
	}
	s.targetFP = added.Fingerprint
	s.expirySeconds = exp
	if err := runUI(s, "action.set_key_expiry", s.setKeyExpiry); err != nil {
		return err
	}
	if s.selected != primary {
		return fmt.Errorf("expiry selected %q", s.selected)
	}
	updated, ok := subByFP(s.keys, primary, added.Fingerprint)
	if !ok {
		return fmt.Errorf("subkey missing after expiry")
	}
	wantExp := added.CreatedUnix + exp
	if added.CreatedUnix == 0 {
		wantExp = exp
	}
	if updated.ExpiresUnix != wantExp {
		return fmt.Errorf("expiry unix %d want %d", updated.ExpiresUnix, wantExp)
	}
	parent, _ = primaryByFP(s.keys, primary)
	if parent.IsRevoked || updated.IsRevoked {
		return fmt.Errorf("revoked during expiry primary=%v sub=%v", parent.IsRevoked, updated.IsRevoked)
	}
	s.targetFP = added.Fingerprint
	s.revokeCode = "retired"
	s.revokeReason = "lifecycle"
	if err := runUI(s, "action.revoke_key", s.revokeKey); err != nil {
		return err
	}
	updated, ok = subByFP(s.keys, primary, added.Fingerprint)
	if !ok || !updated.IsRevoked {
		return fmt.Errorf("subkey revoke ok=%v revoked=%v", ok, updated.IsRevoked)
	}
	parent, ok = primaryByFP(s.keys, primary)
	if !ok || parent.IsRevoked || s.selected != primary {
		return fmt.Errorf("primary after revoke ok=%v revoked=%v selected=%q", ok, parent.IsRevoked, s.selected)
	}
	s.targetFP = primary
	if err := runUI(s, "action.delete_subkey", s.deleteSubkey); err == nil {
		return fmt.Errorf("delete primary as subkey succeeded")
	}
	s.lastErr = ""
	parent, ok = primaryByFP(s.keys, primary)
	if !ok || parent.IsRevoked {
		return fmt.Errorf("primary damaged by refused delete")
	}
	s.targetFP = added.Fingerprint
	if err := runUI(s, "action.delete_subkey", s.deleteSubkey); err != nil {
		return err
	}
	if _, ok := subByFP(s.keys, primary, added.Fingerprint); ok {
		return fmt.Errorf("deleted subkey still listed")
	}
	parent, ok = primaryByFP(s.keys, primary)
	if !ok || parent.IsRevoked || s.selected != primary {
		return fmt.Errorf("primary after delete ok=%v revoked=%v selected=%q", ok, parent.IsRevoked, s.selected)
	}
	if len(parent.Subkeys) != len(before) {
		return fmt.Errorf("subkeys left %d want %d", len(parent.Subkeys), len(before))
	}
	return nil
}

func primaryByFP(keys []rnp.KeySummary, fp string) (rnp.KeySummary, bool) {
	for _, k := range keys {
		if k.Fingerprint == fp && isPrimaryRow(k) {
			return k, true
		}
	}
	return rnp.KeySummary{}, false
}

func subByFP(keys []rnp.KeySummary, primary, fp string) (rnp.KeySummary, bool) {
	parent, ok := primaryByFP(keys, primary)
	if !ok {
		return rnp.KeySummary{}, false
	}
	for _, sk := range parent.Subkeys {
		if sk.Fingerprint == fp {
			return sk, true
		}
	}
	return rnp.KeySummary{}, false
}

func TestI18nComplete(t *testing.T) {
	if err := ValidateCatalog(); err != nil {
		t.Fatal(err)
	}
	keys := []string{
		"status.key_count_one", "status.key_count_many", "tip.copy_lib_path",
		"toast.done", "toast.path_copied", "toast.fingerprint_copied",
		"toast.text_signed", "toast.text_encrypted",
		"btn.ok", "btn.quit", "alert.error", "alert.redacted",
		"op.encrypt", "op.decrypt", "op.sign", "op.verify",
		"btn.paste", "btn.clear", "btn.copy_result", "btn.copy_report",
		"btn.send_decrypt", "btn.send_verify", "btn.goto_keys",
		"label.encrypt_input", "label.decrypt_input", "label.sign_input", "label.verify_input",
		"label.enc_password", "label.enc_password2", "placeholder.enc_password2", "label.decrypt_pass", "label.sign_pass",
		"label.studio_advanced", "mode.password", "mode.pubkey",
		"hint.hybrid_on", "hint.decrypt_pass", "hint.pass_mismatch_live",
		"err.enc_password_empty", "err.enc_recipients_empty", "err.passphrase_mismatch",
		"toast.copied", "toggle.hybrid", "toggle.detached_verify", "status.file_done",
		"dlg.quit_title", "dlg.quit_msg", "dlg.quit_busy_msg",
		"state.idle", "state.busy", "status.ready",
	}
	for _, k := range keys {
		for _, l := range []Lang{LangEN, LangZH, LangJA} {
			v := strings.TrimSpace(Tr(l, k))
			if v == "" || v == k {
				t.Fatalf("translation missing %s/%s: %q", k, l, v)
			}
		}
		zh := Tr(LangZH, k)
		if strings.Contains(zh, "主键") || strings.Contains(zh, "副键") {
			t.Fatalf("simplified term %s: %s", k, zh)
		}
	}
}

func TestStatusBarAndFeedback(t *testing.T) {
	keys := []rnp.KeySummary{
		{Fingerprint: "P1", IsPrimary: true, Subkeys: []rnp.KeySummary{{Fingerprint: "S1", IsSub: true}}},
		{Fingerprint: "P2", IsPrimary: true},
		{Fingerprint: "S2", IsSub: true},
	}
	if countPrimaryKeys(keys) != 2 {
		t.Fatalf("primary count %d", countPrimaryKeys(keys))
	}
	if formatPrimaryCount(LangEN, 0) != "0 keys" || formatPrimaryCount(LangEN, 1) != "1 key" || formatPrimaryCount(LangEN, 3) != "3 keys" {
		t.Fatalf("en counts %q %q %q", formatPrimaryCount(LangEN, 0), formatPrimaryCount(LangEN, 1), formatPrimaryCount(LangEN, 3))
	}
	if formatPrimaryCount(LangZH, 0) != "0 把密钥" || formatPrimaryCount(LangZH, 3) != "3 把密钥" {
		t.Fatalf("zh counts %q %q", formatPrimaryCount(LangZH, 0), formatPrimaryCount(LangZH, 3))
	}
	if formatPrimaryCount(LangJA, 1) != "1 件の鍵" || formatPrimaryCount(LangJA, 3) != "3 件の鍵" {
		t.Fatalf("ja counts %q %q", formatPrimaryCount(LangJA, 1), formatPrimaryCount(LangJA, 3))
	}

	bar := &uiState{langIdx: LangIndex(LangEN), statusKey: "status.ready", keys: keys}
	if bar.statusBarState() != "Ready" {
		t.Fatalf("idle %q", bar.statusBarState())
	}
	if bar.statusBarRight() != "2 keys  ·  Ready" {
		t.Fatalf("right %q", bar.statusBarRight())
	}
	bar.busy = true
	bar.statusKey = "action.encrypt_text"
	if bar.statusBarState() != "busy · Encrypt text" {
		t.Fatalf("busy %q", bar.statusBarState())
	}
	bar.langIdx = LangIndex(LangZH)
	if bar.statusBarRight() != "2 把密钥  ·  忙碌 · 加密文本" {
		t.Fatalf("zh right %q", bar.statusBarRight())
	}

	if feedbackRoute("action.encrypt_text", false, false, false) != "toast" {
		t.Fatal("success toast")
	}
	if feedbackRoute("action.verify_text", false, false, false) != "alert" || feedbackRoute("action.verify_file", false, false, false) != "alert" {
		t.Fatal("verify alert")
	}
	if feedbackRoute("action.save", true, false, false) != "alert" {
		t.Fatal("failure alert")
	}
	if feedbackRoute("action.generate_key", true, false, true) != "inline" {
		t.Fatal("child inline")
	}
	if feedbackRoute("action.generate_key", true, false, false) != "alert" {
		t.Fatal("closed child alert")
	}
	if feedbackRoute("action.load_keyring", false, true, false) != "none" || feedbackRoute("action.load_keyring", true, true, false) != "none" {
		t.Fatal("silent load")
	}

	s := &uiState{langIdx: LangIndex(LangEN), passphrase: "super-secret-pass"}
	s.noteActionFeedback("action.encrypt_text", actionOutcome{}, false)
	if s.alertOpen || len(s.pendingToasts) != 1 || s.pendingToasts[0].Type != "success" || s.pendingToasts[0].Timeout != toastLifetime {
		t.Fatalf("toast %+v alert %v", s.pendingToasts, s.alertOpen)
	}
	if strings.Contains(s.pendingToasts[0].Title, "super-secret-pass") || strings.Contains(s.pendingToasts[0].Description, "BEGIN PGP") {
		t.Fatal("success toast leaked")
	}

	s.pendingToasts = nil
	s.textOutput = "[Math: VALID]\nkey=ABC"
	s.noteActionFeedback("action.verify_text", actionOutcome{}, false)
	if !s.alertOpen || len(s.pendingToasts) != 0 || !strings.Contains(s.alertMsg, "[Math: VALID]") {
		t.Fatalf("verify alert %v %q toasts %d", s.alertOpen, s.alertMsg, len(s.pendingToasts))
	}

	s.alertOpen = false
	s.noteActionFeedback("action.save", actionOutcome{err: fmt.Errorf("disk full super-secret-pass")}, false)
	if !s.alertOpen || len(s.pendingToasts) != 0 || strings.Contains(s.alertMsg, "super-secret-pass") || !strings.Contains(s.alertMsg, "••••") {
		t.Fatalf("failure alert %q", s.alertMsg)
	}

	s.alertOpen = false
	s.alertMsg = ""
	s.noteActionFeedback("action.export_secret", actionOutcome{err: fmt.Errorf("-----BEGIN PGP PRIVATE KEY BLOCK----- super-secret-pass")}, false)
	if !s.alertOpen || strings.Contains(s.alertMsg, "super-secret-pass") || strings.Contains(s.alertMsg, "PRIVATE KEY") {
		t.Fatalf("redacted %q", s.alertMsg)
	}

	s.alertOpen = false
	s.alertMsg = ""
	s.pendingToasts = nil
	s.noteActionFeedback("action.load_keyring", actionOutcome{}, true)
	s.noteActionFeedback("action.load_keyring", actionOutcome{err: fmt.Errorf("missing ring")}, true)
	if s.alertOpen || s.alertMsg != "" || len(s.pendingToasts) != 0 {
		t.Fatalf("silent load notified alert=%v %q toasts=%d", s.alertOpen, s.alertMsg, len(s.pendingToasts))
	}
}

func TestFluentGlassChrome(t *testing.T) {
	if !useIntegratedMenuBar("windows") || !useIntegratedMenuBar("linux") || useIntegratedMenuBar("darwin") {
		t.Fatal("integrated menu bar")
	}
	win := mainWindowOptions("windows")
	if win.TitleBarStyle != mygo.TitleBarHidden || win.TitleBarHeight != 40 || win.Vibrancy != mygo.VibrancyMica {
		t.Fatalf("windows %+v", win.TitleBarStyle)
	}
	lin := mainWindowOptions("linux")
	if lin.TitleBarStyle != mygo.TitleBarDefault || lin.Vibrancy != "" || lin.TitleBarHeight != 0 {
		t.Fatal("linux title bar")
	}
	mac := mainWindowOptions("darwin")
	if mac.TitleBarStyle != mygo.TitleBarDefault || mac.Vibrancy != "" {
		t.Fatal("darwin title bar")
	}

	base := rnpLightTheme(ui.Preferences{})
	if base.Dark || base.Background != ui.Hex("#edf1f7") {
		t.Fatal("light canvas")
	}
	hi := rnpLightTheme(ui.Preferences{HighContrast: true})
	if hi.Dark || hi.Border == base.Border {
		t.Fatal("high contrast")
	}
	scaled := rnpLightTheme(ui.Preferences{TextScale: 1.5})
	if scaled.FontSize <= base.FontSize {
		t.Fatalf("text scale %v", scaled.FontSize)
	}

	rest := rnpButtonFace(rnpBtnSecondary, false, false)
	hover := rnpButtonFace(rnpBtnSecondary, true, false)
	press := rnpButtonFace(rnpBtnSecondary, true, true)
	if colorSum(rest.From) <= colorSum(rest.To) || colorSum(press.From) >= colorSum(rest.From) {
		t.Fatal("secondary face")
	}
	if hover.ShadowBlur <= rest.ShadowBlur || colorSum(hover.To) < colorSum(rest.To) {
		t.Fatal("secondary hover")
	}
	pri := rnpButtonFace(rnpBtnPrimary, false, false)
	priPress := rnpButtonFace(rnpBtnPrimary, false, true)
	if colorSum(pri.From) <= colorSum(pri.To) || pri.Text != ui.Hex("#ffffff") || colorSum(priPress.To) >= colorSum(pri.To) {
		t.Fatal("primary face")
	}
	if pri.Shadow != ui.RGBA(37, 99, 235, 0.28) || rest.Shadow != ui.RGBA(15, 23, 42, 0.10) {
		t.Fatal("shadow")
	}
	danger := rnpButtonFace(rnpBtnDanger, false, false)
	if danger.Text == ui.Hex("#ffffff") || danger.Border.A == 0 || colorSum(danger.From) <= colorSum(danger.To) {
		t.Fatal("danger face")
	}
	if got := rnpButtonFace("other", false, false); got.From != rest.From {
		t.Fatal("face fallback")
	}

	g := menuGlass()
	if g.Style != glass.Regular || g.Clarity != 0.25 || g.Tint != ui.RGBA(255, 255, 255, 0.72) {
		t.Fatal("menu glass")
	}
	for _, kind := range []string{"success", "error", "info"} {
		pane, fg, border := toastGlass(kind)
		if pane.Tint.A == 0 || pane.Clarity == 0 || fg == ui.Hex("#ffffff") || border.A == 0 {
			t.Fatalf("toast %s", kind)
		}
	}
}

func colorSum(c ui.Color) int { return int(c.R) + int(c.G) + int(c.B) }

func TestPassphraseEyeEverywhere(t *testing.T) {
	s := &uiState{langIdx: LangIndex(LangEN)}

	// All passphrase fields start masked by default.
	if s.showTextEncPass || s.showTextEncPass2 || s.showTextDecPass || s.showTextSignPass ||
		s.showFileEncPass || s.showFileEncPass2 || s.showFileDecPass || s.showFileSignPass ||
		s.showUnlockPass || s.subShowPass || s.expiryShowPass || s.revokeShowPass {
		t.Fatal("passphrase visibility flags should default to false")
	}

	// nil guard must be a safe no-op.
	flipPassVisible(nil)

	// Each workspace and subkey modal flag toggles independently.
	toggles := []struct {
		name string
		ptr  *bool
	}{
		{name: "text_enc", ptr: &s.showTextEncPass},
		{name: "text_enc2", ptr: &s.showTextEncPass2},
		{name: "text_dec", ptr: &s.showTextDecPass},
		{name: "text_sign", ptr: &s.showTextSignPass},
		{name: "file_enc", ptr: &s.showFileEncPass},
		{name: "file_enc2", ptr: &s.showFileEncPass2},
		{name: "file_dec", ptr: &s.showFileDecPass},
		{name: "file_sign", ptr: &s.showFileSignPass},
		{name: "unlock", ptr: &s.showUnlockPass},
		{name: "add_subkey", ptr: &s.subShowPass},
		{name: "expiry", ptr: &s.expiryShowPass},
		{name: "revoke", ptr: &s.revokeShowPass},
	}
	for i, tc := range toggles {
		flipPassVisible(tc.ptr)
		if !*tc.ptr {
			t.Fatalf("%s: flip did not show passphrase", tc.name)
		}
		for j, other := range toggles {
			if i != j && *other.ptr {
				t.Fatalf("%s toggle unexpectedly mutated %s", tc.name, other.name)
			}
		}
		flipPassVisible(tc.ptr)
		if *tc.ptr {
			t.Fatalf("%s: second flip did not hide passphrase", tc.name)
		}
	}

	// Subkey modal close and reset helpers hide passphrases and clear inline errors.
	s.subShowPass = true
	s.subErr = "err-add"
	s.closeAddSubWindow()
	if s.subShowPass || s.subErr != "" {
		t.Fatalf("closeAddSubWindow did not reset: show=%v err=%q", s.subShowPass, s.subErr)
	}
	s.subShowPass = true
	s.subErr = "err-add-2"
	s.resetAddSubPassUI()
	if s.subShowPass || s.subErr != "" {
		t.Fatalf("resetAddSubPassUI did not reset: show=%v err=%q", s.subShowPass, s.subErr)
	}

	s.expiryShowPass = true
	s.expiryErr = "err-exp"
	s.closeExpiryWindow()
	if s.expiryShowPass || s.expiryErr != "" {
		t.Fatalf("closeExpiryWindow did not reset: show=%v err=%q", s.expiryShowPass, s.expiryErr)
	}
	s.expiryShowPass = true
	s.expiryErr = "err-exp-2"
	s.resetExpiryPassUI()
	if s.expiryShowPass || s.expiryErr != "" {
		t.Fatalf("resetExpiryPassUI did not reset: show=%v err=%q", s.expiryShowPass, s.expiryErr)
	}

	s.revokeShowPass = true
	s.revokeErr = "err-rev"
	s.closeRevokeWindow()
	if s.revokeShowPass || s.revokeErr != "" {
		t.Fatalf("closeRevokeWindow did not reset: show=%v err=%q", s.revokeShowPass, s.revokeErr)
	}
	s.revokeShowPass = true
	s.revokeErr = "err-rev-2"
	s.resetRevokePassUI()
	if s.revokeShowPass || s.revokeErr != "" {
		t.Fatalf("resetRevokePassUI did not reset: show=%v err=%q", s.revokeShowPass, s.revokeErr)
	}

	// Submit-success close path in applyResult resets each subkey modal's show flag and error.
	s.subShowPass = true
	s.subErr = "stale"
	s.pendingCloseAddSub = false
	s.applyResult("action.add_subkey", actionOutcome{})
	if !s.pendingCloseAddSub || s.subShowPass || s.subErr != "" {
		t.Fatalf("add_subkey success reset failed: close=%v show=%v err=%q", s.pendingCloseAddSub, s.subShowPass, s.subErr)
	}

	s.expiryShowPass = true
	s.expiryErr = "stale"
	s.pendingCloseExpiry = false
	s.applyResult("action.set_key_expiry", actionOutcome{})
	if !s.pendingCloseExpiry || s.expiryShowPass || s.expiryErr != "" {
		t.Fatalf("set_key_expiry success reset failed: close=%v show=%v err=%q", s.pendingCloseExpiry, s.expiryShowPass, s.expiryErr)
	}

	s.revokeShowPass = true
	s.revokeErr = "stale"
	s.pendingCloseRevoke = false
	s.applyResult("action.revoke_key", actionOutcome{})
	if !s.pendingCloseRevoke || s.revokeShowPass || s.revokeErr != "" {
		t.Fatalf("revoke_key success reset failed: close=%v show=%v err=%q", s.pendingCloseRevoke, s.revokeShowPass, s.revokeErr)
	}

	// Action failure keeps the modal open and records the inline error.
	s.pendingCloseAddSub = false
	s.applyResult("action.add_subkey", actionOutcome{err: fmt.Errorf("bad pass")})
	if s.pendingCloseAddSub || s.subErr == "" {
		t.Fatalf("add_subkey failure should keep modal open with error: close=%v err=%q", s.pendingCloseAddSub, s.subErr)
	}

	// Trilingual eye tooltips are present and distinct.
	for _, l := range []Lang{LangEN, LangZH, LangJA} {
		showTip := strings.TrimSpace(Tr(l, "tip.show_passphrase"))
		hideTip := strings.TrimSpace(Tr(l, "tip.hide_passphrase"))
		if showTip == "" || hideTip == "" || showTip == hideTip {
			t.Fatalf("invalid eye tips for %s: show=%q hide=%q", l, showTip, hideTip)
		}
	}

	// Regression guard: .Password() must only live inside passphraseRow in keygen_ui.go.
	matches, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob *.go: %v", err)
	}
	var passwordCalls []string
	for _, name := range matches {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		b, rerr := os.ReadFile(name)
		if rerr != nil {
			t.Fatalf("read %s: %v", name, rerr)
		}
		for i, line := range strings.Split(string(b), "\n") {
			if strings.Contains(line, ".Password(") {
				passwordCalls = append(passwordCalls, fmt.Sprintf("%s:%d", name, i+1))
			}
		}
	}
	if len(passwordCalls) != 1 || !strings.HasPrefix(passwordCalls[0], "keygen_ui.go:") {
		t.Fatalf("bare .Password() call found outside passphraseRow: %v", passwordCalls)
	}
}
