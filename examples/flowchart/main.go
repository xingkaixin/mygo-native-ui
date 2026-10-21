// Flowchart draws a flowchart from the text beside it, as it is typed: a
// sign-up, from the page to the first sign-in, with a check that loops
// back and the steps of the verification email in a subgraph. Clicking a
// step shows it under the chart. Document shows Markdown whose mermaid
// blocks draw as the diagrams their first lines name: a flowchart, a
// sequence diagram and a state chart.
//
//	go run ./examples/flowchart
package main

import (
	"log"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"

	"github.com/xingkaixin/mygo-native-ui/ui/flowchart"
	"github.com/xingkaixin/mygo-native-ui/ui/sequence"
	"github.com/xingkaixin/mygo-native-ui/ui/statechart"
)

const signup = `flowchart TD
%% Signing up, from the form to the first sign-in
Start([Open sign-up page]) --> Form[/Enter email and password/]
Form --> Captcha{{Check captcha}} --> Valid{Valid?}
Valid -- No --> Form
Valid -->|Yes| Exists{Email taken?}
Exists -->|Yes| Login[[Sign-in flow]]
Exists -->|No| Create[(Create account)]
Create --> Mail
subgraph Mail [Verification email]
  Send[Send link] --> Wait((Wait))
  Wait -.->|expired| Send
end
Mail ==> Done([Signed in])
Login --> Done`

const doc = `# Releases

A release goes from a merged pull request to production. Staging takes
every green build; production waits for someone to approve it.

~~~mermaid
flowchart LR
  PR([Merged PR]) --> Build[Build and test]
  Build -->|green| Stage[(Staging)]
  Build -.->|red| Fix[Fix the build]
  Stage --> Approve{Approved?}
  Approve -->|yes| Prod[(Production)]
  Approve -->|no| Fix
  Fix -.-> PR
~~~

The deploy service rolls the release out and reports back.

~~~mermaid
sequenceDiagram
  participant CI
  participant Deploy
  participant Cluster
  CI->>Deploy: Release v1.4.2
  Deploy->>Cluster: Roll out
  Cluster-->>Deploy: Healthy
  Deploy-->>CI: Done
~~~

Each release goes through these states. The buttons under it send the
events the state it is in takes.

~~~mermaid
stateDiagram-v2
  [*] --> Building
  Building --> Staged : pass
  Building --> Failed : fail
  Staged --> Live : approve
  Live --> RolledBack : roll back
  Failed --> [*]
~~~

Other code stays code:

~~~go
fmt.Println("v1.4.2")
~~~
`

type app struct {
	tab    int
	source string
	split  float32
	chart  flowchart.State
	md     markdown.Options
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Padding(t.Space(3), t.Space(4)).Children(func() {
			ui.Segmented(c, &a.tab, "Editor", "Document").Label("Show")
		})
		ui.Divider(c)
		if a.tab == 1 {
			ui.Scroll(c).Grow(1).Children(func() {
				ui.Column(c).Padding(t.Space(5), t.Space(6)).Children(func() {
					markdown.View(c, doc, &a.md)
				})
			})
			return
		}
		ui.Split(c, &a.split, func() {
			ui.Column(c).Fill().Padding(t.Space(3)).Children(func() {
				ui.TextArea(c, &a.source).Font("monospace").Label("Source").Grow(1)
			})
		}, func() {
			ui.Column(c).Fill().Children(func() {
				flowchart.View(c, &a.chart, a.source).Grow(1)
				ui.Divider(c)
				ui.Row(c).Gap(t.Space(2)).Padding(t.Space(3), t.Space(4)).AlignItems(ui.Center).Children(func() {
					n, ok := a.chart.Chosen()
					if !ok {
						ui.Text(c, "Click a step to choose it").TextColor(t.TextMuted)
						return
					}
					ui.Text(c, n.Text).Bold()
					ui.Text(c, n.ID).Font("monospace").TextColor(t.TextMuted)
				})
			})
		}).Grow(1)
	})
}

// mermaid draws a mermaid block as the diagram its first line names, and
// as code while it is being written or when it names another.
func mermaid(c *ui.Context, b *markdown.CodeBlockView) {
	t := c.Theme()
	src, kind := b.Code(), ""
	for line := range strings.Lines(src) {
		if s := strings.TrimSpace(line); s != "" && !strings.HasPrefix(s, "%%") {
			kind = strings.Fields(s)[0]
			break
		}
	}
	if b.Open {
		markdown.DefaultCodeBlock(c, b)
		return
	}
	frame := func() ui.Element {
		return ui.Column(c).Padding(t.Space(3)).Radius(t.Radius).Border(1, t.Border)
	}
	switch kind {
	case "flowchart", "graph":
		f := frame()
		s := ui.Local(f, "mermaid.flowchart", func() flowchart.State { return flowchart.State{} })
		f.Children(func() { flowchart.View(c, s, src) })
	case "sequenceDiagram":
		f := frame()
		s := ui.Local(f, "mermaid.sequence", func() sequence.State { return sequence.State{} })
		f.Children(func() { sequence.View(c, s, src) })
	case "stateDiagram-v2", "stateDiagram":
		f := frame()
		s := ui.Local(f, "mermaid.statechart", func() statechart.State { return statechart.State{} })
		// The state chart fills its parent, which in a document has no
		// height of its own.
		f.Height(t.Space(90)).Children(func() { statechart.View(c, s, src) })
	default:
		markdown.DefaultCodeBlock(c, b)
	}
}

func main() {
	a := &app{source: signup, split: 460}
	a.md.Code = map[string]func(*ui.Context, *markdown.CodeBlockView){"mermaid": mermaid}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Flowchart",
			Width:   1440,
			Height:  940,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
