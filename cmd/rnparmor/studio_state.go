package main

import (
	"strings"

	"github.com/r404r/RnpArmor/internal/rnp"
)

const (
	studioOpEncrypt = 0
	studioOpDecrypt = 1
	studioOpSign    = 2
	studioOpVerify  = 3

	encModePassword = 0
	encModePubKey   = 1

	// studioSplitDefault is the left pane of ui.SplitPercent.
	// That widget stores a percentage from 0 to 100, so 54 is the
	// confirmed 54% / 46% layout (the design ratio 0.54).
	studioSplitDefault float32 = 54
)

// effectiveEncryptMode is the encryption method shown and submitted.
// A choice the user already made is kept. Otherwise a keyring that has
// a primary key starts on public-key encryption, and an empty keyring
// starts on a password.
func effectiveEncryptMode(mode int, touched bool, keys []rnp.KeySummary) int {
	if touched {
		return mode
	}
	labels, _ := recipientOptions(keys)
	if len(labels) > 0 {
		return encModePubKey
	}
	return encModePassword
}

// resolveEncryptCredentials checks one encrypt submit.
// Password mode drops any recipient list. Public-key mode drops any
// password left in a hidden field. Hybrid requires both, recipients
// first. It never substitutes a selected key.
func resolveEncryptCredentials(mode int, hybrid bool, recipientsCSV, pass, pass2 string) (fps []string, password string, errKey string) {
	fps = splitCSV(recipientsCSV)
	if hybrid {
		if len(fps) == 0 {
			return nil, "", "err.enc_recipients_empty"
		}
		if pass == "" {
			return nil, "", "err.enc_password_empty"
		}
		if pass != pass2 {
			return nil, "", "err.passphrase_mismatch"
		}
		return fps, pass, ""
	}
	if mode == encModePubKey {
		if len(fps) == 0 {
			return nil, "", "err.enc_recipients_empty"
		}
		return fps, "", ""
	}
	if pass == "" {
		return nil, "", "err.enc_password_empty"
	}
	if pass != pass2 {
		return nil, "", "err.passphrase_mismatch"
	}
	return nil, pass, ""
}

// effectiveSessionPassphrase prefers a passphrase typed on the current
// task. An empty one falls back to the portable keyring unlock secret,
// which may itself be empty for an unprotected key.
func effectiveSessionPassphrase(in actionInput) string {
	if in.opPassphrase != "" {
		return in.opPassphrase
	}
	return in.passphrase
}

func passConfirmMismatch(pass, pass2 string) bool {
	return pass2 != "" && pass != pass2
}

func (s *uiState) ensureStudioSplit() {
	if s.textSplitPct <= 0 {
		s.textSplitPct = studioSplitDefault
	}
	if s.fileSplitPct <= 0 {
		s.fileSplitPct = studioSplitDefault
	}
}

func (s *uiState) resetTextPassVisibility() {
	s.showTextEncPass = false
	s.showTextEncPass2 = false
	s.showTextDecPass = false
	s.showTextSignPass = false
}

func (s *uiState) resetFilePassVisibility() {
	s.showFileEncPass = false
	s.showFileEncPass2 = false
	s.showFileDecPass = false
	s.showFileSignPass = false
}

func (s *uiState) noteTextOpChanged() {
	s.resetTextPassVisibility()
	s.textFormErr = ""
}

func (s *uiState) noteFileOpChanged() {
	s.resetFilePassVisibility()
	s.fileFormErr = ""
}

func (s *uiState) noteEncryptMode(file bool, next int) {
	if file {
		s.fileEncMode = next
		s.fileEncModeTouched = true
		s.showFileEncPass = false
		s.showFileEncPass2 = false
		s.fileFormErr = ""
		return
	}
	s.textEncMode = next
	s.textEncModeTouched = true
	s.showTextEncPass = false
	s.showTextEncPass2 = false
	s.textFormErr = ""
}

// stageEncrypt validates the studio that is about to encrypt and stores
// the credentials snapshot will hand to librnp. The visible fields stay
// as the user left them.
func (s *uiState) stageEncrypt(file bool) bool {
	mode := effectiveEncryptMode(s.textEncMode, s.textEncModeTouched, s.keys)
	hybrid := s.textEncHybrid
	pass, pass2 := s.encPassword, s.encPassword2
	if file {
		mode = effectiveEncryptMode(s.fileEncMode, s.fileEncModeTouched, s.keys)
		hybrid = s.fileEncHybrid
		pass, pass2 = s.fileEncPassword, s.fileEncPassword2
	}
	fps, password, errKey := resolveEncryptCredentials(mode, hybrid, s.encRecipients, pass, pass2)
	if errKey != "" {
		msg := s.tr(errKey)
		if file {
			s.fileFormErr = msg
		} else {
			s.textFormErr = msg
		}
		return false
	}
	s.textFormErr = ""
	s.fileFormErr = ""
	s.pendingEncSet = true
	s.pendingEncRecipients = strings.Join(fps, ", ")
	s.pendingEncPassword = password
	s.pendingOpPass = ""
	if s.encSign {
		local := s.textSignPass
		if file {
			local = s.fileSignPass
		}
		if local != "" {
			s.pendingOpPass = local
		}
	}
	return true
}

func (s *uiState) submitTextEncrypt() {
	if !s.stageEncrypt(false) {
		return
	}
	s.run("action.encrypt_text", s.encryptText)
}

func (s *uiState) submitFileEncrypt() {
	if !s.stageEncrypt(true) {
		return
	}
	s.run("action.encrypt_file", s.encryptFile)
}

func (s *uiState) submitTextDecrypt() {
	s.textFormErr = ""
	s.pendingOpPass = s.textDecryptPass
	s.run("action.decrypt_text", s.decryptText)
}

func (s *uiState) submitFileDecrypt() {
	s.fileFormErr = ""
	s.pendingOpPass = s.fileDecryptPass
	s.run("action.decrypt_file", s.decryptFile)
}

func (s *uiState) submitTextSign() {
	s.textFormErr = ""
	s.pendingOpPass = s.textSignPass
	s.run("action.sign_text", s.signText)
}

func (s *uiState) submitFileSign() {
	s.fileFormErr = ""
	s.pendingOpPass = s.fileSignPass
	s.run("action.sign_file", s.signFile)
}

func (s *uiState) sendTextToDecrypt() {
	s.textDecryptInput = s.textOutput
	s.textOp = studioOpDecrypt
	s.noteTextOpChanged()
}

func (s *uiState) sendTextToVerify() {
	if s.textMode == string(rnp.SignModeDetached) {
		s.textVerifyInput = s.textInput
		s.textVerifyDetached = s.textOutput
	} else {
		s.textVerifyInput = s.textOutput
		s.textVerifyDetached = ""
	}
	s.textOp = studioOpVerify
	s.noteTextOpChanged()
}

func encryptActionKey(mode int, hybrid, file bool) string {
	switch {
	case hybrid && file:
		return "btn.encrypt_file_hybrid"
	case hybrid:
		return "btn.encrypt_text_hybrid"
	case mode == encModePubKey && file:
		return "btn.encrypt_file_pubkey"
	case mode == encModePubKey:
		return "btn.encrypt_text_pubkey"
	case file:
		return "btn.encrypt_file_password"
	default:
		return "btn.encrypt_text_password"
	}
}
