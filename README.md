# Markdown Tools

Small, sharp tools for manipulating markdown.
I've tailored them to my specific tastes for authoring and presenting plain text.
Maybe they'll be helpful for you too.[^1]

## Installation

There is a [Homebrew][1] formula for `md-tools` in [`danhorst/homebrew-tap`][2].
To install these tools with that formula, use:

```bash
brew tap danhorst/tap && brew install danhorst/tap/md-tools
```

If you don't want to use Homebrew and have [golang][3] set up already, `./bin/md-install` builds and installs all the executables into your `$GOPATH`.
Pre-built binaries are not available at this time.

## Usage

These utilities are built with the [Unix Philosophy][4] in mind.
They accept text via [`STDIN`][5] and output to `STDOUT`.
This let's you _chain_ them with [Unix pipes][6] (`|`).
Use the `-w` flag to replace the contents of a provided file instead of printing to `STDOUT`.

The commands are (mostly) set up in pairs, each responsible for applying or reverting a style convention:

### Links

- `mdref` converts inline-style links to a tidy list of _numbered_ reference-style links at the bottom of the document. Most of the tooling out there to do manipulation like this—[pandoc][7] et. al.—use a text for the link reference, not a number.
- `mdinline` converts all reference-style links to inline links.

### Annotations

- `mdfnt` renumbers footnote references (`[^label]`) to sequential integers in order of first appearance, updating the corresponding definitions.
- `mdsidenote` converts markdown footnotes into HTML literals for [sidenotes][8] that can be styled with [Tufte CSS][9] (or a derivative).
- `mdfootnote` attempts to convert HTML markup for sidenotes back into markdown footnotes.

### Sentence structure

- `mdsplit` takes paragraphs where all the sentences aren't separated by new lines (like [iA Writer][10] expects) and splits each sentence onto it's own line.
- `mdjoin` takes text written in [one sentance per line][11] (the way I like to do it in `vim`) and gloms them together into contiguous paragraphs.

### Tables

- `mdtable` normalizes GFM table column widths so all cells in each column are padded to equal width, making tables visually aligned in plain text.

## Hard wrapping

- `mdwrap` wraps body text to 60 characters. Specify an arbitrary column count with the`-c` flag.
- `mdunwrap` removes hard wrapping and returns text into contiguous paragraphs.

## Colophon

> [!NOTE]
> I don't really know [golang][3] or have much experience with how to properly parse and manipulate text files.
> The heavy lifting was done by [Claude Code][12] in [Zed][13].

Does that make this a "[vibe coding][14]" project?
Sort of.
I've been looking at the source, telling the agent to do things, controlling git commits, and maintaining tight control over the `fixtures` that define success.
The results aren't _stellar_ but they solve problems I've had for a long time and never got around to coding a solution myself.

Although I used [Claude Code][12], the instructions are all in `AGENTS.md`.
There is a utility script (`bin/agent-setup`) that takes care of symlinking `AGENTS.md` to `CLAUDE.md`.

[^1]: Assuming you don't just tell an LLM to do all your formatting for you—what a waste of tokens!

[1]: https://brew.sh/
[2]: https://github.com/danhorst/homebrew-tap
[3]: https://go.dev/
[4]: https://en.wikipedia.org/wiki/Unix_philosophy
[5]: https://en.wikipedia.org/wiki/Standard_streams
[6]: https://en.wikipedia.org/wiki/Pipeline_(Unix)
[7]: https://pandoc.org/
[8]: https://gwern.net/sidenote
[9]: https://edwardtufte.github.io/tufte-css/
[10]: https://ia.net/writer
[11]: https://sive.rs/1s
[12]: https://claude.com/product/claude-code
[13]: https://zed.dev/
[14]: https://en.wikipedia.org/wiki/Vibe_coding
