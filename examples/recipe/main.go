// Recipe bakes a banana bread: scale it, switch to US cups, tick off the
// ingredients and run the timers for browning the butter and baking.
//
//	go run ./examples/recipe
package main

import (
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/recipe"
)

var bread = recipe.Recipe{
	Name:   "Brown butter banana bread",
	Serves: 8,
	Ingredients: []recipe.Ingredient{
		{Amount: 3, Name: "ripe bananas"},
		{Amount: 115, Unit: "g", Name: "butter"},
		{Amount: 150, Unit: "g", Name: "brown sugar"},
		{Amount: 1, Name: "egg"},
		{Amount: 1, Unit: "tsp", Name: "vanilla extract"},
		{Amount: 1, Unit: "tsp", Name: "baking soda"},
		{Amount: 0.25, Unit: "tsp", Name: "salt"},
		{Amount: 190, Unit: "g", Name: "plain flour"},
		{Amount: 60, Unit: "ml", Name: "milk"},
	},
	Steps: []recipe.Step{
		{Text: "Heat the oven to 175°C and line a loaf tin with paper."},
		{Text: "Melt the butter in a pan and let it foam and brown, swirling, until it smells nutty.", Timer: 4 * time.Minute},
		{Text: "Mash the bananas, then stir in the brown butter, sugar, egg and vanilla."},
		{Text: "Fold in the baking soda, salt and flour, then the milk, until no flour shows."},
		{Text: "Bake until a skewer in the middle comes out clean.", Timer: 55 * time.Minute},
		{Text: "Cool in the tin before turning it out to slice.", Timer: 10 * time.Minute},
	},
}

type app struct {
	cooking recipe.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).FillWidth().Padding(t.Space(8)).Children(func() {
			recipe.View(c, &a.cooking, bread)
		})
	})
}

func main() {
	a := &app{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Recipe",
			Width:   900,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
