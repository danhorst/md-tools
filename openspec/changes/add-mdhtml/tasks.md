## 1. Groundwork

- [x] 1.1 Add the "Output Formats" section to `AGENTS.md`: permit markdown→other-text-format tools, exempt them from idempotency and the `.out.md` fixture convention, keep determinism / STDIN-STDOUT / exit codes / one-transformation-per-binary applying unchanged, and state that tools requiring an external runtime do not belong in this repository.
- [x] 1.2 Run `make test` and record that it passes — this is the baseline for the next task.
- [x] 1.3 Extract frontmatter detection from `internal/markdown/transform.go:30-60` into `SplitFrontmatter(content) (frontmatter, body string)`, handling both the `---`/`---` and bare-property-line formats, and make `Transform` call it.
- [x] 1.4 Run `make test` again and confirm every existing fixture still passes byte-identically. Any fixture movement means the extraction changed behavior and must be fixed before continuing.
- [x] 1.5 Generalize `fixtures_test.go`: discover the fixture sibling as `.out.*` instead of hardcoding `.out.md`, and run the idempotency subtest only when the resolved output extension is `.md`.
- [x] 1.6 Run `make test` and confirm the ten existing tools still get both their correctness and idempotency subtests.

## 2. Rendering package

- [x] 2.1 Create `internal/render/` with a goldmark constructor matching `cmd/mdsidenote/main.go:68` — GFM and Footnote extensions, `html.WithUnsafe()`.
- [x] 2.2 Write `internal/render/default.css`, Tufte-derived to match `mdsidenote`'s sidenote markup, including `@media print` rules for page setup and margin sidenotes.
- [x] 2.3 Embed the stylesheet with `go:embed` and expose it from the package.
- [x] 2.4 Add the standalone-document wrapper: doctype, `html`, `head` with charset and title, `body`. Fragment mode emits body content only.

## 3. mdhtml core

- [x] 3.1 Create `cmd/mdhtml/main.go` following the established shape — package doc comment with a `Usage:` block, `var flags = cli.RegisterFlags()`, `main()` delegating to `cli.Run`.
- [x] 3.2 Strip frontmatter via `markdown.SplitFrontmatter` before rendering.
- [x] 3.3 Register `--fragment`, `--title`, `--css`, and `--no-css`. Missing `--css` file is a fatal error naming the file.
- [x] 3.4 Default the title to the input file's base name without extension; empty when reading STDIN.
- [x] 3.5 Reject `-w` with an error explaining it would overwrite the markdown source, suggesting `-i FILE` or shell redirection. Confirm `-i` still writes the rendered HTML to the named file.
- [x] 3.6 Verify determinism: rendering the same input twice with the same flags produces byte-identical output.

## 4. SVG inlining

- [x] 4.1 Register `--inline-svg` and `--base DIR`. Require `--base` when `--inline-svg` is used with STDIN input; error clearly otherwise.
- [x] 4.2 Replace image references ending in `.svg` with the file's contents as an inline `svg` element, resolving relative paths against the input file's directory or `--base`. Leave non-SVG image references as `img` elements.
- [x] 4.3 Namespace identifiers per inlined SVG: collect every declared id, then rewrite `id="X"` to `id="pN-X"` and every `#X` occurrence to `#pN-X`, where `N` is the diagram index.
- [x] 4.4 Make a missing or unreadable referenced `.svg` a fatal error naming the file — never a silent gap.
- [x] 4.5 Confirm that without `--inline-svg` and `--css` the tool reads no file other than its input, including when referenced images do not exist.

## 5. Fixtures

- [x] 5.0 Extend `fixtures_test.go` with an optional `<name>.flags` sidecar supplying per-fixture arguments, and document the convention in `AGENTS.md`. (Added scope, approved mid-implementation: the harness ran every fixture bare, making 5.4 and 5.5 unreachable.)
- [x] 5.1 Add `fixtures/mdhtml/` covering GFM constructs, footnotes, tables, code blocks, and blockquotes.
- [x] 5.2 Add a frontmatter fixture proving the block is stripped, and a companion proving a mid-document `---` still renders as `hr`.
- [x] 5.3 Add a fixture piping `mdsidenote` output through `mdhtml`, proving the sidenote markup survives unchanged.
- [x] 5.4 Add a `--fragment` fixture and an `--inline-svg` fixture with a committed `.svg`, including the same file referenced twice to prove identifiers do not collide.
- [x] 5.5 Prefer `--no-css` or `--fragment` in fixtures wherever possible so future stylesheet edits do not cascade through golden files.
- [x] 5.6 Run `make test` and confirm `mdhtml` fixtures run correctness subtests without idempotency subtests.

## 6. Documentation and release

- [x] 6.1 Add a "Rendering" section to `README.md` describing `mdhtml` and its flags, matching the existing tool-description style.
- [x] 6.2 Add `CHANGELOG.md` entries under `## [Unreleased]`.
- [x] 6.3 Run `make fmt`, `make build`, `make test`, and `go vet ./...` clean.
- [x] 6.4 Verify end to end by hand: `bin/mdhtml doc.md > doc.html` opens correctly styled in a browser, `bin/mdsidenote doc.md | bin/mdhtml` keeps sidenotes, and `bin/mdhtml --inline-svg doc.md` produces inline `svg` elements.

## 7. SVG trust boundary

Added after review: `-inline-svg` moves an SVG from an image context, where browsers
neither execute scripts nor load external resources, into the live document where
both happen. Folded in rather than shipped and patched, since nothing is released.

- [x] 7.1 Add `vetSVG(data []byte) (warnings []string, err error)` to `cmd/mdhtml/`, walking the token stream with `encoding/xml` rather than pattern-matching, so entity-encoded and CDATA-hidden constructs are seen.
- [x] 7.2 Refuse `script`, `iframe`, `embed`, `object`, `foreignObject`, any `on*` attribute, `javascript:` in `href`/`xlink:href`/`src`, `use` with a non-fragment reference, `animate`/`set` targeting `href`, and `@import` in a `style` block. Each error names the file and the construct.
- [x] 7.3 Refuse any `.svg` that fails to parse — a file that cannot be inspected cannot be vouched for.
- [x] 7.4 Mention `htmlLabels: false` in the `foreignObject` error, since stock `mmdc` output trips it and that setting is the fix.
- [x] 7.5 Collect external resource references (`image` with an external `href`, `url(https://…)` in CSS) as warnings rather than refusals — they leak the reader's address but do not execute.
- [x] 7.6 Add `-unsafe-svg` to skip inspection; wire vetting between `os.ReadFile` and `cleanSVG` in `inlineSVGRefs`.
- [x] 7.7 Warn on stderr when a `.svg` reference is skipped for carrying a scheme, keeping stdout clean for pipelines.
- [x] 7.8 Add `flags_test.go` cases: script refused, `on*` refused, `foreignObject` refused with the hint, unparseable refused, `-unsafe-svg` inlines anyway, remote reference warns while stdout stays clean.
- [x] 7.9 Confirm `fixtures/mdhtml/diagram.svg` still passes vetting and all 120 subtests stay green.
- [x] 7.10 Document `-unsafe-svg` and the trust boundary in `README.md` and `CHANGELOG.md`, stating plainly that the check is a guard against accident, not a guarantee.
