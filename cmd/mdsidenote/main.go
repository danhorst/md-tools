// mdsidenote converts Markdown footnotes to Tufte CSS sidenotes.
//
// Usage:
//
//	mdsidenote [file...]
//	cat file.md | mdsidenote
//	mdsidenote -w file.md    # modify file in place
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
)

var writeInPlace = flag.Bool("w", false, "write result to file instead of stdout")

func main() {
	flag.Parse()
	if err := run(flag.Args()); err != nil {
		fmt.Fprintf(os.Stderr, "mdsidenote: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if *writeInPlace {
		if len(args) == 0 {
			return fmt.Errorf("-w requires at least one file argument")
		}
		for _, path := range args {
			if err := processFile(path); err != nil {
				return fmt.Errorf("%s: %w", path, err)
			}
		}
		return nil
	}

	var input io.ReadCloser
	if len(args) == 0 {
		input = os.Stdin
	} else {
		f, err := os.Open(args[0])
		if err != nil {
			return err
		}
		input = f
	}
	defer input.Close()

	data, err := io.ReadAll(input)
	if err != nil {
		return err
	}

	result := transform(string(data))
	_, err = os.Stdout.WriteString(result)
	return err
}

func processFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	result := transform(string(data))

	if result == string(data) {
		return nil
	}

	return os.WriteFile(path, []byte(result), 0644)
}

// footnoteRef represents a footnote reference in the document
type footnoteRef struct {
	start int // byte position of [^label]
	end   int // byte position after [^label]
	index int // the footnote index (from goldmark)
}

// footnoteDef represents a footnote definition
type footnoteDef struct {
	start   int    // byte position of [^label]: ...
	end     int    // byte position after the definition
	ref     string // the footnote reference label
	content string // the rendered HTML content
}

func transform(content string) string {
	source := []byte(content)

	// Create goldmark with footnote extension
	md := goldmark.New(
		goldmark.WithExtensions(extension.Footnote),
		goldmark.WithRendererOptions(html.WithUnsafe()),
	)

	ctx := parser.NewContext()
	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader, parser.WithContext(ctx))

	// Collect footnote references and definitions
	var refs []footnoteRef
	defs := make(map[int]footnoteDef) // keyed by index

	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch node := n.(type) {
		case *extast.FootnoteLink:
			// Find the extent in source
			start, end := findFootnoteRefExtent(node.Index, source)
			if start >= 0 && end >= 0 {
				refs = append(refs, footnoteRef{
					start: start,
					end:   end,
					index: node.Index,
				})
			}

		case *extast.Footnote:
			// Get the footnote content and render to HTML
			refLabel := string(node.Ref)
			htmlContent := renderFootnoteContent(node, source, md)

			// Find the definition extent in source
			start, end := findFootnoteDefExtent(refLabel, source)
			if start >= 0 && end >= 0 {
				defs[node.Index] = footnoteDef{
					start:   start,
					end:     end,
					ref:     refLabel,
					content: htmlContent,
				}
			}
		}

		return ast.WalkContinue, nil
	})

	// Sort refs by position
	sort.Slice(refs, func(i, j int) bool {
		return refs[i].start < refs[j].start
	})

	// Assign sidenote numbers in order of appearance
	sidenoteNum := make(map[int]int) // goldmark index -> sidenote number
	nextNum := 1
	for _, ref := range refs {
		if _, exists := sidenoteNum[ref.index]; !exists {
			sidenoteNum[ref.index] = nextNum
			nextNum++
		}
	}

	// Build the list of definition ranges to exclude
	var defRanges []byteRange
	for _, def := range defs {
		defRanges = append(defRanges, byteRange{start: def.start, end: def.end})
	}
	sort.Slice(defRanges, func(i, j int) bool {
		return defRanges[i].start < defRanges[j].start
	})

	// Build output
	var result strings.Builder
	lastEnd := 0

	for _, ref := range refs {
		// Write content before this ref, excluding definition ranges
		before := excludeRanges(string(source[lastEnd:ref.start]), lastEnd, defRanges)
		result.WriteString(before)

		// Get the sidenote number and content
		num := sidenoteNum[ref.index]
		def, hasDef := defs[ref.index]

		if hasDef {
			// Write the sidenote HTML
			result.WriteString(fmt.Sprintf("\n<label for=\"sidenote-%d\" class=\"margin-toggle sidenote-number\"></label>\n", num))
			result.WriteString(fmt.Sprintf("<input type=\"checkbox\" id=\"sidenote-%d\" class=\"margin-toggle\"/>\n", num))
			result.WriteString("<span class=\"sidenote\">\n")
			result.WriteString(def.content)
			result.WriteString("\n</span>")
		} else {
			// No definition found, leave the reference as-is
			result.WriteString(string(source[ref.start:ref.end]))
		}

		lastEnd = ref.end
	}

	// Write remaining content, excluding definitions
	remaining := excludeRanges(string(source[lastEnd:]), lastEnd, defRanges)
	remaining = strings.TrimRight(remaining, "\n") + "\n\n"
	result.WriteString(remaining)

	return result.String()
}

// renderFootnoteContent renders the content of a footnote to HTML
func renderFootnoteContent(node *extast.Footnote, source []byte, md goldmark.Markdown) string {
	// Collect the text content from the footnote's children
	var content bytes.Buffer

	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		// Render each child node to HTML
		if para, ok := child.(*ast.Paragraph); ok {
			// For paragraphs, render the inline content
			renderInlineHTML(&content, para, source, md)
		}
	}

	return strings.TrimSpace(content.String())
}

// renderInlineHTML renders inline nodes to HTML
func renderInlineHTML(w *bytes.Buffer, node ast.Node, source []byte, md goldmark.Markdown) {
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		switch n := child.(type) {
		case *ast.Text:
			text := n.Segment.Value(source)
			w.Write(text)
			if n.HardLineBreak() {
				w.WriteString("<br/>")
			}
		case *ast.Emphasis:
			if n.Level == 1 {
				w.WriteString("<em>")
				renderInlineHTML(w, n, source, md)
				w.WriteString("</em>")
			} else {
				w.WriteString("<strong>")
				renderInlineHTML(w, n, source, md)
				w.WriteString("</strong>")
			}
		case *ast.CodeSpan:
			w.WriteString("<code>")
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				if t, ok := c.(*ast.Text); ok {
					w.Write(t.Segment.Value(source))
				}
			}
			w.WriteString("</code>")
		case *ast.Link:
			w.WriteString("<a href=\"")
			w.Write(n.Destination)
			w.WriteString("\">")
			renderInlineHTML(w, n, source, md)
			w.WriteString("</a>")
		case *ast.AutoLink:
			url := n.URL(source)
			w.WriteString("<a href=\"")
			w.Write(url)
			w.WriteString("\">")
			w.Write(url)
			w.WriteString("</a>")
		default:
			// For other nodes, try to get text content
			if child.ChildCount() > 0 {
				renderInlineHTML(w, child, source, md)
			}
		}
	}
}

// findFootnoteRefExtent finds the byte range of a footnote reference [^label]
// This searches for the Nth occurrence of a footnote reference pattern
func findFootnoteRefExtent(index int, source []byte) (int, int) {
	// We need to find footnote references in order
	// Search for [^ patterns and track which index we're at
	count := 0
	for i := 0; i < len(source)-2; i++ {
		if source[i] == '[' && source[i+1] == '^' {
			// Find the closing ]
			end := i + 2
			for end < len(source) && source[end] != ']' && source[end] != '\n' {
				end++
			}
			if end < len(source) && source[end] == ']' {
				// Check if this is a reference (not a definition - no colon after)
				afterClose := end + 1
				if afterClose >= len(source) || source[afterClose] != ':' {
					count++
					if count == index {
						return i, end + 1
					}
				}
			}
		}
	}

	return -1, -1
}

// findFootnoteDefExtent finds the byte range of a footnote definition
func findFootnoteDefExtent(label string, source []byte) (int, int) {
	pattern := []byte("[^" + label + "]:")

	idx := bytes.Index(source, pattern)
	if idx < 0 {
		return -1, -1
	}

	// Find the end of the definition - it continues until:
	// - A blank line followed by non-indented content, or
	// - Another footnote definition, or
	// - End of file
	start := idx
	end := idx

	// Skip to end of current line
	for end < len(source) && source[end] != '\n' {
		end++
	}
	if end < len(source) {
		end++ // include the newline
	}

	// Continue through continuation lines (indented or blank)
	for end < len(source) {
		lineStart := end
		lineEnd := end

		// Find end of line
		for lineEnd < len(source) && source[lineEnd] != '\n' {
			lineEnd++
		}

		line := source[lineStart:lineEnd]

		// Check if this is a continuation
		if len(line) == 0 {
			// Blank line - might be part of definition, continue
			end = lineEnd
			if end < len(source) {
				end++
			}
			continue
		}

		// Check for indentation (continuation)
		if len(line) > 0 && (line[0] == ' ' || line[0] == '\t') {
			end = lineEnd
			if end < len(source) {
				end++
			}
			continue
		}

		// Check for another footnote definition
		if bytes.HasPrefix(line, []byte("[^")) {
			break
		}

		// Non-indented, non-blank line - end of definition
		break
	}

	return start, end
}

type byteRange struct {
	start int
	end   int
}

func excludeRanges(content string, contentStart int, ranges []byteRange) string {
	contentEnd := contentStart + len(content)
	var result strings.Builder

	pos := 0
	for _, r := range ranges {
		relStart := r.start - contentStart
		relEnd := r.end - contentStart

		if r.end <= contentStart || r.start >= contentEnd {
			continue
		}

		if relStart < 0 {
			relStart = 0
		}
		if relEnd > len(content) {
			relEnd = len(content)
		}

		if relStart > pos {
			result.WriteString(content[pos:relStart])
		}
		pos = relEnd
	}

	if pos < len(content) {
		result.WriteString(content[pos:])
	}

	return result.String()
}
