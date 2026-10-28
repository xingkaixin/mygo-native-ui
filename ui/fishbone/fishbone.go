// Package fishbone draws a cause-and-effect diagram, as Ishikawa drew
// them: the effect in a box at the head, a spine running to it, and a
// bone slanting off the spine for each category of causes, above and
// below in turn, its name at the far end and its causes along it, each on
// a line of its own. Clicking a cause marks it as a likely root cause,
// in the accent color, and clicking it again unmarks it.
//
//	fishbone.View(c, &app.diagram, "Late deliveries", app.bones)
package fishbone

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// Bone is a category of causes.
type Bone struct {
	Name   string
	Causes []Cause
}

// Cause is a cause of the effect.
type Cause struct {
	Text string
	// Likely marks a likely root cause; a click flips it.
	Likely bool
}

// State is what a diagram keeps: where its causes were drawn and the one
// under the pointer. Keep it in the app's state, one for each diagram.
type State struct {
	hits  []hit
	hover int
}

type hit struct {
	r          ui.Rect
	bone, item int
}

// slope is how steeply the bones rise off the spine, in degrees.
const slope = 62

// View creates the diagram of effect and its causes in bones. Clicks
// write to the Likely of the causes in bones.
func View(c *ui.Context, s *State, effect string, bones []Bone) ui.Element {
	t := c.Theme()
	view := ui.Box(c).Role(ui.RoleImage).Label("Causes of " + effect)
	hover := -1
	if px, py, ok := view.PointerPosition(); ok {
		for i, h := range s.hits {
			if px >= h.r.X && px < h.r.X+h.r.W && py >= h.r.Y && py < h.r.Y+h.r.H {
				hover = i
			}
		}
	}
	if hover+1 != s.hover {
		s.hover = hover + 1
		c.Invalidate()
	}
	if hover >= 0 {
		view.Cursor(ui.CursorPointer)
		if view.Clicked() {
			h := s.hits[hover]
			bones[h.bone].Causes[h.item].Likely = !bones[h.bone].Causes[h.item].Likely
		}
	}
	view.Draw(func(p *ui.Painter, r ui.Rect) {
		s.hits = s.hits[:0]
		small := t.FontSize - 1
		pad := t.Space(4)
		cy := r.Y + r.H/2
		headW := min(r.W*0.2, t.Space(48))
		headSp := ui.Span{Text: effect, Size: t.FontSize + 1, Weight: 700, Color: t.Text}
		_, hh := p.MeasureText(headW-2*t.Space(3), headSp)
		head := ui.Rect{X: r.X + r.W - headW, Y: cy - hh/2 - t.Space(3), W: headW, H: hh + 2*t.Space(3)}
		p.Fill(head, t.Accent.Alpha(0.12), t.Radius+2)
		p.Stroke(head, t.Accent, t.Radius+2, 1.5)
		p.RichText(head.X+t.Space(3), head.Y+t.Space(3), headW-2*t.Space(3), headSp)
		spineEnd := head.X - t.Space(2)
		ink := t.TextMuted
		p.StrokePolyline([][2]float32{{r.X + pad, cy}, {spineEnd, cy}}, 3, ink)
		p.FillPath(new(ui.Path).MoveTo(spineEnd+t.Space(1.5), cy).LineTo(spineEnd-t.Space(2), cy-t.Space(2)).
			LineTo(spineEnd-t.Space(2), cy+t.Space(2)).Close(), ink)
		up := (len(bones) + 1) / 2
		tagH := small * 1.9
		rise := r.H/2 - pad - tagH
		run := rise / float32(math.Tan(slope*math.Pi/180))
		for side := range 2 {
			n := up
			if side == 1 {
				n = len(bones) - up
			}
			if n == 0 {
				continue
			}
			span := (spineEnd - t.Space(4)) - (r.X + pad + run)
			gap := span / float32(n)
			dir := float32(-1)
			if side == 1 {
				dir = 1
			}
			for k := range n {
				bi := side*up + k
				b := bones[bi]
				// The bone meets the spine at bx and ends run to its left.
				bx := r.X + pad + run + gap*float32(k+1) - gap*0.15
				ex, ey := bx-run, cy+dir*rise
				p.StrokePolyline([][2]float32{{bx, cy}, {ex, ey}}, 2, ink)
				tag := ui.Span{Text: b.Name, Size: small, Weight: 600, Color: t.Text}
				tw, th := p.MeasureText(0, tag)
				tagBox := ui.Rect{X: ex - tw/2 - t.Space(2.5), Y: ey - tagH, W: tw + 2*t.Space(2.5), H: tagH}
				if side == 1 {
					tagBox.Y = ey
				}
				p.Fill(tagBox, t.Surface, t.Radius)
				p.Stroke(tagBox, t.Border, t.Radius, 1)
				p.RichText(tagBox.X+t.Space(2.5), tagBox.Y+(tagH-th)/2, 0, tag)
				for j, cause := range b.Causes {
					f := float32(j+1) / float32(len(b.Causes)+1)
					// Causes start at the spine's end of the bone, the first
					// nearest the name, so that each line has room to its left.
					f = 1 - f*0.92
					x, y := bx-run*f, cy+dir*rise*f
					col, weight := t.Text, 400
					if cause.Likely {
						col, weight = t.Accent, 700
					}
					sp := ui.Span{Text: cause.Text, Size: small, Color: col, Weight: weight}
					maxW := gap*0.9 - t.Space(2)
					w, h := p.MeasureText(maxW, sp)
					line := [][2]float32{{x - w - t.Space(2), y}, {x, y}}
					lineCol := ink.Alpha(0.6)
					if cause.Likely {
						lineCol = t.Accent
					}
					p.StrokePolyline(line, 1.2, lineCol)
					// The text sits on the spine's side of its line, where the
					// slanting bone leans away from it.
					box := ui.Rect{X: x - w - t.Space(1.5), Y: y - h - t.Space(0.5), W: w, H: h}
					if side == 0 {
						box.Y = y + t.Space(0.5)
					}
					if len(s.hits) == hover {
						p.Fill(ui.Rect{X: box.X - t.Space(1), Y: box.Y - t.Space(0.5), W: box.W + t.Space(2), H: box.H + t.Space(1)}, t.Accent.Alpha(0.1), t.Radius)
					}
					p.RichText(box.X, box.Y, maxW, sp)
					s.hits = append(s.hits, hit{r: ui.Rect{X: box.X - r.X, Y: box.Y - r.Y, W: box.W, H: box.H + t.Space(1)}, bone: bi, item: j})
				}
			}
		}
	})
	return view
}
