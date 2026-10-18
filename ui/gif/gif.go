// Package gif plays animated GIFs in native UI, of which ui.DecodeBitmap
// shows only the first frame:
//
//	//go:embed spinner.gif
//	var spinnerGIF []byte
//
//	spinner, err := gif.Decode(spinnerGIF)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	gif.Image(c, spinner).Size(48, 48).Radius(8)
//
// Frames show for their delays, as many times over as the GIF's loop count
// says, and compose on its logical screen as browsers compose them: at
// their offsets, with what is under them showing through their
// transparent pixels, and disposed of as each frame asks before the next.
// Where the desktop asks for less motion, a GIF shows its first frame and
// stands still.
package gif

import (
	"bytes"
	"image"
	"image/draw"
	stdgif "image/gif"
	"time"

	"github.com/egoist/mygo/ui"
)

// Animation is a decoded GIF and how far it has played. Decode it once, as
// a ui.Bitmap, not in the view. It has one picture, which every Image
// showing it shares, and starts playing in the first frame an Image shows
// it. It plays on while no Image shows it, and once it has played as many
// times as its loop count says, it stays on its last frame: decode the GIF
// again to play it from the start.
type Animation struct {
	g *stdgif.GIF
	// canvas is the logical screen with the frames up to frame composed on
	// it, and saved the canvas from before a frame that disposes of itself
	// with DisposalPrevious.
	canvas, saved *image.RGBA
	bitmap        *ui.Bitmap
	frame         int
	// start is when a started playing, zero until it plays.
	start time.Time
}

// Decode decodes GIF data, all of its frames.
func Decode(data []byte) (*Animation, error) {
	g, err := stdgif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	a := &Animation{g: g, canvas: image.NewRGBA(image.Rect(0, 0, g.Config.Width, g.Config.Height))}
	a.compose(0)
	a.bitmap = ui.NewBitmap(a.canvas)
	return a, nil
}

// Image creates an element showing a, as ui.Image shows a bitmap, at the
// GIF's size as DIPs unless given another. Each frame of the view shows
// the GIF's frame due then and asks for a frame of the view when the next
// is due.
func Image(c *ui.Context, a *Animation) ui.Element {
	if len(a.g.Image) > 1 {
		a.play(c)
	}
	return ui.Image(c, a.bitmap)
}

// play shows the frame of a due at the view's frame, and asks for a frame
// of the view when the next is due; it shows the first while the desktop
// asks for less motion.
func (a *Animation) play(c *ui.Context) {
	if c.Preferences().ReduceMotion {
		a.start = time.Time{}
		a.show(0)
		return
	}
	now := c.Now()
	if a.start.IsZero() {
		a.start = now
	}
	elapsed, length := now.Sub(a.start), a.length()
	// A loop count of 0 plays forever, -1 once, and n n+1 times, as in
	// image/gif.
	if n := a.g.LoopCount; n != 0 && int(elapsed/length) > max(n, 0) {
		a.show(len(a.g.Image) - 1)
		return
	}
	t, i := elapsed%length, 0
	for t >= a.delay(i) {
		t -= a.delay(i)
		i++
	}
	a.show(i)
	c.After(a.delay(i) - t)
}

// show composes frame i on the canvas, going on from the frame there or,
// for an earlier one, from the first, as each play starts on a clear
// canvas.
func (a *Animation) show(i int) {
	if i == a.frame {
		return
	}
	if i < a.frame {
		a.compose(0)
	}
	for a.frame < i {
		a.compose(a.frame + 1)
	}
	a.bitmap.Update(a.canvas)
}

// delay returns how long frame i shows: 100 ms for a delay of 10 ms or
// less, as browsers play those.
func (a *Animation) delay(i int) time.Duration {
	if d := a.g.Delay[i]; d > 1 {
		return time.Duration(d) * 10 * time.Millisecond
	}
	return 100 * time.Millisecond
}

// length returns how long a play takes.
func (a *Animation) length() time.Duration {
	var d time.Duration
	for i := range a.g.Image {
		d += a.delay(i)
	}
	return d
}

// compose draws frame i on the canvas once the frame before it is disposed
// of as it asks, the first frame on a clear canvas.
func (a *Animation) compose(i int) {
	if i == 0 {
		clear(a.canvas.Pix)
	} else {
		switch a.g.Disposal[i-1] {
		case stdgif.DisposalBackground:
			draw.Draw(a.canvas, a.g.Image[i-1].Rect, image.Transparent, image.Point{}, draw.Src)
		case stdgif.DisposalPrevious:
			copy(a.canvas.Pix, a.saved.Pix)
		}
	}
	if a.g.Disposal[i] == stdgif.DisposalPrevious {
		if a.saved == nil {
			a.saved = image.NewRGBA(a.canvas.Rect)
		}
		copy(a.saved.Pix, a.canvas.Pix)
	}
	m := a.g.Image[i]
	draw.Draw(a.canvas, m.Rect, m, m.Rect.Min, draw.Over)
	a.frame = i
}
