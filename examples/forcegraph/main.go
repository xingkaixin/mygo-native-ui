// Forcegraph shows the imports of Go's standard library, from go list,
// without its internal and vendored packages: a node for each package,
// colored by the first element of its path, linked to those it imports.
// Search for a package, or click one, to list what it imports and what
// imports it.
//
//	go run ./examples/forcegraph
package main

import (
	"log"
	"os/exec"
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/forcegraph"
)

// hidden reports whether path is a package of the standard library's
// own: internal, or vendored.
func hidden(path string) bool {
	return strings.HasPrefix(path, "vendor/") || slices.Contains(strings.Split(path, "/"), "internal")
}

// load lists the packages of the standard library and their imports, as
// nodes, the first element of a path its group, and links from each
// package to those it imports.
func load() ([]forcegraph.Node, []forcegraph.Link, error) {
	out, err := exec.Command("go", "list", "-f", "{{.ImportPath}}{{range .Imports}} {{.}}{{end}}", "std").Output()
	if err != nil {
		return nil, nil, err
	}
	var nodes []forcegraph.Node
	var imports [][]string
	at := map[string]int{}
	for line := range strings.Lines(string(out)) {
		f := strings.Fields(line)
		if len(f) == 0 || hidden(f[0]) {
			continue
		}
		group, _, _ := strings.Cut(f[0], "/")
		at[f[0]] = len(nodes)
		nodes = append(nodes, forcegraph.Node{ID: f[0], Label: f[0], Group: group})
		imports = append(imports, f[1:])
	}
	var links []forcegraph.Link
	for i, imps := range imports {
		for _, p := range imps {
			if j, ok := at[p]; ok {
				links = append(links, forcegraph.Link{From: i, To: j})
			}
		}
	}
	return nodes, links, nil
}

type app struct {
	nodes  []forcegraph.Node
	links  []forcegraph.Link
	paths  []string
	graph  forcegraph.State
	chosen string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Gap(t.Space(3)).Padding(t.Space(3), t.Space(4)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Standard library imports").Bold()
			ui.Text(c, strconv.Itoa(len(a.nodes))+" packages · "+strconv.Itoa(len(a.links))+" imports").TextColor(t.TextMuted).Grow(1)
			ui.Text(c, "Package").TextColor(t.TextMuted)
			ui.Combobox(c, &a.chosen, a.paths).Label("Find a package").Width(t.Space(64))
		})
		ui.Divider(c)
		ui.Row(c).Grow(1).Children(func() {
			forcegraph.View(c, &a.graph, a.nodes, a.links, &a.chosen).Grow(1).AlignSelf(ui.Stretch).Background(t.Surface)
			ui.Divider(c)
			a.side(c)
		})
	})
}

// side lists what the package chosen imports and what imports it, each a
// button choosing that package.
func (a *app) side(c *ui.Context) {
	t := c.Theme()
	i := slices.Index(a.paths, a.chosen)
	ui.Scroll(c).Width(t.Space(68)).AlignSelf(ui.Stretch).Children(func() {
		ui.Column(c).Padding(t.Space(4)).Gap(t.Space(4)).Children(func() {
			if i < 0 {
				ui.Text(c, "Choose a package to list what it imports and what imports it.").TextColor(t.TextMuted)
				return
			}
			var imports, importers []string
			for _, l := range a.links {
				if l.From == i {
					imports = append(imports, a.paths[l.To])
				}
				if l.To == i {
					importers = append(importers, a.paths[l.From])
				}
			}
			slices.Sort(importers)
			ui.Text(c, a.chosen).Bold().FontSize(t.FontSize + 3)
			a.list(c.Key("imports"), "Imports", imports)
			a.list(c.Key("importers"), "Imported by", importers)
		})
	})
}

// list shows title with how many paths there are, and the paths.
func (a *app) list(c *ui.Context, title string, paths []string) {
	t := c.Theme()
	ui.Column(c).Gap(t.Space(1)).Children(func() {
		ui.Row(c).Gap(t.Space(1.5)).Padding(0, 0, t.Space(1)).Children(func() {
			ui.Text(c, title).Bold().FontSize(t.FontSize - 1)
			ui.Text(c, strconv.Itoa(len(paths))).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
		})
		if len(paths) == 0 {
			ui.Text(c, "None").TextColor(t.TextMuted)
		}
		for _, p := range paths {
			row := ui.ButtonBase(c.Key(p)).Justify(ui.Start).Padding(t.Space(1), t.Space(2)).Margin(0, -t.Space(2)).Radius(t.Space(1.5)).Label(p)
			if row.Hovered() {
				row.Background(t.SurfaceHover)
			}
			if row.Clicked() {
				a.chosen = p
			}
			row.Children(func() { ui.Text(c, p).SingleLine() })
		}
	})
}

func main() {
	nodes, links, err := load()
	if err != nil {
		log.Fatal(err)
	}
	a := &app{nodes: nodes, links: links}
	for _, n := range nodes {
		a.paths = append(a.paths, n.ID)
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Imports",
			Width:   1280,
			Height:  820,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
