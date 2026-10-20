// Package stacktrace shows an error and the stack it came from, as coding
// agents show a crash: the error, and the frames under it, the innermost
// first, those of the app in full and each run of those of the runtime,
// the standard library and dependencies folded into a row, "4 library
// frames", which a click opens. A click on a frame chooses it, for the app
// to open its file at its line:
//
//	tr := stacktrace.Parse(output) // once, not every frame
//	...
//	if f, ok := stacktrace.View(c, tr).Chosen(); ok {
//		app.open(f.File, f.Line)
//	}
//
// Parse reads Go panics, Node.js errors and Python tracebacks.
package stacktrace

import (
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Frame is a call in a stack: the function, the file and the line, and
// whether it is a library's, the runtime's, the standard library's or a
// dependency's, rather than the app's.
type Frame struct {
	Func    string
	File    string
	Line    int
	Library bool
}

// Trace is an error and the frames of the stack it came from, the
// innermost first.
type Trace struct {
	Error  string
	Frames []Frame
}

var (
	goFile   = regexp.MustCompile(`^\t(.+?):(\d+)(?: \+0x[0-9a-f]+)?$`)
	goArgs   = regexp.MustCompile(`\([^()]*\)$`)
	jsFrame  = regexp.MustCompile(`^\s+at (?:(?:async )?(.+?) \()?(.+?):(\d+)(?::\d+)?\)?$`)
	pyFrame  = regexp.MustCompile(`^\s*File "(.+)", line (\d+), in (.+)$`)
	pyHeader = "Traceback (most recent call last):"
)

// Parse reads the error and the stack of text: a Go panic, of its first
// goroutine, an error Node.js printed, or a Python traceback. Text it does
// not know is all error, with no frames.
func Parse(text string) Trace {
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	switch {
	case slices.ContainsFunc(lines, func(l string) bool { return strings.HasPrefix(l, "goroutine ") }):
		return parseGo(lines)
	case slices.ContainsFunc(lines, func(l string) bool { return strings.HasPrefix(l, pyHeader) }):
		return parsePython(lines)
	case slices.ContainsFunc(lines, func(l string) bool { return jsFrame.MatchString(l) }):
		return parseJS(lines)
	}
	return Trace{Error: strings.TrimSpace(text)}
}

func parseGo(lines []string) Trace {
	var tr Trace
	from := max(0, slices.IndexFunc(lines, func(l string) bool {
		return strings.HasPrefix(l, "panic: ") || strings.HasPrefix(l, "fatal error: ")
	}))
	i := from
	for i < len(lines) && !strings.HasPrefix(lines[i], "goroutine ") {
		i++
	}
	// A panic recovered and raised again repeats under the first, indented.
	msg := slices.DeleteFunc(slices.Clone(lines[from:i]), func(l string) bool { return strings.HasPrefix(l, "\tpanic: ") })
	tr.Error = strings.TrimSpace(strings.Join(msg, "\n"))
	tr.Error = strings.TrimPrefix(strings.TrimPrefix(tr.Error, "panic: "), "fatal error: ")
	tr.Error = strings.Replace(tr.Error, " [recovered]", "", 1)
	for i++; i+1 < len(lines) && lines[i] != ""; i += 2 {
		m := goFile.FindStringSubmatch(lines[i+1])
		if m == nil {
			break
		}
		fn := lines[i]
		if rest, ok := strings.CutPrefix(fn, "created by "); ok {
			fn, _, _ = strings.Cut(rest, " in goroutine ")
		}
		fn = goArgs.ReplaceAllString(fn, "")
		line, _ := strconv.Atoi(m[2])
		pkg := fn[:strings.LastIndex(fn, "/")+1]
		first, _, _ := strings.Cut(fn[len(pkg):], ".")
		pkg += first
		// The standard library's import paths have no dot in their first
		// element; dependencies come from the module cache.
		std := pkg != "main" && !strings.Contains(strings.Split(pkg, "/")[0], ".")
		tr.Frames = append(tr.Frames, Frame{
			Func: fn, File: m[1], Line: line,
			Library: std || strings.Contains(m[1], "/pkg/mod/"),
		})
	}
	return tr
}

func parseJS(lines []string) Trace {
	var tr Trace
	var msg []string
	for _, l := range lines {
		m := jsFrame.FindStringSubmatch(l)
		if m == nil {
			if tr.Frames == nil {
				msg = append(msg, l)
			}
			continue
		}
		line, _ := strconv.Atoi(m[3])
		file := m[2]
		tr.Frames = append(tr.Frames, Frame{
			Func: m[1], File: file, Line: line,
			Library: strings.HasPrefix(file, "node:") || strings.Contains(file, "node_modules/"),
		})
	}
	tr.Error = strings.TrimSpace(strings.Join(msg, "\n"))
	return tr
}

func parsePython(lines []string) Trace {
	var tr Trace
	// Of chained exceptions, the last traceback is of the one raised.
	start := 0
	for i, l := range lines {
		if strings.HasPrefix(l, pyHeader) {
			start = i
		}
	}
	var msg []string
	for _, l := range lines[start+1:] {
		if m := pyFrame.FindStringSubmatch(l); m != nil {
			line, _ := strconv.Atoi(m[2])
			tr.Frames = append(tr.Frames, Frame{
				Func: m[3], File: m[1], Line: line,
				Library: strings.Contains(m[1], "site-packages") || strings.Contains(m[1], "/lib/python") ||
					strings.HasPrefix(m[1], "<frozen"),
			})
			continue
		}
		if l != "" && !strings.HasPrefix(l, " ") {
			msg = append(msg, l)
		}
	}
	slices.Reverse(tr.Frames)
	tr.Error = strings.TrimSpace(strings.Join(msg, "\n"))
	return tr
}

// Events are what the user did with a trace in a frame. The zero Events
// report nothing.
type Events struct {
	chosen Frame
	ok     bool
}

// Chosen returns the frame the user clicked.
func (e Events) Chosen() (Frame, bool) { return e.chosen, e.ok }

// View creates the view of tr: the error, which can be selected and
// copied, and the frames, each the function and the file's name and line,
// "store.go:19", its path in a tooltip. The runs of library frames open
// and close with a click on their row.
func View(c *ui.Context, tr Trace) Events {
	t := c.Theme()
	var ev Events
	card := ui.Column(c).Radius(t.Space(2.5)).Border(1, t.Danger.Alpha(0.35)).Clip().Role(ui.RoleGroup).Label("Stack trace")
	open := *ui.Local(card, "stacktrace.open", func() map[int]bool { return map[int]bool{} })
	card.Children(func() {
		ui.Text(c, tr.Error).Font("monospace").FontSize(t.FontSize-1).TextColor(t.Danger).Selectable().
			Padding(t.Space(2.5), t.Space(3)).Background(t.Danger.Alpha(0.06))
		if len(tr.Frames) == 0 {
			return
		}
		ui.Column(c).Padding(t.Space(1)).Children(func() {
			for i := 0; i < len(tr.Frames); {
				j := i + 1
				for j < len(tr.Frames) && tr.Frames[j].Library == tr.Frames[i].Library {
					j++
				}
				ui.Column(c.Key(i)).Children(func() {
					if !tr.Frames[i].Library || open[i] {
						if tr.Frames[i].Library {
							if fold(c, j-i, true).Clicked() {
								open[i] = false
							}
						}
						for k, f := range tr.Frames[i:j] {
							if frame(c.Key(k), f).Clicked() {
								ev.chosen, ev.ok = f, true
							}
						}
						return
					}
					if fold(c, j-i, false).Clicked() {
						open[i] = true
					}
				})
				i = j
			}
		})
	})
	return ev
}

// frame creates the row of f.
func frame(c *ui.Context, f Frame) ui.Element {
	t := c.Theme()
	where := filepath.Base(f.File)
	if f.Line > 0 {
		where += ":" + strconv.Itoa(f.Line)
	}
	name := f.Func
	if name == "" {
		name = "<anonymous>"
	}
	b := ui.ButtonBase(c).Justify(ui.Start).Gap(t.Space(3)).Padding(t.Space(1.25), t.Space(2)).Radius(t.Space(1.5)).
		Label(name + " " + where).Tooltip(f.File + ":" + strconv.Itoa(f.Line))
	if b.Hovered() {
		b.Background(t.Text.Alpha(0.05))
	}
	color := t.Text
	if f.Library {
		color = t.TextMuted
	}
	return b.Children(func() {
		ui.Text(c, name).Font("monospace").FontSize(t.FontSize - 1).TextColor(color).SingleLine().Grow(1).Shrink(1)
		ui.Text(c, where).FontSize(t.FontSize - 1).TextColor(t.TextMuted).SingleLine().Shrink(2)
	})
}

// fold creates the row of a run of n library frames, which open shows
// open.
func fold(c *ui.Context, n int, open bool) ui.Element {
	t := c.Theme()
	label := strconv.Itoa(n) + " library frames"
	if n == 1 {
		label = "1 library frame"
	}
	if open {
		label = "Hide " + label
	}
	b := ui.ButtonBase(c).Justify(ui.Start).Gap(t.Space(1.5)).Padding(t.Space(1), t.Space(2)).Radius(t.Space(1.5)).
		Label(label).Expanded(open)
	if b.Hovered() {
		b.Background(t.Text.Alpha(0.05))
	}
	return b.Children(func() {
		ui.Box(c).Size(t.Space(3), t.Space(3)).Draw(func(p *ui.Painter, r ui.Rect) {
			cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W/5
			points := [][2]float32{{cx - k/2, cy - k}, {cx + k/2, cy}, {cx - k/2, cy + k}}
			if open {
				points = [][2]float32{{cx - k, cy - k/2}, {cx, cy + k/2}, {cx + k, cy - k/2}}
			}
			p.StrokePolyline(points, 1.5, t.TextMuted)
		})
		ui.Text(c, label).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
	})
}
