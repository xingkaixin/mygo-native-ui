// Iconpicker picks the icon for a folder from a set of fifty line icons.
//
//	go run ./examples/iconpicker
package main

import (
	"log"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/iconpicker"
)

// set is the icons: a name, keywords, and the shapes of a 24-unit icon
// drawn with a 2-unit line.
var set = [][3]string{
	{"home", "house start", `<path d="M3 11l9-7 9 7"/><path d="M5 10v10h14V10"/><path d="M10 20v-6h4v6"/>`},
	{"search", "find magnify look", `<circle cx="11" cy="11" r="7"/><path d="M16 16l5 5"/>`},
	{"settings", "gear preferences options", `<circle cx="12" cy="12" r="3"/><path d="M12 2v4M12 18v4M2 12h4M18 12h4M5 5l3 3M16 16l3 3M5 19l3-3M16 8l3-3"/>`},
	{"user", "person account profile", `<circle cx="12" cy="8" r="4"/><path d="M4 21c0-4 4-6 8-6s8 2 8 6"/>`},
	{"users", "people team group", `<circle cx="9" cy="8" r="3.5"/><path d="M2 20c0-3.5 3-5.5 7-5.5s7 2 7 5.5"/><path d="M16 4.5a3.5 3.5 0 0 1 0 7M18 14.5c2.5.5 4 2.5 4 5.5"/>`},
	{"heart", "love like favorite", `<path d="M12 20s-8-5-8-11a4.5 4.5 0 0 1 8-2.5A4.5 4.5 0 0 1 20 9c0 6-8 11-8 11z"/>`},
	{"star", "favorite rating bookmark", `<path d="M12 3l2.8 5.8 6.2.9-4.5 4.4 1 6.2L12 17.4l-5.5 2.9 1-6.2L3 9.7l6.2-.9z"/>`},
	{"bell", "notification alert alarm", `<path d="M6 16V11a6 6 0 0 1 12 0v5l2 2H4z"/><path d="M10 21h4"/>`},
	{"mail", "email envelope message", `<rect x="3" y="5" width="18" height="14" rx="2"/><path d="M3 7l9 6 9-6"/>`},
	{"calendar", "date schedule event", `<rect x="3" y="5" width="18" height="16" rx="2"/><path d="M3 10h18M8 3v4M16 3v4"/>`},
	{"clock", "time hour watch", `<circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/>`},
	{"camera", "photo picture", `<path d="M4 8h3l2-3h6l2 3h3v11H4z"/><circle cx="12" cy="13" r="3.5"/>`},
	{"image", "picture photo gallery", `<rect x="3" y="4" width="18" height="16" rx="2"/><circle cx="9" cy="10" r="2"/><path d="M21 17l-5-5-9 8"/>`},
	{"file", "document page paper", `<path d="M6 3h8l4 4v14H6z"/><path d="M14 3v4h4"/>`},
	{"folder", "directory files", `<path d="M3 6h6l2 2h10v11H3z"/>`},
	{"trash", "delete remove bin", `<path d="M4 7h16M9 7V4h6v3M6 7l1 14h10l1-14"/>`},
	{"pencil", "edit write change", `<path d="M15 4l5 5L9 20H4v-5z"/><path d="M13 6l5 5"/>`},
	{"plus", "add new create", `<path d="M12 5v14M5 12h14"/>`},
	{"minus", "remove subtract", `<path d="M5 12h14"/>`},
	{"check", "done ok tick confirm", `<path d="M4 12l5 5L20 6"/>`},
	{"close", "x cancel dismiss", `<path d="M6 6l12 12M18 6L6 18"/>`},
	{"arrow-right", "next forward", `<path d="M4 12h16M14 6l6 6-6 6"/>`},
	{"arrow-left", "back previous", `<path d="M20 12H4M10 6l-6 6 6 6"/>`},
	{"arrow-up", "top increase", `<path d="M12 20V4M6 10l6-6 6 6"/>`},
	{"arrow-down", "bottom decrease", `<path d="M12 4v16M6 14l6 6 6-6"/>`},
	{"download", "save get", `<path d="M12 3v12M7 10l5 5 5-5M4 20h16"/>`},
	{"upload", "send share", `<path d="M12 15V3M7 8l5-5 5 5M4 20h16"/>`},
	{"lock", "secure private password", `<rect x="5" y="11" width="14" height="10" rx="2"/><path d="M8 11V8a4 4 0 0 1 8 0v3"/>`},
	{"unlock", "open public", `<rect x="5" y="11" width="14" height="10" rx="2"/><path d="M8 11V8a4 4 0 0 1 7.5-2"/>`},
	{"cloud", "weather storage sync", `<path d="M7 18a4.5 4.5 0 0 1-.5-9A6 6 0 0 1 18 9.5a4 4 0 0 1-.5 8.5z"/>`},
	{"sun", "light day weather bright", `<circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M2 12h2M20 12h2M5 5l1.5 1.5M17.5 17.5L19 19M5 19l1.5-1.5M17.5 6.5L19 5"/>`},
	{"moon", "dark night sleep", `<path d="M20 14A8 8 0 1 1 10 4a6.5 6.5 0 0 0 10 10z"/>`},
	{"bookmark", "save mark read later", `<path d="M6 3h12v18l-6-4-6 4z"/>`},
	{"tag", "label price category", `<path d="M3 3h8l10 10-8 8L3 11z"/><circle cx="7.5" cy="7.5" r="1.5"/>`},
	{"link", "url chain attach", `<path d="M10 14a4 4 0 0 0 5.7 0l3-3a4 4 0 0 0-5.7-5.7l-1 1"/><path d="M14 10a4 4 0 0 0-5.7 0l-3 3a4 4 0 0 0 5.7 5.7l1-1"/>`},
	{"pin", "location map place", `<path d="M12 21s-7-6.5-7-12a7 7 0 0 1 14 0c0 5.5-7 12-7 12z"/><circle cx="12" cy="9" r="2.5"/>`},
	{"phone", "call telephone", `<path d="M5 3h4l2 5-2.5 1.5a11 11 0 0 0 6 6L16 13l5 2v4a2 2 0 0 1-2 2A16 16 0 0 1 3 5a2 2 0 0 1 2-2z"/>`},
	{"chat", "message comment talk", `<path d="M4 5h16v11H9l-5 4z"/>`},
	{"play", "start video music", `<path d="M7 4l13 8-13 8z"/>`},
	{"pause", "stop wait", `<path d="M8 5v14M16 5v14"/>`},
	{"music", "note song audio", `<path d="M9 18V5l11-2v13"/><circle cx="6.5" cy="18" r="2.5"/><circle cx="17.5" cy="16" r="2.5"/>`},
	{"cart", "shopping buy basket", `<path d="M3 4h3l2.5 11h10l2-8H7"/><circle cx="9.5" cy="19.5" r="1.5"/><circle cx="17.5" cy="19.5" r="1.5"/>`},
	{"gift", "present birthday", `<rect x="3" y="8" width="18" height="5"/><path d="M5 13v8h14v-8M12 8v13M12 8c-3 0-5-1-5-3s3-2 5 3c2-5 5-5 5-3s-2 3-5 3"/>`},
	{"flag", "report mark country", `<path d="M5 21V4M5 4h12l-2 4 2 4H5"/>`},
	{"globe", "world web internet language", `<circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3c3 3 3 15 0 18M12 3c-3 3-3 15 0 18"/>`},
	{"wifi", "wireless network signal", `<path d="M2 9a15 15 0 0 1 20 0M5 12.5a10 10 0 0 1 14 0M8.5 16a5 5 0 0 1 7 0"/><circle cx="12" cy="19.5" r="1"/>`},
	{"battery", "power charge energy", `<rect x="2" y="7" width="17" height="10" rx="2"/><path d="M22 10v4M5 10v4M8 10v4"/>`},
	{"bolt", "flash power lightning fast", `<path d="M13 2L4 14h7l-1 8 9-12h-7z"/>`},
	{"book", "read library docs", `<path d="M4 4h6a2 2 0 0 1 2 2v14a2 2 0 0 0-2-2H4zM20 4h-6a2 2 0 0 0-2 2v14a2 2 0 0 1 2-2h6z"/>`},
	{"code", "developer brackets program", `<path d="M8 7l-5 5 5 5M16 7l5 5-5 5"/>`},
}

type app struct {
	picker iconpicker.State
	icons  []iconpicker.Icon
	icon   string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Padding(t.Space(6)).Background(t.Background).Children(func() {
		iconpicker.View(c, &a.picker, a.icons, &a.icon).Grow(1)
	})
}

func main() {
	a := &app{icon: "folder"}
	for _, ic := range set {
		svg := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><g fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">` + ic[2] + `</g></svg>`
		a.icons = append(a.icons, iconpicker.Icon{Name: ic[0], Keywords: strings.Fields(ic[1]), Image: ui.MustParseSVG([]byte(svg))})
	}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Icon picker",
			Width:   520,
			Height:  560,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
