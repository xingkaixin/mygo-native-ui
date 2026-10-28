// Explorer is the sidebar of an editor over a small Go project: files in
// the colors of how git sees them, src folders folded onto one row, and
// what you do in it listed on the right. Rename with F2, make files and
// folders from the buttons or the context menu, and drag rows between
// folders.
//
//	go run ./examples/explorer
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/explorer"
)

type app struct {
	files  explorer.State
	paths  []string
	status map[string]explorer.Status
	open   string
	log    []string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Width(280).FillHeight().Background(t.Surface).Padding(t.Space(2)).Gap(t.Space(2)).Children(func() {
			ui.Row(c).Gap(t.Space(1)).AlignItems(ui.Center).Padding(0, t.Space(1)).Children(func() {
				ui.Text(c, "Explorer").Bold().Grow(1)
				if ui.Button(c, "New File").Clicked() {
					a.files.NewFile()
				}
				if ui.Button(c, "New Folder").Clicked() {
					a.files.NewFolder()
				}
			})
			ev := explorer.View(c, &a.files, &a.paths, a.status)
			ev.Grow(1)
			switch {
			case ev.Opened != "":
				a.open = ev.Opened
				a.log = append(a.log, "Opened "+ev.Opened)
			case ev.Created != "":
				a.log = append(a.log, "Created "+ev.Created)
			case ev.From != "":
				if a.open == ev.From {
					a.open = ev.To
				}
				if st, ok := a.status[ev.From]; ok {
					delete(a.status, ev.From)
					a.status[ev.To] = st
				}
				a.log = append(a.log, "Moved "+ev.From+" to "+ev.To)
			case ev.Deleted != "":
				a.log = append(a.log, "Deleted "+ev.Deleted)
			}
		})
		ui.Column(c).Grow(1).FillHeight().Padding(t.Space(6)).Gap(t.Space(3)).Children(func() {
			title := a.open
			if title == "" {
				title = "No file open"
			}
			ui.Text(c, title).FontSize(t.FontSize + 4).Bold()
			for i := len(a.log) - 1; i >= 0 && i >= len(a.log)-12; i-- {
				ui.Text(c.Key(i), a.log[i]).TextColor(t.TextMuted)
			}
		})
	})
}

func main() {
	a := &app{
		paths: []string{
			".gitignore", "README.md", "go.mod", "go.sum",
			"cmd/kettle/main.go",
			"internal/server/server.go", "internal/server/routes.go", "internal/server/routes_test.go",
			"internal/store/store.go", "internal/store/migrate.go",
			"pkg/api/v1/types.go", "pkg/api/v1/client.go",
			"node_modules/left-pad/index.js", "node_modules/left-pad/package.json",
			"docs/",
		},
		status: map[string]explorer.Status{
			"README.md":                      explorer.Modified,
			"internal/server/routes.go":      explorer.Modified,
			"internal/server/routes_test.go": explorer.Added,
			"internal/store/migrate.go":      explorer.Untracked,
			"pkg/api/v1/client.go":           explorer.Conflicted,
			"node_modules":                   explorer.Ignored,
		},
		open: "internal/server/routes.go",
	}
	a.files.Reveal(a.open)
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Explorer",
			Width:   860,
			Height:  600,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
