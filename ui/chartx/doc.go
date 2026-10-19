// Package chartx adds to package ui/chart what it leaves out: a
// logarithmic scale, a legend that shows and hides series, zooming and
// panning along x, heatmaps colored by a sequential scale, the bins of
// histograms, calendars of days as GitHub's of contributions, and
// sparklines.
//
// They go where chart's own parts do: Log is a scale as chart.Linear is,
// whose ticks go to chart.Axes and grid lines; Legend is a row to build by
// the plot; Zoom takes the input of the plot's area, keeping the stretch
// of x that the plot shows, from which the x scale is built; Heatmap
// paints the cells of a plot's area in the colors of a Sequential scale;
// and Bins sorts values into bins to draw as chart.Rect bars. Calendar and
// Sparkline stand alone, without a plot.
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
// See examples/chartx and examples/dashboard.
package chartx
