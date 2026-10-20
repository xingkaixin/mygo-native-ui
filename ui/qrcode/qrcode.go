// Package qrcode makes QR codes and shows them, as apps do to share a
// link or a Wi‑Fi network with a phone: Encode writes text in the
// smallest code that holds it at a level of error correction, and View
// draws a code dark on light, with the margin around it that cameras
// need, in either theme.
//
//	code, err := qrcode.Encode("https://example.com", qrcode.M)
//	...
//	qrcode.View(c, code).Size(200, 200)
package qrcode

import (
	"errors"

	"github.com/egoist/mygo/ui"
)

// Level is how much of a code may be lost and still read: about 7% for L,
// 15% for M, 25% for Q and 30% for H, in a bigger code for more.
type Level uint8

const (
	L Level = iota
	M
	Q
	H
)

// Code is a QR code: Size modules a side, of version (Size-17)/4.
type Code struct {
	Size    int
	modules []bool
}

// Dark reports whether the module at column x and row y is dark.
func (q *Code) Dark(x, y int) bool { return q.modules[y*q.Size+x] }

// Version returns the code's version, from 1 to 40.
func (q *Code) Version() int { return (q.Size - 17) / 4 }

// ErrTooLong is the error of text that no code holds at its level.
var ErrTooLong = errors.New("qrcode: text too long")

// eccPerBlock and blocks are ISO/IEC 18004's table 9, by level and
// version.
var eccPerBlock = [4][41]int{
	{0, 7, 10, 15, 20, 26, 18, 20, 24, 30, 18, 20, 24, 26, 30, 22, 24, 28, 30, 28, 28, 28, 28, 30, 30, 26, 28, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30},
	{0, 10, 16, 26, 18, 24, 16, 18, 22, 22, 26, 30, 22, 22, 24, 24, 28, 28, 26, 26, 26, 26, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28, 28},
	{0, 13, 22, 18, 26, 18, 24, 18, 22, 20, 24, 28, 26, 24, 20, 30, 24, 28, 28, 26, 30, 28, 30, 30, 30, 30, 28, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30},
	{0, 17, 28, 22, 16, 22, 28, 26, 26, 24, 28, 24, 28, 22, 24, 24, 30, 28, 28, 26, 28, 30, 24, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30, 30},
}

var blocks = [4][41]int{
	{0, 1, 1, 1, 1, 1, 2, 2, 2, 2, 4, 4, 4, 4, 4, 6, 6, 6, 6, 7, 8, 8, 9, 9, 10, 12, 12, 12, 13, 14, 15, 16, 17, 18, 19, 19, 20, 21, 22, 24, 25},
	{0, 1, 1, 1, 2, 2, 4, 4, 4, 5, 5, 5, 8, 9, 9, 10, 10, 11, 13, 14, 16, 17, 17, 18, 20, 21, 23, 25, 26, 28, 29, 31, 33, 35, 37, 38, 40, 43, 45, 47, 49},
	{0, 1, 1, 2, 2, 4, 4, 6, 6, 8, 8, 8, 10, 12, 16, 12, 17, 16, 18, 21, 20, 23, 23, 25, 27, 29, 34, 34, 35, 38, 40, 43, 45, 48, 51, 53, 56, 59, 62, 65, 68},
	{0, 1, 1, 2, 4, 4, 4, 5, 6, 8, 8, 11, 11, 16, 16, 18, 16, 19, 21, 25, 25, 25, 34, 30, 32, 35, 37, 40, 42, 45, 48, 51, 54, 57, 60, 63, 66, 70, 74, 77, 81},
}

// rawModules returns how many modules of a code of version v hold data
// and error correction, those that no pattern takes.
func rawModules(v int) int {
	n := (16*v+128)*v + 64
	if v >= 2 {
		align := v/7 + 2
		n -= (25*align-10)*align - 55
		if v >= 7 {
			n -= 36
		}
	}
	return n
}

func dataCodewords(v int, l Level) int {
	return rawModules(v)/8 - eccPerBlock[l][v]*blocks[l][v]
}

// Encode returns the smallest code holding text, in byte mode, at level.
func Encode(text string, level Level) (*Code, error) {
	data := []byte(text)
	v := 1
	for ; v <= 40; v++ {
		count := 8
		if v >= 10 {
			count = 16
		}
		if len(data) < 1<<count && 4+count+8*len(data) <= dataCodewords(v, level)*8 {
			break
		}
	}
	if v > 40 {
		return nil, ErrTooLong
	}
	var bits bitBuffer
	count := 8
	if v >= 10 {
		count = 16
	}
	bits.add(0b0100, 4)
	bits.add(len(data), count)
	for _, b := range data {
		bits.add(int(b), 8)
	}
	capacity := dataCodewords(v, level) * 8
	bits.add(0, min(4, capacity-len(bits)))
	bits.add(0, (8-len(bits)%8)%8)
	for pad := 0xEC; len(bits) < capacity; pad ^= 0xEC ^ 0x11 {
		bits.add(pad, 8)
	}
	codewords := make([]byte, len(bits)/8)
	for i, b := range bits {
		if b {
			codewords[i>>3] |= 1 << (7 - i&7)
		}
	}
	q := newCode(v)
	q.drawPatterns(v)
	q.place(interleave(codewords, v, level))
	best, bestScore := 0, -1
	for mask := range 8 {
		q.mask(mask)
		q.drawFormat(level, mask)
		if s := q.penalty(); bestScore < 0 || s < bestScore {
			best, bestScore = mask, s
		}
		q.mask(mask)
	}
	q.mask(best)
	q.drawFormat(level, best)
	return &Code{Size: q.size, modules: q.dark}, nil
}

type bitBuffer []bool

func (b *bitBuffer) add(v, n int) {
	for i := n - 1; i >= 0; i-- {
		*b = append(*b, v>>i&1 != 0)
	}
}

// interleave splits data into the blocks of version v at level, adds each
// block's error correction, and takes a codeword of each block in turn.
func interleave(data []byte, v int, level Level) []byte {
	nb, ecc := blocks[level][v], eccPerBlock[level][v]
	raw := rawModules(v) / 8
	short := nb - raw%nb
	shortLen := raw / nb
	div := divisor(ecc)
	var bs [][]byte
	k := 0
	for i := range nb {
		n := shortLen - ecc
		if i >= short {
			n++
		}
		d := append([]byte(nil), data[k:k+n]...)
		k += n
		e := remainder(d, div)
		if i < short {
			// A placeholder, so that the codewords of all blocks line up.
			d = append(d, 0)
		}
		bs = append(bs, append(d, e...))
	}
	var out []byte
	for i := range bs[0] {
		for j, b := range bs {
			if i != shortLen-ecc || j >= short {
				out = append(out, b[i])
			}
		}
	}
	return out
}

// mul multiplies in GF(2⁸) modulo x⁸+x⁴+x³+x²+1.
func mul(x, y byte) byte {
	var z int
	for i := 7; i >= 0; i-- {
		z = z<<1 ^ (z>>7)*0x11D
		z ^= int(y>>i&1) * int(x)
	}
	return byte(z)
}

// divisor returns the generator polynomial of degree n, its leading term
// left out.
func divisor(n int) []byte {
	d := make([]byte, n)
	d[n-1] = 1
	root := byte(1)
	for range n {
		for j := range d {
			d[j] = mul(d[j], root)
			if j+1 < n {
				d[j] ^= d[j+1]
			}
		}
		root = mul(root, 2)
	}
	return d
}

func remainder(data, div []byte) []byte {
	r := make([]byte, len(div))
	for _, b := range data {
		f := b ^ r[0]
		copy(r, r[1:])
		r[len(r)-1] = 0
		for i := range r {
			r[i] ^= mul(div[i], f)
		}
	}
	return r
}

type code struct {
	size       int
	dark, used []bool
}

func newCode(v int) *code {
	n := v*4 + 17
	return &code{size: n, dark: make([]bool, n*n), used: make([]bool, n*n)}
}

func (q *code) set(x, y int, dark bool) {
	q.dark[y*q.size+x] = dark
	q.used[y*q.size+x] = true
}

func (q *code) drawPatterns(v int) {
	n := q.size
	for i := range n {
		q.set(6, i, i%2 == 0)
		q.set(i, 6, i%2 == 0)
	}
	finder := func(cx, cy int) {
		for dy := -4; dy <= 4; dy++ {
			for dx := -4; dx <= 4; dx++ {
				x, y := cx+dx, cy+dy
				if x >= 0 && x < n && y >= 0 && y < n {
					d := max(abs(dx), abs(dy))
					q.set(x, y, d != 2 && d != 4)
				}
			}
		}
	}
	finder(3, 3)
	finder(n-4, 3)
	finder(3, n-4)
	pos := alignment(v, n)
	for i, x := range pos {
		for j, y := range pos {
			if i == 0 && j == 0 || i == 0 && j == len(pos)-1 || i == len(pos)-1 && j == 0 {
				continue
			}
			for dy := -2; dy <= 2; dy++ {
				for dx := -2; dx <= 2; dx++ {
					q.set(x+dx, y+dy, max(abs(dx), abs(dy)) != 1)
				}
			}
		}
	}
	// The format's modules, drawn for each mask.
	q.drawFormat(0, 0)
	if v >= 7 {
		rem := v
		for range 12 {
			rem = rem<<1 ^ (rem>>11)*0x1F25
		}
		bits := v<<12 | rem
		for i := range 18 {
			bit := bits>>i&1 != 0
			a, b := n-11+i%3, i/3
			q.set(a, b, bit)
			q.set(b, a, bit)
		}
	}
}

// alignment returns the rows and columns of version v's alignment
// patterns.
func alignment(v, n int) []int {
	if v == 1 {
		return nil
	}
	k := v/7 + 2
	step := (v*8 + k*3 + 5) / (k*4 - 4) * 2
	pos := make([]int, k)
	pos[0] = 6
	for i, p := k-1, n-7; i >= 1; i, p = i-1, p-step {
		pos[i] = p
	}
	return pos
}

func (q *code) drawFormat(level Level, mask int) {
	n := q.size
	data := int([4]int{1, 0, 3, 2}[level])<<3 | mask
	rem := data
	for range 10 {
		rem = rem<<1 ^ (rem>>9)*0x537
	}
	bits := (data<<10 | rem) ^ 0x5412
	bit := func(i int) bool { return bits>>i&1 != 0 }
	for i := range 6 {
		q.set(8, i, bit(i))
	}
	q.set(8, 7, bit(6))
	q.set(8, 8, bit(7))
	q.set(7, 8, bit(8))
	for i := 9; i < 15; i++ {
		q.set(14-i, 8, bit(i))
	}
	for i := range 8 {
		q.set(n-1-i, 8, bit(i))
	}
	for i := 8; i < 15; i++ {
		q.set(8, n-15+i, bit(i))
	}
	q.set(8, n-8, true)
}

// place puts data in the modules no pattern takes, in two columns going
// up and down from the bottom right.
func (q *code) place(data []byte) {
	n := q.size
	i := 0
	for right := n - 1; right >= 1; right -= 2 {
		if right == 6 {
			right = 5
		}
		for vert := range n {
			for j := range 2 {
				x := right - j
				y := vert
				if (right+1)&2 == 0 {
					y = n - 1 - vert
				}
				if !q.used[y*n+x] && i < len(data)*8 {
					q.dark[y*n+x] = data[i>>3]>>(7-i&7)&1 != 0
					i++
				}
			}
		}
	}
}

// mask flips the data modules by mask k, which a second call undoes.
func (q *code) mask(k int) {
	n := q.size
	for y := range n {
		for x := range n {
			var flip bool
			switch k {
			case 0:
				flip = (x+y)%2 == 0
			case 1:
				flip = y%2 == 0
			case 2:
				flip = x%3 == 0
			case 3:
				flip = (x+y)%3 == 0
			case 4:
				flip = (x/3+y/2)%2 == 0
			case 5:
				flip = x*y%2+x*y%3 == 0
			case 6:
				flip = (x*y%2+x*y%3)%2 == 0
			case 7:
				flip = ((x+y)%2+x*y%3)%2 == 0
			}
			if flip && !q.used[y*n+x] {
				q.dark[y*n+x] = !q.dark[y*n+x]
			}
		}
	}
}

// penalty scores how hard the code is to read, as ISO/IEC 18004 does to
// choose a mask: long runs, blocks of one color, patterns like the
// finders', and more of one color than the other.
func (q *code) penalty() int {
	n := q.size
	at := func(x, y int, rows bool) bool {
		if rows {
			return q.dark[y*n+x]
		}
		return q.dark[x*n+y]
	}
	score := 0
	finder := []bool{true, false, true, true, true, false, true}
	for _, rows := range []bool{true, false} {
		for y := range n {
			run := 1
			for x := 1; x < n; x++ {
				if at(x, y, rows) == at(x-1, y, rows) {
					run++
					if run == 5 {
						score += 3
					} else if run > 5 {
						score++
					}
				} else {
					run = 1
				}
			}
			for x := 0; x+7 <= n; x++ {
				match := true
				for k, d := range finder {
					if at(x+k, y, rows) != d {
						match = false
						break
					}
				}
				if !match {
					continue
				}
				light := func(from, to int) bool {
					for k := from; k < to; k++ {
						if k >= 0 && k < n && at(k, y, rows) {
							return false
						}
					}
					return true
				}
				if light(x-4, x) || light(x+7, x+11) {
					score += 40
				}
			}
		}
	}
	dark := 0
	for y := range n {
		for x := range n {
			d := q.dark[y*n+x]
			if d {
				dark++
			}
			if x+1 < n && y+1 < n && d == q.dark[y*n+x+1] && d == q.dark[(y+1)*n+x] && d == q.dark[(y+1)*n+x+1] {
				score += 3
			}
		}
	}
	total := n * n
	score += ((abs(dark*20-total*10)+total-1)/total - 1) * 10
	return score
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// View creates the picture of code, black on white with a margin of four
// modules, in either theme, as cameras read best. Give it a size; it
// keeps its modules square in the middle of it.
func View(c *ui.Context, code *Code) ui.Element {
	return ui.Box(c).Role(ui.RoleImage).Draw(func(p *ui.Painter, r ui.Rect) {
		if code == nil {
			return
		}
		side := min(r.W, r.H)
		n := code.Size + 8
		m := side / float32(n)
		x0, y0 := r.X+(r.W-side)/2, r.Y+(r.H-side)/2
		p.Fill(ui.Rect{X: x0, Y: y0, W: side, H: side}, ui.RGB(255, 255, 255), m)
		black := ui.RGB(0, 0, 0)
		for y := range code.Size {
			// A run of dark modules is one rectangle, with no seams between
			// them where m is not a whole number of pixels.
			for x := 0; x < code.Size; x++ {
				if !code.Dark(x, y) {
					continue
				}
				start := x
				for x+1 < code.Size && code.Dark(x+1, y) {
					x++
				}
				p.Fill(ui.Rect{X: x0 + float32(start+4)*m, Y: y0 + float32(y+4)*m, W: float32(x-start+1) * m, H: m + 0.5}, black, 0)
			}
		}
	})
}
