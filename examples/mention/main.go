// Mention is the prompt field of a coding agent: typing @ offers the
// files and folders under the current directory and the people on the
// team, and # the open issues. Below the field, the text it holds and the
// mentions in it follow each edit, and Enter sends the prompt, which
// shows under them.
//
//	go run ./examples/mention
package main

import (
	"fmt"
	"io/fs"
	"log"
	"path/filepath"
	"slices"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/mention"
)

// most is as many files and folders as the field offers.
const most = 2000

var people = []mention.Item{
	{Label: "maya", Detail: "Maya Chen", Kind: mention.Person},
	{Label: "omar", Detail: "Omar Haddad", Kind: mention.Person},
	{Label: "lena", Detail: "Lena Fischer", Kind: mention.Person},
	{Label: "sam", Detail: "Sam Okafor", Kind: mention.Person},
}

var issues = []mention.Item{
	{Label: "142", Detail: "Crash when pasting an image into the prompt", Kind: mention.Issue},
	{Label: "157", Detail: "Mentions lose their tint in dark mode", Kind: mention.Issue},
	{Label: "163", Detail: "Tab moves the focus while suggestions show", Kind: mention.Issue},
	{Label: "171", Detail: "Long paths overflow the suggestion list", Kind: mention.Issue},
	{Label: "188", Detail: "Mention symbols as well as files", Kind: mention.Issue},
}

var kinds = []string{"File", "Folder", "Symbol", "Person", "Issue"}

// message is a prompt sent: its text and the mentions in it.
type message struct {
	text     string
	mentions []mention.Mention
}

type app struct {
	prompt mention.State
	// at are what @ offers: the people, then the folders and files.
	at   []mention.Item
	sent []message
}

// files returns the folders and files under the current directory, with
// slashes, hidden ones left out, up to most of them.
func files() []mention.Item {
	var out []mention.Item
	filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || path == "." {
			return nil
		}
		if strings.HasPrefix(d.Name(), ".") || len(out) == most {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		kind := mention.File
		if d.IsDir() {
			kind = mention.Folder
		}
		out = append(out, mention.Item{Label: filepath.ToSlash(path), Kind: kind})
		return nil
	})
	return out
}

func (a *app) find(trigger rune, query string) []mention.Item {
	if trigger == '#' {
		return issues
	}
	return a.at
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(8)).Gap(t.Space(6)).Children(func() {
		ui.Column(c).Gap(t.Space(1)).Children(func() {
			ui.Text(c, "Ask about this code").Bold().FontSize(t.FontSize + 4)
			ui.Text(c, "Type @ for a file, a folder or a teammate, and # for an issue. Enter sends, Shift+Enter breaks the line.").
				TextColor(t.TextMuted)
		})
		mention.View(c, &a.prompt, "Ask anything", "@#", a.find)
		if a.prompt.Submitted() {
			a.sent = append(a.sent, message{a.prompt.Text, a.prompt.Mentions})
			a.prompt.Text, a.prompt.Mentions = "", nil
		}
		ui.Row(c).Grow(1).Gap(t.Space(6)).AlignItems(ui.Stretch).Children(func() {
			section(c, "Text", func() {
				text := a.prompt.Text
				color := t.Text
				if text == "" {
					text, color = "Nothing typed yet", t.TextMuted
				}
				ui.Text(c, text).Font("monospace").TextColor(color).Selectable()
			})
			section(c, fmt.Sprintf("Mentions (%d)", len(a.prompt.Mentions)), func() {
				if len(a.prompt.Mentions) == 0 {
					ui.Text(c, "None yet").TextColor(t.TextMuted)
				}
				for i, m := range a.prompt.Mentions {
					mentionRow(c.Key(i), m)
				}
			})
			section(c, "Sent", func() {
				if len(a.sent) == 0 {
					ui.Text(c, "Nothing sent yet").TextColor(t.TextMuted)
				}
				for i := len(a.sent) - 1; i >= 0; i-- {
					m := a.sent[i]
					ui.Text(c.Key(i), fmt.Sprintf("%s\n%d mentions", strings.TrimSpace(m.text), len(m.mentions))).
						FontSize(t.FontSize - 1)
				}
			})
		})
	})
}

// section shows title over a scrolling box of what fill builds.
func section(c *ui.Context, title string, fill func()) {
	t := c.Theme()
	ui.Column(c).Grow(1).Basis(0).Gap(t.Space(2)).Children(func() {
		ui.Text(c, title).Bold()
		ui.Scroll(c).Grow(1).Padding(t.Space(3)).Radius(t.Radius).Border(1, t.Border).Children(func() {
			ui.Column(c).Gap(t.Space(2.5)).Children(fill)
		})
	})
}

// mentionRow shows m as the text holds it, its kind and its runes.
func mentionRow(c *ui.Context, m mention.Mention) {
	t := c.Theme()
	ui.Column(c).Gap(t.Space(0.5)).Children(func() {
		ui.Text(c, m.String()).Font("monospace").TextColor(t.Accent)
		ui.Text(c, fmt.Sprintf("%s · runes %d to %d", kinds[m.Item.Kind], m.Start, m.End)).
			FontSize(t.FontSize - 1).TextColor(t.TextMuted)
	})
}

func main() {
	a := &app{at: slices.Concat(people, files())}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Mention",
			Width:   960,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
