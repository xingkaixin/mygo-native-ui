// Circlepack shows where the bytes of a web app's folder go, in KB: click
// node_modules to zoom into the packages, a package to see its files, and
// the circle in view or Escape to come back out.
//
//	go run ./examples/circlepack
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/circlepack"
)

func leaf(name string, kb float64) circlepack.Node { return circlepack.Node{Name: name, Value: kb} }

func dir(name string, kids ...circlepack.Node) circlepack.Node {
	return circlepack.Node{Name: name, Kids: kids}
}

var tree = dir("kettle-web",
	dir("src",
		dir("components", leaf("Button.tsx", 6), leaf("Dialog.tsx", 14), leaf("Table.tsx", 31), leaf("Chart.tsx", 22), leaf("Sidebar.tsx", 11)),
		dir("pages", leaf("Home.tsx", 18), leaf("Billing.tsx", 27), leaf("Settings.tsx", 24), leaf("Team.tsx", 16)),
		dir("lib", leaf("api.ts", 12), leaf("auth.ts", 9), leaf("format.ts", 5)),
		leaf("main.tsx", 3),
	),
	dir("public",
		dir("images", leaf("hero.png", 840), leaf("og.png", 310), leaf("team.jpg", 620)),
		dir("fonts", leaf("Inter.woff2", 290), leaf("Mono.woff2", 180)),
		leaf("favicon.ico", 15),
	),
	dir("node_modules",
		dir("react-dom", leaf("client.js", 1320), leaf("server.js", 980), leaf("profiling.js", 1410)),
		dir("typescript", leaf("tsc.js", 5900), leaf("typescript.js", 8800), leaf("lib.d.ts", 1700)),
		dir("date-fns", leaf("locale", 2100), leaf("esm", 1600), leaf("fp", 900)),
		dir("lodash", leaf("lodash.js", 540), leaf("fp", 620), leaf("modules", 1400)),
		dir("vite", leaf("dist", 3100), leaf("client", 210)),
		dir("esbuild", leaf("bin", 9400)),
		leaf("react", 260),
		leaf("zod", 610),
		leaf("clsx", 8),
	),
	dir("dist", leaf("index.js", 410), leaf("vendor.js", 1250), leaf("index.css", 90)),
)

type app struct {
	pack circlepack.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(4)).Children(func() {
		ui.Text(c, "Disk usage").FontSize(t.FontSize + 8).Bold()
		circlepack.View(c, &a.pack, tree).Grow(1).Background(t.Surface)
	})
}

func main() {
	a := &app{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Circlepack",
			Width:   820,
			Height:  760,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
