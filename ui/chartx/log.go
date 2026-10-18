package chartx

import (
	"math"
	"strconv"

	"github.com/egoist/mygo/ui/chart"
)

// Log maps numbers from Lo to Hi onto the plot by their logarithms, as
// d3's scaleLog and Swift Charts' log scales: each power of ten takes as
// much room as the next, so that values of several orders of magnitude
// show on one plot. It goes where a chart.Linear goes: At places the
// points of lines and marks, Invert reads the pointer, and Ticks go to
// chart.Axes and grid lines.
//
// Lo and Hi are above 0. Values at or below 0 have no logarithm and no
// place on the plot: At returns NaN for them, which breaks a line or an
// area there (chart.Line), so leave them out of other marks.
//
//	y := chartx.NewLog(3, 870_000).Nice() // 1 to 1M
//	ticks := y.Ticks(6)
type Log struct{ Lo, Hi float64 }

// NewLog returns a logarithmic scale from lo to hi, both above 0.
func NewLog(lo, hi float64) Log { return Log{Lo: lo, Hi: hi} }

// At returns where v is along the plot, NaN for v at or below 0.
func (s Log) At(v float64) float32 {
	if v <= 0 {
		return float32(math.NaN())
	}
	a, b := math.Log10(s.Lo), math.Log10(s.Hi)
	if a == b {
		return 0.5
	}
	return float32((math.Log10(v) - a) / (b - a))
}

// Invert returns the value at f along the plot, as where the pointer is.
func (s Log) Invert(f float32) float64 {
	if f == 1 {
		return s.Hi // Lo*(Hi/Lo) can miss it: 0.9999999999999999 from 1e-5 to 1
	}
	return s.Lo * math.Pow(s.Hi/s.Lo, float64(f))
}

// Nice extends the domain to powers of ten, as d3's: 3 to 870,000 becomes
// 1 to 1,000,000.
func (s Log) Nice() Log {
	lo, hi := s.Lo, s.Hi
	reverse := hi < lo
	if reverse {
		lo, hi = hi, lo
	}
	if !(lo > 0) || math.IsInf(hi, 0) {
		return s
	}
	lo = math.Pow10(decade(lo))
	if e := decade(hi); math.Pow10(e) < hi {
		hi = math.Pow10(e + 1)
	}
	if reverse {
		lo, hi = hi, lo
	}
	return Log{Lo: lo, Hi: hi}
}

// Ticks returns ticks within the domain with about count labels, as d3's
// log ticks. For a domain spanning more than count powers of ten, they
// are the powers, or every second, fifth or tenth of them. For fewer,
// they are the powers and the multiples 2 to 9 of each between them,
// labeled from the smallest multiples up as far as count allows: the
// others are minor ticks, with no Label, which the axes leave bare and
// grid lines draw fainter. A domain spanning less than a factor of ten,
// as one zoomed in, gets chart.Linear's ticks, 1, 2 or 5 times a power of
// ten apart, which spread across it where the multiples would bunch at
// one end.
//
// Labels use SI prefixes from a thousand up, and decimals below:
// "0.00005", "200", "20k", "1M".
func (s Log) Ticks(count int) []chart.Tick {
	lo, hi := min(s.Lo, s.Hi), max(s.Lo, s.Hi)
	if count <= 0 || !(lo > 0) || math.IsInf(hi, 0) {
		return nil
	}
	tick := func(v float64, labeled bool) chart.Tick {
		t := chart.Tick{At: s.At(v), Value: v}
		if labeled {
			t.Label = formatSI(v)
		}
		return t
	}
	var out []chart.Tick
	if a, b := log10(lo), log10(hi); b-a > float64(count) {
		for _, e := range chart.NewLinear(a, b).Ticks(count) {
			out = append(out, tick(math.Pow10(int(math.Round(e.Value))), true))
		}
		return out
	}
	var values []float64
	var multiples []int
	for e := decade(lo); e <= decade(hi); e++ {
		for k := 1; k <= 9; k++ {
			v := float64(k) * math.Pow10(e)
			if e < 0 {
				v = float64(k) / math.Pow10(-e) // 0.3, not 0.30000000000000004
			}
			if v < lo {
				continue
			}
			if v > hi {
				break
			}
			values, multiples = append(values, v), append(multiples, k)
		}
	}
	if hi < 10*lo || 2*len(values) < count {
		for _, t := range chart.NewLinear(lo, hi).Ticks(count) {
			out = append(out, tick(t.Value, true))
		}
		return out
	}
	// As d3's log tickFormat: the multiples up to most label about count
	// ticks.
	most := max(1, 10*float64(count)/float64(len(values)))
	for i, v := range values {
		out = append(out, tick(v, float64(multiples[i]) <= most))
	}
	return out
}

// log10 is math.Log10, but exact at powers of ten, where math.Log10 is a
// little off at some (decade): a domain ending at one keeps its tick.
func log10(v float64) float64 {
	if e := decade(v); math.Pow10(e) == v {
		return float64(e)
	}
	return math.Log10(v)
}

// decade returns the exponent of the power of ten at or below v > 0.
// math.Log10 is a little off at some powers: 14.999999999999998 at 1e15.
func decade(v float64) int {
	e := int(math.Floor(math.Log10(v)))
	switch {
	case math.Pow10(e+1) <= v:
		e++
	case math.Pow10(e) > v:
		e--
	}
	return e
}

// formatSI labels a tick with an SI prefix from a thousand up, "20k",
// "1.5M", and in decimals below: "200", "0.00005".
func formatSI(v float64) string {
	const prefixes = "kMGTPE"
	i := -1
	for v >= 1000 && i < len(prefixes)-1 {
		v /= 1000
		i++
	}
	s := strconv.FormatFloat(v, 'f', -1, 64)
	if i >= 0 {
		s += prefixes[i : i+1]
	}
	return s
}
