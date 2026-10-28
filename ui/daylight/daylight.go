// Package daylight shows the daylight of a place on a day, as the sun
// pages of weather apps do: sunrise, sunset and the length of the day, and
// how it changed since the day before; the sun's path across the 24 hours,
// over the sky's light at each hour, night and the three twilights, the
// golden hour and day; and below, the year, a column a day with its
// daylight from sunrise to sunset, the day shown marked. Pointing at the
// path tells the time and the sun's height there, and a click or a drag on
// the year, or Left and Right while it has the focus, shows another day.
// On the day it is now, a mark stands at the time.
//
//	daylight.View(c, &app.sun, daylight.Place{Lat: 64.15, Lon: -21.94}, &app.day)
//
// Times are in the location of *day, which should be the place's.
package daylight

import (
	"math"
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
)

// Place is where on the earth, in degrees: north and east are positive.
type Place struct {
	Lat, Lon float64
}

// Times is the daylight of a day: when the sun rises and sets, each zero
// on a day it does not, and how long it is up.
type Times struct {
	Sunrise, Sunset time.Time
	Daylight        time.Duration
}

// horizon is the sun's height as its top edge meets the horizon, the
// refraction of the air lifting it.
const horizon = -0.833

// Elevation returns the height of the sun above the horizon at t, in
// degrees, by the approximations of the Astronomical Almanac, within a
// hundredth of a degree for the years near 2000.
func Elevation(p Place, t time.Time) float64 {
	rad := math.Pi / 180
	n := float64(t.UnixNano())/float64(24*time.Hour) + 2440587.5 - 2451545.0
	l := math.Mod(280.460+0.9856474*n, 360)
	g := math.Mod(357.528+0.9856003*n, 360) * rad
	lambda := (l + 1.915*math.Sin(g) + 0.020*math.Sin(2*g)) * rad
	eps := (23.439 - 0.0000004*n) * rad
	ra := math.Atan2(math.Cos(eps)*math.Sin(lambda), math.Cos(lambda))
	dec := math.Asin(math.Sin(eps) * math.Sin(lambda))
	gmst := math.Mod(280.46061837+360.98564736629*n, 360) * rad
	h := gmst + p.Lon*rad - ra
	lat := p.Lat * rad
	return math.Asin(math.Sin(lat)*math.Sin(dec)+math.Cos(lat)*math.Cos(dec)*math.Cos(h)) / rad
}

// Day returns the daylight of the day of day, in its location.
func Day(p Place, day time.Time) Times {
	y, m, d := day.Date()
	start := time.Date(y, m, d, 0, 0, 0, 0, day.Location())
	end := start.AddDate(0, 0, 1)
	var tm Times
	const step = 10 * time.Minute
	prev, up := start, Elevation(p, start) > horizon
	var rose time.Time
	if up {
		rose = start
	}
	for at := start.Add(step); ; at = at.Add(step) {
		if at.After(end) {
			at = end
		}
		now := Elevation(p, at) > horizon
		if now != up {
			lo, hi := prev, at
			for hi.Sub(lo) > time.Second {
				mid := lo.Add(hi.Sub(lo) / 2)
				if (Elevation(p, mid) > horizon) == up {
					lo = mid
				} else {
					hi = mid
				}
			}
			cross := hi.Truncate(time.Second)
			if now {
				rose = cross
				if tm.Sunrise.IsZero() {
					tm.Sunrise = cross
				}
			} else {
				tm.Sunset = cross
				tm.Daylight += cross.Sub(rose)
			}
			up = now
		}
		if at.Equal(end) {
			break
		}
		prev = at
	}
	if up {
		tm.Daylight += end.Sub(rose)
	}
	return tm
}

// State is what a view keeps: the time pointed at, and the daylight of
// each day of the year shown. Keep it in the app's state, one for each
// view.
type State struct {
	hover   float32
	year    []Times
	yearFor yearKey
}

type yearKey struct {
	place Place
	year  int
	loc   string
}

// sky is the light of the sky as the sun stands at elevation e.
func sky(e float64) ui.Color {
	switch {
	case e < -18:
		return ui.Hex("#141a33")
	case e < -12:
		return ui.Hex("#1f2852")
	case e < -6:
		return ui.Hex("#2f3f7a")
	case e < horizon:
		return ui.Hex("#5b6fae")
	case e < 6:
		return ui.Hex("#f2c47e")
	}
	return ui.Hex("#a9cdf2")
}

var sun = ui.Hex("#ffc93c")

// View creates the daylight of p on *day.
func View(c *ui.Context, s *State, p Place, day *time.Time) ui.Element {
	t := c.Theme()
	changed := false
	d := *day
	loc := d.Location()
	y, m, dd := d.Date()
	start := time.Date(y, m, dd, 0, 0, 0, 0, loc)
	end := start.AddDate(0, 0, 1)
	today := Day(p, d)
	before := Day(p, start.AddDate(0, 0, -1).Add(12*time.Hour))
	if k := (yearKey{p, y, loc.String()}); s.yearFor != k || s.year == nil {
		s.year, s.yearFor = nil, k
		for at := time.Date(y, 1, 1, 12, 0, 0, 0, loc); at.Year() == y; at = at.AddDate(0, 0, 1) {
			s.year = append(s.year, Day(p, at))
		}
	}
	root := ui.Column(c).Gap(t.Space(4))
	root.Children(func() {
		ui.Row(c).Gap(t.Space(8)).Children(func() {
			stat := func(label, value string) {
				ui.Column(c).Gap(t.Space(0.5)).Children(func() {
					ui.Text(c, label).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
					ui.Text(c, value).FontSize(t.FontSize + 6).Bold().FontFeatures("tnum")
				})
			}
			switch {
			case today.Sunrise.IsZero() && today.Sunset.IsZero() && today.Daylight == 0:
				stat("Sun", "Polar night")
			case today.Sunrise.IsZero() && today.Sunset.IsZero():
				stat("Sun", "Midnight sun")
			default:
				stat("Sunrise", clock(today.Sunrise))
				stat("Sunset", clock(today.Sunset))
			}
			stat("Daylight", length(today.Daylight))
			diff := today.Daylight - before.Daylight
			sign := "+"
			if diff < 0 {
				sign, diff = "−", -diff
			}
			stat("Since yesterday", sign+short(diff))
		})
		s.path(c, p, start, end, today)
		if s.yearView(c, start, day) {
			changed = true
		}
	})
	if changed {
		root.SetChanged()
	}
	return root
}

// path creates the sun's path over the sky of the day from start to end.
func (s *State) path(c *ui.Context, p Place, start, end time.Time, today Times) {
	t := c.Theme()
	const lo, hi = -60.0, 90.0
	span := end.Sub(start)
	now := c.Now().In(start.Location())
	if !now.Before(start) && now.Before(end) {
		c.After(time.Minute)
	}
	chart := ui.Box(c).Height(t.Space(56)).Radius(t.Space(2)).Clip().Role(ui.RoleImage).Label("Path of the sun")
	if x, _, ok := chart.PointerPosition(); ok && chart.Hovered() {
		s.hover = x / max(1, chart.LayoutBounds().W)
	} else {
		s.hover = -1
	}
	hover := s.hover
	chart.Draw(func(pt *ui.Painter, r ui.Rect) {
		at := func(f float32) time.Time { return start.Add(time.Duration(float64(span) * float64(f))) }
		ey := func(e float64) float32 { return r.Y + r.H*float32((hi-e)/(hi-lo)) }
		cols := int(r.W)
		elev := make([]float64, cols+1)
		for i := range elev {
			elev[i] = Elevation(p, at(float32(i)/float32(cols)))
		}
		x0 := 0
		for i := 1; i <= cols; i++ {
			if i == cols || sky(elev[i]) != sky(elev[x0]) {
				pt.Fill(ui.Rect{X: r.X + float32(x0), Y: r.Y, W: float32(i - x0), H: r.H}, sky(elev[x0]), 0)
				x0 = i
			}
		}
		hy := ey(0)
		pt.Fill(ui.Rect{X: r.X, Y: hy, W: r.W, H: r.Y + r.H - hy}, ui.RGBA(0, 0, 0, 0.25), 0)
		pt.Line(r.X, hy, r.X+r.W, hy, 1, ui.RGBA(255, 255, 255, 0.5))
		for h := 3; h < 24; h += 3 {
			x := r.X + r.W*float32(h)/24
			pt.Line(x, r.Y+r.H-6, x, r.Y+r.H, 1, ui.RGBA(255, 255, 255, 0.4))
			label := strconv.Itoa(h)
			w, _ := pt.MeasureText(0, ui.Span{Text: label, Size: t.FontSize - 3})
			pt.Text(x-w/2, r.Y+r.H-8-t.FontSize, label, t.FontSize-3, ui.RGBA(255, 255, 255, 0.7))
		}
		var curve ui.Path
		for i, e := range elev {
			x, y := r.X+float32(i), ey(e)
			if i == 0 {
				curve.MoveTo(x, y)
			} else {
				curve.LineTo(x, y)
			}
		}
		pt.StrokePath(&curve, 2, ui.RGBA(255, 255, 255, 0.85))
		for _, tm := range []time.Time{today.Sunrise, today.Sunset} {
			if tm.IsZero() {
				continue
			}
			x := r.X + r.W*float32(float64(tm.Sub(start))/float64(span))
			var dot ui.Path
			dot.Circle(x, hy, 3)
			pt.FillPath(&dot, ui.RGB(255, 255, 255))
		}
		mark := func(f float32, label string) {
			x := r.X + r.W*f
			e := elev[max(0, min(cols, int(f*float32(cols))))]
			y := min(ey(e), r.Y+r.H-8)
			pt.Line(x, r.Y, x, r.Y+r.H, 1, ui.RGBA(255, 255, 255, 0.35))
			pt.Shadow(ui.Rect{X: x - 7, Y: y - 7, W: 14, H: 14}, 7, 0, 0, 10, 2, sun.Alpha(0.6))
			var dot ui.Path
			dot.Circle(x, y, 7)
			pt.FillPath(&dot, sun)
			pt.StrokePath(&dot, 1.5, ui.RGB(255, 255, 255))
			text := label + "  " + strconv.Itoa(int(math.Round(e))) + "°"
			w, _ := pt.MeasureText(0, ui.Span{Text: text, Size: t.FontSize - 1})
			tx := max(r.X+6, min(r.X+r.W-w-6, x-w/2))
			pt.Text(tx, r.Y+8, text, t.FontSize-1, ui.RGB(255, 255, 255))
		}
		switch {
		case hover >= 0:
			mark(hover, clock(at(hover)))
		case !now.Before(start) && now.Before(end):
			mark(float32(float64(now.Sub(start))/float64(span)), "Now "+clock(now))
		}
	})
}

// yearView creates the year of start's day, and reports whether a click,
// a drag or a key chose another day as *day.
func (s *State) yearView(c *ui.Context, start time.Time, day *time.Time) (changed bool) {
	t := c.Theme()
	y := start.Year()
	loc := start.Location()
	jan := time.Date(y, 1, 1, 12, 0, 0, 0, loc)
	shown := start.YearDay() - 1
	n := len(s.year)
	pick := func(i int) {
		i = max(0, min(n-1, i))
		if i != shown {
			*day, changed = jan.AddDate(0, 0, i), true
		}
	}
	strip := ui.Box(c).Height(t.Space(24)).Radius(t.Space(1.5)).Clip().Focusable().Cursor(ui.CursorPointer).
		Role(ui.RoleSlider).Label("Day").Value(start.Format("January 2"))
	if x, _, ok := strip.PointerPosition(); ok && strip.Pressed() {
		pick(int(x / max(1, strip.LayoutBounds().W) * float32(n)))
	}
	switch {
	case strip.Shortcut(0, ui.KeyLeft):
		pick(shown - 1)
	case strip.Shortcut(0, ui.KeyRight):
		pick(shown + 1)
	}
	year := s.year
	strip.Draw(func(p *ui.Painter, r ui.Rect) {
		p.Fill(r, sky(-20), 0)
		w := r.W / float32(n)
		for i, tm := range year {
			if tm.Daylight == 0 {
				continue
			}
			day := jan.AddDate(0, 0, i)
			mid := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc)
			f := func(at time.Time) float32 { return float32(at.Sub(mid).Hours() / 24) }
			rise, set := float32(0), float32(1)
			if !tm.Sunrise.IsZero() {
				rise = f(tm.Sunrise)
			}
			if !tm.Sunset.IsZero() {
				set = f(tm.Sunset)
			}
			span := func(a, b float32) {
				p.Fill(ui.Rect{X: r.X + float32(i)*w, Y: r.Y + r.H*a, W: w + 0.5, H: r.H * (b - a)}, sky(30), 0)
			}
			if set > rise {
				span(rise, set)
			} else {
				// The sun set after midnight, before it rose again.
				span(0, set)
				span(rise, 1)
			}
		}
		for m := time.February; m <= time.December; m++ {
			x := r.X + w*float32(time.Date(y, m, 1, 12, 0, 0, 0, loc).YearDay()-1)
			p.Line(x, r.Y, x, r.Y+r.H, 1, ui.RGBA(255, 255, 255, 0.25))
		}
		x := r.X + w*(float32(shown)+0.5)
		p.Line(x, r.Y, x, r.Y+r.H, 2, sun)
	})
	ui.Row(c).Children(func() {
		for m := time.January; m <= time.December; m++ {
			ui.Text(c.Key(int(m)), m.String()[:3]).FontSize(t.FontSize - 2).TextColor(t.TextMuted).Grow(1).Width(0)
		}
	})
	return changed
}

func clock(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.Format("15:04")
}

// length writes d as hours and minutes.
func length(d time.Duration) string {
	m := int(d.Round(time.Minute).Minutes())
	return strconv.Itoa(m/60) + "h " + strconv.Itoa(m%60) + "m"
}

// short writes d as minutes and seconds.
func short(d time.Duration) string {
	sec := int(d.Round(time.Second).Seconds())
	if sec < 60 {
		return strconv.Itoa(sec) + "s"
	}
	return strconv.Itoa(sec/60) + "m " + strconv.Itoa(sec%60) + "s"
}
