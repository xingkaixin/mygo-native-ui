// Package spectrum is a spectrum analyzer: the levels of the frequencies
// in the sound heard, on a log axis from 30 Hz up, as bars with their
// peaks held a moment, or as a curve. The bars fall back as the sound
// quiets; hovering reads out the frequency and level there.
//
// The app gives it the samples last heard each frame, as those before
// its player's position, and draws a frame while it plays:
//
//	at := int(player.Position().Seconds() * rate)
//	spectrum.View(c, &app.meter, samples[max(0, at-4096):at], rate)
package spectrum

import (
	"math"
	"math/cmplx"
	"strconv"
	"time"

	"github.com/egoist/mygo/ui"
)

const (
	floor = -84.0
	low   = 30.0
	// fall and drop are how fast bars and peaks fall, in dB a second;
	// hold is how long a peak stays first.
	fall = 60.0
	drop = 24.0
	hold = 700 * time.Millisecond
)

// State is what an analyzer keeps: its levels as they fall, its peaks,
// and whether it draws a curve. Keep it in the app's state, one for each
// analyzer.
type State struct {
	curve  int
	levels []float64
	peaks  []float64
	since  []time.Time
	last   time.Time
	window []float64
	buf    []complex128
}

// analyze returns the level of each of n bands of window, in dB of full
// scale, the bands spaced evenly in log frequency from low to top.
func (s *State) analyze(window []float32, rate, n int) []float64 {
	size := 1
	for size*2 <= min(len(window), 8192) {
		size *= 2
	}
	out := make([]float64, n)
	for i := range out {
		out[i] = floor
	}
	if size < 256 {
		return out
	}
	if len(s.window) != size {
		s.window = make([]float64, size)
		for i := range s.window {
			s.window[i] = 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(size-1))
		}
		s.buf = make([]complex128, size)
	}
	w := window[len(window)-size:]
	for i := range s.buf {
		s.buf[i] = complex(float64(w[i])*s.window[i], 0)
	}
	fft(s.buf)
	// A full-scale sine, through the Hann window, peaks at a quarter of
	// the window's length.
	mag := func(k int) float64 { return cmplx.Abs(s.buf[k]) / (float64(size) / 4) }
	top := min(16000, float64(rate)/2)
	hz := float64(rate) / float64(size)
	for i := range out {
		f0 := low * math.Pow(top/low, float64(i)/float64(n))
		f1 := low * math.Pow(top/low, float64(i+1)/float64(n))
		k0, k1 := int(math.Ceil(f0/hz)), int(f1/hz)
		m := 0.0
		if k1 >= k0 {
			for k := k0; k <= k1 && k < size/2; k++ {
				m = max(m, mag(k))
			}
		} else {
			// A band narrower than a bin takes the level between the bins
			// around its middle.
			c := math.Sqrt(f0*f1) / hz
			k := int(c)
			m = mag(k) + (mag(k+1)-mag(k))*(c-float64(k))
		}
		out[i] = max(floor, 20*math.Log10(max(m, 1e-9)))
	}
	return out
}

// fft transforms x in place, its length a power of two.
func fft(x []complex128) {
	n := len(x)
	for i, j := 1, 0; i < n; i++ {
		bit := n >> 1
		for ; j&bit != 0; bit >>= 1 {
			j ^= bit
		}
		j ^= bit
		if i < j {
			x[i], x[j] = x[j], x[i]
		}
	}
	for l := 2; l <= n; l <<= 1 {
		w := cmplx.Exp(complex(0, -2*math.Pi/float64(l)))
		for i := 0; i < n; i += l {
			wk := complex(1, 0)
			for k := range l / 2 {
				a, b := x[i+k], x[i+k+l/2]*wk
				x[i+k], x[i+k+l/2] = a+b, a-b
				wk *= w
			}
		}
	}
}

// update takes the levels now, letting bars and peaks fall by the time
// since the last frame, and reports whether any is still above the floor.
func (s *State) update(now time.Time, levels []float64) bool {
	if len(s.levels) != len(levels) {
		s.levels = make([]float64, len(levels))
		s.peaks = make([]float64, len(levels))
		s.since = make([]time.Time, len(levels))
		for i := range s.levels {
			s.levels[i], s.peaks[i] = floor, floor
		}
	}
	dt := 0.0
	if !s.last.IsZero() {
		dt = min(0.1, now.Sub(s.last).Seconds())
	}
	s.last = now
	moving := false
	for i, v := range levels {
		s.levels[i] = max(v, s.levels[i]-fall*dt)
		if s.levels[i] >= s.peaks[i] {
			s.peaks[i], s.since[i] = s.levels[i], now
		} else if now.Sub(s.since[i]) > hold {
			s.peaks[i] = max(floor, s.peaks[i]-drop*dt)
		}
		moving = moving || s.levels[i] > floor || s.peaks[i] > floor
	}
	return moving
}

func label(f float64) string {
	if f >= 1000 {
		return strconv.FormatFloat(f/1000, 'f', -1, 64) + "k"
	}
	return strconv.Itoa(int(f))
}

// View creates the analyzer of window, the samples last heard, of one
// channel at rate frames a second; nil when nothing plays.
func View(c *ui.Context, s *State, window []float32, rate int) ui.Element {
	t := c.Theme()
	n := 48
	if s.curve == 1 {
		n = 160
	}
	if s.update(c.Now(), s.analyze(window, rate, n)) {
		c.After(time.Second / 60)
	}
	top := min(16000, float64(rate)/2)
	root := ui.Column(c).Gap(t.Space(2))
	root.Children(func() {
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Grow(1)
			ui.Segmented(c, &s.curve, "Bars", "Curve")
		})
		graph := ui.Box(c).AlignSelf(ui.Stretch).Grow(1).MinHeight(t.Space(40)).Label("Spectrum")
		px, py, over := graph.PointerPosition()
		levels, peaks, curve := s.levels, s.peaks, s.curve == 1
		graph.Draw(func(p *ui.Painter, r ui.Rect) {
			left, bottom := float32(34), float32(18)
			plot := ui.Rect{X: r.X + left, Y: r.Y, W: r.W - left, H: r.H - bottom}
			y := func(db float64) float32 { return plot.Y + plot.H*float32(db/floor) }
			x := func(f float64) float32 { return plot.X + plot.W*float32(math.Log(f/low)/math.Log(top/low)) }
			p.Fill(plot, t.Surface, 4)
			for db := -12.0; db > floor; db -= 12 {
				p.Line(plot.X, y(db), plot.X+plot.W, y(db), 1, t.Border)
				p.Text(r.X, y(db)-7, strconv.Itoa(int(db)), t.FontSize-3, t.TextMuted)
			}
			for _, f := range []float64{50, 100, 200, 500, 1000, 2000, 5000, 10000} {
				if f < top {
					p.Line(x(f), plot.Y, x(f), plot.Y+plot.H, 1, t.Border)
					p.Text(x(f)-8, plot.Y+plot.H+3, label(f), t.FontSize-3, t.TextMuted)
				}
			}
			level := func(db float64) ui.Color {
				return ui.Oklch(0.68, 0.16, float32(150-110*max(0, min(1, 1-db/floor*1.4))))
			}
			bw := plot.W / float32(len(levels))
			if curve {
				pts := make([][2]float32, len(levels))
				var area ui.Path
				area.MoveTo(plot.X, plot.Y+plot.H)
				for i, v := range levels {
					pts[i] = [2]float32{plot.X + (float32(i)+0.5)*bw, y(v)}
					area.LineTo(pts[i][0], pts[i][1])
				}
				area.LineTo(plot.X+plot.W, plot.Y+plot.H).Close()
				p.FillPath(&area, t.Accent.Alpha(0.18))
				p.StrokePolyline(pts, 2, t.Accent)
				for i, v := range peaks {
					pts[i][1] = y(v)
				}
				p.StrokePolyline(pts, 1, t.TextMuted.Alpha(0.6))
			} else {
				for i, v := range levels {
					bx := plot.X + float32(i)*bw + 1
					if v > floor {
						p.Fill(ui.Rect{X: bx, Y: y(v), W: bw - 2, H: plot.Y + plot.H - y(v)}, level(v), 1)
					}
					if pk := peaks[i]; pk > floor {
						p.Fill(ui.Rect{X: bx, Y: y(pk) - 2, W: bw - 2, H: 2}, t.Text.Alpha(0.7), 0)
					}
				}
			}
			if over && px > left && py < r.H-bottom && len(levels) > 0 {
				f := low * math.Pow(top/low, float64((px-left)/plot.W))
				i := max(0, min(len(levels)-1, int((px-left)/bw)))
				text := label(math.Round(f/10)*10) + " Hz · " + strconv.Itoa(int(math.Round(levels[i]))) + " dB"
				p.Line(r.X+px, plot.Y, r.X+px, plot.Y+plot.H, 1, t.TextMuted)
				p.Text(min(r.X+px+6, plot.X+plot.W-110), plot.Y+6, text, t.FontSize-2, t.Text)
			}
		})
	})
	return root
}
