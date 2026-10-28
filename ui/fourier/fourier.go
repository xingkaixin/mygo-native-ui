// Package fourier draws a closed path with epicycles, as a Fourier series
// does: a chain of circles, each as large as a term of the path's
// discrete Fourier transform and turning as many times a period as the
// term's frequency, the end of the chain tracing the path once every eight
// seconds. With fewer terms the trace rounds off the path's
// corners. Drawing on the view with the pointer replaces the path.
//
//	fourier.View(c, &app.fourier, &app.path, 40)
//
// Paths lie in the square from -1 to 1, which fills the view's shorter
// side, with y growing downward.
package fourier

import (
	"math"
	"math/cmplx"
	"slices"
	"time"

	"github.com/egoist/mygo/ui"
)

// Term is a term of a path's Fourier series: a circle of radius Amp,
// turning Freq times a period, starting at angle Phase.
type Term struct {
	Freq       int
	Amp, Phase float64
}

// samples is how many points a path is resampled to before its
// transform.
const samples = 256

// Transform returns the Fourier series of a closed path, the largest
// terms first after the constant one, which puts the chain's center.
func Transform(path [][2]float64) []Term {
	pts := resample(path, samples)
	if pts == nil {
		return nil
	}
	n := len(pts)
	terms := make([]Term, 0, n)
	for k := -n / 2; k < n/2; k++ {
		var sum complex128
		for j, p := range pts {
			sum += complex(p[0], p[1]) * cmplx.Rect(1, -2*math.Pi*float64(k*j)/float64(n))
		}
		sum /= complex(float64(n), 0)
		terms = append(terms, Term{Freq: k, Amp: cmplx.Abs(sum), Phase: cmplx.Phase(sum)})
	}
	slices.SortFunc(terms, func(a, b Term) int {
		switch {
		case a.Freq == 0:
			return -1
		case b.Freq == 0:
			return 1
		case a.Amp > b.Amp:
			return -1
		case a.Amp < b.Amp:
			return 1
		}
		return 0
	})
	return terms
}

// resample returns n points spaced evenly along the closed path.
func resample(path [][2]float64, n int) [][2]float64 {
	if len(path) < 2 {
		return nil
	}
	closed := append(slices.Clone(path), path[0])
	cum := make([]float64, len(closed))
	for i := 1; i < len(closed); i++ {
		cum[i] = cum[i-1] + math.Hypot(closed[i][0]-closed[i-1][0], closed[i][1]-closed[i-1][1])
	}
	total := cum[len(cum)-1]
	if total == 0 {
		return nil
	}
	out := make([][2]float64, n)
	seg := 1
	for i := range n {
		d := total * float64(i) / float64(n)
		for seg < len(cum)-1 && cum[seg] < d {
			seg++
		}
		a, b := closed[seg-1], closed[seg]
		f := 0.0
		if l := cum[seg] - cum[seg-1]; l > 0 {
			f = (d - cum[seg-1]) / l
		}
		out[i] = [2]float64{a[0] + (b[0]-a[0])*f, a[1] + (b[1]-a[1])*f}
	}
	return out
}

// State is what a view keeps: the transform of its path and the stroke
// being drawn. Keep it in the app's state, one for each view.
type State struct {
	path    [][2]float64
	terms   []Term
	start   time.Time
	drawing [][2]float64
	pressed bool
}

const period = 8 * time.Second

// View creates the epicycles of *path with its first terms terms, the
// constant one included. Drawing with the pointer replaces *path.
func View(c *ui.Context, s *State, path *[][2]float64, terms int) ui.Element {
	t := c.Theme()
	if !slices.Equal(s.path, *path) {
		s.path = slices.Clone(*path)
		s.terms = Transform(s.path)
		s.start = time.Time{}
	}
	view := ui.Box(c).Clip().Cursor(ui.CursorCrosshair).Role(ui.RoleImage).Label("Epicycles")
	b := view.LayoutBounds()
	half := float64(min(b.W, b.H)) / 2 * 0.92
	toPath := func(x, y float32) [2]float64 {
		return [2]float64{(float64(x) - float64(b.W)/2) / half, (float64(y) - float64(b.H)/2) / half}
	}
	view.HandleInput(func(in ui.InputEvent) bool {
		switch in.Kind {
		case ui.InputPointerDown:
			s.pressed, s.drawing = true, [][2]float64{toPath(in.X, in.Y)}
		case ui.InputPointerMove:
			if !s.pressed {
				return false
			}
			s.drawing = append(s.drawing, toPath(in.X, in.Y))
		case ui.InputPointerUp:
			if s.pressed && len(s.drawing) > 4 {
				*path = s.drawing
			}
			s.pressed, s.drawing = false, nil
		default:
			return false
		}
		return true
	})
	now := c.Now()
	if s.start.IsZero() {
		s.start = now
	}
	used := s.terms[:min(max(terms, 1), len(s.terms))]
	phase := math.Mod(now.Sub(s.start).Seconds()/period.Seconds(), 1)
	if !s.pressed && len(used) > 0 {
		c.After(time.Second / 60)
	}
	view.Draw(func(p *ui.Painter, r ui.Rect) {
		cx, cy := r.X+r.W/2, r.Y+r.H/2
		at := func(q [2]float64) [2]float32 {
			return [2]float32{cx + float32(q[0]*half), cy + float32(q[1]*half)}
		}
		if len(s.path) > 1 {
			pts := make([][2]float32, 0, len(s.path)+1)
			for _, q := range s.path {
				pts = append(pts, at(q))
			}
			p.StrokePolyline(append(pts, pts[0]), 1.5, t.Border)
		}
		if s.pressed {
			pts := make([][2]float32, len(s.drawing))
			for i, q := range s.drawing {
				pts[i] = at(q)
			}
			p.StrokePolyline(pts, 2.5, t.Accent)
			return
		}
		if len(used) == 0 {
			return
		}
		// The trace of the last turn, fading toward its start.
		const steps, chunks = 400, 20
		trace := make([][2]float32, steps+1)
		for k := range trace {
			trace[k] = at(sum(used, phase-float64(steps-k)/steps))
		}
		for ch := range chunks {
			lo, hi := ch*steps/chunks, (ch+1)*steps/chunks
			p.StrokePolyline(trace[lo:hi+1], 2.5, t.Accent.Alpha(0.15+0.85*float32(ch+1)/chunks))
		}
		var z complex128
		ring := t.TextMuted.Alpha(0.35)
		for _, tm := range used {
			from := z
			z += cmplx.Rect(tm.Amp, tm.Phase+2*math.Pi*float64(tm.Freq)*phase)
			if tm.Freq == 0 {
				continue
			}
			c0, c1 := at([2]float64{real(from), imag(from)}), at([2]float64{real(z), imag(z)})
			if rad := float32(tm.Amp * half); rad > 0.6 {
				var circle ui.Path
				p.StrokePath(circle.Circle(c0[0], c0[1], rad), 1, ring)
			}
			p.StrokePolyline([][2]float32{c0, c1}, 1.2, t.Text.Alpha(0.6))
		}
		tip := at([2]float64{real(z), imag(z)})
		var dot ui.Path
		p.FillPath(dot.Circle(tip[0], tip[1], 4), t.Accent)
	})
	return view
}

// sum returns where the chain of terms ends at phase, a fraction of a
// turn of the first.
func sum(terms []Term, phase float64) [2]float64 {
	var z complex128
	for _, tm := range terms {
		z += cmplx.Rect(tm.Amp, tm.Phase+2*math.Pi*float64(tm.Freq)*phase)
	}
	return [2]float64{real(z), imag(z)}
}
