title: This is an example title in YAML frontmatter.
---

This is the simplest case for processing inline [links][1] to reference-style links in a document with [YAML frontmatter][2]. It is easy to confuse a single YAML property with a "underline" style H2.

Each inline link should be converted to a reference-style link at the end of the document with an incrementing integer, starting at one, as you traverse the document. Links should be de-duplicated if there is more than one occurrence of the link, with the _first_ appearance of the link determining the integer reference.

[1]: https://daringfireball.net/projects/markdown/syntax#link
[2]: https://jekyllrb.com/docs/front-matter/
