// Debugger steps through a small Go program that totals a cart, paused at
// a breakpoint: step over to watch sum change, open items and it to see
// their fields, choose main.main in the stack for its variables, and
// double-click a value to change it. The breakpoint on line 14 stops only
// once sum > 10.
//
//	go run ./examples/debugger
package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/debugger"
)

const file = "cmd/cart/main.go"

var source = strings.Split(`package main

import "fmt"

type Item struct {
	Name  string
	Price int
	Qty   int
}

func total(items []Item) int {
	sum := 0
	for i, it := range items {
		sum += it.Price * it.Qty
	}
	return sum
}

func main() {
	cart := []Item{{"pen", 3, 2}, {"book", 12, 1}, {"mug", 8, 3}}
	fmt.Println(total(cart))
}`, "\n")

type item struct {
	name       string
	price, qty int
}

var cart = []item{{"pen", 3, 2}, {"book", 12, 1}, {"mug", 8, 3}}

// step is where the program is paused: its line in total, or 21 back in
// main, and total's locals, -1 for those not yet set.
type step struct {
	line, sum, i int
}

var steps = func() []step {
	out := []step{{12, -1, -1}, {13, 0, -1}}
	sum := 0
	for i, it := range cart {
		out = append(out, step{14, sum, i})
		sum += it.price * it.qty
		out = append(out, step{13, sum, i})
	}
	return append(out, step{16, sum, -1}, step{21, sum, -1})
}()

type app struct {
	debug       debugger.State
	at          int
	running     bool
	ended       bool
	breakpoints []debugger.Breakpoint
	edits       map[string]string
	opened      int
	openVars    bool
	openStack   bool
	openBreaks  bool
}

func (a *app) stops(s step) bool {
	for _, b := range a.breakpoints {
		if !b.On || b.Line != s.line {
			continue
		}
		var n int
		if _, err := fmt.Sscanf(b.Cond, "sum > %d", &n); err == nil {
			return s.sum > n
		}
		return true
	}
	return false
}

func (a *app) run(cmd debugger.Command) {
	switch cmd {
	case debugger.StepOver, debugger.StepInto:
		a.at++
	case debugger.StepOut:
		a.at = len(steps) - 1
	case debugger.Continue:
		for a.at++; a.at < len(steps) && !a.stops(steps[a.at]); a.at++ {
		}
	case debugger.Restart:
		a.at, a.ended = 0, false
	case debugger.Stop:
		a.ended = true
	case debugger.Pause:
	}
	if a.at >= len(steps) {
		a.ended = true
	}
	a.edits = map[string]string{}
}

func (a *app) value(path, v string) string {
	if e, ok := a.edits[path]; ok {
		return e
	}
	return v
}

func itemVar(a *app, scope, name string, it item, changed bool) debugger.Var {
	at := scope + "." + name
	return debugger.Var{
		Name: name, Type: "main.Item", Value: fmt.Sprintf("{Name: %q, Price: %d, Qty: %d}", it.name, it.price, it.qty), Changed: changed,
		Kids: func() []debugger.Var {
			return []debugger.Var{
				{Name: "Name", Type: "string", Value: a.value(at+".Name", fmt.Sprintf("%q", it.name))},
				{Name: "Price", Type: "int", Value: a.value(at+".Price", fmt.Sprint(it.price))},
				{Name: "Qty", Type: "int", Value: a.value(at+".Qty", fmt.Sprint(it.qty))},
			}
		},
	}
}

func sliceVar(a *app, scope, name string) debugger.Var {
	return debugger.Var{
		Name: name, Type: "[]main.Item", Value: fmt.Sprintf("[]main.Item len: %d, cap: %d", len(cart), len(cart)),
		Kids: func() []debugger.Var {
			out := make([]debugger.Var, len(cart))
			for i, it := range cart {
				out[i] = itemVar(a, scope+"."+name, fmt.Sprintf("[%d]", i), it, false)
			}
			return out
		},
	}
}

func (a *app) scopes() []debugger.Scope {
	if a.ended {
		return nil
	}
	s := steps[a.at]
	globals := debugger.Scope{Name: "Globals", Vars: []debugger.Var{{Name: "version", Type: "string", Value: `"1.4.2"`}}}
	if _, frame := a.debug.Frame(); frame == 1 || s.line == 21 {
		return []debugger.Scope{{Name: "Locals", Vars: []debugger.Var{sliceVar(a, "Locals", "cart")}}, globals}
	}
	var prev step
	if a.at > 0 {
		prev = steps[a.at-1]
	}
	locals := []debugger.Var{sliceVar(a, "Locals", "items")}
	if s.sum >= 0 {
		locals = append(locals, debugger.Var{Name: "sum", Type: "int", Value: a.value("Locals.sum", fmt.Sprint(s.sum)), Changed: s.sum != prev.sum})
	}
	if s.i >= 0 {
		locals = append(locals,
			debugger.Var{Name: "i", Type: "int", Value: a.value("Locals.i", fmt.Sprint(s.i)), Changed: s.i != prev.i},
			itemVar(a, "Locals", "it", cart[s.i], s.i != prev.i))
	}
	return []debugger.Scope{{Name: "Locals", Vars: locals}, globals}
}

func (a *app) threads() []debugger.Thread {
	if a.ended {
		return nil
	}
	s := steps[a.at]
	frames := []debugger.Frame{{Func: "main.total", File: file, Line: s.line}, {Func: "main.main", File: file, Line: 21}, {Func: "runtime.main"}}
	if s.line == 21 {
		frames = frames[1:]
	}
	return []debugger.Thread{
		{Name: "Goroutine 1", Reason: "breakpoint", Frames: frames},
		{Name: "Goroutine 18 gcBgMarkWorker"},
	}
}

func section(c *ui.Context, title string, open *bool, fn func()) {
	ui.Collapsible(c, title, open, fn)
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Width(360).FillHeight().Background(t.Surface).Padding(t.Space(3)).Gap(t.Space(3)).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "Run and Debug").Bold().Grow(1)
				if cmd := debugger.Toolbar(c, !a.ended); cmd != debugger.None {
					a.run(cmd)
				}
			})
			ui.Scroll(c).Grow(1).Children(func() {
				ui.Column(c).Gap(t.Space(2)).Children(func() {
					section(c, "Variables", &a.openVars, func() {
						if ev := debugger.Variables(c, &a.debug, a.scopes()); ev.Path != nil {
							a.edits[strings.Join(ev.Path, ".")] = ev.Value
						}
					})
					section(c, "Call Stack", &a.openStack, func() {
						debugger.Stack(c, &a.debug, a.threads())
					})
					section(c, "Breakpoints", &a.openBreaks, func() {
						if ev := debugger.Breakpoints(c, &a.breakpoints); ev.Opened >= 0 {
							a.opened = a.breakpoints[ev.Opened].Line
						}
					})
				})
			})
		})
		a.code(c)
	})
}

func (a *app) code(c *ui.Context) {
	t := c.Theme()
	line := 0
	if !a.ended {
		line = steps[a.at].line
		if _, frame := a.debug.Frame(); frame == 1 && line != 21 {
			line = 21
		}
	}
	ui.Column(c).Grow(1).FillHeight().Padding(t.Space(4)).Gap(t.Space(3)).Children(func() {
		status := "Paused"
		if a.ended {
			status = "Exited, printed 42"
		}
		ui.Row(c).Gap(t.Space(3)).Children(func() {
			ui.Text(c, file).Bold().Grow(1)
			ui.Text(c, status).TextColor(t.TextMuted)
		})
		ui.Column(c).Font("monospace").FontSize(t.FontSize - 1).Children(func() {
			for i, l := range source {
				n := i + 1
				row := ui.Row(c.Key(n)).Gap(t.Space(2)).Padding(1, t.Space(2)).AlignItems(ui.Center)
				switch {
				case n == line:
					row.Background(t.Warning.Alpha(0.25))
				case n == a.opened:
					row.Background(t.Accent.Alpha(0.1))
				}
				var bp *debugger.Breakpoint
				for j := range a.breakpoints {
					if a.breakpoints[j].Line == n {
						bp = &a.breakpoints[j]
					}
				}
				row.Children(func() {
					ui.Box(c).Size(10, 10).Draw(func(p *ui.Painter, r ui.Rect) {
						if bp == nil {
							return
						}
						color := t.Danger
						if !bp.On {
							color = t.TextMuted
						}
						var d ui.Path
						d.Circle(r.X+5, r.Y+5, 4.5)
						p.FillPath(&d, color)
					})
					ui.Text(c, fmt.Sprint(n)).Width(t.Space(6)).TextAlign(ui.End).TextColor(t.TextMuted)
					ui.Text(c, strings.ReplaceAll(l, "\t", "    ")).NoWrap()
				})
			}
		})
	})
}

func main() {
	a := &app{
		breakpoints: []debugger.Breakpoint{
			{File: file, Line: 12, On: true},
			{File: file, Line: 14, On: true, Cond: "sum > 10"},
			{File: "internal/pricing/tax.go", Line: 31, On: false},
		},
		edits:      map[string]string{},
		openVars:   true,
		openStack:  true,
		openBreaks: true,
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Debugger",
			Width:   1000,
			Height:  700,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
