// Chartx shows a year of daily downloads of four packages, from a few a
// day to over half a million, on a logarithmic scale, with a legend that
// shows and hides each package, and zooming and panning along the year.
//
//	go run ./examples/chartx
package main

import (
	"log"
	"math"
	"math/rand/v2"
	"runtime"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/chart"

	"github.com/xingkaixin/mygo-native-ui/ui/chartx"
)

const days = 365

var (
	first = time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	names = []string{"Core", "CLI", "Plugins", "Templates"}
	// colors read in light and dark windows alike.
	colors = []ui.Color{ui.Hex("#3b82f6"), ui.Hex("#10b981"), ui.Hex("#f59e0b"), ui.Hex("#ec4899")}
)

// downloads is the state the window shows: each package's downloads by
// day, which packages show, the stretch of the year shown, and the y scale
// fitting the packages shown.
type downloads struct {
	daily   [][]float64
	shown   []bool
	visible chart.Span
	y       chartx.Log
}

// newDownloads returns the downloads, the same at every start: each
// package's grow from January's to December's, with fewer at weekends.
func newDownloads() *downloads {
	r := rand.New(rand.NewPCG(1, 2))
	ranges := [][2]float64{{150_000, 600_000}, {20_000, 160_000}, {600, 30_000}, {6, 4_000}}
	d := &downloads{shown: []bool{true, true, true, true}, visible: chart.Span{Lo: 0, Hi: 1}}
	for _, rg := range ranges {
		values := make([]float64, days)
		for i := range values {
			v := rg[0] * math.Pow(rg[1]/rg[0], float64(i)/(days-1)) * math.Exp(0.15*r.NormFloat64())
			if wd := first.AddDate(0, 0, i).Weekday(); wd == time.Saturday || wd == time.Sunday {
				v *= 0.6
			}
			values[i] = v
		}
		d.daily = append(d.daily, values)
	}
	d.fitY()
	return d
}

// fitY fits the y scale to the packages shown, out to powers of ten.
func (d *downloads) fitY() {
	lo, hi := math.Inf(1), math.Inf(-1)
	for s, values := range d.daily {
		if d.shown[s] {
			l, h := chart.Extent(values...)
			lo, hi = min(lo, l), max(hi, h)
		}
	}
	if lo > hi {
		lo, hi = 1, 10
	}
	d.y = chartx.NewLog(lo, hi).Nice()
}

// view builds the window: the legend above the chart, and how to zoom
// below it.
func (d *downloads) view(c *ui.Context) {
	t := c.Theme()
	year := chart.NewTime(first, first.AddDate(0, 0, days-1))
	x := chart.NewTime(year.Invert(d.visible.Lo), year.Invert(d.visible.Hi))
	xt, yt := x.Ticks(6), d.y.Ticks(6)
	xs := make([]float32, days)
	for i := range xs {
		xs[i] = x.At(first.AddDate(0, 0, i))
	}
	label := func(_ chart.Side, tick chart.Tick) {
		ui.Text(c, tick.Label).NoWrap().FontSize(11).TextColor(t.TextMuted)
	}
	zoomKey := "Ctrl"
	if runtime.GOOS == "darwin" {
		zoomKey = "⌘"
	}
	ui.Column(c).Fill().Padding(20).Gap(12).Children(func() {
		ui.Text(c, "Daily downloads").FontSize(16).Bold()
		// The items are padded around their swatches, which line up with
		// the title.
		legend := chartx.Legend(c, names, colors, d.shown).Label("Packages").Margin(0, 0, 0, -t.Space(2))
		if legend.Changed() {
			d.fitY()
		}
		chart.Plot(c, chart.Axes{Left: yt, Bottom: xt, Label: label}, func(area ui.Element) {
			chartx.Zoom(c, area, &d.visible, 14.0/(days-1))
			area.Draw(func(p *ui.Painter, r ui.Rect) {
				for _, tick := range yt {
					color := t.Border
					if tick.Label == "" {
						color = color.Alpha(0.4)
					}
					p.Line(r.X, chart.Y(r, tick.At), r.X+r.W, chart.Y(r, tick.At), 1, color)
				}
				columns := int(math.Ceil(float64(r.W * p.Scale())))
				ys := make([]float32, days)
				p.Clip(r, 0, func() {
					for s, values := range d.daily {
						if !d.shown[s] {
							continue
						}
						for i, v := range values {
							ys[i] = d.y.At(v)
						}
						bx, by := chart.MinMax(nil, nil, xs, ys, columns)
						p.StrokePath(chart.Line(r, bx, by, chart.CurveLinear), 1.5, colors[s])
					}
				})
			})
		}).Label("Downloads by day").Description("Daily downloads of four packages in 2025, on a logarithmic scale").
			Grow(1).GapX(8).GapY(6)
		ui.Text(c, "Drag or swipe sideways to pan, "+zoomKey+"-scroll to zoom, double-click to show the whole year").
			FontSize(12).TextColor(t.TextMuted)
	})
}

func main() {
	d := newDownloads()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Downloads",
			Width:   760,
			Height:  520,
			Content: ui.View(d.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
