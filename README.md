# Markdown Tools

Small, sharp tools for manipulating markdown.
I've tailored them to my specific tastes for authoring and presenting plain text.
Maybe they'll be of some use to you too.

- `mdref` converts inline-style links to a tidy list of _numbered_ reference-style links at the bottom of the document. Most of the tooling out there to do manipulation like this—[pandoc][1] et. al.—use a text for the link reference, not a number.
- `mdsidenote` converts markdown footnotes into HTML literals that can be styled with [Tufte CSS][2] (or a derivative).

`./bin/md-install` builds and installs all the executables into your `$GOPATH`.

> [!NOTE]
> I don't really know [golang][3] or have much experience with how to properly parse and manipulate text files.
> The heavy lifting was done by [Claude Code][4] in [Zed][5].

Does that make this a "[vibe coding][6]" project?
Sort of.
I've been looking at the source, telling the agent to do things, controlling git commits, and maintaining tight control over the `fixtures` that define success.
The results aren't _stellar_ but they solve problems I've had for a long time and never got around to coding a solution myself.

[1]: https://pandoc.org/
[2]: https://edwardtufte.github.io/tufte-css/
[3]: https://go.dev/
[4]: https://claude.com/product/claude-code
[5]: https://zed.dev/
[6]: https://en.wikipedia.org/wiki/Vibe_coding
