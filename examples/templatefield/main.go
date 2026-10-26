// Templatefield sends a request to an API in one of two environments: its
// URL and headers hold {{variables}}, which the environment chosen
// defines, and the request they make shows below. The token is a secret;
// {{region}} is defined in neither.
//
//	go run ./examples/templatefield
package main

import (
	"log"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/kvtable"
	"github.com/xingkaixin/mygo-native-ui/ui/templatefield"
)

// env is an environment: its name, its variables, and those of them that
// are secrets.
type env struct {
	name    string
	vars    map[string]string
	secrets []string
}

func (e *env) Names() []string { return slices.Sorted(maps.Keys(e.vars)) }

func (e *env) Lookup(name string) (value, source string, secret, ok bool) {
	value, ok = e.vars[name]
	return value, e.name, slices.Contains(e.secrets, name), ok
}

var envs = []*env{
	{
		name:    "Staging",
		vars:    map[string]string{"base_url": "https://staging.api.kettle.dev", "version": "v2", "user_id": "42", "token": "stg_9f2c41"},
		secrets: []string{"token"},
	},
	{
		name:    "Production",
		vars:    map[string]string{"base_url": "https://api.kettle.dev", "version": "v2", "user_id": "1007", "token": "prd_77ab03"},
		secrets: []string{"token"},
	},
}

// variable is a variable as the fields write it.
var variable = regexp.MustCompile(`\{\{ *([\w.-]+) *\}\}`)

type app struct {
	env     string
	url     string
	headers []kvtable.Pair
}

func (a *app) current() *env {
	for _, e := range envs {
		if e.name == a.env {
			return e
		}
	}
	return envs[0]
}

// fill returns s with each variable the environment defines replaced by
// its value, secrets by dots.
func (a *app) fill(s string) string {
	e := a.current()
	return variable.ReplaceAllStringFunc(s, func(m string) string {
		name := variable.FindStringSubmatch(m)[1]
		if v, ok := e.vars[name]; ok {
			if slices.Contains(e.secrets, name) {
				return "••••••••"
			}
			return v
		}
		return m
	})
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	e := a.current()
	ui.Column(c).Fill().Background(t.Background).Padding(16).Gap(12).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Select(c, &a.env, []string{"Staging", "Production"}).Label("Environment")
			ui.Row(c).Grow(1).AlignItems(ui.Center).Radius(t.Radius).Background(t.Surface).Border(1, t.Border).Children(func() {
				templatefield.Input(c, &a.url, e).Grow(1).Padding(t.Space(1.5), t.Space(2.5)).
					Label("URL").Placeholder("{{base_url}}/")
			})
		})
		kvtable.EditorWith(c, &a.headers, kvtable.Options{
			KeyLabel: "Header",
			Ranges:   func(v string) []ui.TextRange { return templatefield.Ranges(v, e, t) },
		})
		ui.Text(c, "Request").Bold().FontSize(12)
		lines := []string{"GET " + a.fill(a.url)}
		for _, h := range a.headers {
			if h.On && h.Key != "" {
				lines = append(lines, h.Key+": "+a.fill(h.Value))
			}
		}
		ui.Text(c, strings.Join(lines, "\n")).Font("monospace").FontSize(12).Selectable().
			Padding(10).Radius(6).Background(t.Surface)
	})
}

func main() {
	a := &app{
		env: "Staging",
		url: "{{base_url}}/{{version}}/users/{{user_id}}/kettles?region={{region}}",
		headers: []kvtable.Pair{
			{Key: "Authorization", Value: "Bearer {{token}}", On: true},
			{Key: "Accept", Value: "application/json", On: true},
		},
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Request",
			Width:   760,
			Height:  420,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
