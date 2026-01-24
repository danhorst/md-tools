# Code Block Edge Cases

Links in regular text should be converted: [example][1]

## Fenced Code Blocks

Links inside fenced code blocks should NOT be converted:

```markdown
Here is an example link: [link](https://example.com/in-fenced-block)
```

```go
// Link in Go code
url := "[docs](https://golang.org)"
```

## Inline Code

Links in `[inline code](https://example.com/in-inline)` should not be converted.

But a link after inline code `like this` should work: [after code][2]

## Indented Code Blocks

Indented code blocks should also be preserved:

    [link](https://example.com/in-indented-block)

Back to normal text with a [real link][3].

[1]: https://example.com
[2]: https://example.com/after-inline
[3]: https://example.com/real
