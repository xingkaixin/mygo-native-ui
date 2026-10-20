// Treemap shows what takes the room in a folder, as DaisyDisk and
// WinDirStat do: the folder given, or the Go installation, read in the
// background. Click a folder to go into it, and the path above, Escape or
// Backspace to go back up.
//
//	go run ./examples/treemap [folder]
package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/treemap"
)

type app struct {
	dir   string
	files atomic.Int64
	mu    sync.Mutex
	root  *treemap.Node // nil until read
	tree  treemap.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	a.mu.Lock()
	root := a.root
	a.mu.Unlock()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		if root == nil {
			c.After(100 * time.Millisecond)
			ui.Column(c).Grow(1).Center().Gap(t.Space(3)).Children(func() {
				ui.Spinner(c).Label("Reading")
				ui.Text(c, fmt.Sprintf("Reading %s… %d files", a.dir, a.files.Load())).TextColor(t.TextMuted).FontFeatures("tnum")
			})
			return
		}
		treemap.View(c, &a.tree, root, bytes).Grow(1)
	})
}

// read returns the tree of the folder or file at path, named name,
// counting the files it reads.
func (a *app) read(path, name string) *treemap.Node {
	n := &treemap.Node{Name: name}
	entries, err := os.ReadDir(path)
	if err != nil {
		return n
	}
	for _, e := range entries {
		switch {
		case e.Type()&os.ModeSymlink != 0:
		case e.IsDir():
			kid := a.read(filepath.Join(path, e.Name()), e.Name())
			n.Children = append(n.Children, kid)
			n.Size += kid.Size
		default:
			info, err := e.Info()
			if err != nil {
				continue
			}
			a.files.Add(1)
			n.Children = append(n.Children, &treemap.Node{Name: e.Name(), Size: info.Size()})
			n.Size += info.Size()
		}
	}
	return n
}

// bytes formats n as Finder does, in powers of 1000: "12 KB", "1.4 GB".
func bytes(n int64) string {
	if n < 1000 {
		return fmt.Sprintf("%d bytes", n)
	}
	f, unit := float64(n), ""
	for _, u := range []string{"KB", "MB", "GB", "TB"} {
		f, unit = f/1000, u
		if f < 1000 {
			break
		}
	}
	if f < 10 {
		return fmt.Sprintf("%.1f %s", f, unit)
	}
	return fmt.Sprintf("%.0f %s", f, unit)
}

func main() {
	dir := ""
	if len(os.Args) > 1 {
		dir = os.Args[1]
	} else if out, err := exec.Command("go", "env", "GOROOT").Output(); err == nil {
		dir = strings.TrimSpace(string(out))
	}
	if dir == "" {
		log.Fatal("usage: treemap folder")
	}
	dir, _ = filepath.Abs(dir)
	a := &app{dir: dir}
	go func() {
		root := a.read(dir, filepath.Base(dir))
		a.mu.Lock()
		a.root = root
		a.mu.Unlock()
	}()
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Treemap",
			Width:   1100,
			Height:  720,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
