// Package splitflap shows text on a split-flap display, as the departure
// boards of stations and airports and Vestaboard do: a flap for each
// character, whose top half falls over to show the next character on its
// drum, one after another, until it shows its new one. A flap only turns
// forward, so it may pass the whole drum to get back one letter, and
// flaps with farther to go settle later.
//
//	splitflap.Row(c, app.gate, 4, splitflap.Options{})
//	splitflap.Row(c, app.clock, 8, splitflap.Options{Height: 64, Drum: " 0123456789:"})
//
// Text is padded with blanks or cut to the row's flaps. A letter missing
// from the drum shows as its capital when the drum has that, and as a
// blank otherwise. The text shown when a row is first built shows at
// once; later text flips.
package splitflap

import (
	"math"
	"strings"
	"time"
	"unicode"

	"github.com/egoist/mygo/ui"
)

// Drum is the characters a flap turns through unless Options say
// otherwise, in order, a blank first.
const Drum = " ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789.,:/-'!?&"

// flip is how long a flap takes from one character to the next.
const flip = 60 * time.Millisecond

var (
	face  = ui.RGB(30, 30, 32)
	ink   = ui.RGB(238, 236, 228)
	shade = ui.RGB(0, 0, 0)
)

// Options are how a row looks and turns.
type Options struct {
	// Height is a flap's height in DIPs, 40 when zero. A flap is about
	// two thirds as wide.
	Height float32
	// Drum is the characters a flap turns through, in order, a blank
	// first; Drum when empty. A clock's digits turn faster on a drum of
	// digits alone.
	Drum string
}

// state is what a row keeps from frame to frame: where each flap turns
// to, in characters counted on around the drum, and where it showed.
type state struct {
	to, shown []float32
}

// Row creates a row of n flaps showing text.
func Row(c *ui.Context, text string, n int, o Options) ui.Element {
	h := o.Height
	if h <= 0 {
		h = 40
	}
	drum := []rune(o.Drum)
	if len(drum) == 0 {
		drum = []rune(Drum)
	}
	w := float32(math.Round(float64(h * 0.68)))
	row := ui.Row(c).Gap(max(2, h/16)).Label(strings.TrimSpace(text))
	st := ui.Local(row, "splitflap.state", func() state { return state{} })
	for len(st.to) < n {
		st.to, st.shown = append(st.to, 0), append(st.shown, 0)
	}
	runes := []rune(text)
	row.Children(func() {
		for i := range n {
			r := ' '
			if i < len(runes) {
				r = runes[i]
			}
			at := st.to[i]
			st.to[i] += float32(mod(find(drum, r)-mod(int(at), len(drum)), len(drum)))
			cell := ui.Box(c.Key(i)).Size(w, h)
			a := ui.Animation{Duration: time.Duration(st.to[i]-st.shown[i]) * flip, Ease: ui.Linear}
			pos := cell.AnimateTo("splitflap.at", st.to[i], a)
			st.shown[i] = pos
			flap(c, cell, drum, pos, w, h)
		}
	})
	return row
}

// flap fills cell with a flap at pos on drum: between two characters, the
// top half of the one it leaves falls, then the bottom half of the one it
// comes to falls into place.
func flap(c *ui.Context, cell ui.Element, drum []rune, pos, w, h float32) {
	base := math.Floor(float64(pos))
	f := pos - float32(base)
	from := drum[mod(int(base), len(drum))]
	to := drum[mod(int(base)+1, len(drum))]
	if f < 0.001 {
		to = from
	}
	rad := max(2, h/12)
	cell.DrawOver(func(p *ui.Painter, r ui.Rect) {
		p.Fill(ui.Rect{X: r.X, Y: r.Y + r.H/2 - 0.5, W: r.W, H: 1}, shade.Alpha(0.7), 0)
	}).Children(func() {
		half(c, to, true, 0, w, h, rad)
		half(c, from, false, 0, w, h, rad)
		switch {
		case f < 0.001:
		case f < 0.5:
			half(c, from, true, -180*f, w, h, rad)
		default:
			half(c, to, false, 180*(1-f), w, h, rad)
		}
	})
}

// half builds the top or the bottom half of a flap showing r, turned deg
// degrees about the hinge between the halves; it darkens as it turns
// toward the viewer.
func half(c *ui.Context, r rune, top bool, deg, w, h, rad float32) {
	y, hinge := float32(0), float32(1)
	radii := []float32{rad, rad, 0, 0}
	bg := face.Mix(ui.RGB(255, 255, 255), 0.04)
	if !top {
		y, hinge = h/2, 0
		radii = []float32{0, 0, rad, rad}
		bg = face
	}
	b := ui.Box(c).Absolute().Left(0).Top(y).Size(w, h/2).Clip().Radius(radii...)
	if deg != 0 {
		turn := float32(math.Abs(math.Sin(float64(deg) * math.Pi / 180)))
		bg = bg.Mix(shade, 0.55*turn)
		b.TransformOrigin(0.5, hinge).RotateX(deg).Perspective(h * 5)
	}
	b.Background(bg).Children(func() {
		ui.Box(c).Absolute().Left(0).Top(-y).Size(w, h).Center().Children(func() {
			ui.Text(c, string(r)).Font("monospace").FontSize(h * 0.62).Bold().TextColor(ink)
		})
	})
}

// find returns where r is on drum, or its capital, or 0, the blank.
func find(drum []rune, r rune) int {
	for _, want := range []rune{r, unicode.ToUpper(r)} {
		for i, d := range drum {
			if d == want {
				return i
			}
		}
	}
	return 0
}

func mod(a, n int) int { return (a%n + n) % n }
