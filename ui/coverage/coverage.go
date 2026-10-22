// Package coverage shows how much of a Go program its tests ran, from the
// profile go test -coverprofile writes, as go tool cover -html shows it: the
// files by package, each with the share of its statements run and a bar of
// it, under the totals; and the source of the file chosen, the code the
// tests ran on green and the code they did not on red, block by block to
// the column. Profiles of the count and atomic modes tint the code by how
// often it ran, deeper the more, and the pointer resting on a line shows
// its count by its number. N and Shift+N, or the arrows above the code, go
// to the next and previous code not run.
//
//	app.profile, err = coverage.Parse(string(out)) // once, not every frame
//	...
//	coverage.View(c, &app.coverage, app.profile, app.source).Grow(1)
//
//	func (a *app) source(path string) (string, error) {
//		... // the file of the import path, as "example.com/notes/store/store.go"
//	}
//
// It builds only the rows and lines in view, as ui.List does, so profiles
// and files of any size scroll as fast as small ones.
package coverage

import (
	"cmp"
	"fmt"
	"math"
	"path"
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// Block is statements the profile counts as one: from line StartLine,
// column StartCol, up to line EndLine, column EndCol, not included, lines
// and columns counted from 1 and columns in bytes; how many statements it
// holds, and how many times they ran, 0 or 1 in the set mode.
type Block struct {
	StartLine, StartCol int
	EndLine, EndCol     int
	Statements          int
	Count               int
}

// File is a file of the profile, by its import path, as
// "example.com/notes/store/store.go", and its blocks in order.
type File struct {
	Path   string
	Blocks []Block
}

// Profile is what go test -coverprofile wrote: how it counted, "set",
// "count" or "atomic", and the files, by path.
type Profile struct {
	Mode  string
	Files []File
}

// Parse reads a profile. Blocks named more than once, as when tests of
// several packages cover one with -coverpkg, count once, their counts
// added up, or run when any ran in the set mode.
func Parse(text string) (*Profile, error) {
	p := &Profile{}
	blocks := map[string][]Block{}
	for i, l := range strings.Split(text, "\n") {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		if mode, ok := strings.CutPrefix(l, "mode: "); ok {
			if p.Mode != "" && mode != p.Mode {
				return nil, fmt.Errorf("coverage: line %d: mode %s after mode %s", i+1, mode, p.Mode)
			}
			p.Mode = mode
			continue
		}
		if p.Mode == "" {
			return nil, fmt.Errorf("coverage: line %d: a block before the mode line", i+1)
		}
		colon := strings.LastIndexByte(l, ':')
		var b Block
		if colon < 0 {
			return nil, fmt.Errorf("coverage: line %d: %q is not a block", i+1, l)
		}
		if _, err := fmt.Sscanf(l[colon+1:], "%d.%d,%d.%d %d %d", &b.StartLine, &b.StartCol, &b.EndLine, &b.EndCol, &b.Statements, &b.Count); err != nil {
			return nil, fmt.Errorf("coverage: line %d: %q is not a block", i+1, l)
		}
		blocks[l[:colon]] = append(blocks[l[:colon]], b)
	}
	if p.Mode == "" {
		return nil, fmt.Errorf("coverage: no mode line")
	}
	for name, bs := range blocks {
		slices.SortFunc(bs, func(a, b Block) int {
			return cmp.Or(cmp.Compare(a.StartLine, b.StartLine), cmp.Compare(a.StartCol, b.StartCol),
				cmp.Compare(a.EndLine, b.EndLine), cmp.Compare(a.EndCol, b.EndCol))
		})
		merged := bs[:0]
		for _, b := range bs {
			n := len(merged)
			if n == 0 || merged[n-1].StartLine != b.StartLine || merged[n-1].StartCol != b.StartCol ||
				merged[n-1].EndLine != b.EndLine || merged[n-1].EndCol != b.EndCol {
				merged = append(merged, b)
				continue
			}
			if p.Mode == "set" {
				merged[n-1].Count = max(merged[n-1].Count, b.Count)
			} else {
				merged[n-1].Count += b.Count
			}
		}
		p.Files = append(p.Files, File{Path: name, Blocks: merged})
	}
	slices.SortFunc(p.Files, func(a, b File) int { return strings.Compare(a.Path, b.Path) })
	return p, nil
}

// statements returns how many statements of f ran, and how many it has.
func (f *File) statements() (run, all int) {
	for _, b := range f.Blocks {
		all += b.Statements
		if b.Count > 0 {
			run += b.Statements
		}
	}
	return run, all
}

// entry is a row of the list of files: a file of the package pkg, or the
// package's header with a nil file, and the statements run and in all.
type entry struct {
	pkg      string
	file     *File
	run, all int
}

// key returns the identity of the row, a package's ending in a slash as
// no file's does.
func (e entry) key() string {
	if e.file == nil {
		return e.pkg + "/"
	}
	return e.file.Path
}

// share returns the part of e's statements run, counting none as all.
func (e entry) share() float64 {
	if e.all == 0 {
		return 1
	}
	return float64(e.run) / float64(e.all)
}

// mark is the part of a line from byte from up to to that block of the
// file covers.
type mark struct {
	from, to, block int
}

// gap is the uncovered blocks from first to last with no covered one
// among them, which N goes to together.
type gap struct {
	first, last int
}

// State is what a view keeps: the order of the files, the file chosen, its
// lines, their marks and the tokens highlighted so far, the code not run
// gone to, and the width of the list. Keep it in the app's state, one for
// each view.
type State struct {
	profile  *Profile
	order    int
	arranged int
	rows     []entry
	run, all int
	list     ui.ListState
	row      int
	chosen   string
	split    float32

	file   *File
	err    error
	lines  []string
	marks  [][]mark
	least  int
	most   int
	gaps   []gap
	at     int
	scroll ui.ListState
	lang   *editor.Language
	tokens [][]editor.Token
	next   uint64
}

// arrange orders the files of p by package, by name or by the share run,
// least first.
func (s *State) arrange(p *Profile) {
	s.profile, s.arranged = p, s.order
	files := map[string][]entry{}
	var pkgs []entry
	s.run, s.all = 0, 0
	for i := range p.Files {
		f := &p.Files[i]
		run, all := f.statements()
		s.run, s.all = s.run+run, s.all+all
		dir := path.Dir(f.Path)
		if files[dir] == nil {
			pkgs = append(pkgs, entry{pkg: dir})
		}
		files[dir] = append(files[dir], entry{pkg: dir, file: f, run: run, all: all})
	}
	for i := range pkgs {
		for _, e := range files[pkgs[i].pkg] {
			pkgs[i].run, pkgs[i].all = pkgs[i].run+e.run, pkgs[i].all+e.all
		}
	}
	order := func(a, b entry) int {
		if s.order == 1 {
			if c := cmp.Compare(a.share(), b.share()); c != 0 {
				return c
			}
		}
		return strings.Compare(a.key(), b.key())
	}
	slices.SortFunc(pkgs, order)
	s.rows = s.rows[:0]
	for _, pk := range pkgs {
		fs := files[pk.pkg]
		slices.SortFunc(fs, order)
		s.rows = append(append(s.rows, pk), fs...)
	}
}

// choose sets the row of the file chosen, choosing the first file when
// none of the profile is.
func (s *State) choose() {
	s.row = -1
	for i, e := range s.rows {
		if e.file != nil && (e.file.Path == s.chosen || s.row < 0) {
			s.row = i
		}
		if e.file != nil && e.file.Path == s.chosen {
			break
		}
	}
	if s.row >= 0 {
		s.chosen = s.rows[s.row].file.Path
	}
}

// load reads the source of f and marks its lines with f's blocks.
func (s *State) load(f *File, source func(path string) (string, error)) {
	s.file, s.at = f, -1
	s.lines, s.marks, s.gaps, s.least, s.most = nil, nil, nil, 0, 0
	s.lang, s.tokens, s.next = editor.LanguageFor(path.Base(f.Path)), nil, 0
	s.scroll.ScrollTo(0, ui.Start)
	text, err := source(f.Path)
	if s.err = err; err != nil {
		return
	}
	text = strings.TrimSuffix(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	s.lines = strings.Split(text, "\n")
	s.marks = make([][]mark, len(s.lines))
	for i, b := range f.Blocks {
		s.most = max(s.most, b.Count)
		if b.Count > 0 && (s.least == 0 || b.Count < s.least) {
			s.least = b.Count
		}
		for l := max(b.StartLine, 1); l <= min(b.EndLine, len(s.lines)); l++ {
			from, to := 0, len(s.lines[l-1])
			if l == b.StartLine {
				from = min(b.StartCol-1, to)
			} else {
				from = to - len(strings.TrimLeft(s.lines[l-1], " \t"))
			}
			if l == b.EndLine {
				to = min(b.EndCol-1, to)
			}
			if from < to {
				s.marks[l-1] = append(s.marks[l-1], mark{from, to, i})
			}
		}
		if b.Count > 0 {
			continue
		}
		if n := len(s.gaps); n > 0 && s.gaps[n-1].last == i-1 {
			s.gaps[n-1].last = i
		} else {
			s.gaps = append(s.gaps, gap{i, i})
		}
	}
}

// step goes to the code not run after the code gone to, or before it,
// going around past the last and first.
func (s *State) step(by int) {
	n := len(s.gaps)
	if n == 0 || s.lines == nil {
		return
	}
	if s.at < 0 && by < 0 {
		s.at = 0
	}
	s.at = ((s.at+by)%n + n) % n
	s.scroll.ScrollTo(max(s.file.Blocks[s.gaps[s.at].first].StartLine-1, 0), ui.Center)
}

// tokensOf returns the tokens of line i, tokenizing the lines up to it that
// are not yet, as each starts in the state the one before ended in.
func (s *State) tokensOf(i int) []editor.Token {
	if s.lang == nil || s.lang.Tokenizer == nil {
		return nil
	}
	for len(s.tokens) <= i {
		var toks []editor.Token
		toks, s.next = s.lang.Tokenizer.Tokenize(s.lines[len(s.tokens)], s.next, nil)
		s.tokens = append(s.tokens, toks)
	}
	return s.tokens[i]
}

// lineHeight is the height of a line of code, in sizes of its font.
const lineHeight = 1.6

// View creates the viewer of p: the list of files beside the source of the
// file chosen, which source returns for its path, as the profile names it.
// View calls source once for each file as it is chosen, and shows the
// error it returns in place of the code. Give it a size, or Grow.
//
// The list shows the share of all statements run, and the files by
// package, sorted by name or by the share run, least first, each with its
// share and a bar of it. A click or the arrow keys choose a file. Above
// the code are the file's share and how many stretches of code not run it
// has, with arrows going to the previous and next, around past the last
// and first; N and Shift+N do the same while the focus is in the view.
func View(c *ui.Context, s *State, p *Profile, source func(path string) (string, error)) ui.Element {
	t := c.Theme()
	if s.profile != p || s.arranged != s.order {
		s.arrange(p)
	}
	if s.split == 0 {
		s.split = t.Space(72)
	}
	view := ui.Split(c, &s.split, func() { s.files(c) }, func() {
		s.choose()
		if s.row < 0 {
			ui.Text(c, "The profile has no files").TextColor(t.TextMuted).Padding(t.Space(8))
			return
		}
		if f := s.rows[s.row].file; f != s.file {
			s.load(f, source)
		}
		s.source(c)
	})
	if view.Shortcut(0, ui.KeyN) {
		s.step(1)
	}
	if view.Shortcut(ui.Shift, ui.KeyN) {
		s.step(-1)
	}
	return view
}

// files shows the totals, the order and the list of files.
func (s *State) files(c *ui.Context) {
	t := c.Theme()
	s.choose()
	ui.Column(c).Fill().Background(t.Surface).Children(func() {
		ui.Column(c).Gap(t.Space(2)).Padding(t.Space(3)).Children(func() {
			ui.Row(c).Gap(t.Space(2)).AlignItems(ui.End).Children(func() {
				ui.Text(c, percent(s.run, s.all)).FontSize(t.FontSize + 8).Bold().FontFeatures("tnum")
				ui.Textf(c, "%d of %d statements", s.run, s.all).TextColor(t.TextMuted).FontFeatures("tnum").
					SingleLine().Shrink(1)
			})
			bar(c, s.run, s.all, false).AlignSelf(ui.Stretch)
			ui.Segmented(c, &s.order, "Name", "Coverage").Label("Sort by").Margin(t.Space(1), 0, 0, 0)
		})
		ui.Divider(c)
		s.list.Selected = &s.row
		s.list.Key = func(i int) any { return s.rows[i].key() }
		s.list.Header = func(i int) bool { return s.rows[i].file == nil }
		list := ui.List(c, &s.list, len(s.rows), func(i int) {
			e := s.rows[i]
			if e.file == nil {
				ui.Row(c).Gap(t.Space(2)).Padding(t.Space(1.5), t.Space(2)).AlignItems(ui.Center).
					Background(t.Surface).FontSize(t.FontSize - 1).TextColor(t.TextMuted).Children(func() {
					ui.Text(c, e.pkg).Bold().SingleLine().Grow(1).Shrink(1)
					ui.Text(c, percent(e.run, e.all)).FontFeatures("tnum")
				})
				return
			}
			chosen := i == s.row
			ui.Row(c).Gap(t.Space(2)).Padding(t.Space(1), t.Space(2), t.Space(1), t.Space(4)).AlignItems(ui.Center).
				Children(func() {
					ui.Text(c, path.Base(e.file.Path)).SingleLine().Grow(1).Shrink(1)
					bar(c, e.run, e.all, chosen).Width(t.Space(10)).Shrink(0)
					ui.Text(c, percent(e.run, e.all)).FontSize(t.FontSize - 1).FontFeatures("tnum").
						NoWrap().Width(t.Space(13)).Shrink(0).TextAlign(ui.End)
				})
		}).Grow(1).Padding(0, t.Space(1), t.Space(1))
		if list.Changed() && s.row >= 0 && s.row < len(s.rows) && s.rows[s.row].file != nil {
			s.chosen = s.rows[s.row].file.Path
		}
	})
}

// source shows the header of the file chosen and its lines.
func (s *State) source(c *ui.Context) {
	t := c.Theme()
	f := s.file
	run, all := f.statements()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Gap(t.Space(2)).Padding(t.Space(2), t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, f.Path).Bold().SingleLine().Shrink(1)
			ui.Textf(c, "%s · %d of %d statements", percent(run, all), run, all).TextColor(t.TextMuted).
				FontFeatures("tnum").SingleLine().Shrink(1)
			ui.Spacer(c)
			label := "All covered"
			switch {
			case s.at >= 0:
				label = fmt.Sprintf("%d of %d uncovered", s.at+1, len(s.gaps))
			case len(s.gaps) > 0:
				label = fmt.Sprintf("%d uncovered", len(s.gaps))
			}
			ui.Text(c, label).TextColor(t.TextMuted).FontFeatures("tnum").SingleLine()
			none := len(s.gaps) == 0 || s.lines == nil
			if ui.Button(c, "↑").Label("Previous uncovered").Tooltip("Previous uncovered (Shift+N)").Disabled(none).Clicked() {
				s.step(-1)
			}
			if ui.Button(c, "↓").Label("Next uncovered").Tooltip("Next uncovered (N)").Disabled(none).Clicked() {
				s.step(1)
			}
		})
		ui.Divider(c)
		if s.err != nil {
			ui.Text(c, s.err.Error()).TextColor(t.Danger).Padding(t.Space(4))
			return
		}
		s.code(c)
	})
}

// code shows the lines of the file, tinted by their marks.
func (s *State) code(c *ui.Context) {
	t := c.Theme()
	ct := editor.DefaultTheme(t)
	size := t.FontSize - 1
	rowH := size * lineHeight
	counted := s.profile.Mode != "set"
	numW, _ := c.MeasureText(0, ui.Span{Text: strings.Repeat("0", len(strconv.Itoa(len(s.lines)))), Font: "monospace", Size: size})
	countW, _ := c.MeasureText(0, ui.Span{Text: strconv.Itoa(s.most) + "×", Size: size - 1})
	var at gap
	if s.at >= 0 {
		at = s.gaps[s.at]
	} else {
		at = gap{-1, -1}
	}
	tint := func(m mark) ui.Color {
		b := s.file.Blocks[m.block]
		switch {
		case m.block >= at.first && m.block <= at.last:
			return t.Danger.Alpha(0.45)
		case b.Count == 0:
			return t.Danger.Alpha(0.2)
		case counted:
			return t.Success.Alpha(0.14 + 0.3*heat(b.Count, s.least, s.most))
		}
		return t.Success.Alpha(0.2)
	}
	ui.List(c, &s.scroll, len(s.lines), func(i int) {
		marks := s.marks[i]
		row := ui.Row(c).MinHeight(rowH).AlignItems(ui.Start)
		hovered := row.Hovered()
		row.Children(func() {
			if counted {
				count := ""
				if hovered && len(marks) > 0 {
					most := 0
					for _, m := range marks {
						most = max(most, s.file.Blocks[m.block].Count)
					}
					count = strconv.Itoa(most) + "×"
				}
				ui.Text(c, count).FontSize(size - 1).FixedLineHeight(rowH).FontFeatures("tnum").
					TextColor(t.TextMuted).TextAlign(ui.End).NoWrap().Width(countW).Shrink(0).MarginStart(t.Space(2))
			}
			ui.Text(c, strconv.Itoa(i+1)).Font("monospace").FontSize(size).LineHeight(lineHeight).FontFeatures("tnum").
				TextColor(ct.LineNumber).TextAlign(ui.End).Width(numW).Shrink(0).MarginStart(t.Space(3))
			strip := ui.Box(c).Width(3).Shrink(0).AlignSelf(ui.Stretch).MarginStart(t.Space(3))
			if len(marks) > 0 {
				color := t.Success
				for _, m := range marks {
					if s.file.Blocks[m.block].Count == 0 {
						color = t.Danger
					}
				}
				strip.Background(color.Alpha(0.7))
			}
			ui.RichText(c, spans(s.lines[i], s.tokensOf(i), marks, ct, tint)...).Font("monospace").FontSize(size).
				LineHeight(lineHeight).Grow(1).Shrink(1).Padding(0, t.Space(3))
		})
	}).Grow(1)
}

// heat returns how often a block run count times ran among those run from
// least to most times, from 0 to 1, by the logarithm so that loops do not
// wash out the rest.
func heat(count, least, most int) float32 {
	if most <= least {
		return 1
	}
	return float32(math.Log(float64(count)/float64(least)) / math.Log(float64(most)/float64(least)))
}

// bar shows the share run of all statements, run on green and the rest
// on red, or in the color of the text when plain, as on a row chosen.
func bar(c *ui.Context, run, all int, plain bool) ui.Element {
	t := c.Theme()
	on, off := t.Success, t.Danger.Alpha(0.3)
	if plain {
		on, off = t.AccentText, t.AccentText.Alpha(0.3)
	}
	share := float32(1)
	if all > 0 {
		share = float32(run) / float32(all)
	}
	track := ui.Row(c).Height(4).Radius(2).Clip().Background(off)
	track.Children(func() {
		ui.Box(c).WidthPercent(share * 100).AlignSelf(ui.Stretch).Background(on)
	})
	return track
}

// percent returns run of all as a percentage, "83.3%", or a dash for none.
func percent(run, all int) string {
	if all == 0 {
		return "–"
	}
	return strconv.FormatFloat(100*float64(run)/float64(all), 'f', 1, 64) + "%"
}

// spans returns text in spans of its tokens' colors on its marks' tints,
// tabs set to every fourth column.
func spans(text string, tokens []editor.Token, marks []mark, ct *editor.Theme, tint func(mark) ui.Color) []ui.Span {
	if text == "" {
		return []ui.Span{{Text: " "}}
	}
	cuts := []int{len(text)}
	for _, tok := range tokens {
		cuts = append(cuts, min(tok.End, len(text)))
	}
	for _, m := range marks {
		cuts = append(cuts, m.from, m.to)
	}
	slices.Sort(cuts)
	var out []ui.Span
	col, start, tok := 0, 0, 0
	for _, end := range slices.Compact(cuts) {
		if end <= start {
			continue
		}
		for tok < len(tokens) && tokens[tok].End <= start {
			tok++
		}
		st := editor.TokenStyle{}
		if tok < len(tokens) {
			st = ct.Tokens[tokens[tok].Kind]
		}
		var b strings.Builder
		for _, r := range text[start:end] {
			if r == '\t' {
				n := 4 - col%4
				b.WriteString("    "[:n])
				col += n
				continue
			}
			b.WriteRune(r)
			col++
		}
		sp := ui.Span{Text: b.String(), Color: st.Color, Italic: st.Italic}
		if st.Bold {
			sp.Weight = 700
		}
		for _, m := range marks {
			if m.from <= start && start < m.to {
				sp.Background = tint(m)
			}
		}
		out = append(out, sp)
		start = end
	}
	return out
}
