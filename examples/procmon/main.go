// Procmon watches a made-up Mac's processes, sampled every second: sort
// by a column, switch to Tree, find a process by name, and Stop one to
// quit it and its children.
//
//	go run ./examples/procmon
package main

import (
	"log"
	"math/rand/v2"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/procmon"
)

// proc is a made-up process and how busy it tends to be.
type proc struct {
	procmon.Process
	busy float64
}

const mb = 1 << 20

func seed() []proc {
	p := func(pid, ppid int, name, user string, busy float64, memory uint64, threads int) proc {
		return proc{procmon.Process{PID: pid, PPID: ppid, Name: name, User: user, Memory: memory * mb, Threads: threads}, busy}
	}
	return []proc{
		p(1, 0, "launchd", "root", 0.3, 18, 4),
		p(142, 1, "WindowServer", "_windowserver", 14, 690, 22),
		p(388, 1, "Finder", "alex", 0.6, 210, 9),
		p(391, 1, "Dock", "alex", 0.4, 96, 5),
		p(402, 1, "Safari", "alex", 4, 420, 31),
		p(415, 402, "Safari Web Content", "alex", 9, 380, 18),
		p(416, 402, "Safari Web Content", "alex", 2, 160, 14),
		p(417, 402, "Safari Networking", "alex", 1, 48, 8),
		p(520, 1, "Code", "alex", 6, 310, 34),
		p(523, 520, "Code Helper (GPU)", "alex", 5, 240, 12),
		p(524, 520, "Code Helper (Renderer)", "alex", 11, 520, 21),
		p(530, 520, "gopls", "alex", 3, 880, 19),
		p(610, 1, "Terminal", "alex", 1, 120, 7),
		p(612, 610, "zsh", "alex", 0.1, 6, 1),
		p(650, 612, "go", "alex", 22, 340, 16),
		p(655, 650, "compile", "alex", 78, 410, 14),
		p(201, 1, "mds", "root", 1.5, 64, 11),
		p(204, 201, "mds_stores", "root", 3, 92, 7),
		p(220, 1, "kernel_task", "root", 6, 2400, 290),
		p(330, 1, "Spotlight", "alex", 0.2, 140, 6),
		p(705, 1, "Music", "alex", 1.2, 260, 23),
	}
}

type app struct {
	procs   []proc
	monitor procmon.State
	sampled time.Time
}

func (a *app) sample() {
	out := make([]procmon.Process, len(a.procs))
	for i := range a.procs {
		p := &a.procs[i]
		p.CPU = max(0, p.busy*(0.5+rand.Float64())+rand.NormFloat64()*0.3)
		drift := int64(rand.NormFloat64() * 2 * mb)
		p.Memory = uint64(max(int64(mb), int64(p.Memory)+drift))
		out[i] = p.Process
	}
	a.monitor.Sample(out)
}

// quit takes the process pid and its children out.
func (a *app) quit(pid int) {
	gone := map[int]bool{pid: true}
	for changed := true; changed; {
		changed = false
		for _, p := range a.procs {
			if gone[p.PPID] && !gone[p.PID] {
				gone[p.PID], changed = true, true
			}
		}
	}
	kept := a.procs[:0]
	for _, p := range a.procs {
		if !gone[p.PID] {
			kept = append(kept, p)
		}
	}
	a.procs = kept
	a.sample()
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	if c.Now().Sub(a.sampled) >= time.Second {
		a.sampled = c.Now()
		a.sample()
	}
	c.After(time.Second - c.Now().Sub(a.sampled))
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		if pid, _, ok := procmon.View(c, &a.monitor).Quit(); ok {
			a.quit(pid)
		}
	})
}

func main() {
	a := &app{procs: seed()}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Procmon",
			Width:   900,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
