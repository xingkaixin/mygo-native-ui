package ridgeline

import (
	"math"
	"slices"
	"sort"
)

// ridge is what a chart computes of a group's samples, kept while they
// hold: the finite ones in order, their median, and their bandwidth by
// Silverman's rule, 0 for fewer than two different values. density is the
// samples' density at steps values evenly spaced from lo to hi, with the
// bandwidth times bw.
type ridge struct {
	values []float64
	sorted []float64
	median float64
	h      float64

	density    []float64
	lo, hi, bw float64
}

// steps is how many values a ridge's outline goes through.
const steps = 240

// holds reports whether r is the ridge of values.
func (r *ridge) holds(values []float64) bool {
	return slices.EqualFunc(values, r.values, func(a, b float64) bool {
		return a == b || math.IsNaN(a) && math.IsNaN(b)
	})
}

// newRidge returns the ridge of values, leaving out NaN and the
// infinities.
func newRidge(values []float64) ridge {
	r := ridge{values: slices.Clone(values)}
	for _, v := range values {
		if !math.IsNaN(v) && !math.IsInf(v, 0) {
			r.sorted = append(r.sorted, v)
		}
	}
	slices.Sort(r.sorted)
	n := len(r.sorted)
	if n == 0 {
		return r
	}
	r.median = quantile(r.sorted, 0.5)
	if n < 2 || r.sorted[0] == r.sorted[n-1] {
		return r
	}
	mean, ss := 0.0, 0.0
	for _, v := range r.sorted {
		mean += v
	}
	mean /= float64(n)
	for _, v := range r.sorted {
		ss += (v - mean) * (v - mean)
	}
	sd := math.Sqrt(ss / float64(n-1))
	spread := min(sd, (quantile(r.sorted, 0.75)-quantile(r.sorted, 0.25))/1.34)
	if spread <= 0 {
		spread = sd
	}
	r.h = 0.9 * spread * math.Pow(float64(n), -0.2)
	return r
}

// densities returns r's density at steps values from lo to hi, its
// bandwidth times bw, or nil when it has none.
func (r *ridge) densities(lo, hi, bw float64) []float64 {
	if r.h == 0 {
		return nil
	}
	if r.density != nil && r.lo == lo && r.hi == hi && r.bw == bw {
		return r.density
	}
	r.lo, r.hi, r.bw = lo, hi, bw
	r.density = make([]float64, steps)
	for k := range r.density {
		r.density[k] = r.at(lo+(hi-lo)*float64(k)/(steps-1), bw)
	}
	return r.density
}

// at returns r's density at x, its bandwidth times bw: a kernel density
// estimate with a Gaussian kernel, per unit of x.
func (r *ridge) at(x, bw float64) float64 {
	h := r.h * bw
	if h == 0 {
		return 0
	}
	// Samples more than 4 bandwidths away add less than e^-8 each.
	from, _ := slices.BinarySearch(r.sorted, x-4*h)
	sum := 0.0
	for _, v := range r.sorted[from:] {
		if v > x+4*h {
			break
		}
		u := (v - x) / h
		sum += math.Exp(-u * u / 2)
	}
	return sum / (float64(len(r.sorted)) * h * math.Sqrt(2*math.Pi))
}

// below returns how many of r's samples are below x.
func (r *ridge) below(x float64) int { return sort.SearchFloat64s(r.sorted, x) }

// quantile returns the p quantile of sorted, interpolated between the two
// samples nearest it by rank, as NumPy's percentile computes it (Hyndman
// and Fan's type 7).
func quantile(sorted []float64, p float64) float64 {
	h := max(0, min(1, p)) * float64(len(sorted)-1)
	i := int(h)
	if i+1 >= len(sorted) {
		return sorted[len(sorted)-1]
	}
	return sorted[i] + (h-float64(i))*(sorted[i+1]-sorted[i])
}
