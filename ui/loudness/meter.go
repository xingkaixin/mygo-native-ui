package loudness

import (
	"math"
	"slices"
)

// Meter measures loudness as ITU-R BS.1770 and EBU R128 define it: the
// samples K-weighted, their power summed over the channels and taken
// over 400 ms (momentary), 3 s (short-term) and the whole programme
// (integrated, gated), and the spread of short-term loudness (loudness
// range). Every channel weighs the same, as the front channels do, so it
// suits mono and stereo. Make one with NewMeter and keep it in the app's
// state.
type Meter struct {
	rate     float64
	channels int
	filters  []kfilter
	// block is the 100 ms block being summed: its samples so far and each
	// channel's sum of squares.
	n      int
	sums   []float64
	blocks []float64 // mean power of each finished 100 ms block
	gated  []float64 // power of each 400 ms block, every 100 ms
	short  []float64 // short-term loudness every 100 ms, for its range
	peak   float64
}

// NewMeter returns a meter for audio of sampleRate samples a second in
// channels channels.
func NewMeter(sampleRate float64, channels int) *Meter {
	m := &Meter{rate: sampleRate, channels: channels}
	m.filters = make([]kfilter, channels)
	for i := range m.filters {
		m.filters[i] = newKFilter(sampleRate)
	}
	m.sums = make([]float64, channels)
	return m
}

// Write measures samples, interleaved by channel, from -1 to 1.
func (m *Meter) Write(samples []float32) {
	per := int(m.rate / 10)
	for i := 0; i+m.channels <= len(samples); i += m.channels {
		for ch := range m.channels {
			x := float64(samples[i+ch])
			m.peak = max(m.peak, math.Abs(x))
			y := m.filters[ch].step(x)
			m.sums[ch] += y * y
		}
		m.n++
		if m.n == per {
			power := 0.0
			for ch := range m.sums {
				power += m.sums[ch] / float64(per)
				m.sums[ch] = 0
			}
			m.n = 0
			m.blocks = append(m.blocks, power)
			if len(m.blocks) >= 4 {
				m.gated = append(m.gated, m.mean(4))
			}
			if len(m.blocks) >= 30 {
				m.short = append(m.short, lufs(m.mean(30)))
			}
		}
	}
}

// Reset forgets everything measured.
func (m *Meter) Reset() {
	*m = *NewMeter(m.rate, m.channels)
}

// mean returns the mean power of the last n blocks.
func (m *Meter) mean(n int) float64 {
	sum := 0.0
	for _, p := range m.blocks[len(m.blocks)-n:] {
		sum += p
	}
	return sum / float64(n)
}

func lufs(power float64) float64 {
	if power <= 0 {
		return math.Inf(-1)
	}
	return -0.691 + 10*math.Log10(power)
}

// Momentary returns the loudness of the last 400 ms, in LUFS.
func (m *Meter) Momentary() float64 {
	if len(m.blocks) < 4 {
		return math.Inf(-1)
	}
	return lufs(m.mean(4))
}

// ShortTerm returns the loudness of the last 3 s, in LUFS.
func (m *Meter) ShortTerm() float64 {
	if len(m.blocks) < 30 {
		return math.Inf(-1)
	}
	return lufs(m.mean(30))
}

// Integrated returns the loudness of everything measured, in LUFS: the
// 400 ms blocks louder than -70 LUFS, and of those the ones within 10 LU
// of their mean.
func (m *Meter) Integrated() float64 {
	gate := func(blocks []float64, floor float64) (float64, int) {
		sum, n := 0.0, 0
		for _, p := range blocks {
			if lufs(p) > floor {
				sum += p
				n++
			}
		}
		if n == 0 {
			return 0, 0
		}
		return sum / float64(n), n
	}
	mean, n := gate(m.gated, -70)
	if n == 0 {
		return math.Inf(-1)
	}
	mean, n = gate(m.gated, max(-70, lufs(mean)-10))
	if n == 0 {
		return math.Inf(-1)
	}
	return lufs(mean)
}

// Range returns the loudness range, in LU: how far apart the 10th and
// 95th percentiles of the short-term loudness are, leaving out what is
// quieter than -70 LUFS or than 20 LU below its mean.
func (m *Meter) Range() float64 {
	var loud []float64
	sum := 0.0
	for _, l := range m.short {
		if l > -70 {
			loud = append(loud, l)
			sum += math.Pow(10, (l+0.691)/10)
		}
	}
	if len(loud) == 0 {
		return 0
	}
	floor := lufs(sum/float64(len(loud))) - 20
	var kept []float64
	for _, l := range loud {
		if l > floor {
			kept = append(kept, l)
		}
	}
	if len(kept) < 2 {
		return 0
	}
	slices.Sort(kept)
	at := func(q float64) float64 { return kept[int(math.Round(q*float64(len(kept)-1)))] }
	return at(0.95) - at(0.10)
}

// Peak returns the largest sample measured, in dBFS.
func (m *Meter) Peak() float64 {
	if m.peak == 0 {
		return math.Inf(-1)
	}
	return 20 * math.Log10(m.peak)
}

// History returns the short-term loudness every 100 ms, oldest first.
func (m *Meter) History() []float64 { return m.short }

// kfilter is BS.1770's K-weighting: a high shelf for the head's effect
// and a high-pass filter, each a biquad.
type kfilter struct {
	stages [2]biquad
}

type biquad struct {
	b0, b1, b2, a1, a2 float64
	z1, z2             float64
}

func (q *biquad) step(x float64) float64 {
	y := q.b0*x + q.z1
	q.z1 = q.b1*x - q.a1*y + q.z2
	q.z2 = q.b2*x - q.a2*y
	return y
}

func (k *kfilter) step(x float64) float64 {
	return k.stages[1].step(k.stages[0].step(x))
}

// newKFilter returns the K-weighting at a sample rate. BS.1770 gives its
// coefficients at 48 kHz only; these are the analog prototypes they come
// from, as libebur128 derives them.
func newKFilter(rate float64) kfilter {
	var k kfilter
	{
		f0, gain, q := 1681.974450955533, 3.999843853973347, 0.7071752369554196
		kk := math.Tan(math.Pi * f0 / rate)
		vh := math.Pow(10, gain/20)
		vb := math.Pow(vh, 0.4996667741545416)
		a0 := 1 + kk/q + kk*kk
		k.stages[0] = biquad{
			b0: (vh + vb*kk/q + kk*kk) / a0,
			b1: 2 * (kk*kk - vh) / a0,
			b2: (vh - vb*kk/q + kk*kk) / a0,
			a1: 2 * (kk*kk - 1) / a0,
			a2: (1 - kk/q + kk*kk) / a0,
		}
	}
	{
		f0, q := 38.13547087602444, 0.5003270373238773
		kk := math.Tan(math.Pi * f0 / rate)
		a0 := 1 + kk/q + kk*kk
		k.stages[1] = biquad{b0: 1, b1: -2, b2: 1, a1: 2 * (kk*kk - 1) / a0, a2: (1 - kk/q + kk*kk) / a0}
	}
	return k
}
