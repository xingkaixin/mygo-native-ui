// Package density shows where many points fall on a plane, too many to
// show as dots, as hexagonal bins or as contours of their density: the
// plane with an axis of round ticks along its bottom and its left, each
// with its title. Hexbin counts the points in hexagons of a few DIPs and
// colors each by its count, from light to dark; Contour estimates the
// density with a Gaussian kernel and shades the bands between ten levels
// of it, with a line along each level, over the points as faint dots. A
// legend says what the darkest color counts. The pointer on a hexagon
// shows its count and where it is.
//
//	app.plot.Mode = density.Contour
//	density.View(c, &app.plot, density.Axis{Title: "Duration", Unit: "min"},
//		density.Axis{Title: "Wait", Unit: "min"}, app.eruptions).Grow(1)
//
// The bins and the density are kept until the points, the size or the
// mode change.
package density

import (
	"math"
	"strconv"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"
)

// Mode is how a plot shows the density of its points.
type Mode uint8

const (
	// Hexbin counts the points in hexagons.
	Hexbin Mode = iota
	// Contour shades the bands between levels of the points' density.
	Contour
)

// Axis is an axis of the plane: its title, and the unit that follows it
// in parentheses.
type Axis struct {
	Title, Unit string
}

// State is what a plot keeps: what it made of the points. Mode, which the
// app sets, is how it shows them. Keep it in the app's state, one for each
// plot.
type State struct {
	Mode Mode

	key   [5]any
	bins  map[[2]int]int
	most  int
	grid  []float64
	cols  int
	peak  float64
	lines [][]float32
}

// cell is the side of the squares the density is estimated over, in
// DIPs.
const cell = 4

// levels is how many levels of density the contours draw.
const levels = 10

// View creates the plot of points, each an x and a y, along x and y. Give
// it a size, or grow it in its parent.
func View(c *ui.Context, s *State, x, y Axis, points [][2]float64) ui.Element {
	t := c.Theme()
	xlo, xhi, ylo, yhi := math.Inf(1), math.Inf(-1), math.Inf(1), math.Inf(-1)
	for _, p := range points {
		xlo, xhi = min(xlo, p[0]), max(xhi, p[0])
		ylo, yhi = min(ylo, p[1]), max(yhi, p[1])
	}
	if len(points) == 0 {
		xlo, xhi, ylo, yhi = 0, 1, 0, 1
	}
	xs, ys := chart.NewLinear(xlo, xhi).Nice(6), chart.NewLinear(ylo, yhi).Nice(5)
	xt, yt := xs.Ticks(6), ys.Ticks(5)
	small := t.FontSize - 2
	box := ui.Box(c).Role(ui.RoleImage).Label(name(x, y))
	b := box.LayoutBounds()
	yw := float32(0)
	for _, tk := range yt {
		w, _ := c.MeasureText(0, ui.Span{Text: tk.Label, Size: small})
		yw = max(yw, w)
	}
	left := yw + t.Space(3)
	if y.Title != "" {
		left += t.FontSize + t.Space(2)
	}
	bottom := small + t.Space(3)
	if x.Title != "" {
		bottom += t.FontSize + t.Space(2)
	}
	plot := ui.Rect{X: left, Y: t.Space(8), W: max(1, b.W-left-t.Space(2)), H: max(1, b.H-t.Space(8)-bottom)}
	px := func(v float64) float32 { return plot.X + xs.At(v)*plot.W }
	py := func(v float64) float32 { return plot.Y + plot.H - ys.At(v)*plot.H }
	radius := t.Space(2.5)
	s.make(points, plot, radius, px, py)

	hot, count := [2]int{}, 0
	if mx, my, ok := box.PointerPosition(); ok && box.Hovered() && s.Mode == Hexbin && plot.Contains(mx, my) {
		hot = hexAt(mx-plot.X, my-plot.Y, radius)
		if count = s.bins[hot]; count > 0 {
			cx, cy := hexCenter(hot, radius)
			box.Tooltip(strconv.Itoa(count) + " points near " + format(xs.Invert((cx)/plot.W)) + ", " +
				format(ys.Invert(1-cy/plot.H)))
		}
	}
	mode, dark := s.Mode, t.Dark
	box.Draw(func(p *ui.Painter, r ui.Rect) {
		pl := plot
		pl.X += r.X
		pl.Y += r.Y
		grid := t.Border.Alpha(0.5)
		for _, tk := range xt {
			at := pl.X + tk.At*pl.W
			p.Line(at, pl.Y, at, pl.Y+pl.H, 1, grid)
			w, _ := p.MeasureText(0, ui.Span{Text: tk.Label, Size: small})
			p.Text(at-w/2, pl.Y+pl.H+t.Space(1.5), tk.Label, small, t.TextMuted)
		}
		for _, tk := range yt {
			at := pl.Y + pl.H - tk.At*pl.H
			p.Line(pl.X, at, pl.X+pl.W, at, 1, grid)
			w, h := p.MeasureText(0, ui.Span{Text: tk.Label, Size: small})
			p.Text(pl.X-t.Space(2)-w, at-h/2, tk.Label, small, t.TextMuted)
		}
		axisTitles(p, t, r, pl, x, y, small)
		p.Clip(pl, 0, func() {
			if mode == Hexbin {
				for k, n := range s.bins {
					cx, cy := hexCenter(k, radius)
					col := ramp(float32(math.Sqrt(float64(n)/float64(s.most))), dark)
					if count > 0 && k == hot {
						col = t.Text
					}
					p.FillPath(hexPath(pl.X+cx, pl.Y+cy, radius), col)
				}
				return
			}
			for k, v := range s.grid {
				lv := int(v / s.peak * levels)
				if lv == 0 {
					continue
				}
				p.Fill(ui.Rect{X: pl.X + float32(k%s.cols*cell), Y: pl.Y + float32(k/s.cols*cell), W: cell, H: cell},
					ramp(float32(min(lv, levels))/levels, dark), 0)
			}
			for _, pt := range points {
				p.Fill(ui.Rect{X: r.X + px(pt[0]) - 1, Y: r.Y + py(pt[1]) - 1, W: 2, H: 2}, t.Text.Alpha(0.25), 1)
			}
			for lv, seg := range s.lines {
				var path ui.Path
				for i := 0; i+3 < len(seg); i += 4 {
					path.MoveTo(pl.X+seg[i], pl.Y+seg[i+1]).LineTo(pl.X+seg[i+2], pl.Y+seg[i+3])
				}
				p.StrokePath(&path, 1, ramp(float32(lv+2)/levels, dark).Mix(t.Text, 0.3))
			}
		})
		legend(p, t, r, pl, s, dark, small)
	})
	return box
}

// name names the plot after its axes.
func name(x, y Axis) string {
	if x.Title == "" || y.Title == "" {
		return "Density plot"
	}
	return y.Title + " by " + x.Title
}

// make bins the points, or estimates their density, at their places in
// plot, unless it has for these points, this size and this mode.
func (s *State) make(points [][2]float64, plot ui.Rect, radius float32, px, py func(float64) float32) {
	var first *[2]float64
	if len(points) > 0 {
		first = &points[0]
	}
	key := [5]any{first, len(points), plot.W, plot.H, s.Mode}
	if key == s.key {
		return
	}
	s.key = key
	if s.Mode == Hexbin {
		s.bins, s.most = map[[2]int]int{}, 1
		for _, p := range points {
			k := hexAt(px(p[0])-plot.X, py(p[1])-plot.Y, radius)
			s.bins[k]++
			s.most = max(s.most, s.bins[k])
		}
		return
	}
	cols, rows := int(plot.W/cell)+1, int(plot.H/cell)+1
	grid := make([]float64, cols*rows)
	for _, p := range points {
		i, j := int((px(p[0])-plot.X)/cell), int((py(p[1])-plot.Y)/cell)
		if i >= 0 && i < cols && j >= 0 && j < rows {
			grid[j*cols+i]++
		}
	}
	// The bandwidth follows Scott's rule, n^(-1/6) of the spread, for the
	// spread of the points across the plot.
	sigma := max(2, 0.18*math.Pow(float64(max(1, len(points))), -1.0/6)*float64(min(cols, rows)))
	grid = blur(grid, cols, rows, sigma)
	s.grid, s.cols, s.peak = grid, cols, 1e-12
	for _, v := range grid {
		s.peak = max(s.peak, v)
	}
	s.lines = make([][]float32, levels-1)
	for lv := 1; lv < levels; lv++ {
		s.lines[lv-1] = march(grid, cols, rows, s.peak*float64(lv)/levels)
	}
}

// blur returns grid, cols by rows, convolved with a Gaussian of sigma
// cells, across and then down.
func blur(grid []float64, cols, rows int, sigma float64) []float64 {
	n := int(math.Ceil(sigma * 3))
	kernel := make([]float64, 2*n+1)
	for i := range kernel {
		d := float64(i - n)
		kernel[i] = math.Exp(-d * d / (2 * sigma * sigma))
	}
	pass := func(src []float64, along, across, stride, step int) []float64 {
		dst := make([]float64, len(src))
		for a := range across {
			for i := range along {
				sum := 0.0
				for k, w := range kernel {
					if j := i + k - n; j >= 0 && j < along {
						sum += w * src[a*stride+j*step]
					}
				}
				dst[a*stride+i*step] = sum
			}
		}
		return dst
	}
	return pass(pass(grid, cols, rows, cols, 1), rows, cols, 1, cols)
}

// march returns the segments where grid, cols by rows, crosses level, by
// marching squares, as x0, y0, x1, y1 in DIPs from the grid's top left.
func march(grid []float64, cols, rows int, level float64) []float32 {
	var out []float32
	at := func(i, j int) float64 { return grid[j*cols+i] }
	lerp := func(a, b float64) float32 {
		if a == b {
			return 0.5
		}
		return float32((level - a) / (b - a))
	}
	for j := 0; j+1 < rows; j++ {
		for i := 0; i+1 < cols; i++ {
			a, b, c, d := at(i, j), at(i+1, j), at(i+1, j+1), at(i, j+1)
			k := 0
			if a >= level {
				k |= 8
			}
			if b >= level {
				k |= 4
			}
			if c >= level {
				k |= 2
			}
			if d >= level {
				k |= 1
			}
			if k == 0 || k == 15 {
				continue
			}
			// A cell's value is at its middle, where the shading centers it.
			x, y := float32(i*cell)+cell/2, float32(j*cell)+cell/2
			top := [2]float32{x + lerp(a, b)*cell, y}
			right := [2]float32{x + cell, y + lerp(b, c)*cell}
			bottom := [2]float32{x + lerp(d, c)*cell, y + cell}
			left := [2]float32{x, y + lerp(a, d)*cell}
			seg := func(p, q [2]float32) { out = append(out, p[0], p[1], q[0], q[1]) }
			switch k {
			case 1, 14:
				seg(left, bottom)
			case 2, 13:
				seg(bottom, right)
			case 3, 12:
				seg(left, right)
			case 4, 11:
				seg(top, right)
			case 5:
				seg(left, top)
				seg(bottom, right)
			case 6, 9:
				seg(top, bottom)
			case 7, 8:
				seg(left, top)
			case 10:
				seg(left, bottom)
				seg(top, right)
			}
		}
	}
	return out
}

// hexAt returns the hexagon, pointy at the top, of radius r that x, y is
// in, by its column and row.
func hexAt(x, y, r float32) [2]int {
	w, h := float32(math.Sqrt(3))*r, 1.5*r
	row := int(math.Floor(float64(y / h)))
	best, bd := [2]int{}, float32(math.MaxFloat32)
	for dr := -1; dr <= 1; dr++ {
		rr := row + dr
		shift := float32(0)
		if rr&1 != 0 {
			shift = w / 2
		}
		col := int(math.Floor(float64((x - shift) / w)))
		for dc := 0; dc <= 1; dc++ {
			cx, cy := hexCenter([2]int{col + dc, rr}, r)
			if d := (x-cx)*(x-cx) + (y-cy)*(y-cy); d < bd {
				best, bd = [2]int{col + dc, rr}, d
			}
		}
	}
	return best
}

// hexCenter returns the middle of hexagon k of radius r.
func hexCenter(k [2]int, r float32) (float32, float32) {
	w, h := float32(math.Sqrt(3))*r, 1.5*r
	x := float32(k[0])*w + w/2
	if k[1]&1 != 0 {
		x += w / 2
	}
	return x, float32(k[1])*h + r
}

// hexPath returns the hexagon of radius r about x, y, pointy at the top,
// a hair larger to leave no seams.
func hexPath(x, y, r float32) *ui.Path {
	var p ui.Path
	r += 0.3
	for i := range 6 {
		a := math.Pi/6 + float64(i)*math.Pi/3
		px, py := x+r*float32(math.Cos(a)), y+r*float32(math.Sin(a))
		if i == 0 {
			p.MoveTo(px, py)
		} else {
			p.LineTo(px, py)
		}
	}
	return p.Close()
}

// ramp returns the color of f, from 0, the fewest, to 1, the most: light
// to dark blue in the light theme, and the other way in the dark.
func ramp(f float32, dark bool) ui.Color {
	f = min(max(f, 0), 1)
	if dark {
		return ui.Oklch(0.3+0.55*f, 0.05+0.11*f, 265-45*f)
	}
	return ui.Oklch(0.94-0.52*f, 0.03+0.14*f, 220+45*f)
}

// axisTitles draws the titles of the axes, x's under the plot pl and y's
// turned along its left in r.
func axisTitles(p *ui.Painter, t *ui.Theme, r, pl ui.Rect, x, y Axis, small float32) {
	title := func(a Axis) string {
		if a.Unit != "" {
			return a.Title + " (" + a.Unit + ")"
		}
		return a.Title
	}
	if x.Title != "" {
		span := ui.Span{Text: title(x), Size: t.FontSize - 1, Weight: 600, Color: t.TextMuted}
		w, _ := p.MeasureText(0, span)
		p.RichText(pl.X+(pl.W-w)/2, pl.Y+pl.H+small+t.Space(3), 0, span)
	}
	if y.Title != "" {
		span := ui.Span{Text: title(y), Size: t.FontSize - 1, Weight: 600, Color: t.TextMuted}
		w, h := p.MeasureText(0, span)
		cx, cy := r.X+h/2, pl.Y+pl.H/2
		p.Transform(ui.Rotation(-90), cx, cy, func() {
			p.RichText(cx-w/2, cy-h/2, 0, span)
		})
	}
}

// legend draws the scale of colors above the plot's right end.
func legend(p *ui.Painter, t *ui.Theme, r, pl ui.Rect, s *State, dark bool, small float32) {
	w := min(float32(140), pl.W/3)
	x, y := pl.X+pl.W-w, r.Y
	steps := 20
	for i := range steps {
		f := float32(i) / float32(steps-1)
		if s.Mode == Contour {
			f = float32(int(f*levels)) / levels
		}
		p.Fill(ui.Rect{X: x + float32(i)*w/float32(steps), Y: y + t.Space(1), W: w/float32(steps) + 0.5, H: t.Space(2.5)}, ramp(f, dark), 0)
	}
	lo, hi := "Fewer", "More"
	if s.Mode == Hexbin {
		lo, hi = "1", strconv.Itoa(s.most)
	}
	p.Text(x, y+t.Space(4), lo, small, t.TextMuted)
	wt, _ := p.MeasureText(0, ui.Span{Text: hi, Size: small})
	p.Text(x+w-wt, y+t.Space(4), hi, small, t.TextMuted)
}

// format writes v to two decimals at most.
func format(v float64) string {
	return strconv.FormatFloat(math.Round(v*100)/100, 'f', -1, 64)
}
