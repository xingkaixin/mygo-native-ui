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
| [`ui/kvtable`](ui/kvtable) | Key-value table editor, as for query parameters and headers: rows that turn on and off, a draft row that adds rows as you type, and removing rows; optionally a kind for each row (text, file, secret, JSON), suggested keys and values, rows dragged into another order, and editing the pairs as text. | [`examples/kvtable`](examples/kvtable) |
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
| [`ui/candles`](ui/candles) | Candlestick chart: a volume pane, moving averages, a crosshair snapping to bars, date ticks that follow the zoom, zooming around the pointer and panning. | [`examples/candles`](examples/candles) |
| [`ui/worldmap`](ui/worldmap) | World choropleth in the Equal Earth projection: countries by value on a linear or log scale, a legend, the country under the pointer named, zoom and pan. Embeds Natural Earth's 1:110m countries ([public domain](https://www.naturalearthdata.com/about/terms-of-use/)). | [`examples/worldmap`](examples/worldmap) |
| [`ui/pixelart`](ui/pixelart) | Pixel art editor: pencil, eraser, fill, lines, rectangles and an eyedropper on two buttons, mirroring, a palette, undo by stroke, zoom, and export at whole multiples. | [`examples/pixelart`](examples/pixelart) |
| [`ui/compare`](ui/compare) | Before and after pictures: a slider, side by side, onion skin and changed pixels with their count, zoomed and panned together. | [`examples/compare`](examples/compare) |
| [`ui/forcegraph`](ui/forcegraph) | Force-directed graph, as a notes app's graph view: nodes sized by links and colored by group, neighbors lit on hover, dragging, zoom and pan, positions kept as data changes. | [`examples/forcegraph`](examples/forcegraph) |
| [`ui/terminal`](ui/terminal) | Terminal emulator, an xterm subset for shells, less, vim and top: 16, 256 and RGB colors, wide characters, line drawing, the alternate screen, scrollback, selection and copy, and bracketed paste. | [`examples/terminal`](examples/terminal) |
| [`ui/uptime`](ui/uptime) | Uptime bars, as on status pages: a banner of how the services are, a row a service with its status and uptime, a bar a day over 90, 60 or 30 days as the width allows, and a card of the day's uptime and incidents on hover. | [`examples/uptime`](examples/uptime) |
| [`ui/masonry`](ui/masonry) | Masonry board, as Pinterest's: as many columns as fit, each item below the shortest, only the items in view built, reflowing with the top item kept in view. | [`examples/masonry`](examples/masonry) |
| [`ui/tour`](ui/tour) | Onboarding tour over the app's own UI: the window dimmed around a cut-out that glides from step to step, a popover with the step, "2 of 5", Back, Next and Skip, arrow keys and Esc, targets scrolled into view. | [`examples/tour`](examples/tour) |
| [`ui/jsondiff`](ui/jsondiff) | Structural diff of two JSON documents in one tree: members added, removed and changed (old → new), arrays by index, unchanged parts folded, counts of each, next and previous change by keys, the JSONPath of the chosen row, and parse errors with their line and column. | [`examples/jsondiff`](examples/jsondiff) |
| [`ui/orderbook`](ui/orderbook) | Order book: asks above and bids below the mid price and spread, running totals with depth bars, prices grouped by a tick, sizes flashing as they change, a click reporting a level's price, and a depth chart with a crosshair. | [`examples/orderbook`](examples/orderbook) |
| [`ui/blame`](ui/blame) | Git blame view from `git blame --porcelain`: highlighted lines, each run from one commit with its summary, author and age, a bar deeper for newer commits, a card of the commit on hover, and its lines lit by a click. | [`examples/blame`](examples/blame) |
| [`ui/filterbuilder`](ui/filterbuilder) | Filter builder, as Notion's and Airtable's: conditions joined by And or Or, nested groups, operators and inputs by field type, and incomplete conditions marked and ignored by Match. | [`examples/filterbuilder`](examples/filterbuilder) |
| [`ui/certview`](ui/certview) | X.509 certificate chain viewer: whether it verifies against the system roots and why not, the chain from leaf to root, names, validity as a bar with the days left, alternative names, key, signature, usages, and fingerprints to copy. | [`examples/certview`](examples/certview) |
| [`ui/queryplan`](ui/queryplan) | PostgreSQL query plan from `EXPLAIN (FORMAT JSON)`: a foldable tree with estimated and actual rows, misestimates flagged by their factor, loops, self and total time, cost, each node's share with the slowest marked, and the chosen node's properties. | [`examples/queryplan`](examples/queryplan) |
| [`ui/minimap`](ui/minimap) | Code minimap beside mygo's editor, as Visual Studio Code's: lines in their tokens' colors, a box over the lines in view dragged to scroll, a click jumping there, and marks for matches, problems and changed lines. | [`examples/minimap`](examples/minimap) |
| [`ui/merge`](ui/merge) | Merge conflict resolver for git's markers: Current and Incoming side by side with the diff3 base between, the words that differ highlighted, Accept Current, Incoming or both in either order or edit by hand, "2 of 5 conflicts left" with next and previous, and unchanged lines folded. | [`examples/merge`](examples/merge) |
| [`ui/carousel`](ui/carousel) | Horizontally paged views: dragging or trackpad scrolling with spring snapping, flicks and rubber-banding, arrows on hover, page dots, and ← and →. | [`examples/carousel`](examples/carousel) |
| [`ui/pivot`](ui/pivot) | Pivot table over records: fields dragged or clicked into Rows, Columns and Values, sum, count, average, min and max, nested groups that fold with subtotals and totals, sorting by label or by a column, and frozen headers. | [`examples/pivot`](examples/pivot) |
| [`ui/dock`](ui/dock) | IDE-style docking layout: resizable splits down to tabbed panel groups, tabs dragged to join a group, split it at an edge or dock along a side, a translucent preview of where they land, and closing with × or the middle button. | [`examples/dock`](examples/dock) |
| [`ui/dashboard`](ui/dashboard) | Widget grid, as Grafana's dashboards: widgets dragged by their titles with a placeholder where they land, the others pushed down and the layout packed upward, resized from a corner by whole cells, and widgets moved aside sliding into place. | [`examples/dashboard`](examples/dashboard) |
| [`ui/sunburst`](ui/sunburst) | Sunburst of sizes, as of a folder: rings of arcs as wide as their share, colored by branch, a click zooming into an arc and the middle back out, and the path, size and share of the item under the pointer in the middle. | [`examples/sunburst`](examples/sunburst) |
| [`ui/globe`](ui/globe) | Globe in the orthographic projection: worldmap's countries, a graticule, labeled markers and great-circle arcs hidden beyond the horizon, dragging to turn it with inertia, the wheel to zoom, and the country under the pointer named. | [`examples/globe`](examples/globe) |
| [`ui/whiteboard`](ui/whiteboard) | Infinite whiteboard, as tldraw and FigJam: sticky notes, rectangles, ellipses, text, pen strokes and arrows that stay joined to shapes; zoom and pan over a dot grid, marquee selection, moves that snap with guides, resize handles, text edited in place, and undo. | [`examples/whiteboard`](examples/whiteboard) |
| [`ui/orgchart`](ui/orgchart) | Organization chart: cards with an avatar, name and title in their team's color, elbow lines from managers to reports, people with no reports stacked under their manager, folding with the count under each, zoom, pan and fit, and arrow keys between people. | [`examples/orgchart`](examples/orgchart) |
| [`ui/pianokeys`](ui/pianokeys) | Playable piano keyboard: the mouse gliding across keys with velocity by height, computer keys with octave shifts, held and app-lit notes, note on and off in the same frame, and the chord name. | [`examples/pianokeys`](examples/pianokeys) |
| [`ui/statechart`](ui/statechart) | State machine from Mermaid's stateDiagram-v2, run in place: states laid out left to right with few crossings, arrows named by their events, a composite state in a box, the current state lit, a button for each event it takes, and the last steps. | [`examples/statechart`](examples/statechart) |
| [`ui/colorwheels`](ui/colorwheels) | Color grading wheels: lift, gamma and gain discs, each with a puck and a luminance slider, and Apply to grade a picture. | [`examples/colorwheels`](examples/colorwheels) |
| [`ui/player`](ui/player) | Media transport controls: play and pause, skipping 10 seconds, a scrubber with buffered ranges, chapter gaps and a hover readout of time and chapter, elapsed and total or remaining time, a speed menu, a volume slider with mute, and keyboard shortcuts. | [`examples/player`](examples/player) |
| [`ui/tex`](ui/tex) | TeX math typesetting, as LaTeX's display math: fractions, roots, scripts, large operators with limits, delimiters that grow, matrices, cases and aligned equations, accents, braces and alphabets in the system's math font, mistakes in red where they are. Fills markdown's `Math` option. | [`examples/tex`](examples/tex) |
| [`ui/flowchart`](ui/flowchart) | Flowchart from Mermaid's flowchart text: every node shape, solid, dotted and thick links with text and heads, nested subgraphs, a layered layout with few crossings, the pointer lighting a node's links, a click choosing it, and the lines that do not parse. | [`examples/flowchart`](examples/flowchart) |
| [`ui/spectrogram`](ui/spectrogram) | Spectrogram: levels over time and frequency in magma's colors, the last ten seconds scrolling as a sound plays or a whole clip with a playhead a click moves, a log or linear frequency axis, a dB floor and ceiling with a legend, and the time, frequency, note and level under the pointer. | [`examples/spectrogram`](examples/spectrogram) |
| [`ui/layers`](ui/layers) | Layers panel of a design tool, as Figma's: frames and groups that fold, an icon by kind, lock and eye buttons on hover, hidden layers dimmed, Cmd- and Shift-click selection, renaming in place, and layers dragged to restack or nest, a line showing where and how deep. | [`examples/layers`](examples/layers) |
| [`ui/plotter`](ui/plotter) | Function grapher, as Desmos: y = f(x), x = g(y), polar, parametric and implicit curves, constants, breaks at poles and jumps, roots, extrema and intersections as gray dots, tracing, and a grid that follows panning and zooming around the pointer. | [`examples/plotter`](examples/plotter) |
| [`ui/parallel`](ui/parallel) | Parallel coordinates of many-dimensional rows: an axis per dimension with round ticks or labeled categories, brushes dragged along axes that filter together, moved and resized, names dragged to reorder axes and clicked to flip them, lines colored on a sequential scale, the nearest line lit with its values, and "128 of 1,000". | [`examples/parallel`](examples/parallel) |
| [`ui/logic`](ui/logic) | Logic analyzer and timing diagram: traces with x and z, bus lanes with hex values or labels, a ruler in ns, µs or ms that follows the zoom, cursors A and B snapping to edges with Δ and 1/Δ, and the value under the pointer. Parses WaveDrom's WaveJSON. | [`examples/logic`](examples/logic) |
| [`ui/scatter`](ui/scatter) | Scatter plot of 100,000 points: translucent dots that read denser where they crowd, colored by category with a legend that hides them, zooming around the pointer and panning, a box or, with Option, a lasso selecting, and the nearest point lit with a card. | [`examples/scatter`](examples/scatter) |
| [`ui/availability`](ui/availability) | Group availability grid, as When2meet: my slots painted by dragging a rectangle or with the arrows and Space, the group's shaded by how many are free with a legend, the best outlined, who is free on hover, and a person's slots lit from the list. | [`examples/availability`](examples/availability) |
| [`ui/ruby`](ui/ruby) | Ruby readings above text, as pinyin, zhuyin or furigana: centered at half size, reaching over neighbors or spreading the base when longer, kept on one line with their base, hidden with a tip on hover, and selection copying the bases alone or with their readings. Parses `{東京\|とうきょう}`, `{漢字\|かん\|じ}` and `漢(hàn)` markup. | [`examples/ruby`](examples/ruby) |
| [`ui/radar`](ui/radar) | Radar chart comparing a few items across measures: a range for each axis or one shared, axes where lower is better running inward, rings at round values, a legend that shows, hides and lights items, a card of every value on the axis under the pointer, and items growing out of the middle. | [`examples/radar`](examples/radar) |
| [`ui/boxplot`](ui/boxplot) | Distributions of groups side by side as box plots, violins or jittered points: quartiles, Tukey's whiskers and outliers, a log scale, either orientation, and a card of a group's summary or an outlier's value on hover. | [`examples/boxplot`](examples/boxplot) |
| [`ui/chord`](ui/chord) | Chord diagram of flows between groups: arcs as long as all a group sends and receives, with round ticks and names that keep apart, ribbons as wide as their flows in the sender's color with an arrowhead at the receiver, the pointer lighting a group's ribbons or both ways of a pair with a card, a click keeping a group lit and Esc letting go. | [`examples/chord`](examples/chord) |
| [`ui/upset`](ui/upset) | UpSet plot of set intersections, which reads where a Venn diagram of more than three sets does not: a row a set with its size, a column of joined dots a combination with a bar of the items in exactly those sets, sorted by size or degree, the pointer lighting a column with "In Chrome and Firefox, not Safari" or the columns of a set, and columns selected with a click and Shift-click. | [`examples/upset`](examples/upset) |
| [`ui/toc`](ui/toc) | Table of contents beside a document, as documentation sites show it: headings indented by level on a guide line, the section being read lit as the document scrolls with a bar gliding to it, a click scrolling to a heading, and how much is read. Finds the headings of a `markdown.View`. | [`examples/toc`](examples/toc) |
| [`ui/margincomments`](ui/margincomments) | Comments on ranges of a document in a margin beside it, as Google Docs shows them: tinted ranges, cards level with their first lines that push each other down, the active one level with its range as those above make room, replies, Resolve, and "Add comment" by a selection. The app keeps the threads. | [`examples/margincomments`](examples/margincomments) |
| [`ui/reader`](ui/reader) | Book reader, as Apple Books: text laid into pages that fit the view, justified or ragged, indented, each chapter starting a page, two side by side when wide, turned with the keys, a click or a swipe as they slide, and a scrubber naming the chapter under the pointer. | [`examples/reader`](examples/reader) |
| [`ui/lunar`](ui/lunar) | Month of the Chinese calendar, as Chinese calendar apps show it: the lunar day under each date and the month on 初一, solar terms and festivals in the accent, the year's 干支 and animal, the app's marks on their days, and the chosen day's 干支. Converts days and finds solar terms from 1900 to 2100. | [`examples/lunar`](examples/lunar) |
| [`ui/masked`](ui/masked) | Masked text field, as for phone numbers, dates and product keys: the slots still empty dimmed, the pattern's characters put in as the user types and stepped over, what does not fit a slot refused, and pastes filtered into the slots. | [`examples/masked`](examples/masked) |
| [`ui/classdiagram`](ui/classdiagram) | UML class diagram from Mermaid's classDiagram text: each class a box of name, fields and methods with annotations, static and abstract members, inheritance, realization, composition, aggregation, association and dependency ends, cardinalities and labels, a layered layout with parents above and few crossings, a click choosing a class and lighting its relations, and the lines that do not parse. | [`examples/classdiagram`](examples/classdiagram) |
| [`ui/dot`](ui/dot) | Graph from Graphviz's DOT language: ranks ordered so few edges cross, clusters in boxes, curved edges with their labels beside them, the shapes, styles, arrowheads and colors it names, a node chosen with a click, and the lines that do not parse. | [`examples/dot`](examples/dot) |
| [`ui/transcript`](ui/transcript) | Speech transcript that follows playback, as transcription apps and podcast players show it: each segment's speaker by a colored dot and its start, the word at the playhead lit, words spoken in full and those to come muted, a dotted underline under words heard with low confidence, a click on a word to seek, scrolling that follows the playhead until the user scrolls away, finding text with "2 of 9", and copying words with their speakers and times. | [`examples/transcript`](examples/transcript) |
| [`ui/charmap`](ui/charmap) | Unicode character viewer, as macOS's Character Viewer: groups of blocks in a sidebar, their characters in a grid built only as it shows, search by name, by code point or by the character pasted, the characters used lately, and the name, code point, UTF-8 and UTF-16 bytes, category, block and HTML entity of the one chosen. Names come from `golang.org/x/text/unicode/runenames`. | [`examples/charmap`](examples/charmap) |
| [`ui/staff`](ui/staff) | Music on a staff from ABC notation: clefs, keys, meters, heads, stems, flags, beams, rests, ties, triplets and repeats drawn as paths, systems that fill the width, a click to find a note in the source, wrong lines listed, and a tone per note played at the tempo with the sounding note lit. | [`examples/staff`](examples/staff) |
| [`ui/fretboard`](ui/fretboard) | Guitar fretboard and chord diagrams: strings of any tuning across frets spaced as on a real neck, with inlays, the notes of a scale or a chord as dots named or numbered from the root, a plucked note and a mark toggled by a click, and chord boxes from shapes such as `x32010` with fingers and barres that strum when clicked. | [`examples/fretboard`](examples/fretboard) |
| [`ui/gauge`](ui/gauge) | Dial gauge, as on a dashboard or a car's instruments: an arc of 180° to 270° in the app's colored zones, ticks at round values, a needle that springs to the value and pins past either end, the value and unit under the hub, and a held peak. | [`examples/gauge`](examples/gauge) |
| [`ui/wizard`](ui/wizard) | Setup assistant of steps, as installers walk through: the steps listed down the side or as numbered dots along the top, done ones checked, a check that holds a step with its message by Back and Continue, Enter to continue, and each step sliding in from the side the user goes to. | [`examples/wizard`](examples/wizard) |
| [`ui/openapi`](ui/openapi) | API reference from an OpenAPI 3.0 or 3.1 document in JSON: operations by tag with their methods in color and a search, parameters, request bodies and responses as trees of their schemas, and Try it, which hands the request to the app and shows the response. | [`examples/openapi`](examples/openapi) |
| [`ui/vertical`](ui/vertical) | Chinese and Japanese vertical text, as books set it: columns from right to left, punctuation in its vertical forms, Latin runs on their side, short numbers across one cell, line-start and line-end rules, and columns that scroll sideways when they overflow. | [`examples/vertical`](examples/vertical) |
| [`ui/tilemap`](ui/tilemap) | Slippy map of XYZ raster tiles, as OpenStreetMap's: dragging and flinging to pan, the wheel and buttons to zoom around the pointer, tiles loaded in the background and cached with their parents standing in, markers, flying to a place, a scale bar and the source's attribution. | [`examples/tilemap`](examples/tilemap) |
| [`ui/flamechart`](ui/flamechart) | Flame chart of a trace over time, as Chrome's Performance panel shows one: a lane for each thread, each call a bar from its start to its end stacked by depth, zoomed around the pointer down to microseconds, a ruler that follows the zoom, a call chosen, and a search dimming the rest. Parses the Trace Event Format. | [`examples/flamechart`](examples/flamechart) |
| [`ui/coverage`](ui/coverage) | Go test coverage from a `-coverprofile` file: files by package with their share and a bar, sorted by name or coverage, and the source tinted by block to the column, deeper by hit count in count mode, with N and Shift+N through what the tests did not run. | [`examples/coverage`](examples/coverage) |
| [`ui/cascader`](ui/cascader) | Field choosing a path down a hierarchy, as province, city and district: menus side by side, one a level, the pointer or the arrows opening an option's children in the next; typing finds full paths with the words lit, disabled options are skipped, and children can load lazily behind a spinner. | [`examples/cascader`](examples/cascader) |
| [`ui/transfer`](ui/transfer) | Transfer list for choosing and ordering items, as table columns or permissions: Available and Chosen side by side, each with a count, a search and a select-all check box; rows move with the buttons, a double click or by dragging, reorder within Chosen, and disabled rows stay put. | [`examples/transfer`](examples/transfer) |
| [`ui/swipeactions`](ui/swipeactions) | Swipe actions on list rows, as Mail's: a trackpad swipe or a mouse drag reveals colored buttons at either edge, springs open or closed, and runs the edge's first action when swiped all the way; one row open at a time, and the actions also in the context menu. | [`examples/swipeactions`](examples/swipeactions) |
| [`ui/quadrant`](ui/quadrant) | Quadrant chart, as a priority matrix or Mermaid's quadrantChart: four tinted quadrants with their titles, items as dots whose names keep clear of one another, dragged or moved with the arrow keys, and Mermaid text parsed and written back. | [`examples/quadrant`](examples/quadrant) |
| [`ui/funnel`](ui/funnel) | Conversion funnel: a centered bar a stage with bands narrowing to the next, each stage's count and share of the first, step conversion and drop-off between stages, down or across, segments compared as grouped bars, and a card on hover. | [`examples/funnel`](examples/funnel) |
| [`ui/streamgraph`](ui/streamgraph) | Streamgraph of many series over time: smooth layers around a wiggle, silhouette or zero baseline in inside-out or given order, names inside thick layers and the rest in a legend, a rule reading every layer's value on hover, and a click isolating a layer. | [`examples/streamgraph`](examples/streamgraph) |
| [`ui/ridgeline`](ui/ridgeline) | Distributions of groups as overlapping ridges, a joy plot: a kernel density estimate a group with an adjustable bandwidth, median or quantile marks, colors by group or along a gradient, sorting by median, and a card of a group's density and share below on hover. | [`examples/ridgeline`](examples/ridgeline) |
| [`ui/dendrogram`](ui/dendrogram) | Hierarchical clustering as a tree: single, complete, average or Ward linkage from vectors or distances, a cut dragged into colored clusters, a subtree lit on hover, left to right or top down, and an optional clustered heatmap. | [`examples/dendrogram`](examples/dendrogram) |

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
| [`ui/queue`](ui/queue) | Messages sent while an agent works, queued above the field: "3 queued", which folds, Edit in place, Send now and Remove, and reordering by dragging. | [`examples/agent`](examples/agent) |
| [`ui/modelpicker`](ui/modelpicker) | Model selector for a prompt input: a searchable list grouped by provider, with vision, tools and reasoning icons, the context window and the price in and out. | [`examples/agent`](examples/agent) |
| [`ui/checkpoint`](ui/checkpoint) | A checkpoint between turns: a rule with its time and what it covers, and Restore, which asks in place. | [`examples/chat`](examples/chat) |
| [`ui/promptvars`](ui/promptvars) | Prompt template editor: `{{variables}}` highlighted in the template, a field for each in order, those without a value marked, and a filled-in preview to copy. | [`examples/promptvars`](examples/promptvars) |
| [`ui/branches`](ui/branches) | A chat kept as a tree: edits and retries add versions beside the message, each keeping the conversation after it, and a map of the whole tree with the conversation showing lit, where a click shows a message. | [`examples/chat`](examples/chat) |
| [`ui/mention`](ui/mention) | Mentions in a prompt field, as Cursor's @: a trigger such as `@` or `#` opening a list by the caret that what follows narrows, the matching letters lit, and the item chosen a tinted token with its kind's icon, which the caret steps over and Backspace takes out whole. The app reads the text and the mentions' ranges. | [`examples/mention`](examples/mention) |
| [`ui/logprobs`](ui/logprobs) | A model's output with each token's probability, as logprob viewers show it: tokens tinted deeper the less likely they were, or by the entropy of their alternatives, those below a threshold underlined, a card of the token's probability and top alternatives on hover, pinned by a click and moved with the arrow keys, and the perplexity and least likely token above. The app reads the token pinned. | [`examples/logprobs`](examples/logprobs) |
| [`ui/replay`](ui/replay) | Replay of a computer-use agent's session: each step's screenshot with what it did over it, rings at clicks, a path for drags, a chip of what it typed or pressed, an arrow for scrolls, play at the session's pace, a scrubber, a filmstrip, and the action in words with its reasoning. | [`examples/replay`](examples/replay) |
| [`ui/voiceorb`](ui/voiceorb) | Orb of a voice assistant's conversation mode: a glowing sphere whose surface flows in the accent's colors, breathing at rest, swelling and rippling with the voice it hears, swirling as it thinks and pulsing outward as it speaks, along springs, and a still gradient sphere for less motion. | [`examples/voiceorb`](examples/voiceorb) |
| [`ui/hunks`](ui/hunks) | Hunk-by-hunk review of one file's diff, as agents ask about each edit and git clients stage: old and new line numbers, removed and added lines tinted, Accept and Reject or Stage and Discard, lines chosen by click, Shift-click or drag, j and k to move, y, n and u to decide and undo, "3 of 7 hunks left", and the patch of what was accepted for `git apply --cached`. | [`examples/hunks`](examples/hunks) |
| [`ui/evals`](ui/evals) | A run of LLM evals, as eval tools show one: a row a case and a column a model, each cell a check or cross, its score, latency and the start of the output, headers of pass rate, mean score and median latency that stay as it scrolls both ways, All, Failing and Disagree with their counts, and a pane of the prompt, what is expected and the output, diffed word by word against another model's and rated with a thumb. | [`examples/evals`](examples/evals) |
| [`ui/history`](ui/history) | Chat history sidebar, as AI chat apps list past conversations: New chat and a search, conversations under Pinned, Today, Yesterday, Previous 7 days, Previous 30 days and then by month, the open one lit, a spinner while a reply streams, and a "…" menu to rename in place, pin, or delete after asking in the row; only the rows in view are built. | [`examples/history`](examples/history) |
| [`ui/ragchunks`](ui/ragchunks) | Chunk inspector for retrieval: the document with its chunks tinted by turns, overlaps hatched and numbers at their starts, beside a list of chunks with their size in characters and tokens; a query's hits rank by score with bars, outlined in the text with the rest dimmed, and a splitter by characters or by separators in turn. | [`examples/ragchunks`](examples/ragchunks) |
| [`ui/templatefield`](ui/templatefield) | Single-line field with `{{variables}}`, as in API clients' URL fields: defined and undefined variables in their own colors, a list of variables after `{{` that narrows as you type, and a card with each variable's value and source on hover. |

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
go run ./examples/candles
go run ./examples/worldmap
go run ./examples/pixelart
go run ./examples/compare [before.png after.png]
go run ./examples/forcegraph
go run ./examples/terminal
go run ./examples/uptime
go run ./examples/masonry
go run ./examples/tour
go run ./examples/jsondiff [a.json b.json]
go run ./examples/orderbook
go run ./examples/blame [file]
go run ./examples/filterbuilder
go run ./examples/certview [host[:port] | file.pem]
go run ./examples/queryplan [plan.json]
go run ./examples/minimap [file.go]
go run ./examples/merge [file]
go run ./examples/carousel
go run ./examples/pivot
go run ./examples/dock
go run ./examples/sunburst [folder]
go run ./examples/globe
go run ./examples/whiteboard [board.json]
go run ./examples/promptvars
go run ./examples/orgchart
go run ./examples/pianokeys
go run ./examples/statechart
go run ./examples/colorwheels [picture]
go run ./examples/player [file.wav]
go run ./examples/tex
go run ./examples/flowchart
go run ./examples/spectrogram [file.wav]
go run ./examples/layers
go run ./examples/plotter
go run ./examples/parallel
go run ./examples/mention
go run ./examples/logic
go run ./examples/scatter
go run ./examples/availability
go run ./examples/ruby
go run ./examples/logprobs
go run ./examples/radar
go run ./examples/boxplot
go run ./examples/chord
go run ./examples/upset
go run ./examples/replay
go run ./examples/voiceorb
go run ./examples/toc
go run ./examples/margincomments
go run ./examples/reader
go run ./examples/hunks
go run ./examples/lunar
go run ./examples/masked
go run ./examples/classdiagram
go run ./examples/dot
go run ./examples/transcript
go run ./examples/evals
go run ./examples/charmap
go run ./examples/staff
go run ./examples/fretboard
go run ./examples/gauge
go run ./examples/wizard
go run ./examples/openapi
go run ./examples/vertical
go run ./examples/tilemap
go run ./examples/flamechart [trace.json]
go run ./examples/coverage
go run ./examples/cascader
go run ./examples/transfer
go run ./examples/swipeactions
go run ./examples/quadrant
go run ./examples/funnel
go run ./examples/streamgraph
go run ./examples/ridgeline
go run ./examples/dendrogram
go run ./examples/history
go run ./examples/ragchunks
go run ./examples/templatefield
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
