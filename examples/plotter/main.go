// Plotter graphs the expressions typed on the left, as Desmos does: a
// field for each, with its curve's color by it and what is wrong with it
// under it, a slider for each constant such as a = 0.5, which the others
// use, and the graph on the right. Drag to pan and scroll to zoom; Reset
// view goes back to the origin.
//
//	go run ./examples/plotter
package main

import (
	"log"
	"math"
	"slices"
	"strconv"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/plotter"
)

// hues are the hues of the curves, in the order expressions take them.
var hues = []float32{25, 250, 145, 300, 60, 200}

// line is an expression of the list: its source, and its id, which keys
// its field and picks its hue.
type line struct {
	id     int
	source string
}

type app struct {
	lines []line
	next  int
	// focus is the id of the line Add made, whose field takes the focus.
	focus int
	graph plotter.State
}

func color(id int, dark bool) ui.Color {
	l := float32(0.58)
	if dark {
		l = 0.72
	}
	return ui.Oklch(l, 0.17, hues[id%len(hues)])
}

func (a *app) add(source string) {
	a.lines = append(a.lines, line{id: a.next, source: source})
	a.next++
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	exprs := make([]plotter.Expr, len(a.lines))
	for i, l := range a.lines {
		exprs[i] = plotter.Expr{Source: l.source, Color: color(l.id, t.Dark)}
	}
	errs := plotter.Errors(exprs)
	// The list changes once it is built, which the fields' indexes into
	// exprs and errs are of.
	added, remove := false, -1
	ui.Row(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Width(t.Space(80)).AlignSelf(ui.Stretch).Children(func() {
			ui.Row(c).Padding(t.Space(3), t.Space(4)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "Expressions").Bold().Grow(1)
				added = ui.Button(c, "Add").Clicked()
			})
			ui.Divider(c)
			ui.Scroll(c).Grow(1).Children(func() {
				ui.Column(c).Children(func() {
					for i := range a.lines {
						if a.field(c.Key(a.lines[i].id), &a.lines[i], exprs[i].Color, errs[i]) {
							remove = i
						}
						ui.Divider(c)
					}
				})
			})
		})
		ui.Divider(c).Vertical()
		ui.Column(c).Grow(1).AlignSelf(ui.Stretch).Children(func() {
			ui.Row(c).Padding(t.Space(2), t.Space(4)).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "Drag to pan, scroll to zoom, with Shift or Option along one axis").
					FontSize(t.FontSize - 1).TextColor(t.TextMuted).Grow(1).Shrink(1)
				if ui.Button(c, "Reset view").Clicked() {
					a.graph.Reset()
				}
			})
			ui.Divider(c)
			plotter.View(c, &a.graph, exprs).Grow(1).AlignSelf(ui.Stretch)
		})
	})
	if remove >= 0 {
		a.lines = slices.Delete(a.lines, remove, remove+1)
	}
	if added {
		a.focus = a.next
		a.add("")
	}
}

// field builds the row of l: its color, its field, a button removing it,
// what is wrong with it, and a slider when it defines a constant as a
// number. It reports whether the button was clicked.
func (a *app) field(c *ui.Context, l *line, color ui.Color, err error) bool {
	t := c.Theme()
	removed := false
	dot := t.Space(3)
	name, value, constant := plotter.Constant(l.source)
	row := ui.Column(c).Gap(t.Space(1.5)).Padding(t.Space(2.5), t.Space(3), t.Space(2.5), t.Space(4))
	hovered := row.Hovered()
	row.Children(func() {
		ui.Row(c).Gap(t.Space(2.5)).AlignItems(ui.Center).Children(func() {
			swatch := ui.Box(c).Size(dot, dot).Radius(dot / 2)
			if !constant {
				swatch.Background(color)
			}
			input := ui.TextInput(c, &l.source).Grow(1).Shrink(1).MinWidth(0).Label("Expression").Placeholder("y = f(x)")
			if l.id == a.focus {
				input.AutoFocus()
			}
			x := ui.ButtonBase(c).Size(t.Space(6), t.Space(6)).Radius(t.Space(1)).Center().Label("Remove")
			if x.Hovered() {
				x.Background(t.SurfaceHover)
			}
			removed = x.Clicked()
			x.Children(func() {
				if hovered || input.Focused() {
					ui.Text(c, "×").TextColor(t.TextMuted)
				}
			})
		})
		indent := dot + t.Space(2.5)
		if err != nil {
			ui.Text(c, err.Error()).FontSize(t.FontSize - 1).TextColor(t.Danger).MarginStart(indent)
		}
		if constant {
			lo, hi := min(-5, math.Floor(value)), max(5, math.Ceil(value))
			ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).MarginStart(indent).Children(func() {
				end := func(v float64) {
					ui.Text(c, strconv.FormatFloat(v, 'f', -1, 64)).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
				}
				end(lo)
				if ui.Slider(c, &value, lo, hi).Grow(1).Label(name).Changed() {
					l.source = name + " = " + strconv.FormatFloat(math.Round(value*100)/100, 'f', -1, 64)
				}
				end(hi)
			})
		}
	})
	return removed
}

func main() {
	a := &app{focus: -1}
	for _, s := range []string{"y = sin(x)", "y = a x^2 - 2", "r = 1 + cos(θ)", "(cos(3t), sin(2t))", "a = 0.5"} {
		a.add(s)
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Plotter",
			Width:   1100,
			Height:  700,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
