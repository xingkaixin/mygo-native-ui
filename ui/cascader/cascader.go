// Package cascader chooses a path down a hierarchy, as the province, city
// and district of an address: a field showing the path chosen, "浙江 /
// 杭州 / 西湖区", which opens menus side by side, one for each level. The
// pointer resting on an option, or the arrows reaching it, opens what it
// holds in the next menu, and a click or Enter on one that holds nothing
// chooses its path and closes the menus; Up and Down move in a menu, Right
// goes into the next, Left back, and Escape or a click outside closes
// them. Disabled options show dimmed and take neither.
//
// Typing in the field finds the paths holding every word typed, their
// words lit, for Up, Down and Enter or a click to choose; a button in the
// field clears the path. Changed reports a new path.
//
//	if cascader.Field(c, &app.region, "省 / 市 / 区", regions, false, nil).Changed() {
//		app.save()
//	}
package cascader

import (
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

// Option is an option of a menu: its name, which a path chosen holds, the
// options under it, whether the app loads those as it opens, and whether
// it can be chosen. A Lazy option without Children shows a spinner while
// the app loads them; one that holds none clears Lazy.
type Option struct {
	Name     string
	Children []Option
	Lazy     bool
	Disabled bool
}

// sep joins the names of a path as the field shows it.
const sep = " / "

// state is what a field keeps: whether its menus show, the text in it and
// whether the user typed it, the text the paths were found for, the option
// the menus are on in each, as indexes from the first, and the path found
// that the arrows are on; the option or path last under the pointer, the
// keys the field took since the last frame, the paths given to load,
// whether the input had the focus, whether to select its text, and
// whether the arrows moved to what is not yet scrolled to.
type state struct {
	open, typed bool
	text, found string
	active      []int
	on          int
	hovered     [2]int
	keys        []ui.Key
	asked       map[string]bool
	focused     bool
	selectAll   bool
	scroll      bool
}

// match is a path found by typing, and whether one of its options is
// disabled.
type match struct {
	path     []string
	disabled bool
}

// Field creates the field choosing *chosen, the names of a path down
// options, showing placeholder while it is empty. With anyLevel, an
// option that holds others is chosen as one that holds none is, and
// typing finds its path too. load, unless nil, is called once with the
// path of a Lazy option without Children as it opens, for the app to give
// it its Children, or clear Lazy, in a later frame: one loading in the
// background calls Context.Invalidate when done. Give the field a width,
// or grow it in its parent.
func Field(c *ui.Context, chosen *[]string, placeholder string, options []Option, anyLevel bool, load func(path []string)) ui.Element {
	t := c.Theme()
	before := slices.Clone(*chosen)
	field := ui.Row(c).Gap(t.Space(1.5)).AlignItems(ui.Center).Padding(t.Space(1.5), t.Space(2.5)).
		Radius(t.Radius).Background(t.Surface).Cursor(ui.CursorText).MinWidth(t.Space(50))
	st := ui.Local(field, "cascader.state", func() state {
		return state{hovered: [2]int{-1, -1}, asked: map[string]bool{}}
	})
	if !st.open {
		st.typed, st.text = false, strings.Join(*chosen, sep)
	}
	choose := func(path []string) {
		*chosen = path
		st.open, st.typed, st.text = false, false, strings.Join(path, sep)
	}
	open := func() {
		st.open, st.typed, st.selectAll = true, false, true
		st.active = indexes(options, *chosen)
		st.text = strings.Join(*chosen, sep)
	}
	querying := func() bool { return st.open && st.typed && strings.TrimSpace(st.text) != "" }
	find := func() []match {
		if !querying() {
			return nil
		}
		found := search(options, nil, false, strings.Fields(strings.ToLower(st.text)), anyLevel)
		if st.found != st.text {
			st.found, st.on = st.text, step(len(found), -1, 1, func(i int) bool { return !found[i].disabled })
		}
		return found
	}

	for _, k := range st.keys {
		if !st.open {
			open()
			continue
		}
		if querying() {
			found := find()
			enabled := func(i int) bool { return !found[i].disabled }
			switch k {
			case ui.KeyDown, ui.KeyUp:
				st.on, st.scroll = step(len(found), st.on, arrow(k), enabled), true
			case ui.KeyEnter:
				if st.on >= 0 && st.on < len(found) && enabled(st.on) {
					choose(found[st.on].path)
				}
			}
			continue
		}
		menus := menusOf(options, &st.active)
		at := len(st.active) - 1
		enabled := func(menu []Option) func(int) bool { return func(i int) bool { return !menu[i].Disabled } }
		switch {
		case at < 0 && k != ui.KeyLeft:
			from := -1
			if k == ui.KeyUp {
				from = len(options)
			}
			st.active = []int{step(len(options), from, arrow(k), enabled(options))}
		case k == ui.KeyDown || k == ui.KeyUp:
			st.active[at] = step(len(menus[at]), st.active[at], arrow(k), enabled(menus[at]))
		case k == ui.KeyLeft && at > 0:
			st.active = st.active[:at]
		case k == ui.KeyEnter && !menus[at][st.active[at]].Disabled && (anyLevel || leaf(menus[at][st.active[at]])):
			choose(names(menus, st.active))
		case (k == ui.KeyRight || k == ui.KeyEnter) && at+1 < len(menus):
			if i := step(len(menus[at+1]), -1, 1, enabled(menus[at+1])); i >= 0 {
				st.active = append(st.active, i)
			}
		}
		st.active = slices.DeleteFunc(st.active, func(i int) bool { return i < 0 })
		st.scroll = true
	}
	st.keys = st.keys[:0]

	var input ui.Element
	field.Children(func() {
		input = ui.TextInputBase(c, &st.text).Grow(1).Shrink(1).Placeholder(placeholder).Label(placeholder).
			Role(ui.RoleComboBox).Expanded(st.open)
		input.OnTextKey(func(k ui.TextKey) bool {
			if k.Mods != 0 {
				return false
			}
			switch k.Key {
			case ui.KeyUp, ui.KeyDown, ui.KeyEnter:
			case ui.KeyLeft, ui.KeyRight:
				// While the text is a query, the arrows move the caret in it.
				if !st.open || st.typed && strings.TrimSpace(k.Text()) != "" {
					return false
				}
			default:
				return false
			}
			st.keys = append(st.keys, k.Key)
			return true
		})
		if input.Changed() {
			st.open, st.typed = true, true
		}
		if input.Clicked() && !st.open {
			open()
		}
		if len(*chosen) > 0 {
			clear := iconButton(c, "Clear", cross)
			if clear.Clicked() {
				choose(nil)
				input.Focus()
			}
		}
		if iconButton(c, "Show options", chevron).Clicked() {
			if st.open {
				st.open = false
			} else {
				open()
			}
			input.Focus()
		}
		if st.selectAll {
			input.SetTextSelection(0, utf8.RuneCountInString(st.text))
			st.selectAll = false
		}
	})
	if field.Clicked() && !st.open {
		open()
		input.Focus()
	}
	focused := input.Focused()
	if st.open && st.focused && !focused {
		st.open = false
	}
	st.focused = focused
	switch {
	case focused:
		field.Border(1, t.Accent)
	case field.Hovered():
		field.Border(1, t.Border.Mix(t.Text, 0.25))
	default:
		field.Border(1, t.Border)
	}

	found := find()
	menus := menusOf(options, &st.active)
	for i := range st.active {
		if o := menus[i][st.active[i]]; o.Lazy && len(o.Children) == 0 && load != nil && st.open {
			path := names(menus, st.active[:i+1])
			if key := strings.Join(path, "\x00"); !st.asked[key] {
				st.asked[key] = true
				load(path)
			}
		}
	}

	var active ui.Element
	ui.PopoverBase(c, field, &st.open, func(panel ui.Element) {
		panel.Margin(t.Space(1), 0, 0, 0).Radius(t.Radius+2).Clip().Background(t.Background).Border(1, t.Border).
			Shadow(0, 6, 20, 0, ui.RGBA(0, 0, 0, 0.18)).KeepFocus()
		hovered := [2]int{-1, -1}
		panel.Children(func() {
			if querying() {
				words := strings.Fields(strings.ToLower(st.text))
				ui.Scroll(c).MinWidth(field.Bounds().W).MaxHeight(t.Space(72)).Padding(t.Space(1)).Children(func() {
					if len(found) == 0 {
						ui.Text(c, "No matches").TextColor(t.TextMuted).Padding(t.Space(1.5), t.Space(2))
					}
					for i, m := range found {
						r := row(c.Key(i), i == st.on, m.disabled).Label(strings.Join(m.path, sep))
						r.Children(func() { lit(c, strings.Join(m.path, sep), words, m.disabled) })
						if r.Hovered() && !m.disabled {
							hovered = [2]int{0, i}
						}
						if r.Clicked() {
							choose(m.path)
						}
						if i == st.on {
							active = r
						}
					}
				})
				if hovered != st.hovered && hovered[0] >= 0 {
					st.on = hovered[1]
					c.Invalidate()
				}
				st.hovered = hovered
				return
			}
			picked := indexes(options, *chosen)
			ui.Row(c).AlignItems(ui.Stretch).Children(func() {
				for i, menu := range menus {
					if i > 0 {
						ui.Divider(c)
					}
					ui.Scroll(c.Key(i)).Width(t.Space(44)).MaxHeight(t.Space(72)).Padding(t.Space(1)).Children(func() {
						for j, o := range menu {
							on := i < len(st.active) && st.active[i] == j
							mine := i < len(picked) && picked[i] == j && slices.Equal(picked[:i], st.active[:i])
							r := row(c.Key(j), on, o.Disabled).Label(o.Name)
							loading := o.Lazy && len(o.Children) == 0 && st.asked[strings.Join(append(names(menus, st.active[:i]), o.Name), "\x00")]
							r.Children(func() { option(c, o, mine, loading) })
							if r.Hovered() && !o.Disabled {
								hovered = [2]int{i, j}
							}
							if r.Clicked() {
								if anyLevel || leaf(o) {
									choose(append(names(menus, st.active[:i]), o.Name))
								} else {
									st.active = append(st.active[:i], j)
								}
							}
							if on && i == len(st.active)-1 {
								active = r
							}
						}
					})
				}
			})
			if hovered != st.hovered && hovered[0] >= 0 && (hovered[0] >= len(st.active) || st.active[hovered[0]] != hovered[1]) {
				st.active = append(st.active[:hovered[0]], hovered[1])
				// The menus were built before the pointer moved; nothing
				// else asks for the frame that opens the option's own.
				c.Invalidate()
			}
			st.hovered = hovered
		})
	})
	if active.Valid() {
		input.ActiveDescendant(active)
		if st.scroll {
			active.ScrollIntoView()
		}
	}
	st.scroll = false
	if !slices.Equal(*chosen, before) {
		field.SetChanged()
	}
	return field
}

// menusOf returns the menus active opens, the options first, cutting
// active where an option is gone or holds nothing.
func menusOf(options []Option, active *[]int) [][]Option {
	menus := [][]Option{options}
	for i, a := range *active {
		if a < 0 || a >= len(menus[i]) {
			*active = (*active)[:i]
			break
		}
		kids := menus[i][a].Children
		if len(kids) == 0 {
			*active = (*active)[:i+1]
			break
		}
		menus = append(menus, kids)
	}
	return menus
}

// indexes returns the index of each name of path in the menu of its
// level, as far as they are found.
func indexes(options []Option, path []string) []int {
	var at []int
	for _, name := range path {
		i := slices.IndexFunc(options, func(o Option) bool { return o.Name == name })
		if i < 0 {
			break
		}
		at = append(at, i)
		options = options[i].Children
	}
	return at
}

// names returns the names of the options active is on in menus.
func names(menus [][]Option, active []int) []string {
	path := make([]string, len(active))
	for i, a := range active {
		path[i] = menus[i][a].Name
	}
	return path
}

func leaf(o Option) bool { return len(o.Children) == 0 && !o.Lazy }

// search returns the paths under options, after path, whose names, joined
// as the field shows them, hold every one of words, lowercase: those of
// options that hold nothing, or of every option with anyLevel. Those
// under a disabled option are disabled.
func search(options []Option, path []string, disabled bool, words []string, anyLevel bool) []match {
	var found []match
	for _, o := range options {
		p := append(slices.Clip(path), o.Name)
		d := disabled || o.Disabled
		if anyLevel || leaf(o) {
			text := strings.ToLower(strings.Join(p, sep))
			if !slices.ContainsFunc(words, func(w string) bool { return !strings.Contains(text, w) }) {
				found = append(found, match{p, d})
			}
		}
		found = append(found, search(o.Children, p, d, words, anyLevel)...)
	}
	return found
}

// step returns the index from at by d among n, the next for which ok is
// true, or at if there is none.
func step(n, at, d int, ok func(int) bool) int {
	for i := at + d; i >= 0 && i < n; i += d {
		if ok(i) {
			return i
		}
	}
	return at
}

// arrow returns the way an arrow key steps: 1 for Down and Right, -1 for
// Up.
func arrow(k ui.Key) int {
	if k == ui.KeyUp {
		return -1
	}
	return 1
}

// row creates the row of an option or a path found, lit while on.
func row(c *ui.Context, on, disabled bool) ui.Element {
	t := c.Theme()
	r := ui.ButtonBase(c).Gap(t.Space(2)).Padding(t.Space(1.5), t.Space(2)).Radius(t.Space(1.5)).Disabled(disabled)
	if on {
		r.Background(t.Text.Alpha(0.07))
	}
	return r
}

// option builds the content of o's row: its name, in the accent's color
// on the path chosen, and an arrow for one that holds others, or a spinner
// while they load.
func option(c *ui.Context, o Option, chosen, loading bool) {
	t := c.Theme()
	name := ui.Text(c, o.Name).SingleLine().Grow(1).Shrink(1)
	switch {
	case o.Disabled:
		name.TextColor(t.TextMuted.Alpha(0.6))
	case chosen:
		name.TextColor(t.Accent).FontWeight(600)
	}
	switch {
	case loading:
		ui.Spinner(c).Label("Loading")
	case !leaf(o):
		color := t.TextMuted
		if o.Disabled {
			color = color.Alpha(0.6)
		}
		ui.Box(c).Size(t.Space(2), t.Space(2.5)).Draw(func(p *ui.Painter, r ui.Rect) {
			p.StrokePolyline([][2]float32{{r.X + r.W*0.25, r.Y}, {r.X + r.W*0.75, r.Y + r.H/2}, {r.X + r.W*0.25, r.Y + r.H}}, 1.5, color)
		})
	}
}

// lit builds text with each place it holds one of words, lowercase, in
// the accent's color.
func lit(c *ui.Context, text string, words []string, disabled bool) {
	t := c.Theme()
	color := t.Text
	if disabled {
		color = t.TextMuted.Alpha(0.6)
	}
	lower := strings.ToLower(text)
	on := make([]bool, len(text))
	// Lowercasing changes the length of a few runes; their text shows unlit.
	if len(lower) == len(text) {
		for _, w := range words {
			for i := 0; ; {
				j := strings.Index(lower[i:], w)
				if j < 0 {
					break
				}
				for k := i + j; k < i+j+len(w); k++ {
					on[k] = true
				}
				i += j + len(w)
			}
		}
	}
	var spans []ui.Span
	for i := 0; i < len(text); {
		j := i + 1
		for j < len(text) && on[j] == on[i] {
			j++
		}
		s := ui.Span{Text: text[i:j], Color: color}
		if on[i] && !disabled {
			s.Color, s.Weight = t.Accent, 600
		}
		spans = append(spans, s)
		i = j
	}
	ui.RichText(c, spans...).SingleLine().Grow(1).Shrink(1)
}

// iconButton creates a small button of the field, named label, drawn by
// icon in the muted text color; a press leaves the focus where it is.
func iconButton(c *ui.Context, label string, icon func(p *ui.Painter, r ui.Rect, color ui.Color)) ui.Element {
	t := c.Theme()
	b := ui.ButtonBase(c).Size(t.Space(4), t.Space(4)).Radius(t.Space(2)).Label(label).KeepFocus().Cursor(ui.CursorDefault)
	color := t.TextMuted
	if b.Hovered() {
		color = t.Text
	}
	return b.Draw(func(p *ui.Painter, r ui.Rect) { icon(p, r, color) })
}

func cross(p *ui.Painter, r ui.Rect, color ui.Color) {
	cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W*0.22
	p.StrokePolyline([][2]float32{{cx - k, cy - k}, {cx + k, cy + k}}, 1.5, color)
	p.StrokePolyline([][2]float32{{cx - k, cy + k}, {cx + k, cy - k}}, 1.5, color)
}

func chevron(p *ui.Painter, r ui.Rect, color ui.Color) {
	cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W*0.22
	p.StrokePolyline([][2]float32{{cx - k, cy - k/2}, {cx, cy + k/2}, {cx + k, cy - k/2}}, 1.5, color)
}
