// Radialmenu picks a drawing tool and its color from a pie menu: press
// the right button on the canvas and stroke toward a tool, or let go in
// the middle and click one. Shapes and Color open rings of their own.
//
//	go run ./examples/radialmenu
package main

import (
	"log"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/radialmenu"
)

func icon(body string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">` + body + `</svg>`))
}

var colors = map[string]ui.Color{
	"Red": ui.Hex("#e5484d"), "Orange": ui.Hex("#f76b15"), "Green": ui.Hex("#30a46c"),
	"Blue": ui.Hex("#0090ff"), "Purple": ui.Hex("#8e4ec6"), "Ink": ui.Hex("#1c2024"),
}

var tools = []radialmenu.Item{
	{Label: "Pen", Icon: icon(`<path d="M4 20l4-1 11-11-3-3L5 16z"/>`)},
	{Label: "Shapes", Icon: icon(`<rect x="3" y="3" width="10" height="10" rx="1"/><circle cx="16" cy="16" r="5"/>`), Items: []radialmenu.Item{
		{Label: "Rectangle"}, {Label: "Ellipse"}, {Label: "Line"}, {Label: "Arrow"},
	}},
	{Label: "Text", Icon: icon(`<path d="M5 6V4h14v2M12 4v16M9 20h6"/>`)},
	{Label: "Color", Icon: icon(`<circle cx="12" cy="12" r="9"/><circle cx="12" cy="12" r="3" fill="currentColor"/>`), Items: []radialmenu.Item{
		{Label: "Red"}, {Label: "Orange"}, {Label: "Green"}, {Label: "Blue"}, {Label: "Purple"}, {Label: "Ink"},
	}},
	{Label: "Eraser", Icon: icon(`<path d="M8 20h12M5 15l9-9 5 5-9 9H8z"/>`)},
	{Label: "Fill", Disabled: true},
	{Label: "Select", Icon: icon(`<path d="M5 3l14 8-6 2-2 6z"/>`)},
	{Label: "Hand", Icon: icon(`<path d="M8 13V5a1.5 1.5 0 0 1 3 0v6M11 10V4a1.5 1.5 0 0 1 3 0v6M14 10V5.5a1.5 1.5 0 0 1 3 0V14a6 6 0 0 1-6 6h-1a6 6 0 0 1-5-3l-2-4a1.5 1.5 0 0 1 2.5-1.5L8 13"/>`)},
}

type app struct {
	tool, color string
	log         []string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(5)).Gap(t.Space(4)).Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, a.tool).Bold()
			ui.Box(c).Size(t.Space(3), t.Space(3)).Radius(t.Space(1.5)).Background(colors[a.color])
			ui.Text(c, a.color).TextColor(t.TextMuted)
		})
		canvas := ui.Column(c).Grow(1).Radius(t.Radius*2).Border(1, t.Border).Background(t.Surface).
			AlignItems(ui.Center).Justify(ui.Center).Gap(t.Space(2)).Label("Canvas")
		if path, ok := radialmenu.Attach(c, canvas, tools).Chosen(); ok {
			a.use(path)
		}
		canvas.Children(func() {
			ui.Text(c, "Press the right button here and stroke toward a tool").TextColor(t.TextMuted)
			for _, line := range a.log {
				ui.Text(c, line).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
			}
		})
	})
}

func (a *app) use(path []int) {
	it := tools[path[0]]
	names := []string{it.Label}
	for _, i := range path[1:] {
		it = it.Items[i]
		names = append(names, it.Label)
	}
	if names[0] == "Color" {
		a.color = it.Label
	} else {
		a.tool = it.Label
	}
	a.log = append([]string{strings.Join(names, " › ")}, a.log...)[:min(len(a.log)+1, 6)]
}

func main() {
	a := &app{tool: "Pen", color: "Ink"}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Radialmenu",
			Width:   760,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
