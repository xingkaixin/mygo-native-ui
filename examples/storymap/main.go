// Storymap plans a made-up recipe app: four activities from finding a
// recipe to cooking it, their steps, and stories cut into three
// releases.
//
//	go run ./examples/storymap
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/storymap"
)

var activities = []storymap.Activity{
	{Name: "Find a recipe", Steps: []string{"Search", "Browse"}},
	{Name: "Plan the week", Steps: []string{"Pick meals", "Shopping list"}},
	{Name: "Cook", Steps: []string{"Follow steps", "Time it"}},
	{Name: "Share", Steps: []string{"Rate", "Send to a friend"}},
}

var releases = []string{"First release", "Second release", "Later"}

type app struct {
	stories []storymap.Story
	board   storymap.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.ScrollBoth(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Padding(t.Space(6)).Children(func() {
			storymap.View(c, &a.board, activities, releases, &a.stories)
		})
	})
}

func main() {
	a := &app{stories: []storymap.Story{
		{Title: "Search recipes by name", Step: 0, Release: 0},
		{Title: "Filter by what's in the fridge", Step: 0, Release: 1},
		{Title: "Search by diet: vegan, gluten-free", Step: 0, Release: 2},
		{Title: "Browse by cuisine", Step: 1, Release: 0},
		{Title: "Seasonal picks on the home page", Step: 1, Release: 1},
		{Title: "Add a recipe to a day", Step: 2, Release: 0},
		{Title: "Drag meals between days", Step: 2, Release: 1},
		{Title: "Suggest a week from favorites", Step: 2, Release: 2},
		{Title: "List ingredients for the week", Step: 3, Release: 0},
		{Title: "Merge the same ingredient", Step: 3, Release: 0},
		{Title: "Sort the list by aisle", Step: 3, Release: 1},
		{Title: "Order groceries online", Step: 3, Release: 2},
		{Title: "One step at a time, large text", Step: 4, Release: 0},
		{Title: "Keep the screen awake", Step: 4, Release: 1},
		{Title: "Hands-free: next step by voice", Step: 4, Release: 2},
		{Title: "Timers from the step's text", Step: 5, Release: 1},
		{Title: "Rate a recipe with stars", Step: 6, Release: 1},
		{Title: "Notes on a recipe for next time", Step: 6, Release: 2},
		{Title: "Share a link", Step: 7, Release: 0},
	}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Story map",
			Width:   1560,
			Height:  760,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
