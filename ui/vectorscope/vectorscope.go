// Package vectorscope draws the scopes of color grading for a picture, as
// video editors show them beside the viewer: the vectorscope, each pixel
// at the angle of its hue and as far out as it is saturated, with the
// targets of 75% color bars and the skin tone line; the waveform, each
// column's pixels by their luma from 0 to 100 IRE; and the RGB parade,
// the waveforms of red, green and blue side by side. Traces are brighter
// where more pixels fall. A switch zooms the vectorscope in twice for
// pictures of little color, and the pointer reads out the angle and
// saturation, or the level, under it. The traces are worked out again
// only when the app passes another picture.
//
//	vectorscope.View(c, &app.scopes, app.frame)
package vectorscope

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/egoist/mygo/ui"
)

const (
	vectorscope = iota
	waveform
	parade
)

// State is what the scopes keep: the scope shown, whether the vectorscope
// is zoomed, and the traces of the picture. Keep it in the app's state,
// one for each picture shown.
type State struct {
	scope  int
	zoom   bool
	img    image.Image
	zoomed bool
	traces [3]*ui.Bitmap
}

var (
	screen = ui.RGB(16, 17, 19)
	grid   = ui.RGBA(255, 255, 255, 0.16)
	label  = ui.RGBA(255, 255, 255, 0.5)
)

// luma returns the BT.709 luma and chroma of a color from 0 to 1, Cb and
// Cr from -0.5 to 0.5.
func luma(r, g, b float64) (y, cb, cr float64) {
	y = 0.2126*r + 0.7152*g + 0.0722*b
	return y, (b - y) / 1.8556, (r - y) / 1.5748
}

// each calls fn for the pixels of img, every few of a large one, with
// its column from 0 and its color from 0 to 1.
func each(img image.Image, fn func(x int, r, g, b float64)) {
	bounds := img.Bounds()
	step := max(1, int(math.Sqrt(float64(bounds.Dx()*bounds.Dy())/250000)))
	for y := bounds.Min.Y; y < bounds.Max.Y; y += step {
		for x := bounds.Min.X; x < bounds.Max.X; x += step {
			r, g, b, _ := img.At(x, y).RGBA()
			fn(x-bounds.Min.X, float64(r)/65535, float64(g)/65535, float64(b)/65535)
		}
	}
}

// glow returns how bright a trace is where n of the most, most, fall, on
// a log scale so that a few pixels still show.
func glow(n, most int) float64 {
	if n == 0 {
		return 0
	}
	return min(1, 0.25+0.95*math.Log1p(float64(n))/math.Log1p(float64(most)))
}

const (
	bins   = 256
	levels = 160
)

func vectors(img image.Image, zoom float64) *ui.Bitmap {
	counts := make([]int, bins*bins)
	most := 0
	each(img, func(_ int, r, g, b float64) {
		_, cb, cr := luma(r, g, b)
		x, y := int((0.5+cb*zoom)*bins), int((0.5-cr*zoom)*bins)
		if x < 0 || y < 0 || x >= bins || y >= bins {
			return
		}
		counts[y*bins+x]++
		most = max(most, counts[y*bins+x])
	})
	out := image.NewNRGBA(image.Rect(0, 0, bins, bins))
	for i, n := range counts {
		if n == 0 {
			continue
		}
		v := glow(n, most)
		cb := (float64(i%bins)/bins - 0.5) / zoom
		cr := (0.5 - float64(i/bins)/bins) / zoom
		// The trace takes the hue of where it falls, lifted toward white
		// where it is dense.
		y := 0.55 + 0.35*v
		col := [3]float64{y + 1.5748*cr, y - 0.1873*cb - 0.4681*cr, y + 1.8556*cb}
		for k := range col {
			col[k] = max(0, min(1, col[k]*(1-v*0.3)+v*0.3))
		}
		out.SetNRGBA(i%bins, i/bins, color.NRGBA{uint8(col[0] * 255), uint8(col[1] * 255), uint8(col[2] * 255), uint8(v * 255)})
	}
	return ui.NewBitmap(out)
}

// waves returns the waveforms of luma, or, for a parade, of red, green
// and blue side by side.
func waves(img image.Image, parade bool) *ui.Bitmap {
	cols := min(img.Bounds().Dx(), 240)
	lanes := 1
	if parade {
		cols, lanes = min(cols, 120), 3
	}
	counts := make([][]int, lanes)
	for k := range counts {
		counts[k] = make([]int, cols*levels)
	}
	most := 0
	dx := img.Bounds().Dx()
	each(img, func(x int, r, g, b float64) {
		col := x * cols / dx
		vs := []float64{r, g, b}
		if !parade {
			y, _, _ := luma(r, g, b)
			vs = []float64{y}
		}
		for k, v := range vs {
			i := (levels-1-int(v*(levels-1)))*cols + col
			counts[k][i]++
			most = max(most, counts[k][i])
		}
	})
	tints := [][3]float64{{0.45, 1, 0.6}}
	if parade {
		tints = [][3]float64{{1, 0.35, 0.3}, {0.35, 1, 0.45}, {0.35, 0.6, 1}}
	}
	out := image.NewNRGBA(image.Rect(0, 0, cols*lanes, levels))
	for k, lane := range counts {
		for i, n := range lane {
			if n == 0 {
				continue
			}
			v := glow(n, most)
			var px [3]uint8
			for j, tint := range tints[k] {
				px[j] = uint8(255 * min(1, tint*(0.7+0.3*v)+v*0.25))
			}
			out.SetNRGBA(k*cols+i%cols, i/cols, color.NRGBA{px[0], px[1], px[2], uint8(v * 255)})
		}
	}
	return ui.NewBitmap(out)
}

// targets are where 75% color bars fall on the vectorscope.
var targets = func() (out []struct {
	name   string
	cb, cr float64
}) {
	for _, bar := range []struct {
		name    string
		r, g, b float64
	}{{"R", 1, 0, 0}, {"Mg", 1, 0, 1}, {"B", 0, 0, 1}, {"Cy", 0, 1, 1}, {"G", 0, 1, 0}, {"Yl", 1, 1, 0}} {
		_, cb, cr := luma(bar.r*0.75, bar.g*0.75, bar.b*0.75)
		out = append(out, struct {
			name   string
			cb, cr float64
		}{bar.name, cb, cr})
	}
	return out
}()

// skinTone is the angle of the line that skin of any color falls near,
// counterclockwise from the Cb axis.
const skinTone = 123.0

// View creates the scopes of img.
func View(c *ui.Context, s *State, img image.Image) ui.Element {
	t := c.Theme()
	if s.img != img || s.zoomed != s.zoom {
		s.img, s.zoomed, s.traces = img, s.zoom, [3]*ui.Bitmap{}
	}
	scope := s.scope
	if s.traces[scope] == nil && img != nil {
		switch scope {
		case vectorscope:
			zoom := 1.0
			if s.zoom {
				zoom = 2
			}
			s.traces[scope] = vectors(img, zoom)
		default:
			s.traces[scope] = waves(img, scope == parade)
		}
	}
	root := ui.Column(c).Gap(t.Space(3))
	root.Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Segmented(c, &s.scope, "Vectorscope", "Waveform", "Parade").Label("Scope")
			ui.Spacer(c)
			if scope == vectorscope {
				ui.Toggle(c, &s.zoom, "2×").Tooltip("Zoom in on pictures of little color")
			}
		})
		ratio := float32(1.6)
		if scope == vectorscope {
			ratio = 1
		}
		panel := ui.Box(c).FillWidth().AspectRatio(ratio).Radius(t.Radius).Background(screen).Clip().
			Role(ui.RoleImage).Label([...]string{"Vectorscope", "Waveform", "RGB parade"}[scope])
		px, py, in := panel.PointerPosition()
		trace := s.traces[scope]
		panel.Draw(func(p *ui.Painter, r ui.Rect) {
			if trace == nil {
				return
			}
			if scope == vectorscope {
				s.drawVectors(p, r, trace, px, py, in)
			} else {
				drawWaves(p, r, trace, scope == parade, py, in)
			}
		})
	})
	return root
}

func (s *State) drawVectors(p *ui.Painter, r ui.Rect, trace *ui.Bitmap, px, py float32, in bool) {
	t := p.Theme()
	cx, cy := r.X+r.W/2, r.Y+r.H/2
	rad := min(r.W, r.H)/2 - t.Space(4)
	zoom := float32(1)
	if s.zoom {
		zoom = 2
	}
	var ring ui.Path
	p.StrokePath(ring.Circle(cx, cy, rad), 1, grid)
	var half ui.Path
	p.StrokePath(half.Circle(cx, cy, rad/2).Dash(0, 3, 4), 1, grid)
	p.Line(cx-rad, cy, cx+rad, cy, 1, grid)
	p.Line(cx, cy-rad, cx, cy+rad, 1, grid)
	a := skinTone * math.Pi / 180
	var skin ui.Path
	skin.MoveTo(cx, cy).LineTo(cx+rad*float32(math.Cos(a)), cy-rad*float32(math.Sin(a)))
	p.StrokePath(&skin, 1, ui.RGBA(255, 196, 150, 0.5))
	p.Image(trace, ui.Rect{X: cx - rad, Y: cy - rad, W: 2 * rad, H: 2 * rad}, ui.FillBox)
	box := t.Space(2.5)
	for _, tg := range targets {
		x, y := cx+float32(tg.cb)*2*rad*zoom, cy-float32(tg.cr)*2*rad*zoom
		if math.Hypot(float64(x-cx), float64(y-cy)) > float64(rad) {
			continue
		}
		p.Stroke(ui.Rect{X: x - box/2, Y: y - box/2, W: box, H: box}, label, 1, 1)
		// Labels sit outside the box, away from the middle.
		dx, dy := x-cx, y-cy
		d := float32(math.Hypot(float64(dx), float64(dy)))
		p.Text(x+dx/d*box*1.6-box/2, y+dy/d*box*1.6-box/2, tg.name, t.FontSize-3, label)
	}
	if !in {
		return
	}
	cb, cr := float64((px-r.W/2)/(2*rad*zoom)), float64(-(py-r.H/2)/(2*rad*zoom))
	angle := math.Mod(math.Atan2(cr, cb)*180/math.Pi+360, 360)
	read := fmt.Sprintf("%.0f°  %.0f%%", angle, math.Hypot(cb, cr)*200)
	p.Text(r.X+t.Space(3), r.Y+r.H-t.Space(3)-t.FontSize, read, t.FontSize-2, ui.RGBA(255, 255, 255, 0.8))
}

func drawWaves(p *ui.Painter, r ui.Rect, trace *ui.Bitmap, parade bool, py float32, in bool) {
	t := p.Theme()
	left := r.X + t.Space(8)
	top, bottom := r.Y+t.Space(3), r.Y+r.H-t.Space(3)
	right := r.X + r.W - t.Space(3)
	h := bottom - top
	for ire := 0; ire <= 100; ire += 10 {
		y := bottom - h*float32(ire)/100
		p.Line(left, y, right, y, 1, grid)
		if ire%20 == 0 {
			p.Text(r.X+t.Space(1.5), y-t.FontSize/2, fmt.Sprint(ire), t.FontSize-3, label)
		}
	}
	area := ui.Rect{X: left, Y: top, W: right - left, H: h}
	if !parade {
		p.Image(trace, area, ui.FillBox)
	} else {
		w, _ := trace.Size()
		gap := t.Space(2)
		lane := (area.W - 2*gap) / 3
		for k := range 3 {
			dst := ui.Rect{X: area.X + float32(k)*(lane+gap), Y: area.Y, W: lane, H: area.H}
			p.ImageRegion(trace, ui.Rect{X: float32(k * w / 3), W: float32(w / 3), H: levels}, dst, ui.ImageOptions{})
		}
	}
	py += r.Y
	if in && py >= top && py <= bottom {
		p.Line(left, py, right, py, 1, ui.RGBA(255, 255, 255, 0.4))
		p.Text(right-t.Space(14), py-t.FontSize-2, fmt.Sprintf("%.0f IRE", (bottom-py)/h*100), t.FontSize-2, ui.RGBA(255, 255, 255, 0.8))
	}
}
