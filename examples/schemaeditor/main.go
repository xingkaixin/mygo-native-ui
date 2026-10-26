// Schemaeditor edits the parameters of a model's tool that creates
// calendar events, read from a schema with Parse, and shows the schema
// they make beside them as they change.
//
//	go run ./examples/schemaeditor
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"

	"github.com/xingkaixin/mygo-native-ui/ui/schemaeditor"
)

const tool = `{
  "type": "object",
  "properties": {
    "title": {"type": "string", "description": "What the event is"},
    "start": {"type": "string", "description": "When it starts, in RFC 3339"},
    "duration_minutes": {"type": "integer", "description": "How long it lasts"},
    "visibility": {"type": "string", "enum": ["default", "public", "private"]},
    "attendees": {
      "type": "array",
      "description": "Who is invited",
      "items": {
        "type": "object",
        "properties": {
          "email": {"type": "string"},
          "optional": {"type": "boolean", "description": "Whether they may skip it"}
        },
        "required": ["email"]
      }
    }
  },
  "required": ["title", "start"]
}`

type app struct {
	name   string
	desc   string
	params []schemaeditor.Field
	schema *editor.State
	split  float32
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.SplitPercent(c, &a.split, func() {
			ui.Scroll(c).Fill().Children(func() {
				ui.Column(c).Padding(16).Gap(12).Children(func() {
					ui.Field(c, "Tool", func() {
						ui.TextInput(c, &a.name).Font("monospace")
					})
					ui.Field(c, "Description", func() {
						ui.TextInput(c, &a.desc)
					})
					ui.Text(c, "Parameters").Bold()
					if schemaeditor.Editor(c, &a.params).Changed() {
						a.schema.Buffer.Set(string(schemaeditor.JSON(a.params)))
					}
				})
			})
		}, func() {
			editor.View(c, a.schema).Fill()
		})
	})
}

func main() {
	params, err := schemaeditor.Parse([]byte(tool))
	if err != nil {
		log.Fatal(err)
	}
	buf := editor.NewBuffer(string(schemaeditor.JSON(params)))
	buf.SetLanguage(editor.LanguageFor("schema.json"))
	a := &app{
		name:   "create_event",
		desc:   "Creates an event on the user's calendar",
		params: params,
		schema: &editor.State{Buffer: buf, ReadOnly: true, SoftWrap: true, NoActiveLine: true, Label: "Schema"},
		split:  64,
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Tool parameters",
			Width:   1180,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
