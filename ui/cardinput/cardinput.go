// Package cardinput enters a payment card, as checkout forms do: a card
// above the fields shows what is typed as it would be printed, in the
// colors of the card's brand, and turns over to its back while the
// security code is typed. The number field knows the brand from the first
// digits, groups the digits as the brand prints them, takes no more than
// its numbers have, and goes on to the expiry once the number is whole and
// passes the Luhn check, or says it is mistyped. The expiry takes MM / YY
// and goes on to the code, which takes as many digits as the brand's
// codes have. A month already past says so.
//
//	cardinput.View(c, &app.cardField, &app.card)
//	if ui.PrimaryButton(c, "Pay").Disabled(!app.card.Valid(c.Now())).Clicked() {
//		app.pay()
//	}
package cardinput

import (
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// Card is a payment card: its number in digits, the month and the year,
// such as 2029, through which it is good, the security code, and the
// name on it.
type Card struct {
	Number      string
	Month, Year int
	CVC         string
	Name        string
}

// brand is a card brand: the prefixes of its numbers, each a range of
// prefixes of the same length, how long its numbers are, how it groups
// their digits, how many digits its security codes have, and its colors.
type brand struct {
	name     string
	prefixes [][2]string
	lengths  []int
	groups   []int
	cvc      int
	from, to ui.Color
}

var brands = []brand{
	{"Visa", [][2]string{{"4", "4"}}, []int{13, 16, 19}, []int{4, 4, 4, 4, 3}, 3, ui.Hex("#1a1f71"), ui.Hex("#3a5bd9")},
	{"Mastercard", [][2]string{{"51", "55"}, {"2221", "2720"}}, []int{16}, []int{4, 4, 4, 4}, 3, ui.Hex("#2b2b2b"), ui.Hex("#5a5a5a")},
	{"American Express", [][2]string{{"34", "34"}, {"37", "37"}}, []int{15}, []int{4, 6, 5}, 4, ui.Hex("#1d6fa5"), ui.Hex("#5fb4d6")},
	{"Discover", [][2]string{{"6011", "6011"}, {"65", "65"}, {"644", "649"}}, []int{16, 19}, []int{4, 4, 4, 4, 3}, 3, ui.Hex("#c75a12"), ui.Hex("#f29b38")},
	{"UnionPay", [][2]string{{"62", "62"}}, []int{16, 17, 18, 19}, []int{4, 4, 4, 4, 3}, 3, ui.Hex("#0f5c5a"), ui.Hex("#d6203d")},
	{"JCB", [][2]string{{"3528", "3589"}}, []int{16, 19}, []int{4, 4, 4, 4, 3}, 3, ui.Hex("#0b4a8f"), ui.Hex("#2fa15d")},
	{"Diners Club", [][2]string{{"36", "36"}, {"300", "305"}, {"38", "39"}}, []int{14, 16}, []int{4, 6, 4}, 3, ui.Hex("#3b4a5a"), ui.Hex("#8a99a8")},
}

var unknown = brand{"", nil, []int{16}, []int{4, 4, 4, 4, 3}, 3, ui.Hex("#3a3f4b"), ui.Hex("#6b7280")}

// lookup returns the brand of number, by as many digits as it has.
func lookup(number string) (brand, bool) {
	for _, b := range brands {
		for _, p := range b.prefixes {
			if len(number) >= len(p[0]) && number[:len(p[0])] >= p[0] && number[:len(p[0])] <= p[1] {
				return b, true
			}
		}
	}
	return unknown, false
}

// Brand returns the name of the brand of the card numbered number, such as
// "Visa", or "" when its first digits match none.
func Brand(number string) string {
	b, _ := lookup(number)
	return b.name
}

// luhn reports whether number passes the Luhn check.
func luhn(number string) bool {
	sum := 0
	for i := range len(number) {
		d := int(number[len(number)-1-i] - '0')
		if i%2 == 1 {
			d *= 2
			if d > 9 {
				d -= 9
			}
		}
		sum += d
	}
	return sum%10 == 0
}

// whole reports whether number has a length the brand's numbers have.
func (b brand) whole(number string) bool {
	for _, n := range b.lengths {
		if len(number) == n {
			return true
		}
	}
	return false
}

func (b brand) longest() int { return b.lengths[len(b.lengths)-1] }

// usual returns how long most of the brand's numbers are, 16 but for
// brands whose numbers are shorter.
func (b brand) usual() int { return min(16, b.longest()) }

// Valid reports whether c can be charged at now: its number has its
// brand's length and passes the Luhn check, it is good through now's
// month, and its security code has its brand's digits.
func (c Card) Valid(now time.Time) bool {
	b, _ := lookup(c.Number)
	return b.whole(c.Number) && luhn(c.Number) && c.current(now) && len(c.CVC) == b.cvc
}

// current reports whether c is good through now's month.
func (c Card) current(now time.Time) bool {
	if c.Month < 1 || c.Month > 12 {
		return false
	}
	return c.Year > now.Year() || c.Year == now.Year() && c.Month >= int(now.Month())
}

// State is what a card field keeps: the text of its fields, and the field
// to go on to. Keep it in the app's state, one for each card field.
type State struct {
	number, expiry string
	caret          int
	moved          bool
	focus          int
}

const (
	numberField = iota + 1
	expiryField
	cvcField
	nameField
)

// View creates the field of *card.
func View(c *ui.Context, s *State, card *Card) ui.Element {
	t := c.Theme()
	changed := false
	b, known := lookup(card.Number)
	if digits(s.number) != card.Number {
		s.number = group(card.Number, b.groups)
	}
	if card.Month > 0 && s.expiry == "" {
		s.expiry = expiry(two(card.Month) + two(card.Year%100))
	}
	root := ui.Column(c).Gap(t.Space(5)).Width(t.Space(92))
	root.Children(func() {
		face := ui.Box(c).Size(t.Space(85), t.Space(53.5)).AlignSelf(ui.Center)
		target := float32(0)
		if s.focus == cvcField {
			target = 180
		}
		turn := face.AnimateTo("cardinput.turn", target, ui.Snappy)
		face.Children(func() {
			front(c, *card, b, known).RotateY(turn).Perspective(900).BackfaceHidden()
			back(c, *card, b).RotateY(turn + 180).Perspective(900).BackfaceHidden()
		})
		ui.Column(c).Gap(t.Space(3)).Children(func() {
			numberFocused := false
			ui.Column(c).Gap(t.Space(1)).Children(func() {
				ui.Text(c, "Card number").FontSize(t.FontSize - 1).TextColor(t.TextMuted)
				ui.Row(c).AlignItems(ui.Center).Gap(t.Space(2)).Children(func() {
					field := ui.TextInput(c, &s.number).Grow(1).Placeholder("1234 1234 1234 1234").Label("Card number").
						FontFeatures("tnum").FocusBind(&s.focus, numberField)
					numberFocused = field.Focused()
					if s.moved {
						field.SetTextSelection(s.caret, s.caret)
						s.moved = false
					}
					if field.Changed() {
						_, caret := field.TextSelection()
						typed := []rune(s.number)
						before := len(digits(string(typed[:min(caret, len(typed))])))
						n := digits(s.number)
						nb, _ := lookup(n)
						n = n[:min(len(n), nb.longest())]
						card.Number, changed = n, true
						s.number = group(n, nb.groups)
						s.caret, s.moved = after(s.number, min(before, len(n))), true
						if len(n) >= nb.usual() && nb.whole(n) && luhn(n) {
							s.focus = expiryField
						}
					}
					if known {
						ui.Text(c, b.name).FontSize(t.FontSize - 1).Bold().TextColor(t.TextMuted)
					}
				})
				if n := card.Number; !luhn(n) && (len(n) == b.longest() || b.whole(n) && len(n) >= b.usual() && !numberFocused) {
					ui.Text(c, "This number is mistyped.").FontSize(t.FontSize - 1).TextColor(t.Danger)
				}
			})
			ui.Row(c).Gap(t.Space(3)).Children(func() {
				ui.Column(c).Gap(t.Space(1)).Grow(1).Width(0).Children(func() {
					ui.Text(c, "Expiry").FontSize(t.FontSize - 1).TextColor(t.TextMuted)
					field := ui.TextInput(c, &s.expiry).Placeholder("MM / YY").Label("Expiry").FontFeatures("tnum").
						FocusBind(&s.focus, expiryField)
					if field.Changed() {
						d := digits(s.expiry)
						if len(d) == 1 && d[0] > '1' {
							d = "0" + d
						}
						d = d[:min(len(d), 4)]
						s.expiry = expiry(d)
						field.SetTextSelection(len([]rune(s.expiry)), len([]rune(s.expiry)))
						card.Month, card.Year, changed = 0, 0, true
						if len(d) == 4 {
							m, _ := strconv.Atoi(d[:2])
							y, _ := strconv.Atoi(d[2:])
							card.Month, card.Year = m, 2000+y
							if card.current(c.Now()) {
								s.focus = cvcField
							}
						}
					}
					switch {
					case len(digits(s.expiry)) == 4 && (card.Month < 1 || card.Month > 12):
						ui.Text(c, "There is no such month.").FontSize(t.FontSize - 1).TextColor(t.Danger)
					case len(digits(s.expiry)) == 4 && !card.current(c.Now()):
						ui.Text(c, "This card has expired.").FontSize(t.FontSize - 1).TextColor(t.Danger)
					}
				})
				ui.Column(c).Gap(t.Space(1)).Grow(1).Width(0).Children(func() {
					ui.Text(c, "Security code").FontSize(t.FontSize - 1).TextColor(t.TextMuted)
					field := ui.TextInput(c, &card.CVC).Placeholder(strings.Repeat("•", b.cvc)).Label("Security code").
						FontFeatures("tnum").FocusBind(&s.focus, cvcField)
					if field.Changed() {
						d := digits(card.CVC)
						card.CVC, changed = d[:min(len(d), b.cvc)], true
					}
				})
			})
			ui.Column(c).Gap(t.Space(1)).Children(func() {
				ui.Text(c, "Name on card").FontSize(t.FontSize - 1).TextColor(t.TextMuted)
				if ui.TextInput(c, &card.Name).Label("Name on card").FocusBind(&s.focus, nameField).Changed() {
					changed = true
				}
			})
		})
	})
	if changed {
		root.SetChanged()
	}
	return root
}

// front creates the front of the card.
func front(c *ui.Context, card Card, b brand, known bool) ui.Element {
	t := c.Theme()
	white := ui.RGB(255, 255, 255)
	face := ui.Column(c).Fill().Absolute().Radius(t.Space(4)).Gradient(b.from, b.to, 120).Padding(t.Space(6)).
		Shadow(0, 8, 24, 0, ui.RGBA(0, 0, 0, 0.25)).Justify(ui.SpaceBetween)
	face.Children(func() {
		ui.Row(c).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Size(t.Space(11), t.Space(8)).Radius(t.Space(1.5)).Gradient(ui.Hex("#e8cf8a"), ui.Hex("#b8973f"), 135)
			ui.Box(c).Grow(1)
			if known {
				ui.Text(c, b.name).Bold().Italic().FontSize(t.FontSize + 3).TextColor(white)
			}
		})
		shown := group(card.Number+strings.Repeat("•", max(0, b.usual()-len(card.Number))), b.groups)
		ui.Text(c, shown).Font("monospace").FontSize(t.FontSize + 7).LetterSpacing(1).TextColor(white).SingleLine()
		ui.Row(c).AlignItems(ui.End).Gap(t.Space(4)).Children(func() {
			name := strings.ToUpper(strings.TrimSpace(card.Name))
			if name == "" {
				name = "YOUR NAME"
			}
			ui.Text(c, name).FontSize(t.FontSize).LetterSpacing(1).TextColor(white.Alpha(0.9)).SingleLine().Grow(1).Shrink(1)
			ui.Column(c).AlignItems(ui.End).Children(func() {
				ui.Text(c, "GOOD THRU").FontSize(t.FontSize - 5).TextColor(white.Alpha(0.7))
				mmyy := "MM/YY"
				if card.Month > 0 {
					mmyy = two(card.Month) + "/" + two(card.Year%100)
				}
				ui.Text(c, mmyy).Font("monospace").TextColor(white)
			})
		})
	})
	return face
}

// back creates the back of the card, with the security code.
func back(c *ui.Context, card Card, b brand) ui.Element {
	t := c.Theme()
	face := ui.Column(c).Fill().Absolute().Radius(t.Space(4)).Gradient(b.to, b.from, 120).Clip().Padding(t.Space(5), 0).
		Gap(t.Space(4)).Shadow(0, 8, 24, 0, ui.RGBA(0, 0, 0, 0.25))
	face.Children(func() {
		ui.Box(c).Height(t.Space(10)).Background(ui.RGB(20, 20, 20))
		ui.Row(c).Padding(0, t.Space(6)).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Box(c).Grow(1).Height(t.Space(8)).Radius(t.Space(1)).Background(ui.RGBA(255, 255, 255, 0.85))
			code := card.CVC + strings.Repeat("•", max(0, b.cvc-len(card.CVC)))
			ui.Box(c).Height(t.Space(8)).Padding(0, t.Space(3)).Radius(t.Space(1)).Background(ui.RGB(255, 255, 255)).Center().
				Children(func() {
					ui.Text(c, code).Font("monospace").Bold().TextColor(ui.RGB(20, 20, 20))
				})
		})
	})
	return face
}

func digits(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// group writes s in groups of the sizes given, a space between them.
func group(s string, sizes []int) string {
	var parts []string
	r := []rune(s)
	for _, n := range sizes {
		if len(r) == 0 {
			break
		}
		k := min(n, len(r))
		parts = append(parts, string(r[:k]))
		r = r[k:]
	}
	if len(r) > 0 {
		parts = append(parts, string(r))
	}
	return strings.Join(parts, " ")
}

// after returns the offset in s, in runes, after its n-th digit.
func after(s string, n int) int {
	if n == 0 {
		return 0
	}
	seen := 0
	for i, r := range []rune(s) {
		if r >= '0' && r <= '9' {
			seen++
			if seen == n {
				return i + 1
			}
		}
	}
	return len([]rune(s))
}

// expiry writes the digits d of an expiry as MM / YY.
func expiry(d string) string {
	if len(d) <= 2 {
		return d
	}
	return d[:2] + " / " + d[2:]
}

func two(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}
