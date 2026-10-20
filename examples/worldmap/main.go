// Worldmap shows how many people live in each country, as Natural Earth
// estimates, on a logarithmic scale. Click a country to see its population
// and its rank; scroll to zoom, drag to pan, and double-click to see the
// whole world again.
//
//	go run ./examples/worldmap
package main

import (
	"cmp"
	_ "embed"
	"fmt"
	"log"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/worldmap"
)

// population is Natural Earth's POP_EST of each country, which is in the
// public domain, by ISO 3166-1 alpha-3 code.
//
//go:embed population.csv
var population string

type app struct {
	people map[string]float64
	// rank is each country's place by people, 1 the most.
	rank  map[string]int
	world worldmap.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Gap(t.Space(4)).Padding(t.Space(3), t.Space(4)).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Gap(t.Space(0.5)).Grow(1).Shrink(1).Children(func() {
				ui.Text(c, "World population").Bold().FontSize(t.FontSize + 4)
				ui.Text(c, "Estimates by country, from Natural Earth").TextColor(t.TextMuted).SingleLine()
			})
			ui.Column(c).Gap(t.Space(0.5)).AlignItems(ui.End).Children(func() {
				code, ok := a.world.Chosen()
				if !ok {
					ui.Text(c, "Click a country").TextColor(t.TextMuted)
					return
				}
				ui.Text(c, worldmap.Name(code)).Bold().SingleLine()
				n, ok := a.people[code]
				if !ok {
					ui.Text(c, "No estimate").TextColor(t.TextMuted)
					return
				}
				ui.Text(c, fmt.Sprintf("%s people  ·  #%d of %d", grouped(int64(n)), a.rank[code], len(a.rank))).
					TextColor(t.TextMuted).FontFeatures("tnum").SingleLine()
			})
		})
		ui.Divider(c)
		worldmap.View(c, &a.world, a.people, worldmap.Log, rounded).Grow(1)
	})
}

// rounded formats n as news does: "1.40 billion", "68.2 million", "56,225".
func rounded(n float64) string {
	switch {
	case n >= 1e9:
		return strconv.FormatFloat(n/1e9, 'f', 2, 64) + " billion"
	case n >= 1e6:
		return strconv.FormatFloat(n/1e6, 'f', 1, 64) + " million"
	}
	return grouped(int64(n))
}

// grouped returns n with commas between groups of three digits.
func grouped(n int64) string {
	s := strconv.FormatInt(n, 10)
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}

func main() {
	a := &app{people: map[string]float64{}, rank: map[string]int{}}
	for line := range strings.Lines(population) {
		code, n, _ := strings.Cut(strings.TrimSpace(line), ",")
		if v, err := strconv.ParseFloat(n, 64); err == nil {
			a.people[code] = v
		}
	}
	codes := slices.SortedFunc(maps.Keys(a.people), func(x, y string) int { return cmp.Compare(a.people[y], a.people[x]) })
	for i, code := range codes {
		a.rank[code] = i + 1
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "World Population",
			Width:   1100,
			Height:  680,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
