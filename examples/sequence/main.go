// Sequence draws a sequence diagram from the text beside it, as it is
// typed: a sign-in with OAuth 2's authorization code flow, from the user's
// click to the app calling the API with the token it got.
//
//	go run ./examples/sequence
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/sequence"
)

const login = `%% OAuth 2 authorization code flow
actor U as User
participant B as Browser
participant App as Web app
participant Auth as Auth server
participant API

U->>B: Click "Sign in"
B->>+App: GET /login
App-->>-B: 302 to /authorize?client_id&state
B->>+Auth: GET /authorize
Auth-->>-B: Sign-in page
U->>B: Enter email and password
B->>+Auth: POST /authorize
Auth-->>-B: 302 to /callback?code&state
B->>+App: GET /callback?code&state
App->>App: Check state against the cookie
alt State matches
  App->>+Auth: POST /token with code and secret
  Auth-->>-App: Access and refresh tokens
  Note right of App: Keeps tokens in the session
  loop Each page the user opens
    App->>+API: GET /me with Bearer token
    API-->>-App: 200 profile
  end
  App-->>B: 302 to / with session cookie
else State differs
  App-->>-B: 400 Bad request
end
B->>U: Signed in`

type app struct {
	source  string
	split   float32
	diagram sequence.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Split(c, &a.split, func() {
		ui.Column(c).Fill().Padding(t.Space(3)).Children(func() {
			ui.TextArea(c, &a.source).Font("monospace").Label("Source").Grow(1)
		})
	}, func() {
		sequence.View(c, &a.diagram, a.source).Fill()
	}).Fill().Background(t.Background)
}

func main() {
	a := &app{source: login, split: 460}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Sequence",
			Width:   1440,
			Height:  940,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
