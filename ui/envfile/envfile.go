// Package envfile edits .env files side by side, as the environments of a
// project keep them: a row for each variable any of them sets, a column
// for each file, and its value in each, edited in place. Values of keys
// that look secret, such as API_KEY or DB_PASSWORD, show as dots until
// their row's eye reveals them. A value a file lacks shows as missing in
// the warning color, a key set twice in one file and a key that is not a
// valid name in the danger color, with why in their tooltips. A field
// under the rows adds a variable to every file.
//
//	envfile.View(c, &app.editor, &app.files)
//	save(app.files[0].Name, envfile.Format(app.files[0].Entries))
package envfile

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/egoist/mygo/ui"
)

// Entry is a line of a .env file: a variable and its value, or a comment
// or blank line, with no key.
type Entry struct {
	Key, Value, Comment string
}

// File is a .env file: its name, as "production", and its lines.
type File struct {
	Name    string
	Entries []Entry
}

// Parse reads a .env file: KEY=value lines, with or without export
// before them, values bare, in double quotes with \n escapes, or in single
// quotes, bare values ending at " #", and comment lines kept as they are.
func Parse(s string) []Entry {
	var out []Entry
	for line := range strings.SplitSeq(strings.ReplaceAll(s, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		key, value, ok := strings.Cut(strings.TrimPrefix(trimmed, "export "), "=")
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || !ok {
			out = append(out, Entry{Comment: line})
			continue
		}
		value = strings.TrimSpace(value)
		switch {
		case strings.HasPrefix(value, `"`):
			if v, err := strconv.Unquote(value[:strings.LastIndex(value, `"`)+1]); err == nil {
				value = v
			}
		case strings.HasPrefix(value, "'") && strings.Count(value, "'") > 1:
			value = value[1:strings.LastIndex(value, "'")]
		default:
			if i := strings.Index(value, " #"); i >= 0 {
				value = strings.TrimSpace(value[:i])
			}
		}
		out = append(out, Entry{Key: strings.TrimSpace(key), Value: value})
	}
	for len(out) > 0 && out[len(out)-1] == (Entry{}) {
		out = out[:len(out)-1]
	}
	return out
}

// Format writes entries as a .env file, quoting values with spaces,
// quotes, # or line breaks.
func Format(entries []Entry) string {
	var b strings.Builder
	for _, e := range entries {
		switch {
		case e.Key == "":
			b.WriteString(e.Comment)
		case strings.ContainsAny(e.Value, " \"'#\n\t"):
			b.WriteString(e.Key + "=" + strconv.Quote(e.Value))
		default:
			b.WriteString(e.Key + "=" + e.Value)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

var (
	validKey  = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	secretKey = regexp.MustCompile(`(?i)(SECRET|TOKEN|PASSWORD|PASSWD|PRIVATE|API_?KEY|_KEY$|DSN)`)
)

// Secret reports whether key names a value to hide, as API_KEY or
// DB_PASSWORD do.
func Secret(key string) bool { return secretKey.MatchString(key) }

// State is what an editor keeps: the rows revealed and the key being
// added. Keep it in the app's state, one for each editor.
type State struct {
	revealed map[string]bool
	adding   string
}

var (
	eye    = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M2 12s3.6-7 10-7 10 7 10 7-3.6 7-10 7S2 12 2 12z"/><circle cx="12" cy="12" r="3"/></svg>`))
	eyeOff = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M2 12s3.6-7 10-7c2 0 3.7.7 5.1 1.6M22 12s-3.6 7-10 7c-2 0-3.7-.7-5.1-1.6"/><path d="M3 3l18 18"/></svg>`))
)

// View creates the editor of *files. Changed reports an edit.
func View(c *ui.Context, s *State, files *[]File) ui.Element {
	t := c.Theme()
	if s.revealed == nil {
		s.revealed = map[string]bool{}
	}
	var keys []string
	for _, f := range *files {
		for _, e := range f.Entries {
			if e.Key != "" && !slices.Contains(keys, e.Key) {
				keys = append(keys, e.Key)
			}
		}
	}
	changed := false
	tracks := []ui.Track{ui.Fixed(t.Space(56))}
	for range *files {
		tracks = append(tracks, ui.Fr(1))
	}
	tracks = append(tracks, ui.Fixed(t.Space(8)))
	root := ui.Column(c).Gap(t.Space(3))
	root.Children(func() {
		ui.Grid(c).ColumnTracks(tracks...).GapX(t.Space(2)).GapY(t.Space(1)).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Variable").Bold().FontSize(t.FontSize - 1).TextColor(t.TextMuted)
			for i, f := range *files {
				missing := 0
				for _, k := range keys {
					if !slices.ContainsFunc(f.Entries, func(e Entry) bool { return e.Key == k }) {
						missing++
					}
				}
				ui.Row(c.Key([2]any{"head", i})).Gap(t.Space(2)).AlignItems(ui.Center).Children(func() {
					ui.Text(c, f.Name).Bold().FontSize(t.FontSize - 1)
					if missing > 0 {
						ui.Text(c, fmt.Sprintf("%d missing", missing)).FontSize(t.FontSize - 2).TextColor(t.Warning)
					}
				})
			}
			ui.Box(c)
			for _, k := range keys {
				if s.row(c, k, files) {
					changed = true
				}
			}
		})
		ui.Row(c).Gap(t.Space(2)).Children(func() {
			field := ui.TextInput(c, &s.adding).Placeholder("NEW_VARIABLE").Font("monospace").Width(t.Space(56)).Label("New variable")
			add := strings.TrimSpace(s.adding)
			ok := validKey.MatchString(add) && !slices.Contains(keys, add)
			if (ui.Button(c, "Add Variable").Disabled(!ok).Clicked() || field.Submitted()) && ok {
				list := slices.Clone(*files)
				for i := range list {
					list[i].Entries = append(slices.Clone(list[i].Entries), Entry{Key: add})
				}
				*files, s.adding, changed = list, "", true
			}
		})
	})
	if changed {
		root.SetChanged()
	}
	return root
}

// row creates the cells of key: its name, its value in each file, and its
// eye, and reports an edit.
func (s *State) row(c *ui.Context, key string, files *[]File) bool {
	t := c.Theme()
	changed := false
	problem := ""
	if !validKey.MatchString(key) {
		problem = "Not a valid name: letters, digits and _, not starting with a digit"
	}
	for _, f := range *files {
		n := 0
		for _, e := range f.Entries {
			if e.Key == key {
				n++
			}
		}
		if n > 1 {
			problem = fmt.Sprintf("Set %d times in %s; the last one wins", n, f.Name)
		}
	}
	name := ui.Text(c.Key([2]any{key, "name"}), key).Font("monospace").FontSize(t.FontSize - 1).SingleLine().Ellipsis("…")
	if problem != "" {
		name.TextColor(t.Danger).Tooltip(problem)
	}
	secret := Secret(key) && !s.revealed[key]
	for i := range *files {
		f := &(*files)[i]
		at := -1
		for k, e := range f.Entries {
			if e.Key == key {
				at = k
			}
		}
		if at < 0 {
			add := ui.ButtonBase(c.Key([2]any{key, i})).Height(t.Space(8)).Radius(t.Radius).Border(1, t.Warning.Alpha(0.5)).BorderStyle(ui.BorderDashed).
				Center().Label("Add " + key + " to " + f.Name).Tooltip("Missing in " + f.Name)
			if add.Hovered() {
				add.Background(t.Warning.Alpha(0.08))
			}
			add.Children(func() {
				ui.Text(c, "Missing").FontSize(t.FontSize - 1).TextColor(t.Warning)
			})
			if add.Clicked() {
				list := slices.Clone(*files)
				list[i].Entries = append(slices.Clone(list[i].Entries), Entry{Key: key})
				*files, changed = list, true
			}
			continue
		}
		value := f.Entries[at].Value
		in := ui.TextInput(c.Key([2]any{key, i}), &value).Font("monospace").FontSize(t.FontSize - 1).Label(key + " in " + f.Name)
		if secret {
			in.Password()
		}
		if value == "" {
			in.Placeholder("empty")
		}
		if in.Changed() {
			list := slices.Clone(*files)
			list[i].Entries = slices.Clone(list[i].Entries)
			list[i].Entries[at].Value = value
			*files, changed = list, true
		}
	}
	if Secret(key) {
		ic := eye
		if s.revealed[key] {
			ic = eyeOff
		}
		b := ui.ButtonBase(c.Key([2]any{key, "eye"})).Size(t.Space(7), t.Space(7)).Radius(t.Radius).Center().Label("Reveal " + key)
		if b.Hovered() {
			b.Background(t.SurfaceHover)
		}
		if b.Clicked() {
			s.revealed[key] = !s.revealed[key]
		}
		b.Children(func() { ui.Icon(c, ic).Size(t.Space(4), t.Space(4)).TextColor(t.TextMuted) })
	} else {
		ui.Box(c.Key([2]any{key, "eye"}))
	}
	return changed
}
