// Package segments shows text as the displays of clocks, meters and
// departure boards do: seven-segment digits, fourteen-segment letters, or
// a matrix of five by seven dots, the unlit segments faint behind the lit
// ones. A display looks like a red, green or amber LED, a cyan vacuum
// fluorescent display, or a dark-on-green LCD. A period lights the
// decimal point of the cell before it, and a colon takes a narrow cell of
// its own. Text shorter than the display sits at its right, as numbers
// do, or at its left in a matrix; text longer than a matrix scrolls
// through it.
//
//	segments.View(c, segments.Panel{Kind: segments.Seven, Cells: 4}, "12:34")
//
// Characters a kind has no shape for show blank: seven segments show
// digits, a minus and the letters that read on them.
package segments

import (
	"time"
	"unicode"

	"github.com/egoist/mygo/ui"
)

// Kind is how a display draws its characters.
type Kind int

const (
	// Seven is the segments of digits.
	Seven Kind = iota
	// Fourteen adds diagonals and split bars, for letters.
	Fourteen
	// Matrix is five dots across by seven down.
	Matrix
)

// Look is the light of a display: an LED's red, green or amber, a vacuum
// fluorescent display's cyan, or an LCD's dark segments on green.
type Look int

const (
	RedLED Look = iota
	GreenLED
	AmberLED
	VFD
	LCD
)

// Panel is a display: its kind, its look, how many characters it shows,
// and how tall they are, 48 DIPs when zero.
type Panel struct {
	Kind   Kind
	Look   Look
	Cells  int
	Height float32
}

func (l Look) colors() (lit, back ui.Color) {
	switch l {
	case GreenLED:
		return ui.RGB(57, 255, 110), ui.RGB(6, 16, 9)
	case AmberLED:
		return ui.RGB(255, 176, 0), ui.RGB(18, 12, 4)
	case VFD:
		return ui.RGB(95, 245, 225), ui.RGB(7, 18, 20)
	case LCD:
		return ui.RGB(28, 36, 20), ui.RGB(176, 190, 150)
	}
	return ui.RGB(255, 48, 30), ui.RGB(20, 6, 6)
}

// cell is a character as it shows: its segments or columns lit, and
// whether it is a colon or has its decimal point lit.
type cell struct {
	bits  uint16
	cols  [5]byte
	colon bool
	dot   bool
}

func (p Panel) cells(text string) []cell {
	var out []cell
	for _, r := range text {
		switch {
		case p.Kind != Matrix && r == '.' && len(out) > 0 && !out[len(out)-1].dot && !out[len(out)-1].colon:
			out[len(out)-1].dot = true
		case p.Kind != Matrix && r == ':':
			out = append(out, cell{colon: true})
		case p.Kind == Seven:
			out = append(out, cell{bits: seven(r)})
		case p.Kind == Fourteen:
			out = append(out, cell{bits: fourteen[unicode.ToUpper(r)]})
		default:
			cl := cell{}
			if r >= ' ' && int(r-' ') < len(font) {
				cl.cols = font[r-' ']
			}
			out = append(out, cl)
		}
	}
	return out
}

// View creates the display of text.
func View(c *ui.Context, p Panel, text string) ui.Element {
	h := p.Height
	if h <= 0 {
		h = 48
	}
	cells := p.cells(text)
	n := 0
	for _, cl := range cells {
		if !cl.colon {
			n++
		}
	}
	// Short text sits right, padded with blank cells; colons take no
	// cells of the count.
	if p.Kind != Matrix {
		for ; n < p.Cells; n++ {
			cells = append([]cell{{}}, cells...)
		}
	}
	lit, back := p.Look.colors()
	off := lit.Alpha(0.07)
	if p.Look == LCD {
		off = lit.Alpha(0.06)
	}
	pad := h * 0.22
	adv, colonW := h*0.74, h*0.3
	if p.Kind == Fourteen {
		adv = h * 0.86
	}
	pitch := h / 7
	if p.Kind == Matrix {
		adv = pitch * 6
	}
	w := adv * float32(p.Cells)
	if p.Kind != Matrix {
		w = 0
		for _, cl := range cells {
			if cl.colon {
				w += colonW
			} else {
				w += adv
			}
		}
	}
	e := ui.Box(c).Size(w+2*pad, h+2*pad).Background(back).Radius(h * 0.12).Clip().Role(ui.RoleImage).Label(text)
	look := p.Look
	e.Draw(func(pt *ui.Painter, r ui.Rect) {
		x := r.X + pad
		y := r.Y + pad
		if p.Kind == Matrix {
			shift := 0
			if len(cells) > p.Cells {
				// Long text scrolls a column every 60 ms, a blank cell
				// between its end and its start.
				total := (len(cells) + 1) * 6
				shift = int(pt.Now().UnixMilli()/60) % total
				pt.After(60 * time.Millisecond)
			}
			matrix(pt, x, y, pitch, p.Cells, cells, shift, lit, off)
			return
		}
		for _, cl := range cells {
			if cl.colon {
				colon(pt, x, y, colonW, h, lit, look)
				x += colonW
				continue
			}
			if p.Kind == Seven {
				digit(pt, x, y, h, cl, sevenSegs(h), lit, off, look)
			} else {
				digit(pt, x, y, h, cl, fourteenSegs(h), lit, off, look)
			}
			x += adv
		}
	})
	return e
}

// slant is how far the digits lean, as most displays' do.
const slant = 0.1

// seg is a segment: a bar between two points, its bit in a cell's bits.
type seg struct {
	x0, y0, x1, y1 float32
	bit            int
	thin           bool
}

func sevenSegs(h float32) []seg {
	w, th := h*0.52, h*0.11
	l, r, t, m, b := th/2, w-th/2, th/2, h/2, h-th/2
	return []seg{
		{l, t, r, t, 0, false}, {r, t, r, m, 1, false}, {r, m, r, b, 2, false}, {l, b, r, b, 3, false},
		{l, m, l, b, 4, false}, {l, t, l, m, 5, false}, {l, m, r, m, 6, false},
	}
}

func fourteenSegs(h float32) []seg {
	w, th := h*0.64, h*0.1
	l, r, t, m, b, cx := th/2, w-th/2, th/2, h/2, h-th/2, w/2
	// The diagonals run from just inside the corners to just short of
	// the middle.
	d, e := th*1.25, th*0.75
	return []seg{
		{l, t, r, t, 0, false}, {r, t, r, m, 1, false}, {r, m, r, b, 2, false}, {l, b, r, b, 3, false},
		{l, m, l, b, 4, false}, {l, t, l, m, 5, false}, {l, m, cx, m, 6, false}, {cx, m, r, m, 7, false},
		{l + d, t + d, cx - e, m - d, 8, true}, {cx, t, cx, m, 9, false}, {r - d, t + d, cx + e, m - d, 10, true},
		{l + d, b - d, cx - e, m + d, 11, true}, {cx, m, cx, b, 12, false}, {r - d, b - d, cx + e, m + d, 13, true},
	}
}

// digit draws the segments of cl at x, y, h tall, leaning by slant.
func digit(p *ui.Painter, x, y, h float32, cl cell, segs []seg, lit, off ui.Color, look Look) {
	th := h * 0.1
	lean := func(px, py float32) (float32, float32) { return x + px + (h-py)*slant, y + py }
	for _, s := range segs {
		col := off
		on := cl.bits&(1<<s.bit) != 0
		if on {
			col = lit
		}
		var path ui.Path
		if s.thin {
			x0, y0 := lean(s.x0, s.y0)
			x1, y1 := lean(s.x1, s.y1)
			path.MoveTo(x0, y0).LineTo(x1, y1)
			if on && look != LCD {
				p.StrokePath(&path, th*1.8, lit.Alpha(0.18))
			}
			p.StrokePath(path.Cap(ui.RoundCap), th*0.9, col)
			continue
		}
		bar(&path, s, th, lean)
		if on && look != LCD {
			p.StrokePath(&path, th*0.7, lit.Alpha(0.18))
		}
		p.FillPath(&path, col)
	}
	dx, dy := lean(segs[1].x0+th*1.1, h-th/2)
	col := off
	if cl.dot {
		col = lit
	}
	p.Fill(ui.Rect{X: dx - th*0.55, Y: dy - th*0.55, W: th * 1.1, H: th * 1.1}, col, th*0.55)
}

// bar traces a segment as a hexagon, pointed at both ends, a little short
// of the points it joins.
func bar(path *ui.Path, s seg, th float32, lean func(x, y float32) (float32, float32)) {
	gap := th * 0.12
	ht := th / 2
	pts := [][2]float32{}
	if s.y0 == s.y1 {
		a, b, y := s.x0+gap, s.x1-gap, s.y0
		pts = [][2]float32{{a, y}, {a + ht, y - ht}, {b - ht, y - ht}, {b, y}, {b - ht, y + ht}, {a + ht, y + ht}}
	} else {
		a, b, x := s.y0+gap, s.y1-gap, s.x0
		pts = [][2]float32{{x, a}, {x + ht, a + ht}, {x + ht, b - ht}, {x, b}, {x - ht, b - ht}, {x - ht, a + ht}}
	}
	for i, pt := range pts {
		px, py := lean(pt[0], pt[1])
		if i == 0 {
			path.MoveTo(px, py)
		} else {
			path.LineTo(px, py)
		}
	}
	path.Close()
}

func colon(p *ui.Painter, x, y, w, h float32, lit ui.Color, look Look) {
	d := h * 0.11
	for _, fy := range []float32{0.3, 0.7} {
		cx := x + w/2 + (h-h*fy)*slant - w*0.15
		cy := y + h*fy
		r := ui.Rect{X: cx - d/2, Y: cy - d/2, W: d, H: d}
		if look != LCD {
			p.Shadow(r, d/2, 0, 0, d, 0, lit.Alpha(0.3))
		}
		p.Fill(r, lit, d/2)
	}
}

// matrix draws n cells of dots from x, y, a column every pitch, the text
// moved left by shift columns.
func matrix(p *ui.Painter, x, y, pitch float32, n int, cells []cell, shift int, lit, off ui.Color) {
	cols := n * 6
	total := (len(cells) + 1) * 6
	d := pitch * 0.72
	for col := range cols {
		src := col
		if len(cells) > n {
			src = (col + shift) % total
		}
		var bits byte
		if k := src / 6; src%6 < 5 && k < len(cells) {
			bits = cells[k].cols[src%6]
		}
		for row := range 7 {
			c := off
			if bits&(1<<row) != 0 {
				c = lit
			}
			px, py := x+float32(col)*pitch+(pitch-d)/2, y+float32(row)*pitch+(pitch-d)/2
			p.Fill(ui.Rect{X: px, Y: py, W: d, H: d}, c, d/2)
		}
	}
}

// seven returns the segments of r, a to g as bits 0 to 6.
func seven(r rune) uint16 {
	switch r {
	case '0', 'O', 'D':
		return 0x3F
	case '1', 'I', 'l':
		return 0x06
	case '2', 'Z', 'z':
		return 0x5B
	case '3':
		return 0x4F
	case '4':
		return 0x66
	case '5', 'S', 's':
		return 0x6D
	case '6':
		return 0x7D
	case '7':
		return 0x07
	case '8', 'B':
		return 0x7F
	case '9', 'g':
		return 0x6F
	case 'A', 'a', 'R':
		return 0x77
	case 'b':
		return 0x7C
	case 'C':
		return 0x39
	case 'c':
		return 0x58
	case 'd':
		return 0x5E
	case 'E', 'e':
		return 0x79
	case 'F', 'f':
		return 0x71
	case 'G':
		return 0x3D
	case 'H':
		return 0x76
	case 'h':
		return 0x74
	case 'i':
		return 0x10
	case 'J', 'j':
		return 0x1E
	case 'L':
		return 0x38
	case 'N', 'n':
		return 0x54
	case 'o':
		return 0x5C
	case 'P', 'p':
		return 0x73
	case 'q':
		return 0x67
	case 'r':
		return 0x50
	case 'T', 't':
		return 0x78
	case 'U', 'V':
		return 0x3E
	case 'u', 'v':
		return 0x1C
	case 'Y', 'y':
		return 0x6E
	case '-':
		return 0x40
	case '_':
		return 0x08
	case '=':
		return 0x48
	case '°':
		return 0x63
	}
	return 0
}

// fourteen are the segments of characters on fourteen: a to f as bits 0
// to 5, the middle bars 6 and 7, then the diagonals and the middle
// verticals from top left to bottom right, 8 to 13.
var fourteen = map[rune]uint16{
	'0': 0x0C3F, '1': 0x0006, '2': 0x00DB, '3': 0x008F, '4': 0x00E6, '5': 0x2069, '6': 0x00FD, '7': 0x0007,
	'8': 0x00FF, '9': 0x00EF, 'A': 0x00F7, 'B': 0x128F, 'C': 0x0039, 'D': 0x120F, 'E': 0x00F9, 'F': 0x0071,
	'G': 0x00BD, 'H': 0x00F6, 'I': 0x1209, 'J': 0x001E, 'K': 0x2470, 'L': 0x0038, 'M': 0x0536, 'N': 0x2136,
	'O': 0x003F, 'P': 0x00F3, 'Q': 0x203F, 'R': 0x20F3, 'S': 0x00ED, 'T': 0x1201, 'U': 0x003E, 'V': 0x0C30,
	'W': 0x2836, 'X': 0x2D00, 'Y': 0x1500, 'Z': 0x0C09, '-': 0x00C0, '+': 0x12C0, '*': 0x3FC0, '/': 0x0C00,
	'\\': 0x2100, '_': 0x0008, '=': 0x00C8, '\'': 0x0200, '"': 0x0202, '<': 0x2400, '>': 0x0900, '%': 0x0C24,
	'$': 0x12ED, '!': 0x0006, '?': 0x1083, '(': 0x2400, ')': 0x0900, '[': 0x0039, ']': 0x000F, '°': 0x00E3,
}

// font is a five by seven font of the printable ASCII characters from the
// space, a byte for each column, its lowest bit the top row.
var font = [][5]byte{
	{0x00, 0x00, 0x00, 0x00, 0x00}, {0x00, 0x00, 0x5F, 0x00, 0x00}, {0x00, 0x07, 0x00, 0x07, 0x00}, {0x14, 0x7F, 0x14, 0x7F, 0x14},
	{0x24, 0x2A, 0x7F, 0x2A, 0x12}, {0x23, 0x13, 0x08, 0x64, 0x62}, {0x36, 0x49, 0x55, 0x22, 0x50}, {0x00, 0x05, 0x03, 0x00, 0x00},
	{0x00, 0x1C, 0x22, 0x41, 0x00}, {0x00, 0x41, 0x22, 0x1C, 0x00}, {0x14, 0x08, 0x3E, 0x08, 0x14}, {0x08, 0x08, 0x3E, 0x08, 0x08},
	{0x00, 0x50, 0x30, 0x00, 0x00}, {0x08, 0x08, 0x08, 0x08, 0x08}, {0x00, 0x60, 0x60, 0x00, 0x00}, {0x20, 0x10, 0x08, 0x04, 0x02},
	{0x3E, 0x51, 0x49, 0x45, 0x3E}, {0x00, 0x42, 0x7F, 0x40, 0x00}, {0x42, 0x61, 0x51, 0x49, 0x46}, {0x21, 0x41, 0x45, 0x4B, 0x31},
	{0x18, 0x14, 0x12, 0x7F, 0x10}, {0x27, 0x45, 0x45, 0x45, 0x39}, {0x3C, 0x4A, 0x49, 0x49, 0x30}, {0x01, 0x71, 0x09, 0x05, 0x03},
	{0x36, 0x49, 0x49, 0x49, 0x36}, {0x06, 0x49, 0x49, 0x29, 0x1E}, {0x00, 0x36, 0x36, 0x00, 0x00}, {0x00, 0x56, 0x36, 0x00, 0x00},
	{0x08, 0x14, 0x22, 0x41, 0x00}, {0x14, 0x14, 0x14, 0x14, 0x14}, {0x00, 0x41, 0x22, 0x14, 0x08}, {0x02, 0x01, 0x51, 0x09, 0x06},
	{0x32, 0x49, 0x79, 0x41, 0x3E}, {0x7E, 0x11, 0x11, 0x11, 0x7E}, {0x7F, 0x49, 0x49, 0x49, 0x36}, {0x3E, 0x41, 0x41, 0x41, 0x22},
	{0x7F, 0x41, 0x41, 0x22, 0x1C}, {0x7F, 0x49, 0x49, 0x49, 0x41}, {0x7F, 0x09, 0x09, 0x09, 0x01}, {0x3E, 0x41, 0x49, 0x49, 0x7A},
	{0x7F, 0x08, 0x08, 0x08, 0x7F}, {0x00, 0x41, 0x7F, 0x41, 0x00}, {0x20, 0x40, 0x41, 0x3F, 0x01}, {0x7F, 0x08, 0x14, 0x22, 0x41},
	{0x7F, 0x40, 0x40, 0x40, 0x40}, {0x7F, 0x02, 0x0C, 0x02, 0x7F}, {0x7F, 0x04, 0x08, 0x10, 0x7F}, {0x3E, 0x41, 0x41, 0x41, 0x3E},
	{0x7F, 0x09, 0x09, 0x09, 0x06}, {0x3E, 0x41, 0x51, 0x21, 0x5E}, {0x7F, 0x09, 0x19, 0x29, 0x46}, {0x46, 0x49, 0x49, 0x49, 0x31},
	{0x01, 0x01, 0x7F, 0x01, 0x01}, {0x3F, 0x40, 0x40, 0x40, 0x3F}, {0x1F, 0x20, 0x40, 0x20, 0x1F}, {0x3F, 0x40, 0x38, 0x40, 0x3F},
	{0x63, 0x14, 0x08, 0x14, 0x63}, {0x07, 0x08, 0x70, 0x08, 0x07}, {0x61, 0x51, 0x49, 0x45, 0x43}, {0x00, 0x7F, 0x41, 0x41, 0x00},
	{0x02, 0x04, 0x08, 0x10, 0x20}, {0x00, 0x41, 0x41, 0x7F, 0x00}, {0x04, 0x02, 0x01, 0x02, 0x04}, {0x40, 0x40, 0x40, 0x40, 0x40},
	{0x00, 0x01, 0x02, 0x04, 0x00}, {0x20, 0x54, 0x54, 0x54, 0x78}, {0x7F, 0x48, 0x44, 0x44, 0x38}, {0x38, 0x44, 0x44, 0x44, 0x20},
	{0x38, 0x44, 0x44, 0x48, 0x7F}, {0x38, 0x54, 0x54, 0x54, 0x18}, {0x08, 0x7E, 0x09, 0x01, 0x02}, {0x0C, 0x52, 0x52, 0x52, 0x3E},
	{0x7F, 0x08, 0x04, 0x04, 0x78}, {0x00, 0x44, 0x7D, 0x40, 0x00}, {0x20, 0x40, 0x44, 0x3D, 0x00}, {0x7F, 0x10, 0x28, 0x44, 0x00},
	{0x00, 0x41, 0x7F, 0x40, 0x00}, {0x7C, 0x04, 0x18, 0x04, 0x78}, {0x7C, 0x08, 0x04, 0x04, 0x78}, {0x38, 0x44, 0x44, 0x44, 0x38},
	{0x7C, 0x14, 0x14, 0x14, 0x08}, {0x08, 0x14, 0x14, 0x18, 0x7C}, {0x7C, 0x08, 0x04, 0x04, 0x08}, {0x48, 0x54, 0x54, 0x54, 0x20},
	{0x04, 0x3F, 0x44, 0x40, 0x20}, {0x3C, 0x40, 0x40, 0x20, 0x7C}, {0x1C, 0x20, 0x40, 0x20, 0x1C}, {0x3C, 0x40, 0x30, 0x40, 0x3C},
	{0x44, 0x28, 0x10, 0x28, 0x44}, {0x0C, 0x50, 0x50, 0x50, 0x3C}, {0x44, 0x64, 0x54, 0x4C, 0x44}, {0x00, 0x08, 0x36, 0x41, 0x00},
	{0x00, 0x00, 0x7F, 0x00, 0x00}, {0x00, 0x41, 0x36, 0x08, 0x00}, {0x08, 0x04, 0x08, 0x10, 0x08},
}
