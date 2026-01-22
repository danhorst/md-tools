
title: This is a complex document with lots of different uses of markdown links
category: fixture
---

This is a complex case for processing inline [links][1] to reference-style links in a document with [YAML frontmatter][2][^1] and many different markdown features.

> Links in [blockquotes][3] should also be processed.

Each inline [link][1] should be converted to a reference-style link at the end of the document with an incrementing integer, starting at one, as you traverse the document. Links should be de-duplicated if there is more than one occurrence of the [link][1], with the _first_ appearance of the link determining the integer reference.

One exception is _image_ links. Image links should be left _inline_ to facilitate post-processing of image markup. Here's an example just in case:

![Example image](/images/exmple.png "Image title")

Inline image links should be the _default_ but it might be a good idea to add a flag for processing them as reference style.

[^1]: It is easy to confuse a single YAML property with a "underline" style H2. But, even in footnotes, [links][1] should be pulled out into reference-style.

[1]: https://daringfireball.net/projects/markdown/syntax#link
[2]: https://jekyllrb.com/docs/front-matter/
[3]: https://daringfireball.net/projects/markdown/syntax#blockquote "This blockquote has a title."
