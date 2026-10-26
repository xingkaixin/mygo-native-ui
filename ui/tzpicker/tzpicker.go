// Package tzpicker chooses a time zone by a city in it, as calendars and
// clocks do: a button showing the city chosen and its offset from UTC,
// which opens a panel of some 170 cities across every zone, from the
// furthest behind UTC to the furthest ahead, each with its country, its
// day and time there now and its offset, with the user's own time first.
// Typing finds cities by their names, countries, zones and the
// abbreviations of their zones, as "PST"; Up and Down move among them,
// as the pointer does, and Enter or a click chooses one, which closes
// the panel, as Escape or a click outside it does.
//
// The cities' zones come from the system; import time/tzdata for systems
// without them, as Windows can be.
//
//	if tzpicker.View(c, &app.city).Changed() {
//		app.schedule.Location = app.city.Location()
//	}
package tzpicker

import (
	_ "embed"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/egoist/mygo/i18n"
	"github.com/egoist/mygo/ui"
)

//go:embed cities.txt
var data string

// City is a city and the zone it keeps the time of. Its zero value is
// the user's own time, as the system has it.
type City struct {
	Name, Country, Zone string
}

// Location returns the city's zone, time.Local for the zero City or a
// zone the system lacks.
func (ct City) Location() *time.Location {
	if ct.Zone == "" {
		return time.Local
	}
	if loc := zones()[ct.Zone]; loc != nil {
		return loc
	}
	return time.Local
}

func (ct City) name() string {
	if ct.Name == "" {
		return "Local Time"
	}
	return ct.Name
}

var cities = sync.OnceValues(func() ([]City, map[string]*time.Location) {
	locs := map[string]*time.Location{}
	var list []City
	for line := range strings.Lines(data) {
		f := strings.Split(strings.TrimSpace(line), "|")
		if len(f) != 3 {
			continue
		}
		if _, ok := locs[f[2]]; !ok {
			loc, err := time.LoadLocation(f[2])
			if err != nil {
				continue
			}
			locs[f[2]] = loc
		}
		list = append(list, City{f[0], f[1], f[2]})
	}
	return list, locs
})

func zones() map[string]*time.Location {
	_, locs := cities()
	return locs
}

// state is what a picker keeps: whether its panel shows, the text typed
// in it, the city the arrows are on and the one last under the pointer,
// whether the arrows moved to a city not yet scrolled to, and the place
// of its list.
type state struct {
	open    bool
	query   string
	on      City
	hovered City
	hover   bool
	scroll  bool
	list    ui.ListState
}

// View creates the button choosing *city, which opens the panel of the
// cities. Changed reports a city chosen.
func View(c *ui.Context, city *City) ui.Element {
	t := c.Theme()
	before := *city
	now := c.Now()
	locale := c.Region().FormatLocale(c.Locale())
	button := ui.ButtonBase(c).Gap(t.Space(2)).Padding(t.Space(1.5), t.Space(2), t.Space(1.5), t.Space(3)).
		Radius(t.Radius).Background(t.Surface).Border(1, t.Border).Role(ui.RolePopUpButton).Label("Time zone").
		Value(city.name())
	st := ui.Local(button, "tzpicker.state", func() state { return state{} })
	if button.Clicked() {
		st.open = !st.open
		st.query, st.on, st.scroll = "", *city, true
	}
	if button.Hovered() {
		button.Background(t.SurfaceHover)
	}
	button.Expanded(st.open).Children(func() {
		ui.Text(c, city.name()).SingleLine().Shrink(1)
		ui.Text(c, offset(now, city.Location())).TextColor(t.TextMuted).FontFeatures("tnum").SingleLine().Grow(1)
		chevron(c)
	})
	ui.PopoverBase(c, button, &st.open, func(panel ui.Element) {
		panel.Width(t.Space(110)).Margin(t.Space(1), 0, 0, 0).Radius(t.Space(2.5)).Clip().Background(t.Background).
			Border(1, t.Border).Shadow(0, 8, 24, 0, ui.RGBA(0, 0, 0, 0.16)).Role(ui.RoleDialog).Label("Time zones")
		c.After(now.Truncate(time.Minute).Add(time.Minute).Sub(now))
		found := find(now, st.query)
		if !slices.Contains(found, st.on) {
			st.on, st.scroll = City{}, true
			if len(found) > 0 {
				st.on = found[0]
			}
		}
		choose := func(ct City) {
			*city, st.open = ct, false
		}
		panel.Children(func() {
			var input, active ui.Element
			ui.Row(c).Gap(t.Space(2)).Padding(t.Space(2.5), t.Space(3)).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(t.Space(4), t.Space(4)).Draw(func(p *ui.Painter, r ui.Rect) { magnifier(p, r, t.TextMuted) })
				input = ui.TextInputBase(c, &st.query).AutoFocus().Grow(1).Placeholder("Search cities and time zones").
					Label("Search cities and time zones").Role(ui.RoleComboBox).Expanded(true).OnTextKey(func(k ui.TextKey) bool {
					d := 0
					switch {
					case k.Mods != 0:
					case k.Key == ui.KeyDown:
						d = 1
					case k.Key == ui.KeyUp:
						d = -1
					}
					if d == 0 {
						return false
					}
					if i := slices.Index(found, st.on) + d; i >= 0 && i < len(found) {
						st.on, st.scroll = found[i], true
					}
					return true
				})
			})
			if input.Submitted() && len(found) > 0 {
				choose(st.on)
			}
			ui.Divider(c)
			if st.scroll {
				if i := slices.Index(found, st.on); i >= 0 {
					st.list.ScrollIntoView(i)
				}
				st.scroll = false
			}
			hovered, hover := City{}, false
			st.list.Label = func(i int) string { return found[i].name() }
			ui.List(c, &st.list, len(found), func(i int) {
				ct := found[i]
				on := ct == st.on
				loc := ct.Location()
				row := ui.Row(c).Height(t.Space(11)).Padding(0, t.Space(2.5)).Gap(t.Space(3)).Radius(t.Space(1.5)).
					AlignItems(ui.Center).Label(ct.name()).Selected(on)
				if row.Hovered() {
					hovered, hover = ct, true
				}
				if row.Clicked() {
					choose(ct)
				}
				if on {
					active = row
				}
				muted := t.TextMuted
				if on {
					muted = t.AccentText.Alpha(0.8)
				}
				row.Children(func() {
					ui.Column(c).Grow(1).Shrink(1).Children(func() {
						ui.Text(c, ct.name()).SingleLine()
						place := ct.Country
						if ct.Zone == "" {
							place, _ = now.In(loc).Zone()
						}
						ui.Text(c, place).FontSize(t.FontSize - 2).TextColor(muted).SingleLine()
					})
					ui.Text(c, i18n.FormatDate(locale, now.In(loc), i18n.DateOptions{Skeleton: "Ejm"})).FontFeatures("tnum").NoWrap()
					ui.Text(c, offset(now, loc)).FontSize(t.FontSize - 1).FontFeatures("tnum").TextColor(muted).
						Width(t.Space(18)).TextAlign(ui.End).NoWrap()
				})
			}).Height(t.Space(100)).Padding(t.Space(1)).Label("Cities").Children(func() {
				if len(found) == 0 {
					ui.Text(c, "No cities match").TextColor(t.TextMuted).Padding(t.Space(2))
				}
			})
			// The pointer moving onto a city puts the arrows there;
			// resting there, it leaves the arrows to move on.
			if hover && (hovered != st.hovered || !st.hover) {
				st.on = hovered
			}
			st.hovered, st.hover = hovered, hover
			if active.Valid() {
				input.ActiveDescendant(active)
			}
		})
	})
	if *city != before {
		button.SetChanged()
	}
	return button
}

// find returns the user's own time and the cities holding every word of
// query in their names, countries, zones or the abbreviations of their
// zones, case aside, from the furthest behind UTC at now to the furthest
// ahead.
func find(now time.Time, query string) []City {
	words := strings.Fields(strings.ToLower(query))
	// "PST" should find Los Angeles in summer too, so both of a zone's
	// abbreviations count, its winter's and its summer's.
	jan := time.Date(now.Year(), time.January, 1, 0, 0, 0, 0, time.UTC)
	jul := time.Date(now.Year(), time.July, 1, 0, 0, 0, 0, time.UTC)
	match := func(ct City) bool {
		loc := ct.Location()
		w, _ := jan.In(loc).Zone()
		s, _ := jul.In(loc).Zone()
		text := strings.ToLower(strings.Join([]string{ct.name(), ct.Country, strings.ReplaceAll(ct.Zone, "_", " "), w, s}, " "))
		return !slices.ContainsFunc(words, func(w string) bool { return !strings.Contains(text, w) })
	}
	list, _ := cities()
	var found []City
	for _, ct := range list {
		if match(ct) {
			found = append(found, ct)
		}
	}
	slices.SortStableFunc(found, func(a, b City) int {
		_, x := now.In(a.Location()).Zone()
		_, y := now.In(b.Location()).Zone()
		return x - y
	})
	if match(City{}) {
		found = slices.Insert(found, 0, City{})
	}
	return found
}

// offset writes loc's offset from UTC at now: "GMT", "GMT+9", "GMT−3:30".
func offset(now time.Time, loc *time.Location) string {
	_, sec := now.In(loc).Zone()
	if sec == 0 {
		return "GMT"
	}
	sign := "+"
	if sec < 0 {
		sign, sec = "−", -sec
	}
	if m := sec / 60 % 60; m != 0 {
		return fmt.Sprintf("GMT%s%d:%02d", sign, sec/3600, m)
	}
	return fmt.Sprintf("GMT%s%d", sign, sec/3600)
}

// chevron shows an arrow pointing down.
func chevron(c *ui.Context) {
	t := c.Theme()
	ui.Box(c).Size(t.Space(3), t.Space(3)).Draw(func(p *ui.Painter, r ui.Rect) {
		cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W/5
		p.StrokePolyline([][2]float32{{cx - k, cy - k/2}, {cx, cy + k/2}, {cx + k, cy - k/2}}, 1.5, t.TextMuted)
	})
}

func magnifier(p *ui.Painter, r ui.Rect, color ui.Color) {
	cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W*0.4
	var lens ui.Path
	lens.Circle(cx-k*0.2, cy-k*0.2, k*0.6)
	p.StrokePath(&lens, 1.5, color)
	p.StrokePolyline([][2]float32{{cx + k*0.25, cy + k*0.25}, {cx + k*0.85, cy + k*0.85}}, 1.5, color)
}
