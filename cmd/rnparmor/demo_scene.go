package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/r404r/RnpArmor/internal/rnp"
)

const (
	demoNoteName      = "demo-note.txt"
	demoSignedName    = "demo-signed.asc"
	demoEncryptedName = "demo-encrypted.pgp"
	demoNoteFallback  = "RnpArmor portable OpenPGP demo message."
	demoToastLifetime = 12 * time.Second
)

// demoScene is the screenshot hook. An empty or unknown value keeps the
// normal startup path, including a missing environment variable.
func demoScene() string {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("RNPARMOR_DEMO_SCENE")))
	switch v {
	case "keys", "text", "file", "packet", "about":
		return v
	default:
		return ""
	}
}

// applyDemoScene runs after a successful silent keyring load. It only
// mutates UI state. Opening the About window happens after the lock drops.
func (s *uiState) applyDemoScene() {
	scene := demoScene()
	if scene == "" || s.demoSceneApplied {
		return
	}
	s.demoSceneApplied = true
	switch scene {
	case "keys":
		s.applyDemoKeys()
	case "text":
		s.applyDemoText()
	case "file":
		s.applyDemoFile()
	case "packet":
		s.applyDemoPacket()
	case "about":
		s.demoOpenAbout = true
	}
}

func (s *uiState) applyDemoKeys() {
	k, ok := pickDemoPrimary(s.keys)
	if !ok {
		return
	}
	s.tab = 0
	s.selected = k.Fingerprint
	s.selectedSubkey = ""
	s.collapsedKeys = nil
	s.shownSelected = ""
	s.shownSubkey = ""
	s.keyFilter = 0
}

func (s *uiState) applyDemoText() {
	s.tab = 1
	s.textInput = demoPlainText(s.workspace)
	s.textOutput = readDemoFile(s.workspace, demoSignedName)
	alice, bob, carol := demoPartyPrints(s.keys)
	s.encRecipients = joinDemoRecipients(alice, bob)
	s.textSigner = carol
	s.enqueueToastFor(s.tr("toast.text_signed"), s.tr("toast.done"), "success", demoToastLifetime)
}

func (s *uiState) applyDemoFile() {
	s.tab = 2
	s.fileSrc = filepath.Join(s.workspace, demoNoteName)
	s.fileDst = filepath.Join(s.workspace, demoEncryptedName)
	s.fileSig = filepath.Join(s.workspace, demoSignedName)
	alice, bob, _ := demoPartyPrints(s.keys)
	s.encRecipients = joinDemoRecipients(alice, bob)
}

func (s *uiState) applyDemoPacket() {
	s.tab = 3
	s.packetInput = filepath.Join(s.workspace, demoSignedName)
	s.packetAsJSON = true
	s.featureQuery = ""
}

func pickDemoPrimary(keys []rnp.KeySummary) (rnp.KeySummary, bool) {
	var fallback *rnp.KeySummary
	for i := range keys {
		k := &keys[i]
		if !isPrimaryRow(*k) || len(k.Subkeys) == 0 {
			continue
		}
		if strings.Contains(k.PrimaryUID, "Alice Chen") {
			return *k, true
		}
		if fallback == nil {
			fallback = k
		}
	}
	if fallback != nil {
		return *fallback, true
	}
	return rnp.KeySummary{}, false
}

func demoPartyPrints(keys []rnp.KeySummary) (alice, bob, carol string) {
	for _, k := range keys {
		if !isPrimaryRow(k) {
			continue
		}
		fp := stripID(k.Fingerprint)
		switch {
		case strings.Contains(k.PrimaryUID, "Alice Chen"):
			alice = fp
		case strings.Contains(k.PrimaryUID, "Bob Tanaka"):
			bob = fp
		case strings.Contains(k.PrimaryUID, "Carol Smith"):
			carol = fp
		}
	}
	return alice, bob, carol
}

func joinDemoRecipients(alice, bob string) string {
	switch {
	case alice != "" && bob != "":
		return alice + ", " + bob
	case alice != "":
		return alice
	default:
		return bob
	}
}

func demoPlainText(workspace string) string {
	b, err := os.ReadFile(filepath.Join(workspace, demoNoteName))
	if err != nil || strings.TrimSpace(string(b)) == "" {
		return demoNoteFallback
	}
	return string(b)
}

func readDemoFile(workspace, name string) string {
	b, err := os.ReadFile(filepath.Join(workspace, name))
	if err != nil {
		return err.Error()
	}
	return string(b)
}

// fillDemoPackets asks librnp to parse the seeded signature while the
// session is still open. A parse error stays in the output box and does
// not fail the keyring load. Other scenes leave the result empty.
func fillDemoPackets(sess *rnp.Session, ctx context.Context, ws string) actionResult {
	if demoScene() != "packet" {
		return actionResult{}
	}
	res := actionResult{hasPacketOut: true, hasFeatures: true}
	b, err := os.ReadFile(filepath.Join(ws, demoSignedName))
	if err != nil {
		res.packetOut = err.Error()
		res.features = err.Error()
		return res
	}
	out, err := sess.DumpPackets(ctx, b, true)
	if err != nil {
		res.packetOut = err.Error()
	} else {
		res.packetOut = out
	}
	feat, ferr := sess.SupportedFeatures(ctx, "")
	if ferr != nil {
		res.features = ferr.Error()
	} else {
		res.features = feat
	}
	return res
}
