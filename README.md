# mygo-native-ui

Native UI components for [mygo](https://github.com/egoist/mygo), built only on its public packages.

```sh
go get github.com/xingkaixin/mygo-native-ui
```

Requires mygo v0.5.5 and Go 1.27.1 or later.

## Components

| Package | What it does | Example |
|---|---|---|
| [`ui/gif`](ui/gif) | Plays animated GIFs: frame offsets, transparency, disposal methods and loop counts. Shows the first frame when the desktop asks for less motion. | [`examples/gif`](examples/gif) |
| [`ui/tabs`](ui/tabs) | Document tab bar, as in editors: close buttons, middle-click to close, a modified dot, drag to reorder, and horizontal scrolling when the tabs overflow. | [`examples/tabs`](examples/tabs) |
| [`ui/chartx`](ui/chartx) | Extensions to mygo's `ui/chart`: a logarithmic scale, a legend that shows and hides series, zooming and panning along x, heatmaps with a sequential color scale, histogram bins, a GitHub-style contribution calendar, and sparklines. | [`examples/chartx`](examples/chartx), [`examples/dashboard`](examples/dashboard) |
| [`ui/shortcut`](ui/shortcut) | Keyboard shortcut recorder field, key caps, parsing to and from accelerator strings such as `CmdOrCtrl+Shift+K`, and menu text such as `⇧⌘K`. | [`examples/shortcut`](examples/shortcut) |
| [`ui/palette`](ui/palette) | Command palette: fuzzy matching with highlighted matches, sections, shortcuts by each command, and keyboard navigation. | [`examples/palette`](examples/palette) |
| [`ui/zoomview`](ui/zoomview) | Zoomable, pannable view of any content: fit, actual size, zooming around the pointer, dragging and keyboard shortcuts. | [`examples/zoomview`](examples/zoomview) |
| [`ui/logview`](ui/logview) | Streaming log view: ANSI colors and styles, carriage-return progress lines, a line limit, filtering, and following the end. | [`examples/logview`](examples/logview) |
| [`ui/hexview`](ui/hexview) | Hex dump with an ASCII column, linked selection by mouse and keyboard, and copying as hex or text. | [`examples/hexview`](examples/hexview) |
| [`ui/jsonview`](ui/jsonview) | JSON tree that keeps member order, colors values by type, shows JSONPath of the chosen row, and copies values and paths. | [`examples/jsonview`](examples/jsonview) |
| [`ui/columns`](ui/columns) | Finder's column view: a list for each level of a tree, keyboard navigation between columns, and a preview of the chosen leaf. | [`examples/columns`](examples/columns) |
| [`ui/inspector`](ui/inspector) | Inspector panel parts: collapsible sections, labeled rows, and number fields that scrub by dragging their label and step with the arrow keys. | [`examples/inspector`](examples/inspector) |
| [`ui/kvtable`](ui/kvtable) | Key-value table editor, as for query parameters and headers: rows that turn on and off, a draft row that adds rows as you type, and removing rows. | [`examples/kvtable`](examples/kvtable) |
| [`ui/kanban`](ui/kanban) | Kanban board: cards dragged within and between lanes, with a drop indicator, and chosen and moved with the keyboard. | [`examples/kanban`](examples/kanban) |
| [`ui/settings`](ui/settings) | System Settings layout: rows in rounded groups with labels on the left and controls on the right, switches, rows linking to other panes, and notes. | [`examples/settings`](examples/settings) |
| [`ui/waveform`](ui/waveform) | Audio waveform from precomputed peaks: playhead, selecting a stretch, zooming around the pointer, panning, and following the playhead. | [`examples/waveform`](examples/waveform) |
| [`ui/thumbgrid`](ui/thumbgrid) | Thumbnail grid that loads only what shows: pictures decoded and shrunk in the background, the latest asked for first, and kept in a cache. | [`examples/thumbgrid`](examples/thumbgrid) |
| [`ui/emoji`](ui/emoji) | Emoji picker: search by name, group tabs, frequently used emoji, and a preview of the emoji under the pointer. Embeds Unicode's emoji data ([license](https://www.unicode.org/license.txt)). | [`examples/emoji`](examples/emoji) |
| [`ui/weekview`](ui/weekview) | Calendar week view: overlapping events side by side, dragged to another time or day and stretched by their bottom edge in 15-minute steps, and a line at the time now. | [`examples/weekview`](examples/weekview) |
| [`ui/flamegraph`](ui/flamegraph) | Flame graph of a profile: each function a bar as wide as its share of the samples, a click zooming to it, and a search lighting the matches with their share. Reads folded stacks. | [`examples/flamegraph`](examples/flamegraph) |
| [`ui/treemap`](ui/treemap) | Squarified treemap of sizes, as of a folder: items with their names and sizes and the items in them inside, a click going into one, and a path back up. | [`examples/treemap`](examples/treemap) |
| [`ui/gitgraph`](ui/gitgraph) | Git history as a graph: a lane for each branch, curves where branches part and merge, branch and tag labels, and a commit chosen with a click or the keys. Parses `git log`. | [`examples/gitgraph`](examples/gitgraph) |
| [`ui/nodegraph`](ui/nodegraph) | Node editor, as of a shader: nodes with inputs, outputs and the app's controls, dragged around a canvas that pans and zooms, and links dragged from output to input, without cycles. | [`examples/nodegraph`](examples/nodegraph) |
| [`ui/waterfall`](ui/waterfall) | Timing waterfall, as of a network panel or a trace: nested rows with columns, bars colored by phase, a time scale dragged across to zoom, and a row chosen. Parses HAR files. | [`examples/waterfall`](examples/waterfall) |
| [`ui/regex`](ui/regex) | Regular expression tester: groups colored in the pattern, flag toggles, compile errors, matches highlighted in the sample as you type, and a list of matches and their groups. | [`examples/devtools`](examples/devtools) |
| [`ui/cron`](ui/cron) | Cron schedule field: its five fields with the wrong one marked, what it means in words, and the next times it runs. Parses schedules and macros such as `@daily`. | [`examples/devtools`](examples/devtools) |
| [`ui/heatmap`](ui/heatmap) | Activity calendar, as GitHub's contributions: a square a day over a year, deeper for more, months and weekdays, a legend, and a day chosen with a click. | [`examples/devtools`](examples/devtools) |
| [`ui/easing`](ui/easing) | Cubic Bézier easing editor: the curve with two handles dragged to shape it, overshoot above and below, presets, its `cubic-bezier()` to copy, and a dot playing it. Gives a `ui.Easing`. | [`examples/design`](examples/design) |
| [`ui/gradient`](ui/gradient) | Linear gradient editor: stops dragged along a strip, added with a click and dragged away to remove, the chosen stop's color and place, the angle, and mixing in Oklab. Edits a `ui.LinearGradient`. | [`examples/design`](examples/design) |
| [`ui/crop`](ui/crop) | Picture cropper: the part kept bright and the rest dimmed, corner and edge handles, dragging to move, thirds while it changes, aspect ratios, and arrow keys. | [`examples/crop`](examples/crop) |
| [`ui/outline`](ui/outline) | Outliner, as Workflowy: nested items with guides, bullets that fold, Enter to split, Backspace to join, Tab and Shift+Tab to nest, and Cmd+Up and Cmd+Down to move an item with its children. | [`examples/outline`](examples/outline) |
| [`ui/gantt`](ui/gantt) | Gantt chart: task bars dragged to move and by their ends to resize in whole days, progress, milestones, dependency arrows red when broken, weekends and a line at today. | [`examples/gantt`](examples/gantt) |
| [`ui/dag`](ui/dag) | Graph of jobs that need one another, as a CI pipeline: columns laid out so few links cross, status icons, links lit for the job hovered or chosen. | [`examples/dag`](examples/dag) |
| [`ui/otp`](ui/otp) | One-time code field: a box a digit, typing and pasting filling them, Done in the frame the code is complete. | [`examples/otp`](examples/otp) |
| [`ui/daterange`](ui/daterange) | Range of days on two months side by side, with presets such as Last 7 days, weeks and dates in the user's regional formats. | [`examples/daterange`](examples/daterange) |
| [`ui/schemaform`](ui/schemaform) | Form built from a JSON Schema in its property order: text, switches, drop-downs, lists and groups, checked as the user edits. | [`examples/schemaform`](examples/schemaform) |
| [`ui/uploads`](ui/uploads) | Upload queue: progress, speed and time left for each file and in all, cancel and retry. | [`examples/uploads`](examples/uploads) |
| [`ui/sankey`](ui/sankey) | Flows between stages in columns, bands as wide as their amounts, the pointer lighting a flow or a stage and naming it below. | [`examples/sankey`](examples/sankey) |
| [`ui/sparkline`](ui/sparkline) | Word-sized line and bar charts with the value under the pointer, and stat cards with their change. | [`examples/sparkline`](examples/sparkline) |
| [`ui/mindmap`](ui/mindmap) | Mind map of an outline: branches on both sides in their own colors, trees laid out away from the title, folding. | [`examples/mindmap`](examples/mindmap) |
| [`ui/annotate`](ui/annotate) | Picture markup: boxes, arrows, text and blur in toolbar colors, moving and deleting marks, undo and redo, export at full size. | [`examples/annotate`](examples/annotate) |
| [`ui/qrcode`](ui/qrcode) | QR encoder for versions 1 to 40 at levels L to H, and a view drawing a code dark on light in either theme. | [`examples/qrcode`](examples/qrcode) |
| [`ui/sequencer`](ui/sequencer) | Drum step sequencer: eight synthesized drums over sixteen steps, sample-accurate playback, tempo and a playhead. | [`examples/sequencer`](examples/sequencer) |
| [`ui/spreadsheet`](ui/spreadsheet) | Spreadsheet: formulas with references, ranges and functions, a formula bar, range selection with its sum, and tab-separated copy and paste. | [`examples/spreadsheet`](examples/spreadsheet) |
| [`ui/timeline`](ui/timeline) | Video editing timeline: clips on tracks, moved and trimmed by dragging, snapping to edges, split at the playhead, with zoom. | [`examples/timeline`](examples/timeline) |
| [`ui/seatmap`](ui/seatmap) | Seat map of a cinema or a plane: price zones, sold seats, a limit on seats chosen, a legend and the total. | [`examples/seatmap`](examples/seatmap) |
| [`ui/knob`](ui/knob) | Rotary knob: drag to turn, Shift for fine turns, arrow keys, double-click to reset, linear or log ranges, a value readout. | [`examples/knob`](examples/knob) |
| [`ui/signature`](ui/signature) | Signature pad: smoothed ink that thins with speed, undo and clear, export on transparency. | [`examples/signature`](examples/signature) |
| [`ui/pentool`](ui/pentool) | Bezier pen tool: corners and smooth points, handles that turn together or apart, closed paths, SVG path export. | [`examples/pentool`](examples/pentool) |
| [`ui/curves`](ui/curves) | Tone curves: RGB and channel curves over a histogram, monotone interpolation, applied to a picture. | [`examples/curves`](examples/curves) |
| [`ui/wordcloud`](ui/wordcloud) | Word cloud: sized by count, placed on a spiral with no overlaps, a hover count and a chosen word. | [`examples/wordcloud`](examples/wordcloud) |
| [`ui/spectrum`](ui/spectrum) | Spectrum analyzer: FFT levels on a log frequency axis, as bars with held peaks or as a curve. | [`examples/spectrum`](examples/spectrum) |
| [`ui/pianoroll`](ui/pianoroll) | Piano roll: notes drawn, moved and sized on a grid, quantize, and a looped synth with a playhead. | [`examples/pianoroll`](examples/pianoroll) |
| [`ui/railroad`](ui/railroad) | Railroad diagram of a regular expression: literals, classes and anchors in boxes, branches, bypasses and counted loops, and groups in dashed frames in regex's colors. | [`examples/railroad`](examples/railroad) |
| [`ui/sequence`](ui/sequence) | Sequence diagram from Mermaid-like text: actors, calls, replies, activations, notes, loop, alt and opt frames, and the lines that do not parse. | [`examples/sequence`](examples/sequence) |
| [`ui/erd`](ui/erd) | Entity relationship diagram from SQL: tables from CREATE TABLE, keys, crow's foot relations from row to row, a layered layout with lanes for long relations, and tables dragged by their header. | [`examples/erd`](examples/erd) |
| [`ui/eq`](ui/eq) | Parametric equalizer: cut, shelf and peak bands dragged on a log frequency graph, Q on the wheel, the total response, and a filter that follows without clicks. | [`examples/eq`](examples/eq) |
| [`ui/mixer`](ui/mixer) | Mixing console: channel strips with pan, mute and solo, faders on a console's dB scale, stereo meters with held peaks and a clip light, and a master strip. | [`examples/mixer`](examples/mixer) |
| [`ui/adsr`](ui/adsr) | ADSR envelope editor: attack, decay, sustain and release dragged on the curve, readouts in ms and dB, a dot for each note sounding, and the level a synth plays. | [`examples/adsr`](examples/adsr) |
| [`ui/monthview`](ui/monthview) | Calendar month view: bars for whole and several days across the weeks, timed events, "+3 more", dragging to another day, and the region's first weekday. | [`examples/monthview`](examples/monthview) |
| [`ui/worldclock`](ui/worldclock) | Time zone planner: a row a place, 24 hours aligned to one instant, tinted for night and working hours, a line at now, and a time chosen by click, drag or keys. | [`examples/worldclock`](examples/worldclock) |
| [`ui/keymap`](ui/keymap) | Keyboard map of shortcuts: a US keyboard, modifiers to turn on or hold, keys in their group's colors, conflicts in red, and all of a key's shortcuts on hover. | [`examples/keymap`](examples/keymap) |
| [`ui/bitfield`](ui/bitfield) | Register bit-field editor: a cell a bit in nibbles and bytes, fields with their values and names, click or drag to set bits, and the value as hex, decimal and binary. | [`examples/bitfield`](examples/bitfield) |

### Menu bar

| Package | What it does | Example |
|---|---|---|
| [`ui/trayicon`](ui/trayicon) | Renders native UI into tray icon PNGs, transparent and at 2×, such as a live bar chart. | [`examples/menubar`](examples/menubar) |
| [`ui/menupanel`](ui/menupanel) | Menus inside menu bar panels: checks, shortcuts, details, disabled items, submenus that open in place, and keyboard navigation. | [`examples/menubar`](examples/menubar) |
| [`ui/tiles`](ui/tiles) | Control Center tiles: toggles with an icon and status, and wide sliders. | [`examples/menubar`](examples/menubar) |
| [`ui/ring`](ui/ring) | Ring progress and a ring countdown timer. | [`examples/menubar`](examples/menubar) |

### AI

Parts of an AI chat or coding agent, after [AI Elements](https://ui.shadcn.com/docs/ai-elements).

| Package | What it does | Example |
|---|---|---|
| [`ui/trace`](ui/trace) | Steps of an agent's work: a status icon, title and duration, collapsible and nestable for subagents; tool calls with their input, output or error; and reasoning that streams open and folds to "Thought for 12s". | [`examples/agent`](examples/agent) |
| [`ui/confirm`](ui/confirm) | Approval card for a command or edit: Allow, Always allow and Deny, Enter to allow and Escape to deny, then the answer. | [`examples/agent`](examples/agent) |
| [`ui/promptinput`](ui/promptinput) | Chat input that grows as you type: Enter sends, Shift+Enter breaks the line, files attach by dropping or pasting, a slot for a model selector, and a send button that stops the reply. | [`examples/agent`](examples/agent) |
| [`ui/plan`](ui/plan) | An agent's plan: pending, running and done steps, "3 of 7" with a bar, and done steps folded away. | [`examples/agent`](examples/agent) |
| [`ui/contextmeter`](ui/contextmeter) | Context window usage as a ring, amber and red near the limit, with a hover card of input, output and cached tokens and the cost. | [`examples/agent`](examples/agent) |
| [`ui/changes`](ui/changes) | Tree of the files an agent changed: A, M or D and lines added and removed, the chosen file reported for a diff view. | [`examples/agent`](examples/agent) |
| [`ui/shimmer`](ui/shimmer) | Text with a band of light sweeping across, as for "Thinking…". | [`examples/agent`](examples/agent) |
| [`ui/message`](ui/message) | A chat's turns: the user's message in a bubble and the model's reply, with Copy, Edit and Retry, and arrows between the versions edits and retries make. | [`examples/chat`](examples/chat) |
| [`ui/sources`](ui/sources) | Where a reply comes from: "Used 5 sources", which opens to a numbered list, and citations by a paragraph, "go.dev +2", with a card of the sources on hover. | [`examples/chat`](examples/chat) |
| [`ui/testresults`](ui/testresults) | A run of tests: how many passed, failed and were skipped, with a bar of them, and suites that open to their tests, failures open to what they printed. | [`examples/agent`](examples/agent) |
| [`ui/stacktrace`](ui/stacktrace) | An error and its stack: the app's frames in full, the library's folded, and a frame chosen with a click. Parses Go panics, Node.js errors and Python tracebacks. | [`examples/agent`](examples/agent) |

Each package's doc comment shows how to use it.

## Examples

```sh
go run ./examples/gif [file.gif]
go run ./examples/tabs
go run ./examples/chartx
go run ./examples/dashboard
go run ./examples/shortcut
go run ./examples/palette
go run ./examples/zoomview [image]
go run ./examples/logview [command [args...]]
go run ./examples/hexview [file]
go run ./examples/jsonview [file.json]
go run ./examples/columns [folder]
go run ./examples/inspector
go run ./examples/kvtable
go run ./examples/kanban
go run ./examples/settings
go run ./examples/waveform
go run ./examples/thumbgrid [folder]
go run ./examples/emoji
go run ./examples/weekview
go run ./examples/flamegraph [file.folded]
go run ./examples/treemap [folder]
go run ./examples/gitgraph [repository]
go run ./examples/nodegraph
go run ./examples/waterfall [file.har]
go run ./examples/devtools [repository]
go run ./examples/design
go run ./examples/crop [picture]
go run ./examples/outline
go run ./examples/gantt
go run ./examples/dag
go run ./examples/otp
go run ./examples/daterange
go run ./examples/schemaform
go run ./examples/uploads
go run ./examples/sankey
go run ./examples/sparkline
go run ./examples/mindmap
go run ./examples/annotate
go run ./examples/qrcode
go run ./examples/sequencer
go run ./examples/spreadsheet
go run ./examples/timeline
go run ./examples/seatmap
go run ./examples/knob
go run ./examples/signature
go run ./examples/pentool
go run ./examples/curves
go run ./examples/wordcloud
go run ./examples/spectrum
go run ./examples/pianoroll
go run ./examples/railroad
go run ./examples/sequence
go run ./examples/erd
go run ./examples/eq
go run ./examples/mixer
go run ./examples/adsr
go run ./examples/monthview
go run ./examples/worldclock
go run ./examples/keymap
go run ./examples/bitfield
go run ./examples/menubar
go run ./examples/agent
go run ./examples/chat
```

## Development

The Go version is managed with [mise](https://mise.jdx.dev):

```sh
mise install
mise exec -- go build ./...
```

## License

[MIT](LICENSE)
