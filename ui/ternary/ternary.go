// Package ternary draws a ternary plot, a triangle of mixtures of three
// parts, as the soil texture triangle places soils by their clay, sand
// and silt: one part at the top corner, one at the bottom left and one
// at the bottom right, a grid every tenth of each part with its labels
// along an edge, and each part's name along its edge with an arrow the
// way it grows. Regions, as the classes of soil, are drawn as outlined
// areas with their names. Each point is a dot at its mixture, which is
// read as shares of its three parts' sum. The pointer on a dot shows a
// card of its label, its three shares and the region it falls in; on a
// region, lights it.
//
//	ternary.View(c, ternary.Axes{A: "Clay", B: "Sand", C: "Silt"}, ternary.SoilTexture, samples)
package ternary

import (
	"fmt"
	"math"
	"strconv"

	"github.com/egoist/mygo/ui"
)

// Axes are the names of the parts: A at the top corner, B at the bottom
// left and C at the bottom right.
type Axes struct {
	A, B, C string
}

// Point is a mixture: its label, its parts, of any sum, and its color,
// or the zero Color for the accent.
type Point struct {
	Label   string
	A, B, C float64
	Color   ui.Color
}

// Region is a named area of the triangle, its corners as A, B and C in
// percent.
type Region struct {
	Name    string
	Corners [][3]float64
}

// SoilTexture is the USDA's twelve classes of soil texture, with A as
// clay, B as sand and C as silt.
var SoilTexture = []Region{
	{"Clay", [][3]float64{{100, 0, 0}, {55, 45, 0}, {40, 45, 15}, {40, 20, 40}, {60, 0, 40}}},
	{"Sandy clay", [][3]float64{{55, 45, 0}, {35, 65, 0}, {35, 45, 20}}},
	{"Silty clay", [][3]float64{{60, 0, 40}, {40, 20, 40}, {40, 0, 60}}},
	{"Sandy clay loam", [][3]float64{{35, 45, 20}, {35, 65, 0}, {20, 80, 0}, {20, 52, 28}, {27, 45, 28}}},
	{"Clay loam", [][3]float64{{40, 20, 40}, {40, 45, 15}, {35, 45, 20}, {27, 45, 28}, {27, 20, 53}}},
	{"Silty clay loam", [][3]float64{{40, 0, 60}, {40, 20, 40}, {27, 20, 53}, {27, 0, 73}}},
	{"Loam", [][3]float64{{27, 45, 28}, {20, 52, 28}, {7, 52, 41}, {7, 43, 50}, {27, 23, 50}}},
	{"Silt loam", [][3]float64{{27, 0, 73}, {27, 23, 50}, {7, 43, 50}, {0, 50, 50}, {0, 20, 80}, {12, 8, 80}, {12, 0, 88}}},
	{"Silt", [][3]float64{{12, 0, 88}, {12, 8, 80}, {0, 20, 80}, {0, 0, 100}}},
	{"Sandy loam", [][3]float64{{20, 52, 28}, {20, 80, 0}, {15, 85, 0}, {0, 70, 30}, {0, 50, 50}, {7, 43, 50}, {7, 52, 41}}},
	{"Loamy sand", [][3]float64{{15, 85, 0}, {10, 90, 0}, {0, 85, 15}, {0, 70, 30}}},
	{"Sand", [][3]float64{{10, 90, 0}, {0, 100, 0}, {0, 85, 15}}},
}

// View creates the plot of points over regions. Give it a size, or grow
// it in its parent.
func View(c *ui.Context, axes Axes, regions []Region, points []Point) ui.Element {
	t := c.Theme()
	small := t.FontSize - 2
	box := ui.Box(c).MinWidth(t.Space(90)).MinHeight(t.Space(80)).Role(ui.RoleImage).Label("Ternary plot")
	b := box.LayoutBounds()
	margin := t.Space(16)
	side := max(1, min(b.W-2*margin, (b.H-2*margin)*2/float32(math.Sqrt(3))))
	height := side * float32(math.Sqrt(3)) / 2
	ox, oy := (b.W-side)/2, (b.H-height)/2
	top := [2]float32{ox + side/2, oy}
	bl := [2]float32{ox, oy + height}
	br := [2]float32{ox + side, oy + height}
	at := func(a, bb, cc float64) [2]float32 {
		s := a + bb + cc
		if s <= 0 {
			return top
		}
		fa, fb, fc := float32(a/s), float32(bb/s), float32(cc/s)
		return [2]float32{fa*top[0] + fb*bl[0] + fc*br[0], fa*top[1] + fb*bl[1] + fc*br[1]}
	}
	polys := make([][][2]float32, len(regions))
	for i, rg := range regions {
		for _, k := range rg.Corners {
			polys[i] = append(polys[i], at(k[0], k[1], k[2]))
		}
	}
	regionAt := func(x, y float32) int {
		for i, poly := range polys {
			if inside(poly, x, y) {
				return i
			}
		}
		return -1
	}
	hot, hotRegion := -1, -1
	if px, py, over := box.PointerPosition(); over {
		best := t.Space(3)
		for i, pt := range points {
			q := at(pt.A, pt.B, pt.C)
			if d := float32(math.Hypot(float64(q[0]-px), float64(q[1]-py))); d < best {
				hot, best = i, d
			}
		}
		if hot < 0 {
			hotRegion = regionAt(px, py)
		}
	}
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		if r.W != b.W || r.H != b.H {
			c.Invalidate()
		}
		off := func(q [2]float32) [2]float32 { return [2]float32{r.X + q[0], r.Y + q[1]} }
		path := func(pts [][2]float32) *ui.Path {
			var pa ui.Path
			for i, q := range pts {
				q = off(q)
				if i == 0 {
					pa.MoveTo(q[0], q[1])
				} else {
					pa.LineTo(q[0], q[1])
				}
			}
			return pa.Close()
		}
		p.FillPath(path([][2]float32{top, bl, br}), t.Surface)
		for i, poly := range polys {
			if i == hotRegion {
				p.FillPath(path(poly), t.Accent.Alpha(0.14))
			}
		}
		for k := 1; k < 10; k++ {
			f := float64(k) * 10
			for _, ln := range [][2][2]float32{
				{at(f, 100-f, 0), at(f, 0, 100-f)},
				{at(100-f, f, 0), at(0, f, 100-f)},
				{at(100-f, 0, f), at(0, 100-f, f)},
			} {
				a, z := off(ln[0]), off(ln[1])
				p.StrokePolyline([][2]float32{a, z}, 1, t.Text.Alpha(0.07))
			}
			label := strconv.Itoa(k * 10)
			w, h := p.MeasureText(0, ui.Span{Text: label, Size: small})
			gap := t.Space(1.5)
			q := off(at(f, 100-f, 0))
			p.Text(q[0]-w-gap-t.Space(1), q[1]-h/2, label, small, t.TextMuted)
			q = off(at(0, f, 100-f))
			p.Text(q[0]-w/2, q[1]+gap, label, small, t.TextMuted)
			q = off(at(100-f, 0, f))
			p.Text(q[0]+gap+t.Space(1), q[1]-h/2, label, small, t.TextMuted)
		}
		for i, poly := range polys {
			ring := make([][2]float32, 0, len(poly)+1)
			for _, q := range append(poly, poly[0]) {
				ring = append(ring, off(q))
			}
			p.StrokePolyline(ring, 1.25, t.Text.Alpha(0.35))
			name := regions[i].Name
			cx, cy := centroid(poly)
			sp := ui.Span{Text: name, Size: small, Color: t.Text.Alpha(0.6), Weight: 600}
			if i == hotRegion {
				sp.Color = t.Text
			}
			w, h := p.MeasureText(0, sp)
			p.RichText(r.X+cx-w/2, r.Y+cy-h/2, 0, sp)
		}
		p.StrokePolyline([][2]float32{off(top), off(bl), off(br), off(top)}, 1.5, t.Text.Alpha(0.6))
		mid := off([2]float32{top[0], (top[1] + 2*bl[1]) / 3})
		edge(p, t, off(bl), off(top), mid, axes.A)
		edge(p, t, off(br), off(bl), mid, axes.B)
		edge(p, t, off(top), off(br), mid, axes.C)
		for i, pt := range points {
			q := off(at(pt.A, pt.B, pt.C))
			d := t.Space(1.25)
			if i == hot {
				d += 1.5
			}
			col := pt.Color
			if col == ui.Transparent {
				col = t.Accent
			}
			p.Fill(ui.Rect{X: q[0] - d - 1.5, Y: q[1] - d - 1.5, W: 2*d + 3, H: 2*d + 3}, t.Background, d+1.5)
			p.Fill(ui.Rect{X: q[0] - d, Y: q[1] - d, W: 2 * d, H: 2 * d}, col, d)
		}
		if hot >= 0 {
			pt := points[hot]
			q := at(pt.A, pt.B, pt.C)
			region := ""
			if i := regionAt(q[0], q[1]); i >= 0 {
				region = regions[i].Name
			}
			card(p, t, r, off(q), axes, pt, region)
		}
	})
	return box
}

// edge paints name along the edge from a to b, on the side away from
// the triangle's middle, with an arrow from a to b, the way its part
// grows.
func edge(p *ui.Painter, t *ui.Theme, a, b, middle [2]float32, name string) {
	if name == "" {
		return
	}
	dx, dy := b[0]-a[0], b[1]-a[1]
	l := float32(math.Hypot(float64(dx), float64(dy)))
	ux, uy := dx/l, dy/l
	nx, ny := -uy, ux
	if nx*((a[0]+b[0])/2-middle[0])+ny*((a[1]+b[1])/2-middle[1]) < 0 {
		nx, ny = -nx, -ny
	}
	angle := float32(math.Atan2(float64(dy), float64(dx)) * 180 / math.Pi)
	if angle > 90 || angle < -90 {
		angle += 180
	}
	mx, my := (a[0]+b[0])/2+nx*t.Space(10), (a[1]+b[1])/2+ny*t.Space(10)
	sp := ui.Span{Text: name + " (%)", Size: t.FontSize - 1, Weight: 600, Color: t.Text}
	w, h := p.MeasureText(0, sp)
	p.Transform(ui.Rotation(angle), mx, my, func() {
		p.RichText(mx-w/2, my-h/2, 0, sp)
	})
	s, e := 0.3*l, 0.7*l
	ax, ay := a[0]+nx*t.Space(6), a[1]+ny*t.Space(6)
	from := [2]float32{ax + ux*s, ay + uy*s}
	to := [2]float32{ax + ux*e, ay + uy*e}
	col := t.TextMuted.Alpha(0.6)
	p.StrokePolyline([][2]float32{from, to}, 1, col)
	head := t.Space(1.25)
	var tip ui.Path
	tip.MoveTo(to[0]+ux*head, to[1]+uy*head).
		LineTo(to[0]-ux*head+nx*head, to[1]-uy*head+ny*head).
		LineTo(to[0]-ux*head-nx*head, to[1]-uy*head-ny*head).Close()
	p.FillPath(&tip, col)
}

// card paints the card of pt at q.
func card(p *ui.Painter, t *ui.Theme, r ui.Rect, q [2]float32, axes Axes, pt Point, region string) {
	small := t.FontSize - 2
	s := pt.A + pt.B + pt.C
	share := func(v float64) string {
		if s <= 0 {
			return "0%"
		}
		return fmt.Sprintf("%.0f%%", v/s*100)
	}
	lines := [][]ui.Span{{{Text: pt.Label, Size: t.FontSize - 1, Weight: 600, Color: t.Text}}}
	if region != "" {
		lines = append(lines, []ui.Span{{Text: region, Size: small, Color: t.TextMuted}})
	}
	for _, part := range []struct {
		name string
		v    float64
	}{{axes.A, pt.A}, {axes.B, pt.B}, {axes.C, pt.C}} {
		lines = append(lines, []ui.Span{
			{Text: part.name + "  ", Size: small, Color: t.TextMuted},
			{Text: share(part.v), Size: small, Color: t.Text, Features: "tnum"},
		})
	}
	pad := t.Space(2.5)
	w, h := float32(0), float32(0)
	for _, l := range lines {
		lw, lh := p.MeasureText(0, l...)
		w, h = max(w, lw), h+lh
	}
	w, h = w+2*pad, h+2*pad
	x, y := q[0]+t.Space(3), q[1]-h/2
	if x+w > r.X+r.W {
		x = q[0] - t.Space(3) - w
	}
	y = max(r.Y, min(y, r.Y+r.H-h))
	bx := ui.Rect{X: x, Y: y, W: w, H: h}
	p.Shadow(bx, t.Radius+2, 0, 2, 10, 0, ui.RGBA(0, 0, 0, 0.2))
	p.Fill(bx, t.Background, t.Radius+2)
	p.Stroke(bx, t.Border, t.Radius+2, 1)
	ty := y + pad
	for _, l := range lines {
		_, lh := p.RichText(x+pad, ty, 0, l...)
		ty += lh
	}
}

// inside reports whether x, y is inside poly.
func inside(poly [][2]float32, x, y float32) bool {
	in := false
	for i, j := 0, len(poly)-1; i < len(poly); j, i = i, i+1 {
		a, b := poly[i], poly[j]
		if (a[1] > y) != (b[1] > y) && x < (b[0]-a[0])*(y-a[1])/(b[1]-a[1])+a[0] {
			in = !in
		}
	}
	return in
}

// centroid returns the middle of poly's area.
func centroid(poly [][2]float32) (float32, float32) {
	var area, cx, cy float32
	for i := range poly {
		a, b := poly[i], poly[(i+1)%len(poly)]
		cross := a[0]*b[1] - b[0]*a[1]
		area += cross
		cx += (a[0] + b[0]) * cross
		cy += (a[1] + b[1]) * cross
	}
	if area == 0 {
		return poly[0][0], poly[0][1]
	}
	return cx / (3 * area), cy / (3 * area)
}
