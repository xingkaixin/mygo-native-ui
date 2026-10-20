// Terminal runs the user's shell, $SHELL or /bin/sh, in a window, in a
// pseudo-terminal it opens with the standard library alone, on macOS and
// Linux. The window takes the title the shell sets, and closes when the
// shell exits.
//
//	go run ./examples/terminal
package main

import (
	"cmp"
	"io"
	"log"
	"os"
	"os/exec"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/terminal"
)

// app is the state the window shows: the terminal, the pseudo-terminal
// the shell runs in once the view has sized it, and the title shown.
type app struct {
	term  terminal.Terminal
	pty   *os.File
	win   *mygo.Window
	title string
}

func (a *app) view(c *ui.Context) {
	if title := a.term.Title(); title != a.title {
		a.title = title
		a.win.SetTitle(cmp.Or(title, "Terminal"))
	}
	terminal.View(c, &a.term).Fill().AutoFocus()
}

// resize starts the shell at the terminal's first size, so that it draws
// its prompt for it, and gives it the sizes that follow.
func (a *app) resize(cols, rows int) {
	if a.pty != nil {
		setSize(a.pty, cols, rows)
		return
	}
	pty, err := startPTY(exec.Command(cmp.Or(os.Getenv("SHELL"), "/bin/sh")), cols, rows)
	if err != nil {
		log.Fatal(err)
	}
	a.pty = pty
	go func() {
		io.Copy(&a.term, pty)
		a.win.Close()
	}()
}

func main() {
	a := &app{}
	a.term.Send = func(b []byte) {
		if a.pty != nil {
			a.pty.Write(b)
		}
	}
	a.term.Resize = a.resize
	mygo.App.WhenReady(func() {
		a.win = mygo.NewWindow(mygo.WindowOptions{
			Title:   "Terminal",
			Width:   760,
			Height:  480,
			Content: ui.View(a.view),
		})
		a.win.OnClosed(func() {
			if a.pty != nil {
				a.pty.Close()
			}
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
