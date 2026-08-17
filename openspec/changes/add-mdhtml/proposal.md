## Why

`md-tools` can generate Tufte-style sidenote markup with `mdsidenote`, but has no way to render it — the output is HTML literals inside a markdown file that nothing in the toolkit can turn into a viewable document.
More broadly, every tool here manipulates markdown source and stops there, so the toolkit cannot produce a finished artifact.

`mdhtml` closes that gap with no new dependencies: goldmark is already a direct dependency, and rendering markdown to HTML needs no browser and no external runtime.
It is also the first step of a markdown→HTML→PDF pipeline whose remaining, browser-dependent stages are deliberately being built in a separate repository.

## What Changes

- New `mdhtml` binary rendering a markdown document to standalone HTML on STDOUT.
- Configurable output: `--css FILE` to override the embedded stylesheet, `--no-css` for bare output, `--fragment` to emit only the body, `--title STRING` for the document title.
- `--inline-svg` reads referenced `.svg` files and inlines them as `<svg>` elements, producing a self-contained document. Relative references resolve against the input file's directory; with STDIN input the flag requires `--base DIR`.
- YAML frontmatter is stripped rather than rendered.
- An embedded default stylesheet, Tufte-derived to match `mdsidenote`, including `@media print` rules.
- `AGENTS.md` gains an "Output Formats" section permitting markdown→other-text-format tools, exempting them from the idempotency requirement and the `.out.md` fixture convention, and stating that tools requiring an external runtime do not belong in this repository.
- `fixtures_test.go` discovers `.out.*` siblings instead of hardcoding `.out.md`, and runs the idempotency subtest only when the output extension is `.md`.

Not breaking. No existing tool changes behavior, and no existing fixture changes.

## Capabilities

### New Capabilities

- `html-rendering`: Rendering a markdown document to HTML — document structure, stylesheet handling, frontmatter, and resolution of referenced SVG assets into a self-contained document.

### Modified Capabilities

None. No existing tool's requirements change.

## Impact

**New code**

- `cmd/mdhtml/main.go`
- `internal/render/` — shared goldmark configuration and the `go:embed`ded default stylesheet
- `fixtures/mdhtml/`

**Modified**

- `AGENTS.md` — new "Output Formats" section
- `fixtures_test.go` — generalized output-extension discovery, conditional idempotency subtest
- `README.md` — new "Rendering" section
- `CHANGELOG.md` — `[Unreleased]`

**Dependencies**

None added. `go.mod` is unchanged — `github.com/yuin/goldmark` is already a direct dependency, used by `mdref`, `mdinline`, `mdfnt`, and `mdsidenote`.

**Build and CI**

Unchanged. `Makefile`, `bin/md-build`, `bin/md-install`, and `.github/workflows/ci.yml` all discover `cmd/*` automatically. No browser in CI; `make test` stays fast.

**Explicitly out of scope**

`mdmermaid` and `mdpdf`. Both require headless Chrome and belong in a separate `md-render` repository — a boundary this change writes into `AGENTS.md` so it is not reopened.
