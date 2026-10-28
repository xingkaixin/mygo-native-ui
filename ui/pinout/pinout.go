// Package pinout shows the pins of a board or a chip, as pinout.xyz and
// datasheets do: the pins in their rows, colored by what they carry, each
// with its name and the functions it can take as tags in the colors of
// their buses. Hovering over a pin lights it and its row; clicking one
// chooses it. Hovering over a tag, or clicking one in the legend above,
// lights every pin of that bus or kind and fades the rest, as the I2C
// pins or the grounds.
//
//	pinout.View(c, &app.pins, &app.pin, pinout.Board{Pins: pins})
//
// A Header numbers its pins across the rows, odd on the left, as pin
// headers do; a DIP down the left side and back up the right, as a chip
// in a dual in-line package.
package pinout

import (
	"strconv"

	"github.com/egoist/mygo/ui"
)

// Kind is what a pin carries.
type Kind int

const (
	// IO is a pin for signals, such as a GPIO.
	IO Kind = iota
	// Power supplies a voltage.
	Power
	// Ground is a ground.
	Ground
	// Other is anything else, such as a reset or a reference.
	Other
)

func (k Kind) String() string {
	switch k {
	case Power:
		return "Power"
	case Ground:
		return "Ground"
	case Other:
		return "Other"
	}
	return "I/O"
}

// Function is what else a pin can be, as "SDA" of the bus "I2C".
type Function struct {
	Name, Group string
}

// Pin is a pin: its name, as "GPIO 2" or "3V3", what it carries, and its
// functions.
type Pin struct {
	Name      string
	Kind      Kind
	Functions []Function
}

// Layout is how a board's pins are numbered.
type Layout int

const (
	// Header numbers pins across, 1 and 2 in the first row.
	Header Layout = iota
	// DIP numbers pins down the left side, then up the right.
	DIP
)

// Board is a part's pins, in order from pin 1, an even number of them.
type Board struct {
	Layout Layout
	Pins   []Pin
}

// State is what a pinout keeps: the bus or kind lit from the legend. Keep
// it in the app's state, one for each pinout.
type State struct {
	group string
	hover string
}

var palette = []ui.Color{
	ui.Oklch(0.6, 0.15, 255), ui.Oklch(0.65, 0.16, 50), ui.Oklch(0.58, 0.16, 300), ui.Oklch(0.62, 0.12, 195),
	ui.Oklch(0.6, 0.17, 0), ui.Oklch(0.68, 0.14, 95), ui.Oklch(0.55, 0.1, 230), ui.Oklch(0.6, 0.12, 135),
}

func kindColor(k Kind, dark bool) ui.Color {
	switch k {
	case Power:
		return ui.Oklch(0.6, 0.2, 25)
	case Ground:
		if dark {
			return ui.Oklch(0.55, 0, 0)
		}
		return ui.Oklch(0.3, 0, 0)
	case Other:
		return ui.Oklch(0.65, 0.03, 260)
	}
	return ui.Oklch(0.62, 0.15, 150)
}

// View creates the pinout of b, *selected the index of the chosen pin,
// or -1 for none.
func View(c *ui.Context, s *State, selected *int, b Board) ui.Element {
	t := c.Theme()
	colors := map[string]ui.Color{}
	var groups []string
	for _, p := range b.Pins {
		for _, f := range p.Functions {
			if _, ok := colors[f.Group]; !ok {
				colors[f.Group] = palette[len(groups)%len(palette)]
				groups = append(groups, f.Group)
			}
		}
	}
	// Tags and the legend say they are hovered as they are built, after
	// the pins above them; a change shows in the next frame.
	hovered := s.hover
	active := s.group
	if hovered != "" {
		active = hovered
	}
	s.hover = ""
	lit := func(p Pin) bool {
		if active == "" || p.Kind.String() == active {
			return true
		}
		for _, f := range p.Functions {
			if f.Group == active {
				return true
			}
		}
		return false
	}
	root := ui.Column(c).Gap(t.Space(4)).Role(ui.RoleGroup).Label("Pinout")
	root.Children(func() {
		ui.Row(c).Gap(t.Space(1.5)).Wrap().Justify(ui.Center).Children(func() {
			kinds := []Kind{}
			for _, k := range []Kind{Power, Ground, IO, Other} {
				for _, p := range b.Pins {
					if p.Kind == k {
						kinds = append(kinds, k)
						break
					}
				}
			}
			for _, k := range kinds {
				legend(c.Key("kind"+k.String()), s, k.String(), kindColor(k, t.Dark), true)
			}
			for _, g := range groups {
				legend(c.Key("group"+g), s, g, colors[g], false)
			}
		})
		n := len(b.Pins) / 2
		ui.Column(c).Children(func() {
			for r := range n {
				l, rt := 2*r, 2*r+1
				if b.Layout == DIP {
					l, rt = r, len(b.Pins)-1-r
				}
				ui.Row(c.Key(r)).AlignItems(ui.Stretch).Children(func() {
					side(c.Key("l"), s, selected, b.Pins, l, true, colors, lit)
					board(c.Key("b"), s, selected, b, l, rt, r, n, lit)
					side(c.Key("r"), s, selected, b.Pins, rt, false, colors, lit)
				})
			}
		})
	})
	if s.hover != hovered {
		c.Invalidate()
	}
	return root
}

// legend creates the tag of a bus or kind above the pins, which lights its
// pins while hovered and until clicked again once clicked.
func legend(c *ui.Context, s *State, name string, col ui.Color, kind bool) {
	t := c.Theme()
	on := s.group == name
	e := ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Padding(t.Space(1), t.Space(2.5)).Radius(t.Space(3)).
		Cursor(ui.CursorPointer).Role(ui.RoleToggleButton).Checked(on).Label(name)
	if e.Clicked() {
		if on {
			s.group = ""
		} else {
			s.group = name
		}
		on = !on
	}
	if e.Hovered() {
		s.hover = name
	}
	switch {
	case on:
		e.Background(col.Alpha(0.22)).Border(1, col)
	case e.Hovered():
		e.Background(col.Alpha(0.12)).Border(1, col.Alpha(0.5))
	default:
		e.Border(1, t.Border)
	}
	e.Children(func() {
		r := float32(4)
		if kind {
			r = 2
		}
		ui.Box(c).Size(8, 8).Background(col).Radius(r)
		ui.Text(c, name).FontSize(t.FontSize - 1)
	})
}

// side creates the name and tags of pin i, beside the board on the left
// or the right.
func side(c *ui.Context, s *State, selected *int, pins []Pin, i int, left bool, colors map[string]ui.Color, lit func(Pin) bool) {
	t := c.Theme()
	p := pins[i]
	e := ui.Row(c).Grow(1).Basis(0).Gap(t.Space(1.5)).AlignItems(ui.Center).PaddingX(t.Space(2)).Height(t.Space(7)).
		Cursor(ui.CursorPointer).Role(ui.RoleButton).Label("Pin " + strconv.Itoa(i+1) + " " + p.Name)
	if left {
		e.Justify(ui.End)
	}
	if e.Clicked() {
		toggle(selected, i)
	}
	style(e, t, *selected == i, e.Hovered(), lit(p))
	e.Children(func() {
		name := func() {
			ui.Text(c, p.Name).FontWeight(600).FontSize(t.FontSize - 1).NoWrap()
		}
		tags := func() {
			for k := range p.Functions {
				f := p.Functions[k]
				if left {
					f = p.Functions[len(p.Functions)-1-k]
				}
				col := colors[f.Group]
				tag := ui.Text(c.Key(k), f.Name).FontSize(t.FontSize-2).FontWeight(500).NoWrap().TextColor(col).
					Background(col.Alpha(0.14)).Padding(t.Space(0.25), t.Space(1.5)).Radius(t.Space(1)).Tooltip(f.Group)
				if tag.Hovered() {
					s.hover = f.Group
				}
			}
		}
		if left {
			tags()
			name()
		} else {
			name()
			tags()
		}
	})
}

// board creates the row of the board between pins l and r, the row'th of
// n.
func board(c *ui.Context, s *State, selected *int, b Board, l, r, row, n int, lit func(Pin) bool) {
	t := c.Theme()
	pcb := ui.Oklch(0.36, 0.07, 160)
	gap := t.Space(3)
	if b.Layout == DIP {
		pcb, gap = ui.Oklch(0.22, 0, 0), t.Space(16)
	}
	e := ui.Row(c).Gap(gap).AlignItems(ui.Center).PaddingX(t.Space(1.5)).Background(pcb)
	switch {
	case row == 0 && row == n-1:
		e.Radius(t.Space(1.5))
	case row == 0:
		e.Radius(t.Space(1.5), t.Space(1.5), 0, 0)
	case row == n-1:
		e.Radius(0, 0, t.Space(1.5), t.Space(1.5))
	}
	if b.Layout == DIP && row == 0 {
		e.Draw(func(p *ui.Painter, rc ui.Rect) {
			var notch ui.Path
			notch.MoveTo(rc.X+rc.W/2-8, rc.Y).QuadTo(rc.X+rc.W/2, rc.Y+14, rc.X+rc.W/2+8, rc.Y)
			p.FillPath(&notch, t.Background.Alpha(0.35))
		})
	}
	e.Children(func() {
		pin(c.Key(l), s, selected, b.Pins, l, lit)
		pin(c.Key(r), s, selected, b.Pins, r, lit)
	})
}

func pin(c *ui.Context, s *State, selected *int, pins []Pin, i int, lit func(Pin) bool) {
	t := c.Theme()
	p := pins[i]
	size := t.Space(4.5)
	e := ui.Box(c).Size(size, size).Background(kindColor(p.Kind, t.Dark)).Border(2, ui.Oklch(0.82, 0.08, 85)).
		Cursor(ui.CursorPointer).Tooltip(strconv.Itoa(i + 1))
	// Pin 1 is square, as boards mark it.
	if i == 0 {
		e.Radius(2)
	} else {
		e.Radius(size / 2)
	}
	if e.Clicked() {
		toggle(selected, i)
	}
	if !lit(p) {
		e.Opacity(0.25)
	}
	if *selected == i {
		e.Shadow(0, 0, 0, 3, t.Accent)
	}
}

func toggle(selected *int, i int) {
	if *selected == i {
		*selected = -1
	} else {
		*selected = i
	}
}

func style(e ui.Element, t *ui.Theme, chosen, hovered, lit bool) {
	switch {
	case chosen:
		e.Background(t.Accent.Alpha(0.14))
	case hovered:
		e.Background(t.Text.Alpha(0.05))
	}
	if !lit {
		e.Opacity(0.3)
	}
}
