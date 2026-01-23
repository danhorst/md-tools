title: This is an example of an extremely long title in YAML frontmatter that would normally be wrapped.
---

I like to write markdown in `vim` using the one-sentence-per
line rule. A sentence is a complete idea and keeping it on a
line by itself makes it easy to visualize and move around.
This means that paragraphs will consist of a stack of
unwrapped sentences. In its raw form, this is good for me,
the author, and less good for the reader. When _reading_
text, the paragraphs should be wrapped to 60 characters[^1]
with a ragged left alignment [^2].

There are some particularities with Markdown that should
change how wrapping happens. One case is _explicit_ line
breaks (two trailing spaces). Another case is blockquotes,
especially blockquotes with [GFM alerts][1].

The rest of this text has been reused from the simplest case
for processing inline [links][2] to reference-style links in
a document with [YAML frontmatter][3]. It is easy to confuse
a single YAML property with a "underline" style H2.

> This is a normal blockquote with long enough sentences
> that they need to be wrapped. The line length _includes_
> the leading syntax for the blockquote.

This prose is short.  
But it is wrapped on purpose.  
Each line has two trailing spaces.  

> [!NOTE]
> This blockquote is an "alert" or "callout" style because
> it has a special heading. The _heading_ should not be
> wrapped but the rest of the text should be.

Each inline link should be converted to a reference-style
link at the end of the document with an incrementing
integer, starting at one, as you traverse the document.
Links should be de-duplicated if there is more than one
occurrence of the link, with the _first_ appearance of the
link determining the integer reference.

[^1]: Lines with 50–70 characters give the best reading experience. This tool will default to 60 characters but can be customized with the `-c` flag.
[^2]: You can't justify monospace text without introducing hard hyphens and we're not going to go there.

[1]: https://docs.github.com/en/get-started/writing-on-github/getting-started-with-writing-and-formatting-on-github/basic-writing-and-formatting-syntax#alerts
[2]: https://daringfireball.net/projects/markdown/syntax#link
[3]: https://jekyllrb.com/docs/front-matter/#fake_id_to_make_this_link_beyond_the_length_of_the_word_wrap
