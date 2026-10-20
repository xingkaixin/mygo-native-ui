# AGENTS.md

Native UI components for [mygo](https://github.com/egoist/mygo). One package per component, built only on mygo's public packages.

## Commands

```sh
mise exec -- go build ./...
mise exec -- go vet ./...
mise exec -- gofmt -l .
mise exec -- go run ./examples/<name>
```

There are no tests. Check UI changes by running the example.

## Layout

- `ui/<name>/`: one component. The package doc comment carries a usage snippet.
- `examples/<name>/main.go`: a runnable demo. Related components may share one (`menubar`, `agent`).
- A new component needs its package, an example, a row in the README table, and a `go run` line in README.

## API conventions

- Builders take `c *ui.Context` first, then state, then data, then build callbacks.
- Builders return `ui.Element`, or an `Events` value when the app must react in the same frame (`tabs.Bar`, `palette.Show`).
- Values the app owns are passed as pointers (`*bool`, `*int`, `*[]Pair`).
- What only the component needs lives in a `State` with unexported fields, and its zero value works. Export a field only when the app edits it (`Playhead`, `Recent`).
- Per-element state uses `ui.Local` with keys prefixed by the package name (`"tabs.state"`), since apps share that key space.
- A component imports another only when it is built from it (`plan` → `trace`, `contextmeter` → `ring`).

## Docs

Doc comments describe behavior from the user's side: what shows, what input does, what the app must do in the same frame. Match the existing wording, e.g. "State is what a view keeps … Keep it in the app's state, one for each view."

## Commits

`type(scope): Subject`, where scope is the package name. The component, its example (`docs(examples)`), and the README (`docs`) go in separate commits.
