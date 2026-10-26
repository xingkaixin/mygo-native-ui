// Spanlabel labels the people, organizations, places and dates of two
// short documents, one in English and one in Chinese, some spans already
// labeled and two of them overlapping.
//
//	go run ./examples/spanlabel
package main

import (
	"fmt"
	"log"
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/spanlabel"
)

var labels = []spanlabel.Label{{Name: "PERSON"}, {Name: "ORG"}, {Name: "PLACE"}, {Name: "DATE"}}

type doc struct {
	text  string
	spans []spanlabel.Span
	state spanlabel.State
}

// mark labels the first run of the text that is part.
func (d *doc) mark(part, label string) *doc {
	i := strings.Index(d.text, part)
	start := utf8.RuneCountInString(d.text[:i])
	d.spans = append(d.spans, spanlabel.Span{Start: start, End: start + utf8.RuneCountInString(part), Label: label})
	return d
}

type app struct {
	docs []*doc
	open int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	d := a.docs[a.open]
	ui.Column(c).Fill().Background(t.Background).Padding(20).Gap(16).Children(func() {
		ui.Segmented(c, &a.open, "English", "中文").Label("Document")
		spanlabel.View(c.Key(a.open), &d.state, d.text, labels, &d.spans)
		ui.Divider(c)
		ui.Column(c).Gap(4).Children(func() {
			runes := []rune(d.text)
			for _, sp := range d.spans {
				ui.Text(c, fmt.Sprintf("%-7s %3d–%-3d %s", sp.Label, sp.Start, sp.End, string(runes[sp.Start:sp.End]))).
					Font("monospace").FontSize(12).TextColor(t.TextMuted)
			}
		})
	})
}

func main() {
	en := &doc{text: "On March 4, 2026, Ada Lovelace joined Kettle Labs in Cambridge as head of research. " +
		"She had taught at the University of Cambridge for nine years, and before that advised the " +
		"Royal Society in London on its first computing committee."}
	en.mark("March 4, 2026", "DATE").mark("Ada Lovelace", "PERSON").mark("Kettle Labs", "ORG").
		mark("Cambridge", "PLACE").mark("University of Cambridge", "ORG").mark("Royal Society", "ORG")
	en.spans = append(en.spans, spanlabel.Span{Start: en.spans[4].End - len("Cambridge"), End: en.spans[4].End, Label: "PLACE"})
	zh := &doc{text: "2025年10月，李华在杭州加入了云溪科技，负责北京研发中心的工作。此前她在浙江大学任教六年。"}
	zh.mark("2025年10月", "DATE").mark("李华", "PERSON").mark("杭州", "PLACE")
	a := &app{docs: []*doc{en, zh}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Labels",
			Width:   760,
			Height:  520,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
