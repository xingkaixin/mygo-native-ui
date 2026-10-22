// Package tex typesets TeX math, as LaTeX's display math: italic
// variables, upright digits, operators and function names, the room TeX
// puts around each kind of atom, and display, text, script and
// scriptscript styles with the sizes and shifts of an OpenType math
// font's MATH table. It sets fractions, binomials, roots, sub- and
// superscripts and primes, large operators with limits above and below in
// display style and beside in text, delimiters that grow with what they
// hold (\left, \middle, \right, \big and the others), matrices, cases and
// aligned equations, accents, braces with labels, \text, \mathbb and the
// other alphabets, spacing commands and \color. A command it does not
// know, a brace without its pair and the like show red, where they are,
// and the rest of the formula typesets around them; Err tells what is
// wrong.
//
//	tex.View(c, `x = \frac{-b \pm \sqrt{b^2 - 4ac}}{2a}`)
//
// It is what markdown's Math option asks for, to show $$…$$ blocks:
//
//	app.md.Math = func(c *ui.Context, src string) { tex.View(c, src) }
//
// The glyphs are the system's math font, drawn from its outlines: STIX
// Two Math on macOS, Cambria Math on Windows, and STIX Two, Latin Modern,
// TeX Gyre, DejaVu or Noto Sans Math where Linux has one. Without one,
// the system's serif font sets them, larger for large delimiters.
package tex

import (
	"github.com/egoist/mygo/ui"
	"golang.org/x/image/font/sfnt"
)

// formula is a formula laid out: its box and items, and the edges of its
// ink, which may stick out of the box.
type formula struct {
	box
	x0, y0, x1, y1 float32
}

type cache struct {
	src  string
	size float32
	f    *formula
}

// View typesets src, TeX math, in display style, a fifth larger than the
// theme's text and in its colors, in the middle of the width it gets,
// scrolling sideways when it is wider. Assistive technology reads src.
func View(c *ui.Context, src string) ui.Element {
	t := c.Theme()
	size := t.FontSize * 1.2
	return ui.Row(c).AlignSelf(ui.Stretch).Justify(ui.Center).Children(func() {
		ui.ScrollHorizontal(c).MaxWidthPercent(100).Children(func() {
			box := ui.Box(c)
			k := ui.Local(box, "tex.formula", func() cache { return cache{} })
			if k.f == nil || k.src != src || k.size != size {
				*k = cache{src: src, size: size, f: typeset(src, size)}
			}
			f := k.f
			box.Size(f.x1-f.x0, f.y1-f.y0).Role(ui.RoleImage).Label(src).Draw(func(p *ui.Painter, r ui.Rect) {
				f.paint(p, t, r.X-f.x0, r.Y-f.y0)
			})
		})
	})
}

func typeset(src string, size float32) *formula {
	atoms, _ := parse(src)
	l := &layout{m: mathFace(), size: size}
	f := &formula{box: l.list(atoms, display)}
	f.x0, f.y0, f.x1, f.y1 = 0, -f.h, f.w, f.d
	for _, it := range f.items {
		x0, y0, x1, y1 := it.x, it.y, it.x+it.w, it.y+it.h
		switch {
		case it.g != nil:
			x0, y0, x1, y1 = it.x+it.g.x0*it.em, it.y+it.g.y0*it.em, it.x+it.g.x1*it.em, it.y+it.g.y1*it.em
		case len(it.sys) > 0:
			last := it.sys[len(it.sys)-1]
			x1, y0, y1 = it.x+last.X+last.Advance, it.y-it.em, it.y+0.3*it.em
		}
		f.x0, f.y0, f.x1, f.y1 = min(f.x0, x0), min(f.y0, y0), max(f.x1, x1), max(f.y1, y1)
	}
	return f
}

// paint draws f with its origin at x, y, in the colors of t.
func (f *formula) paint(p *ui.Painter, t *ui.Theme, x, y float32) {
	var path ui.Path
	for _, it := range f.items {
		c := it.ink.color(t)
		switch {
		case it.sys != nil:
			p.Glyphs(it.sys, x+it.x, y+it.y, c)
			continue
		case it.g != nil:
			it.g.path(&path, x+it.x, y+it.y, it.em)
		default:
			x0, y0 := x+it.x, y+it.y
			path.Reset().MoveTo(x0, y0).LineTo(x0+it.w, y0).LineTo(x0+it.w, y0+it.h).LineTo(x0, y0+it.h).Close()
		}
		p.FillPath(&path, c)
	}
}

// path sets p to o's outline with its origin at x, y, em DIPs to the em.
func (o *outline) path(p *ui.Path, x, y, em float32) {
	p.Reset()
	pts := o.pts
	pt := func() (float32, float32) {
		q := pts[0]
		pts = pts[1:]
		return x + q[0]*em, y + q[1]*em
	}
	for i, op := range o.ops {
		switch op {
		case sfnt.SegmentOpMoveTo:
			if i > 0 {
				p.Close()
			}
			p.MoveTo(pt())
		case sfnt.SegmentOpLineTo:
			p.LineTo(pt())
		case sfnt.SegmentOpQuadTo:
			cx, cy := pt()
			x1, y1 := pt()
			p.QuadTo(cx, cy, x1, y1)
		case sfnt.SegmentOpCubeTo:
			c1x, c1y := pt()
			c2x, c2y := pt()
			x1, y1 := pt()
			p.CubeTo(c1x, c1y, c2x, c2y, x1, y1)
		}
	}
	if len(o.ops) > 0 {
		p.Close()
	}
}
