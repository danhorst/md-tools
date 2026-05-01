# Changelog

## v1.0.2 — 2026-04-30

### Bug fixes

- **`mdref`** — fix silent skip of inline links whose text begins with emphasis or strong markup (e.g. `[_foo_](url)`, `[**bar**](url)`).

## v1.0.1 — 2026-04-29

### Bug fixes

- **`mdref`** — fix silent skip of inline links whose text is a code span (e.g. `` [`Foo`](url) ``).

## v1.0.0 — 2026-04-29

Initial release. Ten composable CLI tools for manipulating GitHub Flavored Markdown.

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
