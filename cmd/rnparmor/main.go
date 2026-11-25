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

	libPath    string
	version    string
	lastErr    string
	busy       bool
	statusKey  string
	statusDone bool
	workspace  string
	passphrase string

	keys      []rnp.KeySummary
	selected  string
	keyJSON   string
	features  string
	packetOut string

	keyReq rnp.KeyGenRequest

	textInput          string
	textOutput         string
	textDetached       string
	textSigner         string
	textVerifyInput    string
	textVerifyDetached string
	textMode           string

	encRecipients string
	encPassword   string
	encCipher     string
	encCompress   string
	encSign       bool

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
		libPath:       path,
		version:       v,
		statusKey:     "status.ready",
		statusDone:    false,
		workspace:     mustWd(),
		passphrase:    "change-me",
		encCipher:     "AES256",
		encCompress:   "zlib",
		textMode:      string(rnp.SignModeDetached),
		featureQuery:  "",
		packetAsJSON:  true,
		textInput:     "hello from RnpArmor",
		textVerifyInput: "",
		langIdx:       LangIndex(lang),
		keyReq: rnp.KeyGenRequest{
			UserID:      "User <user@example.com>",
			PrimaryAlg:  "EDDSA",
			PrimaryCurve: "Ed25519",
			SubkeyAlg:   "ECDH",
			SubkeyCurve: "Curve25519",
			HashAlg:     "SHA256",
			SymmCipher:  "AES256",
		},
	}

	mygo.App.WhenReady(func() {
		if runtime.GOOS == "darwin" {
			if err := mygo.App.Dock.SetIcon(icons.AppIconPNG); err != nil {
				log.Printf("dock icon: %v", err)
			}
		}
		w := mygo.NewWindow(mygo.WindowOptions{
			Title:   "RnpArmor",
			Width:   1120,
			Height:  780,
			Content: ui.View(st.view),
		})
		if err := w.SetIcon(icons.AppIconPNG); err != nil {
			log.Printf("window icon: %v", err)
		}
		w.OnClosed(func() {
			st.closeAboutWindow()
		})
		st.mainWin = w
		st.applyMainMenu()
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
	if s.mainWin == nil || s.mainWin.IsDestroyed() {
		return
	}
	menu := mygo.NewMenu(s.menuTemplate())
	if runtime.GOOS == "darwin" {
		mygo.App.SetMenu(menu)
		return
	}
	s.mainWin.SetMenu(menu)
}

func (s *uiState) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Padding(14).Gap(10).Children(func() {
		ui.Column(c).Gap(8).Children(func() {
			ui.Text(c, s.tr("title.app")).FontSize(24).Bold()
			ui.Textf(c, "librnp %s", s.version).TextColor(t.TextMuted)
			ui.Text(c, s.libPath).FontSize(11).TextColor(t.TextMuted)
			ui.Text(c, s.statusLine()).FontSize(12).TextColor(t.TextMuted)
		})
		ui.Divider(c)

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

		ui.Scroll(c).Fill().Children(func() {
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

func (s *uiState) viewKeys(c *ui.Context) {
	t := c.Theme()
	ui.Form(c, func() {
		ui.Field(c, s.tr("label.passphrase"), func() {
			ui.TextInput(c, &s.passphrase).Placeholder(s.tr("placeholder.session_passphrase"))
		})
		ui.Field(c, s.tr("label.workspace"), func() {
			ui.TextInput(c, &s.workspace).Placeholder(s.tr("placeholder.workspace"))
		})
		ui.Field(c, s.tr("label.user_id"), func() {
			ui.TextInput(c, &s.keyReq.UserID)
		})
		ui.Field(c, s.tr("label.primary"), func() {
			ui.Row(c).Gap(8).Children(func() {
				ui.Select(c, &s.keyReq.PrimaryAlg, []string{"EDDSA", "RSA", "ECDSA", "DSA"}).Width(180)
				ui.TextInput(c, &s.keyReq.PrimaryCurve).Placeholder(s.tr("placeholder.curve"))
				bits := float64(s.keyReq.PrimaryBits)
				if ui.NumberInput(c, &bits, 0, 8192, 1).Width(120).Changed() {
					s.keyReq.PrimaryBits = int(bits)
				}
			})
		})
		ui.Field(c, s.tr("label.subkey"), func() {
			ui.Row(c).Gap(8).Children(func() {
				ui.Select(c, &s.keyReq.SubkeyAlg, []string{"ECDH", "RSA", "ELGAMAL", ""}).Width(180)
				ui.TextInput(c, &s.keyReq.SubkeyCurve).Placeholder(s.tr("placeholder.curve"))
				bits := float64(s.keyReq.SubkeyBits)
				if ui.NumberInput(c, &bits, 0, 8192, 1).Width(120).Changed() {
					s.keyReq.SubkeyBits = int(bits)
				}
			})
		})
		ui.Field(c, s.tr("label.crypto"), func() {
			ui.Row(c).Gap(8).Children(func() {
				ui.Select(c, &s.keyReq.HashAlg, []string{"SHA256", "SHA384", "SHA512", "SHA3-256"}).Width(180)
				ui.Select(c, &s.keyReq.SymmCipher, []string{"AES256", "AES192", "AES128", "Camellia256"}).Width(180)
			})
		})
	})

	ui.Row(c).Gap(8).Padding(8, 0, 12, 0).Children(func() {
		if ui.PrimaryButton(c, s.tr("btn.open_session")).Clicked() {
			s.run("action.open_session", s.openSession)
		}
		if ui.Button(c, s.tr("btn.generate_key")).Clicked() {
			s.run("action.generate_key", s.generateKey)
		}
		if ui.Button(c, s.tr("btn.reload_keys")).Clicked() {
			s.run("action.list_keys", s.reloadKeys)
		}
		if ui.Button(c, s.tr("btn.save_keyring")).Clicked() {
			s.run("action.save_keyring", s.saveKeyring)
		}
		if ui.Button(c, s.tr("btn.load_keyring")).Clicked() {
			s.run("action.load_keyring", s.loadKeyring)
		}
	})

	ui.Divider(c)
	ui.Text(c, s.tr("label.keyring")).Bold().Padding(8, 0, 4, 0)
	ui.Text(c, s.tr("hint.keys_alg")).FontSize(12).Padding(0, 0, 6, 0)
	for i := range s.keys {
		k := s.keys[i]
		line := fmt.Sprintf("%s | %s | %s", shorten(k.Fingerprint), k.Algorithm, k.PrimaryUID)
		if ui.Button(c, line).Clicked() {
			s.selected = k.Fingerprint
		}
	}
	if s.selected != "" {
		ui.Textf(c, "%s: %s", s.tr("label.selected"), s.selected).TextColor(t.TextMuted)
		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c, s.tr("btn.key_json")).Clicked() {
				s.run("action.key_json", s.loadKeyJSON)
			}
			if ui.Button(c, s.tr("btn.export_public")).Clicked() {
				s.run("action.export_public", func(sess *rnp.Session, ctx context.Context) error {
					v, err := sess.ExportKey(ctx, s.selected, false)
					if err == nil {
						s.keyJSON = v
					}
					return err
				})
			}
			if ui.Button(c, s.tr("btn.export_secret")).Clicked() {
				s.run("action.export_secret", func(sess *rnp.Session, ctx context.Context) error {
					v, err := sess.ExportKey(ctx, s.selected, true)
					if err == nil {
						s.keyJSON = v
					}
					return err
				})
			}
			if ui.Button(c, s.tr("btn.delete_key")).Clicked() {
				s.run("action.delete_key", func(sess *rnp.Session, ctx context.Context) error {
					return sess.DeleteKey(ctx, s.selected)
				})
			}
		})
	}
	ui.TextArea(c, &s.keyJSON).Height(220).Placeholder(s.tr("placeholder.key_output"))
}

func (s *uiState) viewTextStudio(c *ui.Context) {
	ui.Form(c, func() {
		ui.Field(c, s.tr("label.plain_text"), func() {
			ui.TextArea(c, &s.textInput).Height(120)
		})
		ui.Field(c, s.tr("label.recipients"), func() {
			ui.TextInput(c, &s.encRecipients)
		})
		ui.Field(c, s.tr("label.password"), func() {
			ui.TextInput(c, &s.encPassword)
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
			ui.TextInput(c, &s.textSigner)
		})
		ui.Field(c, s.tr("label.detached_verify_sig"), func() {
			ui.TextArea(c, &s.textVerifyDetached).Height(90)
		})
	})

	ui.Row(c).Gap(8).Padding(8, 0, 8, 0).Children(func() {
		if ui.PrimaryButton(c, s.tr("btn.encrypt_text")).Clicked() {
			s.run("action.encrypt_text", s.encryptText)
		}
		if ui.Button(c, s.tr("btn.decrypt_text")).Clicked() {
			s.run("action.decrypt_text", s.decryptText)
		}
		if ui.Button(c, s.tr("btn.sign_text")).Clicked() {
			s.run("action.sign_text", s.signText)
		}
		if ui.Button(c, s.tr("btn.verify_text")).Clicked() {
			s.run("action.verify_text", s.verifyText)
		}
	})
	ui.Text(c, s.tr("hint.text_alg")).FontSize(12)
	ui.Text(c, s.tr("hint.verify_badge")).FontSize(12)
	ui.TextArea(c, &s.textOutput).Height(280).Placeholder(s.tr("placeholder.result_output"))
}

func (s *uiState) viewFileWorkbench(c *ui.Context) {
	ui.Form(c, func() {
		ui.Field(c, s.tr("label.source"), func() { ui.TextInput(c, &s.fileSrc) })
		ui.Field(c, s.tr("label.destination"), func() { ui.TextInput(c, &s.fileDst) })
		ui.Field(c, s.tr("label.signature_path"), func() { ui.TextInput(c, &s.fileSig) })
		ui.Field(c, s.tr("label.detached_verify"), func() { ui.Switch(c, &s.fileDetached).Label(s.tr("toggle.detached")) })
	})
	ui.Row(c).Gap(8).Padding(8, 0, 8, 0).Children(func() {
		if ui.PrimaryButton(c, s.tr("btn.encrypt_file")).Clicked() {
			s.run("action.encrypt_file", s.encryptFile)
		}
		if ui.Button(c, s.tr("btn.decrypt_file")).Clicked() {
			s.run("action.decrypt_file", s.decryptFile)
		}
		if ui.Button(c, s.tr("btn.sign_file")).Clicked() {
			s.run("action.sign_file", s.signFile)
		}
		if ui.Button(c, s.tr("btn.verify_file")).Clicked() {
			s.run("action.verify_file", s.verifyFile)
		}
	})
	ui.Text(c, s.tr("hint.file_alg")).FontSize(12)
	ui.TextArea(c, &s.textOutput).Height(320).Placeholder(s.tr("placeholder.result_output"))
}

func (s *uiState) viewPackets(c *ui.Context) {
	ui.Form(c, func() {
		ui.Field(c, s.tr("label.packet_input_file"), func() { ui.TextInput(c, &s.packetInput) })
		ui.Field(c, s.tr("label.as_json"), func() { ui.Switch(c, &s.packetAsJSON).Label(s.tr("toggle.json")) })
		ui.Field(c, s.tr("label.feature_query"), func() {
			ui.TextInput(c, &s.featureQuery).Placeholder(s.tr("placeholder.feature_query"))
		})
	})
	ui.Row(c).Gap(8).Padding(8, 0, 8, 0).Children(func() {
		if ui.PrimaryButton(c, s.tr("btn.dump_packets")).Clicked() {
			s.run("action.dump_packets", s.dumpPackets)
		}
		if ui.Button(c, s.tr("btn.query_features")).Clicked() {
			s.run("action.query_features", s.queryFeatures)
		}
	})
	ui.Text(c, s.tr("hint.packet_alg")).FontSize(12)
	ui.Text(c, s.tr("label.packet_output")).Bold()
	ui.TextArea(c, &s.packetOut).Height(220)
	ui.Text(c, s.tr("label.features")).Bold().Padding(8, 0, 0, 0)
	ui.TextArea(c, &s.features).Height(220)
}

func (s *uiState) viewPortable(c *ui.Context) {
	t := c.Theme()
	ui.Form(c, func() {
		ui.Field(c, s.tr("label.workspace"), func() { ui.TextInput(c, &s.workspace) })
		ui.Field(c, s.tr("label.passphrase"), func() { ui.TextInput(c, &s.passphrase) })
	})
	ui.Row(c).Gap(8).Padding(8, 0, 8, 0).Children(func() {
		if ui.PrimaryButton(c, s.tr("btn.open_load")).Clicked() {
			s.run("action.open_load", func(sess *rnp.Session, ctx context.Context) error {
				if err := sess.LoadFrom(ctx, s.workspace); err != nil {
					return err
				}
				s.sessionLoaded = true
				return nil
			})
		}
		if ui.Button(c, s.tr("btn.save")).Clicked() {
			s.run("action.save", s.saveKeyring)
		}
		if ui.Button(c, s.tr("btn.reload_keys")).Clicked() {
			s.run("action.list_keys", s.reloadKeys)
		}
	})
	ui.Text(c, s.tr("note.portable_files")).TextColor(t.TextMuted)
	ui.Text(c, s.tr("note.portable_pass")).TextColor(t.TextMuted)
}

func (s *uiState) viewAboutWindow(c *ui.Context) {
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

	ui.Column(c).Fill().Padding(20).Gap(12).Children(func() {
		ui.Row(c).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Text(c, s.tr("title.about")).FontSize(22).Bold().Grow(1)
			if ui.Button(c, s.tr("btn.close")).Clicked() {
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

func (s *uiState) run(title string, fn func(sess *rnp.Session, ctx context.Context) error) {
	s.mu.Lock()
	if s.busy {
		s.mu.Unlock()
		return
	}
	lib := s.libPath
	passphrase := s.passphrase
	workspace := strings.TrimSpace(s.workspace)
	loaded := s.sessionLoaded
	s.busy = true
	s.statusKey = title
	s.statusDone = false
	s.lastErr = ""
	s.mu.Unlock()

	pass := []byte(passphrase)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)

	go func() {
		defer cancel()
		defer func() {
			for i := range pass {
				pass[i] = 0
			}
		}()

		sess, err := rnp.Open(lib, pass)
		if err != nil {
			s.finish(title, err)
			return
		}
		defer sess.Close()

		if workspace != "" {
			if abs, aerr := normalizeWorkspace(workspace); aerr == nil {
				workspace = abs
			}
		}
		if workspace != "" && (loaded || hasKeyringFiles(workspace)) {
			if lerr := sess.LoadFrom(ctx, workspace); lerr != nil {
				s.finish(title, lerr)
				return
			}
		}
		err = <-sess.Go(ctx, func(ctx context.Context) error {
			return fn(sess, ctx)
		})
		if err == nil {
			if workspace != "" {
				if perr := s.persistToWorkspace(sess, ctx, workspace); perr != nil {
					err = perr
				} else {
					s.mu.Lock()
					s.sessionLoaded = true
					s.mu.Unlock()
				}
			}
			if err == nil {
				_ = s.pullKeys(sess, ctx)
			}
		}
		s.finish(title, err)
	}()
}

func (s *uiState) finish(title string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.busy = false
	s.statusKey = title
	if err != nil {
		s.lastErr = s.tr(title) + ": " + err.Error()
		s.statusDone = false
	} else {
		s.lastErr = ""
		s.statusDone = true
	}
	go s.invalidateAll()
}

func (s *uiState) openSession(sess *rnp.Session, ctx context.Context) error {
	_, err := sess.Version(ctx)
	return err
}

func (s *uiState) generateKey(sess *rnp.Session, ctx context.Context) error {
	_, err := sess.GenerateKeyWithSpec(ctx, s.keyReq)
	return err
}

func (s *uiState) reloadKeys(sess *rnp.Session, ctx context.Context) error {
	return s.pullKeys(sess, ctx)
}

func (s *uiState) pullKeys(sess *rnp.Session, ctx context.Context) error {
	keys, err := sess.ListKeys(ctx)
	if err != nil {
		return err
	}
	s.keys = keys
	if s.selected == "" && len(keys) > 0 {
		s.selected = keys[0].Fingerprint
	}
	return nil
}

func (s *uiState) saveKeyring(sess *rnp.Session, ctx context.Context) error {
	ws, err := normalizeWorkspace(s.workspace)
	if err != nil {
		if strings.TrimSpace(s.workspace) == "" {
			return fmt.Errorf("%s", s.tr("err.workspace_empty"))
		}
		return err
	}
	if err := os.MkdirAll(ws, 0o700); err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(ws, "pubring.gpg")); err == nil {
		_ = os.Remove(filepath.Join(ws, "pubring.gpg"))
	}
	if _, err := os.Stat(filepath.Join(ws, "secring.gpg")); err == nil {
		_ = os.Remove(filepath.Join(ws, "secring.gpg"))
	}
	return sess.SaveTo(ctx, ws)
}

func (s *uiState) loadKeyring(sess *rnp.Session, ctx context.Context) error {
	ws, err := normalizeWorkspace(s.workspace)
	if err != nil {
		if strings.TrimSpace(s.workspace) == "" {
			return fmt.Errorf("%s", s.tr("err.workspace_empty"))
		}
		return err
	}
	return sess.LoadFrom(ctx, ws)
}

func (s *uiState) loadKeyJSON(sess *rnp.Session, ctx context.Context) error {
	if strings.TrimSpace(s.selected) == "" {
		return fmt.Errorf("%s", s.tr("err.no_key_selected"))
	}
	v, err := sess.KeyToJSON(ctx, s.selected)
	if err == nil {
		s.keyJSON = v
	}
	return err
}

func (s *uiState) encryptText(sess *rnp.Session, ctx context.Context) error {
	opts := rnp.EncryptOptions{
		RecipientFprints: splitCSV(s.encRecipients),
		Password:         s.encPassword,
		SignerFprint:     pickSigner(s.encSign, s.textSigner, s.selected),
		Cipher:           s.encCipher,
		Compression:      s.encCompress,
		Armor:            true,
	}
	v, err := sess.EncryptText(ctx, s.textInput, opts)
	if err == nil {
		s.textOutput = v
		s.textVerifyInput = v
	}
	return err
}

func (s *uiState) decryptText(sess *rnp.Session, ctx context.Context) error {
	v, err := sess.DecryptText(ctx, s.textVerifyInput)
	if err == nil {
		s.textOutput = v.Plaintext
	}
	return err
}

func (s *uiState) signText(sess *rnp.Session, ctx context.Context) error {
	signer := strings.TrimSpace(s.textSigner)
	if signer == "" {
		signer = s.selected
	}
	v, err := sess.SignText(ctx, s.textInput, signer, rnp.SignMode(s.textMode), "SHA256")
	if err == nil {
		s.textOutput = v
		if s.textMode == string(rnp.SignModeDetached) {
			s.textVerifyDetached = v
			s.textVerifyInput = s.textInput
		} else {
			s.textVerifyInput = v
		}
	}
	return err
}

func (s *uiState) verifyText(sess *rnp.Session, ctx context.Context) error {
	v, err := sess.VerifyText(ctx, s.textVerifyInput, s.textVerifyDetached)
	if err == nil {
		mathBadge := s.tr("verify.math_invalid")
		if v.MathValid {
			mathBadge = s.tr("verify.math_valid")
		}
		identityBadge := s.tr("verify.identity_unverified")
		if v.IdentityTrusted {
			identityBadge = s.tr("verify.identity_trusted")
		}
		s.textOutput = fmt.Sprintf("%s %s\n%s=%d\n%s=%s\n%s=%s\n%s=%s\n%s=%d\n%s=%d",
			mathBadge,
			identityBadge,
			s.tr("verify.status"), v.Status,
			s.tr("verify.keyid"), v.SignerKeyID,
			s.tr("verify.fingerprint"), v.SignerFingerprint,
			s.tr("verify.hash"), v.HashAlg,
			s.tr("verify.created"), v.CreatedUnix,
			s.tr("verify.expires"), v.ExpiresUnix,
		)
	}
	return err
}

func (s *uiState) encryptFile(sess *rnp.Session, ctx context.Context) error {
	opts := rnp.EncryptOptions{
		RecipientFprints: splitCSV(s.encRecipients),
		Password:         s.encPassword,
		SignerFprint:     pickSigner(s.encSign, s.textSigner, s.selected),
		Cipher:           s.encCipher,
		Compression:      s.encCompress,
		Armor:            strings.HasSuffix(strings.ToLower(s.fileDst), ".asc"),
	}
	return sess.EncryptFileWithOpts(ctx, s.fileSrc, s.fileDst, opts)
}

func (s *uiState) decryptFile(sess *rnp.Session, ctx context.Context) error {
	return sess.DecryptFile(ctx, s.fileSrc, s.fileDst)
}

func (s *uiState) signFile(sess *rnp.Session, ctx context.Context) error {
	signer := strings.TrimSpace(s.textSigner)
	if signer == "" {
		signer = s.selected
	}
	mode := rnp.SignMode(s.textMode)
	if mode == "" {
		mode = rnp.SignModeDetached
	}
	return sess.SignFileWithOpts(ctx, s.fileSrc, s.fileSig, signer, mode, true, "SHA256")
}

func (s *uiState) verifyFile(sess *rnp.Session, ctx context.Context) error {
	v, err := sess.VerifyFileWithOpts(ctx, s.fileSrc, s.fileSig, s.fileDetached)
	if err == nil {
		mathBadge := s.tr("verify.math_invalid")
		if v.MathValid {
			mathBadge = s.tr("verify.math_valid")
		}
		identityBadge := s.tr("verify.identity_unverified")
		if v.IdentityTrusted {
			identityBadge = s.tr("verify.identity_trusted")
		}
		s.textOutput = fmt.Sprintf("%s %s\n%s=%d\n%s=%s\n%s=%s\n%s=%s\n%s=%d\n%s=%d",
			mathBadge,
			identityBadge,
			s.tr("verify.status"), v.Status,
			s.tr("verify.keyid"), v.SignerKeyID,
			s.tr("verify.fingerprint"), v.SignerFingerprint,
			s.tr("verify.hash"), v.HashAlg,
			s.tr("verify.created"), v.CreatedUnix,
			s.tr("verify.expires"), v.ExpiresUnix,
		)
	}
	return err
}

func (s *uiState) dumpPackets(sess *rnp.Session, ctx context.Context) error {
	b, err := os.ReadFile(s.packetInput)
	if err != nil {
		return err
	}
	v, err := sess.DumpPackets(ctx, b, s.packetAsJSON)
	if err == nil {
		s.packetOut = v
	}
	return err
}

func (s *uiState) queryFeatures(sess *rnp.Session, ctx context.Context) error {
	v, err := sess.SupportedFeatures(ctx, s.featureQuery)
	if err == nil {
		s.features = v
	}
	return err
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

func shorten(fp string) string {
	if len(fp) <= 16 {
		return fp
	}
	return fp[:8] + "…" + fp[len(fp)-8:]
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
	if _, err := os.Stat(filepath.Join(dir, "pubring.gpg")); err != nil {
		return false
	}
	if _, err := os.Stat(filepath.Join(dir, "secring.gpg")); err != nil {
		return false
	}
	return true
}

func fileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func (s *uiState) persistToWorkspace(sess *rnp.Session, ctx context.Context, ws string) error {
	if ws == "" {
		return nil
	}
	if !filepath.IsAbs(ws) {
		var err error
		ws, err = filepath.Abs(ws)
		if err != nil {
			return err
		}
	}
	if err := os.MkdirAll(ws, 0o700); err != nil {
		return err
	}
	_ = os.Remove(filepath.Join(ws, "pubring.gpg"))
	_ = os.Remove(filepath.Join(ws, "secring.gpg"))
	return sess.SaveTo(ctx, ws)
}

func init() {
	log.SetFlags(0)
}
