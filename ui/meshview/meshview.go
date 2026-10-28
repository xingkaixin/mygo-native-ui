// Package meshview shows a 3D model as a mesh of triangles, as a CAD
// app's viewport or Quick Look does: the faces lit from over the viewer's
// shoulder, smooth across shallow angles and creased at sharp ones, the
// nearer hiding the farther. Dragging turns the model about its up axis
// and tilts it, and the wheel or a pinch zooms. A cube in the corner
// shows which sides face the viewer; a click on one of its faces turns
// the model to look at that side, and a double-click on the model turns
// it back to where it started.
//
//	m, err := meshview.ParseSTL(data)
//	meshview.View(c, &app.viewer, m.ZUp()).Grow(1)
//
// The model's up axis is y, as OBJ files have it; Mesh.ZUp turns one
// whose up is z, as most STL files have it.
package meshview

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// Mesh is a model's corners and the triangles between them, each
// triangle three indices into Vertices.
type Mesh struct {
	Vertices [][3]float64
	Faces    [][3]int
}

// ZUp returns m turned so that its z axis points up, as y does on screen.
func (m Mesh) ZUp() Mesh {
	out := Mesh{Vertices: make([][3]float64, len(m.Vertices)), Faces: m.Faces}
	for i, v := range m.Vertices {
		out.Vertices[i] = [3]float64{v[0], v[2], -v[1]}
	}
	return out
}

// ParseOBJ returns the mesh of a Wavefront OBJ file: its vertices, and
// its faces cut into triangles. Texture coordinates, normals, groups and
// materials are left out.
func ParseOBJ(src string) (Mesh, error) {
	var m Mesh
	for n, line := range strings.Split(src, "\n") {
		f := strings.Fields(line)
		if len(f) == 0 {
			continue
		}
		switch f[0] {
		case "v":
			if len(f) < 4 {
				return Mesh{}, fmt.Errorf("obj: line %d: %q", n+1, line)
			}
			var v [3]float64
			for k := range 3 {
				x, err := strconv.ParseFloat(f[k+1], 64)
				if err != nil {
					return Mesh{}, fmt.Errorf("obj: line %d: %w", n+1, err)
				}
				v[k] = x
			}
			m.Vertices = append(m.Vertices, v)
		case "f":
			idx := make([]int, 0, len(f)-1)
			for _, s := range f[1:] {
				s, _, _ = strings.Cut(s, "/")
				i, err := strconv.Atoi(s)
				if err != nil {
					return Mesh{}, fmt.Errorf("obj: line %d: %w", n+1, err)
				}
				// Negative indices count back from the last vertex.
				if i < 0 {
					i += len(m.Vertices)
				} else {
					i--
				}
				if i < 0 || i >= len(m.Vertices) {
					return Mesh{}, fmt.Errorf("obj: line %d: vertex %s of %d", n+1, s, len(m.Vertices))
				}
				idx = append(idx, i)
			}
			for k := 2; k < len(idx); k++ {
				m.Faces = append(m.Faces, [3]int{idx[0], idx[k-1], idx[k]})
			}
		}
	}
	if len(m.Faces) == 0 {
		return Mesh{}, errors.New("obj: no faces")
	}
	return m, nil
}

// ParseSTL returns the mesh of an STL file, binary or ASCII, its
// triangles joined where their corners meet.
func ParseSTL(data []byte) (Mesh, error) {
	var tris [][3][3]float64
	if len(data) >= 84 {
		n := int(binary.LittleEndian.Uint32(data[80:84]))
		if 84+50*n == len(data) {
			tris = make([][3][3]float64, n)
			for i := range n {
				// Each triangle is its normal, its corners, and two bytes.
				o := 84 + 50*i + 12
				for v := range 3 {
					for k := range 3 {
						tris[i][v][k] = float64(math.Float32frombits(binary.LittleEndian.Uint32(data[o+12*v+4*k:])))
					}
				}
			}
			return weld(tris), nil
		}
	}
	f := strings.Fields(string(data))
	var t [3][3]float64
	v := 0
	for i := 0; i < len(f); i++ {
		if f[i] != "vertex" {
			continue
		}
		if i+3 >= len(f) {
			return Mesh{}, errors.New("stl: vertex cut short")
		}
		for k := range 3 {
			x, err := strconv.ParseFloat(f[i+1+k], 64)
			if err != nil {
				return Mesh{}, fmt.Errorf("stl: %w", err)
			}
			t[v][k] = x
		}
		i += 3
		if v++; v == 3 {
			tris, v = append(tris, t), 0
		}
	}
	if len(tris) == 0 {
		return Mesh{}, errors.New("stl: no triangles")
	}
	return weld(tris), nil
}

// weld makes one vertex of the corners at the same point, which smooth
// shading needs, and leaves out triangles with two corners on one.
func weld(tris [][3][3]float64) Mesh {
	var m Mesh
	at := map[[3]float64]int{}
	for _, t := range tris {
		var f [3]int
		for v, p := range t {
			i, ok := at[p]
			if !ok {
				i = len(m.Vertices)
				at[p] = i
				m.Vertices = append(m.Vertices, p)
			}
			f[v] = i
		}
		if f[0] != f[1] && f[1] != f[2] && f[0] != f[2] {
			m.Faces = append(m.Faces, f)
		}
	}
	return m
}

// Style is how the model's triangles draw.
type Style int

const (
	// Shaded fills the triangles.
	Shaded Style = iota
	// Edges fills them and outlines each.
	Edges
	// Wireframe outlines them alone, those behind showing through.
	Wireframe
)

// State is what a viewer keeps: how the model is turned and zoomed, and
// its shading worked out once for each mesh. Keep it in the app's state,
// one for each view. Style is the app's to change.
type State struct {
	Style Style

	yaw, pitch, zoom float64
	ready            bool
	mesh             *[3]int
	faces            int
	prep             prep
	img              [2]*image.RGBA
	flip             int
	depth            []float32
	bitmap           *ui.Bitmap
	drawn            frame
}

// frame is what the last picture was drawn of, to draw it again only
// when something changed.
type frame struct {
	yaw, pitch, zoom float32
	w, h             int
	style            Style
	base, edge       ui.Color
	mesh             *[3]int
}

// prep is a mesh worked out for drawing: its middle, how far it reaches
// from there, a normal at each corner of each face, and its edges.
type prep struct {
	center  [3]float64
	reach   float64
	normals [][3][3]float64
	edges   [][2]int
}

// crease is the angle between two faces past which the edge between them
// shows sharp.
var crease = math.Cos(40 * math.Pi / 180)

func prepare(m Mesh) prep {
	lo, hi := m.Vertices[0], m.Vertices[0]
	for _, v := range m.Vertices {
		for k := range 3 {
			lo[k], hi[k] = min(lo[k], v[k]), max(hi[k], v[k])
		}
	}
	p := prep{center: [3]float64{(lo[0] + hi[0]) / 2, (lo[1] + hi[1]) / 2, (lo[2] + hi[2]) / 2}}
	for _, v := range m.Vertices {
		p.reach = max(p.reach, length(sub(v, p.center)))
	}
	p.reach = max(p.reach, 1e-9)
	// A face's normal before it is made a unit is as long as twice its
	// area, so that the sum at a corner leans toward the larger faces.
	area := make([][3]float64, len(m.Faces))
	unit := make([][3]float64, len(m.Faces))
	around := make([][]int, len(m.Vertices))
	seen := map[[2]int]bool{}
	for i, f := range m.Faces {
		a, b, c := m.Vertices[f[0]], m.Vertices[f[1]], m.Vertices[f[2]]
		area[i] = cross(sub(b, a), sub(c, a))
		unit[i] = normalize(area[i])
		for k, v := range f {
			around[v] = append(around[v], i)
			e := [2]int{v, f[(k+1)%3]}
			if e[0] > e[1] {
				e[0], e[1] = e[1], e[0]
			}
			if !seen[e] {
				seen[e] = true
				p.edges = append(p.edges, e)
			}
		}
	}
	p.normals = make([][3][3]float64, len(m.Faces))
	for i, f := range m.Faces {
		for k, v := range f {
			var sum [3]float64
			for _, j := range around[v] {
				if dot(unit[i], unit[j]) >= crease {
					sum = add(sum, area[j])
				}
			}
			p.normals[i][k] = normalize(sum)
		}
	}
	return p
}

var turnAnim = ui.Animation{Spring: ui.SpringOptions{Duration: 450 * time.Millisecond}}

const homeYaw, homePitch = -0.6, 0.45

// View creates the viewer of m. Give it a size, or grow it in its parent.
func View(c *ui.Context, s *State, m Mesh) ui.Element {
	t := c.Theme()
	view := ui.Box(c).Role(ui.RoleImage).Label("3D model").Cursor(ui.CursorGrab).Clip()
	if !s.ready {
		s.yaw, s.pitch, s.zoom, s.ready = homeYaw, homePitch, 1, true
	}
	if len(m.Faces) == 0 {
		return view
	}
	if id := &m.Faces[0]; id != s.mesh || len(m.Faces) != s.faces {
		s.mesh, s.faces, s.prep = id, len(m.Faces), prepare(m)
	}
	view.HandleInput(func(in ui.InputEvent) bool {
		switch in.Kind {
		case ui.InputScroll:
			s.zoom = max(0.2, min(8, s.zoom*math.Exp2(-float64(in.DY)/300)))
		case ui.InputZoom:
			s.zoom = max(0.2, min(8, s.zoom*float64(in.Scale)))
		default:
			return false
		}
		return true
	})
	dragged := false
	if dx, dy, ok := view.Dragged(); ok {
		s.yaw += float64(dx) * 0.01
		s.pitch = max(-math.Pi/2, min(math.Pi/2, s.pitch+float64(dy)*0.01))
		view.Cursor(ui.CursorGrabbing)
		dragged = true
	}
	if view.DoubleClicked() {
		s.yaw, s.pitch, s.zoom = nearest(homeYaw, s.yaw), homePitch, 1
	}
	var yaw, pitch float32
	if dragged {
		yaw, pitch = view.Jump("yaw", float32(s.yaw)), view.Jump("pitch", float32(s.pitch))
	} else {
		yaw, pitch = view.AnimateTo("yaw", float32(s.yaw), turnAnim), view.AnimateTo("pitch", float32(s.pitch), turnAnim)
	}
	zoom := view.AnimateTo("zoom", float32(s.zoom), ui.Animation{Spring: ui.SpringOptions{Duration: 150 * time.Millisecond}})
	base := ui.Oklch(0.78, 0.035, 250)
	edge := ui.RGB(30, 34, 44).Alpha(0.55)
	if t.Dark {
		base, edge = ui.Oklch(0.72, 0.035, 250), ui.RGB(10, 12, 18).Alpha(0.6)
	}
	style := s.Style
	view.Draw(func(p *ui.Painter, r ui.Rect) {
		if style == Wireframe {
			s.wire(p, m, r, float64(yaw), float64(pitch), float64(zoom), t.Text)
			return
		}
		sc := p.Scale()
		w, h := int(r.W*sc), int(r.H*sc)
		if w <= 0 || h <= 0 {
			return
		}
		f := frame{yaw, pitch, zoom, w, h, style, base, edge, s.mesh}
		if s.bitmap == nil || f != s.drawn {
			img := s.render(m, f, sc)
			if s.bitmap == nil {
				s.bitmap = ui.NewBitmap(img)
			} else {
				s.bitmap.Update(img)
			}
			s.drawn = f
		}
		p.Image(s.bitmap, r, ui.FillBox)
	})
	view.Children(func() {
		cube(c, s, float64(yaw), float64(pitch))
	})
	return view
}

// nearest returns the angle a turn short of from that is a whole number
// of turns from to, so that turning there goes the short way round.
func nearest(to, from float64) float64 {
	return to + 2*math.Pi*math.Round((from-to)/(2*math.Pi))
}

// turn turns v by yaw about the up axis, then tilts it by pitch toward the
// viewer, who looks down the z axis from its far end.
func turn(v [3]float64, yaw, pitch float64) [3]float64 {
	cy, sy := math.Cos(yaw), math.Sin(yaw)
	x, z := v[0]*cy+v[2]*sy, -v[0]*sy+v[2]*cy
	cp, sp := math.Cos(pitch), math.Sin(pitch)
	return [3]float64{x, v[1]*cp - z*sp, v[1]*sp + z*cp}
}

// project returns where each vertex of m shows in a picture w by h, and
// how near it is.
func (s *State) project(m Mesh, w, h, yaw, pitch, zoom float64) [][3]float64 {
	pr := s.prep
	k := min(w, h) / 2 / pr.reach * zoom * 0.92
	// The eye stands back as far as four models, for a little perspective.
	eye := pr.reach * 4
	out := make([][3]float64, len(m.Vertices))
	for i, v := range m.Vertices {
		q := turn(sub(v, pr.center), yaw, pitch)
		f := eye / (eye - q[2])
		out[i] = [3]float64{w/2 + q[0]*k*f, h/2 - q[1]*k*f, q[2]}
	}
	return out
}

// light is where the key light comes from, over the viewer's left
// shoulder; a weaker one at the eye keeps the sides it misses from going
// black.
var light = normalize([3]float64{-0.3, 0.5, 1})

// render draws the model's triangles into a picture of f.w by f.h pixels,
// sc of them a DIP, keeping the nearest at each pixel.
func (s *State) render(m Mesh, f frame, sc float32) *image.RGBA {
	w, h := f.w, f.h
	// The picture shown waits for the next Update before its pixels may
	// change, so two take turns.
	s.flip ^= 1
	img := s.img[s.flip]
	if img == nil || img.Rect.Dx() != w || img.Rect.Dy() != h {
		img = image.NewRGBA(image.Rect(0, 0, w, h))
		s.img[s.flip] = img
	}
	clear(img.Pix)
	if cap(s.depth) < w*h {
		s.depth = make([]float32, w*h)
	}
	depth := s.depth[:w*h]
	for i := range depth {
		depth[i] = float32(math.Inf(-1))
	}
	yaw, pitch := float64(f.yaw), float64(f.pitch)
	pts := s.project(m, float64(w), float64(h), yaw, pitch, float64(f.zoom))
	line := 0.9 * float64(sc)
	br, bg, bb := float64(f.base.R), float64(f.base.G), float64(f.base.B)
	er, eg, eb, ea := float64(f.edge.R), float64(f.edge.G), float64(f.edge.B), float64(f.edge.A)/255
	for i, face := range m.Faces {
		a, b, c := pts[face[0]], pts[face[1]], pts[face[2]]
		area := (b[0]-a[0])*(c[1]-a[1]) - (b[1]-a[1])*(c[0]-a[0])
		if math.Abs(area) < 1e-9 {
			continue
		}
		var shade [3]float64
		for k := range 3 {
			n := turn(s.prep.normals[i][k], yaw, pitch)
			// Insides seen through an open mesh are lit as outsides.
			if n[2] < 0 {
				n = [3]float64{-n[0], -n[1], -n[2]}
			}
			shade[k] = 0.2 + 0.5*max(0, dot(n, light)) + 0.3*n[2]
		}
		var lens [3]float64
		if f.style == Edges {
			lens = [3]float64{math.Hypot(c[0]-b[0], c[1]-b[1]), math.Hypot(a[0]-c[0], a[1]-c[1]), math.Hypot(b[0]-a[0], b[1]-a[1])}
		}
		x0, x1 := max(0, int(min(a[0], b[0], c[0]))), min(w-1, int(max(a[0], b[0], c[0])))
		y0, y1 := max(0, int(min(a[1], b[1], c[1]))), min(h-1, int(max(a[1], b[1], c[1])))
		for y := y0; y <= y1; y++ {
			py := float64(y) + 0.5
			for x := x0; x <= x1; x++ {
				px := float64(x) + 0.5
				// Each weight is the area of the triangle the pixel makes
				// with the edge across from a corner, signed as the face is.
				w0 := ((c[0]-b[0])*(py-b[1]) - (c[1]-b[1])*(px-b[0])) / area
				w1 := ((a[0]-c[0])*(py-c[1]) - (a[1]-c[1])*(px-c[0])) / area
				w2 := 1 - w0 - w1
				if w0 < 0 || w1 < 0 || w2 < 0 {
					continue
				}
				z := float32(w0*a[2] + w1*b[2] + w2*c[2])
				at := y*w + x
				if z <= depth[at] {
					continue
				}
				depth[at] = z
				l := w0*shade[0] + w1*shade[1] + w2*shade[2]
				r, g, bl := br*l, bg*l, bb*l
				if f.style == Edges {
					d := min(w0*math.Abs(area)/lens[0], w1*math.Abs(area)/lens[1], w2*math.Abs(area)/lens[2])
					if k := max(0, min(1, line-d+0.5)) * ea; k > 0 {
						r, g, bl = r+(er-r)*k, g+(eg-g)*k, bl+(eb-bl)*k
					}
				}
				o := 4 * at
				img.Pix[o], img.Pix[o+1], img.Pix[o+2], img.Pix[o+3] = byte(min(255, r)), byte(min(255, g)), byte(min(255, bl)), 255
			}
		}
	}
	return img
}

// wire draws each edge of m once, the farther ones fainter.
func (s *State) wire(p *ui.Painter, m Mesh, r ui.Rect, yaw, pitch, zoom float64, col ui.Color) {
	pts := s.project(m, float64(r.W), float64(r.H), yaw, pitch, zoom)
	reach := s.prep.reach
	for _, e := range s.prep.edges {
		a, b := pts[e[0]], pts[e[1]]
		near := ((a[2]+b[2])/2/reach + 1) / 2
		p.StrokePolyline([][2]float32{{r.X + float32(a[0]), r.Y + float32(a[1])}, {r.X + float32(b[0]), r.Y + float32(b[1])}}, 1, col.Alpha(float32(0.15+0.55*near)))
	}
}

// sides are the faces of the view cube: the way each faces, its name,
// and the turn that faces it to the viewer.
var sides = []struct {
	n          [3]float64
	name       string
	yaw, pitch float64
}{
	{[3]float64{0, 0, 1}, "Front", 0, 0},
	{[3]float64{0, 0, -1}, "Back", math.Pi, 0},
	{[3]float64{1, 0, 0}, "Right", -math.Pi / 2, 0},
	{[3]float64{-1, 0, 0}, "Left", math.Pi / 2, 0},
	{[3]float64{0, 1, 0}, "Top", 0, math.Pi / 2},
	{[3]float64{0, -1, 0}, "Bottom", 0, -math.Pi / 2},
}

// corners returns the corners of the face of the unit cube facing n, in
// order around it.
func corners(n [3]float64) [4][3]float64 {
	// u and v are the face's two other axes.
	u := [3]float64{n[1], n[2], n[0]}
	v := cross(n, u)
	var out [4][3]float64
	for i, c := range [4][2]float64{{-1, -1}, {1, -1}, {1, 1}, {-1, 1}} {
		out[i] = add(n, add(scale(u, c[0]), scale(v, c[1])))
	}
	return out
}

// cube creates the view cube in the top right corner of the viewer.
func cube(c *ui.Context, s *State, yaw, pitch float64) {
	t := c.Theme()
	size := t.Space(22)
	e := ui.Box(c).Absolute().Top(t.Space(3)).Right(t.Space(3)).Size(size, size).Cursor(ui.CursorPointer).
		Role(ui.RoleGroup).Label("View cube")
	k := float64(size) / 2 / 1.9
	at := func(v [3]float64) (float32, float32) {
		q := turn(v, yaw, pitch)
		return float32(float64(size)/2 + q[0]*k), float32(float64(size)/2 - q[1]*k)
	}
	// The cube is convex, so the faces toward the viewer never overlap.
	hover := -1
	px, py, over := e.PointerPosition()
	for i, sd := range sides {
		if turn(sd.n, yaw, pitch)[2] <= 0.05 || !over {
			continue
		}
		var poly [4][2]float32
		for j, v := range corners(sd.n) {
			poly[j][0], poly[j][1] = at(v)
		}
		if inside(poly, px, py) {
			hover = i
		}
	}
	if e.Clicked() && hover >= 0 {
		sd := sides[hover]
		s.pitch = sd.pitch
		s.yaw = nearest(sd.yaw, s.yaw)
	}
	e.Draw(func(p *ui.Painter, r ui.Rect) {
		for i, sd := range sides {
			nz := turn(sd.n, yaw, pitch)[2]
			if nz <= 0.05 {
				continue
			}
			var path ui.Path
			for j, v := range corners(sd.n) {
				x, y := at(v)
				if j == 0 {
					path.MoveTo(r.X+x, r.Y+y)
				} else {
					path.LineTo(r.X+x, r.Y+y)
				}
			}
			path.Close()
			fill := t.Surface.Mix(t.Text, float32(0.12*(1-nz)))
			if i == hover {
				fill = t.Accent.Mix(t.Surface, 0.55)
			}
			p.FillPath(&path, fill.Alpha(0.92))
			p.StrokePath(&path, 1, t.Border)
			if nz > 0.4 {
				cx, cy := at(sd.n)
				span := ui.Span{Text: sd.name, Size: t.FontSize - 3, Color: t.Text.Alpha(float32(nz)), Weight: 600}
				tw, th := p.MeasureText(0, span)
				p.RichText(r.X+cx-tw/2, r.Y+cy-th/2, 0, span)
			}
		}
	})
}

// inside reports whether x, y is inside the convex polygon poly.
func inside(poly [4][2]float32, x, y float32) bool {
	sign := float32(0)
	for i := range poly {
		a, b := poly[i], poly[(i+1)%4]
		c := (b[0]-a[0])*(y-a[1]) - (b[1]-a[1])*(x-a[0])
		if c*sign < 0 {
			return false
		}
		if c != 0 {
			sign = c
		}
	}
	return true
}

func sub(a, b [3]float64) [3]float64 { return [3]float64{a[0] - b[0], a[1] - b[1], a[2] - b[2]} }
func add(a, b [3]float64) [3]float64 { return [3]float64{a[0] + b[0], a[1] + b[1], a[2] + b[2]} }
func scale(a [3]float64, k float64) [3]float64 {
	return [3]float64{a[0] * k, a[1] * k, a[2] * k}
}
func dot(a, b [3]float64) float64 { return a[0]*b[0] + a[1]*b[1] + a[2]*b[2] }
func cross(a, b [3]float64) [3]float64 {
	return [3]float64{a[1]*b[2] - a[2]*b[1], a[2]*b[0] - a[0]*b[2], a[0]*b[1] - a[1]*b[0]}
}
func length(a [3]float64) float64 { return math.Sqrt(dot(a, a)) }
func normalize(a [3]float64) [3]float64 {
	if l := length(a); l > 0 {
		return scale(a, 1/l)
	}
	return a
}
