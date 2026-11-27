package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui"

	"github.com/r404r/RnpArmor/internal/rnp"
)

const toastLifetime = 4 * time.Second

// flushPendingToasts moves queued notices into the window before the
// toast viewport for this frame is built.
func (s *uiState) flushPendingToasts(c *ui.Context) {
	if len(s.pendingToasts) == 0 {
		return
	}
	pending := s.pendingToasts
	s.pendingToasts = nil
	for _, ts := range pending {
		c.AddToast(ts)
	}
}

func (s *uiState) viewStatusBar(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Shrink(0).Padding(6, 12).Gap(12).AlignItems(ui.Center).
		Background(t.Surface).BorderWidth(1, 0, 0, 0).BorderColor(t.Border).
		Role(ui.RoleStatus).Children(func() {
		ui.Textf(c, "librnp %s", s.version).SingleLine().Shrink(0).TextColor(t.TextMuted)
		shown := s.libPath
		if shown == "" {
			shown = "—"
		}
		tip := s.libPath
		if tip != "" {
			tip += "\n"
		}
		tip += s.tr("tip.copy_lib_path")
		path := ui.Text(c, shown).Grow(1).Shrink(1).SingleLine().Tooltip(tip)
		if path.Clicked() {
			s.copyLibPath()
		}
		ui.Text(c, s.statusBarRight()).SingleLine().Shrink(0).TextColor(t.TextMuted)
	})
}

func (s *uiState) viewToastViewport(c *ui.Context) {
	t := c.Theme()
	ui.ToastViewportBase(c, func(viewport ui.Element, toasts []ui.Toast) {
		viewport.Bottom(44).AlignItems(ui.Center).Gap(8).Padding(0, 16)
		for _, ts := range toasts {
			p := ui.ToastBase(c, ts)
			bg, fg := toastColors(t, ts.Type)
			p.Root.Row().AlignItems(ui.Center).Gap(12).Padding(10, 14).Radius(8).
				MaxWidth(520).Background(bg).TextColor(fg).
				Shadow(0, 6, 20, 0, ui.RGBA(0, 0, 0, 0.25))
			p.Root.Children(func() {
				ui.Column(c).Shrink(1).Gap(2).Children(func() {
					ui.Text(c, ts.Title).Bold().SingleLine()
					if ts.Description != "" {
						ui.Text(c, ts.Description).FontSize(12)
					}
				})
				p.CloseButton().Label(s.tr("btn.close")).Children(func() {
					ui.Text(c, "×").TextColor(fg)
				})
			})
		}
	})
}

func toastColors(t *ui.Theme, kind string) (bg, fg ui.Color) {
	fg = t.AccentText
	switch kind {
	case "error":
		return t.Danger, fg
	case "info":
		return t.Accent, fg
	default:
		return t.Success, fg
	}
}

func countPrimaryKeys(keys []rnp.KeySummary) int {
	n := 0
	for _, k := range keys {
		if isPrimaryRow(k) {
			n++
		}
	}
	return n
}

func formatPrimaryCount(lang Lang, n int) string {
	if lang == LangEN && n == 1 {
		return Tr(lang, "status.key_count_one")
	}
	return fmt.Sprintf(Tr(lang, "status.key_count_many"), n)
}

func (s *uiState) statusBarRight() string {
	return formatPrimaryCount(s.lang(), countPrimaryKeys(s.keys)) + "  ·  " + s.statusBarState()
}

func (s *uiState) statusBarState() string {
	if !s.busy {
		return s.tr("status.ready")
	}
	name := s.tr(s.statusKey)
	if name == "" || name == s.statusKey {
		return s.tr("state.busy")
	}
	return s.tr("state.busy") + " · " + name
}

func (s *uiState) copyLibPath() {
	s.copyTextAndToast(s.libPath, s.tr("toast.path_copied"))
}

func (s *uiState) copyFingerprintText(text string) {
	s.copyTextAndToast(text, s.tr("toast.fingerprint_copied"))
}

func (s *uiState) copyTextAndToast(text, title string) {
	go func() {
		err := mygo.Clipboard.Write(transfer.TextData(text))
		mygo.RunOnMain(func() {
			if err != nil {
				s.reportAsyncErr(err)
				return
			}
			s.enqueueToast(title, "", "success")
			s.invalidateAll()
		})
	}()
}

func (s *uiState) enqueueToast(title, desc, kind string) {
	s.toastSeq++
	s.pendingToasts = append(s.pendingToasts, ui.Toast{
		ID:          fmt.Sprintf("fb-%d", s.toastSeq),
		Title:       title,
		Description: desc,
		Type:        kind,
		Timeout:     toastLifetime,
	})
}

func (s *uiState) showAlert(title, msg string) {
	s.alertOpen = true
	s.alertTitle = title
	s.alertMsg = msg
}

// noteActionFeedback applies scheme A after applyResult.
// A child-window failure stays on that window's inline error
// (newKeyErr, importErr, subErr, expiryErr, revokeErr). A main-window
// AlertDialog would open behind the child and pull focus off the field
// the user is correcting. Once that child is closed, the alert is the
// only place the error can be seen. Startup keyring load is silent.
func (s *uiState) noteActionFeedback(title string, out actionOutcome, silent bool) {
	switch feedbackRoute(title, out.err != nil, silent, s.childOwnsError(title)) {
	case "none", "inline":
		return
	case "alert":
		if out.err != nil {
			s.showAlert(s.tr("alert.error"), s.safeNotice(s.tr(title)+": "+out.err.Error()))
			return
		}
		s.showAlert(s.tr(title), s.safeNotice(s.textOutput))
	default:
		s.enqueueToast(s.tr(title), s.tr("toast.done"), "success")
	}
}

func feedbackRoute(title string, failed, silent, childOpen bool) string {
	if silent {
		return "none"
	}
	if failed {
		if childOpen && childActionTitle(title) {
			return "inline"
		}
		return "alert"
	}
	switch title {
	case "action.verify_text", "action.verify_file":
		return "alert"
	default:
		return "toast"
	}
}

func childActionTitle(title string) bool {
	switch title {
	case "action.generate_key", "action.import_keys", "action.add_subkey", "action.set_key_expiry", "action.revoke_key":
		return true
	default:
		return false
	}
}

func (s *uiState) childOwnsError(title string) bool {
	if !childActionTitle(title) {
		return false
	}
	var w *mygo.Window
	switch title {
	case "action.generate_key":
		w = s.newKeyWin
	case "action.import_keys":
		w = s.importWin
	case "action.add_subkey":
		w = s.addSubWin
	case "action.set_key_expiry":
		w = s.expiryWin
	case "action.revoke_key":
		w = s.revokeWin
	}
	return w != nil && !w.IsDestroyed()
}

func (s *uiState) safeNotice(msg string) string {
	for _, sec := range []string{s.passphrase, s.newKeyPass, s.newKeyPass2, s.encPassword, s.subPass, s.expiryPass, s.revokePass} {
		if len(sec) < 4 {
			continue
		}
		msg = strings.ReplaceAll(msg, sec, "••••")
	}
	if strings.Contains(msg, "BEGIN PGP PRIVATE") || strings.Contains(msg, "BEGIN PGP SECRET") {
		return s.tr("alert.redacted")
	}
	if len(msg) > 600 {
		msg = msg[:600]
	}
	return msg
}
