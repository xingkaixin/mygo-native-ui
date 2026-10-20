// Package filterbuilder edits a filter of records, as Notion's and
// Airtable's filters do: conditions joined by And or by Or, one choice
// for each group, the first under "Where" and the second choosing it.
// A condition picks a field, then an operator for the field's kind, then
// a value in an input that fits it: text, a number or two, a date or a
// number of days, the options to check, or yes or no. Add filter and Add
// group add a condition and a group of conditions, two levels deep at
// most, and × removes one. A condition still missing its field, operator
// or value draws that control's border in the Warning color, and Match
// ignores it. Changed reports an edit.
//
//	filterbuilder.View(c, &app.filter, fields)
//	for _, r := range app.records {
//		if filterbuilder.Match(&app.filter, fields, r) {
//			...
//		}
//	}
package filterbuilder

import (
	"math"
	"slices"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// Kind is what a field holds, which decides its operators and input.
type Kind uint8

const (
	Text Kind = iota
	Number
	Date
	Select
	Bool
)

// Field is a field of the records, by its Name, holding a Kind of value;
// Options are a Select field's.
type Field struct {
	Name    string
	Kind    Kind
	Options []string
}

// Op is an operator, as the builder shows it.
type Op string

const (
	Contains    Op = "contains"
	NotContains Op = "does not contain"
	Is          Op = "is"
	StartsWith  Op = "starts with"
	IsEmpty     Op = "is empty"
	Equal       Op = "="
	NotEqual    Op = "≠"
	Less        Op = "<"
	Greater     Op = ">"
	Between     Op = "between"
	Before      Op = "is before"
	After       Op = "is after"
	WithinLast  Op = "is within the last"
	AnyOf       Op = "is any of"
)

// ops are the operators of each kind, of which a new condition takes the
// first.
var ops = [...][]Op{
	Text:   {Contains, NotContains, Is, StartsWith, IsEmpty},
	Number: {Equal, NotEqual, Less, Greater, Between},
	Date:   {Is, Before, After, WithinLast},
	Select: {AnyOf},
	Bool:   {Is},
}

// Filter is conditions joined by And, or by Or while Or is set.
type Filter struct {
	Or         bool
	Conditions []Condition
}

// Condition tests the value of Field by Op against the value for its
// field's kind: Text; Number, and To for Between, or the days of
// WithinLast; Date; Values, the options any of which matches; or Bool.
// With Group set, it is a group of conditions in its place.
type Condition struct {
	Field  string
	Op     Op
	Text   string
	Number float64
	To     float64
	Date   time.Time
	Values []string
	Bool   bool
	Group  *Filter
}

// The part of a condition missing, as missing returns it.
const (
	complete = iota
	noField
	noOp
	noValue
)

// missing returns the field of cd and the part of cd not chosen or filled
// in yet.
func missing(cd *Condition, fields []Field) (Field, int) {
	i := slices.IndexFunc(fields, func(f Field) bool { return f.Name == cd.Field })
	if i < 0 {
		return Field{}, noField
	}
	f := fields[i]
	switch {
	case !slices.Contains(ops[f.Kind], cd.Op):
		return f, noOp
	case f.Kind == Text && cd.Op != IsEmpty && cd.Text == "",
		f.Kind == Date && cd.Op != WithinLast && cd.Date.IsZero(),
		f.Kind == Select && len(cd.Values) == 0:
		return f, noValue
	}
	return f, complete
}

// start returns a new condition on f, by its first operator.
func start(f Field) Condition {
	cd := Condition{Field: f.Name, Op: ops[f.Kind][0]}
	switch f.Kind {
	case Date:
		cd.Date = time.Now()
	case Bool:
		cd.Bool = true
	}
	return cd
}

// Match reports whether record matches filter, leaving out the conditions
// that are incomplete and the groups with none that is complete: while
// none is, every record matches. record holds the value of each field by
// its name: a string for a Text or Select field, a float64 or an int for
// a Number, a time.Time for a Date and a bool for a Bool.
func Match(filter *Filter, fields []Field, record map[string]any) bool {
	ok, _ := match(filter, fields, record)
	return ok
}

// match reports whether record matches g, and whether g has a complete
// condition to count.
func match(g *Filter, fields []Field, record map[string]any) (bool, bool) {
	counts, some, all := false, false, true
	for i := range g.Conditions {
		cd := &g.Conditions[i]
		var m bool
		if cd.Group != nil {
			var sub bool
			if m, sub = match(cd.Group, fields, record); !sub {
				continue
			}
		} else {
			f, part := missing(cd, fields)
			if part != complete {
				continue
			}
			m = test(cd, f.Kind, record[cd.Field])
		}
		counts, some, all = true, some || m, all && m
	}
	if !counts {
		return true, false
	}
	if g.Or {
		return some, true
	}
	return all, true
}

// test reports whether v, the value of a field of kind k, passes cd.
func test(cd *Condition, k Kind, v any) bool {
	switch k {
	case Text:
		s, _ := v.(string)
		ls, lt := strings.ToLower(s), strings.ToLower(cd.Text)
		switch cd.Op {
		case Contains:
			return strings.Contains(ls, lt)
		case NotContains:
			return !strings.Contains(ls, lt)
		case Is:
			return ls == lt
		case StartsWith:
			return strings.HasPrefix(ls, lt)
		case IsEmpty:
			return strings.TrimSpace(s) == ""
		}
	case Number:
		var n float64
		switch v := v.(type) {
		case float64:
			n = v
		case int:
			n = float64(v)
		default:
			return false
		}
		switch cd.Op {
		case Equal:
			return n == cd.Number
		case NotEqual:
			return n != cd.Number
		case Less:
			return n < cd.Number
		case Greater:
			return n > cd.Number
		case Between:
			return n >= min(cd.Number, cd.To) && n <= max(cd.Number, cd.To)
		}
	case Date:
		d, ok := v.(time.Time)
		if !ok || d.IsZero() {
			return false
		}
		switch cd.Op {
		case Is:
			return day(d).Equal(day(cd.Date))
		case Before:
			return day(d).Before(day(cd.Date))
		case After:
			return day(d).After(day(cd.Date))
		case WithinLast:
			ago := day(time.Now()).Sub(day(d)).Hours() / 24
			return ago >= 0 && ago <= cd.Number
		}
	case Select:
		s, _ := v.(string)
		return slices.Contains(cd.Values, s)
	case Bool:
		b, _ := v.(bool)
		return b == cd.Bool
	}
	return false
}

// day returns the calendar day of t, as midnight in UTC, so that days of
// different locations compare.
func day(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// maxDepth is how many levels of groups there are: the filter's, its
// groups', and theirs.
const maxDepth = 3

// View creates the builder of *filter over fields: a row for each
// condition, a box for each group with the rows of its own, and Add
// filter and Add group below each group's rows. Changed reports an edit.
func View(c *ui.Context, filter *Filter, fields []Field) ui.Element {
	t := c.Theme()
	b := builder{fields: fields}
	e := ui.Column(c).Gap(t.Space(2))
	e.Children(func() { b.group(c, filter, 1) })
	if b.changed {
		e.SetChanged()
	}
	return e
}

// builder is what View keeps through the groups it builds.
type builder struct {
	fields  []Field
	changed bool
}

// group builds the rows of g, at depth, and its buttons adding to it.
func (b *builder) group(c *ui.Context, g *Filter, depth int) {
	t := c.Theme()
	remove := -1
	for i := range g.Conditions {
		cd := &g.Conditions[i]
		row := ui.Row(c.Key(i)).Gap(t.Space(2)).AlignItems(ui.Center)
		if cd.Group != nil {
			row.AlignItems(ui.Start)
		}
		row.Children(func() {
			// A group's joiner and button line up with its first row, inside
			// its border and padding.
			top := float32(0)
			if cd.Group != nil {
				top = t.Space(2) + 1
			}
			b.joiner(c, g, i).Margin(top, 0, 0, 0)
			if cd.Group != nil {
				ui.Column(c).Grow(1).MinWidth(0).Gap(t.Space(2)).Padding(t.Space(2)).
					Radius(t.Radius+2).Border(1, t.Border).Background(t.Text.Alpha(0.02)).Children(func() {
					b.group(c, cd.Group, depth+1)
				})
			} else {
				b.condition(c, cd)
			}
			ui.Row(c).MinHeight(t.Space(8)).AlignItems(ui.Center).Margin(top, 0, 0, 0).Children(func() {
				x := ui.ButtonBase(c).Size(t.Space(6), t.Space(6)).Radius(t.Space(1)).Center().Label("Remove").Tooltip("Remove")
				if x.Hovered() {
					x.Background(t.SurfaceHover)
				}
				if x.Clicked() {
					remove = i
				}
				x.Children(func() { ui.Text(c, "×").TextColor(t.TextMuted) })
			})
		})
	}
	ui.Row(c).Gap(t.Space(1)).Children(func() {
		if add(c, "Add filter") {
			cd := Condition{}
			if len(b.fields) > 0 {
				cd = start(b.fields[0])
			}
			g.Conditions = append(g.Conditions, cd)
			b.changed = true
		}
		if depth < maxDepth && add(c, "Add group") {
			sub := &Filter{}
			if len(b.fields) > 0 {
				sub.Conditions = []Condition{start(b.fields[0])}
			}
			g.Conditions = append(g.Conditions, Condition{Group: sub})
			b.changed = true
		}
	})
	if remove >= 0 {
		g.Conditions = slices.Delete(g.Conditions, remove, remove+1)
		b.changed = true
	}
}

// add creates a quiet button of label, and reports a click.
func add(c *ui.Context, label string) bool {
	t := c.Theme()
	btn := ui.ButtonBase(c).Padding(t.Space(1.5), t.Space(3.5)).Radius(t.Radius).Gap(t.Space(1.5)).Label(label)
	if btn.Hovered() {
		btn.Background(t.SurfaceHover)
	}
	btn.Children(func() {
		ui.Text(c, "+").TextColor(t.TextMuted)
		ui.Text(c, label).TextColor(t.TextMuted)
	})
	return btn.Clicked()
}

// joiner creates what stands before the condition i of g: "Where" before
// the first, a select of And and Or before the second, and its choice
// before the rest.
func (b *builder) joiner(c *ui.Context, g *Filter, i int) ui.Element {
	t := c.Theme()
	word := "And"
	if g.Or {
		word = "Or"
	}
	return ui.Row(c).Width(t.Space(20)).Shrink(0).MinHeight(t.Space(8)).AlignItems(ui.Center).Children(func() {
		switch i {
		case 0:
			word = "Where"
		case 1:
			if ui.Select(c, &word, []string{"And", "Or"}).MinWidth(0).Grow(1).Label("Join with").Changed() {
				g.Or, b.changed = word == "Or", true
			}
			return
		}
		// As far in as the select's text, so that the words line up.
		ui.Text(c, word).TextColor(t.TextMuted).PaddingStart(t.Space(3.5) + 1)
	})
}

// condition creates the inputs of cd: its field, its operator and its
// value.
func (b *builder) condition(c *ui.Context, cd *Condition) {
	t := c.Theme()
	names := make([]string, len(b.fields))
	for i, f := range b.fields {
		names[i] = f.Name
	}
	name := cd.Field
	field := ui.Select(c, &name, names).Width(t.Space(36)).Label("Field")
	if field.Changed() {
		*cd = start(b.fields[slices.Index(names, name)])
		b.changed = true
	}
	f, part := missing(cd, b.fields)
	if part == noField {
		field.BorderColor(t.Warning)
		ui.Box(c).Grow(1)
		return
	}
	labels := make([]string, len(ops[f.Kind]))
	for i, o := range ops[f.Kind] {
		labels[i] = string(o)
	}
	op := string(cd.Op)
	opSel := ui.Select(c, &op, labels).Width(t.Space(40)).Label("Operator")
	if opSel.Changed() {
		cd.Op, b.changed = Op(op), true
		// A zero date would open the calendar in the year 1.
		if f.Kind == Date && cd.Date.IsZero() {
			cd.Date = time.Now()
		}
	}
	if f, part = missing(cd, b.fields); part == noOp {
		opSel.BorderColor(t.Warning)
		ui.Box(c).Grow(1)
		return
	}
	ui.Row(c).Grow(1).MinWidth(0).Gap(t.Space(2)).Wrap().AlignItems(ui.Center).Children(func() {
		if b.value(c, cd, f, part == noValue) {
			b.changed = true
		}
	})
}

// value creates the input of cd's value, of a field f, its border marked
// while missing, and reports an edit.
func (b *builder) value(c *ui.Context, cd *Condition, f Field, missing bool) bool {
	t := c.Theme()
	mark := func(e ui.Element) ui.Element {
		if missing {
			e.BorderColor(t.Warning)
		}
		return e
	}
	switch {
	case f.Kind == Text && cd.Op != IsEmpty:
		return mark(ui.TextInput(c, &cd.Text).Grow(1).MinWidth(t.Space(30)).Placeholder("Value").Label("Value")).Changed()
	case f.Kind == Number:
		changed := ui.NumberInput(c, &cd.Number, -math.MaxFloat64, math.MaxFloat64, 1).Label("Value").Changed()
		if cd.Op == Between {
			ui.Text(c, "and").TextColor(t.TextMuted)
			changed = ui.NumberInput(c, &cd.To, -math.MaxFloat64, math.MaxFloat64, 1).Label("To").Changed() || changed
		}
		return changed
	case f.Kind == Date && cd.Op == WithinLast:
		changed := ui.NumberInput(c, &cd.Number, 0, math.MaxFloat64, 1).Label("Days").Changed()
		ui.Text(c, "days").TextColor(t.TextMuted)
		return changed
	case f.Kind == Date:
		return mark(ui.DateInput(c, &cd.Date).Label("Date")).Changed()
	case f.Kind == Select:
		return anyOf(c, cd, f.Options, mark)
	case f.Kind == Bool:
		yes := "No"
		if cd.Bool {
			yes = "Yes"
		}
		if ui.Select(c, &yes, []string{"Yes", "No"}).Label("Value").Changed() {
			cd.Bool = yes == "Yes"
			return true
		}
	}
	return false
}

// anyOf creates a button showing the options of cd checked, which opens
// them to check, and reports a change.
func anyOf(c *ui.Context, cd *Condition, options []string, mark func(ui.Element) ui.Element) bool {
	t := c.Theme()
	btn := mark(ui.Button(c, "").MinWidth(t.Space(35)).Shrink(1).Justify(ui.SpaceBetween).Label("Values"))
	open := ui.Local(btn, "filterbuilder.open", func() bool { return false })
	if btn.Clicked() {
		*open = !*open
	}
	btn.Children(func() {
		if len(cd.Values) == 0 {
			ui.Text(c, "Choose").TextColor(t.TextMuted).SingleLine()
		} else {
			ui.Text(c, strings.Join(cd.Values, ", ")).SingleLine().Shrink(1).MinWidth(0)
		}
		ui.Box(c).Size(t.Space(2.5), t.Space(2.5)).Shrink(0).Draw(func(p *ui.Painter, r ui.Rect) {
			var path ui.Path
			path.MoveTo(r.X+r.W*0.1, r.Y+r.H*0.3).LineTo(r.X+r.W*0.5, r.Y+r.H*0.7).LineTo(r.X+r.W*0.9, r.Y+r.H*0.3)
			p.StrokePath(&path, 1.5, t.TextMuted)
		})
	})
	changed := false
	ui.Popover(c, btn, open, func() {
		ui.Column(c).Padding(t.Space(1.5), t.Space(2)).Gap(t.Space(2)).Children(func() {
			for _, o := range options {
				on := slices.Contains(cd.Values, o)
				if !ui.Checkbox(c.Key(o), &on, o).Changed() {
					continue
				}
				var vs []string
				for _, x := range options {
					if x == o && on || x != o && slices.Contains(cd.Values, x) {
						vs = append(vs, x)
					}
				}
				cd.Values, changed = vs, true
			}
		})
	})
	return changed
}
