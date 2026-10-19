package chartx

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// Sequential maps numbers from Lo to Hi onto colors, as d3's
// scaleSequential: Lo takes the first of Colors, Hi the last, and the
// values between them the colors between, evenly spaced and blended. It
// colors the cells of a heatmap, from few to many. Values beyond the
// domain take the colors at its ends, and NaN takes ui.Transparent.
//
//	color := chartx.NewSequential(0, most, chartx.Viridis...)
//	cell := color.At(counts[day][hour])
//
// Gradient paints the scale as a bar, for a legend whose ends are Lo and
// Hi.
type Sequential struct {
	Lo, Hi float64
	Colors []ui.Color
}

// NewSequential returns a sequential scale from lo to hi through colors.
func NewSequential(lo, hi float64, colors ...ui.Color) Sequential {
	return Sequential{Lo: lo, Hi: hi, Colors: colors}
}

// At returns the color of v.
func (s Sequential) At(v float64) ui.Color {
	n := len(s.Colors)
	if n == 0 || math.IsNaN(v) {
		return ui.Transparent
	}
	f := 0.5
	if s.Hi != s.Lo {
		f = max(0, min(1, (v-s.Lo)/(s.Hi-s.Lo)))
	}
	at := f * float64(n-1)
	i := int(at)
	if i >= n-1 {
		return s.Colors[n-1]
	}
	return s.Colors[i].Mix(s.Colors[i+1], float32(at-float64(i)))
}

// Gradient returns the scale's colors from left to right, Lo to Hi, as an
// element's LinearGradient or a Painter's FillGradient.
func (s Sequential) Gradient() ui.LinearGradient {
	stops := make([]ui.GradientStop, len(s.Colors))
	for i, c := range s.Colors {
		stops[i] = ui.GradientStop{Color: c, At: float32(i) / float32(max(1, len(s.Colors)-1))}
	}
	return ui.LinearGradient{Angle: 90, Stops: stops}
}

// Viridis is matplotlib's viridis, from dark purple through blue and
// green to yellow, whose colors grow lighter all the way, so that they
// keep their order in gray and to most color-blind eyes.
var Viridis = []ui.Color{
	ui.Hex("#440154"), ui.Hex("#472c7a"), ui.Hex("#3b518b"), ui.Hex("#2c718e"), ui.Hex("#21908d"),
	ui.Hex("#27ad81"), ui.Hex("#5cc863"), ui.Hex("#aadc32"), ui.Hex("#fde725"),
}
