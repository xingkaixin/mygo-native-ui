// Package plasmid draws a circular plasmid map, as the sequence editors
// of molecular biology do: the backbone as a ring with the base pairs
// counted around it from the top, the features as arcs pointing the way
// their strand reads, outside the ring for the forward strand and inside
// for the reverse, and the enzymes' cut sites as ticks. Labels stand
// outside, pushed apart where they crowd, with lines back to what they
// name. Pointing at a feature tells its range in the middle; a click
// selects it, and a drag around the ring selects the bases it sweeps,
// shaded as a wedge.
//
//	plasmid.View(c, &app.mapView, app.plasmid, &app.selection)
//
// Changed reports a new selection.
package plasmid

import (
	"math"
	"slices"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// Plasmid is a circular sequence: its name, its length in base pairs, its
// features, and the sites where enzymes cut it.
type Plasmid struct {
	Name     string
	Length   int
	Features []Feature
	Sites    []Site
}

// Feature is a stretch of a plasmid: from Start up to End, counted from 0,
// running through 0 when End is before Start; Strand is 1 when it reads
// forward, -1 when reverse, and 0 when it does not read either way.
type Feature struct {
	Name       string
	Start, End int
	Strand     int
	Color      ui.Color
}

// Site is where an enzyme cuts a plasmid: before base At.
type Site struct {
	Name string
	At   int
}

// Range is a stretch of bases from Start up to End, running through 0
// when End is before Start, and none when they are equal.
type Range struct {
	Start, End int
}

// Len returns the bases in r on a plasmid length long.
func (r Range) Len(length int) int {
	return ((r.End-r.Start)%length + length) % length
}

// State is what a map keeps: the feature pointed at and the drag. Keep it
// in the app's state, one for each map.
type State struct {
	hover    int
	dragging bool
	from     int
}

// label is a name outside the ring, for the thing at angle a.
type label struct {
	text  string
	a     float64
	r     float32
	color ui.Color
	y     float32
	right bool
}

// View creates the map of p, selecting *sel.
func View(c *ui.Context, s *State, p Plasmid, sel *Range) ui.Element {
	t := c.Theme()
	changed := false
	n := max(1, p.Length)
	angle := func(bp float64) float64 { return bp/float64(n)*2*math.Pi - math.Pi/2 }
	view := ui.Box(c).Height(t.Space(130)).Role(ui.RoleImage).Label(p.Name + ", " + strconv.Itoa(n) + " bp").Cursor(ui.CursorCrosshair)
	b := view.LayoutBounds()
	cx, cy := b.W/2, b.H/2
	ring := min(b.W, b.H)/2 - t.Space(26)
	track := float32(12)
	bpAt := func(x, y float32) (int, float32) {
		dx, dy := float64(x-cx), float64(y-cy)
		a := math.Atan2(dy, dx) + math.Pi/2
		if a < 0 {
			a += 2 * math.Pi
		}
		return int(a/(2*math.Pi)*float64(n)) % n, float32(math.Hypot(dx, dy))
	}
	inside := func(f Feature, bp int) bool {
		if f.End >= f.Start {
			return bp >= f.Start && bp < f.End
		}
		return bp >= f.Start || bp < f.End
	}
	radius := func(f Feature) float32 {
		switch f.Strand {
		case 1:
			return ring + track/2 + 3
		case -1:
			return ring - track/2 - 3
		}
		return ring
	}
	s.hover = -1
	x, y, in := view.PointerPosition()
	bp, d := bpAt(x, y)
	if in {
		for i, f := range p.Features {
			if r := radius(f); math.Abs(float64(d-r)) <= float64(track)/2+2 && inside(f, bp) {
				s.hover = i
			}
		}
	}
	switch {
	case !view.Pressed():
		s.dragging = false
	case !s.dragging:
		s.dragging, s.from = true, bp
	default:
		if r := (Range{s.from, bp}); r != *sel && r.Len(n) > 0 {
			*sel, changed = r, true
		}
	}
	if view.Clicked() {
		switch {
		case s.hover >= 0:
			f := p.Features[s.hover]
			*sel, changed = Range{f.Start, f.End}, true
		case d < ring-track && sel.Len(n) > 0:
			*sel, changed = Range{}, true
		}
	}
	hover, selection := s.hover, *sel
	view.Draw(func(pt *ui.Painter, r ui.Rect) {
		cx, cy := r.X+r.W/2, r.Y+r.H/2
		seg := func(x0, y0, x1, y1, w float32, col ui.Color) {
			var l ui.Path
			l.MoveTo(x0, y0)
			l.LineTo(x1, y1)
			pt.StrokePath(&l, w, col)
		}
		polar := func(a float64, rad float32) (float32, float32) {
			return cx + rad*float32(math.Cos(a)), cy + rad*float32(math.Sin(a))
		}
		arc := func(path *ui.Path, a0, a1 float64, rad float32, move bool) {
			steps := max(2, int(math.Abs(a1-a0)/0.02))
			for k := 0; k <= steps; k++ {
				x, y := polar(a0+(a1-a0)*float64(k)/float64(steps), rad)
				if k == 0 && move {
					path.MoveTo(x, y)
				} else {
					path.LineTo(x, y)
				}
			}
		}
		if selection.Len(n) > 0 {
			a0 := angle(float64(selection.Start))
			a1 := a0 + float64(selection.Len(n))/float64(n)*2*math.Pi
			var wedge ui.Path
			wedge.MoveTo(cx, cy)
			arc(&wedge, a0, a1, ring+track+6, false)
			wedge.Close()
			pt.FillPath(&wedge, t.Accent.Alpha(0.14))
			for _, a := range []float64{a0, a1} {
				x, y := polar(a, ring+track+6)
				seg(cx, cy, x, y, 1, t.Accent.Alpha(0.6))
			}
		}
		var backbone ui.Path
		backbone.Circle(cx, cy, ring+1.5)
		pt.StrokePath(&backbone, 1, t.Text.Alpha(0.7))
		backbone.Reset()
		backbone.Circle(cx, cy, ring-1.5)
		pt.StrokePath(&backbone, 1, t.Text.Alpha(0.7))
		step := tickStep(n)
		for bp := 0; bp < n; bp += step {
			a := angle(float64(bp))
			x0, y0 := polar(a, ring-track-8)
			x1, y1 := polar(a, ring-track-13)
			seg(x0, y0, x1, y1, 1, t.TextMuted.Alpha(0.6))
			if bp == 0 {
				continue
			}
			text := thousands(bp)
			w, h := pt.MeasureText(0, ui.Span{Text: text, Size: t.FontSize - 4})
			// The label's box keeps clear of the tick whichever way it faces.
			inset := float32(math.Abs(math.Cos(a)))*w/2 + float32(math.Abs(math.Sin(a)))*h/2
			lx, ly := polar(a, ring-track-17-inset)
			pt.Text(lx-w/2, ly-h/2, text, t.FontSize-4, t.TextMuted)
		}
		var labels []label
		for i, f := range p.Features {
			length := Range{f.Start, f.End}.Len(n)
			a0 := angle(float64(f.Start))
			a1 := a0 + float64(length)/float64(n)*2*math.Pi
			rad := radius(f)
			head := math.Min(float64(track)/float64(rad), (a1-a0)*0.5)
			var shape ui.Path
			switch f.Strand {
			case 1:
				arc(&shape, a0, a1-head, rad+track/2, true)
				x, y := polar(a1, rad)
				shape.LineTo(x, y)
				arc(&shape, a1-head, a0, rad-track/2, false)
			case -1:
				x, y := polar(a0, rad)
				shape.MoveTo(x, y)
				arc(&shape, a0+head, a1, rad+track/2, false)
				arc(&shape, a1, a0+head, rad-track/2, false)
			default:
				arc(&shape, a0, a1, rad+track/2, true)
				arc(&shape, a1, a0, rad-track/2, false)
			}
			shape.Close()
			col := f.Color
			if hover >= 0 && hover != i {
				col = col.Alpha(0.45)
			}
			pt.FillPath(&shape, col)
			if hover == i {
				pt.StrokePath(&shape, 1.5, t.Text)
			}
			labels = append(labels, label{text: f.Name, a: (a0 + a1) / 2, r: rad + track/2, color: t.Text})
		}
		for _, st := range p.Sites {
			a := angle(float64(st.At))
			x0, y0 := polar(a, ring-4)
			x1, y1 := polar(a, ring+track+8)
			seg(x0, y0, x1, y1, 1, t.Text.Alpha(0.6))
			labels = append(labels, label{text: st.Name + " (" + thousands(st.At) + ")", a: a, r: ring + track + 8, color: t.TextMuted})
		}
		place(labels, cx, cy, ring+track+t.Space(10), t.FontSize+1, func(l label) {
			ax, ay := polar(l.a, l.r)
			ex := cx + (ring+track+t.Space(10))*float32(math.Cos(l.a))
			size := t.FontSize - 2
			w, _ := pt.MeasureText(0, ui.Span{Text: l.text, Size: size})
			tx := ex + 4
			if !l.right {
				tx = ex - 4 - w
			}
			seg(ax, ay, ex, l.y, 1, t.TextMuted.Alpha(0.5))
			pt.Text(tx, l.y-size/2-2, l.text, size, l.color)
		})
	})
	view.Center().Children(func() {
		ui.Column(c).Gap(t.Space(0.5)).AlignItems(ui.Center).PassThrough().Children(func() {
			line := func(text string, big bool, col ui.Color) {
				tx := ui.Text(c, text).TextColor(col).PassThrough()
				if big {
					tx.FontSize(t.FontSize + 4).Bold()
				} else {
					tx.FontSize(t.FontSize - 1).FontFeatures("tnum")
				}
			}
			span := func(r Range) string {
				return thousands(r.Start+1) + " – " + thousands((r.End-1+n)%n+1)
			}
			switch {
			case hover >= 0:
				f := p.Features[hover]
				line(f.Name, true, t.Text)
				line(span(Range{f.Start, f.End}), false, t.TextMuted)
				line(thousands(Range{f.Start, f.End}.Len(n))+" bp", false, t.TextMuted)
			case selection.Len(n) > 0:
				line(thousands(selection.Len(n))+" bp", true, t.Accent)
				line(span(selection), false, t.TextMuted)
			default:
				line(p.Name, true, t.Text)
				line(thousands(n)+" bp", false, t.TextMuted)
			}
		})
	})
	if changed {
		view.SetChanged()
	}
	return view
}

// place sets each label beside the ring at radius out, on the side its
// angle faces, pushing those that would overlap apart by gap, and draws
// it.
func place(labels []label, cx, cy, out, gap float32, draw func(label)) {
	for _, right := range []bool{true, false} {
		var side []label
		for _, l := range labels {
			if (math.Cos(l.a) >= 0) == right {
				l.right = right
				l.y = cy + out*float32(math.Sin(l.a))
				side = append(side, l)
			}
		}
		slices.SortFunc(side, func(a, b label) int {
			switch {
			case a.y < b.y:
				return -1
			case a.y > b.y:
				return 1
			}
			return 0
		})
		for i := 1; i < len(side); i++ {
			side[i].y = max(side[i].y, side[i-1].y+gap)
		}
		// Labels pushed past the bottom come back up, the whole run with them.
		if k := len(side) - 1; k >= 0 {
			if over := side[k].y - (cy + out + gap); over > 0 {
				side[k].y -= over
				for i := k - 1; i >= 0; i-- {
					side[i].y = min(side[i].y, side[i+1].y-gap)
				}
			}
		}
		for _, l := range side {
			draw(l)
		}
	}
}

// tickStep returns the base pairs between ticks, about twelve around.
func tickStep(n int) int {
	for m := 1; ; m *= 10 {
		for _, s := range []int{1, 2, 5} {
			if s*m*12 >= n {
				return s * m
			}
		}
	}
}

// thousands writes n with commas between its thousands.
func thousands(n int) string {
	s := strconv.Itoa(n)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}
