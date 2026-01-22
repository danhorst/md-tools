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
	"os"
	"sort"
	"strings"

	"github.com/dbh/md-tools/internal/cli"
	"github.com/dbh/md-tools/internal/markdown"
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
	if err := cli.Run(flag.Args(), *writeInPlace, "mdsidenote", transform); err != nil {
		fmt.Fprintf(os.Stderr, "mdsidenote: %v\n", err)
		os.Exit(1)
	}
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
	var defRanges []markdown.ByteRange
	for _, def := range defs {
		defRanges = append(defRanges, markdown.ByteRange{Start: def.start, End: def.end})
	}
	sort.Slice(defRanges, func(i, j int) bool {
		return defRanges[i].Start < defRanges[j].Start
	})

	// Build output
	var result strings.Builder
	lastEnd := 0

	for _, ref := range refs {
		// Write content before this ref, excluding definition ranges
		before := markdown.ExcludeRanges(string(source[lastEnd:ref.start]), lastEnd, defRanges)
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
	remaining := markdown.ExcludeRanges(string(source[lastEnd:]), lastEnd, defRanges)
	remaining = strings.TrimRight(remaining, "\n") + "\n\n"
	result.WriteString(remaining)

	return result.String()
}

// renderFootnoteContent renders the content of a footnote to HTML using goldmark
func renderFootnoteContent(node *extast.Footnote, source []byte, md goldmark.Markdown) string {
	var buf bytes.Buffer

	// Render each child paragraph's content
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if para, ok := child.(*ast.Paragraph); ok {
			// Get the source range covered by this paragraph
			if para.Lines().Len() > 0 {
				// Extract the markdown source for this paragraph
				var paraSource bytes.Buffer
				for i := 0; i < para.Lines().Len(); i++ {
					line := para.Lines().At(i)
					paraSource.Write(line.Value(source))
				}

				// Parse and render just this content
				md.Convert(paraSource.Bytes(), &buf)
			}
		}
	}

	// Strip the <p> tags that goldmark wraps around the content
	result := strings.TrimSpace(buf.String())
	result = strings.TrimPrefix(result, "<p>")
	result = strings.TrimSuffix(result, "</p>")

	return result
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
