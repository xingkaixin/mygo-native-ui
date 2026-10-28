// Semver tries ranges against a package's versions: type a range, or
// pick one of those below the field, and watch which versions it takes,
// the bounds it means, and the prereleases it leaves out.
//
//	go run ./examples/semver
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/semver"
)

type app struct {
	query    string
	versions []semver.Version
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(8)).Gap(t.Space(4)).Children(func() {
		ui.Text(c, "left-pad").FontSize(t.FontSize + 8).Bold()
		ui.Row(c).Wrap().Gap(t.Space(2)).Children(func() {
			for _, r := range []string{"^1.2.0", "~1.2", ">=1.0.0 <2", "1.x || >=2.1", "1.0.0 - 1.3", "^1.3.0-alpha.0", "^0.9", "*"} {
				if ui.Button(c.Key(r), r).Clicked() {
					a.query = r
				}
			}
		})
		semver.View(c, &a.query, a.versions)
	})
}

func main() {
	a := &app{query: "^1.2.0"}
	for _, s := range []string{
		"0.9.0", "0.9.1", "0.10.0", "1.0.0-beta.1", "1.0.0-rc.1", "1.0.0", "1.0.1", "1.1.0", "1.2.0", "1.2.1", "1.2.4",
		"1.3.0-alpha.1", "1.3.0-alpha.2", "1.3.0", "1.4.0", "1.4.1", "2.0.0-beta.1", "2.0.0", "2.0.1", "2.1.0", "2.2.0", "3.0.0-rc.0",
	} {
		v, err := semver.ParseVersion(s)
		if err != nil {
			log.Fatal(err)
		}
		a.versions = append(a.versions, v)
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Semver",
			Width:   760,
			Height:  520,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
