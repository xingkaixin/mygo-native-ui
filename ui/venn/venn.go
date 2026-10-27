// Package venn shows how two or three sets overlap, as an Euler diagram:
// a circle a set, as large as it, the circles overlapping as much as the
// sets share, two by two, and the count of each part in it. The parts of
// three sets can only come close to their counts. Hovering a part lights
// it and names it below, and a click chooses it, so that the app can list
// its items; a click outside lets go.
//
//	venn.View(c, &app.diagram, app.sets).Size(480, 360)
//	people := venn.Items(app.sets, app.diagram.Chosen)
package venn

import (
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/shader"
)

// Set is a set of items, by name, and the color its circle shows in.
type Set struct {
	Name  string
	Color ui.Color
	Items []string
}

// Region is a part of the diagram: the sets its items are in and no
// others, as bits, 1 for the first set, 2 for the second and 4 for the
// third. 0 is none.
type Region uint8

// In reports whether the items of r are in set i.
func (r Region) In(i int) bool { return r&(1<<i) != 0 }

// regions returns the region of each item of sets.
func regions(sets []Set) map[string]Region {
	of := map[string]Region{}
	for i, s := range sets {
		for _, it := range s.Items {
			of[it] |= 1 << i
		}
	}
	return of
}

// Items returns the items in r, sorted.
func Items(sets []Set, r Region) []string {
	var out []string
	for it, in := range regions(sets) {
		if in == r {
			out = append(out, it)
		}
	}
	slices.Sort(out)
	return out
}

// Describe returns r in words, as "Design and Research, not Engineering".
func Describe(sets []Set, r Region) string {
	var in, out []string
	for i, s := range sets {
		if r.In(i) {
			in = append(in, s.Name)
		} else {
			out = append(out, s.Name)
		}
	}
	switch {
	case len(in) == 0:
		return ""
	case len(out) == 0 && len(in) == 2:
		return "In both"
	case len(out) == 0 && len(in) == 3:
		return "In all three"
	case len(in) == 1:
		return "Only " + in[0]
	}
	return strings.Join(in, " and ") + ", not " + strings.Join(out, " or ")
}

// lens returns the area two circles of radii a and b share, d apart.
func lens(a, b, d float64) float64 {
	switch {
	case d >= a+b:
		return 0
	case d <= math.Abs(a-b):
		return math.Pi * min(a, b) * min(a, b)
	}
	x := a*a*math.Acos((d*d+a*a-b*b)/(2*d*a)) + b*b*math.Acos((d*d+b*b-a*a)/(2*d*b))
	return x - 0.5*math.Sqrt((-d+a+b)*(d+a-b)*(d-a+b)*(d+a+b))
}

// apart returns how far apart circles of radii a and b are to share
// area.
func apart(a, b, area float64) float64 {
	if area <= 0 {
		return (a + b) * 1.04
	}
	lo, hi := math.Abs(a-b), a+b
	for range 50 {
		mid := (lo + hi) / 2
		if lens(a, b, mid) > area {
			lo = mid
		} else {
			hi = mid
		}
	}
	return (lo + hi) / 2
}

// circle is a set's circle: its center and radius.
type circle struct{ x, y, r float64 }

func (c circle) has(x, y float64) bool { return math.Hypot(x-c.x, y-c.y) < c.r }

// layout returns the circles of sets, the first at 0, 0, and how many
// items each region holds.
func layout(sets []Set) ([]circle, [8]int) {
	var counts [8]int
	for _, r := range regions(sets) {
		counts[r]++
	}
	n := len(sets)
	size := make([]float64, n)
	cs := make([]circle, n)
	for i, s := range sets {
		size[i] = float64(len(s.Items))
		cs[i].r = math.Sqrt(max(size[i], 0.5) / math.Pi)
	}
	shared := func(i, j int) float64 {
		k := 0
		for r, c := range counts {
			if Region(r).In(i) && Region(r).In(j) {
				k += c
			}
		}
		return float64(k)
	}
	if n >= 2 {
		cs[1].x = apart(cs[0].r, cs[1].r, shared(0, 1))
	}
	if n == 3 {
		ab := cs[1].x
		ac := apart(cs[0].r, cs[2].r, shared(0, 2))
		bc := apart(cs[1].r, cs[2].r, shared(1, 2))
		if ab == 0 {
			cs[2].x = ac
		} else {
			cs[2].x = (ac*ac - bc*bc + ab*ab) / (2 * ab)
			cs[2].y = math.Sqrt(max(0, ac*ac-cs[2].x*cs[2].x))
		}
	}
	return cs, counts
}

// spots returns the point of each region furthest from the circles'
// edges, sampled on a grid over the circles, and whether it has one.
func spots(cs []circle) ([8][2]float64, [8]bool) {
	var at [8][2]float64
	var ok [8]bool
	var best [8]float64
	x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, c := range cs {
		x0, y0 = min(x0, c.x-c.r), min(y0, c.y-c.r)
		x1, y1 = max(x1, c.x+c.r), max(y1, c.y+c.r)
	}
	const steps = 72
	for j := range steps {
		for i := range steps {
			x := x0 + (x1-x0)*(float64(i)+0.5)/steps
			y := y0 + (y1-y0)*(float64(j)+0.5)/steps
			var r Region
			room := math.Inf(1)
			for k, c := range cs {
				d := math.Hypot(x-c.x, y-c.y)
				if d < c.r {
					r |= 1 << k
				}
				room = min(room, math.Abs(d-c.r))
			}
			if r != 0 && room > best[r] {
				best[r], at[r], ok[r] = room, [2]float64{x, y}, true
			}
		}
	}
	return at, ok
}

// State is what a diagram keeps: the region chosen. Keep it in the app's
// state, one for each view.
type State struct {
	// Chosen is the region chosen, which a click changes, 0 for none.
	Chosen Region
	held   bool
}

// View creates the diagram of two or three sets; others show nothing.
// Changed reports a new choice. Give it a size, or grow it in its parent.
func View(c *ui.Context, s *State, sets []Set) ui.Element {
	t := c.Theme()
	view := ui.Box(c).Role(ui.RoleGroup).Label("Venn diagram")
	if len(sets) < 2 || len(sets) > 3 {
		return view
	}
	cs, counts := layout(sets)
	b := view.LayoutBounds()
	x0, y0, x1, y1 := math.Inf(1), math.Inf(1), math.Inf(-1), math.Inf(-1)
	for _, c := range cs {
		x0, y0 = min(x0, c.x-c.r), min(y0, c.y-c.r)
		x1, y1 = max(x1, c.x+c.r), max(y1, c.y+c.r)
	}
	// Room above and below for the sets' names, and for the caption.
	label := float64(t.FontSize) * 2.2
	k := max(0.01, min((float64(b.W)-24)/(x1-x0), (float64(b.H)-3*label)/(y1-y0)))
	ox := float64(b.W)/2 - (x0+x1)/2*k
	oy := (float64(b.H)-label)/2 - (y0+y1)/2*k
	at := make([]circle, len(cs))
	for i, c := range cs {
		at[i] = circle{c.x*k + ox, c.y*k + oy, c.r * k}
	}
	var hover Region
	px, py, over := view.PointerPosition()
	if over {
		for i, c := range at {
			if c.has(float64(px), float64(py)) {
				hover |= 1 << i
			}
		}
	}
	if hover != 0 {
		view.Cursor(ui.CursorPointer)
	}
	if view.Pressed() && !s.held {
		if hover == s.Chosen {
			s.Chosen = 0
		} else {
			s.Chosen = hover
		}
		view.SetChanged()
	}
	s.held = view.Pressed()
	spot, has := spots(cs)
	chosen := s.Chosen
	caption := Describe(sets, hover)
	shown := hover
	if hover == 0 {
		caption, shown = Describe(sets, chosen), chosen
	}
	if caption != "" {
		caption = fmt.Sprintf("%s — %d", caption, counts[shown])
	}
	cy := 0.0
	for _, c := range at {
		cy += c.y / float64(len(at))
	}
	view.Draw(func(p *ui.Painter, r ui.Rect) {
		circ := make([]float32, 9)
		for i, c := range at {
			circ[3*i], circ[3*i+1], circ[3*i+2] = float32(c.x), float32(c.y), float32(c.r)
		}
		in := func(m Region) []float32 {
			v := make([]float32, 3)
			for i := range 3 {
				if m.In(i) {
					v[i] = 1
				}
			}
			return v
		}
		v := vennShader.NewValues().Floats(uCirc, circ...).
			Floats(uHoverIn, in(hover)...).Float(uHoverOn, bit(hover != 0 && hover != chosen)).
			Floats(uPickIn, in(chosen)...).Float(uPickOn, bit(chosen != 0)).
			Color(uHoverC, t.Text.Alpha(0.1)).Color(uPickC, t.Accent.Alpha(0.35))
		cols := make([]float32, 12)
		for i, s := range sets {
			col := s.Color.SRGB()
			cols[4*i], cols[4*i+1], cols[4*i+2], cols[4*i+3] = float32(col.R)/255, float32(col.G)/255, float32(col.B)/255, 1
		}
		v.Floats(uCol, cols...)
		vennShader.Draw(p, r, shader.Options{Values: v})
		for m := Region(1); m < 8; m++ {
			if !has[m] || counts[m] == 0 {
				continue
			}
			x, y := spot[m][0]*k+ox, spot[m][1]*k+oy
			text := fmt.Sprint(counts[m])
			span := ui.Span{Text: text, Size: t.FontSize + 2, Weight: 600, Color: t.Text, Features: "tnum"}
			w, h := p.MeasureText(0, span)
			p.RichText(r.X+float32(x)-w/2, r.Y+float32(y)-h/2, 0, span)
		}
		for i, c := range at {
			span := ui.Span{Text: fmt.Sprintf("%s  %d", sets[i].Name, len(sets[i].Items)), Weight: 600, Color: sets[i].Color}
			w, h := p.MeasureText(0, span)
			y := c.y - c.r - 6 - float64(h)
			if c.y > cy+1 {
				y = c.y + c.r + 6
			}
			p.RichText(r.X+float32(c.x)-w/2, r.Y+float32(y), 0, span)
		}
		if caption != "" {
			span := ui.Span{Text: caption, Color: t.TextMuted, Features: "tnum"}
			w, h := p.MeasureText(0, span)
			p.RichText(r.X+r.W/2-w/2, r.Y+r.H-h, 0, span)
		}
	})
	return view
}

func bit(b bool) float32 {
	if b {
		return 1
	}
	return 0
}

// vennShader fills the circles of the sets, each a light wash of its
// color, darker where they overlap, and the regions hovered and chosen,
// whose pixels it finds from the circles they are in and out of, with
// the edges smoothed over a pixel.
var vennShader = shader.MustCompile(`package main

// Circ are the circles, x, y and radius in DIPs; a radius of 0 is none.
var Circ [3]vec3
// Col are their colors, not premultiplied.
var Col [3]vec4
// HoverIn and PickIn are 1 for the circles the region hovered and the
// one chosen are in, 0 for those they are out of; HoverOn and PickOn
// whether there are such regions.
var HoverIn vec3
var HoverOn float
var PickIn vec3
var PickOn float
var HoverC vec4
var PickC vec4

func cover(c vec3, pos vec2) float {
	if c.z <= 0 {
		return 0
	}
	return clamp(0.5-(length(pos-c.xy)-c.z)*pixelScale(), 0, 1)
}

func ring(c vec3, pos vec2) float {
	if c.z <= 0 {
		return 0
	}
	d := abs(length(pos-c.xy)-c.z) * pixelScale()
	return clamp(0.75*pixelScale()+0.5-d, 0, 1)
}

func over(dst, src vec4) vec4 {
	return src + dst*(1-src.a)
}

func paint(c vec4, a float) vec4 {
	return vec4(c.rgb*c.a, c.a) * a
}

func Fragment(dst vec4, pos vec2) vec4 {
	a := vec3(cover(Circ[0], pos), cover(Circ[1], pos), cover(Circ[2], pos))
	out := vec4(0)
	out = over(out, paint(Col[0], 0.22*a.x))
	out = over(out, paint(Col[1], 0.22*a.y))
	out = over(out, paint(Col[2], 0.22*a.z))
	h := mix(1-a, a, HoverIn)
	out = over(out, paint(HoverC, h.x*h.y*h.z*HoverOn))
	k := mix(1-a, a, PickIn)
	out = over(out, paint(PickC, k.x*k.y*k.z*PickOn))
	out = over(out, paint(Col[0], ring(Circ[0], pos)))
	out = over(out, paint(Col[1], ring(Circ[1], pos)))
	out = over(out, paint(Col[2], ring(Circ[2], pos)))
	return out
}
`)

var (
	uCirc, uCol       = vennShader.Uniform("Circ"), vennShader.Uniform("Col")
	uHoverIn, uPickIn = vennShader.Uniform("HoverIn"), vennShader.Uniform("PickIn")
	uHoverOn, uPickOn = vennShader.Uniform("HoverOn"), vennShader.Uniform("PickOn")
	uHoverC, uPickC   = vennShader.Uniform("HoverC"), vennShader.Uniform("PickC")
)
