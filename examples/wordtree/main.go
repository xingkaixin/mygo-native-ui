// Wordtree shows what made-up reviews of a coffee grinder say around a
// phrase. Type a phrase, or click a word in the tree to grow the phrase
// up to it.
//
//	go run ./examples/wordtree
package main

import (
	"log"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/wordtree"
)

const reviews = `The grinder is quiet enough to use before anyone else is up.
The grinder is easy to clean once you find the release button.
The grinder is louder than my old one but much faster.
The grinder is worth every penny.
The grinder is a little messy with static on dry days.
The grinder is quiet and consistent.
The grinder makes a consistent grind for espresso.
The grinder makes a mess on the counter.
The grinder makes my mornings better.
The grinder stopped working after two months.
The grinder stopped working after a year of daily use.
I love how the grinder looks on my counter.
I love how quiet it is.
I love the timer on the grinder.
I wish the grinder had a bigger hopper.
I wish the grinder was easier to clean.
I wish the hopper held more beans.
My old grinder was louder and slower.
My old grinder never made a consistent grind.
After a week the grinder is still as quiet as on the first day.
After two months the grinder stopped grinding fine enough for espresso.
For espresso the grinder is consistent and fast.
For pour over the grinder is great once you find the right setting.
Cleaning the grinder takes five minutes.
Cleaning the burrs is easy.
The burrs are easy to remove and clean.
The burrs are sharp and consistent.
The hopper is too small for a family.
The hopper is easy to fill.
Customer service replaced the grinder in a week.
Customer service was slow to answer.
Overall the grinder is worth every penny.
Overall the grinder is a solid choice for the price.
Overall I love it.`

type app struct{ root string }

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(4)).Children(func() {
		ui.SearchField(c, &a.root).Placeholder("Phrase").Width(t.Space(70))
		ui.ScrollBoth(c).Grow(1).AlignSelf(ui.Stretch).Children(func() {
			wordtree.View(c, &a.root, strings.Split(reviews, "\n"))
		})
	})
}

func main() {
	a := &app{root: "the grinder"}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Word Tree",
			Width:   1000,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
