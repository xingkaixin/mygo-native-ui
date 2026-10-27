// Stringcatalog translates a notes app's strings into French, German,
// Japanese and Russian: choose a language on the left, type the
// translations in place, show only the missing ones, and edit an English
// text to see its translations marked for review.
//
//	go run ./examples/stringcatalog
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/stringcatalog"
)

type tr = stringcatalog.Translation

func text(v string) *tr { return &tr{Value: v} }

func review(v string) *tr { return &tr{Value: v, Review: true} }

func forms(kv ...string) *tr {
	t := &tr{Forms: map[string]string{}}
	for i := 0; i+1 < len(kv); i += 2 {
		t.Forms[kv[i]] = kv[i+1]
	}
	return t
}

var catalog = stringcatalog.Catalog{
	Source:    "en",
	Languages: []string{"fr", "de", "ja", "ru"},
	Entries: []stringcatalog.Entry{
		{Key: "notes.new", Comment: "Toolbar button that starts a note", Text: map[string]*tr{
			"en": text("New Note"), "fr": text("Nouvelle note"), "de": text("Neue Notiz"), "ja": text("新規メモ"), "ru": text("Новая заметка"),
		}},
		{Key: "notes.count", Comment: "Under a folder's name", Plural: true, Text: map[string]*tr{
			"en": forms("one", "%d note", "other", "%d notes"),
			"fr": forms("one", "%d note", "many", "%d de notes", "other", "%d notes"),
			"de": forms("one", "%d Notiz", "other", "%d Notizen"),
			"ja": forms("other", "%d 件のメモ"),
			"ru": forms("one", "%d заметка", "few", "%d заметки"),
		}},
		{Key: "notes.pin", Comment: "Context menu item", Text: map[string]*tr{
			"en": text("Pin Note"), "fr": text("Épingler la note"), "de": review("Notiz anheften"), "ja": text("メモをピンで固定"),
		}},
		{Key: "notes.delete.confirm", Comment: "Alert title before a note is deleted", Text: map[string]*tr{
			"en": text("Delete this note? You can't undo this."), "fr": review("Supprimer cette note ?"), "de": text("Diese Notiz löschen? Dies kann nicht widerrufen werden."),
		}},
		{Key: "notes.shared", Comment: "Badge on a note others can edit", Text: map[string]*tr{
			"en": text("Shared"), "fr": text("Partagée"), "de": text("Geteilt"), "ja": text("共有"), "ru": text("Общая"),
		}},
		{Key: "folder.empty", Comment: "Shown in a folder with no notes", Text: map[string]*tr{
			"en": text("No Notes"), "fr": text("Aucune note"), "ja": review("メモなし"),
		}},
		{Key: "search.placeholder", Text: map[string]*tr{
			"en": text("Search all notes"), "de": text("Alle Notizen durchsuchen"), "ja": text("すべてのメモを検索"), "ru": text("Поиск по заметкам"),
		}},
		{Key: "sync.status", Comment: "Footer; %@ is a time, as 9:41", Text: map[string]*tr{
			"en": text("Synced at %@"), "fr": text("Synchronisé à %@"), "de": text("Synchronisiert um %@"), "ja": text("%@ に同期済み"), "ru": text("Синхронизировано в %@"),
		}},
		{Key: "attachments.count", Comment: "On a note's row", Plural: true, Text: map[string]*tr{
			"en": forms("one", "%d attachment", "other", "%d attachments"),
			"de": forms("one", "%d Anhang", "other", "%d Anhänge"),
			"ru": forms("one", "%d вложение", "few", "%d вложения", "many", "%d вложений", "other", "%d вложения"),
		}},
		{Key: "settings.title", Text: map[string]*tr{
			"en": text("Settings"), "fr": text("Réglages"), "de": text("Einstellungen"), "ja": text("設定"), "ru": text("Настройки"),
		}},
	},
}

type app struct {
	catalog stringcatalog.Catalog
	editor  stringcatalog.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Children(func() {
		stringcatalog.View(c, &a.editor, &a.catalog).Grow(1)
	})
}

func main() {
	a := &app{catalog: catalog}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Localizable",
			Width:   1100,
			Height:  680,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
