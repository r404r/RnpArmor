package main

import (
	"runtime"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"
)

const (
	rnpBtnSecondary = "secondary"
	rnpBtnPrimary   = "primary"
	rnpBtnDanger    = "danger"
)

// passEyeIcon and passEyeOffIcon are 24px strokes in currentColor so they
// stay sharp on a high-DPI display and follow the button text color.
var (
	passEyeIcon    = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7S2 12 2 12z"/><circle cx="12" cy="12" r="3"/></svg>`))
	passEyeOffIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M3 3l18 18"/><path d="M10.58 10.58A3 3 0 0 0 13.42 13.42"/><path d="M9.88 5.09A10.9 10.9 0 0 1 12 5c6.5 0 10 7 10 7a18.4 18.4 0 0 1-3.17 4.15"/><path d="M6.11 6.11C3.6 7.86 2 12 2 12s3.5 7 10 7a11 11 0 0 0 4.13-.78"/></svg>`))
)

// rnpFace is the painted look of one restrained raised button.
type rnpFace struct {
	From, To        ui.Color
	Border, Text    ui.Color
	Shadow          ui.Color
	ShadowY         float32
	ShadowBlur      float32
	Highlight, Fold ui.Color
}

// useIntegratedMenuBar reports whether the window draws the File,
// Workspace, Language and Help menus itself. macOS keeps the system
// menu bar. Windows 11 Mica has no native menu bar, and Linux keeps
// the window manager's title buttons, so both draw this bar.
func useIntegratedMenuBar(goos string) bool {
	return goos != "darwin"
}

// mainWindowOptions is the main window chrome for goos.
// Windows hides the title bar and uses Mica. Linux and macOS keep the
// ordinary title bar so window buttons stay with the window manager.
func mainWindowOptions(goos string) mygo.WindowOptions {
	opts := mygo.WindowOptions{
		Title:         "RnpArmor",
		Width:         1120,
		Height:        780,
		TitleBarStyle: mygo.TitleBarDefault,
	}
	if goos == "windows" {
		opts.TitleBarStyle = mygo.TitleBarHidden
		opts.TitleBarHeight = 40
		opts.Vibrancy = mygo.VibrancyMica
	}
	return opts
}

// canvasColor is the cool gray behind cards when the window has no Mica.
func canvasColor() ui.Color { return ui.Hex("#edf1f7") }

// rnpLightTheme is the only appearance: a light theme that still follows
// high contrast, text size and an accent the desktop actually set.
func rnpLightTheme(p ui.Preferences) *ui.Theme {
	t := ui.LightTheme()
	t.Background = canvasColor()
	t.Surface = ui.Hex("#ffffff")
	t.SurfaceHover = ui.Hex("#f3f6fb")
	t.SurfacePressed = ui.Hex("#e7edf5")
	t.Border = ui.Hex("#d8dfe9")
	t.Radius = 8
	t.Dark = false
	return t.WithPreferences(p)
}

// prepareSurface applies the light theme, drawn menus, and the canvas.
// A Mica window stays transparent so the material shows through.
func prepareSurface(c *ui.Context) {
	c.SetMenuPresentation(ui.PopoverMenus)
	c.SetTheme(rnpLightTheme(c.Preferences()))
	if c.Vibrancy() {
		c.Root().Background(ui.Transparent)
		return
	}
	c.Root().Background(canvasColor())
}

// rnpCard is a white raised panel on the canvas.
func rnpCard(c *ui.Context) ui.Element {
	return ui.Column(c).Radius(10).Border(1, ui.Hex("#d8dfe9")).
		Background(ui.Hex("#ffffff")).
		Shadow(0, 2, 8, 0, ui.RGBA(15, 23, 42, 0.07))
}

func rnpButton(c *ui.Context, label string) ui.Element {
	return paintRnpButton(c, label, rnpBtnSecondary)
}

// rnpPassEye is the show/hide control beside a passphrase field.
// shown false draws the open eye (Show). shown true draws the slashed eye (Hide).
func rnpPassEye(c *ui.Context, shown *bool, showLabel, hideLabel string) ui.Element {
	label := showLabel
	icon := passEyeIcon
	if shown != nil && *shown {
		label = hideLabel
		icon = passEyeOffIcon
	}
	b := ui.ButtonBase(c)
	face := secondaryFace(b.Hovered(), b.Pressed())
	b.Padding(4, 6).Radius(7).Cursor(ui.CursorPointer).
		Gradient(face.From, face.To, 180).
		Border(1, face.Border).
		TextColor(face.Text).
		Tooltip(label).
		Label(label)
	if b.Pressed() {
		b.Translate(0, 1)
	}
	b.Children(func() {
		ui.Icon(c, icon).FontSize(16)
	})
	if b.Clicked() {
		flipPassVisible(shown)
	}
	return b
}

func flipPassVisible(shown *bool) {
	if shown == nil {
		return
	}
	*shown = !*shown
}

func rnpPrimaryButton(c *ui.Context, label string) ui.Element {
	return paintRnpButton(c, label, rnpBtnPrimary)
}

func rnpDangerButton(c *ui.Context, label string) ui.Element {
	return paintRnpButton(c, label, rnpBtnDanger)
}

// rnpDisclosure is the table's expand mark. It stays a tight text button
// so a raised button does not make the row taller.
func rnpDisclosure(c *ui.Context, mark string) ui.Element {
	b := ui.ButtonBase(c).Padding(0, 3).Radius(4).Cursor(ui.CursorPointer)
	b.Children(func() { ui.Text(c, mark).FontSize(12) })
	return b
}

func paintRnpButton(c *ui.Context, label, kind string) ui.Element {
	b := ui.ButtonBase(c)
	face := rnpButtonFace(kind, b.Hovered(), b.Pressed())
	b.Padding(6, 14).Gap(6).Radius(7).Cursor(ui.CursorPointer).
		Gradient(face.From, face.To, 180).
		Border(1, face.Border).
		TextColor(face.Text).
		Shadow(0, face.ShadowY, face.ShadowBlur, 0, face.Shadow)
	if b.Pressed() {
		b.Translate(0, 1)
	}
	b.DrawOver(func(p *ui.Painter, r ui.Rect) {
		if r.W < 4 || r.H < 4 {
			return
		}
		p.Fill(ui.Rect{X: r.X + 1, Y: r.Y + 1, W: r.W - 2, H: 1}, face.Highlight, 0)
		p.Fill(ui.Rect{X: r.X + 1, Y: r.Y + r.H - 2, W: r.W - 2, H: 1}, face.Fold, 0)
	})
	if label != "" {
		b.Children(func() { ui.Text(c, label).SingleLine() })
	}
	return b
}

// rnpButtonFace is the vertical gradient, hairline and shadow for kind.
// Hover lifts the color and the shadow. Press darkens and the caller
// shifts the button down by one DIP.
func rnpButtonFace(kind string, hovered, pressed bool) rnpFace {
	switch kind {
	case rnpBtnPrimary:
		return primaryFace(hovered, pressed)
	case rnpBtnDanger:
		return dangerFace(hovered, pressed)
	default:
		return secondaryFace(hovered, pressed)
	}
}

func secondaryFace(hovered, pressed bool) rnpFace {
	face := rnpFace{
		From: ui.Hex("#ffffff"), To: ui.Hex("#f3f6fb"),
		Border: ui.Hex("#d5dde8"), Text: ui.Hex("#1e293b"),
		Shadow: ui.RGBA(15, 23, 42, 0.10), ShadowY: 1, ShadowBlur: 3,
		Highlight: ui.RGBA(255, 255, 255, 0.85), Fold: ui.RGBA(15, 23, 42, 0.08),
	}
	if pressed {
		face.From, face.To = ui.Hex("#e8eef6"), ui.Hex("#dde5f0")
		face.Shadow = ui.RGBA(15, 23, 42, 0.08)
		face.ShadowY, face.ShadowBlur = 1, 2
		return face
	}
	if hovered {
		face.From, face.To = ui.Hex("#ffffff"), ui.Hex("#f8fafc")
		face.Shadow = ui.RGBA(15, 23, 42, 0.16)
		face.ShadowY, face.ShadowBlur = 2, 6
	}
	return face
}

func primaryFace(hovered, pressed bool) rnpFace {
	face := rnpFace{
		From: ui.Hex("#3b82f6"), To: ui.Hex("#1d4ed8"),
		Border: ui.Hex("#1e3a8a"), Text: ui.Hex("#ffffff"),
		Shadow: ui.RGBA(37, 99, 235, 0.28), ShadowY: 2, ShadowBlur: 6,
		Highlight: ui.RGBA(255, 255, 255, 0.35), Fold: ui.RGBA(15, 23, 42, 0.22),
	}
	if pressed {
		face.From, face.To = ui.Hex("#1d4ed8"), ui.Hex("#1e3a8a")
		face.ShadowY, face.ShadowBlur = 1, 3
		return face
	}
	if hovered {
		face.From, face.To = ui.Hex("#60a5fa"), ui.Hex("#2563eb")
		face.Shadow = ui.RGBA(37, 99, 235, 0.36)
		face.ShadowY, face.ShadowBlur = 3, 10
	}
	return face
}

func dangerFace(hovered, pressed bool) rnpFace {
	face := rnpFace{
		From: ui.Hex("#fff8f8"), To: ui.Hex("#ffe8ea"),
		Border: ui.Hex("#fecdd3"), Text: ui.Hex("#9f1239"),
		Shadow: ui.RGBA(159, 18, 57, 0.12), ShadowY: 1, ShadowBlur: 3,
		Highlight: ui.RGBA(255, 255, 255, 0.80), Fold: ui.RGBA(159, 18, 57, 0.12),
	}
	if pressed {
		face.From, face.To = ui.Hex("#ffe4e6"), ui.Hex("#fecdd3")
		face.ShadowY, face.ShadowBlur = 1, 2
		return face
	}
	if hovered {
		face.From, face.To = ui.Hex("#ffffff"), ui.Hex("#fff1f2")
		face.Shadow = ui.RGBA(159, 18, 57, 0.18)
		face.ShadowY, face.ShadowBlur = 2, 6
	}
	return face
}

func menuGlass() glass.Glass {
	return glass.Glass{
		Style:   glass.Regular,
		Tint:    ui.RGBA(255, 255, 255, 0.72),
		Clarity: 0.25,
	}
}

func glassMenuPanel(panel ui.Element) {
	panel.Material(menuGlass()).
		Radius(10).
		Border(1, ui.RGBA(15, 23, 42, 0.12)).
		NoShadow().
		Shadow(0, 10, 28, 0, ui.RGBA(15, 23, 42, 0.20))
}

func glassMenuItem(item ui.Element) {
	item.Radius(6)
}

func menuPopoverOptions() ui.PopoverMenuButtonOptions {
	return ui.PopoverMenuButtonOptions{Panel: glassMenuPanel, Item: glassMenuItem}
}

func (s *uiState) viewIntegratedMenuBar(c *ui.Context) {
	if !useIntegratedMenuBar(runtime.GOOS) {
		return
	}
	bar := c.TitleBar()
	h := float32(40)
	if bar.Height > h {
		h = bar.Height
	}
	ui.Row(c).Shrink(0).Height(h).Padding(0, bar.Right, 0, bar.Left).Gap(2).AlignItems(ui.Center).
		Background(ui.RGBA(255, 255, 255, 0.62)).
		BorderWidth(0, 0, 1, 0).BorderColor(ui.Hex("#d8dfe9")).
		Children(func() {
			ui.Row(c).Shrink(0).Gap(8).AlignItems(ui.Center).Padding(0, 10).DragWindow().Children(func() {
				ui.Text(c, s.tr("title.app")).Bold().SingleLine()
			})
			s.menuBarButton(c, s.tr("menu.file"), s.menuFile)
			s.menuBarButton(c, s.tr("menu.workspace"), s.menuWorkspace)
			s.menuBarButton(c, s.tr("menu.language"), s.menuLanguage)
			s.menuBarButton(c, s.tr("menu.help"), s.menuHelp)
			ui.Box(c).Grow(1).Height(h).DragWindow()
		})
}

func (s *uiState) menuBarButton(c *ui.Context, label string, build func(*ui.Menu)) {
	b := ui.PopoverMenuButton(c, label, build, menuPopoverOptions())
	bg := ui.RGBA(255, 255, 255, 0.35)
	if b.Pressed() {
		bg = ui.RGBA(226, 232, 240, 0.92)
	} else if b.Hovered() {
		bg = ui.RGBA(255, 255, 255, 0.88)
	}
	b.Background(bg).Padding(4, 10).Radius(6).Cursor(ui.CursorPointer).
		Border(1, ui.RGBA(15, 23, 42, 0.08))
}

func (s *uiState) menuFile(m *ui.Menu) {
	if m.Item(s.tr("btn.choose_folder")).Chosen() {
		s.chooseWorkspace()
	}
	if m.Item(s.tr("btn.open_session")).Chosen() {
		s.run("action.open_session", s.openSession)
	}
	if m.Item(s.tr("btn.load_keyring")).Chosen() {
		s.run("action.load_keyring", s.loadKeyring)
	}
	if m.Item(s.tr("btn.save_keyring")).Chosen() {
		s.run("action.save_keyring", s.saveKeyring)
	}
	m.Separator()
	if m.Item(s.tr("menu.file.quit")).Chosen() {
		mygo.App.Quit()
	}
}

func (s *uiState) menuWorkspace(m *ui.Menu) {
	items := []struct {
		key string
		tab int
	}{
		{key: "menu.workspace.keys", tab: 0},
		{key: "menu.workspace.text", tab: 1},
		{key: "menu.workspace.file", tab: 2},
		{key: "menu.workspace.packet", tab: 3},
		{key: "menu.workspace.portable", tab: 4},
	}
	for _, item := range items {
		tab := item.tab
		if m.Item(s.tr(item.key)).Radio(s.tab == tab).Chosen() {
			s.selectTab(tab)
		}
	}
}

func (s *uiState) menuLanguage(m *ui.Menu) {
	if m.Item("English").Radio(s.lang() == LangEN).Chosen() {
		s.setLang(LangEN)
	}
	if m.Item("中文").Radio(s.lang() == LangZH).Chosen() {
		s.setLang(LangZH)
	}
	if m.Item("日本語").Radio(s.lang() == LangJA).Chosen() {
		s.setLang(LangJA)
	}
}

func (s *uiState) menuHelp(m *ui.Menu) {
	if m.Item(s.tr("menu.help.about")).Chosen() {
		s.openAboutWindow()
	}
}

// toastGlass is the frosted card for a success, error or info toast.
// Text stays dark so it reads on the light tint.
func toastGlass(kind string) (glass.Glass, ui.Color, ui.Color) {
	switch kind {
	case "error":
		return glass.Glass{Style: glass.Regular, Tint: ui.RGBA(220, 38, 38, 0.22), Clarity: 0.40},
			ui.Hex("#7f1d1d"), ui.RGBA(220, 38, 38, 0.45)
	case "info":
		return glass.Glass{Style: glass.Regular, Tint: ui.RGBA(37, 99, 235, 0.20), Clarity: 0.40},
			ui.Hex("#1e3a8a"), ui.RGBA(37, 99, 235, 0.40)
	default:
		return glass.Glass{Style: glass.Regular, Tint: ui.RGBA(22, 163, 74, 0.20), Clarity: 0.40},
			ui.Hex("#14532d"), ui.RGBA(22, 163, 74, 0.40)
	}
}
