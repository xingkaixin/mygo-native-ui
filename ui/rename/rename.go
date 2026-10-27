// Package rename renames many files at once, as Finder's Rename Items
// does: replace text in their names, add text before or after them, or
// name them all alike with a number counting up. Below the rule, each
// name shows beside what it becomes, the part changed lit; names that
// would clash, ignoring case as the file systems of macOS and Windows
// do, or that are empty or hold a slash, show in red and hold the Rename
// button back. The extension stays as it is.
//
//	ev := rename.View(c, &app.renamer, app.names)
//	if ev.Renamed {
//		app.moveFiles(app.names, ev.Names)
//	}
package rename

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Mode is what a rule does to the names.
type Mode int

const (
	// Replace replaces Find with With.
	Replace Mode = iota
	// Add adds Text before or after the names.
	Add
	// Format names them Text and a number, Start for the first, Digits
	// wide at least.
	Format
)

// Rule is how names change. Each changes the name without its extension.
type Rule struct {
	Mode Mode
	// Find and With are Replace's: Find is a regular expression with
	// Regexp, in which With may name its groups as $1, and matches any
	// case unless MatchCase.
	Find, With        string
	Regexp, MatchCase bool
	// Text is the text Add adds, and the name Format gives; Before puts
	// it before the name, or the number before it.
	Text   string
	Before bool
	Start  int
	Digits int
}

// Apply returns names as the rule renames them, or an error for a Find
// that is not a regular expression.
func (r Rule) Apply(names []string) ([]string, error) {
	var re *regexp.Regexp
	if r.Mode == Replace && r.Find != "" {
		pat := r.Find
		if !r.Regexp {
			pat = regexp.QuoteMeta(pat)
		}
		if !r.MatchCase {
			pat = "(?i)" + pat
		}
		var err error
		if re, err = regexp.Compile(pat); err != nil {
			return nil, err
		}
	}
	out := make([]string, len(names))
	for i, name := range names {
		ext := path.Ext(name)
		stem := strings.TrimSuffix(name, ext)
		if stem == "" {
			stem, ext = ext, ""
		}
		switch r.Mode {
		case Replace:
			if re != nil {
				with := r.With
				if !r.Regexp {
					with = strings.ReplaceAll(with, "$", "$$")
				}
				stem = re.ReplaceAllString(stem, with)
			}
		case Add:
			if r.Before {
				stem = r.Text + stem
			} else {
				stem += r.Text
			}
		case Format:
			n := fmt.Sprintf("%0*d", max(1, r.Digits), r.Start+i)
			if r.Before {
				stem = n + r.Text
			} else {
				stem = r.Text + n
			}
		}
		out[i] = stem + ext
	}
	return out, nil
}

// Problems returns why each of to cannot be a name, "" where it can.
func Problems(to []string) []string {
	out := make([]string, len(to))
	seen := map[string]int{}
	for _, n := range to {
		seen[strings.ToLower(n)]++
	}
	for i, n := range to {
		switch {
		case strings.TrimSpace(n) == "":
			out[i] = "Empty name"
		case strings.ContainsAny(n, "/:"):
			out[i] = "Has a slash or colon"
		case seen[strings.ToLower(n)] > 1:
			out[i] = "Same name as another"
		}
	}
	return out
}

// State is what a renamer keeps: its rule. Keep it in the app's state,
// one for each view.
type State struct {
	rule  Rule
	ready bool
	list  ui.ListState
}

// Events is a renamer, and whether the Rename button was clicked this
// frame, with the names it gave, in the order of the names in.
type Events struct {
	ui.Element
	Renamed bool
	Names   []string
}

// View creates the renamer of names.
func View(c *ui.Context, s *State, names []string) Events {
	t := c.Theme()
	if !s.ready {
		s.rule, s.ready = Rule{Start: 1, Digits: 2}, true
	}
	to, err := s.rule.Apply(names)
	if err != nil {
		to = names
	}
	problems := Problems(to)
	changes, bad := 0, 0
	for i := range names {
		if to[i] != names[i] {
			changes++
		}
		if problems[i] != "" {
			bad++
		}
	}
	ev := Events{}
	ev.Element = ui.Column(c).Gap(t.Space(4)).Children(func() {
		s.rules(c, err)
		ui.Column(c).Grow(1).Radius(t.Radius).Border(1, t.Border).Clip().Children(func() {
			ui.Grid(c).ColumnTracks(ui.Fr(1), ui.Fixed(t.Space(5)), ui.Fr(1)).Padding(t.Space(2), t.Space(3)).Background(t.Surface).Children(func() {
				ui.Text(c, "Name").FontSize(t.FontSize - 1).TextColor(t.TextMuted).Bold()
				ui.Box(c)
				ui.Text(c, "New Name").FontSize(t.FontSize - 1).TextColor(t.TextMuted).Bold()
			})
			ui.Divider(c)
			ui.List(c, &s.list, len(names), func(i int) {
				row := ui.Grid(c).ColumnTracks(ui.Fr(1), ui.Fixed(t.Space(5)), ui.Fr(1)).Padding(t.Space(1.5), t.Space(3)).AlignItems(ui.Center)
				if i%2 == 1 {
					row.Background(t.Text.Alpha(0.025))
				}
				row.Children(func() {
					ui.Text(c, names[i]).SingleLine().Ellipsis("…").TextColor(t.TextMuted)
					ui.Text(c, "→").TextColor(t.TextMuted)
					ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
						ui.RichText(c, lit(names[i], to[i], t, problems[i] != "")...).SingleLine().Ellipsis("…").Grow(1)
						if problems[i] != "" {
							ui.Text(c, problems[i]).FontSize(t.FontSize - 2).TextColor(t.Danger).SingleLine()
						}
					})
				})
			}).Grow(1)
		})
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			note := fmt.Sprintf("%d of %d names change", changes, len(names))
			if bad > 0 {
				note = fmt.Sprintf("%d names can't be used", bad)
			}
			ui.Text(c, note).TextColor(t.TextMuted).Grow(1)
			label := "Rename"
			if changes > 0 {
				label = fmt.Sprintf("Rename %d Items", changes)
			}
			if ui.PrimaryButton(c, label).Disabled(changes == 0 || bad > 0 || err != nil).Clicked() {
				ev.Renamed, ev.Names = true, to
			}
		})
	})
	return ev
}

// lit returns to's spans with what differs from name lit, all in red when
// bad.
func lit(name, to string, t *ui.Theme, bad bool) []ui.Span {
	if bad {
		return []ui.Span{{Text: to, Color: t.Danger}}
	}
	a, b := []rune(name), []rune(to)
	pre := 0
	for pre < len(a) && pre < len(b) && a[pre] == b[pre] {
		pre++
	}
	suf := 0
	for suf < len(a)-pre && suf < len(b)-pre && a[len(a)-1-suf] == b[len(b)-1-suf] {
		suf++
	}
	mid := string(b[pre : len(b)-suf])
	if mid == "" {
		return []ui.Span{{Text: to}}
	}
	return []ui.Span{
		{Text: string(b[:pre])},
		{Text: mid, Background: t.Accent.Alpha(0.2), Radius: 3},
		{Text: string(b[len(b)-suf:])},
	}
}

// rules creates the rule's controls, err below them.
func (s *State) rules(c *ui.Context, err error) {
	t := c.Theme()
	r := &s.rule
	ui.Column(c).Gap(t.Space(3)).Children(func() {
		mode := int(r.Mode)
		if ui.Segmented(c, &mode, "Replace Text", "Add Text", "Format").Label("Rule").AlignSelf(ui.Start).Changed() {
			r.Mode = Mode(mode)
		}
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			switch r.Mode {
			case Replace:
				ui.Text(c, "Find").TextColor(t.TextMuted)
				ui.TextInput(c, &r.Find).Grow(1).Placeholder("Text").Label("Find")
				ui.Text(c, "Replace with").TextColor(t.TextMuted)
				ui.TextInput(c, &r.With).Grow(1).Label("Replace with")
				ui.Checkbox(c, &r.MatchCase, "Match case")
				ui.Checkbox(c, &r.Regexp, "Regular expression")
			case Add:
				ui.TextInput(c, &r.Text).Grow(1).Placeholder("Text").Label("Text to add")
				where := "after name"
				if r.Before {
					where = "before name"
				}
				if ui.Select(c, &where, []string{"after name", "before name"}).Width(t.Space(36)).Changed() {
					r.Before = where == "before name"
				}
			case Format:
				ui.Text(c, "Name").TextColor(t.TextMuted)
				ui.TextInput(c, &r.Text).Grow(1).Placeholder("Name").Label("Custom name")
				where := "after name"
				if r.Before {
					where = "before name"
				}
				ui.Text(c, "Number").TextColor(t.TextMuted)
				if ui.Select(c, &where, []string{"after name", "before name"}).Width(t.Space(36)).Changed() {
					r.Before = where == "before name"
				}
				ui.Text(c, "Start at").TextColor(t.TextMuted)
				start := float64(r.Start)
				if ui.NumberInput(c, &start, 0, 99999, 1).Label("Start numbers at").Changed() {
					r.Start = int(start)
				}
				ui.Text(c, "Digits").TextColor(t.TextMuted)
				digits := float64(r.Digits)
				if ui.NumberInput(c, &digits, 1, 6, 1).Label("Digits").Changed() {
					r.Digits = int(digits)
				}
			}
		})
		if err != nil {
			ui.Text(c, strings.TrimPrefix(err.Error(), "error parsing regexp: ")).FontSize(t.FontSize - 1).TextColor(t.Danger)
		}
	})
}
