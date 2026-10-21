package boxplot

import (
	"math"
	"slices"
)

// summary is what a chart computes of a group's samples, kept while they
// hold: how many are finite, the smallest and the largest, the quartiles
// and the median, the mean, the ends of the whiskers and the outliers
// beyond them, by their indexes, and the smallest above 0, where a log
// scale starts. density is the samples' density at evenly spaced values
// of the axis from dlo to dhi, each a fraction of the densest; log says
// that the axis, and the density, is over the samples' logarithms.
type summary struct {
	values []float64
	log    bool

	n                              int
	min, q1, median, q3, max, mean float64
	low, high                      float64
	outliers                       []int
	pos                            float64

	density  []float32
	dlo, dhi float64
}

// densitySteps is how many values a violin's outline goes through.
const densitySteps = 96

// holds reports whether m is the summary of values on an axis over their
// logarithms or not, as log says.
func (m *summary) holds(values []float64, log bool) bool {
	return m.log == log && slices.EqualFunc(values, m.values, func(a, b float64) bool {
		return a == b || math.IsNaN(a) && math.IsNaN(b)
	})
}

// summarize returns the summary of values, leaving out NaN and the
// infinities, its density over their logarithms when log.
func summarize(values []float64, log bool) summary {
	m := summary{values: slices.Clone(values), log: log, pos: math.Inf(1)}
	sorted := make([]float64, 0, len(values))
	sum := 0.0
	for _, v := range values {
		if !finite(v) {
			continue
		}
		sorted = append(sorted, v)
		sum += v
		if v > 0 {
			m.pos = min(m.pos, v)
		}
	}
	m.n = len(sorted)
	if m.n == 0 {
		return m
	}
	slices.Sort(sorted)
	m.min, m.max, m.mean = sorted[0], sorted[m.n-1], sum/float64(m.n)
	m.q1, m.median, m.q3 = quantile(sorted, 0.25), quantile(sorted, 0.5), quantile(sorted, 0.75)

	reach := 1.5 * (m.q3 - m.q1)
	lo, hi := m.q1-reach, m.q3+reach
	m.low, m.high = m.q1, m.q3
	for _, v := range sorted {
		if v >= lo {
			m.low = min(v, m.q1)
			break
		}
	}
	for j := m.n - 1; j >= 0; j-- {
		if sorted[j] <= hi {
			m.high = max(sorted[j], m.q3)
			break
		}
	}
	for j, v := range values {
		if finite(v) && (v < lo || v > hi) {
			m.outliers = append(m.outliers, j)
		}
	}

	xs := sorted
	if log {
		xs = make([]float64, 0, m.n)
		for _, v := range sorted {
			if v > 0 {
				xs = append(xs, math.Log10(v))
			}
		}
	}
	m.density, m.dlo, m.dhi = kde(xs)
	return m
}

// quantile returns the p quantile of sorted, interpolated between the two
// samples nearest it by rank, as NumPy's percentile and Excel's
// QUARTILE.INC compute it (Hyndman and Fan's type 7).
func quantile(sorted []float64, p float64) float64 {
	h := p * float64(len(sorted)-1)
	i := int(h)
	if i+1 >= len(sorted) {
		return sorted[len(sorted)-1]
	}
	return sorted[i] + (h-float64(i))*(sorted[i+1]-sorted[i])
}

// kde returns the density of xs, ascending, at densitySteps values evenly
// spaced from the smallest to the largest, each a fraction of the
// densest: a kernel density estimate, its kernel Gaussian and its
// bandwidth Silverman's rule of thumb, 0.9 × min(σ, IQR / 1.34) × n^-⅕.
// It is nil for fewer than two different values.
func kde(xs []float64) (density []float32, lo, hi float64) {
	n := len(xs)
	if n < 2 || xs[0] == xs[n-1] {
		return nil, 0, 0
	}
	mean, ss := 0.0, 0.0
	for _, x := range xs {
		mean += x
	}
	mean /= float64(n)
	for _, x := range xs {
		ss += (x - mean) * (x - mean)
	}
	sd := math.Sqrt(ss / float64(n-1))
	spread := min(sd, (quantile(xs, 0.75)-quantile(xs, 0.25))/1.34)
	if spread <= 0 {
		spread = sd
	}
	h := 0.9 * spread * math.Pow(float64(n), -0.2)
	lo, hi = xs[0], xs[n-1]
	density = make([]float32, densitySteps)
	peak := float32(0)
	for k := range density {
		at := lo + (hi-lo)*float64(k)/float64(densitySteps-1)
		// Samples more than 4 bandwidths away add less than e^-8 each.
		from, _ := slices.BinarySearch(xs, at-4*h)
		sum := 0.0
		for _, x := range xs[from:] {
			if x > at+4*h {
				break
			}
			u := (x - at) / h
			sum += math.Exp(-u * u / 2)
		}
		density[k] = float32(sum)
		peak = max(peak, density[k])
	}
	for k := range density {
		density[k] /= peak
	}
	return density, lo, hi
}

// jitter returns where sample j sits across its column, from -1 to 1:
// scattered by a hash of j, so that it stays put from frame to frame.
func jitter(j int) float32 {
	z := uint64(j)*0x9e3779b97f4a7c15 + 0x632be59bd9b4e019
	z = (z ^ z>>30) * 0xbf58476d1ce4e5b9
	z = (z ^ z>>27) * 0x94d049bb133111eb
	z ^= z >> 31
	return float32(z>>40)/(1<<23) - 1
}

func finite(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) }
