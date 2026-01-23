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
	"regexp"
	"sort"
	"strconv"
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
	start      int    // byte position of [^label]: ...
	end        int    // byte position after the definition
	ref        string // the footnote reference label
	content    string // the rendered HTML content
	rawContent string // the raw markdown content (for reference tracking)
}

// linkDef represents a reference-style link definition
type linkDef struct {
	label string
	url   string
	start int // byte position in source
	end   int // byte position after definition
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

	// Collect link reference definitions
	linkDefs := collectLinkDefs(source)

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
			// Get the footnote content
			refLabel := string(node.Ref)
			rawContent := extractFootnoteRawContent(node, source)

			// Find the definition extent in source
			start, end := findFootnoteDefExtent(refLabel, source)
			if start >= 0 && end >= 0 {
				defs[node.Index] = footnoteDef{
					start:      start,
					end:        end,
					ref:        refLabel,
					rawContent: rawContent,
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

	// Track which link references are used in footnotes vs body
	footnoteRefs := make(map[string]bool)
	for _, def := range defs {
		for label := range findRefLinksInText(def.rawContent) {
			footnoteRefs[label] = true
		}
	}

	// Determine which references are used in body (non-footnote) text
	bodyRefs := findBodyRefLinks(source, defs)

	// References to keep: used in body text
	// References to remove: only used in footnotes
	refsToRemove := make(map[string]bool)
	for label := range footnoteRefs {
		if !bodyRefs[label] {
			refsToRemove[label] = true
		}
	}

	// Render footnote content with reference links resolved
	for idx, def := range defs {
		htmlContent := renderFootnoteContentWithRefs(def.rawContent, linkDefs, md)
		def.content = htmlContent
		defs[idx] = def
	}

	// Build the list of definition ranges to exclude (footnote defs)
	var defRanges []markdown.ByteRange
	for _, def := range defs {
		defRanges = append(defRanges, markdown.ByteRange{Start: def.start, End: def.end})
	}
	sort.Slice(defRanges, func(i, j int) bool {
		return defRanges[i].Start < defRanges[j].Start
	})

	// Build the list of link definition ranges to exclude
	var linkDefRanges []markdown.ByteRange
	for _, ld := range linkDefs {
		if refsToRemove[ld.label] {
			linkDefRanges = append(linkDefRanges, markdown.ByteRange{Start: ld.start, End: ld.end})
		}
	}
	sort.Slice(linkDefRanges, func(i, j int) bool {
		return linkDefRanges[i].Start < linkDefRanges[j].Start
	})

	// Combine all ranges to exclude
	allExcludeRanges := append(defRanges, linkDefRanges...)
	sort.Slice(allExcludeRanges, func(i, j int) bool {
		return allExcludeRanges[i].Start < allExcludeRanges[j].Start
	})

	// Build output
	var result strings.Builder
	lastEnd := 0

	for _, ref := range refs {
		// Write content before this ref, excluding definition ranges
		before := markdown.ExcludeRanges(string(source[lastEnd:ref.start]), lastEnd, allExcludeRanges)
		result.WriteString(before)

		// Get the sidenote number and content
		num := sidenoteNum[ref.index]
		def, hasDef := defs[ref.index]

		if hasDef {
			// Write the sidenote HTML
			result.WriteString(fmt.Sprintf("\n<label for=\"sidenote-%d\" class=\"margin-toggle sidenote-number\"></label>\n", num))
			result.WriteString(fmt.Sprintf("<input type=\"checkbox\" id=\"sidenote-%d\" class=\"margin-toggle\"/>\n", num))
			result.WriteString("<span class=\"sidenote\">")
			result.WriteString("<span class=\"hidden\">(</span>")
			result.WriteString(def.content)
			result.WriteString("<span class=\"hidden\">)</span>")
			result.WriteString("</span>")
		} else {
			// No definition found, leave the reference as-is
			result.WriteString(string(source[ref.start:ref.end]))
		}

		lastEnd = ref.end
	}

	// Write remaining content, excluding definitions
	remaining := markdown.ExcludeRanges(string(source[lastEnd:]), lastEnd, allExcludeRanges)

	// Renumber remaining link references
	remaining = renumberLinkRefs(remaining, linkDefs, refsToRemove)

	remaining = strings.TrimRight(remaining, "\n") + "\n"
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

// collectLinkDefs finds all reference-style link definitions in the source
func collectLinkDefs(source []byte) []linkDef {
	var defs []linkDef
	// Match [label]: url patterns
	re := regexp.MustCompile(`(?m)^\[([^\]]+)\]:\s*(\S+).*$`)
	matches := re.FindAllSubmatchIndex(source, -1)

	for _, match := range matches {
		// Skip footnote definitions [^label]:
		label := string(source[match[2]:match[3]])
		if strings.HasPrefix(label, "^") {
			continue
		}

		defs = append(defs, linkDef{
			label: label,
			url:   string(source[match[4]:match[5]]),
			start: match[0],
			end:   match[1] + 1, // include newline
		})
	}

	return defs
}

// extractFootnoteRawContent extracts the raw markdown content from a footnote
func extractFootnoteRawContent(node *extast.Footnote, source []byte) string {
	var buf bytes.Buffer

	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if para, ok := child.(*ast.Paragraph); ok {
			if para.Lines().Len() > 0 {
				for i := 0; i < para.Lines().Len(); i++ {
					line := para.Lines().At(i)
					buf.Write(line.Value(source))
				}
			}
		}
	}

	return strings.TrimSpace(buf.String())
}

// findRefLinksInText finds all reference-style link labels used in text
func findRefLinksInText(text string) map[string]bool {
	refs := make(map[string]bool)
	// Match [text][label] or [label][] or [label] patterns
	re := regexp.MustCompile(`\[([^\]]+)\]\[([^\]]*)\]|\[([^\]]+)\](?:\[([^\]]*)\])?`)
	matches := re.FindAllStringSubmatch(text, -1)

	for _, match := range matches {
		if match[2] != "" {
			// [text][label] form
			refs[match[2]] = true
		} else if match[4] != "" {
			// [text][label] form (alternate capture)
			refs[match[4]] = true
		} else if match[1] != "" {
			// Could be [label][] or just [label] used as reference
			refs[match[1]] = true
		} else if match[3] != "" {
			refs[match[3]] = true
		}
	}

	return refs
}

// findBodyRefLinks finds reference links used in the body (non-footnote) text
func findBodyRefLinks(source []byte, footnoteDefs map[int]footnoteDef) map[string]bool {
	bodyRefs := make(map[string]bool)

	// Build ranges to exclude (footnote definitions and link definitions)
	var excludeRanges []markdown.ByteRange
	for _, def := range footnoteDefs {
		excludeRanges = append(excludeRanges, markdown.ByteRange{Start: def.start, End: def.end})
	}

	// Also exclude link definition lines
	linkDefRe := regexp.MustCompile(`(?m)^\[[^\]]+\]:\s*\S+.*$`)
	linkDefMatches := linkDefRe.FindAllIndex(source, -1)
	for _, match := range linkDefMatches {
		excludeRanges = append(excludeRanges, markdown.ByteRange{Start: match[0], End: match[1]})
	}

	sort.Slice(excludeRanges, func(i, j int) bool {
		return excludeRanges[i].Start < excludeRanges[j].Start
	})

	// Get body text by excluding footnote definitions
	bodyText := markdown.ExcludeRanges(string(source), 0, excludeRanges)

	// Find all reference links in body
	for label := range findRefLinksInText(bodyText) {
		bodyRefs[label] = true
	}

	return bodyRefs
}

// renderFootnoteContentWithRefs renders footnote content with reference links resolved
func renderFootnoteContentWithRefs(rawContent string, linkDefs []linkDef, md goldmark.Markdown) string {
	// Build a map of label -> url
	linkMap := make(map[string]string)
	for _, ld := range linkDefs {
		linkMap[ld.label] = ld.url
	}

	// Replace reference links with inline links
	// Handle [text][label] form
	re1 := regexp.MustCompile(`\[([^\]]+)\]\[([^\]]+)\]`)
	content := re1.ReplaceAllStringFunc(rawContent, func(match string) string {
		parts := re1.FindStringSubmatch(match)
		if len(parts) == 3 {
			text := parts[1]
			label := parts[2]
			if url, ok := linkMap[label]; ok {
				return "[" + text + "](" + url + ")"
			}
		}
		return match
	})

	// Handle [label][] form (empty second bracket)
	re2 := regexp.MustCompile(`\[([^\]]+)\]\[\]`)
	content = re2.ReplaceAllStringFunc(content, func(match string) string {
		parts := re2.FindStringSubmatch(match)
		if len(parts) == 2 {
			label := parts[1]
			if url, ok := linkMap[label]; ok {
				return "[" + label + "](" + url + ")"
			}
		}
		return match
	})

	// Now render through goldmark
	var buf bytes.Buffer
	md.Convert([]byte(content), &buf)

	// Strip the <p> tags
	result := strings.TrimSpace(buf.String())
	result = strings.TrimPrefix(result, "<p>")
	result = strings.TrimSuffix(result, "</p>")

	return result
}

// renumberLinkRefs renumbers link references after removing some
func renumberLinkRefs(text string, linkDefs []linkDef, removed map[string]bool) string {
	// Build old -> new label mapping for numeric labels
	var keptLabels []string
	for _, ld := range linkDefs {
		if !removed[ld.label] {
			keptLabels = append(keptLabels, ld.label)
		}
	}

	// Create renumbering map (only for numeric labels)
	renumber := make(map[string]string)
	newNum := 1
	for _, label := range keptLabels {
		if _, err := strconv.Atoi(label); err == nil {
			renumber[label] = strconv.Itoa(newNum)
			newNum++
		}
	}

	// Replace in text - both usages [text][N] and definitions [N]:
	result := text

	// Replace reference usages [text][N]
	for old, new := range renumber {
		if old != new {
			re := regexp.MustCompile(`\](\[` + regexp.QuoteMeta(old) + `\])`)
			result = re.ReplaceAllString(result, "]["+new+"]")
		}
	}

	// Replace definitions [N]:
	for old, new := range renumber {
		if old != new {
			re := regexp.MustCompile(`(?m)^\[` + regexp.QuoteMeta(old) + `\]:`)
			result = re.ReplaceAllString(result, "["+new+"]:")
		}
	}

	return result
}
