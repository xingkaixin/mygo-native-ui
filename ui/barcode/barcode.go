// Package barcode makes linear barcodes and shows them, as shipping
// labels, tickets and products carry them: Code 128 for any text of
// printable ASCII, and EAN-13 for the numbers on products. View draws a
// code black on white with the quiet zones scanners need, in either
// theme, and its text under it: an EAN-13's digits in its three groups,
// between the guard bars that reach down past the others.
//
//	code, err := barcode.Code128("SHIP-2041-77")
//	...
//	barcode.View(c, code).Size(280, 90)
package barcode

import (
	"errors"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Code is a barcode: its bars, a module each, and the text it holds.
type Code struct {
	// Text is what the code holds, as shown under it: an EAN-13's with its
	// check digit.
	Text    string
	modules []bool
	guards  []bool
	ean     bool
}

// Modules returns how many modules wide the code is, without its quiet
// zones.
func (b *Code) Modules() int { return len(b.modules) }

// Dark reports whether the i-th module is a bar.
func (b *Code) Dark(i int) bool { return b.modules[i] }

// ErrText is the error of text a kind of code cannot hold.
var ErrText = errors.New("barcode: text not encodable")

// ErrCheck is the error of an EAN-13 whose check digit is wrong.
var ErrCheck = errors.New("barcode: wrong check digit")

// patterns are the widths of Code 128's bars and spaces, by value, bar
// first; the last is the stop pattern.
var patterns = [107]string{
	"212222", "222122", "222221", "121223", "121322", "131222", "122213", "122312", "132212", "221213",
	"221312", "231212", "112232", "122132", "122231", "113222", "123122", "123221", "223211", "221132",
	"221231", "213212", "223112", "312131", "311222", "321122", "321221", "312212", "322112", "322211",
	"212123", "212321", "232121", "111323", "131123", "131321", "112313", "132113", "132311", "211313",
	"231113", "231311", "112133", "112331", "132131", "113123", "113321", "133121", "313121", "211331",
	"231131", "213113", "213311", "213131", "311123", "311321", "331121", "312113", "312311", "332111",
	"314111", "221411", "431111", "111224", "111422", "121124", "121421", "141122", "141221", "112214",
	"112412", "122114", "122411", "142112", "142211", "241211", "221114", "413111", "241112", "134111",
	"111242", "121142", "121241", "114212", "124112", "124211", "411212", "421112", "421211", "212141",
	"214121", "412121", "111143", "111341", "131141", "114113", "114311", "411113", "411311", "113141",
	"114131", "311141", "411131", "211412", "211214", "211232", "2331112",
}

// Code 128's values that switch subsets and start and stop a code.
const (
	codeC  = 99
	codeB  = 100
	startB = 104
	startC = 105
	stop   = 106
)

// Code128 returns the Code 128 code of text, printable ASCII, the
// shortest of subsets B and C: runs of digits two to a symbol in C.
func Code128(text string) (*Code, error) {
	if text == "" {
		return nil, ErrText
	}
	for i := range len(text) {
		if text[i] < 32 || text[i] > 126 {
			return nil, ErrText
		}
	}
	digits := func(i int) int {
		n := 0
		for i+n < len(text) && text[i+n] >= '0' && text[i+n] <= '9' {
			n++
		}
		return n
	}
	var vals []int
	c := false
	if d := digits(0); d >= 4 || d == len(text) && d >= 2 {
		vals, c = append(vals, startC), true
	} else {
		vals = append(vals, startB)
	}
	for i := 0; i < len(text); {
		if c {
			if digits(i) >= 2 {
				vals = append(vals, int(text[i]-'0')*10+int(text[i+1]-'0'))
				i += 2
				continue
			}
			vals, c = append(vals, codeB), false
		}
		// A run long enough to save symbols goes in C, an odd one after
		// its first digit in B.
		if d := digits(i); d >= 6 || d >= 4 && i+d == len(text) {
			if d%2 == 1 {
				vals = append(vals, int(text[i])-32)
				i++
			}
			vals, c = append(vals, codeC), true
			continue
		}
		vals = append(vals, int(text[i])-32)
		i++
	}
	sum := vals[0]
	for i, v := range vals[1:] {
		sum += (i + 1) * v
	}
	vals = append(vals, sum%103, stop)
	code := &Code{Text: text}
	for _, v := range vals {
		bar := true
		for _, w := range patterns[v] {
			for range w - '0' {
				code.modules = append(code.modules, bar)
			}
			bar = !bar
		}
	}
	return code, nil
}

// The digits of EAN-13: L and G code the left half, by the parity the
// first digit gives, and R, L's complement, the right.
var (
	eanL   = [10]string{"0001101", "0011001", "0010011", "0111101", "0100011", "0110001", "0101111", "0111011", "0110111", "0001011"}
	eanG   = [10]string{"0100111", "0110011", "0011011", "0100001", "0011101", "0111001", "0000101", "0010001", "0001001", "0010111"}
	parity = [10]string{"LLLLLL", "LLGLGG", "LLGGLG", "LLGGGL", "LGLLGG", "LGGLLG", "LGGGLL", "LGLGLG", "LGLGGL", "LGGLGL"}
)

// EAN13 returns the EAN-13 code of digits: twelve, to which it adds the
// check digit, or thirteen, whose last it checks.
func EAN13(digits string) (*Code, error) {
	if len(digits) != 12 && len(digits) != 13 || strings.Trim(digits, "0123456789") != "" {
		return nil, ErrText
	}
	sum := 0
	for i := range 12 {
		w := 1
		if i%2 == 1 {
			w = 3
		}
		sum += int(digits[i]-'0') * w
	}
	check := byte('0' + (10-sum%10)%10)
	if len(digits) == 13 && digits[12] != check {
		return nil, ErrCheck
	}
	digits = digits[:12] + string(check)
	code := &Code{Text: digits, ean: true}
	put := func(bits string, guard bool) {
		for _, b := range bits {
			code.modules = append(code.modules, b == '1')
			code.guards = append(code.guards, guard)
		}
	}
	put("101", true)
	for i, p := range parity[digits[0]-'0'] {
		d := digits[1+i] - '0'
		if p == 'L' {
			put(eanL[d], false)
		} else {
			put(eanG[d], false)
		}
	}
	put("01010", true)
	for i := 7; i < 13; i++ {
		r := []byte(eanL[digits[i]-'0'])
		for j := range r {
			r[j] ^= 1
		}
		put(string(r), false)
	}
	put("101", true)
	return code, nil
}

// View creates the picture of code, black on white with its quiet zones
// and its text under it, in either theme. Give it a size; its modules are
// as wide as fit, and its bars as tall.
func View(c *ui.Context, code *Code) ui.Element {
	e := ui.Box(c).Role(ui.RoleImage)
	if code == nil {
		return e
	}
	e.Label("Barcode").Value(code.Text)
	return e.Draw(func(p *ui.Painter, r ui.Rect) {
		// Quiet zones, in modules: 10 each side for Code 128, and for
		// EAN-13 11 on the left, where the first digit shows, and 7 on the
		// right.
		left, right := 10, 10
		if code.ean {
			left, right = 11, 7
		}
		n := len(code.modules) + left + right
		m := r.W / float32(n)
		size := min(m*8, r.H/4)
		x0 := r.X + (r.W-m*float32(n))/2
		p.Fill(r, ui.RGB(255, 255, 255), 2)
		black := ui.RGB(0, 0, 0)
		top := r.Y + m*4
		bottom := r.Y + r.H - size - m*4
		for i := 0; i < len(code.modules); i++ {
			if !code.modules[i] {
				continue
			}
			start := i
			for i+1 < len(code.modules) && code.modules[i+1] {
				i++
			}
			h := bottom - top
			if code.ean && code.guards[start] {
				h += size / 2
			}
			p.Fill(ui.Rect{X: x0 + float32(left+start)*m, Y: top, W: float32(i-start+1) * m, H: h}, black, 0)
		}
		digit := func(s string, mid float32) {
			span := ui.Span{Text: s, Font: "monospace", Size: size, Color: black}
			w, _ := p.MeasureText(0, span)
			p.RichText(x0+mid*m-w/2, bottom+m, 0, span)
		}
		if !code.ean {
			digit(code.Text, float32(left)+float32(len(code.modules))/2)
			return
		}
		// Each digit sits under its own seven modules, the first left of
		// the bars.
		at := float32(left)
		digit(code.Text[:1], at-4)
		for i := range 6 {
			digit(code.Text[1+i:2+i], at+3+float32(i)*7+3.5)
			digit(code.Text[7+i:8+i], at+50+float32(i)*7+3.5)
		}
	})
}
