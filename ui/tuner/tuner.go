// Package tuner shows the pitch of a sound against the nearest note, as
// an instrument tuner does: the note's name and octave, the frequency,
// and a needle on a dial from 50 cents flat to 50 sharp, green within 5
// cents. Detect finds the pitch of a stretch of samples, with the YIN
// method.
//
//	hz := tuner.Detect(app.samples[pos:pos+2048], 44100)
//	tuner.View(c, hz)
//
// The needle glides to each new reading. With no pitch, 0, the note shows
// a dash and the needle rests in the middle, dimmed.
package tuner

import (
	"fmt"
	"math"

	"github.com/egoist/mygo/ui"
)

// lowest and highest bound the pitches Detect finds, in Hz: from below
// a bass's low E to above a soprano's high C.
const lowest, highest = 40, 1600

// threshold is the YIN threshold: how unlike itself a period may be and
// still count.
const threshold = 0.12

// Detect returns the pitch of samples, mono at rate Hz, in Hz, or 0 if
// they are quiet or have none. Give it at least twice the longest period
// it should find: 2048 samples at 44.1 kHz reach down to 43 Hz.
func Detect(samples []float32, rate int) float64 {
	n := len(samples) / 2
	var power float64
	for _, v := range samples {
		power += float64(v) * float64(v)
	}
	if n == 0 || math.Sqrt(power/float64(len(samples))) < 0.01 {
		return 0
	}
	minTau := max(2, rate/highest)
	maxTau := min(n-1, rate/lowest)
	if minTau >= maxTau {
		return 0
	}
	d := make([]float64, maxTau+2)
	for tau := 1; tau <= maxTau+1 && tau < n; tau++ {
		var sum float64
		for i := range n {
			diff := float64(samples[i]) - float64(samples[i+tau])
			sum += diff * diff
		}
		d[tau] = sum
	}
	// The cumulative mean normalizes each difference by those of shorter
	// lags, so that lag 0 does not win and a dip reads the same at any
	// loudness.
	d[0] = 1
	var running float64
	for tau := 1; tau < len(d); tau++ {
		running += d[tau]
		if running == 0 {
			d[tau] = 1
			continue
		}
		d[tau] *= float64(tau) / running
	}
	for tau := minTau; tau <= maxTau; tau++ {
		if d[tau] >= threshold {
			continue
		}
		for tau+1 <= maxTau && d[tau+1] < d[tau] {
			tau++
		}
		// A parabola through the dip and its neighbors places the period
		// between samples.
		a, b, c := d[tau-1], d[tau], d[tau+1]
		shift := 0.0
		if den := a - 2*b + c; den != 0 {
			shift = (a - c) / (2 * den)
		}
		return float64(rate) / (float64(tau) + shift)
	}
	return 0
}

var names = [12]string{"C", "C♯", "D", "D♯", "E", "F", "F♯", "G", "G♯", "A", "A♯", "B"}

// note returns the note nearest hz, with A4 at 440 Hz: its name, its
// octave, and how many cents hz is above it.
func note(hz float64) (string, int, float64) {
	midi := 69 + 12*math.Log2(hz/440)
	n := math.Round(midi)
	i := int(n)
	return names[(i%12+12)%12], i/12 - 1, (midi - n) * 100
}

// View creates the tuner showing hz, 0 for no pitch.
func View(c *ui.Context, hz float64) ui.Element {
	t := c.Theme()
	name, octave, cents := "–", 0, 0.0
	heard := hz > 0
	if heard {
		name, octave, cents = note(hz)
	}
	inTune := heard && math.Abs(cents) < 5
	root := ui.Column(c).AlignItems(ui.Center).Gap(t.Space(2)).Label("Tuner")
	if heard {
		root.Value(fmt.Sprintf("%s%d, %+.0f cents", name, octave, cents))
	}
	needle := root.AnimateTo("tuner.needle", float32(cents), ui.Smooth)
	color := t.Text
	switch {
	case !heard:
		color = t.TextMuted.Alpha(0.5)
	case inTune:
		color = t.Success
	case math.Abs(cents) > 20:
		color = t.Danger
	default:
		color = t.Warning
	}
	root.Children(func() {
		ui.Box(c).Size(320, 170).Draw(func(p *ui.Painter, r ui.Rect) {
			dial(p, t, r, needle, color)
		})
		ui.Row(c).AlignItems(ui.End).Children(func() {
			ui.Text(c, name).FontSize(64).Bold().TextColor(color)
			if heard {
				ui.Text(c, fmt.Sprint(octave)).FontSize(22).TextColor(t.TextMuted).Margin(0, 0, t.Space(2.5), t.Space(1))
			}
		})
		ui.Row(c).Gap(t.Space(4)).Children(func() {
			freq, off := "– Hz", "– cents"
			if heard {
				freq, off = fmt.Sprintf("%.1f Hz", hz), fmt.Sprintf("%+.0f cents", cents)
			}
			ui.Text(c, freq).FontFeatures("tnum").TextColor(t.TextMuted)
			ui.Text(c, off).FontFeatures("tnum").TextColor(t.TextMuted)
		})
	})
	return root
}

// dial draws the dial in r: ticks every 5 cents from -50 to 50 along an
// arc, the band in tune, and the needle at cents in color.
func dial(p *ui.Painter, t *ui.Theme, r ui.Rect, cents float32, color ui.Color) {
	cx, cy := r.X+r.W/2, r.Y+r.H-8
	radius := min(r.W/2-12, r.H-24)
	// at returns the point at cents, dist from the center: a cent is a
	// degree of the dial.
	at := func(cents, dist float32) (float32, float32) {
		a := float64(cents) * math.Pi / 180
		return cx + dist*float32(math.Sin(a)), cy - dist*float32(math.Cos(a))
	}
	band := &ui.Path{}
	for i := float32(-5); i <= 5; i++ {
		x, y := at(i, radius)
		if i == -5 {
			band.MoveTo(x, y)
		} else {
			band.LineTo(x, y)
		}
	}
	p.StrokePath(band, 10, t.Success.Alpha(0.25))
	for v := -50; v <= 50; v += 5 {
		long := float32(6)
		switch {
		case v%25 == 0:
			long = 16
		case v%10 == 0:
			long = 10
		}
		x0, y0 := at(float32(v), radius-long/2)
		x1, y1 := at(float32(v), radius+long/2)
		p.StrokePolyline([][2]float32{{x0, y0}, {x1, y1}}, 1.5, t.TextMuted.Alpha(0.7))
		if v%25 == 0 {
			label := fmt.Sprintf("%+d", v)
			if v == 0 {
				label = "0"
			}
			lx, ly := at(float32(v), radius+18)
			p.Text(lx-float32(len(label))*3, ly-6, label, 10, t.TextMuted)
		}
	}
	nx, ny := at(max(-55, min(55, cents)), radius-4)
	p.StrokePolyline([][2]float32{{cx, cy}, {nx, ny}}, 2.5, color)
	p.FillPath((&ui.Path{}).Circle(cx, cy, 6), color)
}
