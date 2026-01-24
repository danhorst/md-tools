# Markdown Tools

Small, sharp tools for manipulating markdown.
I've tailored them to my specific tastes for authoring and presenting plain text.
Maybe they'll be helpful for you too.[^1]

## Usage

These utilities are built with the [Unix Philosophy][1] in mind.
They accept text via [`STDIN`](https://en.wikipedia.org/wiki/Standard_streams) and output to `STDOUT`.
This let's you _chain_ them with [Unix pipes][2] (`|`).
They will replace the contents of a provided filename if you use the `-w` flag.

If you have [golang][3] installed, `./bin/md-install` builds and installs all the executables into your `$GOPATH`.
I do not have pre-built binaries available at this time.

The commands are set up in pairs, each responsible for applying or reverting a style convention:

### Links

- `mdref` converts inline-style links to a tidy list of _numbered_ reference-style links at the bottom of the document. Most of the tooling out there to do manipulation like this—[pandoc][4] et. al.—use a text for the link reference, not a number.
- `mdinline` converts all reference-style links to inline links.

### Annotations

- `mdsidenote` converts markdown footnotes into HTML literals for [sidenotes][5] that can be styled with [Tufte CSS][6] (or a derivative).
- `mdfootnote` attempts to convert HTML markup for sidenotes back into markdown footnotes.

### Sentence structure

- `mdsplit` takes paragraphs where all the sentences aren't separated by new lines (like [iA Writer][7] expects) and splits each sentence onto it's own line.
- `mdjoin` takes text written in [one sentance per line][8] (the way I like to do it in `vim`) and gloms them together into contiguous paragraphs.

## Hard wrapping

- `mdwrap` wraps body text to 60 characters. The `-c` flag lets you set an arbitrary number.
- `mdunwrap` removes hard wrapping and returns text into contiguous paragraphs.

## Colophon

> [!NOTE]
> I don't really know [golang][3] or have much experience with how to properly parse and manipulate text files.
> The heavy lifting was done by [Claude Code][9] in [Zed][10].

Does that make this a "[vibe coding][11]" project?
Sort of.
I've been looking at the source, telling the agent to do things, controlling git commits, and maintaining tight control over the `fixtures` that define success.
The results aren't _stellar_ but they solve problems I've had for a long time and never got around to coding a solution myself.

[^1]: Assuming you don't just tell an LLM to do all your formatting for you—what a waste of tokens!

[1]: https://en.wikipedia.org/wiki/Unix_philosophy
[2]: https://en.wikipedia.org/wiki/Pipeline_(Unix)
[3]: https://go.dev/
[4]: https://pandoc.org/
[5]: https://gwern.net/sidenote
[6]: https://edwardtufte.github.io/tufte-css/
[7]: https://ia.net/writer
[8]: https://sive.rs/1s
[9]: https://claude.com/product/claude-code
[10]: https://zed.dev/
[11]: https://en.wikipedia.org/wiki/Vibe_coding
