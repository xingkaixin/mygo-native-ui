package chartx

import (
	"math"
	"sort"

	"github.com/egoist/mygo/ui/chart"
)

// Bin is a bin of a histogram: how many values are from Lo up to Hi, Hi
// left out but by the last bin.
type Bin struct {
	Lo, Hi float64
	Count  int
}

// Bins sorts values into the bins of a histogram, as d3's bin: about
// count bins of one round width, 1, 2 or 5 times a power of ten as
// chart.Linear's ticks are apart, from a multiple of it at or below the
// least value to one at or above the greatest. NaN values go in none, and
// equal values in one bin of no width. Draw them as bars on a linear x
// scale from the first bin's Lo to the last's Hi:
//
//	bins := chartx.Bins(app.latencies, 20)
//	x := chart.NewLinear(bins[0].Lo, bins[len(bins)-1].Hi)
//	y := chart.NewLinear(0, most).Nice(5)
//	for _, b := range bins {
//		chart.Rect(c, x.Span(b.Lo, b.Hi), y.Span(0, float64(b.Count)))
//	}
func Bins(values []float64, count int) []Bin {
	lo, hi := math.Inf(1), math.Inf(-1)
	n := 0
	for _, v := range values {
		if !math.IsNaN(v) {
			lo, hi, n = min(lo, v), max(hi, v), n+1
		}
	}
	if n == 0 {
		return nil
	}
	ticks := chart.NewLinear(lo, hi).Ticks(max(1, count))
	if lo == hi || len(ticks) < 2 {
		return []Bin{{Lo: lo, Hi: hi, Count: n}}
	}
	step := ticks[1].Value - ticks[0].Value
	// Bounds are whole multiples of the step, divided by its inverse when
	// below 1, as d3's ticks, so that 0.1 steps give 0.3 and not
	// 0.30000000000000004.
	at := func(k float64) float64 { return k * step }
	from := math.Floor(lo / step)
	to := math.Ceil(hi / step)
	if step < 1 {
		inv := math.Round(1 / step)
		at = func(k float64) float64 { return k / inv }
		from, to = math.Floor(lo*inv), math.Ceil(hi*inv)
	}
	bins := make([]Bin, max(1, int(to-from)))
	for i := range bins {
		bins[i] = Bin{Lo: at(from + float64(i)), Hi: at(from + float64(i) + 1)}
	}
	for _, v := range values {
		if math.IsNaN(v) {
			continue
		}
		i := sort.Search(len(bins), func(i int) bool { return bins[i].Lo > v }) - 1
		bins[max(0, i)].Count++
	}
	return bins
}
