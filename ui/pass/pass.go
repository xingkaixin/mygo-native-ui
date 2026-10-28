// Package pass shows a pass as a wallet keeps it, a boarding pass or a
// ticket, on its issuer's color: the issuer and the fields wanted first,
// as the gate, at the top; the main fields large, from and to with a plane
// between them for travel; rows of smaller fields under them; and past a
// tear line, its barcode. The i button turns the pass over to the details
// on its back, and Done turns it back.
//
//	pass.View(c, &app.flipped, app.boarding)
package pass

import (
	"math"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/barcode"
)

// Field is a label and its value.
type Field struct {
	Label, Value string
}

// Pass is a pass: who issued it and its color, the fields at its top,
// its main fields, the rows of fields under them, the fields on its back,
// and its barcode. Travel puts a plane between the two main fields.
type Pass struct {
	Issuer  string
	Color   ui.Color
	Header  []Field
	Main    []Field
	Travel  bool
	Rows    [][]Field
	Back    []Field
	Barcode *barcode.Code
}

// State is what a pass keeps: whether it shows its back, and when it
// turned. Keep it in the app's state, one for each pass.
type State struct {
	back   bool
	turned time.Time
}

const turn = 450 * time.Millisecond

var plane = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><path d="M21 16v-2l-8-5V3.5a1.5 1.5 0 0 0-3 0V9l-8 5v2l8-2.5V19l-2 1.5V22l3.5-1 3.5 1v-1.5L13 19v-5.5z" fill="currentColor" transform="rotate(90 12 12)"/></svg>`))

// View creates the pass p.
func View(c *ui.Context, s *State, p Pass) ui.Element {
	t := c.Theme()
	w, h := t.Space(86), t.Space(122)
	k := 1.0
	if !s.turned.IsZero() && !c.Preferences().ReduceMotion {
		k = min(1, float64(c.Now().Sub(s.turned))/float64(turn))
	}
	if k < 1 {
		c.AnimationFrame()
	}
	eased := (1 - math.Cos(math.Pi*k)) / 2
	angle := float32(eased * 180)
	if !s.back {
		angle = 180 - angle
	}
	if s.turned.IsZero() {
		angle = 0
	}
	flip := func() { s.back, s.turned = !s.back, c.Now() }
	root := ui.Box(c).Size(w, h).Perspective(t.Space(400)).Label(p.Issuer + " pass")
	root.Children(func() {
		front(c, p, w, h, flip).RotateY(angle)
		back(c, p, w, h, flip).RotateY(angle - 180)
	})
	return root
}

// face creates a side of the pass.
func face(c *ui.Context, w, h float32, bg ui.Color) ui.Element {
	t := c.Theme()
	return ui.Column(c).Absolute().Left(0).Top(0).Size(w, h).Radius(t.Space(4)).Background(bg).Clip().BackfaceHidden().
		Shadow(0, 6, 18, 0, ui.RGBA(0, 0, 0, 0.18))
}

// front creates the front of p.
func front(c *ui.Context, p Pass, w, h float32, flip func()) ui.Element {
	t := c.Theme()
	ink := ink(p.Color)
	muted := ink.Alpha(0.7)
	label := func(s string) ui.Element {
		return ui.Text(c, strings.ToUpper(s)).FontSize(t.FontSize - 3).Bold().TextColor(muted)
	}
	e := face(c, w, h, p.Color)
	e.Children(func() {
		ui.Column(c).Grow(1).Padding(t.Space(4), t.Space(5)).Gap(t.Space(5)).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(t.Space(4)).Children(func() {
				ui.Text(c, p.Issuer).FontSize(t.FontSize + 3).Bold().TextColor(ink).Grow(1)
				for i, f := range p.Header {
					ui.Column(c.Key(i)).AlignItems(ui.End).Children(func() {
						label(f.Label)
						ui.Text(c, f.Value).FontSize(t.FontSize + 4).Bold().TextColor(ink)
					})
				}
			})
			ui.Row(c).AlignItems(ui.End).Children(func() {
				for i, f := range p.Main {
					if i > 0 && p.Travel {
						ui.Box(c).Grow(1).Center().Height(t.Space(11)).Children(func() {
							ui.Icon(c, plane).Size(t.Space(6), t.Space(6)).TextColor(ink)
						})
					}
					col := ui.Column(c.Key(i))
					if i > 0 {
						col.AlignItems(ui.End)
					}
					col.Children(func() {
						label(f.Label)
						ui.Text(c, f.Value).FontSize(t.FontSize + 22).Bold().TextColor(ink).FontFeatures("tnum")
					})
				}
			})
			for j, row := range p.Rows {
				ui.Row(c.Key(j)).Gap(t.Space(3)).Children(func() {
					for i, f := range row {
						col := ui.Column(c.Key(i)).Grow(1).Gap(t.Space(0.5))
						if i == len(row)-1 && i > 0 {
							col.AlignItems(ui.End)
						}
						col.Children(func() {
							label(f.Label)
							ui.Text(c, f.Value).FontSize(t.FontSize + 1).Bold().TextColor(ink).FontFeatures("tnum")
						})
					}
				})
			}
		})
		ui.Box(c).FillWidth().Height(t.Space(6)).Draw(func(pt *ui.Painter, r ui.Rect) {
			y := r.Y + r.H/2
			var line ui.Path
			line.MoveTo(r.X+r.H, y)
			line.LineTo(r.X+r.W-r.H, y)
			pt.StrokePath(line.Dash(0, 4, 4), 1.5, ink.Alpha(0.35))
			// The notches are holes through the pass, so they take the
			// color of what is behind it.
			for _, x := range []float32{r.X, r.X + r.W} {
				var notch ui.Path
				notch.Circle(x, y, r.H/2)
				pt.FillPath(&notch, t.Background)
			}
		})
		ui.Column(c).Padding(t.Space(3), t.Space(5), t.Space(11), t.Space(5)).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Padding(t.Space(2)).Radius(t.Space(2)).Background(ui.RGB(255, 255, 255)).Children(func() {
				barcode.View(c, p.Barcode).Size(w-t.Space(18), t.Space(22))
			})
		})
		info := ui.ButtonBase(c).Absolute().Right(t.Space(3)).Bottom(t.Space(3)).Size(t.Space(6), t.Space(6)).Radius(t.Space(3)).
			Border(1.5, muted).Center().Label("Details").Cursor(ui.CursorPointer)
		if info.Clicked() {
			flip()
		}
		info.Children(func() {
			ui.Text(c, "i").FontSize(t.FontSize - 2).Bold().TextColor(muted)
		})
	})
	return e
}

// back creates the back of p, its details.
func back(c *ui.Context, p Pass, w, h float32, flip func()) ui.Element {
	t := c.Theme()
	e := face(c, w, h, t.Surface).Border(1, t.Border)
	e.Children(func() {
		ui.Row(c).AlignItems(ui.Center).Padding(t.Space(3), t.Space(4)).Children(func() {
			ui.Text(c, p.Issuer).Bold().Grow(1)
			if ui.Button(c, "Done").Clicked() {
				flip()
			}
		})
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).Padding(0, t.Space(5), t.Space(4), t.Space(5)).Children(func() {
				for i, f := range p.Back {
					ui.Column(c.Key(i)).Gap(t.Space(1)).Padding(t.Space(3), 0).Children(func() {
						ui.Text(c, f.Label).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
						ui.Text(c, f.Value).Selectable()
					})
					if i < len(p.Back)-1 {
						ui.Box(c.Key(-1 - i)).FillWidth().Height(1).Background(t.Border)
					}
				}
			})
		})
	})
	return e
}

// ink returns black or white, whichever reads on bg.
func ink(bg ui.Color) ui.Color {
	if 0.299*float32(bg.R)+0.587*float32(bg.G)+0.114*float32(bg.B) > 160 {
		return ui.RGB(20, 20, 20)
	}
	return ui.RGB(255, 255, 255)
}
