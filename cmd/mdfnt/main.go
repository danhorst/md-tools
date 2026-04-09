// mdfnt renumbers footnote references ([^label]) to sequential integers
// in order of first appearance, and updates corresponding definitions.
//
// Usage:
//
//	mdfnt [file...]
//	cat file.md | mdfnt
//	mdfnt -w file.md    # modify file in place
package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/dbh/md-tools/internal/cli"
	"github.com/dbh/md-tools/internal/markdown"
	"github.com/yuin/goldmark"
	goldast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

var writeInPlace = flag.Bool("w", false, "write result to file instead of stdout")

func main() {
	flag.Parse()
	if err := cli.Run(flag.Args(), *writeInPlace, "mdfnt", transform); err != nil {
		fmt.Fprintf(os.Stderr, "mdfnt: %v\n", err)
		os.Exit(1)
	}
}

var (
	// footnoteAnyRe matches [^label] — both references and definitions.
	footnoteAnyRe = regexp.MustCompile(`\[\^([^\]]+)\]`)
	// footnoteDefLineRe matches a footnote definition at the start of a line.
	footnoteDefLineRe = regexp.MustCompile(`^\[\^([^\]]+)\]:`)
)

type replacement struct {
	start   int
	end     int
	newText string
}

func transform(content string) string {
	source := []byte(content)

	// Parse the document to find code-block and inline-code ranges to exclude.
	md := goldmark.New()
	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader)
	codeRanges := findCodeRanges(source, doc)

	// Scan for footnote references in body text (not definitions, not in code).
	labelToNum := make(map[string]int)
	nextNum := 1

	type refMatch struct {
		start, end int
		label      string
	}
	var bodyRefs []refMatch

	for _, m := range footnoteAnyRe.FindAllSubmatchIndex(source, -1) {
		start, end := m[0], m[1]
		label := string(source[m[2]:m[3]])

		// Skip definitions: [^label]: has ':' immediately after the closing ']'.
		if end < len(source) && source[end] == ':' {
			continue
		}

		// Skip occurrences inside code blocks or inline code.
		if inAnyRange(start, end, codeRanges) {
			continue
		}

		if _, seen := labelToNum[label]; !seen {
			labelToNum[label] = nextNum
			nextNum++
		}
		bodyRefs = append(bodyRefs, refMatch{start, end, label})
	}

	// Scan for footnote definitions. Assign numbers for any orphaned labels
	// (definitions with no corresponding reference in body text).
	type defMatch struct {
		start, end int // byte range of [^label] (the bracket portion, without ':')
		label      string
	}
	var defs []defMatch

	offset := 0
	for _, line := range strings.Split(content, "\n") {
		if m := footnoteDefLineRe.FindStringSubmatchIndex(line); m != nil {
			label := line[m[2]:m[3]]
			// m[0] = start of '[', m[3] = end of label, so ']' is at m[3].
			defs = append(defs, defMatch{
				start: offset + m[0],
				end:   offset + m[3] + 1, // exclusive, just past ']'
				label: label,
			})
			if _, seen := labelToNum[label]; !seen {
				labelToNum[label] = nextNum
				nextNum++
			}
		}
		offset += len(line) + 1 // +1 for the '\n' separator
	}

	// Build the replacement list.
	var replacements []replacement

	for _, ref := range bodyRefs {
		replacements = append(replacements, replacement{
			start:   ref.start,
			end:     ref.end,
			newText: fmt.Sprintf("[^%d]", labelToNum[ref.label]),
		})
	}
	for _, def := range defs {
		replacements = append(replacements, replacement{
			start:   def.start,
			end:     def.end,
			newText: fmt.Sprintf("[^%d]", labelToNum[def.label]),
		})
	}

	if len(replacements) == 0 {
		return content
	}

	sort.Slice(replacements, func(i, j int) bool {
		return replacements[i].start < replacements[j].start
	})

	// Apply replacements in a single pass over the source.
	var result strings.Builder
	pos := 0
	for _, r := range replacements {
		result.WriteString(content[pos:r.start])
		result.WriteString(r.newText)
		pos = r.end
	}
	result.WriteString(content[pos:])

	return result.String()
}

// findCodeRanges returns the byte ranges of fenced code blocks, indented code
// blocks, and inline code spans in the parsed document.
func findCodeRanges(source []byte, doc goldast.Node) []markdown.ByteRange {
	var ranges []markdown.ByteRange
	goldast.Walk(doc, func(n goldast.Node, entering bool) (goldast.WalkStatus, error) {
		if !entering {
			return goldast.WalkContinue, nil
		}
		switch node := n.(type) {
		case *goldast.FencedCodeBlock:
			lines := node.Lines()
			if lines.Len() > 0 {
				ranges = append(ranges, markdown.ByteRange{
					Start: lines.At(0).Start,
					End:   lines.At(lines.Len() - 1).Stop,
				})
			}
		case *goldast.CodeBlock:
			lines := node.Lines()
			if lines.Len() > 0 {
				ranges = append(ranges, markdown.ByteRange{
					Start: lines.At(0).Start,
					End:   lines.At(lines.Len() - 1).Stop,
				})
			}
		case *goldast.CodeSpan:
			for child := node.FirstChild(); child != nil; child = child.NextSibling() {
				if t, ok := child.(*goldast.Text); ok {
					ranges = append(ranges, markdown.ByteRange{
						Start: t.Segment.Start,
						End:   t.Segment.Stop,
					})
				}
			}
		}
		return goldast.WalkContinue, nil
	})
	return ranges
}

// inAnyRange reports whether [start, end) overlaps with any of the given ranges.
func inAnyRange(start, end int, ranges []markdown.ByteRange) bool {
	for _, r := range ranges {
		if start >= r.Start && end <= r.End {
			return true
		}
	}
	return false
}
