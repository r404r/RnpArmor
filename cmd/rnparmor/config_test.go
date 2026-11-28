package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/egoist/mygo"
)

func TestPreferSavedWorkspace(t *testing.T) {
	const home = "/Users/me"
	cases := []struct {
		saved, fallback, want string
	}{
		{saved: "/Users/me/keys", fallback: "/", want: "/Users/me/keys"},
		{saved: "/home/me/keys", fallback: home, want: "/home/me/keys"},
		{saved: `C:\Users\me\keys`, fallback: "/", want: `C:\Users\me\keys`},
		{saved: "C:/Users/me/keys", fallback: "/", want: "C:/Users/me/keys"},
		{saved: "/", fallback: home, want: home},
		{saved: `C:\`, fallback: home, want: home},
		{saved: "C:/", fallback: home, want: home},
		{saved: "C:", fallback: home, want: home},
		{saved: "", fallback: home, want: home},
		{saved: ".", fallback: home, want: home},
		{saved: "keys", fallback: home, want: home},
		{saved: `C:keys`, fallback: home, want: home},
	}
	for _, tc := range cases {
		got := preferSavedWorkspace(tc.saved, tc.fallback)
		if got != tc.want {
			t.Errorf("preferSavedWorkspace(%q, %q) = %q, want %q", tc.saved, tc.fallback, got, tc.want)
		}
	}
}

func TestConfigFilePath(t *testing.T) {
	root := t.TempDir()
	exeDir := filepath.Join(root, "portable")
	userDir := filepath.Join(root, "user")
	beside := filepath.Join(exeDir, "config.json")
	userFile := filepath.Join(userDir, "RnpArmor", "config.json")

	if got := configFilePath(exeDir, userDir, true, true); got != beside {
		t.Fatalf("portable writable: got %q want %q", got, beside)
	}
	if got := configFilePath(exeDir, userDir, false, true); got != userFile {
		t.Fatalf("no PORTABLE.txt: got %q want %q", got, userFile)
	}
	if got := configFilePath(exeDir, userDir, true, false); got != userFile {
		t.Fatalf("portable not writable: got %q want %q", got, userFile)
	}
	appDir := filepath.Join(root, "RnpArmor.app", "Contents", "MacOS")
	if got := configFilePath(appDir, userDir, true, true); got != userFile {
		t.Fatalf("app bundle: got %q want %q", got, userFile)
	}
	if got := configFilePath("", "", false, false); got != "" {
		t.Fatalf("no dirs: got %q", got)
	}
}

func TestResolveConfigFilePortableAndUserDir(t *testing.T) {
	userDir := t.TempDir()
	exeDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(exeDir, "PORTABLE.txt"), []byte("portable\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	exe := filepath.Join(exeDir, "rnparmor.exe")
	got := resolveConfigFile(exe, userDir)
	want := filepath.Join(exeDir, "config.json")
	if got != want {
		t.Fatalf("PORTABLE.txt present: got %q want %q", got, want)
	}

	plainDir := t.TempDir()
	got = resolveConfigFile(filepath.Join(plainDir, "rnparmor"), userDir)
	want = filepath.Join(userDir, "RnpArmor", "config.json")
	if got != want {
		t.Fatalf("no PORTABLE.txt: got %q want %q", got, want)
	}

	appDir := filepath.Join(t.TempDir(), "RnpArmor.app", "Contents", "MacOS")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, "PORTABLE.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got = resolveConfigFile(filepath.Join(appDir, "rnparmor"), userDir)
	if got != want {
		t.Fatalf("bundle PORTABLE.txt: got %q want %q", got, want)
	}
}

func TestUserConfigRoundTripOmitsSecrets(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "RnpArmor", "config.json")
	s := &uiState{
		persistConfig:  true,
		configPath:     path,
		workspace:      "/home/me/keys",
		savedWorkspace: "/home/me/keys",
		keepWorkspace:  true,
		langIdx:        LangIndex(LangZH),
		langDirty:      true,
		tab:            0,
		keySplitPct:    40,
		textSplitPct:   61,
		fileSplitPct:   33,
		encCipher:      "AES128",
		encCompress:    "zip",
		packetAsJSON:   false,
		passphrase:     "super-secret-pass",
		encPassword:    "password-value",
		textInput:      "plaintext hello",
	}
	s.savePreferences()

	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	// Windows does not store Unix permission bits. Go reports 0666/0777
	// there, so the 0600/0700 checks only run on Unix.
	if runtime.GOOS != "windows" {
		if fi.Mode().Perm() != 0o600 {
			t.Fatalf("config mode %o, want 600", fi.Mode().Perm())
		}
	}
	di, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && di.Mode().Perm() != 0o700 {
		t.Fatalf("config dir mode %o, want 700", di.Mode().Perm())
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	low := strings.ToLower(string(b))
	for _, bad := range []string{"passphrase", "password", "plaintext", "super-secret", "ciphertext"} {
		if strings.Contains(low, bad) {
			t.Fatalf("config contains %q:\n%s", bad, b)
		}
	}
	if !strings.Contains(string(b), `"tab": 0`) || !strings.Contains(string(b), `"packetAsJSON": false`) {
		t.Fatalf("zero values were dropped:\n%s", b)
	}
	var raw map[string]any
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	allowed := map[string]bool{
		"workspace": true, "lang": true, "tab": true,
		"keySplitPct": true, "textSplitPct": true, "fileSplitPct": true,
		"encCipher": true, "encCompress": true, "packetAsJSON": true,
	}
	for k := range raw {
		if !allowed[k] {
			t.Fatalf("unexpected config key %q", k)
		}
	}

	cfg := loadUserConfig(path)
	if cfg.Workspace != "/home/me/keys" || cfg.Lang != "zh" || cfg.Tab == nil || *cfg.Tab != 0 {
		t.Fatalf("round trip header: %+v", cfg)
	}
	if cfg.KeySplitPct == nil || *cfg.KeySplitPct != 40 || cfg.TextSplitPct == nil || *cfg.TextSplitPct != 61 || cfg.FileSplitPct == nil || *cfg.FileSplitPct != 33 {
		t.Fatalf("round trip splits: %+v", cfg)
	}
	if cfg.EncCipher != "AES128" || cfg.EncCompress != "zip" || cfg.PacketAsJSON == nil || *cfg.PacketAsJSON {
		t.Fatalf("round trip cipher: %+v", cfg)
	}

	dst := &uiState{
		workspace:    "/tmp/fallback",
		langIdx:      LangIndex(LangEN),
		tab:          1,
		keySplitPct:  58,
		textSplitPct: studioSplitDefault,
		fileSplitPct: studioSplitDefault,
		encCipher:    "AES256",
		encCompress:  "zlib",
		packetAsJSON: true,
	}
	dst.applyUserConfig(cfg, false)
	if dst.workspace != "/home/me/keys" || dst.lang() != LangZH || dst.tab != 0 || dst.keySplitPct != 40 || dst.textSplitPct != 61 || dst.fileSplitPct != 33 {
		t.Fatalf("apply: ws=%q lang=%s tab=%d splits=%v %v %v", dst.workspace, dst.lang(), dst.tab, dst.keySplitPct, dst.textSplitPct, dst.fileSplitPct)
	}
	if dst.encCipher != "AES128" || dst.encCompress != "zip" || dst.packetAsJSON {
		t.Fatalf("apply cipher=%q compress=%q json=%v", dst.encCipher, dst.encCompress, dst.packetAsJSON)
	}
	if dst.langDirty {
		t.Fatal("loading lang marked it dirty")
	}
}

func TestConfigDoesNotStoreFilesystemRoot(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	s := &uiState{
		persistConfig:  true,
		configPath:     path,
		workspace:      "/",
		savedWorkspace: "/",
		keepWorkspace:  true,
		langIdx:        LangIndex(LangEN),
		keySplitPct:    58,
		textSplitPct:   studioSplitDefault,
		fileSplitPct:   studioSplitDefault,
		encCipher:      "AES256",
		encCompress:    "zlib",
		packetAsJSON:   true,
	}
	s.savePreferences()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), `"workspace"`) {
		t.Fatalf("root workspace was written:\n%s", b)
	}

	for _, root := range []string{"/", `C:\`, "C:/", "C:", "", ".", "rel/keys"} {
		loaded := loadUserConfig(path)
		loaded.Workspace = root
		dst := &uiState{workspace: "/home/me", keySplitPct: 58, textSplitPct: 54, fileSplitPct: 54, encCipher: "AES256", encCompress: "zlib", packetAsJSON: true}
		dst.applyUserConfig(loaded, false)
		if dst.workspace != "/home/me" || dst.keepWorkspace {
			t.Fatalf("root %q replaced workspace with %q keep=%v", root, dst.workspace, dst.keepWorkspace)
		}
	}
}

func TestCorruptConfigKeepsDefaultsAndFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	const bad = "{not json"
	if err := os.WriteFile(path, []byte(bad), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg := loadUserConfig(path)
	if cfg != (userConfig{}) {
		t.Fatalf("corrupt config = %+v", cfg)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != bad {
		t.Fatalf("corrupt file changed to %q", b)
	}
	if got := loadUserConfig(filepath.Join(dir, "missing.json")); got != (userConfig{}) {
		t.Fatalf("missing config = %+v", got)
	}

	dst := &uiState{
		workspace:    "/home/fallback",
		langIdx:      LangIndex(LangJA),
		tab:          2,
		keySplitPct:  58,
		textSplitPct: 54,
		fileSplitPct: 54,
		encCipher:    "AES256",
		encCompress:  "zlib",
		packetAsJSON: true,
	}
	dst.applyUserConfig(cfg, false)
	if dst.workspace != "/home/fallback" || dst.lang() != LangJA || dst.tab != 2 || !dst.packetAsJSON || dst.encCipher != "AES256" {
		t.Fatalf("defaults changed: %+v", dst)
	}
}

func TestLangFlagDoesNotRewriteSavedLang(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	s := &uiState{
		persistConfig: true,
		configPath:    path,
		langIdx:       LangIndex(LangEN),
		keySplitPct:   58,
		textSplitPct:  studioSplitDefault,
		fileSplitPct:  studioSplitDefault,
		encCipher:     "AES256",
		encCompress:   "zlib",
		packetAsJSON:  true,
	}
	s.applyUserConfig(userConfig{Lang: "zh"}, true)
	if s.lang() != LangEN || s.langDirty || s.savedLang != "zh" {
		t.Fatalf("flag override lang=%s dirty=%v saved=%q", s.lang(), s.langDirty, s.savedLang)
	}
	s.savePreferences()
	cfg := loadUserConfig(path)
	if cfg.Lang != "zh" {
		t.Fatalf("flag rewrote lang to %q", cfg.Lang)
	}

	s.setLang(LangJA)
	cfg = loadUserConfig(path)
	if cfg.Lang != "ja" || !s.langDirty {
		t.Fatalf("setLang lang=%q dirty=%v", cfg.Lang, s.langDirty)
	}
}

func TestAllowedQuitWritesPreferences(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	s := &uiState{
		persistConfig: true,
		configPath:    path,
		langIdx:       LangIndex(LangEN),
		tab:           4,
		keySplitPct:   42,
		textSplitPct:  55,
		fileSplitPct:  63,
		encCipher:     "Camellia256",
		encCompress:   "bzip2",
		packetAsJSON:  true,
		mainWin:       &mygo.Window{},
	}
	if runtime.GOOS != "darwin" {
		e := &mygo.QuitEvent{}
		s.handleBeforeQuit(e)
		if e.DefaultPrevented() {
			t.Fatal("non-darwin quit was intercepted")
		}
	} else {
		s.quitConfirmed = true
		s.handleBeforeQuit(&mygo.QuitEvent{})
	}
	cfg := loadUserConfig(path)
	if cfg.Tab == nil || *cfg.Tab != 4 || cfg.KeySplitPct == nil || *cfg.KeySplitPct != 42 {
		t.Fatalf("quit save: %+v", cfg)
	}
	if cfg.EncCipher != "Camellia256" || cfg.EncCompress != "bzip2" || cfg.PacketAsJSON == nil || !*cfg.PacketAsJSON {
		t.Fatalf("quit save cipher: %+v", cfg)
	}
	if cfg.Workspace != "" {
		t.Fatalf("quit saved startup workspace %q", cfg.Workspace)
	}

	off := &uiState{persistConfig: false, configPath: filepath.Join(dir, "nope.json"), tab: 3}
	off.handleBeforeQuit(nil)
	if _, err := os.Stat(off.configPath); !os.IsNotExist(err) {
		t.Fatal("disabled persist wrote a config file")
	}
}

func TestApplyRejectsInvalidPreferences(t *testing.T) {
	zero := float32(0)
	huge := float32(101)
	tab := 7
	s := &uiState{
		workspace:    "/keep",
		langIdx:      LangIndex(LangEN),
		tab:          1,
		keySplitPct:  58,
		textSplitPct: 54,
		fileSplitPct: 54,
		encCipher:    "AES256",
		encCompress:  "zlib",
		packetAsJSON: true,
	}
	s.applyUserConfig(userConfig{
		Workspace:    `C:\`,
		Lang:         "nope",
		Tab:          &tab,
		KeySplitPct:  &zero,
		TextSplitPct: &huge,
		EncCipher:    "IDEA",
		EncCompress:  "none",
	}, false)
	if s.workspace != "/keep" || s.lang() != LangEN || s.tab != 1 || s.keySplitPct != 58 || s.textSplitPct != 54 || s.encCipher != "AES256" || s.encCompress != "zlib" || !s.packetAsJSON {
		t.Fatalf("invalid config applied: ws=%q lang=%s tab=%d split=%v cipher=%s compress=%s json=%v", s.workspace, s.lang(), s.tab, s.keySplitPct, s.encCipher, s.encCompress, s.packetAsJSON)
	}
}
