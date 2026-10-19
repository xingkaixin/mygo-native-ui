// Package trace shows what an agent does, step by step, as AI chat apps
// show it under a reply: each step an icon of how it goes, a title, a
// detail, and what it holds, which a click on it opens and closes.
//
//	trace.Reasoning(c, &msg.thought, msg.thinking, msg.streaming)
//	trace.Tool(c, trace.Call{Name: "read_file", Input: `{"path":"go.mod"}`, Status: trace.Done, Output: mod}, &msg.open[0])
//	trace.Step(c, trace.Running, "Explore the repository", "3 tools", &msg.open[1], func() {
//		... // a subagent's own steps, as nested Steps and Tools
//	})
//
// A chain of thought, a task or a subagent is Steps in a column, nested
// in one another as deep as the agent's work goes.
package trace

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"

	"github.com/xingkaixin/mygo-native-ui/ui/shimmer"
)

// Status is how a step goes.
type Status uint8

const (
	Pending Status = iota
	Running
	Done
	Failed
)

// Step creates a step: an icon of status, title, which shimmers while the
// step runs, and detail muted after it, and below them, while *open is
// true, what body builds, along a line from the icon. A click on the title
// opens and closes the step; with body nil, there is nothing to open.
func Step(c *ui.Context, status Status, title, detail string, open *bool, body func()) ui.Element {
	t := c.Theme()
	icon := t.FontSize + 2
	return ui.Column(c).Children(func() {
		head := ui.ButtonBase(c).Justify(ui.Start).Gap(t.Space(2)).Padding(t.Space(1), 0).Label(title)
		if body != nil {
			head.Expanded(*open)
			if head.Clicked() {
				*open = !*open
			}
		}
		head.Children(func() {
			Icon(c, status, icon)
			if status == Running {
				shimmer.Text(c, title)
			} else {
				ui.Text(c, title).SingleLine().Shrink(1)
			}
			if detail != "" {
				ui.Text(c, detail).TextColor(t.TextMuted).SingleLine().Shrink(1)
			}
			if body != nil && (head.Hovered() || *open) {
				chevron(c, *open)
			}
		})
		if body != nil && *open {
			ui.Column(c).Gap(t.Space(1)).Padding(t.Space(1), 0, t.Space(2), icon+t.Space(2)).
				Draw(func(p *ui.Painter, r ui.Rect) {
					p.Fill(ui.Rect{X: r.X + icon/2 - 0.5, Y: r.Y, W: 1, H: r.H}, t.Border, 0)
				}).Children(body)
		}
	})
}

// Icon shows status in a square size wide: a circle pending, a spinner
// running, a check done, and a cross failed.
func Icon(c *ui.Context, status Status, size float32) {
	t := c.Theme()
	if status == Running {
		ui.Row(c).Size(size, size).Justify(ui.Center).AlignItems(ui.Center).Children(func() {
			ui.Spinner(c).Label("Running")
		})
		return
	}
	ui.Box(c).Size(size, size).Draw(func(p *ui.Painter, r ui.Rect) {
		cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W*0.28
		switch status {
		case Pending:
			var circle ui.Path
			circle.Circle(cx, cy, r.W*0.36)
			p.StrokePath(&circle, 1.5, t.TextMuted)
		case Done:
			p.StrokePolyline([][2]float32{{cx - k, cy}, {cx - k/3, cy + k*0.7}, {cx + k, cy - k*0.7}}, 1.75, t.Success)
		case Failed:
			p.StrokePolyline([][2]float32{{cx - k, cy - k}, {cx + k, cy + k}}, 1.75, t.Danger)
			p.StrokePolyline([][2]float32{{cx + k, cy - k}, {cx - k, cy + k}}, 1.75, t.Danger)
		}
	})
}

// chevron shows an arrow pointing right, or down when open.
func chevron(c *ui.Context, open bool) {
	t := c.Theme()
	ui.Box(c).Size(t.Space(3), t.Space(3)).Draw(func(p *ui.Painter, r ui.Rect) {
		cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W/5
		points := [][2]float32{{cx - k/2, cy - k}, {cx + k/2, cy}, {cx - k/2, cy + k}}
		if open {
			points = [][2]float32{{cx - k, cy - k/2}, {cx, cy + k/2}, {cx + k, cy - k/2}}
		}
		p.StrokePolyline(points, 1.5, t.TextMuted)
	})
}

// Thought is what a Reasoning keeps: when the thinking began and ended,
// and whether it is open and the user opened or closed it. Keep it with
// the message, one for each.
type Thought struct {
	start, end time.Time
	open, set  bool
}

// Reasoning creates a step of the model's thinking, text, Markdown: open
// while thinking, its title "Thinking…" shimmering with the seconds it
// has taken, and closed once done, titled with how long it took, unless
// the user opened or closed it.
func Reasoning(c *ui.Context, th *Thought, text string, thinking bool) ui.Element {
	t := c.Theme()
	now := c.Now()
	switch {
	case thinking && th.start.IsZero():
		th.start = now
	case !thinking && !th.start.IsZero() && th.end.IsZero():
		th.end = now
	}
	if !th.set {
		th.open = thinking
	}
	was := th.open
	title, detail, status := "Thought", "", Done
	switch {
	case thinking:
		title, detail, status = "Thinking…", elapsed(now.Sub(th.start)), Running
		c.After(time.Second - now.Sub(th.start)%time.Second)
	case !th.start.IsZero():
		title = "Thought for " + elapsed(th.end.Sub(th.start))
	}
	theme := markdown.DefaultTheme(t)
	theme.Text = t.TextMuted
	o := &markdown.Options{Theme: theme}
	step := Step(c, status, title, detail, &th.open, func() {
		if thinking {
			markdown.Stream(c, text, o)
		} else {
			markdown.View(c, text, o)
		}
	})
	if th.open != was {
		th.set = true
	}
	return step
}

// elapsed formats d as "4s" or "1m 4s".
func elapsed(d time.Duration) string {
	s := int(d.Seconds())
	if s < 60 {
		return fmt.Sprintf("%ds", s)
	}
	return fmt.Sprintf("%dm %ds", s/60, s%60)
}

// Call is a tool call of an agent: the tool's name, its input and output,
// JSON or text, how it goes, and the error it failed with.
type Call struct {
	Name          string
	Input, Output string
	Status        Status
	Err           string
}

// Tool creates a step of call, titled with the tool's name and, as its
// detail, the first string of its input, as a command or a path; open,
// it shows the input and the output, or the error, JSON indented.
func Tool(c *ui.Context, call Call, open *bool) ui.Element {
	t := c.Theme()
	return Step(c, call.Status, call.Name, summary(call.Input), open, func() {
		block(c, "Input", call.Input, t.Text)
		switch call.Status {
		case Done:
			block(c, "Output", call.Output, t.Text)
		case Failed:
			block(c, "Error", call.Err, t.Danger)
		}
	})
}

// block shows text, JSON indented, under label, in a box that scrolls
// past a few lines.
func block(c *ui.Context, label, text string, color ui.Color) {
	t := c.Theme()
	if text == "" {
		return
	}
	var b bytes.Buffer
	if json.Indent(&b, []byte(text), "", "  ") == nil {
		text = b.String()
	}
	ui.Text(c, label).FontSize(t.FontSize - 2).Bold().TextColor(t.TextMuted)
	ui.Scroll(c).MaxHeight(t.Space(50)).Radius(t.Space(1.5)).Background(t.Surface).Children(func() {
		ui.Text(c, text).Font("monospace").FontSize(t.FontSize - 1).TextColor(color).Selectable().Padding(t.Space(2))
	})
}

// summary returns the first string among the values of input, a JSON
// object, in their order, on one line, or "".
func summary(input string) string {
	dec := json.NewDecoder(strings.NewReader(input))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
		return ""
	}
	for dec.More() {
		if _, err := dec.Token(); err != nil {
			return ""
		}
		var v any
		if dec.Decode(&v) != nil {
			return ""
		}
		if s, ok := v.(string); ok {
			first, _, _ := strings.Cut(s, "\n")
			return first
		}
	}
	return ""
}
