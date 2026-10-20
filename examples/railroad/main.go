// Railroad draws a regular expression as a railroad diagram as it is
// typed: start from an email address, an ISO 8601 date, a semantic
// version or a URL, or type a pattern of your own.
//
//	go run ./examples/railroad
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/railroad"
)

var presets = []struct {
	name, pattern string
}{
	{"Email address", `^(?P<user>[\w.+-]+)@(?P<domain>[\w-]+(?:\.[\w-]+)*\.[a-zA-Z]{2,})$`},
	{"ISO 8601 date", `^(?P<year>\d{4})-(?P<month>\d{2})-(?P<day>\d{2})(?:T(\d{2}):(\d{2})(?::(\d{2})(?:\.\d{1,9})?)?(Z|[+-]\d{2}:?\d{2})?)?$`},
	{"Semantic version", `^v?(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-([\w.-]+))?(?:\+([\w.-]+))?$`},
	{"URL", `^(https?)://([^/:?#\s]+)(?::(\d+))?(/[^?#\s]*)?(?:\?([^#\s]*))?(?:#(.*))?$`},
}

type app struct {
	pattern string
	diagram railroad.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Gap(t.Space(3)).Padding(t.Space(4)).Children(func() {
			ui.TextInput(c, &a.pattern).Font("monospace").Placeholder("Regular expression").Label("Pattern")
			ui.Row(c).Wrap().Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "Start from").FontSize(t.FontSize - 1).TextColor(t.TextMuted)
				for i, p := range presets {
					b := ui.Button(c.Key(i), p.name)
					if a.pattern == p.pattern {
						b.Border(1.5, t.Accent)
					}
					if b.Clicked() {
						a.pattern = p.pattern
					}
				}
			})
		})
		ui.Divider(c)
		railroad.View(c, &a.diagram, a.pattern).Grow(1).Background(t.Surface)
	})
}

func main() {
	a := &app{pattern: presets[0].pattern}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Railroad",
			Width:   1100,
			Height:  620,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
