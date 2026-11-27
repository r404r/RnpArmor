package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"

	"github.com/r404r/RnpArmor/assets/icons"
	"github.com/r404r/RnpArmor/internal/rnp"
)

const (
	keyFilterAll = iota
	keyFilterMine
	keyFilterPublic

	keyIDModeShort = 0
	keyIDModeFull  = 1
)

func (s *uiState) viewKeys(c *ui.Context) {
	t := c.Theme()
	s.ensureKeyTable()
	ui.Column(c).Gap(8).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			if ui.PrimaryButton(c, s.tr("btn.new_key")).Clicked() {
				s.openNewKeyWindow()
			}
			if ui.Button(c, s.tr("btn.import_file")).Clicked() {
				s.chooseImportFile()
			}
			if ui.Button(c, s.tr("btn.import_text")).Clicked() {
				s.openImportTextWindow()
			}
			if ui.Button(c, s.tr("btn.refresh")).Clicked() {
				s.run("action.list_keys", s.reloadKeys)
			}
			s.workspaceChooser(c)
		})
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Segmented(c, &s.keyFilter, s.tr("seg.all"), s.tr("seg.mine"), s.tr("seg.public"))
			ui.Segmented(c, &s.keyIDMode, s.tr("seg.id16"), s.tr("seg.fp40"))
			ui.SearchField(c, &s.keySearch).Placeholder(s.tr("placeholder.search")).Grow(1)
		})

		primaries := filterPrimaryKeys(s.keys, s.keyFilter, s.keySearch)
		if s.keyTable.Sort != nil {
			sortKeyRows(primaries, *s.keyTable.Sort, s.keyIDMode)
		}
		rows := buildKeyTableRows(primaries, s.collapsedKeys)
		s.bindKeyTable(rows)
		detailIdx := -1
		for i := range primaries {
			if s.selected != "" && primaries[i].Fingerprint == s.selected {
				detailIdx = i
				break
			}
		}
		titleKey, idWidth := idColumnSpec(s.keyIDMode)
		cols := []ui.TableColumn{
			{Title: s.tr("col.type"), ID: "type", Width: 120, MinWidth: 100, Sortable: true},
			{Title: s.tr("col.name"), ID: "name", Width: 170, MinWidth: 120, Sortable: true},
			{Title: s.tr(titleKey), ID: "id", Width: idWidth, MinWidth: 120, Sortable: true},
			{Title: s.tr("col.algorithm"), ID: "alg", Width: 110, MinWidth: 96, Sortable: true},
			{Title: s.tr("col.status"), ID: "status", Width: 78, MinWidth: 68, Sortable: true},
		}
		ui.SplitPercent(c, &s.keySplitPct, func() {
			if len(rows) == 0 {
				key := "empty.keys"
				if strings.TrimSpace(s.keySearch) != "" || s.keyFilter != keyFilterAll {
					key = "empty.search"
				}
				ui.Text(c, s.tr(key)).TextColor(t.TextMuted).Padding(8, 0, 8, 0)
			}
			ui.Table(c, &s.keyTable, cols, len(rows), func(row, col int) {
				if row < 0 || row >= len(rows) {
					return
				}
				item := rows[row]
				k := item.Key
				switch col {
				case 0:
					if item.Sub {
						ui.Text(c, subkeyTypeText(s.lang()))
						return
					}
					ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
						if len(k.Subkeys) > 0 {
							mark := "▾"
							if keyCollapsed(s.collapsedKeys, k.Fingerprint) {
								mark = "▸"
							}
							fp := k.Fingerprint
							if ui.Button(c, mark).Clicked() {
								s.toggleCollapsed(fp)
							}
						}
						if k.HasSecret {
							ui.Text(c, s.tr("badge.secret"))
						} else {
							ui.Text(c, s.tr("badge.public"))
						}
					})
				case 1:
					if item.Sub {
						ui.Text(c, subkeyUsageText(s.lang(), k))
						return
					}
					ui.Text(c, k.PrimaryUID)
				case 2:
					ui.Text(c, keyIDText(k.Fingerprint, k.KeyID, s.keyIDMode))
				case 3:
					ui.Text(c, k.Algorithm)
				default:
					ui.Text(c, s.tr("badge."+keyStatusCode(k, time.Now().UTC())))
				}
			}).Height(520)
		}, func() {
			s.viewKeyDetail(c, primaries, detailIdx)
		}).Height(560)
	})
}

func (s *uiState) viewKeyDetail(c *ui.Context, rows []rnp.KeySummary, idx int) {
	t := c.Theme()
	ui.Scroll(c).Fill().Children(func() {
		ui.Column(c).Padding(8).Gap(8).Children(func() {
			if idx < 0 || idx >= len(rows) {
				ui.Text(c, s.tr("empty.select")).TextColor(t.TextMuted)
				return
			}
			k := rows[idx]
			ui.Text(c, k.PrimaryUID).Bold()
			ui.Row(c).Gap(6).Children(func() {
				if k.HasSecret {
					ui.Badge(c, s.tr("badge.secret_public"))
					if !k.IsProtected {
						ui.Badge(c, s.tr("badge.unprotected"))
					}
				} else {
					ui.Badge(c, s.tr("badge.public"))
				}
				ui.Badge(c, s.tr("badge."+keyStatusCode(k, time.Now().UTC())))
				if k.CanCertify {
					ui.Badge(c, s.tr("badge.certify"))
				}
				if k.CanSign {
					ui.Badge(c, s.tr("badge.sign"))
				}
				if k.CanEncrypt {
					ui.Badge(c, s.tr("badge.encrypt"))
				}
				if k.CanAuth {
					ui.Badge(c, s.tr("badge.auth"))
				}
			})
			grouped := groupFingerprint(k.Fingerprint)
			ui.Text(c, s.tr("label.fingerprint")).FontSize(12).TextColor(t.TextMuted)
			ui.Text(c, grouped)
			if ui.Button(c, s.tr("btn.copy_fingerprint")).Clicked() {
				text := grouped
				go func() {
					_ = mygo.Clipboard.Write(transfer.TextData(text))
				}()
			}
			ui.Textf(c, "%s: %s", s.tr("label.key_id"), keyIDText(k.Fingerprint, k.KeyID, keyIDModeShort))
			ui.Textf(c, "%s: %s", s.tr("col.algorithm"), k.Algorithm)
			ui.Textf(c, "%s: %s", s.tr("label.created"), formatKeyDate(k.CreatedUnix, s.tr("label.never")))
			ui.Textf(c, "%s: %s", s.tr("label.expires"), formatKeyDate(k.ExpiresUnix, s.tr("label.never")))

			ui.Row(c).Gap(8).AlignItems(ui.Center).Padding(8, 0, 0, 0).Children(func() {
				ui.Text(c, s.tr("label.subkeys")).Bold()
				if k.HasSecret {
					if ui.Button(c, s.tr("btn.add_subkey")).Clicked() {
						s.openAddSubkeyWindow()
					}
				}
			})
			if len(k.Subkeys) == 0 {
				ui.Text(c, s.tr("empty.subkeys")).FontSize(12).TextColor(t.TextMuted)
			}
			for _, sub := range k.Subkeys {
				s.viewSubkeyCard(c, k.HasSecret, sub)
			}

			ui.Row(c).Gap(8).Padding(8, 0, 0, 0).Children(func() {
				if ui.Button(c, s.tr("btn.copy_public")).Clicked() {
					s.run("action.copy_public", s.copyPublicKey)
				}
				if ui.Button(c, s.tr("btn.export_public_asc")).Clicked() {
					s.exportArmored("public")
				}
			})
			if k.HasSecret {
				ui.Row(c).Gap(8).Children(func() {
					if ui.Button(c, s.tr("btn.export_secret_asc")).Clicked() {
						s.exportArmored("secret")
					}
					if ui.Button(c, s.tr("btn.copy_secret")).Clicked() {
						s.run("action.copy_secret", s.copySecretKey)
					}
					if ui.Button(c, s.tr("btn.revocation")).Clicked() {
						s.exportArmored("revocation")
					}
				})
			}
			if ui.Button(c, s.tr("btn.delete_key_ellipsis")).Clicked() {
				s.confirmDelete()
			}

			ui.Collapsible(c, s.tr("label.raw_json"), &s.keyJSONOpen, func() {
				if ui.Button(c, s.tr("btn.load_json")).Clicked() {
					s.run("action.key_json", s.loadKeyJSON)
				}
				ui.TextArea(c, &s.keyJSON).Height(160).Placeholder(s.tr("placeholder.key_output"))
			})
		})
	})
}

func (s *uiState) workspaceChooser(c *ui.Context) {
	path := s.workspace
	ui.Row(c).Gap(8).Grow(1).Children(func() {
		ui.TextInput(c, &path).ReadOnly(true).Grow(1)
		if ui.Button(c, s.tr("btn.choose_folder")).Clicked() {
			s.chooseWorkspace()
		}
	})
}

func (s *uiState) chooseWorkspace() {
	parent := s.mainWin
	def := s.workspace
	title := s.tr("dlg.choose_folder")
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent:            parent,
			Title:             title,
			DefaultPath:       def,
			Directory:         true,
			CreateDirectories: true,
		})
		if err != nil {
			s.reportAsyncErr(err)
			return
		}
		if len(paths) == 0 || strings.TrimSpace(paths[0]) == "" {
			return
		}
		dir := paths[0]
		if abs, aerr := normalizeWorkspace(dir); aerr == nil {
			dir = abs
		}
		mygo.RunOnMain(func() {
			s.workspace = dir
			s.run("action.list_keys", s.reloadKeys)
		})
	}()
}

func (s *uiState) chooseImportFile() {
	parent := s.mainWin
	title := s.tr("dlg.import_file")
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent: parent,
			Title:  title,
			Filters: []mygo.FileFilter{{
				Name:       "OpenPGP",
				Extensions: []string{"asc", "pgp", "gpg", "key"},
			}},
		})
		if err != nil {
			s.reportAsyncErr(err)
			return
		}
		if len(paths) == 0 || strings.TrimSpace(paths[0]) == "" {
			return
		}
		b, err := os.ReadFile(paths[0])
		if err != nil {
			s.reportAsyncErr(err)
			return
		}
		mygo.RunOnMain(func() {
			s.importBytes = b
			s.run("action.import_keys", s.importKeyBytes)
		})
	}()
}

func (s *uiState) exportArmored(kind string) {
	if strings.TrimSpace(s.selected) == "" {
		s.lastErr = s.tr("err.no_key_selected")
		return
	}
	parent := s.mainWin
	fp := strings.TrimSpace(s.selected)
	titleKey := "dlg.export_public"
	suffix := ".asc"
	if kind == "secret" {
		titleKey = "dlg.export_secret"
	}
	if kind == "revocation" {
		titleKey = "dlg.export_revocation"
		suffix = ".rev.asc"
	}
	title := s.tr(titleKey)
	risk := s.tr("hint.secret_export")
	cont := s.tr("btn.export_secret_asc")
	cancel := s.tr("btn.cancel")
	base := fp
	if len(base) > 16 {
		base = base[len(base)-16:]
	}
	def := filepath.Join(s.workspace, base+suffix)
	go func() {
		if kind == "secret" {
			res, err := mygo.Dialog.Message(mygo.MessageOptions{
				Parent:             parent,
				Type:               mygo.MessageWarning,
				Title:              title,
				Message:            risk,
				Buttons:            []string{cont, cancel},
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
		}
		path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
			Parent:      parent,
			Title:       title,
			DefaultPath: def,
			Filters:     []mygo.FileFilter{{Name: "ASCII Armor", Extensions: []string{"asc"}}},
		})
		if err != nil {
			s.reportAsyncErr(err)
			return
		}
		if strings.TrimSpace(path) == "" {
			return
		}
		mygo.RunOnMain(func() {
			s.exportPath = path
			s.exportKind = kind
			switch kind {
			case "secret":
				s.run("action.export_secret_file", s.exportSecretFile)
			case "revocation":
				s.run("action.export_revocation", s.exportRevocation)
			default:
				s.run("action.export_public_file", s.exportPublicFile)
			}
		})
	}()
}

func (s *uiState) confirmDelete() {
	if strings.TrimSpace(s.selected) == "" {
		s.lastErr = s.tr("err.no_key_selected")
		return
	}
	parent := s.mainWin
	fp := strings.TrimSpace(s.selected)
	hasSecret := false
	for _, k := range s.keys {
		if k.Fingerprint == fp && k.HasSecret {
			hasSecret = true
			break
		}
	}
	title := s.tr("dlg.delete_title")
	msg := s.tr(deleteWarningKey(hasSecret))
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
			s.run("action.delete_key", s.deleteKey)
		})
	}()
}

func (s *uiState) reportAsyncErr(err error) {
	if err == nil {
		return
	}
	mygo.RunOnMain(func() {
		s.lastErr = err.Error()
		s.statusDone = false
		s.invalidateAll()
	})
}

func (s *uiState) openNewKeyWindow() {
	w, h := newKeyWindowSize()
	s.openChild(s.tr("title.new_key"), w, h, s.viewNewKey, &s.newKeyWin)
}

func (s *uiState) openImportTextWindow() {
	s.openChild(s.tr("title.import_text"), 560, 420, s.viewImportText, &s.importWin)
}

func (s *uiState) openChild(title string, width, height int, content func(*ui.Context), slot **mygo.Window) {
	if *slot != nil && !(*slot).IsDestroyed() {
		(*slot).Show()
		(*slot).Focus()
		return
	}
	opts := mygo.WindowOptions{
		Title:   title,
		Width:   width,
		Height:  height,
		Modal:   false,
		Content: ui.View(content),
	}
	if s.mainWin != nil && !s.mainWin.IsDestroyed() {
		opts.Parent = s.mainWin
	}
	win := mygo.NewWindow(opts)
	if err := win.SetIcon(icons.AppIconPNG); err != nil {
		log.Printf("child icon: %v", err)
	}
	if runtime.GOOS != "darwin" {
		win.SetMenu(nil)
	}
	win.OnClosed(func() {
		if *slot == win {
			*slot = nil
		}
		s.invalidateAll()
	})
	*slot = win
}

func (s *uiState) closeNewKeyWindow() {
	if s.newKeyWin != nil && !s.newKeyWin.IsDestroyed() {
		s.newKeyWin.Close()
	}
}

func (s *uiState) closeImportWindow() {
	if s.importWin != nil && !s.importWin.IsDestroyed() {
		s.importWin.Close()
	}
}

func (s *uiState) viewImportText(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Padding(16).Gap(8).Children(func() {
		ui.Text(c, s.tr("title.import_text")).FontSize(18).Bold()
		ui.TextArea(c, &s.importText).Height(220).Placeholder(s.tr("placeholder.import_text"))
		if s.importErr != "" {
			ui.Text(c, s.importErr).TextColor(t.Danger)
		}
		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c, s.tr("btn.paste_clipboard")).Clicked() {
				b, err := mygo.Clipboard.ReadFormat(transfer.Text)
				if err != nil {
					s.importErr = err.Error()
					return
				}
				s.importText = string(b)
				s.importErr = ""
			}
			if ui.PrimaryButton(c, s.tr("btn.import")).Clicked() {
				if strings.TrimSpace(s.importText) == "" {
					s.importErr = s.tr("err.import_empty")
					return
				}
				s.importErr = ""
				s.importBytes = []byte(s.importText)
				s.closeImportOnSuccess = true
				s.run("action.import_keys", s.importKeyBytes)
			}
			if ui.Button(c, s.tr("btn.close")).Clicked() {
				s.closeImportWindow()
			}
		})
	})
}

func (s *uiState) adoptPassphrase(pass string) {
	s.passphrase = pass
	buf := []byte(pass)
	for i := range buf {
		buf[i] = 0
	}
	s.newKeyPass = ""
	s.newKeyPass2 = ""
	s.subPass = ""
	s.expiryPass = ""
	s.revokePass = ""
}

func (s *uiState) ensureKeyTable() {
	if s.keyTable.Selected == nil || s.keyTable.Sort == nil {
		s.keySel = -1
		s.keyTable.Selected = &s.keySel
		s.keyTable.Sort = &s.keySort
	}
	s.keyTable.RowHeight = 34
	if s.keySplitPct <= 0 {
		s.keySplitPct = 58
	}
}

func (s *uiState) bindKeyTable(rows []keyTableRow) {
	s.ensureKeyTable()
	sort := ui.SortOrder{}
	if s.keyTable.Sort != nil {
		sort = *s.keyTable.Sort
	}
	sig := fmt.Sprintf("%d\x00%s\x00%s\x00%t\x00%s", s.keyFilter, s.keySearch, sort.Column, sort.Descending, collapseSig(s.collapsedKeys))
	picked := -1
	if s.keyTable.Selected != nil {
		picked = *s.keyTable.Selected
	}
	if s.selected == s.shownSelected && s.selectedSubkey == s.shownSubkey && sig == s.shownSig && picked >= 0 && picked < len(rows) {
		primary, sub := selectionForTableRow(rows[picked])
		if primary != s.selected || sub != s.selectedSubkey {
			s.selected = primary
			s.selectedSubkey = sub
		}
	}
	idx := -1
	for i := range rows {
		if s.selectedSubkey != "" {
			if rows[i].Sub && rows[i].Key.Fingerprint == s.selectedSubkey && rows[i].PrimaryFP == s.selected {
				idx = i
				break
			}
			continue
		}
		if !rows[i].Sub && rows[i].Key.Fingerprint == s.selected {
			idx = i
			break
		}
	}
	if s.keyTable.Selected != nil {
		*s.keyTable.Selected = idx
	}
	s.shownSelected = s.selected
	s.shownSubkey = s.selectedSubkey
	s.shownSig = sig
	s.keyTable.Key = func(i int) any {
		if i < 0 || i >= len(rows) {
			return i
		}
		if rows[i].Sub {
			return "s:" + rows[i].Key.Fingerprint
		}
		return "p:" + rows[i].Key.Fingerprint
	}
}

func (s *uiState) importKeyBytes(sess *rnp.Session, ctx context.Context, in actionInput) (actionResult, error) {
	if strings.TrimSpace(string(in.importData)) == "" {
		return actionResult{}, fmt.Errorf("%s", in.tr("err.import_empty"))
	}
	defer func() {
		for i := range in.importData {
			in.importData[i] = 0
		}
	}()
	if _, err := sess.ImportKeys(ctx, in.importData); err != nil {
		return actionResult{}, err
	}
	return actionResult{persist: persistUpdate}, nil
}

func (s *uiState) exportPublicFile(sess *rnp.Session, ctx context.Context, in actionInput) (actionResult, error) {
	return writeKeyExport(sess, ctx, in, false, false)
}

func (s *uiState) exportSecretFile(sess *rnp.Session, ctx context.Context, in actionInput) (actionResult, error) {
	return writeKeyExport(sess, ctx, in, true, false)
}

func (s *uiState) exportRevocation(sess *rnp.Session, ctx context.Context, in actionInput) (actionResult, error) {
	return writeKeyExport(sess, ctx, in, false, true)
}

func writeKeyExport(sess *rnp.Session, ctx context.Context, in actionInput, secret, rev bool) (actionResult, error) {
	if strings.TrimSpace(in.selected) == "" {
		return actionResult{}, fmt.Errorf("%s", in.tr("err.no_key_selected"))
	}
	var (
		v   string
		err error
	)
	if rev {
		v, err = sess.ExportRevocation(ctx, in.selected, "retired", "")
	} else {
		v, err = sess.ExportKey(ctx, in.selected, secret)
	}
	if err != nil {
		return actionResult{}, err
	}
	res := actionResult{hasKeyJSON: true, keyJSON: v}
	path := strings.TrimSpace(in.exportPath)
	if path != "" {
		mode := os.FileMode(0o644)
		if secret || rev {
			mode = 0o600
		}
		if err := writeExportFile(path, []byte(v), mode); err != nil {
			return actionResult{}, err
		}
	}
	return res, nil
}

func (s *uiState) copyPublicKey(sess *rnp.Session, ctx context.Context, in actionInput) (actionResult, error) {
	return copySelectedKey(sess, ctx, in, false)
}

func (s *uiState) copySecretKey(sess *rnp.Session, ctx context.Context, in actionInput) (actionResult, error) {
	return copySelectedKey(sess, ctx, in, true)
}

func copySelectedKey(sess *rnp.Session, ctx context.Context, in actionInput, secret bool) (actionResult, error) {
	if strings.TrimSpace(in.selected) == "" {
		return actionResult{}, fmt.Errorf("%s", in.tr("err.no_key_selected"))
	}
	v, err := sess.ExportKey(ctx, in.selected, secret)
	if err != nil {
		return actionResult{}, err
	}
	return actionResult{hasCopy: true, copyText: v, copyConcealed: secret}, nil
}

func deliverClipboard(res *actionResult) error {
	if res == nil || !res.hasCopy {
		return nil
	}
	text := res.copyText
	concealed := res.copyConcealed
	res.copyText = ""
	res.hasCopy = false
	res.copyConcealed = false
	buf := []byte(text)
	defer func() {
		for i := range buf {
			buf[i] = 0
		}
	}()
	var opts []mygo.ClipboardOptions
	if concealed {
		opts = append(opts, mygo.ClipboardOptions{Concealed: true})
	}
	return mygo.Clipboard.Write(transfer.TextData(string(buf)), opts...)
}

func writeExportFile(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, mode); err != nil {
		return err
	}
	return os.Chmod(path, mode)
}

func (s *uiState) deleteKeyConfirmed(confirmed bool) error {
	button := 1
	if confirmed {
		button = 0
	}
	if !confirmButton(button) {
		return nil
	}
	s.run("action.delete_key", s.deleteKey)
	if s.busy {
		return fmt.Errorf("delete: still busy")
	}
	if s.lastErr != "" {
		return fmt.Errorf("delete: %s", s.lastErr)
	}
	return nil
}

func confirmButton(index int) bool { return index == 0 }

func deleteWarningKey(hasSecret bool) string {
	if hasSecret {
		return "dlg.delete_secret"
	}
	return "dlg.delete_public"
}

func validateNewKey(name, email, pass, confirm string, expiry int, base rnp.KeyGenRequest) (rnp.KeyGenRequest, string) {
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)
	if name == "" && email == "" {
		return rnp.KeyGenRequest{}, "err.name_or_email"
	}
	if pass != confirm {
		return rnp.KeyGenRequest{}, "err.pass_mismatch"
	}
	uid := name
	switch {
	case name != "" && email != "":
		uid = name + " <" + email + ">"
	case email != "":
		uid = email
	}
	req := base
	req.UserID = uid
	req.ExpirationSec = expirySeconds(expiry)
	req.Unprotected = pass == "" && confirm == ""
	return req, ""
}

func expirySeconds(idx int) uint32 {
	const year = 365 * 24 * 60 * 60
	switch idx {
	case 1:
		return year
	case 2:
		return 2 * year
	case 3:
		return 5 * year
	default:
		return 0
	}
}

func isPrimaryRow(k rnp.KeySummary) bool {
	if k.IsPrimary {
		return true
	}
	return !k.IsSub
}

func filterPrimaryKeys(keys []rnp.KeySummary, filter int, query string) []rnp.KeySummary {
	q := strings.ToLower(strings.TrimSpace(query))
	out := make([]rnp.KeySummary, 0, len(keys))
	for _, k := range keys {
		if !isPrimaryRow(k) {
			continue
		}
		switch filter {
		case keyFilterMine:
			if !k.HasSecret {
				continue
			}
		case keyFilterPublic:
			if k.HasSecret {
				continue
			}
		}
		if q != "" && !keyMatchesQuery(k, q) {
			continue
		}
		out = append(out, k)
	}
	return out
}

func keyMatchesQuery(k rnp.KeySummary, q string) bool {
	if keyBlobMatch(k, q) {
		return true
	}
	for i := range k.Subkeys {
		if keyBlobMatch(k.Subkeys[i], q) {
			return true
		}
	}
	return false
}

func keyBlobMatch(k rnp.KeySummary, q string) bool {
	blob := strings.ToLower(k.PrimaryUID + " " + k.KeyID + " " + k.Fingerprint + " " + k.Algorithm)
	return strings.Contains(blob, q)
}

func sortKeyRows(rows []rnp.KeySummary, order ui.SortOrder, idMode int) {
	if strings.TrimSpace(order.Column) == "" {
		return
	}
	slices.SortStableFunc(rows, func(a, b rnp.KeySummary) int {
		cmp := strings.Compare(strings.ToLower(keySortValue(a, order.Column, idMode)), strings.ToLower(keySortValue(b, order.Column, idMode)))
		if order.Descending {
			return -cmp
		}
		return cmp
	})
}

func keySortValue(k rnp.KeySummary, column string, idMode int) string {
	switch column {
	case "type":
		if k.HasSecret {
			return "secret"
		}
		return "public"
	case "name":
		return k.PrimaryUID
	case "id":
		return keyIDText(k.Fingerprint, k.KeyID, idMode)
	case "alg":
		return k.Algorithm
	case "status":
		return keyStatusCode(k, time.Now().UTC())
	default:
		return k.PrimaryUID
	}
}

func idColumnSpec(mode int) (titleKey string, width float32) {
	if mode == keyIDModeFull {
		return "col.fingerprint", 340
	}
	return "col.key_id", 150
}

func keyIDText(fp, keyID string, mode int) string {
	fp = stripID(fp)
	keyID = stripID(keyID)
	if mode == keyIDModeFull {
		if fp != "" {
			return fp
		}
		return keyID
	}
	if len(keyID) >= 16 {
		return keyID[len(keyID)-16:]
	}
	if keyID != "" {
		return keyID
	}
	if len(fp) >= 16 {
		return fp[len(fp)-16:]
	}
	return fp
}

func groupFingerprint(fp string) string {
	raw := stripID(fp)
	if raw == "" {
		return ""
	}
	var b strings.Builder
	for i := 0; i < len(raw); i++ {
		if i > 0 && i%4 == 0 {
			b.WriteByte(' ')
		}
		b.WriteByte(raw[i])
	}
	return b.String()
}

func stripID(s string) string {
	return strings.Map(func(r rune) rune {
		if r == ' ' || r == '\n' || r == '\t' {
			return -1
		}
		return r
	}, strings.TrimSpace(s))
}

func keyStatusCode(k rnp.KeySummary, now time.Time) string {
	if k.IsRevoked {
		return "revoked"
	}
	if k.ExpiresUnix > 0 && int64(k.ExpiresUnix) <= now.Unix() {
		return "expired"
	}
	return "valid"
}

func formatKeyDate(sec uint32, never string) string {
	if sec == 0 {
		return never
	}
	return time.Unix(int64(sec), 0).UTC().Format("2006-01-02")
}
