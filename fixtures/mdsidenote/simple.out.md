title: Simple footnote conversion
category: fixture
---

This is an example of a paragraph that includes a footnote.
Since I use a theme that's based on [Tufte CSS](https://edwardtufte.github.io/tufte-css/), the normal markdown footnote formatting doesn't work with the sidenote styles.
My ideal publishing workflow would let me author my markdown in my preferred style and tool chain then produce tidy, theme-ready HTML and polished, formatted plain-text representations of the same content.
<label for="sidenote-1" class="margin-toggle sidenote-number"></label>
<input type="checkbox" id="sidenote-1" class="margin-toggle"/>
<span class="sidenote">
The thing to note here is that the markdown I <em>write</em> is not the same as the markdown I want to <em>present</em>.
</span>
In lieu of a site builder that does all of that, I have been manually converting my markdown footnotes into the inline HTML needed for sidenotes.

