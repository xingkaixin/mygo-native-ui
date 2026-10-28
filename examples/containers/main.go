// Containers lists a shop's containers: a Compose project of a web app, a
// worker, Postgres and Redis, and a few on their own. CPU moves every
// second; stop, start, restart and remove containers from their rows.
//
//	go run ./examples/containers
package main

import (
	"log"
	"math"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/containers"
)

type app struct {
	list       containers.State
	containers []containers.Container
	tick       time.Time
	rng        *rand.Rand
}

const mb = 1 << 20

// sample adds a second of CPU use to each running container, keeping
// the last 30.
func (a *app) sample() {
	for i := range a.containers {
		ct := &a.containers[i]
		if ct.Status != containers.Running {
			continue
		}
		last := 3.0
		if len(ct.CPU) > 0 {
			last = ct.CPU[len(ct.CPU)-1]
		}
		next := math.Max(0.1, last+a.rng.NormFloat64()*4)
		ct.CPU = append(ct.CPU[max(0, len(ct.CPU)-29):], math.Min(180, next))
	}
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	now := c.Now()
	if now.Sub(a.tick) >= time.Second {
		a.tick = now
		a.sample()
	}
	c.After(time.Second - now.Sub(a.tick))
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Children(func() {
		ui.Scroll(c).Grow(1).Children(func() {
			ev := containers.View(c, &a.list, a.containers)
			if ev.ID == "" {
				return
			}
			i := slices.IndexFunc(a.containers, func(ct containers.Container) bool { return ct.ID == ev.ID })
			ct := &a.containers[i]
			switch ev.Action {
			case containers.Start, containers.Restart:
				ct.Status, ct.Since, ct.CPU = containers.Running, now, nil
			case containers.Stop:
				ct.Status, ct.Since, ct.ExitCode = containers.Exited, now, 0
			case containers.Remove:
				a.containers = slices.Delete(a.containers, i, i+1)
			}
		})
	})
}

func main() {
	now := time.Now()
	a := &app{rng: rand.New(rand.NewPCG(3, 4)), containers: []containers.Container{
		{ID: "4f1c2a9be7d3", Name: "shop-web-1", Image: "shop/web:2.4.1", Project: "shop", Status: containers.Running, Since: now.Add(-3 * time.Hour), Ports: []containers.Port{{Host: 3000, Container: 3000}}, Memory: 212 * mb, MemoryLimit: 512 * mb},
		{ID: "9ab03e771c20", Name: "shop-worker-1", Image: "shop/worker:2.4.1", Project: "shop", Status: containers.Running, Since: now.Add(-3 * time.Hour), Memory: 148 * mb, MemoryLimit: 256 * mb},
		{ID: "c03d5e1f8a44", Name: "shop-db-1", Image: "postgres:17-alpine", Project: "shop", Status: containers.Running, Since: now.Add(-26 * time.Hour), Ports: []containers.Port{{Host: 5432, Container: 5432}}, Memory: 460 * mb, MemoryLimit: 512 * mb},
		{ID: "7e6f19d02b88", Name: "shop-cache-1", Image: "redis:7.4", Project: "shop", Status: containers.Restarting, Since: now.Add(-2 * time.Minute), Ports: []containers.Port{{Host: 6379, Container: 6379}}, Memory: 12 * mb, MemoryLimit: 128 * mb},
		{ID: "1d2e3f4a5b6c", Name: "shop-migrate-1", Image: "shop/web:2.4.1", Project: "shop", Status: containers.Exited, Since: now.Add(-3 * time.Hour), ExitCode: 0},
		{ID: "a1b2c3d4e5f6", Name: "mailpit", Image: "axllent/mailpit:latest", Status: containers.Running, Since: now.Add(-50 * time.Hour), Ports: []containers.Port{{Host: 8025, Container: 8025}, {Host: 1025, Container: 1025}}, Memory: 34 * mb, MemoryLimit: 2048 * mb},
		{ID: "ff00ee11dd22", Name: "minio", Image: "minio/minio:RELEASE.2026-09", Status: containers.Paused, Since: now.Add(-5 * time.Hour), Ports: []containers.Port{{Host: 9000, Container: 9000}}, Memory: 96 * mb, MemoryLimit: 1024 * mb},
		{ID: "0badc0ffee00", Name: "load-test", Image: "grafana/k6:0.54", Status: containers.Exited, Since: now.Add(-40 * time.Minute), ExitCode: 137},
	}}
	for range 30 {
		a.sample()
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Containers",
			Width:   1180,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
