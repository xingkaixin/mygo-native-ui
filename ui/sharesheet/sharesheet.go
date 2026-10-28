// Package sharesheet builds a share sheet, as iOS and macOS show one for
// what is being shared: the item at the top with its picture, title and
// detail; a row of people to send it to, each with the app it goes
// through on their picture; a row of apps; and a list of actions, each
// with its symbol. The rows scroll sideways when they hold more than fit.
// A click on a person, an app or an action reports it, as does the close
// button.
//
//	ev := sharesheet.View(c, item, people, apps, actions)
//	switch {
//	case ev.Person != nil:
//		send(ev.Person.Name)
//	case ev.Closed:
//		app.sharing = false
//	}
package sharesheet

import "github.com/egoist/mygo/ui"

// Item is what is shared: its title, a detail such as its size, and its
// picture, if any.
type Item struct {
	Title, Detail string
	Image         *ui.Bitmap
}

// Target is an app or an action: its name, its symbol, drawn in white on
// an app's color or in the text color by an action, and an app's color.
type Target struct {
	Name  string
	Icon  *ui.SVG
	Color ui.Color
}

// Person is someone to share with: their name, their picture, initials
// without one, and the app it goes through.
type Person struct {
	Name  string
	Image *ui.Bitmap
	Via   Target
}

// Events is a sheet, and what was clicked in it in the frame: a person,
// an app or an action, nil when none was, or the close button.
type Events struct {
	ui.Element
	Person *Person
	App    *Target
	Action *Target
	Closed bool
}

var closeIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M7 7l10 10M17 7L7 17" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round"/></svg>`))

// View creates the sheet sharing item.
func View(c *ui.Context, item Item, people []Person, apps, actions []Target) Events {
	t := c.Theme()
	var ev Events
	root := ui.Column(c).Width(t.Space(96)).Gap(t.Space(4)).Padding(t.Space(4)).Radius(t.Space(4)).Background(t.Background).Role(ui.RoleDialog).Label("Share")
	root.Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			if item.Image != nil {
				ui.Image(c, item.Image).Size(t.Space(12), t.Space(12)).Fit(ui.Cover).Radius(t.Space(2))
			}
			ui.Column(c).Grow(1).Shrink(1).Children(func() {
				ui.Text(c, item.Title).Bold().SingleLine().Ellipsis("…")
				if item.Detail != "" {
					ui.Text(c, item.Detail).FontSize(t.FontSize - 1).TextColor(t.TextMuted).SingleLine()
				}
			})
			x := ui.ButtonBase(c).Size(t.Space(7), t.Space(7)).Radius(t.Space(3.5)).Background(t.Text.Alpha(0.08)).Center().Label("Close")
			ev.Closed = x.Clicked() || root.Shortcut(0, ui.KeyEscape)
			x.Children(func() {
				ui.Icon(c, closeIcon).Size(t.Space(3.5), t.Space(3.5)).TextColor(t.TextMuted)
			})
		})
		ui.Divider(c)
		if len(people) > 0 {
			strip(c, "People", func() {
				for i, p := range people {
					if person(c.Key(i), p) {
						chosen := p
						ev.Person = &chosen
					}
				}
			})
		}
		if len(apps) > 0 {
			strip(c, "Apps", func() {
				for i, a := range apps {
					if app(c.Key(i), a) {
						chosen := a
						ev.App = &chosen
					}
				}
			})
		}
		if len(actions) > 0 {
			ui.Column(c).Radius(t.Space(3)).Background(t.Surface).Clip().Children(func() {
				for i, a := range actions {
					if i > 0 {
						ui.Divider(c.Key(-1 - i)).MarginStart(t.Space(4))
					}
					row := ui.ButtonBase(c.Key(i)).Padding(t.Space(3), t.Space(4)).Label(a.Name)
					if row.Hovered() {
						row.Background(t.SurfaceHover)
					}
					if row.Clicked() {
						chosen := a
						ev.Action = &chosen
					}
					row.Children(func() {
						ui.Row(c).FillWidth().AlignItems(ui.Center).Children(func() {
							ui.Text(c, a.Name).Grow(1)
							if a.Icon != nil {
								ui.Icon(c, a.Icon).Size(t.Space(5), t.Space(5)).TextColor(t.Text)
							}
						})
					})
				}
			})
		}
	})
	if ev.Person != nil || ev.App != nil || ev.Action != nil || ev.Closed {
		root.SetChanged()
	}
	ev.Element = root
	return ev
}

// strip creates a row that scrolls sideways, named label, of what fn
// builds.
func strip(c *ui.Context, label string, fn func()) {
	t := c.Theme()
	ui.ScrollHorizontal(c).Label(label).Margin(0, -t.Space(4)).Children(func() {
		ui.Row(c).Gap(t.Space(2)).Padding(0, t.Space(4)).Children(fn)
	})
}

// tile creates a tile of a strip, a picture above its name, and reports
// a click.
func tile(c *ui.Context, name string, picture func(d float32)) bool {
	t := c.Theme()
	d := t.Space(15)
	b := ui.ButtonBase(c).Width(d+t.Space(4)).Padding(t.Space(1.5), 0).Radius(t.Space(3)).Label(name)
	if b.Hovered() {
		b.Background(t.SurfaceHover)
	}
	b.Children(func() {
		ui.Column(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
			picture(d)
			ui.Text(c, name).FontSize(t.FontSize - 2).TextAlign(ui.Center).MaxLines(2).Ellipsis("…")
		})
	})
	return b.Clicked()
}

func person(c *ui.Context, p Person) bool {
	t := c.Theme()
	return tile(c, p.Name, func(d float32) {
		ui.Box(c).Size(d, d).Children(func() {
			ui.Avatar(c, p.Name, p.Image).Size(d, d).Radius(d / 2)
			if p.Via.Icon != nil {
				b := d * 0.36
				ui.Box(c).Absolute().Right(-2).Bottom(-2).Size(b, b).Radius(b*0.26).Background(p.Via.Color).
					Border(2, t.Background).Center().Children(func() {
					ui.Icon(c, p.Via.Icon).Size(b*0.6, b*0.6).TextColor(ui.RGB(255, 255, 255))
				})
			}
		})
	})
}

func app(c *ui.Context, a Target) bool {
	return tile(c, a.Name, func(d float32) {
		ui.Box(c).Size(d, d).Radius(d * 0.24).Background(a.Color).Center().Children(func() {
			if a.Icon != nil {
				ui.Icon(c, a.Icon).Size(d*0.55, d*0.55).TextColor(ui.RGB(255, 255, 255))
			}
		})
	})
}
