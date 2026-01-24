
title: This is a complex document with lots of different uses of markdown links
category: fixture
---

This is a complex case for processing inline [links](https://daringfireball.net/projects/markdown/syntax#link) to reference-style links in a document with [YAML frontmatter](https://jekyllrb.com/docs/front-matter/)[^1] and many different markdown features.

> Links in [blockquotes](https://daringfireball.net/projects/markdown/syntax#blockquote "This blockquote has a title.") should also be processed.

Each inline [link](https://daringfireball.net/projects/markdown/syntax#link) should be converted to a reference-style link at the end of the document with an incrementing integer, starting at one, as you traverse the document. Links should be [de-duplicated](https://en.wikipedia.org/wiki/Data_deduplication) if there is more than one occurrence of the [link](https://daringfireball.net/projects/markdown/syntax#link), with the _first_ appearance of the link determining the integer reference.

One exception is _image_ links. Image links should be left _inline_ to facilitate post-processing of image markup. Here's an example just in case:

![Example image](/images/exmple.png "Image title")

Inline image links should be the _default_ but it might be a good idea to add a flag for processing them as reference style.

Because this tool is meant to be able to be used _incrementally_ then we may need to update, or remove a block of links at the end of the document during the processing step.

[^1]: It is easy to confuse a single YAML property with a "underline" style H2. But, even in footnotes, [links](https://daringfireball.net/projects/markdown/syntax#link) should be pulled out into reference-style.
