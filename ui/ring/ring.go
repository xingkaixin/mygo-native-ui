// Package ring shows progress as a ring, as Apple's activity rings and
// circular progress views do: a track around, filled clockwise from the
// top as far as the value, with room inside for a label. It fits where a
// bar does not, as a timer in a menu bar panel or a gauge on a tile.
//
//	ring.Progress(c, app.used/app.size, t.Accent).Size(56, 56).Children(func() {
//		ui.Textf(c, "%.0f%%", 100*app.used/app.size).Bold()
//	})
//	ring.Countdown(c, app.end, 25*time.Minute, t.Danger).Size(120, 120)
//
// A ring is as wide as the smaller of its element's width and height, its
// stroke a tenth of that, and its track the color of its fill, faint.
package ring

import (
	"fmt"
	"math"
	"time"

	"github.com/egoist/mygo/ui"
)

// Progress creates a ring filled to value, between 0 and 1, in color. It
// is 40 DIPs wide, or as Size sets, and centers its children. Assistive
// technology sees a progress indicator; name it with Label.
func Progress(c *ui.Context, value float64, color ui.Color) ui.Element {
	f := float32(0)
	if value > 0 {
		f = float32(min(1, value))
	}
	e := ui.Column(c).Size(40, 40).Center().Role(ui.RoleProgress).Range(0, 1, max(0, min(1, value)))
	e.Draw(func(p *ui.Painter, r ui.Rect) {
		d := min(r.W, r.H)
		w := max(2, d/10)
		cx, cy, radius := r.X+r.W/2, r.Y+r.H/2, (d-w)/2
		var track ui.Path
		p.StrokePath(track.Circle(cx, cy, radius), w, color.Alpha(0.2))
		if f > 0 {
			p.StrokePath(fromTop(cx, cy, radius).Trim(0, f), w, color)
		}
	})
	return e
}

// fromTop returns a circle that starts at the top and goes clockwise, for
// Trim to fill as a clock's hand goes.
func fromTop(cx, cy, r float32) *ui.Path {
	const k = 0.5522847498 // of a cubic Bézier closest to a quarter circle
	var p ui.Path
	p.MoveTo(cx, cy-r)
	p.CubeTo(cx+r*k, cy-r, cx+r, cy-r*k, cx+r, cy)
	p.CubeTo(cx+r, cy+r*k, cx+r*k, cy+r, cx, cy+r)
	p.CubeTo(cx-r*k, cy+r, cx-r, cy+r*k, cx-r, cy)
	p.CubeTo(cx-r, cy-r*k, cx-r*k, cy-r, cx, cy-r)
	return &p
}

// Countdown creates a ring of the time left until end, of total, with the
// time left inside, "24:59", or "1:04:59" from an hour up: full at the
// start, empty at the end. It moves on by itself each second, and stops
// at 0:00. Give it a size; its text takes the font size of its parent,
// or FontSize.
func Countdown(c *ui.Context, end time.Time, total time.Duration, color ui.Color) ui.Element {
	left := max(0, end.Sub(c.Now()))
	if left > 0 {
		// The next frame comes as the seconds shown change.
		c.After(left - left.Truncate(time.Second) + time.Millisecond)
	}
	value := 0.0
	if total > 0 {
		value = float64(left) / float64(total)
	}
	text := clock(left)
	return Progress(c, value, color).Value(text).Children(func() {
		ui.Text(c, text).FontFeatures("tnum").NoWrap()
	})
}

// clock returns d as a timer shows it, rounded up to the second, so that
// it reads 0:00 only once the time is up.
func clock(d time.Duration) string {
	s := int(math.Ceil(d.Seconds()))
	if s >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", s/3600, s/60%60, s%60)
	}
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}
