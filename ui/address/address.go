// Package address enters a postal address as its country writes one, as
// checkout forms do: the country first, which clears the region and the
// postal code, then its fields in its order and with its labels, such as
// Postcode and County for the United Kingdom or the postal code before
// the prefecture for Japan, a list of its states or provinces where it
// has one, and its postal code checked against its pattern and tidied
// once the field is left, "k1a0b1" into "K1A 0B1". A field shows what is
// wrong with it once it has been left, or once the app asks with Check.
// Lines returns the address as an envelope carries it.
//
//	address.View(c, &app.form, &app.address)
//	if ui.PrimaryButton(c, "Continue").Clicked() && app.form.Check(app.address) {
//		ship(app.address.Lines())
//	}
package address

import (
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Field is a field of an address.
type Field uint8

const (
	Line1 Field = iota
	Line2
	District
	City
	Region
	Postal
	fields
)

// Part is a field as a country's form shows it: its label, and whether it
// can be left empty.
type Part struct {
	Field    Field
	Label    string
	Optional bool
}

// Subdivision is a state, province or prefecture: the code an envelope
// carries, if it has one, and its name.
type Subdivision struct {
	Code, Name string
}

// Country is how a country's forms ask for an address: its ISO 3166 code
// and name; its fields, a row of them each; the regions to choose from,
// if any; the pattern of its postal codes and one of them; and how an
// envelope carries the address, a line each, where %1, %2, %d, %c, %r and
// %p stand for the two address lines, the district, the city, the region
// and the postal code.
type Country struct {
	Code, Name      string
	Rows            [][]Part
	Regions         []Subdivision
	Postal, Example string
	Envelope        string
}

func regions(s string) []Subdivision {
	var out []Subdivision
	for _, r := range strings.Split(s, "|") {
		code, name, ok := strings.Cut(r, " ")
		if !ok || strings.ToUpper(code) != code {
			code, name = "", r
		}
		out = append(out, Subdivision{code, name})
	}
	return out
}

var (
	street = []Part{{Field: Line1, Label: "Street address"}}
	unit   = []Part{{Field: Line2, Label: "Apartment, suite, unit", Optional: true}}
)

// Countries are the countries offered, by name.
var Countries = []Country{
	{
		Code: "AU", Name: "Australia",
		Rows:     [][]Part{street, unit, {{City, "Suburb", false}, {Region, "State", false}, {Postal, "Postcode", false}}},
		Regions:  regions("ACT Australian Capital Territory|NSW New South Wales|NT Northern Territory|QLD Queensland|SA South Australia|TAS Tasmania|VIC Victoria|WA Western Australia"),
		Postal:   `^\d{4}$`,
		Example:  "2000",
		Envelope: "%1\n%2\n%c %r %p",
	},
	{
		Code: "BR", Name: "Brazil",
		Rows:     [][]Part{{{Postal, "CEP", false}}, {{Line1, "Street and number", false}}, {{Line2, "Complement", true}}, {{District, "Neighborhood", false}}, {{City, "City", false}, {Region, "State", false}}},
		Postal:   `^\d{5}-\d{3}$`,
		Example:  "01310-100",
		Envelope: "%1\n%2\n%d\n%c - %r\n%p",
	},
	{
		Code: "CA", Name: "Canada",
		Rows:     [][]Part{street, unit, {{City, "City", false}, {Region, "Province", false}, {Postal, "Postal code", false}}},
		Regions:  regions("AB Alberta|BC British Columbia|MB Manitoba|NB New Brunswick|NL Newfoundland and Labrador|NS Nova Scotia|NT Northwest Territories|NU Nunavut|ON Ontario|PE Prince Edward Island|QC Quebec|SK Saskatchewan|YT Yukon"),
		Postal:   `^[A-Z]\d[A-Z] \d[A-Z]\d$`,
		Example:  "K1A 0B1",
		Envelope: "%1\n%2\n%c %r %p",
	},
	{
		Code: "CN", Name: "China",
		Rows:     [][]Part{{{Region, "Province", false}, {City, "City", false}, {District, "District", false}}, street, {{Postal, "Postal code", false}}},
		Regions:  regions("Anhui|Beijing|Chongqing|Fujian|Gansu|Guangdong|Guangxi|Guizhou|Hainan|Hebei|Heilongjiang|Henan|Hubei|Hunan|Inner Mongolia|Jiangsu|Jiangxi|Jilin|Liaoning|Ningxia|Qinghai|Shaanxi|Shandong|Shanghai|Shanxi|Sichuan|Tianjin|Tibet|Xinjiang|Yunnan|Zhejiang"),
		Postal:   `^\d{6}$`,
		Example:  "100000",
		Envelope: "%r %c %d\n%1\n%p",
	},
	{
		Code: "FR", Name: "France",
		Rows:     [][]Part{street, {{Line2, "Apartment, building", true}}, {{Postal, "Postal code", false}, {City, "City", false}}},
		Postal:   `^\d{5}$`,
		Example:  "75001",
		Envelope: "%1\n%2\n%p %c",
	},
	{
		Code: "DE", Name: "Germany",
		Rows:     [][]Part{{{Line1, "Street and house number", false}}, {{Line2, "Additional address info", true}}, {{Postal, "Postal code", false}, {City, "City", false}}},
		Postal:   `^\d{5}$`,
		Example:  "10115",
		Envelope: "%1\n%2\n%p %c",
	},
	{
		Code: "IN", Name: "India",
		Rows:     [][]Part{{{Line1, "Flat, house, building", false}}, {{Line2, "Area, street, sector", true}}, {{City, "City", false}, {Postal, "PIN code", false}}, {{Region, "State", false}}},
		Postal:   `^\d{6}$`,
		Example:  "110001",
		Envelope: "%1\n%2\n%c %p\n%r",
	},
	{
		Code: "JP", Name: "Japan",
		Rows:     [][]Part{{{Postal, "Postal code", false}}, {{Region, "Prefecture", false}}, {{City, "City, ward, town", false}}, street, {{Line2, "Building, room", true}}},
		Regions:  regions("Hokkaido|Aomori|Iwate|Miyagi|Akita|Yamagata|Fukushima|Ibaraki|Tochigi|Gunma|Saitama|Chiba|Tokyo|Kanagawa|Niigata|Toyama|Ishikawa|Fukui|Yamanashi|Nagano|Gifu|Shizuoka|Aichi|Mie|Shiga|Kyoto|Osaka|Hyogo|Nara|Wakayama|Tottori|Shimane|Okayama|Hiroshima|Yamaguchi|Tokushima|Kagawa|Ehime|Kochi|Fukuoka|Saga|Nagasaki|Kumamoto|Oita|Miyazaki|Kagoshima|Okinawa"),
		Postal:   `^\d{3}-\d{4}$`,
		Example:  "100-0001",
		Envelope: "〒%p\n%r %c\n%1\n%2",
	},
	{
		Code: "NL", Name: "Netherlands",
		Rows:     [][]Part{{{Line1, "Street and house number", false}}, unit, {{Postal, "Postcode", false}, {City, "City", false}}},
		Postal:   `^\d{4} [A-Z]{2}$`,
		Example:  "1012 AB",
		Envelope: "%1\n%2\n%p %c",
	},
	{
		Code: "GB", Name: "United Kingdom",
		Rows:     [][]Part{{{Line1, "Address line 1", false}}, {{Line2, "Address line 2", true}}, {{City, "Town or city", false}, {Region, "County", true}}, {{Postal, "Postcode", false}}},
		Postal:   `^[A-Z]{1,2}\d[A-Z\d]? \d[A-Z]{2}$`,
		Example:  "SW1A 1AA",
		Envelope: "%1\n%2\n%c\n%r\n%p",
	},
	{
		Code: "US", Name: "United States",
		Rows:     [][]Part{street, unit, {{City, "City", false}, {Region, "State", false}, {Postal, "ZIP code", false}}},
		Regions:  regions("AL Alabama|AK Alaska|AZ Arizona|AR Arkansas|CA California|CO Colorado|CT Connecticut|DE Delaware|DC District of Columbia|FL Florida|GA Georgia|HI Hawaii|ID Idaho|IL Illinois|IN Indiana|IA Iowa|KS Kansas|KY Kentucky|LA Louisiana|ME Maine|MD Maryland|MA Massachusetts|MI Michigan|MN Minnesota|MS Mississippi|MO Missouri|MT Montana|NE Nebraska|NV Nevada|NH New Hampshire|NJ New Jersey|NM New Mexico|NY New York|NC North Carolina|ND North Dakota|OH Ohio|OK Oklahoma|OR Oregon|PA Pennsylvania|RI Rhode Island|SC South Carolina|SD South Dakota|TN Tennessee|TX Texas|UT Utah|VT Vermont|VA Virginia|WA Washington|WV West Virginia|WI Wisconsin|WY Wyoming"),
		Postal:   `^\d{5}(-\d{4})?$`,
		Example:  "94103",
		Envelope: "%1\n%2\n%c, %r %p",
	},
}

// Lookup returns the country of code, as "GB".
func Lookup(code string) (Country, bool) {
	i := slices.IndexFunc(Countries, func(c Country) bool { return c.Code == code })
	if i < 0 {
		return Country{}, false
	}
	return Countries[i], true
}

func (c Country) part(f Field) (Part, bool) {
	for _, row := range c.Rows {
		for _, p := range row {
			if p.Field == f {
				return p, true
			}
		}
	}
	return Part{}, false
}

// tidy writes a postal code as the country does: in capitals, and, where
// its example has spaces or hyphens, with them as many characters from
// the end as there, which holds for codes of any length, unless that
// still leaves it not a valid code.
func (c Country) tidy(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	if !strings.ContainsAny(c.Example, " -") {
		return s
	}
	sep := map[int]rune{}
	ex := []rune(c.Example)
	n := 0
	for i := len(ex) - 1; i >= 0; i-- {
		if ex[i] == ' ' || ex[i] == '-' {
			sep[n] = ex[i]
		} else {
			n++
		}
	}
	plain := []rune(strings.NewReplacer(" ", "", "-", "").Replace(s))
	var b strings.Builder
	for i, r := range plain {
		if mark, ok := sep[len(plain)-i]; ok && i > 0 {
			b.WriteRune(mark)
		}
		b.WriteRune(r)
	}
	if ok, _ := regexp.MatchString(c.Postal, b.String()); !ok {
		return s
	}
	return b.String()
}

// Address is a postal address: the ISO 3166 code of its country and its
// fields, those its country does not ask for empty.
type Address struct {
	Country                                      string
	Line1, Line2, District, City, Region, Postal string
}

func (a *Address) field(f Field) *string {
	return [...]*string{&a.Line1, &a.Line2, &a.District, &a.City, &a.Region, &a.Postal}[f]
}

// Problem returns what is wrong with field f of a, or "" when nothing is.
func (a Address) Problem(f Field) string {
	country, _ := Lookup(a.Country)
	part, ok := country.part(f)
	if !ok {
		return ""
	}
	v := strings.TrimSpace(*a.field(f))
	switch {
	case v == "" && !part.Optional:
		return "Required"
	case f == Postal && v != "" && country.Postal != "":
		if ok, _ := regexp.MatchString(country.Postal, v); !ok {
			return "Should look like " + country.Example
		}
	}
	return ""
}

// Valid reports whether no field of a has a problem.
func (a Address) Valid() bool {
	for f := range fields {
		if a.Problem(f) != "" {
			return false
		}
	}
	_, ok := Lookup(a.Country)
	return ok
}

// Lines returns a as an envelope carries it, a line each, regions by
// their codes where they have them, and the country's name in capitals
// last.
func (a Address) Lines() []string {
	country, _ := Lookup(a.Country)
	region := a.Region
	for _, r := range country.Regions {
		if r.Name == a.Region && r.Code != "" {
			region = r.Code
		}
	}
	rep := strings.NewReplacer("%1", a.Line1, "%2", a.Line2, "%d", a.District, "%c", a.City, "%r", region, "%p", a.Postal)
	var out []string
	for _, l := range strings.Split(rep.Replace(country.Envelope), "\n") {
		// What is left of a line with fields missing, such as "Paris," or
		// " - SP", loses its stray separators.
		l = strings.Trim(strings.Join(strings.Fields(l), " "), " ,-")
		if l != "" && l != "〒" {
			out = append(out, l)
		}
	}
	return append(out, strings.ToUpper(country.Name))
}

// State is what a form keeps: which fields have been left, which show
// their problems, and which have the focus. Keep it in the app's state,
// one for each form.
type State struct {
	left    [fields]bool
	focused [fields]bool
}

// Check shows the problems of every field of a, as before the app uses
// it, and reports whether there are none.
func (s *State) Check(a Address) bool {
	for f := range s.left {
		s.left[f] = true
	}
	return a.Valid()
}

var names = func() []string {
	var out []string
	for _, c := range Countries {
		out = append(out, c.Name)
	}
	return out
}()

// View creates the form of *a, United States when its country is not
// set. Changed reports a change.
func View(c *ui.Context, s *State, a *Address) ui.Element {
	t := c.Theme()
	if _, ok := Lookup(a.Country); !ok {
		a.Country = "US"
	}
	country, _ := Lookup(a.Country)
	changed := false
	root := ui.Column(c).Gap(t.Space(4))
	root.Children(func() {
		name := country.Name
		ui.Field(c, "Country or region", func() {
			if !ui.Select(c, &name, names).Changed() {
				return
			}
			i := slices.Index(names, name)
			if i < 0 || Countries[i].Code == a.Country {
				return
			}
			country, a.Country, changed = Countries[i], Countries[i].Code, true
			s.left = [fields]bool{}
			a.Region, a.Postal = "", ""
		})
		for i, row := range country.Rows {
			ui.Row(c.Key(country.Code + strconv.Itoa(i))).Gap(t.Space(3)).AlignItems(ui.Start).Children(func() {
				for _, part := range row {
					if s.part(c, country, part, a, len(row) > 1) {
						changed = true
					}
				}
			})
		}
	})
	if changed {
		root.SetChanged()
	}
	return root
}

// part creates the field of part in a row, narrower when a short field
// shares the row, and reports whether it changed.
func (s *State) part(c *ui.Context, country Country, part Part, a *Address, shared bool) bool {
	f := part.Field
	v := a.field(f)
	label := part.Label
	if part.Optional {
		label += " (optional)"
	}
	changed := false
	field := ui.Field(c.Key(f), label, func() {
		if f == Region && len(country.Regions) > 0 {
			var options []string
			for _, r := range country.Regions {
				options = append(options, r.Name)
			}
			if ui.Select(c, v, options).Changed() {
				s.left[f], changed = true, true
			}
			return
		}
		in := ui.TextInput(c, v)
		if f == Postal {
			in.Placeholder(country.Example).FontFeatures("tnum")
		}
		changed = in.Changed()
		focused := in.Focused()
		if s.focused[f] && !focused {
			s.left[f] = true
			if f == Postal && country.tidy(*v) != *v {
				*v, changed = country.tidy(*v), true
			}
		}
		s.focused[f] = focused
	}).Grow(2).Basis(0)
	if shared && (f == Postal || f == Region) {
		field.Grow(1)
	}
	if s.left[f] {
		field.Error(a.Problem(f))
	}
	return changed
}
