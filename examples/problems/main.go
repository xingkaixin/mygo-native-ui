// Problems shows the problems panel of a web app's workspace, from the
// type checker, the linter and the spell checker. Filter them, hide a
// severity, fold a file, and open a problem to see it named below.
//
//	go run ./examples/problems
package main

import (
	"fmt"
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/problems"
)

var diagnostics = []problems.Diagnostic{
	{Path: "src/api/users.ts", Line: 42, Col: 7, Severity: problems.Error, Message: "Property 'email' does not exist on type 'Session'.", Source: "ts", Code: "2339"},
	{Path: "src/api/users.ts", Line: 18, Col: 1, Severity: problems.Warning, Message: "'fetchAll' is declared but its value is never read.", Source: "ts", Code: "6133"},
	{Path: "src/api/users.ts", Line: 57, Col: 12, Severity: problems.Error, Message: "Argument of type 'string | undefined' is not assignable to parameter of type 'string'.", Source: "ts", Code: "2345"},
	{Path: "src/components/Avatar.tsx", Line: 9, Col: 5, Severity: problems.Warning, Message: "img elements must have an alt prop, either with meaningful text, or an empty string.", Source: "eslint", Code: "jsx-a11y/alt-text"},
	{Path: "src/components/Avatar.tsx", Line: 3, Col: 10, Severity: problems.Info, Message: "\"Gravatar\": Unknown word.", Source: "cSpell"},
	{Path: "src/components/Table.tsx", Line: 88, Col: 3, Severity: problems.Warning, Message: "React Hook useEffect has a missing dependency: 'rows'.", Source: "eslint", Code: "react-hooks/exhaustive-deps"},
	{Path: "src/components/Table.tsx", Line: 120, Col: 21, Severity: problems.Error, Message: "Type 'number' is not assignable to type 'string'.", Source: "ts", Code: "2322"},
	{Path: "src/lib/date.ts", Line: 4, Col: 1, Severity: problems.Info, Message: "File is a CommonJS module; it may be converted to an ES module.", Source: "ts", Code: "80001"},
	{Path: "src/lib/date.ts", Line: 31, Col: 14, Severity: problems.Warning, Message: "Unexpected any. Specify a different type.", Source: "eslint", Code: "@typescript-eslint/no-explicit-any"},
	{Path: "README.md", Line: 12, Col: 1, Severity: problems.Info, Message: "\"Tanstack\": Unknown word.", Source: "cSpell"},
}

type app struct {
	panel  problems.State
	opened string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Gap(t.Space(3)).Children(func() {
		ui.Text(c, "Problems").FontSize(t.FontSize + 4).Bold()
		ev := problems.View(c, &a.panel, diagnostics)
		ev.Grow(1)
		if ev.Opened >= 0 {
			d := diagnostics[ev.Opened]
			a.opened = fmt.Sprintf("Opened %s:%d:%d", d.Path, d.Line, d.Col)
		}
		ui.Text(c, a.opened).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
	})
}

func main() {
	a := &app{}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Problems",
			Width:   880,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
