// mdwrap wraps Markdown paragraphs to a specified width (default 60).
//
// Usage:
//
//	mdwrap [file...]
//	cat file.md | mdwrap
//	mdwrap -c 80 file.md  # wrap to 80 columns
//	mdwrap -w file.md     # modify file in place
package main

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/dbh/md-tools/internal/cli"
)

var (
	writeInPlace = flag.Bool("w", false, "write result to file instead of stdout")
	wrapWidth    = flag.Int("c", 60, "column width to wrap to")
)

func main() {
	flag.Parse()
	if err := cli.Run(flag.Args(), *writeInPlace, "mdwrap", transform); err != nil {
		fmt.Fprintf(os.Stderr, "mdwrap: %v\n", err)
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
			if i+1 < len(lines) && looksLikeFrontmatterProperty(lines[i+1]) {
				hasFrontmatter = true
				result = append(result, lines[i])
				i++
			}
		} else if looksLikeFrontmatterProperty(lines[i]) {
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
		if isFootnoteDefinition(line) {
			result = append(result, line)
			i++
			continue
		}

		// Check for link reference definition
		if isLinkRefDefinition(line) {
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
		if isListItem(line) {
			result = append(result, line)
			i++
			continue
		}

		// Check for blockquote (preserve as-is for now, will handle in future fixture)
		if strings.HasPrefix(strings.TrimSpace(line), ">") {
			result = append(result, line)
			i++
			continue
		}

		// Check for horizontal rule
		if isHorizontalRule(line) {
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
				isFootnoteDefinition(l) ||
				isLinkRefDefinition(l) ||
				strings.HasPrefix(l, "#") ||
				isListItem(l) ||
				strings.HasPrefix(strings.TrimSpace(l), ">") ||
				isHorizontalRule(l) {
				break
			}

			paraLines = append(paraLines, l)
			i++
		}

		if len(paraLines) > 0 {
			wrapped := wrapParagraph(paraLines)
			result = append(result, wrapped...)
		}
	}

	// Ensure single trailing newline
	output := strings.Join(result, "\n")
	output = strings.TrimRight(output, "\n") + "\n"

	return output
}

func looksLikeFrontmatterProperty(line string) bool {
	// Simple heuristic: contains a colon not at the start
	trimmed := strings.TrimSpace(line)
	if trimmed == "" || trimmed == "---" {
		return false
	}
	idx := strings.Index(trimmed, ":")
	return idx > 0
}

func isFootnoteDefinition(line string) bool {
	matched, _ := regexp.MatchString(`^\[\^[^\]]+\]:`, line)
	return matched
}

func isLinkRefDefinition(line string) bool {
	// [label]: URL
	// Must not be a footnote definition
	if isFootnoteDefinition(line) {
		return false
	}
	matched, _ := regexp.MatchString(`^\[[^\]]+\]:\s*\S`, line)
	return matched
}

func isListItem(line string) bool {
	trimmed := strings.TrimSpace(line)
	// Unordered: -, *, +
	if len(trimmed) > 1 && (trimmed[0] == '-' || trimmed[0] == '*' || trimmed[0] == '+') && trimmed[1] == ' ' {
		return true
	}
	// Ordered: 1. 2. etc
	matched, _ := regexp.MatchString(`^\d+\.\s`, trimmed)
	return matched
}

func isHorizontalRule(line string) bool {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < 3 {
		return false
	}
	// ---, ***, ___ with optional spaces between
	dashes := strings.ReplaceAll(trimmed, " ", "")
	if len(dashes) >= 3 {
		allSame := true
		ch := dashes[0]
		if ch == '-' || ch == '*' || ch == '_' {
			for _, c := range dashes {
				if byte(c) != ch {
					allSame = false
					break
				}
			}
			return allSame
		}
	}
	return false
}

func wrapParagraph(lines []string) []string {
	// Join all lines into one, then wrap
	text := strings.Join(lines, " ")

	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	var result []string
	var currentLine strings.Builder

	for _, word := range words {
		if currentLine.Len() == 0 {
			currentLine.WriteString(word)
		} else if currentLine.Len()+1+len(word) <= *wrapWidth {
			currentLine.WriteString(" ")
			currentLine.WriteString(word)
		} else {
			result = append(result, currentLine.String())
			currentLine.Reset()
			currentLine.WriteString(word)
		}
	}

	if currentLine.Len() > 0 {
		result = append(result, currentLine.String())
	}

	return result
}
