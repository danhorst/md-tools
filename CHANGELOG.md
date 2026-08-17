# Changelog

## [Unreleased]

### New tools

- **`mdhtml`** — render a markdown document to standalone HTML. Emits a complete document by default (`-fragment` for body content only), with an embedded Tufte-derived stylesheet sized to `mdsidenote`'s sidenote markup — override with `-css FILE`, drop with `-no-css`. `-title` sets the document title, defaulting to the input file's base name. YAML frontmatter is stripped rather than rendered. `-inline-svg` replaces referenced `.svg` files with their contents inline, namespacing identifiers so the same diagram can appear twice without colliding; relative references resolve against the input file's directory, or `-base DIR` when reading from stdin. `-w` is refused, since it would overwrite the markdown source with HTML — use `-i FILE` or shell redirection.

  GFM tables (with per-column alignment), task lists, strikethrough, autolinks, and footnotes all render. GitHub alerts (`> [!NOTE]`, `[!TIP]`, `[!IMPORTANT]`, `[!WARNING]`, `[!CAUTION]`) become callouts carrying GitHub's `markdown-alert` class names — they aren't in the GFM spec and goldmark doesn't implement them, but `AGENTS.md` names GFM as the target flavor and this repo's own README uses them. Markers are matched case-insensitively; an unrecognized marker leaves the blockquote alone. Raw HTML still passes through unfiltered, unlike GitHub's tagfilter, since that is what lets `mdsidenote` output survive rendering.

  Inlining moves an SVG out of an image context, where browsers execute no scripts and load nothing external, into the live document where both happen. `mdhtml` therefore refuses to inline an SVG containing a `<script>`, an `on*` handler, a `<foreignObject>`, a `javascript:` URL, a `<use>` pointing at another document, SMIL rewriting a link target, or an `@import`, and refuses any file it cannot parse. `-unsafe-svg` skips the check. Inspection walks the XML token stream rather than matching patterns, so entity-encoded payloads are caught; it is a guard against accident, not a guarantee against a determined attacker. External resource references warn rather than refuse — they leak the reader's address without executing. Remote `.svg` references are never fetched, since that would break deterministic output; they stay `<img>` tags and warn on stderr.

### Changes

- `AGENTS.md` gains an "Output Formats" section permitting markdown → other-text-format tools, exempting them from idempotency and the `.out.md` fixture convention, and stating that tools requiring an external runtime do not belong in this repository.
- Fixtures may supply per-fixture command-line arguments in an optional `<name>.flags` file, and the golden sibling is now discovered as `<name>.out.<ext>` rather than assumed to be `.out.md`. Idempotency is only asserted for tools whose output is markdown.
- Frontmatter detection moves out of `markdown.Transform` into `markdown.SplitFrontmatter`. No behavior change — every existing fixture passes unaltered.

## [1.1.6] - 2026-08-06

### Bug fixes

- **`mdsplit`** — don't split a sentence after a known abbreviation (`e.g.`, `i.e.`, `n.b.`, `et al.`, and the honorifics `Mr.`/`Mrs.`/`Dr.`/`Ms.`/`St.`/`Jr.`/`Sr.`) even when followed by a capitalized word or digit, e.g. `(e.g. 15m/20m)` no longer splits after `e.g.`. `etc.` is deliberately excluded — it's commonly used as a genuine sentence terminus, and telling that apart from mid-sentence use needs a look-ahead this matcher doesn't do.

## [1.1.5] - 2026-07-14

### Changes

- Update all dependencies to latest (`golang.org/x/net` v0.47.0 → v0.57.0, `goldmark` v1.7.16 → v1.8.4, `html-to-markdown/v2` v2.5.0 → v2.5.2, `dom` v0.2.0 → v0.3.1); bumps `go.mod` toolchain directive to 1.25.0.

## [1.1.4] - 2026-06-21

### Features

- **`mdwrap`** — add `-f` to wrap footnote bodies to the column width, indenting continuation lines four spaces (valid footnote continuation in strict engines).

### Bug fixes

- **`mdsplit`** — never split sentences inside a footnote definition. A definition and its continuation lines are now treated as one block and passed through unchanged, so multi-sentence footnotes stay portable. This also applies to `mdjoin`, `mdwrap`, and `mdunwrap`.

## [1.1.3] - 2026-06-21

### Bug fixes

- **`mdsplit`** — don't split a sentence inside an inline span (code spans, links, emphasis `*`/`_`/`**`, and strikethrough `~~`); previously a sentence boundary within the span orphaned its delimiters across lines. A sentence that ends inside a span (e.g. `**Done.** Next`) now breaks after the span.

## [1.1.2] - 2026-06-21

### Bug fixes

- **`mdsplit`** — fix sentences being split inside inline footnotes (`^[...]`), and fix two sentences being joined onto one line when a footnote (`[^1]` or `^[...]`) follows the terminal punctuation of the first.

## [1.1.1] - 2026-06-16

### Features

- Add `-v` and `-version` flags to every tool, printing `<tool> <version>`.

### Changes

- `-i` is now a boolean flag with the destination file passed as a positional argument (mirroring `-w`). The command form `mdsplit X | mdtable -i X` is unchanged.
- Aligned help output: all flag names line up in a single column regardless of length.

## [1.1.0] - 2026-06-16

### Features

- Add `-i FILE` flag to every tool: read stdin, write the transformed result to `FILE`. Enables the natural pipe chain `mdsplit X | mdtable -i X` to round-trip a file through both transforms in one command. `-i` is mutually exclusive with `-w` and requires data on stdin.

## [1.0.6] - 2026-06-02

### Bug fixes

- **`mdtable`** — fix `|` inside backtick code spans being treated as a column delimiter.

## [1.0.5] - 2026-05-28

### Bug fixes

- **`mdtable`** — fix column misalignment when cells contain multi-byte Unicode characters (e.g. `—`, `→`); width was measured in bytes instead of display columns.
- **`mdwrap`** — fix lines wrapping too early when words contain multi-byte Unicode characters; same byte-vs-rune miscounting as `mdtable`.

## [1.0.4] - 2026-05-27

### Bug fixes

- **`mdsplit`** — fix sentence boundary not recognized when the next sentence starts with a backtick (or any non-lowercase character such as `_`, `[`, `*`, or a digit).

## [1.0.3] - 2026-04-30

### Tooling

- Add `scripts/release` to automate tagging, SHA256 computation, and Homebrew tap updates.

## [1.0.2] - 2026-04-30

### Bug fixes

- **`mdref`** — fix silent skip of inline links whose text begins with emphasis or strong markup (e.g. `[_foo_](url)`, `[**bar**](url)`).

## [1.0.1] - 2026-04-29

### Bug fixes

- **`mdref`** — fix silent skip of inline links whose text is a code span (e.g. `` [`Foo`](url) ``).

## [1.0.0] - 2026-04-29

Initial release.
Ten composable CLI tools for manipulating GitHub Flavored Markdown.

### Links

- **`mdref`** — converts inline-style links to numbered reference-style links collected at the bottom of the document.
- **`mdinline`** — converts reference-style links back to inline links.

### Annotations

- **`mdfnt`** — renumbers footnote references (`[^label]`) to sequential integers in order of first appearance, updating the corresponding definitions.
- **`mdsidenote`** — converts markdown footnotes to HTML sidenote markup compatible with Tufte CSS.
- **`mdfootnote`** — converts Tufte CSS sidenote HTML markup back to markdown footnotes.

### Sentence structure

- **`mdsplit`** — splits sentences within paragraphs onto individual lines (one sentence per line).
- **`mdjoin`** — joins one-sentence-per-line paragraphs back into contiguous paragraph text.

### Tables

- **`mdtable`** — normalizes GFM table column widths, padding each cell to the column maximum for visual alignment in plain text.

### Hard wrapping

- **`mdwrap`** — wraps body text to 60 characters (configurable with `-c`). Table and list structure is preserved.
- **`mdunwrap`** — removes hard line breaks, restoring text into contiguous paragraphs. Table and list structure is preserved.

[Unreleased]: https://github.com/danhorst/md-tools/compare/v1.1.6...HEAD
[1.1.6]: https://github.com/danhorst/md-tools/compare/1.1.5...1.1.6
[1.1.5]: https://github.com/danhorst/md-tools/compare/1.1.4...1.1.5
[1.1.4]: https://github.com/danhorst/md-tools/compare/1.1.3...1.1.4
[1.1.3]: https://github.com/danhorst/md-tools/compare/1.1.2...1.1.3
[1.1.2]: https://github.com/danhorst/md-tools/compare/1.1.1...1.1.2
[1.1.1]: https://github.com/danhorst/md-tools/compare/1.1.0...1.1.1
[1.1.0]: https://github.com/danhorst/md-tools/compare/1.0.6...1.1.0
[1.0.6]: https://github.com/danhorst/md-tools/compare/1.0.5...1.0.6
[1.0.5]: https://github.com/danhorst/md-tools/compare/1.0.4...1.0.5
[1.0.4]: https://github.com/danhorst/md-tools/compare/1.0.3...1.0.4
[1.0.3]: https://github.com/danhorst/md-tools/compare/v1.0.2...v1.0.3
[1.0.2]: https://github.com/danhorst/md-tools/compare/v1.0.1...v1.0.2
[1.0.1]: https://github.com/danhorst/md-tools/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/danhorst/md-tools/releases/tag/v1.0.0
