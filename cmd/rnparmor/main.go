package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/r404r/RnpArmor/assets/icons"
	"github.com/r404r/RnpArmor/internal/rnp"
)

// appVersion is injected at build time by -ldflags
// (for example: -X main.appVersion=v1.2.3). Default is dev.
var appVersion = "dev"

type uiState struct {
	mu sync.Mutex

	libPath        string
	version        string
	lastErr        string
	busy           bool
	statusKey      string
	statusDone     bool
	workspace      string
	passphrase     string
	showUnlockPass bool

	keys           []rnp.KeySummary
	selected       string
	selectedSubkey string
	collapsedKeys  map[string]bool
	keyJSON        string
	features       string
	packetOut      string

	keyReq rnp.KeyGenRequest

	textInput          string
	textOutput         string
	textDetached       string
	textSigner         string
	textVerifyInput    string
	textVerifyDetached string
	textMode           string

	encRecipients   string
	recipientPick   string
	encPassword     string
	showTextEncPass bool
	showFileEncPass bool
	encCipher       string
	encCompress     string
	encSign         bool

	fileSrc      string
	fileDst      string
	fileSig      string
	fileDetached bool

	packetInput   string
	packetAsJSON  bool
	featureQuery  string
	sessionLoaded bool
	tab           int
	langIdx       int
	mainWin       *mygo.Window
	aboutWin      *mygo.Window

	keyFilter     int
	keyIDMode     int
	keySearch     string
	keySplitPct   float32
	keyTable      ui.ListState
	keySel        int
	keySort       ui.SortOrder
	shownSelected string
	shownSubkey   string
	shownSig      string
	keyJSONOpen   bool

	newKeyWin       *mygo.Window
	newKeyName      string
	newKeyEmail     string
	newKeyPass      string
	newKeyPass2     string
	newKeyShowPass  bool
	newKeyShowPass2 bool
	newKeyChecked   bool
	newKeyExpiry    int
	newKeyPreset    string
	newKeyAdvanced  bool
	newKeyErr       string

	importWin            *mygo.Window
	importText           string
	importErr            string
	importBytes          []byte
	closeImportOnSuccess bool

	exportPath string
	exportKind string

	pendingCloseNew    bool
	pendingCloseImport bool

	addSubWin      *mygo.Window
	subUsage       string
	subAlg         string
	subCurve       string
	subBits        int
	subHash        string
	subCipher      string
	subExpiry      int
	subPass        string
	subShowPass    bool
	subErr         string
	subAdvanced    bool
	subReq         rnp.SubkeyGenRequest
	expiryWin      *mygo.Window
	expiryTarget   string
	expiryCreated  uint32
	expiryChoice   int
	expiryPass     string
	expiryShowPass bool
	expiryErr      string
	revokeWin      *mygo.Window
	revokeTarget   string
	revokeCode     string
	revokeReason   string
	revokePass     string
	revokeShowPass bool
	revokeErr      string
	targetFP       string
	expirySeconds  uint32

	pendingCloseAddSub bool
	pendingCloseExpiry bool
	pendingCloseRevoke bool

	pendingToasts []ui.Toast
	toastSeq      uint64
	alertOpen     bool
	alertTitle    string
	alertMsg      string

	demoSceneApplied bool
	demoOpenAbout    bool
}

func main() {
	smoke := flag.Bool("smoke-test", false, "probe librnp and exit")
	smokeOut := flag.String("smoke-out", "", "write smoke-test output to file")
	langFlag := flag.String("lang", "", "ui language: en|zh|ja")
	flag.Parse()

	path, err := rnp.DefaultLibrary()
	if err != nil {
		fmt.Fprintln(os.Stderr, "status=error")
		fmt.Fprintln(os.Stderr, "step=resolve")
		fmt.Fprintln(os.Stderr, "error="+err.Error())
		os.Exit(1)
	}
	v, err := rnp.Probe(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "status=error")
		fmt.Fprintln(os.Stderr, "step=probe")
		fmt.Fprintln(os.Stderr, "library="+path)
		fmt.Fprintln(os.Stderr, "error="+err.Error())
		os.Exit(1)
	}
	if err := ValidateCatalog(); err != nil {
		fmt.Fprintln(os.Stderr, "status=error")
		fmt.Fprintln(os.Stderr, "step=i18n")
		fmt.Fprintln(os.Stderr, "error="+err.Error())
		os.Exit(1)
	}
	lang := ResolveLang(*langFlag)

	if *smoke {
		smokeText := buildSmokeOutput(appVersion, path, v)
		if err := writeSmokeOutput(*smokeOut, smokeText); err != nil {
			fmt.Fprintln(os.Stderr, "status=error")
			fmt.Fprintln(os.Stderr, "step=smoke-out")
			fmt.Fprintln(os.Stderr, "error="+err.Error())
			os.Exit(1)
		}
		fmt.Print(smokeText)
		os.Exit(0)
	}

	st := &uiState{
		libPath:         path,
		version:         v,
		statusKey:       "status.ready",
		statusDone:      false,
		workspace:       mustWd(),
		passphrase:      "",
		keySplitPct:     58,
		encCipher:       "AES256",
		encCompress:     "zlib",
		textMode:        string(rnp.SignModeDetached),
		featureQuery:    "",
		packetAsJSON:    true,
		textInput:       "hello from RnpArmor",
		textVerifyInput: "",
		langIdx:         LangIndex(lang),
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

	mygo.App.WhenReady(func() {
		if runtime.GOOS == "darwin" {
			if err := mygo.App.Dock.SetIcon(icons.AppIconPNG); err != nil {
				log.Printf("dock icon: %v", err)
			}
		}
		opts := mainWindowOptions(runtime.GOOS)
		opts.Content = ui.View(st.view)
		w := mygo.NewWindow(opts)
		if err := w.SetIcon(icons.AppIconPNG); err != nil {
			log.Printf("window icon: %v", err)
		}
		w.OnClosed(func() {
			st.closeAboutWindow()
			st.closeNewKeyWindow()
			st.closeImportWindow()
			st.closeAddSubWindow()
			st.closeExpiryWindow()
			st.closeRevokeWindow()
		})
		st.mainWin = w
		st.applyMainMenu()
		st.autoLoadKeyring()
	})
	if err := mygo.App.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func mustWd() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return wd
}

func buildSmokeOutput(appVer, libPath, version string) string {
	lines := []string{
		"status=ok",
		"app_version=" + appVer,
		"library=" + libPath,
		"version=" + version,
		"i18n=ok",
	}
	return strings.Join(lines, "\n") + "\n"
}

func writeSmokeOutput(path, text string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(text), 0o600)
}

func (s *uiState) lang() Lang { return LangByIndex(s.langIdx) }

func (s *uiState) tr(key string) string { return Tr(s.lang(), key) }

func (s *uiState) setLang(l Lang) {
	idx := LangIndex(l)
	if idx == s.langIdx {
		return
	}
	s.langIdx = idx
	s.applyMainMenu()
	if s.mainWin != nil && !s.mainWin.IsDestroyed() {
		s.mainWin.SetTitle(s.tr("title.app"))
	}
	if s.aboutWin != nil && !s.aboutWin.IsDestroyed() {
		s.aboutWin.SetTitle(s.tr("title.about"))
	}
	if s.newKeyWin != nil && !s.newKeyWin.IsDestroyed() {
		s.newKeyWin.SetTitle(s.tr("title.new_key"))
	}
	if s.importWin != nil && !s.importWin.IsDestroyed() {
		s.importWin.SetTitle(s.tr("title.import_text"))
	}
	if s.addSubWin != nil && !s.addSubWin.IsDestroyed() {
		s.addSubWin.SetTitle(s.tr("title.add_subkey"))
	}
	if s.expiryWin != nil && !s.expiryWin.IsDestroyed() {
		s.expiryWin.SetTitle(s.tr("title.subkey_expiry"))
	}
	if s.revokeWin != nil && !s.revokeWin.IsDestroyed() {
		s.revokeWin.SetTitle(s.tr("title.subkey_revoke"))
	}
	s.invalidateAll()
}

func (s *uiState) selectTab(i int) {
	if s.tab != i {
		s.tab = i
		s.applyMainMenu()
	}
	s.invalidateAll()
	if s.mainWin != nil && !s.mainWin.IsDestroyed() {
		s.mainWin.Focus()
	}
}

func (s *uiState) closeMainWindow() {
	if s.mainWin != nil && !s.mainWin.IsDestroyed() {
		s.mainWin.Close()
	}
}

func (s *uiState) openAboutWindow() {
	if s.aboutWin != nil && !s.aboutWin.IsDestroyed() {
		s.aboutWin.Show()
		s.aboutWin.Focus()
		return
	}
	opts := mygo.WindowOptions{
		Title:           s.tr("title.about"),
		Width:           580,
		Height:          520,
		Modal:           false,
		DisableResize:   true,
		DisableMaximize: true,
		DisableMinimize: true,
		Content:         ui.View(s.viewAboutWindow),
	}
	if s.mainWin != nil && !s.mainWin.IsDestroyed() {
		opts.Parent = s.mainWin
	}
	w := mygo.NewWindow(opts)
	if err := w.SetIcon(icons.AppIconPNG); err != nil {
		log.Printf("about icon: %v", err)
	}
	if runtime.GOOS != "darwin" {
		w.SetMenu(nil)
	}
	w.OnClosed(func() {
		s.aboutWin = nil
		s.invalidateAll()
	})
	s.aboutWin = w
	s.invalidateAll()
}

func (s *uiState) closeAboutWindow() {
	if s.aboutWin != nil && !s.aboutWin.IsDestroyed() {
		s.aboutWin.Close()
	}
	s.invalidateAll()
}

func (s *uiState) invalidateAll() {
	if s.mainWin != nil && !s.mainWin.IsDestroyed() {
		s.mainWin.Invalidate()
	}
	if s.aboutWin != nil && !s.aboutWin.IsDestroyed() {
		s.aboutWin.Invalidate()
	}
	if s.newKeyWin != nil && !s.newKeyWin.IsDestroyed() {
		s.newKeyWin.Invalidate()
	}
	if s.importWin != nil && !s.importWin.IsDestroyed() {
		s.importWin.Invalidate()
	}
	if s.addSubWin != nil && !s.addSubWin.IsDestroyed() {
		s.addSubWin.Invalidate()
	}
	if s.expiryWin != nil && !s.expiryWin.IsDestroyed() {
		s.expiryWin.Invalidate()
	}
	if s.revokeWin != nil && !s.revokeWin.IsDestroyed() {
		s.revokeWin.Invalidate()
	}
}

func (s *uiState) menuTemplate() []*mygo.MenuItem {
	wsItems := []struct {
		key string
		tab int
	}{
		{key: "menu.workspace.keys", tab: 0},
		{key: "menu.workspace.text", tab: 1},
		{key: "menu.workspace.file", tab: 2},
		{key: "menu.workspace.packet", tab: 3},
		{key: "menu.workspace.portable", tab: 4},
	}
	workspaceMenu := make([]*mygo.MenuItem, 0, len(wsItems))
	for _, item := range wsItems {
		tab := item.tab
		workspaceMenu = append(workspaceMenu, &mygo.MenuItem{
			Label:   s.tr(item.key),
			Type:    mygo.MenuItemRadio,
			Checked: s.tab == tab,
			Click: func(*mygo.MenuItem, *mygo.Window) {
				s.selectTab(tab)
			},
		})
	}

	currentLang := s.lang()
	langMenu := []*mygo.MenuItem{
		{Label: "English", Type: mygo.MenuItemRadio, Checked: currentLang == LangEN, Click: func(*mygo.MenuItem, *mygo.Window) { s.setLang(LangEN) }},
		{Label: "中文", Type: mygo.MenuItemRadio, Checked: currentLang == LangZH, Click: func(*mygo.MenuItem, *mygo.Window) { s.setLang(LangZH) }},
		{Label: "日本語", Type: mygo.MenuItemRadio, Checked: currentLang == LangJA, Click: func(*mygo.MenuItem, *mygo.Window) { s.setLang(LangJA) }},
	}

	return []*mygo.MenuItem{
		{
			Label: s.tr("menu.file"),
			Submenu: []*mygo.MenuItem{
				{Label: s.tr("btn.choose_folder"), Click: func(*mygo.MenuItem, *mygo.Window) { s.chooseWorkspace() }},
				{Label: s.tr("btn.open_session"), Click: func(*mygo.MenuItem, *mygo.Window) { s.run("action.open_session", s.openSession) }},
				{Label: s.tr("btn.load_keyring"), Click: func(*mygo.MenuItem, *mygo.Window) { s.run("action.load_keyring", s.loadKeyring) }},
				{Label: s.tr("btn.save_keyring"), Click: func(*mygo.MenuItem, *mygo.Window) { s.run("action.save_keyring", s.saveKeyring) }},
				mygo.Separator(),
				{Label: s.tr("menu.file.quit"), Click: func(*mygo.MenuItem, *mygo.Window) { mygo.App.Quit() }},
			},
		},
		{
			Label:   s.tr("menu.workspace"),
			Submenu: workspaceMenu,
		},
		{
			Label:   s.tr("menu.language"),
			Submenu: langMenu,
		},
		{
			Label: s.tr("menu.help"),
			Submenu: []*mygo.MenuItem{
				{Label: s.tr("menu.help.about"), Click: func(*mygo.MenuItem, *mygo.Window) { s.openAboutWindow() }},
			},
		},
	}
}

func (s *uiState) applyMainMenu() {
	if useIntegratedMenuBar(runtime.GOOS) {
		return
	}
	if s.mainWin == nil || s.mainWin.IsDestroyed() {
		return
	}
	mygo.App.SetMenu(mygo.NewMenu(s.menuTemplate()))
}

func (s *uiState) view(c *ui.Context) {
	prepareSurface(c)
	s.flushPendingToasts(c)
	ui.Column(c).Fill().Children(func() {
		s.viewIntegratedMenuBar(c)
		ui.Column(c).Padding(14).Gap(10).Grow(1).MinHeight(0).Children(func() {
			if !useIntegratedMenuBar(runtime.GOOS) {
				ui.Text(c, s.tr("title.app")).FontSize(24).Bold()
			}
			prevTab := s.tab
			ui.Tabs(c, &s.tab,
				s.tr("tab.keys"),
				s.tr("tab.text"),
				s.tr("tab.file"),
				s.tr("tab.packet"),
				s.tr("tab.portable"),
			).Value("workspaces")
			if s.tab != prevTab {
				s.applyMainMenu()
			}
			ui.Scroll(c).Grow(1).MinHeight(0).Children(func() {
				switch s.tab {
				case 0:
					s.viewKeys(c)
				case 1:
					s.viewTextStudio(c)
				case 2:
					s.viewFileWorkbench(c)
				case 3:
					s.viewPackets(c)
				default:
					s.viewPortable(c)
				}
			})
		})
		s.viewStatusBar(c)
	})
	s.viewToastViewport(c)
	ui.AlertDialog(c, &s.alertOpen, s.alertTitle, s.alertMsg, s.tr("btn.ok"))
}

func (s *uiState) statusLine() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	state := s.tr("state.idle")
	if s.busy {
		state = s.tr("state.busy")
	}
	if s.lastErr != "" {
		return state + " | " + s.lastErr
	}
	status := s.tr(s.statusKey)
	if status == s.statusKey {
		status = s.statusKey
	}
	if s.statusDone {
		status = status + " " + s.tr("state.done")
	}
	return state + " | " + status
}

func (s *uiState) viewTextStudio(c *ui.Context) {
	rnpCard(c).Padding(14).Gap(8).Children(func() {
		ui.Form(c, func() {
			ui.Field(c, s.tr("label.plain_text"), func() {
				ui.TextArea(c, &s.textInput).Height(120)
			})
			ui.Field(c, s.tr("label.recipients"), func() {
				s.recipientPicker(c)
			})
			ui.Field(c, s.tr("label.password"), func() {
				s.passphraseRow(c, &s.encPassword, &s.showTextEncPass, s.tr("placeholder.symm_pass"))
			})
			ui.Field(c, s.tr("label.cipher_compression"), func() {
				ui.Row(c).Gap(8).Children(func() {
					ui.Select(c, &s.encCipher, []string{"AES256", "AES192", "AES128", "Camellia256"}).Width(180)
					ui.Select(c, &s.encCompress, []string{"zlib", "bzip2", "zip", "uncompressed"}).Width(180)
					ui.Switch(c, &s.encSign).Label(s.tr("toggle.encrypt_sign"))
				})
			})
			ui.Field(c, s.tr("label.sign_mode"), func() {
				ui.Select(c, &s.textMode, []string{string(rnp.SignModeCleartext), string(rnp.SignModeDetached), string(rnp.SignModeInline)}).Width(220)
			})
			ui.Field(c, s.tr("label.signer_fingerprint"), func() {
				s.signerPicker(c)
			})
			ui.Field(c, s.tr("label.detached_verify_sig"), func() {
				ui.TextArea(c, &s.textVerifyDetached).Height(90)
			})
		})

		ui.Row(c).Gap(8).Padding(8, 0, 8, 0).Children(func() {
			if rnpPrimaryButton(c, s.tr("btn.encrypt_text")).Clicked() {
				s.run("action.encrypt_text", s.encryptText)
			}
			if rnpButton(c, s.tr("btn.decrypt_text")).Clicked() {
				s.run("action.decrypt_text", s.decryptText)
			}
			if rnpButton(c, s.tr("btn.sign_text")).Clicked() {
				s.run("action.sign_text", s.signText)
			}
			if rnpButton(c, s.tr("btn.verify_text")).Clicked() {
				s.run("action.verify_text", s.verifyText)
			}
		})
		ui.Text(c, s.tr("hint.text_alg")).FontSize(12)
		ui.Text(c, s.tr("hint.verify_badge")).FontSize(12)
		ui.TextArea(c, &s.textOutput).Height(280).Placeholder(s.tr("placeholder.result_output"))
	})
}

func (s *uiState) viewFileWorkbench(c *ui.Context) {
	rnpCard(c).Padding(14).Gap(8).Children(func() {
		ui.Form(c, func() {
			ui.Field(c, s.tr("label.source"), func() {
				s.filePathEditors(c, &s.fileSrc, "dlg.browse_source", "", "")
			})
			ui.Field(c, s.tr("label.destination"), func() {
				s.filePathEditors(c, &s.fileDst, "", "dlg.save_destination", s.fileSrc)
			})
			ui.Field(c, s.tr("label.signature_path"), func() {
				s.filePathEditors(c, &s.fileSig, "dlg.browse_signature", "dlg.save_signature", s.fileSrc)
			})
			ui.Field(c, s.tr("label.detached_verify"), func() { ui.Switch(c, &s.fileDetached).Label(s.tr("toggle.detached")) })
			ui.Field(c, s.tr("label.recipients"), func() { s.recipientPicker(c) })
			ui.Field(c, s.tr("label.password"), func() {
				s.passphraseRow(c, &s.encPassword, &s.showFileEncPass, s.tr("placeholder.symm_pass"))
			})
			ui.Field(c, s.tr("label.signer_fingerprint"), func() { s.signerPicker(c) })
			ui.Field(c, s.tr("toggle.encrypt_sign"), func() {
				ui.Switch(c, &s.encSign).Label(s.tr("toggle.encrypt_sign"))
			})
		})
		ui.Row(c).Gap(8).Padding(8, 0, 8, 0).Children(func() {
			if rnpPrimaryButton(c, s.tr("btn.encrypt_file")).Clicked() {
				s.run("action.encrypt_file", s.encryptFile)
			}
			if rnpButton(c, s.tr("btn.decrypt_file")).Clicked() {
				s.run("action.decrypt_file", s.decryptFile)
			}
			if rnpButton(c, s.tr("btn.sign_file")).Clicked() {
				s.run("action.sign_file", s.signFile)
			}
			if rnpButton(c, s.tr("btn.verify_file")).Clicked() {
				s.run("action.verify_file", s.verifyFile)
			}
		})
		ui.Text(c, s.tr("hint.file_alg")).FontSize(12)
		ui.TextArea(c, &s.textOutput).Height(320).Placeholder(s.tr("placeholder.result_output"))
	})
}

func (s *uiState) viewPackets(c *ui.Context) {
	rnpCard(c).Padding(14).Gap(8).Children(func() {
		ui.Form(c, func() {
			ui.Field(c, s.tr("label.packet_input_file"), func() {
				s.filePathEditors(c, &s.packetInput, "dlg.browse_packet", "", "")
			})
			ui.Field(c, s.tr("label.as_json"), func() { ui.Switch(c, &s.packetAsJSON).Label(s.tr("toggle.json")) })
			ui.Field(c, s.tr("label.feature_query"), func() {
				ui.TextInput(c, &s.featureQuery).Placeholder(s.tr("placeholder.feature_query"))
			})
		})
		ui.Row(c).Gap(8).Padding(8, 0, 8, 0).Children(func() {
			if rnpPrimaryButton(c, s.tr("btn.dump_packets")).Clicked() {
				s.run("action.dump_packets", s.dumpPackets)
			}
			if rnpButton(c, s.tr("btn.query_features")).Clicked() {
				s.run("action.query_features", s.queryFeatures)
			}
		})
		ui.Text(c, s.tr("hint.packet_alg")).FontSize(12)
		ui.Text(c, s.tr("label.packet_output")).Bold()
		ui.TextArea(c, &s.packetOut).Height(220)
		ui.Text(c, s.tr("label.features")).Bold().Padding(8, 0, 0, 0)
		ui.TextArea(c, &s.features).Height(220)
	})
}

func (s *uiState) viewPortable(c *ui.Context) {
	t := c.Theme()
	rnpCard(c).Padding(14).Gap(8).Children(func() {
		ui.Form(c, func() {
			ui.Field(c, s.tr("label.workspace"), func() { s.workspaceChooser(c) })
			ui.Field(c, s.tr("label.passphrase"), func() {
				s.passphraseRow(c, &s.passphrase, &s.showUnlockPass, s.tr("placeholder.unlock_pass"))
			})
		})
		ui.Row(c).Gap(8).Padding(8, 0, 8, 0).Children(func() {
			if rnpPrimaryButton(c, s.tr("btn.open_load")).Clicked() {
				s.run("action.open_load", s.openAndLoad)
			}
			if rnpButton(c, s.tr("btn.save")).Clicked() {
				s.run("action.save", s.saveKeyring)
			}
			if rnpButton(c, s.tr("btn.reload_keys")).Clicked() {
				s.run("action.list_keys", s.reloadKeys)
			}
		})
		ui.Text(c, s.tr("note.portable_files")).TextColor(t.TextMuted)
		ui.Text(c, s.tr("note.portable_pass")).TextColor(t.TextMuted)
	})
}

func (s *uiState) viewAboutWindow(c *ui.Context) {
	prepareSurface(c)
	t := c.Theme()
	absWS := s.workspace
	if strings.TrimSpace(absWS) != "" {
		if v, err := filepath.Abs(absWS); err == nil {
			absWS = v
		}
	}
	pubPath := filepath.Join(absWS, "pubring.gpg")
	secPath := filepath.Join(absWS, "secring.gpg")
	pubExists := fileExists(pubPath)
	secExists := fileExists(secPath)

	rnpStack := fmt.Sprintf("MyGo (https://github.com/egoist/mygo v0.10.0) | purego v0.11.1 | Botan 3.10.0")
	rnpUpstream := fmt.Sprintf("https://github.com/rnpgp/rnp (librnp %s)", s.version)

	ui.Column(c).Fill().Padding(16).Children(func() {
		rnpCard(c).Fill().Padding(16).Gap(12).Children(func() {
			ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
				ui.Text(c, s.tr("title.about")).FontSize(22).Bold().Grow(1)
				if rnpButton(c, s.tr("btn.close")).Clicked() {
					s.closeAboutWindow()
				}
			})
			ui.Text(c, s.tr("about.no_menu_hint")).FontSize(12).TextColor(t.TextMuted)
			ui.Divider(c)

			ui.Scroll(c).Fill().Children(func() {
				ui.Column(c).Gap(10).Children(func() {
					s.aboutField(c, s.tr("about.app_version"), appVersion)
					s.aboutField(c, s.tr("about.author"), "r404r (https://github.com/r404r)")
					s.aboutField(c, s.tr("about.repo_url"), "https://github.com/r404r/RnpArmor")
					s.aboutField(c, s.tr("about.rnp_upstream"), rnpUpstream)
					s.aboutField(c, s.tr("about.local_lib_path"), s.libPath)
					s.aboutField(c, s.tr("about.keyring_dir"), absWS)
					s.aboutField(c, s.tr("about.pubring_exists"), s.yesNo(pubExists))
					s.aboutField(c, s.tr("about.secring_exists"), s.yesNo(secExists))
					s.aboutField(c, s.tr("about.runtime"), rnpStack)
					ui.Text(c, s.tr("about.explicit_keyring_note")).FontSize(12).TextColor(t.TextMuted)
				})
			})
		})
	})
}

func (s *uiState) aboutField(c *ui.Context, label, value string) {
	ui.Field(c, label, func() {
		ui.TextInput(c, &value).ReadOnly(true)
	})
}

func (s *uiState) yesNo(v bool) string {
	if v {
		return s.tr("about.yes")
	}
	return s.tr("about.no")
}

type sessionAction func(sess *rnp.Session, ctx context.Context, in actionInput) (actionResult, error)

type persistMode int

const (
	persistNone persistMode = iota
	persistUpdate
	persistDelete
)

// actionInput is a main-thread snapshot. The worker must not read uiState.
type actionInput struct {
	libPath            string
	passphrase         string
	workspace          string
	selected           string
	keyReq             rnp.KeyGenRequest
	textInput          string
	textOutput         string
	textSigner         string
	textVerifyInput    string
	textVerifyDetached string
	textMode           string
	encRecipients      string
	encPassword        string
	encCipher          string
	encCompress        string
	encSign            bool
	fileSrc            string
	fileDst            string
	fileSig            string
	fileDetached       bool
	packetInput        string
	packetAsJSON       bool
	featureQuery       string
	langIdx            int
	importData         []byte
	exportPath         string
	exportKind         string
	subReq             rnp.SubkeyGenRequest
	targetFP           string
	expirySeconds      uint32
	revokeCode         string
	revokeReason       string
}

func (in actionInput) tr(key string) string {
	return Tr(LangByIndex(in.langIdx), key)
}

// actionResult carries UI writes. Pointer-free flags keep unset fields intact.
type actionResult struct {
	persist persistMode
	saved   bool

	hasKeyJSON            bool
	keyJSON               string
	hasTextOutput         bool
	textOutput            string
	hasTextVerifyInput    bool
	textVerifyInput       string
	hasTextVerifyDetached bool
	textVerifyDetached    string
	hasFeatures           bool
	features              string
	hasPacketOut          bool
	packetOut             string
	hasSessionLoaded      bool
	sessionLoaded         bool
	hasForceSelect        bool
	forceSelect           string
	hasCopy               bool
	copyText              string
	copyConcealed         bool
	hasSelectSubkey       bool
	selectSubkey          string
}

type actionOutcome struct {
	res      actionResult
	keys     []rnp.KeySummary
	selected string
	haveKeys bool
	err      error
}

func (s *uiState) snapshot() actionInput {
	in := actionInput{
		libPath:            s.libPath,
		passphrase:         s.passphrase,
		workspace:          s.workspace,
		selected:           s.selected,
		keyReq:             s.keyReq,
		textInput:          s.textInput,
		textOutput:         s.textOutput,
		textSigner:         s.textSigner,
		textVerifyInput:    s.textVerifyInput,
		textVerifyDetached: s.textVerifyDetached,
		textMode:           s.textMode,
		encRecipients:      s.encRecipients,
		encPassword:        s.encPassword,
		encCipher:          s.encCipher,
		encCompress:        s.encCompress,
		encSign:            s.encSign,
		fileSrc:            s.fileSrc,
		fileDst:            s.fileDst,
		fileSig:            s.fileSig,
		fileDetached:       s.fileDetached,
		packetInput:        s.packetInput,
		packetAsJSON:       s.packetAsJSON,
		featureQuery:       s.featureQuery,
		langIdx:            s.langIdx,
		importData:         s.importBytes,
		exportPath:         s.exportPath,
		exportKind:         s.exportKind,
		subReq:             s.subReq,
		targetFP:           s.targetFP,
		expirySeconds:      s.expirySeconds,
		revokeCode:         s.revokeCode,
		revokeReason:       s.revokeReason,
	}
	s.importBytes = nil
	s.exportPath = ""
	s.exportKind = ""
	return in
}

func invalidateWins(mainWin, aboutWin *mygo.Window) {
	if mainWin != nil && !mainWin.IsDestroyed() {
		mainWin.Invalidate()
	}
	if aboutWin != nil && !aboutWin.IsDestroyed() {
		aboutWin.Invalidate()
	}
}

func (s *uiState) run(title string, fn sessionAction) {
	s.startRun(title, fn, false)
}

func (s *uiState) runSilent(title string, fn sessionAction) {
	s.startRun(title, fn, true)
}

func (s *uiState) startRun(title string, fn sessionAction, silent bool) {
	s.mu.Lock()
	if s.busy {
		s.mu.Unlock()
		return
	}
	in := s.snapshot()
	mainWin := s.mainWin
	aboutWin := s.aboutWin
	headless := mainWin == nil || mainWin.IsDestroyed()
	s.busy = true
	s.statusKey = title
	s.statusDone = false
	s.lastErr = ""
	s.mu.Unlock()
	// Busy is visible before the worker starts. Do not hold s.mu across Invalidate.
	invalidateWins(mainWin, aboutWin)

	var done chan struct{}
	if headless {
		done = make(chan struct{})
	}
	go func() {
		if done != nil {
			defer close(done)
		}
		out := performAction(in, fn)
		if out.err == nil {
			if cerr := deliverClipboard(&out.res); cerr != nil {
				out.err = cerr
			}
		} else {
			out.res.hasCopy = false
			out.res.copyText = ""
		}
		s.applyUI(func() {
			s.applyResult(title, out)
			s.noteActionFeedback(title, out, silent)
			if out.err == nil && title == "action.load_keyring" {
				s.applyDemoScene()
			}
		})
	}()
	if done != nil {
		<-done
	}
}

// applyUI writes UI state on the main thread when a window exists.
// Headless tests run fn under s.mu and wait in run. Never call Update
// or Invalidate while holding s.mu: the UI thread also locks it.
func (s *uiState) applyUI(fn func()) {
	s.mu.Lock()
	win := s.mainWin
	useMain := win != nil && !win.IsDestroyed()
	s.mu.Unlock()
	if useMain {
		win.Update(func() {
			s.mu.Lock()
			fn()
			about := s.aboutWin
			newW := s.newKeyWin
			imp := s.importWin
			addW := s.addSubWin
			expW := s.expiryWin
			revW := s.revokeWin
			closeNew := s.pendingCloseNew
			closeImp := s.pendingCloseImport
			closeAdd := s.pendingCloseAddSub
			closeExp := s.pendingCloseExpiry
			closeRev := s.pendingCloseRevoke
			openAbout := s.demoOpenAbout
			s.pendingCloseNew = false
			s.pendingCloseImport = false
			s.pendingCloseAddSub = false
			s.pendingCloseExpiry = false
			s.pendingCloseRevoke = false
			s.demoOpenAbout = false
			s.mu.Unlock()
			if about != nil && !about.IsDestroyed() {
				about.Invalidate()
			}
			if newW != nil && !newW.IsDestroyed() {
				newW.Invalidate()
			}
			if imp != nil && !imp.IsDestroyed() {
				imp.Invalidate()
			}
			if addW != nil && !addW.IsDestroyed() {
				addW.Invalidate()
			}
			if expW != nil && !expW.IsDestroyed() {
				expW.Invalidate()
			}
			if revW != nil && !revW.IsDestroyed() {
				revW.Invalidate()
			}
			if closeNew && newW != nil && !newW.IsDestroyed() {
				newW.Close()
			}
			if closeImp && imp != nil && !imp.IsDestroyed() {
				imp.Close()
			}
			if closeAdd && addW != nil && !addW.IsDestroyed() {
				addW.Close()
			}
			if closeExp && expW != nil && !expW.IsDestroyed() {
				expW.Close()
			}
			if closeRev && revW != nil && !revW.IsDestroyed() {
				revW.Close()
			}
			if openAbout {
				s.openAboutWindow()
			}
		})
		return
	}
	s.mu.Lock()
	fn()
	s.pendingCloseNew = false
	s.pendingCloseImport = false
	s.pendingCloseAddSub = false
	s.pendingCloseExpiry = false
	s.pendingCloseRevoke = false
	s.mu.Unlock()
}

func (s *uiState) applyResult(title string, out actionOutcome) {
	res := out.res
	if res.hasKeyJSON {
		s.keyJSON = res.keyJSON
	}
	if res.hasTextOutput {
		s.textOutput = res.textOutput
	}
	if res.hasTextVerifyInput {
		s.textVerifyInput = res.textVerifyInput
	}
	if res.hasTextVerifyDetached {
		s.textVerifyDetached = res.textVerifyDetached
	}
	if res.hasFeatures {
		s.features = res.features
	}
	if res.hasPacketOut {
		s.packetOut = res.packetOut
	}
	if out.haveKeys {
		kept := s.selected
		s.keys = out.keys
		if res.hasForceSelect {
			s.selected = res.forceSelect
		} else {
			stillThere := false
			for i := range out.keys {
				if kept != "" && out.keys[i].Fingerprint == kept {
					stillThere = true
					break
				}
			}
			if stillThere {
				s.selected = kept
			} else {
				s.selected = out.selected
			}
		}
		if res.hasSelectSubkey {
			s.selectedSubkey = res.selectSubkey
		} else if s.selectedSubkey != "" && !subkeyStillPresent(s.keys, s.selected, s.selectedSubkey) {
			s.selectedSubkey = ""
		}
	}
	if res.hasSessionLoaded {
		s.sessionLoaded = res.sessionLoaded
	}
	s.busy = false
	s.statusKey = title
	if out.err != nil {
		s.lastErr = s.tr(title) + ": " + s.safeNotice(out.err.Error())
		s.statusDone = false
		s.closeImportOnSuccess = false
		if title == "action.generate_key" {
			s.newKeyErr = s.lastErr
		}
		if title == "action.import_keys" {
			s.importErr = s.lastErr
		}
		if title == "action.add_subkey" {
			s.subErr = s.lastErr
		}
		if title == "action.set_key_expiry" {
			s.expiryErr = s.lastErr
		}
		if title == "action.revoke_key" {
			s.revokeErr = s.lastErr
		}
		return
	}
	s.lastErr = ""
	s.statusDone = true
	if res.hasForceSelect && res.forceSelect != "" {
		s.pendingCloseNew = true
		s.newKeyErr = ""
	}
	if s.closeImportOnSuccess && title == "action.import_keys" {
		s.pendingCloseImport = true
		s.closeImportOnSuccess = false
		s.importText = ""
		s.importErr = ""
	}
	if title == "action.add_subkey" {
		s.pendingCloseAddSub = true
		s.resetAddSubPassUI()
	}
	if title == "action.set_key_expiry" {
		s.pendingCloseExpiry = true
		s.resetExpiryPassUI()
	}
	if title == "action.revoke_key" {
		s.pendingCloseRevoke = true
		s.resetRevokePassUI()
	}
}

func subkeyStillPresent(keys []rnp.KeySummary, primary, sub string) bool {
	if sub == "" {
		return false
	}
	for _, k := range keys {
		if k.Fingerprint != primary {
			continue
		}
		for _, sk := range k.Subkeys {
			if sk.Fingerprint == sub {
				return true
			}
		}
	}
	return false
}

func performAction(in actionInput, fn sessionAction) actionOutcome {
	pass := []byte(in.passphrase)
	defer func() {
		for i := range pass {
			pass[i] = 0
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	fail := func(res actionResult, err error) actionOutcome {
		return actionOutcome{res: res, selected: in.selected, err: err}
	}

	var (
		sess *rnp.Session
		err  error
	)
	if len(pass) == 0 {
		sess, err = rnp.OpenOptionalPass(in.libPath, nil)
	} else {
		sess, err = rnp.Open(in.libPath, pass)
	}
	if err != nil {
		return fail(actionResult{}, err)
	}
	defer sess.Close()

	ws := strings.TrimSpace(in.workspace)
	if ws != "" {
		if abs, aerr := normalizeWorkspace(ws); aerr == nil {
			ws = abs
		}
	}
	// Load when either keyring file exists. A public-only directory has
	// pubring.gpg and no secring.gpg. An empty directory is not imported.
	if ws != "" && hasKeyringFiles(ws) {
		if lerr := sess.LoadFrom(ctx, ws); lerr != nil {
			return fail(actionResult{}, lerr)
		}
	}

	// Session methods already call Do. sess.Go would lock Session.mu twice and deadlock.
	res, err := fn(sess, ctx, in)
	if err != nil {
		return fail(res, err)
	}
	if ws != "" && res.persist != persistNone {
		if perr := persistToWorkspace(sess, ctx, ws, res.persist == persistDelete); perr != nil {
			return fail(res, perr)
		}
		res.hasSessionLoaded = true
		res.sessionLoaded = true
	}
	if res.saved {
		res.hasSessionLoaded = true
		res.sessionLoaded = true
	}
	keys, selected, kerr := refreshKeyList(sess, ctx, in.selected)
	if kerr != nil {
		return fail(res, kerr)
	}
	return actionOutcome{res: res, keys: keys, selected: selected, haveKeys: true}
}

func (s *uiState) openSession(sess *rnp.Session, ctx context.Context, _ actionInput) (actionResult, error) {
	_, err := sess.Version(ctx)
	return actionResult{}, err
}

func (s *uiState) generateKey(sess *rnp.Session, ctx context.Context, in actionInput) (actionResult, error) {
	sum, err := sess.GenerateKeyWithSpec(ctx, in.keyReq)
	if err != nil {
		return actionResult{}, err
	}
	res := actionResult{persist: persistUpdate}
	if sum.Fingerprint != "" {
		res.hasForceSelect = true
		res.forceSelect = sum.Fingerprint
	}
	return res, nil
}

func (s *uiState) reloadKeys(_ *rnp.Session, _ context.Context, _ actionInput) (actionResult, error) {
	// Key list refresh happens in performAction after a successful call.
	return actionResult{}, nil
}

func (s *uiState) saveKeyring(sess *rnp.Session, ctx context.Context, in actionInput) (actionResult, error) {
	if strings.TrimSpace(in.workspace) == "" {
		return actionResult{}, fmt.Errorf("%s", in.tr("err.workspace_empty"))
	}
	if err := commitKeyring(sess, ctx, in.workspace, false); err != nil {
		return actionResult{}, err
	}
	return actionResult{saved: true}, nil
}

func (s *uiState) loadKeyring(sess *rnp.Session, ctx context.Context, in actionInput) (actionResult, error) {
	ws, err := normalizeWorkspace(in.workspace)
	if err != nil {
		if strings.TrimSpace(in.workspace) == "" {
			return actionResult{}, fmt.Errorf("%s", in.tr("err.workspace_empty"))
		}
		return actionResult{}, err
	}
	if err := sess.LoadFrom(ctx, ws); err != nil {
		return actionResult{}, err
	}
	return fillDemoPackets(sess, ctx, ws), nil
}

func (s *uiState) openAndLoad(sess *rnp.Session, ctx context.Context, in actionInput) (actionResult, error) {
	ws, err := normalizeWorkspace(in.workspace)
	if err != nil {
		if strings.TrimSpace(in.workspace) == "" {
			return actionResult{}, fmt.Errorf("%s", in.tr("err.workspace_empty"))
		}
		return actionResult{}, err
	}
	if err := sess.LoadFrom(ctx, ws); err != nil {
		return actionResult{}, err
	}
	return actionResult{hasSessionLoaded: true, sessionLoaded: true}, nil
}

func (s *uiState) loadKeyJSON(sess *rnp.Session, ctx context.Context, in actionInput) (actionResult, error) {
	if strings.TrimSpace(in.selected) == "" {
		return actionResult{}, fmt.Errorf("%s", in.tr("err.no_key_selected"))
	}
	v, err := sess.KeyToJSON(ctx, in.selected)
	if err != nil {
		return actionResult{}, err
	}
	return actionResult{hasKeyJSON: true, keyJSON: v}, nil
}

func (s *uiState) exportPublic(sess *rnp.Session, ctx context.Context, in actionInput) (actionResult, error) {
	return exportSelected(sess, ctx, in, false)
}

func (s *uiState) exportSecret(sess *rnp.Session, ctx context.Context, in actionInput) (actionResult, error) {
	return exportSelected(sess, ctx, in, true)
}

func exportSelected(sess *rnp.Session, ctx context.Context, in actionInput, secret bool) (actionResult, error) {
	if strings.TrimSpace(in.selected) == "" {
		return actionResult{}, fmt.Errorf("%s", in.tr("err.no_key_selected"))
	}
	v, err := sess.ExportKey(ctx, in.selected, secret)
	if err != nil {
		return actionResult{}, err
	}
	return actionResult{hasKeyJSON: true, keyJSON: v}, nil
}

func (s *uiState) deleteKey(sess *rnp.Session, ctx context.Context, in actionInput) (actionResult, error) {
	if strings.TrimSpace(in.selected) == "" {
		return actionResult{}, fmt.Errorf("%s", in.tr("err.no_key_selected"))
	}
	if err := sess.DeleteKey(ctx, in.selected); err != nil {
		return actionResult{}, err
	}
	return actionResult{persist: persistDelete}, nil
}

func (s *uiState) encryptText(sess *rnp.Session, ctx context.Context, in actionInput) (actionResult, error) {
	opts := rnp.EncryptOptions{
		RecipientFprints: recipientFingerprints(in),
		Password:         in.encPassword,
		SignerFprint:     pickSigner(in.encSign, in.textSigner, in.selected),
		Cipher:           in.encCipher,
		Compression:      in.encCompress,
		Armor:            true,
	}
	v, err := sess.EncryptText(ctx, in.textInput, opts)
	if err != nil {
		return actionResult{}, err
	}
	return actionResult{hasTextOutput: true, textOutput: v, hasTextVerifyInput: true, textVerifyInput: v}, nil
}

func (s *uiState) decryptText(sess *rnp.Session, ctx context.Context, in actionInput) (actionResult, error) {
	body := resolveDecryptInput(in.textInput, in.textVerifyInput, in.textOutput)
	v, err := sess.DecryptText(ctx, body)
	if err != nil {
		return actionResult{}, err
	}
	return actionResult{hasTextOutput: true, textOutput: v.Plaintext}, nil
}

func (s *uiState) signText(sess *rnp.Session, ctx context.Context, in actionInput) (actionResult, error) {
	signer := strings.TrimSpace(in.textSigner)
	if signer == "" {
		signer = strings.TrimSpace(in.selected)
	}
	v, err := sess.SignText(ctx, in.textInput, signer, rnp.SignMode(in.textMode), "SHA256")
	if err != nil {
		return actionResult{}, err
	}
	res := actionResult{hasTextOutput: true, textOutput: v, hasTextVerifyInput: true}
	if in.textMode == string(rnp.SignModeDetached) {
		res.textVerifyInput = in.textInput
		res.hasTextVerifyDetached = true
		res.textVerifyDetached = v
	} else {
		res.textVerifyInput = v
		res.hasTextVerifyDetached = true
		res.textVerifyDetached = ""
	}
	return res, nil
}

func (s *uiState) verifyText(sess *rnp.Session, ctx context.Context, in actionInput) (actionResult, error) {
	msg, detached := resolveVerifyInput(in.textInput, in.textVerifyInput, in.textOutput, in.textVerifyDetached)
	v, err := sess.VerifyText(ctx, msg, detached)
	if err != nil {
		return actionResult{}, err
	}
	return actionResult{hasTextOutput: true, textOutput: formatVerify(in, v)}, nil
}

func (s *uiState) encryptFile(sess *rnp.Session, ctx context.Context, in actionInput) (actionResult, error) {
	opts := rnp.EncryptOptions{
		RecipientFprints: recipientFingerprints(in),
		Password:         in.encPassword,
		SignerFprint:     pickSigner(in.encSign, in.textSigner, in.selected),
		Cipher:           in.encCipher,
		Compression:      in.encCompress,
		Armor:            strings.HasSuffix(strings.ToLower(in.fileDst), ".asc"),
	}
	return actionResult{}, sess.EncryptFileWithOpts(ctx, in.fileSrc, in.fileDst, opts)
}

func (s *uiState) decryptFile(sess *rnp.Session, ctx context.Context, in actionInput) (actionResult, error) {
	return actionResult{}, sess.DecryptFile(ctx, in.fileSrc, in.fileDst)
}

func (s *uiState) signFile(sess *rnp.Session, ctx context.Context, in actionInput) (actionResult, error) {
	signer := strings.TrimSpace(in.textSigner)
	if signer == "" {
		signer = strings.TrimSpace(in.selected)
	}
	mode := rnp.SignMode(in.textMode)
	if mode == "" {
		mode = rnp.SignModeDetached
	}
	return actionResult{}, sess.SignFileWithOpts(ctx, in.fileSrc, in.fileSig, signer, mode, true, "SHA256")
}

func (s *uiState) verifyFile(sess *rnp.Session, ctx context.Context, in actionInput) (actionResult, error) {
	v, err := sess.VerifyFileWithOpts(ctx, in.fileSrc, in.fileSig, in.fileDetached)
	if err != nil {
		return actionResult{}, err
	}
	return actionResult{hasTextOutput: true, textOutput: formatVerify(in, v)}, nil
}

func (s *uiState) dumpPackets(sess *rnp.Session, ctx context.Context, in actionInput) (actionResult, error) {
	b, err := os.ReadFile(in.packetInput)
	if err != nil {
		return actionResult{}, err
	}
	v, err := sess.DumpPackets(ctx, b, in.packetAsJSON)
	if err != nil {
		return actionResult{}, err
	}
	return actionResult{hasPacketOut: true, packetOut: v}, nil
}

func (s *uiState) queryFeatures(sess *rnp.Session, ctx context.Context, in actionInput) (actionResult, error) {
	v, err := sess.SupportedFeatures(ctx, in.featureQuery)
	if err != nil {
		return actionResult{}, err
	}
	return actionResult{hasFeatures: true, features: v}, nil
}

func formatVerify(in actionInput, v rnp.VerifyResult) string {
	mathBadge := in.tr("verify.math_invalid")
	if v.MathValid {
		mathBadge = in.tr("verify.math_valid")
	}
	identityBadge := in.tr("verify.identity_unverified")
	if v.IdentityTrusted {
		identityBadge = in.tr("verify.identity_trusted")
	}
	return fmt.Sprintf("%s %s\n%s=%d\n%s=%s\n%s=%s\n%s=%s\n%s=%d\n%s=%d",
		mathBadge,
		identityBadge,
		in.tr("verify.status"), v.Status,
		in.tr("verify.keyid"), v.SignerKeyID,
		in.tr("verify.fingerprint"), v.SignerFingerprint,
		in.tr("verify.hash"), v.HashAlg,
		in.tr("verify.created"), v.CreatedUnix,
		in.tr("verify.expires"), v.ExpiresUnix,
	)
}

func recipientFingerprints(in actionInput) []string {
	fps := splitCSV(in.encRecipients)
	if len(fps) == 0 && strings.TrimSpace(in.encPassword) == "" && strings.TrimSpace(in.selected) != "" {
		return []string{strings.TrimSpace(in.selected)}
	}
	return fps
}

// resolveDecryptInput picks the armored message the user can actually see.
// A pasted ciphertext in the input box wins over the hidden verify buffer.
func resolveDecryptInput(textInput, textVerifyInput, textOutput string) string {
	if strings.Contains(textInput, "-----BEGIN PGP MESSAGE-----") {
		return textInput
	}
	if strings.TrimSpace(textVerifyInput) != "" {
		return textVerifyInput
	}
	if strings.Contains(textOutput, "-----BEGIN PGP MESSAGE-----") {
		return textOutput
	}
	return textInput
}

// resolveVerifyInput returns the signed message and the detached signature.
// A non-empty detached signature keeps detached mode. Otherwise an armored
// blob in the visible input, the verify buffer, or the output is used.
func resolveVerifyInput(textInput, textVerifyInput, textOutput, detachedSig string) (string, string) {
	if strings.TrimSpace(detachedSig) != "" {
		plain := textInput
		if strings.TrimSpace(plain) == "" {
			plain = textVerifyInput
		}
		return plain, detachedSig
	}
	if strings.Contains(textInput, "-----BEGIN PGP ") {
		return textInput, ""
	}
	if strings.TrimSpace(textVerifyInput) != "" {
		return textVerifyInput, ""
	}
	if strings.Contains(textOutput, "-----BEGIN PGP ") {
		return textOutput, ""
	}
	return textInput, ""
}

func splitCSV(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func pickSigner(on bool, explicit, selected string) string {
	if !on {
		return ""
	}
	if strings.TrimSpace(explicit) != "" {
		return strings.TrimSpace(explicit)
	}
	return strings.TrimSpace(selected)
}

func normalizeWorkspace(v string) (string, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return "", fmt.Errorf("workspace is empty")
	}
	if filepath.IsAbs(v) {
		return v, nil
	}
	abs, err := filepath.Abs(v)
	if err != nil {
		return "", err
	}
	return abs, nil
}

func hasKeyringFiles(dir string) bool {
	if dir == "" || !filepath.IsAbs(dir) {
		return false
	}
	if fileExists(filepath.Join(dir, "pubring.gpg")) {
		return true
	}
	return fileExists(filepath.Join(dir, "secring.gpg"))
}

func (s *uiState) autoLoadKeyring() {
	abs, err := normalizeWorkspace(s.workspace)
	if err != nil || !hasKeyringFiles(abs) {
		return
	}
	s.workspace = abs
	s.runSilent("action.load_keyring", s.loadKeyring)
}

func fileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func refreshKeyList(sess *rnp.Session, ctx context.Context, selected string) ([]rnp.KeySummary, string, error) {
	keys, err := sess.ListKeys(ctx)
	if err != nil {
		return nil, selected, err
	}
	found := false
	for i := range keys {
		if keys[i].Fingerprint == selected {
			found = true
			break
		}
	}
	if !found {
		selected = defaultSelectedKey(keys)
	}
	return keys, selected, nil
}

// defaultSelectedKey prefers a primary secret key (CanCertify && HasSecret) so
// signing, exporting, and deleting default to the primary keypair rather than a
// subkey when librnp enumerates keys in arbitrary order.
func defaultSelectedKey(keys []rnp.KeySummary) string {
	for i := range keys {
		if keys[i].CanCertify && keys[i].HasSecret {
			return keys[i].Fingerprint
		}
	}
	for i := range keys {
		if keys[i].CanCertify {
			return keys[i].Fingerprint
		}
	}
	if len(keys) > 0 {
		return keys[0].Fingerprint
	}
	return ""
}

func persistToWorkspace(sess *rnp.Session, ctx context.Context, ws string, deleteEmpty bool) error {
	if strings.TrimSpace(ws) == "" {
		return nil
	}
	return commitKeyring(sess, ctx, ws, deleteEmpty)
}

// commitKeyring checks key counts before deleting pubring.gpg or secring.gpg.
// deleteEmpty is the delete-key path: an empty session removes the files and
// returns nil. An explicit save of an empty session returns ErrNoKey and
// leaves existing files in place.
func commitKeyring(sess *rnp.Session, ctx context.Context, ws string, deleteEmpty bool) error {
	abs, err := normalizeWorkspace(ws)
	if err != nil {
		return err
	}
	pubN, secN, err := sess.Counts(ctx)
	if err != nil {
		return err
	}
	pubPath := filepath.Join(abs, "pubring.gpg")
	secPath := filepath.Join(abs, "secring.gpg")
	if pubN == 0 && secN == 0 {
		if !deleteEmpty {
			return rnp.ErrNoKey
		}
		if err := removeKeyringFile(pubPath); err != nil {
			return err
		}
		return removeKeyringFile(secPath)
	}
	if err := os.MkdirAll(abs, 0o700); err != nil {
		return err
	}
	if err := removeKeyringFile(pubPath); err != nil {
		return err
	}
	if err := removeKeyringFile(secPath); err != nil {
		return err
	}
	return sess.SaveTo(ctx, abs)
}

func removeKeyringFile(path string) error {
	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func init() {
	log.SetFlags(0)
}
