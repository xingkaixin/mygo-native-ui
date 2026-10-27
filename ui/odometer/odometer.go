// Package odometer shows a number whose digits roll as it changes, as a
// count of followers or a price ticking: each digit turns on its wheel to
// its new value, upward as the number grows and downward as it shrinks,
// the ones turning a whole round more when the number jumps by tens or
// more. Digits and separators come and go at the left as the number
// gains and loses places.
//
//	odometer.View(c, strconv.Itoa(app.followers)).FontSize(40).Bold()
//
// The text may hold other characters besides digits, as a currency sign,
// separators and a decimal point; they stay as they are. Its digits take
// tabular figures, so that they keep their width as they turn. The font,
// its size and color set on the view go to the digits.
package odometer

import (
	"math"
	"math/big"
	"strings"

	"github.com/egoist/mygo/ui"
)

// change makes characters that come and go grow from nothing and fade.
var change = ui.ElementTransition{Enter: &ui.Motion{Collapse: true}, Exit: &ui.Motion{Collapse: true}}

// slot is a character's place, counted from the right: whether it holds a
// digit, and where its wheel turns to, in digits, counted on past 9 and
// below 0.
type slot struct {
	digit bool
	at    float32
}

// state is what a view keeps from frame to frame: the text it showed and
// its places.
type state struct {
	text  string
	slots []slot
}

// View creates a view of text, rolling its digits from the text it
// showed in the frame before.
func View(c *ui.Context, text string) ui.Element {
	row := ui.Row(c).FontFeatures("tnum").Label(text)
	st := ui.Local(row, "odometer.state", func() state { return state{} })
	runes := []rune(text)
	n := len(runes)
	if text != st.text {
		dir, spins := trend(st.text, text)
		slots := make([]slot, n)
		for k := range slots {
			r := runes[n-1-k]
			if r < '0' || r > '9' {
				continue
			}
			to := int(r - '0')
			if k >= len(st.slots) || !st.slots[k].digit || st.text == "" {
				slots[k] = slot{true, float32(to)}
				continue
			}
			at := st.slots[k].at
			from := mod(int(math.Round(float64(at))), 10)
			turn := 0
			if k == 0 {
				turn = spins
			}
			switch {
			case dir > 0:
				at += float32(mod(to-from, 10) + 10*turn)
			case dir < 0:
				at -= float32(mod(from-to, 10) + 10*turn)
			default:
				at += float32(to - from)
			}
			slots[k] = slot{true, at}
		}
		st.text, st.slots = text, slots
	}

	row.Children(func() {
		for i, r := range runes {
			k := n - 1 - i
			if !st.slots[k].digit {
				ui.Text(c.Key(k), string(r)).Transition(change)
				continue
			}
			cell := ui.Box(c.Key(k)).Clip().Transition(change)
			pos := cell.AnimateTo("odometer.roll", st.slots[k].at, ui.Smooth)
			base := math.Floor(float64(pos))
			frac := pos - float32(base)
			h := cell.LayoutBounds().H
			cell.Children(func() {
				ui.Text(c, digit(base)).Translate(0, -frac*h)
				if frac > 0.001 {
					ui.Text(c, digit(base+1)).Absolute().Left(0).Top(0).Translate(0, (1-frac)*h)
				}
			})
		}
	})
	return row
}

// trend returns whether the digits of b make a larger number than those
// of a, 1, a smaller one, -1, or the same, 0; and whether they are tens
// or more apart, 1, so that the ones turn a whole round more.
func trend(a, b string) (dir, spins int) {
	x, y := number(a), number(b)
	d := new(big.Int).Sub(y, x)
	dir = d.Sign()
	if d.Abs(d).Cmp(big.NewInt(10)) >= 0 {
		spins = 1
	}
	return dir, spins
}

// number reads the digits of s, all others aside, as one number.
func number(s string) *big.Int {
	var b strings.Builder
	b.WriteByte('0')
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	v, _ := new(big.Int).SetString(b.String(), 10)
	return v
}

func digit(at float64) string { return string(rune('0' + mod(int(at), 10))) }

func mod(a, n int) int { return (a%n + n) % n }
