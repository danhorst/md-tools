package markdown

import (
	"fmt"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// ReferenceRanges returns only definitions recognized by the Markdown parser.
// Code examples and other literal text are never candidates for removal.
func ReferenceRanges(source []byte, doc ast.Node) []ByteRange {
	var ranges []ByteRange
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		ref, ok := n.(*ast.LinkReferenceDefinition)
		if !ok || strings.HasPrefix(string(ref.Label), "^") || ref.Lines().Len() == 0 {
			return ast.WalkContinue, nil
		}
		lines := ref.Lines()
		start, end := lines.At(0).Start, lines.At(lines.Len()-1).Stop
		for end < len(source) && source[end] != '\n' {
			end++
		}
		if end < len(source) {
			end++
		}
		ranges = append(ranges, ByteRange{start, end})
		return ast.WalkContinue, nil
	})
	sort.Slice(ranges, func(i, j int) bool { return ranges[i].Start < ranges[j].Start })
	return ranges
}

// RewriteLinks preserves source labels while converting resolved links. Images
// stay inline in both modes, including images originally written as references.
func RewriteLinks(content string, references bool) string {
	source := []byte(content)
	doc := goldmark.New().Parser().Parse(text.NewReader(source))
	ranges := ReferenceRanges(source, doc)
	type edit struct {
		start, end  int
		replacement string
	}
	type ref struct{ url, title string }
	var edits []edit
	var refs []ref
	numbers := make(map[ref]int)
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		var destination, title []byte
		image := false
		switch node := n.(type) {
		case *ast.Link:
			destination, title = node.Destination, node.Title
		case *ast.Image:
			destination, title, image = node.Destination, node.Title, true
		default:
			return ast.WalkContinue, nil
		}
		start, end, label := findLinkExtent(n, source)
		if start < 0 || end < 0 {
			return ast.WalkContinue, nil
		}
		value := ref{string(destination), string(title)}
		var replacement string
		if references && !image {
			number, exists := numbers[value]
			if !exists {
				refs = append(refs, value)
				number = len(refs)
				numbers[value] = number
			}
			replacement = fmt.Sprintf("[%s][%d]", label, number)
		} else {
			// Already inline: leave its escaping and title delimiters untouched.
			if strings.HasPrefix(string(source[start+len(label)+2:end]), "(") {
				return ast.WalkContinue, nil
			}
			target := value.url
			if strings.ContainsAny(target, " ()") {
				target = "<" + target + ">"
			}
			if value.title != "" {
				target += fmt.Sprintf(" %q", value.title)
			}
			replacement = "[" + label + "](" + target + ")"
		}
		edits = append(edits, edit{start, end, replacement})
		return ast.WalkSkipChildren, nil
	})
	for _, r := range ranges {
		edits = append(edits, edit{r.Start, r.End, ""})
	}
	sort.SliceStable(edits, func(i, j int) bool { return edits[i].start < edits[j].start })
	var out strings.Builder
	pos := 0
	for _, e := range edits {
		if e.start < pos {
			continue
		}
		out.WriteString(content[pos:e.start])
		out.WriteString(e.replacement)
		pos = e.end
	}
	out.WriteString(content[pos:])
	result := strings.TrimRight(out.String(), "\n") + "\n"
	out.Reset()
	out.WriteString(result)
	if len(refs) > 0 {
		out.WriteString("\n")
		for i, r := range refs {
			target := r.url
			if strings.ContainsAny(target, " ()") {
				target = "<" + target + ">"
			}
			if r.title != "" {
				target += fmt.Sprintf(" %q", r.title)
			}
			fmt.Fprintf(&out, "[%d]: %s\n", i+1, target)
		}
	}
	return out.String()
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
func findLinkExtent(node ast.Node, source []byte) (start, end int, linkText string) {
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
	depthSquare := 1
	i := start + 1
	inCode := false
	for i < len(source) && source[i] != '\n' {
		if source[i] == '\\' && i+1 < len(source) {
			i += 2
			continue
		}
		if source[i] == '`' {
			inCode = !inCode
			i++
			continue
		}
		if !inCode && source[i] == '[' {
			depthSquare++
		}
		if !inCode && source[i] == ']' {
			depthSquare--
			if depthSquare == 0 {
				closeSquare = i
				break
			}
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
		end = i
		return
	}

	if source[i] == '(' {
		// Inline link — scan for the closing ')'.
		depth := 1
		i++
		for i < len(source) && depth > 0 {
			if source[i] == '\\' && i+1 < len(source) {
				i += 2
				continue
			}
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
