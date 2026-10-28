// Package currency is a field for an amount of money, as checkout and
// invoicing forms have: the currency's symbol before the amount, its
// digits grouped by thousands with as many decimals as the currency has,
// none for yen and three for dinars, and the currency's code after it,
// which opens the list of currencies to choose another.
//
//	currency.Field(c, &app.price, &app.amount, &app.code)
//
// The amount is in the currency's minor unit, cents for dollars. The field
// takes digits and one decimal point, and drops other characters and
// decimals past the currency's, so a paste of "$1,234.50" gives 123450.
// Enter, or the focus leaving, groups the amount again. Choosing another
// currency keeps the amount's figures, losing decimals the new one lacks.
package currency

import (
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Currency is a currency: its ISO 4217 code, its symbol, its name, and the
// decimals of its minor unit.
type Currency struct {
	Code, Symbol, Name string
	Digits             int
}

// Currencies are the currencies the field knows, in the order its list
// shows them.
var Currencies = []Currency{
	{"USD", "$", "US Dollar", 2},
	{"EUR", "€", "Euro", 2},
	{"GBP", "£", "British Pound", 2},
	{"JPY", "¥", "Japanese Yen", 0},
	{"CNY", "¥", "Chinese Yuan", 2},
	{"KRW", "₩", "South Korean Won", 0},
	{"INR", "₹", "Indian Rupee", 2},
	{"CHF", "CHF ", "Swiss Franc", 2},
	{"CAD", "CA$", "Canadian Dollar", 2},
	{"AUD", "A$", "Australian Dollar", 2},
	{"BRL", "R$", "Brazilian Real", 2},
	{"KWD", "KD ", "Kuwaiti Dinar", 3},
}

// lookup returns the currency of code, or a currency of two decimals
// written by its code.
func lookup(code string) Currency {
	for _, c := range Currencies {
		if c.Code == code {
			return c
		}
	}
	return Currency{Code: code, Symbol: code + " ", Name: code, Digits: 2}
}

// Format writes amount, in the minor unit of the currency of code, with
// its symbol: $1,234.50, ¥1,235.
func Format(amount int64, code string) string {
	cur := lookup(code)
	sign := ""
	if amount < 0 {
		sign, amount = "-", -amount
	}
	return sign + cur.Symbol + group(amount, cur.Digits)
}

func group(amount int64, digits int) string {
	s := strconv.FormatInt(amount, 10)
	if len(s) <= digits {
		s = strings.Repeat("0", digits-len(s)+1) + s
	}
	whole, frac := s[:len(s)-digits], s[len(s)-digits:]
	for i := len(whole) - 3; i > 0; i -= 3 {
		whole = whole[:i] + "," + whole[i:]
	}
	if digits == 0 {
		return whole
	}
	return whole + "." + frac
}

// clean keeps the digits of text, its commas, and its first decimal point
// with up to digits figures after it.
func clean(text string, digits int) string {
	var b strings.Builder
	point, after, whole := false, 0, 0
	for _, r := range text {
		switch {
		case r == '.' && !point && digits > 0:
			point = true
		case r == ',' && !point:
		case r >= '0' && r <= '9' && point && after < digits:
			after++
		case r >= '0' && r <= '9' && !point && whole < 13:
			whole++
		default:
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// parse reads text cleaned by clean as an amount in the minor unit.
func parse(text string, digits int) int64 {
	whole, frac, _ := strings.Cut(strings.ReplaceAll(text, ",", ""), ".")
	frac += strings.Repeat("0", digits-len(frac))
	n, _ := strconv.ParseInt(whole+frac, 10, 64)
	return n
}

// State is what a field keeps: the text being typed and whether the list
// of currencies shows. Keep it in the app's state, one for each field.
type State struct {
	text    string
	editing bool
	open    bool
}

// Field creates the field of *amount, in the minor unit of the currency of
// *code.
func Field(c *ui.Context, s *State, amount *int64, code *string) ui.Element {
	t := c.Theme()
	cur := lookup(*code)
	if !s.editing {
		s.text = ""
		if *amount != 0 {
			s.text = group(*amount, cur.Digits)
		}
	}
	field := ui.Row(c).AlignItems(ui.Center).Radius(t.Radius).Background(t.Surface).Border(1, t.Border)
	var in ui.Element
	field.Children(func() {
		ui.Text(c, strings.TrimSpace(cur.Symbol)).TextColor(t.TextMuted).Padding(0, 0, 0, t.Space(2.5))
		in = ui.TextInputBase(c, &s.text).Grow(1).Padding(t.Space(1.5), t.Space(2.5), t.Space(1.5), t.Space(1.5)).
			FontFeatures("tnum").Label("Amount").Placeholder(group(0, cur.Digits))
		ui.Box(c).Width(1).Height(t.Space(4)).Background(t.Border)
		pick := ui.ButtonBase(c).Gap(t.Space(1)).Padding(t.Space(1.5), t.Space(2.5)).AlignItems(ui.Center).Label("Currency").
			Children(func() {
				ui.Text(c, cur.Code).FontWeight(500)
				ui.Text(c, "▾").FontSize(t.FontSize - 3).TextColor(t.TextMuted)
			})
		if pick.Hovered() {
			pick.Background(t.Text.Alpha(0.05))
		}
		if pick.Clicked() {
			s.open = !s.open
		}
		ui.PopoverBase(c, pick, &s.open, func(panel ui.Element) {
			panel.Margin(t.Space(1), 0, 0, 0).Padding(t.Space(1)).Radius(t.Space(2.5)).Width(t.Space(68)).
				Background(t.Background).Border(1, t.Border).Shadow(0, 8, 24, 0, ui.RGBA(0, 0, 0, 0.16)).Label("Currencies")
			ui.Scroll(c).MaxHeight(t.Space(80)).Children(func() {
				for i, o := range Currencies {
					row := ui.ButtonBase(c.Key(i)).Gap(t.Space(2)).Padding(t.Space(1.5), t.Space(2)).Radius(t.Space(1.5)).
						Justify(ui.Start).AlignItems(ui.Center).Label(o.Name)
					on := o.Code == cur.Code
					if row.Hovered() || on {
						row.Background(t.Text.Alpha(0.06))
					}
					if row.Clicked() {
						*amount = rescale(*amount, cur.Digits, o.Digits)
						*code, s.open, s.editing = o.Code, false, false
						c.Invalidate()
					}
					row.Children(func() {
						ui.Text(c, o.Code).FontWeight(600).Width(t.Space(10))
						ui.Text(c, o.Name).Grow(1).SingleLine()
						sym := ui.Text(c, strings.TrimSpace(o.Symbol)).TextColor(t.TextMuted)
						if on {
							sym.TextColor(t.Accent)
						}
					})
				}
			})
		})
	})
	if in.Focused() {
		field.Border(1, t.Accent)
		s.editing = true
	}
	if in.Changed() {
		if kept := clean(s.text, cur.Digits); kept != s.text {
			s.text = kept
			c.Invalidate()
		}
		*amount = parse(s.text, cur.Digits)
	}
	if s.editing && (!in.Focused() || in.Submitted()) {
		s.editing = false
		c.Invalidate()
	}
	return field
}

// rescale returns amount, in a minor unit of from decimals, in one of to.
func rescale(amount int64, from, to int) int64 {
	for ; from < to; from++ {
		amount *= 10
	}
	for ; from > to; from-- {
		amount /= 10
	}
	return amount
}
