// Nodegraph mixes colors in a graph of nodes, as a shader editor does:
// colors to pick, nodes that mix and lighten them, and outputs that show
// what comes out, all worked out again as the graph changes. Add nodes
// with the buttons above it, drag from an output to an input to link
// them, and press Delete to remove the node chosen.
//
//	go run ./examples/nodegraph
package main

import (
	"fmt"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/nodegraph"
)

// kind is what a node does.
type kind uint8

const (
	pick kind = iota
	mix
	lighten
	output
)

// op is a node's data: what it does, the color it picks, and how much it
// mixes or lightens.
type op struct {
	kind   kind
	color  ui.Color
	amount float64
}

// newNode returns a node of kind k at x, y.
func newNode(k kind, x, y float32) nodegraph.Node[op] {
	n := nodegraph.Node[op]{X: x, Y: y, Data: op{kind: k, amount: 0.5}}
	switch k {
	case pick:
		n.Title, n.Outputs, n.Data.color = "Color", []string{"Color"}, ui.Hex("#5b8def")
	case mix:
		n.Title, n.Inputs, n.Outputs = "Mix", []string{"A", "B"}, []string{"Color"}
	case lighten:
		n.Title, n.Inputs, n.Outputs, n.Data.amount = "Lighten", []string{"Color"}, []string{"Color"}, 0.3
	case output:
		n.Title, n.Inputs = "Output", []string{"Color"}
	}
	return n
}

type app struct {
	graph  nodegraph.Graph[op]
	canvas nodegraph.State
}

// value returns the color node i puts out.
func (a *app) value(i int) ui.Color {
	n := a.graph.Nodes[i]
	in := func(k int) ui.Color {
		if l, ok := a.graph.Input(i, k); ok {
			return a.value(l.From)
		}
		return ui.Hex("#808080")
	}
	switch n.Data.kind {
	case pick:
		return n.Data.color
	case mix:
		return in(0).Mix(in(1), float32(n.Data.amount))
	case lighten:
		return in(0).Mix(ui.RGB(255, 255, 255), float32(n.Data.amount))
	}
	return in(0)
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Gap(t.Space(2)).Padding(t.Space(2), t.Space(3)).AlignItems(ui.Center).Children(func() {
			for _, b := range []struct {
				label string
				k     kind
			}{{"Color", pick}, {"Mix", mix}, {"Lighten", lighten}, {"Output", output}} {
				if ui.Button(c, "Add "+b.label).Clicked() {
					x, y := a.canvas.Center()
					a.graph.Nodes = append(a.graph.Nodes, newNode(b.k, x-90, y-60))
				}
			}
			ui.Box(c).Grow(1)
			ui.Text(c, "Drag from an output to an input to link them").FontSize(t.FontSize - 1).TextColor(t.TextMuted)
		})
		ui.Divider(c)
		nodegraph.View(c, &a.canvas, &a.graph, func(n *nodegraph.Node[op]) {
			a.body(c, n)
		}).Grow(1)
	})
}

// body shows the controls of n, and the color it puts out.
func (a *app) body(c *ui.Context, n *nodegraph.Node[op]) {
	t := c.Theme()
	i := 0
	for k := range a.graph.Nodes {
		if &a.graph.Nodes[k] == n {
			i = k
		}
	}
	switch n.Data.kind {
	case pick:
		ui.ColorWell(c, &n.Data.color).Label("Color")
		return
	case mix, lighten:
		ui.Slider(c, &n.Data.amount, 0, 1).Label("Amount").Width(t.Space(40))
	}
	color := a.value(i)
	h := t.Space(6)
	if n.Data.kind == output {
		h = t.Space(24)
	}
	ui.Box(c).Height(h).MinWidth(t.Space(40)).Radius(t.Space(1.5)).Background(color).Border(1, t.Border)
	if n.Data.kind == output {
		ui.Text(c, fmt.Sprintf("#%02X%02X%02X", color.R, color.G, color.B)).Font("monospace").FontSize(t.FontSize - 1).
			TextColor(t.TextMuted).Selectable()
	}
}

func main() {
	a := &app{}
	rose, blue := newNode(pick, 40, 40), newNode(pick, 40, 240)
	rose.Data.color, blue.Data.color = ui.Hex("#ff4d8d"), ui.Hex("#3d7bff")
	a.graph = nodegraph.Graph[op]{
		Nodes: []nodegraph.Node[op]{rose, blue, newNode(mix, 300, 120), newNode(lighten, 560, 150), newNode(output, 820, 110)},
		Links: []nodegraph.Link{
			{From: 0, Out: 0, To: 2, In: 0},
			{From: 1, Out: 0, To: 2, In: 1},
			{From: 2, Out: 0, To: 3, In: 0},
			{From: 3, Out: 0, To: 4, In: 0},
		},
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Node Graph",
			Width:   1120,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
