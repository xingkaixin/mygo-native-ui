// Package screenplay shows a screenplay set as scripts are typed: in a
// monospaced face on a page, scene headings in capitals with their
// numbers in both margins, character cues and dialogue indented to their
// columns, parentheticals between them, and transitions to the right. A
// title page opens it when the script has one. Beside the page, its
// scenes, a click on one scrolling to it, and its characters, the most
// lines first.
//
//	script := screenplay.Parse(fountain)
//	screenplay.View(c, &app.reader, script)
package screenplay

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"
)

// State is what a reader keeps: the scene to scroll to. Keep it in the
// app's state, one for each reader.
type State struct {
	jump int
}

// View creates the reader of sc.
func View(c *ui.Context, s *State, sc Script) ui.Element {
	t := c.Theme()
	size := t.FontSize + 1
	// A monospaced face is six tenths of an em wide, so a page's sixty
	// columns are as wide as the six inches of a printed one.
	ch := size * 0.6
	line := size * 1.3
	root := ui.Row(c).Fill().AlignItems(ui.Stretch).Role(ui.RoleGroup).Label("Screenplay")
	root.Children(func() {
		ui.Scroll(c).Width(t.Space(64)).Background(t.Surface).BorderWidth(0, 1, 0, 0).BorderColor(t.Border).Children(func() {
			ui.Column(c).Padding(t.Space(3)).Gap(t.Space(1)).Children(func() {
				ui.Text(c, "Scenes").FontSize(t.FontSize - 1).FontWeight(600).TextColor(t.TextMuted).MarginY(t.Space(1))
				for _, l := range sc.Lines {
					if l.Kind != Heading {
						continue
					}
					row := ui.Row(c.Key(l.Scene)).Gap(t.Space(2)).Padding(t.Space(1.5)).Radius(t.Radius).Cursor(ui.CursorPointer)
					row.Children(func() {
						ui.Text(c, fmt.Sprint(l.Scene)).Width(t.Space(5)).TextColor(t.TextMuted).FontFeatures("tnum")
						ui.Text(c, l.Text).Grow(1).Basis(0).FontSize(t.FontSize - 1).MaxLines(2)
					})
					if row.Hovered() {
						row.Background(t.Border.Alpha(0.5))
					}
					if row.Clicked() {
						s.jump = l.Scene
					}
				}
				ui.Text(c, "Characters").FontSize(t.FontSize - 1).FontWeight(600).TextColor(t.TextMuted).MarginY(t.Space(1))
				for _, who := range cast(sc) {
					ui.Row(c.Key("who" + who.name)).Gap(t.Space(2)).Padding(t.Space(1.5)).Children(func() {
						ui.Text(c, who.name).Grow(1).FontSize(t.FontSize - 1)
						ui.Text(c, fmt.Sprint(who.lines)).FontSize(t.FontSize - 1).TextColor(t.TextMuted).FontFeatures("tnum")
					})
				}
			})
		})
		ui.Scroll(c).Grow(1).Background(t.Surface).Children(func() {
			page := ui.Column(c).Width(ch*76).AlignSelf(ui.Center).MarginY(t.Space(6)).Background(t.Background).
				Border(1, t.Border).Shadow(0, 2, 12, 0, ui.RGBA(0, 0, 0, 0.08)).PaddingY(line * 3)
			page.Children(func() {
				text := func(key any, s string) ui.Element {
					return ui.Text(c.Key(key), s).Font("monospace").FontSize(size).LineHeight(1.3)
				}
				if sc.Title != "" {
					ui.Column(c).AlignItems(ui.Center).Gap(line).MarginY(line * 4).Children(func() {
						text("title", strings.ToUpper(sc.Title)).FontWeight(700)
						if sc.Credit != "" || sc.Author != "" {
							text("credit", cmp.Or(sc.Credit, "Written by"))
							text("author", sc.Author)
						}
					})
				}
				for i, l := range sc.Lines {
					gap := line
					if i > 0 && (l.Kind == Dialogue || l.Kind == Parenthetical) {
						gap = 0
					}
					switch l.Kind {
					case Heading:
						row := ui.Row(c.Key(i)).Margin(line*2, 0, 0, 0).Children(func() {
							num := fmt.Sprint(l.Scene)
							text("n", num).Width(ch * 8).PaddingStart(ch * 2).TextColor(t.TextMuted)
							text("h", l.Text).Width(ch * 60).FontWeight(700)
							text("m", num).Width(ch * 8).TextAlign(ui.End).PaddingEnd(ch * 2).TextColor(t.TextMuted)
						})
						if s.jump == l.Scene {
							row.ScrollIntoView()
							s.jump = 0
						}
					case Action:
						text(i, l.Text).Width(ch*60).Margin(gap, 0, 0, ch*8)
					case Character:
						text(i, strings.ToUpper(l.Text)).Margin(gap, 0, 0, ch*30)
					case Parenthetical:
						text(i, l.Text).Width(ch*25).Margin(gap, 0, 0, ch*24)
					case Dialogue:
						text(i, l.Text).Width(ch*35).Margin(gap, 0, 0, ch*18)
					case Transition:
						text(i, strings.ToUpper(l.Text)).Width(ch*60).TextAlign(ui.End).Margin(gap, 0, 0, ch*8)
					case Centered:
						text(i, l.Text).Width(ch*60).TextAlign(ui.Center).Margin(gap, 0, 0, ch*8)
					}
				}
			})
		})
	})
	return root
}

type speaker struct {
	name  string
	lines int
}

// cast returns who speaks in sc and how many times, the most first.
func cast(sc Script) []speaker {
	counts := map[string]int{}
	for _, l := range sc.Lines {
		if l.Kind != Character {
			continue
		}
		name := l.Text
		if i := strings.Index(name, "("); i > 0 {
			name = name[:i]
		}
		counts[strings.ToUpper(strings.TrimSpace(strings.TrimSuffix(name, "^")))]++
	}
	var out []speaker
	for n, k := range counts {
		out = append(out, speaker{n, k})
	}
	slices.SortFunc(out, func(a, b speaker) int {
		return cmp.Or(cmp.Compare(b.lines, a.lines), strings.Compare(a.name, b.name))
	})
	return out
}
