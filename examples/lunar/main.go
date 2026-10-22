// Lunar is a month with the Chinese calendar, as Chinese calendar apps
// show it, and a family's dates on their days: birthdays kept by the
// Chinese calendar, which fall on another day each year, and others by the
// Gregorian. Click a day, or move with the arrows, to read it beside the
// month.
//
//	go run ./examples/lunar
package main

import (
	"fmt"
	"log"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/lunar"
)

// date is a day the family keeps every year, as when says: month and day
// of the Chinese calendar when lunar, else of the Gregorian.
type date struct {
	title, when string
	lunar       bool
	month, day  int
	color       ui.Color
}

var dates = []date{
	{"妈妈生日", "每年农历九月十二", true, 9, 12, ui.RGB(255, 149, 0)},
	{"外婆生日", "每年农历腊月初六", true, 12, 6, ui.RGB(255, 45, 85)},
	{"结婚纪念日", "每年10月18日", false, 10, 18, ui.RGB(175, 82, 222)},
	{"小雨生日", "每年3月14日", false, 3, 14, ui.RGB(48, 176, 199)},
}

type app struct {
	day time.Time
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Fill().Background(t.Background).AlignItems(ui.Stretch).Children(func() {
		ui.Column(c).Grow(1).Padding(t.Space(5)).Children(func() {
			lunar.View(c, &a.day, marks).Grow(1)
		})
		ui.Divider(c).Vertical()
		a.side(c)
	})
}

// side writes out the day chosen and lists the family's dates, those on
// the day first.
func (a *app) side(c *ui.Context) {
	t := c.Theme()
	weekdays := []rune("日一二三四五六")
	ui.Column(c).Width(t.Space(68)).Padding(t.Space(6), t.Space(5)).Gap(t.Space(5)).Lang("zh-Hans").Children(func() {
		ui.Column(c).Gap(t.Space(0.5)).Children(func() {
			ui.Text(c, "星期"+string(weekdays[a.day.Weekday()])).FontSize(t.FontSize - 1).Bold().TextColor(t.Accent)
			ui.Text(c, fmt.Sprintf("%d年%d月%d日", a.day.Year(), a.day.Month(), a.day.Day())).FontSize(t.FontSize + 6).Bold()
		})
		lunar.Detail(c, a.day)
		ui.Divider(c)
		ui.Column(c).Gap(t.Space(3)).Children(func() {
			ui.Text(c, "纪念日").FontSize(t.FontSize - 1).Bold().TextColor(t.TextMuted)
			on := marks(a.day)
			for i, d := range dates {
				today := false
				for _, m := range on {
					today = today || m.Title == d.title
				}
				ui.Row(c.Key(i)).Gap(t.Space(2.5)).AlignItems(ui.Stretch).Children(func() {
					ui.Box(c).Width(3).Radius(1.5).Background(d.color)
					ui.Column(c).Gap(t.Space(0.5)).Grow(1).Children(func() {
						title := ui.Text(c, d.title)
						if today {
							title.Bold()
						}
						ui.Text(c, d.when).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
					})
					if today {
						ui.Text(c, "就是这天").FontSize(t.FontSize - 1).Bold().TextColor(d.color).AlignSelf(ui.Center)
					}
				})
			}
		})
	})
}

// marks returns the family's dates on day. A date of the Chinese calendar
// is kept in its month, not in a leap month of the same number.
func marks(day time.Time) []lunar.Mark {
	l, ok := lunar.Of(day)
	var out []lunar.Mark
	for _, d := range dates {
		if d.lunar && ok && !l.Leap && l.Month == d.month && l.Day == d.day ||
			!d.lunar && int(day.Month()) == d.month && day.Day() == d.day {
			out = append(out, lunar.Mark{Title: d.title, Color: d.color})
		}
	}
	return out
}

func main() {
	y, m, d := time.Now().Date()
	a := &app{day: time.Date(y, m, d, 0, 0, 0, 0, time.Local)}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "农历",
			Width:   1080,
			Height:  720,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
