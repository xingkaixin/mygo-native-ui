// Graphql explores the schema of a made-up issue tracker: queries,
// mutations and a subscription, an interface and a union, enums, inputs
// and a deprecated field. Pass a schema file to explore it instead.
//
//	go run ./examples/graphql [schema.graphql]
package main

import (
	"log"
	"os"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/graphql"
)

const sdl = `
"""
Where every read starts.
"""
type Query {
  "The signed-in user."
  viewer: User!
  "A repository by its owner and name."
  repository(owner: String!, name: String!): Repository
  "Looks up any object by its global ID."
  node(id: ID!): Node
  """
  Searches issues, users and repositories. Results come back best match
  first, a page at a time.
  """
  search(query: String!, type: SearchType = ISSUE, first: Int = 20, after: String): SearchConnection!
}

type Mutation {
  "Opens an issue."
  createIssue(input: CreateIssueInput!): Issue!
  "Closes an issue, saying why."
  closeIssue(id: ID!, reason: CloseReason = COMPLETED): Issue!
  addLabel(issueId: ID!, label: String!): Issue!
}

type Subscription {
  "Each comment as it's posted on the issue."
  commentAdded(issueId: ID!): Comment!
}

"An object with a global ID."
interface Node {
  id: ID!
}

"""
Someone with an account. Users own repositories and open issues.
"""
type User implements Node {
  id: ID!
  login: String!
  name: String
  avatarUrl(size: Int = 64): URL!
  repositories(first: Int = 10, after: String): [Repository!]!
  createdAt: DateTime!
}

type Repository implements Node {
  id: ID!
  owner: User!
  name: String!
  "The repository's description, in **Markdown**."
  description: String
  isPrivate: Boolean!
  issues(states: [IssueState!], first: Int = 25, after: String, orderBy: IssueOrder): [Issue!]!
  stargazerCount: Int!
  stars: Int! @deprecated(reason: "Use ` + "`stargazerCount`" + `.")
}

type Issue implements Node {
  id: ID!
  number: Int!
  title: String!
  body: String
  state: IssueState!
  author: User
  labels: [String!]!
  comments(first: Int = 30): [Comment!]!
  closedAt: DateTime
}

type Comment implements Node {
  id: ID!
  author: User
  body: String!
  createdAt: DateTime!
}

enum IssueState {
  OPEN
  CLOSED
}

enum CloseReason {
  "The work is done."
  COMPLETED
  "It won't be done."
  NOT_PLANNED
  DUPLICATE @deprecated(reason: "Close as NOT_PLANNED and link the original.")
}

enum SearchType {
  ISSUE
  USER
  REPOSITORY
}

"Anything a search can find."
union SearchResult = Issue | User | Repository

type SearchConnection {
  nodes: [SearchResult!]!
  totalCount: Int!
  endCursor: String
}

input CreateIssueInput {
  repositoryId: ID!
  title: String!
  body: String
  labels: [String!] = []
}

input IssueOrder {
  field: String! = "CREATED_AT"
  direction: String! = "DESC"
}

"An ISO 8601 date and time, as ` + "`2026-10-28T09:30:00Z`" + `."
scalar DateTime

"An absolute URL."
scalar URL
`

type app struct {
	schema *graphql.Schema
	docs   graphql.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Column(c).Fill().Background(t.Background).Padding(t.Space(6)).Children(func() {
		graphql.View(c, &a.docs, a.schema).Grow(1)
	})
}

func main() {
	src := sdl
	if len(os.Args) > 1 {
		data, err := os.ReadFile(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
		src = string(data)
	}
	schema, err := graphql.Parse(src)
	if err != nil {
		log.Fatal(err)
	}
	a := &app{schema: schema}
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "GraphQL",
			Width:   560,
			Height:  720,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
