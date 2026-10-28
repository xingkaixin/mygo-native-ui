// Package metro follows a ride on a metro line, as the transit views of
// map apps do: the stops down the line in its color, the train between
// them, and the minutes to each stop ahead. Stops behind the train fade,
// and those long passed fold into a row that opens them, as do the stops
// past the one to get off at. The stop to get off at is marked, and the
// header tells how long until it; a click on a stop ahead makes it the one.
// Stops where other lines meet show those lines' badges.
//
//	metro.View(c, &app.ride, line, stops, app.train, &app.to)
//
// The app moves the train, at, as it runs: 2.5 is halfway from the third
// stop to the fourth. Changed reports a new stop to get off at.
package metro

import (
	"math"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// Line is a metro line: the name on its badge, such as "2", and its color.
type Line struct {
	Name  string
	Color ui.Color
}

// Stop is a stop of the line: its name, the other lines met there, and
// the minutes from the line's first stop to it.
type Stop struct {
	Name      string
	Transfers []Line
	Minutes   float64
}

// State is what a ride keeps: which of its folded stops are open. Keep it
// in the app's state, one for each ride.
type State struct {
	openBefore, openAfter bool
}

// entry is a row of the ride: a stop, or a fold of the stops from to to.
type entry struct {
	stop     int
	from, to int
	before   bool
}

// View creates the ride on line along stops, the train at at, getting off
// at *to.
func View(c *ui.Context, s *State, line Line, stops []Stop, at float64, to *int) ui.Element {
	t := c.Theme()
	changed := false
	n := len(stops)
	at = math.Max(0, math.Min(float64(n-1), at))
	*to = max(0, min(n-1, *to))
	next := int(math.Ceil(at))
	var rows []entry
	first := max(0, int(at)-1)
	if first >= 2 && !s.openBefore {
		rows = append(rows, entry{from: 0, to: first, before: true, stop: -1})
	} else {
		first = 0
	}
	last := *to
	if last < next {
		last = next
	}
	for i := first; i <= last && i < n; i++ {
		rows = append(rows, entry{stop: i})
	}
	if n-1-last >= 2 && !s.openAfter {
		rows = append(rows, entry{from: last + 1, to: n, stop: -1})
	} else {
		for i := last + 1; i < n; i++ {
			rows = append(rows, entry{stop: i})
		}
	}
	minutes := func(f float64) float64 {
		if n < 2 {
			return 0
		}
		i := min(n-2, int(f))
		return stops[i].Minutes + (stops[i+1].Minutes-stops[i].Minutes)*(f-float64(i))
	}
	now := minutes(at)
	rowH := t.Space(11)
	gutter := t.Space(12)
	root := ui.Column(c).Gap(t.Space(4))
	root.Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			badge(c, line, t.Space(8))
			ui.Column(c).Gap(t.Space(0.5)).Grow(1).Children(func() {
				ui.Text(c, "Toward "+stops[n-1].Name).Bold()
				left := stops[*to].Minutes - now
				var msg string
				switch {
				case *to < next || left <= 0:
					msg = "At " + stops[*to].Name
				default:
					k := *to - int(at)
					stopsWord := " stops"
					if k == 1 {
						stopsWord = " stop"
					}
					msg = strconv.Itoa(k) + stopsWord + " to " + stops[*to].Name
				}
				ui.Text(c, msg).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
			})
			if left := stops[*to].Minutes - now; left > 0 {
				ui.Row(c).Gap(t.Space(1)).AlignItems(ui.Baseline).Children(func() {
					ui.Text(c, strconv.Itoa(int(math.Ceil(left)))).FontSize(t.FontSize + 12).Bold().FontFeatures("tnum")
					ui.Text(c, "min").TextColor(t.TextMuted)
				})
			}
		})
		visual := func(stop int) (float32, bool) {
			for k, e := range rows {
				if e.stop == stop {
					return float32(k)*rowH + rowH/2, true
				}
			}
			return 0, false
		}
		dest := *to
		list := ui.Column(c)
		list.DrawOver(func(p *ui.Painter, r ui.Rect) {
			x := r.X + gutter/2
			ahead, behind := line.Color, line.Color.Alpha(0.3)
			trainY := r.Y
			if y0, ok := visual(int(at)); ok {
				trainY = r.Y + y0
				if y1, ok := visual(next); ok {
					trainY += (y1 - y0) * float32(at-math.Floor(at))
				}
			}
			top, bottom := r.Y+rowH/2, r.Y+float32(len(rows)-1)*rowH+rowH/2
			if rows[0].stop < 0 {
				top = r.Y
			}
			if rows[len(rows)-1].stop < 0 {
				bottom = r.Y + r.H
			}
			p.Line(x, top, x, trainY, 6, behind)
			p.Line(x, trainY, x, bottom, 6, ahead)
			for k, e := range rows {
				y := r.Y + float32(k)*rowH + rowH/2
				if e.stop < 0 {
					for d := float32(-1); d <= 1; d++ {
						var dot ui.Path
						dot.Circle(x, y+d*5, 1.5)
						p.FillPath(&dot, t.Background)
					}
					continue
				}
				col := ahead
				if float64(e.stop) < at {
					col = behind
				}
				var dot ui.Path
				switch {
				case e.stop == dest:
					dot.Circle(x, y, 8)
					p.FillPath(&dot, col)
					var in ui.Path
					in.Circle(x, y, 3.5)
					p.FillPath(&in, t.Background)
				case len(stops[e.stop].Transfers) > 0:
					dot.Circle(x, y, 6)
					p.FillPath(&dot, t.Background)
					p.StrokePath(&dot, 2.5, t.Text.Alpha(0.8))
				default:
					dot.Circle(x, y, 4.5)
					p.FillPath(&dot, t.Background)
					p.StrokePath(&dot, 2, col)
				}
			}
			train := ui.Rect{X: x - 9, Y: trainY - 12, W: 18, H: 24}
			p.Shadow(train, 6, 0, 2, 6, 0, ui.RGBA(0, 0, 0, 0.3))
			p.Fill(train, ui.RGB(255, 255, 255), 6)
			p.Fill(ui.Rect{X: train.X + 2, Y: train.Y + 2, W: 14, H: 20}, line.Color, 4.5)
			p.Fill(ui.Rect{X: train.X + 5, Y: train.Y + 5, W: 8, H: 5}, ui.RGBA(255, 255, 255, 0.85), 1.5)
		})
		list.Children(func() {
			for k, e := range rows {
				if e.stop < 0 {
					fold := ui.ButtonBase(c.Key(k)).Height(rowH).PaddingStart(gutter).AlignItems(ui.Center).Justify(ui.Start).Radius(t.Space(1.5))
					if fold.Hovered() {
						fold.Background(t.SurfaceHover)
					}
					label := strconv.Itoa(e.to-e.from) + " stops"
					if e.before {
						label += " passed"
					} else {
						label += " more to " + stops[n-1].Name
					}
					fold.Label(label)
					fold.Children(func() { ui.Text(c, label).FontSize(t.FontSize - 1).TextColor(t.TextMuted) })
					if fold.Clicked() {
						if e.before {
							s.openBefore = true
						} else {
							s.openAfter = true
						}
					}
					continue
				}
				i := e.stop
				st := stops[i]
				passed := float64(i) < at
				row := ui.ButtonBase(c.Key(k)).Height(rowH).PaddingStart(gutter).PaddingEnd(t.Space(3)).Gap(t.Space(2)).
					AlignItems(ui.Center).Radius(t.Space(1.5)).Label(st.Name).Disabled(passed)
				if row.Hovered() && !passed {
					row.Background(t.SurfaceHover)
				}
				if row.Clicked() && i != *to {
					*to, changed = i, true
				}
				row.Children(func() {
					name := ui.Text(c, st.Name).SingleLine().Shrink(1)
					switch {
					case passed:
						name.TextColor(t.TextMuted)
					case i == *to:
						name.Bold()
					}
					for j, l := range st.Transfers {
						badge(c.Key(j), l, t.Space(5))
					}
					ui.Box(c).Grow(1)
					if passed {
						return
					}
					left := st.Minutes - now
					label := strconv.Itoa(int(math.Ceil(left))) + " min"
					if left < 0.5 {
						label = "Now"
					}
					tx := ui.Text(c, label).FontSize(t.FontSize - 1).FontFeatures("tnum")
					if i != *to {
						tx.TextColor(t.TextMuted)
					} else {
						tx.Bold()
					}
				})
			}
		})
	})
	if changed {
		root.SetChanged()
	}
	return root
}

// badge creates the round badge of l, size across.
func badge(c *ui.Context, l Line, size float32) {
	ui.Box(c).Size(size, size).Shrink(0).Radius(size / 2).Background(l.Color).Center().Children(func() {
		ui.Text(c, l.Name).FontSize(size * 0.55).Bold().TextColor(ui.RGB(255, 255, 255))
	})
}
