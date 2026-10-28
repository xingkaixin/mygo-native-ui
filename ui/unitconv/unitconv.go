// Package unitconv converts a value between the units of a quantity:
// length, mass, temperature, volume, area, speed, time, data, pressure
// and energy, chosen in a sidebar. The value typed is in the unit
// marked, and every unit of the quantity shows it converted; clicking
// another unit converts the value into it and types in that unit from
// then on.
//
//	unitconv.View(c, &app.units)
package unitconv

import (
	"math"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// unit is a unit of a quantity: v of it is v*scale+offset of the
// quantity's base unit.
type unit struct {
	name, symbol  string
	scale, offset float64
}

type quantity struct {
	name  string
	units []unit
}

var quantities = []quantity{
	{"Length", []unit{
		{"Millimeter", "mm", 1e-3, 0}, {"Centimeter", "cm", 1e-2, 0}, {"Meter", "m", 1, 0}, {"Kilometer", "km", 1e3, 0},
		{"Inch", "in", 0.0254, 0}, {"Foot", "ft", 0.3048, 0}, {"Yard", "yd", 0.9144, 0}, {"Mile", "mi", 1609.344, 0},
		{"Nautical mile", "nmi", 1852, 0},
	}},
	{"Mass", []unit{
		{"Milligram", "mg", 1e-6, 0}, {"Gram", "g", 1e-3, 0}, {"Kilogram", "kg", 1, 0}, {"Tonne", "t", 1e3, 0},
		{"Ounce", "oz", 0.028349523125, 0}, {"Pound", "lb", 0.45359237, 0}, {"Stone", "st", 6.35029318, 0},
	}},
	{"Temperature", []unit{
		{"Celsius", "°C", 1, 273.15}, {"Fahrenheit", "°F", 5.0 / 9, 459.67 * 5 / 9}, {"Kelvin", "K", 1, 0},
	}},
	{"Volume", []unit{
		{"Milliliter", "mL", 1e-3, 0}, {"Liter", "L", 1, 0}, {"Cubic meter", "m³", 1e3, 0},
		{"Teaspoon", "tsp", 0.00492892159375, 0}, {"Tablespoon", "tbsp", 0.01478676478125, 0}, {"Cup", "cup", 0.2365882365, 0},
		{"Pint", "pt", 0.473176473, 0}, {"Quart", "qt", 0.946352946, 0}, {"Gallon", "gal", 3.785411784, 0},
	}},
	{"Area", []unit{
		{"Square centimeter", "cm²", 1e-4, 0}, {"Square meter", "m²", 1, 0}, {"Hectare", "ha", 1e4, 0}, {"Square kilometer", "km²", 1e6, 0},
		{"Square foot", "ft²", 0.09290304, 0}, {"Acre", "ac", 4046.8564224, 0}, {"Square mile", "mi²", 2589988.110336, 0},
	}},
	{"Speed", []unit{
		{"Meter per second", "m/s", 1, 0}, {"Kilometer per hour", "km/h", 1 / 3.6, 0}, {"Mile per hour", "mph", 0.44704, 0},
		{"Knot", "kn", 1852.0 / 3600, 0}, {"Foot per second", "ft/s", 0.3048, 0},
	}},
	{"Time", []unit{
		{"Millisecond", "ms", 1e-3, 0}, {"Second", "s", 1, 0}, {"Minute", "min", 60, 0}, {"Hour", "h", 3600, 0},
		{"Day", "d", 86400, 0}, {"Week", "wk", 604800, 0}, {"Year", "yr", 31557600, 0},
	}},
	{"Data", []unit{
		{"Bit", "bit", 0.125, 0}, {"Byte", "B", 1, 0}, {"Kilobyte", "kB", 1e3, 0}, {"Megabyte", "MB", 1e6, 0},
		{"Gigabyte", "GB", 1e9, 0}, {"Terabyte", "TB", 1e12, 0},
		{"Kibibyte", "KiB", 1 << 10, 0}, {"Mebibyte", "MiB", 1 << 20, 0}, {"Gibibyte", "GiB", 1 << 30, 0},
	}},
	{"Pressure", []unit{
		{"Pascal", "Pa", 1, 0}, {"Kilopascal", "kPa", 1e3, 0}, {"Bar", "bar", 1e5, 0}, {"Atmosphere", "atm", 101325, 0},
		{"Pound per square inch", "psi", 6894.757293168, 0}, {"Millimeter of mercury", "mmHg", 133.322387415, 0},
	}},
	{"Energy", []unit{
		{"Joule", "J", 1, 0}, {"Kilojoule", "kJ", 1e3, 0}, {"Calorie", "cal", 4.184, 0}, {"Kilocalorie", "kcal", 4184, 0},
		{"Watt-hour", "Wh", 3600, 0}, {"Kilowatt-hour", "kWh", 3.6e6, 0}, {"Electronvolt", "eV", 1.602176634e-19, 0},
		{"British thermal unit", "BTU", 1055.05585262, 0},
	}},
}

// State is what a converter keeps: the quantity chosen, the unit typed
// in and what was typed. Keep it in the app's state, one for each
// converter.
type State struct {
	quantity, unit string
	input          string
}

// format writes v to six significant digits, in plain decimals while
// they stay short.
func format(v float64) string {
	if v == 0 || math.IsInf(v, 0) || math.IsNaN(v) {
		return strconv.FormatFloat(v, 'g', -1, 64)
	}
	if a := math.Abs(v); a < 1e-4 || a >= 1e12 {
		return strconv.FormatFloat(v, 'g', 6, 64)
	}
	digits := max(0, 5-int(math.Floor(math.Log10(math.Abs(v)))))
	s := strconv.FormatFloat(v, 'f', digits, 64)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	}
	return s
}

func parse(s string) (float64, error) {
	s = strings.NewReplacer(",", "", "_", "", " ", "").Replace(s)
	if s == "" {
		return 1, nil
	}
	return strconv.ParseFloat(s, 64)
}

// View creates the converter.
func View(c *ui.Context, s *State) ui.Element {
	t := c.Theme()
	q := quantities[0]
	for _, it := range quantities {
		if it.name == s.quantity {
			q = it
		}
	}
	s.quantity = q.name
	from := -1
	for i, u := range q.units {
		if u.name == s.unit || s.unit == "" && from < 0 && u.scale == 1 {
			from = i
		}
	}
	s.unit = q.units[from].name
	root := ui.Row(c).Gap(t.Space(8)).AlignItems(ui.Stretch)
	root.Children(func() {
		if ui.Sidebar(c, &s.quantity, func() {
			for _, it := range quantities {
				ui.SidebarItem(c, it.name, nil, it.name)
			}
		}).Width(t.Space(44)).Changed() {
			s.unit = ""
		}
		ui.Column(c).Gap(t.Space(4)).Grow(1).Children(func() {
			v, err := parse(s.input)
			ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
				ui.TextInput(c, &s.input).Placeholder("1").Label(q.units[from].name).FontSize(t.FontSize + 6).FontFeatures("tnum").Width(t.Space(64))
				ui.Text(c, q.units[from].name).FontSize(t.FontSize + 2).Bold()
			})
			if err != nil {
				ui.Text(c, "Type a number, as 12.5 or 3e8").TextColor(t.Danger)
			}
			base := v*q.units[from].scale + q.units[from].offset
			ui.Column(c).Gap(2).Children(func() {
				for i, u := range q.units {
					value := (base - u.offset) / u.scale
					text := format(value)
					if err != nil {
						text = "–"
					}
					row := ui.ButtonBase(c.Key(u.name)).Justify(ui.Start).Padding(t.Space(2), t.Space(3)).Radius(t.Radius).Label(u.name)
					if i == from {
						row.Background(t.Accent.Alpha(0.12))
					}
					if row.Clicked() && i != from && err == nil {
						s.unit, s.input = u.name, text
					}
					row.Children(func() {
						ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Baseline).Children(func() {
							ui.Text(c, text).FontSize(t.FontSize + 2).FontFeatures("tnum").Bold().Width(t.Space(48)).TextAlign(ui.End)
							ui.Text(c, u.symbol).Width(t.Space(14))
							ui.Text(c, u.name).TextColor(t.TextMuted)
						})
					})
				}
			})
		})
	})
	return root
}
