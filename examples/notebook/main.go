// Notebook works out compound interest in a notebook whose kernel is a
// small calculator: each line sets a name (rate = 0.05), prints
// (print(x)) or, last in a cell, shows a value. It knows + - * / %,
// parentheses, pow, sqrt and round. Shift+Enter runs a cell; Run all runs
// them in order, and Restart forgets the names.
//
//	go run ./examples/notebook
package main

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"

	"github.com/xingkaixin/mygo-native-ui/ui/notebook"
)

// took is how long the kernel takes over a cell, so that [*] shows.
const took = 500 * time.Millisecond

type app struct {
	nb      notebook.State
	cells   []notebook.Cell
	vars    map[string]float64
	runs    int
	queue   []*editor.State
	started time.Time
}

func newApp() *app {
	a := &app{nb: notebook.State{Language: "cell.py"}, vars: map[string]float64{}}
	a.cells = []notebook.Cell{
		notebook.NewCell(notebook.Markdown, "# Compound interest\n\nWhat **1,000** grows to at a yearly rate, compounded each year. Run the cells in order with **Shift+Enter**."),
		notebook.NewCell(notebook.Code, "principal = 1000\nrate = 0.05\nyears = 10"),
		notebook.NewCell(notebook.Code, "balance = principal * pow(1 + rate, years)\nround(balance, 2)"),
		notebook.NewCell(notebook.Markdown, "## Monthly\n\nCompounded each month, the same rate earns a little more."),
		notebook.NewCell(notebook.Code, "monthly = principal * pow(1 + rate / 12, years * 12)\nprint(round(monthly, 2))\nround(monthly - balance, 2)"),
		notebook.NewCell(notebook.Code, "# A name never set raises an error\nbalance / months"),
	}
	return a
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	a.step(c)
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Padding(t.Space(2), t.Space(4)).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "interest.ipynb").Bold().Grow(1)
			if ui.Button(c, "Run all").Clicked() {
				for i := range a.cells {
					if a.cells[i].Kind == notebook.Code {
						a.enqueue(&a.cells[i])
					}
				}
			}
			if ui.Button(c, "Restart").Clicked() {
				a.vars, a.runs, a.queue = map[string]float64{}, 0, nil
				for i := range a.cells {
					a.cells[i].Count, a.cells[i].Outputs, a.cells[i].Running = 0, nil, false
				}
			}
		})
		ui.Divider(c)
		ui.Scroll(c).Grow(1).Children(func() {
			ui.Column(c).Padding(t.Space(4), t.Space(4), t.Space(12)).Children(func() {
				if ev := notebook.View(c, &a.nb, &a.cells); ev.Run >= 0 {
					a.enqueue(&a.cells[ev.Run])
				}
			})
		})
	})
}

func (a *app) enqueue(cl *notebook.Cell) {
	if cl.Running {
		return
	}
	cl.Running = true
	if len(a.queue) == 0 {
		a.started = time.Time{}
	}
	a.queue = append(a.queue, cl.Source)
}

// step runs the first cell of the queue once it has had its time.
func (a *app) step(c *ui.Context) {
	if len(a.queue) == 0 {
		return
	}
	now := c.Now()
	if a.started.IsZero() {
		a.started = now
	}
	if wait := took - now.Sub(a.started); wait > 0 {
		c.After(wait)
		return
	}
	src := a.queue[0]
	a.queue, a.started = a.queue[1:], now
	for i := range a.cells {
		cl := &a.cells[i]
		if cl.Source != src {
			continue
		}
		a.runs++
		cl.Count, cl.Running, cl.Outputs = a.runs, false, a.exec(cl.Text())
	}
	if len(a.queue) > 0 {
		c.After(took)
	}
}

// exec runs the lines of src, and returns what they printed.
func (a *app) exec(src string) []notebook.Output {
	var out []notebook.Output
	lines := strings.Split(src, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if h := strings.Index(line, "#"); h >= 0 {
			line = strings.TrimSpace(line[:h])
		}
		if line == "" {
			continue
		}
		name, expr, assign := strings.Cut(line, "=")
		if !assign || strings.ContainsAny(name, "()") {
			name, expr = "", line
		}
		if inner, ok := strings.CutPrefix(expr, "print("); ok && strings.HasSuffix(inner, ")") {
			v, err := a.eval(strings.TrimSuffix(inner, ")"))
			if err != nil {
				return append(out, notebook.Output{Text: err.Error(), Error: true})
			}
			out = append(out, notebook.Output{Text: format(v)})
			continue
		}
		v, err := a.eval(expr)
		if err != nil {
			return append(out, notebook.Output{Text: err.Error(), Error: true})
		}
		switch {
		case name != "":
			a.vars[strings.TrimSpace(name)] = v
		case i == len(lines)-1:
			out = append(out, notebook.Output{Text: format(v)})
		}
	}
	return out
}

func (a *app) eval(src string) (float64, error) {
	e, err := parser.ParseExpr(src)
	if err != nil {
		return 0, fmt.Errorf("SyntaxError: invalid syntax: %s", strings.TrimSpace(src))
	}
	return a.value(e)
}

func (a *app) value(e ast.Expr) (float64, error) {
	switch e := e.(type) {
	case *ast.BasicLit:
		return strconv.ParseFloat(e.Value, 64)
	case *ast.Ident:
		v, ok := a.vars[e.Name]
		if !ok {
			return 0, fmt.Errorf("NameError: name '%s' is not defined", e.Name)
		}
		return v, nil
	case *ast.ParenExpr:
		return a.value(e.X)
	case *ast.UnaryExpr:
		v, err := a.value(e.X)
		if e.Op == token.SUB {
			v = -v
		}
		return v, err
	case *ast.BinaryExpr:
		x, err := a.value(e.X)
		if err != nil {
			return 0, err
		}
		y, err := a.value(e.Y)
		if err != nil {
			return 0, err
		}
		switch e.Op {
		case token.ADD:
			return x + y, nil
		case token.SUB:
			return x - y, nil
		case token.MUL:
			return x * y, nil
		case token.QUO, token.REM:
			if y == 0 {
				return 0, errors.New("ZeroDivisionError: division by zero")
			}
			if e.Op == token.REM {
				return math.Mod(x, y), nil
			}
			return x / y, nil
		}
	case *ast.CallExpr:
		fn, _ := e.Fun.(*ast.Ident)
		var args []float64
		for _, arg := range e.Args {
			v, err := a.value(arg)
			if err != nil {
				return 0, err
			}
			args = append(args, v)
		}
		switch {
		case fn == nil:
		case fn.Name == "pow" && len(args) == 2:
			return math.Pow(args[0], args[1]), nil
		case fn.Name == "sqrt" && len(args) == 1:
			return math.Sqrt(args[0]), nil
		case fn.Name == "round" && len(args) == 2:
			p := math.Pow(10, args[1])
			return math.Round(args[0]*p) / p, nil
		case fn.Name == "round" && len(args) == 1:
			return math.Round(args[0]), nil
		default:
			return 0, fmt.Errorf("NameError: name '%s' is not defined", fn.Name)
		}
	}
	return 0, errors.New("SyntaxError: invalid syntax")
}

func format(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }

func main() {
	a := newApp()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Notebook",
			Width:   820,
			Height:  760,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
