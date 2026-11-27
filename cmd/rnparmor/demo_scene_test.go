package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/r404r/RnpArmor/internal/rnp"
)

func TestDemoSceneName(t *testing.T) {
	t.Setenv("RNPARMOR_DEMO_SCENE", "")
	if demoScene() != "" {
		t.Fatal("empty")
	}
	t.Setenv("RNPARMOR_DEMO_SCENE", "  KEYS ")
	if demoScene() != "keys" {
		t.Fatalf("keys %q", demoScene())
	}
	for _, name := range []string{"text", "file", "packet", "about"} {
		t.Setenv("RNPARMOR_DEMO_SCENE", name)
		if demoScene() != name {
			t.Fatalf("%s", name)
		}
	}
	t.Setenv("RNPARMOR_DEMO_SCENE", "nope")
	if demoScene() != "" {
		t.Fatal("unknown")
	}
}

func TestDemoSceneEmptyHasNoSideEffect(t *testing.T) {
	t.Setenv("RNPARMOR_DEMO_SCENE", "")
	s := &uiState{tab: 4, workspace: t.TempDir(), langIdx: LangIndex(LangEN)}
	s.applyDemoScene()
	if s.tab != 4 || s.demoSceneApplied || s.demoOpenAbout || len(s.pendingToasts) != 0 || s.aboutWin != nil {
		t.Fatalf("empty scene changed state tab=%d applied=%v about=%v toasts=%d", s.tab, s.demoSceneApplied, s.demoOpenAbout, len(s.pendingToasts))
	}
	t.Setenv("RNPARMOR_DEMO_SCENE", "bogus")
	s.applyDemoScene()
	if s.tab != 4 || s.demoSceneApplied {
		t.Fatal("unknown scene")
	}
}

func demoKeys() []rnp.KeySummary {
	return []rnp.KeySummary{
		{Fingerprint: "BOBFP", PrimaryUID: "Bob Tanaka <bob@example.org>", IsPrimary: true, Subkeys: []rnp.KeySummary{{Fingerprint: "BOBSUB"}}},
		{Fingerprint: "ALICEFP", PrimaryUID: "Alice Chen <alice@example.com>", IsPrimary: true, HasSecret: true, Subkeys: []rnp.KeySummary{{Fingerprint: "AS1"}, {Fingerprint: "AS2"}}},
		{Fingerprint: "CAROLFP", PrimaryUID: "Carol Smith <carol@example.com>", IsPrimary: true, HasSecret: true, Subkeys: []rnp.KeySummary{{Fingerprint: "CS"}}},
	}
}

func TestDemoKeysScenePrefersAlice(t *testing.T) {
	t.Setenv("RNPARMOR_DEMO_SCENE", "keys")
	s := &uiState{
		tab:            2,
		keys:           demoKeys(),
		selected:       "BOBFP",
		selectedSubkey: "BOBSUB",
		collapsedKeys:  map[string]bool{"ALICEFP": true},
		langIdx:        LangIndex(LangEN),
	}
	s.applyDemoScene()
	if s.tab != 0 || s.selected != "ALICEFP" || s.selectedSubkey != "" || len(s.collapsedKeys) != 0 {
		t.Fatalf("keys tab=%d sel=%s sub=%s collapsed=%v", s.tab, s.selected, s.selectedSubkey, s.collapsedKeys)
	}
	if len(s.pendingToasts) != 0 || s.demoOpenAbout || s.aboutWin != nil {
		t.Fatal("keys scene opened extra UI")
	}
}

func TestDemoTextSceneToast(t *testing.T) {
	dir := t.TempDir()
	note := "RnpArmor portable OpenPGP demo message.\n"
	armor := "-----BEGIN PGP SIGNED MESSAGE-----\ndemo\n"
	if err := os.WriteFile(filepath.Join(dir, demoNoteName), []byte(note), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, demoSignedName), []byte(armor), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("RNPARMOR_DEMO_SCENE", "text")
	s := &uiState{workspace: dir, keys: demoKeys(), langIdx: LangIndex(LangEN), tab: 0}
	s.applyDemoScene()
	if s.tab != 1 || s.textInput != note || s.textOutput != armor {
		t.Fatalf("text tab=%d in=%q out=%q", s.tab, s.textInput, s.textOutput)
	}
	if s.encRecipients != "ALICEFP, BOBFP" || s.textSigner != "CAROLFP" {
		t.Fatalf("parties rec=%q signer=%q", s.encRecipients, s.textSigner)
	}
	if len(s.pendingToasts) != 1 || s.pendingToasts[0].Timeout != demoToastLifetime || s.pendingToasts[0].Timeout == toastLifetime {
		t.Fatalf("toast %+v", s.pendingToasts)
	}
	if s.pendingToasts[0].Title != "Text signed" || s.pendingToasts[0].Description != "Done" || s.pendingToasts[0].Type != "success" {
		t.Fatalf("toast copy %+v", s.pendingToasts[0])
	}
	if demoToastLifetime != 12*time.Second {
		t.Fatal("lifetime")
	}
	s.applyDemoScene()
	if len(s.pendingToasts) != 1 {
		t.Fatal("toast stacked")
	}
}

func TestDemoFileAndPacketScenes(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("RNPARMOR_DEMO_SCENE", "file")
	s := &uiState{workspace: dir, keys: demoKeys(), langIdx: LangIndex(LangZH)}
	s.applyDemoScene()
	if s.tab != 2 || s.fileSrc != filepath.Join(dir, demoNoteName) || s.fileDst != filepath.Join(dir, demoEncryptedName) || s.fileSig != filepath.Join(dir, demoSignedName) {
		t.Fatalf("file src=%s dst=%s sig=%s", s.fileSrc, s.fileDst, s.fileSig)
	}
	if s.encRecipients != "ALICEFP, BOBFP" || len(s.pendingToasts) != 0 {
		t.Fatalf("file recipients %q toasts %d", s.encRecipients, len(s.pendingToasts))
	}

	t.Setenv("RNPARMOR_DEMO_SCENE", "packet")
	p := &uiState{workspace: dir, packetAsJSON: false, featureQuery: "hash algorithm", langIdx: LangIndex(LangEN)}
	p.applyDemoScene()
	if p.tab != 3 || p.packetInput != filepath.Join(dir, demoSignedName) || !p.packetAsJSON || p.featureQuery != "" {
		t.Fatalf("packet tab=%d in=%q json=%v q=%q", p.tab, p.packetInput, p.packetAsJSON, p.featureQuery)
	}
}

func TestDemoAboutSetsFlagOnly(t *testing.T) {
	t.Setenv("RNPARMOR_DEMO_SCENE", "about")
	s := &uiState{tab: 0, langIdx: LangIndex(LangJA)}
	s.applyDemoScene()
	if !s.demoOpenAbout || s.aboutWin != nil || s.tab != 0 {
		t.Fatalf("about flag=%v win=%v tab=%d", s.demoOpenAbout, s.aboutWin != nil, s.tab)
	}
}

func TestFillDemoPacketsSkipsOtherScenes(t *testing.T) {
	t.Setenv("RNPARMOR_DEMO_SCENE", "keys")
	res := fillDemoPackets(nil, nil, t.TempDir())
	if res.hasPacketOut || res.hasFeatures || res.packetOut != "" {
		t.Fatalf("keys scene filled packets %+v", res)
	}
	t.Setenv("RNPARMOR_DEMO_SCENE", "")
	res = fillDemoPackets(nil, nil, t.TempDir())
	if res.hasPacketOut || res.hasFeatures {
		t.Fatal("empty scene filled packets")
	}
}
