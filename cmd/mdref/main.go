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

var writeInPlace = flag.Bool("w", false, "write result to file instead of stdout")

func main() {
	flag.Parse()
	if err := cli.Run(flag.Args(), *writeInPlace, "mdref", transform); err != nil {
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

		// Get link text from children
		var textBuf bytes.Buffer
		for child := link.FirstChild(); child != nil; child = child.NextSibling() {
			if textNode, ok := child.(*ast.Text); ok {
				textBuf.Write(textNode.Segment.Value(source))
			}
		}
		linkText := textBuf.String()

		// Find the extent of this link in the source
		start, end := findLinkExtent(link, source)
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

// findLinkExtent finds the start and end byte positions of a link node in the source
func findLinkExtent(node *ast.Link, source []byte) (int, int) {
	if node.ChildCount() == 0 {
		return -1, -1
	}

	// Get the first text child's segment to find where the link text starts
	firstChild := node.FirstChild()
	if firstChild == nil {
		return -1, -1
	}

	textNode, ok := firstChild.(*ast.Text)
	if !ok {
		return -1, -1
	}

	// The '[' should be just before the text segment
	start := textNode.Segment.Start - 1
	if start < 0 || source[start] != '[' {
		return -1, -1
	}

	// Find the last text child to get the end of link text
	lastChild := node.LastChild()
	lastText, ok := lastChild.(*ast.Text)
	if !ok {
		return -1, -1
	}
	textEnd := lastText.Segment.Stop

	// Scan forward to find the end of the link: ) for inline, ] for reference
	end := textEnd
	depth := 0
	for end < len(source) {
		ch := source[end]
		if ch == '(' {
			depth++
		} else if ch == ')' {
			if depth > 0 {
				depth--
			}
			if depth == 0 {
				end++
				break
			}
		} else if ch == ']' && end > textEnd {
			// End of reference-style link
			end++
			break
		} else if ch == '\n' {
			// Don't go past end of line
			break
		}
		end++
	}

	return start, end
}
