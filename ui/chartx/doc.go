// Package chartx adds to package ui/chart three things it leaves out: a
// logarithmic scale, a legend that shows and hides series, and zooming
// and panning along x. They go where chart's own parts do: Log is a scale
// as chart.Linear is, whose ticks go to chart.Axes and grid lines; Legend
// is a row to build by the plot; and Zoom takes the input of the plot's
// area, keeping the stretch of x that the plot shows, from which the x
// scale is built.
//
//	all := chart.NewTime(first, last)
//	x := chart.NewTime(all.Invert(app.view.Lo), all.Invert(app.view.Hi))
//	y := chartx.NewLog(lo, hi).Nice()
//	if chartx.Legend(c, names, colors, app.shown).Changed() {
//		app.fit()
//	}
//	chart.Plot(c, chart.Axes{Left: y.Ticks(6), Bottom: x.Ticks(6)}, func(area ui.Element) {
//		chartx.Zoom(c, area, &app.view, 0.02)
//		...
//	})
//
// See examples/chartx.
package chartx
