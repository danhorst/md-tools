// mdref converts inline Markdown links to reference-style links.
//
// Usage:
//
//	mdref [file...]
//	cat file.md | mdref
//	mdref -w file.md    # modify file in place
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
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var flags = cli.RegisterFlags()

func main() {
	flag.Parse()
	if err := cli.Run("mdref", flags, flag.Args(), transform); err != nil {
		fmt.Fprintf(os.Stderr, "mdref: %v\n", err)
		os.Exit(1)
	}
}

// linkInfo represents a link found in the document with its position
type linkInfo struct {
	start int    // start position in content (byte offset)
	end   int    // end position in content (byte offset)
	text  string // link text
	url   string // destination URL
	title string // optional title
}

// reference holds URL and title for a reference definition
type reference struct {
	url   string
	title string
}

// transform converts inline links to reference-style links.
func transform(content string) string {
	source := []byte(content)

	// Parse the document with a context to capture reference definitions
	md := goldmark.New()
	ctx := parser.NewContext()
	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader, parser.WithContext(ctx))

	// Build a map of reference labels to their definitions
	refDefs := make(map[string]reference)
	for _, ref := range ctx.References() {
		label := strings.ToLower(string(ref.Label()))
		refDefs[label] = reference{
			url:   string(ref.Destination()),
			title: string(ref.Title()),
		}
	}

	// Find byte ranges of reference definitions in the source to exclude them
	refDefRanges := findRefDefRanges(source)

	// Convert to markdown.ByteRange for the shared utility
	excludeRanges := make([]markdown.ByteRange, len(refDefRanges))
	for i, r := range refDefRanges {
		excludeRanges[i] = markdown.ByteRange{Start: r.start, End: r.end}
	}

	// Collect all links from the AST
	var links []linkInfo

	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		link, ok := n.(*ast.Link)
		if !ok {
			return ast.WalkContinue, nil
		}

		// Find the extent of this link in the source (also extracts link text)
		start, end, linkText := findLinkExtent(link, source)
		if start < 0 || end < 0 {
			return ast.WalkContinue, nil
		}

		// Skip links that are inside reference definitions
		for _, r := range refDefRanges {
			if start >= r.start && end <= r.end {
				return ast.WalkContinue, nil
			}
		}

		links = append(links, linkInfo{
			start: start,
			end:   end,
			text:  linkText,
			url:   string(link.Destination),
			title: string(link.Title),
		})

		return ast.WalkContinue, nil
	})

	// Sort links by position in document
	sort.Slice(links, func(i, j int) bool {
		return links[i].start < links[j].start
	})

	// Build output excluding reference definitions
	urlToRef := make(map[string]int)
	var refs []reference
	var result strings.Builder
	lastEnd := 0

	for _, link := range links {
		// Write content before this link, but skip reference definition ranges
		result.WriteString(markdown.ExcludeRanges(string(source[lastEnd:link.start]), lastEnd, excludeRanges))

		// Create deduplication key
		refKey := link.url
		if link.title != "" {
			refKey = link.url + "\x00" + link.title
		}

		// Get or assign reference number
		refNum, exists := urlToRef[refKey]
		if !exists {
			refNum = len(refs) + 1
			urlToRef[refKey] = refNum
			refs = append(refs, reference{url: link.url, title: link.title})
		}

		// Write the reference-style link
		result.WriteString(fmt.Sprintf("[%s][%d]", link.text, refNum))

		lastEnd = link.end
	}

	// Write remaining content, excluding reference definitions
	remaining := string(source[lastEnd:])
	remaining = markdown.ExcludeRanges(remaining, lastEnd, excludeRanges)
	remaining = strings.TrimRight(remaining, "\n") + "\n"
	result.WriteString(remaining)

	// Append new reference definitions
	if len(refs) > 0 {
		result.WriteString("\n")
		for i, ref := range refs {
			if ref.title != "" {
				fmt.Fprintf(&result, "[%d]: %s %q\n", i+1, ref.url, ref.title)
			} else {
				fmt.Fprintf(&result, "[%d]: %s\n", i+1, ref.url)
			}
		}
	}

	return result.String()
}

// refDefRange represents a range of bytes for a reference definition in the source.
// This is internal to mdref; the shared markdown.ByteRange is used for exclusion.
type refDefRange struct {
	start int
	end   int
}

// findRefDefRanges finds the byte ranges of reference definitions in source.
// Reference definitions are lines like: [label]: url "title"
func findRefDefRanges(source []byte) []refDefRange {
	var ranges []refDefRange
	lines := bytes.Split(source, []byte("\n"))
	offset := 0

	for _, line := range lines {
		lineLen := len(line)
		trimmed := bytes.TrimSpace(line)

		// Check if line starts with [ and contains ]:
		if len(trimmed) > 0 && trimmed[0] == '[' {
			closeBracket := bytes.Index(trimmed, []byte("]:"))
			if closeBracket > 1 {
				label := trimmed[1:closeBracket]
				// Skip footnote definitions (start with ^)
				if len(label) > 0 && label[0] != '^' {
					// This is a reference definition - mark the whole line
					ranges = append(ranges, refDefRange{
						start: offset,
						end:   offset + lineLen + 1, // +1 for newline
					})
				}
			}
		}

		offset += lineLen + 1 // +1 for newline
	}

	return ranges
}

// nodeContentStart returns the byte offset of the first content character inside
// an inline node. It handles *ast.Text directly and recurses into any container
// node (Emphasis, Strong, CodeSpan, etc.) via its first child.
func nodeContentStart(n ast.Node) int {
	if t, ok := n.(*ast.Text); ok {
		return t.Segment.Start
	}
	if first := n.FirstChild(); first != nil {
		return nodeContentStart(first)
	}
	return -1
}

// findLinkExtent finds the start and end byte positions of a link node in the
// source and returns the raw link text (the bytes between [ and ]).
// This handles both plain-text and code-span link text (e.g. [`Foo`](url)).
func findLinkExtent(node *ast.Link, source []byte) (start, end int, linkText string) {
	start, end = -1, -1

	if node.ChildCount() == 0 {
		return
	}

	firstChild := node.FirstChild()
	if firstChild == nil {
		return
	}

	// Locate the first content byte inside the link text, then scan back to '['.
	contentStart := nodeContentStart(firstChild)
	if contentStart < 0 {
		return
	}
	pos := contentStart - 1
	for pos >= 0 && source[pos] != '[' && source[pos] != '\n' {
		pos--
	}
	if pos < 0 || source[pos] != '[' {
		return
	}
	start = pos

	// Scan forward from start+1 to find the matching ']', skipping over code
	// spans so that a backtick inside the link text doesn't confuse the scan.
	closeSquare := -1
	i := start + 1
	inCode := false
	for i < len(source) && source[i] != '\n' {
		if source[i] == '`' {
			inCode = !inCode
			i++
			continue
		}
		if !inCode && source[i] == ']' {
			closeSquare = i
			break
		}
		i++
	}
	if closeSquare < 0 {
		start = -1
		return
	}

	linkText = string(source[start+1 : closeSquare])

	// Determine whether this is an inline link ](url) or a reference link ][ref].
	i = closeSquare + 1
	if i >= len(source) {
		start = -1
		return
	}

	if source[i] == '(' {
		// Inline link — scan for the closing ')'.
		depth := 1
		i++
		for i < len(source) && depth > 0 {
			if source[i] == '(' {
				depth++
			} else if source[i] == ')' {
				depth--
			}
			i++
		}
		end = i
	} else if source[i] == '[' {
		// Reference link — scan for the closing ']'.
		depth := 1
		i++
		for i < len(source) && depth > 0 {
			if source[i] == '[' {
				depth++
			} else if source[i] == ']' {
				depth--
			}
			i++
		}
		end = i
	} else {
		// Collapsed/shortcut reference: [text] with no trailing (...) or [...].
		end = closeSquare + 1
	}

	return
}
