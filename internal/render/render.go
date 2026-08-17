// Package render converts Markdown into HTML documents.
package render

import (
	_ "embed"
	"fmt"
	stdhtml "html"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

// DefaultCSS is the stylesheet embedded in standalone documents unless the
// caller overrides or suppresses it.
//
//go:embed default.css
var DefaultCSS string

// New returns the goldmark instance used for rendering. Raw HTML passes
// through unescaped so markup produced by other tools in the toolkit —
// mdsidenote's sidenotes in particular — survives rendering.
func New() goldmark.Markdown {
	return goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Footnote),
		goldmark.WithRendererOptions(html.WithUnsafe()),
	)
}

// Document wraps rendered body HTML in a standalone document. An empty css
// omits the style element entirely.
func Document(body, title, css string) string {
	var b strings.Builder

	b.WriteString("<!DOCTYPE html>\n<html>\n<head>\n")
	b.WriteString("<meta charset=\"utf-8\">\n")
	b.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">\n")
	fmt.Fprintf(&b, "<title>%s</title>\n", stdhtml.EscapeString(title))

	if css != "" {
		b.WriteString("<style>\n")
		b.WriteString(css)
		if !strings.HasSuffix(css, "\n") {
			b.WriteString("\n")
		}
		b.WriteString("</style>\n")
	}

	b.WriteString("</head>\n<body>\n")
	b.WriteString(strings.TrimRight(body, "\n"))
	b.WriteString("\n</body>\n</html>\n")

	return b.String()
}
