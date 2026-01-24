// mdjoin joins Markdown sentences into single-line paragraphs.
//
// Usage:
//
//	mdjoin [file...]
//	cat file.md | mdjoin
//	mdjoin -w file.md    # modify file in place
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/dbh/md-tools/internal/cli"
	"github.com/dbh/md-tools/internal/markdown"
)

var writeInPlace = flag.Bool("w", false, "write result to file instead of stdout")

func main() {
	flag.Parse()
	if err := cli.Run(flag.Args(), *writeInPlace, "mdjoin", transform); err != nil {
		fmt.Fprintf(os.Stderr, "mdjoin: %v\n", err)
		os.Exit(1)
	}
}

func transform(content string) string {
	lines := strings.Split(content, "\n")

	var result []string
	i := 0

	// Handle YAML frontmatter
	// Two formats:
	// 1. Starts with --- and ends with ---
	// 2. Starts with a property line and ends with ---
	if i < len(lines) {
		hasFrontmatter := false
		if strings.TrimSpace(lines[i]) == "---" {
			// Check if next line looks like a property
			if i+1 < len(lines) && markdown.LooksLikeFrontmatterProperty(lines[i+1]) {
				hasFrontmatter = true
				result = append(result, lines[i])
				i++
			}
		} else if markdown.LooksLikeFrontmatterProperty(lines[i]) {
			// Check if there's a closing --- somewhere
			for j := i + 1; j < len(lines); j++ {
				if strings.TrimSpace(lines[j]) == "---" {
					hasFrontmatter = true
					break
				}
				if strings.TrimSpace(lines[j]) == "" {
					// Blank line before --- means no frontmatter
					break
				}
			}
		}

		if hasFrontmatter {
			// Copy until closing ---
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

	// Process the rest of the document
	for i < len(lines) {
		line := lines[i]

		// Check for code block
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

		// Check for indented code block (4 spaces or tab)
		if strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t") {
			result = append(result, line)
			i++
			continue
		}

		// Check for footnote definition
		if markdown.IsFootnoteDefinition(line) {
			result = append(result, line)
			i++
			continue
		}

		// Check for link reference definition
		if markdown.IsLinkRefDefinition(line) {
			result = append(result, line)
			i++
			continue
		}

		// Check for blank line
		if strings.TrimSpace(line) == "" {
			result = append(result, line)
			i++
			continue
		}

		// Check for header
		if strings.HasPrefix(line, "#") {
			result = append(result, line)
			i++
			continue
		}

		// Check for list item (preserve as-is for now)
		if markdown.IsListItem(line) {
			result = append(result, line)
			i++
			continue
		}

		// Check for blockquote
		if strings.HasPrefix(strings.TrimSpace(line), ">") {
			// Collect all consecutive blockquote lines
			var bqLines []string
			for i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), ">") {
				bqLines = append(bqLines, lines[i])
				i++
			}
			unwrapped := unwrapBlockquote(bqLines)
			result = append(result, unwrapped...)
			continue
		}

		// Check for horizontal rule
		if markdown.IsHorizontalRule(line) {
			result = append(result, line)
			i++
			continue
		}

		// Regular paragraph - collect all lines until a break
		var paraLines []string
		for i < len(lines) {
			l := lines[i]

			// Stop at blank line
			if strings.TrimSpace(l) == "" {
				break
			}

			// Stop at special constructs
			if strings.HasPrefix(strings.TrimSpace(l), "```") ||
				strings.HasPrefix(strings.TrimSpace(l), "~~~") ||
				strings.HasPrefix(l, "    ") ||
				strings.HasPrefix(l, "\t") ||
				markdown.IsFootnoteDefinition(l) ||
				markdown.IsLinkRefDefinition(l) ||
				strings.HasPrefix(l, "#") ||
				markdown.IsListItem(l) ||
				strings.HasPrefix(strings.TrimSpace(l), ">") ||
				markdown.IsHorizontalRule(l) {
				break
			}

			// Check for explicit line break (two trailing spaces)
			if strings.HasSuffix(l, "  ") {
				paraLines = append(paraLines, l)
				i++
				break // End paragraph here, preserving the hard break
			}

			paraLines = append(paraLines, l)
			i++
		}

		if len(paraLines) > 0 {
			unwrapped := unwrapParagraph(paraLines)
			result = append(result, unwrapped...)
		}
	}

	// Ensure single trailing newline
	output := strings.Join(result, "\n")
	output = strings.TrimRight(output, "\n") + "\n"

	return output
}

// unwrapParagraph joins lines into a single line.
func unwrapParagraph(lines []string) []string {
	// Check if last line has explicit line break (two trailing spaces)
	hasHardBreak := len(lines) > 0 && strings.HasSuffix(lines[len(lines)-1], "  ")

	// Join all lines into one
	text := strings.Join(lines, " ")
	// Normalize multiple spaces
	text = strings.Join(strings.Fields(text), " ")

	if hasHardBreak {
		text += "  "
	}

	return []string{text}
}

// unwrapBlockquote unwraps blockquote lines into single lines per paragraph.
func unwrapBlockquote(lines []string) []string {
	if len(lines) == 0 {
		return nil
	}

	const prefix = "> "

	// Strip prefix and collect content, grouping by GFM alert headers
	var result []string
	var contentLines []string

	for _, line := range lines {
		// Strip the > and optional space
		content := strings.TrimPrefix(line, ">")
		content = strings.TrimPrefix(content, " ")

		// Check for GFM alert header like [!NOTE]
		if strings.HasPrefix(content, "[!") && strings.Contains(content, "]") {
			// Flush any pending content first
			if len(contentLines) > 0 {
				joined := joinLines(contentLines)
				result = append(result, prefix+joined)
				contentLines = nil
			}
			// Add alert header as-is
			result = append(result, prefix+content)
			continue
		}

		contentLines = append(contentLines, content)
	}

	// Flush remaining content
	if len(contentLines) > 0 {
		joined := joinLines(contentLines)
		result = append(result, prefix+joined)
	}

	return result
}

// joinLines joins lines into a single line.
func joinLines(lines []string) string {
	text := strings.Join(lines, " ")
	return strings.Join(strings.Fields(text), " ")
}
