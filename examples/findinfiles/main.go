// Findinfiles searches a small TypeScript project: type a name, such as
// fetchUser, open a match to see its line, then show the replace field
// to rename it across the files. Try a regular expression with a group,
// as user(\w+), replaced by account$1.
//
//	go run ./examples/findinfiles
package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/findinfiles"
)

var project = []findinfiles.File{
	{Path: "src/api/users.ts", Text: `import { http } from "./http";

export async function fetchUser(id: string) {
  const res = await http.get(` + "`/users/${id}`" + `);
  return res.json();
}

export async function fetchUsers() {
  return http.get("/users").then((r) => r.json());
}
`},
	{Path: "src/api/http.ts", Text: `export const http = {
  get: (url: string) => fetch(BASE + url),
};

const BASE = "https://api.example.com";
`},
	{Path: "src/pages/Profile.tsx", Text: `import { fetchUser } from "../api/users";

export function Profile({ userId }: { userId: string }) {
  const user = useQuery(() => fetchUser(userId));
  if (!user) return <Spinner />;
  return <h1>{user.name}</h1>;
}
`},
	{Path: "src/pages/Team.tsx", Text: `import { fetchUsers } from "../api/users";

export function Team() {
  const users = useQuery(fetchUsers);
  return <ul>{users.map((u) => <li key={u.id}>{u.name}</li>)}</ul>;
}
`},
	{Path: "src/hooks/useQuery.ts", Text: `export function useQuery<T>(load: () => Promise<T>) {
  const [data, setData] = useState<T>();
  useEffect(() => { load().then(setData); }, []);
  return data;
}
`},
	{Path: "README.md", Text: `# Team directory

Call fetchUser with an id to load one person, or fetchUsers for everyone.
`},
}

type app struct {
	search findinfiles.State
	files  []findinfiles.File
	open   findinfiles.Match
	note   string
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Width(380).FillHeight().Background(t.Surface).Padding(t.Space(3)).Gap(t.Space(2)).Children(func() {
			ui.Text(c, "Search").Bold()
			ev := findinfiles.View(c, &a.search, &a.files)
			ev.Grow(1)
			if ev.Opened.Path != "" {
				a.open = ev.Opened
			}
			if ev.Replaced > 0 {
				a.note = fmt.Sprintf("Replaced %d %s", ev.Replaced, map[bool]string{true: "match", false: "matches"}[ev.Replaced == 1])
			}
			if a.note != "" {
				ui.Text(c, a.note).FontSize(t.FontSize - 1).TextColor(t.TextMuted)
			}
		})
		ui.Column(c).Grow(1).FillHeight().Padding(t.Space(5)).Gap(t.Space(3)).Children(func() {
			text := ""
			for _, f := range a.files {
				if f.Path == a.open.Path {
					text = f.Text
				}
			}
			if text == "" {
				ui.Text(c, "Open a match to see it here").TextColor(t.TextMuted)
				return
			}
			ui.Text(c, fmt.Sprintf("%s:%d:%d", a.open.Path, a.open.Line, a.open.Col)).Bold()
			ui.Scroll(c).Grow(1).Children(func() {
				for i, l := range strings.Split(strings.TrimSuffix(text, "\n"), "\n") {
					row := ui.Row(c.Key(i)).Gap(t.Space(4)).Padding(1, t.Space(2)).Font("monospace").FontSize(t.FontSize - 1)
					if i+1 == a.open.Line {
						row.Background(t.Warning.Alpha(0.18))
					}
					row.Children(func() {
						ui.Text(c, fmt.Sprint(i+1)).Width(t.Space(6)).TextAlign(ui.End).TextColor(t.TextMuted)
						ui.Text(c, l).NoWrap()
					})
				}
			})
		})
	})
}

func main() {
	a := &app{files: project}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Find in Files",
			Width:   1000,
			Height:  640,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
