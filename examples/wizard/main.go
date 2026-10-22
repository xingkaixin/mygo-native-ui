// Wizard is a "New project" assistant: the project's name and location,
// a template chosen from cards, options with switches and drop-downs, and
// a summary, whose Create shows the project made, with Start over. The
// toolbar shows the steps down the side, as macOS installers list them,
// or along the top.
//
//	go run ./examples/wizard
package main

import (
	"fmt"
	"log"
	"path"
	"slices"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/wizard"
)

var (
	cliIcon     = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M5 7l5 5-5 5M12 18h7"/></svg>`))
	webIcon     = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3a14 14 0 0 1 0 18M12 3a14 14 0 0 0 0 18"/></svg>`))
	desktopIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="4" width="18" height="16" rx="2.5"/><path d="M3 9h18"/></svg>`))
	libraryIcon = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linejoin="round"><path d="M12 3l8 4.5v9L12 21l-8-4.5v-9z"/><path d="M4 7.5l8 4.5 8-4.5M12 12v9"/></svg>`))
)

// template is a kind of project to start from.
type template struct {
	name, about string
	icon        *ui.SVG
}

var templates = []template{
	{"Command-line tool", "A program run from the terminal, with flags and subcommands.", cliIcon},
	{"Web service", "An HTTP server with routing, logging and a health check.", webIcon},
	{"Desktop app", "A window with a menu bar and a sidebar, built with mygo.", desktopIcon},
	{"Library", "A package for other programs to import, with examples.", libraryIcon},
}

// taken are the projects already in ~/Developer.
var taken = []string{"dotfiles", "notes", "website"}

// app is the project the assistant sets up, the step it is at, and
// whether the project is made.
type app struct {
	indicator int
	wizard    wizard.State
	step      int
	created   bool

	name, location, about string
	template              string
	goVersion, license    string
	git, readme, ci       bool
}

func newApp() *app {
	return &app{location: "~/Developer", goVersion: "1.27", license: "MIT", git: true, readme: true}
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Toolbar(c, func() {
			ui.Text(c, "New Project").Bold()
			ui.Spacer(c)
			ui.Segmented(c, &a.indicator, "Sidebar", "Top").Label("Steps")
		}).Padding(t.Space(2), t.Space(4))
		ui.Divider(c)
		ui.Column(c).Grow(1).MinHeight(0).Children(func() {
			if a.created {
				a.finished(c)
				return
			}
			at := wizard.Sidebar
			if a.indicator == 1 {
				at = wizard.Top
			}
			if wizard.View(c, &a.wizard, &a.step, a.steps(c), at, "Create").Done() {
				a.created = true
			}
		})
	})
}

func (a *app) steps(c *ui.Context) []wizard.Step {
	return []wizard.Step{
		{Title: "Name and location", Subtitle: "Choose what the project is called and where it goes.", Build: func() { a.naming(c) }, Check: a.checkName},
		{Title: "Template", Subtitle: "Start from a template. You can change all of it later.", Build: func() { a.templates(c) }, Check: func() string {
			if a.template == "" {
				return "Choose a template."
			}
			return ""
		}},
		{Title: "Options", Subtitle: "Set the project up the way you work.", Build: func() { a.options(c) }, Check: func() string {
			if a.ci && !a.git {
				return "CI needs a Git repository. Turn on Git, or turn off CI."
			}
			return ""
		}},
		{Title: "Summary", Subtitle: "Check the project before creating it.", Build: func() { a.summary(c) }},
	}
}

func (a *app) checkName() string {
	switch {
	case a.name == "":
		return "Enter a name for the project."
	case strings.Trim(a.name, "abcdefghijklmnopqrstuvwxyz0123456789-") != "":
		return "Use only lowercase letters, digits and hyphens in the name."
	case strings.TrimSpace(a.location) == "":
		return "Choose where the project goes."
	case a.location == "~/Developer" && slices.Contains(taken, a.name):
		return fmt.Sprintf("There is already a project named %s in %s.", a.name, a.location)
	}
	return ""
}

func (a *app) naming(c *ui.Context) {
	t := c.Theme()
	where := "Choose a name to see where the project goes."
	if a.name != "" {
		where = "Creates " + path.Join(a.location, a.name)
	}
	ui.Form(c, func() {
		ui.Field(c, "Name", func() {
			if ui.TextInput(c, &a.name).Placeholder("weather-cli").AutoFocus().Submitted() {
				a.wizard.Continue()
			}
		}).Description("Lowercase letters, digits and hyphens.")
		ui.Field(c, "Location", func() {
			if ui.TextInput(c, &a.location).Submitted() {
				a.wizard.Continue()
			}
		}).Description(where)
		ui.Field(c, "Description", func() {
			ui.TextArea(c, &a.about).Height(t.Space(20))
		}).Description("Optional. Opens the README.")
	}).MaxWidth(t.Space(130))
}

func (a *app) templates(c *ui.Context) {
	t := c.Theme()
	ui.RadioGroup(c, func() {
		for i, tp := range templates {
			card := ui.RadioBase(c.Key(i), &a.template, tp.name).AlignItems(ui.Start).Gap(t.Space(3)).Padding(t.Space(3.5)).
				Radius(t.Space(2.5)).Border(1, t.Border).Label(tp.name)
			switch {
			case a.template == tp.name:
				card.Border(1, t.Accent).Shadow(0, 0, 0, 1, t.Accent).Background(t.Accent.Alpha(0.08))
			case card.Hovered():
				card.Background(t.Text.Alpha(0.03))
			}
			card.Children(func() {
				ui.Row(c).Size(t.Space(9), t.Space(9)).Radius(t.Space(2)).Background(t.Accent.Alpha(0.12)).
					Justify(ui.Center).AlignItems(ui.Center).Shrink(0).Children(func() {
					ui.Icon(c, tp.icon).FontSize(t.FontSize + 5).TextColor(t.Accent)
				})
				ui.Column(c).Gap(t.Space(1)).Grow(1).Shrink(1).Children(func() {
					ui.Text(c, tp.name).Bold()
					ui.Text(c, tp.about).FontSize(t.FontSize - 1).TextColor(t.TextMuted).LineHeight(1.35)
				})
			})
		}
	}).Grid().Columns(2).Gap(t.Space(3)).Label("Template").MaxWidth(t.Space(150))
}

func (a *app) options(c *ui.Context) {
	t := c.Theme()
	toggle := func(label string, on *bool, about string) {
		ui.Field(c.Key(label), label, func() {
			ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Center).Children(func() {
				ui.Switch(c, on)
				ui.Text(c, about).TextColor(t.TextMuted).Shrink(1)
			})
		})
	}
	ui.Form(c, func() {
		ui.Field(c, "Go version", func() {
			ui.Select(c, &a.goVersion, []string{"1.27", "1.26", "1.25"}).Width(t.Space(45))
		})
		ui.Field(c, "License", func() {
			ui.Select(c, &a.license, []string{"MIT", "Apache 2.0", "BSD 3-Clause", "None"}).Width(t.Space(45))
		})
		toggle("Git", &a.git, "Start a repository with a first commit.")
		toggle("README", &a.readme, "Add a README with the description.")
		toggle("CI", &a.ci, "Add a workflow that builds and vets every push.")
	})
}

func (a *app) summary(c *ui.Context) {
	var extras []string
	for _, x := range []struct {
		on   bool
		name string
	}{{a.git, "Git repository"}, {a.readme, "README"}, {a.ci, "CI workflow"}} {
		if x.on {
			extras = append(extras, x.name)
		}
	}
	if len(extras) == 0 {
		extras = []string{"None"}
	}
	ui.Form(c, func() {
		for _, row := range [][2]string{
			{"Name", a.name},
			{"Location", path.Join(a.location, a.name)},
			{"Template", a.template},
			{"Go version", a.goVersion},
			{"License", a.license},
			{"Includes", strings.Join(extras, ", ")},
		} {
			ui.Field(c.Key(row[0]), row[0], func() { ui.Text(c, row[1]).Selectable() })
		}
	})
}

func (a *app) finished(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Grow(1).Justify(ui.Center).AlignItems(ui.Center).Gap(t.Space(3)).Padding(t.Space(8)).Children(func() {
		ui.Box(c).Size(t.Space(14), t.Space(14)).Draw(func(p *ui.Painter, r ui.Rect) {
			cx, cy, k := r.X+r.W/2, r.Y+r.H/2, r.W*0.22
			var circle ui.Path
			circle.Circle(cx, cy, r.W/2)
			p.FillPath(&circle, t.Success)
			p.StrokePolyline([][2]float32{{cx - k, cy}, {cx - k*0.3, cy + k*0.7}, {cx + k, cy - k*0.65}}, 3, ui.RGB(255, 255, 255))
		})
		ui.Text(c, "Created "+a.name).FontSize(t.FontSize+7).Bold().Margin(t.Space(2), 0, 0, 0)
		ui.Text(c, fmt.Sprintf("A %s in %s.", strings.ToLower(a.template), path.Join(a.location, a.name))).TextColor(t.TextMuted)
		if ui.Button(c, "Start over").Margin(t.Space(3), 0, 0, 0).Clicked() {
			indicator := a.indicator
			*a = *newApp()
			a.indicator = indicator
		}
	})
}

func main() {
	a := newApp()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "New Project",
			Width:   860,
			Height:  600,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
