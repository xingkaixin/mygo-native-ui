// Dock lays out an IDE's panels: Explorer and Outline on the left, three
// files in the editor, and Terminal and Problems under it. Dragging a tab
// moves its panel to another group, to a group's edge to split it, or to
// an edge of the window to dock it across that side; the dividers drag to
// resize, and a tab's close button or the middle button closes it.
//
//	go run ./examples/dock
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/dock"
)

// panel is what a tab of the dock shows: its title and its content.
type panel struct {
	title string
	build func(c *ui.Context)
}

// app is the state the window shows: the layout of the panels, which the
// app could save, and the dock's state.
type app struct {
	layout *dock.Layout
	dock   dock.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		dock.View(c, &a.dock, a.layout, func(id string) string {
			return panels[id].title
		}, func(id string) {
			panels[id].build(c)
		}).Grow(1)
		ui.Divider(c)
		ui.Row(c).Padding(t.Space(1), t.Space(3)).Gap(t.Space(3)).AlignItems(ui.Center).Background(t.Surface).Children(func() {
			status := "No panels open"
			if id := a.dock.Active(); id != "" {
				status = panels[id].title
			}
			ui.Text(c, status).TextColor(t.TextMuted).Grow(1)
			if ui.Button(c, "Reset Layout").Clicked() {
				a.layout = initialLayout()
			}
		})
	})
}

// initialLayout puts Explorer and Outline at the left of the editor's
// files, and Terminal and Problems under them.
func initialLayout() *dock.Layout {
	return &dock.Layout{
		Percent: 22,
		First: &dock.Layout{
			Vertical: true,
			Percent:  62,
			First:    &dock.Layout{Panels: []string{"explorer"}},
			Second:   &dock.Layout{Panels: []string{"outline"}},
		},
		Second: &dock.Layout{
			Vertical: true,
			Percent:  68,
			First:    &dock.Layout{Panels: []string{"main.go", "server.go", "README.md"}},
			Second:   &dock.Layout{Panels: []string{"terminal", "problems"}},
		},
	}
}

func main() {
	a := &app{layout: initialLayout()}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Dock",
			Width:   1180,
			Height:  760,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

var panels = map[string]panel{
	"explorer":  {"Explorer", explorer},
	"outline":   {"Outline", outline},
	"main.go":   {"main.go", source(mainGo)},
	"server.go": {"server.go", source(serverGo)},
	"README.md": {"README.md", source(readme)},
	"terminal":  {"Terminal", terminal},
	"problems":  {"Problems", problems},
}

func explorer(c *ui.Context) {
	t := c.Theme()
	files := []struct {
		depth int
		name  string
		dir   bool
	}{
		{0, "notes", true},
		{1, "cmd", true},
		{2, "main.go", false},
		{1, "internal", true},
		{2, "server.go", false},
		{2, "store.go", false},
		{2, "store_test.go", false},
		{1, "README.md", false},
		{1, "go.mod", false},
	}
	ui.Scroll(c).Grow(1).Padding(t.Space(2), 0).Children(func() {
		for _, f := range files {
			ui.Row(c).Padding(t.Space(1), t.Space(3)).PaddingStart(t.Space(3) + float32(f.depth)*t.Space(4)).Children(func() {
				if f.dir {
					ui.Text(c, "▾ "+f.name).Bold().SingleLine()
				} else {
					ui.Text(c, f.name).SingleLine().PaddingStart(t.Space(3.5))
				}
			})
		}
	})
}

func outline(c *ui.Context) {
	t := c.Theme()
	symbols := [][2]string{
		{"func", "main()"},
		{"type", "config"},
		{"func", "loadConfig(path string)"},
		{"type", "server"},
		{"func", "newServer(cfg config)"},
		{"func", "(*server) ServeHTTP"},
	}
	ui.Scroll(c).Grow(1).Padding(t.Space(2), 0).Children(func() {
		for _, s := range symbols {
			ui.Row(c).Padding(t.Space(1), t.Space(3)).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, s[0]).Width(t.Space(10)).TextColor(t.Accent).Font("monospace").SingleLine()
				ui.Text(c, s[1]).Font("monospace").SingleLine().Shrink(1)
			})
		}
	})
}

// source shows a file's text, as an editor would.
func source(text string) func(c *ui.Context) {
	return func(c *ui.Context) {
		t := c.Theme()
		ui.Scroll(c).Grow(1).Padding(t.Space(3), t.Space(5)).Children(func() {
			ui.Text(c, text).Font("monospace").LineHeight(1.5).Selectable()
		})
	}
}

func terminal(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Grow(1).Padding(t.Space(2), t.Space(4)).Children(func() {
		ui.Text(c, `$ go test ./...
?       example.com/notes/cmd           [no test files]
ok      example.com/notes/internal      0.412s
$ go run ./cmd
2026/10/20 09:41:07 listening on :8080
2026/10/20 09:41:12 GET /notes 182µs`).Font("monospace").LineHeight(1.5).Selectable()
	})
}

func problems(c *ui.Context) {
	t := c.Theme()
	items := []struct {
		err       bool
		text, loc string
	}{
		{true, "undefined: store.remove", "server.go:28:5"},
		{false, "error strings should not be capitalized", "main.go:12:20"},
		{false, "exported func NewServer returns unexported type", "server.go:11:6"},
	}
	ui.Scroll(c).Grow(1).Padding(t.Space(2), 0).Children(func() {
		for _, p := range items {
			color := t.Warning
			if p.err {
				color = t.Danger
			}
			ui.Row(c).Padding(t.Space(1), t.Space(4)).Gap(t.Space(2.5)).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(t.Space(2), t.Space(2)).Radius(t.Space(1)).Background(color)
				ui.Text(c, p.text).SingleLine().Shrink(1)
				ui.Text(c, p.loc).TextColor(t.TextMuted).SingleLine()
			})
		}
	})
}

const mainGo = `package main

import (
    "log"
    "net/http"

    "example.com/notes/internal"
)

func main() {
    cfg, err := loadConfig("config.yaml")
    if err != nil {
        log.Fatal("Loading config: ", err)
    }
    srv := internal.NewServer(cfg.Addr)
    log.Printf("listening on %s", cfg.Addr)
    log.Fatal(http.ListenAndServe(cfg.Addr, srv))
}`

const serverGo = `package internal

import "net/http"

type server struct {
    store *store
    mux   *http.ServeMux
}

func NewServer(addr string) *server {
    s := &server{store: newStore(), mux: http.NewServeMux()}
    s.mux.HandleFunc("GET /notes", s.listNotes)
    s.mux.HandleFunc("DELETE /notes/{id}", s.deleteNote)
    return s
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    s.mux.ServeHTTP(w, r)
}`

const readme = `# Notes

A small server that keeps notes in memory.

    go run ./cmd

Then list them with curl localhost:8080/notes.`
