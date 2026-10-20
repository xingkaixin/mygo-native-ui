// Minimap opens a Go file in mygo's code editor with a minimap beside it,
// as Visual Studio Code has: the file given, or fmt's print.go from the Go
// installation. The find field marks its matches in the minimap, and
// Enter selects the next one. The minimap also marks a few made-up
// problems, and the lines changed since a made-up earlier version of the
// file, which the editor's gutter shows too.
//
//	go run ./examples/minimap [file.go]
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"

	"github.com/xingkaixin/mygo-native-ui/ui/minimap"
)

// app is the state the window shows: the editor, the changes since the
// earlier version, and the matches of the find field as of a version of
// the text.
type app struct {
	name    string
	code    *editor.State
	diff    *editor.DiffState
	query   string
	found   string
	version uint64
	matches []int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Padding(8, 12).Gap(10).AlignItems(ui.Center).Children(func() {
			ui.Text(c, a.name).Bold().SingleLine().Shrink(1)
			ui.Spacer(c)
			field := ui.SearchField(c, &a.query).Placeholder("Find").Label("Find").Width(220)
			a.find()
			if field.Submitted() {
				a.next()
			}
			count := ""
			if a.query != "" {
				count = fmt.Sprintf("%d matches", len(a.matches))
			}
			ui.Text(c, count).FontSize(12).FontFeatures("tnum").TextColor(t.TextMuted).Width(76)
		})
		ui.Divider(c)
		ui.Row(c).Grow(1).AlignItems(ui.Stretch).Children(func() {
			editor.View(c, a.code).Grow(1)
			minimap.View(c, a.code, a.marks())
		})
	})
}

// find finds the query in the text, case aside, when either changed.
func (a *app) find() {
	buf := a.code.Buffer
	if a.query == a.found && buf.Version() == a.version {
		return
	}
	a.found, a.version, a.matches = a.query, buf.Version(), a.matches[:0]
	if a.query == "" {
		return
	}
	text, q := strings.ToLower(buf.String()), strings.ToLower(a.query)
	for i := 0; ; {
		k := strings.Index(text[i:], q)
		if k < 0 {
			break
		}
		a.matches = append(a.matches, i+k)
		i += k + len(q)
	}
}

// next selects the match after the caret, or the first.
func (a *app) next() {
	if len(a.matches) == 0 {
		return
	}
	i, _ := slices.BinarySearch(a.matches, a.code.Cursor())
	if i == len(a.matches) {
		i = 0
	}
	a.code.Select(a.matches[i], a.matches[i]+len(a.query))
}

// marks returns the matches, the editor's problems and the changed lines
// for the minimap.
func (a *app) marks() []minimap.Mark {
	buf := a.code.Buffer
	var marks []minimap.Mark
	for _, h := range a.diff.Hunks() {
		kind := minimap.Modified
		switch {
		case h.OldStart == h.OldEnd:
			kind = minimap.Added
		case h.NewStart == h.NewEnd:
			kind = minimap.Deleted
		}
		start := buf.Offset(editor.Pos{Line: h.NewStart})
		end := buf.Offset(editor.Pos{Line: max(h.NewStart, h.NewEnd-1)})
		marks = append(marks, minimap.Mark{Kind: kind, Start: start, End: end})
	}
	for _, m := range a.matches {
		marks = append(marks, minimap.Mark{Kind: minimap.Match, Start: m, End: m + len(a.found)})
	}
	for _, d := range a.code.Diagnostics() {
		kind := minimap.Warning
		if d.Severity == editor.SeverityError {
			kind = minimap.Error
		}
		marks = append(marks, minimap.Mark{Kind: kind, Start: d.Start, End: d.End})
	}
	return marks
}

// problems makes up problems with functions spread through text: on the
// name of the first function after each fraction of it.
func problems(text string) []editor.Diagnostic {
	var out []editor.Diagnostic
	for _, p := range []struct {
		at       float64
		severity editor.Severity
		message  string
	}{
		{0.22, editor.SeverityWarning, "func %s is unused (U1000)"},
		{0.48, editor.SeverityError, "missing return"},
		{0.63, editor.SeverityWarning, "%s: the error result is not checked"},
		{0.86, editor.SeverityError, "%s redeclared in this block"},
	} {
		k := strings.Index(text[int(p.at*float64(len(text))):], "\nfunc ")
		if k < 0 {
			continue
		}
		start := int(p.at*float64(len(text))) + k + len("\nfunc ")
		if strings.HasPrefix(text[start:], "(") {
			start += strings.Index(text[start:], ") ") + 2
		}
		n := strings.IndexAny(text[start:], "([ ")
		if n <= 0 {
			continue
		}
		end := start + n
		name := text[start:end]
		message := p.message
		if strings.Contains(message, "%s") {
			message = fmt.Sprintf(message, name)
		}
		out = append(out, editor.Diagnostic{Start: start, End: end, Severity: p.severity, Message: message, Source: "made up"})
	}
	return out
}

// earlier makes up an earlier version of text: without lines added since,
// with lines since changed as they were, and with lines since deleted.
func earlier(text string) string {
	lines := strings.SplitAfter(text, "\n")
	at := func(f float64) int { return int(f * float64(len(lines))) }
	changed := func(i int) { lines[i] = strings.Replace(lines[i], "\n", " // TODO: remove\n", 1) }
	changed(at(0.38))
	changed(at(0.9))
	changed(at(0.9) + 1)
	lines = slices.Insert(lines, at(0.7), "\t// Deprecated: kept for compatibility.\n", "\t_ = 0\n")
	lines = slices.Delete(lines, at(0.12), at(0.12)+4)
	return strings.Join(lines, "")
}

func main() {
	path := ""
	if len(os.Args) > 1 {
		path = os.Args[1]
	} else if out, err := exec.Command("go", "env", "GOROOT").Output(); err == nil {
		path = filepath.Join(strings.TrimSpace(string(out)), "src", "fmt", "print.go")
	}
	src, err := os.ReadFile(path)
	if err != nil {
		log.Fatal("usage: minimap [file.go]: ", err)
	}
	buf := editor.NewBuffer(string(src))
	buf.SetLanguage(editor.LanguageFor(path))
	base := editor.NewBuffer(earlier(string(src)))
	a := &app{
		name: filepath.Base(path),
		code: &editor.State{Buffer: buf, DiffBase: base},
		diff: &editor.DiffState{Old: base, New: buf},
	}
	a.code.SetDiagnostics(buf.Version(), problems(string(src)))
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Minimap",
			Width:   1100,
			Height:  760,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
