package main

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// userConfig is the on-disk preference file. It has no passphrase,
// password, plaintext, ciphertext, or key material fields.
type userConfig struct {
	Workspace    string   `json:"workspace,omitempty"`
	Lang         string   `json:"lang,omitempty"`
	Tab          *int     `json:"tab,omitempty"`
	KeySplitPct  *float32 `json:"keySplitPct,omitempty"`
	TextSplitPct *float32 `json:"textSplitPct,omitempty"`
	FileSplitPct *float32 `json:"fileSplitPct,omitempty"`
	EncCipher    string   `json:"encCipher,omitempty"`
	EncCompress  string   `json:"encCompress,omitempty"`
	PacketAsJSON *bool    `json:"packetAsJSON,omitempty"`
}

func validSplitPct(v float32) bool {
	return v > 0 && v <= 100
}

func validEncCipher(v string) bool {
	switch v {
	case "AES256", "AES192", "AES128", "Camellia256":
		return true
	default:
		return false
	}
}

func validEncCompress(v string) bool {
	switch v {
	case "zlib", "bzip2", "zip", "uncompressed":
		return true
	default:
		return false
	}
}

// usableSavedWorkspace reports whether a stored path may replace the
// startup directory. It must be non-empty, absolute on either Unix or
// Windows, and not a filesystem root. Relative paths and roots are unsaved.
// The check is a string rule so Linux tests accept Windows paths.
func usableSavedWorkspace(p string) bool {
	p = strings.TrimSpace(p)
	if p == "" || startupWorkspaceNeedsHome(p) {
		return false
	}
	if strings.HasPrefix(p, "/") {
		return true
	}
	return isWindowsAbsPath(p)
}

func isWindowsAbsPath(p string) bool {
	if len(p) < 3 || p[1] != ':' || !isASCIILetter(p[0]) {
		return false
	}
	return p[2] == '/' || p[2] == '\\'
}

// preferSavedWorkspace keeps a usable saved path ahead of the startup
// fallback (working directory, or home when that directory is a root).
func preferSavedWorkspace(saved, fallback string) string {
	if usableSavedWorkspace(saved) {
		return strings.TrimSpace(saved)
	}
	return fallback
}

// pathInsideAppBundle reports a macOS .app Contents directory. Config must
// not be written inside the bundle, even if PORTABLE.txt is present.
func pathInsideAppBundle(p string) bool {
	n := strings.ReplaceAll(p, "\\", "/")
	return strings.Contains(n, ".app/Contents")
}

// configFilePath picks the config file from already-known facts.
// When PORTABLE.txt exists beside the executable and that directory is
// writable, the file is config.json there. Otherwise it is
// <userConfigDir>/RnpArmor/config.json. An empty user config dir with no
// usable portable dir returns an empty path.
func configFilePath(exeDir, userConfigDir string, portableExists, portableWritable bool) string {
	exeDir = strings.TrimSpace(exeDir)
	if portableExists && portableWritable && exeDir != "" && !pathInsideAppBundle(exeDir) {
		return filepath.Join(exeDir, "config.json")
	}
	userConfigDir = strings.TrimSpace(userConfigDir)
	if userConfigDir == "" {
		return ""
	}
	return filepath.Join(userConfigDir, "RnpArmor", "config.json")
}

func dirWritable(dir string) bool {
	f, err := os.CreateTemp(dir, ".rnparmor-w-*")
	if err != nil {
		return false
	}
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)
	return true
}

// resolveConfigFile looks for PORTABLE.txt beside exePath and probes
// whether that directory is writable. userConfigDir is passed in so tests
// never call os.UserConfigDir.
func resolveConfigFile(exePath, userConfigDir string) string {
	exeDir := ""
	if strings.TrimSpace(exePath) != "" {
		exeDir = filepath.Dir(exePath)
	}
	portable := false
	writable := false
	if exeDir != "" && !pathInsideAppBundle(exeDir) {
		st, err := os.Stat(filepath.Join(exeDir, "PORTABLE.txt"))
		if err == nil && !st.IsDir() {
			portable = true
			writable = dirWritable(exeDir)
		}
	}
	return configFilePath(exeDir, userConfigDir, portable, writable)
}

func locateUserConfigPath() string {
	exe, err := os.Executable()
	if err != nil {
		exe = ""
	}
	userDir, err := os.UserConfigDir()
	if err != nil {
		userDir = ""
	}
	return resolveConfigFile(exe, userDir)
}

// loadUserConfig reads preferences. A missing file, an empty path, or
// corrupt JSON yields the zero config and does not delete the file.
func loadUserConfig(path string) userConfig {
	path = strings.TrimSpace(path)
	if path == "" {
		return userConfig{}
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return userConfig{}
	}
	var cfg userConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return userConfig{}
	}
	return cfg
}

func writeUserConfig(path string, cfg userConfig) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
		if err := os.Chmod(dir, 0o700); err != nil {
			return err
		}
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	f, err := os.CreateTemp(dir, ".config-*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	done := false
	defer func() {
		if !done {
			_ = os.Remove(tmp)
		}
	}()
	if err := f.Chmod(0o600); err != nil {
		_ = f.Close()
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(path)
		if err2 := os.Rename(tmp, path); err2 != nil {
			return err2
		}
	}
	done = true
	return os.Chmod(path, 0o600)
}

// applyUserConfig overlays a loaded file onto defaults already stored in s.
// langFlagSet means --lang chose the language for this run; the file's lang
// stays savedLang and is not marked dirty.
func (s *uiState) applyUserConfig(cfg userConfig, langFlagSet bool) {
	s.savedLang = cfg.Lang
	s.langDirty = false
	if !langFlagSet {
		if l, ok := parseLang(cfg.Lang); ok {
			s.langIdx = LangIndex(l)
		}
	}
	if usableSavedWorkspace(cfg.Workspace) {
		ws := strings.TrimSpace(cfg.Workspace)
		s.workspace = ws
		s.savedWorkspace = ws
		s.keepWorkspace = true
	}
	if cfg.Tab != nil && *cfg.Tab >= 0 && *cfg.Tab <= 4 {
		s.tab = *cfg.Tab
	}
	if cfg.KeySplitPct != nil && validSplitPct(*cfg.KeySplitPct) {
		s.keySplitPct = *cfg.KeySplitPct
	}
	if cfg.TextSplitPct != nil && validSplitPct(*cfg.TextSplitPct) {
		s.textSplitPct = *cfg.TextSplitPct
	}
	if cfg.FileSplitPct != nil && validSplitPct(*cfg.FileSplitPct) {
		s.fileSplitPct = *cfg.FileSplitPct
	}
	if validEncCipher(cfg.EncCipher) {
		s.encCipher = cfg.EncCipher
	}
	if validEncCompress(cfg.EncCompress) {
		s.encCompress = cfg.EncCompress
	}
	if cfg.PacketAsJSON != nil {
		s.packetAsJSON = *cfg.PacketAsJSON
	}
}

func (s *uiState) snapshotUserConfig() userConfig {
	cfg := userConfig{}
	if s.keepWorkspace && usableSavedWorkspace(s.savedWorkspace) {
		cfg.Workspace = strings.TrimSpace(s.savedWorkspace)
	}
	if s.langDirty {
		cfg.Lang = string(s.lang())
	} else {
		cfg.Lang = s.savedLang
	}
	tab := s.tab
	if tab < 0 || tab > 4 {
		tab = 0
	}
	cfg.Tab = &tab
	key, text, file := s.keySplitPct, s.textSplitPct, s.fileSplitPct
	cfg.KeySplitPct = &key
	cfg.TextSplitPct = &text
	cfg.FileSplitPct = &file
	if validEncCipher(s.encCipher) {
		cfg.EncCipher = s.encCipher
	}
	if validEncCompress(s.encCompress) {
		cfg.EncCompress = s.encCompress
	}
	pkt := s.packetAsJSON
	cfg.PacketAsJSON = &pkt
	return cfg
}

func (s *uiState) savePreferences() {
	if s == nil || !s.persistConfig || strings.TrimSpace(s.configPath) == "" {
		return
	}
	if err := writeUserConfig(s.configPath, s.snapshotUserConfig()); err != nil {
		log.Printf("save config: %v", err)
	}
}
