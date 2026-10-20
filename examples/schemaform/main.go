// Schemaform calls the tools of an MCP server, as MCP inspectors do: the
// form of the tool chosen, built from its input schema, beside the
// arguments it makes, and Call, which shows what is wrong first.
//
//	go run ./examples/schemaform
package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/schemaform"
)

type tool struct {
	name, about string
	schema      *schemaform.Schema
	form        schemaform.State
	args        any
}

var tools = []struct{ name, about, schema string }{
	{"create_issue", "Opens an issue in a repository.", `{
		"type": "object",
		"properties": {
			"repo": {"type": "string", "title": "Repository", "description": "As owner/name.", "pattern": "^[\\w.-]+/[\\w.-]+$"},
			"title": {"type": "string", "title": "Title", "minLength": 3, "maxLength": 80},
			"body": {"type": "string", "title": "Body"},
			"priority": {"type": "string", "title": "Priority", "enum": ["low", "medium", "high"], "default": "medium"},
			"labels": {"type": "array", "title": "Labels", "items": {"type": "string"}, "maxItems": 4},
			"estimate": {"type": "integer", "title": "Estimate", "description": "In hours.", "minimum": 1, "maximum": 40},
			"draft": {"type": "boolean", "title": "Draft", "default": false},
			"assignee": {"type": "object", "title": "Assignee", "properties": {
				"login": {"type": "string", "title": "Login"},
				"notify": {"type": "boolean", "title": "Notify", "default": true}
			}}
		},
		"required": ["repo", "title"]
	}`},
	{"search_issues", "Finds issues by text.", `{
		"type": "object",
		"properties": {
			"query": {"type": "string", "title": "Query", "minLength": 2},
			"state": {"type": "string", "title": "State", "enum": ["open", "closed", "all"], "default": "open"},
			"limit": {"type": "integer", "title": "Limit", "minimum": 1, "maximum": 100, "default": 20}
		},
		"required": ["query"]
	}`},
}

type app struct {
	tools  []*tool
	chosen int
	result string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	tl := a.tools[a.chosen]
	ui.Row(c).Fill().Background(t.Background).AlignItems(ui.Stretch).Children(func() {
		ui.Column(c).Width(t.Space(52)).Padding(t.Space(3)).Gap(t.Space(1)).Children(func() {
			ui.Text(c, "Tools").Bold().Padding(t.Space(1), t.Space(2))
			for i, x := range a.tools {
				b := ui.ButtonBase(c.Key(i)).Padding(t.Space(1.5), t.Space(2)).Radius(t.Radius).Justify(ui.Start).Label(x.name)
				if i == a.chosen {
					b.Background(t.Accent.Alpha(0.12))
				}
				if b.Clicked() {
					a.chosen, a.result = i, ""
				}
				b.Children(func() {
					ui.Column(c).Gap(t.Space(0.5)).Children(func() {
						ui.Text(c, x.name).Font("monospace")
						ui.Text(c, x.about).FontSize(t.FontSize - 2).TextColor(t.TextMuted)
					})
				})
			}
		})
		ui.Divider(c).Vertical()
		ui.Scroll(c).Grow(1).Shrink(1).Children(func() {
			ui.Column(c).Padding(t.Space(6)).Gap(t.Space(5)).Children(func() {
				ui.Text(c, tl.name).Font("monospace").Bold().FontSize(t.FontSize + 4)
				if schemaform.View(c, &tl.form, tl.schema, &tl.args).Changed() {
					a.result = ""
				}
				ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
					if ui.PrimaryButton(c, "Call").Clicked() {
						if p := schemaform.Validate(tl.schema, tl.args); len(p) > 0 {
							tl.form.ShowErrors()
							a.result = fmt.Sprintf("%d problems to fix", len(p))
						} else {
							a.result = "Called " + tl.name
						}
					}
					if a.result != "" {
						ui.Text(c, a.result).TextColor(t.TextMuted)
					}
				})
			})
		})
		ui.Divider(c).Vertical()
		ui.Column(c).Width(t.Space(80)).Padding(t.Space(4)).Gap(t.Space(2)).Background(t.Surface).Children(func() {
			ui.Text(c, "Arguments").Bold()
			out, _ := json.MarshalIndent(tl.args, "", "  ")
			ui.Text(c, string(out)).Font("monospace").FontSize(t.FontSize - 1)
		})
	})
}

func main() {
	a := &app{}
	for _, x := range tools {
		s, err := schemaform.Parse([]byte(x.schema))
		if err != nil {
			log.Fatal(x.name, ": ", err)
		}
		a.tools = append(a.tools, &tool{name: x.name, about: x.about, schema: s})
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Schema form",
			Width:   1180,
			Height:  760,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
