// Package phoneinput enters a phone number from any country, as sign-up
// forms do: a button with the country's flag and calling code, which
// opens a list of the countries to search, and a field that groups the
// digits as the country writes them while they are typed. Pasting or
// typing a number that starts with + or 00 picks its country. A check
// shows once the number has as many digits as the country's numbers do,
// and how many are missing shows below until then.
//
//	phoneinput.View(c, &app.phoneField, &app.phone)
//	if app.phone.Valid() {
//		send(app.phone.E164())
//	}
package phoneinput

import (
	"fmt"
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Country is a country's numbers: its ISO 3166 code, name and calling
// code, how many digits its numbers have after the calling code, and how
// it groups them, a # for each digit.
type Country struct {
	Code, Name, Dial string
	Min, Max         int
	Format           string
}

// Countries are the countries offered, by name.
var Countries = []Country{
	{"AR", "Argentina", "54", 10, 10, "## ####-####"},
	{"AU", "Australia", "61", 9, 9, "### ### ###"},
	{"AT", "Austria", "43", 10, 11, "### #######"},
	{"BE", "Belgium", "32", 9, 9, "### ## ## ##"},
	{"BR", "Brazil", "55", 10, 11, "(##) #####-####"},
	{"CA", "Canada", "1", 10, 10, "(###) ###-####"},
	{"CN", "China", "86", 11, 11, "### #### ####"},
	{"DK", "Denmark", "45", 8, 8, "## ## ## ##"},
	{"EG", "Egypt", "20", 10, 10, "## #### ####"},
	{"FI", "Finland", "358", 9, 10, "## ### ####"},
	{"FR", "France", "33", 9, 9, "# ## ## ## ##"},
	{"DE", "Germany", "49", 10, 11, "#### #######"},
	{"HK", "Hong Kong", "852", 8, 8, "#### ####"},
	{"IN", "India", "91", 10, 10, "##### #####"},
	{"ID", "Indonesia", "62", 9, 12, "###-####-#####"},
	{"IE", "Ireland", "353", 9, 9, "## ### ####"},
	{"IL", "Israel", "972", 9, 9, "##-###-####"},
	{"IT", "Italy", "39", 9, 10, "### ### ####"},
	{"JP", "Japan", "81", 10, 10, "##-####-####"},
	{"MY", "Malaysia", "60", 9, 10, "##-### #####"},
	{"MX", "Mexico", "52", 10, 10, "## #### ####"},
	{"NL", "Netherlands", "31", 9, 9, "# ########"},
	{"NZ", "New Zealand", "64", 8, 10, "## ### #####"},
	{"NG", "Nigeria", "234", 10, 10, "### ### ####"},
	{"NO", "Norway", "47", 8, 8, "### ## ###"},
	{"PH", "Philippines", "63", 10, 10, "### ### ####"},
	{"PL", "Poland", "48", 9, 9, "### ### ###"},
	{"PT", "Portugal", "351", 9, 9, "### ### ###"},
	{"RU", "Russia", "7", 10, 10, "### ###-##-##"},
	{"SG", "Singapore", "65", 8, 8, "#### ####"},
	{"ZA", "South Africa", "27", 9, 9, "## ### ####"},
	{"KR", "South Korea", "82", 9, 10, "##-####-####"},
	{"ES", "Spain", "34", 9, 9, "### ## ## ##"},
	{"SE", "Sweden", "46", 9, 9, "##-### ## ##"},
	{"CH", "Switzerland", "41", 9, 9, "## ### ## ##"},
	{"TW", "Taiwan", "886", 9, 9, "### ### ###"},
	{"TH", "Thailand", "66", 9, 9, "## ### ####"},
	{"TR", "Türkiye", "90", 10, 10, "### ### ## ##"},
	{"UA", "Ukraine", "380", 9, 9, "## ### ## ##"},
	{"AE", "United Arab Emirates", "971", 9, 9, "## ### ####"},
	{"GB", "United Kingdom", "44", 10, 10, "#### ######"},
	{"US", "United States", "1", 10, 10, "(###) ###-####"},
	{"VN", "Vietnam", "84", 9, 9, "## ### ## ##"},
}

// Lookup returns the country of code, as "GB".
func Lookup(code string) (Country, bool) {
	i := slices.IndexFunc(Countries, func(c Country) bool { return c.Code == code })
	if i < 0 {
		return Country{}, false
	}
	return Countries[i], true
}

// Flag returns the flag of the country of code, as an emoji.
func Flag(code string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(code) {
		b.WriteRune(0x1F1E6 + r - 'A')
	}
	return b.String()
}

// Number is a phone number: the code of its country and its digits
// after the calling code.
type Number struct {
	Country  string
	National string
}

// E164 returns n as E.164 writes it, as "+14155550123".
func (n Number) E164() string {
	c, _ := Lookup(n.Country)
	return "+" + c.Dial + n.National
}

// Valid reports whether n has as many digits as its country's numbers.
func (n Number) Valid() bool {
	c, ok := Lookup(n.Country)
	return ok && len(n.National) >= c.Min && len(n.National) <= c.Max
}

// String returns n grouped as its country writes it, after its calling
// code, as "+44 7911 123456".
func (n Number) String() string {
	c, _ := Lookup(n.Country)
	return "+" + c.Dial + " " + group(n.National, c.Format)
}

// group returns digits in the groups of format, the digits past it
// after.
func group(digits, format string) string {
	var b strings.Builder
	i := 0
	for _, r := range format {
		if i == len(digits) {
			break
		}
		if r == '#' {
			b.WriteByte(digits[i])
			i++
		} else {
			b.WriteRune(r)
		}
	}
	b.WriteString(digits[i:])
	return b.String()
}

// after returns the offset in text after its n-th digit.
func after(text []rune, n int) int {
	if n == 0 {
		return 0
	}
	for i, r := range text {
		if r >= '0' && r <= '9' {
			n--
			if n == 0 {
				return i + 1
			}
		}
	}
	return len(text)
}

func onlyDigits(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, s)
}

// Parse returns the number s, in the country of its calling code when it
// starts with + or 00, else in the country of code.
func Parse(s, code string) Number {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "+") || strings.HasPrefix(s, "00") {
		d := strings.TrimPrefix(onlyDigits(s), "00")
		best := -1
		for i, c := range Countries {
			// Of the countries sharing a calling code, the first in
			// Countries with the code wins unless it is code's.
			if strings.HasPrefix(d, c.Dial) && (best < 0 || len(c.Dial) > len(Countries[best].Dial) || c.Dial == Countries[best].Dial && c.Code == code) {
				best = i
			}
		}
		if best >= 0 {
			c := Countries[best]
			return Number{Country: c.Code, National: trim(d[len(c.Dial):], c)}
		}
	}
	c, _ := Lookup(code)
	return Number{Country: code, National: trim(onlyDigits(s), c)}
}

// trim drops the trunk prefix 0 written before numbers inside a country,
// which Italy keeps, and the digits past the longest number.
func trim(d string, c Country) string {
	if c.Code != "IT" {
		d = strings.TrimLeft(d, "0")
	}
	if c.Max > 0 && len(d) > c.Max {
		d = d[:c.Max]
	}
	return d
}

// State is what a field keeps: the text shown, and the list of countries
// with its search. Keep it in the app's state, one for each field.
type State struct {
	text  string
	shown Number
	open  bool
	query string
	// caret is where the caret goes once the field shows the text grouped,
	// a frame after the typing, when moved.
	caret int
	moved bool
}

// View creates the field of *n. Changed reports a change.
func View(c *ui.Context, s *State, n *Number) ui.Element {
	t := c.Theme()
	if n.Country == "" {
		n.Country = "US"
	}
	country, _ := Lookup(n.Country)
	if *n != s.shown {
		s.text, s.shown = group(n.National, country.Format), *n
	}
	changed := false
	root := ui.Column(c).Gap(t.Space(1.5))
	root.Children(func() {
		ui.Row(c).Height(t.Space(9)).Radius(t.Radius).Border(1, t.Border).Background(t.Surface).AlignItems(ui.Center).Children(func() {
			pick := ui.ButtonBase(c).FillHeight().Padding(0, t.Space(2.5)).Role(ui.RolePopUpButton).Label("Country").Expanded(s.open)
			if pick.Clicked() {
				s.open, s.query = !s.open, ""
			}
			if pick.Hovered() || s.open {
				pick.Background(t.SurfaceHover)
			}
			pick.Children(func() {
				ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Children(func() {
					ui.Text(c, Flag(country.Code)).FontSize(t.FontSize + 3)
					ui.Text(c, "+"+country.Dial).FontFeatures("tnum")
					ui.Text(c, "▾").FontSize(t.FontSize - 3).TextColor(t.TextMuted)
				})
			})
			ui.Popover(c, pick, &s.open, func() {
				if code, ok := s.countries(c, n.Country); ok {
					n.Country, s.open = code, false
					c2, _ := Lookup(code)
					n.National = trim(n.National, c2)
					changed = true
				}
			})
			ui.Divider(c).Vertical()
			field := ui.TextInput(c, &s.text).Grow(1).Placeholder(group(strings.Repeat("0", country.Max), country.Format)).
				Label("Phone number").FontFeatures("tnum").Border(0, ui.Color{}).Background(ui.Color{})
			if s.moved {
				field.SetTextSelection(s.caret, s.caret)
				s.moved = false
			}
			if field.Changed() {
				typed := []rune(s.text)
				_, caret := field.TextSelection()
				next := Parse(s.text, n.Country)
				*n, changed = next, true
				c2, _ := Lookup(next.Country)
				s.text, s.shown = group(next.National, c2.Format), next
				// The caret stays after as many digits as it was, which the
				// grouping moved, or at the end when it was there.
				at := len([]rune(s.text))
				if caret < len(typed) && next.Country == country.Code {
					at = after([]rune(s.text), len(onlyDigits(string(typed[:caret]))))
				}
				s.caret, s.moved = at, true
			}
			if n.Valid() {
				ui.Text(c, "✓").TextColor(t.Success).Bold().Padding(0, t.Space(3))
			}
		})
		country, _ = Lookup(n.Country)
		if left := country.Min - len(n.National); left > 0 && n.National != "" {
			ui.Text(c, fmt.Sprintf("%d more %s", left, plural(left, "digit", "digits"))).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
		}
	})
	if changed {
		root.SetChanged()
	}
	return root
}

func plural(n int, one, other string) string {
	if n == 1 {
		return one
	}
	return other
}

// countries creates the list of countries to search, current checked,
// and returns the code of the one clicked.
func (s *State) countries(c *ui.Context, current string) (string, bool) {
	t := c.Theme()
	chosen, ok := "", false
	ui.Column(c).Width(t.Space(80)).Height(t.Space(72)).Children(func() {
		ui.TextInput(c, &s.query).Placeholder("Search").AutoFocus().Label("Search countries").Margin(t.Space(2))
		q := strings.ToLower(strings.TrimPrefix(s.query, "+"))
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).Padding(0, t.Space(1), t.Space(1), t.Space(1)).Children(func() {
				for _, ct := range Countries {
					if q != "" && !strings.Contains(strings.ToLower(ct.Name), q) && !strings.HasPrefix(ct.Dial, q) && strings.ToLower(ct.Code) != q {
						continue
					}
					row := ui.ButtonBase(c.Key(ct.Code)).FillWidth().Padding(t.Space(1.5), t.Space(2)).Radius(t.Space(1.5)).Label(ct.Name)
					if row.Hovered() {
						row.Background(t.SurfaceHover)
					}
					if row.Clicked() {
						chosen, ok = ct.Code, true
					}
					row.Children(func() {
						ui.Row(c).FillWidth().Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
							ui.Text(c, Flag(ct.Code)).FontSize(t.FontSize + 2)
							ui.Text(c, ct.Name).Grow(1).SingleLine().Ellipsis("…")
							ui.Text(c, "+"+ct.Dial).TextColor(t.TextMuted).FontFeatures("tnum")
							mark := ui.Text(c, "✓").TextColor(t.Accent).Width(t.Space(3))
							if ct.Code != current {
								mark.TextColor(ui.Color{})
							}
						})
					})
				}
			})
		})
	})
	return chosen, ok
}
