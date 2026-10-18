// Gif plays an animated GIF: the file given as its argument, or one it
// draws at startup, where a ball rolls along a track and back.
//
//	go run ./examples/gif [file.gif]
package main

import (
	"bytes"
	"image"
	"image/color"
	stdgif "image/gif"
	"log"
	"os"
	"path/filepath"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/gif"
)

// player is what the window shows: an animation and what it is.
type player struct {
	anim    *gif.Animation
	caption string
}

func (s *player) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Padding(20).Gap(12).Children(func() {
		ui.Column(c).Grow(1).Padding(16).Background(t.Surface).Border(1, t.Border).Radius(12).Children(func() {
			gif.Image(c, s.anim).Fill().Fit(ui.ScaleDown)
		})
		ui.Text(c, s.caption).FontSize(12).TextColor(t.TextMuted)
		if c.Preferences().ReduceMotion {
			ui.Text(c, "Reduce Motion is on: showing the first frame.").FontSize(12).TextColor(t.TextMuted)
		}
	})
}

// The drawn GIF's logical screen, the space at the ends of its track, and
// its ball's diameter.
const (
	screenW, screenH = 360, 120
	pad              = 16
	ball             = 40
)

// rolling draws a GIF of a ball rolling along a dashed track and back. The
// track is the first frame, which stays (DisposalNone). Rolling right, each
// frame of the ball gives back what was under it (DisposalPrevious), so
// the track shows again behind it; rolling back, each clears its box to
// transparent (DisposalBackground), so the ball wipes the track away. The
// ball's frames are transparent around it, and the screen around the
// track, which shows what is behind the image.
func rolling() []byte {
	palette := color.Palette{
		color.RGBA{},
		color.RGBA{R: 0x9c, G: 0xa3, B: 0xaf, A: 0xff},
		color.RGBA{R: 0x3b, G: 0x82, B: 0xf6, A: 0xff},
		color.RGBA{R: 0x93, G: 0xc5, B: 0xfd, A: 0xff},
	}
	track := image.NewPaletted(image.Rect(0, 0, screenW, screenH), palette)
	for x := pad; x+16 <= screenW-pad; x += 24 {
		for y := screenH/2 - 3; y < screenH/2+3; y++ {
			for dx := range 16 {
				track.SetColorIndex(x+dx, y, 1)
			}
		}
	}
	g := &stdgif.GIF{
		Image:    []*image.Paletted{track},
		Delay:    []int{40},
		Disposal: []byte{stdgif.DisposalNone},
	}
	const steps = 16
	travel := screenW - 2*pad - ball
	for i := range 2 * steps {
		pos, disposal := i, byte(stdgif.DisposalPrevious)
		if i >= steps {
			pos, disposal = 2*steps-1-i, stdgif.DisposalBackground
		}
		x := pad + travel*pos/(steps-1)
		g.Image = append(g.Image, ballFrame(x, (screenH-ball)/2, palette))
		g.Delay = append(g.Delay, 5)
		g.Disposal = append(g.Disposal, disposal)
	}
	var buf bytes.Buffer
	if err := stdgif.EncodeAll(&buf, g); err != nil {
		log.Fatal(err)
	}
	return buf.Bytes()
}

// ballFrame draws the ball in a frame of its own size at x, y on the
// screen, transparent around it.
func ballFrame(x, y int, palette color.Palette) *image.Paletted {
	m := image.NewPaletted(image.Rect(x, y, x+ball, y+ball), palette)
	const r = ball / 2.0
	for py := range ball {
		for px := range ball {
			dx, dy := float64(px)+0.5-r, float64(py)+0.5-r
			hx, hy := dx+r/3, dy+r/3
			switch {
			case hx*hx+hy*hy <= r*r/9:
				m.SetColorIndex(x+px, y+py, 3)
			case dx*dx+dy*dy <= r*r:
				m.SetColorIndex(x+px, y+py, 2)
			}
		}
	}
	return m
}

func main() {
	data, caption := rolling(), "Rolling right, the ball restores the track behind it (disposal Previous); rolling back, it clears it (Background)."
	if len(os.Args) > 1 {
		var err error
		if data, err = os.ReadFile(os.Args[1]); err != nil {
			log.Fatal(err)
		}
		caption = filepath.Base(os.Args[1])
	}
	anim, err := gif.Decode(data)
	if err != nil {
		log.Fatal(err)
	}
	s := &player{anim: anim, caption: caption}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "GIF",
			Width:   480,
			Height:  300,
			Content: ui.View(s.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
