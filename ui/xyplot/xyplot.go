// Package xyplot compares pictures made over two settings, as the outputs
// of an image model are compared across seeds and guidance: a grid with a
// column for each value of one setting and a row for each of the other,
// headed by their values, with the settings' names in the corner. A cell
// the app has no picture for yet shows a spinner. Pointing at a cell
// lights its row's and column's values; a click opens it large, under its
// two values, where the arrow keys move through the grid and Escape, a
// click or Grid goes back. Swap axes turns the grid so the rows become the
// columns, and the slider sizes the cells.
//
//	xyplot.View(c, &app.plot, app.cfg, app.seeds, func(x, y int) *ui.Bitmap {
//		return app.images[y][x]
//	})
package xyplot

import (
	"github.com/egoist/mygo/ui"
)

// Axis is a setting and the values it was tried at.
type Axis struct {
	Name   string
	Values []string
}

// State is what a plot keeps: the size of the cells, whether the axes are
// swapped, and the cell open, if one is. Keep it in the app's state, one
// for each plot.
type State struct {
	size    float64
	swapped bool
	open    bool
	col     int
	row     int
	hover   [2]int
}

// View creates the plot of the pictures picture gives for each value x of
// xs and y of ys, nil for one not made yet.
func View(c *ui.Context, s *State, xs, ys Axis, picture func(x, y int) *ui.Bitmap) ui.Element {
	t := c.Theme()
	if s.size == 0 {
		s.size = 128
	}
	cols, rows := xs, ys
	if s.swapped {
		cols, rows = ys, xs
	}
	at := func(col, row int) *ui.Bitmap {
		if s.swapped {
			return picture(row, col)
		}
		return picture(col, row)
	}
	nc, nr := len(cols.Values), len(rows.Values)
	root := ui.Column(c).Gap(t.Space(4)).Focusable().FocusRing(false).Label(xs.Name + " by " + ys.Name)
	if s.open {
		switch {
		case root.Shortcut(0, ui.KeyEscape):
			s.open = false
		case root.Shortcut(0, ui.KeyLeft):
			s.col = max(0, s.col-1)
		case root.Shortcut(0, ui.KeyRight):
			s.col = min(nc-1, s.col+1)
		case root.Shortcut(0, ui.KeyUp):
			s.row = max(0, s.row-1)
		case root.Shortcut(0, ui.KeyDown):
			s.row = min(nr-1, s.row+1)
		}
	}
	hover := s.hover
	s.hover = [2]int{-1, -1}
	root.Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			if s.open {
				if ui.Button(c, "Grid").Clicked() {
					s.open = false
				}
				return
			}
			if ui.Button(c, "Swap axes").Clicked() {
				s.swapped = !s.swapped
			}
			ui.Box(c).Grow(1)
			ui.Text(c, "Size").TextColor(t.TextMuted)
			ui.Slider(c, &s.size, 64, 256).Width(t.Space(40)).Label("Size")
		})
		if s.open && s.col < nc && s.row < nr {
			ui.Text(c, cols.Name+" "+cols.Values[s.col]+" · "+rows.Name+" "+rows.Values[s.row]).Bold()
			big := ui.Box(c).Grow(1).FillWidth().MinHeight(t.Space(40)).Cursor(ui.CursorPointer).Role(ui.RoleImage).
				Label(cols.Values[s.col] + ", " + rows.Values[s.row])
			if big.Clicked() {
				s.open = false
			}
			pic := at(s.col, s.row)
			if pic == nil {
				big.Center().Children(func() { ui.Spinner(c) })
				return
			}
			big.Draw(func(p *ui.Painter, r ui.Rect) {
				p.Image(pic, r, ui.Contain)
			})
			return
		}
		size := float32(s.size)
		head := t.Space(20)
		ui.ScrollBoth(c).Grow(1).FillWidth().Children(func() {
			ui.Column(c).Gap(t.Space(1.5)).Children(func() {
				ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.End).Children(func() {
					ui.Column(c).Width(head).Gap(t.Space(0.5)).Children(func() {
						ui.Text(c, cols.Name+" →").FontSize(t.FontSize - 1).Bold()
						ui.Text(c, rows.Name+" ↓").FontSize(t.FontSize - 1).Bold()
					})
					for i, v := range cols.Values {
						value(c.Key(i), v, hover[0] == i).Width(size).TextAlign(ui.Center)
					}
				})
				for j, rv := range rows.Values {
					ui.Row(c.Key(j)).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
						value(c, rv, hover[1] == j).Width(head).TextAlign(ui.End).Padding(0, t.Space(2), 0, 0)
						for i := range nc {
							cell := ui.Box(c.Key(i)).Size(size, size).Radius(t.Space(1)).Background(t.Surface).Clip().
								Cursor(ui.CursorPointer).Role(ui.RoleImage).Label(cols.Values[i] + ", " + rv)
							if cell.Hovered() {
								s.hover = [2]int{i, j}
							}
							if cell.Clicked() {
								s.open, s.col, s.row = true, i, j
								root.Focus()
							}
							pic := at(i, j)
							if pic == nil {
								cell.Center().Children(func() { ui.Spinner(c) })
								continue
							}
							lit := hover == [2]int{i, j}
							cell.Draw(func(p *ui.Painter, r ui.Rect) {
								p.Image(pic, r, ui.Cover)
								if lit {
									p.Stroke(r, t.Accent, t.Space(1), 2)
								}
							})
						}
					})
				}
			})
		})
	})
	// The headers above were built with the cell pointed at last frame.
	if hover != s.hover {
		c.AnimationFrame()
	}
	return root
}

// value creates the text of a setting's value, in the accent color when
// lit.
func value(c *ui.Context, v string, lit bool) ui.Element {
	t := c.Theme()
	e := ui.Text(c, v).FontSize(t.FontSize - 1).FontFeatures("tnum").TextColor(t.TextMuted)
	if lit {
		e.TextColor(t.Accent).Bold()
	}
	return e
}
