// Familytree draws four generations of a made-up family, from a couple
// married in 1921 to their great-grandchildren.
//
//	go run ./examples/familytree
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/familytree"
)

func spouse(name string, born, died int) *familytree.Person {
	return &familytree.Person{Name: name, Born: born, Died: died}
}

var family = familytree.Family{
	Person: familytree.Person{Name: "Arthur Hale", Born: 1896, Died: 1971},
	Spouse: spouse("Edith Moore", 1899, 1984),
	Children: []familytree.Family{
		{
			Person: familytree.Person{Name: "Walter Hale", Born: 1922, Died: 1999},
			Spouse: spouse("June Price", 1925, 2011),
			Children: []familytree.Family{
				{Person: familytree.Person{Name: "Carol Hale", Born: 1948, Died: 2020}, Spouse: spouse("Dennis Kerr", 1946, 2015),
					Children: []familytree.Family{
						{Person: familytree.Person{Name: "Megan Kerr", Born: 1974}},
						{Person: familytree.Person{Name: "Owen Kerr", Born: 1977}, Spouse: spouse("Priya Nair", 1979, 0)},
					}},
				{Person: familytree.Person{Name: "Stephen Hale", Born: 1951}, Spouse: spouse("Linda Park", 1953, 0),
					Children: []familytree.Family{
						{Person: familytree.Person{Name: "Laura Hale", Born: 1980}},
					}},
			},
		},
		{Person: familytree.Person{Name: "Dorothy Hale", Born: 1925, Died: 1932}},
		{
			Person: familytree.Person{Name: "Margaret Hale", Born: 1928, Died: 2017},
			Spouse: spouse("Frank Doyle", 1926, 1994),
			Children: []familytree.Family{
				{Person: familytree.Person{Name: "Patrick Doyle", Born: 1953}, Spouse: spouse("Anne Byrne", 1955, 0),
					Children: []familytree.Family{
						{Person: familytree.Person{Name: "Sean Doyle", Born: 1982}},
						{Person: familytree.Person{Name: "Ciara Doyle", Born: 1985}},
					}},
				{Person: familytree.Person{Name: "Helen Doyle", Born: 1956}},
			},
		},
	},
}

type app struct {
	tree familytree.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.ScrollBoth(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Padding(t.Space(6)).Children(func() {
			familytree.View(c, &a.tree, family)
		})
	})
}

func main() {
	a := &app{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Family tree",
			Width:   1280,
			Height:  440,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
