// Package jigsaw is a jigsaw puzzle of a picture: the picture cut into
// pieces with tabs and blanks, scattered on a table around the frame they
// go in, where the picture shows faintly. Dragging a piece moves it, above
// the others; let go near its place, it snaps in and stays. How many
// pieces are in shows under the table, and once all are, the cuts between
// them fade away.
//
//	jigsaw.View(c, &app.puzzle, app.photo, 6, 4)
package jigsaw

import (
	"fmt"
	"image"
	"math/rand/v2"
	"slices"
	"time"

	"golang.org/x/image/vector"

	"github.com/egoist/mygo/ui"
)

// State is what a puzzle keeps: its pieces and where each lies. Keep it in
// the app's state, one for each puzzle.
type State struct {
	img        image.Image
	cols, rows int
	pieces     []piece
	ghost      *ui.Bitmap
	// order is the pieces from the bottom up.
	order []int
	drag  int
	done  time.Time
}

// piece is a piece: its picture, its outline, and where its cell lies on
// the table, all in cells from its cell's top left.
type piece struct {
	bmp    *ui.Bitmap
	bx, by float32
	bw, bh float32
	shape  [][2]float32
	x, y   float32
	home   [2]float32
	placed bool
}

// Shuffle cuts the picture anew and scatters the pieces.
func (s *State) Shuffle() { s.img = nil }

// The table's margin around the frame, in cells.
const (
	side = 2.2
	top  = 1.5
)

const fade = 600 * time.Millisecond

var (
	table = ui.Hex("#2F3A40")
	frame = ui.Hex("#232B30")
)

// View creates the puzzle of img, cut into cols×rows pieces.
func View(c *ui.Context, s *State, img image.Image, cols, rows int) ui.Element {
	t := c.Theme()
	if s.img != img || s.cols != cols || s.rows != rows {
		s.cut(img, cols, rows)
	}
	b := img.Bounds()
	cw := t.Space(90) / float32(cols)
	ch := cw * float32(b.Dy()) * float32(cols) / (float32(b.Dx()) * float32(rows))
	in := 0
	for _, p := range s.pieces {
		if p.placed {
			in++
		}
	}
	now := c.Now()
	if in == len(s.pieces) && s.done.IsZero() {
		s.done = now
	}
	tw, th := (float32(cols)+2*side)*cw, (float32(rows)+2*top)*ch
	root := ui.Column(c).Gap(t.Space(3)).AlignItems(ui.Center)
	root.Children(func() {
		e := ui.Box(c).Size(tw, th).Radius(t.Space(2)).Background(table).Clip().Role(ui.RoleImage).
			Label("Jigsaw puzzle").Value(fmt.Sprintf("%d of %d pieces in place", in, len(s.pieces)))
		px, py, over := e.PointerPosition()
		ux, uy := px/cw, py/ch
		hit := -1
		if over {
			for _, i := range slices.Backward(s.order) {
				p := s.pieces[i]
				if !p.placed && inside(p.shape, ux-p.x, uy-p.y) {
					hit = i
					break
				}
			}
		}
		pressed := e.Pressed()
		switch {
		case pressed && s.drag == 0 && hit >= 0:
			s.drag = hit + 1
			k := slices.Index(s.order, hit)
			s.order = append(slices.Delete(s.order, k, k+1), hit)
		case !pressed && s.drag > 0:
			p := &s.pieces[s.drag-1]
			if abs(p.x-p.home[0]) < 0.3 && abs(p.y-p.home[1]) < 0.3 {
				p.x, p.y, p.placed = p.home[0], p.home[1], true
				k := slices.Index(s.order, s.drag-1)
				s.order = slices.Insert(slices.Delete(s.order, k, k+1), 0, s.drag-1)
				e.SetChanged()
			}
			s.drag = 0
		}
		if dx, dy, ok := e.Dragged(); ok && s.drag > 0 {
			p := &s.pieces[s.drag-1]
			p.x = max(-0.5, min(float32(cols)+2*side-0.5, p.x+dx/cw))
			p.y = max(-0.5, min(float32(rows)+2*top-0.5, p.y+dy/ch))
		}
		switch {
		case s.drag > 0:
			e.Cursor(ui.CursorGrabbing)
		case hit >= 0:
			e.Cursor(ui.CursorGrab)
		}
		pieces, order, drag := slices.Clone(s.pieces), slices.Clone(s.order), s.drag
		ghost, done := s.ghost, s.done
		e.Draw(func(p *ui.Painter, r ui.Rect) {
			fr := ui.Rect{X: r.X + side*cw, Y: r.Y + top*ch, W: float32(cols) * cw, H: float32(rows) * ch}
			p.Fill(fr, frame, 0)
			p.ImageRegion(ghost, ui.Rect{}, fr, ui.ImageOptions{Tint: ui.RGBA(255, 255, 255, 0.16)})
			p.Stroke(fr, ui.RGBA(255, 255, 255, 0.12), 0, 1)
			cut := float32(0.4)
			if !done.IsZero() {
				k := float32(p.Now().Sub(done)) / float32(fade)
				if k < 1 {
					p.AnimationFrame()
				}
				cut *= max(0, 1-k)
			}
			for _, i := range order {
				pc := pieces[i]
				ox, oy := r.X+pc.x*cw, r.Y+pc.y*ch
				outline := func(dx, dy float32) *ui.Path {
					var path ui.Path
					for k, pt := range pc.shape {
						x, y := ox+pt[0]*cw+dx, oy+pt[1]*ch+dy
						if k == 0 {
							path.MoveTo(x, y)
						} else {
							path.LineTo(x, y)
						}
					}
					path.Close()
					return &path
				}
				if !pc.placed {
					lift := float32(2)
					if i == drag-1 {
						lift = 6
					}
					p.FillPath(outline(lift/2, lift), ui.RGBA(0, 0, 0, 0.35))
				}
				p.Image(pc.bmp, ui.Rect{X: ox + pc.bx*cw, Y: oy + pc.by*ch, W: pc.bw * cw, H: pc.bh * ch}, ui.FillBox)
				if cut > 0 {
					p.StrokePath(outline(0, 0), 1, ui.RGBA(0, 0, 0, cut))
				}
			}
		})
		note := fmt.Sprintf("%d of %d pieces in place", in, len(s.pieces))
		if in == len(s.pieces) {
			note = "Done"
		}
		ui.Text(c, note).TextColor(t.TextMuted)
	})
	return root
}

// cut cuts img into cols×rows pieces and scatters them around the frame.
func (s *State) cut(img image.Image, cols, rows int) {
	*s = State{img: img, cols: cols, rows: rows, ghost: ui.NewBitmap(img)}
	b := img.Bounds()
	cw, ch := float32(b.Dx())/float32(cols), float32(b.Dy())/float32(rows)
	size := min(cw, ch)
	sign := func() float32 {
		if rand.IntN(2) == 0 {
			return -1
		}
		return 1
	}
	// Each edge is cut once, so the pieces on either side of it fit.
	across := make([][][][2]float32, rows+1)
	for r := range rows + 1 {
		for col := range cols {
			k := float32(0)
			if r > 0 && r < rows {
				k = sign()
			}
			y := float32(r) * ch
			across[r] = append(across[r], edge([2]float32{float32(col) * cw, y}, [2]float32{float32(col+1) * cw, y}, [2]float32{0, 1}, size*k))
		}
	}
	down := make([][][][2]float32, rows)
	for r := range rows {
		for col := range cols + 1 {
			k := float32(0)
			if col > 0 && col < cols {
				k = sign()
			}
			x := float32(col) * cw
			down[r] = append(down[r], edge([2]float32{x, float32(r) * ch}, [2]float32{x, float32(r+1) * ch}, [2]float32{1, 0}, size*k))
		}
	}
	pad := int(size*0.3) + 2
	for r := range rows {
		for col := range cols {
			var outline [][2]float32
			outline = append(outline, along(across[r][col], false)...)
			outline = append(outline, along(down[r][col+1], false)...)
			outline = append(outline, along(across[r+1][col], true)...)
			outline = append(outline, along(down[r][col], true)...)
			x0, y0 := float32(col)*cw, float32(r)*ch
			box := image.Rect(int(x0)-pad, int(y0)-pad, int(x0+cw)+pad, int(y0+ch)+pad).Intersect(image.Rect(0, 0, b.Dx(), b.Dy()))
			z := vector.NewRasterizer(box.Dx(), box.Dy())
			shape := make([][2]float32, len(outline))
			for k, pt := range outline {
				x, y := pt[0]-float32(box.Min.X), pt[1]-float32(box.Min.Y)
				if k == 0 {
					z.MoveTo(x, y)
				} else {
					z.LineTo(x, y)
				}
				shape[k] = [2]float32{(pt[0] - x0) / cw, (pt[1] - y0) / ch}
			}
			z.ClosePath()
			dst := image.NewRGBA(image.Rect(0, 0, box.Dx(), box.Dy()))
			z.Draw(dst, dst.Bounds(), img, box.Min.Add(b.Min))
			s.pieces = append(s.pieces, piece{
				bmp: ui.NewBitmap(dst),
				bx:  (float32(box.Min.X) - x0) / cw, by: (float32(box.Min.Y) - y0) / ch,
				bw: float32(box.Dx()) / cw, bh: float32(box.Dy()) / ch,
				shape: shape,
				home:  [2]float32{side + float32(col), top + float32(r)},
			})
		}
	}
	tw, th := float32(cols)+2*side, float32(rows)+2*top
	for i := range s.pieces {
		p := &s.pieces[i]
		for range 200 {
			p.x, p.y = 0.15+rand.Float32()*(tw-1.3), 0.15+rand.Float32()*(th-1.3)
			if p.x+1 < side || p.x > tw-side || p.y+1 < top || p.y > th-top {
				break
			}
		}
		s.order = append(s.order, i)
	}
	rand.Shuffle(len(s.order), func(i, j int) { s.order[i], s.order[j] = s.order[j], s.order[i] })
}

// tab is a tab's outline along an edge from 0 to 1, standing out to 1.
var tab = [][2]float32{
	{0.38, 0}, {0.42, 0.04}, {0.36, 0.10}, {0.36, 0.14},
	{0.36, 0.22}, {0.44, 0.26}, {0.5, 0.26},
	{0.56, 0.26}, {0.64, 0.22}, {0.64, 0.14},
	{0.64, 0.10}, {0.58, 0.04}, {0.62, 0},
}

// edge returns the points along an edge from a to b, with a tab
// k long toward n, or into the other side for a negative k, or none for 0.
func edge(a, b, n [2]float32, k float32) [][2]float32 {
	out := [][2]float32{a}
	if k == 0 {
		return append(out, b)
	}
	shift := (rand.Float32() - 0.5) * 0.08
	at := func(q [2]float32) [2]float32 {
		u := q[0] + shift
		return [2]float32{a[0] + (b[0]-a[0])*u + n[0]*q[1]*k, a[1] + (b[1]-a[1])*u + n[1]*q[1]*k}
	}
	out = append(out, at(tab[0]))
	for i := 1; i+2 < len(tab); i += 3 {
		p0, p1, p2, p3 := at(tab[i-1]), at(tab[i]), at(tab[i+1]), at(tab[i+2])
		for step := 1; step <= 8; step++ {
			t := float32(step) / 8
			u := 1 - t
			out = append(out, [2]float32{
				u*u*u*p0[0] + 3*u*u*t*p1[0] + 3*u*t*t*p2[0] + t*t*t*p3[0],
				u*u*u*p0[1] + 3*u*u*t*p1[1] + 3*u*t*t*p2[1] + t*t*t*p3[1],
			})
		}
	}
	return append(out, b)
}

// along returns the points of e from its start, or from its end when
// back, without the point it ends at.
func along(e [][2]float32, back bool) [][2]float32 {
	if back {
		e = slices.Clone(e)
		slices.Reverse(e)
	}
	return e[:len(e)-1]
}

// inside reports whether x, y lies inside the outline.
func inside(shape [][2]float32, x, y float32) bool {
	in := false
	for i, j := 0, len(shape)-1; i < len(shape); j, i = i, i+1 {
		a, b := shape[i], shape[j]
		if (a[1] > y) != (b[1] > y) && x < a[0]+(y-a[1])*(b[0]-a[0])/(b[1]-a[1]) {
			in = !in
		}
	}
	return in
}

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
