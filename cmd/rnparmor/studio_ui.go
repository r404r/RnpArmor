package main

import (
	"path/filepath"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/r404r/RnpArmor/internal/rnp"
)

// keyPickerLabel is the drop-down line for one primary key.
// A UID is shown as "Alice <alice@example.com> [1111222233334444]".
// Without a UID the line is the fingerprint and, when set, the algorithm.
func keyPickerLabel(k rnp.KeySummary) string {
	id := keyIDText(k.Fingerprint, k.KeyID, keyIDModeShort)
	uid := strings.TrimSpace(k.PrimaryUID)
	if uid == "" {
		fp := stripID(k.Fingerprint)
		if fp == "" {
			fp = id
		}
		alg := strings.TrimSpace(k.Algorithm)
		if alg == "" || fp == "" {
			if fp != "" {
				return fp
			}
			return alg
		}
		return fp + " " + alg
	}
	if id == "" {
		return uid
	}
	return uid + " [" + id + "]"
}

// recipientOptions lists primary keys only. Subkeys stay out of the picker.
func recipientOptions(keys []rnp.KeySummary) (labels, fps []string) {
	for _, k := range keys {
		if !isPrimaryRow(k) {
			continue
		}
		labels = append(labels, keyPickerLabel(k))
		fps = append(fps, stripID(k.Fingerprint))
	}
	return labels, fps
}

// appendRecipientFingerprint adds fp to a comma list.
// Matching ignores case and spaces. An empty fingerprint is left unchanged.
// A duplicate returns the original text so a hand-edited list is not rewritten.
func appendRecipientFingerprint(list, fp string) string {
	fp = stripID(fp)
	if fp == "" {
		return list
	}
	parts := splitCSV(list)
	for _, p := range parts {
		if strings.EqualFold(stripID(p), fp) {
			return list
		}
	}
	if len(parts) == 0 {
		return fp
	}
	return strings.Join(parts, ", ") + ", " + fp
}

// clearRecipientList is the Clear button: the comma list becomes empty.
func clearRecipientList(_ string) string { return "" }

func fingerprintForLabel(label string, labels, fps []string) string {
	for i, l := range labels {
		if l == label && i < len(fps) {
			return fps[i]
		}
	}
	return ""
}

// signerOptions starts with the default item (empty fingerprint), then
// primary keys that have a secret. Public-only keys and subkeys are omitted.
func signerOptions(keys []rnp.KeySummary, defaultLabel string) (labels, fps []string) {
	labels = []string{defaultLabel}
	fps = []string{""}
	for _, k := range keys {
		if !isPrimaryRow(k) || !k.HasSecret {
			continue
		}
		labels = append(labels, keyPickerLabel(k))
		fps = append(fps, stripID(k.Fingerprint))
	}
	return labels, fps
}

// signerLabelFor is the drop-down line that matches textSigner.
// An empty or unknown fingerprint stays on the default item.
func signerLabelFor(textSigner string, labels, fps []string) string {
	want := stripID(textSigner)
	if want != "" {
		for i, fp := range fps {
			if fp == "" || i >= len(labels) {
				continue
			}
			if strings.EqualFold(stripID(fp), want) {
				return labels[i]
			}
		}
	}
	if len(labels) > 0 {
		return labels[0]
	}
	return ""
}

// applySignerPick writes the fingerprint for a drop-down change.
// When the visible choice did not change, current is kept, including a
// fingerprint typed by hand that is not in the list.
func applySignerPick(picked, previous string, labels, fps []string, current string) string {
	if picked == previous {
		return current
	}
	for i, label := range labels {
		if label == picked && i < len(fps) {
			return fps[i]
		}
	}
	return current
}

// dialogDefaultPath is the path a file dialog opens on.
// A filled current path wins. Otherwise the sibling file's directory is
// used when it has one, and the workspace is last.
func dialogDefaultPath(current, sibling, workspace string) string {
	current = strings.TrimSpace(current)
	if current != "" {
		return current
	}
	sibling = strings.TrimSpace(sibling)
	if sibling != "" {
		dir := filepath.Dir(sibling)
		if dir != "" && dir != "." {
			return dir
		}
	}
	return strings.TrimSpace(workspace)
}

func openPGPFileFilters(openpgpName, allName string) []mygo.FileFilter {
	return []mygo.FileFilter{
		{Name: openpgpName, Extensions: []string{"asc", "pgp", "gpg", "key", "sig"}},
		{Name: allName, Extensions: []string{"*"}},
	}
}

func fileDialogFilters(titleKey, openpgpName, allName string) []mygo.FileFilter {
	filters := openPGPFileFilters(openpgpName, allName)
	if titleKey == "dlg.browse_source" || titleKey == "dlg.save_destination" {
		return []mygo.FileFilter{filters[1], filters[0]}
	}
	return filters
}

func (s *uiState) recipientPicker(c *ui.Context) {
	labels, fps := recipientOptions(s.keys)
	ui.Column(c).Gap(6).Grow(1).Children(func() {
		if len(labels) == 0 {
			ui.Text(c, s.tr("empty.recipient_keys")).FontSize(12)
		} else {
			if !labelIn(s.recipientPick, labels) {
				s.recipientPick = labels[0]
			}
			ui.Row(c).Gap(8).Grow(1).Children(func() {
				ui.Select(c, &s.recipientPick, labels).Grow(1)
				if ui.Button(c, s.tr("btn.add_recipient")).Clicked() {
					s.encRecipients = appendRecipientFingerprint(s.encRecipients, fingerprintForLabel(s.recipientPick, labels, fps))
				}
				if ui.Button(c, s.tr("btn.clear_recipients")).Clicked() {
					s.encRecipients = clearRecipientList(s.encRecipients)
				}
			})
		}
		ui.TextInput(c, &s.encRecipients).Placeholder(s.tr("placeholder.recipients"))
	})
}

func labelIn(v string, labels []string) bool {
	for _, l := range labels {
		if l == v {
			return true
		}
	}
	return false
}

func (s *uiState) signerPicker(c *ui.Context) {
	labels, fps := signerOptions(s.keys, s.tr("signer.use_selected"))
	previous := signerLabelFor(s.textSigner, labels, fps)
	cur := previous
	ui.Column(c).Gap(6).Grow(1).Children(func() {
		if len(labels) > 0 {
			ui.Select(c, &cur, labels).Grow(1)
			s.textSigner = applySignerPick(cur, previous, labels, fps, s.textSigner)
		}
		ui.TextInput(c, &s.textSigner).Placeholder(s.tr("placeholder.signer"))
	})
}

// filePathEditors is a path box plus native browse and save buttons.
// openTitle or saveTitle empty hides that button. saveSibling is the other
// path whose directory is the save dialog's fallback.
func (s *uiState) filePathEditors(c *ui.Context, value *string, openTitle, saveTitle, saveSibling string) {
	ui.Row(c).Gap(8).Grow(1).Children(func() {
		ui.TextInput(c, value).Grow(1)
		if openTitle != "" && ui.Button(c, s.tr("btn.browse")).Clicked() {
			s.pickOpenPath(openTitle, dialogDefaultPath(*value, "", s.workspace), value)
		}
		if saveTitle != "" && ui.Button(c, s.tr("btn.save_as")).Clicked() {
			s.pickSavePath(saveTitle, dialogDefaultPath(*value, saveSibling, s.workspace), value)
		}
	})
}

// pickOpenPath asks for one existing file. The dialog runs off the UI
// goroutine: Dialog.Open pumps events, and calling it on the click path
// deadlocks the main thread. Do not hold s.mu across Dialog or Invalidate.
func (s *uiState) pickOpenPath(titleKey, def string, dst *string) {
	if dst == nil {
		return
	}
	parent := s.mainWin
	title := s.tr(titleKey)
	filters := fileDialogFilters(titleKey, s.tr("filter.openpgp"), s.tr("filter.all_files"))
	go func() {
		paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
			Parent:      parent,
			Title:       title,
			DefaultPath: def,
			Filters:     filters,
		})
		if err != nil {
			s.reportAsyncErr(err)
			return
		}
		if len(paths) == 0 || strings.TrimSpace(paths[0]) == "" {
			return
		}
		path := paths[0]
		mygo.RunOnMain(func() {
			*dst = path
			s.invalidateAll()
		})
	}()
}

// pickSavePath asks for an output path. Cancel (empty path) is ignored.
// Same threading rule as pickOpenPath.
func (s *uiState) pickSavePath(titleKey, def string, dst *string) {
	if dst == nil {
		return
	}
	parent := s.mainWin
	title := s.tr(titleKey)
	filters := fileDialogFilters(titleKey, s.tr("filter.openpgp"), s.tr("filter.all_files"))
	go func() {
		path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
			Parent:      parent,
			Title:       title,
			DefaultPath: def,
			Filters:     filters,
		})
		if err != nil {
			s.reportAsyncErr(err)
			return
		}
		if strings.TrimSpace(path) == "" {
			return
		}
		mygo.RunOnMain(func() {
			*dst = path
			s.invalidateAll()
		})
	}()
}
