title: This is an example of an extremely long title in YAML frontmatter that would normally be wrapped.
---

I like to write markdown in `vim` using the one-sentence-per line rule.
A sentence is a complete idea and keeping it on a line by itself makes it easy to visualize and move around.
This means that paragraphs will consist of a stack of unwrapped sentences.
In its raw form, this is good for me, the author, and less good for the reader.
When _reading_ text, the paragraphs should be wrapped to 80 characters[^1] with a ragged left alignment [^2].

The rest of this text has been reused from the simplest case for processing inline [links][1] to reference-style links in a document with [YAML frontmatter][2].
It is easy to confuse a single YAML property with a "underline" style H2.

Each inline link should be converted to a reference-style link at the end of the document with an incrementing integer, starting at one, as you traverse the document.
Links should be de-duplicated if there is more than one occurrence of the link, with the _first_ appearance of the link determining the integer reference.

[^1]: Maybe this should be configurable?
[^2]: You can't justify monospace text without introducing hard hyphens and we're not going to go there.

[1]: https://daringfireball.net/projects/markdown/syntax#link
[2]: https://jekyllrb.com/docs/front-matter/#fake_id_to_make_this_link_beyond_the_length_of_the_word_wrap
