// Package spectrogram shows how loud each frequency of a sound is over
// time, as audio editors' spectral views do: time across, frequency up
// on a log axis from 20 Hz, or a linear one from 0, to half the sample
// rate, and the level in dB as a color from black through purple and
// orange to pale yellow, matplotlib's magma. A legend beside it reads
// the colors in dB. Hovering reads out the time, the frequency with its
// nearest note, and the level there.
//
// Live shows the last ten seconds of a sound as it plays, the newest at
// the right. The app gives it the samples heard since the last frame,
// as those up to its player's position, each frame while it plays; given
// none, as while paused, it stands still:
//
//	at := int(player.Position().Seconds() * rate)
//	spectrogram.Live(c, &app.live, samples[app.heard:at], rate).Grow(1)
//	app.heard = at
//
// Clip shows the whole of a clip, from Analyze, with a playhead, which a
// click or a drag moves, as waveform's does:
//
//	gram := spectrogram.Analyze(samples, rate) // once, not every frame
//	...
//	app.whole.Playhead = player.Position().Seconds() / duration
//	if spectrogram.Clip(c, &app.whole, gram).Height(160).Changed() {
//		player.Seek(time.Duration(app.whole.Playhead * duration * float64(time.Second)))
//	}
package spectrogram

import (
	"cmp"
	"image"
	"math"
	"strconv"

	"github.com/egoist/mygo/ui"
)

const (
	// low is where a log axis starts, in Hz.
	low = 20.0
	// span is how many seconds Live shows.
	span = 10.0
	// rows is how many rows of levels the picture has, spaced along the
	// frequency axis, and wide the most columns Analyze makes.
	rows = 512
	wide = 4096
)

// magma is matplotlib's magma colormap in 256 steps, from Matt Zucker's
// polynomial fit of it.
var magma = func() (lut [256][4]uint8) {
	k := [7][3]float64{
		{-0.002136485053939582, -0.000749655052795221, -0.005386127855323933},
		{0.2516605407371642, 0.6775232436837668, 2.494026599312351},
		{8.353717279216625, -3.577719514958484, 0.3144679030132573},
		{-27.66873308576866, 14.26473078096533, -13.64921318813922},
		{52.17613981234068, -27.94360607168351, 12.94416944238394},
		{-50.76852536473588, 29.04658282127291, 4.23415299384598},
		{18.65570506591883, -11.48977351997711, -5.601961508734096},
	}
	for i := range lut {
		x := float64(i) / 255
		for ch := range 3 {
			v := 0.0
			for d := 6; d >= 0; d-- {
				v = v*x + k[d][ch]
			}
			lut[i][ch] = uint8(math.Round(255 * max(0, min(1, v))))
		}
		lut[i][3] = 255
	}
	return lut
}()

// legend is the colormap upright, brightest at the top.
var legend = func() *ui.Bitmap {
	img := image.NewRGBA(image.Rect(0, 0, 1, len(magma)))
	for i, c := range magma {
		copy(img.Pix[4*(len(magma)-1-i):], c[:])
	}
	return ui.NewBitmap(img)
}()

var black = ui.RGB(magma[0][0], magma[0][1], magma[0][2])

// fftSize returns the window for rate: a power of two near 43 ms, 2048
// samples at 48 kHz.
func fftSize(rate int) int {
	n := 256
	for n*3/2 < rate*2048/48000 {
		n *= 2
	}
	return n
}

// window is a Hann window, the buffer it transforms in, and the power it
// adds up.
type window struct {
	hann []float64
	buf  []complex128
	acc  []float64
}

func newWindow(size int) *window {
	w := &window{hann: make([]float64, size), buf: make([]complex128, size), acc: make([]float64, size/2+1)}
	for i := range w.hann {
		w.hann[i] = 0.5 - 0.5*math.Cos(2*math.Pi*float64(i)/float64(size-1))
	}
	return w
}

// add adds the power of each bin of the window of samples from at, zero
// outside them, to acc, so that a full-scale sine's bin is 1.
func (w *window) add(samples []float32, at int) {
	for i := range w.buf {
		v := 0.0
		if j := at + i; j >= 0 && j < len(samples) {
			v = float64(samples[j]) * w.hann[i]
		}
		w.buf[i] = complex(v, 0)
	}
	fft(w.buf)
	// Through the Hann window, a full-scale sine peaks at a quarter of the
	// window's length.
	k := 4 / float64(len(w.buf))
	for b := range w.acc {
		re, im := real(w.buf[b])*k, imag(w.buf[b])*k
		w.acc[b] += re*re + im*im
	}
}

// levels writes the mean of n windows' power in acc to out in dB, and
// clears acc.
func (w *window) levels(n int, out []float32) {
	for b, v := range w.acc {
		out[b] = float32(10 * math.Log10(max(v/float64(n), 1e-20)))
	}
	clear(w.acc)
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
		a := -2 * math.Pi / float64(l)
		w := complex(math.Cos(a), math.Sin(a))
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

// Analysis is the levels of a clip at each time and frequency, from
// Analyze, for Clip.
type Analysis struct {
	rate, size, hop int
	seconds         float64
	frames          [][]float32
}

var empty = &Analysis{rate: 48000, size: fftSize(48000), hop: 512}

// Analyze finds the levels of samples, of one channel at rate frames a
// second, for Clip: in Hann windows of about 43 ms, 2048 samples at
// 48 kHz, every quarter window, or, for a clip longer than 4096 of those
// steps, in 4096 steps, each the mean of the windows across it. A long
// clip takes a while: call it once, or on a goroutine.
func Analyze(samples []float32, rate int) *Analysis {
	if rate <= 0 || len(samples) == 0 {
		return empty
	}
	size := fftSize(rate)
	hop := max(size/4, (len(samples)+wide-1)/wide)
	n := (len(samples) + hop - 1) / hop
	a := &Analysis{rate: rate, size: size, hop: hop, seconds: float64(len(samples)) / float64(rate), frames: make([][]float32, n)}
	w := newWindow(size)
	bins := size/2 + 1
	data := make([]float32, n*bins)
	per := max(1, hop/size)
	for j := range a.frames {
		for m := range per {
			mid := j*hop + (2*m+1)*hop/(2*per)
			w.add(samples, mid-size/2)
		}
		a.frames[j] = data[j*bins : (j+1)*bins]
		w.levels(per, a.frames[j])
	}
	return a
}

// State is what a spectrogram shows and keeps: its scales, a clip's
// playhead, and the levels it shows and their picture. Keep it in the
// app's state, one for each view. The zero State shows -100 to 0 dB on
// a log frequency axis.
type State struct {
	// Linear spaces frequencies evenly from 0, where by default each
	// octave from 20 Hz takes as much room.
	Linear bool
	// Floor and Ceiling are the levels drawn black and brightest, in dB
	// of full scale. A Floor not below the Ceiling is 100 dB below it.
	Floor, Ceiling float64
	// Playhead is where the clip plays, a fraction of it, which Clip
	// shows and a click moves.
	Playhead float64

	rate, size, hop int
	// seconds is the time across the view.
	seconds float64
	// frames are the levels of each column in dB, by bin; Live's are a
	// ring, the oldest at head, nil until heard.
	frames [][]float32
	head   int
	clip   *Analysis
	// win and tail are Live's window and the samples it has yet to take.
	win  *window
	tail []float32
	// bands are where each row of the picture takes its level from, img
	// the picture, and drawn what it was drawn for.
	bands []band
	img   *image.RGBA
	bm    *ui.Bitmap
	drawn look
}

// look is what the picture depends on besides the levels.
type look struct {
	rate, size int
	linear     bool
	lo, hi     float64
}

// levels returns the floor and the ceiling.
func (s *State) levels() (lo, hi float64) {
	if s.Floor >= s.Ceiling {
		return s.Ceiling - 100, s.Ceiling
	}
	return s.Floor, s.Ceiling
}

// band is where a row of the picture takes its level from: the loudest
// of bins k0 to k1, or, for a row between two bins, k1 to k0, at a
// fraction from k1.
type band struct {
	k0, k1 int
	at     float32
}

func (b band) level(f []float32) float32 {
	if b.k1 < b.k0 {
		return f[b.k1] + (f[b.k0]-f[b.k1])*b.at
	}
	v := f[b.k0]
	for _, x := range f[b.k0+1 : b.k1+1] {
		v = max(v, x)
	}
	return v
}

// freq returns the frequency at u, from 0 at the bottom of the axis to 1
// at its top, and at the u of f.
func (s *State) freq(u float64) float64 {
	top := float64(s.rate) / 2
	if s.Linear {
		return u * top
	}
	return low * math.Pow(top/low, u)
}

func (s *State) at(f float64) float64 {
	top := float64(s.rate) / 2
	if s.Linear {
		return f / top
	}
	return math.Log(f/low) / math.Log(top/low)
}

// row returns the bottom, middle and top of row r, as freq takes them.
func row(r int) (u0, mid, u1 float64) {
	u0, u1 = float64(rows-1-r)/rows, float64(rows-r)/rows
	return u0, (u0 + u1) / 2, u1
}

// picture draws the picture anew if it is not of the look now, and
// reports whether it did.
func (s *State) picture() bool {
	lo, hi := s.levels()
	l := look{s.rate, s.size, s.Linear, lo, hi}
	if s.img != nil && l == s.drawn {
		return false
	}
	s.drawn = l
	if n := max(1, len(s.frames)); s.img == nil || s.img.Rect.Dx() != n {
		s.img = image.NewRGBA(image.Rect(0, 0, n, rows))
		for i := 0; i < len(s.img.Pix); i += 4 {
			copy(s.img.Pix[i:], magma[0][:])
		}
	}
	bins := s.size/2 + 1
	hz := float64(s.rate) / float64(s.size)
	s.bands = s.bands[:0]
	for r := range rows {
		u0, mid, u1 := row(r)
		b := band{k0: min(bins-1, int(math.Ceil(s.freq(u0)/hz))), k1: min(bins-1, int(s.freq(u1)/hz))}
		if b.k1 < b.k0 {
			b.at = float32(s.freq(mid)/hz) - float32(b.k1)
		}
		s.bands = append(s.bands, b)
	}
	for j := range s.frames {
		s.paint(j)
	}
	return true
}

// paint draws column j of the picture from its levels, black with none.
func (s *State) paint(j int) {
	f := s.frames[j]
	lo, hi := s.drawn.lo, s.drawn.hi
	for r, b := range s.bands {
		c := magma[0]
		if f != nil {
			c = magma[int(255*max(0, min(1, (float64(b.level(f))-lo)/(hi-lo))))]
		}
		copy(s.img.Pix[r*s.img.Stride+4*j:], c[:])
	}
}

// show hands the picture to its bitmap.
func (s *State) show() {
	if s.bm == nil {
		s.bm = ui.NewBitmap(s.img)
	} else {
		s.bm.Update(s.img)
	}
}

// listen analyzes heard after the samples before it, a window every hop,
// and reports whether it painted any.
func (s *State) listen(heard []float32) bool {
	// Only what fits in the ring shows.
	if keep := len(s.frames)*s.hop + s.size; len(heard) > keep {
		heard, s.tail = heard[len(heard)-keep:], s.tail[:0]
	}
	s.tail = append(s.tail, heard...)
	n := 0
	for ; n+s.size <= len(s.tail); n += s.hop {
		s.win.add(s.tail, n)
		if s.frames[s.head] == nil {
			s.frames[s.head] = make([]float32, s.size/2+1)
		}
		s.win.levels(1, s.frames[s.head])
		s.paint(s.head)
		s.head = (s.head + 1) % len(s.frames)
	}
	s.tail = s.tail[:copy(s.tail, s.tail[n:])]
	return n > 0
}

// Live creates a view of the last ten seconds of a sound, of which heard
// is what was heard since the last frame: samples of one channel at rate
// frames a second. Give it a size, or grow it in its parent.
func Live(c *ui.Context, s *State, heard []float32, rate int) ui.Element {
	if rate <= 0 {
		rate = cmp.Or(s.rate, 48000)
	}
	if s.win == nil || s.rate != rate {
		s.clip, s.rate, s.size = nil, rate, fftSize(rate)
		s.hop = s.size / 4
		s.frames = make([][]float32, int(math.Ceil(span*float64(rate)/float64(s.hop))))
		s.seconds = float64(len(s.frames)*s.hop) / float64(rate)
		s.head, s.tail, s.win, s.img = 0, s.tail[:0], newWindow(s.size), nil
	}
	drew := s.picture()
	if s.listen(heard) || drew {
		s.show()
	}
	return s.view(c)
}

// Clip creates a view of the whole of a, from Analyze, with the playhead
// at s.Playhead. Give it a size, or grow it in its parent. Changed
// reports the user moving the playhead: a press or a drag moves it to
// the pointer.
func Clip(c *ui.Context, s *State, a *Analysis) ui.Element {
	if a == nil {
		a = empty
	}
	if s.clip != a {
		s.clip, s.win, s.tail, s.img = a, nil, nil, nil
		s.rate, s.size, s.hop, s.seconds, s.frames, s.head = a.rate, a.size, a.hop, a.seconds, a.frames, 0
	}
	if s.picture() {
		s.show()
	}
	view := s.view(c)
	if _, _, ok := view.Dragged(); ok {
		px, _, _ := view.PointerPosition()
		b := view.LayoutBounds()
		g := plot(c.Theme(), ui.Rect{W: b.W, H: b.H})
		if f := max(0, min(1, float64((px-g.X)/g.W))); f != s.Playhead {
			s.Playhead = f
			view.SetChanged()
		}
	}
	return view
}

// plot returns where the levels show in a view of r, leaving room for
// the axes and the legend.
func plot(t *ui.Theme, r ui.Rect) ui.Rect {
	left, top, right, bottom := t.Space(9), t.Space(2), t.Space(14), t.Space(5)
	return ui.Rect{X: r.X + left, Y: r.Y + top, W: max(1, r.W-left-right), H: max(1, r.H-top-bottom)}
}

// view creates the view of the levels and their picture.
func (s *State) view(c *ui.Context) ui.Element {
	t := c.Theme()
	view := ui.Box(c).Cursor(ui.CursorCrosshair).Label("Spectrogram")
	px, py, over := view.PointerPosition()
	clip := s.clip != nil
	bm, frames, head := s.bm, s.frames, s.head
	lo, hi := s.levels()
	view.Draw(func(p *ui.Painter, r ui.Rect) {
		g := plot(t, r)
		small := t.FontSize - 3
		label := func(text string, x, y, ax, ay float32) {
			sp := ui.Span{Text: text, Size: small, Color: t.TextMuted, Features: "tnum"}
			w, h := p.MeasureText(0, sp)
			p.RichText(max(r.X, min(r.X+r.W-w, x-w*ax)), max(r.Y, min(r.Y+r.H-h, y-h*ay)), 0, sp)
		}
		n := float32(len(frames))
		p.Clip(g, 3, func() {
			p.Fill(g, black, 0)
			if len(frames) > 0 {
				old := g.W * (n - float32(head)) / n
				p.ImageRegion(bm, ui.Rect{X: float32(head), W: n - float32(head), H: rows}, ui.Rect{X: g.X, Y: g.Y, W: old, H: g.H}, ui.ImageOptions{})
				if head > 0 {
					p.ImageRegion(bm, ui.Rect{W: float32(head), H: rows}, ui.Rect{X: g.X + old, Y: g.Y, W: g.W - old, H: g.H}, ui.ImageOptions{})
				}
			}
			if clip {
				x := g.X + g.W*float32(max(0, min(1, s.Playhead)))
				p.Fill(ui.Rect{X: x - 1, Y: g.Y, W: 2, H: g.H}, ui.RGB(255, 255, 255), 1)
			}
		})

		for _, f := range s.ticks(g.H) {
			y := g.Y + g.H*float32(1-s.at(f))
			p.Line(g.X-t.Space(1), y, g.X, y, 1, t.Border)
			label(axis(f), g.X-t.Space(1.5), y, 1, 0.5)
		}
		step := nice(s.seconds, g.W, t.Space(16), []float64{0.1, 0.2, 0.5, 1, 2, 5, 10, 15, 30, 60, 120, 300, 600, 1200})
		for i := 0; s.seconds > 0 && float64(i)*step <= s.seconds+1e-9; i++ {
			v := math.Round(float64(i)*step*10) / 10
			x := g.X + g.W*float32(v/s.seconds)
			text := strconv.FormatFloat(v, 'f', -1, 64) + " s"
			if !clip {
				x = g.X + g.W - g.W*float32(v/s.seconds)
				if v > 0 {
					text = "-" + text
				}
			}
			p.Line(x, g.Y+g.H, x, g.Y+g.H+t.Space(1), 1, t.Border)
			label(text, x, g.Y+g.H+t.Space(1.25), 0.5, 0)
		}

		bar := ui.Rect{X: g.X + g.W + t.Space(3), Y: g.Y, W: t.Space(2), H: g.H}
		p.Clip(bar, 2, func() { p.ImageRegion(legend, ui.Rect{}, bar, ui.ImageOptions{}) })
		db := nice(hi-lo, g.H, t.Space(6), []float64{5, 10, 20, 50, 100})
		for v := math.Ceil(lo/db) * db; v <= hi; v += db {
			y := g.Y + g.H*float32((hi-v)/(hi-lo))
			text := strconv.Itoa(int(v))
			if v+db > hi {
				text += " dB"
			}
			label(text, bar.X+bar.W+t.Space(1.5), y, 0, 0.5)
		}

		x, y := r.X+px, r.Y+py
		if !over || x < g.X || x >= g.X+g.W || y < g.Y || y >= g.Y+g.H {
			return
		}
		guide := ui.RGBA(255, 255, 255, 0.35)
		p.Line(x, g.Y, x, g.Y+g.H, 1, guide)
		p.Line(g.X, y, g.X+g.W, y, 1, guide)
		sp := ui.Span{Text: s.readout(frames, head, float64((x-g.X)/g.W), float64((y-g.Y)/g.H)), Size: t.FontSize - 2, Color: cmp.Or(t.InverseText, t.Background), Features: "tnum"}
		w, h := p.MeasureText(0, sp)
		pad := t.Space(2)
		box := ui.Rect{X: x + t.Space(3), Y: y + t.Space(3), W: w + 2*pad, H: h + pad}
		if box.X+box.W > g.X+g.W {
			box.X = max(r.X, x-t.Space(3)-box.W)
		}
		if box.Y+box.H > g.Y+g.H {
			box.Y = max(r.Y, y-t.Space(3)-box.H)
		}
		p.Fill(box, cmp.Or(t.Inverse, t.Text), 5)
		p.RichText(box.X+pad, box.Y+pad/2, 0, sp)
	})
	return view
}

// ticks returns the frequencies to label on an axis h DIPs high: 1, 2
// and 5 times powers of ten on a log axis, as many on a linear one as
// leave room between.
func (s *State) ticks(h float32) []float64 {
	top := float64(s.rate) / 2
	gap := float64(16 / h)
	var out []float64
	if s.Linear {
		step := nice(top, h, 28, []float64{100, 200, 500, 1000, 2000, 5000, 10000, 20000})
		for f := 0.0; f <= top; f += step {
			out = append(out, f)
		}
		return out
	}
	last := math.Inf(-1)
	for d := 10.0; d < top; d *= 10 {
		for _, m := range []float64{1, 2, 5} {
			if f := d * m; f >= low && f <= top && s.at(f)-last >= gap {
				out, last = append(out, f), s.at(f)
			}
		}
	}
	return out
}

// nice returns the first of steps that puts gap DIPs between ticks
// across a range of size shown over px DIPs, or the last.
func nice(size float64, px, gap float32, steps []float64) float64 {
	for _, st := range steps {
		if float32(st/size)*px >= gap {
			return st
		}
	}
	return steps[len(steps)-1]
}

// axis labels a frequency, in kHz from 1000.
func axis(f float64) string {
	if f >= 1000 {
		return strconv.FormatFloat(f/1000, 'f', -1, 64) + "k"
	}
	return strconv.Itoa(int(f))
}

var notes = [...]string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}

// readout says the time, the frequency and its nearest note, and the
// level at fx across the plot and fy down it.
func (s *State) readout(frames [][]float32, head int, fx, fy float64) string {
	j := min(len(frames)-1, int(fx*float64(len(frames))))
	r := min(rows-1, int(fy*rows))
	_, mid, _ := row(r)
	f := s.freq(mid)
	t := fx * s.seconds
	if s.clip == nil {
		t -= s.seconds
	}
	out := strconv.FormatFloat(t, 'f', 2, 64) + " s   "
	if f >= 1000 {
		out += strconv.FormatFloat(f/1000, 'f', 2, 64) + " kHz"
	} else {
		out += strconv.FormatFloat(f, 'f', 0, 64) + " Hz"
	}
	if f >= low {
		m := int(math.Round(69 + 12*math.Log2(f/440)))
		out += " " + notes[m%12] + strconv.Itoa(m/12-1)
	}
	if j >= 0 && len(s.bands) == rows {
		if lv := frames[(head+j)%len(frames)]; lv != nil {
			db := "-∞"
			// Silence reads as the least level windows give.
			if v := s.bands[r].level(lv); v > -199 {
				db = strconv.FormatFloat(float64(v), 'f', 0, 64)
			}
			out += "   " + db + " dB"
		}
	}
	return out
}
