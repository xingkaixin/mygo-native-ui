// Orgchart shows the organization chart of a company of 120 people, as
// an HR app does: the chart to zoom, pan and fold, and the person chosen,
// with whom they report to and how many people are under them, below.
//
//	go run ./examples/orgchart
package main

import (
	"fmt"
	"log"
	"runtime"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/orgchart"
)

type app struct {
	people []orgchart.Person
	chart  orgchart.State
	chosen int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	cmd := "Ctrl"
	if runtime.GOOS == "darwin" {
		cmd = "⌘"
	}
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Padding(t.Space(3)).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Northwind").Bold()
			ui.Textf(c, "%d people", len(a.people)).TextColor(t.TextMuted).Grow(1)
			ui.Text(c, fmt.Sprintf("Drag or scroll to pan, %s-scroll to zoom, double-click to zoom in and out; arrows move between people.", cmd)).
				FontSize(t.FontSize - 1).TextColor(t.TextMuted).SingleLine().Shrink(1)
		})
		ui.Divider(c)
		orgchart.View(c, &a.chart, a.people, &a.chosen).Grow(1).Background(t.Surface).AutoFocus()
		ui.Divider(c)
		// Below the chart, the zoom it reads is this frame's.
		ui.Row(c).Padding(t.Space(2.5), t.Space(3)).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Column(c).Gap(t.Space(0.5)).Grow(1).Shrink(1).Children(func() {
				if a.chosen >= 0 && a.chosen < len(a.people) {
					ui.Text(c, a.people[a.chosen].Name).Bold()
					ui.Text(c, a.detail(a.chosen)).TextColor(t.TextMuted).SingleLine()
				}
			})
			if ui.Button(c, "−").Label("Zoom out").Clicked() {
				a.chart.ZoomOut()
			}
			ui.Textf(c, "%.0f%%", a.chart.Zoom()*100).FontFeatures("tnum").MinWidth(t.Space(12)).TextAlign(ui.Center)
			if ui.Button(c, "+").Label("Zoom in").Clicked() {
				a.chart.ZoomIn()
			}
			if ui.Button(c, "Fit").Clicked() {
				a.chart.Fit()
			}
		})
	})
}

// detail returns the title, team and manager of person i, and how many
// people report to them.
func (a *app) detail(i int) string {
	p := a.people[i]
	s := p.Title + " · " + p.Team
	if m := p.Manager; m >= 0 && m != i {
		s += " · Reports to " + a.people[m].Name
	}
	direct, all := 0, 0
	var walk func(m int)
	walk = func(m int) {
		for j, q := range a.people {
			if q.Manager == m && j != m {
				if m == i {
					direct++
				}
				all++
				walk(j)
			}
		}
	}
	walk(i)
	if direct > 0 {
		s += fmt.Sprintf(" · %d direct reports, %d people in all", direct, all)
	}
	return s
}

// company returns Northwind's people: the chief executive, the heads of
// its departments, their managers and the people on their teams.
func company() []orgchart.Person {
	var people []orgchart.Person
	add := func(name, title, team string, manager int) int {
		people = append(people, orgchart.Person{Name: name, Title: title, Team: team, Manager: manager})
		return len(people) - 1
	}
	names := []string{
		"Ava Kowalski", "Ethan Brooks", "Lena Fischer", "Mateo Rossi", "Chloe Dubois", "Arjun Mehta",
		"Isla MacLeod", "Kenji Watanabe", "Sara Lindgren", "Omar Haddad", "Julia Novak", "Tariq Aziz",
		"Emily Carter", "Diego Ramírez", "Yuki Sato", "Freya Nilsen", "Samuel Osei", "Clara Vogel",
		"Nikhil Rao", "Hannah Weiss", "Leo Marchetti", "Amelia Scott", "Jonas Berg", "Fatima Zahra",
		"Ryan Murphy", "Sofia Costa", "Daniel Levi", "Mira Patel", "Oscar Lund", "Grace Mwangi",
		"Hugo Lefèvre", "Anya Ivanova", "Caleb Turner", "Nora Jensen", "Ibrahim Diallo", "Lucy Bennett",
		"Felipe Santos", "Ingrid Holm", "Kofi Asante", "Maria Lopez", "Ben Whitaker", "Elif Yilmaz",
		"Theo Laurent", "Rosa Martín", "Adam Novotný", "Leila Karimi", "Jack Sullivan", "Mina Park",
		"Pablo Herrera", "Astrid Olsen", "Victor Hugo Lima", "Chiara Bianchi", "Emeka Obi", "Lily Zhang",
		"Marek Wójcik", "Selin Kaya", "Noel Fitzgerald", "Aiko Mori", "Rafael Duarte", "Ella Thompson",
		"Hassan Ali", "Vera Sokolova", "Luca Ferrari", "Maya Brown", "Stefan Horvat", "Imani Johnson",
		"Kai Andersen", "Lucia Romero", "Tomás Silva", "Zoe Clarke", "Anton Petrov", "Naomi Cohen",
		"Ruben Visser", "Layla Hassan", "Finn O'Connor", "Esther Mbeki", "Max Hoffmann", "Priyanka Das",
		"Gabriel Moreau", "Hana Kobayashi", "Pietro Conti", "Alice Wong", "Sven Eriksson", "Amina Yusuf",
		"Nathan Reid", "Camila Torres", "Erik Dahl", "Ruth Abebe", "Jin Ho Lee", "Paula Gómez",
	}
	next := 0
	team := func(manager, n int, title, dept string) {
		for range n {
			add(names[next%len(names)], title, dept, manager)
			next++
		}
	}

	ceo := add("Maya Chen", "Chief Executive Officer", "Executive", -1)
	add("Owen Hart", "Executive Assistant", "Executive", ceo)

	cto := add("Daniel Okafor", "Chief Technology Officer", "Engineering", ceo)
	for _, m := range []struct {
		name string
		n    int
	}{{"Priya Raman", 6}, {"Lucas Moreau", 6}, {"Hana Suzuki", 5}, {"Marcus Bell", 5}, {"Ravi Patel", 5}} {
		team(add(m.name, "Engineering Manager", "Engineering", cto), m.n, "Software Engineer", "Engineering")
	}
	add("Elena Petrova", "Principal Engineer", "Engineering", cto)
	add("Tom Ridley", "Program Manager", "Engineering", cto)

	cpo := add("Sofia Alvarez", "Chief Product Officer", "Product", ceo)
	team(add("Noah Kim", "Director of Design", "Design", cpo), 6, "Product Designer", "Design")
	team(add("Amara Nwosu", "Group Product Manager", "Product", cpo), 5, "Product Manager", "Product")
	add("Iris Lam", "Product Analyst", "Product", cpo)

	sales := add("James Whitfield", "VP of Sales", "Sales", ceo)
	team(add("Rachel Adeyemi", "Sales Manager, East", "Sales", sales), 6, "Account Executive", "Sales")
	team(add("Tom Becker", "Sales Manager, West", "Sales", sales), 6, "Account Executive", "Sales")
	team(add("Inês Duarte", "Sales Manager, EMEA", "Sales", sales), 4, "Account Executive", "Sales")
	add("Kyle Morgan", "Sales Operations", "Sales", sales)
	add("Dana Wu", "Sales Engineer", "Sales", sales)

	marketing := add("Grace Liu", "VP of Marketing", "Marketing", ceo)
	team(add("Felix Wagner", "Content Lead", "Marketing", marketing), 3, "Content Writer", "Marketing")
	team(add("Zara Ahmed", "Growth Lead", "Marketing", marketing), 3, "Growth Marketer", "Marketing")
	add("Jade Martin", "Brand Designer", "Marketing", marketing)
	add("Hiro Tanaka", "Product Marketing", "Marketing", marketing)
	add("Bea Santos", "Events Manager", "Marketing", marketing)

	success := add("Carlos Mendes", "VP of Customer Success", "Customer Success", ceo)
	team(add("Aisha Bello", "Support Manager", "Customer Success", success), 7, "Support Specialist", "Customer Success")
	team(add("Liam O'Brien", "Customer Success Lead", "Customer Success", success), 6, "Account Manager", "Customer Success")
	add("Ruth Klein", "Support Operations", "Customer Success", success)

	finance := add("Robert Lindqvist", "Chief Financial Officer", "Finance", ceo)
	team(add("Mei Tanaka", "Controller", "Finance", finance), 4, "Accountant", "Finance")
	add("Ari Golan", "FP&A Analyst", "Finance", finance)
	add("Claire Dupont", "Legal Counsel", "Finance", finance)

	hr := add("Nadia Haddad", "VP of People", "People", ceo)
	team(add("Ben Carter", "Recruiting Lead", "People", hr), 4, "Recruiter", "People")
	add("Olivia Grant", "HR Business Partner", "People", hr)
	add("Sam Rivera", "Office Manager", "People", hr)
	add("Tess Ward", "People Operations", "People", hr)
	return people
}

func main() {
	a := &app{people: company()}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Org chart",
			Width:   1280,
			Height:  800,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
