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
| [`ui/response`](ui/response) | HTTP response viewer, as API clients show one under the request: the status in its class's color, time and size, tabs for the body, cookies and headers, a JSON body pretty, raw or as a tree, other text highlighted, and the time's DNS, connecting, TLS, waiting and downloading phases on hover, measured with `httptrace`. | [`examples/response`](examples/response) |
| [`ui/matrix`](ui/matrix) | Confusion and correlation matrices: names down the side and slanted across the top, right predictions blue and wrong ones red, counts or shares by actual or predicted class, recall, precision and accuracy in the margins, coefficients from -1 to 1 in two hues, and a cell lit with its row and column, told of on hover and chosen with a click. | [`examples/matrix`](examples/matrix) |
| [`ui/brush`](ui/brush) | Counts over time as bars, as log explorers show events, stacked by series with a legend, and a window chosen by dragging across them, moved by dragging inside it and resized by its edges in whole bars, the rest dimmed, with how many events it holds and when. | [`examples/brush`](examples/brush) |
| [`ui/horizon`](ui/horizon) | Horizon chart of many series in little height, as Cubism draws one: a short row a series cut into bands that fold over one another, blue above zero and red below, on one scale or each row's own, names and values over the rows, and a line across every row reading them at the pointer. | [`examples/horizon`](examples/horizon) |
| [`ui/pagination`](ui/pagination) | Page numbers between Previous and Next, the first, last and nearby pages with ellipses in a constant number of buttons, and a table footer with rows per page and "21–30 of 412"; the arrow keys turn the page. | [`examples/pagination`](examples/pagination) |
| [`ui/storagebar`](ui/storagebar) | Storage usage bar, as macOS's Storage settings: how much of the capacity is used, the parts in their colors and the space left gray, a legend of sizes in powers of 1,000, and a part lit from the bar or the legend on hover. | [`examples/storagebar`](examples/storagebar) |
| [`ui/request`](ui/request) | HTTP request editor, as API clients have one: the method in its color, a URL whose query and the Params table follow each other, headers, and a body of none, JSON or text, JSON checked and formatted; Cmd-Enter sends, and the request becomes an `http.Request`. | [`examples/request`](examples/request) |
| [`ui/frames`](ui/frames) | WebSocket and SSE message log, as browsers' developer tools show one: sent and received frames with arrows, time, event and size, a search, All, Sent and Received, following the end, and the chosen frame's data below, JSON formatted and binary as a hex dump. | [`examples/frames`](examples/frames) |
| [`ui/disasm`](ui/disasm) | Disassembly view: addresses, bytes, mnemonics and operands in columns, labels, comments, the program counter, and arrows of the jumps in lanes beside them; a double click follows a jump, and Escape comes back. | [`examples/disasm`](examples/disasm) |
| [`ui/barcode`](ui/barcode) | Code 128 and EAN-13 barcodes, with the check digit worked out and the digits under the bars. | [`examples/barcode`](examples/barcode) |
| [`ui/yearview`](ui/yearview) | Calendar year view: twelve small months in as many columns as fit, today ringed, days with events dotted in their colors, a card of a day's events on hover, and a click choosing a day. | [`examples/yearview`](examples/yearview) |
| [`ui/agenda`](ui/agenda) | Agenda list of events by day: day headings that stay at the top, Today and Tomorrow, all-day and multi-day events ("Day 2 of 3"), the event going on marked Now, and past events dimmed. | [`examples/agenda`](examples/agenda) |
| [`ui/notifications`](ui/notifications) | Notification inbox: Today, Yesterday and Earlier, unread dots, actions on each, mark read and remove on hover, and Mark all read. | [`examples/notifications`](examples/notifications) |
| [`ui/bilingual`](ui/bilingual) | Translation editor, as CAT tools have one: source and target segment by segment, placeholders lit and those missing from the target marked, Cmd-Enter confirming a segment and moving to the next to do, notes, progress, and All or To do. | [`examples/bilingual`](examples/bilingual) |
| [`ui/subtitle`](ui/subtitle) | Subtitle editor on a waveform: cues as blocks dragged to move or by their edges to retime, `[` and `]` setting the start and end at the playhead, a list of cues to edit with characters per second, and SRT parsed and written. | [`examples/subtitle`](examples/subtitle) |
| [`ui/bullet`](ui/bullet) | Bullet charts: a measure as a bar over shaded ranges, a target tick, and lower-is-better measures. | [`examples/bullet`](examples/bullet) |
| [`ui/bump`](ui/bump) | Bump chart of ranks over time: lines between each period's ranks, names at both ends, a card on hover, and a series held lit by a click. | [`examples/bump`](examples/bump) |
| [`ui/waffle`](ui/waffle) | Waffle chart: a hundred squares shared among parts by the largest remainders, with a legend of percentages and a part lit on hover. | [`examples/waffle`](examples/waffle) |
| [`ui/beeswarm`](ui/beeswarm) | Beeswarm of samples along an axis, a row a group: dots pushed apart so none hide another, smaller where they crowd, and a dot's value and label on hover. | [`examples/beeswarm`](examples/beeswarm) |
| [`ui/density`](ui/density) | Density of many points: hexagonal bins in a sequential scale, or contours of a smoothed grid, with a legend and a hexagon's count on hover. | [`examples/density`](examples/density) |
| [`ui/boxmodel`](ui/boxmodel) | Box model editor, as browsers' developer tools: content, padding, border and margin nested in their colors, each side's size dragged sideways or typed in, Option changing the opposite side with it. | [`examples/boxmodel`](examples/boxmodel) |
| [`ui/rulers`](ui/rulers) | Rulers around a zoomview, as design tools have them: ticks and numbers following the zoom and pan, a mark at the pointer, and guides pulled out of the rulers, moved, and dragged back to remove. | [`examples/rulers`](examples/rulers) |
| [`ui/contrast`](ui/contrast) | WCAG 2 contrast checker: text and background wells, a preview of large and normal text and a control, the ratio, AA and AAA for each, and the nearest text color that passes. | [`examples/contrast`](examples/contrast) |
| [`ui/lightbox`](ui/lightbox) | Photo lightbox over the window: the photo fitted on a dark backdrop to zoom and pan, its caption and place in the set, previous and next by button or arrow key, and a strip of thumbnails. | [`examples/lightbox`](examples/lightbox) |
| [`ui/tuner`](ui/tuner) | Instrument tuner: the nearest note and octave, the frequency, and a needle from 50 cents flat to 50 sharp, green within 5; pitch detection with YIN. | [`examples/tuner`](examples/tuner) |
| [`ui/fontpicker`](ui/fontpicker) | Font family picker: each family's name in its own face, a search that narrows the list, monospaced families alone with a toggle, and the arrow keys and Enter to choose. | [`examples/fontpicker`](examples/fontpicker) |
| [`ui/skeleton`](ui/skeleton) | Loading placeholders: boxes, circles and lines of text in the shape of what is coming, with a shine sweeping across them in step, still for less motion. | [`examples/skeleton`](examples/skeleton) |
| [`ui/histslider`](ui/histslider) | Range slider over a histogram of the values, as booking sites filter prices: the bars in range lit, and the low and high values written below. | [`examples/histslider`](examples/histslider) |
| [`ui/keyframes`](ui/keyframes) | Keyframe editor of animated properties: a lane a track with its value at the playhead, keys shaped by their easing, dragged in whole frames, added by double click and eased or deleted from their menu, and a ruler to scrub. | [`examples/keyframes`](examples/keyframes) |
| [`ui/facepile`](ui/facepile) | Overlapping avatars of who is here, ringed in their cursor colors, those away faded, names on hover, a click to follow one, and "+3" opening a list of everyone. | [`examples/collab`](examples/collab) |
| [`ui/reactions`](ui/reactions) | Emoji reactions under a message: chips with counts, lit where the user reacted, who reacted on hover, a click adding or taking back, and a button opening the emoji picker. | [`examples/collab`](examples/collab) |
| [`ui/tzpicker`](ui/tzpicker) | Time zone picker by city: some 170 cities from the furthest behind UTC to the furthest ahead, with their day, time and offset now, and a search by city, country, zone or abbreviation such as `PST`. | [`examples/tzpicker`](examples/tzpicker) |
| [`ui/password`](ui/password) | New password field: an eye to show what is typed, a meter of four bars from Weak to Strong, and the app's rules checked as the password meets them. | [`examples/password`](examples/password) |
| [`ui/xypad`](ui/xypad) | XY pad setting two values at once, as a filter's cutoff and resonance: a puck pressed or dragged into place, Shift for fine moves, arrow keys, double click to reset, and linear or log axes. | [`examples/xypad`](examples/xypad) |
| [`ui/jwt`](ui/jwt) | JSON Web Token decoder: the token's three parts in their colors, whether it has expired or is not valid yet, the header's and payload's claims named, and times as dates and as how long from now. | [`examples/jwt`](examples/jwt) |
| [`ui/dirdiff`](ui/dirdiff) | Two folders compared as a tree: sizes and times on each side, files added, removed, changed or the same, folders that differ opened, counts of each, and a toggle to only the differences. | [`examples/dirdiff`](examples/dirdiff) |
| [`ui/scope`](ui/scope) | Oscilloscope: a screen of divisions with up to two channels, a trigger level dragged on the screen, time and volts a division in 1, 2, 5 steps, each channel's frequency and peak-to-peak volts, and XY mode for Lissajous figures. | [`examples/scope`](examples/scope) |
| [`ui/wheelpicker`](ui/wheelpicker) | Picker wheels, as iOS's: choices round a drum with the chosen one in a shared band, dragged and flicked to settle on a choice, a choice a scroll notch, clicks above and below the band, arrow keys, and wheels that go round for hours and minutes. | [`examples/wheelpicker`](examples/wheelpicker) |
| [`ui/radialmenu`](ui/radialmenu) | Pie menu, as Blender's and Maya's marking menus: items round the pointer where the right button presses, a stroke toward one choosing it on release, a click choosing once open, rings of items within items, and the arrow keys and Enter. | [`examples/radialmenu`](examples/radialmenu) |
| [`ui/switcher`](ui/switcher) | Switcher as ⌘Tab's and Ctrl+Tab's: Tab with a modifier held goes through the items, most recent first, letting go chooses, a quick press goes back to the one before, and the row of items shows only while the modifier stays down. | [`examples/switcher`](examples/switcher) |
| [`ui/gridpicker`](ui/gridpicker) | Table size picker, as Insert Table in Word and Notion: cells lit from the corner to the pointer, the grid growing past it, the size below, and the arrow keys and Enter. | [`examples/gridpicker`](examples/gridpicker) |
| [`ui/odometer`](ui/odometer) | A number whose digits roll on wheels as it changes, up as it grows and down as it shrinks, with separators and places coming and going at the left. | [`examples/odometer`](examples/odometer) |
| [`ui/packet`](ui/packet) | Packet details over its bytes, as Wireshark's panes: protocol fields in a tree whose choice selects their bytes in a hex dump, a click on a byte finding its field, and a dissector of Ethernet, ARP, IPv4, IPv6, TCP, UDP, ICMP and HTTP/1. | [`examples/packet`](examples/packet) |
| [`ui/colprofile`](ui/colprofile) | Table whose header sums up each column, as Data Wrangler: histograms of numbers, the most common values of text, the share of empty cells, and a click on a bar or a value keeping its rows, shown over all the rows in every histogram. | [`examples/colprofile`](examples/colprofile) |
| [`ui/fieldmap`](ui/fieldmap) | Columns of an imported file mapped to fields: lines between them, dragged from a column to a field or chosen from the field's menu, matches by name, samples of what each field takes, and required fields marked until filled. | [`examples/fieldmap`](examples/fieldmap) |
| [`ui/cidr`](ui/cidr) | Subnet planner, as the Visual Subnet Calculator: subnets divided in halves and joined back, a bar of the network's addresses, the bits of a subnet's address by network, subnet and hosts, and subnets overlapping ranges taken elsewhere marked. IPv4 and IPv6. | [`examples/cidr`](examples/cidr) |
| [`ui/procmon`](ui/procmon) | Process monitor, as Activity Monitor: CPU with a graph of its last minute, memory, threads, PID and user, sorted by column, a list or a tree of parents and children, a search, Stop after asking, and a graph of the CPU load. | [`examples/procmon`](examples/procmon) |
| [`ui/recurrence`](ui/recurrence) | Custom repeat rule editor, as Calendar's: frequency and interval, weekday toggles, a grid of month days or "the second Tuesday", months, an end, the rule in words, its next times, and RRULE text. | [`examples/recurrence`](examples/recurrence) |
| [`ui/chessboard`](ui/chessboard) | Chess board: pieces moved by dragging or two clicks, dots and rings where a piece may go, the last move and check lit, promotion choices, arrows and circles drawn with the right button, and the board turned for black. The app supplies the rules. | [`examples/chessboard`](examples/chessboard) |
| [`ui/goban`](ui/goban) | Go board of 9, 13 or 19 lines: shaded stones, star points, a ghost stone following the pointer, the last stone ringed, captures and suicide by the rules, and a reader for an SGF game's main line. | [`examples/goban`](examples/goban) |
| [`ui/xiangqi`](ui/xiangqi) | Chinese chess board: discs on the points with the river, palaces and start marks, files numbered for each side, pieces moved by dragging or two clicks, dots and rings where a piece may go, the last move and check lit. The app supplies the rules. | [`examples/xiangqi`](examples/xiangqi) |
| [`ui/panorama`](ui/panorama) | 360° photo viewer: an equirectangular picture drawn by a shader as a camera sees it, dragging with momentum, zooming between 30° and 100°, arrow keys, hotspots that report a click, and a compass. | [`examples/panorama`](examples/panorama) |
| [`ui/displays`](ui/displays) | Display arrangement, as the Displays settings: displays as large as their resolutions, dragged to snap edge to edge and line up, an outline where one lands, and the menu bar dragged to choose the main display. | [`examples/displays`](examples/displays) |
| [`ui/stringcatalog`](ui/stringcatalog) | String catalog editor, as Xcode's: languages with how much is translated, a row a string with key, comment, source and translation typed in place, a field a plural form by CLDR, New, Translated and Needs Review, search and filters. | [`examples/stringcatalog`](examples/stringcatalog) |
| [`ui/meshgradient`](ui/meshgradient) | Mesh gradient editor, as SwiftUI's MeshGradient: a grid of colored points blended by a shader, points dragged with edges sliding along the border, a color well for the chosen point, arrow-key nudges, and painting a mesh as a background. | [`examples/meshgradient`](examples/meshgradient) |
| [`ui/pagesetup`](ui/pagesetup) | Page setup: paper, orientation, margins and scale beside a preview of the app's page, margins dragged on dashed guides, Normal, Narrow and Wide, and inches or millimeters by region. | [`examples/pagesetup`](examples/pagesetup) |
| [`ui/meteogram`](ui/meteogram) | Hourly weather forecast as a meteogram: sky symbols, a temperature line red above freezing and blue below, precipitation bars, wind arrows and speeds, shaded nights, and an hour's numbers on hover. | [`examples/meteogram`](examples/meteogram) |
| [`ui/rename`](ui/rename) | Batch rename, as Finder's Rename Items: replace text with an optional regular expression, add text, or format with a counter, each name beside what it becomes with the change lit, and clashes and bad names held back. | [`examples/rename`](examples/rename) |
| [`ui/venn`](ui/venn) | Euler diagram of two or three sets: circles as large as the sets, overlapping as much as they share, counts in each part, and a part lit on hover and chosen with a click. | [`examples/venn`](examples/venn) |
| [`ui/periodic`](ui/periodic) | Periodic table of the 118 elements: cells colored by kind, the lanthanides and actinides below, the element under the pointer named at the top, a legend that lights a kind, a search, and an element chosen with a click. | [`examples/periodic`](examples/periodic) |
| [`ui/molecule`](ui/molecule) | Ball-and-stick molecule viewer: atoms in their CPK colors, single, double and triple bonds, turning by drag, zoom, depth fog, distances and angles measured with clicks, bonds found from covalent radii, and XYZ files. | [`examples/molecule`](examples/molecule) |
| [`ui/slides`](ui/slides) | Slide show, as Keynote's: a strip of thumbnails, a presenter view with the next slide, notes, a timer and the clock, the slide alone filling the screen, and the keys to move through it. | [`examples/slides`](examples/slides) |
| [`ui/phoneinput`](ui/phoneinput) | Phone number field: a flag and calling code opening a country list to search, digits grouped as the country writes them while typed, + or 00 picking the country, and E.164 output. | [`examples/phoneinput`](examples/phoneinput) |
| [`ui/callgrid`](ui/callgrid) | Video call layout: a grid of tiles as large as fit, a speaker view with a strip, a shared screen on the stage, initials while a camera is off, mute and hand badges, a speaking outline, and pinning by double-click. | [`examples/callgrid`](examples/callgrid) |
| [`ui/hold`](ui/hold) | Hold to confirm and slide to confirm controls for dangerous actions: a button that fills while pressed and drains when let go, and a knob dragged to the end of its track. | [`examples/hold`](examples/hold) |
| [`ui/stopwatch`](ui/stopwatch) | Stopwatch and timer, as the Clock app: laps with the fastest and slowest lit, and a countdown around a ring with pause, presets, and Finished in the frame it ends. | [`examples/stopwatch`](examples/stopwatch) |
| [`ui/splitflap`](ui/splitflap) | Split-flap display, as station boards and Vestaboard: a flap for each character whose halves fall over in 3D, one character after another around its drum, and a drum of digits for clocks. | [`examples/splitflap`](examples/splitflap) |
| [`ui/flashcards`](ui/flashcards) | Deck of flashcards, as Anki: a card turned over in 3D by a click or Space, then thrown right when known and left to see again, by drag, flick or arrow keys, tilting and marked as it goes, the next cards moving up. | [`examples/flashcards`](examples/flashcards) |
| [`ui/sheet`](ui/sheet) | Bottom sheet, as Maps on iPhone: detents of DIPs or fractions of the view, dragged and flicked between with rubber banding, a grabber that raises it, and the view under it dimmed near the top. | [`examples/sheet`](examples/sheet) |
| [`ui/hanzi`](ui/hanzi) | Stroke order of Chinese characters, as Hanzi Writer: strokes drawn in order along their medians on a 米字格 grid, and a quiz that judges each stroke traced, showing where it goes after three misses. | [`examples/hanzi`](examples/hanzi) |
| [`ui/hud`](ui/hud) | Volume and brightness HUD, as macOS: a square of Liquid Glass with an icon and sixteen notches that comes up as the level changes, fades out after, and sets the level by dragging. | [`examples/hud`](examples/hud) |
| [`ui/notebook`](ui/notebook) | Notebook, as Jupyter: code cells with execution counts, editors and outputs, Markdown cells rendered until double-clicked, Shift+Enter and Cmd+Enter to run, and cells added, moved and deleted. The app runs the code. | [`examples/notebook`](examples/notebook) |
| [`ui/ramp`](ui/ramp) | OKLCH color ramp editor, as Radix Colors are made: swatches with WCAG contrast, curves of lightness, chroma and hue whose points are dragged, the sRGB gamut shaded on the chroma curve, and steps fitted into it. | [`examples/ramp`](examples/ramp) |
| [`ui/crossword`](ui/crossword) | Crossword, as the NYT Mini: numbered squares, the word and square being filled lit, typing that moves on through words, arrows, Tab and Space, clue lists that go to a word, Check, and Solved in the frame the last letter is right. | [`examples/crossword`](examples/crossword) |
| [`ui/querybar`](ui/querybar) | Search query input, as GitHub's issue search: `key:value` terms colored as typed, a value it does not know in red, `-` to exclude, suggestions for keys and values, and the query parsed into terms and free text. | [`examples/querybar`](examples/querybar) |
| [`ui/explorer`](ui/explorer) | File explorer, as VS Code's: a tree of paths with single-folder chains compacted, git status colors and letters, renaming and new files in place, rows dragged into folders, a context menu, and revealing a path. | [`examples/explorer`](examples/explorer) |
| [`ui/findinfiles`](ui/findinfiles) | Find in files, as VS Code's search: match case, whole word and regular expression toggles, results grouped by file with folding headers, replace with `$1` groups previewed, and replacing one match, a file or all. | [`examples/findinfiles`](examples/findinfiles) |
| [`ui/debugger`](ui/debugger) | Debugger panels: a toolbar with F5 to F11 shortcuts, threads and their call stacks, variables in scopes that unfold and edit in place with changed values lit, and breakpoints turned on and off with conditions. | [`examples/debugger`](examples/debugger) |
| [`ui/presence`](ui/presence) | Multiplayer presence, as Figma: other people's cursors smoothed between updates with name tags that fade, their selections, faces that follow a person when clicked, and a frame in their color while following. | [`examples/presence`](examples/presence) |
| [`ui/bracket`](ui/bracket) | Tournament bracket: rounds of matches joined by lines, seeds and scores, winners bold, live matches and start times marked, a team's path lit under the pointer, and the champion at the end. | [`examples/bracket`](examples/bracket) |
| [`ui/sudoku`](ui/sudoku) | Sudoku: givens bold, entries in the accent color, notes, clashes in red, the row, column, box and digit of the square chosen tinted, keyboard entry, undo, and Solved in the frame it is. | [`examples/sudoku`](examples/sudoku) |
| [`ui/loupe`](ui/loupe) | Eyedropper over an image: a loupe of magnified pixels with a grid following the pointer, the hex of the pixel under it, a click or Enter to pick, arrows to move a pixel, and Cmd+C to copy. | [`examples/loupe`](examples/loupe) |
| [`ui/cohort`](ui/cohort) | Retention table: a row for each cohort with its size and the share still active in each period, shaded against the best after the first, a row for all users, and the row and column of a share lit. | [`examples/cohort`](examples/cohort) |
| [`ui/circlepack`](ui/circlepack) | Zoomable circle packing, as d3's: a tree as circles in circles colored by branch, names where they fit, a click to zoom in, Escape or a click outside to zoom out, and the path down shown in the corner. | [`examples/circlepack`](examples/circlepack) |
| [`ui/windrose`](ui/windrose) | Wind rose: sixteen directions stacked by speed band, rings of the share of time, the calm share in the middle, a tooltip per wedge, and a legend whose bands hide and show. | [`examples/windrose`](examples/windrose) |
| [`ui/elevation`](ui/elevation) | Elevation profile of a route, as cycling apps: the ground colored by grade, climbs marked with their length and grade, a readout under the pointer, and a stretch dragged to measure its gain, loss and grade. | [`examples/elevation`](examples/elevation) |
| [`ui/timecode`](ui/timecode) | Timecode field, as video editors: frames written as hours, minutes, seconds and frames at the footage's rate with drop-frame, scrubbing by dragging, Up and Down to step, and typing digits or `+`/`-` offsets from the right. | [`examples/timecode`](examples/timecode) |
| [`ui/currency`](ui/currency) | Money field: the currency's symbol, digits grouped as typed or pasted, its own number of decimals, a list to pick another currency, and amounts kept in minor units. | [`examples/currency`](examples/currency) |
| [`ui/duration`](ui/duration) | Duration field that reads time as people write it, `1h 30m`, `90m`, `1.5h`, `1:30`, `2d 4h`: what the text reads as shown while typing, text it cannot read in red, and Up and Down by 15 minutes. | [`examples/duration`](examples/duration) |
| [`ui/shogi`](ui/shogi) | Shogi board: pentagonal pieces facing the opponent, pieces in hand with counts, drops, a choice to promote where a piece may and promotion where it must, the last move and check lit, and the board turned for gote. The app supplies the rules. | [`examples/shogi`](examples/shogi) |
| [`ui/nonogram`](ui/nonogram) | Nonogram, as picross: clues for each row and column dimmed once met, clicks and drags along a line to fill or cross out, the line under the pointer tinted, keyboard play, and the picture shown when solved. | [`examples/nonogram`](examples/nonogram) |
| [`ui/problems`](ui/problems) | Problems panel, as VS Code's: diagnostics grouped by file with severity marks, sources and codes, line and column, a text filter, toggles with counts for each severity, folding files, and opening a problem. | [`examples/problems`](examples/problems) |
| [`ui/poll`](ui/poll) | Chat poll: one or several choices, a vote by click or Vote, results that fill with each share and check the user's choices, a retract button, and a closed poll with its leader bold. | [`examples/poll`](examples/poll) |
| [`ui/cloze`](ui/cloze) | Fill-in-the-blank exercise, as language courses set them: blanks as wide as their longest answer written `{colour|color}` in the text, Enter to the next blank, Check marking each right or wrong with the answer on hover, Reveal and Reset. | [`examples/cloze`](examples/cloze) |
| [`ui/typing`](ui/typing) | Typing test: the passage turning from muted to the text color as it is typed right and red where wrong, a gliding caret, words a minute and accuracy as they go, and a result with a line of the speed and a mark at each error. | [`examples/typing`](examples/typing) |
| [`ui/confetti`](ui/confetti) | Confetti thrown over the window: squares, strips and dots burst from a point or showered from the top, tumbling, drifting and fading as they fall, letting the pointer through, and none where the desktop asks for less motion. | [`examples/confetti`](examples/confetti) |
| [`ui/island`](ui/island) | Live activities in a black capsule at the top of the window, as the Dynamic Island: a glance at each end, a circle for a second activity, a click springing it open to the activity in full, and Escape or a click outside springing it back. | [`examples/island`](examples/island) |
| [`ui/thermostat`](ui/thermostat) | Thermostat dial: a ring of ticks around the target in large figures, the stretch to the room's temperature lit orange or blue, handles dragged round the ring, a heat-to and cool-to range in Auto, keys and scrolling, and the modes below. | [`examples/thermostat`](examples/thermostat) |
| [`ui/swatches`](ui/swatches) | Palette from a picture: its main colors by k-means, a ring where each is found, a bar of how much each takes, swatches with hex and share, the rest of the picture dimmed for the one pointed at, and a click to copy its hex. | [`examples/swatches`](examples/swatches) |
| [`ui/shadows`](ui/shadows) | Box shadow stack editor: a card casting the shadows on a stage dragged to move the light, layers to show, hide, add and remove, fields for the offsets, blur, spread, color and inset, and the stack as CSS with Copy. | [`examples/shadows`](examples/shadows) |
| [`ui/filters`](ui/filters) | Photo adjustments drawn by a shader: exposure, contrast, saturation, warmth, tint, vignette and blur, sliders that fill from zero and reset on a double click, and After, Split with a line dragged across, and Before. | [`examples/filters`](examples/filters) |
| [`ui/rebase`](ui/rebase) | Interactive rebase todo, as `git rebase -i`: a row a commit on a line with its action, pick, reword, edit, squash, fixup or drop, from a menu or its letter, squashes set in, drops struck through, rewording in place, drag or Alt+Up and Alt+Down to reorder, and the todo written out. | [`examples/rebase`](examples/rebase) |
| [`ui/flags`](ui/flags) | Feature flags across environments: a row a flag, a switch and a pie of its rollout in each environment, a slider in a popover to roll it out, a confirmation before a switch in production, and a search. | [`examples/flags`](examples/flags) |
| [`ui/daylight`](ui/daylight) | Daylight of a place on a day: sunrise, sunset, day length and its change since yesterday, the sun's path over the sky's light through the day with the time and height under the pointer, and the year's daylight to pick another day. | [`examples/daylight`](examples/daylight) |
| [`ui/metro`](ui/metro) | Ride along a metro line: the stops down the line in its color, the train between them, minutes to each stop ahead, transfer badges, stops passed and past the destination folded, and a click to get off elsewhere. | [`examples/metro`](examples/metro) |
| [`ui/plasmid`](ui/plasmid) | Circular plasmid map: base pairs counted around the ring, features as arcs pointing the way their strand reads, cut sites as ticks, labels pushed apart where they crowd, a feature's range on hover, and a selection by click or drag around the ring. | [`examples/plasmid`](examples/plasmid) |
| [`ui/cardinput`](ui/cardinput) | Payment card field: a card that shows what is typed in its brand's colors and turns over for the security code, digits grouped by brand, the Luhn check, the expiry checked against the month, and focus moving on as each field is whole. | [`examples/cardinput`](examples/cardinput) |
| [`ui/cards`](ui/cards) | Playing cards and a hand: corner indices, pips and framed court cards, a back that turns over, and a fanned hand where the card pointed at rises, a dragged card takes a new place, and a card dragged up or double-clicked is played. | [`examples/cards`](examples/cards) |
| [`ui/dice`](ui/dice) | Dice tray: six-sided dice with pips and the others as shapes with numbers, tumbling one after another on Roll or Space, a click to hold a die through a roll, and the total. | [`examples/dice`](examples/dice) |
| [`ui/metronome`](ui/metronome) | Metronome: a pendulum swinging to the beat with its weight set by the tempo, beats to accent, play or mute, a tempo from tapping or dragging the figure, and Space, ↑ and ↓. | [`examples/metronome`](examples/metronome) |
| [`ui/mahjong`](ui/mahjong) | Mahjong tiles and a hand: dots, bamboo, characters, winds and dragons drawn on their faces, melds laid down beside the hand, the tile chosen raised and discarded on a second click, and the tiles that would complete the hand. | [`examples/mahjong`](examples/mahjong) |
| [`ui/datadiff`](ui/datadiff) | Diff of two tables by key: rows added, removed and changed, the cells changed with their old values and a number's difference, unchanged rows folded, J and K between changes, and Show all rows. | [`examples/datadiff`](examples/datadiff) |
| [`ui/starmap`](ui/starmap) | Star map of the sky over a place at a time: the bright stars and the lines of the constellations, the horizon and the compass, a sky that darkens with the sun, a star's name on hover, and a drag or the arrows through time. | [`examples/starmap`](examples/starmap) |
| [`ui/fifths`](ui/fifths) | Circle of fifths: major keys, their relative minors and their signatures in three rings, a click to choose a key, the degrees of its chords marked around the circle, and its scale and chords beside it. | [`examples/fifths`](examples/fifths) |
| [`ui/tablature`](ui/tablature) | Guitar tablature: frets on the six strings, beats with their lengths as stems and beams, hammer-ons, slides, bends, vibrato and dead notes, bars wrapping to the width, a playhead, and editing by keyboard. | [`examples/tablature`](examples/tablature) |
| [`ui/levels`](ui/levels) | Levels adjustment, as photo editors: a histogram of the picture by channel, the input black, gray and white points and the output range dragged on sliders, Auto, Reset, and the picture adjusted. | [`examples/levels`](examples/levels) |
| [`ui/voicenote`](ui/voicenote) | Voice message in a chat: a play button, the note's loudness as bars that fill as it plays, a click or drag along them to seek, 1×, 1.5× and 2× speeds, a transcript, and a dot until it is heard. | [`examples/voicenote`](examples/voicenote) |
| [`ui/stories`](ui/stories) | Stories, as social apps: a tray of people ringed while they have unseen stories, and a viewer with a bar for each story filling as it plays, taps to step, holding to pause, and keys. | [`examples/stories`](examples/stories) |
| [`ui/reversi`](ui/reversi) | Reversi board: dots on the squares that can be played, the move under the pointer shown faintly with the discs it would turn, discs turning one after another, passing, and the count of each side. | [`examples/reversi`](examples/reversi) |
| [`ui/backgammon`](ui/backgammon) | Backgammon board for two players: the points, the bar and the tray for borne-off checkers, dice that fade as they are used, a click to light where a checker can go, hitting, entering, bearing off, and Undo. | [`examples/backgammon`](examples/backgammon) |
| [`ui/jigsaw`](ui/jigsaw) | Jigsaw puzzle of a picture: pieces cut with tabs and blanks scattered around the frame, dragging a piece above the others, a snap into place near its spot, a count of the pieces in, and the cuts fading when it is whole. | [`examples/jigsaw`](examples/jigsaw) |
| [`ui/spinwheel`](ui/spinwheel) | Wheel of chances: a slice for each choice with its label along it, a spin that slows to a stop with the pointer flicking at each slice, and the slice it lands on lit. | [`examples/spinwheel`](examples/spinwheel) |
| [`ui/recipe`](ui/recipe) | Recipe to cook from: servings that scale every amount, metric and US units with fractions and °F, ingredients ticked off, steps marked done, and timers counting down around a ring, side by side. | [`examples/recipe`](examples/recipe) |
| [`ui/pass`](ui/pass) | Wallet pass, a boarding pass or a ticket: the issuer and fields on its color, from and to with a plane between them, a tear line with notches over its barcode, and the details on its back, turned over in 3D. | [`examples/pass`](examples/pass) |
| [`ui/clockpicker`](ui/clockpicker) | Time picker on a clock face: the hour and minute large above it, AM and PM, a hand dragged to the hour and then the minute, an inner ring of hours on a 24-hour clock, and the arrow keys. | [`examples/clockpicker`](examples/clockpicker) |
| [`ui/patternlock`](ui/patternlock) | Pattern lock on a grid of nine dots: a line dragged through the dots and following the pointer, a dot passed over joining on the way, and a wrong pattern turning red and shaking before it clears. | [`examples/patternlock`](examples/patternlock) |
| [`ui/ticker`](ui/ticker) | Tape of quotes running across a strip: each symbol with its price and its change up in green or down in red, a seamless loop, a flash when a price moves, a stop on hover, and a click to choose a quote. | [`examples/ticker`](examples/ticker) |
| [`ui/calculator`](ui/calculator) | Calculator with a paper tape, as an adding machine: the keypad and its display, each number printed with its operation and each total under a rule, percent, a click on a line to enter it again, and the keyboard. | [`examples/calculator`](examples/calculator) |
| [`ui/iconpicker`](ui/iconpicker) | Icon picker: a search over the icons' names and keywords, a grid of the icons found with their names on hover, the arrow keys and Enter from the search, and the icon chosen large with its name. | [`examples/iconpicker`](examples/iconpicker) |
| [`ui/typescale`](ui/typescale) | Modular type scale: a base size and a ratio by its musical name, a row for each step with its name, pixels, rems and a line of sample text at its size, and the steps as CSS custom properties with Copy. | [`examples/typescale`](examples/typescale) |
| [`ui/msa`](ui/msa) | Multiple sequence alignment viewer: names down the left, residues in Clustal or base colors scrolling sideways under a ruler, a consensus with a bar of its share, identity shading, a residue's place on hover, and a column to choose. | [`examples/msa`](examples/msa) |
| [`ui/updater`](ui/updater) | Software update prompt, as Sparkle's: the new version and the installed one, the release notes of each version since in Markdown, Skip This Version, Remind Me Later and Install Update, a download bar with Cancel, Install and Relaunch, Try Again after a failure, and a check box to update without asking. | [`examples/updater`](examples/updater) |
| [`ui/licenses`](ui/licenses) | Open-source acknowledgements, as an About window's: a search, a chip for each license with its count that filters, packages with their versions and licenses tinted permissive, weak copyleft or strong copyleft, and the chosen package's license text with a link. | [`examples/licenses`](examples/licenses) |
| [`ui/permissions`](ui/permissions) | First-run permissions, as an app asks for the camera, the microphone or screen recording: a row each with its icon, a Required tag and why, Allow, Open System Settings after a no, a check once allowed, and Continue once the required ones are. | [`examples/permissions`](examples/permissions) |
| [`ui/spc`](ui/spc) | Control chart of a process, as Shewhart's individuals chart: a center line and limits from a baseline's moving range, one-, two- and three-sigma lines, points breaking Western Electric rules 1 to 4 in red with the rule's number, and a card of a point's value, sigmas and rules. | [`examples/spc`](examples/spc) |
| [`ui/mosaic`](ui/mosaic) | Marimekko chart: a column a category as wide as its share of the total, split into blocks a series as tall as their share of the column, names and shares where they fit, a legend, and a series lit with a card on hover. | [`examples/mosaic`](examples/mosaic) |
| [`ui/pyramid`](ui/pyramid) | Population pyramid: back-to-back bars a group, the youngest at the bottom, on one scale, counts or shares, an earlier count as a step outline over the bars, and a card of both sides' counts, shares and change on hover. | [`examples/pyramid`](examples/pyramid) |
| [`ui/ternary`](ui/ternary) | Ternary plot of mixtures of three parts, as the soil texture triangle: a grid every tenth with labels and arrows along the edges, named regions such as the twelve USDA soil classes, dots at their mixtures, and a card of a dot's shares and region on hover. | [`examples/ternary`](examples/ternary) |
| [`ui/floorplan`](ui/floorplan) | Floor plan to furnish: rooms with thick walls, their names and areas, doors with their swing, a grid and a scale bar, wall lengths on hover, and furniture to scale dragged on a 5 cm grid, turned with a double click or R, and outlined in red when it crosses a wall or another piece. | [`examples/floorplan`](examples/floorplan) |
| [`ui/patchbay`](ui/patchbay) | Patch cables between module jacks, as a modular synthesizer: panels of inputs and outputs, sagging colored cords, a cable pulled from a jack with the jacks it fits ringed, an input taking one cable, and a patched cable pulled out to move or drop. | [`examples/patchbay`](examples/patchbay) |
| [`ui/karnaugh`](ui/karnaugh) | Karnaugh map of two to four inputs beside its truth table: cells in Gray code that a click turns 0, 1 or don't care, the simplest sum of products with a loop in its color for each product, split where it wraps, and a cell and its row lit together. | [`examples/karnaugh`](examples/karnaugh) |
| [`ui/tactics`](ui/tactics) | Football tactics board: the pitch to scale with its lines, players as numbered discs in their side's color with their names, the ball, both dragged, and runs and passes drawn as arrows on the grass, a double click removing one. | [`examples/tactics`](examples/tactics) |
| [`ui/chmod`](ui/chmod) | Unix permission editor: read, write and execute tiles for owner, group and others with each row's octal digit, setuid, setgid and sticky, a typed octal number, the mode as `ls -l` writes it, the octal and symbolic `chmod` commands, and a warning when others can write. | [`examples/chmod`](examples/chmod) |
| [`ui/deptree`](ui/deptree) | Dependency tree, as `npm ls` and `cargo tree`: packages with their versions and sizes, repeats marked deduped, packages installed at more than one version flagged, a search that keeps the paths down to matches, and the chosen package's size with all it brings in, dependents and every path that installs it. | [`examples/deptree`](examples/deptree) |
| [`ui/smith`](ui/smith) | Smith chart of impedance sweeps: constant resistance circles and reactance arcs with labels, traces from their first to their last frequency, a shaded VSWR circle, and on hover the nearest point's frequency, impedance, Γ, VSWR, return loss and equivalent inductor or capacitor. | [`examples/smith`](examples/smith) |
| [`ui/abacus`](ui/abacus) | Soroban that reads its number: rods with a bead worth five above the beam and four worth one below, slid to the beam by a click, dots every third rod, each rod's digit and the number above, and the arrows and digit keys on a focused rod. | [`examples/abacus`](examples/abacus) |
| [`ui/meshview`](ui/meshview) | 3D mesh viewer, as a CAD viewport or Quick Look: lit faces, smooth across shallow angles and creased at sharp ones, turning and tilting by drag, zoom by wheel or pinch, and a view cube whose faces turn the model to that side. | [`examples/meshview`](examples/meshview) |
| [`ui/colorblind`](ui/colorblind) | Color vision deficiency simulator wrapping any part of an app, as Sim Daltonism: protan, deutan and tritan vision by Machado's model with a severity from 0 to 1, and achromatopsia, while what it wraps still takes input. | [`examples/colorblind`](examples/colorblind) |
| [`ui/totp`](ui/totp) | Authenticator list of one-time codes: issuer, account and the code in two groups, a ring counting down to the next code, red with the next code shown in the last five seconds, a click to copy, and a filter. | [`examples/totp`](examples/totp) |
| [`ui/marey`](ui/marey) | Train graph, as Marey drew the Paris–Lyon timetable: stations down the side by distance, hours across, each train a line from stop to stop in its service's color, its times on hover, scrolling through the day and zooming by pinch. | [`examples/marey`](examples/marey) |
| [`ui/pinout`](ui/pinout) | Pinout of a board or chip, as pinout.xyz: pins colored by what they carry with their functions as tags, a pin and its row lit on hover, and a tag or legend entry lighting every pin of its bus or kind. | [`examples/pinout`](examples/pinout) |
| [`ui/graphql`](ui/graphql) | GraphQL schema explorer, as GraphiQL's docs: root types and types by kind, fields with arguments and types, enum values, union members and implementations, type names as links with Back, deprecations struck through, and a search. | [`examples/graphql`](examples/graphql) |
| [`ui/rack`](ui/rack) | Rack elevation, as NetBox: devices in the units they fill, colored by role with a status light, units used and power drawn against the budget, a device's details on hover, and dragging a device to free units, the slot green or red. | [`examples/rack`](examples/rack) |
| [`ui/segments`](ui/segments) | Segment displays of clocks and meters: seven-segment digits, fourteen-segment letters or a 5×7 dot matrix, unlit segments faint, red, green or amber LED, cyan VFD or LCD looks, decimal points and colons, and long text scrolling. | [`examples/segments`](examples/segments) |
| [`ui/exif`](ui/exif) | Photo info panel, as Photos and Lightroom: camera and lens, aperture, shutter, ISO and focal length as tiles, date, size, exposure, location to copy, and every tag by directory, read from JPEG and TIFF files. | [`examples/exif`](examples/exif) |
| [`ui/identicon`](ui/identicon) | GitHub-style identicon from any string: a mirrored five-by-five grid in one color picked from a hash, so the same name always draws the same picture. | [`examples/identicon`](examples/identicon) |
| [`ui/randomart`](ui/randomart) | SSH key random art, as ssh-keygen prints it: the bishop's walk over a 17×9 field from a SHA256 fingerprint, the squares tinted by visits, and the fingerprint below. | [`examples/randomart`](examples/randomart) |
| [`ui/orrery`](ui/orrery) | Orrery of the eight planets on a date from JPL orbital elements: orbits on a square-root scale, scrolling through the days, zoom by pinch, and a planet's distance from the Sun and the Earth and its light time on hover. | [`examples/orrery`](examples/orrery) |
| [`ui/avionics`](ui/avionics) | Primary flight display of a glass cockpit: an attitude indicator with a pitch ladder and bank arc, an airspeed tape colored by the aircraft's V-speeds, altitude and vertical speed tapes, and a heading tape. | [`examples/avionics`](examples/avionics) |
| [`ui/vectorfield`](ui/vectorfield) | Vector field plot: arrows colored by strength, specks drifting with the flow and trailing behind, a streamline through a clicked point, and the field read out under the pointer. | [`examples/vectorfield`](examples/vectorfield) |
| [`ui/automaton`](ui/automaton) | Cellular automata of Life's kind by any B/S rule on a wrapping board: cells brightening when born and deepening with age, fading ghosts, patterns such as the Gosper gun, and dragging to paint cells alive or dead. | [`examples/automaton`](examples/automaton) |
| [`ui/sorting`](ui/sorting) | Sorting algorithms at work on bars: the compared pair and moved bars lit, comparison and move counts, a green sweep when done, and several views racing bubble, insertion, selection, shell, merge, quick and heap sort. | [`examples/sorting`](examples/sorting) |
| [`ui/fourier`](ui/fourier) | Epicycles drawing a closed path by its Fourier series: a chain of circles turning at each term's frequency, fewer terms rounding the corners off, and a path drawn with the pointer replacing it. | [`examples/fourier`](examples/fourier) |
| [`ui/arcdiagram`](ui/arcdiagram) | Arc diagram of a network: nodes in a row sized by links and colored by group, links as arcs as wide as their weights, and a node's arcs and neighbors lit on hover with its links in and out. | [`examples/arcdiagram`](examples/arcdiagram) |
| [`ui/hypnogram`](ui/hypnogram) | Night's sleep by stage, as sleep trackers: time asleep, a lane for awake, REM, core and deep with each spell joined to the next, the spell under the pointer, and each stage's time and share. | [`examples/hypnogram`](examples/hypnogram) |
| [`ui/splom`](ui/splom) | Scatterplot matrix: every pair of columns plotted with histograms on the diagonal, points colored by group, and a rectangle brushed in any plot lighting its rows in all of them. | [`examples/splom`](examples/splom) |
| [`ui/familytree`](ui/familytree) | Family tree of descendants: a card a person with their years, couples side by side, children bracketed below, and a click on a person reading out their life and lighting their line of descent. | [`examples/familytree`](examples/familytree) |
| [`ui/fishbone`](ui/fishbone) | Ishikawa cause-and-effect diagram: the effect at the head, bones of categories slanting off the spine above and below, their causes along them, and a click marking a likely root cause. | [`examples/fishbone`](examples/fishbone) |
| [`ui/screenplay`](ui/screenplay) | Screenplay from Fountain text, set as scripts are typed: scene headings numbered in both margins, character cues, dialogue and parentheticals at their columns, a title page, and scenes to jump to and characters by lines beside it. | [`examples/screenplay`](examples/screenplay) |
| [`ui/lyrics`](ui/lyrics) | Synced lyrics, as music players: the sung line large with its words lighting up from enhanced LRC times, the view gliding to keep it in place, breathing dots in breaks, and a click on a line to seek. | [`examples/lyrics`](examples/lyrics) |
| [`ui/loudness`](ui/loudness) | EBU R128 loudness meter: momentary and short-term bars against a target, integrated loudness, loudness range and sample peak, and the last minute as a graph, measured with the K-weighting and gating of BS.1770. | [`examples/loudness`](examples/loudness) |
| [`ui/timesheet`](ui/timesheet) | Weekly timesheet: a row for each project and task, today marked and the weekend shaded, row and day totals, the week against the hours due, and hours typed in place as 1:30, 1.5 or 90m. | [`examples/timesheet`](examples/timesheet) |
| [`ui/mailthread`](ui/mailthread) | Email thread, as mail apps: newest and unread messages open and the rest folded to a line, the middle of a long thread collapsed to a count, quoted text behind •••, recipients and attachments. | [`examples/mailthread`](examples/mailthread) |
| [`ui/storymap`](ui/storymap) | User story map: activities and their steps along the top, story cards under each step cut across by releases, and cards dragged between steps, releases and places with a dashed slot where they land. | [`examples/storymap`](examples/storymap) |
| [`ui/mancala`](ui/mancala) | Mancala by the rules of Kalah: pits that can be played lit under the pointer, seeds sown one at a time, extra turns and captures, and each store's count with the result. | [`examples/mancala`](examples/mancala) |
| [`ui/draughts`](ui/draughts) | English draughts: forced and multiple jumps, kings crowned on the far row, a lifted piece's landing squares marked, moves sliding with taken pieces fading, and each side's count with the result. | [`examples/draughts`](examples/draughts) |
| [`ui/tangram`](ui/tangram) | Tangram: the seven tans dragged onto a silhouette, snapping corner to corner, turned an eighth by a click and over by a right-click, and run together into one shape once they fill it. | [`examples/tangram`](examples/tangram) |
| [`ui/address`](ui/address) | Postal address form, as checkout forms have one: the country's fields in its order and with its labels, its states or provinces to choose from, the postal code checked against its pattern and tidied once left, and the address as an envelope carries it. | [`examples/address`](examples/address) |
| [`ui/clipboard`](ui/clipboard) | Clipboard history, as clipboard managers show it: a search, pinned clips first, colors as swatches and images as thumbnails, ⌘1 to ⌘9 by the first nine, the arrows and Enter, and the chosen clip in full with its counts, host or color values. | [`examples/clipboard`](examples/clipboard) |
| [`ui/vectorscope`](ui/vectorscope) | Scopes of color grading, as video editors show them: a vectorscope with 75% bar targets, the skin tone line and a 2× zoom, a luma waveform in IRE and an RGB parade, traces brighter where more pixels fall, and the pointer reading out the angle and saturation or the level. | [`examples/vectorscope`](examples/vectorscope) |
| [`ui/solitaire`](ui/solitaire) | Klondike solitaire built from `cards`: the stock dealing one or three, cards dragged between piles with those on them, a double click sending a card home, cards turning over and flying where they land, the game finishing itself once all are face up, and the moves, time and Undo. | [`examples/solitaire`](examples/solitaire) |
| [`ui/wifi`](ui/wifi) | Wi-Fi menu of a menu bar panel, as macOS shows it: a switch, the network joined in an accent circle, known networks and other networks under a disclosure with signal bars and locks, a password field opening in place, a spinner while joining, and the app's failure shown above the field. | [`examples/wifi`](examples/wifi) |
| [`ui/snaplayout`](ui/snaplayout) | Snap layouts, as a window's zoom button offers them: miniature screens split into halves, thirds and quarters, each zone lighting up under the pointer and chosen with a click or Space, and the window's frame for a zone with gaps between windows. | [`examples/snaplayout`](examples/snaplayout) |
| [`ui/sharesheet`](ui/sharesheet) | Share sheet, as iOS and macOS show one: the item with its picture and detail, a row of people with the app it goes through on their picture, a row of apps, and a list of actions with their symbols, the rows scrolling sideways. | [`examples/sharesheet`](examples/sharesheet) |
| [`ui/battery`](ui/battery) | Battery over the last day, as macOS's Battery settings show it: the level now with its glyph and when it last charged, the level in quarter-hour bars, red at 20% and below, with charging shaded behind, the screen time of each hour, and a time picked across both charts. | [`examples/battery`](examples/battery) |
| [`ui/quicklook`](ui/quicklook) | File previews over the window, as Quick Look shows them: a picture fitted with its size in pixels, text with numbered lines, other files as their kind's icon with name, size and date, the arrows and buttons moving between files, and Space, Escape or × closing it. | [`examples/quicklook`](examples/quicklook) |
| [`ui/envfile`](ui/envfile) | .env files side by side, a column for each environment: values edited in place, secrets as dots until their eye reveals them, missing values in the warning color with a click to add, keys set twice or not valid names flagged, and a variable added to every file. | [`examples/envfile`](examples/envfile) |
| [`ui/semver`](ui/semver) | Version range tester, as npm's semver calculator: a range in npm's syntax with carets, tildes, x-ranges, hyphen ranges and ||, the bounds it means or its error, and the versions by major with those matched lit, the highest ringed, and prereleases dimmed. | [`examples/semver`](examples/semver) |
| [`ui/containers`](ui/containers) | Container list with status, ports, CPU sparklines and memory, grouped by Compose project | [`examples/containers`](examples/containers) |
| [`ui/sprite`](ui/sprite) | Sprite sheet player with onion skin, mirror, ping-pong and frame ranges dragged on the sheet | [`examples/sprite`](examples/sprite) |
| [`ui/glyph`](ui/glyph) | Font glyph browser with metric lines, outline points and search by character, code point or name | [`examples/glyph`](examples/glyph) |
| [`ui/trimbar`](ui/trimbar) | Clip trim bar over a filmstrip with in and out handles, playhead, and I/O keys | [`examples/trimbar`](examples/trimbar) |
| [`ui/drumpad`](ui/drumpad) | MPC-style drum pads: velocity by where a pad is hit, keyboard layout from 1–4 to Z–V, Full Level, and glow that fades | [`examples/drumpad`](examples/drumpad) |
| [`ui/fifteen`](ui/fifteen) | Fifteen puzzle: row and column slides, arrow keys, tiles in place lit, moves and a timer, and solvable shuffles | [`examples/fifteen`](examples/fifteen) |
| [`ui/gomoku`](ui/gomoku) | Gomoku on a goban board: five in a row struck through, undo, and a suggested move for a computer side | [`examples/gomoku`](examples/gomoku) |
| [`ui/dominoes`](ui/dominoes) | Draw dominoes for two: a double-six line with doubles across, lifted playable tiles, end picking, drawing and passing, and scoring | [`examples/dominoes`](examples/dominoes) |
| [`ui/morris`](ui/morris) | Nine men's morris: placing, moving and flying, mills lit, pieces to take ringed, and a computer side | [`examples/morris`](examples/morris) |
| [`ui/kakuro`](ui/kakuro) | Kakuro: digits typed into runs, the sets of digits that make each run's sum, the digits a cell could take, and mistakes in red | [`examples/kakuro`](examples/kakuro) |
| [`ui/wordsearch`](ui/wordsearch) | Word search: words hidden in eight directions, a band dragged along the nearest one, found words kept in their own colors and struck from the list | [`examples/wordsearch`](examples/wordsearch) |
| [`ui/resistor`](ui/resistor) | Resistor color bands: four, five or six bands picked on a drawn resistor, value, tolerance range and tempco, and bands from a typed value | [`examples/resistor`](examples/resistor) |
| [`ui/punnett`](ui/punnett) | Punnett square for up to three genes: gametes crossed, children tinted by phenotype, phenotype and genotype ratios, and the pointed child marked | [`examples/punnett`](examples/punnett) |
| [`ui/morse`](ui/morse) | Morse code: letters drawn as dots and dashes, played as a tone at a set speed with each letter lit, and a key whose presses read back into letters | [`examples/morse`](examples/morse) |
| [`ui/braille`](ui/braille) | Braille, uncontracted: each character's six-dot cell over what it reads with capital and number signs, and a Perkins-style writer chorded on F D S J K L or clicked | [`examples/braille`](examples/braille) |
| [`ui/sliderule`](ui/sliderule) | Slide rule with K, A, B, CI, C, D and L scales: the slide and hairline cursor dragged, and each scale read under the hairline | [`examples/sliderule`](examples/sliderule) |
| [`ui/unitconv`](ui/unitconv) | Unit converter for ten quantities: a value typed in one unit read in every other, and a unit clicked to type in it | [`examples/unitconv`](examples/unitconv) |
| [`ui/slopegraph`](ui/slopegraph) | Slope graph and dumbbell chart of before and after values: lines colored by rise or fall with labels kept apart, or a row each on a shared axis, the item pointed at marked with its change | [`examples/slopegraph`](examples/slopegraph) |
| [`ui/hexbin`](ui/hexbin) | Hexagonal binning of many points: bins of a set radius shaded by count on a log scale, a key of the shades, and the bin pointed at outlined with its count | [`examples/hexbin`](examples/hexbin) |
| [`ui/voronoi`](ui/voronoi) | Voronoi diagram of sites dragged, added and removed, each cell shaded and the one pointed at lit, with the Delaunay triangulation drawn over it | [`examples/voronoi`](examples/voronoi) |

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
| [`ui/templatefield`](ui/templatefield) | Single-line field with `{{variables}}`, as in API clients' URL fields: defined and undefined variables in their own colors, a list of variables after `{{` that narrows as you type, and a card with each variable's value and source on hover. | [`examples/templatefield`](examples/templatefield) |
| [`ui/schemaeditor`](ui/schemaeditor) | JSON Schema editor for a tool's parameters or a structured output: a row a field with its name, type, whether it is required and its description, objects and arrays nested with guides, the values a string may take, taken names in red, focus on a field just added, and the schema written in field order and parsed back. | [`examples/schemaeditor`](examples/schemaeditor) |
| [`ui/spanlabel`](ui/spanlabel) | Span labeler for training data, as Prodigy and Label Studio: labels with number keys, spans tinted with their label's name after them, overlapping spans underlined on levels, drags snapped to whole words or to characters in Chinese and Japanese, and a span selected to relabel or delete. | [`examples/spanlabel`](examples/spanlabel) |
| [`ui/tokenizer`](ui/tokenizer) | A text as a model's tokenizer splits it, as tokenizer playgrounds show it: token and character counts, each token tinted by turns or the list of IDs, partial characters as hex bytes, whitespace shown as marks, and every token with the same ID lit on hover. | [`examples/tokenizer`](examples/tokenizer) |
| [`ui/question`](ui/question) | A question an agent asks the user: options with a line each, one or several chosen, an answer in the user's own words, previews beside the options, the arrows, number keys, Space and Enter, then the answer. | [`examples/question`](examples/question) |
| [`ui/arena`](ui/arena) | Two answers side by side to vote between, as model arenas: A, B, a tie or both bad by button or key, then the models named and the vote marked. | [`examples/arena`](examples/arena) |
| [`ui/bboxlabel`](ui/bboxlabel) | Bounding box labeler for object detection data: boxes dragged across a picture with guides, moved and resized by their corners, classes chosen with number keys, and a tag on each box. | [`examples/bboxlabel`](examples/bboxlabel) |
| [`ui/runs`](ui/runs) | Training curves of runs, as W&B and TensorBoard: a line a run smoothed by an exponential moving average over the raw values, a log scale, each run's value at the step under the pointer, and a legend that hides and shows runs. | [`examples/runs`](examples/runs) |
| [`ui/redline`](ui/redline) | Suggested edits to prose, as Track Changes: removals struck in red and insertions underlined in green, a pinned card with who suggested an edit and why, Accept and Reject, Accept all, Reject all and Undo, J and K, Y and N, and a word diff to make the edits from two texts. | [`examples/redline`](examples/redline) |
| [`ui/maskpaint`](ui/maskpaint) | Mask painter for inpainting: the masked part tinted over the picture, a brush and an eraser with soft edges, a ring cursor as wide as they paint, `[` and `]` to size them, Invert, Clear and Undo, and how much is masked. | [`examples/maskpaint`](examples/maskpaint) |
| [`ui/keypoints`](ui/keypoints) | Keypoint labeler for pose data, as COCO's: the joints of each person on a picture joined by their bones, dragging a joint or a whole skeleton, joints marked hidden or missing, a click to place a missing one, flipping, and a list of the joints. | [`examples/keypoints`](examples/keypoints) |
| [`ui/polygonlabel`](ui/polygonlabel) | Polygon labeler for segmentation data: clicks laying down a polygon's points closed on the first, polygons filled in their class's colors and tagged, points dragged, added at an edge's middle and removed, and classes by number keys. | [`examples/polygonlabel`](examples/polygonlabel) |
| [`ui/triage`](ui/triage) | Labeling queue, one item at a time: a strip of the items filled as they get labels, the item on a card, labels by click or number key going on to the next unlabeled item, ← and →, Skip, and the counts when all are done. | [`examples/triage`](examples/triage) |
| [`ui/xyplot`](ui/xyplot) | X/Y plot comparing pictures over two settings, as seeds by guidance: a grid headed by their values, spinners for pictures still coming, the row and column lit on hover, a cell opened large with the arrows through the grid, Swap axes, and a size slider. | [`examples/xyplot`](examples/xyplot) |
| [`ui/tooltoggles`](ui/tooltoggles) | Tool switches grouped by the server that gives them, as an agent's MCP servers: a switch for all of a server's tools and one for each, how many are on, a tag on tools that write, and a search. | [`examples/tooltoggles`](examples/tooltoggles) |
| [`ui/attention`](ui/attention) | Attention heads of a transformer, as BertViz: a layer, chips to show heads in their colors, lines from each token to the tokens it attends to as strong as the weight, a token's lines alone on hover, and the heads averaged as a matrix. | [`examples/attention`](examples/attention) |
| [`ui/embeddings`](ui/embeddings) | Embedding projector, as TensorFlow's: items as dots in their group's color, a search that rings matches, chips that hide groups, nearest neighbors by cosine similarity drawn on hover, and a chosen item's neighbors listed with their similarity. | [`examples/embeddings`](examples/embeddings) |
| [`ui/sampling`](ui/sampling) | What sampling settings do to the next token: temperature, top-k, top-p and min-p sliders over the candidates, each with its probability and its share once the others are cut, cut candidates tagged with what cut them, and how many are left with their entropy. | [`examples/sampling`](examples/sampling) |

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
go run ./examples/response
go run ./examples/matrix
go run ./examples/brush
go run ./examples/horizon
go run ./examples/pagination
go run ./examples/storagebar
go run ./examples/history
go run ./examples/ragchunks
go run ./examples/templatefield
go run ./examples/schemaeditor
go run ./examples/spanlabel
go run ./examples/tokenizer
go run ./examples/request
go run ./examples/frames
go run ./examples/disasm
go run ./examples/barcode
go run ./examples/yearview
go run ./examples/agenda
go run ./examples/notifications
go run ./examples/bilingual
go run ./examples/subtitle
go run ./examples/bullet
go run ./examples/bump
go run ./examples/waffle
go run ./examples/beeswarm
go run ./examples/density
go run ./examples/boxmodel
go run ./examples/rulers
go run ./examples/contrast
go run ./examples/lightbox
go run ./examples/tuner
go run ./examples/fontpicker
go run ./examples/skeleton
go run ./examples/histslider
go run ./examples/keyframes
go run ./examples/collab
go run ./examples/tzpicker
go run ./examples/password
go run ./examples/xypad
go run ./examples/jwt
go run ./examples/dirdiff
go run ./examples/scope
go run ./examples/question
go run ./examples/arena
go run ./examples/bboxlabel
go run ./examples/runs
go run ./examples/wheelpicker
go run ./examples/radialmenu
go run ./examples/switcher
go run ./examples/gridpicker
go run ./examples/odometer
go run ./examples/packet
go run ./examples/colprofile
go run ./examples/fieldmap
go run ./examples/cidr
go run ./examples/procmon
go run ./examples/recurrence
go run ./examples/chessboard
go run ./examples/goban
go run ./examples/xiangqi
go run ./examples/panorama
go run ./examples/displays
go run ./examples/stringcatalog
go run ./examples/meshgradient
go run ./examples/pagesetup
go run ./examples/meteogram
go run ./examples/rename
go run ./examples/venn
go run ./examples/periodic
go run ./examples/molecule
go run ./examples/slides
go run ./examples/phoneinput
go run ./examples/callgrid
go run ./examples/hold
go run ./examples/stopwatch
go run ./examples/splitflap
go run ./examples/flashcards
go run ./examples/sheet
go run ./examples/hanzi
go run ./examples/hud
go run ./examples/notebook
go run ./examples/ramp
go run ./examples/crossword
go run ./examples/querybar
go run ./examples/explorer
go run ./examples/findinfiles
go run ./examples/debugger
go run ./examples/presence
go run ./examples/bracket
go run ./examples/sudoku
go run ./examples/loupe
go run ./examples/cohort
go run ./examples/circlepack
go run ./examples/windrose
go run ./examples/elevation
go run ./examples/timecode
go run ./examples/currency
go run ./examples/duration
go run ./examples/shogi
go run ./examples/nonogram
go run ./examples/problems
go run ./examples/poll
go run ./examples/redline
go run ./examples/maskpaint
go run ./examples/cloze
go run ./examples/typing
go run ./examples/confetti
go run ./examples/island
go run ./examples/thermostat
go run ./examples/swatches
go run ./examples/shadows
go run ./examples/filters
go run ./examples/rebase
go run ./examples/flags
go run ./examples/daylight
go run ./examples/metro
go run ./examples/plasmid
go run ./examples/cardinput
go run ./examples/cards
go run ./examples/dice
go run ./examples/metronome
go run ./examples/mahjong
go run ./examples/keypoints
go run ./examples/datadiff
go run ./examples/starmap
go run ./examples/fifths
go run ./examples/tablature
go run ./examples/levels
go run ./examples/voicenote
go run ./examples/stories
go run ./examples/reversi
go run ./examples/backgammon
go run ./examples/jigsaw
go run ./examples/spinwheel
go run ./examples/polygonlabel
go run ./examples/triage
go run ./examples/xyplot
go run ./examples/tooltoggles
go run ./examples/recipe
go run ./examples/pass
go run ./examples/clockpicker
go run ./examples/patternlock
go run ./examples/ticker
go run ./examples/calculator
go run ./examples/iconpicker
go run ./examples/typescale
go run ./examples/msa
go run ./examples/updater
go run ./examples/licenses
go run ./examples/permissions
go run ./examples/attention
go run ./examples/embeddings
go run ./examples/sampling
go run ./examples/spc
go run ./examples/mosaic
go run ./examples/pyramid
go run ./examples/ternary
go run ./examples/floorplan
go run ./examples/patchbay
go run ./examples/karnaugh
go run ./examples/tactics
go run ./examples/chmod
go run ./examples/deptree
go run ./examples/smith
go run ./examples/abacus
go run ./examples/meshview
go run ./examples/colorblind
go run ./examples/totp
go run ./examples/marey
go run ./examples/pinout
go run ./examples/graphql
go run ./examples/rack
go run ./examples/segments
go run ./examples/exif
go run ./examples/identicon
go run ./examples/randomart
go run ./examples/orrery
go run ./examples/avionics
go run ./examples/vectorfield
go run ./examples/automaton
go run ./examples/sorting
go run ./examples/fourier
go run ./examples/arcdiagram
go run ./examples/hypnogram
go run ./examples/splom
go run ./examples/familytree
go run ./examples/fishbone
go run ./examples/screenplay
go run ./examples/lyrics
go run ./examples/loudness
go run ./examples/timesheet
go run ./examples/mailthread
go run ./examples/storymap
go run ./examples/mancala
go run ./examples/draughts
go run ./examples/tangram
go run ./examples/address
go run ./examples/clipboard
go run ./examples/vectorscope
go run ./examples/solitaire
go run ./examples/wifi
go run ./examples/snaplayout
go run ./examples/sharesheet
go run ./examples/battery
go run ./examples/quicklook
go run ./examples/envfile
go run ./examples/semver
go run ./examples/containers
go run ./examples/sprite
go run ./examples/glyph
go run ./examples/trimbar
go run ./examples/drumpad
go run ./examples/fifteen
go run ./examples/gomoku
go run ./examples/dominoes
go run ./examples/morris
go run ./examples/kakuro
go run ./examples/wordsearch
go run ./examples/resistor
go run ./examples/punnett
go run ./examples/morse
go run ./examples/braille
go run ./examples/sliderule
go run ./examples/unitconv
go run ./examples/slopegraph
go run ./examples/hexbin
go run ./examples/voronoi
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
