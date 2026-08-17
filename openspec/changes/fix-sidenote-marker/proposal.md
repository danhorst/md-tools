## Why

`mdsidenote` emits its marker on a line of its own, so the markdown reads `…content.` newline `<label …>`.
A newline inside a paragraph is a space, so every renderer produces `content. ¹` — a space between the text and the marker that annotates it.

Fixing that exposes a larger defect behind it.
`mdsidenote` and `mdfootnote` are a documented inverse pair, but the pairing does not survive the pipeline it exists to be used in:

- `mdfootnote`'s recovery pattern hard-codes three literal newlines, so `mdsidenote | mdjoin` already produces markdown it cannot reverse — and it fails silently, returning the input unchanged when the pattern misses.
- Worse, its capture for a note's content matches any tag, including the note's own closing one, so it runs to the last `</span>` in the document. On any document with two or more sidenotes it swallows the body text and notes that follow. The shipped `complex` fixture does not round-trip: two paragraphs of prose are absorbed into the first definition and the second note is lost, and the golden commits that output as correct.
- `mdwrap` splits on whitespace, so it breaks *inside* tag attributes, defeating recovery even once the pattern tolerates whitespace.

The round-trip is the contract these two tools exist to provide. It should hold after any reflow the toolkit itself performs.

## What Changes

- `mdsidenote` attaches the marker to the text it annotates instead of prefixing a newline.
- `mdfootnote` tolerates arbitrary whitespace between the marker's three elements, so recovery works whether the source is attached, on its own line, or space-joined.
- `mdfootnote` bounds each note's content to that note, ending the match at the note's own terminator rather than the document's last closing tag. **This is a data-loss fix**: documents with two or more sidenotes currently lose text.
- `mdwrap` treats an HTML tag as a single unbreakable token instead of splitting it on the spaces between its attributes.
- New round-trip tests covering `mdfootnote` recovery after `mdjoin`, `mdsplit`, and `mdwrap`.

Not breaking. `mdfootnote` still recovers documents written in the old layout, so existing markdown keeps working; `mdfootnote | mdsidenote` converts it to the new form.

## Capabilities

### New Capabilities

- `sidenote-round-trip`: Converting markdown footnotes to Tufte sidenote markup and back, including what must survive the toolkit's own reflow tools.

### Modified Capabilities

None. No existing capability is specified — `openspec/specs/` holds only `html-rendering`, which this does not touch.

## Impact

**Modified**

- `cmd/mdsidenote/main.go` — drop the leading newline from the emitted markup
- `cmd/mdfootnote/main.go` — relax `sidenotePattern` whitespace
- `cmd/mdwrap/main.go` — tag-aware tokenizer in `wrapToWidth`

**Fixtures**

Six regenerate, and `fixtures/mdfootnote/*.in.md` must stay byte-identical to `fixtures/mdsidenote/*.out.md`:

- `fixtures/mdsidenote/{simple,complex}.out.md`
- `fixtures/mdfootnote/{simple,complex}.in.md`
- `fixtures/mdhtml/sidenote.{in.md,out.html}`
- `fixtures/mdfootnote/complex.out.md` — currently encodes the swallowed text; regenerated from corrected output

New `fixtures/mdwrap/sidenote.*`. No existing `mdwrap` golden moves — none contains inline HTML.

**Tests**

New `roundtrip_test.go`.

**Dependencies**

None added.

**Out of scope**

`mdhtml` gains no compensating whitespace normalization; fixing the source removes the need. Its `sidenote` golden regenerates as a consequence, nothing more.
