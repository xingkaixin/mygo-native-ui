// Package semver tries a version range against a package's versions, as
// npm's semver calculator does: the range typed in npm's syntax, with
// carets, tildes, x-ranges, hyphen ranges and sets joined by ||, written
// out below as the bounds it means, or what is wrong with it in red; and
// the versions under it by major version, those it matches in the accent
// color, the highest of them ringed, and prereleases dimmed, which a
// range matches only when it names a prerelease of the same version.
//
//	semver.View(c, &app.query, app.versions)
package semver

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Version is a semantic version: major, minor and patch, and the dotted
// identifiers of a prerelease, if any. Build metadata is dropped, as it
// takes no part in matching.
type Version struct {
	Major, Minor, Patch int
	Pre                 []string
}

// ParseVersion reads a version such as "1.4.0-rc.1", with or without a v
// or = before it.
func ParseVersion(s string) (Version, error) {
	s = strings.TrimLeft(strings.TrimSpace(s), "v=")
	s, _, _ = strings.Cut(s, "+")
	core, pre, hasPre := strings.Cut(s, "-")
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("%q is not major.minor.patch", s)
	}
	var n [3]int
	for i, p := range parts {
		v, err := strconv.Atoi(p)
		if err != nil || v < 0 {
			return Version{}, fmt.Errorf("%q is not a number in %q", p, s)
		}
		n[i] = v
	}
	v := Version{Major: n[0], Minor: n[1], Patch: n[2]}
	if hasPre {
		v.Pre = strings.Split(pre, ".")
	}
	return v, nil
}

func (v Version) String() string {
	s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if len(v.Pre) > 0 {
		s += "-" + strings.Join(v.Pre, ".")
	}
	return s
}

// Compare orders v and o by semantic version precedence: -1, 0 or 1.
func (v Version) Compare(o Version) int {
	if c := cmp.Or(cmp.Compare(v.Major, o.Major), cmp.Compare(v.Minor, o.Minor), cmp.Compare(v.Patch, o.Patch)); c != 0 {
		return c
	}
	switch {
	case len(v.Pre) == 0 && len(o.Pre) == 0:
		return 0
	case len(v.Pre) == 0:
		return 1
	case len(o.Pre) == 0:
		return -1
	}
	for i := range min(len(v.Pre), len(o.Pre)) {
		a, aErr := strconv.Atoi(v.Pre[i])
		b, bErr := strconv.Atoi(o.Pre[i])
		var c int
		switch {
		case aErr == nil && bErr == nil:
			c = cmp.Compare(a, b)
		case aErr == nil:
			c = -1
		case bErr == nil:
			c = 1
		default:
			c = strings.Compare(v.Pre[i], o.Pre[i])
		}
		if c != 0 {
			return c
		}
	}
	return cmp.Compare(len(v.Pre), len(o.Pre))
}

func (v Version) sameCore(o Version) bool {
	return v.Major == o.Major && v.Minor == o.Minor && v.Patch == o.Patch
}

type comparator struct {
	op string
	v  Version
}

func (c comparator) match(v Version) bool {
	d := v.Compare(c.v)
	switch c.op {
	case "<":
		return d < 0
	case "<=":
		return d <= 0
	case ">":
		return d > 0
	case ">=":
		return d >= 0
	}
	return d == 0
}

// Range is a version range: sets of bounds, of which a version must meet
// every bound of one set.
type Range [][]comparator

// partial is a version as a range writes it, with missing or x parts: n
// says how many parts it has.
type partial struct {
	v Version
	n int
}

func parsePartial(s string) (partial, error) {
	s = strings.TrimLeft(s, "v=")
	s, _, _ = strings.Cut(s, "+")
	core, pre, hasPre := strings.Cut(s, "-")
	var p partial
	if core == "" {
		return p, nil
	}
	fields := strings.Split(core, ".")
	if len(fields) > 3 {
		return p, fmt.Errorf("%q has too many parts", s)
	}
	for i, f := range fields {
		if f == "x" || f == "X" || f == "*" {
			break
		}
		v, err := strconv.Atoi(f)
		if err != nil || v < 0 {
			return p, fmt.Errorf("%q is not a version", s)
		}
		*[...]*int{&p.v.Major, &p.v.Minor, &p.v.Patch}[i] = v
		p.n = i + 1
	}
	if hasPre {
		if p.n < 3 {
			return p, fmt.Errorf("%q has a prerelease but not every part", s)
		}
		p.v.Pre = strings.Split(pre, ".")
	}
	return p, nil
}

// next returns the lowest prerelease of the version after p's last part,
// the exclusive upper bound of p as an x-range.
func (p partial) next() Version {
	switch p.n {
	case 1:
		return Version{Major: p.v.Major + 1, Pre: []string{"0"}}
	case 2:
		return Version{Major: p.v.Major, Minor: p.v.Minor + 1, Pre: []string{"0"}}
	}
	return Version{Major: p.v.Major, Minor: p.v.Minor, Patch: p.v.Patch + 1, Pre: []string{"0"}}
}

// ParseRange reads a range in npm's syntax, such as "^1.2.0", "~1.4",
// ">=2.0.0 <3", "1.x || 2.1.0 - 2.3".
func ParseRange(s string) (Range, error) {
	var r Range
	for set := range strings.SplitSeq(s, "||") {
		cs, err := parseSet(strings.TrimSpace(set))
		if err != nil {
			return nil, err
		}
		r = append(r, cs)
	}
	return r, nil
}

func parseSet(s string) ([]comparator, error) {
	if lo, hi, ok := strings.Cut(s, " - "); ok {
		a, err := parsePartial(strings.TrimSpace(lo))
		if err != nil {
			return nil, err
		}
		b, err := parsePartial(strings.TrimSpace(hi))
		if err != nil {
			return nil, err
		}
		out := []comparator{{">=", a.v}}
		switch {
		case b.n == 0:
		case b.n < 3:
			out = append(out, comparator{"<", b.next()})
		default:
			out = append(out, comparator{"<=", b.v})
		}
		return out, nil
	}
	// Operators may stand apart from their versions, as in ">= 1.2".
	fields := strings.Fields(s)
	for i := 0; i < len(fields)-1; i++ {
		if strings.Trim(fields[i], "<>=~^") == "" {
			fields[i] += fields[i+1]
			fields = slices.Delete(fields, i+1, i+2)
		}
	}
	var out []comparator
	for _, f := range fields {
		op := f[:len(f)-len(strings.TrimLeft(f, "<>=~^"))]
		p, err := parsePartial(f[len(op):])
		if err != nil {
			return nil, err
		}
		cs, err := desugar(op, p)
		if err != nil {
			return nil, err
		}
		out = append(out, cs...)
	}
	if len(out) == 0 {
		out = []comparator{{">=", Version{}}}
	}
	return out, nil
}

func desugar(op string, p partial) ([]comparator, error) {
	v := p.v
	switch op {
	case "", "=":
		if p.n == 0 {
			return []comparator{{">=", Version{}}}, nil
		}
		if p.n < 3 {
			return []comparator{{">=", v}, {"<", p.next()}}, nil
		}
		return []comparator{{"=", v}}, nil
	case "~", "~>":
		if p.n == 0 {
			return []comparator{{">=", Version{}}}, nil
		}
		upper := partial{v, min(p.n, 2)}
		return []comparator{{">=", v}, {"<", upper.next()}}, nil
	case "^":
		if p.n == 0 {
			return []comparator{{">=", Version{}}}, nil
		}
		// The bound stops at the first part that is not 0, or the last
		// part given.
		upper := partial{v, 1}
		switch {
		case v.Major == 0 && p.n >= 2 && (v.Minor > 0 || p.n == 2):
			upper.n = 2
		case v.Major == 0 && v.Minor == 0 && p.n == 3:
			upper.n = 3
		}
		return []comparator{{">=", v}, {"<", upper.next()}}, nil
	case ">":
		if p.n < 3 {
			return []comparator{{">=", p.next()}}, nil
		}
	case "<":
		if p.n < 3 {
			return []comparator{{"<", Version{Major: v.Major, Minor: v.Minor, Pre: []string{"0"}}}}, nil
		}
	case ">=":
	case "<=":
		if p.n < 3 {
			return []comparator{{"<", p.next()}}, nil
		}
	default:
		return nil, fmt.Errorf("%q is not an operator", op)
	}
	return []comparator{{op, v}}, nil
}

// Match reports whether v is in r: it meets every bound of a set, and,
// if it is a prerelease, a bound of that set names a prerelease of the
// same version.
func (r Range) Match(v Version) bool {
	for _, set := range r {
		ok := true
		for _, c := range set {
			ok = ok && c.match(v)
		}
		if ok && len(v.Pre) > 0 {
			ok = slices.ContainsFunc(set, func(c comparator) bool { return len(c.v.Pre) > 0 && c.v.sameCore(v) })
		}
		if ok {
			return true
		}
	}
	return false
}

// String writes r out as the bounds it means, as ">=1.2.0 <2.0.0-0".
func (r Range) String() string {
	var sets []string
	for _, set := range r {
		var cs []string
		for _, c := range set {
			cs = append(cs, c.op+c.v.String())
		}
		sets = append(sets, strings.Join(cs, " "))
	}
	return strings.Join(sets, " || ")
}

// View creates the range field of *query and versions, newest first,
// under it.
func View(c *ui.Context, query *string, versions []Version) ui.Element {
	t := c.Theme()
	r, err := ParseRange(*query)
	sorted := slices.SortedFunc(slices.Values(versions), func(a, b Version) int { return b.Compare(a) })
	matched, highest := 0, -1
	for i, v := range sorted {
		if err == nil && r.Match(v) {
			matched++
			if highest < 0 {
				highest = i
			}
		}
	}
	root := ui.Column(c).Gap(t.Space(4))
	changed := false
	root.Children(func() {
		f := ui.Field(c, "Range", func() {
			changed = ui.TextInput(c, query).Font("monospace").Placeholder("^1.2.0").Changed()
		})
		if err != nil {
			f.Error(err.Error())
		} else {
			f.Description("Means " + r.String())
		}
		summary := fmt.Sprintf("%d of %d versions match", matched, len(sorted))
		if highest >= 0 {
			summary += ", the highest " + sorted[highest].String()
		}
		ui.Text(c, summary).TextColor(t.TextMuted)
		ui.Column(c).Gap(t.Space(3)).Children(func() {
			for i := 0; i < len(sorted); {
				major := sorted[i].Major
				ui.Row(c.Key(major)).Gap(t.Space(3)).AlignItems(ui.Start).Children(func() {
					ui.Text(c, fmt.Sprintf("%d.x", major)).Font("monospace").Bold().Width(t.Space(10)).Padding(t.Space(1), 0)
					ui.Row(c).Wrap().Gap(t.Space(1.5)).Grow(1).Shrink(1).Children(func() {
						for ; i < len(sorted) && sorted[i].Major == major; i++ {
							chip(c.Key(sorted[i].String()), sorted[i], err == nil && r.Match(sorted[i]), i == highest)
						}
					})
				})
			}
		})
	})
	if changed {
		root.SetChanged()
	}
	return root
}

func chip(c *ui.Context, v Version, match, highest bool) {
	t := c.Theme()
	e := ui.Text(c, v.String()).Font("monospace").FontSize(t.FontSize-1).Padding(t.Space(0.75), t.Space(2)).Radius(t.Space(1.5))
	switch {
	case match:
		e.Background(t.Accent).TextColor(t.AccentText)
	default:
		e.Background(t.Surface).TextColor(t.Text)
	}
	if len(v.Pre) > 0 && !match {
		e.TextColor(t.TextMuted).Italic()
	}
	if highest {
		e.Border(2, t.Accent.Mix(t.Text, 0.4)).Tooltip("The highest version the range matches")
	}
}
