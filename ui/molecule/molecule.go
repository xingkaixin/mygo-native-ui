// Package molecule shows a molecule in 3D as balls and sticks, as
// chemistry viewers do: each atom a shaded ball in its element's color,
// each bond a stick, two or three for double and triple bonds, the
// nearer drawn over the farther. Dragging turns the molecule, and the
// wheel or a pinch zooms. Clicking two atoms measures how far apart they
// are, three the angle at the middle one; clicking empty space clears
// them.
//
//	m, err := molecule.ParseXYZ(file)
//	molecule.View(c, &app.viewer, m).Grow(1)
//
// Coordinates are in ångströms.
package molecule

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Atom is an atom: its element's symbol, as "C", and where it is.
type Atom struct {
	Element string
	X, Y, Z float64
}

// Bond joins atoms A and B, by index, Order times: 1, 2 or 3.
type Bond struct {
	A, B  int
	Order int
}

// Molecule is atoms and the bonds between them.
type Molecule struct {
	Atoms []Atom
	Bonds []Bond
}

// covalent are the covalent radii of the elements, in ångströms.
var covalent = map[string]float64{
	"H": 0.31, "C": 0.76, "N": 0.71, "O": 0.66, "F": 0.57, "P": 1.07, "S": 1.05,
	"Cl": 1.02, "Br": 1.20, "I": 1.39, "B": 0.84, "Si": 1.11, "Na": 1.66, "Fe": 1.32,
}

func radius(el string) float64 {
	if r, ok := covalent[el]; ok {
		return r
	}
	return 1.2
}

// Connect returns single bonds between the atoms closer than their
// covalent radii together, and a little more.
func Connect(atoms []Atom) []Bond {
	var out []Bond
	for i := range atoms {
		for j := i + 1; j < len(atoms); j++ {
			a, b := atoms[i], atoms[j]
			if math.Sqrt(sq(a.X-b.X)+sq(a.Y-b.Y)+sq(a.Z-b.Z)) < radius(a.Element)+radius(b.Element)+0.4 {
				out = append(out, Bond{A: i, B: j, Order: 1})
			}
		}
	}
	return out
}

func sq(v float64) float64 { return v * v }

// ParseXYZ returns the molecule of an XYZ file, its bonds joined as
// Connect joins them: a line with the number of atoms, a comment, then a
// line for each atom with its element and coordinates.
func ParseXYZ(src string) (Molecule, error) {
	lines := strings.Split(strings.TrimSpace(src), "\n")
	n, err := strconv.Atoi(strings.TrimSpace(lines[0]))
	if err != nil {
		return Molecule{}, fmt.Errorf("xyz: atom count %q", lines[0])
	}
	if len(lines) < n+2 {
		return Molecule{}, fmt.Errorf("xyz: %d atoms, %d lines", n, len(lines)-2)
	}
	atoms := make([]Atom, n)
	for i, line := range lines[2 : n+2] {
		f := strings.Fields(line)
		if len(f) < 4 {
			return Molecule{}, fmt.Errorf("xyz: line %d: %q", i+3, line)
		}
		var xyz [3]float64
		for k := range 3 {
			if xyz[k], err = strconv.ParseFloat(f[k+1], 64); err != nil {
				return Molecule{}, fmt.Errorf("xyz: line %d: %w", i+3, err)
			}
		}
		atoms[i] = Atom{Element: f[0], X: xyz[0], Y: xyz[1], Z: xyz[2]}
	}
	return Molecule{Atoms: atoms, Bonds: Connect(atoms)}, nil
}

// colors are the elements' colors, as CPK has them.
var colors = map[string]ui.Color{
	"H": ui.RGB(240, 240, 240), "C": ui.RGB(80, 80, 88), "N": ui.RGB(48, 80, 248), "O": ui.RGB(230, 40, 30),
	"F": ui.RGB(144, 224, 80), "Cl": ui.RGB(31, 200, 31), "Br": ui.RGB(166, 41, 41), "I": ui.RGB(148, 0, 148),
	"P": ui.RGB(255, 128, 0), "S": ui.RGB(240, 200, 40), "B": ui.RGB(255, 181, 181), "Fe": ui.RGB(224, 102, 51),
}

func color(el string) ui.Color {
	if c, ok := colors[el]; ok {
		return c
	}
	return ui.RGB(221, 119, 255)
}

// State is what a viewer keeps: how the molecule is turned and zoomed,
// and the atoms picked. Keep it in the app's state, one for each view.
type State struct {
	rot    [9]float64
	zoom   float64
	picked []int
	ready  bool
	held   bool
	moved  float32
	// atoms are how many atoms the picks were made among, which another
	// molecule clears.
	atoms int
}

func (s *State) turn(ax, ay float64) {
	cx, sx, cy, sy := math.Cos(ax), math.Sin(ax), math.Cos(ay), math.Sin(ay)
	// Turning about the screen's axes multiplies on the left, so that a
	// drag turns the molecule as it shows, however turned it is already.
	rx := [9]float64{1, 0, 0, 0, cx, -sx, 0, sx, cx}
	ry := [9]float64{cy, 0, sy, 0, 1, 0, -sy, 0, cy}
	s.rot = mul(rx, mul(ry, s.rot))
}

func mul(a, b [9]float64) [9]float64 {
	var m [9]float64
	for i := range 3 {
		for j := range 3 {
			for k := range 3 {
				m[3*i+j] += a[3*i+k] * b[3*k+j]
			}
		}
	}
	return m
}

// spot is an atom as it shows: where on screen, how near, and how large.
type spot struct {
	x, y, z, r float64
}

// View creates the viewer of m. Give it a size, or grow it in its parent.
func View(c *ui.Context, s *State, m Molecule) ui.Element {
	t := c.Theme()
	view := ui.Box(c).Role(ui.RoleImage).Label("Molecule").Cursor(ui.CursorGrab)
	if !s.ready {
		s.rot, s.zoom, s.ready = [9]float64{1, 0, 0, 0, 1, 0, 0, 0, 1}, 1, true
		s.turn(-0.35, 0.5)
	}
	if s.atoms != len(m.Atoms) {
		s.picked, s.atoms = nil, len(m.Atoms)
	}
	if len(m.Atoms) == 0 {
		return view
	}
	view.HandleInput(func(in ui.InputEvent) bool {
		switch in.Kind {
		case ui.InputScroll:
			s.zoom = max(0.3, min(4, s.zoom*math.Exp2(-float64(in.DY)/300)))
		case ui.InputZoom:
			s.zoom = max(0.3, min(4, s.zoom*float64(in.Scale)))
		default:
			return false
		}
		return true
	})
	if dx, dy, ok := view.Dragged(); ok {
		s.turn(float64(dy)*0.01, float64(dx)*0.01)
		s.moved += float32(math.Hypot(float64(dx), float64(dy)))
		view.Cursor(ui.CursorGrabbing)
	}
	var cx, cy, cz float64
	for _, a := range m.Atoms {
		cx, cy, cz = cx+a.X, cy+a.Y, cz+a.Z
	}
	n := float64(len(m.Atoms))
	cx, cy, cz = cx/n, cy/n, cz/n
	reach := 1.0
	for _, a := range m.Atoms {
		reach = max(reach, math.Sqrt(sq(a.X-cx)+sq(a.Y-cy)+sq(a.Z-cz))+0.6)
	}
	b := view.LayoutBounds()
	k := float64(min(b.W, b.H)) / 2 / reach * s.zoom
	// The eye stands back as far as four molecules, for a little
	// perspective.
	eye := reach * 4
	spots := make([]spot, len(m.Atoms))
	for i, a := range m.Atoms {
		x, y, z := a.X-cx, a.Y-cy, a.Z-cz
		r := s.rot
		px, py, pz := r[0]*x+r[1]*y+r[2]*z, r[3]*x+r[4]*y+r[5]*z, r[6]*x+r[7]*y+r[8]*z
		f := eye / (eye - pz)
		size := 0.32
		if a.Element == "H" {
			size = 0.22
		}
		spots[i] = spot{float64(b.W)/2 + px*k*f, float64(b.H)/2 - py*k*f, pz, size * k * f}
	}
	hover := -1
	px, py, over := view.PointerPosition()
	if over {
		for i, sp := range spots {
			if math.Hypot(float64(px)-sp.x, float64(py)-sp.y) < sp.r+2 && (hover < 0 || sp.z > spots[hover].z) {
				hover = i
			}
		}
	}
	if hover >= 0 && !view.Pressed() {
		view.Cursor(ui.CursorPointer)
	}
	if view.Pressed() && !s.held {
		s.moved = 0
	}
	if s.held && !view.Pressed() && s.moved < 4 {
		switch {
		case hover < 0:
			s.picked = nil
		case len(s.picked) == 3 || slices.Contains(s.picked, hover):
			s.picked = []int{hover}
		default:
			s.picked = append(s.picked, hover)
		}
	}
	s.held = view.Pressed()
	picked := slices.Clone(s.picked)
	label := func(i int) string { return fmt.Sprintf("%s%d", m.Atoms[i].Element, i+1) }
	measure := ""
	switch len(picked) {
	case 2:
		a, b := m.Atoms[picked[0]], m.Atoms[picked[1]]
		measure = fmt.Sprintf("%s–%s  %.3f Å", label(picked[0]), label(picked[1]), math.Sqrt(sq(a.X-b.X)+sq(a.Y-b.Y)+sq(a.Z-b.Z)))
	case 3:
		a, o, b := m.Atoms[picked[0]], m.Atoms[picked[1]], m.Atoms[picked[2]]
		ux, uy, uz := a.X-o.X, a.Y-o.Y, a.Z-o.Z
		vx, vy, vz := b.X-o.X, b.Y-o.Y, b.Z-o.Z
		cos := (ux*vx + uy*vy + uz*vz) / math.Sqrt((ux*ux+uy*uy+uz*uz)*(vx*vx+vy*vy+vz*vz))
		measure = fmt.Sprintf("%s–%s–%s  %.1f°", label(picked[0]), label(picked[1]), label(picked[2]), math.Acos(max(-1, min(1, cos)))*180/math.Pi)
	}
	tip := ""
	if hover >= 0 {
		tip = label(hover)
	}
	view.Draw(func(p *ui.Painter, r ui.Rect) {
		// Atoms and bonds draw from the farthest to the nearest, a bond
		// as near as its middle.
		type item struct {
			z    float64
			atom int
			bond int
		}
		items := make([]item, 0, len(spots)+len(m.Bonds))
		for i, sp := range spots {
			items = append(items, item{sp.z, i, -1})
		}
		for i, bd := range m.Bonds {
			items = append(items, item{(spots[bd.A].z + spots[bd.B].z) / 2, -1, i})
		}
		slices.SortStableFunc(items, func(a, b item) int {
			switch {
			case a.z < b.z:
				return -1
			case a.z > b.z:
				return 1
			}
			return 0
		})
		at := func(sp spot) (float32, float32) { return r.X + float32(sp.x), r.Y + float32(sp.y) }
		// The farther side fades toward the background, to tell it from the
		// nearer.
		fog := func(el string, z float64) ui.Color {
			return color(el).Mix(t.Background, float32(max(0, (reach-z)/(2*reach))*0.45))
		}
		for _, it := range items {
			if it.bond >= 0 {
				bd := m.Bonds[it.bond]
				stick(p, spots[bd.A], spots[bd.B], r, k, bd.Order, fog(m.Atoms[bd.A].Element, it.z), fog(m.Atoms[bd.B].Element, it.z))
				continue
			}
			sp := spots[it.atom]
			x, y := at(sp)
			ball(p, x, y, float32(sp.r), fog(m.Atoms[it.atom].Element, sp.z))
			if i := slices.Index(picked, it.atom); i >= 0 {
				rr := float32(sp.r) + 4
				ring := ui.Rect{X: x - rr, Y: y - rr, W: 2 * rr, H: 2 * rr}
				p.Stroke(ring, t.Accent, rr, 2)
			}
		}
		if tip != "" {
			span := ui.Span{Text: tip, Size: t.FontSize - 1, Color: t.Text, Weight: 600}
			w, h := p.MeasureText(0, span)
			box := ui.Rect{X: r.X + px + 14, Y: r.Y + py + 10, W: w + 12, H: h + 6}
			p.Fill(box, t.Surface, 4)
			p.Stroke(box, t.Border, 4, 1)
			p.RichText(box.X+6, box.Y+3, 0, span)
		}
		if measure != "" {
			span := ui.Span{Text: measure, Color: t.Text, Weight: 600, Features: "tnum"}
			w, h := p.MeasureText(0, span)
			box := ui.Rect{X: r.X + r.W/2 - w/2 - 10, Y: r.Y + r.H - h - 22, W: w + 20, H: h + 10}
			p.Fill(box, t.Surface.Alpha(0.92), 6)
			p.Stroke(box, t.Accent, 6, 1)
			p.RichText(box.X+10, box.Y+5, 0, span)
		}
	})
	return view
}

// ball draws an atom of color col at x, y, rad wide, lit from the top
// left.
func ball(p *ui.Painter, x, y, rad float32, col ui.Color) {
	r := ui.Rect{X: x - rad, Y: y - rad, W: 2 * rad, H: 2 * rad}
	p.FillGradient(r, ui.LinearGradient{From: col.Mix(ui.RGB(255, 255, 255), 0.25), To: col.Mix(ui.RGB(0, 0, 0), 0.45), Angle: 150, Oklab: true}, rad)
	spot := rad * 0.35
	p.Shadow(ui.Rect{X: x - rad*0.35 - spot/2, Y: y - rad*0.4 - spot/2, W: spot, H: spot}, spot/2, 0, 0, rad*0.5, 0, ui.RGBA(255, 255, 255, 0.55))
	p.Stroke(r, ui.RGB(0, 0, 0).Alpha(0.25), rad, 0.75)
}

// stick draws a bond from a to b, order sticks side by side, each half
// in its atom's color.
func stick(p *ui.Painter, a, b spot, r ui.Rect, k float64, order int, ca, cb ui.Color) {
	dx, dy := b.x-a.x, b.y-a.y
	l := math.Hypot(dx, dy)
	if l < 0.5 {
		return
	}
	nx, ny := -dy/l, dx/l
	w := float32(0.11 * k)
	gap := 0.14 * k
	offsets := []float64{0}
	switch order {
	case 2:
		offsets, w = []float64{-gap / 2, gap / 2}, w*0.7
	case 3:
		offsets, w = []float64{-gap, 0, gap}, w*0.6
	}
	mx, my := (a.x+b.x)/2, (a.y+b.y)/2
	for _, o := range offsets {
		ox, oy := nx*o, ny*o
		pt := func(x, y float64) [2]float32 { return [2]float32{r.X + float32(x+ox), r.Y + float32(y+oy)} }
		p.StrokePolyline([][2]float32{pt(a.x, a.y), pt(b.x, b.y)}, w+1.5, ui.RGB(0, 0, 0).Alpha(0.35))
		p.StrokePolyline([][2]float32{pt(a.x, a.y), pt(mx, my)}, w, ca.Mix(ui.RGB(255, 255, 255), 0.1))
		p.StrokePolyline([][2]float32{pt(mx, my), pt(b.x, b.y)}, w, cb.Mix(ui.RGB(255, 255, 255), 0.1))
	}
}
