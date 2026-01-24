// mdinline converts reference-style Markdown links to inline links.
//
// Usage:
//
//	mdinline [file...]
//	cat file.md | mdinline
//	mdinline -w file.md    # modify file in place
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
	if err := cli.Run(flag.Args(), *writeInPlace, "mdinline", transform); err != nil {
		fmt.Fprintf(os.Stderr, "mdinline: %v\n", err)
		os.Exit(1)
	}
}

// linkInfo represents a reference-style link found in the document
type linkInfo struct {
	start int    // start position in content (byte offset)
	end   int    // end position in content (byte offset)
	text  string // link text
	url   string // resolved destination URL
	title string // optional title
}

// transform converts reference-style links to inline links.
func transform(content string) string {
	source := []byte(content)

	// Parse the document with a context to capture reference definitions
	md := goldmark.New()
	ctx := parser.NewContext()
	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader, parser.WithContext(ctx))

	// Build a map of reference labels to their definitions
	refDefs := make(map[string]struct {
		url   string
		title string
	})
	for _, ref := range ctx.References() {
		label := strings.ToLower(string(ref.Label()))
		refDefs[label] = struct {
			url   string
			title string
		}{
			url:   string(ref.Destination()),
			title: string(ref.Title()),
		}
	}

	// Find byte ranges of reference definitions to exclude them from output
	refDefRanges := findRefDefRanges(source)
	excludeRanges := make([]markdown.ByteRange, len(refDefRanges))
	for i, r := range refDefRanges {
		excludeRanges[i] = markdown.ByteRange{Start: r.start, End: r.end}
	}

	// Collect all reference-style links from the AST
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

		// Check if this is a reference-style link by examining source
		linkSource := string(source[start:end])
		if isInlineLink(linkSource) {
			// Already an inline link, skip it
			return ast.WalkContinue, nil
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

	// Build output
	var result strings.Builder
	lastEnd := 0

	for _, link := range links {
		// Write content before this link, excluding reference definition ranges
		result.WriteString(markdown.ExcludeRanges(string(source[lastEnd:link.start]), lastEnd, excludeRanges))

		// Write the inline-style link
		if link.title != "" {
			result.WriteString(fmt.Sprintf("[%s](%s %q)", link.text, link.url, link.title))
		} else {
			result.WriteString(fmt.Sprintf("[%s](%s)", link.text, link.url))
		}

		lastEnd = link.end
	}

	// Write remaining content, excluding reference definitions
	remaining := string(source[lastEnd:])
	remaining = markdown.ExcludeRanges(remaining, lastEnd, excludeRanges)
	remaining = strings.TrimRight(remaining, "\n") + "\n"
	result.WriteString(remaining)

	return result.String()
}

// isInlineLink checks if the link source is an inline link [text](url)
func isInlineLink(source string) bool {
	// Find the ] that closes the link text
	closeBracket := strings.Index(source, "]")
	if closeBracket < 0 || closeBracket+1 >= len(source) {
		return false
	}
	// Check if followed by (
	return source[closeBracket+1] == '('
}

// refDefRange represents a range of bytes for a reference definition
type refDefRange struct {
	start int
	end   int
}

// findRefDefRanges finds the byte ranges of reference definitions in source
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
					ranges = append(ranges, refDefRange{
						start: offset,
						end:   offset + lineLen + 1,
					})
				}
			}
		}

		offset += lineLen + 1
	}

	return ranges
}

// findLinkExtent finds the start and end byte positions of a link node
func findLinkExtent(node *ast.Link, source []byte) (int, int) {
	if node.ChildCount() == 0 {
		return -1, -1
	}

	firstChild := node.FirstChild()
	if firstChild == nil {
		return -1, -1
	}

	textNode, ok := firstChild.(*ast.Text)
	if !ok {
		return -1, -1
	}

	start := textNode.Segment.Start - 1
	if start < 0 || source[start] != '[' {
		return -1, -1
	}

	lastChild := node.LastChild()
	lastText, ok := lastChild.(*ast.Text)
	if !ok {
		return -1, -1
	}
	textEnd := lastText.Segment.Stop

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
			end++
			break
		} else if ch == '\n' {
			break
		}
		end++
	}

	return start, end
}
