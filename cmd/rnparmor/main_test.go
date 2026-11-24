package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/egoist/mygo"
)

func TestValidateCatalogIncludesMenuAndAboutKeys(t *testing.T) {
	if err := ValidateCatalog(); err != nil {
		t.Fatalf("ValidateCatalog() error = %v", err)
	}

	keys := []string{
		"menu.file",
		"menu.workspace",
		"menu.language",
		"menu.help",
		"menu.help.about",
		"btn.close",
		"about.no_menu_hint",
	}
	langs := []Lang{LangEN, LangZH, LangJA}
	for _, k := range keys {
		for _, l := range langs {
			v := strings.TrimSpace(Tr(l, k))
			if v == "" || v == k {
				t.Fatalf("translation missing or fallback key for %s/%s: %q", k, l, v)
			}
		}
	}
}

func TestMenuTemplateLayoutAndRadioLanguage(t *testing.T) {
	s := &uiState{langIdx: LangIndex(LangEN)}
	items := s.menuTemplate()
	if got, want := len(items), 4; got != want {
		t.Fatalf("top menu count = %d, want %d", got, want)
	}

	if got := items[0].Label; got != Tr(LangEN, "menu.file") {
		t.Fatalf("menu[0] label = %q", got)
	}
	if got := len(items[0].Submenu); got != 5 {
		t.Fatalf("file submenu count = %d, want 5", got)
	}
	if items[0].Submenu[0].Label != Tr(LangEN, "btn.open_session") {
		t.Fatalf("file[0] label = %q", items[0].Submenu[0].Label)
	}
	if items[0].Submenu[1].Label != Tr(LangEN, "btn.load_keyring") {
		t.Fatalf("file[1] label = %q", items[0].Submenu[1].Label)
	}
	if items[0].Submenu[2].Label != Tr(LangEN, "btn.save_keyring") {
		t.Fatalf("file[2] label = %q", items[0].Submenu[2].Label)
	}
	if items[0].Submenu[3].Type != mygo.MenuItemSeparator {
		t.Fatalf("file[3] type = %q, want separator", items[0].Submenu[3].Type)
	}
	if items[0].Submenu[4].Label != Tr(LangEN, "menu.file.quit") {
		t.Fatalf("file[4] label = %q", items[0].Submenu[4].Label)
	}

	if got := len(items[1].Submenu); got != 5 {
		t.Fatalf("workspace submenu count = %d, want 5", got)
	}
	for i, it := range items[1].Submenu {
		if it.Type != mygo.MenuItemRadio {
			t.Fatalf("workspace submenu item %d type = %q, want radio", i, it.Type)
		}
		wantChecked := i == 0
		if it.Checked != wantChecked {
			t.Fatalf("workspace submenu item %d checked = %v, want %v at default tab 0", i, it.Checked, wantChecked)
		}
	}

	if got := len(items[2].Submenu); got != 3 {
		t.Fatalf("language submenu count = %d, want 3", got)
	}
	for i, it := range items[2].Submenu {
		if it.Type != mygo.MenuItemRadio {
			t.Fatalf("language submenu item %d type = %q, want radio", i, it.Type)
		}
		wantChecked := i == LangIndex(LangEN)
		if it.Checked != wantChecked {
			t.Fatalf("language submenu item %d checked = %v, want %v", i, it.Checked, wantChecked)
		}
	}
	if got := len(items[3].Submenu); got != 1 {
		t.Fatalf("help submenu count = %d, want 1", got)
	}

	s.tab = 3
	s.langIdx = LangIndex(LangJA)
	items = s.menuTemplate()
	for i, it := range items[1].Submenu {
		wantChecked := i == 3
		if it.Checked != wantChecked {
			t.Fatalf("workspace submenu item %d checked = %v, want %v when tab=3", i, it.Checked, wantChecked)
		}
	}
	for i, it := range items[2].Submenu {
		wantChecked := i == LangIndex(LangJA)
		if it.Checked != wantChecked {
			t.Fatalf("language submenu item %d checked = %v, want %v when lang=ja", i, it.Checked, wantChecked)
		}
	}
}

func TestBuildSmokeOutputAndWriteSmokeOutput(t *testing.T) {
	got := buildSmokeOutput("v1.2.3", `C:\\RnpArmor\\rnp.dll`, "0.18.1")
	for _, wantLine := range []string{
		"status=ok",
		"app_version=v1.2.3",
		`library=C:\\RnpArmor\\rnp.dll`,
		"version=0.18.1",
		"i18n=ok",
	} {
		if !strings.Contains(got, wantLine+"\n") {
			t.Fatalf("smoke output missing line %q in %q", wantLine, got)
		}
	}

	if err := writeSmokeOutput("", got); err != nil {
		t.Fatalf("writeSmokeOutput(empty) error = %v", err)
	}

	root := t.TempDir()
	out := filepath.Join(root, "nested", "smoke", "smoke-out.txt")
	if err := writeSmokeOutput(out, got); err != nil {
		t.Fatalf("writeSmokeOutput(%q) error = %v", out, err)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", out, err)
	}
	if string(b) != got {
		t.Fatalf("smoke file content mismatch: got %q want %q", string(b), got)
	}
}
