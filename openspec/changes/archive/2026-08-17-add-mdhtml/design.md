## Context

See `proposal.md` — Why.

Constraints that shape the approach:

- `AGENTS.md` is binding and states five invariants that every tool satisfies today: one transformation per binary, STDIN/STDOUT with defined exit codes, byte-for-byte determinism, idempotency `T(T(x)) == T(x)`, and golden `.in.md`/`.out.md` fixtures.
  `mdhtml` satisfies all of them except idempotency and the `.out.md` convention, because its output is not in the input domain.
- `AGENTS.md` also forbids inferring behavior from context: "Automatic heuristics beyond explicitly defined behavior" is an explicit non-goal.
- `fixtures_test.go` auto-discovers `fixtures/*/*.in.md`, hardcodes a `.out.md` sibling, and runs an idempotency subtest for every fixture with no opt-out.
- `cli.Run`'s `TransformFunc` is `func(string) string` and is shared by all ten existing tools.
- goldmark is already a direct dependency. `cmd/mdsidenote/main.go:68` establishes the configuration pattern, including `html.WithUnsafe()`.

## Goals / Non-Goals

**Goals:**

- Keep the markdown-facing work pure Go, deterministic, and covered by the existing fixture harness.
- Amend `AGENTS.md` narrowly enough that its remaining invariants keep their force.
- Leave the ten existing tools byte-identical in behavior.

**Non-Goals:**

- Any browser or external runtime. That boundary is the subject of a decision below.
- A general-purpose HTML sanitizer. Raw HTML passes through by design.
- Templating, partials, or a site generator. One document in, one document out.

## Decisions

### Split on the browser dependency, not on the feature

The original design put `mdmermaid`, `mdhtml`, and `mdpdf` all in this repo.
That required exempting renderers from idempotency, byte determinism, golden fixtures, and the static-binary rule — four of five invariants, which is not an amendment but a second constitution.
`AGENTS.md` works as an agent-control document precisely because its invariants are absolute; once it says "except for renderers," every future change begins with an agent deciding which category it is in.

The costs are not evenly distributed across the three tools.
`mdmermaid` and `mdpdf` need headless Chrome, which means `chromedp` plus `cdproto` — by far the largest dependency this repo would take on — a vendored 3MB JavaScript blob, and a browser in CI.
`mdhtml` needs none of that: goldmark is already here, it is deterministic, and it is golden-file testable.

So the line falls on the browser dependency rather than on "rendering." `mdhtml` stays; `mdmermaid` and `mdpdf` go to a separate `md-render` repo.
`md-tools` keeps its fast, dependency-light `brew install`, and the heavy dependency lands only on people who want PDFs.

*Alternative considered:* move all three out, leaving `md-tools` untouched.
Rejected because `mdhtml` costs this repo essentially nothing and is the natural companion to `mdsidenote`, which currently generates Tufte sidenote markup that nothing in the toolkit can render.

### The `AGENTS.md` amendment closes the door behind itself

The new "Output Formats" section permits markdown→other-text-format tools and exempts them from idempotency and the `.out.md` convention — nothing more.
Determinism, STDIN/STDOUT, exit codes, and one-transformation-per-binary continue to apply to `mdhtml` unchanged, because it satisfies them.

It also states that tools requiring an external runtime do not belong in this repository.
Without that sentence the amendment reads as an invitation to add `mdpdf` here later, which is the outcome the split exists to prevent.

### The seam with `md-render` is ordinary markdown

`mdmermaid` will replace a ` ```mermaid ` fence with a plain `![](doc-diagram-1.svg)` image reference.
`mdhtml` inlines any referenced SVG and knows nothing about Mermaid.
There is no private protocol between the repos and no ordering dependency in the code — only in the user's pipeline.

This also means `--inline-svg` is not speculative work for a tool that does not exist yet: inlining referenced SVG is useful for any figure.

### `--inline-svg` is an explicit flag, not inferred

Self-contained output could be inferred — inline when invoked with a file argument, skip when reading STDIN.
`AGENTS.md` rules that out, and it is the right call here: the failure mode of getting it wrong is silent.
An unresolved relative reference produces blank space in the eventual PDF with no error.

Default off also keeps the common path a strict text filter that reads nothing but its input, which is what makes golden-file testing trivial.
When input is on STDIN there is no directory to resolve against, so the flag requires `--base DIR` rather than guessing the working directory.

### `-w` is rejected rather than left undefined

Across the toolkit `-w` rewrites the input file with the result. For `mdhtml` that would replace a markdown source file with HTML — silent data loss for a habitual `-w` user.

Simply not registering it yields Go's `flag provided but not defined: -w`, which is accurate but unhelpful.
Registering it and failing with an explanation that names the alternatives (`-i FILE`, shell redirection) is a few lines and prevents the accident.

`-i` is kept: it writes the result to a named file, which is safe and useful at the end of a pipe.

### SVG identifiers are namespaced by prefixing declared ids

Mermaid SVGs carry an internal `<style>` block whose rules are scoped under the root element's id, plus `url(#…)` and `href="#…"` references to markers and gradients.
Inlining two of them into one document collides — guaranteed for two copies of the same diagram, since deterministic id generation makes identical content produce identical ids.

Approach: collect every id declared in the file, then for each one rewrite `id="X"` to `id="pN-X"` and every `#X` occurrence to `#pN-X`, where `N` is the diagram's index in the document.
Rewriting only ids the file actually declares avoids touching unrelated fragment links, and covering `#X` in one pass handles `href`, `url()`, and CSS selectors together.

*Alternative considered:* parse the SVG as XML and rewrite the tree.
Rejected — an XML round-trip risks reformatting the document, and the `<style>` block's CSS selectors are opaque text that a tree rewrite would miss anyway.

### Frontmatter detection is extracted, not duplicated

`Transform` in `internal/markdown/transform.go:30-60` already detects frontmatter in two formats — `---`/`---` and a bare property line terminated by `---` — but passes it through, and the logic is inline rather than exposed.
`mdhtml` needs to strip it.

Extract a `SplitFrontmatter(content) (frontmatter, body string)` into `internal/markdown` and have `Transform` call it.
Two copies of two-format frontmatter detection would drift.

This touches code shared by all ten tools, which is normally out of bounds for a change this size.
It is acceptable here because `Transform`'s behavior must stay byte-identical and the existing `frontmatter` fixtures across the tool suite prove it — that is what the golden files are for. If any existing fixture moves, the extraction is wrong.

### The fixture harness generalizes by two lines of intent

`fixtures_test.go` gains exactly two behavior changes: discover the sibling as `.out.*` rather than `.out.md`, and run the idempotency subtest only when the output extension is `.md`.

Everything else stays — auto-discovery, per-tool build, stdin piping, byte comparison.
`mdhtml` is then covered by the same harness as every other tool, with `.out.html` goldens.

*Alternative considered:* a separate test file for `mdhtml`.
Rejected because it would duplicate the build-and-run scaffolding and put `mdhtml` outside the mechanism that makes adding fixtures free.

### `internal/render` holds the goldmark configuration and embedded assets

A package rather than inlining in `cmd/mdhtml/main.go`, because the default stylesheet is a `go:embed` asset and the goldmark configuration needs to match `mdsidenote`'s.
It stays internal; there is no public API to keep stable.

## Risks / Trade-offs

**`html.WithUnsafe()` means raw HTML passes through unfiltered** → Required, and the point: `mdsidenote`'s output is HTML literals in a markdown file, and rendering it is the main reason `mdhtml` stays in this repo. The consequence is that `mdhtml` must not be pointed at untrusted markdown. It is a local authoring tool operating on the user's own documents; the same assumption already holds for `mdsidenote`.

**`--inline-svg` turns a sandboxed asset into live code** → A browser treats `<img src="x.svg">` as an image: scripts do not run, external fetches are blocked, there is no DOM access. Inlining the same file as `<svg>` puts it in the live document where all three happen. Crossing that boundary is the point — it is what lets the stylesheet restyle diagrams and keeps diagram text selectable in the PDF — but it makes a generated or downloaded asset executable, and `mdhtml` output is meant to be published, so a script that reaches danhorst.com is stored XSS on your own origin.

Mitigated by refusing to inline SVG containing active content — `script`, `on*` handlers, `foreignObject`, `javascript:` URLs, external `use`, SMIL targeting `href`, `@import` — with `-unsafe-svg` to override, and refusing anything that fails to parse.

Refusal rather than sanitizing, deliberately. Detecting constructs in order to reject is tractable and fails safe: over-rejecting costs the user a flag. Sanitizing has to neutralize every vector correctly, and under-stripping fails silently while manufacturing confidence that the output is safe. Go has no maintained SVG sanitizer, and adding a half-correct one would be worse than none.

The check walks the token stream with `encoding/xml` rather than running regexes: a token walk sees names after entity decoding, is not fooled by CDATA, and handles namespace prefixes. Go's decoder does not resolve external entities, so XXE is not a concern. This is stdlib, so no dependency is added.

What it is not: a defense against a determined attacker. mXSS turns on how *browsers* re-parse markup, not on how we parse it. The real guarantee is only inlining SVG you generated, and the README and `--help` say so rather than implying the check is stronger than it is.

**Fetching remote references was rejected** → It would break the byte-for-byte determinism `AGENTS.md` makes binding, since a remote resource can change or be unreachable, and it opens an SSRF path where a document names an internal host and the response lands in HTML you publish. Nothing needs it: `mdmermaid` writes local sidecar files. Remote references stay `img` elements and warn on stderr, so an incomplete document is not mistaken for a self-contained one.

**Id namespacing assumes the SVG scopes its CSS under a root id** → Mermaid does. A hand-written SVG with global class selectors in its `<style>` block can still leak styling across inlined diagrams. Out of scope to fix; namespacing ids is the correct 90% and a global-selector SVG is a pathological input.

**The default stylesheet is taste-driven and will churn** → It ships as an embedded default with `--css` to override and `--no-css` to opt out entirely, so churn in the default never blocks a user with an opinion. Fixtures should use `--no-css` or `--fragment` wherever possible so stylesheet edits do not cascade through golden files.

**Extracting `SplitFrontmatter` touches all ten tools' shared path** → Mitigated by the existing fixture suite, which must pass unchanged. Run `make test` before and after the extraction as a distinct step, not folded into the `mdhtml` work.

**Deriving the title from the input filename is a small heuristic** → Bounded and documented, with `--title` as the explicit override, and empty rather than guessed when reading STDIN. Deriving from the first `H1` was considered and rejected as the kind of content inspection `AGENTS.md` warns against.

## Migration Plan

None required. Purely additive to end users: a new binary, no existing tool's behavior changes, no dependency added, no CI change.

Released as a minor version (`scripts/release v1.2.0`) per `AGENTS.md` — new tool, backwards compatible.
