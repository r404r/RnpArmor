// Command rnparmor is the minimal MyGo window for this slice.
// It shows whether librnp loaded and which version string the library
// reported. It does not generate keys, ask for a passphrase, or touch a
// keyring. File operations stay in internal/rnp and are started with
// Session.Go from a worker, not from this UI thread.
package main

import (
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/r404r/RnpArmor/internal/rnp"
)

type screen struct {
	path    string
	version string
	loadErr string
}

func main() {
	path, err := rnp.DefaultLibrary()
	sc := &screen{path: path}
	if err != nil {
		sc.loadErr = err.Error()
	} else if v, err := rnp.Probe(path); err != nil {
		sc.loadErr = err.Error()
	} else {
		sc.version = v
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "RnpArmor",
			Width:   560,
			Height:  300,
			Content: ui.View(sc.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func (s *screen) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Padding(24).Gap(10).Children(func() {
		ui.Text(c, "RnpArmor").FontSize(22).Bold()
		ui.Text(c, "最小垂直切片：只展示 librnp 是否加载。不生成密钥，不读默认钥匙圈。").FontSize(13).TextColor(t.TextMuted)
		if s.loadErr != "" {
			ui.Text(c, "未加载 librnp").FontSize(16)
			ui.Text(c, s.loadErr).FontSize(12)
		} else {
			ui.Textf(c, "librnp %s", s.version).FontSize(16)
		}
		if s.path != "" {
			ui.Text(c, s.path).FontSize(11).TextColor(t.TextMuted)
		}
		ui.Textf(c, "Go %s  %s/%s", runtime.Version(), runtime.GOOS, runtime.GOARCH).FontSize(11).TextColor(t.TextMuted)
	})
}

func init() {
	log.SetFlags(0)
}
