package main

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"

	"github.com/r404r/RnpArmor/internal/rnp"
)

const (
	presetModern   = "modern"
	presetRSA4096  = "rsa4096"
	presetRSA3072  = "rsa3072"
	presetNISTP256 = "nist_p256"
	presetCustom   = "custom"
)

func presetIDs() []string {
	return []string{presetModern, presetRSA4096, presetRSA3072, presetNISTP256, presetCustom}
}

func presetLabelKey(id string) string {
	switch id {
	case presetRSA4096:
		return "preset.rsa4096"
	case presetRSA3072:
		return "preset.rsa3072"
	case presetNISTP256:
		return "preset.nist_p256"
	case presetCustom:
		return "preset.custom"
	default:
		return "preset.modern"
	}
}

func presetDescKey(id string) string {
	return presetLabelKey(id) + ".desc"
}

func primaryAlgTipKey(alg string) string {
	switch strings.ToUpper(strings.TrimSpace(alg)) {
	case "RSA":
		return "tip.primary.rsa"
	case "ECDSA":
		return "tip.primary.ecdsa"
	case "DSA":
		return "tip.primary.dsa"
	default:
		return "tip.primary.eddsa"
	}
}

func subkeyAlgTipKey(alg string) string {
	switch strings.ToUpper(strings.TrimSpace(alg)) {
	case "RSA":
		return "tip.subkey.rsa"
	case "ELGAMAL":
		return "tip.subkey.elgamal"
	default:
		return "tip.subkey.ecdh"
	}
}

func newKeyWindowSize() (width, height int) { return 560, 640 }

// applyKeyPreset fills the algorithm fields for a named profile.
// Identity fields on base are kept. custom only corrects an invalid curve or size.
func applyKeyPreset(id string, base rnp.KeyGenRequest) rnp.KeyGenRequest {
	if id == presetCustom {
		return syncSubkeyAlgParams(syncPrimaryAlgParams(base))
	}
	req := base
	switch id {
	case presetRSA4096:
		req.PrimaryAlg, req.PrimaryCurve, req.PrimaryBits = "RSA", "", 4096
		req.SubkeyAlg, req.SubkeyCurve, req.SubkeyBits = "RSA", "", 4096
	case presetRSA3072:
		req.PrimaryAlg, req.PrimaryCurve, req.PrimaryBits = "RSA", "", 3072
		req.SubkeyAlg, req.SubkeyCurve, req.SubkeyBits = "RSA", "", 3072
	case presetNISTP256:
		req.PrimaryAlg, req.PrimaryCurve, req.PrimaryBits = "ECDSA", "NIST P-256", 0
		req.SubkeyAlg, req.SubkeyCurve, req.SubkeyBits = "ECDH", "NIST P-256", 0
	default:
		req.PrimaryAlg, req.PrimaryCurve, req.PrimaryBits = "EDDSA", "Ed25519", 0
		req.SubkeyAlg, req.SubkeyCurve, req.SubkeyBits = "ECDH", "Curve25519", 0
	}
	req.HashAlg = "SHA256"
	req.SymmCipher = "AES256"
	return req
}

func presetForRequest(req rnp.KeyGenRequest) string {
	for _, id := range []string{presetModern, presetRSA4096, presetRSA3072, presetNISTP256} {
		if keygenSpecEqual(req, applyKeyPreset(id, req)) {
			return id
		}
	}
	return presetCustom
}

// notePresetDrift switches a named preset to custom when the request no
// longer matches it. The bool asks the advanced section to open.
func notePresetDrift(preset string, req rnp.KeyGenRequest) (string, bool) {
	if preset == presetCustom {
		return presetCustom, false
	}
	if strings.TrimSpace(preset) == "" {
		preset = presetModern
	}
	if keygenSpecEqual(req, applyKeyPreset(preset, req)) {
		return preset, false
	}
	return presetCustom, true
}

func keygenSpecEqual(a, b rnp.KeyGenRequest) bool {
	return a.PrimaryAlg == b.PrimaryAlg &&
		a.PrimaryBits == b.PrimaryBits &&
		a.PrimaryCurve == b.PrimaryCurve &&
		a.SubkeyAlg == b.SubkeyAlg &&
		a.SubkeyBits == b.SubkeyBits &&
		a.SubkeyCurve == b.SubkeyCurve &&
		a.HashAlg == b.HashAlg &&
		a.SymmCipher == b.SymmCipher
}

// syncPrimaryAlgParams drops a curve or size the new algorithm cannot use.
func syncPrimaryAlgParams(req rnp.KeyGenRequest) rnp.KeyGenRequest {
	switch strings.ToUpper(strings.TrimSpace(req.PrimaryAlg)) {
	case "RSA":
		req.PrimaryAlg = "RSA"
		req.PrimaryCurve = ""
		if !bitsAllowed(req.PrimaryBits, 2048, 3072, 4096) {
			req.PrimaryBits = 4096
		}
	case "DSA":
		req.PrimaryAlg = "DSA"
		req.PrimaryCurve = ""
		if !bitsAllowed(req.PrimaryBits, 2048, 3072) {
			req.PrimaryBits = 3072
		}
	case "ECDSA":
		req.PrimaryAlg = "ECDSA"
		req.PrimaryBits = 0
		if !oneOf(req.PrimaryCurve, "NIST P-256", "NIST P-384", "NIST P-521") {
			req.PrimaryCurve = "NIST P-256"
		}
	default:
		req.PrimaryAlg = "EDDSA"
		req.PrimaryCurve = "Ed25519"
		req.PrimaryBits = 0
	}
	return req
}

// syncSubkeyAlgParams drops a curve or size the encryption subkey cannot use.
func syncSubkeyAlgParams(req rnp.KeyGenRequest) rnp.KeyGenRequest {
	switch strings.ToUpper(strings.TrimSpace(req.SubkeyAlg)) {
	case "RSA":
		req.SubkeyAlg = "RSA"
		req.SubkeyCurve = ""
		if !bitsAllowed(req.SubkeyBits, 2048, 3072, 4096) {
			req.SubkeyBits = 4096
		}
	case "ELGAMAL":
		req.SubkeyAlg = "ELGAMAL"
		req.SubkeyCurve = ""
		if !bitsAllowed(req.SubkeyBits, 2048, 3072, 4096) {
			req.SubkeyBits = 3072
		}
	default:
		req.SubkeyAlg = "ECDH"
		req.SubkeyBits = 0
		if !oneOf(req.SubkeyCurve, "Curve25519", "NIST P-256", "NIST P-384", "NIST P-521") {
			req.SubkeyCurve = "Curve25519"
		}
	}
	return req
}

func bitsAllowed(v int, allowed ...int) bool {
	for _, n := range allowed {
		if v == n {
			return true
		}
	}
	return false
}

func oneOf(v string, allowed ...string) bool {
	for _, s := range allowed {
		if v == s {
			return true
		}
	}
	return false
}

func presetIDForLabel(label string, labels, ids []string) string {
	for i, l := range labels {
		if l == label && i < len(ids) {
			return ids[i]
		}
	}
	return ""
}

func (s *uiState) viewNewKey(c *ui.Context) {
	t := c.Theme()
	if strings.TrimSpace(s.newKeyPreset) == "" {
		s.newKeyPreset = presetForRequest(s.keyReq)
	}
	if next, open := notePresetDrift(s.newKeyPreset, s.keyReq); next != s.newKeyPreset {
		s.newKeyPreset = next
		if open {
			s.newKeyAdvanced = true
		}
	}
	choices := []string{s.tr("exp.never"), s.tr("exp.1y"), s.tr("exp.2y"), s.tr("exp.5y")}
	if s.newKeyExpiry < 0 || s.newKeyExpiry >= len(choices) {
		s.newKeyExpiry = 0
	}
	curExp := choices[s.newKeyExpiry]
	ids := presetIDs()
	labels := make([]string, len(ids))
	for i, id := range ids {
		labels[i] = s.tr(presetLabelKey(id))
	}
	curPreset := labels[0]
	for i, id := range ids {
		if id == s.newKeyPreset && i < len(labels) {
			curPreset = labels[i]
		}
	}

	ui.Column(c).Fill().Children(func() {
		ui.Scroll(c).Fill().Children(func() {
			ui.Column(c).Padding(16).Gap(8).Children(func() {
				ui.Text(c, s.tr("title.new_key")).FontSize(18).Bold()
				ui.Form(c, func() {
					ui.Field(c, s.tr("label.name"), func() {
						ui.TextInput(c, &s.newKeyName).Placeholder(s.tr("placeholder.name"))
					})
					ui.Field(c, s.tr("label.email"), func() {
						ui.TextInput(c, &s.newKeyEmail).Placeholder(s.tr("placeholder.email"))
					})
					ui.Field(c, s.tr("label.passphrase"), func() {
						ui.TextInput(c, &s.newKeyPass).Password().Placeholder(s.tr("placeholder.new_pass"))
					})
					ui.Field(c, s.tr("label.confirm_pass"), func() {
						ui.TextInput(c, &s.newKeyPass2).Password().Placeholder(s.tr("placeholder.confirm_pass"))
					})
					ui.Field(c, s.tr("label.expiration"), func() {
						ui.Select(c, &curExp, choices)
					})
					ui.Field(c, s.tr("label.alg_preset"), func() {
						ui.Select(c, &curPreset, labels).Tooltip(s.tr("tip.preset.roles"))
						if id := presetIDForLabel(curPreset, labels, ids); id != "" && id != s.newKeyPreset {
							s.newKeyPreset = id
							if id == presetCustom {
								s.newKeyAdvanced = true
							} else {
								s.keyReq = applyKeyPreset(id, s.keyReq)
							}
						}
					}).Description(s.tr(presetDescKey(s.newKeyPreset)))
				})
				for i, label := range choices {
					if curExp == label {
						s.newKeyExpiry = i
					}
				}
				ui.Collapsible(c, s.tr("label.advanced"), &s.newKeyAdvanced, func() {
					ui.Field(c, s.tr("label.primary"), func() {
						s.primaryAlgControls(c)
					}).Description(s.tr(primaryAlgTipKey(s.keyReq.PrimaryAlg)))
					ui.Field(c, s.tr("label.enc_subkey"), func() {
						s.subkeyAlgControls(c)
					}).Description(s.tr(subkeyAlgTipKey(s.keyReq.SubkeyAlg)))
					ui.Field(c, s.tr("label.crypto"), func() {
						s.cryptoControls(c)
					}).Description(s.tr("tip.crypto.desc"))
				})
				if s.newKeyErr != "" {
					ui.Text(c, s.newKeyErr).TextColor(t.Danger)
				}
				ui.Row(c).Gap(8).Children(func() {
					if ui.PrimaryButton(c, s.tr("btn.generate")).Clicked() {
						req, errKey := validateNewKey(s.newKeyName, s.newKeyEmail, s.newKeyPass, s.newKeyPass2, s.newKeyExpiry, s.keyReq)
						if errKey != "" {
							s.newKeyErr = s.tr(errKey)
							return
						}
						s.newKeyErr = ""
						s.adoptPassphrase(s.newKeyPass)
						s.keyReq = req
						s.run("action.generate_key", s.generateKey)
					}
					if ui.Button(c, s.tr("btn.close")).Clicked() {
						s.closeNewKeyWindow()
					}
				})
			})
		})
	})
}

func (s *uiState) primaryAlgControls(c *ui.Context) {
	algs := []string{"EDDSA", "RSA", "ECDSA", "DSA"}
	ui.Row(c).Gap(8).Children(func() {
		ui.Select(c, &s.keyReq.PrimaryAlg, algs).Width(140).Tooltip(s.tr("tip.primary.role"))
		s.keyReq = syncPrimaryAlgParams(s.keyReq)
		switch s.keyReq.PrimaryAlg {
		case "RSA":
			bitsSelect(c, &s.keyReq.PrimaryBits, []int{2048, 3072, 4096})
		case "DSA":
			bitsSelect(c, &s.keyReq.PrimaryBits, []int{2048, 3072})
		case "ECDSA":
			ui.Select(c, &s.keyReq.PrimaryCurve, []string{"NIST P-256", "NIST P-384", "NIST P-521"}).Width(160)
		default:
			fixed := "Ed25519"
			ui.TextInput(c, &fixed).ReadOnly(true).Width(140)
		}
	})
}

func (s *uiState) subkeyAlgControls(c *ui.Context) {
	algs := []string{"ECDH", "RSA", "ELGAMAL"}
	ui.Row(c).Gap(8).Children(func() {
		ui.Select(c, &s.keyReq.SubkeyAlg, algs).Width(140).Tooltip(s.tr("tip.subkey.role"))
		s.keyReq = syncSubkeyAlgParams(s.keyReq)
		switch s.keyReq.SubkeyAlg {
		case "RSA":
			bitsSelect(c, &s.keyReq.SubkeyBits, []int{2048, 3072, 4096})
		case "ELGAMAL":
			bitsSelect(c, &s.keyReq.SubkeyBits, []int{2048, 3072, 4096})
		default:
			ui.Select(c, &s.keyReq.SubkeyCurve, []string{"Curve25519", "NIST P-256", "NIST P-384", "NIST P-521"}).Width(160)
		}
	})
}

func (s *uiState) cryptoControls(c *ui.Context) {
	hashes := []string{"SHA256", "SHA384", "SHA512", "SHA3-256"}
	ciphers := []string{"AES256", "AES192", "AES128", "Camellia256"}
	if !oneOf(s.keyReq.HashAlg, hashes...) {
		s.keyReq.HashAlg = "SHA256"
	}
	if !oneOf(s.keyReq.SymmCipher, ciphers...) {
		s.keyReq.SymmCipher = "AES256"
	}
	ui.Row(c).Gap(8).Children(func() {
		ui.Select(c, &s.keyReq.HashAlg, hashes).Width(160).Tooltip(s.tr("tip.crypto.role"))
		ui.Select(c, &s.keyReq.SymmCipher, ciphers).Width(160).Tooltip(s.tr("tip.crypto.role"))
	})
}

func bitsSelect(c *ui.Context, dst *int, allowed []int) {
	if dst == nil || len(allowed) == 0 {
		return
	}
	labels := make([]string, len(allowed))
	for i, n := range allowed {
		labels[i] = strconv.Itoa(n)
	}
	cur := strconv.Itoa(*dst)
	if !oneOf(cur, labels...) {
		cur = labels[0]
		*dst = allowed[0]
	}
	ui.Select(c, &cur, labels).Width(120)
	if n, err := strconv.Atoi(cur); err == nil && bitsAllowed(n, allowed...) {
		*dst = n
	}
}
