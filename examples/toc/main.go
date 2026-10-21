// Toc shows the guide to Lantern, a made-up generator of documentation
// sites, with its table of contents on the right. Scroll the guide to
// light the section being read, and click a heading to go to it; how much
// of the guide is read shows above the headings.
//
//	go run ./examples/toc
package main

import (
	"log"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"

	"github.com/xingkaixin/mygo-native-ui/ui/toc"
)

const guide = `# The Lantern guide

Lantern turns a folder of Markdown files into a documentation site: pages
with a sidebar, search, and a table of contents on every page. This guide
takes you from installing it to deploying your first site.

## Introduction

Lantern is a single program with no plugins to install and no runtime to
keep up to date. It reads your pages, renders them to HTML, and writes a
folder you can host anywhere.

### What Lantern does

- Renders CommonMark and GitHub's tables, task lists and footnotes.
- Builds the sidebar from your folders, in the order you choose.
- Indexes every page for search, offline, in the browser.
- Rebuilds only the pages that changed while you write.

### What it does not do

Lantern has no server side. Comments, analytics and sign-in belong to the
host you deploy to, or to a script you add to the theme.

## Installation

Lantern runs on macOS, Linux and Windows. Each release is a single binary
of about 12 MB.

### Installing with Homebrew

~~~sh
brew install lantern-docs/tap/lantern
lantern version
~~~

### Installing from source

You need Go 1.24 or later. The build takes about a minute on a laptop:

~~~sh
git clone https://example.com/lantern/lantern.git
cd lantern
go install ./cmd/lantern
~~~

### Checking the install

Run ` + "`lantern doctor`" + `. It checks that the binary is on your path,
that it can write to its cache, and that your terminal shows colors.

## Your first site

A site is a folder holding a configuration file and a folder of pages.

### Creating a project

~~~sh
lantern new handbook
cd handbook
~~~

This makes the configuration, a home page and a page to start from:

~~~text
handbook/
├── lantern.toml
└── pages/
    ├── index.md
    └── getting-started.md
~~~

### Writing pages

Each Markdown file is a page, and each folder a section of the sidebar.
The first heading of a page is its title, unless its front matter sets
one. Keep one topic to a page: readers land on pages from search, not
from the home page.

### Previewing changes as you write

~~~sh
lantern serve --open
~~~

The preview reloads the page in the browser when you save. Errors show
over the page rather than in the terminal, with the file and line.

## Configuration

Everything Lantern needs to know about a site lives in lantern.toml, at
the root of the project.

### The lantern.toml file

~~~toml
title = "Handbook"
base_url = "https://docs.example.com/"
language = "en"

[search]
enabled = true
min_word_length = 3
~~~

Settings left out take their defaults, which lantern config prints.

### Navigation and sidebars

The sidebar lists sections in the order of their folders' names. Number
the folders to choose another, as 01-basics and 02-guides: the numbers do
not show. A section's index.md is the page its title opens.

### Themes, fonts and colors that follow the reader's system setting

The default theme has a light and a dark look, and follows the reader's
system unless they pick one. To change the accent color and fonts:

~~~toml
[theme]
accent = "#d97706"
font = "Inter"
code_font = "JetBrains Mono"
~~~

## Writing content

### Front matter

Front matter is TOML between lines of three pluses, before the text:

~~~toml
+++
title = "Rotating keys"
weight = 20
draft = true
+++
~~~

Pages marked as drafts show in the preview and are left out of builds.

### Links between pages

Link to a page by its path from the pages folder, without the extension.
Lantern checks every link as it builds and stops on one that leads
nowhere, naming the page and the line.

### Code blocks and highlighting

Fenced code is highlighted in the language its fence names. More than
200 languages are known; the others show as plain text.

~~~go
func main() {
	site, err := lantern.Load("handbook")
	if err != nil {
		log.Fatal(err)
	}
	site.Build("public")
}
~~~

## Search

### Building the index

The index is built with the site, as a file of a few hundred kilobytes
for a thousand pages. It loads the first time a reader opens search.

### Tuning results

1. Titles weigh more than headings, and headings more than text.
2. Words shorter than min_word_length are not indexed.
3. Pages with search = false in their front matter are left out.

## Deploying

lantern build writes the site to the public folder. Upload that folder
as it is.

### Static hosts

Any host that serves files works. Set the build command to lantern build
and the folder to public. Builds on hosts cache their downloads in
.lantern-cache, which you can keep between builds to save a minute.

### Serving a site from a subdirectory behind a reverse proxy

Set base_url to the full address, path included, and have the proxy pass
the path on unchanged:

~~~nginx
location /docs/ {
    proxy_pass http://127.0.0.1:8080/docs/;
}
~~~

## Troubleshooting

### Pages missing from the build

A page missing from the site is most often a draft, a file whose name
starts with an underscore, or a file outside the pages folder. Run
lantern build --verbose to list what it skipped and why.

### Slow rebuilds on large sites

Rebuilds read only the pages that changed, but a change to the theme or
to lantern.toml builds everything again. On sites of more than 5,000
pages, turn off the search index while you write:

~~~sh
lantern serve --no-search
~~~

That is all. Questions and fixes to this guide are welcome on the
project's forum.
`

type app struct {
	md     markdown.Options
	scroll ui.ScrollState
	toc    toc.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Fill().Background(t.Background).AlignItems(ui.Stretch).Children(func() {
		ui.Scroll(c).TrackScroll(&a.scroll).Grow(1).Children(func() {
			ui.Column(c).MaxWidth(t.Space(190)).Padding(t.Space(10), t.Space(12), t.Space(24)).Children(func() {
				markdown.View(c, guide, &a.md)
			})
		})
		ui.Column(c).Width(t.Space(70)).Padding(t.Space(10), t.Space(6), t.Space(6), t.Space(4)).Children(func() {
			toc.View(c, &a.toc, &a.scroll).Shrink(1)
		})
	})
}

func main() {
	a := &app{toc: toc.State{Progress: true}}
	a.md.Block = a.toc.Block
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Table of contents",
			Width:   1180,
			Height:  820,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
