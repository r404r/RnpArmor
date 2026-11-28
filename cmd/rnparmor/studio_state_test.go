package main

import (
	"strings"
	"testing"

	"github.com/r404r/RnpArmor/internal/rnp"
)

func TestEffectiveEncryptMode(t *testing.T) {
	keys := []rnp.KeySummary{{Fingerprint: "AA", IsPrimary: true}}
	if got := effectiveEncryptMode(encModePassword, false, keys); got != encModePubKey {
		t.Fatalf("untouched with keys = %d", got)
	}
	if got := effectiveEncryptMode(encModePassword, false, nil); got != encModePassword {
		t.Fatalf("empty keyring = %d", got)
	}
	subs := []rnp.KeySummary{{Fingerprint: "SUB", IsSub: true}}
	if got := effectiveEncryptMode(encModePubKey, false, subs); got != encModePassword {
		t.Fatalf("subkeys only = %d", got)
	}
	if got := effectiveEncryptMode(encModePassword, true, keys); got != encModePassword {
		t.Fatalf("touched password = %d", got)
	}
	if got := effectiveEncryptMode(encModePubKey, true, nil); got != encModePubKey {
		t.Fatalf("touched pubkey = %d", got)
	}
}

func TestResolveEncryptCredentials(t *testing.T) {
	fps, pass, errKey := resolveEncryptCredentials(encModePassword, false, "AA", "pw", "pw")
	if errKey != "" || pass != "pw" || fps != nil {
		t.Fatalf("password mode fps=%#v pass=%q err=%s", fps, pass, errKey)
	}
	_, _, errKey = resolveEncryptCredentials(encModePassword, false, "AA", "", "")
	if errKey != "err.enc_password_empty" {
		t.Fatalf("empty password %s", errKey)
	}
	_, _, errKey = resolveEncryptCredentials(encModePassword, false, "", "a", "b")
	if errKey != "err.passphrase_mismatch" {
		t.Fatalf("mismatch %s", errKey)
	}

	fps, pass, errKey = resolveEncryptCredentials(encModePubKey, false, " AA , BB ", "stale", "stale")
	if errKey != "" || pass != "" || len(fps) != 2 || fps[0] != "AA" || fps[1] != "BB" {
		t.Fatalf("pubkey fps=%#v pass=%q err=%s", fps, pass, errKey)
	}
	_, _, errKey = resolveEncryptCredentials(encModePubKey, false, "  ", "pw", "pw")
	if errKey != "err.enc_recipients_empty" {
		t.Fatalf("empty recipients %s", errKey)
	}

	_, _, errKey = resolveEncryptCredentials(encModePassword, true, "", "pw", "pw")
	if errKey != "err.enc_recipients_empty" {
		t.Fatalf("hybrid recipients %s", errKey)
	}
	_, _, errKey = resolveEncryptCredentials(encModePubKey, true, "AA", "", "")
	if errKey != "err.enc_password_empty" {
		t.Fatalf("hybrid password %s", errKey)
	}
	_, _, errKey = resolveEncryptCredentials(encModePubKey, true, "AA", "p", "q")
	if errKey != "err.passphrase_mismatch" {
		t.Fatalf("hybrid mismatch %s", errKey)
	}
	fps, pass, errKey = resolveEncryptCredentials(encModePassword, true, "AA, BB", "p", "p")
	if errKey != "" || pass != "p" || len(fps) != 2 {
		t.Fatalf("hybrid ok fps=%#v pass=%q err=%s", fps, pass, errKey)
	}
}

func TestStageEncryptDoesNotUseSelected(t *testing.T) {
	s := &uiState{
		langIdx:            LangIndex(LangEN),
		textEncMode:        encModePassword,
		textEncModeTouched: true,
		encPassword:        "pw",
		encPassword2:       "pw",
		encRecipients:      "SHOULD-NOT",
		selected:           "SELECTED",
		fileEncPassword:    "file-old",
	}
	if !s.stageEncrypt(false) {
		t.Fatal(s.textFormErr)
	}
	in := s.snapshot()
	if in.encPassword != "pw" {
		t.Fatalf("password %q", in.encPassword)
	}
	if got := recipientFingerprints(in); len(got) != 0 {
		t.Fatalf("password mode leaked %#v", got)
	}
	if strings.Contains(in.encRecipients, "SELECTED") {
		t.Fatalf("selected leaked %q", in.encRecipients)
	}

	file := &uiState{
		langIdx:            LangIndex(LangEN),
		fileEncMode:        encModePubKey,
		fileEncModeTouched: true,
		encRecipients:      "AABB",
		fileEncPassword:    "stale",
		fileEncPassword2:   "stale",
		selected:           "SELECTED",
		encPassword:        "text-secret",
	}
	if !file.stageEncrypt(true) {
		t.Fatal(file.fileFormErr)
	}
	in = file.snapshot()
	if in.encPassword != "" {
		t.Fatalf("pubkey mode kept password %q", in.encPassword)
	}
	got := recipientFingerprints(in)
	if len(got) != 1 || got[0] != "AABB" {
		t.Fatalf("pubkey fps %#v", got)
	}

	bad := &uiState{
		langIdx:            LangIndex(LangEN),
		textEncMode:        encModePassword,
		textEncModeTouched: true,
		encPassword:        "a",
		encPassword2:       "b",
		selected:           "SELECTED",
	}
	if bad.stageEncrypt(false) {
		t.Fatal("mismatch submitted")
	}
	if bad.textFormErr != Tr(LangEN, "err.passphrase_mismatch") {
		t.Fatalf("form %q", bad.textFormErr)
	}
	if bad.pendingEncSet {
		t.Fatal("failed stage left credentials")
	}

	smart := &uiState{
		langIdx:      LangIndex(LangEN),
		keys:         []rnp.KeySummary{{Fingerprint: "AA", IsPrimary: true}},
		encPassword:  "pw",
		encPassword2: "pw",
		selected:     "SELECTED",
	}
	if smart.stageEncrypt(false) {
		t.Fatal("smart default should require a recipient")
	}
	if smart.textFormErr != Tr(LangEN, "err.enc_recipients_empty") {
		t.Fatalf("smart form %q", smart.textFormErr)
	}
}

func TestSessionPassphrasePreference(t *testing.T) {
	if got := effectiveSessionPassphrase(actionInput{passphrase: "ring", opPassphrase: "local"}); got != "local" {
		t.Fatalf("local %q", got)
	}
	if got := effectiveSessionPassphrase(actionInput{passphrase: "ring"}); got != "ring" {
		t.Fatalf("fallback %q", got)
	}
	if got := effectiveSessionPassphrase(actionInput{}); got != "" {
		t.Fatalf("empty %q", got)
	}
}

func TestSendTextBetweenTasks(t *testing.T) {
	s := &uiState{
		textOp:          studioOpEncrypt,
		textInput:       "plain",
		textOutput:      "CIPHER",
		textMode:        string(rnp.SignModeDetached),
		showTextEncPass: true,
		textFormErr:     "stale",
	}
	s.sendTextToDecrypt()
	if s.textOp != studioOpDecrypt || s.textDecryptInput != "CIPHER" || s.showTextEncPass || s.textFormErr != "" {
		t.Fatalf("decrypt send op=%d box=%q show=%v err=%q", s.textOp, s.textDecryptInput, s.showTextEncPass, s.textFormErr)
	}
	s.textOutput = "SIG"
	s.showTextSignPass = true
	s.sendTextToVerify()
	if s.textOp != studioOpVerify || s.textVerifyInput != "plain" || s.textVerifyDetached != "SIG" || s.showTextSignPass {
		t.Fatalf("detached verify in=%q det=%q show=%v", s.textVerifyInput, s.textVerifyDetached, s.showTextSignPass)
	}
	s.textMode = string(rnp.SignModeCleartext)
	s.textOutput = "BLOCK"
	s.sendTextToVerify()
	if s.textVerifyInput != "BLOCK" || s.textVerifyDetached != "" || s.textOp != studioOpVerify {
		t.Fatalf("clear verify in=%q det=%q", s.textVerifyInput, s.textVerifyDetached)
	}
}

func TestEncryptModeSwitchClearsSecrets(t *testing.T) {
	s := &uiState{showTextEncPass: true, showTextEncPass2: true, textFormErr: "x", textEncMode: encModePassword}
	s.noteEncryptMode(false, encModePubKey)
	if s.textEncMode != encModePubKey || !s.textEncModeTouched || s.showTextEncPass || s.showTextEncPass2 || s.textFormErr != "" {
		t.Fatalf("text mode %+v", s.textEncModeTouched)
	}
	s.showFileDecPass = true
	s.fileFormErr = "y"
	s.noteFileOpChanged()
	if s.showFileDecPass || s.fileFormErr != "" {
		t.Fatal("file op reset")
	}
}

func TestPassConfirmMismatch(t *testing.T) {
	if passConfirmMismatch("a", "") || !passConfirmMismatch("a", "b") || passConfirmMismatch("a", "a") {
		t.Fatal("mismatch highlight")
	}
}
