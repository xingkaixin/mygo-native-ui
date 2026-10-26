// Question is an agent planning a settings page and asking two things on
// the way: which layout, each previewed beside the options, and which
// sections, several at once. Answer with the pointer, or the arrows, the
// number keys, Space and Enter, or type an answer of your own.
//
//	go run ./examples/question
package main

import (
	"log"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/question"
)

type app struct {
	layout, sections question.Answer
}

var sections = question.Question{
	Header: "Sections",
	Text:   "Which sections should it have?",
	Multi:  true,
	Options: []question.Option{
		{Label: "Account", Description: "Name, email and password"},
		{Label: "Notifications", Description: "Email and push, by kind"},
		{Label: "Billing", Description: "Plan, payment method and invoices"},
		{Label: "Integrations", Description: "Connected apps and API keys"},
	},
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	layout := layoutQuestion(c)
	ui.Scroll(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Padding(t.Space(6)).Gap(t.Space(4)).MaxWidth(t.Space(190)).Children(func() {
			say(c, "I'll build the settings page. Two questions first.")
			question.Ask(c.Key("layout"), &a.layout, layout)
			if !a.layout.Done {
				return
			}
			question.Ask(c.Key("sections"), &a.sections, sections)
			if !a.sections.Done {
				return
			}
			say(c, "Building a "+strings.ToLower(answer(a.layout, layout))+" settings page with "+answer(a.sections, sections)+".")
		})
	})
}

func say(c *ui.Context, s string) {
	ui.Text(c, s)
}

func answer(a question.Answer, q question.Question) string {
	var parts []string
	for _, i := range a.Chosen {
		parts = append(parts, q.Options[i].Label)
	}
	if a.Other != "" {
		parts = append(parts, a.Other)
	}
	return strings.Join(parts, ", ")
}

// layoutQuestion asks for a layout, each option previewed as a
// wireframe.
func layoutQuestion(c *ui.Context) question.Question {
	t := c.Theme()
	block := func(w, h float32) ui.Element {
		return ui.Box(c).Width(w).Height(h).Radius(t.Space(1)).Background(t.Text.Alpha(0.1))
	}
	lines := func(n int) {
		ui.Column(c).Gap(t.Space(2)).Grow(1).Children(func() {
			for i := range n {
				block(0, t.Space(3)).Width(t.Space(float32(40 - 6*(i%3))))
			}
		})
	}
	sidebar := func() {
		ui.Row(c).Gap(t.Space(3)).Grow(1).Children(func() {
			ui.Column(c).Gap(t.Space(1.5)).Width(t.Space(18)).Padding(t.Space(2)).Radius(t.Space(1.5)).Background(t.Text.Alpha(0.05)).Children(func() {
				for i := range 4 {
					b := block(0, t.Space(3))
					if i == 0 {
						b.Background(t.Accent.Alpha(0.5))
					}
				}
			})
			lines(5)
		})
	}
	tabs := func() {
		ui.Column(c).Gap(t.Space(3)).Children(func() {
			ui.Row(c).Gap(t.Space(1.5)).Children(func() {
				for i := range 4 {
					b := block(t.Space(12), t.Space(4))
					if i == 0 {
						b.Background(t.Accent.Alpha(0.5))
					}
				}
			})
			lines(5)
		})
	}
	page := func() {
		ui.Column(c).Gap(t.Space(4)).Children(func() {
			for range 3 {
				ui.Column(c).Gap(t.Space(2)).Children(func() {
					block(t.Space(20), t.Space(3.5)).Background(t.Text.Alpha(0.2))
					lines(2)
				})
			}
		})
	}
	return question.Question{
		Header: "Layout",
		Text:   "How should the settings page be laid out?",
		Options: []question.Option{
			{Label: "Sidebar", Description: "Sections listed on the left, one shown at a time", Preview: sidebar},
			{Label: "Tabs", Description: "Sections as tabs across the top", Preview: tabs},
			{Label: "One long page", Description: "Every section in a column, with a table of contents", Preview: page},
		},
	}
}

func main() {
	a := &app{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Question",
			Width:   820,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
