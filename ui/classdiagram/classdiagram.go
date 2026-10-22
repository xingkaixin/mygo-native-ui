// Package classdiagram draws a UML class diagram from text, as Mermaid's
// classDiagram does: each class a box of three parts, its name under its
// annotation, its fields and its methods, a static member underlined and
// an abstract one in italics; and the relations between classes as lines
// that end as UML has them, a hollow triangle for inheritance and
// realization, a filled diamond for composition, a hollow one for
// aggregation and an open arrowhead for association and dependency,
// dashed for dependency and realization, with cardinalities by their ends
// and labels on them. The classes are laid out in ranks down the diagram,
// or across it, a parent above the classes that inherit from it and a
// whole above its parts, and ordered so that few lines cross. A click
// chooses a class, which Chosen returns, and lights it and its relations.
// A diagram too big to fit scrolls both ways. The lines that are wrong
// are listed under it, by number with what is wrong, and the diagram
// shows what the others say.
//
//	classdiagram.View(c, &app.diagram, app.source).Grow(1)
//	if cl, ok := app.diagram.Chosen(); ok {
//		... // show cl.Name and its members
//	}
//
// The source has a statement a line; lines starting with %% are
// comments:
//
//	classDiagram           the header
//	direction LR           the way the ranks go: TB down, BT up, LR right, RL left
//	class A                a class
//	class A~T~             one with a type parameter, shown as A<T>
//	class A { ... }        one with the members on the lines up to }, a line each
//	A : +member            a member of A
//	<<interface>> A        an annotation, also on a line of its own in A's body
//	A <|-- B               B inherits from A
//	A *-- B, A o-- B       A is composed of B, aggregates B
//	A --> B, A -- B        A is associated with B, linked to it
//	A ..> B, A ..|> B      A depends on B, realizes it
//	A .. B                 a dashed link
//	A "1" --> "*" B        cardinalities, by either end
//	A --> B : text         a label
//
// Each relation reads the other way too, as B --|> A, B <-- A or
// B <|.. A. A member with parentheses is a method, as +pay(amount Money)
// Receipt, and any other a field, as +total: Money or Money total. It
// starts with its visibility, + public, - private, # protected or ~
// package, and ends in $ when it is static and * when it is abstract.
// Lines of style, classDef, cssClass, click, link and callback are
// skipped.
package classdiagram

import (
	"fmt"
	"math"

	"github.com/egoist/mygo/ui"
)

// Class is a class of a diagram: its id, as relations name it; its name
// as it shows, with its type parameters, as Page<T>; its annotation, as
// interface, "" for none; and its fields and methods as they show.
type Class struct {
	ID, Name, Annotation string
	Fields, Methods      []string
}

// State is what a view keeps: the diagram as it parsed and laid out the
// source it was last given, and the class chosen. Keep it in the app's
// state, one for each view.
type State struct {
	key key
	d   *diagram
	// chosen is the id of the class chosen, "" for none.
	chosen string
}

// Chosen returns the class chosen with a click, while the diagram has it.
func (s *State) Chosen() (Class, bool) {
	if s.d == nil {
		return Class{}, false
	}
	i := s.d.find(s.chosen)
	if i < 0 {
		return Class{}, false
	}
	cl := s.d.classes[i]
	out := Class{ID: cl.id, Name: cl.name, Annotation: cl.note}
	for _, m := range cl.fields {
		out.Fields = append(out.Fields, m.head+m.tail)
	}
	for _, m := range cl.methods {
		out.Methods = append(out.Methods, m.head+m.tail)
	}
	return out, true
}

// key is what a layout is for: the source, and the theme's sizes.
type key struct {
	src         string
	font, space float32
}

type class struct {
	id, name, note  string
	fields, methods []member
	// loops are its relations to itself.
	loops []int
	// head is how tall the part of its name is, fh and mh those of its
	// fields and of its methods; w and h are its size, x and y its center.
	head, fh, mh, w, h, x, y float32
}

type relation struct {
	from, to int
	// ends and cards are what it ends in and its cardinalities, at from
	// and at to.
	ends   [2]end
	dashed bool
	cards  [2]string
	label  string
	line   int
	// pts are its line, from where it leaves from to where it meets to;
	// its label is lw by lh centered at lx, ly, and its cardinalities are
	// cw wide, centered at at.
	pts            [][2]float32
	lw, lh, lx, ly float32
	cw             [2]float32
	at             [2][2]float32
}

type diagram struct {
	dir     direction
	classes []class
	rels    []relation
	errs    []problem
	w, h    float32
	z       sizes
}

// find returns the class of id, -1 for none.
func (d *diagram) find(id string) int {
	for i, cl := range d.classes {
		if id != "" && cl.id == id {
			return i
		}
	}
	return -1
}

// at returns the class at x, y, -1 for none.
func (d *diagram) at(x, y float32) int {
	for i, cl := range d.classes {
		if abs(x-cl.x) <= cl.w/2 && abs(y-cl.y) <= cl.h/2 {
			return i
		}
	}
	return -1
}

// View creates the diagram that src describes. Give it a size, or Grow;
// it scrolls when it does not fit, and the lines that are wrong show
// under it.
func View(c *ui.Context, s *State, src string) ui.Element {
	t := c.Theme()
	k := key{src, t.FontSize, t.Space(1)}
	if s.d == nil || s.key != k {
		s.key, s.d = k, parse(src)
		layout(c, s.d)
	}
	d := s.d
	return ui.Column(c).Children(func() {
		ui.ScrollBoth(c).Grow(1).Shrink(1).Children(func() {
			box := ui.Box(c).Size(d.w, d.h)
			hot := -1
			if px, py, ok := box.PointerPosition(); ok {
				hot = d.at(px, py)
			}
			if hot >= 0 {
				box.Cursor(ui.CursorPointer)
			}
			if box.Clicked() {
				s.chosen = ""
				if hot >= 0 {
					s.chosen = d.classes[hot].id
				}
			}
			chosen := d.find(s.chosen)
			box.Draw(func(p *ui.Painter, r ui.Rect) {
				draw(p, t, d, r.X, r.Y, chosen)
			})
		})
		if len(d.errs) == 0 {
			return
		}
		ui.Column(c).Gap(t.Space(1)).Padding(t.Space(2), t.Space(3)).Children(func() {
			for i, e := range d.errs {
				ui.Text(c.Key(i), fmt.Sprintf("Line %d: %s", e.line, e.msg)).FontSize(t.FontSize - 1).TextColor(t.Danger)
			}
		})
	})
}

// spans returns the text of m as it shows: its head in ink and its type
// in muted, underlined when it is static and slanted when it is
// abstract.
func spans(m member, size float32, ink, muted ui.Color) []ui.Span {
	out := []ui.Span{{Text: m.head, Size: size, Color: ink, Italic: m.abstract, Underline: m.static}}
	if m.tail != "" {
		out = append(out, ui.Span{Text: m.tail, Size: size, Color: muted, Italic: m.abstract, Underline: m.static})
	}
	return out
}

// draw paints d with its left at ox and top at oy, lighting class chosen
// and its relations.
func draw(p *ui.Painter, t *ui.Theme, d *diagram, ox, oy float32, chosen int) {
	z := d.z
	lit := func(r relation) bool { return chosen >= 0 && (r.from == chosen || r.to == chosen) }
	for _, on := range []bool{false, true} {
		for _, r := range d.rels {
			if lit(r) != on {
				continue
			}
			// Opaque, so that a line under a marker or another line does
			// not show darker.
			color, width := t.Background.Mix(t.Text, 0.5), float32(1.25)
			if on {
				color, width = t.Accent, 2
			}
			d.drawRelation(p, t, r, ox, oy, color, width)
		}
	}
	for _, r := range d.rels {
		ink := t.TextMuted
		if lit(r) {
			ink = t.Accent
		}
		for j, card := range r.cards {
			if card == "" {
				continue
			}
			x, y := ox+r.at[j][0]-r.cw[j]/2, oy+r.at[j][1]-z.sh/2
			p.Fill(ui.Rect{X: x - 1, Y: y, W: r.cw[j] + 2, H: z.sh}, t.Background, 0)
			p.RichText(x, y, 0, ui.Span{Text: card, Size: t.FontSize - 1, Color: ink})
		}
		if r.label == "" || r.lw == 0 {
			continue
		}
		box := ui.Rect{X: ox + r.lx - r.lw/2, Y: oy + r.ly - r.lh/2, W: r.lw, H: r.lh}
		p.Fill(box, t.Background, t.Space(1.5))
		p.RichText(box.X+t.Space(1.5), oy+r.ly-z.sh/2, 0, ui.Span{Text: r.label, Size: t.FontSize - 1, Color: ink})
	}
	for i, cl := range d.classes {
		drawClass(p, t, z, cl, ox, oy, i == chosen)
	}
}

// drawClass draws cl as a box of three parts: its name under its
// annotation, its fields and its methods.
func drawClass(p *ui.Painter, t *ui.Theme, z sizes, cl class, ox, oy float32, chosen bool) {
	r := ui.Rect{X: ox + cl.x - cl.w/2, Y: oy + cl.y - cl.h/2, W: cl.w, H: cl.h}
	fill, head, border, width := t.Surface, t.Text.Alpha(0.05), t.Border, float32(1)
	if chosen {
		fill, head, border, width = t.Surface.Mix(t.Accent, 0.08), t.Accent.Alpha(0.14), t.Accent, 2
	}
	p.Fill(r, t.Background, z.radius)
	p.Fill(r, fill, z.radius)
	p.Clip(r, z.radius, func() {
		p.Fill(ui.Rect{X: r.X, Y: r.Y, W: r.W, H: cl.head}, head, 0)
	})
	for _, y := range []float32{r.Y + cl.head, r.Y + cl.head + cl.fh} {
		p.Line(r.X, y, r.X+r.W, y, 1, border)
	}
	p.Stroke(r, border, z.radius, width)

	y := r.Y + z.padY
	center := func(s ui.Span, h float32) {
		w, _ := p.MeasureText(0, s)
		p.RichText(r.X+(r.W-w)/2, y, 0, s)
		y += h
	}
	if cl.note != "" {
		center(ui.Span{Text: "«" + cl.note + "»", Size: t.FontSize - 1, Color: t.TextMuted}, z.sh)
	}
	center(ui.Span{Text: cl.name, Size: t.FontSize, Weight: 600, Color: t.Text}, z.th)
	rows := func(top float32, ms []member) {
		for i, m := range ms {
			p.RichText(r.X+z.padX, top+z.inset+float32(i)*z.row+(z.row-z.sh)/2, 0, spans(m, t.FontSize-1, t.Text, t.TextMuted)...)
		}
	}
	rows(r.Y+cl.head, cl.fields)
	rows(r.Y+cl.head+cl.fh, cl.methods)
}

// drawRelation draws r moved by ox, oy, with what it ends in.
func (d *diagram) drawRelation(p *ui.Painter, t *ui.Theme, r relation, ox, oy float32, color ui.Color, width float32) {
	z := d.z
	if len(r.pts) < 2 {
		return
	}
	pts := make([][2]float32, len(r.pts))
	for i, pt := range r.pts {
		pts[i] = [2]float32{ox + pt[0], oy + pt[1]}
	}
	back := func(e end) float32 {
		switch e {
		case triangle:
			return z.tri
		case filled, hollow:
			return z.dia
		}
		return 0
	}
	body := cut(pts, back(r.ends[0]), back(r.ends[1]))
	if r.dashed {
		dashes(p, body, t.Space(1.5), t.Space(1), width, color)
	} else if len(body) > 1 {
		p.StrokePolyline(body, width, color)
	}
	for j, e := range r.ends {
		if e == bare {
			continue
		}
		end := pts
		if j == 0 {
			end = reversed(pts)
		}
		tip := end[len(end)-1]
		from := point(end, max(z.tri, z.dia))
		dx, dy := tip[0]-from[0], tip[1]-from[1]
		if l := float32(math.Hypot(float64(dx), float64(dy))); l > 0 {
			dx, dy = dx/l, dy/l
		} else {
			dx, dy = 0, 1
		}
		// at returns the point a back from the tip and b to its side.
		at := func(a, b float32) (float32, float32) { return tip[0] - dx*a - dy*b, tip[1] - dy*a + dx*b }
		var path ui.Path
		switch e {
		case arrow:
			l, w := z.arrow, z.arrow*0.55
			x0, y0 := at(l, w)
			x1, y1 := at(l, -w)
			p.StrokePolyline([][2]float32{{x0, y0}, tip, {x1, y1}}, width, color)
			continue
		case triangle:
			l, w := z.tri, z.tri*0.6
			x0, y0 := at(l, w)
			x1, y1 := at(l, -w)
			path.MoveTo(tip[0], tip[1]).LineTo(x0, y0).LineTo(x1, y1).Close()
		case filled, hollow:
			l, w := z.dia, z.dia*0.3
			x0, y0 := at(l/2, w)
			x1, y1 := at(l, 0)
			x2, y2 := at(l/2, -w)
			path.MoveTo(tip[0], tip[1]).LineTo(x0, y0).LineTo(x1, y1).LineTo(x2, y2).Close()
		}
		if e == filled {
			p.FillPath(&path, color)
		} else {
			p.FillPath(&path, t.Background)
		}
		p.StrokePath(&path, width, color)
	}
}

func reversed(pts [][2]float32) [][2]float32 {
	out := make([][2]float32, len(pts))
	for i, pt := range pts {
		out[len(pts)-1-i] = pt
	}
	return out
}

func dist(a, b [2]float32) float32 {
	return float32(math.Hypot(float64(b[0]-a[0]), float64(b[1]-a[1])))
}

func lerp(a, b [2]float32, f float32) [2]float32 {
	return [2]float32{a[0] + (b[0]-a[0])*f, a[1] + (b[1]-a[1])*f}
}

// point returns the point d along pts back from its end.
func point(pts [][2]float32, d float32) [2]float32 {
	for i := len(pts) - 1; i > 0; i-- {
		l := dist(pts[i-1], pts[i])
		if l >= d && l > 0 {
			return lerp(pts[i], pts[i-1], d/l)
		}
		d -= l
	}
	return pts[0]
}

// cut returns pts without d0 of its length at the start and d1 at the
// end.
func cut(pts [][2]float32, d0, d1 float32) [][2]float32 {
	start := func(pts [][2]float32, d float32) [][2]float32 {
		for i := 1; i < len(pts); i++ {
			l := dist(pts[i-1], pts[i])
			if l > d {
				return append([][2]float32{lerp(pts[i-1], pts[i], d/l)}, pts[i:]...)
			}
			d -= l
		}
		return nil
	}
	pts = start(pts, d0)
	if len(pts) < 2 {
		return nil
	}
	return reversed(start(reversed(pts), d1))
}

// dashes draws the line through pts in dashes on long, off apart.
func dashes(p *ui.Painter, pts [][2]float32, on, off, width float32, color ui.Color) {
	if len(pts) < 2 {
		return
	}
	dash := [][2]float32{pts[0]}
	drawing, left := true, on
	for i := 1; i < len(pts); i++ {
		a, b := pts[i-1], pts[i]
		l, at := dist(a, b), float32(0)
		for l-at > left {
			at += left
			q := lerp(a, b, at/l)
			if drawing {
				p.StrokePolyline(append(dash, q), width, color)
				left = off
			} else {
				dash = [][2]float32{q}
				left = on
			}
			drawing = !drawing
		}
		left -= l - at
		if drawing {
			dash = append(dash, b)
		}
	}
	if drawing && len(dash) > 1 {
		p.StrokePolyline(dash, width, color)
	}
}

func abs(v float32) float32 { return max(v, -v) }
