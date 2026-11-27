package main

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/r404r/RnpArmor/internal/rnp"
)

const yearSeconds = 365 * 24 * 60 * 60

// keyTableRow is one flat row in the key table. A subkey row keeps PrimaryFP
// so selecting it does not replace the selected primary key.
type keyTableRow struct {
	Key       rnp.KeySummary
	PrimaryFP string
	Sub       bool
}

// buildKeyTableRows lists each primary, then its subkeys while that primary
// is expanded. A missing collapsed entry means expanded.
func buildKeyTableRows(primaries []rnp.KeySummary, collapsed map[string]bool) []keyTableRow {
	rows := make([]keyTableRow, 0, len(primaries))
	for _, p := range primaries {
		fp := p.Fingerprint
		rows = append(rows, keyTableRow{Key: p, PrimaryFP: fp, Sub: false})
		if keyCollapsed(collapsed, fp) {
			continue
		}
		for _, sub := range p.Subkeys {
			rows = append(rows, keyTableRow{Key: sub, PrimaryFP: fp, Sub: true})
		}
	}
	return rows
}

func keyCollapsed(collapsed map[string]bool, fp string) bool {
	return collapsed != nil && collapsed[fp]
}

func collapseSig(collapsed map[string]bool) string {
	if len(collapsed) == 0 {
		return ""
	}
	keys := make([]string, 0, len(collapsed))
	for fp, on := range collapsed {
		if on {
			keys = append(keys, fp)
		}
	}
	if len(keys) == 0 {
		return ""
	}
	// Stable signature so a map iteration does not look like a new filter.
	for i := 1; i < len(keys); i++ {
		j := i
		for j > 0 && keys[j] < keys[j-1] {
			keys[j], keys[j-1] = keys[j-1], keys[j]
			j--
		}
	}
	return strings.Join(keys, ",")
}

// selectionForTableRow maps a click to the primary fingerprint that the rest
// of the app uses, plus the subkey fingerprint when the row is a subkey.
func selectionForTableRow(row keyTableRow) (primaryFP, subFP string) {
	if row.Sub {
		return row.PrimaryFP, row.Key.Fingerprint
	}
	return row.Key.Fingerprint, ""
}

func (s *uiState) toggleCollapsed(fp string) {
	if strings.TrimSpace(fp) == "" {
		return
	}
	if s.collapsedKeys == nil {
		s.collapsedKeys = map[string]bool{}
	}
	if s.collapsedKeys[fp] {
		delete(s.collapsedKeys, fp)
		return
	}
	s.collapsedKeys[fp] = true
}

// expirationFromNow converts "years from now" into the seconds-from-creation
// value librnp stores. Zero years means the key does not expire.
func expirationFromNow(createdUnix uint32, now time.Time, years int) uint32 {
	if years <= 0 {
		return 0
	}
	span := uint64(years) * yearSeconds
	if createdUnix == 0 || now.Unix() <= int64(createdUnix) {
		if span > math.MaxUint32 {
			return math.MaxUint32
		}
		return uint32(span)
	}
	total := uint64(now.Unix()-int64(createdUnix)) + span
	if total > math.MaxUint32 {
		return math.MaxUint32
	}
	return uint32(total)
}

func subkeyUsageIDs() []string {
	return []string{"encrypt", "sign", "authenticate"}
}

func subkeyUsageLabelKey(id string) string {
	switch id {
	case "sign":
		return "usage.sign"
	case "authenticate":
		return "usage.auth"
	default:
		return "usage.encrypt"
	}
}

func subkeyUsageDescKey(id string) string {
	return subkeyUsageLabelKey(id) + ".desc"
}

func legalSubkeyAlgs(usage string) []string {
	switch usage {
	case "sign", "authenticate":
		return []string{"EDDSA", "RSA", "ECDSA"}
	default:
		return []string{"ECDH", "RSA", "ELGAMAL"}
	}
}

func addSubkeyAlgTipKey(alg string) string {
	switch strings.ToUpper(strings.TrimSpace(alg)) {
	case "RSA":
		return "tip.addsub.rsa"
	case "ECDSA":
		return "tip.addsub.ecdsa"
	case "ELGAMAL":
		return "tip.addsub.elgamal"
	case "EDDSA":
		return "tip.addsub.eddsa"
	default:
		return "tip.addsub.ecdh"
	}
}

// syncAddedSubkey drops a curve or size the usage cannot use, and replaces
// an illegal algorithm with the recommended one for that usage.
func syncAddedSubkey(usage, alg, curve string, bits int) (string, string, int) {
	usage = normalizeUIUsage(usage)
	alg = strings.ToUpper(strings.TrimSpace(alg))
	if !rnp.UsageAllowsAlg(usage, alg) {
		alg = ""
	}
	switch usage {
	case "sign", "authenticate":
		switch alg {
		case "RSA":
			curve = ""
			if !bitsAllowed(bits, 2048, 3072, 4096) {
				bits = 4096
			}
		case "ECDSA":
			bits = 0
			if !oneOf(curve, "NIST P-256", "NIST P-384", "NIST P-521") {
				curve = "NIST P-256"
			}
		default:
			alg = "EDDSA"
			curve = "Ed25519"
			bits = 0
		}
	default:
		switch alg {
		case "RSA":
			curve = ""
			if !bitsAllowed(bits, 2048, 3072, 4096) {
				bits = 4096
			}
		case "ELGAMAL":
			curve = ""
			if !bitsAllowed(bits, 2048, 3072, 4096) {
				bits = 3072
			}
		default:
			alg = "ECDH"
			bits = 0
			if !oneOf(curve, "Curve25519", "NIST P-256", "NIST P-384", "NIST P-521") {
				curve = "Curve25519"
			}
		}
	}
	return alg, curve, bits
}

func normalizeUIUsage(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "sign":
		return "sign"
	case "authenticate":
		return "authenticate"
	default:
		return "encrypt"
	}
}

func subkeyTypeText(lang Lang) string {
	return "  └ " + Tr(lang, "table.subkey")
}

func subkeyUsageText(lang Lang, k rnp.KeySummary) string {
	return "  " + Tr(lang, subkeyUsageRowKey(k))
}

func subkeyUsageRowKey(k rnp.KeySummary) string {
	switch {
	case k.CanEncrypt:
		return "table.use.encrypt"
	case k.CanAuth:
		return "table.use.auth"
	case k.CanSign:
		return "table.use.sign"
	default:
		return "table.use.sign"
	}
}

func validateAddSubkey(usage, alg, curve, hash, cipher string, bits, expiry int, typed, sessionPass string, primaryProtected bool) (rnp.SubkeyGenRequest, string) {
	usage = normalizeUIUsage(usage)
	alg, curve, bits = syncAddedSubkey(usage, alg, curve, bits)
	if !rnp.UsageAllowsAlg(usage, alg) {
		return rnp.SubkeyGenRequest{}, "err.subkey_alg"
	}
	if primaryProtected && typed == "" && sessionPass == "" {
		return rnp.SubkeyGenRequest{}, "err.pass_empty"
	}
	if strings.TrimSpace(hash) == "" {
		hash = "SHA256"
	}
	if strings.TrimSpace(cipher) == "" {
		cipher = "AES256"
	}
	return rnp.SubkeyGenRequest{
		Usage:         usage,
		Algorithm:     alg,
		Bits:          bits,
		Curve:         curve,
		HashAlg:       hash,
		Cipher:        cipher,
		ExpirationSec: expirySeconds(expiry),
		Unprotected:   !primaryProtected && typed == "",
	}, ""
}

// primaryProtected is false only when the selected primary secret is stored
// without a passphrase. A missing or public key still requires one.
func (s *uiState) primaryProtected() bool {
	fp := strings.TrimSpace(s.selected)
	if fp == "" {
		return true
	}
	for _, k := range s.keys {
		if k.Fingerprint != fp || !isPrimaryRow(k) {
			continue
		}
		return !(k.HasSecret && !k.IsProtected)
	}
	return true
}

func subkeyPassPlaceholder(s *uiState, primaryProtected bool) string {
	if !primaryProtected {
		return s.tr("placeholder.pass_optional_unprotected")
	}
	if s.passphrase != "" {
		return s.tr("placeholder.pass_reuse")
	}
	return s.tr("placeholder.pass_required")
}

func (s *uiState) openAddSubkeyWindow() {
	if strings.TrimSpace(s.selected) == "" {
		s.lastErr = s.tr("err.no_key_selected")
		s.invalidateAll()
		return
	}
	if s.subUsage == "" {
		s.subUsage = "encrypt"
	}
	s.subAlg, s.subCurve, s.subBits = syncAddedSubkey(s.subUsage, s.subAlg, s.subCurve, s.subBits)
	if s.subHash == "" {
		s.subHash = "SHA256"
	}
	if s.subCipher == "" {
		s.subCipher = "AES256"
	}
	s.resetAddSubPassUI()
	already := s.addSubWin != nil && !s.addSubWin.IsDestroyed()
	s.openChild(s.tr("title.add_subkey"), 560, 680, s.viewAddSubkey, &s.addSubWin)
	if !already && s.addSubWin != nil && !s.addSubWin.IsDestroyed() {
		s.addSubWin.OnClosed(func() {
			s.resetAddSubPassUI()
		})
	}
}

func (s *uiState) closeAddSubWindow() {
	s.resetAddSubPassUI()
	if s.addSubWin != nil && !s.addSubWin.IsDestroyed() {
		s.addSubWin.Close()
	}
}

func (s *uiState) resetAddSubPassUI() {
	s.subShowPass = false
	s.subErr = ""
}

func (s *uiState) openExpiryWindow(fp string, created uint32) {
	s.expiryTarget = fp
	s.expiryCreated = created
	s.resetExpiryPassUI()
	already := s.expiryWin != nil && !s.expiryWin.IsDestroyed()
	s.openChild(s.tr("title.subkey_expiry"), 480, 420, s.viewSubkeyExpiry, &s.expiryWin)
	if !already && s.expiryWin != nil && !s.expiryWin.IsDestroyed() {
		s.expiryWin.OnClosed(func() {
			s.resetExpiryPassUI()
		})
	}
}

func (s *uiState) closeExpiryWindow() {
	s.resetExpiryPassUI()
	if s.expiryWin != nil && !s.expiryWin.IsDestroyed() {
		s.expiryWin.Close()
	}
}

func (s *uiState) resetExpiryPassUI() {
	s.expiryShowPass = false
	s.expiryErr = ""
}

func (s *uiState) openRevokeWindow(fp string) {
	s.revokeTarget = fp
	if s.revokeCode == "" {
		s.revokeCode = "retired"
	}
	s.resetRevokePassUI()
	already := s.revokeWin != nil && !s.revokeWin.IsDestroyed()
	s.openChild(s.tr("title.subkey_revoke"), 500, 460, s.viewRevokeSubkey, &s.revokeWin)
	if !already && s.revokeWin != nil && !s.revokeWin.IsDestroyed() {
		s.revokeWin.OnClosed(func() {
			s.resetRevokePassUI()
		})
	}
}

func (s *uiState) closeRevokeWindow() {
	s.resetRevokePassUI()
	if s.revokeWin != nil && !s.revokeWin.IsDestroyed() {
		s.revokeWin.Close()
	}
}

func (s *uiState) resetRevokePassUI() {
	s.revokeShowPass = false
	s.revokeErr = ""
}

func (s *uiState) viewAddSubkey(c *ui.Context) {
	prepareSurface(c)
	t := c.Theme()
	ids := subkeyUsageIDs()
	labels := make([]string, len(ids))
	for i, id := range ids {
		labels[i] = s.tr(subkeyUsageLabelKey(id))
	}
	if s.subUsage == "" {
		s.subUsage = "encrypt"
	}
	curUsage := s.tr(subkeyUsageLabelKey(s.subUsage))
	s.subAlg, s.subCurve, s.subBits = syncAddedSubkey(s.subUsage, s.subAlg, s.subCurve, s.subBits)
	choices := subkeyExpiryChoices(s)
	if s.subExpiry < 0 || s.subExpiry >= len(choices) {
		s.subExpiry = 0
	}
	curExp := choices[s.subExpiry]
	primaryProtected := s.primaryProtected()
	passPh := subkeyPassPlaceholder(s, primaryProtected)

	ui.Column(c).Fill().Padding(12).Children(func() {
		rnpCard(c).Fill().Children(func() {
			ui.Scroll(c).Fill().Children(func() {
				ui.Column(c).Padding(16).Gap(8).Children(func() {
					ui.Text(c, s.tr("title.add_subkey")).FontSize(18).Bold()
					ui.Form(c, func() {
						ui.Field(c, s.tr("label.subkey_usage"), func() {
							ui.Select(c, &curUsage, labels).Tooltip(s.tr(subkeyUsageDescKey(s.subUsage)))
							if id := presetIDForLabel(curUsage, labels, ids); id != "" && id != s.subUsage {
								s.subUsage = id
								s.subAlg, s.subCurve, s.subBits = syncAddedSubkey(s.subUsage, "", "", 0)
							}
						}).Description(s.tr(subkeyUsageDescKey(s.subUsage)))
						ui.Field(c, s.tr("label.subkey_alg"), func() {
							s.addSubkeyAlgControls(c)
						}).Description(s.tr(addSubkeyAlgTipKey(s.subAlg)))
						ui.Field(c, s.tr("label.expiration"), func() {
							ui.Select(c, &curExp, choices).Tooltip(s.tr("tip.addsub.expiry"))
						}).Description(s.tr("tip.addsub.expiry"))
						ui.Field(c, s.tr("label.session_pass"), func() {
							s.passphraseRow(c, &s.subPass, &s.subShowPass, passPh)
						})
					})
					for i, label := range choices {
						if curExp == label {
							s.subExpiry = i
						}
					}
					ui.Collapsible(c, s.tr("label.advanced"), &s.subAdvanced, func() {
						s.addSubkeyCryptoControls(c)
					})
					if s.subErr != "" {
						ui.Text(c, s.subErr).TextColor(t.Danger)
					}
					ui.Row(c).Gap(8).Children(func() {
						if rnpPrimaryButton(c, s.tr("btn.add_subkey_do")).Clicked() {
							req, errKey := validateAddSubkey(s.subUsage, s.subAlg, s.subCurve, s.subHash, s.subCipher, s.subBits, s.subExpiry, s.subPass, s.passphrase, primaryProtected)
							if errKey != "" {
								s.subErr = s.tr(errKey)
								return
							}
							if s.subPass != "" {
								s.adoptPassphrase(s.subPass)
							}
							s.subErr = ""
							s.subShowPass = false
							s.subReq = req
							s.run("action.add_subkey", s.addSubkey)
						}
						if rnpButton(c, s.tr("btn.close")).Clicked() {
							s.closeAddSubWindow()
						}
					})
				})
			})
		})
	})
}

func (s *uiState) addSubkeyAlgControls(c *ui.Context) {
	algs := legalSubkeyAlgs(s.subUsage)
	ui.Row(c).Gap(8).Children(func() {
		ui.Select(c, &s.subAlg, algs).Width(140).Tooltip(s.tr(addSubkeyAlgTipKey(s.subAlg)))
		s.subAlg, s.subCurve, s.subBits = syncAddedSubkey(s.subUsage, s.subAlg, s.subCurve, s.subBits)
		switch s.subAlg {
		case "RSA":
			bitsSelect(c, &s.subBits, []int{2048, 3072, 4096})
		case "ELGAMAL":
			bitsSelect(c, &s.subBits, []int{2048, 3072, 4096})
		case "ECDSA":
			ui.Select(c, &s.subCurve, []string{"NIST P-256", "NIST P-384", "NIST P-521"}).Width(160)
		case "ECDH":
			ui.Select(c, &s.subCurve, []string{"Curve25519", "NIST P-256", "NIST P-384", "NIST P-521"}).Width(160)
		default:
			fixed := "Ed25519"
			ui.TextInput(c, &fixed).ReadOnly(true).Width(140)
		}
	})
}

func (s *uiState) addSubkeyCryptoControls(c *ui.Context) {
	hashes := []string{"SHA256", "SHA384", "SHA512"}
	ciphers := []string{"AES256", "AES192", "AES128", "Camellia256"}
	if !oneOf(s.subHash, hashes...) {
		s.subHash = "SHA256"
	}
	if !oneOf(s.subCipher, ciphers...) {
		s.subCipher = "AES256"
	}
	ui.Field(c, s.tr("label.crypto"), func() {
		ui.Row(c).Gap(8).Children(func() {
			ui.Select(c, &s.subHash, hashes).Width(160).Tooltip(s.tr("tip.crypto.role"))
			ui.Select(c, &s.subCipher, ciphers).Width(160).Tooltip(s.tr("tip.crypto.role"))
		})
	}).Description(s.tr("tip.crypto.desc"))
}

func subkeyExpiryChoices(s *uiState) []string {
	return []string{s.tr("exp.never_sub"), s.tr("exp.plus1"), s.tr("exp.plus2"), s.tr("exp.plus5")}
}

func (s *uiState) viewSubkeyExpiry(c *ui.Context) {
	prepareSurface(c)
	t := c.Theme()
	choices := subkeyExpiryChoices(s)
	if s.expiryChoice < 0 || s.expiryChoice >= len(choices) {
		s.expiryChoice = 0
	}
	cur := choices[s.expiryChoice]
	primaryProtected := s.primaryProtected()
	passPh := subkeyPassPlaceholder(s, primaryProtected)
	ui.Column(c).Fill().Padding(12).Children(func() {
		rnpCard(c).Fill().Padding(16).Gap(8).Children(func() {
			ui.Text(c, s.tr("title.subkey_expiry")).FontSize(18).Bold()
			ui.Text(c, s.tr("hint.expiry_subkey")).TextColor(t.TextMuted)
			ui.Field(c, s.tr("label.expiration"), func() {
				ui.Select(c, &cur, choices).Tooltip(s.tr("tip.addsub.expiry"))
			}).Description(s.tr("tip.addsub.expiry"))
			for i, label := range choices {
				if cur == label {
					s.expiryChoice = i
				}
			}
			ui.Field(c, s.tr("label.session_pass"), func() {
				s.passphraseRow(c, &s.expiryPass, &s.expiryShowPass, passPh)
			})
			if s.expiryErr != "" {
				ui.Text(c, s.expiryErr).TextColor(t.Danger)
			}
			ui.Row(c).Gap(8).Children(func() {
				if rnpPrimaryButton(c, s.tr("btn.apply_expiry")).Clicked() {
					if primaryProtected && s.expiryPass == "" && s.passphrase == "" {
						s.expiryErr = s.tr("err.pass_empty")
						return
					}
					if strings.TrimSpace(s.expiryTarget) == "" {
						s.expiryErr = s.tr("err.no_key_selected")
						return
					}
					if s.expiryPass != "" {
						s.adoptPassphrase(s.expiryPass)
					}
					years := 0
					switch s.expiryChoice {
					case 1:
						years = 1
					case 2:
						years = 2
					case 3:
						years = 5
					}
					s.expiryErr = ""
					s.expiryShowPass = false
					s.targetFP = s.expiryTarget
					s.expirySeconds = expirationFromNow(s.expiryCreated, time.Now().UTC(), years)
					s.run("action.set_key_expiry", s.setKeyExpiry)
				}
				if rnpButton(c, s.tr("btn.close")).Clicked() {
					s.closeExpiryWindow()
				}
			})
		})
	})
}

func revokeCodeIDs() []string {
	return []string{"retired", "superseded", "compromised", "no"}
}

func revokeCodeLabelKey(id string) string {
	switch id {
	case "superseded":
		return "reason.superseded"
	case "compromised":
		return "reason.compromised"
	case "no":
		return "reason.no"
	default:
		return "reason.retired"
	}
}

func (s *uiState) viewRevokeSubkey(c *ui.Context) {
	prepareSurface(c)
	t := c.Theme()
	ids := revokeCodeIDs()
	labels := make([]string, len(ids))
	for i, id := range ids {
		labels[i] = s.tr(revokeCodeLabelKey(id))
	}
	if s.revokeCode == "" {
		s.revokeCode = "retired"
	}
	cur := s.tr(revokeCodeLabelKey(s.revokeCode))
	passPh := subkeyPassPlaceholder(s, s.primaryProtected())
	ui.Column(c).Fill().Padding(12).Children(func() {
		rnpCard(c).Fill().Padding(16).Gap(8).Children(func() {
			ui.Text(c, s.tr("title.subkey_revoke")).FontSize(18).Bold()
			ui.Text(c, s.tr("hint.revoke_subkey")).TextColor(t.Danger)
			ui.Field(c, s.tr("label.revoke_code"), func() {
				ui.Select(c, &cur, labels)
				if id := presetIDForLabel(cur, labels, ids); id != "" {
					s.revokeCode = id
				}
			})
			ui.Field(c, s.tr("label.revoke_reason"), func() {
				ui.TextInput(c, &s.revokeReason).Placeholder(s.tr("label.revoke_reason"))
			})
			ui.Field(c, s.tr("label.session_pass"), func() {
				s.passphraseRow(c, &s.revokePass, &s.revokeShowPass, passPh)
			})
			if s.revokeErr != "" {
				ui.Text(c, s.revokeErr).TextColor(t.Danger)
			}
			ui.Row(c).Gap(8).Children(func() {
				if rnpDangerButton(c, s.tr("btn.subkey_revoke_do")).Clicked() {
					s.confirmRevokeSubkey()
				}
				if rnpButton(c, s.tr("btn.close")).Clicked() {
					s.closeRevokeWindow()
				}
			})
		})
	})
}

func (s *uiState) confirmRevokeSubkey() {
	if s.primaryProtected() && s.revokePass == "" && s.passphrase == "" {
		s.revokeErr = s.tr("err.pass_empty")
		return
	}
	if strings.TrimSpace(s.revokeTarget) == "" {
		s.revokeErr = s.tr("err.no_key_selected")
		return
	}
	code := s.revokeCode
	if code == "" {
		code = "retired"
	}
	reason := s.revokeReason
	target := s.revokeTarget
	typed := s.revokePass
	parent := s.revokeWin
	if parent == nil || parent.IsDestroyed() {
		parent = s.mainWin
	}
	title := s.tr("dlg.revoke_subkey_title")
	msg := s.tr("dlg.revoke_subkey")
	ok := s.tr("btn.subkey_revoke_do")
	cancel := s.tr("btn.cancel")
	s.revokeErr = ""
	go func() {
		res, err := mygo.Dialog.Message(mygo.MessageOptions{
			Parent:             parent,
			Type:               mygo.MessageWarning,
			Title:              title,
			Message:            msg,
			Detail:             target,
			Buttons:            []string{ok, cancel},
			CancelButton:       1,
			DestructiveButtons: []int{0},
		})
		if err != nil {
			s.reportAsyncErr(err)
			return
		}
		if !confirmButton(res.Button) {
			return
		}
		mygo.RunOnMain(func() {
			if typed != "" {
				s.adoptPassphrase(typed)
			}
			s.revokeShowPass = false
			s.revokeCode = code
			s.revokeReason = reason
			s.targetFP = target
			s.run("action.revoke_key", s.revokeKey)
		})
	}()
}

func (s *uiState) confirmDeleteSubkey(fp string) {
	fp = strings.TrimSpace(fp)
	if fp == "" {
		s.lastErr = s.tr("err.no_key_selected")
		s.invalidateAll()
		return
	}
	parent := s.mainWin
	title := s.tr("dlg.delete_subkey_title")
	msg := s.tr("dlg.delete_subkey")
	del := s.tr("btn.delete")
	cancel := s.tr("btn.cancel")
	go func() {
		res, err := mygo.Dialog.Message(mygo.MessageOptions{
			Parent:             parent,
			Type:               mygo.MessageWarning,
			Title:              title,
			Message:            msg,
			Detail:             fp,
			Buttons:            []string{del, cancel},
			CancelButton:       1,
			DestructiveButtons: []int{0},
		})
		if err != nil {
			s.reportAsyncErr(err)
			return
		}
		if !confirmButton(res.Button) {
			return
		}
		mygo.RunOnMain(func() {
			s.targetFP = fp
			s.run("action.delete_subkey", s.deleteSubkey)
		})
	}()
}

func (s *uiState) viewSubkeyCard(c *ui.Context, parentHasSecret bool, sub rnp.KeySummary) {
	t := c.Theme()
	selected := s.selectedSubkey != "" && s.selectedSubkey == sub.Fingerprint
	ui.Column(c).Gap(4).Padding(4, 0, 4, 0).Children(func() {
		ui.Row(c).Gap(6).Children(func() {
			label := s.tr("label.subkey")
			text := ui.Text(c, label)
			if selected {
				text.Bold().TextColor(t.Accent)
			}
			if sub.CanEncrypt {
				ui.Badge(c, s.tr("badge.encrypt"))
			}
			if sub.CanSign {
				ui.Badge(c, s.tr("badge.sign"))
			}
			if sub.CanAuth {
				ui.Badge(c, s.tr("badge.auth"))
			}
			if sub.HasSecret && !sub.IsProtected {
				ui.Badge(c, s.tr("badge.unprotected"))
			}
			ui.Badge(c, s.tr("badge."+keyStatusCode(sub, time.Now().UTC())))
		})
		ui.Textf(c, "%s   %s", sub.Algorithm, keyIDText(sub.Fingerprint, sub.KeyID, keyIDModeShort)).FontSize(12)
		ui.Textf(c, "%s: %s    %s: %s",
			s.tr("label.created"), formatKeyDate(sub.CreatedUnix, s.tr("label.never")),
			s.tr("label.expires"), formatKeyDate(sub.ExpiresUnix, s.tr("label.never")),
		).FontSize(12).TextColor(t.TextMuted)
		if selected {
			grouped := groupFingerprint(sub.Fingerprint)
			ui.Text(c, grouped).FontSize(12)
			if rnpButton(c, s.tr("btn.copy_fingerprint")).Clicked() {
				s.copyFingerprintText(grouped)
			}
		}
		if parentHasSecret {
			fp := sub.Fingerprint
			created := sub.CreatedUnix
			revoked := sub.IsRevoked
			ui.Row(c).Gap(6).Children(func() {
				if rnpButton(c, s.tr("btn.subkey_expiry")).Clicked() {
					s.openExpiryWindow(fp, created)
				}
				if !revoked && rnpDangerButton(c, s.tr("btn.subkey_revoke")).Clicked() {
					s.openRevokeWindow(fp)
				}
				if rnpDangerButton(c, s.tr("btn.subkey_delete")).Clicked() {
					s.confirmDeleteSubkey(fp)
				}
			})
		}
	})
}

func (s *uiState) addSubkey(sess *rnp.Session, ctx context.Context, in actionInput) (actionResult, error) {
	fp := strings.TrimSpace(in.selected)
	if fp == "" {
		return actionResult{}, fmt.Errorf("%s", in.tr("err.no_key_selected"))
	}
	sum, err := sess.AddSubkey(ctx, fp, in.subReq)
	if err != nil {
		return actionResult{}, err
	}
	res := actionResult{persist: persistUpdate}
	if sum.Fingerprint != "" {
		res.hasSelectSubkey = true
		res.selectSubkey = sum.Fingerprint
	}
	return res, nil
}

func (s *uiState) setKeyExpiry(sess *rnp.Session, ctx context.Context, in actionInput) (actionResult, error) {
	fp := strings.TrimSpace(in.targetFP)
	if fp == "" {
		return actionResult{}, fmt.Errorf("%s", in.tr("err.no_key_selected"))
	}
	if err := sess.SetKeyExpiration(ctx, fp, in.expirySeconds); err != nil {
		return actionResult{}, err
	}
	return actionResult{persist: persistUpdate}, nil
}

func (s *uiState) revokeKey(sess *rnp.Session, ctx context.Context, in actionInput) (actionResult, error) {
	fp := strings.TrimSpace(in.targetFP)
	if fp == "" {
		return actionResult{}, fmt.Errorf("%s", in.tr("err.no_key_selected"))
	}
	if err := sess.RevokeKey(ctx, fp, in.revokeCode, in.revokeReason); err != nil {
		return actionResult{}, err
	}
	return actionResult{persist: persistUpdate}, nil
}

func (s *uiState) deleteSubkey(sess *rnp.Session, ctx context.Context, in actionInput) (actionResult, error) {
	fp := strings.TrimSpace(in.targetFP)
	if fp == "" {
		return actionResult{}, fmt.Errorf("%s", in.tr("err.no_key_selected"))
	}
	if err := sess.DeleteSubkey(ctx, fp); err != nil {
		return actionResult{}, err
	}
	return actionResult{persist: persistUpdate}, nil
}
