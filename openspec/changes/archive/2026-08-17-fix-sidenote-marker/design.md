## Context

See `proposal.md` — Why.

The relevant code is three call sites:

- `cmd/mdsidenote/main.go:205` prepends `\n` to the emitted markup unconditionally, with no check for where the reference sat.
- `cmd/mdfootnote/main.go:40-44` matches the markup with three literal `\n` in `sidenotePattern`, and `transform` returns the input unchanged when nothing matches — so a miss is silent.
- `cmd/mdwrap/main.go:151` builds its token list with `strings.Fields`, which splits on any whitespace including the spaces inside a tag's attribute list.

`AGENTS.md` states that fixtures define behavior, so the six fixtures encoding the three-line layout are the thing being changed, not merely a consequence.

## Goals / Non-Goals

**Goals:**

- The marker renders against its referent in any correct markdown renderer.
- The round-trip holds after any reflow this toolkit performs.
- Documents already written in the old layout keep working.

**Non-Goals:**

- Compensating for the layout in `mdhtml`. Fixing the source removes the need, and a renderer-side patch would mask a future regression here.
- Making `mdsplit`/`mdjoin`/`mdwrap` generally HTML-aware. Only the tag-splitting defect in `mdwrap` is in scope; block-level HTML handling in `internal/markdown/transform.go` is a separate concern.

## Decisions

### Fix the source, not the renderer

The space is in the markdown. Any correct renderer reproduces it, so stripping it in `mdhtml` would fix one consumer and leave the document wrong for every other — including danhorst.com.

The reasoning that initially argued against a source fix was that `mdjoin` and `mdwrap` would reintroduce the space. Measured, they do not: those tools insert spaces at line boundaries, and attaching the marker removes the boundary. `mdwrap` never breaks before the marker at any width from 20 to 70, because it breaks at spaces and there is none.

### Relax recovery rather than pin the layout

`mdfootnote` could have been left alone by keeping the leading newline and stripping it downstream. Relaxing its pattern to `\s*` is better on three counts: it makes the recovery independent of a formatting choice it should never have depended on; it is backward compatible, so existing documents still reverse; and it repairs the pre-existing `mdjoin` and `mdsplit` breakage as a side effect.

The leading `\s*` is consumed by the replacement, which is exactly what reattaches `[^N]` to the preceding text — so one pattern change covers the attached, line-separated, and space-joined layouts identically.

*Migration:* `mdfootnote | mdsidenote` converts an old document to the new layout, using the pair as intended rather than needing a one-off tool.

### Tag detection errs toward treating `<` as text

In `mdwrap`, `<` opens a tag only when followed by `/` or an ASCII letter, and the tag ends at the next `>`. With no `>` following, the `<` is ordinary text.

That keeps `a < b` and `5<10` from swallowing the rest of a paragraph. The failure mode of the strict reading is a wrapped line that is merely wider than requested; the failure mode of a greedy reading is a paragraph collapsed onto one line, which is much worse.

An over-long tag overflows its line rather than splitting, the standard treatment for any unbreakable token.

### Tests before the fix

Three of the four round-trip cases fail today. They are written and observed failing first, so they are known to be load-bearing rather than merely passing.

## Risks / Trade-offs

**`mdsidenote`'s output format changes** → Any document reprocessed produces different bytes than before. It is a defect correction rather than an interface change, and `mdfootnote` reads both layouts, so nothing that worked stops working. Called out in the changelog because the output is what users commit.

**Six fixtures change at once, across four tool directories** → They must move together, and `fixtures/mdfootnote/*.in.md` must stay byte-identical to `fixtures/mdsidenote/*.out.md`; if that invariant slips, the pair is being tested against markup neither tool produces. Regenerate rather than hand-edit, then diff to confirm only the newline moved.

**A relaxed pattern could over-match** → `\s*` between three long, fully-specified literal element strings is still tightly anchored; the elements themselves are unchanged. The pattern does not become more permissive about what counts as sidenote markup, only about how it is laid out.

**`mdwrap` output changes for any document containing inline HTML** → No existing fixture contains any, so nothing regenerates, but a user's wrapped document with inline HTML will re-wrap differently on next run. That is the point, and the new behavior is the correct one.
