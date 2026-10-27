// Fieldmap imports contacts from a spreadsheet into a CRM: drag from a
// column of the file to a field, or choose the column in the field's
// menu. Import waits for every required field.
//
//	go run ./examples/fieldmap
package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/fieldmap"
)

var columns = []fieldmap.Column{
	{Name: "First Name", Samples: []string{"Ada", "Grace", "Linus"}},
	{Name: "Last Name", Samples: []string{"Lovelace", "Hopper", "Torvalds"}},
	{Name: "E-mail Address", Samples: []string{"ada@analytical.io", "grace@navy.mil", "linus@kernel.org"}},
	{Name: "Mobile", Samples: []string{"+44 20 7946 0958", "+1 202 555 0143", "+358 9 4245 1234"}},
	{Name: "Organization", Samples: []string{"Analytical Engines", "US Navy", "Linux Foundation"}},
	{Name: "Job Title", Samples: []string{"Mathematician", "Rear Admiral", "Maintainer"}},
	{Name: "City", Samples: []string{"London", "Arlington", "Portland"}},
	{Name: "Notes", Samples: []string{"Met at the expo", "", "Wants a demo"}},
}

var fields = []fieldmap.Field{
	{Name: "First name", Required: true},
	{Name: "Last name"},
	{Name: "Email", Required: true},
	{Name: "Phone number"},
	{Name: "Company"},
	{Name: "Title"},
	{Name: "Lead source", Required: true},
	{Name: "Country"},
}

type app struct {
	mapping  []int
	imported string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	if a.mapping == nil {
		a.mapping = fieldmap.Match(columns, fields)
	}
	var missing []string
	filled := 0
	for i, f := range fields {
		switch {
		case a.mapping[i] >= 0:
			filled++
		case f.Required:
			missing = append(missing, f.Name)
		}
	}

	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).Padding(t.Space(5), t.Space(6)).Gap(t.Space(4)).Children(func() {
				ui.Text(c, "Import contacts.csv").FontSize(t.FontSize + 6).Bold()
				if fieldmap.View(c, &a.mapping, columns, fields).Changed() {
					a.imported = ""
				}
			})
		})
		ui.Divider(c)
		ui.Row(c).Padding(t.Space(3), t.Space(6)).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
			status := fmt.Sprintf("%d of %d fields filled", filled, len(fields))
			if len(missing) > 0 {
				status += " · " + strings.Join(missing, ", ") + " needed"
			}
			if a.imported != "" {
				status = a.imported
			}
			ui.Text(c, status).TextColor(t.TextMuted)
			ui.Spacer(c)
			if ui.Button(c, "Match by name").Clicked() {
				a.mapping, a.imported = fieldmap.Match(columns, fields), ""
			}
			if ui.Button(c, "Clear").Clicked() {
				for i := range a.mapping {
					a.mapping[i] = -1
				}
				a.imported = ""
			}
			if ui.PrimaryButton(c, "Import 3 contacts").Disabled(len(missing) > 0).Clicked() {
				a.imported = "Imported 3 contacts"
			}
		})
	})
}

func main() {
	a := &app{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Fieldmap",
			Width:   880,
			Height:  760,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
