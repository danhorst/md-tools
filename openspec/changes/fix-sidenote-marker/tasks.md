## 1. Prove the tests bite

- [x] 1.1 Add `roundtrip_test.go` at the repo root, in `package fixtures_test`, covering `mdsidenote | mdfootnote` and the same with `mdjoin`, `mdsplit`, and `mdwrap` interposed.
- [x] 1.2 Run it against the current code and record which cases fail. The three reflow cases must fail now; if any passes, the test is not exercising what it claims.

## 2. mdsidenote

- [x] 2.1 Drop the leading `\n` from the emitted markup at `cmd/mdsidenote/main.go:205` so the marker attaches to the text the reference followed.
- [x] 2.2 Confirm a reference at the start of a line introduces no blank line, and a mid-sentence reference preserves the sentence tail.

## 3. mdfootnote

- [x] 3.1 Relax the three literal `\n` in `sidenotePattern` (`cmd/mdfootnote/main.go:40-44`) to `\s*`.
- [x] 3.2 Verify recovery for all three layouts — attached, line-separated, space-joined — and that the recovered `[^N]` attaches to the preceding text in each.

## 4. mdwrap

- [x] 4.1 Replace `strings.Fields` in `wrapToWidth` (`cmd/mdwrap/main.go:148-177`) with a tokenizer that keeps an HTML tag whole: `<` followed by `/` or an ASCII letter opens a tag, ending at the next `>`; with no `>`, the `<` is ordinary text.
- [x] 4.2 An over-long tag overflows onto its own line rather than splitting.
- [x] 4.3 Confirm `a < b` and `5<10` are unaffected and do not absorb the rest of the paragraph.

## 5. Fixtures

- [x] 5.1 Regenerate `fixtures/mdsidenote/{simple,complex}.out.md` and `fixtures/mdfootnote/{simple,complex}.in.md`, keeping the two pairs byte-identical to each other.
- [x] 5.2 Regenerate `fixtures/mdhtml/sidenote.{in.md,out.html}`.
- [x] 5.3 Diff all six and confirm only the marker's newline moved — nothing else in the markup or prose.
- [x] 5.4 Add `fixtures/mdwrap/sidenote.*` covering a wrapped paragraph carrying sidenote markup.

## 6. Verify

- [x] 6.1 Re-run `roundtrip_test.go`; all four cases now pass.
- [x] 6.2 `make fmt`, `make build`, `make test`, `go vet ./...` clean, with no fixture moving that this change did not intend.
- [x] 6.3 Render `mdsidenote` output through `mdhtml` and confirm no whitespace between the referent and the marker.
- [x] 6.4 Confirm the migration path: an old-layout document through `mdfootnote | mdsidenote` comes out in the new layout.
- [x] 6.5 Add `CHANGELOG.md` entries under `## [Unreleased]`, noting that `mdsidenote`'s output format changes.

## 8. Emphasis fidelity

Surfaced while writing the invertibility assertion: `mdfootnote` renormalized
`_emphasis_` to `*emphasis*`, so even a clean document did not round-trip.

- [x] 8.1 Build the html-to-markdown converter explicitly rather than using the package helper, which hardcodes its plugins, and pin the emphasis delimiter to `_`.
- [x] 8.2 Record the one bound that remains: a reference-style link inside a note is inlined by `mdsidenote`, which discards the label, so recovery cannot restore the reference form. Spec'd as a bound, not a defect.

## 7. Bounded recovery

Added after the round-trip tests exposed it: `mdfootnote`'s content capture matches
any tag, including its own closing one, so it runs to the document's last `</span>`.
On two or more sidenotes it absorbs the body text and notes that follow, and the
shipped `complex` golden commits that output as correct.

- [x] 7.1 Anchor the content capture on `mdsidenote`'s own terminator — content sits between `<span class="hidden">(</span>` and `<span class="hidden">)</span></span>` — so a non-greedy match ends at the right note while still permitting nested markup inside one.
- [x] 7.2 Regenerate `fixtures/mdfootnote/complex.out.md` from corrected output, and confirm the recovered document matches the original source rather than the previously committed mangling.
- [x] 7.3 Assert invertibility directly: `mdsidenote | mdfootnote` returns the original for both the `simple` and `complex` fixtures. A golden can encode a bug — that is how this one survived — an invertibility assertion cannot.
- [x] 7.4 Confirm a note whose content contains nested inline elements still recovers whole.
