// Bilingual translates a made-up app's English strings into Simplified
// Chinese: some confirmed, some drafts, one missing a placeholder, and
// some left to do. Press Cmd-Enter in a translation to confirm it and go
// to the next.
//
//	go run ./examples/bilingual
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/bilingual"
)

var strings = []bilingual.Segment{
	{Source: "Welcome back, {name}!", Target: "欢迎回来，{name}！", Confirmed: true},
	{Source: "You have %d unread messages.", Target: "你有 %d 条未读消息。", Confirmed: true},
	{Source: "Your trial ends in {days} days.", Target: "你的试用期还剩几天。", Note: "Banner at the top of the dashboard"},
	{Source: "Save changes", Target: "保存更改"},
	{Source: "Are you sure you want to delete <b>{file}</b>? This cannot be undone.", Note: "Confirmation dialog"},
	{Source: "Invite people to {workspace}"},
	{Source: "Last edited by {user} on {date}", Target: "{user} 于 {date} 最后编辑", Confirmed: true},
	{Source: "Upload failed. Check your connection and try again."},
	{Source: "Showing %d–%d of %d results"},
	{Source: "Sign out of all devices", Note: "Account settings, under Security"},
}

type app struct {
	segments []bilingual.Segment
	editor   bilingual.State
	saves    int
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		if bilingual.View(c, &a.editor, "English", "简体中文", a.segments).Grow(1).Changed() {
			a.saves++
		}
	})
}

func main() {
	a := &app{segments: strings}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Translation",
			Width:   900,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
