// Logview shows the output of a command as it comes, in its colors: the
// command given as its arguments, or a build it makes up, with steps,
// warnings, errors and a progress bar. A field filters the lines; the
// buttons copy the whole log and clear it.
//
//	go run ./examples/logview [command [args...]]
package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/logview"
)

// app is the state the window shows: the log, the filter, and whether the
// command still runs.
type app struct {
	log     logview.Log
	filter  string
	running bool
	title   string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		ui.Row(c).Padding(8, 12).Gap(8).AlignItems(ui.Center).Children(func() {
			status := "Done"
			if a.running {
				status = "Running"
			}
			ui.Text(c, a.title).Bold().SingleLine().Shrink(1)
			ui.Badge(c, status)
			ui.Spacer(c)
			ui.SearchField(c, &a.filter).Width(200)
			if ui.Button(c, "Copy All").Clicked() {
				c.WriteClipboard(a.log.Text())
			}
			if ui.Button(c, "Clear").Clicked() {
				a.log.Clear()
			}
		})
		ui.Divider(c)
		logview.View(c, &a.log, a.filter).Grow(1)
		ui.Divider(c)
		ui.Textf(c, "%d lines", a.log.Len()).FontSize(12).TextColor(t.TextMuted).Padding(6, 12)
	})
}

// stream writes what r reads to the log, on the main thread, until r ends.
func (a *app) stream(win *mygo.Window, r io.Reader) {
	buf := make([]byte, 32<<10)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			chunk := bytes.Clone(buf[:n])
			win.Update(func() { a.log.Write(chunk) })
		}
		if err != nil {
			return
		}
	}
}

// run runs args, its output and errors to the log.
func (a *app) run(win *mygo.Window, args []string) {
	cmd := exec.Command(args[0], args[1:]...)
	// Asked to, many tools color their output to a pipe as to a terminal.
	cmd.Env = append(os.Environ(), "FORCE_COLOR=1", "CLICOLOR_FORCE=1")
	r, w := io.Pipe()
	cmd.Stdout, cmd.Stderr = w, w
	go a.stream(win, r)
	err := cmd.Run()
	w.Close()
	win.Update(func() {
		if err != nil {
			fmt.Fprintf(&a.log, "\n\x1b[31m%v\x1b[0m\n", err)
		}
		a.running = false
	})
}

// build writes a made-up build to w, at the pace of one.
func build(w io.Writer) {
	r := rand.New(rand.NewPCG(1, 2))
	pkgs := []string{"internal/config", "internal/store", "internal/http", "internal/auth", "cmd/server", "cmd/migrate"}
	fmt.Fprintf(w, "\x1b[1m▶ Building\x1b[0m example.com/notes \x1b[2m(go1.27)\x1b[0m\n")
	for round := 1; ; round++ {
		for _, p := range pkgs {
			time.Sleep(time.Duration(80+r.IntN(300)) * time.Millisecond)
			fmt.Fprintf(w, "  \x1b[32m✓\x1b[0m %-18s \x1b[2m%dms\x1b[0m\n", p, 20+r.IntN(400))
			switch r.IntN(8) {
			case 0:
				fmt.Fprintf(w, "  \x1b[33mwarning\x1b[0m: %s/handler.go:%d: unused variable \x1b[1merr\x1b[0m\n", p, 10+r.IntN(200))
			case 1:
				fmt.Fprintf(w, "  \x1b[31;1merror\x1b[0m: %s/store.go:%d: undefined: \x1b[4mnote.Body\x1b[24m\n", p, 10+r.IntN(200))
			}
		}
		fmt.Fprintf(w, "\x1b[1m▶ Testing\x1b[0m round %d\n", round)
		for pct := 0; pct <= 100; pct += 5 {
			bar := strings.Repeat("█", pct/5) + strings.Repeat("░", 20-pct/5)
			fmt.Fprintf(w, "\r  \x1b[36m%s\x1b[0m %3d%%", bar, pct)
			time.Sleep(60 * time.Millisecond)
		}
		fmt.Fprintf(w, "\n  \x1b[42;30m PASS \x1b[0m %d tests \x1b[2m(%.1fs)\x1b[0m\n\n", 40+r.IntN(30), 0.5+r.Float64()*2)
		time.Sleep(time.Second)
	}
}

func main() {
	a := &app{running: true, title: "Build"}
	args := os.Args[1:]
	if len(args) > 0 {
		a.title = strings.Join(args, " ")
	}
	mygo.App.WhenReady(func() {
		win := mygo.NewWindow(mygo.WindowOptions{
			Title:   "Log",
			Width:   860,
			Height:  560,
			Content: ui.View(a.view),
		})
		if len(args) > 0 {
			go a.run(win, args)
			return
		}
		r, w := io.Pipe()
		go a.stream(win, r)
		go build(w)
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
