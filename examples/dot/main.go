// Dot draws a graph from the DOT source beside it, as it is typed: a
// build pipeline in clusters, from a push to production; a media player's
// states from left to right; or an office network, whose links have no
// direction. Clicking a node names it under the graph.
//
//	go run ./examples/dot
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/dot"
)

var samples = []struct {
	name, src string
}{
	{"Build pipeline", `digraph pipeline {
  // A release, from a push to production
  node [shape=box, style=rounded]

  push [label="git push", shape=ellipse]
  push -> {lint compile}

  subgraph cluster_build {
    label="Build"
    lint
    compile [label="compile\ngo build ./..."]
    unit [label="unit tests"]
    e2e [label="end-to-end\ntests"]
    compile -> {unit e2e}
  }

  image [label="image\nregistry", shape=cylinder]
  {lint unit e2e} -> image

  subgraph cluster_deploy {
    label="Deploy"
    style=filled
    fillcolor="#dbeafe"
    staging
    smoke [label="smoke tests"]
    approve [label="approved?", shape=diamond]
    staging -> smoke -> approve
  }
  image -> staging

  prod [label="production", style="rounded,filled",
        fillcolor="#16a34a"]
  rollback [shape=note, color="#dc2626",
            fontcolor="#dc2626"]
  approve -> prod [label="yes"]
  approve -> rollback [label="no", style=dashed]
  rollback -> staging [style=dotted,
                       arrowhead=empty]
}`},
	{"State machine", `digraph player {
  // A media player, from opening a file to its end
  rankdir=LR
  node [shape=circle]

  start [shape=point]
  ended [shape=doublecircle]

  start -> idle
  idle -> loading [label="open"]
  loading -> ready [label="loaded"]
  loading -> failed [label="error",
                     color="#dc2626"]
  ready -> playing [label="play"]
  playing -> paused [label="pause"]
  paused -> playing [label="resume"]
  playing -> buffering [label="stall"]
  buffering -> playing [label="filled"]
  playing -> playing [label="seek"]
  playing -> ended [label="finish"]
  failed -> idle [label="retry", style=dashed]
}`},
	{"Network", `graph office {
  // The office network: no edge has a direction
  node [shape=box, style=rounded]

  internet [shape=ellipse, style=dashed]
  internet -- router [label="fiber", penwidth=2]
  router -- firewall -- core
  core [label="core switch", style="rounded,filled",
        fillcolor=lightsteelblue]

  subgraph cluster_servers {
    label="Server room"
    nas [label="NAS", shape=cylinder]
    db [label="database", shape=cylinder]
    web [label="intranet"]
  }
  core -- {nas db web}

  subgraph cluster_floor {
    label="Floor 2"
    { rank=same; ap1 ap2 }
    ap1 [label="access point\neast"]
    ap2 [label="access point\nwest"]
    printer
  }
  core -- {ap1 ap2 printer}
  ap1 -- ap2 [style=dotted, label="mesh"]
}`},
}

type app struct {
	sample, shown int
	source        string
	split         float32
	graph         dot.State
	chosen        string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Padding(t.Space(3), t.Space(4)).Children(func() {
			names := make([]string, len(samples))
			for i, s := range samples {
				names[i] = s.name
			}
			ui.Segmented(c, &a.sample, names...).Label("Sample")
			if a.sample != a.shown {
				a.shown, a.source, a.chosen = a.sample, samples[a.sample].src, ""
			}
		})
		ui.Divider(c)
		ui.Split(c, &a.split, func() {
			ui.Column(c).Fill().Padding(t.Space(3)).Children(func() {
				ui.TextArea(c, &a.source).Font("monospace").Label("Source").Grow(1)
			})
		}, func() {
			ui.Column(c).Fill().Children(func() {
				dot.View(c, &a.graph, a.source, &a.chosen).Grow(1)
				ui.Divider(c)
				ui.Row(c).Gap(t.Space(2)).Padding(t.Space(3), t.Space(4)).AlignItems(ui.Center).Children(func() {
					if a.chosen == "" {
						ui.Text(c, "Click a node to choose it").TextColor(t.TextMuted)
						return
					}
					ui.Text(c, "Chosen").TextColor(t.TextMuted)
					ui.Text(c, a.chosen).Font("monospace").Bold()
				})
			})
		}).Grow(1)
	})
}

func main() {
	a := &app{source: samples[0].src, split: 500}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "DOT",
			Width:   1440,
			Height:  900,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
