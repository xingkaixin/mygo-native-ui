// Palette shows a command palette over a small editor: ⌘K (Ctrl+K on
// Linux and Windows) opens it, typing finds a command, and Enter runs it.
// The commands toggle the sidebar and word wrap, change the font size and
// the theme, and open the files, and their shortcuts, shown in the
// palette, run them too.
//
//	go run ./examples/palette
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/palette"
	"github.com/xingkaixin/mygo-native-ui/ui/shortcut"
)

// command is a command of the editor: what the palette shows, and what it
// does.
type command struct {
	palette.Item
	run func(e *editor)
}

// file is a file the editor opens.
type file struct {
	name, text string
}

// editor is the state the window shows: its files, the one open, how it
// shows it, the palette, and the last command run.
type editor struct {
	files    []file
	open     int
	sidebar  bool
	wrap     bool
	dark     bool
	fontSize float32
	palette  bool
	query    string
	last     string
}

// commands returns the editor's commands, the files last, to open each.
func (e *editor) commands() []command {
	key := func(mods ui.Modifiers, k ui.Key) shortcut.Shortcut { return shortcut.Shortcut{Mods: mods, Key: k} }
	cmds := []command{
		{palette.Item{Title: "Toggle Sidebar", Section: "View", Shortcut: key(ui.Cmd, ui.KeyB)}, func(e *editor) { e.sidebar = !e.sidebar }},
		{palette.Item{Title: "Toggle Word Wrap", Section: "View", Shortcut: key(ui.Alt, ui.KeyZ)}, func(e *editor) { e.wrap = !e.wrap }},
		{palette.Item{Title: "Increase Font Size", Section: "View", Shortcut: key(ui.Cmd, ui.KeyEqual)}, func(e *editor) { e.fontSize = min(e.fontSize+1, 24) }},
		{palette.Item{Title: "Decrease Font Size", Section: "View", Shortcut: key(ui.Cmd, ui.KeyMinus)}, func(e *editor) { e.fontSize = max(e.fontSize-1, 9) }},
		{palette.Item{Title: "Reset Font Size", Section: "View", Shortcut: key(ui.Cmd, ui.Key0)}, func(e *editor) { e.fontSize = 13 }},
		{palette.Item{Title: "Dark Theme", Section: "Preferences"}, func(e *editor) { e.dark = true }},
		{palette.Item{Title: "Light Theme", Section: "Preferences"}, func(e *editor) { e.dark = false }},
		{palette.Item{Title: "Next File", Section: "Go", Shortcut: key(ui.Cmd|ui.Alt, ui.KeyRight)}, func(e *editor) { e.open = (e.open + 1) % len(e.files) }},
		{palette.Item{Title: "Previous File", Section: "Go", Shortcut: key(ui.Cmd|ui.Alt, ui.KeyLeft)}, func(e *editor) {
			e.open = (e.open + len(e.files) - 1) % len(e.files)
		}},
	}
	for i, f := range e.files {
		cmds = append(cmds, command{palette.Item{Title: f.name, Section: "Open File"}, func(e *editor) { e.open = i }})
	}
	return cmds
}

func (e *editor) view(c *ui.Context) {
	if e.dark {
		c.SetTheme(ui.DarkTheme())
	} else {
		c.SetTheme(ui.LightTheme())
	}
	t := c.Theme()
	cmds := e.commands()
	items := make([]palette.Item, len(cmds))
	for i, cmd := range cmds {
		items[i] = cmd.Item
		if cmd.Shortcut.Pressed(c) {
			e.do(cmd)
		}
	}
	if c.Shortcut(ui.Cmd, ui.KeyK) {
		e.palette, e.query = true, ""
	}
	if i, ok := palette.Show(c, &e.palette, &e.query, items).Chosen(); ok {
		e.do(cmds[i])
	}

	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Grow(1).AlignItems(ui.Stretch).Children(func() {
			if e.sidebar {
				ui.Column(c).Width(180).Padding(8).Gap(2).Background(t.Surface).BorderEndWidth(1).BorderColor(t.Border).Children(func() {
					for i, f := range e.files {
						row := ui.Text(c.Key(i), f.name).Padding(4, 8).Radius(t.Radius)
						if i == e.open {
							row.Background(t.SurfacePressed)
						}
						if row.Clicked() {
							e.open = i
						}
					}
				})
			}
			// Scrolling sideways, the text has no width to wrap at.
			scroll := ui.ScrollBoth(c)
			if e.wrap {
				scroll = ui.Scroll(c)
			}
			scroll.Grow(1).Padding(16, 20).Children(func() {
				text := ui.Text(c, e.files[e.open].text).Font("monospace").FontSize(e.fontSize).LineHeight(1.5).Selectable()
				if !e.wrap {
					text.NoWrap()
				}
			})
		})
		ui.Divider(c)
		ui.Row(c).Padding(6, 12).Gap(8).AlignItems(ui.Center).Background(t.Surface).Children(func() {
			last := "No command run yet"
			if e.last != "" {
				last = "Ran " + e.last
			}
			ui.Text(c, last).FontSize(12).TextColor(t.TextMuted).Grow(1)
			ui.Text(c, "Commands").FontSize(12).TextColor(t.TextMuted)
			shortcut.Keys(c, shortcut.Shortcut{Mods: ui.Cmd, Key: ui.KeyK})
		})
	})
}

// do runs cmd and notes it.
func (e *editor) do(cmd command) {
	cmd.run(e)
	e.last = cmd.Title
}

func main() {
	e := &editor{files: files, sidebar: true, fontSize: 13}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Palette",
			Width:   900,
			Height:  600,
			Content: ui.View(e.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

var files = []file{
	{name: "main.go", text: `package main

import "fmt"

func main() {
    for i, name := range []string{"Ada", "Grace", "Linus"} {
        fmt.Printf("%d. Hello, %s! This line is long enough to show what word wrap does when it is on.\n", i+1, name)
    }
}`},
	{name: "README.md", text: `# Hello

Greets a few people by name.

    go run .`},
	{name: "go.mod", text: `module example.com/hello

go 1.24`},
}
