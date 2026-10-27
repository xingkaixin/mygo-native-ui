// Package gridpicker chooses the size of a table as Insert Table does in
// Word and Notion: the cells from the top left corner to the one under
// the pointer light up, the size shows below them, and a click chooses
// it. The grid grows a row and a column past the pointer, up to the
// largest size allowed. While it has the keyboard focus, the arrow keys
// move the lit corner and Enter chooses.
//
//	if rows, cols, ok := gridpicker.View(c, 10, 10).Chosen(); ok {
//		app.insertTable(rows, cols)
//	}
//
// It fits in a popover below a toolbar button.
package gridpicker

import (
	"fmt"

	"github.com/egoist/mygo/ui"
)

// least is how many rows and columns the grid shows to begin with.
const least = 5

// Events are what the user did with a picker in a frame.
type Events struct {
	rows, cols int
}

// Chosen returns the size the user chose.
func (e Events) Chosen() (rows, cols int, ok bool) { return e.rows, e.cols, e.rows > 0 }

// state is what a picker keeps from frame to frame: the lit size, from
// the pointer or the keys, zero when nothing is lit.
type state struct {
	rows, cols int
}

// View creates a picker of sizes up to maxRows by maxCols.
func View(c *ui.Context, maxRows, maxCols int) Events {
	var ev Events
	t := c.Theme()
	cell, gap := t.Space(4.5), t.Space(0.75)
	root := ui.Column(c).Gap(t.Space(2)).Padding(t.Space(1)).Focusable().Label("Table size")
	st := ui.Local(root, "gridpicker.state", func() state { return state{} })
	lit := func(r, k int) {
		st.rows, st.cols = max(1, min(maxRows, r)), max(1, min(maxCols, k))
	}
	switch {
	case root.Shortcut(0, ui.KeyRight):
		lit(st.rows, st.cols+1)
	case root.Shortcut(0, ui.KeyLeft):
		lit(st.rows, st.cols-1)
	case root.Shortcut(0, ui.KeyDown):
		lit(st.rows+1, st.cols)
	case root.Shortcut(0, ui.KeyUp):
		lit(st.rows-1, st.cols)
	case root.Shortcut(0, ui.KeyEnter) && st.rows > 0:
		ev = Events{st.rows, st.cols}
	}
	root.Children(func() {
		rows := min(maxRows, max(least, st.rows+1))
		cols := min(maxCols, max(least, st.cols+1))
		pitch := cell + gap
		w := float32(cols)*pitch - gap
		grid := ui.Box(c).Size(w, float32(rows)*pitch-gap).Cursor(ui.CursorPointer)
		if x, y, over := grid.PointerPosition(); over {
			lit(int(y/pitch)+1, int(x/pitch)+1)
		}
		if grid.Clicked() && st.rows > 0 {
			ev = Events{st.rows, st.cols}
		}
		r0, c0 := st.rows, st.cols
		grid.Draw(func(p *ui.Painter, r ui.Rect) {
			for i := range rows {
				for k := range cols {
					b := ui.Rect{X: r.X + float32(k)*pitch, Y: r.Y + float32(i)*pitch, W: cell, H: cell}
					if i < r0 && k < c0 {
						p.Fill(b, t.Accent.Alpha(0.25), 2)
						p.Stroke(b, t.Accent, 2, 1)
					} else {
						p.Fill(b, t.Surface, 2)
						p.Stroke(b, t.Border, 2, 1)
					}
				}
			}
		})
		label := "Insert table"
		if st.rows > 0 {
			label = fmt.Sprintf("%d × %d", st.rows, st.cols)
		}
		root.Value(label)
		ui.Text(c, label).FontSize(t.FontSize - 1).FontFeatures("tnum").TextColor(t.TextMuted).TextAlign(ui.Center).Width(w)
	})
	return ev
}
