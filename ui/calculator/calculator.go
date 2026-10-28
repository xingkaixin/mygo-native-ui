// Package calculator is a calculator with a paper tape, as an adding
// machine keeps one: the keypad and its display beside a strip of paper
// that prints each number entered with the operation that follows it, and
// each total under a rule. It works left to right, as such machines do: 2
// + 3 × 4 is 20. % takes a percent of the total for + and −, and of 100
// otherwise; C clears the entry, then AC everything, leaving a gap on the
// tape. A click on a line of the tape enters its number again. The keys
// type too: digits, the period, + − * / and x, Enter or = for the total,
// Backspace, and Escape to clear.
//
//	calculator.View(c, &app.calc)
package calculator

import (
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// State is what a calculator keeps: the total so far, the operation
// waiting, the entry, and the tape. Keep it in the app's state, one for
// each calculator.
type State struct {
	total float64
	op    string
	entry string
	shown float64
	err   bool
	tape  []line
	seen  int
}

// line is a line of the tape: a number, and the operation after it, or a
// total, or a gap when empty.
type line struct {
	value float64
	op    string
	total bool
	gap   bool
}

var (
	paper = ui.Hex("#FBF8EF")
	print = ui.Hex("#2B2B2B")
	red   = ui.Hex("#C92A2A")
)

func (s *State) apply(op string, v float64) {
	switch op {
	case "+":
		s.total += v
	case "−":
		s.total -= v
	case "×":
		s.total *= v
	case "÷":
		if v == 0 {
			s.err = true
		}
		s.total /= v
	default:
		s.total = v
	}
}

func (s *State) value() float64 {
	if s.entry == "" {
		return s.shown
	}
	v, _ := strconv.ParseFloat(s.entry, 64)
	return v
}

func (s *State) press(key string) {
	if s.err && key != "C" && key != "AC" {
		return
	}
	switch key {
	case "0", "1", "2", "3", "4", "5", "6", "7", "8", "9":
		if len(strings.TrimLeft(s.entry, "-0.")) < 15 {
			if s.entry == "0" {
				s.entry = ""
			}
			s.entry += key
		}
	case ".":
		if !strings.Contains(s.entry, ".") {
			if s.entry == "" {
				s.entry = "0"
			}
			s.entry += "."
		}
	case "⌫":
		s.entry = strings.TrimSuffix(s.entry[:max(0, len(s.entry)-1)], "-")
	case "±":
		switch {
		case s.entry == "":
			s.entry = strconv.FormatFloat(-s.shown, 'f', -1, 64)
		case strings.HasPrefix(s.entry, "-"):
			s.entry = s.entry[1:]
		default:
			s.entry = "-" + s.entry
		}
	case "%":
		v := s.value() / 100
		if s.op == "+" || s.op == "−" {
			v *= s.total
		}
		s.entry = strconv.FormatFloat(v, 'f', -1, 64)
	case "+", "−", "×", "÷":
		if s.entry == "" && s.op != "" && len(s.tape) > 0 {
			s.op = key
			s.tape[len(s.tape)-1].op = key
			return
		}
		v := s.value()
		s.apply(s.op, v)
		s.tape = append(s.tape, line{value: v, op: key})
		s.op, s.entry, s.shown = key, "", s.total
	case "=":
		v := s.value()
		if s.op == "" {
			s.total = v
		} else {
			s.apply(s.op, v)
			s.tape = append(s.tape, line{value: v, op: "="})
		}
		s.tape = append(s.tape, line{value: s.total, total: true})
		s.op, s.entry, s.shown = "", "", s.total
	case "C":
		s.entry, s.err = "", false
	case "AC":
		if len(s.tape) > 0 && !s.tape[len(s.tape)-1].gap {
			s.tape = append(s.tape, line{gap: true})
		}
		s.total, s.op, s.entry, s.shown, s.err = 0, "", "", 0, false
	}
}

// View creates the calculator.
func View(c *ui.Context, s *State) ui.Element {
	t := c.Theme()
	root := ui.Row(c).Gap(t.Space(4)).AlignItems(ui.Stretch).Focusable().FocusRing(false).AutoFocus().Label("Calculator")
	keys := []struct {
		mods ui.Modifiers
		key  ui.Key
		does string
	}{
		{0, ui.KeyPeriod, "."}, {0, ui.KeyBackspace, "⌫"}, {0, ui.KeyDelete, "⌫"},
		{ui.Shift, ui.KeyEqual, "+"}, {0, ui.KeyMinus, "−"}, {ui.Shift, ui.Key8, "×"}, {0, ui.KeyX, "×"},
		{0, ui.KeySlash, "÷"}, {ui.Shift, ui.Key5, "%"}, {0, ui.KeyEqual, "="}, {0, ui.KeyEnter, "="},
		{0, ui.KeyEscape, "C"},
	}
	digits := []ui.Key{ui.Key0, ui.Key1, ui.Key2, ui.Key3, ui.Key4, ui.Key5, ui.Key6, ui.Key7, ui.Key8, ui.Key9}
	for d, k := range digits {
		if root.Shortcut(0, k) {
			s.press(strconv.Itoa(d))
		}
	}
	for _, k := range keys {
		if root.Shortcut(k.mods, k.key) {
			if k.does == "C" && s.entry == "" {
				s.press("AC")
			} else {
				s.press(k.does)
			}
		}
	}
	root.Children(func() {
		keypad(c, s)
		tape(c, s)
	})
	return root
}

// keypad creates the display and the keys.
func keypad(c *ui.Context, s *State) {
	t := c.Theme()
	key := t.Space(15)
	gap := t.Space(2)
	ui.Column(c).Gap(gap).Padding(t.Space(4)).Radius(t.Space(5)).Background(ui.Hex("#1C1C1E")).Children(func() {
		shown := s.entry
		switch {
		case s.err:
			shown = "Error"
		case shown == "":
			shown = number(s.shown)
		default:
			shown = group(shown)
		}
		ui.Column(c).Width(key*4+gap*3).Height(t.Space(24)).Justify(ui.End).AlignItems(ui.End).Padding(0, t.Space(2)).Children(func() {
			op := s.op
			if op == "" {
				op = " "
			}
			ui.Text(c, op).FontSize(t.FontSize + 2).TextColor(ui.RGBA(255, 255, 255, 0.5))
			size := t.FontSize + 30
			if len(shown) > 9 {
				size = t.FontSize + 30 - float32(len(shown)-9)*2.6
			}
			ui.Text(c, shown).FontSize(size).FontFeatures("tnum").TextColor(ui.RGB(255, 255, 255)).Label("Display").Ellipsis("…")
		})
		clear := "AC"
		if s.entry != "" || s.err {
			clear = "C"
		}
		rows := [][]string{
			{clear, "±", "%", "÷"},
			{"7", "8", "9", "×"},
			{"4", "5", "6", "−"},
			{"1", "2", "3", "+"},
			{"0", ".", "="},
		}
		for _, row := range rows {
			ui.Row(c.Key(row[0])).Gap(gap).Children(func() {
				for _, k := range row {
					w := key
					if k == "0" {
						w = key*2 + gap
					}
					bg, fg := ui.Hex("#3A3A3C"), ui.RGB(255, 255, 255)
					switch {
					case strings.Contains("+−×÷=", k):
						bg = ui.Hex("#FF9F0A")
						if k == s.op && s.entry == "" {
							bg, fg = ui.RGB(255, 255, 255), ui.Hex("#FF9F0A")
						}
					case strings.Contains("AC±%", k):
						bg, fg = ui.Hex("#A5A5A5"), ui.RGB(0, 0, 0)
					}
					b := ui.ButtonBase(c.Key(k)).Size(w, key).Radius(key / 2).Background(bg).Center().Label(k).Cursor(ui.CursorPointer)
					if b.Pressed() {
						b.Background(bg.Mix(ui.RGB(255, 255, 255), 0.3))
					}
					if b.Clicked() {
						s.press(k)
					}
					b.Children(func() {
						ui.Text(c, k).FontSize(t.FontSize + 10).TextColor(fg)
					})
				}
			})
		}
	})
}

// tape creates the paper tape, newest at the bottom.
func tape(c *ui.Context, s *State) {
	t := c.Theme()
	ui.Column(c).Width(t.Space(60)).Radius(t.Space(1)).Background(paper).Shadow(0, 2, 8, 0, ui.RGBA(0, 0, 0, 0.15)).Children(func() {
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).FillWidth().Padding(t.Space(4), t.Space(4)).Gap(t.Space(1)).Children(func() {
				if len(s.tape) == 0 {
					ui.Text(c, "The tape prints here").FontSize(t.FontSize - 1).TextColor(print.Alpha(0.4))
				}
				for i, l := range s.tape {
					if l.gap {
						ui.Box(c.Key(i)).Height(t.Space(4))
						continue
					}
					if l.total {
						ui.Box(c.Key(-1 - i)).FillWidth().Height(1).Background(print.Alpha(0.6))
					}
					row := ui.ButtonBase(c.Key(i)).FillWidth().Gap(t.Space(3)).Padding(t.Space(0.5), t.Space(1)).Radius(t.Space(1)).
						Label(number(l.value)).Cursor(ui.CursorPointer)
					if row.Hovered() {
						row.Background(print.Alpha(0.06))
					}
					if row.Clicked() {
						s.entry = strconv.FormatFloat(l.value, 'f', -1, 64)
						s.err = false
					}
					if i == len(s.tape)-1 && s.seen != len(s.tape) {
						row.ScrollIntoView()
					}
					col := print
					if l.value < 0 {
						col = red
					}
					mark := l.op
					if l.total {
						mark = "*"
					}
					row.Children(func() {
						text := ui.Text(c, number(l.value)).Grow(1).TextAlign(ui.End).Font("monospace").FontFeatures("tnum").TextColor(col)
						if l.total {
							text.Bold()
						}
						ui.Text(c, mark).Width(t.Space(4)).Font("monospace").TextColor(print.Alpha(0.6))
					})
				}
			})
		})
		s.seen = len(s.tape)
		ui.Row(c).Padding(t.Space(2)).Justify(ui.End).Children(func() {
			if ui.Button(c, "Clear tape").Disabled(len(s.tape) == 0).Clicked() {
				s.tape = slices.Delete(s.tape, 0, len(s.tape))
			}
		})
	})
}

// number writes v as the display shows it: grouped by thousands, to ten
// places at most, or in powers of ten past what the display holds.
func number(v float64) string {
	if math.IsInf(v, 0) || math.IsNaN(v) {
		return "Error"
	}
	if a := math.Abs(v); a >= 1e15 || a != 0 && a < 1e-9 {
		return strconv.FormatFloat(v, 'g', 9, 64)
	}
	return group(strconv.FormatFloat(math.Round(v*1e10)/1e10, 'f', -1, 64))
}

// group puts commas between the thousands of the whole part of n.
func group(n string) string {
	sign := ""
	if strings.HasPrefix(n, "-") {
		sign, n = "-", n[1:]
	}
	whole, frac, dot := strings.Cut(n, ".")
	var b strings.Builder
	for i, r := range whole {
		if i > 0 && (len(whole)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	if dot {
		return sign + b.String() + "." + frac
	}
	return sign + b.String()
}
