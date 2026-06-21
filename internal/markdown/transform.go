package markdown

import "strings"

// Handlers defines the tool-specific behavior for paragraph and blockquote
// processing in a Markdown document transformation.
type Handlers struct {
	// Paragraph is called with the collected lines of a regular paragraph and
	// returns the transformed lines.
	Paragraph func(lines []string) []string
	// Blockquote is called with consecutive blockquote lines ("> " prefix
	// intact) and returns the transformed lines.
	Blockquote func(lines []string) []string
	// Footnote is called with a footnote definition's lines (the "[^label]:"
	// line plus any continuation lines) and returns the transformed lines.
	// When nil, footnote definitions are emitted verbatim.
	Footnote func(lines []string) []string
}

// Transform applies a Markdown-aware transformation to content, routing each
// block-level construct to the appropriate handler or emitting it unchanged.
// Frontmatter, code blocks, headers, list items (with continuations), table
// rows, and horizontal rules are passed through; paragraphs and blockquotes
// are delegated to h.
func Transform(content string, h Handlers) string {
	lines := strings.Split(content, "\n")
	var result []string
	i := 0

	// Handle YAML frontmatter (two formats: ---/--- or property-line/---)
	if i < len(lines) {
		hasFrontmatter := false
		if strings.TrimSpace(lines[i]) == "---" {
			if i+1 < len(lines) && LooksLikeFrontmatterProperty(lines[i+1]) {
				hasFrontmatter = true
				result = append(result, lines[i])
				i++
			}
		} else if LooksLikeFrontmatterProperty(lines[i]) {
			for j := i + 1; j < len(lines); j++ {
				if strings.TrimSpace(lines[j]) == "---" {
					hasFrontmatter = true
					break
				}
				if strings.TrimSpace(lines[j]) == "" {
					break
				}
			}
		}
		if hasFrontmatter {
			for i < len(lines) && strings.TrimSpace(lines[i]) != "---" {
				result = append(result, lines[i])
				i++
			}
			if i < len(lines) {
				result = append(result, lines[i])
				i++
			}
		}
	}

	for i < len(lines) {
		line := lines[i]

		// Fenced code block
		if strings.HasPrefix(strings.TrimSpace(line), "```") || strings.HasPrefix(strings.TrimSpace(line), "~~~") {
			fence := strings.TrimSpace(line)[:3]
			result = append(result, line)
			i++
			for i < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[i]), fence) {
				result = append(result, lines[i])
				i++
			}
			if i < len(lines) {
				result = append(result, lines[i])
				i++
			}
			continue
		}

		// Indented code block (4 spaces or tab)
		if strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t") {
			result = append(result, line)
			i++
			continue
		}

		// Footnote definition and its continuation lines. Without a Footnote
		// handler the block is emitted verbatim, so a multi-sentence footnote
		// stays on one line and renders portably across Markdown engines.
		if IsFootnoteDefinition(line) {
			fnLines := []string{line}
			i++
			for i < len(lines) && IsFootnoteContinuation(lines[i]) {
				fnLines = append(fnLines, lines[i])
				i++
			}
			if h.Footnote != nil {
				result = append(result, h.Footnote(fnLines)...)
			} else {
				result = append(result, fnLines...)
			}
			continue
		}

		// Link reference definition
		if IsLinkRefDefinition(line) {
			result = append(result, line)
			i++
			continue
		}

		// Blank line
		if strings.TrimSpace(line) == "" {
			result = append(result, line)
			i++
			continue
		}

		// Header
		if strings.HasPrefix(line, "#") {
			result = append(result, line)
			i++
			continue
		}

		// List item and continuation lines (indented 1-3 spaces)
		if IsListItem(line) {
			result = append(result, line)
			i++
			for i < len(lines) {
				l := lines[i]
				if strings.TrimSpace(l) == "" {
					break
				}
				if !strings.HasPrefix(l, " ") || strings.HasPrefix(l, "    ") {
					break
				}
				result = append(result, l)
				i++
			}
			continue
		}

		// Blockquote
		if strings.HasPrefix(strings.TrimSpace(line), ">") {
			var bqLines []string
			for i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), ">") {
				bqLines = append(bqLines, lines[i])
				i++
			}
			result = append(result, h.Blockquote(bqLines)...)
			continue
		}

		// Horizontal rule
		if IsHorizontalRule(line) {
			result = append(result, line)
			i++
			continue
		}

		// Table row
		if IsTableRow(line) {
			for i < len(lines) && IsTableRow(lines[i]) {
				result = append(result, lines[i])
				i++
			}
			continue
		}

		// Regular paragraph — collect until a block boundary
		var paraLines []string
		for i < len(lines) {
			l := lines[i]
			if strings.TrimSpace(l) == "" {
				break
			}
			if strings.HasPrefix(strings.TrimSpace(l), "```") ||
				strings.HasPrefix(strings.TrimSpace(l), "~~~") ||
				strings.HasPrefix(l, "    ") ||
				strings.HasPrefix(l, "\t") ||
				IsFootnoteDefinition(l) ||
				IsLinkRefDefinition(l) ||
				strings.HasPrefix(l, "#") ||
				IsListItem(l) ||
				strings.HasPrefix(strings.TrimSpace(l), ">") ||
				IsHorizontalRule(l) ||
				IsTableRow(l) {
				break
			}
			// Explicit line break (two trailing spaces) ends the paragraph
			if strings.HasSuffix(l, "  ") {
				paraLines = append(paraLines, l)
				i++
				break
			}
			paraLines = append(paraLines, l)
			i++
		}
		if len(paraLines) > 0 {
			result = append(result, h.Paragraph(paraLines)...)
		}
	}

	output := strings.Join(result, "\n")
	output = strings.TrimRight(output, "\n") + "\n"
	return output
}

// TransformBlockquote applies a blockquote-aware transformation to consecutive
// blockquote lines. The flush function receives accumulated content lines with
// the "> " prefix stripped, and must return the transformed output lines with
// the prefix added back. GFM alert headers and table rows are emitted as-is
// without passing through flush.
func TransformBlockquote(lines []string, flush func([]string) []string) []string {
	if len(lines) == 0 {
		return nil
	}
	const prefix = "> "
	var result []string
	var contentLines []string

	flushPending := func() {
		if len(contentLines) > 0 {
			result = append(result, flush(contentLines)...)
			contentLines = nil
		}
	}

	for _, line := range lines {
		content := strings.TrimPrefix(line, ">")
		content = strings.TrimPrefix(content, " ")

		// GFM alert header (e.g. [!NOTE]) — flush pending, emit as-is
		if strings.HasPrefix(content, "[!") && strings.Contains(content, "]") {
			flushPending()
			result = append(result, prefix+content)
			continue
		}

		// Table row — flush pending, emit as-is
		if IsTableRow(content) {
			flushPending()
			result = append(result, prefix+content)
			continue
		}

		contentLines = append(contentLines, content)
	}

	flushPending()
	return result
}
