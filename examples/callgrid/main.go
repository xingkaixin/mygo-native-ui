// Callgrid holds a stand-up of nine people, their cameras drawn as
// silhouettes and their voices taking turns: switch between the grid and
// the speaker layout, share your screen, mute yourself or raise your
// hand, and double-click someone to pin them.
//
//	go run ./examples/callgrid
package main

import (
	"log"
	"math"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/callgrid"
)

var names = []string{"You", "Amara Okafor", "Lukas Brandt", "Mei Tanaka", "Diego Ramos", "Sofia Rossi", "Noah Klein", "Priya Nair", "Elif Kaya"}

// camera returns a camera showing a person against a wall of hue.
func camera(hue float32, start time.Time) func(p *ui.Painter, r ui.Rect) {
	return func(p *ui.Painter, r ui.Rect) {
		p.FillGradient(r, ui.LinearGradient{From: ui.Oklch(0.55, 0.06, hue), To: ui.Oklch(0.3, 0.05, hue+40), Angle: 160, Oklab: true}, 0)
		sway := float32(math.Sin(p.Now().Sub(start).Seconds()*1.3+float64(hue))) * r.W * 0.01
		cx, s := r.X+r.W/2+sway, r.H
		skin := ui.Oklch(0.72, 0.06, 50+hue/8)
		shirt := ui.Oklch(0.4, 0.09, hue+180)
		var body ui.Path
		body.MoveTo(cx-s*0.42, r.Y+r.H).CubeTo(cx-s*0.4, r.Y+r.H*0.72, cx-s*0.22, r.Y+r.H*0.68, cx, r.Y+r.H*0.68).
			CubeTo(cx+s*0.22, r.Y+r.H*0.68, cx+s*0.4, r.Y+r.H*0.72, cx+s*0.42, r.Y+r.H).Close()
		p.FillPath(&body, shirt)
		var head ui.Path
		p.FillPath(head.Circle(cx, r.Y+r.H*0.45, s*0.17), skin)
		var hair ui.Path
		hair.MoveTo(cx-s*0.18, r.Y+r.H*0.43).CubeTo(cx-s*0.18, r.Y+r.H*0.22, cx+s*0.18, r.Y+r.H*0.22, cx+s*0.18, r.Y+r.H*0.43).
			CubeTo(cx+s*0.1, r.Y+r.H*0.34, cx-s*0.1, r.Y+r.H*0.34, cx-s*0.18, r.Y+r.H*0.43).Close()
		p.FillPath(&hair, ui.Oklch(0.25, 0.03, hue))
	}
}

// screen draws a shared editor window.
func screen(p *ui.Painter, r ui.Rect) {
	p.Fill(r, ui.RGB(30, 31, 36), 0)
	p.Fill(ui.Rect{X: r.X, Y: r.Y, W: r.W, H: 28}, ui.RGB(44, 45, 52), 0)
	for i, c := range []ui.Color{ui.RGB(255, 95, 87), ui.RGB(254, 188, 46), ui.RGB(40, 200, 64)} {
		p.Fill(ui.Rect{X: r.X + 12 + float32(i)*18, Y: r.Y + 9, W: 10, H: 10}, c, 5)
	}
	lines := []string{"func standup(team []Person) {", "    for _, p := range team {", "        p.Share(yesterday, today)", "        if p.Blocked() {", "            help(p)", "        }", "    }", "}"}
	for i, l := range lines {
		p.RichText(r.X+24, r.Y+48+float32(i)*22, 0, ui.Span{Text: l, Font: "monospace", Size: 15, Color: ui.RGB(212, 214, 222)})
	}
}

type app struct {
	call    callgrid.State
	start   time.Time
	sharing bool
	muted   bool
	hand    bool
	count   int
}

func (a *app) people(now time.Time) []callgrid.Participant {
	t := now.Sub(a.start).Seconds()
	// The others take turns of four seconds to speak, one at a time.
	turn := 1 + int(t/4)%(a.count-1)
	var out []callgrid.Participant
	for i := range a.count {
		hue := float32(i) * 40
		p := callgrid.Participant{Name: names[i], Color: ui.Oklch(0.55, 0.12, hue)}
		if i != 3 && i != 6 {
			p.Video = camera(hue, a.start)
		}
		if i == turn {
			p.Level = float32(0.3 + 0.6*math.Abs(math.Sin(t*7)*math.Sin(t*2.3)))
		}
		p.Muted = i == 5 || i == 7
		p.Hand = i == 4
		if i == 0 {
			p.Muted, p.Hand = a.muted, a.hand
			if a.sharing {
				p.Screen = screen
			}
		}
		out = append(out, p)
	}
	return out
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	c.AnimationFrame()
	ui.Column(c).Fill().Background(ui.RGB(16, 16, 18)).Children(func() {
		callgrid.View(c, &a.call, a.people(c.Now())).Grow(1)
		ui.Row(c).Padding(t.Space(3), t.Space(4)).Gap(t.Space(3)).AlignItems(ui.Center).Background(t.Surface).Children(func() {
			layout := int(a.call.Layout)
			if ui.Segmented(c, &layout, "Grid", "Speaker").Label("Layout").Changed() {
				a.call.Layout = callgrid.Layout(layout)
			}
			people := float64(a.count)
			ui.NumberInput(c, &people, 2, 9, 1).Label("People")
			a.count = int(people)
			ui.Spacer(c)
			ui.Checkbox(c, &a.muted, "Mute")
			ui.Checkbox(c, &a.hand, "Raise hand")
			ui.Checkbox(c, &a.sharing, "Share screen")
		})
	})
}

func main() {
	a := &app{start: time.Now(), count: 9}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Stand-up",
			Width:   1080,
			Height:  700,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
