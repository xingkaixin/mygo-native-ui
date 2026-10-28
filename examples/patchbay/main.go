// Patchbay patches a small made-up modular synthesizer: a sequencer
// plays an oscillator through a filter and an amplifier shaped by an
// envelope, with an LFO sweeping the filter. Drag from a jack to pull a
// cable, from a patched input to move or drop its cable; the patch
// sheet below lists what goes where.
//
//	go run ./examples/patchbay
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/patchbay"
)

func in(name string) patchbay.Jack  { return patchbay.Jack{Name: name} }
func out(name string) patchbay.Jack { return patchbay.Jack{Name: name, Out: true} }

var modules = []patchbay.Module{
	{Name: "SEQ", Jacks: []patchbay.Jack{in("Clock"), in("Reset"), out("Pitch"), out("Gate")}},
	{Name: "VCO", Jacks: []patchbay.Jack{in("Pitch"), in("FM"), out("Saw"), out("Square"), out("Sine"), out("Tri")}},
	{Name: "LFO", Jacks: []patchbay.Jack{in("Rate"), in("Sync"), out("Tri"), out("Square")}},
	{Name: "VCF", Jacks: []patchbay.Jack{in("In"), in("Cutoff"), in("Res"), in("FM"), out("LP"), out("HP")}},
	{Name: "ADSR", Jacks: []patchbay.Jack{in("Gate"), in("Retrig"), out("Env"), out("Inv")}},
	{Name: "VCA", Jacks: []patchbay.Jack{in("In"), in("CV"), out("Out")}},
	{Name: "OUT", Jacks: []patchbay.Jack{in("Left"), in("Right")}},
}

func port(m, j int) patchbay.Port { return patchbay.Port{Module: m, Jack: j} }

var (
	cables = []patchbay.Cable{
		{From: port(0, 2), To: port(1, 0), Color: 0},
		{From: port(0, 3), To: port(4, 0), Color: 1},
		{From: port(1, 2), To: port(3, 0), Color: 1},
		{From: port(3, 4), To: port(5, 0), Color: 2},
		{From: port(4, 2), To: port(5, 1), Color: 4},
		{From: port(5, 2), To: port(6, 0), Color: 5},
		{From: port(2, 2), To: port(3, 1), Color: 3},
	}
	rack patchbay.State
)

func name(p patchbay.Port) string {
	return modules[p.Module].Name + " " + modules[p.Module].Jacks[p.Jack].Name
}

func view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(6)).Children(func() {
		ui.Text(c, "Patch").Bold().FontSize(t.FontSize + 4)
		patchbay.View(c, &rack, modules, &cables)
		ui.Column(c).Gap(t.Space(1.5)).Children(func() {
			ui.Text(c, "Patch sheet").FontWeight(600).TextColor(t.TextMuted).FontSize(t.FontSize - 1)
			for i, cb := range cables {
				ui.Row(c.Key(i)).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(t.Space(4), t.Space(1.5)).Radius(t.Space(0.75)).Background(patchbay.Color(cb.Color))
					ui.Text(c, name(cb.From)+"  →  "+name(cb.To)).FontSize(t.FontSize - 1)
				})
			}
		})
	})
}

func main() {
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Patchbay",
			Width:   1080,
			Height:  640,
			Content: ui.View(view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
