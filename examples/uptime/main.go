// Uptime is the status page of a SaaS: eight services over the last 90
// days, generated with their incidents, a database failover three weeks
// ago, search down and webhooks slow today, and an assistant watched only
// for six weeks. Point at a bar to read its day; make the window narrow
// to show 60 or 30 days.
//
//	go run ./examples/uptime
package main

import (
	"log"
	"math/rand/v2"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/uptime"
)

const days = 90

// catalog is the services of the page: their names, the days they have
// been watched, and the titles of what goes wrong with them.
var catalog = []struct {
	name    string
	watched int
	titles  []string
}{
	{"API", days, []string{"Elevated API error rates", "Increased API latency", "API requests timing out in us-east"}},
	{"Dashboard", days, []string{"Dashboard fails to load for some users", "Slow page loads"}},
	{"Authentication", days, []string{"Sign-in failures with SAML SSO", "Delayed two-factor codes"}},
	{"Webhooks", days, []string{"Delayed webhook deliveries", "Webhook retries backing up"}},
	{"Search", days, []string{"Search results out of date", "Search queries failing"}},
	{"File storage", days, []string{"Uploads failing in eu-west", "Slow downloads from the CDN"}},
	{"Email delivery", days, []string{"Outbound email delayed", "Bounces from one mail provider"}},
	{"AI assistant", 41, []string{"Assistant responses timing out", "Degraded answer quality"}},
}

// history returns the services over the days to today, from a seeded
// generator so that the page is the same each run.
func history() []uptime.Service {
	r := rand.New(rand.NewPCG(7, 90))
	services := make([]uptime.Service, len(catalog))
	for i, s := range catalog {
		sv := uptime.Service{Name: s.name, Days: make([]uptime.Day, s.watched)}
		for k := range sv.Days {
			if r.Float64() < 0.04 {
				minutes := 3 + r.ExpFloat64()*35
				sv.Days[k].Incidents = append(sv.Days[k].Incidents, uptime.Incident{
					Title:    s.titles[r.IntN(len(s.titles))],
					Duration: time.Duration(minutes) * time.Minute,
				})
			}
		}
		services[i] = sv
	}
	failover := uptime.Incident{Title: "Database failover", Duration: 2*time.Hour + 10*time.Minute}
	for _, i := range []int{0, 1, 2} {
		d := &services[i].Days[days-1-23]
		d.Incidents = append(d.Incidents, failover)
	}
	services[3].Status = uptime.Degraded
	today := &services[3].Days[days-1]
	today.Incidents = append(today.Incidents, uptime.Incident{Title: "Delayed webhook deliveries", Duration: 80 * time.Minute})
	services[4].Status = uptime.Outage
	today = &services[4].Days[days-1]
	today.Incidents = append(today.Incidents, uptime.Incident{Title: "Search queries failing", Duration: 47 * time.Minute})
	for i := range services {
		for k := range services[i].Days {
			d := &services[i].Days[k]
			var down time.Duration
			for _, in := range d.Incidents {
				down += in.Duration
			}
			d.Uptime = 1 - min(1, down.Hours()/24)
		}
	}
	return services
}

type app struct {
	services []uptime.Service
	today    time.Time
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).FillWidth().MaxWidth(t.Space(240)).Margin(0, ui.Auto).Padding(t.Space(8)).Gap(t.Space(5)).Children(func() {
			ui.Text(c, "Acme Cloud status").Bold().FontSize(t.FontSize + 8)
			uptime.View(c, a.services, a.today)
		})
	})
}

func main() {
	a := &app{services: history(), today: time.Now()}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Uptime",
			Width:   1000,
			Height:  860,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
