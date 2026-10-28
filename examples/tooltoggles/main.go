// Tooltoggles sets up which of five servers' tools an agent may call: the
// ones that only read are on, and those that write start off.
//
//	go run ./examples/tooltoggles
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/tooltoggles"
)

type app struct {
	tools   tooltoggles.State
	servers []tooltoggles.Server
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Scroll(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).FillWidth().Padding(t.Space(8)).Children(func() {
			tooltoggles.View(c, &a.tools, &a.servers)
		})
	})
}

// server returns a server whose tools are on unless they write.
func server(name string, tools ...tooltoggles.Tool) tooltoggles.Server {
	for i := range tools {
		tools[i].On = !tools[i].Writes
	}
	return tooltoggles.Server{Name: name, Tools: tools}
}

func main() {
	a := &app{servers: []tooltoggles.Server{
		server("GitHub",
			tooltoggles.Tool{Name: "search_code", Description: "Search code across repositories"},
			tooltoggles.Tool{Name: "get_file", Description: "Read a file at a ref"},
			tooltoggles.Tool{Name: "list_pull_requests", Description: "List the open pull requests of a repository"},
			tooltoggles.Tool{Name: "get_pull_request_diff", Description: "Read the diff of a pull request"},
			tooltoggles.Tool{Name: "create_issue", Description: "Open an issue", Writes: true},
			tooltoggles.Tool{Name: "create_pull_request", Description: "Open a pull request from a branch", Writes: true},
			tooltoggles.Tool{Name: "merge_pull_request", Description: "Merge a pull request", Writes: true},
		),
		server("Linear",
			tooltoggles.Tool{Name: "search_issues", Description: "Search issues by text, team or state"},
			tooltoggles.Tool{Name: "get_issue", Description: "Read an issue and its comments"},
			tooltoggles.Tool{Name: "create_issue", Description: "Create an issue in a team", Writes: true},
			tooltoggles.Tool{Name: "update_issue", Description: "Change an issue's state, assignee or estimate", Writes: true},
		),
		server("Filesystem",
			tooltoggles.Tool{Name: "read_file", Description: "Read a file under the allowed folders"},
			tooltoggles.Tool{Name: "list_directory", Description: "List a folder's entries"},
			tooltoggles.Tool{Name: "search_files", Description: "Find files by name"},
			tooltoggles.Tool{Name: "write_file", Description: "Create or overwrite a file", Writes: true},
			tooltoggles.Tool{Name: "move_file", Description: "Move or rename a file", Writes: true},
		),
		server("Slack",
			tooltoggles.Tool{Name: "search_messages", Description: "Search messages in channels you are in"},
			tooltoggles.Tool{Name: "read_thread", Description: "Read a thread's replies"},
			tooltoggles.Tool{Name: "post_message", Description: "Post a message to a channel", Writes: true},
		),
		server("Postgres",
			tooltoggles.Tool{Name: "query", Description: "Run a read-only SQL query"},
			tooltoggles.Tool{Name: "list_tables", Description: "List the tables and their columns"},
		),
	}}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Tools",
			Width:   620,
			Height:  720,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
