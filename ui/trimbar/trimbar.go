// Package trimbar trims a clip, as QuickTime and Photos do: its frames in
// a strip, the part kept framed in yellow between two handles and the
// rest dimmed, and a playhead over it. Dragging a handle moves the in or
// out point, with the playhead on it to show its frame; dragging or
// clicking between them moves the playhead. I and O put the in and out
// points at the playhead. Under the strip, the in point, how long the
// part kept is, and the out point.
//
//	trimbar.View(c, &app.trim, app.length, &app.in, &app.out, &app.playhead, app.frames)
package trimbar

import (
	"fmt"
	"time"

	"github.com/egoist/mygo/ui"
)

// shortest is how close the handles come together.
const shortest = time.Second

// State is what a trim bar keeps: the part being dragged. Keep it in the
// app's state, one for each trim bar.
type State struct {
	part part
}

type part uint8

const (
	none part = iota
	inHandle
	outHandle
	body
)

var (
	yellow  = ui.Hex("#FFC400")
	handleW = float32(14)
	chevron = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3.5" stroke-linecap="round" stroke-linejoin="round"><path d="M15 5l-7 7 7 7"/></svg>`))
)

// View creates the trim bar of a clip of length, its in and out points
// and playhead, and frames, its pictures at even times from its start.
func View(c *ui.Context, s *State, length time.Duration, in, out, playhead *time.Duration, frames []*ui.Bitmap) ui.Element {
	t := c.Theme()
	if *out <= *in {
		*out = length
	}
	changed := false
	if c.Shortcut(0, ui.KeyI) && *playhead < *out-shortest {
		*in, changed = *playhead, true
	}
	if c.Shortcut(0, ui.KeyO) && *playhead > *in+shortest {
		*out, changed = *playhead, true
	}
	root := ui.Column(c).Gap(t.Space(2))
	root.Children(func() {
		strip := ui.Box(c).FillWidth().Height(t.Space(16)).Label("Trim").Cursor(ui.CursorPointer)
		bounds := strip.LayoutBounds()
		span := max(1, bounds.W-2*handleW)
		at := func(d time.Duration) float32 { return handleW + span*float32(d)/float32(length) }
		when := func(x float32) time.Duration {
			return time.Duration(float64(min(max(x-handleW, 0), span)) / float64(span) * float64(length))
		}
		_, _, held := strip.Dragged()
		x, _, _ := strip.PointerPosition()
		if held {
			if s.part == none {
				switch {
				case x >= at(*in)-handleW-4 && x <= at(*in)+2:
					s.part = inHandle
				case x >= at(*out)-2 && x <= at(*out)+handleW+4:
					s.part = outHandle
				default:
					s.part = body
				}
			}
			d := when(x)
			switch s.part {
			case inHandle:
				*in = min(max(d, 0), *out-shortest)
				*playhead = *in
			case outHandle:
				*out = max(min(d, length), *in+shortest)
				*playhead = *out
			case body:
				*playhead = min(max(d, *in), *out)
			}
			changed = true
		} else {
			s.part = none
		}
		*playhead = min(max(*playhead, *in), *out)
		xIn, xOut, xPlay := at(*in), at(*out), at(*playhead)
		strip.Draw(func(p *ui.Painter, r ui.Rect) {
			area := ui.Rect{X: r.X + handleW, Y: r.Y + 3, W: r.W - 2*handleW, H: r.H - 6}
			p.Clip(area, 4, func() {
				p.Fill(area, ui.RGB(0, 0, 0), 0)
				if len(frames) > 0 {
					fw, fh := frames[0].Size()
					slot := area.H * float32(fw) / float32(max(1, fh))
					n := int(area.W/slot) + 1
					for i := range n {
						f := frames[min(len(frames)-1, i*len(frames)/n)]
						p.Image(f, ui.Rect{X: area.X + float32(i)*slot, Y: area.Y, W: slot, H: area.H}, ui.Cover)
					}
				}
				dim := ui.RGBA(0, 0, 0, 0.6)
				p.Fill(ui.Rect{X: area.X, Y: area.Y, W: r.X + xIn - area.X, H: area.H}, dim, 0)
				p.Fill(ui.Rect{X: r.X + xOut, Y: area.Y, W: area.X + area.W - r.X - xOut, H: area.H}, dim, 0)
			})
			frame := ui.Rect{X: r.X + xIn - handleW, Y: r.Y, W: xOut - xIn + 2*handleW, H: r.H}
			p.Fill(ui.Rect{X: frame.X, Y: frame.Y, W: frame.W, H: 3}, yellow, 0)
			p.Fill(ui.Rect{X: frame.X, Y: frame.Y + frame.H - 3, W: frame.W, H: 3}, yellow, 0)
			for k, hx := range []float32{frame.X, r.X + xOut} {
				h := ui.Rect{X: hx, Y: r.Y, W: handleW, H: r.H}
				// Only the outer corners are round: the clip cuts off the
				// inner half of a rounded rectangle twice as wide.
				wide := ui.Rect{X: hx - handleW*float32(k), Y: r.Y, W: 2 * handleW, H: r.H}
				p.Clip(h, 0, func() { p.Fill(wide, yellow, 6) })
				icon := ui.Rect{X: h.X + 1, Y: h.Y + h.H/2 - 6, W: 12, H: 12}
				if k == 1 {
					p.Transform(ui.Scaling(-1, 1), icon.X+icon.W/2, icon.Y, func() { p.Icon(chevron, icon, ui.RGBA(0, 0, 0, 0.65)) })
				} else {
					p.Icon(chevron, icon, ui.RGBA(0, 0, 0, 0.65))
				}
			}
			line := ui.Rect{X: r.X + xPlay - 1.5, Y: r.Y - 4, W: 3, H: r.H + 8}
			p.Shadow(line, 1.5, 0, 0, 4, 0, ui.RGBA(0, 0, 0, 0.45))
			p.Fill(line, ui.RGB(255, 255, 255), 1.5)
		})
		ui.Row(c).Children(func() {
			ui.Text(c, clock(*in)).FontFeatures("tnum").TextColor(t.TextMuted).Width(t.Space(20))
			ui.Spacer(c)
			ui.Text(c, clock(*out-*in)).FontFeatures("tnum").Bold()
			ui.Spacer(c)
			ui.Text(c, clock(*out)).FontFeatures("tnum").TextColor(t.TextMuted).Width(t.Space(20)).TextAlign(ui.End)
		})
	})
	if changed {
		root.SetChanged()
	}
	return root
}

// clock writes d as minutes, seconds and tenths, as 1:04.3.
func clock(d time.Duration) string {
	tenths := int(d / (100 * time.Millisecond))
	return fmt.Sprintf("%d:%02d.%d", tenths/600, tenths/10%60, tenths%10)
}
