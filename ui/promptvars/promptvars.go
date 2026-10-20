// Package promptvars is a prompt template editor, as model playgrounds
// have one: the template, its {{variables}} highlighted as it is typed; a
// field for each variable, in the order they first appear, taking a value
// of several lines; and a preview of the prompt with the values filled in,
// which Copy copies. A variable without a value shows in the warning
// color in the template and on its field, and as a placeholder in the
// preview. Fill fills a template, as the prompt to send.
//
//	promptvars.View(c, &app.template, &app.values).Grow(1)
//	if ui.PrimaryButton(c, "Run").Clicked() {
//		app.run(promptvars.Fill(app.template, app.values))
//	}
package promptvars

import (
	"fmt"
	"regexp"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

// variable is a variable as a template writes it, "{{name}}", with spaces
// inside the braces or without.
var variable = regexp.MustCompile(`\{\{ *([\w.-]+) *\}\}`)

// copied is how long Copy shows Copied.
const copied = 1500 * time.Millisecond

// Fill returns template with each variable replaced by its value in
// values; a variable without a value, or with an empty one, stays as
// written.
func Fill(template string, values map[string]string) string {
	return variable.ReplaceAllStringFunc(template, func(m string) string {
		if v := values[variable.FindStringSubmatch(m)[1]]; v != "" {
			return v
		}
		return m
	})
}

// View creates the editor of *template and of the values of its
// variables, *values, which it makes when nil: the template and the fields
// beside the preview. Give it a size, or Grow; the fields and the preview
// scroll when they do not fit. Values of variables no longer in the
// template stay in *values.
func View(c *ui.Context, template *string, values *map[string]string) ui.Element {
	t := c.Theme()
	if *values == nil {
		*values = map[string]string{}
	}
	vals := *values
	warn := warning(t)
	var matches [][]int
	return ui.Row(c).Gap(t.Space(6)).AlignItems(ui.Stretch).Children(func() {
		ui.Column(c).Grow(1).Gap(t.Space(3)).Children(func() {
			heading(c, "Template", func() {})
			area := ui.TextArea(c, template).Font("monospace").Lines(6, 14).Label("Template")
			matches = variable.FindAllStringSubmatchIndex(*template, -1)
			area.TextRanges(ranges(*template, matches, vals, t.Accent, warn)...)
			names := names(*template, matches)
			empty := 0
			for _, n := range names {
				if vals[n] == "" {
					empty++
				}
			}
			heading(c, "Variables", func() {
				if empty > 0 {
					ui.Text(c, fmt.Sprintf("%d of %d without a value", empty, len(names))).
						FontSize(t.FontSize - 1).TextColor(warn)
				}
			}).Margin(t.Space(2), 0, 0, 0)
			ui.Scroll(c).Grow(1).Children(func() {
				ui.Column(c).Gap(t.Space(4)).Children(func() {
					if len(names) == 0 {
						ui.Text(c, "Write {{name}} in the template to add a variable.").TextColor(t.TextMuted)
					}
					for _, n := range names {
						field(c.Key(n), n, vals, warn)
					}
				})
			})
		})
		ui.Divider(c).Vertical()
		col := ui.Column(c).Grow(1).Gap(t.Space(3))
		at := ui.Local(col, "promptvars.copied", func() time.Time { return time.Time{} })
		col.Children(func() {
			heading(c, "Preview", func() {
				label := "Copy"
				if left := copied - c.Now().Sub(*at); left > 0 {
					c.After(left)
					label = "Copied"
				}
				if ui.Button(c, label).Clicked() {
					c.WriteClipboard(Fill(*template, vals))
					*at = c.Now()
				}
			}).Justify(ui.SpaceBetween)
			ui.Scroll(c).Grow(1).Padding(t.Space(1.5), t.Space(2.5)).Radius(t.Radius).Border(1, t.Border).Children(func() {
				ui.RichText(c, preview(*template, matches, vals, warn)...).Selectable()
			})
		})
	})
}

// warning is the color of a variable without a value, darker than the
// theme's Warning in a light theme for text on white to read.
func warning(t *ui.Theme) ui.Color {
	if t.Dark {
		return t.Warning
	}
	return ui.Oklch(0.58, 0.15, 60)
}

// heading shows the title of a section, and what extra builds after it, in
// a row as high as a button, so that the sections' titles line up.
func heading(c *ui.Context, title string, extra func()) ui.Element {
	t := c.Theme()
	return ui.Row(c).Height(t.Space(7)).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
		ui.Text(c, title).Bold()
		extra()
	})
}

// names returns the names of the variables matched in template, in the
// order they first appear.
func names(template string, matches [][]int) []string {
	var out []string
	seen := map[string]bool{}
	for _, m := range matches {
		n := template[m[2]:m[3]]
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	return out
}

// field shows the name of variable n above a text area of its value in
// vals, marked while it has none.
func field(c *ui.Context, n string, vals map[string]string, warn ui.Color) {
	t := c.Theme()
	v := vals[n]
	color := t.Accent
	if v == "" {
		color = warn
	}
	ui.Column(c).Gap(t.Space(1.5)).Children(func() {
		ui.Row(c).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, n).Font("monospace").FontSize(t.FontSize - 1).FontWeight(600).TextColor(color)
			if v == "" {
				ui.Text(c, "No value").FontSize(t.FontSize - 2).TextColor(warn)
			}
		})
		area := ui.TextArea(c, &v).Lines(1, 8).MinHeight(0).Label(n)
		if v == "" {
			area.Border(1, warn.Alpha(0.6))
		}
		if area.Changed() {
			vals[n] = v
		}
	})
}

// ranges returns the ranges of template to color: each variable in the
// accent color, or warn without a value, its braces fainter than its name.
func ranges(template string, matches [][]int, vals map[string]string, accent, warn ui.Color) []ui.TextRange {
	var out []ui.TextRange
	at, n := 0, 0
	// The offsets come in order, so the runes are counted on from the last.
	runes := func(b int) int {
		n += utf8.RuneCountInString(template[at:b])
		at = b
		return n
	}
	for _, m := range matches {
		color := accent
		if vals[template[m[2]:m[3]]] == "" {
			color = warn
		}
		start, nameStart, nameEnd, end := runes(m[0]), runes(m[2]), runes(m[3]), runes(m[1])
		out = append(out,
			ui.TextRange{Start: start, End: end, Color: color.Alpha(0.55)},
			ui.TextRange{Start: nameStart, End: nameEnd, Color: color, Weight: 600})
	}
	return out
}

// preview returns the spans of template with the values in vals filled
// in, and a placeholder for each variable without one.
func preview(template string, matches [][]int, vals map[string]string, warn ui.Color) []ui.Span {
	var out []ui.Span
	at := 0
	for _, m := range matches {
		out = append(out, ui.Span{Text: template[at:m[0]]})
		n := template[m[2]:m[3]]
		if v := vals[n]; v != "" {
			out = append(out, ui.Span{Text: v})
		} else {
			out = append(out, ui.Span{Text: n, Font: "monospace", Color: warn, Background: warn.Alpha(0.15), Radius: 4})
		}
		at = m[1]
	}
	return append(out, ui.Span{Text: template[at:]})
}
