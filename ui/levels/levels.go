// Package levels adjusts a picture's levels, as photo editors' Levels
// dialog does: the histogram of the picture's luminance, or of one of its
// channels, with the input levels as three triangles under it, black,
// gray and white, and the output levels as two over a black-to-white bar.
// Dragging the black and white triangles sets the shades that become
// black and white, the part of the histogram clipped shaded; the gray
// triangle sets the midtones' gamma, and stays between them as they move.
// Auto sets black and white where a tenth of a percent of the pixels lie
// beyond them, for each channel, and Reset puts the channel back.
//
// The levels of each channel apply first, then those of RGB: Apply makes
// the adjusted picture, which the app makes again when Changed reports an
// edit.
//
//	if levels.View(c, &app.levels, app.photo, &app.adjust).Changed() {
//		app.preview = ui.NewBitmap(levels.Apply(app.photo, app.adjust))
//	}
package levels

import (
	"fmt"
	"image"
	"image/draw"
	"math"

	"github.com/egoist/mygo/ui"
)

// Levels maps a channel's shades: those from InBlack to InWhite, from 0
// to 255, spread over OutBlack to OutWhite, bent by Gamma, above 1
// brightening the midtones. The zero Levels changes nothing.
type Levels struct {
	InBlack, InWhite   float64
	Gamma              float64
	OutBlack, OutWhite float64
}

var identity = Levels{InWhite: 255, Gamma: 1, OutWhite: 255}

func (l Levels) norm() Levels {
	if l == (Levels{}) {
		return identity
	}
	return l
}

// Map returns where the shade v, from 0 to 255, goes.
func (l Levels) Map(v float64) float64 {
	l = l.norm()
	t := (v - l.InBlack) / max(1, l.InWhite-l.InBlack)
	t = math.Pow(max(0, min(1, t)), 1/max(0.01, l.Gamma))
	return l.OutBlack + (l.OutWhite-l.OutBlack)*t
}

// Channel is which levels apply: those of all three, or of red, green or
// blue alone.
type Channel uint8

const (
	RGB Channel = iota
	Red
	Green
	Blue
)

// Adjustment is the levels of each channel, by Channel.
type Adjustment [4]Levels

// Apply returns img with adj applied.
func Apply(img image.Image, adj Adjustment) *image.RGBA {
	var lut [3][256]uint8
	for ch := range 3 {
		for v := range 256 {
			lut[ch][v] = uint8(math.Round(adj[RGB].Map(adj[ch+1].Map(float64(v)))))
		}
	}
	out := image.NewRGBA(img.Bounds())
	draw.Draw(out, out.Rect, img, img.Bounds().Min, draw.Src)
	for i := 0; i < len(out.Pix); i += 4 {
		out.Pix[i] = lut[0][out.Pix[i]]
		out.Pix[i+1] = lut[1][out.Pix[i+1]]
		out.Pix[i+2] = lut[2][out.Pix[i+2]]
	}
	return out
}

type histogram [4][256]int

func count(img image.Image) *histogram {
	var h histogram
	b := img.Bounds()
	rgba, ok := img.(*image.RGBA)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			var r, g, bl uint8
			if ok {
				i := rgba.PixOffset(x, y)
				r, g, bl = rgba.Pix[i], rgba.Pix[i+1], rgba.Pix[i+2]
			} else {
				cr, cg, cb, _ := img.At(x, y).RGBA()
				r, g, bl = uint8(cr>>8), uint8(cg>>8), uint8(cb>>8)
			}
			lum := (299*int(r) + 587*int(g) + 114*int(bl)) / 1000
			h[RGB][lum]++
			h[Red][r]++
			h[Green][g]++
			h[Blue][bl]++
		}
	}
	return &h
}

// State is what a levels panel keeps: the channel shown, the picture's
// histogram and the triangle dragged. Keep it in the app's state, one for
// each panel.
type State struct {
	channel int
	img     image.Image
	hist    *histogram
	drag    int // the triangle dragged, plus 1
}

var (
	channelNames  = []string{"RGB", "Red", "Green", "Blue"}
	channelColors = []ui.Color{ui.RGB(128, 128, 128), ui.Hex("#E03131"), ui.Hex("#2F9E44"), ui.Hex("#1C7ED6")}
)

// View creates the levels panel of *adj for img.
func View(c *ui.Context, s *State, img image.Image, adj *Adjustment) ui.Element {
	t := c.Theme()
	if s.img != img {
		s.img, s.hist = img, count(img)
	}
	for i := range adj {
		adj[i] = adj[i].norm()
	}
	changed := false
	root := ui.Column(c).Gap(t.Space(3)).Width(t.Space(76)).Label("Levels")
	root.Children(func() {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Segmented(c, &s.channel, channelNames...).Label("Channel")
			ui.Box(c).Grow(1)
			if ui.Button(c, "Auto").Tooltip("Clip a tenth of a percent at each end").Clicked() {
				auto(s.hist, adj)
				changed = true
			}
		})
		l := &adj[s.channel]
		hist(c, s, *l)
		if input(c, s, l) {
			changed = true
		}
		ui.Row(c).Children(func() {
			for i, v := range []string{fmt.Sprintf("%.0f", l.InBlack), fmt.Sprintf("%.2f", l.Gamma), fmt.Sprintf("%.0f", l.InWhite)} {
				if i > 0 {
					ui.Box(c.Key(i)).Grow(1)
				}
				ui.Text(c.Key(-1-i), v).FontFeatures("tnum").TextColor(t.TextMuted)
			}
		})
		ui.Text(c, "Output levels").FontSize(t.FontSize - 2).TextColor(t.TextMuted)
		if output(c, s, l) {
			changed = true
		}
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, fmt.Sprintf("%.0f", l.OutBlack)).FontFeatures("tnum").TextColor(t.TextMuted)
			ui.Box(c).Grow(1)
			if ui.Button(c, "Reset "+channelNames[s.channel]).Disabled(*l == identity).Clicked() {
				*l = identity
				changed = true
			}
			ui.Box(c).Grow(1)
			ui.Text(c, fmt.Sprintf("%.0f", l.OutWhite)).FontFeatures("tnum").TextColor(t.TextMuted)
		})
	})
	if changed {
		root.SetChanged()
	}
	return root
}

// hist draws the histogram of the channel shown, shading what l clips.
func hist(c *ui.Context, s *State, l Levels) {
	t := c.Theme()
	h := s.hist[s.channel]
	col := channelColors[s.channel]
	ui.Box(c).FillWidth().Height(t.Space(30)).Radius(t.Space(1)).Background(t.Text.Alpha(0.04)).Role(ui.RoleImage).
		Label(channelNames[s.channel] + " histogram").Draw(func(p *ui.Painter, r ui.Rect) {
		top := 1
		for _, n := range h {
			top = max(top, n)
		}
		// A square root keeps the few darkest and lightest shades in view
		// beside a peak.
		scale := func(n int) float32 { return float32(math.Sqrt(float64(n) / float64(top))) }
		w := r.W / 256
		var path ui.Path
		path.MoveTo(r.X, r.Y+r.H)
		for i, n := range h {
			x := r.X + float32(i)*w
			y := r.Y + r.H - scale(n)*(r.H-4)
			path.LineTo(x, y)
			path.LineTo(x+w, y)
		}
		path.LineTo(r.X+r.W, r.Y+r.H)
		path.Close()
		p.FillPath(&path, col.Alpha(0.75))
		shade := t.Background.Alpha(0.6)
		if lb := float32(l.InBlack) * w; lb > 0 {
			p.Fill(ui.Rect{X: r.X, Y: r.Y, W: lb, H: r.H}, shade, 0)
		}
		if lw := float32(l.InWhite) * w; lw < r.W {
			p.Fill(ui.Rect{X: r.X + lw, Y: r.Y, W: r.W - lw, H: r.H}, shade, 0)
		}
	})
}

// triangle draws a slider's triangle pointing up at x.
func triangle(p *ui.Painter, x, y float32, fill, edge ui.Color) {
	var tri ui.Path
	tri.MoveTo(x, y)
	tri.LineTo(x+7, y+11)
	tri.LineTo(x-7, y+11)
	tri.Close()
	p.FillPath(&tri, fill)
	p.StrokePath(&tri, 1, edge)
}

// track is a row of triangles to drag along a width standing for 0 to
// 255: it returns the track, and while one is dragged, which and to what
// shade.
func track(c *ui.Context, s *State, key string, xs []float64, first int) (ui.Element, int, float64) {
	t := c.Theme()
	e := ui.Box(c.Key(key)).FillWidth().Height(t.Space(4)).Cursor(ui.CursorResizeEW)
	px, _, _ := e.PointerPosition()
	w := e.Bounds().W
	if w <= 0 {
		return e, -1, 0
	}
	v := max(0, min(255, float64(px/w*255)))
	if e.Pressed() && s.drag == 0 {
		best := math.Inf(1)
		for i, x := range xs {
			if d := math.Abs(x - v); d < best {
				best, s.drag = d, first+i+1
			}
		}
	}
	if !e.Pressed() {
		if s.drag > first && s.drag <= first+len(xs) {
			s.drag = 0
		}
		return e, -1, 0
	}
	if s.drag > first && s.drag <= first+len(xs) {
		return e, s.drag - first - 1, v
	}
	return e, -1, 0
}

// input creates the input triangles of l.
func input(c *ui.Context, s *State, l *Levels) bool {
	mid := l.InBlack + (l.InWhite-l.InBlack)*math.Pow(0.5, l.Gamma)
	e, which, v := track(c, s, "in", []float64{l.InBlack, mid, l.InWhite}, 0)
	changed := which >= 0
	switch which {
	case 0:
		l.InBlack = math.Round(min(v, l.InWhite-2))
	case 1:
		f := (v - l.InBlack) / max(1, l.InWhite-l.InBlack)
		f = max(0.001, min(0.999, f))
		l.Gamma = max(0.1, min(9.99, math.Round(math.Log(f)/math.Log(0.5)*100)/100))
	case 2:
		l.InWhite = math.Round(max(v, l.InBlack+2))
	}
	lv := *l
	mid = lv.InBlack + (lv.InWhite-lv.InBlack)*math.Pow(0.5, lv.Gamma)
	e.Label("Input levels").Value(fmt.Sprintf("%.0f, %.2f, %.0f", lv.InBlack, lv.Gamma, lv.InWhite))
	e.Draw(func(p *ui.Painter, r ui.Rect) {
		at := func(v float64) float32 { return r.X + float32(v/255)*r.W }
		triangle(p, at(lv.InBlack), r.Y, ui.RGB(0, 0, 0), ui.RGB(0, 0, 0))
		triangle(p, at(mid), r.Y, ui.RGB(140, 140, 140), ui.RGB(90, 90, 90))
		triangle(p, at(lv.InWhite), r.Y, ui.RGB(255, 255, 255), ui.RGB(90, 90, 90))
	})
	return changed
}

// output creates the bar of output levels and its triangles.
func output(c *ui.Context, s *State, l *Levels) bool {
	t := c.Theme()
	ui.Box(c).FillWidth().Height(t.Space(3)).Radius(t.Space(0.5)).
		LinearGradient(ui.LinearGradient{From: ui.RGB(0, 0, 0), To: ui.RGB(255, 255, 255), Angle: 90})
	e, which, v := track(c, s, "out", []float64{l.OutBlack, l.OutWhite}, 3)
	switch which {
	case 0:
		l.OutBlack = math.Round(min(v, l.OutWhite))
	case 1:
		l.OutWhite = math.Round(max(v, l.OutBlack))
	}
	lv := *l
	e.Label("Output levels").Value(fmt.Sprintf("%.0f, %.0f", lv.OutBlack, lv.OutWhite))
	e.Draw(func(p *ui.Painter, r ui.Rect) {
		at := func(v float64) float32 { return r.X + float32(v/255)*r.W }
		triangle(p, at(lv.OutBlack), r.Y, ui.RGB(0, 0, 0), ui.RGB(0, 0, 0))
		triangle(p, at(lv.OutWhite), r.Y, ui.RGB(255, 255, 255), ui.RGB(90, 90, 90))
	})
	return which >= 0
}

// auto sets each channel's black and white where a tenth of a percent of
// its pixels lie beyond them.
func auto(h *histogram, adj *Adjustment) {
	total := 0
	for _, n := range h[RGB] {
		total += n
	}
	clip := total / 1000
	for ch := Red; ch <= Blue; ch++ {
		lo, hi, sum := 0, 255, 0
		for lo < 254 && sum+h[ch][lo] <= clip {
			sum += h[ch][lo]
			lo++
		}
		sum = 0
		for hi > lo+2 && sum+h[ch][hi] <= clip {
			sum += h[ch][hi]
			hi--
		}
		adj[ch] = Levels{InBlack: float64(lo), InWhite: float64(hi), Gamma: 1, OutWhite: 255}
	}
	adj[RGB] = identity
}
