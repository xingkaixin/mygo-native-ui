// Package iconpicker picks an icon from a set by searching, as design
// tools and app builders do: a search field over a grid of the icons,
// filtered to those whose name or keywords have every word typed, and the
// icon chosen large beside its name under it. Pointing at an icon shows
// its name; a click chooses it. The arrow keys move a ring through the
// grid from the search field, and Enter chooses the icon in it.
//
//	iconpicker.View(c, &app.picker, app.icons, &app.icon)
package iconpicker

import (
	"fmt"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Icon is an icon, its name and the words it is also found by.
type Icon struct {
	Name     string
	Keywords []string
	Image    *ui.SVG
}

// State is what a picker keeps: the search, the icon in the ring and how
// many fit across. Keep it in the app's state, one for each picker.
type State struct {
	query  string
	ring   int
	across int
}

// View creates the picker of icons, *chosen the name of the one chosen.
func View(c *ui.Context, s *State, icons []Icon, chosen *string) ui.Element {
	t := c.Theme()
	words := strings.Fields(strings.ToLower(s.query))
	var found []Icon
	for _, ic := range icons {
		text := strings.ToLower(ic.Name + " " + strings.Join(ic.Keywords, " "))
		ok := true
		for _, w := range words {
			ok = ok && strings.Contains(text, w)
		}
		if ok {
			found = append(found, ic)
		}
	}
	s.ring = max(0, min(len(found)-1, s.ring))
	cell, gap := t.Space(11), t.Space(1)
	root := ui.Column(c).Gap(t.Space(3))
	root.Children(func() {
		search := ui.TextInput(c, &s.query).FillWidth().Placeholder("Search icons").Label("Search icons").AutoFocus()
		if search.Changed() {
			s.ring = 0
		}
		across := max(1, s.across)
		// The field would take the arrows and Enter for its text, so they
		// are taken before it gets them.
		search.OnTextKey(func(k ui.TextKey) bool {
			if k.Mods != 0 || len(found) == 0 {
				return false
			}
			switch k.Key {
			case ui.KeyRight:
				s.ring = min(len(found)-1, s.ring+1)
			case ui.KeyLeft:
				s.ring = max(0, s.ring-1)
			case ui.KeyDown:
				s.ring = min(len(found)-1, s.ring+across)
			case ui.KeyUp:
				s.ring = max(0, s.ring-across)
			case ui.KeyEnter:
				*chosen = found[s.ring].Name
			default:
				return false
			}
			return true
		})
		ring := search.Focused()
		grid := ui.Scroll(c).Grow(1).FillWidth().MinHeight(t.Space(40))
		grid.Children(func() {
			if len(found) == 0 {
				ui.Text(c, "No icons match").TextColor(t.TextMuted).Padding(t.Space(4))
				return
			}
			ui.Row(c).Wrap().Gap(gap).FillWidth().Role(ui.RoleList).Label("Icons").Children(func() {
				for i, ic := range found {
					on := ic.Name == *chosen
					b := ui.ButtonBase(c.Key(ic.Name)).Size(cell, cell).Radius(t.Space(2)).Center().Label(ic.Name).
						Selected(on).Cursor(ui.CursorPointer).Tooltip(ic.Name)
					switch {
					case on:
						b.Background(t.Accent)
					case b.Hovered():
						b.Background(t.SurfaceHover)
					}
					if i == s.ring && ring {
						b.Border(2, t.Accent)
					}
					if b.Clicked() {
						*chosen = ic.Name
						s.ring = i
						root.SetChanged()
					}
					b.Children(func() {
						col := t.Text
						if on {
							col = t.AccentText
						}
						ui.Icon(c, ic.Image).Size(t.Space(5), t.Space(5)).TextColor(col)
					})
				}
			})
		})
		if w := grid.Bounds().W; w > 0 {
			s.across = max(1, int((w+gap)/(cell+gap)))
		}
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Padding(t.Space(3)).Radius(t.Space(2.5)).Background(t.Surface).Children(func() {
			var pick *Icon
			for i := range icons {
				if icons[i].Name == *chosen {
					pick = &icons[i]
				}
			}
			ui.Box(c).Size(t.Space(14), t.Space(14)).Radius(t.Space(2.5)).Background(t.Background).Border(1, t.Border).Center().Children(func() {
				if pick != nil {
					ui.Icon(c, pick.Image).Size(t.Space(8), t.Space(8))
				}
			})
			ui.Column(c).Grow(1).Gap(t.Space(0.5)).Children(func() {
				name := "No icon"
				if pick != nil {
					name = pick.Name
				}
				ui.Text(c, name).Bold().Font("monospace")
				ui.Text(c, fmt.Sprintf("%d of %d icons", len(found), len(icons))).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
			})
		})
	})
	return root
}
