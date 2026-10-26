// Package facepile shows who is here, as collaborative apps show the
// people in a document beside its title: their avatars overlapping in a
// row, each ringed in the color of their cursor, or in the background to
// stand apart, those away faded, and past the room given "+3", which
// opens a list of everyone. The pointer over an avatar names its person;
// a click on one chooses that person, as to follow them, which Chosen
// reports in the same frame.
//
//	if i, ok := facepile.View(c, app.people, 4).Chosen(); ok {
//		app.follow(app.people[i])
//	}
package facepile

import (
	"fmt"

	"github.com/egoist/mygo/ui"
)

// Person is someone here: their name, their picture, nil for their
// initials, the color of their cursor, zero for none, and whether they
// are away, as with the document in a tab they left.
type Person struct {
	Name  string
	Image *ui.Bitmap
	Color ui.Color
	Away  bool
}

// Events is what the user did to a pile in a frame.
type Events struct {
	chosen int
}

// Chosen returns the index of the person clicked in this frame.
func (e Events) Chosen() (int, bool) { return e.chosen - 1, e.chosen > 0 }

// View creates the pile of people, showing at most limit of them before
// "+N".
func View(c *ui.Context, people []Person, limit int) Events {
	t := c.Theme()
	size, ring := t.Space(8), float32(2)
	shown := people
	if len(people) > limit && limit > 0 {
		shown = people[:limit-1]
	}
	var ev Events
	ui.Row(c).Label("People").Role(ui.RoleGroup).Children(func() {
		for i, p := range shown {
			face := ui.ButtonBase(c.Key(i)).Size(size, size).Radius(size / 2).Label(p.Name).Tooltip(p.Name)
			if i > 0 {
				face.MarginStart(-size / 4)
			}
			if face.Clicked() {
				ev.chosen = i + 1
			}
			face.Children(func() { avatar(c, p, size, ring) })
		}
		if rest := len(people) - len(shown); rest > 0 {
			more := ui.ButtonBase(c).Size(size, size).Radius(size/2).MarginStart(-size/4).Background(t.Surface).
				Border(ring, t.Background).Label(fmt.Sprintf("%d more", rest)).Role(ui.RolePopUpButton)
			open := ui.Local(more, "facepile.open", func() bool { return false })
			if more.Clicked() {
				*open = !*open
			}
			if more.Hovered() || *open {
				more.Background(t.SurfaceHover)
			}
			more.Expanded(*open).Children(func() {
				ui.Text(c, fmt.Sprintf("+%d", rest)).FontSize(t.FontSize - 2).Bold().FontFeatures("tnum")
			})
			ui.Popover(c, more, open, func() {
				ui.Column(c).Gap(t.Space(0.5)).MinWidth(t.Space(52)).Children(func() {
					for i, p := range people {
						row := ui.ButtonBase(c.Key(i)).Justify(ui.Start).Gap(t.Space(2.5)).Padding(t.Space(1.5), t.Space(2)).
							Radius(t.Space(1.5)).Label(p.Name)
						if row.Hovered() {
							row.Background(t.Text.Alpha(0.06))
						}
						if row.Clicked() {
							ev.chosen, *open = i+1, false
						}
						row.Children(func() {
							avatar(c, p, t.Space(6), ring)
							name := ui.Text(c, p.Name).SingleLine().Grow(1)
							if p.Away {
								name.TextColor(t.TextMuted)
								ui.Text(c, "Away").FontSize(t.FontSize - 2).TextColor(t.TextMuted)
							}
						})
					}
				})
			})
		}
	})
	return ev
}

// avatar shows p's picture or initials in a circle size across, inside a
// ring of their color.
func avatar(c *ui.Context, p Person, size, ring float32) {
	t := c.Theme()
	color := p.Color
	if color.A == 0 {
		color = t.Background
	}
	frame := ui.Box(c).Size(size, size).Radius(size/2).Border(ring, color).Background(t.Background).Clip()
	if p.Away {
		frame.Opacity(0.45)
	}
	frame.Children(func() {
		inner := size - 2*ring
		ui.Avatar(c, p.Name, p.Image).Size(inner, inner).Radius(inner / 2).FontSize(inner * 0.4)
	})
}
