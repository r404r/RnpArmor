package main

import (
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/r404r/RnpArmor/internal/rnp"
)

func (s *uiState) viewTextStudio(c *ui.Context) {
	s.ensureStudioSplit()
	if s.textOp < studioOpEncrypt || s.textOp > studioOpVerify {
		s.textOp = studioOpEncrypt
	}
	if ui.Segmented(c, &s.textOp,
		s.tr("op.encrypt"), s.tr("op.decrypt"), s.tr("op.sign"), s.tr("op.verify"),
	).FillEqually().Changed() {
		s.noteTextOpChanged()
	}
	ui.SplitPercent(c, &s.textSplitPct, func() {
		ui.Scroll(c).Fill().Children(func() {
			rnpCard(c).Padding(12).Gap(8).Children(func() {
				s.viewTextLeft(c)
			})
		})
	}, func() {
		rnpCard(c).Padding(12).Gap(8).Fill().Children(func() {
			s.viewTextRight(c)
		})
	}).Grow(1).MinHeight(0)
}

func (s *uiState) viewFileWorkbench(c *ui.Context) {
	s.ensureStudioSplit()
	if s.fileOp < studioOpEncrypt || s.fileOp > studioOpVerify {
		s.fileOp = studioOpEncrypt
	}
	if ui.Segmented(c, &s.fileOp,
		s.tr("op.encrypt"), s.tr("op.decrypt"), s.tr("op.sign"), s.tr("op.verify"),
	).FillEqually().Changed() {
		s.noteFileOpChanged()
	}
	ui.SplitPercent(c, &s.fileSplitPct, func() {
		ui.Scroll(c).Fill().Children(func() {
			rnpCard(c).Padding(12).Gap(8).Children(func() {
				s.viewFileLeft(c)
			})
		})
	}, func() {
		rnpCard(c).Padding(12).Gap(8).Fill().Children(func() {
			s.viewFileRight(c)
		})
	}).Grow(1).MinHeight(0)
}

func (s *uiState) viewTextLeft(c *ui.Context) {
	switch s.textOp {
	case studioOpDecrypt:
		s.labeledTools(c, "label.decrypt_input", &s.textDecryptInput)
		s.pgpBadge(c, s.textDecryptInput)
		ui.TextArea(c, &s.textDecryptInput).Height(140)
		s.optionalPassField(c, "label.decrypt_pass", "hint.decrypt_pass", &s.textDecryptPass, &s.showTextDecPass)
		s.formError(c, s.textFormErr)
		if rnpPrimaryButton(c, s.tr("btn.decrypt_text")).Clicked() {
			s.submitTextDecrypt()
		}
	case studioOpSign:
		s.labeledTools(c, "label.sign_input", &s.textInput)
		ui.TextArea(c, &s.textInput).Height(140)
		s.signModePicker(c)
		ui.Field(c, s.tr("label.signer_fingerprint"), func() { s.signerPicker(c) })
		s.optionalPassField(c, "label.sign_pass", "hint.decrypt_pass", &s.textSignPass, &s.showTextSignPass)
		s.formError(c, s.textFormErr)
		if rnpPrimaryButton(c, s.tr("btn.sign_text")).Clicked() {
			s.submitTextSign()
		}
	case studioOpVerify:
		s.labeledTools(c, "label.verify_input", &s.textVerifyInput)
		s.pgpBadge(c, s.textVerifyInput)
		ui.TextArea(c, &s.textVerifyInput).Height(120)
		ui.Field(c, s.tr("label.detached_sig"), func() {
			ui.TextArea(c, &s.textVerifyDetached).Height(80)
		})
		ui.Text(c, s.tr("hint.detached_only")).FontSize(12).TextColor(c.Theme().TextMuted)
		s.formError(c, s.textFormErr)
		if rnpPrimaryButton(c, s.tr("btn.verify_text")).Clicked() {
			s.textFormErr = ""
			s.run("action.verify_text", s.verifyText)
		}
	default:
		s.labeledTools(c, "label.encrypt_input", &s.textInput)
		ui.TextArea(c, &s.textInput).Height(140)
		s.encryptModeBlock(c, false)
		s.advancedEncrypt(c, false)
		s.formError(c, s.textFormErr)
		mode := effectiveEncryptMode(s.textEncMode, s.textEncModeTouched, s.keys)
		if rnpPrimaryButton(c, s.tr(encryptActionKey(mode, s.textEncHybrid, false))).Clicked() {
			s.submitTextEncrypt()
		}
	}
}

func (s *uiState) viewTextRight(c *ui.Context) {
	t := c.Theme()
	switch s.textOp {
	case studioOpDecrypt:
		ui.Text(c, s.tr("label.decrypt_result")).Bold()
		s.copyClearRow(c, &s.textOutput, "btn.copy_result", nil)
		ui.TextArea(c, &s.textOutput).Grow(1).MinHeight(160).Placeholder(s.tr("placeholder.result_output"))
	case studioOpSign:
		ui.Text(c, s.tr("label.sign_result")).Bold()
		s.copyClearRow(c, &s.textOutput, "btn.copy_result", func() {
			if rnpButton(c, s.tr("btn.send_verify")).Clicked() {
				s.sendTextToVerify()
			}
		})
		ui.TextArea(c, &s.textOutput).Grow(1).MinHeight(160).Placeholder(s.tr("placeholder.result_output"))
	case studioOpVerify:
		ui.Text(c, s.tr("label.verify_result")).Bold()
		s.copyClearRow(c, &s.textOutput, "btn.copy_report", nil)
		ui.Text(c, s.tr("hint.verify_badge")).FontSize(12).TextColor(t.TextMuted)
		ui.TextArea(c, &s.textOutput).Grow(1).MinHeight(160).Placeholder(s.tr("placeholder.result_output"))
	default:
		ui.Text(c, s.tr("label.result")).Bold()
		s.copyClearRow(c, &s.textOutput, "btn.copy_result", func() {
			if rnpButton(c, s.tr("btn.send_decrypt")).Clicked() {
				s.sendTextToDecrypt()
			}
		})
		ui.TextArea(c, &s.textOutput).Grow(1).MinHeight(160).Placeholder(s.tr("placeholder.result_output"))
		ui.Text(c, s.textEncryptSummary()).FontSize(12).TextColor(t.TextMuted)
	}
}

func (s *uiState) viewFileLeft(c *ui.Context) {
	switch s.fileOp {
	case studioOpDecrypt:
		ui.Field(c, s.tr("label.source"), func() {
			s.filePathEditors(c, &s.fileSrc, "dlg.browse_source", "", "")
		})
		ui.Field(c, s.tr("label.destination"), func() {
			s.filePathEditors(c, &s.fileDst, "", "dlg.save_destination", s.fileSrc)
		})
		s.optionalPassField(c, "label.decrypt_pass", "hint.decrypt_pass", &s.fileDecryptPass, &s.showFileDecPass)
		s.formError(c, s.fileFormErr)
		if rnpPrimaryButton(c, s.tr("btn.decrypt_file")).Clicked() {
			s.submitFileDecrypt()
		}
	case studioOpSign:
		ui.Field(c, s.tr("label.source"), func() {
			s.filePathEditors(c, &s.fileSrc, "dlg.browse_source", "", "")
		})
		ui.Field(c, s.tr("label.signature_path"), func() {
			s.filePathEditors(c, &s.fileSig, "", "dlg.save_signature", s.fileSrc)
		})
		s.signModePicker(c)
		ui.Field(c, s.tr("label.signer_fingerprint"), func() { s.signerPicker(c) })
		s.optionalPassField(c, "label.sign_pass", "hint.decrypt_pass", &s.fileSignPass, &s.showFileSignPass)
		s.formError(c, s.fileFormErr)
		if rnpPrimaryButton(c, s.tr("btn.sign_file")).Clicked() {
			s.submitFileSign()
		}
	case studioOpVerify:
		ui.Field(c, s.tr("label.source"), func() {
			s.filePathEditors(c, &s.fileSrc, "dlg.browse_source", "", "")
		})
		ui.Field(c, s.tr("label.signature_path"), func() {
			s.filePathEditors(c, &s.fileSig, "dlg.browse_signature", "", s.fileSrc)
		})
		s.labeledSwitch(c, &s.fileDetached, "toggle.detached_verify")
		s.formError(c, s.fileFormErr)
		if rnpPrimaryButton(c, s.tr("btn.verify_file")).Clicked() {
			s.fileFormErr = ""
			s.run("action.verify_file", s.verifyFile)
		}
	default:
		ui.Field(c, s.tr("label.source"), func() {
			s.filePathEditors(c, &s.fileSrc, "dlg.browse_source", "", "")
		})
		ui.Field(c, s.tr("label.destination"), func() {
			s.filePathEditors(c, &s.fileDst, "", "dlg.save_destination", s.fileSrc)
		})
		ui.Text(c, s.tr("hint.armor_asc")).FontSize(12).TextColor(c.Theme().TextMuted)
		s.encryptModeBlock(c, true)
		s.advancedEncrypt(c, true)
		s.formError(c, s.fileFormErr)
		mode := effectiveEncryptMode(s.fileEncMode, s.fileEncModeTouched, s.keys)
		if rnpPrimaryButton(c, s.tr(encryptActionKey(mode, s.fileEncHybrid, true))).Clicked() {
			s.submitFileEncrypt()
		}
	}
}

func (s *uiState) viewFileRight(c *ui.Context) {
	t := c.Theme()
	ui.Text(c, s.tr("label.file_summary")).Bold()
	ui.Text(c, s.fileTaskSummary()).FontSize(12).TextColor(t.TextMuted)
	copyKey := "btn.copy_result"
	if s.fileOp == studioOpVerify {
		copyKey = "btn.copy_report"
	}
	s.copyClearRow(c, &s.fileOutput, copyKey, nil)
	ui.TextArea(c, &s.fileOutput).Grow(1).MinHeight(160).Placeholder(s.tr("placeholder.result_output"))
}

func (s *uiState) labeledTools(c *ui.Context, labelKey string, dst *string) {
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Text(c, s.tr(labelKey)).Bold().Grow(1)
		if rnpButton(c, s.tr("btn.paste")).Clicked() && dst != nil {
			*dst = mygo.Clipboard.ReadText()
		}
		if rnpButton(c, s.tr("btn.clear")).Clicked() && dst != nil {
			*dst = ""
		}
	})
}

func (s *uiState) copyClearRow(c *ui.Context, text *string, copyKey string, extra func()) {
	ui.Row(c).Gap(8).Children(func() {
		if rnpButton(c, s.tr(copyKey)).Clicked() && text != nil {
			s.copyTextAndToast(*text, s.tr("toast.copied"))
		}
		if rnpButton(c, s.tr("btn.clear")).Clicked() && text != nil {
			*text = ""
		}
		if extra != nil {
			extra()
		}
	})
}

func (s *uiState) formError(c *ui.Context, msg string) {
	if strings.TrimSpace(msg) == "" {
		return
	}
	ui.Text(c, msg).TextColor(c.Theme().Danger)
}

func (s *uiState) pgpBadge(c *ui.Context, body string) {
	key := ""
	switch {
	case strings.Contains(body, "-----BEGIN PGP MESSAGE-----"):
		key = "badge.pgp_message"
	case strings.Contains(body, "-----BEGIN PGP SIGNED MESSAGE-----"), strings.Contains(body, "-----BEGIN PGP SIGNATURE-----"):
		key = "badge.pgp_signed"
	case strings.TrimSpace(body) != "":
		key = "badge.not_pgp"
	}
	if key == "" {
		return
	}
	ui.Badge(c, s.tr(key))
}

func (s *uiState) labeledSwitch(c *ui.Context, on *bool, key string) {
	ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Switch(c, on).Label(s.tr(key))
		ui.Text(c, s.tr(key))
	})
}

func (s *uiState) optionalPassField(c *ui.Context, labelKey, hintKey string, value *string, shown *bool) {
	ui.Field(c, s.tr(labelKey), func() {
		s.passphraseRow(c, value, shown, s.tr("placeholder.unlock_pass"))
	})
	ui.Text(c, s.tr(hintKey)).FontSize(12).TextColor(c.Theme().TextMuted)
}

func (s *uiState) passwordPair(c *ui.Context, file bool) {
	pass, pass2 := &s.encPassword, &s.encPassword2
	show, show2 := &s.showTextEncPass, &s.showTextEncPass2
	if file {
		pass, pass2 = &s.fileEncPassword, &s.fileEncPassword2
		show, show2 = &s.showFileEncPass, &s.showFileEncPass2
	}
	ui.Field(c, s.tr("label.enc_password"), func() {
		s.passphraseRow(c, pass, show, s.tr("placeholder.symm_pass"))
	})
	ui.Field(c, s.tr("label.enc_password2"), func() {
		s.passphraseRow(c, pass2, show2, s.tr("placeholder.enc_password2"))
	})
	if passConfirmMismatch(*pass, *pass2) {
		ui.Text(c, s.tr("hint.pass_mismatch_live")).TextColor(c.Theme().Danger)
	}
}

func (s *uiState) encryptModeBlock(c *ui.Context, file bool) {
	t := c.Theme()
	hybrid := s.textEncHybrid
	mode := effectiveEncryptMode(s.textEncMode, s.textEncModeTouched, s.keys)
	if file {
		hybrid = s.fileEncHybrid
		mode = effectiveEncryptMode(s.fileEncMode, s.fileEncModeTouched, s.keys)
	}
	ui.Text(c, s.tr("label.encrypt_mode")).Bold()
	if hybrid {
		ui.Text(c, s.tr("hint.hybrid_on")).FontSize(12).TextColor(t.TextMuted)
		s.recipientPicker(c)
		s.passwordPair(c, file)
		return
	}
	cur := mode
	if ui.Segmented(c, &cur, s.tr("mode.password"), s.tr("mode.pubkey")).FillEqually().Changed() {
		s.noteEncryptMode(file, cur)
		mode = cur
	}
	if mode == encModePubKey {
		ui.Text(c, s.tr("hint.mode_pubkey")).FontSize(12).TextColor(t.TextMuted)
		s.recipientPicker(c)
		return
	}
	ui.Text(c, s.tr("hint.mode_password")).FontSize(12).TextColor(t.TextMuted)
	s.passwordPair(c, file)
}

func (s *uiState) advancedEncrypt(c *ui.Context, file bool) {
	open := &s.textAdvOpen
	signPass := &s.textSignPass
	showSign := &s.showTextSignPass
	hybrid := &s.textEncHybrid
	if file {
		open = &s.fileAdvOpen
		signPass = &s.fileSignPass
		showSign = &s.showFileSignPass
		hybrid = &s.fileEncHybrid
	}
	ui.Collapsible(c, s.tr("label.studio_advanced"), open, func() {
		ui.Field(c, s.tr("label.cipher"), func() {
			ui.Select(c, &s.encCipher, []string{"AES256", "AES192", "AES128", "Camellia256"}).Width(180)
		})
		ui.Field(c, s.tr("label.compression"), func() {
			ui.Select(c, &s.encCompress, []string{"zlib", "bzip2", "zip", "uncompressed"}).Width(180)
		})
		s.labeledSwitch(c, &s.encSign, "toggle.encrypt_sign")
		if s.encSign {
			ui.Field(c, s.tr("label.signer_fingerprint"), func() { s.signerPicker(c) })
			ui.Field(c, s.tr("label.sign_pass"), func() {
				s.passphraseRow(c, signPass, showSign, s.tr("placeholder.unlock_pass"))
			})
		}
		wasHybrid := *hybrid
		s.labeledSwitch(c, hybrid, "toggle.hybrid")
		if *hybrid != wasHybrid {
			if file {
				s.showFileEncPass = false
				s.showFileEncPass2 = false
				s.fileFormErr = ""
			} else {
				s.showTextEncPass = false
				s.showTextEncPass2 = false
				s.textFormErr = ""
			}
		}
		ui.Text(c, s.tr("hint.hybrid_on")).FontSize(12).TextColor(c.Theme().TextMuted)
	})
}

func (s *uiState) signModePicker(c *ui.Context) {
	labels := []string{s.tr("signmode.cleartext"), s.tr("signmode.detached"), s.tr("signmode.inline")}
	values := []string{
		string(rnp.SignModeCleartext),
		string(rnp.SignModeDetached),
		string(rnp.SignModeInline),
	}
	idx := 1
	for i, v := range values {
		if s.textMode == v {
			idx = i
			break
		}
	}
	cur := labels[idx]
	ui.Field(c, s.tr("label.sign_mode"), func() {
		ui.Select(c, &cur, labels).Grow(1)
	})
	for i, label := range labels {
		if label == cur {
			s.textMode = values[i]
			break
		}
	}
	ui.Text(c, s.tr("hint.sign_modes")).FontSize(12).TextColor(c.Theme().TextMuted)
}

func (s *uiState) textEncryptSummary() string {
	return s.encryptSummary(false)
}

func (s *uiState) encryptSummary(file bool) string {
	hybrid := s.textEncHybrid
	mode := effectiveEncryptMode(s.textEncMode, s.textEncModeTouched, s.keys)
	if file {
		hybrid = s.fileEncHybrid
		mode = effectiveEncryptMode(s.fileEncMode, s.fileEncModeTouched, s.keys)
	}
	name := s.tr("mode.password")
	if hybrid {
		name = s.tr("toggle.hybrid")
	} else if mode == encModePubKey {
		name = s.tr("mode.pubkey")
	}
	cipher := s.encCipher
	if cipher == "" {
		cipher = "AES256"
	}
	comp := s.encCompress
	if comp == "" {
		comp = "zlib"
	}
	return name + " · " + cipher + " / " + comp
}

func (s *uiState) fileTaskSummary() string {
	switch s.fileOp {
	case studioOpDecrypt:
		return s.tr("op.decrypt") + "\n" + s.fileSrc + " → " + s.fileDst
	case studioOpSign:
		return s.tr("op.sign") + " · " + s.textMode + "\n" + s.fileSrc + " → " + s.fileSig
	case studioOpVerify:
		return s.tr("op.verify") + "\n" + s.fileSrc
	default:
		return s.encryptSummary(true) + "\n" + s.fileSrc + " → " + s.fileDst
	}
}
