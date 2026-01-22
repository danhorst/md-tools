# Markdown Tools

Small, sharp tools for manipulating markdown that are specific to the way I write and publish text.
Maybe they'll be of some use to you too.

- `mdref` converts inline-style links to a tidy list of reference-style _numbered_ links at the bottom of the document. Most of the tooling out there to do this kind of manipulation—like [pandoc][1]—uses a text representation not a number.
- `mdsidenote` converts markdown footnotes into HTML literals that can be styled with [Tufte CSS][2] (or a derivative).

I don't really know [golang][3] or have much experience with how to properly parse and manipulate text files.
The heavy lifting was done by [Claude Code][4] in [Zed][5].
Does that make this a "[vibe coding][6]" project?
Sort of.
I've been looking at the source, telling the agent to do things, controlling source control, and maintaining tight control over the `fixtures` that define success.

[1]: https://pandoc.org/
[2]: https://edwardtufte.github.io/tufte-css/
[3]: https://go.dev/
[4]: https://claude.com/product/claude-code
[5]: https://zed.dev/
[6]: https://en.wikipedia.org/wiki/Vibe_coding
